package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"clientpulse/services/api/internal/pdf"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func createTestUser(t *testing.T, pool *pgxpool.Pool, email string) string {
	t.Helper()
	ctx := context.Background()
	var userID string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, display_name, status, email_verified_at)
		VALUES ($1, 'hash', 'Test User', 'active', NOW())
		RETURNING id
	`, email).Scan(&userID)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return userID
}

func createTestAsset(t *testing.T, pool *pgxpool.Pool, orgID, clientID, name string) string {
	t.Helper()
	ctx := context.Background()
	var assetID string
	err := pool.QueryRow(ctx, `
		INSERT INTO assets (organization_id, client_id, name, type, url, environment, is_active)
		VALUES ($1, $2, $3, 'website', 'https://test.example', 'production', true)
		RETURNING id
	`, orgID, clientID, name).Scan(&assetID)
	if err != nil {
		t.Fatalf("failed to create test asset: %v", err)
	}
	return assetID
}

func TestMaintenanceActivityCRUDAndAuditTrail(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("maint-org-%d", time.Now().UnixNano())
	orgID, clientID := createTestOrgAndClient(t, adminPool, "Maintenance Org", slug)
	assetID := createTestAsset(t, adminPool, orgID, clientID, "Main eCommerce Site")
	userEmail := fmt.Sprintf("dev-%d@test.example", time.Now().UnixNano())
	userID := createTestUser(t, adminPool, userEmail)

	auditRepo := NewAuditRepository(appPool)
	maintRepo := NewMaintenanceRepository(appPool, auditRepo)

	// 1. Create Maintenance Activity
	dur := 45
	perfAt := time.Date(2026, 10, 15, 10, 30, 0, 0, time.UTC)
	act, err := maintRepo.CreateMaintenanceActivity(ctx, orgID, CreateMaintenanceParams{
		ClientID:        clientID,
		AssetID:         &assetID,
		Title:           "WordPress Core & Security Update",
		Description:     "Upgraded WordPress to latest point release and verified WooCommerce checkout.",
		Category:        "security_review",
		PerformedAt:     perfAt,
		DurationMinutes: &dur,
		CreatedBy:       userID,
	})
	if err != nil {
		t.Fatalf("failed creating maintenance activity: %v", err)
	}

	if act.ID == "" || act.Title != "WordPress Core & Security Update" {
		t.Fatalf("activity creation mismatch: %+v", act)
	}

	// 2. Read Activity
	fetched, err := maintRepo.GetMaintenanceActivity(ctx, orgID, act.ID)
	if err != nil {
		t.Fatalf("failed fetching activity: %v", err)
	}
	if fetched.ClientName == "" || fetched.AssetName == nil || *fetched.AssetName != "Main eCommerce Site" {
		t.Errorf("expected joined client and asset metadata, got: %+v", fetched)
	}

	// 3. Filter by Category & Date
	list, count, err := maintRepo.ListMaintenanceActivities(ctx, orgID, MaintenanceFilter{
		ClientID: &clientID,
		Category: &act.Category,
		Limit:    10,
	})
	if err != nil || count != 1 || len(list) != 1 {
		t.Fatalf("expected 1 activity in list, got count=%d, list=%d, err=%v", count, len(list), err)
	}

	// 4. Update Activity before report -> should succeed without audit flag
	newTitle := "Updated: WordPress Core Patches"
	updated, err := maintRepo.UpdateMaintenanceActivity(ctx, orgID, act.ID, UpdateMaintenanceParams{
		Title:     &newTitle,
		UpdatedBy: userID,
	})
	if err != nil {
		t.Fatalf("failed updating activity: %v", err)
	}
	if updated.Title != newTitle {
		t.Errorf("title not updated: %s", updated.Title)
	}

	// 5. Generate a report covering the activity date
	brandingRepo := NewBrandingRepository(appPool)
	reportRepo := NewReportRepository(appPool, brandingRepo)

	periodStart := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 10, 31, 23, 59, 59, 0, time.UTC)

	calcData, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
	if err != nil {
		t.Fatalf("failed calculating report data: %v", err)
	}

	rep, err := reportRepo.CreateReport(ctx, orgID, clientID, "monthly_care", periodStart, periodEnd, "UTC", "en", "Oct Report", *calcData, userID)
	if err != nil {
		t.Fatalf("failed creating report: %v", err)
	}
	if rep.VersionNumber != 1 {
		t.Fatalf("expected version 1, got %d", rep.VersionNumber)
	}

	// 6. Now edit the activity that is included in the generated report -> must log audit!
	postReportTitle := "Post-Report Modification"
	_, err = maintRepo.UpdateMaintenanceActivity(ctx, orgID, act.ID, UpdateMaintenanceParams{
		Title:     &postReportTitle,
		UpdatedBy: userID,
	})
	if err != nil {
		t.Fatalf("failed updating activity post-report: %v", err)
	}

	// Verify audit log entry exists in DB
	var auditCount int
	err = adminPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_logs
		WHERE organization_id = $1 AND action = 'maintenance_activity.updated_after_report'
		  AND entity_id = $2
	`, orgID, act.ID).Scan(&auditCount)
	if err != nil || auditCount == 0 {
		t.Fatalf("expected audit log entry for post-report activity edit, got count=%d, err=%v", auditCount, err)
	}
}

func TestReportSnapshotVersioningAndImmutability(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("snapshot-org-%d", time.Now().UnixNano())
	orgID, clientID := createTestOrgAndClient(t, adminPool, "Snapshot Org", slug)
	userEmail := fmt.Sprintf("author-%d@test.example", time.Now().UnixNano())
	userID := createTestUser(t, adminPool, userEmail)

	auditRepo := NewAuditRepository(appPool)
	maintRepo := NewMaintenanceRepository(appPool, auditRepo)
	brandingRepo := NewBrandingRepository(appPool)
	reportRepo := NewReportRepository(appPool, brandingRepo)

	periodStart := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 9, 30, 23, 59, 59, 0, time.UTC)

	// Create initial activity in September
	v1Title := "Initial Activity Version 1"
	act, err := maintRepo.CreateMaintenanceActivity(ctx, orgID, CreateMaintenanceParams{
		ClientID:    clientID,
		Title:       v1Title,
		Description: "Initial maintenance work description.",
		Category:    "backup",
		PerformedAt: time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC),
		CreatedBy:   userID,
	})
	if err != nil {
		t.Fatalf("failed to create activity: %v", err)
	}

	// 1. Generate Report Version 1
	calc1, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
	if err != nil {
		t.Fatalf("failed calculating report 1: %v", err)
	}

	repV1, err := reportRepo.CreateReport(ctx, orgID, clientID, "monthly_care", periodStart, periodEnd, "UTC", "en", "Sept Report V1", *calc1, userID)
	if err != nil {
		t.Fatalf("failed saving report v1: %v", err)
	}
	if repV1.VersionNumber != 1 {
		t.Fatalf("expected version 1, got %d", repV1.VersionNumber)
	}

	// 2. Edit the source activity
	v2Title := "Modified Activity Title for Version 2"
	_, err = maintRepo.UpdateMaintenanceActivity(ctx, orgID, act.ID, UpdateMaintenanceParams{
		Title:     &v2Title,
		UpdatedBy: userID,
	})
	if err != nil {
		t.Fatalf("failed editing activity: %v", err)
	}

	// 3. Generate Report Version 2 for the same period
	calc2, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
	if err != nil {
		t.Fatalf("failed calculating report 2: %v", err)
	}

	repV2, err := reportRepo.CreateReport(ctx, orgID, clientID, "monthly_care", periodStart, periodEnd, "UTC", "en", "Sept Report V2", *calc2, userID)
	if err != nil {
		t.Fatalf("failed saving report v2: %v", err)
	}
	if repV2.VersionNumber != 2 {
		t.Fatalf("expected version 2, got %d", repV2.VersionNumber)
	}

	// 4. Verify Version 1 snapshot is completely unchanged!
	repV1Recheck, err := reportRepo.GetReport(ctx, orgID, repV1.ID)
	if err != nil {
		t.Fatalf("failed fetching report v1: %v", err)
	}

	var snap1 pdf.ReportData
	if err := json.Unmarshal(repV1Recheck.SnapshotJSON, &snap1); err != nil {
		t.Fatalf("failed parsing v1 snapshot: %v", err)
	}

	if len(snap1.Maintenance) != 1 || snap1.Maintenance[0].Title != v1Title {
		t.Fatalf("SNAPSHOT IMMUTABILITY VIOLATION: Report v1 was altered by later activity edit! Got title: %s", snap1.Maintenance[0].Title)
	}

	// 5. Verify Version 2 snapshot contains the updated activity
	repV2Recheck, err := reportRepo.GetReport(ctx, orgID, repV2.ID)
	if err != nil {
		t.Fatalf("failed fetching report v2: %v", err)
	}

	var snap2 pdf.ReportData
	if err := json.Unmarshal(repV2Recheck.SnapshotJSON, &snap2); err != nil {
		t.Fatalf("failed parsing v2 snapshot: %v", err)
	}

	if len(snap2.Maintenance) != 1 || snap2.Maintenance[0].Title != v2Title {
		t.Fatalf("Report v2 did not reflect updated activity! Got: %s", snap2.Maintenance[0].Title)
	}
}

func TestDatabaseLevelReportSnapshotProtection(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("db-prot-org-%d", time.Now().UnixNano())
	orgID, clientID := createTestOrgAndClient(t, adminPool, "Snapshot Protection Org", slug)
	userID := createTestUser(t, adminPool, fmt.Sprintf("db-prot-%d@test.example", time.Now().UnixNano()))

	reportRepo := NewReportRepository(appPool, NewBrandingRepository(appPool))

	periodStart := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 8, 31, 23, 59, 59, 0, time.UTC)

	calc, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
	if err != nil {
		t.Fatalf("failed calculating report: %v", err)
	}

	rep, err := reportRepo.CreateReport(ctx, orgID, clientID, "monthly_care", periodStart, periodEnd, "UTC", "en", "August Report", *calc, userID)
	if err != nil {
		t.Fatalf("failed creating report: %v", err)
	}

	// 1. Attempt unauthorized SQL mutation of snapshot_json using clientpulse_app runtime role
	err = WithTenantTx(ctx, appPool, orgID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE reports
			SET snapshot_json = '{"tampered": true}'::jsonb
			WHERE id = $1 AND organization_id = $2
		`, rep.ID, orgID)
		return err
	})
	if err == nil {
		t.Fatalf("CRITICAL SECURITY DEFECT: clientpulse_app was able to mutate snapshot_json in PostgreSQL!")
	}
	if !strings.Contains(err.Error(), "immutable") {
		t.Errorf("expected trigger immutability error message, got: %v", err)
	}

	// 2. Attempt unauthorized SQL deletion using clientpulse_app runtime role
	err = WithTenantTx(ctx, appPool, orgID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			DELETE FROM reports
			WHERE id = $1 AND organization_id = $2
		`, rep.ID, orgID)
		return err
	})
	if err == nil {
		t.Fatalf("CRITICAL SECURITY DEFECT: clientpulse_app was able to delete historical report from PostgreSQL!")
	}

	// 3. Legitimate operation: Updating PDF storage key must succeed
	pdfKey := fmt.Sprintf("reports/%s/%s_v1.pdf", orgID, rep.ID)
	err = reportRepo.UpdateReportPDFStorageKey(ctx, orgID, rep.ID, pdfKey)
	if err != nil {
		t.Fatalf("legitimate UpdateReportPDFStorageKey failed: %v", err)
	}

	// Verify pdf_storage_key was updated
	fetched, err := reportRepo.GetReport(ctx, orgID, rep.ID)
	if err != nil {
		t.Fatalf("failed fetching report after pdf key update: %v", err)
	}
	if fetched.PDFStorageKey == nil || *fetched.PDFStorageKey != pdfKey {
		t.Errorf("expected pdf_storage_key %q, got %+v", pdfKey, fetched.PDFStorageKey)
	}
}

func TestConcurrentReportVersioning(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slug := fmt.Sprintf("concurrent-report-org-%d", time.Now().UnixNano())
	orgID, clientID := createTestOrgAndClient(t, adminPool, "Concurrent Report Org", slug)
	userID := createTestUser(t, adminPool, fmt.Sprintf("conc-u-%d@test.example", time.Now().UnixNano()))

	reportRepo := NewReportRepository(appPool, NewBrandingRepository(appPool))

	periodStart := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 9, 30, 23, 59, 59, 0, time.UTC)

	calc, err := reportRepo.CalculateReportData(ctx, orgID, clientID, periodStart, periodEnd, "UTC", "en")
	if err != nil {
		t.Fatalf("failed calculating report: %v", err)
	}

	const concurrencyCount = 10
	var wg sync.WaitGroup
	errCh := make(chan error, concurrencyCount)
	versionCh := make(chan int, concurrencyCount)

	startBarrier := make(chan struct{})

	for i := 0; i < concurrencyCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-startBarrier // Synchronize start of all 10 goroutines

			rep, err := reportRepo.CreateReport(
				ctx,
				orgID,
				clientID,
				"monthly_care",
				periodStart,
				periodEnd,
				"UTC",
				"en",
				fmt.Sprintf("Report Concurrent Run %d", workerID),
				*calc,
				userID,
			)
			if err != nil {
				errCh <- fmt.Errorf("worker %d failed: %w", workerID, err)
				return
			}
			versionCh <- rep.VersionNumber
		}(i)
	}

	// Release all workers simultaneously
	close(startBarrier)
	wg.Wait()
	close(errCh)
	close(versionCh)

	// Verify no errors occurred
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		t.Fatalf("concurrent CreateReport encountered %d errors: %v", len(errs), errs)
	}

	// Verify 10 distinct versions were assigned: 1..10
	versions := make(map[int]bool)
	for v := range versionCh {
		if versions[v] {
			t.Errorf("duplicate version number detected: %d", v)
		}
		versions[v] = true
	}

	if len(versions) != concurrencyCount {
		t.Errorf("expected %d distinct versions, got %d: %+v", concurrencyCount, len(versions), versions)
	}

	for v := 1; v <= concurrencyCount; v++ {
		if !versions[v] {
			t.Errorf("missing expected sequential version %d", v)
		}
	}
}

func TestSecureReportSharingAndPublicLookup(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slugA := fmt.Sprintf("share-org-a-%d", time.Now().UnixNano())
	slugB := fmt.Sprintf("share-org-b-%d", time.Now().UnixNano())
	orgA, clientA := createTestOrgAndClient(t, adminPool, "Share Org A", slugA)
	orgB, _ := createTestOrgAndClient(t, adminPool, "Share Org B", slugB)
	userID := createTestUser(t, adminPool, fmt.Sprintf("u-%d@test.example", time.Now().UnixNano()))

	brandingRepo := NewBrandingRepository(appPool)
	reportRepo := NewReportRepository(appPool, brandingRepo)

	periodStart := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 8, 31, 23, 59, 59, 0, time.UTC)

	calc, err := reportRepo.CalculateReportData(ctx, orgA, clientA, periodStart, periodEnd, "UTC", "en")
	if err != nil {
		t.Fatalf("failed calculating report: %v", err)
	}

	rep, err := reportRepo.CreateReport(ctx, orgA, clientA, "monthly_care", periodStart, periodEnd, "UTC", "en", "August Care Report", *calc, userID)
	if err != nil {
		t.Fatalf("failed creating report: %v", err)
	}

	// 1. Create Share Token with 7-day expiry
	exp := time.Now().UTC().Add(7 * 24 * time.Hour)
	tokenPlaintext, shareRecord, err := reportRepo.CreateShareToken(ctx, orgA, rep.ID, &exp, userID)
	if err != nil {
		t.Fatalf("failed creating share token: %v", err)
	}

	if len(tokenPlaintext) < 40 || !strings.HasPrefix(tokenPlaintext, "cp_sh_") {
		t.Fatalf("invalid share token format: %s", tokenPlaintext)
	}

	// 2. Unauthenticated Public Lookup
	pubView, _, err := reportRepo.LookupReportByShareToken(ctx, tokenPlaintext)
	if err != nil {
		t.Fatalf("failed looking up public report: %v", err)
	}
	if pubView.Title != "August Care Report" || pubView.VersionNumber != 1 {
		t.Fatalf("public view data mismatch: %+v", pubView)
	}

	// 3. Org B attempts to revoke Org A's share link -> must fail
	err = reportRepo.RevokeShareToken(ctx, orgB, shareRecord.ID)
	if err == nil {
		t.Fatalf("SECURITY VIOLATION: Org B was able to revoke Org A's share link!")
	}

	// 4. Org A revokes the share link
	err = reportRepo.RevokeShareToken(ctx, orgA, shareRecord.ID)
	if err != nil {
		t.Fatalf("failed revoking share token: %v", err)
	}

	// 5. Subsequent public lookup must immediately fail
	_, _, err = reportRepo.LookupReportByShareToken(ctx, tokenPlaintext)
	if !errors.Is(err, ErrShareTokenNotFound) {
		t.Fatalf("expected ErrShareTokenNotFound after revocation, got: %v", err)
	}

	// 6. Expired token test
	pastExp := time.Now().UTC().Add(-1 * time.Hour)
	expiredPlaintext, _, err := reportRepo.CreateShareToken(ctx, orgA, rep.ID, &pastExp, userID)
	if err != nil {
		t.Fatalf("failed creating expired token: %v", err)
	}
	_, _, err = reportRepo.LookupReportByShareToken(ctx, expiredPlaintext)
	if !errors.Is(err, ErrShareTokenNotFound) {
		t.Fatalf("expected ErrShareTokenNotFound for expired link, got: %v", err)
	}
}

func TestBrandingTenantIsolationAndContrast(t *testing.T) {
	adminPool, appPool, _ := setupTestRoles(t)
	ctx := context.Background()

	slugA := fmt.Sprintf("brand-a-%d", time.Now().UnixNano())
	slugB := fmt.Sprintf("brand-b-%d", time.Now().UnixNano())
	orgA, _ := createTestOrgAndClient(t, adminPool, "Brand Org A", slugA)
	orgB, _ := createTestOrgAndClient(t, adminPool, "Brand Org B", slugB)

	brandingRepo := NewBrandingRepository(appPool)

	// 1. Contrast calculation test
	ratio, accessible := CalculateContrastRatio("#0ea5e9") // Light sky blue
	if ratio < 1.0 {
		t.Errorf("invalid contrast ratio: %f", ratio)
	}
	_ = accessible

	darkColorRatio, darkAccessible := CalculateContrastRatio("#0f172a") // Slate 900
	if !darkAccessible || darkColorRatio < 4.5 {
		t.Errorf("dark color should have high contrast: %f, accessible=%v", darkColorRatio, darkAccessible)
	}

	// 2. Set branding for Org A
	dispNameA := "Starlight Digital"
	primaryColorA := "#1e40af"
	accentColorA := "#7c3aed"
	locA := "fa"

	savedA, err := brandingRepo.UpsertBranding(ctx, orgA, UpdateBrandingParams{
		DisplayName:         &dispNameA,
		PrimaryColor:        &primaryColorA,
		AccentColor:         &accentColorA,
		DefaultReportLocale: &locA,
	})
	if err != nil {
		t.Fatalf("failed saving branding for Org A: %v", err)
	}
	if savedA.DisplayName != dispNameA || savedA.DefaultReportLocale != "fa" {
		t.Fatalf("branding mismatch: %+v", savedA)
	}

	// 3. Org B fetches branding -> must NOT see Org A's data
	brandB, err := brandingRepo.GetBranding(ctx, orgB)
	if err != nil {
		t.Fatalf("failed fetching Org B branding: %v", err)
	}
	if brandB.DisplayName == dispNameA {
		t.Fatalf("SECURITY VIOLATION: Org B saw Org A's branding display name!")
	}
}
