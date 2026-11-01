package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/ssrf"
	"clientpulse/services/api/internal/worker"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// mockPublicResolver implements ssrf.IPResolver for controlled test targets.
type mockPublicResolver struct {
	publicIP net.IP
}

func (m *mockPublicResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	return []net.IP{m.publicIP}, nil
}

// TestScheduledMonitoringEndToEndEvidence executes a real, end-to-end check cycle
// and captures full runtime proof of execution, state transitions, and cross-tenant isolation.
func TestScheduledMonitoringEndToEndEvidence(t *testing.T) {
	ctx := context.Background()

	adminPool, err := pgxpool.New(ctx, "postgres://clientpulse:clientpulse_dev_password@localhost:5433/clientpulse?sslmode=disable")
	if err != nil {
		t.Fatalf("failed to connect admin pool: %v", err)
	}
	defer adminPool.Close()

	appPool, err := pgxpool.New(ctx, "postgres://clientpulse_app:clientpulse_app_dev_password@localhost:5433/clientpulse?sslmode=disable")
	if err != nil {
		t.Fatalf("failed to connect app pool: %v", err)
	}
	defer appPool.Close()

	workerPool, err := pgxpool.New(ctx, "postgres://clientpulse_worker:clientpulse_worker_dev_password@localhost:5433/clientpulse?sslmode=disable")
	if err != nil {
		t.Fatalf("failed to connect worker pool: %v", err)
	}
	defer workerPool.Close()

	// 1. Provision Authenticated Organizations Org 1 and Org 2
	var org1ID, org2ID string
	err = adminPool.QueryRow(ctx, "INSERT INTO organizations (name, slug) VALUES ('Evidence Org 1', 'evidence-org-1') RETURNING id").Scan(&org1ID)
	if err != nil {
		t.Fatalf("failed to create org 1: %v", err)
	}
	defer adminPool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", org1ID)

	err = adminPool.QueryRow(ctx, "INSERT INTO organizations (name, slug) VALUES ('Evidence Org 2', 'evidence-org-2') RETURNING id").Scan(&org2ID)
	if err != nil {
		t.Fatalf("failed to create org 2: %v", err)
	}
	defer adminPool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", org2ID)

	// 2. Controlled Target HTTP Server
	var requestCount int64
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ClientPulse Real Check Target Healthy"))
	}))
	defer targetServer.Close()

	// 3. Create Real Client, Asset, and Monitor in Org 1
	clientRepo := repository.NewClientRepository(appPool)
	assetRepo := repository.NewAssetRepository(appPool)
	monitorRepo := repository.NewMonitorRepository(appPool)
	workerMonitorRepo := repository.NewMonitorRepository(workerPool)
	workerResultRepo := repository.NewCheckResultRepository(workerPool)

	clientWebsite := "https://alpha.example.com"
	client, err := clientRepo.CreateClient(ctx, org1ID, "Alpha Client", nil, nil, &clientWebsite, nil)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	asset, err := assetRepo.CreateAsset(ctx, org1ID, client.ID, "Target Server Asset", "http://target.clientpulse-test.org", "website", "production", nil)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	expectedText := "Healthy"
	mon, err := monitorRepo.CreateMonitor(ctx, org1ID, repository.CreateMonitorParams{
		AssetID:           asset.ID,
		Method:            "GET",
		IntervalSeconds:   60,
		TimeoutSeconds:    5,
		ExpectedStatusMin: 200,
		ExpectedStatusMax: 299,
		ExpectedText:      &expectedText,
	})
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	// Make monitor immediately due for the scheduler (oldest next_check_at so it is picked first)
	_, err = adminPool.Exec(ctx, "UPDATE monitors SET next_check_at = NOW() - INTERVAL '1 hour', lease_acquired_at = NULL, lease_worker_id = NULL WHERE id = $1", mon.ID)
	if err != nil {
		t.Fatalf("failed to set monitor due: %v", err)
	}

	t.Logf("=== PROVISIONED ENTITIES ===")
	t.Logf("Organization 1 ID: %s", org1ID)
	t.Logf("Client ID:         %s (%s)", client.ID, client.Name)
	t.Logf("Asset ID:          %s (%s)", asset.ID, asset.URL)
	t.Logf("Monitor ID:        %s (initial status: %s, next_check: %s)", mon.ID, mon.Status, mon.NextCheckAt.Format(time.RFC3339))

	// 4. Test Resolver: Resolves test hostname to a public routable IP so SSRF guard classifies as safe public target
	publicTestIP := net.ParseIP("93.184.216.34")
	mockResolver := &mockPublicResolver{publicIP: publicTestIP}
	guard := ssrf.NewGuard(mockResolver)

	// 5. Injected test transport routes the HTTP request to the controlled test server
	// while all URL validation, IP classification, and protocol assertions remain 100% active.
	rewriter := &rewriteTransport{
		targetURL: targetServer.URL,
		transport: targetServer.Client().Transport,
	}

	schedCfg := worker.SchedulerConfig{
		WorkerID:      "evidence-worker-01",
		Concurrency:   2,
		PollPeriod:    100 * time.Millisecond,
		RetentionDays: 90,
	}
	workerHbRepo := repository.NewHeartbeatRepository(workerPool)
	workerIncRepo := repository.NewIncidentRepository(workerPool)
	sched := worker.NewScheduler(workerPool, workerMonitorRepo, workerResultRepo, workerHbRepo, workerIncRepo, guard, schedCfg, nil)
	sched.SetTestTransport(rewriter)

	schedCtx, schedCancel := context.WithTimeout(ctx, 2*time.Second)
	defer schedCancel()

	// Run scheduler until timeout
	_ = sched.Run(schedCtx)

	// 6. Verify Execution Evidence
	if atomic.LoadInt64(&requestCount) == 0 {
		t.Fatalf("expected target server to receive at least 1 check request, received 0")
	}

	// Fetch Check Result row from PostgreSQL
	var resultRow repository.CheckResult
	err = repository.WithTenantTx(ctx, appPool, org1ID, func(tx pgx.Tx) error {
		query := `
			SELECT id, organization_id, monitor_id, scheduled_at, started_at, completed_at,
			       latency_ms, result, http_status_code, worker_id, execution_type, created_at
			FROM check_results
			WHERE monitor_id = $1 AND organization_id = $2
			ORDER BY created_at DESC
			LIMIT 1
		`
		return tx.QueryRow(ctx, query, mon.ID, org1ID).Scan(
			&resultRow.ID, &resultRow.OrganizationID, &resultRow.MonitorID,
			&resultRow.ScheduledAt, &resultRow.StartedAt, &resultRow.CompletedAt,
			&resultRow.LatencyMs, &resultRow.Result, &resultRow.HTTPStatusCode,
			&resultRow.WorkerID, &resultRow.ExecutionType, &resultRow.CreatedAt,
		)
	})
	if err != nil {
		t.Fatalf("failed to retrieve stored check result row: %v", err)
	}

	// Fetch updated Monitor row from PostgreSQL
	updatedMon, err := monitorRepo.GetMonitorByID(ctx, org1ID, mon.ID)
	if err != nil {
		t.Fatalf("failed to retrieve updated monitor row: %v", err)
	}

	t.Logf("=== RUNTIME EXECUTION EVIDENCE ===")
	t.Logf("Monitor ID:             %s", updatedMon.ID)
	t.Logf("Check Scheduled At:     %s", resultRow.ScheduledAt.Format(time.RFC3339))
	t.Logf("Check Started At:       %s", resultRow.StartedAt.Format(time.RFC3339))
	t.Logf("Check Completed At:     %s", resultRow.CompletedAt.Format(time.RFC3339))
	t.Logf("Check Latency:          %d ms", resultRow.LatencyMs)
	t.Logf("HTTP Status Code:       %d", *resultRow.HTTPStatusCode)
	t.Logf("Check Result Outcome:   %s", resultRow.Result)
	t.Logf("Worker Identifier:      %s", resultRow.WorkerID)
	t.Logf("Execution Type:         %s", resultRow.ExecutionType)
	t.Logf("Updated Monitor Status: %s", updatedMon.Status)
	t.Logf("Next Scheduled Check:   %s", updatedMon.NextCheckAt.Format(time.RFC3339))
	t.Logf("Consecutive Failures:   %d", updatedMon.ConsecutiveFailures)

	if resultRow.Result != "success" {
		t.Fatalf("expected check result success, got %s", resultRow.Result)
	}
	if updatedMon.Status != "up" {
		t.Fatalf("expected updated monitor status 'up', got %s", updatedMon.Status)
	}
	if !updatedMon.NextCheckAt.After(resultRow.CompletedAt) {
		t.Fatalf("expected next_check_at (%s) to be in the future of completed_at (%s)", updatedMon.NextCheckAt, resultRow.CompletedAt)
	}

	// 7. Verify Multi-Tenant Isolation: Org 2 CANNOT read Org 1 check result or monitor
	t.Run("Org2_Isolation_CheckResult_Invisible", func(t *testing.T) {
		var org2Count int
		err := repository.WithTenantTx(ctx, appPool, org2ID, func(tx pgx.Tx) error {
			return tx.QueryRow(ctx, "SELECT COUNT(*) FROM check_results WHERE id = $1", resultRow.ID).Scan(&org2Count)
		})
		if err != nil {
			t.Fatalf("failed querying check_results in Org 2: %v", err)
		}
		if org2Count != 0 {
			t.Fatalf("TENANT ISOLATION LEAK: Org 2 was able to see CheckResult belonging to Org 1!")
		}
	})

	t.Run("Org2_Isolation_Monitor_Invisible", func(t *testing.T) {
		_, err := monitorRepo.GetMonitorByID(ctx, org2ID, mon.ID)
		if err == nil {
			t.Fatalf("TENANT ISOLATION LEAK: Org 2 was able to retrieve Monitor belonging to Org 1!")
		}
	})
}

type rewriteTransport struct {
	targetURL string
	transport http.RoundTripper
}

func (r *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	parsedTarget, err := url.Parse(r.targetURL)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = parsedTarget.Scheme
	req.URL.Host = parsedTarget.Host
	if r.transport != nil {
		return r.transport.RoundTrip(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}
