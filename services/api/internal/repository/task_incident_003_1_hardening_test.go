package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"clientpulse/services/api/internal/testhelper"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSchedulerObservabilityHardening(t *testing.T) {
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

	t.Run("TestCase1_DynamicStaleMonitorThresholds_60s_300s_3600s", func(t *testing.T) {
		slug := fmt.Sprintf("dynamic-thresh-%d", time.Now().UnixNano()%100000)
		orgID, clientID := createTenant(slug)
		defer adminDB.Pool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgID)

		var assetID string
		_ = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO assets (organization_id, client_id, name, url, is_active)
			VALUES ($1, $2, 'Dynamic Asset', 'https://dyn.example.com', true)
			RETURNING id
		`, orgID, clientID).Scan(&assetID)

		// 1. 60s monitor (threshold = max(3*60s, 180s) = 180s)
		// 1a. Overdue by 120s (2m) -> NOT stale yet
		var mon60Fresh string
		_ = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, next_check_at, last_checked_at)
			VALUES ($1, $2, 60, 10, 'up', true, NOW() - INTERVAL '120 seconds', NOW() - INTERVAL '180 seconds')
			RETURNING id
		`, orgID, assetID).Scan(&mon60Fresh)

		// 1b. Overdue by 240s (4m) -> STALE!
		var mon60Stale string
		_ = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, next_check_at, last_checked_at)
			VALUES ($1, $2, 60, 10, 'up', true, NOW() - INTERVAL '240 seconds', NOW() - INTERVAL '300 seconds')
			RETURNING id
		`, orgID, assetID).Scan(&mon60Stale)

		// 2. 300s (5m) monitor (threshold = max(3*300s, 180s) = 900s = 15m)
		// 2a. Overdue by 600s (10m) -> NOT stale yet (would be falsely flagged if fixed 5m threshold was used)
		var mon300Fresh string
		_ = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, next_check_at, last_checked_at)
			VALUES ($1, $2, 300, 10, 'up', true, NOW() - INTERVAL '600 seconds', NOW() - INTERVAL '900 seconds')
			RETURNING id
		`, orgID, assetID).Scan(&mon300Fresh)

		// 2b. Overdue by 1000s (16m40s) -> STALE!
		var mon300Stale string
		_ = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, next_check_at, last_checked_at)
			VALUES ($1, $2, 300, 10, 'up', true, NOW() - INTERVAL '1000 seconds', NOW() - INTERVAL '1300 seconds')
			RETURNING id
		`, orgID, assetID).Scan(&mon300Stale)

		// 3. 3600s (1h) monitor (threshold = max(3*3600s, 180s) = 10800s = 3 hours)
		// 3a. Overdue by 1200s (20m) -> NOT stale! (Prevents false positive where 15m fixed threshold would falsely trigger)
		var mon3600Fresh string
		_ = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, next_check_at, last_checked_at)
			VALUES ($1, $2, 3600, 10, 'up', true, NOW() - INTERVAL '1200 seconds', NOW() - INTERVAL '4800 seconds')
			RETURNING id
		`, orgID, assetID).Scan(&mon3600Fresh)

		// 3b. Overdue by 11500s (>3.1 hours) -> STALE!
		var mon3600Stale string
		_ = adminDB.Pool.QueryRow(ctx, `
			INSERT INTO monitors (organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, next_check_at, last_checked_at)
			VALUES ($1, $2, 3600, 10, 'up', true, NOW() - INTERVAL '11500 seconds', NOW() - INTERVAL '15100 seconds')
			RETURNING id
		`, orgID, assetID).Scan(&mon3600Stale)

		// Query dynamic stale monitors (multiplier 3, min tolerance 180s, lease timeout 60s)
		items, err := telemetryRepo.DetectStaleMonitors(ctx, 3, 180, 60, 100)
		if err != nil {
			t.Fatalf("DetectStaleMonitors failed: %v", err)
		}

		itemMap := make(map[string]*StaleMonitorItem)
		for _, item := range items {
			itemMap[item.MonitorID] = item
		}

		// Verify non-stale monitors are excluded
		if _, exists := itemMap[mon60Fresh]; exists {
			t.Fatalf("60s monitor overdue by 120s was falsely flagged as stale")
		}
		if _, exists := itemMap[mon300Fresh]; exists {
			t.Fatalf("300s monitor overdue by 10m was falsely flagged as stale")
		}
		if _, exists := itemMap[mon3600Fresh]; exists {
			t.Fatalf("3600s monitor overdue by 20m was falsely flagged as stale (fixed threshold defect)")
		}

		// Verify stale monitors are detected with proper effective thresholds
		item60, exists := itemMap[mon60Stale]
		if !exists {
			t.Fatalf("expected 60s monitor overdue by 240s to be detected as stale")
		}
		if item60.EffectiveThresholdSeconds != 180 {
			t.Fatalf("expected effective threshold 180 for 60s monitor, got %d", item60.EffectiveThresholdSeconds)
		}

		item300, exists := itemMap[mon300Stale]
		if !exists {
			t.Fatalf("expected 300s monitor overdue by 1000s to be detected as stale")
		}
		if item300.EffectiveThresholdSeconds != 900 {
			t.Fatalf("expected effective threshold 900 for 300s monitor, got %d", item300.EffectiveThresholdSeconds)
		}

		item3600, exists := itemMap[mon3600Stale]
		if !exists {
			t.Fatalf("expected 3600s monitor overdue by 11500s to be detected as stale")
		}
		if item3600.EffectiveThresholdSeconds != 10800 {
			t.Fatalf("expected effective threshold 10800 for 3600s monitor, got %d", item3600.EffectiveThresholdSeconds)
		}
	})

	t.Run("TestCase2_MultiWorkerLifecycleAndAggregation", func(t *testing.T) {
		cleanInstances := func() {
			_, _ = adminDB.Pool.Exec(ctx, "DELETE FROM scheduler_instances;")
		}
		cleanInstances()
		defer cleanInstances()

		worker1 := "worker-test-01"
		worker2 := "worker-test-02"
		inst1 := "inst-01"
		inst2 := "inst-02"

		// 1. No registered instances -> Unknown
		agg, err := telemetryRepo.GetSchedulerAggregateHealth(ctx, 20)
		if err != nil {
			t.Fatalf("failed getting agg health: %v", err)
		}
		if agg.AggregateState != "Unknown" || agg.TotalActiveInstances != 0 {
			t.Fatalf("expected Unknown for empty instances, got %s (total: %d)", agg.AggregateState, agg.TotalActiveInstances)
		}

		// 2. One healthy Worker -> Healthy
		now := time.Now().UTC()
		err = telemetryRepo.RecordPollCycle(ctx, RecordCycleParams{
			InstanceID:           inst1,
			WorkerID:             worker1,
			StartedAt:            now.Add(-10 * time.Second),
			PollStartedAt:        now.Add(-2 * time.Second),
			PollCompletedAt:      now.Add(-1 * time.Second),
			DurationMs:           15,
			DueMonitorsCount:     0,
			ClaimedMonitorsCount: 0,
			IsSuccess:            true,
		}, false)
		if err != nil {
			t.Fatalf("failed recording cycle: %v", err)
		}

		agg, err = telemetryRepo.GetSchedulerAggregateHealth(ctx, 20)
		if err != nil {
			t.Fatalf("failed getting agg health: %v", err)
		}
		if agg.AggregateState != "Healthy" || agg.HealthyInstances != 1 {
			t.Fatalf("expected Healthy for 1 active worker, got %s (healthy: %d)", agg.AggregateState, agg.HealthyInstances)
		}

		// 3. One healthy and one stale Worker -> Degraded
		oldTime := now.Add(-35 * time.Second)
		err = telemetryRepo.RecordPollCycle(ctx, RecordCycleParams{
			InstanceID:           inst2,
			WorkerID:             worker2,
			StartedAt:            oldTime.Add(-10 * time.Second),
			PollStartedAt:        oldTime.Add(-1 * time.Second),
			PollCompletedAt:      oldTime,
			DurationMs:           15,
			DueMonitorsCount:     0,
			ClaimedMonitorsCount: 0,
			IsSuccess:            true,
		}, false)
		if err != nil {
			t.Fatalf("failed recording inst2: %v", err)
		}

		agg, err = telemetryRepo.GetSchedulerAggregateHealth(ctx, 20)
		if err != nil {
			t.Fatalf("failed getting agg health: %v", err)
		}
		if agg.AggregateState != "Degraded" || agg.HealthyInstances != 1 || agg.StaleInstances != 1 {
			t.Fatalf("expected Degraded for 1 healthy + 1 stale, got %s (healthy=%d, stale=%d)", agg.AggregateState, agg.HealthyInstances, agg.StaleInstances)
		}

		// 4. All Workers stale -> Stale
		_, _ = adminDB.Pool.Exec(ctx, "UPDATE scheduler_instances SET last_poll_completed_at = NOW() - INTERVAL '30 seconds', updated_at = NOW() - INTERVAL '30 seconds' WHERE instance_id = $1", inst1)
		agg, err = telemetryRepo.GetSchedulerAggregateHealth(ctx, 20)
		if err != nil {
			t.Fatalf("failed getting agg health: %v", err)
		}
		if agg.AggregateState != "Stale" || agg.HealthyInstances != 0 || agg.StaleInstances != 2 {
			t.Fatalf("expected Stale when all stale, got %s (healthy=%d, stale=%d)", agg.AggregateState, agg.HealthyInstances, agg.StaleInstances)
		}

		// 5. Worker restart with new instance ID (supersedes old instance row for worker1)
		inst1New := "inst-01-restarted"
		err = telemetryRepo.RecordPollCycle(ctx, RecordCycleParams{
			InstanceID:           inst1New,
			WorkerID:             worker1,
			StartedAt:            time.Now().UTC(),
			PollStartedAt:        time.Now().UTC().Add(-1 * time.Second),
			PollCompletedAt:      time.Now().UTC(),
			DurationMs:           10,
			DueMonitorsCount:     0,
			ClaimedMonitorsCount: 0,
			IsSuccess:            true,
		}, false)
		if err != nil {
			t.Fatalf("failed recording restarted cycle: %v", err)
		}

		// Stop worker 2 gracefully -> should NOT cause stale alert
		_ = telemetryRepo.StopInstance(ctx, inst2)

		agg, err = telemetryRepo.GetSchedulerAggregateHealth(ctx, 20)
		if err != nil {
			t.Fatalf("failed getting agg health: %v", err)
		}
		if agg.AggregateState != "Healthy" || agg.HealthyInstances != 1 || agg.StaleInstances != 0 {
			t.Fatalf("expected Healthy after restart and stopping worker2, got %s (healthy=%d, stale=%d)", agg.AggregateState, agg.HealthyInstances, agg.StaleInstances)
		}
	})

	t.Run("TestCase3_LeastPrivilegeGlobalTelemetryAccess", func(t *testing.T) {
		// 1. clientpulse_app CANNOT SELECT directly from scheduler_instances
		_, err := appDB.Pool.Exec(ctx, "SELECT * FROM scheduler_instances;")
		if err == nil {
			t.Fatalf("SECURITY DEFECT: clientpulse_app was able to direct SELECT scheduler_instances")
		}

		// 2. clientpulse_app CANNOT SELECT directly from scheduler_cycle_history
		_, err = appDB.Pool.Exec(ctx, "SELECT * FROM scheduler_cycle_history;")
		if err == nil {
			t.Fatalf("SECURITY DEFECT: clientpulse_app was able to direct SELECT scheduler_cycle_history")
		}

		// 3. clientpulse_app CAN call get_scheduler_aggregate_health to get aggregate state
		agg, err := appDB.Pool.Query(ctx, "SELECT * FROM get_scheduler_aggregate_health(20);")
		if err != nil {
			t.Fatalf("clientpulse_app failed calling authorized procedure get_scheduler_aggregate_health: %v", err)
		}
		agg.Close()
	})
}

