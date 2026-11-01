package repository

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func createTestMonitorWithInterval(t *testing.T, pool *pgxpool.Pool, orgID, assetID string, createdAt time.Time, intervalSec int, status string, isActive bool) string {
	t.Helper()
	ctx := context.Background()
	var monID string
	err := pool.QueryRow(ctx, `
		INSERT INTO monitors (organization_id, asset_id, method, interval_seconds, timeout_seconds,
		                      expected_status_min, expected_status_max, status, is_active, created_at, updated_at)
		VALUES ($1, $2, 'GET', $3, 10, 200, 399, $4, $5, $6, $6)
		RETURNING id
	`, orgID, assetID, intervalSec, status, isActive, createdAt).Scan(&monID)
	if err != nil {
		t.Fatalf("failed creating test monitor: %v", err)
	}
	return monID
}

func insertScheduledCheckResult(t *testing.T, pool *pgxpool.Pool, orgID, monID string, at time.Time, result string, latencyMs int64) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO check_results (organization_id, monitor_id, scheduled_at, started_at, completed_at,
		                           latency_ms, result, execution_type, worker_id, created_at)
		VALUES ($1, $2, $3, $3, $3, $4, $5, 'scheduled', 'test-worker', $3)
	`, orgID, monID, at, latencyMs, result)
	if err != nil {
		t.Fatalf("failed inserting check result: %v", err)
	}
}

func insertManualCheckResult(t *testing.T, pool *pgxpool.Pool, orgID, monID string, at time.Time, result string, latencyMs int64) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO check_results (organization_id, monitor_id, scheduled_at, started_at, completed_at,
		                           latency_ms, result, execution_type, worker_id, created_at)
		VALUES ($1, $2, $3, $3, $3, $4, $5, 'manual', 'test-worker', $3)
	`, orgID, monID, at, latencyMs, result)
	if err != nil {
		t.Fatalf("failed inserting manual check result: %v", err)
	}
}

// TestScenario1_Active30DaysAllChecksRecorded verifies 100% coverage and 100% uptime when all scheduled checks ran successfully.
func TestScenario1_Active30DaysAllChecksRecorded(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("scen1-org-%d", time.Now().UnixNano())
	orgID, clientID := createTestOrgAndClient(t, adminPool, "Scenario 1 Org", slug)
	assetID := createTestAsset(t, adminPool, orgID, clientID, "App Server")

	periodStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC)
	intervalSec := 86400 // 1 check per day -> 30 expected checks

	monID := createTestMonitorWithInterval(t, adminPool, orgID, assetID, periodStart.Add(-24*time.Hour), intervalSec, "up", true)

	// Insert all 30 daily checks as success
	for day := 1; day <= 30; day++ {
		checkTime := time.Date(2026, 6, day, 12, 0, 0, 0, time.UTC)
		insertScheduledCheckResult(t, adminPool, orgID, monID, checkTime, "success", 120)
	}

	reportRepo := NewReportRepository(appPool, NewBrandingRepository(appPool))
	data, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
	if err != nil {
		t.Fatalf("failed calculating report: %v", err)
	}

	if len(data.Monitors) != 1 {
		t.Fatalf("expected 1 monitor, got %d", len(data.Monitors))
	}
	m := data.Monitors[0]
	if m.InsufficientData {
		t.Errorf("expected InsufficientData=false, got true")
	}
	if math.Abs(m.CoveragePercentage-100.0) > 0.01 {
		t.Errorf("expected 100.0%% coverage, got %.2f%%", m.CoveragePercentage)
	}
	if m.UptimePercentage == nil || math.Abs(*m.UptimePercentage-100.0) > 0.01 {
		t.Errorf("expected 100.0%% uptime, got %+v", m.UptimePercentage)
	}
	if m.TotalChecks != 30 || m.FailedChecks != 0 {
		t.Errorf("expected 30 total checks, 0 failed, got total=%d failed=%d", m.TotalChecks, m.FailedChecks)
	}
}

// TestScenario2_Active30DaysMissing5DaysChecks verifies coverage is strictly penalized by missing scheduled checks.
func TestScenario2_Active30DaysMissing5DaysChecks(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("scen2-org-%d", time.Now().UnixNano())
	orgID, clientID := createTestOrgAndClient(t, adminPool, "Scenario 2 Org", slug)
	assetID := createTestAsset(t, adminPool, orgID, clientID, "App Server")

	periodStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC)
	intervalSec := 86400 // 30 expected checks

	monID := createTestMonitorWithInterval(t, adminPool, orgID, assetID, periodStart.Add(-24*time.Hour), intervalSec, "up", true)

	// Record checks for only 25 days (days 1 through 25); days 26..30 are missing
	for day := 1; day <= 25; day++ {
		checkTime := time.Date(2026, 6, day, 12, 0, 0, 0, time.UTC)
		insertScheduledCheckResult(t, adminPool, orgID, monID, checkTime, "success", 115)
	}

	reportRepo := NewReportRepository(appPool, NewBrandingRepository(appPool))
	data, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
	if err != nil {
		t.Fatalf("failed calculating report: %v", err)
	}

	m := data.Monitors[0]
	expectedCoverage := (25.0 / 30.0) * 100.0 // ~83.33%
	if math.Abs(m.CoveragePercentage-expectedCoverage) > 0.5 {
		t.Errorf("expected ~%.2f%% coverage for 5 missing days, got %.2f%%", expectedCoverage, m.CoveragePercentage)
	}
	if m.UptimePercentage == nil || math.Abs(*m.UptimePercentage-100.0) > 0.01 {
		t.Errorf("expected 100.0%% uptime for monitored healthy periods, got %+v", m.UptimePercentage)
	}
	if m.TotalChecks != 25 {
		t.Errorf("expected 25 total checks recorded, got %d", m.TotalChecks)
	}
}

// TestScenario3_FailedExecutedChecksCountTowardCoverageNotUptime proves that failed checks
// contribute to monitoring coverage (monitoring executed) but do not count toward uptime.
func TestScenario3_FailedExecutedChecksCountTowardCoverageNotUptime(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("scen3-org-%d", time.Now().UnixNano())
	orgID, clientID := createTestOrgAndClient(t, adminPool, "Scenario 3 Org", slug)
	assetID := createTestAsset(t, adminPool, orgID, clientID, "API Server")

	periodStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC)
	intervalSec := 86400 // 30 expected checks

	monID := createTestMonitorWithInterval(t, adminPool, orgID, assetID, periodStart.Add(-24*time.Hour), intervalSec, "up", true)

	// 27 successful checks + 3 failed checks = 30 total executed checks
	for day := 1; day <= 27; day++ {
		checkTime := time.Date(2026, 6, day, 12, 0, 0, 0, time.UTC)
		insertScheduledCheckResult(t, adminPool, orgID, monID, checkTime, "success", 100)
	}
	for day := 28; day <= 30; day++ {
		checkTime := time.Date(2026, 6, day, 12, 0, 0, 0, time.UTC)
		insertScheduledCheckResult(t, adminPool, orgID, monID, checkTime, "failure", 0)
	}

	reportRepo := NewReportRepository(appPool, NewBrandingRepository(appPool))
	data, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
	if err != nil {
		t.Fatalf("failed calculating report: %v", err)
	}

	m := data.Monitors[0]
	// Coverage must be 100%: all 30 checks were executed and recorded!
	if math.Abs(m.CoveragePercentage-100.0) > 0.01 {
		t.Errorf("expected 100.0%% coverage because failed checks were executed, got %.2f%%", m.CoveragePercentage)
	}
	// Uptime must NOT be 100%: 3 days of failures = 27/30 = 90.0% uptime
	expectedUptime := (27.0 / 30.0) * 100.0
	if m.UptimePercentage == nil || math.Abs(*m.UptimePercentage-expectedUptime) > 0.5 {
		t.Errorf("expected %.2f%% uptime, got %+v", expectedUptime, m.UptimePercentage)
	}
	if m.TotalChecks != 30 || m.FailedChecks != 3 {
		t.Errorf("expected total=30 failed=3, got total=%d failed=%d", m.TotalChecks, m.FailedChecks)
	}
}

// TestScenario4_MonitorPausedForPartOfReportingPeriod verifies paused periods are excluded from
// expected checks and the uptime denominator.
func TestScenario4_MonitorPausedForPartOfReportingPeriod(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("scen4-org-%d", time.Now().UnixNano())
	orgID, clientID := createTestOrgAndClient(t, adminPool, "Scenario 4 Org", slug)
	assetID := createTestAsset(t, adminPool, orgID, clientID, "Paused Web App")

	periodStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC)
	intervalSec := 86400

	monID := createTestMonitorWithInterval(t, adminPool, orgID, assetID, periodStart.Add(-24*time.Hour), intervalSec, "up", true)

	// Paused for 10 days: from June 11 00:00:00 to June 21 00:00:00
	pauseAt := time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC)
	resumeAt := time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC)

	_, err := adminPool.Exec(ctx, `
		INSERT INTO audit_logs (organization_id, action, entity_type, entity_id, created_at)
		VALUES ($1, 'monitor.paused', 'monitor', $2, $3),
		       ($1, 'monitor.resumed', 'monitor', $2, $4)
	`, orgID, monID, pauseAt, resumeAt)
	if err != nil {
		t.Fatalf("failed inserting pause audit logs: %v", err)
	}

	// Active days: 1..10 (10 checks) and 21..30 (10 checks) = 20 executed checks total
	for day := 1; day <= 10; day++ {
		checkTime := time.Date(2026, 6, day, 12, 0, 0, 0, time.UTC)
		insertScheduledCheckResult(t, adminPool, orgID, monID, checkTime, "success", 120)
	}
	for day := 21; day <= 30; day++ {
		checkTime := time.Date(2026, 6, day, 12, 0, 0, 0, time.UTC)
		insertScheduledCheckResult(t, adminPool, orgID, monID, checkTime, "success", 120)
	}

	reportRepo := NewReportRepository(appPool, NewBrandingRepository(appPool))
	data, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
	if err != nil {
		t.Fatalf("failed calculating report: %v", err)
	}

	m := data.Monitors[0]
	// Paused days excluded from expected checks -> expected = 20, actual = 20 -> 100% coverage
	if math.Abs(m.CoveragePercentage-100.0) > 0.01 {
		t.Errorf("expected 100.0%% coverage for unpaused active period, got %.2f%%", m.CoveragePercentage)
	}
	// Paused days excluded from uptime denominator -> 100% uptime
	if m.UptimePercentage == nil || math.Abs(*m.UptimePercentage-100.0) > 0.01 {
		t.Errorf("expected 100.0%% uptime during active unpaused period, got %+v", m.UptimePercentage)
	}
	if m.TotalChecks != 20 {
		t.Errorf("expected 20 checks recorded, got %d", m.TotalChecks)
	}
}

// TestScenario5_MonitorCreatedHalfwayThroughReportingPeriod verifies monitor creation mid-period
// normalizes expected checks and uptime to the post-creation window.
func TestScenario5_MonitorCreatedHalfwayThroughReportingPeriod(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("scen5-org-%d", time.Now().UnixNano())
	orgID, clientID := createTestOrgAndClient(t, adminPool, "Scenario 5 Org", slug)
	assetID := createTestAsset(t, adminPool, orgID, clientID, "Mid-Month App")

	periodStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC)
	intervalSec := 86400

	// Created on June 16 00:00:00 (15 days remaining: June 16 through June 30)
	createdAt := time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC)
	monID := createTestMonitorWithInterval(t, adminPool, orgID, assetID, createdAt, intervalSec, "up", true)

	// Record checks for all 15 active days (days 16..30)
	for day := 16; day <= 30; day++ {
		checkTime := time.Date(2026, 6, day, 12, 0, 0, 0, time.UTC)
		insertScheduledCheckResult(t, adminPool, orgID, monID, checkTime, "success", 95)
	}

	reportRepo := NewReportRepository(appPool, NewBrandingRepository(appPool))
	data, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
	if err != nil {
		t.Fatalf("failed calculating report: %v", err)
	}

	m := data.Monitors[0]
	// 15 days active, 15 checks expected, 15 executed -> 100% coverage
	if math.Abs(m.CoveragePercentage-100.0) > 0.01 {
		t.Errorf("expected 100.0%% coverage for mid-month created monitor, got %.2f%%", m.CoveragePercentage)
	}
	// 100% uptime
	if m.UptimePercentage == nil || math.Abs(*m.UptimePercentage-100.0) > 0.01 {
		t.Errorf("expected 100.0%% uptime, got %+v", m.UptimePercentage)
	}
	if m.TotalChecks != 15 {
		t.Errorf("expected 15 checks, got %d", m.TotalChecks)
	}
}

// TestScenario6_MonitorDeletedOrDeactivatedDuringPeriod verifies monitor deactivation/deletion
// correctly truncates the eligible monitoring window.
func TestScenario6_MonitorDeletedOrDeactivatedDuringPeriod(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("scen6-org-%d", time.Now().UnixNano())
	orgID, clientID := createTestOrgAndClient(t, adminPool, "Scenario 6 Org", slug)
	assetID := createTestAsset(t, adminPool, orgID, clientID, "Deactivated App")

	periodStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC)
	intervalSec := 86400

	// Created before period, but asset archived on June 20 (20 days active: June 1 through June 20)
	archivedAt := time.Date(2026, 6, 20, 23, 59, 59, 0, time.UTC)
	monID := createTestMonitorWithInterval(t, adminPool, orgID, assetID, periodStart.Add(-24*time.Hour), intervalSec, "up", true)

	// Set asset archived_at to June 20
	_, err := adminPool.Exec(ctx, `UPDATE assets SET archived_at = $1 WHERE id = $2`, archivedAt, assetID)
	if err != nil {
		t.Fatalf("failed setting archived_at: %v", err)
	}

	// 20 checks recorded during days 1..20
	for day := 1; day <= 20; day++ {
		checkTime := time.Date(2026, 6, day, 12, 0, 0, 0, time.UTC)
		insertScheduledCheckResult(t, adminPool, orgID, monID, checkTime, "success", 110)
	}

	reportRepo := NewReportRepository(appPool, NewBrandingRepository(appPool))
	data, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
	if err != nil {
		t.Fatalf("failed calculating report: %v", err)
	}

	if len(data.Monitors) != 1 {
		t.Fatalf("expected 1 monitor, got %d", len(data.Monitors))
	}
	m := data.Monitors[0]
	// 20 days active, 20 expected checks, 20 recorded -> 100% coverage
	if math.Abs(m.CoveragePercentage-100.0) > 0.01 {
		t.Errorf("expected 100.0%% coverage for period prior to archiving, got %.2f%%", m.CoveragePercentage)
	}
	// 100% uptime
	if m.UptimePercentage == nil || math.Abs(*m.UptimePercentage-100.0) > 0.01 {
		t.Errorf("expected 100.0%% uptime, got %+v", m.UptimePercentage)
	}
	if m.TotalChecks != 20 {
		t.Errorf("expected 20 checks, got %d", m.TotalChecks)
	}
}

// TestScenario7_PeriodWithNoValidMonitoringData proves that 0 checks results in InsufficientData=true,
// nil UptimePercentage, and 0% coverage (no fabricated 100% uptime).
func TestScenario7_PeriodWithNoValidMonitoringData(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("scen7-org-%d", time.Now().UnixNano())
	orgID, clientID := createTestOrgAndClient(t, adminPool, "Scenario 7 Org", slug)
	assetID := createTestAsset(t, adminPool, orgID, clientID, "Zero Data App")

	periodStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC)
	intervalSec := 86400

	// Monitor created, but ZERO checks executed
	createTestMonitorWithInterval(t, adminPool, orgID, assetID, periodStart.Add(-24*time.Hour), intervalSec, "up", true)

	reportRepo := NewReportRepository(appPool, NewBrandingRepository(appPool))
	data, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
	if err != nil {
		t.Fatalf("failed calculating report: %v", err)
	}

	m := data.Monitors[0]
	if !m.InsufficientData {
		t.Errorf("expected InsufficientData=true when no checks exist, got false")
	}
	if m.UptimePercentage != nil {
		t.Errorf("expected nil UptimePercentage (not fabricated), got %v", *m.UptimePercentage)
	}
	if m.CoveragePercentage != 0.0 {
		t.Errorf("expected 0.0%% coverage for zero checks, got %.2f%%", m.CoveragePercentage)
	}
	if m.TotalChecks != 0 {
		t.Errorf("expected 0 checks, got %d", m.TotalChecks)
	}
}

// TestManualChecksStrictlyExcluded verifies manual checks do not count towards scheduled coverage or uptime.
func TestManualChecksStrictlyExcluded(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("scen-man-org-%d", time.Now().UnixNano())
	orgID, clientID := createTestOrgAndClient(t, adminPool, "Manual Check Org", slug)
	assetID := createTestAsset(t, adminPool, orgID, clientID, "Manual Only App")

	periodStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC)
	intervalSec := 86400

	monID := createTestMonitorWithInterval(t, adminPool, orgID, assetID, periodStart.Add(-24*time.Hour), intervalSec, "up", true)

	// Insert ONLY manual checks
	for day := 1; day <= 10; day++ {
		checkTime := time.Date(2026, 6, day, 12, 0, 0, 0, time.UTC)
		insertManualCheckResult(t, adminPool, orgID, monID, checkTime, "success", 100)
	}

	reportRepo := NewReportRepository(appPool, NewBrandingRepository(appPool))
	data, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
	if err != nil {
		t.Fatalf("failed calculating report: %v", err)
	}

	m := data.Monitors[0]
	if !m.InsufficientData {
		t.Errorf("expected InsufficientData=true when only manual checks exist, got false")
	}
	if m.UptimePercentage != nil {
		t.Errorf("expected nil UptimePercentage, got %v", *m.UptimePercentage)
	}
	if m.CoveragePercentage != 0.0 {
		t.Errorf("expected 0.0%% coverage when only manual checks exist, got %.2f%%", m.CoveragePercentage)
	}
	if m.TotalChecks != 0 {
		t.Errorf("expected 0 scheduled checks, got %d", m.TotalChecks)
	}
}

// TestOpenIncidentBoundaryOverlap verifies open incidents cap outage at covEnd.
func TestOpenIncidentBoundaryOverlap(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("scen-inc-org-%d", time.Now().UnixNano())
	orgID, clientID := createTestOrgAndClient(t, adminPool, "Open Incident Org", slug)
	assetID := createTestAsset(t, adminPool, orgID, clientID, "Open Incident App")

	periodStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 6, 30, 23, 59, 59, 0, time.UTC)
	intervalSec := 86400

	monID := createTestMonitorWithInterval(t, adminPool, orgID, assetID, periodStart.Add(-24*time.Hour), intervalSec, "down", true)

	// All 30 checks executed
	for day := 1; day <= 30; day++ {
		checkTime := time.Date(2026, 6, day, 12, 0, 0, 0, time.UTC)
		res := "success"
		if day >= 26 {
			res = "failure"
		}
		insertScheduledCheckResult(t, adminPool, orgID, monID, checkTime, res, 100)
	}

	// Incident opened on June 26 00:00:00 and is STILL OPEN (resolved_at IS NULL)
	incConfirmedAt := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	_, err := adminPool.Exec(ctx, `
		INSERT INTO incidents (organization_id, monitor_id, failure_category, status, summary, first_failure_at, confirmed_at)
		VALUES ($1, $2, 'http_5xx', 'open', 'Server crashed', $3, $3)
	`, orgID, monID, incConfirmedAt)
	if err != nil {
		t.Fatalf("failed inserting open incident: %v", err)
	}

	reportRepo := NewReportRepository(appPool, NewBrandingRepository(appPool))
	data, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
	if err != nil {
		t.Fatalf("failed calculating report: %v", err)
	}

	m := data.Monitors[0]
	// Outage from June 26 00:00:00 to June 30 23:59:59 = 5 days = 5/30 = ~16.67% outage -> ~83.33% uptime
	if m.UptimePercentage == nil {
		t.Fatalf("expected non-nil uptime percentage")
	}
	expectedUptime := (25.0 / 30.0) * 100.0 // ~83.33%
	if math.Abs(*m.UptimePercentage-expectedUptime) > 1.0 {
		t.Errorf("expected ~%.2f%% uptime with 5 days open incident, got %.2f%%", expectedUptime, *m.UptimePercentage)
	}
	if m.IncidentCount != 1 {
		t.Errorf("expected 1 incident, got %d", m.IncidentCount)
	}
}
