package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// TestIncidentLifecycle_Regression_TaskIncident004 executes Tests A through J
// as specified in TASK-INCIDENT-004.
func TestIncidentLifecycle_Regression_TaskIncident004(t *testing.T) {
	ctx := context.Background()
	adminPool, appPool, _ := setupTestRoles(t)

	incidentRepo := NewIncidentRepository(adminPool)
	heartbeatRepo := NewHeartbeatRepository(adminPool)
	monitorRepo := NewMonitorRepository(adminPool)
	checkResultRepo := NewCheckResultRepository(adminPool)
	userRepo := NewUserRepository(adminPool)

	// Helper to create test user
	createTestUser := func(email, name string) *User {
		u, err := userRepo.CreateUser(ctx, email, "hashpass123", name)
		if err != nil {
			t.Fatalf("failed to create user %s: %v", email, err)
		}
		return u
	}

	// Helper to create test monitor
	createTestMonitor := func(orgID, clientID string, name string) *Monitor {
		var assetID string
		err := adminPool.QueryRow(ctx, `
			INSERT INTO assets (organization_id, client_id, name, type, url)
			VALUES ($1, $2, $3, 'website', 'https://example.com/health')
			RETURNING id
		`, orgID, clientID, name+" Asset").Scan(&assetID)
		if err != nil {
			t.Fatalf("failed creating test asset: %v", err)
		}

		mon, err := monitorRepo.CreateMonitor(ctx, orgID, CreateMonitorParams{
			AssetID:           assetID,
			Method:            "GET",
			TimeoutSeconds:    5,
			IntervalSeconds:   60,
			ExpectedStatusMin: 200,
			ExpectedStatusMax: 299,
		})
		if err != nil {
			t.Fatalf("failed creating test monitor: %v", err)
		}
		return mon
	}

	createMonitorIncident := func(orgID, monID string, firstFail, confAt time.Time, cat, summary string) *Incident {
		var inc *Incident
		err := WithTenantTx(ctx, adminPool, orgID, func(tx pgx.Tx) error {
			var err error
			inc, _, err = incidentRepo.CreateMonitorIncidentTx(ctx, tx, orgID, monID, firstFail, confAt, nil, cat, summary)
			return err
		})
		if err != nil {
			t.Fatalf("failed to create test monitor incident: %v", err)
		}
		return inc
	}

	createHeartbeatIncident := func(orgID, hbID string, firstFail, confAt time.Time, cat, summary string) *Incident {
		var inc *Incident
		err := WithTenantTx(ctx, adminPool, orgID, func(tx pgx.Tx) error {
			var err error
			inc, _, err = incidentRepo.CreateHeartbeatIncidentTx(ctx, tx, orgID, hbID, firstFail, confAt, cat, summary)
			return err
		})
		if err != nil {
			t.Fatalf("failed to create test heartbeat incident: %v", err)
		}
		return inc
	}

	// =========================================================================
	// Test A: Acknowledge Semantics
	// =========================================================================
	t.Run("TestA_AcknowledgeSemantics", func(t *testing.T) {
		orgID, clientID := createTestOrgAndClient(t, adminPool, "Org Ack Test", fmt.Sprintf("org-ack-%d", time.Now().UnixNano()))
		mon := createTestMonitor(orgID, clientID, "Mon Ack")
		operator := createTestUser(fmt.Sprintf("op-ack-%d@test.local", time.Now().UnixNano()), "Operator Ack")

		// 1. Create open incident
		now := time.Now().UTC()
		openInc := createMonitorIncident(orgID, mon.ID, now.Add(-5*time.Minute), now, "http_failure", "Server 500 error")

		// Check initial dashboard stats
		initialStats, err := checkResultRepo.GetDashboardStats(ctx, orgID)
		if err != nil {
			t.Fatalf("failed getting dashboard stats: %v", err)
		}
		if initialStats.OpenIncidents != 1 || initialStats.UnacknowledgedIncidents != 1 || initialStats.AcknowledgedIncidents != 0 {
			t.Fatalf("unexpected initial counts: open=%d, unack=%d, ack=%d",
				initialStats.OpenIncidents, initialStats.UnacknowledgedIncidents, initialStats.AcknowledgedIncidents)
		}

		// 2. Acknowledge it
		ackedInc, err := incidentRepo.AcknowledgeIncident(ctx, orgID, openInc.ID, operator.ID)
		if err != nil {
			t.Fatalf("failed to acknowledge incident: %v", err)
		}

		// 3. Verify status remains open
		if ackedInc.Status != "open" {
			t.Fatalf("expected status 'open', got %s", ackedInc.Status)
		}

		// 4. Verify acknowledged_at and acknowledged_by
		if ackedInc.AcknowledgedAt == nil || ackedInc.AcknowledgedBy == nil || *ackedInc.AcknowledgedBy != operator.ID {
			t.Fatalf("acknowledged metadata mismatch: at=%v, by=%v", ackedInc.AcknowledgedAt, ackedInc.AcknowledgedBy)
		}

		// 5. Verify it remains included in Open Incident count
		// 6. Verify it is excluded from Unacknowledged count
		postAckStats, err := checkResultRepo.GetDashboardStats(ctx, orgID)
		if err != nil {
			t.Fatalf("failed getting post-ack stats: %v", err)
		}
		if postAckStats.OpenIncidents != 1 {
			t.Fatalf("expected OpenIncidents = 1, got %d", postAckStats.OpenIncidents)
		}
		if postAckStats.UnacknowledgedIncidents != 0 {
			t.Fatalf("expected UnacknowledgedIncidents = 0, got %d", postAckStats.UnacknowledgedIncidents)
		}
		if postAckStats.AcknowledgedIncidents != 1 {
			t.Fatalf("expected AcknowledgedIncidents = 1, got %d", postAckStats.AcknowledgedIncidents)
		}

		// 7. Repeat acknowledgement -> idempotency check (ErrIncidentAlreadyAcked, no timestamp change)
		origAckAt := *ackedInc.AcknowledgedAt
		_, err = incidentRepo.AcknowledgeIncident(ctx, orgID, openInc.ID, operator.ID)
		if !errors.Is(err, ErrIncidentAlreadyAcked) {
			t.Fatalf("expected ErrIncidentAlreadyAcked on repeat ack, got %v", err)
		}

		// Verify timestamp was not altered
		reloaded, err := incidentRepo.GetIncidentByID(ctx, orgID, openInc.ID)
		if err != nil {
			t.Fatalf("failed getting reloaded incident: %v", err)
		}
		if !reloaded.AcknowledgedAt.Equal(origAckAt) {
			t.Fatalf("expected acknowledged_at unchanged, was %v, now %v", origAckAt, reloaded.AcknowledgedAt)
		}

		// Verify audit entry for acknowledgment
		var auditCount int
		err = adminPool.QueryRow(ctx, `
			SELECT count(*) FROM audit_logs
			WHERE organization_id = $1 AND entity_id = $2 AND action = 'incident.acknowledged'
		`, orgID, openInc.ID).Scan(&auditCount)
		if err != nil || auditCount != 1 {
			t.Fatalf("expected exactly 1 audit log entry for acknowledgement, got %d (err: %v)", auditCount, err)
		}
	})

	// =========================================================================
	// Test B: Manual Resolution
	// =========================================================================
	t.Run("TestB_ManualResolution", func(t *testing.T) {
		orgID, clientID := createTestOrgAndClient(t, adminPool, "Org Man Res", fmt.Sprintf("org-man-%d", time.Now().UnixNano()))
		mon := createTestMonitor(orgID, clientID, "Mon Man")
		operator := createTestUser(fmt.Sprintf("op-man-%d@test.local", time.Now().UnixNano()), "Operator Man")

		now := time.Now().UTC()
		openInc := createMonitorIncident(orgID, mon.ID, now.Add(-10*time.Minute), now, "http_failure", "Server 502")

		// Manually resolve incident
		note := "Operator restarted service manually after verifying logs"
		resolvedInc, err := incidentRepo.ResolveIncidentManually(ctx, orgID, openInc.ID, operator.ID, &note)
		if err != nil {
			t.Fatalf("failed manual resolution: %v", err)
		}

		// 3. Verify status becomes resolved
		if resolvedInc.Status != "resolved" {
			t.Fatalf("expected status 'resolved', got %s", resolvedInc.Status)
		}

		// 4. Verify actor and timestamp
		if resolvedInc.ResolvedAt == nil {
			t.Fatalf("expected resolved_at to be set")
		}
		if resolvedInc.ResolvedBy == nil || *resolvedInc.ResolvedBy != operator.ID {
			t.Fatalf("expected resolved_by %s, got %v", operator.ID, resolvedInc.ResolvedBy)
		}

		// 5. Verify resolution source is manual
		if resolvedInc.ResolutionType == nil || *resolvedInc.ResolutionType != "manual" {
			t.Fatalf("expected resolution_type 'manual', got %v", resolvedInc.ResolutionType)
		}
		if resolvedInc.ResolutionNote == nil || *resolvedInc.ResolutionNote != note {
			t.Fatalf("expected resolution_note %q, got %v", note, resolvedInc.ResolutionNote)
		}

		// 6. Verify audit log
		var auditCount int
		err = adminPool.QueryRow(ctx, `
			SELECT count(*) FROM audit_logs
			WHERE organization_id = $1 AND entity_id = $2 AND action = 'incident.resolved' AND actor_user_id = $3
		`, orgID, openInc.ID, operator.ID).Scan(&auditCount)
		if err != nil || auditCount != 1 {
			t.Fatalf("expected 1 manual resolution audit entry with actor, got %d", auditCount)
		}

		// 7. Verify incident history remains accessible
		historyInc, err := incidentRepo.GetIncidentByID(ctx, orgID, openInc.ID)
		if err != nil {
			t.Fatalf("failed fetching historical incident: %v", err)
		}
		if historyInc.Status != "resolved" || historyInc.ResolutionType == nil || *historyInc.ResolutionType != "manual" {
			t.Fatalf("historical incident metadata corrupt: %+v", historyInc)
		}

		// Repeat manual resolution -> must return ErrIncidentAlreadyResolved
		_, err = incidentRepo.ResolveIncidentManually(ctx, orgID, openInc.ID, operator.ID, nil)
		if !errors.Is(err, ErrIncidentAlreadyResolved) {
			t.Fatalf("expected ErrIncidentAlreadyResolved on second resolution, got %v", err)
		}

		// Attempt acknowledgement on resolved incident -> must return ErrIncidentAlreadyResolved
		_, err = incidentRepo.AcknowledgeIncident(ctx, orgID, openInc.ID, operator.ID)
		if !errors.Is(err, ErrIncidentAlreadyResolved) {
			t.Fatalf("expected ErrIncidentAlreadyResolved on acknowledging resolved incident, got %v", err)
		}
	})

	// =========================================================================
	// Test C: Automatic Website Recovery
	// =========================================================================
	t.Run("TestC_AutomaticWebsiteRecovery", func(t *testing.T) {
		orgID, clientID := createTestOrgAndClient(t, adminPool, "Org Web Rec", fmt.Sprintf("org-rec-%d", time.Now().UnixNano()))
		mon := createTestMonitor(orgID, clientID, "Mon Rec")

		// 1. Confirm monitor is down and create confirmed outage incident
		tFail1 := time.Now().UTC().Add(-2 * time.Minute)
		_, _, _ = monitorRepo.RecordScheduledCheckOutcomeWithResult(ctx, ScheduledCheckOutcomeParams{
			OrganizationID:  orgID,
			MonitorID:       mon.ID,
			CheckSuccess:    false,
			CompletedAt:     tFail1,
			ScheduledAt:     tFail1,
			IntervalSeconds: 60,
		})

		tFail2 := time.Now().UTC().Add(-1 * time.Minute)
		st2, incOpened, err := monitorRepo.RecordScheduledCheckOutcomeWithResult(ctx, ScheduledCheckOutcomeParams{
			OrganizationID:  orgID,
			MonitorID:       mon.ID,
			CheckSuccess:    false,
			CompletedAt:     tFail2,
			ScheduledAt:     tFail2,
			IntervalSeconds: 60,
		})
		if err != nil || incOpened == nil || st2 != "down" {
			t.Fatalf("expected monitor down with open incident, got status=%s, inc=%v, err=%v", st2, incOpened, err)
		}

		// 2. Successful recovery check
		tRec := time.Now().UTC().Truncate(time.Microsecond)
		var checkResultID string
		// Insert dummy check_result row for foreign key
		err = adminPool.QueryRow(ctx, `
			INSERT INTO check_results (organization_id, monitor_id, scheduled_at, started_at, completed_at, latency_ms, result, http_status_code, worker_id)
			VALUES ($1, $2, $3, $3, $3, 120, 'success', 200, 'worker-test')
			RETURNING id
		`, orgID, mon.ID, tRec).Scan(&checkResultID)
		if err != nil {
			t.Fatalf("failed inserting recovery check result: %v", err)
		}

		stRec, incResolved, err := monitorRepo.RecordScheduledCheckOutcomeWithResult(ctx, ScheduledCheckOutcomeParams{
			OrganizationID:  orgID,
			MonitorID:       mon.ID,
			CheckSuccess:    true,
			CompletedAt:     tRec,
			ScheduledAt:     tRec,
			IntervalSeconds: 60,
			CheckResultID:   &checkResultID,
		})
		if err != nil {
			t.Fatalf("failed recording recovery outcome: %v", err)
		}
		if stRec != "up" {
			t.Fatalf("expected monitor status 'up', got %s", stRec)
		}

		// 3. Verify incident resolves exactly once with automatic_recovery
		if incResolved == nil || incResolved.Status != "resolved" {
			t.Fatalf("expected resolved incident, got %v", incResolved)
		}
		if incResolved.ResolutionType == nil || *incResolved.ResolutionType != "automatic_recovery" {
			t.Fatalf("expected resolution_type 'automatic_recovery', got %v", incResolved.ResolutionType)
		}
		if incResolved.ResolvedBy != nil {
			t.Fatalf("expected resolved_by to be nil for system recovery, got %v", incResolved.ResolvedBy)
		}

		// 4. Verify failure and recovery timestamps
		if !incResolved.ResolvedAt.Equal(tRec) {
			t.Fatalf("expected resolved_at %v, got %v", tRec, incResolved.ResolvedAt)
		}

		// 5. Subsequent check passes without error or duplicate incident creation
		tRec2 := tRec.Add(60 * time.Second)
		stRec2, incRec2, err := monitorRepo.RecordScheduledCheckOutcomeWithResult(ctx, ScheduledCheckOutcomeParams{
			OrganizationID:  orgID,
			MonitorID:       mon.ID,
			CheckSuccess:    true,
			CompletedAt:     tRec2,
			ScheduledAt:     tRec2,
			IntervalSeconds: 60,
		})
		if err != nil || stRec2 != "up" || incRec2 != nil {
			t.Fatalf("expected clean subsequent success: status=%s, inc=%v, err=%v", stRec2, incRec2, err)
		}

		// Check system-attributed audit log
		var auditCount int
		err = adminPool.QueryRow(ctx, `
			SELECT count(*) FROM audit_logs
			WHERE organization_id = $1 AND entity_id = $2 AND action = 'incident.resolved' AND actor_user_id IS NULL
		`, orgID, incOpened.ID).Scan(&auditCount)
		if err != nil || auditCount != 1 {
			t.Fatalf("expected 1 system-attributed audit log for automatic website recovery, got %d", auditCount)
		}
	})

	// =========================================================================
	// Test D: Automatic Heartbeat Recovery
	// =========================================================================
	t.Run("TestD_AutomaticHeartbeatRecovery", func(t *testing.T) {
		orgID, clientID := createTestOrgAndClient(t, adminPool, "Org HB Rec", fmt.Sprintf("org-hb-%d", time.Now().UnixNano()))

		// Create heartbeat 1
		hb1, token1, err := heartbeatRepo.CreateHeartbeat(ctx, orgID, CreateHeartbeatParams{
			ClientID:                clientID,
			Name:                    "HB Recovery Target",
			ExpectedIntervalSeconds: 300,
			GracePeriodSeconds:      60,
		})
		if err != nil {
			t.Fatalf("failed creating heartbeat: %v", err)
		}

		// Create heartbeat 2 (unrelated)
		hb2, _, err := heartbeatRepo.CreateHeartbeat(ctx, orgID, CreateHeartbeatParams{
			ClientID:                clientID,
			Name:                    "HB Unrelated",
			ExpectedIntervalSeconds: 300,
			GracePeriodSeconds:      60,
		})
		if err != nil {
			t.Fatalf("failed creating heartbeat 2: %v", err)
		}

		// Create missed incident for HB 1
		now := time.Now().UTC()
		inc1 := createHeartbeatIncident(orgID, hb1.ID, now.Add(-5*time.Minute), now, "heartbeat_missed", "Missed ping")

		// Create missed incident for HB 2
		inc2 := createHeartbeatIncident(orgID, hb2.ID, now.Add(-5*time.Minute), now, "heartbeat_missed", "Missed ping 2")

		// 6. Send invalid heartbeat token -> must return error and not resolve any incident
		_, _, err = heartbeatRepo.RecordHeartbeatPing(ctx, "invalid-token-1234567890123456")
		if !errors.Is(err, ErrHeartbeatNotFound) {
			t.Fatalf("expected ErrHeartbeatNotFound for invalid token, got %v", err)
		}

		// 2. Send valid ping for HB 1
		hb1Pinged, resolvedInc1, err := heartbeatRepo.RecordHeartbeatPing(ctx, token1)
		if err != nil {
			t.Fatalf("failed recording valid ping: %v", err)
		}
		if hb1Pinged.Status != "healthy" {
			t.Fatalf("expected hb1 status 'healthy', got %s", hb1Pinged.Status)
		}

		// 4. Verify corresponding incident resolves with automatic_recovery
		if resolvedInc1 == nil || resolvedInc1.ID != inc1.ID {
			t.Fatalf("expected inc1 to be resolved, got %v", resolvedInc1)
		}
		if resolvedInc1.ResolutionType == nil || *resolvedInc1.ResolutionType != "automatic_recovery" {
			t.Fatalf("expected resolution_type 'automatic_recovery', got %v", resolvedInc1.ResolutionType)
		}
		if resolvedInc1.ResolvedBy != nil {
			t.Fatalf("expected resolved_by nil for system recovery, got %v", resolvedInc1.ResolvedBy)
		}

		// 5. Verify unrelated heartbeat monitor's incident remains open
		inc2Reloaded, err := incidentRepo.GetIncidentByID(ctx, orgID, inc2.ID)
		if err != nil {
			t.Fatalf("failed reloading inc2: %v", err)
		}
		if inc2Reloaded.Status != "open" {
			t.Fatalf("unrelated incident must remain open, got %s", inc2Reloaded.Status)
		}
	})

	// =========================================================================
	// Test E: Heartbeat Monitor Deletion
	// =========================================================================
	t.Run("TestE_HeartbeatMonitorDeletion", func(t *testing.T) {
		orgID, clientID := createTestOrgAndClient(t, adminPool, "Org HB Del", fmt.Sprintf("org-del-%d", time.Now().UnixNano()))
		operator := createTestUser(fmt.Sprintf("op-del-%d@test.local", time.Now().UnixNano()), "Operator Del")

		// 1. Create Heartbeat Monitor
		hb, _, err := heartbeatRepo.CreateHeartbeat(ctx, orgID, CreateHeartbeatParams{
			ClientID:                clientID,
			Name:                    "HB To Be Deleted",
			ExpectedIntervalSeconds: 300,
			GracePeriodSeconds:      60,
		})
		if err != nil {
			t.Fatalf("failed creating heartbeat: %v", err)
		}

		// 2. Create associated open incident
		now := time.Now().UTC()
		openInc := createHeartbeatIncident(orgID, hb.ID, now.Add(-10*time.Minute), now, "heartbeat_missed", "Ping not received")

		// 3. Acknowledge the incident
		ackedInc, err := incidentRepo.AcknowledgeIncident(ctx, orgID, openInc.ID, operator.ID)
		if err != nil {
			t.Fatalf("failed acknowledging incident: %v", err)
		}

		// Check stats before deletion
		statsBefore, err := checkResultRepo.GetDashboardStats(ctx, orgID)
		if err != nil {
			t.Fatalf("failed getting stats before: %v", err)
		}
		if statsBefore.OpenIncidents != 1 || statsBefore.ResolvedIncidents != 0 {
			t.Fatalf("stats before: open=%d, res=%d", statsBefore.OpenIncidents, statsBefore.ResolvedIncidents)
		}

		// 4. Delete the Heartbeat Monitor as authorized user
		err = heartbeatRepo.ArchiveHeartbeatWithActor(ctx, orgID, hb.ID, operator.ID)
		if err != nil {
			t.Fatalf("failed archiving heartbeat: %v", err)
		}

		// 5. Verify monitor is archived
		reloadedHB, err := heartbeatRepo.GetHeartbeatByID(ctx, orgID, hb.ID)
		if err != nil {
			t.Fatalf("failed getting heartbeat: %v", err)
		}
		if reloadedHB.ArchivedAt == nil {
			t.Fatalf("expected archived_at to be set on heartbeat")
		}

		// 6. Verify Incident is retained
		// 7. Verify status becomes resolved
		// 8. Verify resolution type is source_monitor_deleted
		reloadedInc, err := incidentRepo.GetIncidentByID(ctx, orgID, openInc.ID)
		if err != nil {
			t.Fatalf("failed getting incident after deletion: %v", err)
		}
		if reloadedInc.Status != "resolved" {
			t.Fatalf("expected incident status 'resolved', got %s", reloadedInc.Status)
		}
		if reloadedInc.ResolutionType == nil || *reloadedInc.ResolutionType != "source_monitor_deleted" {
			t.Fatalf("expected resolution_type 'source_monitor_deleted', got %v", reloadedInc.ResolutionType)
		}

		// 9. Verify acknowledgement history remains
		if reloadedInc.AcknowledgedAt == nil || reloadedInc.AcknowledgedBy == nil || *reloadedInc.AcknowledgedBy != operator.ID {
			t.Fatalf("expected acknowledgement preserved: at=%v, by=%v", reloadedInc.AcknowledgedAt, reloadedInc.AcknowledgedBy)
		}
		if !reloadedInc.AcknowledgedAt.Equal(*ackedInc.AcknowledgedAt) {
			t.Fatalf("acknowledgement timestamp altered: was %v, now %v", ackedInc.AcknowledgedAt, reloadedInc.AcknowledgedAt)
		}

		// 10. Verify correct deletion actor and timestamp
		if reloadedInc.ResolvedBy == nil || *reloadedInc.ResolvedBy != operator.ID {
			t.Fatalf("expected resolved_by %s, got %v", operator.ID, reloadedInc.ResolvedBy)
		}
		if reloadedInc.ResolvedAt == nil {
			t.Fatalf("expected resolved_at to be populated")
		}

		// 11. Verify no automatic_recovery metadata created
		if reloadedInc.RecoveryResultID != nil {
			t.Fatalf("expected recovery_result_id nil, got %v", reloadedInc.RecoveryResultID)
		}

		// 12. Verify incident detail lookup still works without active heartbeat monitor
		if reloadedInc.TargetName != "HB To Be Deleted" {
			t.Fatalf("expected target name preserved, got %q", reloadedInc.TargetName)
		}
		if reloadedInc.HeartbeatArchivedAt == nil {
			t.Fatalf("expected HeartbeatArchivedAt to be populated in joined metadata")
		}

		// 13. Verify Open Incident count decreases
		// 14. Verify Resolved Incident count increases
		statsAfter, err := checkResultRepo.GetDashboardStats(ctx, orgID)
		if err != nil {
			t.Fatalf("failed getting stats after: %v", err)
		}
		if statsAfter.OpenIncidents != 0 {
			t.Fatalf("expected OpenIncidents = 0, got %d", statsAfter.OpenIncidents)
		}
		if statsAfter.ResolvedIncidents != 1 {
			t.Fatalf("expected ResolvedIncidents = 1, got %d", statsAfter.ResolvedIncidents)
		}
	})

	// =========================================================================
	// Test F: Multiple Incidents Handling
	// =========================================================================
	t.Run("TestF_MultipleIncidentsHandling", func(t *testing.T) {
		orgID, clientID := createTestOrgAndClient(t, adminPool, "Org Multi", fmt.Sprintf("org-multi-%d", time.Now().UnixNano()))
		operator := createTestUser(fmt.Sprintf("op-multi-%d@test.local", time.Now().UnixNano()), "Operator Multi")

		hb, _, err := heartbeatRepo.CreateHeartbeat(ctx, orgID, CreateHeartbeatParams{
			ClientID:                clientID,
			Name:                    "HB Multi Incidents",
			ExpectedIntervalSeconds: 300,
			GracePeriodSeconds:      60,
		})
		if err != nil {
			t.Fatalf("failed creating heartbeat: %v", err)
		}

		// Create historical resolved incident first
		past := time.Now().UTC().Add(-2 * time.Hour)
		oldInc := createHeartbeatIncident(orgID, hb.ID, past.Add(-30*time.Minute), past, "heartbeat_missed", "Old missed")
		var oldResolved *Incident
		err = WithTenantTx(ctx, adminPool, orgID, func(tx pgx.Tx) error {
			var err error
			oldResolved, err = incidentRepo.ResolveIncidentTx(ctx, tx, orgID, oldInc.ID, past.Add(10*time.Minute), nil)
			return err
		})
		if err != nil {
			t.Fatalf("failed resolving old incident: %v", err)
		}

		// Now create open incident
		now := time.Now().UTC()
		currInc := createHeartbeatIncident(orgID, hb.ID, now.Add(-15*time.Minute), now, "heartbeat_missed", "Current missed")

		// Delete heartbeat
		err = heartbeatRepo.ArchiveHeartbeatWithActor(ctx, orgID, hb.ID, operator.ID)
		if err != nil {
			t.Fatalf("failed archiving heartbeat: %v", err)
		}

		// Verify current incident was resolved as source_monitor_deleted
		currReloaded, _ := incidentRepo.GetIncidentByID(ctx, orgID, currInc.ID)
		if currReloaded.Status != "resolved" || currReloaded.ResolutionType == nil || *currReloaded.ResolutionType != "source_monitor_deleted" {
			t.Fatalf("current incident not properly resolved: %+v", currReloaded)
		}

		// Verify historical incident was NOT altered
		oldReloaded, _ := incidentRepo.GetIncidentByID(ctx, orgID, oldInc.ID)
		if oldReloaded.Status != "resolved" || oldReloaded.ResolutionType == nil || *oldReloaded.ResolutionType != "automatic_recovery" {
			t.Fatalf("old resolved incident altered: was %+v, now %+v", oldResolved, oldReloaded)
		}
	})

	// =========================================================================
	// Test G: Repeated and Concurrent Deletion
	// =========================================================================
	t.Run("TestG_RepeatedAndConcurrentDeletion", func(t *testing.T) {
		orgID, clientID := createTestOrgAndClient(t, adminPool, "Org Concurr", fmt.Sprintf("org-conc-%d", time.Now().UnixNano()))
		operator := createTestUser(fmt.Sprintf("op-conc-%d@test.local", time.Now().UnixNano()), "Operator Conc")

		hb, _, err := heartbeatRepo.CreateHeartbeat(ctx, orgID, CreateHeartbeatParams{
			ClientID:                clientID,
			Name:                    "HB Concurrent Del",
			ExpectedIntervalSeconds: 300,
			GracePeriodSeconds:      60,
		})
		if err != nil {
			t.Fatalf("failed creating heartbeat: %v", err)
		}

		now := time.Now().UTC()
		_ = createHeartbeatIncident(orgID, hb.ID, now.Add(-10*time.Minute), now, "heartbeat_missed", "Missed")

		// Launch 5 concurrent deletion attempts
		const concurrency = 5
		var wg sync.WaitGroup
		errs := make([]error, concurrency)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				errs[idx] = heartbeatRepo.ArchiveHeartbeatWithActor(ctx, orgID, hb.ID, operator.ID)
			}(i)
		}
		wg.Wait()

		successCount := 0
		notFoundCount := 0
		for _, e := range errs {
			if e == nil {
				successCount++
			} else if errors.Is(e, ErrHeartbeatNotFound) {
				notFoundCount++
			} else {
				t.Errorf("unexpected error in concurrent deletion: %v", e)
			}
		}

		// Exactly 1 must succeed, others receive ErrHeartbeatNotFound (idempotent 404)
		if successCount != 1 {
			t.Fatalf("expected exactly 1 success, got %d (notFound=%d)", successCount, notFoundCount)
		}

		// Verify only 1 heartbeat.deleted audit record was created
		var hbAuditCount int
		_ = adminPool.QueryRow(ctx, "SELECT count(*) FROM audit_logs WHERE organization_id = $1 AND entity_id = $2 AND action = 'heartbeat.deleted'", orgID, hb.ID).Scan(&hbAuditCount)
		if hbAuditCount != 1 {
			t.Fatalf("expected 1 heartbeat.deleted audit entry, got %d", hbAuditCount)
		}
	})

	// =========================================================================
	// Test H: RBAC and Cross-Tenant Isolation
	// =========================================================================
	t.Run("TestH_CrossTenantIsolation", func(t *testing.T) {
		orgA, clientA := createTestOrgAndClient(t, adminPool, "Org A", fmt.Sprintf("org-a-%d", time.Now().UnixNano()))
		orgB, clientB := createTestOrgAndClient(t, adminPool, "Org B", fmt.Sprintf("org-b-%d", time.Now().UnixNano()))
		userA := createTestUser(fmt.Sprintf("user-a-%d@test.local", time.Now().UnixNano()), "User A")

		hbA, _, _ := heartbeatRepo.CreateHeartbeat(ctx, orgA, CreateHeartbeatParams{
			ClientID: clientA, Name: "HB Org A", ExpectedIntervalSeconds: 300, GracePeriodSeconds: 60,
		})
		hbB, _, _ := heartbeatRepo.CreateHeartbeat(ctx, orgB, CreateHeartbeatParams{
			ClientID: clientB, Name: "HB Org B", ExpectedIntervalSeconds: 300, GracePeriodSeconds: 60,
		})

		now := time.Now().UTC()
		incA := createHeartbeatIncident(orgA, hbA.ID, now.Add(-10*time.Minute), now, "heartbeat_missed", "Missed A")
		incB := createHeartbeatIncident(orgB, hbB.ID, now.Add(-10*time.Minute), now, "heartbeat_missed", "Missed B")

		// Org A user attempts to acknowledge Org B's incident using Org A tenant context -> must fail with ErrIncidentNotFound
		_, err := incidentRepo.AcknowledgeIncident(ctx, orgA, incB.ID, userA.ID)
		if !errors.Is(err, ErrIncidentNotFound) {
			t.Fatalf("SECURITY VIOLATION: Cross-tenant acknowledge succeeded or returned unexpected error: %v", err)
		}

		// Org A user attempts to manually resolve Org B's incident -> must fail with ErrIncidentNotFound
		_, err = incidentRepo.ResolveIncidentManually(ctx, orgA, incB.ID, userA.ID, nil)
		if !errors.Is(err, ErrIncidentNotFound) {
			t.Fatalf("SECURITY VIOLATION: Cross-tenant manual resolve succeeded: %v", err)
		}

		// Deleting Org A's heartbeat must NOT affect Org B's incident
		_ = heartbeatRepo.ArchiveHeartbeatWithActor(ctx, orgA, hbA.ID, userA.ID)

		// Verify Inc A is resolved
		reloadedIncA, _ := incidentRepo.GetIncidentByID(ctx, orgA, incA.ID)
		if reloadedIncA.Status != "resolved" {
			t.Fatalf("expected incA to be resolved")
		}

		// Verify Inc B remains open
		reloadedIncB, _ := incidentRepo.GetIncidentByID(ctx, orgB, incB.ID)
		if reloadedIncB.Status != "open" {
			t.Fatalf("SECURITY VIOLATION: incB was resolved by deleting incA's monitor!")
		}

		// AppPool role attempting to read another tenant's incident without app.current_organization_id fails closed
		var count int
		_ = appPool.QueryRow(ctx, "SELECT count(*) FROM incidents").Scan(&count)
		if count != 0 {
			t.Fatalf("SECURITY VIOLATION: appPool read %d rows without tenant context set!", count)
		}
	})

	// =========================================================================
	// Test I: Dashboard Aggregates
	// =========================================================================
	t.Run("TestI_DashboardAggregates", func(t *testing.T) {
		orgID, clientID := createTestOrgAndClient(t, adminPool, "Org Aggregates", fmt.Sprintf("org-agg-%d", time.Now().UnixNano()))
		mon := createTestMonitor(orgID, clientID, "Mon Agg")
		operator := createTestUser(fmt.Sprintf("op-agg-%d@test.local", time.Now().UnixNano()), "Operator Agg")

		hb, _, _ := heartbeatRepo.CreateHeartbeat(ctx, orgID, CreateHeartbeatParams{
			ClientID: clientID, Name: "HB Agg", ExpectedIntervalSeconds: 300, GracePeriodSeconds: 60,
		})

		now := time.Now().UTC()

		// 1. Create open website incident
		webInc := createMonitorIncident(orgID, mon.ID, now.Add(-20*time.Minute), now, "http_failure", "500")

		// 2. Create open heartbeat incident
		_ = createHeartbeatIncident(orgID, hb.ID, now.Add(-10*time.Minute), now, "heartbeat_missed", "Missed ping")

		// 3. Acknowledge website incident
		_, _ = incidentRepo.AcknowledgeIncident(ctx, orgID, webInc.ID, operator.ID)

		stats, err := checkResultRepo.GetDashboardStats(ctx, orgID)
		if err != nil {
			t.Fatalf("failed getting stats: %v", err)
		}

		// Total open = 2 (1 web + 1 hb)
		if stats.OpenIncidents != 2 {
			t.Fatalf("expected OpenIncidents=2, got %d", stats.OpenIncidents)
		}
		// Acknowledged = 1 (web)
		if stats.AcknowledgedIncidents != 1 {
			t.Fatalf("expected AcknowledgedIncidents=1, got %d", stats.AcknowledgedIncidents)
		}
		// Unacknowledged = 1 (hb)
		if stats.UnacknowledgedIncidents != 1 {
			t.Fatalf("expected UnacknowledgedIncidents=1, got %d", stats.UnacknowledgedIncidents)
		}
		// Website outage incidents = 1
		if stats.WebsiteOutageIncidents != 1 {
			t.Fatalf("expected WebsiteOutageIncidents=1, got %d", stats.WebsiteOutageIncidents)
		}
		// Heartbeat missed incidents = 1
		if stats.HeartbeatMissedIncidents != 1 {
			t.Fatalf("expected HeartbeatMissedIncidents=1, got %d", stats.HeartbeatMissedIncidents)
		}

		// 4. Manually resolve website incident
		_, _ = incidentRepo.ResolveIncidentManually(ctx, orgID, webInc.ID, operator.ID, nil)

		// 5. Delete heartbeat monitor (resolves hb incident via source deletion)
		_ = heartbeatRepo.ArchiveHeartbeatWithActor(ctx, orgID, hb.ID, operator.ID)

		postStats, err := checkResultRepo.GetDashboardStats(ctx, orgID)
		if err != nil {
			t.Fatalf("failed getting post stats: %v", err)
		}
		if postStats.OpenIncidents != 0 {
			t.Fatalf("expected OpenIncidents=0, got %d", postStats.OpenIncidents)
		}
		if postStats.ResolvedIncidents != 2 {
			t.Fatalf("expected ResolvedIncidents=2, got %d", postStats.ResolvedIncidents)
		}
		if postStats.WebsiteOutageIncidents != 0 {
			t.Fatalf("expected WebsiteOutageIncidents=0, got %d", postStats.WebsiteOutageIncidents)
		}
		if postStats.HeartbeatMissedIncidents != 0 {
			t.Fatalf("expected HeartbeatMissedIncidents=0, got %d", postStats.HeartbeatMissedIncidents)
		}
	})

	// =========================================================================
	// Test J: Historical Compatibility
	// =========================================================================
	t.Run("TestJ_HistoricalCompatibility", func(t *testing.T) {
		orgID, clientID := createTestOrgAndClient(t, adminPool, "Org Legacy", fmt.Sprintf("org-leg-%d", time.Now().UnixNano()))
		mon := createTestMonitor(orgID, clientID, "Mon Legacy")

		// Insert legacy incident row directly with NULL resolution_type, NULL resolved_by, NULL resolution_note
		var legacyIncID string
		past := time.Now().UTC().Add(-48 * time.Hour)
		err := adminPool.QueryRow(ctx, `
			INSERT INTO incidents (
				organization_id, monitor_id, first_failure_at, confirmed_at, resolved_at,
				status, failure_category, notification_state, summary, duration_seconds
			)
			VALUES ($1, $2, $3, $3, $4, 'resolved', 'http_failure', 'delivered', 'Legacy resolved incident', 600)
			RETURNING id
		`, orgID, mon.ID, past, past.Add(10*time.Minute)).Scan(&legacyIncID)
		if err != nil {
			t.Fatalf("failed inserting legacy incident: %v", err)
		}

		// Read via GetIncidentByID
		legInc, err := incidentRepo.GetIncidentByID(ctx, orgID, legacyIncID)
		if err != nil {
			t.Fatalf("failed reading legacy incident: %v", err)
		}
		if legInc.Status != "resolved" {
			t.Fatalf("expected status 'resolved', got %s", legInc.Status)
		}
		if legInc.ResolutionType != nil {
			t.Fatalf("expected nil resolution_type for legacy incident, got %v", *legInc.ResolutionType)
		}
		if legInc.ResolvedBy != nil {
			t.Fatalf("expected nil resolved_by for legacy incident, got %v", *legInc.ResolvedBy)
		}

		// Read via ListIncidents
		list, total, err := incidentRepo.ListIncidents(ctx, orgID, IncidentFilter{Status: "resolved"})
		if err != nil || total < 1 || len(list) < 1 {
			t.Fatalf("failed listing legacy incident: list=%d, total=%d, err=%v", len(list), total, err)
		}
		if list[0].ID != legacyIncID {
			t.Fatalf("expected legacy incident in list, got %s", list[0].ID)
		}
	})
}
