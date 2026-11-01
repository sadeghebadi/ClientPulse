package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"clientpulse/services/api/internal/auth"
	"clientpulse/services/api/internal/checker"
	"clientpulse/services/api/internal/mail"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/service"
	"clientpulse/services/api/internal/ssrf"
	"clientpulse/services/api/internal/testhelper"
	"clientpulse/services/api/internal/worker"
)

func TestPhase2EndToEndScheduledWorkflow(t *testing.T) {
	db := testhelper.GetTestDB(t)
	defer db.Close()

	cfg := testhelper.GetTestConfig()

	userRepo := repository.NewUserRepository(db.Pool)
	sessionRepo := repository.NewSessionRepository(db.Pool)
	orgRepo := repository.NewOrganizationRepository(db.Pool)
	tokenRepo := repository.NewTokenRepository(db.Pool)
	auditRepo := repository.NewAuditRepository(db.Pool)
	clientRepo := repository.NewClientRepository(db.Pool)
	assetRepo := repository.NewAssetRepository(db.Pool)
	monitorRepo := repository.NewMonitorRepository(db.Pool)
	resultRepo := repository.NewCheckResultRepository(db.Pool)
	ssrfGuard := ssrf.NewGuard(nil)
	chk := checker.NewChecker(ssrfGuard, "e2e-api-worker")

	mockMailer := mail.NewMockMailer()
	authService := service.NewAuthService(userRepo, sessionRepo, orgRepo, tokenRepo, auditRepo, mockMailer, cfg)
	orgService := service.NewOrgService(orgRepo, userRepo, sessionRepo, auditRepo, cfg)

	deps := Dependencies{
		AuthHdr:          NewAuthHandler(authService, false),
		OrgHdr:           NewOrgHandler(orgService),
		ClientHdr:        NewClientHandler(clientRepo, auditRepo),
		AssetHdr:         NewAssetHandler(assetRepo, auditRepo, ssrfGuard, cfg.AppEnv),
		MonitorHdr:       NewMonitorHandler(monitorRepo, resultRepo, assetRepo, auditRepo, chk, ssrfGuard),
		AuthMiddleware:   NewAuthMiddleware(sessionRepo, userRepo, cfg),
		TenantMiddleware: NewTenantMiddleware(orgRepo, sessionRepo),
	}

	testLog := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, testLog, deps)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Launch real monitoring scheduler daemon in background
	workerID := "e2e-scheduler-worker"
	sched := worker.NewScheduler(
		db.Pool,
		monitorRepo,
		resultRepo,
		ssrfGuard,
		worker.SchedulerConfig{
			WorkerID:    workerID,
			Concurrency: 5,
			PollPeriod:  200 * time.Millisecond,
		},
		testLog,
	)

	go func() {
		_ = sched.Run(ctx)
	}()

	// 1. User registers and verifies email
	randSuffix, _ := auth.GenerateSecureToken(4)
	signupRes, err := authService.Signup(ctx, service.SignupRequest{
		Email:            fmt.Sprintf("e2e-%s@example.com", randSuffix),
		Password:         "Secure-E2E-Pass123!",
		DisplayName:      "E2E User",
		OrganizationName: "E2E Care Agency",
	})
	if err != nil {
		t.Fatalf("signup failed: %v", err)
	}

	err = userRepo.SetEmailVerified(ctx, signupRes.User.ID)
	if err != nil {
		t.Fatalf("failed to verify email: %v", err)
	}

	orgID := signupRes.Organization.ID
	sessToken := signupRes.RawSession
	csrfToken := signupRes.RawCSRF

	// 2. User creates a Client
	createClientBody := []byte(`{"name":"Acme Healthcare","website":"https://example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients", bytes.NewReader(createClientBody))
	req.Header.Set("X-Organization-ID", orgID)
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: sessToken})
	req.AddCookie(&http.Cookie{Name: "clientpulse_csrf", Value: csrfToken})
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for client, got %d (body: %s)", w.Code, w.Body.String())
	}

	var clientResp struct {
		Client struct {
			ID string `json:"id"`
		} `json:"client"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &clientResp)
	clientID := clientResp.Client.ID

	// 3. SSRF Check: Attempt to create an asset with internal metadata IP -> Blocked
	ssrfAssetBody := []byte(fmt.Sprintf(`{"client_id":%q,"name":"Cloud Metadata","url":"http://169.254.169.254/latest","type":"website","environment":"production"}`, clientID))
	req = httptest.NewRequest(http.MethodPost, "/api/v1/assets", bytes.NewReader(ssrfAssetBody))
	req.Header.Set("X-Organization-ID", orgID)
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: sessToken})
	req.AddCookie(&http.Cookie{Name: "clientpulse_csrf", Value: csrfToken})
	w = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for SSRF attack, got %d", w.Code)
	}

	// 4. Create valid Website Asset pointing to public target https://example.com
	validAssetBody := []byte(fmt.Sprintf(`{"client_id":%q,"name":"Public Portal","url":"https://example.com","type":"website","environment":"production"}`, clientID))
	req = httptest.NewRequest(http.MethodPost, "/api/v1/assets", bytes.NewReader(validAssetBody))
	req.Header.Set("X-Organization-ID", orgID)
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: sessToken})
	req.AddCookie(&http.Cookie{Name: "clientpulse_csrf", Value: csrfToken})
	w = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for valid asset, got %d (body: %s)", w.Code, w.Body.String())
	}

	var assetResp struct {
		Asset struct {
			ID string `json:"id"`
		} `json:"asset"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &assetResp)
	assetID := assetResp.Asset.ID

	// 5. Create Monitor for Asset
	createMonBody := []byte(fmt.Sprintf(`{"asset_id":%q,"method":"GET","interval_seconds":60,"timeout_seconds":10,"expected_status_min":200,"expected_status_max":399}`, assetID))
	req = httptest.NewRequest(http.MethodPost, "/api/v1/monitors", bytes.NewReader(createMonBody))
	req.Header.Set("X-Organization-ID", orgID)
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: sessToken})
	req.AddCookie(&http.Cookie{Name: "clientpulse_csrf", Value: csrfToken})
	w = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for monitor, got %d", w.Code)
	}

	var monResp struct {
		Monitor struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"monitor"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &monResp)
	monitorID := monResp.Monitor.ID

	// 6. Wait for the real background Go Worker scheduler to claim and execute the check
	t.Logf("Waiting for background Go worker to claim and execute monitor %s...", monitorID)
	var checkFound bool
	for i := 0; i < 25; i++ {
		time.Sleep(300 * time.Millisecond)

		results, total, err := resultRepo.ListMonitorResults(ctx, orgID, monitorID, 10, 0)
		if err == nil && total > 0 && len(results) > 0 {
			res := results[0]
			t.Logf("Real scheduled check detected: Result=%s, HTTPStatus=%v, Latency=%dms", res.Result, res.HTTPStatusCode, res.LatencyMs)
			checkFound = true
			if res.ExecutionType != "scheduled" {
				t.Fatalf("expected execution_type 'scheduled', got %s", res.ExecutionType)
			}
			break
		}
	}

	if !checkFound {
		t.Fatal("TIMED OUT: Real Go Worker did not execute scheduled check within 7.5 seconds")
	}

	// 7. Verify Dashboard returns real live statistics
	dashReq := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/stats", nil)
	dashReq.Header.Set("X-Organization-ID", orgID)
	dashReq.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: sessToken})
	w = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, dashReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for dashboard stats, got %d", w.Code)
	}

	var stats map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &stats)
	if stats["total_clients"].(float64) < 1 {
		t.Fatalf("expected at least 1 total client, got %v", stats["total_clients"])
	}
	if stats["total_assets"].(float64) < 1 {
		t.Fatalf("expected at least 1 total asset, got %v", stats["total_assets"])
	}

	// 8. Manual "Test Now" Check
	testNowReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/monitors/%s/test", monitorID), nil)
	testNowReq.Header.Set("X-Organization-ID", orgID)
	testNowReq.Header.Set("X-CSRF-Token", csrfToken)
	testNowReq.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: sessToken})
	testNowReq.AddCookie(&http.Cookie{Name: "clientpulse_csrf", Value: csrfToken})
	w = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, testNowReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for manual Test Now, got %d (body: %s)", w.Code, w.Body.String())
	}

	var testNowResp struct {
		Result struct {
			ExecutionType string `json:"execution_type"`
		} `json:"result"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &testNowResp)
	if testNowResp.Result.ExecutionType != "manual" {
		t.Fatalf("expected manual check to have execution_type 'manual', got %s", testNowResp.Result.ExecutionType)
	}

	// 9. Pause Monitor -> Status becomes 'paused'
	pauseReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/monitors/%s/pause", monitorID), bytes.NewReader([]byte("{}")))
	pauseReq.Header.Set("X-Organization-ID", orgID)
	pauseReq.Header.Set("X-CSRF-Token", csrfToken)
	pauseReq.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: sessToken})
	pauseReq.AddCookie(&http.Cookie{Name: "clientpulse_csrf", Value: csrfToken})
	w = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, pauseReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for pause, got %d", w.Code)
	}

	monAfterPause, err := monitorRepo.GetMonitorByID(ctx, orgID, monitorID)
	if err != nil || monAfterPause.Status != "paused" {
		t.Fatalf("expected monitor status 'paused', got %v (err: %v)", monAfterPause.Status, err)
	}

	// 10. Resume Monitor -> Status becomes 'pending'
	resumeReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/monitors/%s/resume", monitorID), bytes.NewReader([]byte("{}")))
	resumeReq.Header.Set("X-Organization-ID", orgID)
	resumeReq.Header.Set("X-CSRF-Token", csrfToken)
	resumeReq.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: sessToken})
	resumeReq.AddCookie(&http.Cookie{Name: "clientpulse_csrf", Value: csrfToken})
	w = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, resumeReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for resume, got %d", w.Code)
	}

	monAfterResume, err := monitorRepo.GetMonitorByID(ctx, orgID, monitorID)
	if err != nil || monAfterResume.Status != "pending" {
		t.Fatalf("expected monitor status 'pending', got %v (err: %v)", monAfterResume.Status, err)
	}
}

