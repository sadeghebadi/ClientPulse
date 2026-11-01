package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Helper to open connection pool as a specific role
func openPoolAsRole(t *testing.T, user, password string) *pgxpool.Pool {
	t.Helper()
	connStr := fmt.Sprintf("postgres://%s:%s@localhost:5433/clientpulse?sslmode=disable", user, password)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to connect as %s: %v", user, err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("failed to ping as %s: %v", user, err)
	}
	return pool
}

// 1. Security Tests: Database Roles & SECURITY DEFINER Function Permissions
func TestDatabaseRolePermissionsAndClaimIsolation(t *testing.T) {
	ctx := context.Background()

	appPool := openPoolAsRole(t, "clientpulse_app", "clientpulse_app_dev_password")
	defer appPool.Close()

	workerPool := openPoolAsRole(t, "clientpulse_worker", "clientpulse_worker_dev_password")
	defer workerPool.Close()

	// Finding 1 Test 1: API role (clientpulse_app) MUST NOT be able to execute claim_due_monitors
	t.Run("API_Role_Denied_Execute_ClaimDueMonitors", func(t *testing.T) {
		query := `SELECT id FROM claim_due_monitors('worker-test', 10, 60)`
		rows, err := appPool.Query(ctx, query)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
			}
			err = rows.Err()
		}
		if err == nil {
			t.Fatalf("SECURITY VIOLATION: clientpulse_app was able to execute claim_due_monitors!")
		}
		// Expect permission denied error
		t.Logf("Expected permission denied error for clientpulse_app: %v", err)
	})

	// Finding 1 Test 2: Dedicated Worker role CAN execute claim_due_monitors
	t.Run("Worker_Role_Allowed_Execute_ClaimDueMonitors", func(t *testing.T) {
		query := `SELECT id FROM claim_due_monitors('worker-test', 10, 60)`
		rows, err := workerPool.Query(ctx, query)
		if err != nil {
			t.Fatalf("worker role failed to execute claim_due_monitors: %v", err)
		}
		defer rows.Close()
		for rows.Next() {
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("worker role failed during claim_due_monitors iteration: %v", err)
		}
	})

	// Finding 1 Test 3: Worker role CANNOT modify schema (CREATE TABLE denied)
	t.Run("Worker_Role_Denied_Schema_Modification", func(t *testing.T) {
		_, err := workerPool.Exec(ctx, "CREATE TABLE public.malicious_worker_table (id INT)")
		if err == nil {
			t.Fatalf("SECURITY VIOLATION: worker role was able to create tables in public schema!")
		}
		t.Logf("Expected schema modification denied error for worker: %v", err)
	})

	// Finding 1 Test 4: Worker role CANNOT modify or delete audit_logs
	t.Run("Worker_Role_Denied_AuditLog_Modification", func(t *testing.T) {
		_, err := workerPool.Exec(ctx, "DELETE FROM public.audit_logs")
		if err == nil {
			t.Fatalf("SECURITY VIOLATION: worker role was able to delete from audit_logs!")
		}
		_, err = workerPool.Exec(ctx, "UPDATE public.audit_logs SET action = 'tampered'")
		if err == nil {
			t.Fatalf("SECURITY VIOLATION: worker role was able to update audit_logs!")
		}
		t.Logf("Expected audit log modification denied error for worker: %v", err)
	})

	// Finding 1 Test 5: Parameter validation inside claim_due_monitors
	t.Run("ClaimDueMonitors_Parameter_Validation", func(t *testing.T) {
		execQuery := func(q string) error {
			rows, err := workerPool.Query(ctx, q)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
			}
			return rows.Err()
		}

		// Excessive batch size (> 100) must fail
		err := execQuery("SELECT id FROM claim_due_monitors('w', 101, 60)")
		if err == nil {
			t.Fatalf("expected error for excessive batch_size > 100, got nil")
		}

		// Zero batch size must fail
		err = execQuery("SELECT id FROM claim_due_monitors('w', 0, 60)")
		if err == nil {
			t.Fatalf("expected error for batch_size 0, got nil")
		}

		// Invalid lease timeout (< 5s) must fail
		err = execQuery("SELECT id FROM claim_due_monitors('w', 10, 2)")
		if err == nil {
			t.Fatalf("expected error for lease_timeout < 5, got nil")
		}

		// Empty worker ID must fail
		err = execQuery("SELECT id FROM claim_due_monitors('', 10, 60)")
		if err == nil {
			t.Fatalf("expected error for empty worker id, got nil")
		}
	})
}

// 2. Integration Tests: Composite Tenant Foreign Keys
func TestCompositeTenantForeignKeys(t *testing.T) {
	ctx := context.Background()
	adminPool := openPoolAsRole(t, "clientpulse", "clientpulse_dev_password")
	defer adminPool.Close()

	appPool := openPoolAsRole(t, "clientpulse_app", "clientpulse_app_dev_password")
	defer appPool.Close()

	// Setup two test organizations
	var orgA, orgB string
	err := adminPool.QueryRow(ctx, "INSERT INTO organizations (name, slug) VALUES ('Org A FK Test', 'org-a-fk') RETURNING id").Scan(&orgA)
	if err != nil {
		t.Fatalf("failed to create org A: %v", err)
	}
	defer adminPool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgA)

	err = adminPool.QueryRow(ctx, "INSERT INTO organizations (name, slug) VALUES ('Org B FK Test', 'org-b-fk') RETURNING id").Scan(&orgB)
	if err != nil {
		t.Fatalf("failed to create org B: %v", err)
	}
	defer adminPool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgB)

	// Create Client in Org A
	var clientA string
	err = adminPool.QueryRow(ctx, "INSERT INTO clients (organization_id, name) VALUES ($1, 'Client A') RETURNING id", orgA).Scan(&clientA)
	if err != nil {
		t.Fatalf("failed to create client in Org A: %v", err)
	}

	// Create Asset in Org A
	var assetA string
	err = adminPool.QueryRow(ctx, "INSERT INTO assets (organization_id, client_id, name, url) VALUES ($1, $2, 'Asset A', 'https://a.com') RETURNING id", orgA, clientA).Scan(&assetA)
	if err != nil {
		t.Fatalf("failed to create asset in Org A: %v", err)
	}

	// Create Monitor in Org A
	var monitorA string
	err = adminPool.QueryRow(ctx, "INSERT INTO monitors (organization_id, asset_id, method) VALUES ($1, $2, 'GET') RETURNING id", orgA, assetA).Scan(&monitorA)
	if err != nil {
		t.Fatalf("failed to create monitor in Org A: %v", err)
	}

	// Test 1: Attempt to create an Asset belonging to Org B pointing to Client A (Org A)
	t.Run("CrossTenant_Asset_To_Client_MustFail", func(t *testing.T) {
		err := WithTenantTx(ctx, appPool, orgB, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, "INSERT INTO assets (organization_id, client_id, name, url) VALUES ($1, $2, 'Hacked Asset', 'https://evil.com')", orgB, clientA)
			return err
		})
		if err == nil {
			t.Fatalf("DATA INTEGRITY VIOLATION: Cross-tenant asset was inserted pointing to client in another org!")
		}
		t.Logf("Expected foreign key violation for cross-tenant asset: %v", err)
	})

	// Test 2: Attempt to create a Monitor belonging to Org B pointing to Asset A (Org A)
	t.Run("CrossTenant_Monitor_To_Asset_MustFail", func(t *testing.T) {
		err := WithTenantTx(ctx, appPool, orgB, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, "INSERT INTO monitors (organization_id, asset_id, method) VALUES ($1, $2, 'GET')", orgB, assetA)
			return err
		})
		if err == nil {
			t.Fatalf("DATA INTEGRITY VIOLATION: Cross-tenant monitor was inserted pointing to asset in another org!")
		}
		t.Logf("Expected foreign key violation for cross-tenant monitor: %v", err)
	})

	// Test 3: Attempt to create a CheckResult belonging to Org B pointing to Monitor A (Org A)
	t.Run("CrossTenant_CheckResult_To_Monitor_MustFail", func(t *testing.T) {
		now := time.Now().UTC()
		err := WithTenantTx(ctx, appPool, orgB, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, `
				INSERT INTO check_results (
					organization_id, monitor_id, scheduled_at, started_at, completed_at,
					latency_ms, result, worker_id
				) VALUES ($1, $2, $3, $3, $3, 100, 'success', 'test-worker')
			`, orgB, monitorA, now)
			return err
		})
		if err == nil {
			t.Fatalf("DATA INTEGRITY VIOLATION: Cross-tenant check_result was inserted pointing to monitor in another org!")
		}
		t.Logf("Expected foreign key violation for cross-tenant check result: %v", err)
	})
}

// 3. State Transition Machine & Consecutive Failure Thresholds Tests
func TestMonitorStateTransitionsAndConsecutiveFailures(t *testing.T) {
	ctx := context.Background()
	adminPool := openPoolAsRole(t, "clientpulse", "clientpulse_dev_password")
	defer adminPool.Close()

	appPool := openPoolAsRole(t, "clientpulse_app", "clientpulse_app_dev_password")
	defer appPool.Close()

	monitorRepo := NewMonitorRepository(appPool)

	var orgID string
	err := adminPool.QueryRow(ctx, "INSERT INTO organizations (name, slug) VALUES ('State Transition Org', 'state-trans-org') RETURNING id").Scan(&orgID)
	if err != nil {
		t.Fatalf("failed to create test org: %v", err)
	}
	defer adminPool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgID)

	var clientID string
	err = adminPool.QueryRow(ctx, "INSERT INTO clients (organization_id, name) VALUES ($1, 'State Client') RETURNING id", orgID).Scan(&clientID)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	var assetID string
	err = adminPool.QueryRow(ctx, "INSERT INTO assets (organization_id, client_id, name, url) VALUES ($1, $2, 'State Asset', 'https://example.com') RETURNING id", orgID, clientID).Scan(&assetID)
	if err != nil {
		t.Fatalf("failed to create asset: %v", err)
	}

	// Create monitor (initially pending, consecutive_failures = 0)
	mon, err := monitorRepo.CreateMonitor(ctx, orgID, CreateMonitorParams{
		AssetID:         assetID,
		Method:          "GET",
		IntervalSeconds: 60,
	})
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}
	if mon.Status != "pending" {
		t.Fatalf("expected initial status pending, got %s", mon.Status)
	}

	now := time.Now().UTC()

	// Step 1: 1st failed scheduled check -> suspected_down (consecutive_failures = 1)
	st1, err := monitorRepo.RecordScheduledCheckOutcome(ctx, orgID, mon.ID, false, now, now, 60)
	if err != nil {
		t.Fatalf("failed outcome 1: %v", err)
	}
	if st1 != "suspected_down" {
		t.Fatalf("expected suspected_down after 1st failure, got %s", st1)
	}

	// Verify in DB
	mCheck, _ := monitorRepo.GetMonitorByID(ctx, orgID, mon.ID)
	if mCheck.Status != "suspected_down" || mCheck.ConsecutiveFailures != 1 {
		t.Fatalf("expected DB status suspected_down with 1 failure, got %s (%d)", mCheck.Status, mCheck.ConsecutiveFailures)
	}

	// Step 2: 2nd consecutive failed scheduled check -> down (consecutive_failures = 2)
	st2, err := monitorRepo.RecordScheduledCheckOutcome(ctx, orgID, mon.ID, false, now, now, 60)
	if err != nil {
		t.Fatalf("failed outcome 2: %v", err)
	}
	if st2 != "down" {
		t.Fatalf("expected down after 2nd failure, got %s", st2)
	}

	mCheck, _ = monitorRepo.GetMonitorByID(ctx, orgID, mon.ID)
	if mCheck.Status != "down" || mCheck.ConsecutiveFailures != 2 {
		t.Fatalf("expected DB status down with 2 failures, got %s (%d)", mCheck.Status, mCheck.ConsecutiveFailures)
	}

	// Step 3: Recovery on successful scheduled check -> up (consecutive_failures reset to 0)
	st3, err := monitorRepo.RecordScheduledCheckOutcome(ctx, orgID, mon.ID, true, now, now, 60)
	if err != nil {
		t.Fatalf("failed outcome 3: %v", err)
	}
	if st3 != "up" {
		t.Fatalf("expected up on recovery, got %s", st3)
	}

	mCheck, _ = monitorRepo.GetMonitorByID(ctx, orgID, mon.ID)
	if mCheck.Status != "up" || mCheck.ConsecutiveFailures != 0 {
		t.Fatalf("expected DB status up with 0 failures, got %s (%d)", mCheck.Status, mCheck.ConsecutiveFailures)
	}

	// Step 4: Paused monitors must remain paused
	if err := monitorRepo.PauseMonitor(ctx, orgID, mon.ID); err != nil {
		t.Fatalf("failed to pause monitor: %v", err)
	}

	stPaused, err := monitorRepo.RecordScheduledCheckOutcome(ctx, orgID, mon.ID, false, now, now, 60)
	if err != nil {
		t.Fatalf("failed outcome on paused: %v", err)
	}
	if stPaused != "paused" {
		t.Fatalf("expected paused monitor to stay paused on failure, got %s", stPaused)
	}
}

// 4. Check Results Retention & Hourly Aggregation Tests
func TestRetentionPruningAndHourlyAggregation(t *testing.T) {
	ctx := context.Background()
	adminPool := openPoolAsRole(t, "clientpulse", "clientpulse_dev_password")
	defer adminPool.Close()

	appPool := openPoolAsRole(t, "clientpulse_app", "clientpulse_app_dev_password")
	defer appPool.Close()

	workerPool := openPoolAsRole(t, "clientpulse_worker", "clientpulse_worker_dev_password")
	defer workerPool.Close()

	resultRepo := NewCheckResultRepository(appPool)
	workerResultRepo := NewCheckResultRepository(workerPool)

	var orgID string
	err := adminPool.QueryRow(ctx, "INSERT INTO organizations (name, slug) VALUES ('Retention Test Org', 'retention-org') RETURNING id").Scan(&orgID)
	if err != nil {
		t.Fatalf("failed to create test org: %v", err)
	}
	defer adminPool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgID)

	var clientID, assetID, monitorID string
	_ = adminPool.QueryRow(ctx, "INSERT INTO clients (organization_id, name) VALUES ($1, 'C') RETURNING id", orgID).Scan(&clientID)
	_ = adminPool.QueryRow(ctx, "INSERT INTO assets (organization_id, client_id, name, url) VALUES ($1, $2, 'A', 'https://a.com') RETURNING id", orgID, clientID).Scan(&assetID)
	_ = adminPool.QueryRow(ctx, "INSERT INTO monitors (organization_id, asset_id, method, is_active, status) VALUES ($1, $2, 'GET', false, 'paused') RETURNING id", orgID, assetID).Scan(&monitorID)

	now := time.Now().UTC()
	recentTime := now.Add(-1 * time.Hour)
	expiredTime := now.Add(-95 * 24 * time.Hour) // 95 days old

	// Insert scheduled check result for current hour (should generate hourly aggregate)
	_, err = resultRepo.CreateCheckResult(ctx, CreateCheckResultParams{
		OrganizationID: orgID,
		MonitorID:      monitorID,
		ScheduledAt:    recentTime,
		StartedAt:      recentTime,
		CompletedAt:    recentTime,
		LatencyMs:      120,
		Result:         "success",
		WorkerID:       "w1",
		ExecutionType:  "scheduled",
	})
	if err != nil {
		t.Fatalf("failed to create recent check result: %v", err)
	}

	// Insert expired check result (95 days old)
	err = WithTenantTx(ctx, appPool, orgID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO check_results (
				organization_id, monitor_id, scheduled_at, started_at, completed_at,
				latency_ms, result, worker_id, execution_type, created_at
			) VALUES ($1, $2, $3, $3, $3, 200, 'success', 'w1', 'scheduled', $4)
		`, orgID, monitorID, expiredTime, expiredTime)
		return err
	})
	if err != nil {
		t.Fatalf("failed to insert expired check result: %v", err)
	}

	// Verify both check results exist initially
	var totalBefore int
	_ = adminPool.QueryRow(ctx, "SELECT COUNT(*) FROM check_results WHERE organization_id = $1", orgID).Scan(&totalBefore)
	if totalBefore < 2 {
		t.Fatalf("expected at least 2 results before pruning, got %d", totalBefore)
	}

	// Verify hourly aggregate was created for the recent check
	bucketHour := recentTime.Truncate(time.Hour)
	aggs, err := resultRepo.ListMonitorAggregates(ctx, orgID, monitorID, bucketHour.Add(-time.Hour), bucketHour.Add(time.Hour))
	if err != nil {
		t.Fatalf("failed to list aggregates: %v", err)
	}
	if len(aggs) == 0 {
		t.Fatalf("expected hourly aggregate to be present, got 0")
	}
	if aggs[0].TotalChecks != 1 || aggs[0].SuccessfulChecks != 1 {
		t.Fatalf("unexpected aggregate stats: %+v", aggs[0])
	}

	// Execute bounded batch pruning (90 days retention threshold) via dedicated worker role
	prunedCount, err := workerResultRepo.PruneExpiredCheckResults(ctx, 90, 500)
	if err != nil {
		t.Fatalf("failed to prune expired check results: %v", err)
	}
	if prunedCount == 0 {
		t.Fatalf("expected at least 1 expired record to be pruned, got 0")
	}
	t.Logf("Pruned %d expired records", prunedCount)

	// Verify recent record still exists
	var recentCount int
	_ = adminPool.QueryRow(ctx, "SELECT COUNT(*) FROM check_results WHERE organization_id = $1 AND created_at > $2", orgID, now.Add(-2*time.Hour)).Scan(&recentCount)
	if recentCount != 1 {
		t.Fatalf("expected recent check result to remain intact, found %d", recentCount)
	}

	// Verify hourly aggregate remains intact and queryable even after raw pruning
	aggsAfter, err := resultRepo.ListMonitorAggregates(ctx, orgID, monitorID, bucketHour.Add(-time.Hour), bucketHour.Add(time.Hour))
	if err != nil || len(aggsAfter) == 0 {
		t.Fatalf("hourly aggregates were lost after pruning!")
	}

	// Test idempotency: re-running cleanup should delete 0 rows
	rePruned, err := workerResultRepo.PruneExpiredCheckResults(ctx, 90, 500)
	if err != nil {
		t.Fatalf("failed to re-run pruning: %v", err)
	}
	if rePruned != 0 {
		t.Fatalf("expected 0 rows on immediate re-pruning, got %d", rePruned)
	}
}
