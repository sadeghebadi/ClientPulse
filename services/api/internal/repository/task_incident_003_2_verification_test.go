package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"clientpulse/services/api/internal/testhelper"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTaskIncident003_2_SchedulerHealthStateAndLeaseSemantics(t *testing.T) {
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
	telemetryRepo := NewSchedulerTelemetryRepository(workerPool)

	cleanInstances := func() {
		_, _ = adminDB.Pool.Exec(ctx, "DELETE FROM scheduler_instances;")
	}
	cleanInstances()
	defer cleanInstances()

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

	t.Run("Section2_MultiWorkerAggregateStateMatrix", func(t *testing.T) {
		// Test cases matching the mandatory table in Section 2:
		// 1. Healthy + Healthy -> Healthy
		// 2. Healthy + Degraded -> Degraded
		// 3. Healthy + Stale -> Degraded
		// 4. Degraded + Degraded -> Degraded
		// 5. Degraded + Stale -> Degraded
		// 6. Stale + Stale -> Stale
		// 7. Stopped + Stopped -> Unknown
		// 8. Unknown + No instance -> Unknown

		setWorkerState := func(instID, workerID, state string) {
			now := time.Now().UTC()
			var lastPollCompletedAt *time.Time
			var lastSuccessfulPollAt *time.Time
			var startedAt time.Time
			consecutiveErrors := 0
			status := "active"

			switch state {
			case "Healthy":
				startedAt = now.Add(-10 * time.Minute)
				tVal := now.Add(-1 * time.Second)
				lastPollCompletedAt = &tVal
				lastSuccessfulPollAt = &tVal
				consecutiveErrors = 0
			case "Degraded":
				startedAt = now.Add(-10 * time.Minute)
				tVal := now.Add(-1 * time.Second)
				lastPollCompletedAt = &tVal
				consecutiveErrors = 4 // >= 3
			case "Stale":
				startedAt = now.Add(-10 * time.Minute)
				tVal := now.Add(-35 * time.Second) // > 20s
				lastPollCompletedAt = &tVal
			case "Stopped":
				startedAt = now.Add(-10 * time.Minute)
				tVal := now.Add(-1 * time.Second)
				lastPollCompletedAt = &tVal
				status = "stopped"
			case "Unknown":
				startedAt = now.Add(-2 * time.Second) // within 20s startup
				status = "active"
			}

			query := `
				INSERT INTO scheduler_instances (
					instance_id, worker_id, started_at, last_poll_started_at,
					last_poll_completed_at, last_successful_poll_at, consecutive_errors,
					status, updated_at
				)
				VALUES ($1, $2, $3, $4, $4, $5, $6, $7, NOW())
				ON CONFLICT (instance_id) DO UPDATE SET
					worker_id = EXCLUDED.worker_id,
					started_at = EXCLUDED.started_at,
					last_poll_started_at = EXCLUDED.last_poll_started_at,
					last_poll_completed_at = EXCLUDED.last_poll_completed_at,
					last_successful_poll_at = EXCLUDED.last_successful_poll_at,
					consecutive_errors = EXCLUDED.consecutive_errors,
					status = EXCLUDED.status,
					updated_at = NOW();
			`
			_, err := adminDB.Pool.Exec(ctx, query, instID, workerID, startedAt, lastPollCompletedAt, lastSuccessfulPollAt, consecutiveErrors, status)
			if err != nil {
				t.Fatalf("failed setting worker state %s: %v", state, err)
			}
		}

		testMatrix := []struct {
			name        string
			stateA      string
			stateB      string
			expectedAgg string
		}{
			{"Healthy_Healthy", "Healthy", "Healthy", "Healthy"},
			{"Healthy_Degraded", "Healthy", "Degraded", "Degraded"},
			{"Healthy_Stale", "Healthy", "Stale", "Degraded"},
			{"Degraded_Degraded", "Degraded", "Degraded", "Degraded"},
			{"Degraded_Stale", "Degraded", "Stale", "Degraded"},
			{"Stale_Stale", "Stale", "Stale", "Stale"},
			{"Stopped_Stopped", "Stopped", "Stopped", "Unknown"},
		}

		for _, tc := range testMatrix {
			t.Run(tc.name, func(t *testing.T) {
				cleanInstances()
				setWorkerState("inst-A", "worker-A", tc.stateA)
				setWorkerState("inst-B", "worker-B", tc.stateB)

				agg, err := telemetryRepo.GetSchedulerAggregateHealth(ctx, 20)
				if err != nil {
					t.Fatalf("GetSchedulerAggregateHealth failed: %v", err)
				}
				if agg.AggregateState != tc.expectedAgg {
					t.Fatalf("[%s] expected aggregate state '%s', got '%s' (total=%d, healthy=%d, stale=%d, degraded=%d)",
						tc.name, tc.expectedAgg, agg.AggregateState, agg.TotalActiveInstances, agg.HealthyInstances, agg.StaleInstances, agg.DegradedInstances)
				}
			})
		}

		// 8. Unknown + No instance -> Unknown
		t.Run("Unknown_NoInstance", func(t *testing.T) {
			cleanInstances()
			setWorkerState("inst-A", "worker-A", "Unknown")
			agg, err := telemetryRepo.GetSchedulerAggregateHealth(ctx, 20)
			if err != nil {
				t.Fatalf("GetSchedulerAggregateHealth failed: %v", err)
			}
			if agg.AggregateState != "Unknown" {
				t.Fatalf("expected Unknown state, got %s", agg.AggregateState)
			}
		})
	})

	t.Run("Section4_LeaseTimeoutAuthoritativeConfiguration", func(t *testing.T) {
		slug := fmt.Sprintf("lease-cfg-%d", time.Now().UnixNano()%100000)
		orgID, assetID := createTenantAsset(slug)
		defer adminDB.Pool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgID)

		// Create a monitor leased 90 seconds ago
		// If lease timeout is 60s -> lease is EXPIRED (overdue, detected as stale)
		// If lease timeout is 120s -> lease is ACTIVE (valid, NOT detected as stale)
		var monID string
		err := adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (
				organization_id, asset_id, interval_seconds, timeout_seconds,
				status, is_active, next_check_at, last_checked_at,
				lease_acquired_at, lease_worker_id
			)
			VALUES (
				$1, $2, 60, 10, 'up', true,
				NOW() - INTERVAL '300 seconds', NOW() - INTERVAL '360 seconds',
				NOW() - INTERVAL '90 seconds', 'worker-holding-lease'
			)
			RETURNING id
		`, orgID, assetID).Scan(&monID)
		if err != nil {
			t.Fatalf("failed inserting leased monitor: %v", err)
		}

		// 1. With lease timeout = 120 seconds:
		// Lease acquired 90s ago is within 120s -> Active valid lease suppresses premature warning!
		items120, err := telemetryRepo.DetectStaleMonitors(ctx, 3, 180, 120, 100)
		if err != nil {
			t.Fatalf("DetectStaleMonitors(120s) failed: %v", err)
		}
		for _, it := range items120 {
			if it.MonitorID == monID {
				t.Fatalf("monitor with active lease (90s old vs 120s timeout) was falsely flagged as stale")
			}
		}

		// 2. With lease timeout = 60 seconds:
		// Lease acquired 90s ago has expired (>60s) -> Must NOT be suppressed indefinitely!
		items60, err := telemetryRepo.DetectStaleMonitors(ctx, 3, 180, 60, 100)
		if err != nil {
			t.Fatalf("DetectStaleMonitors(60s) failed: %v", err)
		}
		foundExpired := false
		for _, it := range items60 {
			if it.MonitorID == monID {
				foundExpired = true
				break
			}
		}
		if !foundExpired {
			t.Fatalf("expected monitor with expired lease (90s old vs 60s timeout) to be flagged as stale")
		}
	})

	t.Run("Section5_FirstCheckAndEdgeCases", func(t *testing.T) {
		slug := fmt.Sprintf("edge-cases-%d", time.Now().UnixNano()%100000)
		orgID, assetID := createTenantAsset(slug)
		defer adminDB.Pool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgID)

		// 1. New monitor with NULL last_checked_at (first check)
		// Overdue beyond threshold (e.g. 250s overdue when threshold is 180s)
		var monFirstCheckStale string
		_ = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, next_check_at, last_checked_at)
			VALUES ($1, $2, 60, 10, 'pending', true, NOW() - INTERVAL '250 seconds', NULL)
			RETURNING id
		`, orgID, assetID).Scan(&monFirstCheckStale)

		// 2. Monitor with NULL next_check_at -> Must be safely ignored (not crash or false flag)
		var monNullNextCheck string
		_ = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, next_check_at, last_checked_at)
			VALUES ($1, $2, 60, 10, 'up', true, NULL, NOW() - INTERVAL '1 hour')
			RETURNING id
		`, orgID, assetID).Scan(&monNullNextCheck)

		items, err := telemetryRepo.DetectStaleMonitors(ctx, 3, 180, 60, 100)
		if err != nil {
			t.Fatalf("DetectStaleMonitors failed: %v", err)
		}

		itemMap := make(map[string]*StaleMonitorItem)
		for _, it := range items {
			itemMap[it.MonitorID] = it
		}

		// Verify first check stale is detected and flagged as is_first_check = true
		if it, exists := itemMap[monFirstCheckStale]; !exists {
			t.Fatalf("expected first-check overdue monitor to be flagged as stale")
		} else if !it.IsFirstCheck {
			t.Fatalf("expected is_first_check to be true for first check monitor")
		}

		// Verify NULL next_check_at is not falsely flagged
		if _, exists := itemMap[monNullNextCheck]; exists {
			t.Fatalf("monitor with NULL next_check_at was falsely flagged as stale")
		}
	})
}

