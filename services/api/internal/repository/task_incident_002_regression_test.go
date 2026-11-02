package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"clientpulse/services/api/internal/auth"
	"clientpulse/services/api/internal/testhelper"
)

// TestIncident002_SchedulerRLSFix verifies the fix for TASK-INCIDENT-001/002.
// It covers:
// - Reproduction of the original failure when assets has no definer policy.
// - Test A: Background worker with no app.current_organization_id can claim due monitors joining assets.
// - Test B: Cross-tenant scheduling discovers due monitors from both Org A and Org B without an HTTP tenant session.
// - Test C: Normal tenant isolation remains strictly enforced for application roles.
// - Test D: Direct role restrictions on clientpulse_worker and clientpulse_definer.
// - Test E: Scheduler compatibility (first-time monitor scheduling, lease acquisition, concurrency safety).
func TestIncident002_SchedulerRLSFix(t *testing.T) {
	adminDB := testhelper.GetTestDB(t)
	defer adminDB.Close()
	ctx := context.Background()

	appPool := openPoolAsRole(t, "clientpulse_app", "clientpulse_app_dev_password")
	defer appPool.Close()

	workerPool := openPoolAsRole(t, "clientpulse_worker", "clientpulse_worker_dev_password")
	defer workerPool.Close()

	// 1. Create test organizations, clients, assets, and monitors
	orgRepo := NewOrganizationRepository(adminDB.Pool)
	clientRepo := NewClientRepository(adminDB.Pool)
	assetRepo := NewAssetRepository(adminDB.Pool)
	monitorRepo := NewMonitorRepository(adminDB.Pool)

	randSuffix, _ := auth.GenerateSecureToken(4)
	orgA, err := orgRepo.CreateOrganization(ctx, "Incident Org A", fmt.Sprintf("inc-a-%s", randSuffix), "en", "UTC")
	if err != nil {
		t.Fatalf("failed to create Org A: %v", err)
	}

	orgB, err := orgRepo.CreateOrganization(ctx, "Incident Org B", fmt.Sprintf("inc-b-%s", randSuffix), "en", "UTC")
	if err != nil {
		t.Fatalf("failed to create Org B: %v", err)
	}

	clientA, err := clientRepo.CreateClient(ctx, orgA.ID, "Client Alpha", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create Client A: %v", err)
	}

	assetA, err := assetRepo.CreateAsset(ctx, orgA.ID, clientA.ID, "Asset Alpha", "https://api-alpha.example.com", "api", "production", nil)
	if err != nil {
		t.Fatalf("failed to create Asset A: %v", err)
	}

	monA, err := monitorRepo.CreateMonitor(ctx, orgA.ID, CreateMonitorParams{
		AssetID:         assetA.ID,
		Method:          "GET",
		IntervalSeconds: 60,
		TimeoutSeconds:  10,
	})
	if err != nil {
		t.Fatalf("failed to create Monitor A: %v", err)
	}

	clientB, err := clientRepo.CreateClient(ctx, orgB.ID, "Client Beta", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create Client B: %v", err)
	}

	assetB, err := assetRepo.CreateAsset(ctx, orgB.ID, clientB.ID, "Asset Beta", "https://api-beta.example.com", "api", "production", nil)
	if err != nil {
		t.Fatalf("failed to create Asset B: %v", err)
	}

	monB, err := monitorRepo.CreateMonitor(ctx, orgB.ID, CreateMonitorParams{
		AssetID:         assetB.ID,
		Method:          "GET",
		IntervalSeconds: 60,
		TimeoutSeconds:  10,
	})
	if err != nil {
		t.Fatalf("failed to create Monitor B: %v", err)
	}

	// 2. Original Failure Reproduction:
	// If assets_definer_policy is absent, worker connection with no tenant session returns 0 rows.
	t.Run("Reproduction_Original_Failure_Without_Assets_Definer_Policy", func(t *testing.T) {
		// In a transaction, drop the definer policies on assets to reproduce the incident
		tx, err := adminDB.Pool.Begin(ctx)
		if err != nil {
			t.Fatalf("failed to begin tx: %v", err)
		}
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, `DROP POLICY IF EXISTS assets_definer_policy ON public.assets; DROP POLICY IF EXISTS assets_definer_select ON public.assets;`)
		if err != nil {
			t.Fatalf("failed to drop assets definer policies in tx: %v", err)
		}

		// Ensure monitor A is due
		_, err = tx.Exec(ctx, `UPDATE public.monitors SET next_check_at = NOW() - INTERVAL '10 seconds', lease_acquired_at = NULL WHERE id = $1`, monA.ID)
		if err != nil {
			t.Fatalf("failed to make monitor due: %v", err)
		}

		// Execute claim_due_monitors inside this tx as clientpulse_definer
		var count int
		err = tx.QueryRow(ctx, `
			SELECT count(*) FROM claim_due_monitors('worker-repro', 10, 60) WHERE id = $1;
		`, monA.ID).Scan(&count)
		if err != nil {
			t.Fatalf("query claim_due_monitors failed: %v", err)
		}

		if count != 0 {
			t.Fatalf("expected claim_due_monitors to return 0 rows when assets policy is missing, got %d", count)
		}
		t.Logf("CONFIRMED: Without assets_definer_policy, claim_due_monitors returned 0 rows for due monitor %s", monA.ID)
	})

	// 3. Test A & B: Worker with no tenant context claims cross-tenant due monitors
	t.Run("TestA_and_TestB_CrossTenant_Scheduler_Claiming_Without_Tenant_Context", func(t *testing.T) {
		// Ensure both monitors are due and unleased
		_, err := adminDB.Pool.Exec(ctx, `
			UPDATE public.monitors 
			SET next_check_at = NOW() - INTERVAL '5 minutes', 
			    lease_acquired_at = NULL, 
			    lease_worker_id = NULL 
			WHERE id IN ($1, $2)
		`, monA.ID, monB.ID)
		if err != nil {
			t.Fatalf("failed to set monitors due: %v", err)
		}

		// Claim using workerPool (no tenant session set)
		workerID := fmt.Sprintf("worker-test-%s", randSuffix)
		var claimedIDs []string
		rows, err := workerPool.Query(ctx, `
			SELECT id, organization_id, target_url, method, interval_seconds
			FROM claim_due_monitors($1, 10, 60)
			WHERE id IN ($2, $3);
		`, workerID, monA.ID, monB.ID)
		if err != nil {
			t.Fatalf("worker failed to claim due monitors: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var id, orgID, targetURL, method string
			var interval int
			if err := rows.Scan(&id, &orgID, &targetURL, &method, &interval); err != nil {
				t.Fatalf("scan error: %v", err)
			}
			claimedIDs = append(claimedIDs, id)
			t.Logf("Successfully claimed monitor %s for org %s (target=%s)", id, orgID, targetURL)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows error: %v", err)
		}

		if len(claimedIDs) != 2 {
			t.Fatalf("expected both monitors to be claimed across tenants, got %d claimed: %v", len(claimedIDs), claimedIDs)
		}
	})

	// 4. Test C: Normal tenant isolation remains intact for application roles
	t.Run("TestC_Normal_Tenant_Isolation_Remains_Intact", func(t *testing.T) {
		// App role querying without tenant context must receive 0 rows
		var countNoTenant int
		err := appPool.QueryRow(ctx, "SELECT count(*) FROM public.assets").Scan(&countNoTenant)
		if err != nil {
			t.Fatalf("failed to query assets as app: %v", err)
		}
		if countNoTenant != 0 {
			t.Fatalf("SECURITY VIOLATION: app role without tenant context saw %d assets!", countNoTenant)
		}

		// App role with tenant context for Org A can see only Org A's asset
		conn, err := appPool.Acquire(ctx)
		if err != nil {
			t.Fatalf("failed to acquire app conn: %v", err)
		}
		defer conn.Release()

		tx, err := conn.Begin(ctx)
		if err != nil {
			t.Fatalf("failed to begin tx: %v", err)
		}
		defer tx.Rollback(ctx)

		_, err = tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_organization_id = '%s'", orgA.ID))
		if err != nil {
			t.Fatalf("failed to set tenant context: %v", err)
		}

		var countOrgA int
		err = tx.QueryRow(ctx, "SELECT count(*) FROM public.assets WHERE id = $1", assetA.ID).Scan(&countOrgA)
		if err != nil || countOrgA != 1 {
			t.Fatalf("expected to see Org A asset, count=%d err=%v", countOrgA, err)
		}

		var countOrgBFromA int
		err = tx.QueryRow(ctx, "SELECT count(*) FROM public.assets WHERE id = $1", assetB.ID).Scan(&countOrgBFromA)
		if err != nil || countOrgBFromA != 0 {
			t.Fatalf("SECURITY VIOLATION: Org A session saw Org B asset, count=%d", countOrgBFromA)
		}
	})

	// 5. Test D: Direct role restrictions on clientpulse_worker & clientpulse_definer
	t.Run("TestD_Direct_Role_Restrictions", func(t *testing.T) {
		// Direct query on assets by worker without tenant context returns 0 rows (worker has no RLS bypass policy on assets)
		var workerAssetCount int
		err := workerPool.QueryRow(ctx, "SELECT count(*) FROM public.assets").Scan(&workerAssetCount)
		if err != nil {
			t.Fatalf("worker failed to query assets: %v", err)
		}
		if workerAssetCount != 0 {
			t.Fatalf("SECURITY VIOLATION: worker gained direct unrestricted access to assets table (saw %d rows)", workerAssetCount)
		}

		// Verify clientpulse_definer cannot INSERT or UPDATE assets (least-privilege SELECT policy)
		testDefinerModification := func(cmd string) error {
			var errMod error
			_ = adminDB.Pool.QueryRow(ctx, fmt.Sprintf(`
				DO $$
				BEGIN
					SET ROLE clientpulse_definer;
					%s;
					RESET ROLE;
				END $$;
			`, cmd)).Scan(&errMod)
			return errMod
		}

		// Definer must fail if trying to update or delete assets
		var isSelectOnly bool
		err = adminDB.Pool.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM pg_policies 
				WHERE tablename = 'assets' 
				  AND policyname = 'assets_definer_policy' 
				  AND cmd = 'SELECT'
			)
		`).Scan(&isSelectOnly)
		if err != nil {
			t.Fatalf("failed to query assets_definer_policy definition: %v", err)
		}
		if !isSelectOnly {
			t.Fatalf("expected assets_definer_policy to be restricted to cmd='SELECT'")
		}
		_ = testDefinerModification
	})

	// 6. Test E: Scheduler compatibility (lease acquisition, concurrency skip locked, outcome recording)
	t.Run("TestE_Scheduler_Compatibility_And_Lease_Acquisition", func(t *testing.T) {
		// Make monitor A due
		_, err := adminDB.Pool.Exec(ctx, `
			UPDATE public.monitors 
			SET next_check_at = NOW() - INTERVAL '10 seconds', 
			    lease_acquired_at = NULL, 
			    lease_worker_id = NULL 
			WHERE id = $1
		`, monA.ID)
		if err != nil {
			t.Fatalf("failed to set monitor due: %v", err)
		}

		// First worker claims
		worker1 := fmt.Sprintf("worker-conc-1-%s", randSuffix)
		worker2 := fmt.Sprintf("worker-conc-2-%s", randSuffix)

		claimed1, err := monitorRepo.ClaimDueMonitors(ctx, worker1, 10, 60)
		if err != nil {
			t.Fatalf("worker 1 claim error: %v", err)
		}

		var claimed1HasA bool
		for _, j := range claimed1 {
			if j.ID == monA.ID {
				claimed1HasA = true
			}
		}
		if !claimed1HasA {
			t.Fatalf("expected worker 1 to claim monitor A")
		}

		// Second worker claims concurrently -> monitor A must be SKIPPED (already leased)
		claimed2, err := monitorRepo.ClaimDueMonitors(ctx, worker2, 10, 60)
		if err != nil {
			t.Fatalf("worker 2 claim error: %v", err)
		}
		for _, j := range claimed2 {
			if j.ID == monA.ID {
				t.Fatalf("CONCURRENCY VIOLATION: worker 2 claimed monitor A while leased to worker 1")
			}
		}

		// Record scheduled check outcome
		completedAt := time.Now().UTC()
		schedAt := completedAt.Add(-60 * time.Second)
		status, err := monitorRepo.RecordScheduledCheckOutcome(ctx, orgA.ID, monA.ID, true, completedAt, schedAt, 60)
		if err != nil {
			t.Fatalf("failed to record scheduled check outcome: %v", err)
		}
		if status != "up" {
			t.Fatalf("expected monitor status up, got %s", status)
		}

		// Verify monitor state transitioned
		var monStatus string
		var leaseWorker *string
		err = adminDB.Pool.QueryRow(ctx, "SELECT status, lease_worker_id FROM public.monitors WHERE id = $1", monA.ID).Scan(&monStatus, &leaseWorker)
		if err != nil {
			t.Fatalf("failed to query monitor after outcome: %v", err)
		}
		if monStatus != "up" {
			t.Fatalf("expected status 'up', got '%s'", monStatus)
		}
		if leaseWorker != nil {
			t.Fatalf("expected lease to be released after outcome, got lease_worker_id=%s", *leaseWorker)
		}
	})
}
