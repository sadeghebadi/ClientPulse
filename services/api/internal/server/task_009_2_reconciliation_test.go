package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"clientpulse/services/api/internal/auth"
	"clientpulse/services/api/internal/database"
)

func TestTask009_2_DatabaseAndBootstrapReconciliation(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()

	suffix, _ := auth.GenerateSecureToken(4)
	adminEmail := fmt.Sprintf("pa-rec-%s@pulse.test", suffix)
	adminSignup := createTestUser(t, env, adminEmail, "Recon Admin", "Pulse Recon")
	_, err := env.adminRepo.GrantAdmin(context.Background(), adminSignup.User.ID, nil, "Reconciliation test setup")
	if err != nil {
		t.Fatalf("grant admin failed: %v", err)
	}

	sessionCookie := adminSignup.RawSession
	csrfCookie := adminSignup.RawCSRF
	csrfHeader := adminSignup.RawCSRF
	orgID := adminSignup.Organization.ID

	ctx := context.Background()

	// 1. Draft published_at is NULL & price can be NULL (unconfigured)
	planSlug := fmt.Sprintf("recon-plan-%s", strings.ToLower(suffix))
	createPlanPayload := map[string]interface{}{
		"name":                       "Recon Draft Plan",
		"slug":                       planSlug,
		"description":                "Plan for testing lifecycle",
		"price_amount":               nil,
		"currency":                   nil,
		"billing_interval":           nil,
		"max_clients":                10,
		"max_assets":                 20,
		"max_monitors":               20,
		"max_heartbeats":             10,
		"max_members":                5,
		"min_check_interval_seconds": 60,
		"retention_days":             30,
		"custom_branding_enabled":    false,
		"public_reports_enabled":     true,
	}

	resp := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans", createPlanPayload, sessionCookie, csrfCookie, orgID, csrfHeader)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for draft plan, got %d: %s", resp.Code, resp.Body.String())
	}

	var planRes struct {
		Plan struct {
			ID   string `json:"id"`
			Slug string `json:"slug"`
		} `json:"plan"`
		Version struct {
			ID          string `json:"id"`
			Status      string `json:"status"`
			PublishedAt *time.Time `json:"published_at"`
			PriceAmount *int64 `json:"price_amount"`
		} `json:"version"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &planRes); err != nil {
		t.Fatalf("unmarshal plan response failed: %v", err)
	}

	if planRes.Version.PublishedAt != nil {
		t.Fatalf("expected draft published_at to be NULL, got %v", planRes.Version.PublishedAt)
	}
	if planRes.Version.PriceAmount != nil {
		t.Fatalf("expected unconfigured price_amount to be nil, got %v", *planRes.Version.PriceAmount)
	}

	// Verify directly in DB that Draft cannot be accidentally published through database default
	var dbStatus string
	var dbPublishedAt *time.Time
	var dbPriceAmount *int64
	var dbCurrency, dbBillingInterval *string
	err = env.dbPool.Pool.QueryRow(ctx, `
		SELECT status, published_at, price_amount, currency, billing_interval
		FROM subscription_plan_versions WHERE id = $1
	`, planRes.Version.ID).Scan(&dbStatus, &dbPublishedAt, &dbPriceAmount, &dbCurrency, &dbBillingInterval)
	if err != nil {
		t.Fatalf("query version from db: %v", err)
	}
	if dbStatus != "draft" {
		t.Fatalf("expected DB status to be 'draft', got %s", dbStatus)
	}
	if dbPublishedAt != nil {
		t.Fatalf("expected DB published_at to be NULL, got %v", dbPublishedAt)
	}
	if dbPriceAmount != nil || dbCurrency != nil || dbBillingInterval != nil {
		t.Fatalf("expected price, currency, and interval to be NULL in DB for unconfigured draft")
	}

	// 2. Draft can be edited
	updateDraftPayload := map[string]interface{}{
		"name":                       "Recon Draft Plan Updated",
		"description":                "Updated description",
		"price_amount":               0, // explicit zero price
		"currency":                   "USD",
		"billing_interval":           "monthly",
		"max_clients":                15,
		"max_assets":                 25,
		"max_monitors":               25,
		"max_heartbeats":             15,
		"max_members":                10,
		"min_check_interval_seconds": 60,
		"retention_days":             30,
		"custom_branding_enabled":    true,
		"public_reports_enabled":     true,
	}

	updateResp := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/plans/"+planRes.Plan.ID, updateDraftPayload, sessionCookie, csrfCookie, orgID, csrfHeader)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("expected 200 OK updating draft plan, got %d: %s", updateResp.Code, updateResp.Body.String())
	}

	// Verify explicit zero price differs from NULL
	err = env.dbPool.Pool.QueryRow(ctx, `SELECT price_amount FROM subscription_plan_versions WHERE id = $1`, planRes.Version.ID).Scan(&dbPriceAmount)
	if err != nil || dbPriceAmount == nil || *dbPriceAmount != 0 {
		t.Fatalf("expected explicit zero price (0), got %v (err: %v)", dbPriceAmount, err)
	}

	// 3. Invalid ISO currency code is rejected
	invalidCurrPayload := map[string]interface{}{
		"name":                       "Invalid Currency Plan",
		"slug":                       fmt.Sprintf("curr-plan-%s", strings.ToLower(suffix)),
		"description":                "Invalid currency test",
		"price_amount":               1000,
		"currency":                   "AAA", // Invalid 3-letter code
		"billing_interval":           "monthly",
		"max_clients":                10,
		"max_assets":                 10,
		"max_monitors":               10,
		"max_heartbeats":             5,
		"max_members":                5,
		"min_check_interval_seconds": 60,
		"retention_days":             30,
	}
	currResp := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans", invalidCurrPayload, sessionCookie, csrfCookie, orgID, csrfHeader)
	if currResp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for invalid currency 'AAA', got %d: %s", currResp.Code, currResp.Body.String())
	}

	// Valid ISO currencies (USD, EUR, GBP, JPY) are accepted
	validCurrPayload := invalidCurrPayload
	validCurrPayload["currency"] = "EUR"
	validResp := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans", validCurrPayload, sessionCookie, csrfCookie, orgID, csrfHeader)
	if validResp.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for valid currency 'EUR', got %d: %s", validResp.Code, validResp.Body.String())
	}

	// 4. Publish Draft and verify immutability
	pubResp := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans/"+planRes.Plan.ID+"/publish", nil, sessionCookie, csrfCookie, orgID, csrfHeader)
	if pubResp.Code != http.StatusOK {
		t.Fatalf("expected 200 OK publishing plan, got %d: %s", pubResp.Code, pubResp.Body.String())
	}

	// Direct DB attempt to modify published version must fail due to trigger
	_, err = env.dbPool.Pool.Exec(ctx, "UPDATE subscription_plan_versions SET max_clients = 999 WHERE id = $1", planRes.Version.ID)
	if err == nil {
		t.Fatalf("expected error modifying published version via DB, got nil")
	}
	if !strings.Contains(err.Error(), "Cannot modify published plan version") {
		t.Fatalf("expected 'Cannot modify published plan version' error, got: %v", err)
	}

	// 5. Subscription expiration uses a single consistent field (ends_at)
	var colCount int
	err = env.dbPool.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.columns 
		WHERE table_name = 'organization_subscriptions' AND column_name = 'expires_at'
	`).Scan(&colCount)
	if err != nil {
		t.Fatalf("query column count: %v", err)
	}
	if colCount != 0 {
		t.Fatalf("expected 'expires_at' column to NOT exist on organization_subscriptions, found %d", colCount)
	}

	// 6. Verify intended initial Platform Admin account state
	var intendedUserID string
	var intendedStatus string
	var emailVerifiedAt *time.Time
	err = env.dbPool.Pool.QueryRow(ctx, `
		SELECT id, status, email_verified_at 
		FROM users 
		WHERE email = 'admin-f6okyq@example.com'
	`).Scan(&intendedUserID, &intendedStatus, &emailVerifiedAt)
	if err != nil {
		t.Fatalf("intended admin user admin-f6okyq@example.com query failed: %v", err)
	}
	if intendedStatus != "active" || emailVerifiedAt == nil {
		t.Fatalf("expected intended admin user to be active and verified, got status=%s, verified=%v", intendedStatus, emailVerifiedAt)
	}
	if intendedUserID != "faed7cb7-3ca7-4b81-8959-53cb0ddbfb84" {
		t.Fatalf("expected intended admin user UUID to be faed7cb7-3ca7-4b81-8959-53cb0ddbfb84, got %s", intendedUserID)
	}

	// Verify that grant history and audit log entries exist for this user
	var grantAuditLogsCount int
	err = env.dbPool.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM platform_admin_audit_logs 
		WHERE target_id = $1 AND action = 'platform_admin.granted'
	`, intendedUserID).Scan(&grantAuditLogsCount)
	if err != nil {
		t.Fatalf("query audit logs: %v", err)
	}
	if grantAuditLogsCount < 1 {
		t.Fatalf("expected at least 1 platform_admin.granted audit log for %s, found %d", intendedUserID, grantAuditLogsCount)
	}

	// 7. Verify intended Organization Owner account remains non-Platform-Admin
	var minduUserID string
	err = env.dbPool.Pool.QueryRow(ctx, `SELECT id FROM users WHERE email = 'admin@mindu.ir'`).Scan(&minduUserID)
	if err != nil {
		t.Fatalf("query mindu user: %v", err)
	}
	var minduGrantCount int
	err = env.dbPool.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM platform_admin_grants 
		WHERE user_id = $1 AND revoked_at IS NULL
	`, minduUserID).Scan(&minduGrantCount)
	if err != nil {
		t.Fatalf("query mindu grants: %v", err)
	}
	if minduGrantCount != 0 {
		t.Fatalf("expected admin@mindu.ir to have 0 platform admin grants, found %d", minduGrantCount)
	}
}

func TestTask009_2_CLIListActiveAndRevoked(t *testing.T) {
	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	db, err := database.New(ctx, database.Config{DatabaseURL: testDBURL})
	if err != nil {
		t.Fatalf("connect DB: %v", err)
	}
	defer db.Close()

	// Query active and revoked grants
	rows, err := db.Pool.Query(ctx, `
		SELECT g.id, g.user_id, u.email, g.role, g.granted_at, g.revoked_at
		FROM platform_admin_grants g
		JOIN users u ON u.id = g.user_id
		ORDER BY g.granted_at DESC
	`)
	if err != nil {
		t.Fatalf("query grants: %v", err)
	}
	defer rows.Close()

	hasActive := false
	hasRevoked := false
	for rows.Next() {
		var gid, uid, email, role string
		var grantedAt time.Time
		var revokedAt *time.Time
		if err := rows.Scan(&gid, &uid, &email, &role, &grantedAt, &revokedAt); err != nil {
			t.Fatalf("scan grant: %v", err)
		}
		if revokedAt == nil {
			hasActive = true
		} else {
			hasRevoked = true
		}
	}

	if !hasActive {
		t.Errorf("expected at least 1 active platform admin grant in DB")
	}
	if !hasRevoked {
		t.Errorf("expected at least 1 revoked platform admin grant in DB")
	}
}
