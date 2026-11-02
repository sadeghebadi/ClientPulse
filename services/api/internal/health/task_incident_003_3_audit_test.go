package health

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"clientpulse/services/api/internal/database"
	"clientpulse/services/api/internal/migrate"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/testhelper"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTaskIncident003_3_FinalConsistencyAudit(t *testing.T) {
	adminDB := testhelper.GetTestDB(t)
	if adminDB == nil {
		t.Skip("skipping integration test: database not reachable")
	}
	defer adminDB.Close()

	workerDBURL := "postgres://clientpulse_worker:clientpulse_worker_dev_password@localhost:5433/clientpulse?sslmode=disable"
	workerPool, err := pgxpool.New(context.Background(), workerDBURL)
	if err != nil {
		t.Fatalf("failed connecting as clientpulse_worker: %v", err)
	}
	defer workerPool.Close()

	ctx := context.Background()
	telemetryRepo := repository.NewSchedulerTelemetryRepository(workerPool)
	monitorRepo := repository.NewMonitorRepository(workerPool)

	// Helper to create tenant org, client, and asset
	createTenantAsset := func(slug string) (string, string) {
		var orgID, clientID, assetID string
		err := adminDB.Pool.QueryRow(ctx, "INSERT INTO organizations (name, slug) VALUES ($1, $2) RETURNING id", "Org "+slug, slug).Scan(&orgID)
		if err != nil {
			t.Fatalf("failed creating org: %v", err)
		}
		err = adminDB.Pool.QueryRow(ctx, "INSERT INTO clients (organization_id, name) VALUES ($1, 'Client') RETURNING id", orgID).Scan(&clientID)
		if err != nil {
			t.Fatalf("failed creating client: %v", err)
		}
		err = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO assets (organization_id, client_id, name, url, is_active)
			VALUES ($1, $2, 'Asset', 'https://asset.example.com', true)
			RETURNING id
		`, orgID, clientID).Scan(&assetID)
		if err != nil {
			t.Fatalf("failed creating asset: %v", err)
		}
		return orgID, assetID
	}

	// =========================================================================
	// AREA 1: Canonical Lease Fields & Lease Lifecycle Regressions
	// =========================================================================
	t.Run("Area1_LeaseTimestampAndTimeoutConsistency", func(t *testing.T) {
		slug := fmt.Sprintf("lease-audit-%d", time.Now().UnixNano()%100000)
		orgID, assetID := createTenantAsset(slug)
		defer adminDB.Pool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgID)

		// Create a monitor ready to be claimed
		var monID string
		err := adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (
				organization_id, asset_id, interval_seconds, timeout_seconds,
				status, is_active, next_check_at, last_checked_at
			)
			VALUES ($1, $2, 60, 10, 'up', true, NOW() - INTERVAL '10 seconds', NOW() - INTERVAL '70 seconds')
			RETURNING id
		`, orgID, assetID).Scan(&monID)
		if err != nil {
			t.Fatalf("failed creating monitor: %v", err)
		}

		// 1. Claim monitor via claim_due_monitors -> Verify lease_acquired_at and lease_worker_id are set
		jobs, err := monitorRepo.ClaimDueMonitors(ctx, "worker-audit-1", 10, 60)
		if err != nil {
			t.Fatalf("ClaimDueMonitors failed: %v", err)
		}
		var claimedJob *repository.ClaimedMonitorJob
		for _, j := range jobs {
			if j.ID == monID {
				claimedJob = j
				break
			}
		}
		if claimedJob == nil {
			t.Fatalf("expected monitor %s to be claimed", monID)
		}

		// Verify database row has lease_acquired_at and lease_worker_id populated
		var leaseAcquiredAt time.Time
		var leaseWorkerID string
		err = adminDB.Pool.QueryRow(ctx, `
			SELECT lease_acquired_at, lease_worker_id
			FROM monitors WHERE id = $1
		`, monID).Scan(&leaseAcquiredAt, &leaseWorkerID)
		if err != nil {
			t.Fatalf("failed to query monitor lease fields: %v", err)
		}
		if leaseWorkerID != "worker-audit-1" {
			t.Fatalf("expected lease_worker_id = worker-audit-1, got %s", leaseWorkerID)
		}
		if time.Since(leaseAcquiredAt) > 10*time.Second {
			t.Fatalf("expected fresh lease_acquired_at, got %v", leaseAcquiredAt)
		}

		// Regression Test 1: Newly acquired valid lease is EXCLUDED from detect_stale_monitors
		// Make next_check_at overdue beyond threshold (3*60s = 180s) to ensure stale check triggers if unleased
		_, err = adminDB.Pool.Exec(ctx, `
			UPDATE monitors SET next_check_at = NOW() - INTERVAL '300 seconds' WHERE id = $1
		`, monID)
		if err != nil {
			t.Fatalf("failed updating next_check_at: %v", err)
		}

		staleItems, err := telemetryRepo.DetectStaleMonitors(ctx, 3, 180, 60, 100)
		if err != nil {
			t.Fatalf("DetectStaleMonitors failed: %v", err)
		}
		for _, it := range staleItems {
			if it.MonitorID == monID {
				t.Fatalf("REGRESSION: Newly acquired valid lease was NOT excluded from detect_stale_monitors!")
			}
		}

		// Regression Test 2: Lease remains valid before configured expiry (e.g. 50s old with 60s expiry)
		_, err = adminDB.Pool.Exec(ctx, `
			UPDATE monitors SET lease_acquired_at = NOW() - INTERVAL '50 seconds' WHERE id = $1
		`, monID)
		if err != nil {
			t.Fatalf("failed updating lease_acquired_at: %v", err)
		}

		staleItems, err = telemetryRepo.DetectStaleMonitors(ctx, 3, 180, 60, 100)
		if err != nil {
			t.Fatalf("DetectStaleMonitors failed: %v", err)
		}
		for _, it := range staleItems {
			if it.MonitorID == monID {
				t.Fatalf("REGRESSION: Active lease within 60s timeout was falsely considered stale!")
			}
		}

		// Regression Test 3: Lease becomes expired after configured expiry (e.g. 65s old with 60s expiry)
		_, err = adminDB.Pool.Exec(ctx, `
			UPDATE monitors SET lease_acquired_at = NOW() - INTERVAL '65 seconds' WHERE id = $1
		`, monID)
		if err != nil {
			t.Fatalf("failed updating lease_acquired_at to expired: %v", err)
		}

		staleItems, err = telemetryRepo.DetectStaleMonitors(ctx, 3, 180, 60, 100)
		if err != nil {
			t.Fatalf("DetectStaleMonitors failed: %v", err)
		}
		foundExpired := false
		for _, it := range staleItems {
			if it.MonitorID == monID {
				foundExpired = true
				break
			}
		}
		if !foundExpired {
			t.Fatalf("REGRESSION: Expired lease (>60s) was not detected as stale!")
		}

		// Regression Test 4: Normal lease release (e.g., when check completes or resets) allows normal stale detection
		_, err = adminDB.Pool.Exec(ctx, `
			UPDATE monitors SET lease_acquired_at = NULL, lease_worker_id = NULL WHERE id = $1
		`, monID)
		if err != nil {
			t.Fatalf("failed clearing lease: %v", err)
		}

		staleItems, err = telemetryRepo.DetectStaleMonitors(ctx, 3, 180, 60, 100)
		if err != nil {
			t.Fatalf("DetectStaleMonitors failed: %v", err)
		}
		foundUnleased := false
		for _, it := range staleItems {
			if it.MonitorID == monID {
				foundUnleased = true
				break
			}
		}
		if !foundUnleased {
			t.Fatalf("REGRESSION: Unleased overdue monitor was not detected by detect_stale_monitors!")
		}

		// Regression Test 5: Inconsistent lease metadata handled safely (e.g. lease_worker_id set but lease_acquired_at NULL)
		_, err = adminDB.Pool.Exec(ctx, `
			UPDATE monitors SET lease_acquired_at = NULL, lease_worker_id = 'orphaned-worker' WHERE id = $1
		`, monID)
		if err != nil {
			t.Fatalf("failed setting inconsistent lease metadata: %v", err)
		}

		staleItems, err = telemetryRepo.DetectStaleMonitors(ctx, 3, 180, 60, 100)
		if err != nil {
			t.Fatalf("DetectStaleMonitors failed: %v", err)
		}
		foundInconsistent := false
		for _, it := range staleItems {
			if it.MonitorID == monID {
				foundInconsistent = true
				break
			}
		}
		if !foundInconsistent {
			t.Fatalf("REGRESSION: Inconsistent lease metadata (null acquired_at) suppressed stale detection!")
		}
	})

	// =========================================================================
	// AREA 2: PostgreSQL Function Signatures & System Catalogs
	// =========================================================================
	t.Run("Area2_FunctionSignaturesAndCatalogVerification", func(t *testing.T) {
		// System catalog query to verify no obsolete overloads of detect_stale_monitors exist
		rows, err := adminDB.Pool.Query(ctx, `
			SELECT n.nspname as schema,
			       p.proname as function_name,
			       pg_get_function_identity_arguments(p.oid) as identity_args,
			       r.rolname as owner,
			       p.prosecdef as security_definer,
			       array_to_string(p.proconfig, ',') as search_path,
			       p.pronargs,
			       p.pronargdefaults
			FROM pg_proc p
			JOIN pg_namespace n ON n.oid = p.pronamespace
			JOIN pg_roles r ON r.oid = p.proowner
			WHERE p.proname = 'detect_stale_monitors' AND n.nspname = 'public';
		`)
		if err != nil {
			t.Fatalf("catalog query failed: %v", err)
		}
		defer rows.Close()

		var count int
		var identityArgs string
		var owner string
		var isSecDef bool
		var searchPath string
		var pronargs, pronargdefaults int

		for rows.Next() {
			count++
			var schema, fn string
			if err := rows.Scan(&schema, &fn, &identityArgs, &owner, &isSecDef, &searchPath, &pronargs, &pronargdefaults); err != nil {
				t.Fatalf("failed scanning catalog row: %v", err)
			}
		}

		if count != 1 {
			t.Fatalf("FUNCTION OVERLOAD DETECTED: expected exactly 1 signature for detect_stale_monitors, found %d", count)
		}

		expectedIdentity := "p_interval_multiplier integer, p_min_tolerance_seconds integer, p_lease_timeout_seconds integer, p_limit integer"
		if identityArgs != expectedIdentity {
			t.Fatalf("unexpected function signature: expected '%s', got '%s'", expectedIdentity, identityArgs)
		}

		if owner != "clientpulse_definer" {
			t.Fatalf("expected function owner clientpulse_definer, got %s", owner)
		}
		if !isSecDef {
			t.Fatalf("expected function to be SECURITY DEFINER")
		}
		if searchPath != "search_path=pg_catalog, public, pg_temp" {
			t.Fatalf("expected hardened search_path, got %s", searchPath)
		}

		// Verify clientpulse_app CANNOT execute detect_stale_monitors
		appDBURL := "postgres://clientpulse_app:clientpulse_app_dev_password@localhost:5433/clientpulse?sslmode=disable"
		appPool, err := pgxpool.New(context.Background(), appDBURL)
		if err != nil {
			t.Fatalf("failed connecting as clientpulse_app: %v", err)
		}
		defer appPool.Close()

		_, err = appPool.Exec(ctx, "SELECT * FROM detect_stale_monitors(3, 180, 60, 10)")
		if err == nil {
			t.Fatalf("SECURITY VIOLATION: clientpulse_app was able to execute detect_stale_monitors!")
		}

		// Verify clientpulse_worker CAN execute detect_stale_monitors with the repository arguments
		_, err = workerPool.Exec(ctx, "SELECT * FROM detect_stale_monitors(3, 180, 60, 10)")
		if err != nil {
			t.Fatalf("clientpulse_worker failed executing detect_stale_monitors: %v", err)
		}
	})

	// =========================================================================
	// AREA 3: Active Monitors with Missing Scheduling Timestamps
	// =========================================================================
	t.Run("Area3_MissingSchedulingTimestamps", func(t *testing.T) {
		slug := fmt.Sprintf("sched-anomaly-%d", time.Now().UnixNano()%100000)
		orgID, assetID := createTenantAsset(slug)
		defer adminDB.Pool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgID)

		// 1. Temporarily relax NOT NULL constraint in test transaction or table to simulate
		// potential historical anomaly rows or database corruption edge cases safely
		_, err := adminDB.Pool.Exec(ctx, `ALTER TABLE monitors ALTER COLUMN next_check_at DROP NOT NULL;`)
		if err != nil {
			t.Fatalf("failed dropping not null constraint for anomaly test: %v", err)
		}
		defer func() {
			// Restore NOT NULL constraint, cleaning up test nulls first
			_, _ = adminDB.Pool.Exec(ctx, `UPDATE monitors SET next_check_at = NOW() WHERE next_check_at IS NULL;`)
			_, _ = adminDB.Pool.Exec(ctx, `ALTER TABLE monitors ALTER COLUMN next_check_at SET NOT NULL;`)
		}()

		// 1. Newly created monitor with NULL next_check_at within grace period (e.g. created 10 seconds ago)
		var monNewWithinGrace string
		err = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (
				organization_id, asset_id, interval_seconds, timeout_seconds,
				status, is_active, next_check_at, last_checked_at, created_at
			)
			VALUES ($1, $2, 60, 10, 'pending', true, NULL, NULL, NOW() - INTERVAL '10 seconds')
			RETURNING id
		`, orgID, assetID).Scan(&monNewWithinGrace)
		if err != nil {
			t.Fatalf("failed creating new monitor within grace period: %v", err)
		}

		// 2. Active monitor with NULL next_check_at beyond grace period (e.g. created 15 minutes ago)
		var monActiveBeyondGrace string
		err = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (
				organization_id, asset_id, interval_seconds, timeout_seconds,
				status, is_active, next_check_at, last_checked_at, created_at
			)
			VALUES ($1, $2, 60, 10, 'pending', true, NULL, NULL, NOW() - INTERVAL '15 minutes')
			RETURNING id
		`, orgID, assetID).Scan(&monActiveBeyondGrace)
		if err != nil {
			t.Fatalf("failed creating active monitor beyond grace period: %v", err)
		}

		// 3. Paused monitor with NULL next_check_at
		var monPausedNull string
		err = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (
				organization_id, asset_id, interval_seconds, timeout_seconds,
				status, is_active, next_check_at, last_checked_at, created_at
			)
			VALUES ($1, $2, 60, 10, 'paused', true, NULL, NULL, NOW() - INTERVAL '15 minutes')
			RETURNING id
		`, orgID, assetID).Scan(&monPausedNull)
		if err != nil {
			t.Fatalf("failed creating paused monitor: %v", err)
		}

		// 4. Disabled monitor with NULL next_check_at
		var monDisabledNull string
		err = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (
				organization_id, asset_id, interval_seconds, timeout_seconds,
				status, is_active, next_check_at, last_checked_at, created_at
			)
			VALUES ($1, $2, 60, 10, 'pending', false, NULL, NULL, NOW() - INTERVAL '15 minutes')
			RETURNING id
		`, orgID, assetID).Scan(&monDisabledNull)
		if err != nil {
			t.Fatalf("failed creating disabled monitor: %v", err)
		}

		// 5. Active monitor with valid scheduled timestamp (not overdue)
		var monValidScheduled string
		err = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (
				organization_id, asset_id, interval_seconds, timeout_seconds,
				status, is_active, next_check_at, last_checked_at, created_at
			)
			VALUES ($1, $2, 60, 10, 'up', true, NOW() + INTERVAL '50 seconds', NOW() - INTERVAL '10 seconds', NOW() - INTERVAL '15 minutes')
			RETURNING id
		`, orgID, assetID).Scan(&monValidScheduled)
		if err != nil {
			t.Fatalf("failed creating valid scheduled monitor: %v", err)
		}

		// 6. Monitor whose first check succeeds -> Transitions from pending to up with updated next_check_at
		var monFirstCheckSuccess string
		err = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (
				organization_id, asset_id, interval_seconds, timeout_seconds,
				status, is_active, next_check_at, last_checked_at, created_at
			)
			VALUES ($1, $2, 60, 10, 'pending', true, NOW() - INTERVAL '5 seconds', NULL, NOW() - INTERVAL '1 minute')
			RETURNING id
		`, orgID, assetID).Scan(&monFirstCheckSuccess)
		if err != nil {
			t.Fatalf("failed creating first check monitor: %v", err)
		}
		// Simulate check result recorder update
		_, err = adminDB.Pool.Exec(ctx, `
			UPDATE monitors 
			SET status = 'up', last_checked_at = NOW(), next_check_at = NOW() + INTERVAL '60 seconds'
			WHERE id = $1
		`, monFirstCheckSuccess)
		if err != nil {
			t.Fatalf("failed updating monitor after first check: %v", err)
		}

		// 7. Monitor whose next scheduled execution becomes overdue beyond dynamic threshold (e.g. 60s interval, overdue 250s)
		var monOverdueScheduled string
		err = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (
				organization_id, asset_id, interval_seconds, timeout_seconds,
				status, is_active, next_check_at, last_checked_at, created_at
			)
			VALUES ($1, $2, 60, 10, 'up', true, NOW() - INTERVAL '250 seconds', NOW() - INTERVAL '310 seconds', NOW() - INTERVAL '1 hour')
			RETURNING id
		`, orgID, assetID).Scan(&monOverdueScheduled)
		if err != nil {
			t.Fatalf("failed creating overdue monitor: %v", err)
		}

		// A. Verify detect_stale_monitors behavior:
		// Stale monitor query must detect only monOverdueScheduled.
		// It MUST NOT include NULL next_check_at monitors (neither normal initialization nor anomaly).
		staleItems, err := telemetryRepo.DetectStaleMonitors(ctx, 3, 180, 60, 100)
		if err != nil {
			t.Fatalf("DetectStaleMonitors failed: %v", err)
		}
		staleMap := make(map[string]bool)
		for _, it := range staleItems {
			staleMap[it.MonitorID] = true
		}

		if !staleMap[monOverdueScheduled] {
			t.Fatalf("expected overdue scheduled monitor %s to be detected as stale", monOverdueScheduled)
		}
		if staleMap[monNewWithinGrace] {
			t.Fatalf("new monitor within grace period was falsely flagged in detect_stale_monitors")
		}
		if staleMap[monActiveBeyondGrace] {
			t.Fatalf("anomaly monitor with NULL next_check_at was falsely mixed into detect_stale_monitors")
		}
		if staleMap[monPausedNull] || staleMap[monDisabledNull] || staleMap[monValidScheduled] || staleMap[monFirstCheckSuccess] {
			t.Fatalf("unexpected monitor flagged in detect_stale_monitors")
		}

		// B. Anomaly Detection Verification:
		// Query unscheduled anomaly monitors (active, not paused, next_check_at IS NULL, created_at > grace period e.g. 5 minutes)
		// Return only minimal sanitized identifiers (id, organization_id, status, created_at)
		type UnscheduledAnomaly struct {
			MonitorID      string
			OrganizationID string
			Status         string
			CreatedAt      time.Time
		}

		var anomalies []UnscheduledAnomaly
		rows, err := adminDB.Pool.Query(ctx, `
			SELECT id, organization_id, status, created_at
			FROM monitors
			WHERE is_active = true
			  AND status != 'paused'
			  AND next_check_at IS NULL
			  AND created_at < NOW() - INTERVAL '5 minutes'
		`)
		if err != nil {
			t.Fatalf("failed querying unscheduled anomalies: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var a UnscheduledAnomaly
			if err := rows.Scan(&a.MonitorID, &a.OrganizationID, &a.Status, &a.CreatedAt); err != nil {
				t.Fatalf("failed scanning anomaly row: %v", err)
			}
			anomalies = append(anomalies, a)
		}

		anomalyMap := make(map[string]bool)
		for _, a := range anomalies {
			anomalyMap[a.MonitorID] = true
		}

		// Active beyond grace period MUST be flagged as an unscheduled initialization anomaly
		if !anomalyMap[monActiveBeyondGrace] {
			t.Fatalf("expected active monitor beyond grace period %s to be flagged as unscheduled anomaly", monActiveBeyondGrace)
		}
		// Newly created within grace period MUST NOT be flagged as anomaly
		if anomalyMap[monNewWithinGrace] {
			t.Fatalf("newly created monitor within grace period %s was falsely flagged as unscheduled anomaly", monNewWithinGrace)
		}
		// Paused, disabled, and valid monitors MUST NOT be flagged as anomalies
		if anomalyMap[monPausedNull] || anomalyMap[monDisabledNull] || anomalyMap[monValidScheduled] || anomalyMap[monOverdueScheduled] {
			t.Fatalf("paused, disabled, or valid monitors were falsely flagged as unscheduled anomalies")
		}
	})

	// =========================================================================
	// AREA 4: Scheduler Runtime Staleness Threshold & Timing Verification
	// =========================================================================
	t.Run("Area4_SchedulerTimingAndStalenessThresholds", func(t *testing.T) {
		// Test formula: max(10 * pollPeriod, 20 seconds)

		// 1. Normal 1-second poll cadence -> threshold is 20s
		eval1s := NewSchedulerHealthEvaluator(1 * time.Second)
		if thresh := eval1s.StaleThreshold(); thresh != 20*time.Second {
			t.Fatalf("expected 20s threshold for 1s cadence, got %v", thresh)
		}

		// 2. Normal 5-second poll cadence -> threshold is 50s
		eval5s := NewSchedulerHealthEvaluator(5 * time.Second)
		if thresh := eval5s.StaleThreshold(); thresh != 50*time.Second {
			t.Fatalf("expected 50s threshold for 5s cadence, got %v", thresh)
		}

		now := time.Now().UTC()

		// 3. Poll cycle near its upper expected duration:
		// A cycle that took 18 seconds to complete on a 1-second cadence
		// (poll started at now - 18s, completed at now - 0.5s)
		lastCompleted := now.Add(-500 * time.Millisecond)
		lastSuccess := now.Add(-500 * time.Millisecond)
		started := now.Add(-1 * time.Hour)
		dur := int64(18000)
		instNearUpper := &repository.SchedulerInstanceState{
			InstanceID:           "inst-timing-1",
			StartedAt:            started,
			LastPollStartedAt:    &lastCompleted,
			LastPollCompletedAt:  &lastCompleted,
			LastSuccessfulPollAt: &lastSuccess,
			PollDurationMs:       &dur,
			ConsecutiveErrors:    0,
		}
		snap := eval1s.Evaluate(instNearUpper, now)
		if snap.State != SchedulerStateHealthy {
			t.Fatalf("expected Healthy for poll cycle near upper bound that recently completed, got %s", snap.State)
		}

		// 4. A genuinely frozen scheduler:
		// Completed last poll 25 seconds ago on a 1s cadence -> Stale
		frozenCompleted := now.Add(-25 * time.Second)
		instFrozen := &repository.SchedulerInstanceState{
			InstanceID:           "inst-timing-frozen",
			StartedAt:            started,
			LastPollCompletedAt:  &frozenCompleted,
			LastSuccessfulPollAt: &frozenCompleted,
			ConsecutiveErrors:    0,
		}
		snap = eval1s.Evaluate(instFrozen, now)
		if snap.State != SchedulerStateStale {
			t.Fatalf("expected Stale for frozen scheduler (>20s inactivity), got %s", snap.State)
		}

		// 5. Database lock wait or in-progress claim:
		// Polling loop is blocked inside a query timeout (bounded by claimCtx: 5 seconds)
		// Since last completed poll was 10s ago (< 20s), it is still considered Healthy
		lockWaitLastCompleted := now.Add(-10 * time.Second)
		instLockWait := &repository.SchedulerInstanceState{
			InstanceID:           "inst-timing-lock",
			StartedAt:            started,
			LastPollCompletedAt:  &lockWaitLastCompleted,
			LastSuccessfulPollAt: &lockWaitLastCompleted,
			ConsecutiveErrors:    0,
		}
		snap = eval1s.Evaluate(instLockWait, now)
		if snap.State != SchedulerStateHealthy {
			t.Fatalf("expected Healthy during in-progress claim wait (<20s), got %s", snap.State)
		}

		// 6. A failed polling cycle that completes and is recorded:
		// Repeated errors (e.g. 3 consecutive errors) with recent completion -> Degraded, NOT Stale
		instFailedCycle := &repository.SchedulerInstanceState{
			InstanceID:          "inst-timing-err",
			StartedAt:           started,
			LastPollCompletedAt: &lastCompleted,
			ConsecutiveErrors:   3,
		}
		snap = eval1s.Evaluate(instFailedCycle, now)
		if snap.State != SchedulerStateDegraded {
			t.Fatalf("expected Degraded for cycle with repeated errors, got %s", snap.State)
		}

		// 7. A successful recovery cycle:
		// Consecutive errors reset to 0 -> Healthy
		instRecovery := &repository.SchedulerInstanceState{
			InstanceID:           "inst-timing-rec",
			StartedAt:            started,
			LastPollCompletedAt:  &lastCompleted,
			LastSuccessfulPollAt: &lastCompleted,
			ConsecutiveErrors:    0,
		}
		snap = eval1s.Evaluate(instRecovery, now)
		if snap.State != SchedulerStateHealthy {
			t.Fatalf("expected Healthy after recovery cycle, got %s", snap.State)
		}
	})
}

// TestMigrationCompatibility verifies upgrade, down, and reapplication for migrations 14, 15, and 16.
func TestMigrationCompatibility_15_and_16(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	adminDBURL := os.Getenv("TEST_DATABASE_URL")
	if adminDBURL == "" {
		adminDBURL = "postgres://clientpulse:clientpulse_dev_password@localhost:5433/clientpulse?sslmode=disable"
	}

	adminDB, err := database.New(ctx, database.Config{DatabaseURL: adminDBURL})
	if err != nil {
		t.Skip("skipping migration compatibility test: database not reachable at", adminDBURL)
		return
	}
	defer adminDB.Close()

	migDirCandidates := []string{
		"../../../../migrations",
		"../../../migrations",
		"../../migrations",
		"./migrations",
	}
	var migDir string
	for _, candidate := range migDirCandidates {
		absPath, err := filepath.Abs(candidate)
		if err == nil {
			if _, err := os.Stat(filepath.Join(absPath, "000016_scheduler_lease_config_and_state_semantics.up.sql")); err == nil {
				migDir = absPath
				break
			}
		}
	}
	if migDir == "" {
		t.Fatalf("could not locate migrations directory containing 000016_scheduler_lease_config_and_state_semantics.up.sql")
	}

	// Helper to create and drop a fresh database
	createDB := func(dbName string) (*database.DB, func()) {
		_, _ = adminDB.Pool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", dbName))
		_, err := adminDB.Pool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", dbName))
		if err != nil {
			t.Fatalf("failed to create database %s: %v", dbName, err)
		}

		freshDBURL := fmt.Sprintf("postgres://clientpulse:clientpulse_dev_password@localhost:5433/%s?sslmode=disable", dbName)
		freshDB, err := database.New(ctx, database.Config{DatabaseURL: freshDBURL})
		if err != nil {
			t.Fatalf("failed to connect to fresh db %s: %v", dbName, err)
		}

		cleanup := func() {
			freshDB.Close()
			_, _ = adminDB.Pool.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", dbName))
		}
		return freshDB, cleanup
	}

	// Scenario 1: Fresh Database applies complete chain (1 through 16)
	t.Run("Scenario1_FreshDatabaseAppliesCompleteChain", func(t *testing.T) {
		dbName := fmt.Sprintf("test_mig_fresh_%d", time.Now().UnixNano()%100000)
		db, cleanup := createDB(dbName)
		defer cleanup()

		engine := migrate.New(db.Pool, migDir)
		appliedCount, err := engine.Up(ctx)
		if err != nil {
			t.Fatalf("engine.Up failed on fresh database: %v", err)
		}
		t.Logf("Fresh DB applied %d migrations", appliedCount)

		// Verify version 16 in schema_migrations
		var v16Applied bool
		err = db.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = 16)").Scan(&v16Applied)
		if err != nil || !v16Applied {
			t.Fatalf("expected version 16 to be applied, err: %v", err)
		}

		// Verify exactly 1 signature for detect_stale_monitors
		var count int
		err = db.Pool.QueryRow(ctx, `
			SELECT count(*) FROM pg_proc p
			JOIN pg_namespace n ON n.oid = p.pronamespace
			WHERE p.proname = 'detect_stale_monitors' AND n.nspname = 'public';
		`).Scan(&count)
		if err != nil || count != 1 {
			t.Fatalf("expected exactly 1 signature for detect_stale_monitors, got %d, err: %v", count, err)
		}
	})

	// Scenario 2: Existing version 15 database upgrades to version 16 successfully
	t.Run("Scenario2_UpgradeFromVersion15To16", func(t *testing.T) {
		dbName := fmt.Sprintf("test_mig_up15_%d", time.Now().UnixNano()%100000)
		db, cleanup := createDB(dbName)
		defer cleanup()

		engine := migrate.New(db.Pool, migDir)
		allMigrations, err := engine.LoadMigrations()
		if err != nil {
			t.Fatalf("failed to load migrations: %v", err)
		}

		if err := engine.EnsureTable(ctx); err != nil {
			t.Fatalf("failed to ensure schema_migrations: %v", err)
		}

		// Apply version 1 to 15
		for _, m := range allMigrations {
			if m.Version > 15 {
				continue
			}
			_, err := db.Pool.Exec(ctx, m.UpSQL)
			if err != nil {
				t.Fatalf("failed to apply migration %d: %v", m.Version, err)
			}
			_, err = db.Pool.Exec(ctx, "INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", m.Version, m.Name)
			if err != nil {
				t.Fatalf("failed to record migration %d: %v", m.Version, err)
			}
		}

		// Verify version 15 has 3-argument signature
		var sig15 string
		err = db.Pool.QueryRow(ctx, `
			SELECT pg_get_function_identity_arguments(p.oid)
			FROM pg_proc p
			JOIN pg_namespace n ON n.oid = p.pronamespace
			WHERE p.proname = 'detect_stale_monitors' AND n.nspname = 'public';
		`).Scan(&sig15)
		if err != nil {
			t.Fatalf("failed querying v15 signature: %v", err)
		}
		expectedV15Sig := "p_interval_multiplier integer, p_min_tolerance_seconds integer, p_limit integer"
		if sig15 != expectedV15Sig {
			t.Fatalf("expected v15 signature '%s', got '%s'", expectedV15Sig, sig15)
		}

		// Now apply engine.Up to apply version 16
		applied, err := engine.Up(ctx)
		if err != nil {
			t.Fatalf("engine.Up failed upgrading from 15 to 16: %v", err)
		}
		if applied != 1 {
			t.Fatalf("expected 1 migration applied, got %d", applied)
		}

		// Verify version 16 replaced it cleanly without leaving 3-arg overload behind
		var countAfterUp int
		var sig16 string
		err = db.Pool.QueryRow(ctx, `
			SELECT count(*), max(pg_get_function_identity_arguments(p.oid))
			FROM pg_proc p
			JOIN pg_namespace n ON n.oid = p.pronamespace
			WHERE p.proname = 'detect_stale_monitors' AND n.nspname = 'public';
		`).Scan(&countAfterUp, &sig16)
		if err != nil {
			t.Fatalf("failed querying v16 signatures: %v", err)
		}
		if countAfterUp != 1 {
			t.Fatalf("OVERLOAD DETECTED after upgrade: expected 1 function signature, got %d", countAfterUp)
		}
		expectedV16Sig := "p_interval_multiplier integer, p_min_tolerance_seconds integer, p_lease_timeout_seconds integer, p_limit integer"
		if sig16 != expectedV16Sig {
			t.Fatalf("expected v16 signature '%s', got '%s'", expectedV16Sig, sig16)
		}

		// Down migration: Rollback 16 -> should restore version 15 definition
		rolledBack, err := engine.Down(ctx)
		if err != nil {
			t.Fatalf("engine.Down failed rolling back 16: %v", err)
		}
		if rolledBack != 16 {
			t.Fatalf("expected rolledBackVersion 16, got %d", rolledBack)
		}

		// Verify version 15 signature was accurately restored
		var countAfterDown int
		var restoredSig string
		err = db.Pool.QueryRow(ctx, `
			SELECT count(*), max(pg_get_function_identity_arguments(p.oid))
			FROM pg_proc p
			JOIN pg_namespace n ON n.oid = p.pronamespace
			WHERE p.proname = 'detect_stale_monitors' AND n.nspname = 'public';
		`).Scan(&countAfterDown, &restoredSig)
		if err != nil {
			t.Fatalf("failed querying restored signature: %v", err)
		}
		if countAfterDown != 1 {
			t.Fatalf("OVERLOAD DETECTED after rollback: expected 1 signature, got %d", countAfterDown)
		}
		if restoredSig != expectedV15Sig {
			t.Fatalf("expected restored signature '%s', got '%s'", expectedV15Sig, restoredSig)
		}

		// Reapplication: Up back to 16
		reApplied, err := engine.Up(ctx)
		if err != nil {
			t.Fatalf("engine.Up failed re-applying 16: %v", err)
		}
		if reApplied != 1 {
			t.Fatalf("expected 1 migration re-applied, got %d", reApplied)
		}

		var countAfterReapply int
		err = db.Pool.QueryRow(ctx, `
			SELECT count(*)
			FROM pg_proc p
			JOIN pg_namespace n ON n.oid = p.pronamespace
			WHERE p.proname = 'detect_stale_monitors' AND n.nspname = 'public';
		`).Scan(&countAfterReapply)
		if err != nil || countAfterReapply != 1 {
			t.Fatalf("expected 1 signature after reapplication, got %d, err: %v", countAfterReapply, err)
		}
	})
}
