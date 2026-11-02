package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"clientpulse/services/api/internal/testhelper"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSchedulerObservabilityAndDiagnostics(t *testing.T) {
	adminDB := testhelper.GetTestDB(t)
	if adminDB == nil {
		t.Skip("skipping integration test: database not reachable")
	}
	defer adminDB.Close()

	appDB := testhelper.GetAppTestDB(t)
	if appDB == nil {
		t.Skip("skipping integration test: app database not reachable")
	}
	defer appDB.Close()

	workerDBURL := "postgres://clientpulse_worker:clientpulse_worker_dev_password@localhost:5433/clientpulse?sslmode=disable"
	workerPool, err := pgxpool.New(context.Background(), workerDBURL)
	if err != nil {
		t.Fatalf("failed connecting as clientpulse_worker: %v", err)
	}
	defer workerPool.Close()

	ctx := context.Background()
	telemetryRepo := NewSchedulerTelemetryRepository(workerPool)

	// Helper to create tenant org and client
	createTenant := func(slug string) (string, string) {
		var orgID, clientID string
		err := adminDB.Pool.QueryRow(ctx, "INSERT INTO organizations (name, slug) VALUES ($1, $2) RETURNING id", "Org "+slug, slug).Scan(&orgID)
		if err != nil {
			t.Fatalf("failed creating org: %v", err)
		}
		err = adminDB.Pool.QueryRow(ctx, "INSERT INTO clients (organization_id, name) VALUES ($1, 'Client') RETURNING id", orgID).Scan(&clientID)
		if err != nil {
			t.Fatalf("failed creating client: %v", err)
		}
		return orgID, clientID
	}

	t.Run("TestCase1_TelemetryPersistenceAndMultipleWorkers", func(t *testing.T) {
		instA := fmt.Sprintf("worker-inst-A-%d", time.Now().UnixNano()%100000)
		instB := fmt.Sprintf("worker-inst-B-%d", time.Now().UnixNano()%100000)
		workerA := "worker-host-1"
		workerB := "worker-host-2"
		startedAt := time.Now().UTC()

		// Register instances
		err := telemetryRepo.RegisterInstance(ctx, instA, workerA, startedAt)
		if err != nil {
			t.Fatalf("failed to register instA: %v", err)
		}
		err = telemetryRepo.RegisterInstance(ctx, instB, workerB, startedAt)
		if err != nil {
			t.Fatalf("failed to register instB: %v", err)
		}

		// Record poll cycle for instA (success, 2 claimed)
		pollStart := time.Now().UTC()
		pollEnd := pollStart.Add(25 * time.Millisecond)
		err = telemetryRepo.RecordPollCycle(ctx, RecordCycleParams{
			InstanceID:           instA,
			WorkerID:             workerA,
			StartedAt:            startedAt,
			PollStartedAt:        pollStart,
			PollCompletedAt:      pollEnd,
			DurationMs:           25,
			DueMonitorsCount:     2,
			ClaimedMonitorsCount: 2,
			CompletedChecksCount: 2,
			FailedChecksCount:    0,
			IsSuccess:            true,
		}, true)
		if err != nil {
			t.Fatalf("failed recording cycle for instA: %v", err)
		}

		// Record poll cycle for instB (empty poll - 0 claimed is healthy!)
		err = telemetryRepo.RecordPollCycle(ctx, RecordCycleParams{
			InstanceID:           instB,
			WorkerID:             workerB,
			StartedAt:            startedAt,
			PollStartedAt:        pollStart,
			PollCompletedAt:      pollEnd,
			DurationMs:           10,
			DueMonitorsCount:     0,
			ClaimedMonitorsCount: 0,
			CompletedChecksCount: 0,
			FailedChecksCount:    0,
			IsSuccess:            true,
		}, false)
		if err != nil {
			t.Fatalf("failed recording cycle for instB: %v", err)
		}

		// Verify instA state
		var dueA, claimedA, errCountA int
		err = workerPool.QueryRow(ctx, "SELECT due_monitors_count, claimed_monitors_count, consecutive_errors FROM scheduler_instances WHERE instance_id = $1", instA).Scan(&dueA, &claimedA, &errCountA)
		if err != nil || dueA != 2 || claimedA != 2 || errCountA != 0 {
			t.Fatalf("unexpected state for instA: due=%d, claimed=%d, errs=%d", dueA, claimedA, errCountA)
		}

		// Verify instB state (0 claimed, 0 consecutive errors)
		var dueB, claimedB, errCountB int
		err = workerPool.QueryRow(ctx, "SELECT due_monitors_count, claimed_monitors_count, consecutive_errors FROM scheduler_instances WHERE instance_id = $1", instB).Scan(&dueB, &claimedB, &errCountB)
		if err != nil || dueB != 0 || claimedB != 0 || errCountB != 0 {
			t.Fatalf("unexpected state for instB: due=%d, claimed=%d, errs=%d", dueB, claimedB, errCountB)
		}
	})

	t.Run("TestCase2_FailuresAndRecoveryState", func(t *testing.T) {
		instID := fmt.Sprintf("worker-fail-rec-%d", time.Now().UnixNano()%100000)
		workerID := "worker-fail-test"
		startedAt := time.Now().UTC()

		_ = telemetryRepo.RegisterInstance(ctx, instID, workerID, startedAt)

		// Record 3 consecutive failed cycles
		for i := 1; i <= 3; i++ {
			errCat := "db_connection_timeout"
			errMsg := "dial timeout"
			err := telemetryRepo.RecordPollCycle(ctx, RecordCycleParams{
				InstanceID:        instID,
				WorkerID:          workerID,
				StartedAt:         startedAt,
				PollStartedAt:     time.Now().UTC(),
				PollCompletedAt:   time.Now().UTC(),
				DurationMs:        5000,
				ErrorCategory:     &errCat,
				ErrorMessage:      &errMsg,
				IsSuccess:         false,
			}, true)
			if err != nil {
				t.Fatalf("failed recording failed cycle %d: %v", i, err)
			}
		}

		// Verify consecutive_errors = 3
		var consecutiveErrors int
		err := workerPool.QueryRow(ctx, "SELECT consecutive_errors FROM scheduler_instances WHERE instance_id = $1", instID).Scan(&consecutiveErrors)
		if err != nil || consecutiveErrors != 3 {
			t.Fatalf("expected 3 consecutive errors, got %d (err: %v)", consecutiveErrors, err)
		}

		// Now record a successful recovery cycle
		err = telemetryRepo.RecordPollCycle(ctx, RecordCycleParams{
			InstanceID:           instID,
			WorkerID:             workerID,
			StartedAt:            startedAt,
			PollStartedAt:        time.Now().UTC(),
			PollCompletedAt:      time.Now().UTC(),
			DurationMs:           12,
			DueMonitorsCount:     1,
			ClaimedMonitorsCount: 1,
			IsSuccess:            true,
		}, true)
		if err != nil {
			t.Fatalf("failed recording recovery cycle: %v", err)
		}

		// Verify consecutive_errors reset to 0
		err = workerPool.QueryRow(ctx, "SELECT consecutive_errors FROM scheduler_instances WHERE instance_id = $1", instID).Scan(&consecutiveErrors)
		if err != nil || consecutiveErrors != 0 {
			t.Fatalf("expected 0 consecutive errors after recovery, got %d", consecutiveErrors)
		}
	})

	t.Run("TestCase3_SilentZeroClaimMismatchDetection", func(t *testing.T) {
		// Create a test organization with an active asset and an active due monitor
		slug := fmt.Sprintf("mismatch-test-%d", time.Now().UnixNano()%100000)
		orgID, clientID := createTenant(slug)
		defer adminDB.Pool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgID)

		// 1. Create an asset
		var assetID string
		err := adminDB.Pool.QueryRow(ctx, `
			INSERT INTO assets (organization_id, client_id, name, url, is_active)
			VALUES ($1, $2, 'Mismatch Asset', 'https://mismatch.example.com', true)
			RETURNING id
		`, orgID, clientID).Scan(&assetID)
		if err != nil {
			t.Fatalf("failed creating asset: %v", err)
		}

		// 2. Create a due monitor
		var monitorID string
		err = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (
				organization_id, asset_id, interval_seconds, timeout_seconds,
				status, is_active, next_check_at
			)
			VALUES ($1, $2, 60, 10, 'pending', true, NOW() - INTERVAL '10 seconds')
			RETURNING id
		`, orgID, assetID).Scan(&monitorID)
		if err != nil {
			t.Fatalf("failed creating monitor: %v", err)
		}

		// With active asset:
		diag, err := telemetryRepo.DiagnoseSchedulingState(ctx, 60)
		if err != nil {
			t.Fatalf("DiagnoseSchedulingState failed: %v", err)
		}
		t.Logf("Baseline diagnosis: total=%d, eligible=%d, leased=%d, mismatch=%d",
			diag.TotalCandidates, diag.EligibleJoined, diag.LeasedCount, diag.MismatchCandidates)

		if diag.MismatchCandidates > 0 {
			t.Fatalf("expected 0 mismatch candidates when asset is active and joined, got %d", diag.MismatchCandidates)
		}

		// Now deactivate the asset (or simulate asset missing from join)
		_, err = adminDB.Pool.Exec(ctx, "UPDATE assets SET is_active = false WHERE id = $1", assetID)
		if err != nil {
			t.Fatalf("failed updating asset is_active: %v", err)
		}

		// Now due candidate exists, but asset is inactive -> Mismatch detected!
		diagMismatch, err := telemetryRepo.DiagnoseSchedulingState(ctx, 60)
		if err != nil {
			t.Fatalf("DiagnoseSchedulingState failed: %v", err)
		}
		t.Logf("After asset deactivated: total=%d, eligible=%d, leased=%d, mismatch=%d",
			diagMismatch.TotalCandidates, diagMismatch.EligibleJoined, diagMismatch.LeasedCount, diagMismatch.MismatchCandidates)

		if diagMismatch.MismatchCandidates < 1 {
			t.Fatalf("expected at least 1 mismatch candidate detected, got %d", diagMismatch.MismatchCandidates)
		}

		// Restore asset active
		_, _ = adminDB.Pool.Exec(ctx, "UPDATE assets SET is_active = true WHERE id = $1", assetID)
	})

	t.Run("TestCase4_StaleMonitorDetectionRules", func(t *testing.T) {
		slug := fmt.Sprintf("stale-rules-%d", time.Now().UnixNano()%100000)
		orgID, clientID := createTenant(slug)
		defer adminDB.Pool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgID)

		var assetID string
		_ = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO assets (organization_id, client_id, name, url, is_active)
			VALUES ($1, $2, 'Stale Asset', 'https://stale.example.com', true)
			RETURNING id
		`, orgID, clientID).Scan(&assetID)

		// 1. Disabled monitor -> should NOT be stale
		var disabledMonID string
		_ = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, next_check_at)
			VALUES ($1, $2, 60, 10, 'up', false, NOW() - INTERVAL '1 hour')
			RETURNING id
		`, orgID, assetID).Scan(&disabledMonID)

		// 2. Paused monitor -> should NOT be stale
		var pausedMonID string
		_ = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, next_check_at)
			VALUES ($1, $2, 60, 10, 'paused', true, NOW() - INTERVAL '1 hour')
			RETURNING id
		`, orgID, assetID).Scan(&pausedMonID)

		// 3. Monitor with valid active lease -> should NOT be stale prematurely
		var leasedMonID string
		_ = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, next_check_at, lease_acquired_at, lease_worker_id)
			VALUES ($1, $2, 60, 10, 'up', true, NOW() - INTERVAL '1 hour', NOW(), 'active-worker')
			RETURNING id
		`, orgID, assetID).Scan(&leasedMonID)

		// 4. Overdue pending first check within tolerance -> should NOT be stale (e.g. 2 min overdue when threshold is 15 min)
		var freshPendingMonID string
		_ = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, next_check_at, last_checked_at)
			VALUES ($1, $2, 60, 10, 'pending', true, NOW() - INTERVAL '2 minutes', NULL)
			RETURNING id
		`, orgID, assetID).Scan(&freshPendingMonID)

		// 5. Stale pending first check beyond threshold (e.g. 25 min overdue with 15 min threshold)
		var stalePendingMonID string
		_ = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, next_check_at, last_checked_at)
			VALUES ($1, $2, 60, 10, 'pending', true, NOW() - INTERVAL '25 minutes', NULL)
			RETURNING id
		`, orgID, assetID).Scan(&stalePendingMonID)

		// 6. Stale active running monitor beyond threshold
		var staleActiveMonID string
		_ = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, next_check_at, last_checked_at)
			VALUES ($1, $2, 60, 10, 'up', true, NOW() - INTERVAL '30 minutes', NOW() - INTERVAL '31 minutes')
			RETURNING id
		`, orgID, assetID).Scan(&staleActiveMonID)

		// Run detect_stale_monitors with multiplier 3, min tolerance 180s (3 min), lease timeout 60s
		staleList, err := telemetryRepo.DetectStaleMonitors(ctx, 3, 180, 60, 100)
		if err != nil {
			t.Fatalf("DetectStaleMonitors failed: %v", err)
		}

		staleMap := make(map[string]*StaleMonitorItem)
		for _, item := range staleList {
			staleMap[item.MonitorID] = item
		}

		// Assertions:
		if _, exists := staleMap[disabledMonID]; exists {
			t.Fatalf("disabled monitor %s was falsely classified as stale", disabledMonID)
		}
		if _, exists := staleMap[pausedMonID]; exists {
			t.Fatalf("paused monitor %s was falsely classified as stale", pausedMonID)
		}
		if _, exists := staleMap[leasedMonID]; exists {
			t.Fatalf("actively leased monitor %s was falsely classified as stale", leasedMonID)
		}
		if _, exists := staleMap[freshPendingMonID]; exists {
			t.Fatalf("fresh pending monitor %s within threshold was falsely classified as stale", freshPendingMonID)
		}

		if item, exists := staleMap[stalePendingMonID]; !exists {
			t.Fatalf("expected stale pending monitor %s to be detected", stalePendingMonID)
		} else if !item.IsFirstCheck {
			t.Fatalf("expected is_first_check to be true for %s", stalePendingMonID)
		}

		if item, exists := staleMap[staleActiveMonID]; !exists {
			t.Fatalf("expected stale active monitor %s to be detected", staleActiveMonID)
		} else if item.IsFirstCheck {
			t.Fatalf("expected is_first_check to be false for %s", staleActiveMonID)
		}
	})

	t.Run("TestCase5_SecurityAndTenantIsolation", func(t *testing.T) {
		// 1. clientpulse_app cannot call diagnose_monitor_scheduling_state
		_, err := appDB.Pool.Exec(ctx, "SELECT * FROM diagnose_monitor_scheduling_state(60)")
		if err == nil {
			t.Fatalf("SECURITY VIOLATION: clientpulse_app was able to execute diagnose_monitor_scheduling_state")
		}

		// 2. clientpulse_app cannot call detect_stale_monitors
		_, err = appDB.Pool.Exec(ctx, "SELECT * FROM detect_stale_monitors(3, 180, 60, 10)")
		if err == nil {
			t.Fatalf("SECURITY VIOLATION: clientpulse_app was able to execute detect_stale_monitors")
		}

		// 3. clientpulse_app cannot INSERT into scheduler_instances
		_, err = appDB.Pool.Exec(ctx, "INSERT INTO scheduler_instances (instance_id, worker_id, started_at) VALUES ('hacked', 'w', NOW())")
		if err == nil {
			t.Fatalf("SECURITY VIOLATION: clientpulse_app was able to INSERT into scheduler_instances")
		}

		// 4. Ordinary tenant isolation on monitors and assets remains intact
		// When query is run as clientpulse_app with a tenant context, other tenant monitors cannot be read
		tenantOrgA, _ := createTenant(fmt.Sprintf("tenant-iso-a-%d", time.Now().UnixNano()%1000000))
		defer adminDB.Pool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", tenantOrgA)

		tenantOrgB, _ := createTenant(fmt.Sprintf("tenant-iso-b-%d", time.Now().UnixNano()%1000000))
		defer adminDB.Pool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", tenantOrgB)

		txA, err := appDB.Pool.Begin(ctx)
		if err != nil {
			t.Fatalf("failed to begin tx as app role: %v", err)
		}
		defer txA.Rollback(ctx)

		_, _ = txA.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_organization_id = '%s'", tenantOrgA))

		var countB int
		err = txA.QueryRow(ctx, "SELECT COUNT(*) FROM monitors WHERE organization_id = $1", tenantOrgB).Scan(&countB)
		if err != nil {
			t.Fatalf("failed querying monitors in txA: %v", err)
		}
		if countB != 0 {
			t.Fatalf("RLS isolation failure: tenant A saw tenant B monitors (count=%d)", countB)
		}
	})

	t.Run("TestCase6_CycleHistoryRetentionPruning", func(t *testing.T) {
		instID := fmt.Sprintf("retention-test-%d", time.Now().UnixNano()%100000)

		// Insert an old cycle (> 10 days old)
		_, err := workerPool.Exec(ctx, `
			INSERT INTO scheduler_cycle_history (
				instance_id, poll_started_at, poll_completed_at, duration_ms,
				due_monitors_count, claimed_monitors_count, completed_checks_count, failed_checks_count,
				created_at
			)
			VALUES ($1, NOW() - INTERVAL '15 days', NOW() - INTERVAL '15 days', 10, 0, 0, 0, 0, NOW() - INTERVAL '15 days');
		`, instID)
		if err != nil {
			t.Fatalf("failed inserting old cycle history: %v", err)
		}

		// Insert a fresh cycle (< 1 day old)
		_, err = workerPool.Exec(ctx, `
			INSERT INTO scheduler_cycle_history (
				instance_id, poll_started_at, poll_completed_at, duration_ms,
				due_monitors_count, claimed_monitors_count, completed_checks_count, failed_checks_count,
				created_at
			)
			VALUES ($1, NOW(), NOW(), 10, 0, 0, 0, 0, NOW());
		`, instID)
		if err != nil {
			t.Fatalf("failed inserting fresh cycle history: %v", err)
		}

		// Prune older than 7 days
		pruned, err := telemetryRepo.PruneCycleHistory(ctx, 7)
		if err != nil {
			t.Fatalf("PruneCycleHistory failed: %v", err)
		}
		if pruned < 1 {
			t.Fatalf("expected at least 1 old cycle history pruned, got %d", pruned)
		}

		// Verify fresh record remains
		var remainingCount int
		err = workerPool.QueryRow(ctx, "SELECT COUNT(*) FROM scheduler_cycle_history WHERE instance_id = $1", instID).Scan(&remainingCount)
		if err != nil || remainingCount != 1 {
			t.Fatalf("expected exactly 1 remaining fresh cycle history, got %d (err: %v)", remainingCount, err)
		}
	})
}
