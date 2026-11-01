package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"clientpulse/services/api/internal/pdf"
)

// TestReportResponseTimeRegression covers all 8 requirements specified in BUGFIX - Care Reports Generation Failure:
// 1. CalculateReportData successfully queries response-time metrics using the canonical schema (latency_ms).
// 2. Average response time is calculated correctly.
// 3. Median response time is calculated correctly.
// 4. Missing response-time values (e.g. latency <= 0 or failed checks) are handled safely.
// 5. Reports still work when there are no check results.
// 6. Report generation works for both English and Persian locales.
// 7. Existing uptime and monitoring coverage calculations remain correct.
// 8. The full report creation flow works against the actual PostgreSQL schema.
func TestReportResponseTimeRegression(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("rt-regress-org-%d", time.Now().UnixNano())
	orgID, clientID := createTestOrgAndClient(t, adminPool, "ResponseTime Regression Org", slug)
	userEmail := fmt.Sprintf("user-%d@test.example", time.Now().UnixNano())
	userID := createTestUser(t, adminPool, userEmail)

	assetID := createTestAsset(t, adminPool, orgID, clientID, "ResponseTime App")

	periodStart := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 10, 1, 1, 0, 0, 0, time.UTC) // 1 hour window

	// Monitor with 300s (5-minute) interval -> 12 expected slots in 1 hour
	monID := createTestMonitorWithInterval(t, adminPool, orgID, assetID, periodStart, 300, "up", true)

	brandingRepo := NewBrandingRepository(appPool)
	reportRepo := NewReportRepository(appPool, brandingRepo)

	// Step 1: Verify reports still work when there are NO check results (Requirement 5)
	t.Run("ZeroCheckResultsHandledSafely", func(t *testing.T) {
		data, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
		if err != nil {
			t.Fatalf("CalculateReportData failed with zero check results: %v", err)
		}
		if len(data.Monitors) != 1 {
			t.Fatalf("expected 1 monitor, got %d", len(data.Monitors))
		}
		var m *pdf.MonitorReportData
		for i := range data.Monitors {
			if data.Monitors[i].ID == monID {
				m = &data.Monitors[i]
				break
			}
		}
		if m == nil {
			t.Fatalf("expected to find monitor %s in report data", monID)
		}
		if m.AvgResponseTimeMs != nil {
			t.Errorf("expected AvgResponseTimeMs to be nil when no checks exist, got %v", *m.AvgResponseTimeMs)
		}
		if m.MedianResponseTimeMs != nil {
			t.Errorf("expected MedianResponseTimeMs to be nil when no checks exist, got %v", *m.MedianResponseTimeMs)
		}
		if !m.InsufficientData {
			t.Errorf("expected InsufficientData to be true when no checks exist")
		}
		if m.UptimePercentage != nil {
			t.Errorf("expected UptimePercentage to be nil when no checks exist, got %v", *m.UptimePercentage)
		}
	})

	// Step 2: Insert checks with distinct response times and verify canonical schema queries, avg, median (Requirements 1, 2, 3, 4)
	// We insert 6 checks:
	// - Slot 0 (00:00): success, latency_ms = 100
	// - Slot 1 (00:05): success, latency_ms = 200
	// - Slot 2 (00:10): success, latency_ms = 300
	// - Slot 3 (00:15): success, latency_ms = 400
	// - Slot 4 (00:20): success, latency_ms = 500
	// - Slot 5 (00:25): failure, latency_ms = 0 (Missing response time / failed check)
	// Successful checks: 5 checks with latencies [100, 200, 300, 400, 500]
	// Expected Average: (100+200+300+400+500) / 5 = 300 ms
	// Expected Median: 300 ms
	t.Run("CanonicalLatencyAverageAndMedianCalculation", func(t *testing.T) {
		insertScheduledCheckResult(t, adminPool, orgID, monID, periodStart.Add(0*time.Minute), "success", 100)
		insertScheduledCheckResult(t, adminPool, orgID, monID, periodStart.Add(5*time.Minute), "success", 200)
		insertScheduledCheckResult(t, adminPool, orgID, monID, periodStart.Add(10*time.Minute), "success", 300)
		insertScheduledCheckResult(t, adminPool, orgID, monID, periodStart.Add(15*time.Minute), "success", 400)
		insertScheduledCheckResult(t, adminPool, orgID, monID, periodStart.Add(20*time.Minute), "success", 500)
		insertScheduledCheckResult(t, adminPool, orgID, monID, periodStart.Add(25*time.Minute), "failure", 0)

		data, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
		if err != nil {
			t.Fatalf("CalculateReportData failed using canonical latency_ms schema: %v", err)
		}
		if len(data.Monitors) != 1 {
			t.Fatalf("expected 1 monitor, got %d", len(data.Monitors))
		}
		var m *pdf.MonitorReportData
		for i := range data.Monitors {
			if data.Monitors[i].ID == monID {
				m = &data.Monitors[i]
				break
			}
		}
		if m == nil {
			t.Fatalf("expected to find monitor %s in report data", monID)
		}

		if m.AvgResponseTimeMs == nil {
			t.Fatalf("expected AvgResponseTimeMs to be populated, got nil")
		}
		if *m.AvgResponseTimeMs != 300 {
			t.Errorf("expected AvgResponseTimeMs = 300, got %d", *m.AvgResponseTimeMs)
		}

		if m.MedianResponseTimeMs == nil {
			t.Fatalf("expected MedianResponseTimeMs to be populated, got nil")
		}
		if *m.MedianResponseTimeMs != 300 {
			t.Errorf("expected MedianResponseTimeMs = 300, got %d", *m.MedianResponseTimeMs)
		}

		if m.TotalChecks != 6 {
			t.Errorf("expected 6 total checks, got %d", m.TotalChecks)
		}
		if m.FailedChecks != 1 {
			t.Errorf("expected 1 failed check, got %d", m.FailedChecks)
		}
	})

	// Step 2b: Test Even count of latency values to verify median interpolation
	t.Run("EvenLatencyCountMedianInterpolation", func(t *testing.T) {
		evenAssetID := createTestAsset(t, adminPool, orgID, clientID, "Even Latency App")
		evenMonID := createTestMonitorWithInterval(t, adminPool, orgID, evenAssetID, periodStart, 300, "up", true)

		// 4 success checks: 100, 200, 300, 400
		// Avg = 250, Median = 250
		insertScheduledCheckResult(t, adminPool, orgID, evenMonID, periodStart.Add(0*time.Minute), "success", 100)
		insertScheduledCheckResult(t, adminPool, orgID, evenMonID, periodStart.Add(5*time.Minute), "success", 200)
		insertScheduledCheckResult(t, adminPool, orgID, evenMonID, periodStart.Add(10*time.Minute), "success", 300)
		insertScheduledCheckResult(t, adminPool, orgID, evenMonID, periodStart.Add(15*time.Minute), "success", 400)

		data, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
		if err != nil {
			t.Fatalf("CalculateReportData failed: %v", err)
		}
		var target *pdf.MonitorReportData
		for i := range data.Monitors {
			if data.Monitors[i].ID == evenMonID {
				target = &data.Monitors[i]
				break
			}
		}
		if target == nil {
			t.Fatalf("expected to find even latency monitor in report data")
		}
		if target.AvgResponseTimeMs == nil || *target.AvgResponseTimeMs != 250 {
			t.Errorf("expected avg response time 250, got %v", target.AvgResponseTimeMs)
		}
		if target.MedianResponseTimeMs == nil || *target.MedianResponseTimeMs != 250 {
			t.Errorf("expected median response time 250, got %v", target.MedianResponseTimeMs)
		}
	})

	// Step 3: Test English and Persian locales (Requirement 6)
	t.Run("EnglishAndPersianLocales", func(t *testing.T) {
		enData, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
		if err != nil {
			t.Fatalf("CalculateReportData failed for English locale: %v", err)
		}
		if enData.Locale != "en" {
			t.Errorf("expected locale en, got %s", enData.Locale)
		}
		if enData.Summary.ExecutiveStatement == "" {
			t.Errorf("expected non-empty executive statement in English")
		}

		faData, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "fa")
		if err != nil {
			t.Fatalf("CalculateReportData failed for Persian locale: %v", err)
		}
		if faData.Locale != "fa" {
			t.Errorf("expected locale fa, got %s", faData.Locale)
		}
		if faData.Summary.ExecutiveStatement == "" {
			t.Errorf("expected non-empty executive statement in Persian")
		}
		if faData.Monitors[0].LimitationNotice == "" {
			t.Errorf("expected non-empty limitation notice in Persian")
		}
	})

	// Step 4: Verify Uptime & Coverage calculations remain correct (Requirement 7)
	t.Run("UptimeAndCoverageIntegrity", func(t *testing.T) {
		data, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
		if err != nil {
			t.Fatalf("CalculateReportData failed: %v", err)
		}
		var m *pdf.MonitorReportData
		for i := range data.Monitors {
			if data.Monitors[i].ID == monID {
				m = &data.Monitors[i]
				break
			}
		}
		if m == nil {
			t.Fatalf("expected to find monitor %s in report data", monID)
		}
		// In 1 hour, expected checks = 12 (every 5 min: 00, 05, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60 -> 13 slots)
		// Total executed checks = 6 (5 success, 1 failure).
		if m.CoveragePercentage <= 0 || m.CoveragePercentage > 100 {
			t.Errorf("invalid coverage percentage: %.2f%%", m.CoveragePercentage)
		}
		if m.UptimePercentage == nil {
			t.Fatalf("expected non-nil uptime percentage")
		}
		// 5 successful checks, 1 failure -> uptime should be around (5/6)*100 = 83.33%
		if *m.UptimePercentage < 80.0 || *m.UptimePercentage > 85.0 {
			t.Errorf("expected uptime around 83.33%%, got %.2f%%", *m.UptimePercentage)
		}
	})

	// Step 5: Full report creation flow against actual PostgreSQL schema (Requirement 8)
	t.Run("FullReportCreationFlowAgainstPostgres", func(t *testing.T) {
		calc, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
		if err != nil {
			t.Fatalf("CalculateReportData failed: %v", err)
		}

		rep, err := reportRepo.CreateReport(
			ctx, orgID, clientID, "monthly_care",
			periodStart, periodEnd, "UTC", "en",
			"Care Report October 2026", *calc, userID,
		)
		if err != nil {
			t.Fatalf("CreateReport failed against real PostgreSQL schema: %v", err)
		}

		if rep.ID == "" {
			t.Errorf("expected generated report ID to be non-empty")
		}
		if rep.VersionNumber != 1 {
			t.Errorf("expected report version 1, got %d", rep.VersionNumber)
		}

		// Verify snapshot retrieved from database contains response time metrics
		retrieved, err := reportRepo.GetReport(ctx, orgID, rep.ID)
		if err != nil {
			t.Fatalf("GetReportByID failed: %v", err)
		}
		var snap pdf.ReportData
		if err := json.Unmarshal(retrieved.SnapshotJSON, &snap); err != nil {
			t.Fatalf("failed unmarshaling SnapshotJSON: %v", err)
		}
		var savedMon *pdf.MonitorReportData
		for i := range snap.Monitors {
			if snap.Monitors[i].ID == monID {
				savedMon = &snap.Monitors[i]
				break
			}
		}
		if savedMon == nil {
			t.Fatalf("expected to find monitor %s in saved snapshot", monID)
		}
		if savedMon.AvgResponseTimeMs == nil || *savedMon.AvgResponseTimeMs != 300 {
			t.Errorf("expected saved snapshot AvgResponseTimeMs = 300, got %v", savedMon.AvgResponseTimeMs)
		}
		if savedMon.MedianResponseTimeMs == nil || *savedMon.MedianResponseTimeMs != 300 {
			t.Errorf("expected saved snapshot MedianResponseTimeMs = 300, got %v", savedMon.MedianResponseTimeMs)
		}
	})
}
