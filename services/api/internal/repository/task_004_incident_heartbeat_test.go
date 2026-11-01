package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func strPtr(s string) *string {
	return &s
}

func setupTestRoles(t *testing.T) (*pgxpool.Pool, *pgxpool.Pool, *pgxpool.Pool) {
	t.Helper()
	adminPool := openPoolAsRole(t, "clientpulse", "clientpulse_dev_password")
	appPool := openPoolAsRole(t, "clientpulse_app", "clientpulse_app_dev_password")
	workerPool := openPoolAsRole(t, "clientpulse_worker", "clientpulse_worker_dev_password")

	t.Cleanup(func() {
		adminPool.Close()
		appPool.Close()
		workerPool.Close()
	})
	return adminPool, appPool, workerPool
}

func createTestOrgAndClient(t *testing.T, pool *pgxpool.Pool, name, slug string) (string, string) {
	t.Helper()
	ctx := context.Background()
	var orgID, clientID string

	err := pool.QueryRow(ctx, `
		INSERT INTO organizations (name, slug)
		VALUES ($1, $2)
		RETURNING id
	`, name, slug).Scan(&orgID)
	if err != nil {
		t.Fatalf("failed to create test org: %v", err)
	}

	err = pool.QueryRow(ctx, `
		INSERT INTO clients (organization_id, name)
		VALUES ($1, $2)
		RETURNING id
	`, orgID, name+" Client").Scan(&clientID)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	return orgID, clientID
}

func TestDatabaseRoleHeartbeatPermissions(t *testing.T) {
	ctx := context.Background()
	_, appPool, workerPool := setupTestRoles(t)

	// Test 1: API role (clientpulse_app) MUST NOT execute claim_missed_heartbeats
	t.Run("API_Role_Denied_Execute_ClaimMissedHeartbeats", func(t *testing.T) {
		rows, err := appPool.Query(ctx, "SELECT id FROM claim_missed_heartbeats('worker-test', 10, 60)")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
			}
			err = rows.Err()
		}
		if err == nil {
			t.Fatalf("SECURITY VIOLATION: clientpulse_app was able to execute claim_missed_heartbeats!")
		}
		t.Logf("Expected permission denied error: %v", err)
	})

	// Test 2: Worker role CAN execute claim_missed_heartbeats
	t.Run("Worker_Role_Allowed_Execute_ClaimMissedHeartbeats", func(t *testing.T) {
		rows, err := workerPool.Query(ctx, "SELECT id FROM claim_missed_heartbeats('worker-test', 10, 60)")
		if err != nil {
			t.Fatalf("Worker role should be able to execute claim_missed_heartbeats: %v", err)
		}
		rows.Close()
	})

	// Test 3: Unauthenticated lookup_heartbeat_by_token_hash can be called by clientpulse_app
	t.Run("App_Role_Allowed_LookupHeartbeatByTokenHash", func(t *testing.T) {
		var id, orgID, tokHash, status string
		var archivedAt *time.Time
		err := appPool.QueryRow(ctx, "SELECT id, organization_id, token_hash, status, archived_at FROM lookup_heartbeat_by_token_hash('nonexistent')").
			Scan(&id, &orgID, &tokHash, &status, &archivedAt)
		if !errors.Is(err, pgx.ErrNoRows) && err != nil {
			t.Fatalf("unexpected error calling lookup_heartbeat_by_token_hash: %v", err)
		}
	})
}

func TestCompositeTenantForeignKeysIncidentsAndHeartbeats(t *testing.T) {
	ctx := context.Background()
	adminPool, appPool, _ := setupTestRoles(t)

	orgA, clientA := createTestOrgAndClient(t, adminPool, "Org A Composite", "org-a-comp")
	defer adminPool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgA)

	orgB, clientB := createTestOrgAndClient(t, adminPool, "Org B Composite", "org-b-comp")
	defer adminPool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgB)

	// Create Asset in Org A
	var assetA string
	err := adminPool.QueryRow(ctx, `
		INSERT INTO assets (organization_id, client_id, name, url)
		VALUES ($1, $2, 'Asset A', 'https://a.com')
		RETURNING id
	`, orgA, clientA).Scan(&assetA)
	if err != nil {
		t.Fatalf("failed to create asset in Org A: %v", err)
	}

	// 1. Cross-tenant client on heartbeat must fail
	t.Run("CrossTenant_Heartbeat_To_Client_MustFail", func(t *testing.T) {
		err := WithTenantTx(ctx, appPool, orgB, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `
				INSERT INTO heartbeat_monitors (organization_id, client_id, name, token_hash)
				VALUES ($1, $2, 'Bad HB', 'hash-12345678901234567890123456789012')
			`, orgB, clientA)
			return err
		})
		if err == nil {
			t.Fatalf("SECURITY VIOLATION: Cross-tenant heartbeat was inserted successfully!")
		}
		if !strings.Contains(err.Error(), "fk_heartbeats_client_org") && !strings.Contains(err.Error(), "violates foreign key") {
			t.Fatalf("expected foreign key violation for cross-tenant client, got %v", err)
		}
	})

	// 2. Cross-tenant asset on heartbeat must fail
	t.Run("CrossTenant_Heartbeat_To_Asset_MustFail", func(t *testing.T) {
		err := WithTenantTx(ctx, appPool, orgB, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `
				INSERT INTO heartbeat_monitors (organization_id, client_id, asset_id, name, token_hash)
				VALUES ($1, $2, $3, 'Bad Asset HB', 'hash-22345678901234567890123456789012')
			`, orgB, clientB, assetA)
			return err
		})
		if err == nil {
			t.Fatalf("SECURITY VIOLATION: Cross-tenant asset on heartbeat succeeded!")
		}
		t.Logf("Expected foreign key violation for cross-tenant asset: %v", err)
	})

	// 3. Create valid heartbeat in Org A
	var hbA string
	err = adminPool.QueryRow(ctx, `
		INSERT INTO heartbeat_monitors (organization_id, client_id, name, token_hash)
		VALUES ($1, $2, 'Valid HB Org A', 'hash-32345678901234567890123456789012')
		RETURNING id
	`, orgA, clientA).Scan(&hbA)
	if err != nil {
		t.Fatalf("failed to create valid heartbeat in Org A: %v", err)
	}

	// 4. Cross-tenant incident referencing Heartbeat A from Org B must fail
	t.Run("CrossTenant_Incident_To_Heartbeat_MustFail", func(t *testing.T) {
		err := WithTenantTx(ctx, appPool, orgB, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `
				INSERT INTO incidents (organization_id, heartbeat_monitor_id, first_failure_at, confirmed_at, status)
				VALUES ($1, $2, NOW(), NOW(), 'open')
			`, orgB, hbA)
			return err
		})
		if err == nil {
			t.Fatalf("SECURITY VIOLATION: Cross-tenant incident was created!")
		}
		t.Logf("Expected foreign key violation: %v", err)
	})

	// 5. Single target constraint on incidents
	t.Run("Incident_Must_Reference_Exactly_One_Target", func(t *testing.T) {
		// Attempt incident with BOTH monitor_id and heartbeat_monitor_id
		err := WithTenantTx(ctx, appPool, orgA, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `
				INSERT INTO incidents (organization_id, monitor_id, heartbeat_monitor_id, first_failure_at, confirmed_at, status)
				VALUES ($1, '00000000-0000-0000-0000-000000000001', $2, NOW(), NOW(), 'open')
			`, orgA, hbA)
			return err
		})
		if err == nil {
			t.Fatalf("expected check constraint violation for dual target incident")
		}

		// Attempt incident with NEITHER monitor_id nor heartbeat_monitor_id
		err = WithTenantTx(ctx, appPool, orgA, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `
				INSERT INTO incidents (organization_id, first_failure_at, confirmed_at, status)
				VALUES ($1, NOW(), NOW(), 'open')
			`, orgA)
			return err
		})
		if err == nil {
			t.Fatalf("expected check constraint violation for zero target incident")
		}
	})
}

func TestIncidentLifecycleAndThresholds(t *testing.T) {
	ctx := context.Background()
	adminPool, appPool, _ := setupTestRoles(t)

	clientRepo := NewClientRepository(appPool)
	assetRepo := NewAssetRepository(appPool)
	monitorRepo := NewMonitorRepository(appPool)
	incidentRepo := NewIncidentRepository(appPool)
	notifRepo := NewNotificationEventRepository(appPool)

	orgA, _ := createTestOrgAndClient(t, adminPool, "Incident Lifecycle Org", "inc-lifecycle-org")
	defer adminPool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgA)

	clientA, err := clientRepo.CreateClient(ctx, orgA, "Alpha Client", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	assetA, err := assetRepo.CreateAsset(ctx, orgA, clientA.ID, "Alpha Asset", "https://alpha.example.com", "website", "production", nil)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}
	monA, err := monitorRepo.CreateMonitor(ctx, orgA, CreateMonitorParams{
		AssetID:           assetA.ID,
		Method:            "GET",
		IntervalSeconds:   60,
		TimeoutSeconds:    10,
		ExpectedStatusMin: 200,
		ExpectedStatusMax: 299,
		FollowRedirects:   true,
		MaxRedirects:      5,
	})
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	now := time.Now().UTC()

	// 1. First failure -> suspected_down, NO incident created
	st1, inc1, err := monitorRepo.RecordScheduledCheckOutcomeWithResult(ctx, ScheduledCheckOutcomeParams{
		OrganizationID:  orgA,
		MonitorID:       monA.ID,
		CheckSuccess:    false,
		CompletedAt:     now,
		ScheduledAt:     now,
		IntervalSeconds: 60,
		ErrorCategory:   strPtr("connection_refused"),
		ErrorMessage:    strPtr("dial tcp: connection refused"),
	})
	if err != nil {
		t.Fatalf("failed 1st check outcome: %v", err)
	}
	if st1 != "suspected_down" {
		t.Fatalf("expected status suspected_down, got %s", st1)
	}
	if inc1 != nil {
		t.Fatalf("expected NO incident after 1st failure, got %+v", inc1)
	}

	// Verify no incident exists in DB
	incidents, total, err := incidentRepo.ListIncidents(ctx, orgA, IncidentFilter{MonitorID: monA.ID})
	if err != nil || total != 0 || len(incidents) != 0 {
		t.Fatalf("expected 0 incidents after 1 failure, got %d", total)
	}

	// 2. Second consecutive failure -> down, EXACTLY ONE incident created
	failTime2 := now.Add(60 * time.Second)
	st2, inc2, err := monitorRepo.RecordScheduledCheckOutcomeWithResult(ctx, ScheduledCheckOutcomeParams{
		OrganizationID:  orgA,
		MonitorID:       monA.ID,
		CheckSuccess:    false,
		CompletedAt:     failTime2,
		ScheduledAt:     failTime2,
		IntervalSeconds: 60,
		ErrorCategory:   strPtr("connection_refused"),
		ErrorMessage:    strPtr("dial tcp: connection refused"),
	})
	if err != nil {
		t.Fatalf("failed 2nd check outcome: %v", err)
	}
	if st2 != "down" {
		t.Fatalf("expected status down after 2nd failure, got %s", st2)
	}
	if inc2 == nil {
		t.Fatalf("expected incident to be created after 2nd failure")
	}
	if inc2.Status != "open" {
		t.Fatalf("expected incident status open, got %s", inc2.Status)
	}
	if inc2.FailureCategory != "connection_refused" {
		t.Fatalf("expected failure_category connection_refused, got %s", inc2.FailureCategory)
	}

	// 3. Third consecutive failure -> remains down, keeps SAME incident open, NO duplicate
	failTime3 := now.Add(120 * time.Second)
	st3, inc3, err := monitorRepo.RecordScheduledCheckOutcomeWithResult(ctx, ScheduledCheckOutcomeParams{
		OrganizationID:  orgA,
		MonitorID:       monA.ID,
		CheckSuccess:    false,
		CompletedAt:     failTime3,
		ScheduledAt:     failTime3,
		IntervalSeconds: 60,
		ErrorCategory:   strPtr("connection_refused"),
		ErrorMessage:    strPtr("dial tcp: connection refused"),
	})
	if err != nil {
		t.Fatalf("failed 3rd check outcome: %v", err)
	}
	if st3 != "down" {
		t.Fatalf("expected status down, got %s", st3)
	}
	if inc3 != nil {
		t.Fatalf("expected nil incident returned on duplicate failure, got %+v", inc3)
	}

	// Verify only 1 open incident exists in DB
	incidents, total, err = incidentRepo.ListIncidents(ctx, orgA, IncidentFilter{MonitorID: monA.ID, Status: "open"})
	if err != nil || total != 1 || len(incidents) != 1 {
		t.Fatalf("expected exactly 1 open incident, got count %d", total)
	}
	openInc := incidents[0]
	if openInc.ID != inc2.ID {
		t.Fatalf("expected incident ID %s, got %s", inc2.ID, openInc.ID)
	}

	// 4. Acknowledging incident does NOT resolve it
	userRepo := NewUserRepository(adminPool)
	testUser, _ := userRepo.CreateUser(ctx, fmt.Sprintf("acker-%d@pulse.test", time.Now().UnixNano()), "passwordHash", "Incident Ack User")

	ackedInc, err := incidentRepo.AcknowledgeIncident(ctx, orgA, openInc.ID, testUser.ID)
	if err != nil {
		t.Fatalf("failed to acknowledge incident: %v", err)
	}
	if ackedInc.Status != "open" {
		t.Fatalf("acknowledgement must NOT change status to resolved; got %s", ackedInc.Status)
	}
	if ackedInc.AcknowledgedAt == nil || ackedInc.AcknowledgedBy == nil {
		t.Fatalf("expected acknowledged_at and acknowledged_by to be set")
	}

	// Attempt duplicate acknowledgement -> must fail
	_, err = incidentRepo.AcknowledgeIncident(ctx, orgA, openInc.ID, testUser.ID)
	if !errors.Is(err, ErrIncidentAlreadyAcked) {
		t.Fatalf("expected ErrIncidentAlreadyAcked, got %v", err)
	}

	// 5. Recovery threshold (1 success) -> resolves the SAME incident
	recoveryTime := now.Add(180 * time.Second)
	stRec, incRec, err := monitorRepo.RecordScheduledCheckOutcomeWithResult(ctx, ScheduledCheckOutcomeParams{
		OrganizationID:  orgA,
		MonitorID:       monA.ID,
		CheckSuccess:    true,
		CompletedAt:     recoveryTime,
		ScheduledAt:     recoveryTime,
		IntervalSeconds: 60,
	})
	if err != nil {
		t.Fatalf("failed recovery check outcome: %v", err)
	}
	if stRec != "up" {
		t.Fatalf("expected monitor status up after recovery, got %s", stRec)
	}
	if incRec == nil {
		t.Fatalf("expected resolved incident to be returned on recovery")
	}
	if incRec.ID != openInc.ID {
		t.Fatalf("expected resolved incident ID %s, got %s", openInc.ID, incRec.ID)
	}
	if incRec.Status != "resolved" {
		t.Fatalf("expected incident status resolved, got %s", incRec.Status)
	}
	if incRec.ResolvedAt == nil {
		t.Fatalf("expected resolved_at to be non-nil")
	}
	if incRec.DurationSeconds == nil || *incRec.DurationSeconds < 0 {
		t.Fatalf("expected positive duration_seconds, got %v", incRec.DurationSeconds)
	}

	// 6. Flapping protection: subsequent successes do NOT generate duplicate recovery events
	succTime2 := now.Add(240 * time.Second)
	stSucc2, incSucc2, err := monitorRepo.RecordScheduledCheckOutcomeWithResult(ctx, ScheduledCheckOutcomeParams{
		OrganizationID:  orgA,
		MonitorID:       monA.ID,
		CheckSuccess:    true,
		CompletedAt:     succTime2,
		ScheduledAt:     succTime2,
		IntervalSeconds: 60,
	})
	if err != nil || stSucc2 != "up" || incSucc2 != nil {
		t.Fatalf("expected clean success with nil incident on subsequent up checks: err=%v, st=%s, inc=%v", err, stSucc2, incSucc2)
	}

	// Verify notification events generated
	events, err := notifRepo.GetEventsByIncidentID(ctx, orgA, openInc.ID)
	if err != nil {
		t.Fatalf("failed to get incident notification events: %v", err)
	}
	if len(events) < 3 {
		t.Fatalf("expected at least 3 events (opened, acknowledged, resolved), got %d", len(events))
	}
	types := []string{}
	for _, ev := range events {
		types = append(types, ev.EventType)
	}
	expectedTypes := []string{"incident.opened", "incident.acknowledged", "incident.resolved"}
	for _, exp := range expectedTypes {
		found := false
		for _, t := range types {
			if t == exp {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected event type %s in %v", exp, types)
		}
	}
}

func TestHeartbeatMonitoringLifecycleAndRecovery(t *testing.T) {
	ctx := context.Background()
	adminPool, appPool, workerPool := setupTestRoles(t)

	clientRepo := NewClientRepository(appPool)
	hbRepo := NewHeartbeatRepository(appPool)
	incidentRepo := NewIncidentRepository(appPool)
	workerHbRepo := NewHeartbeatRepository(workerPool)

	orgA, _ := createTestOrgAndClient(t, adminPool, "Heartbeat Lifecycle Org", "hb-lifecycle-org")
	defer adminPool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgA)

	clientA, err := clientRepo.CreateClient(ctx, orgA, "Heartbeat Client", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// 1. Create Heartbeat monitor
	hb, token, err := hbRepo.CreateHeartbeat(ctx, orgA, CreateHeartbeatParams{
		ClientID:                clientA.ID,
		Name:                    "Nightly DB Backup",
		ExpectedIntervalSeconds: 3600,
		GracePeriodSeconds:      720,
	})
	if err != nil {
		t.Fatalf("failed to create heartbeat: %v", err)
	}
	if hb.Status != "pending" {
		t.Fatalf("expected initial status pending, got %s", hb.Status)
	}
	if len(token) < 32 {
		t.Fatalf("expected secure token length >= 32, got %d", len(token))
	}

	// Verify token is NOT stored in plaintext
	var storedHash string
	err = adminPool.QueryRow(ctx, "SELECT token_hash FROM heartbeat_monitors WHERE id = $1", hb.ID).Scan(&storedHash)
	if err != nil || storedHash == token {
		t.Fatalf("plaintext token was improperly stored in database! hash=%s, token=%s", storedHash, token)
	}

	// 2. Send valid ping -> status becomes healthy
	hbPinged, inc, err := hbRepo.RecordHeartbeatPing(ctx, token)
	if err != nil {
		t.Fatalf("valid ping failed: %v", err)
	}
	if hbPinged.Status != "healthy" {
		t.Fatalf("expected healthy status after ping, got %s", hbPinged.Status)
	}
	if hbPinged.LastReceivedAt == nil {
		t.Fatalf("expected last_received_at to be set")
	}
	if hbPinged.NextExpectedAt == nil {
		t.Fatalf("expected next_expected_at to be scheduled")
	}
	if inc != nil {
		t.Fatalf("no incident should be resolved on initial ping")
	}

	// 3. Invalid token -> rejected
	_, _, err = hbRepo.RecordHeartbeatPing(ctx, "completely-invalid-token-1234567890")
	if !errors.Is(err, ErrHeartbeatNotFound) {
		t.Fatalf("expected ErrHeartbeatNotFound for invalid token, got %v", err)
	}

	// 4. Token rotation -> old token invalidated, new token accepted
	newToken, err := hbRepo.RotateToken(ctx, orgA, hb.ID)
	if err != nil {
		t.Fatalf("failed to rotate token: %v", err)
	}
	if newToken == token {
		t.Fatalf("new rotated token must differ from old token")
	}

	// Old token must fail
	_, _, err = hbRepo.RecordHeartbeatPing(ctx, token)
	if !errors.Is(err, ErrHeartbeatNotFound) {
		t.Fatalf("expected old token to fail after rotation, got %v", err)
	}

	// New token must succeed
	hbRotated, _, err := hbRepo.RecordHeartbeatPing(ctx, newToken)
	if err != nil || hbRotated.Status != "healthy" {
		t.Fatalf("new token failed: %v", err)
	}

	// 5. Missed heartbeat detection
	// Simulate deadline passing: update next_expected_at to the past
	pastTime := time.Now().UTC().Add(-2 * time.Hour)
	_, err = adminPool.Exec(ctx, "UPDATE heartbeat_monitors SET next_expected_at = $1 WHERE id = $2", pastTime, hb.ID)
	if err != nil {
		t.Fatalf("failed to update next_expected_at: %v", err)
	}

	claimedJobs, err := workerHbRepo.ClaimMissedHeartbeats(ctx, "test-worker-01", 10, 60)
	if err != nil {
		t.Fatalf("failed to claim missed heartbeats: %v", err)
	}
	if len(claimedJobs) == 0 {
		t.Fatalf("expected overdue heartbeat to be claimed")
	}

	foundClaim := false
	for _, job := range claimedJobs {
		if job.ID == hb.ID {
			foundClaim = true
			// Create heartbeat incident
			err = WithTenantTx(ctx, adminPool, job.OrganizationID, func(tx pgx.Tx) error {
				inc, created, err := incidentRepo.CreateHeartbeatIncidentTx(
					ctx, tx, job.OrganizationID, job.ID,
					*job.NextExpectedAt, time.Now().UTC(),
					"heartbeat_missed",
					"Heartbeat missed deadline",
				)
				if err != nil {
					return err
				}
				if !created || inc == nil {
					return fmt.Errorf("expected incident to be created")
				}
				return nil
			})
			if err != nil {
				t.Fatalf("failed to create missed incident: %v", err)
			}
		}
	}
	if !foundClaim {
		t.Fatalf("overdue heartbeat %s was not among claimed jobs", hb.ID)
	}

	// Verify open incident exists in DB
	hbIncidents, total, err := incidentRepo.ListIncidents(ctx, orgA, IncidentFilter{HeartbeatMonitorID: hb.ID, Status: "open"})
	if err != nil || total != 1 {
		t.Fatalf("expected 1 open heartbeat incident, got %d", total)
	}
	openHbInc := hbIncidents[0]

	// 6. Recovery: sending a valid ping resolves the open incident automatically
	hbRecovered, resolvedInc, err := hbRepo.RecordHeartbeatPing(ctx, newToken)
	if err != nil {
		t.Fatalf("recovery ping failed: %v", err)
	}
	if hbRecovered.Status != "healthy" {
		t.Fatalf("expected healthy status on recovery, got %s", hbRecovered.Status)
	}
	if resolvedInc == nil {
		t.Fatalf("expected open incident to be resolved automatically on heartbeat ping")
	}
	if resolvedInc.ID != openHbInc.ID {
		t.Fatalf("expected resolved incident ID %s, got %s", openHbInc.ID, resolvedInc.ID)
	}
	if resolvedInc.Status != "resolved" {
		t.Fatalf("expected status resolved, got %s", resolvedInc.Status)
	}

	// 7. Revocation
	err = hbRepo.RevokeToken(ctx, orgA, hb.ID)
	if err != nil {
		t.Fatalf("failed to revoke token: %v", err)
	}
	_, _, err = hbRepo.RecordHeartbeatPing(ctx, newToken)
	if !errors.Is(err, ErrHeartbeatRevoked) && !errors.Is(err, ErrHeartbeatNotFound) {
		t.Fatalf("expected revocation rejection, got %v", err)
	}
}

func TestPhase3TenantIsolation(t *testing.T) {
	ctx := context.Background()
	adminPool, appPool, _ := setupTestRoles(t)

	clientRepo := NewClientRepository(appPool)
	hbRepo := NewHeartbeatRepository(appPool)
	incidentRepo := NewIncidentRepository(appPool)

	orgA, _ := createTestOrgAndClient(t, adminPool, "Tenant Isolation Org A", "tenant-iso-org-a")
	defer adminPool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgA)

	orgB, _ := createTestOrgAndClient(t, adminPool, "Tenant Isolation Org B", "tenant-iso-org-b")
	defer adminPool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgB)

	clientA, _ := clientRepo.CreateClient(ctx, orgA, "Client Org A", nil, nil, nil, nil)
	_, _ = clientRepo.CreateClient(ctx, orgB, "Client Org B", nil, nil, nil, nil)

	// Org A creates a heartbeat
	hbA, tokenA, err := hbRepo.CreateHeartbeat(ctx, orgA, CreateHeartbeatParams{
		ClientID:                clientA.ID,
		Name:                    "Org A Heartbeat",
		ExpectedIntervalSeconds: 3600,
	})
	if err != nil {
		t.Fatalf("failed to create Org A heartbeat: %v", err)
	}

	// Org B attempts to retrieve Org A's heartbeat -> must fail with ErrHeartbeatNotFound
	_, err = hbRepo.GetHeartbeatByID(ctx, orgB, hbA.ID)
	if !errors.Is(err, ErrHeartbeatNotFound) {
		t.Fatalf("expected ErrHeartbeatNotFound for cross-tenant access, got %v", err)
	}

	// Org B attempts to rotate Org A's token -> must fail
	_, err = hbRepo.RotateToken(ctx, orgB, hbA.ID)
	if !errors.Is(err, ErrHeartbeatNotFound) {
		t.Fatalf("expected ErrHeartbeatNotFound on cross-tenant rotate, got %v", err)
	}

	// Org B attempts to attach Heartbeat to Client A -> must fail because Client A is in Org A
	_, _, err = hbRepo.CreateHeartbeat(ctx, orgB, CreateHeartbeatParams{
		ClientID:                clientA.ID,
		Name:                    "Cross Tenant Client HB",
		ExpectedIntervalSeconds: 3600,
	})
	if err == nil {
		t.Fatalf("expected failure creating heartbeat with cross-tenant client")
	}

	// Ping from tokenA is valid, but attributes to Org A exclusively
	pingedA, _, err := hbRepo.RecordHeartbeatPing(ctx, tokenA)
	if err != nil || pingedA.OrganizationID != orgA {
		t.Fatalf("ping failed or wrong org attributed: err=%v, org=%s", err, pingedA.OrganizationID)
	}

	// Org B lists incidents -> does NOT see Org A's incidents
	incListB, totalB, err := incidentRepo.ListIncidents(ctx, orgB, IncidentFilter{})
	if err != nil {
		t.Fatalf("failed to list incidents for Org B: %v", err)
	}
	if totalB != 0 || len(incListB) != 0 {
		t.Fatalf("Org B should see 0 incidents, got %d", totalB)
	}
}

func TestNewlyCreatedHeartbeatWithoutPingCanBecomeMissed(t *testing.T) {
	adminPool, appPool, workerPool := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("hb-deadline-%d", time.Now().UnixNano())
	orgID, clientID := createTestOrgAndClient(t, adminPool, "HB First Deadline Org", slug)

	hbRepo := NewHeartbeatRepository(appPool)
	workerHbRepo := NewHeartbeatRepository(workerPool)

	// 1. Create a brand new heartbeat with NO pings
	interval := 300 // 5 mins
	grace := 60     // 1 min
	hb, _, err := hbRepo.CreateHeartbeat(ctx, orgID, CreateHeartbeatParams{
		ClientID:                clientID,
		Name:                    "Unpinged Backup Worker",
		ExpectedIntervalSeconds: interval,
		GracePeriodSeconds:      grace,
	})
	if err != nil {
		t.Fatalf("failed to create heartbeat: %v", err)
	}

	// Verify initial status is pending
	if hb.Status != "pending" {
		t.Fatalf("expected initial status 'pending', got '%s'", hb.Status)
	}

	// Verify next_expected_at was initialized to creation + interval + grace
	if hb.NextExpectedAt == nil {
		t.Fatalf("expected next_expected_at to be non-nil upon creation")
	}
	expectedDeadline := hb.CreatedAt.Add(time.Duration(interval+grace) * time.Second)
	diff := hb.NextExpectedAt.Sub(expectedDeadline)
	if diff < -2*time.Second || diff > 2*time.Second {
		t.Errorf("expected deadline ~%v, got %v (diff: %v)", expectedDeadline, *hb.NextExpectedAt, diff)
	}

	// 2. Simulate deadline expiry without ANY ping arriving (e.g. initial deadline passes)
	pastDeadline := time.Now().UTC().Add(-10 * time.Minute)
	_, err = adminPool.Exec(ctx, "UPDATE heartbeat_monitors SET next_expected_at = $1 WHERE id = $2", pastDeadline, hb.ID)
	if err != nil {
		t.Fatalf("failed setting overdue next_expected_at: %v", err)
	}

	// 3. Worker claims missed heartbeats -> Must claim this pending unpinged heartbeat
	claimedJobs, err := workerHbRepo.ClaimMissedHeartbeats(ctx, "test-worker-deadline", 10, 60)
	if err != nil {
		t.Fatalf("failed to claim missed heartbeats: %v", err)
	}

	claimed := false
	for _, job := range claimedJobs {
		if job.ID == hb.ID {
			claimed = true
			if job.Status != "missed" {
				t.Errorf("expected claimed job status 'missed', got '%s'", job.Status)
			}
			break
		}
	}
	if !claimed {
		t.Fatalf("expected newly created unpinged heartbeat %s to be claimed as missed", hb.ID)
	}

	// 4. Verify in DB that status transitioned to 'missed'
	var updatedStatus string
	err = adminPool.QueryRow(ctx, "SELECT status FROM heartbeat_monitors WHERE id = $1", hb.ID).Scan(&updatedStatus)
	if err != nil {
		t.Fatalf("failed to query updated status: %v", err)
	}
	if updatedStatus != "missed" {
		t.Fatalf("expected heartbeat status to become 'missed', got '%s'", updatedStatus)
	}
}

