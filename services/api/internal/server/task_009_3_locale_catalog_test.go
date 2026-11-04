package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/service"
)

func createTestUserHelper(t *testing.T, env *testAdminEnv, email, name, orgName string) *service.AuthResult {
	t.Helper()
	res, err := env.authService.Signup(context.Background(), service.SignupRequest{
		Email:            email,
		Password:         "SecurePassword123!",
		DisplayName:      name,
		OrganizationName: orgName,
	})
	if err != nil {
		t.Fatalf("failed to create test user %s: %v", email, err)
	}
	if err := env.userRepo.SetEmailVerified(context.Background(), res.User.ID); err != nil {
		t.Fatalf("failed to verify email for %s: %v", email, err)
	}
	return res
}

func TestPlatformAdmin_LocalizationSettings_Lifecycle(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()
	ctx := context.Background()

	// Ensure clean baseline state
	_, _ = env.dbPool.Pool.Exec(ctx, "UPDATE platform_localization_settings SET enabled_locales = ARRAY['en', 'fa'], default_locale = 'en' WHERE id = 1")
	defer func() {
		_, _ = env.dbPool.Pool.Exec(context.Background(), "UPDATE platform_localization_settings SET enabled_locales = ARRAY['en', 'fa'], default_locale = 'en' WHERE id = 1")
	}()

	// 1. Check public localization endpoint
	wPub := doJSONReq(env.srv, "GET", "/api/v1/public/localization", nil, "", "", "", "")
	if wPub.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for public localization, got %d: %s", wPub.Code, wPub.Body.String())
	}
	var pubCfg repository.PublicLocalizationConfig
	if err := json.Unmarshal(wPub.Body.Bytes(), &pubCfg); err != nil {
		t.Fatalf("failed to decode public localization: %v", err)
	}
	if len(pubCfg.EnabledLocales) < 2 {
		t.Fatalf("expected at least 2 enabled locales initially, got %v", pubCfg.EnabledLocales)
	}
	if pubCfg.DefaultLocale == "" {
		t.Fatalf("expected non-empty default locale")
	}

	// 2. Create ordinary user and platform admin user
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	orgUser := createTestUserHelper(t, env, "owner-"+suffix+"@test.org", "Owner "+suffix, "Org "+suffix)
	adminUser := createTestUserHelper(t, env, "pa-"+suffix+"@test.org", "Admin "+suffix, "AdminOrg "+suffix)

	_, err := env.adminRepo.GrantAdmin(ctx, adminUser.User.ID, nil, "Bootstrap admin for localization test")
	if err != nil {
		t.Fatalf("failed to grant platform admin: %v", err)
	}

	// 3. Org Owner cannot update localization settings (403 Forbidden)
	unauthBody := map[string]interface{}{
		"enabled_locales": []string{"en"},
		"default_locale":  "en",
		"reason":          "Attempt unauthorized change",
	}
	wForbidden := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/settings/localization", unauthBody, orgUser.RawSession, orgUser.RawCSRF, "", orgUser.RawCSRF)
	if wForbidden.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for non-platform admin, got %d", wForbidden.Code)
	}

	// 4. Platform Admin can fetch localization settings
	wAdminGet := doJSONReq(env.srv, "GET", "/api/v1/platform-admin/settings/localization", nil, adminUser.RawSession, adminUser.RawCSRF, "", "")
	if wAdminGet.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for admin localization get, got %d: %s", wAdminGet.Code, wAdminGet.Body.String())
	}

	// 5. Validation: Cannot disable all languages (empty enabled_locales)
	emptyBody := map[string]interface{}{
		"enabled_locales": []string{},
		"default_locale":  "en",
	}
	wEmpty := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/settings/localization", emptyBody, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wEmpty.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for empty enabled_locales, got %d", wEmpty.Code)
	}

	// 6. Validation: Default locale must be in enabled locales
	invalidDefBody := map[string]interface{}{
		"enabled_locales": []string{"fa"},
		"default_locale":  "en", // 'en' is not enabled
	}
	wInvalidDef := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/settings/localization", invalidDefBody, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wInvalidDef.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request when default locale is not in enabled locales, got %d", wInvalidDef.Code)
	}

	// 7. Update to English only (Persian disabled)
	enOnlyBody := map[string]interface{}{
		"enabled_locales": []string{"en"},
		"default_locale":  "en",
		"reason":          "Temporarily disabling Persian for maintenance",
	}
	wUpdate := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/settings/localization", enOnlyBody, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wUpdate.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid localization update, got %d: %s", wUpdate.Code, wUpdate.Body.String())
	}

	// Verify public config reflects update
	wPub2 := doJSONReq(env.srv, "GET", "/api/v1/public/localization", nil, "", "", "", "")
	if wPub2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", wPub2.Code)
	}
	var pubCfg2 repository.PublicLocalizationConfig
	_ = json.Unmarshal(wPub2.Body.Bytes(), &pubCfg2)
	if len(pubCfg2.EnabledLocales) != 1 || pubCfg2.EnabledLocales[0] != "en" {
		t.Fatalf("expected only 'en' enabled, got %v", pubCfg2.EnabledLocales)
	}

	// Re-enable Persian with Persian default
	bothBody := map[string]interface{}{
		"enabled_locales": []string{"en", "fa"},
		"default_locale":  "fa",
		"reason":          "Re-enabled Persian and set as default",
	}
	wBoth := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/settings/localization", bothBody, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wBoth.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", wBoth.Code, wBoth.Body.String())
	}

	// Verify audit log exists
	wAudit := doJSONReq(env.srv, "GET", "/api/v1/platform-admin/audit-logs?action=localization.updated", nil, adminUser.RawSession, adminUser.RawCSRF, "", "")
	if wAudit.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for audit logs, got %d", wAudit.Code)
	}
	var auditRes struct {
		AuditLogs []repository.PlatformAdminAuditLog `json:"audit_logs"`
	}
	_ = json.Unmarshal(wAudit.Body.Bytes(), &auditRes)
	if len(auditRes.AuditLogs) == 0 {
		t.Fatalf("expected at least one localization.updated audit log record")
	}
}

func TestPlatformAdmin_OfferCreationAndMonetarySafety(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()
	ctx := context.Background()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	adminUser := createTestUserHelper(t, env, "pa-cat-"+suffix+"@test.org", "Admin "+suffix, "AdminOrg "+suffix)
	_, err := env.adminRepo.GrantAdmin(ctx, adminUser.User.ID, nil, "Admin grant")
	if err != nil {
		t.Fatalf("grant admin failed: %v", err)
	}

	// 1. Create Base Plan with technical entitlements
	planReq := map[string]interface{}{
		"name":                       "Enterprise Pulse " + suffix,
		"slug":                       "pulse-plan-" + suffix,
		"description":                "Core technical entitlements",
		"max_clients":                25,
		"max_assets":                 100,
		"max_monitors":               250,
		"max_heartbeats":             50,
		"max_members":                10,
		"min_check_interval_seconds": 60,
		"retention_days":             90,
		"custom_branding_enabled":    true,
		"public_reports_enabled":     true,
	}
	wPlan := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans", planReq, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wPlan.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for plan, got %d: %s", wPlan.Code, wPlan.Body.String())
	}
	var planRes struct {
		Plan    repository.SubscriptionPlan        `json:"plan"`
		Version repository.SubscriptionPlanVersion `json:"version"`
	}
	_ = json.Unmarshal(wPlan.Body.Bytes(), &planRes)
	planID := planRes.Plan.ID

	// Publish the base plan so it is active
	// Note: in 000019, publishing plan version requires price_amount or can be published
	// Let's update version to have price before publishing base plan
	updatePlanReq := map[string]interface{}{
		"name":                       planRes.Plan.Name,
		"description":                planRes.Plan.Description,
		"price_amount":               0,
		"currency":                   "USD",
		"billing_interval":           "monthly",
		"max_clients":                25,
		"max_assets":                 100,
		"max_monitors":               250,
		"max_heartbeats":             50,
		"max_members":                10,
		"min_check_interval_seconds": 60,
		"retention_days":             90,
		"custom_branding_enabled":    true,
		"public_reports_enabled":     true,
	}
	wUpPlan := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/plans/"+planID, updatePlanReq, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wUpPlan.Code != http.StatusOK {
		t.Fatalf("failed to update plan: %s", wUpPlan.Body.String())
	}
	wPubPlan := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans/"+planID+"/publish", nil, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wPubPlan.Code != http.StatusOK {
		t.Fatalf("failed to publish base plan: %s", wPubPlan.Body.String())
	}

	// 2. Create English Localized Offer ($29/month)
	enOfferReq := map[string]interface{}{
		"locale":               "en",
		"market":               "global",
		"name":                 "Pulse Enterprise Global",
		"description":          "High-performance monitoring for digital teams.",
		"features_summary":     []string{"25 Client Workspaces", "250 Monitors", "Custom Domain"},
		"display_price":        29,
		"currency":             "USD",
		"display_denomination": "usd",
		"denomination_ratio":   100, // 29 USD = 2900 cents
		"billing_interval":     "monthly",
	}
	wEnOffer := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans/"+planID+"/offers", enOfferReq, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wEnOffer.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for EN offer, got %d: %s", wEnOffer.Code, wEnOffer.Body.String())
	}
	var enOfferRes struct {
		Offer   repository.SubscriptionOffer        `json:"offer"`
		Version repository.SubscriptionOfferVersion `json:"version"`
	}
	_ = json.Unmarshal(wEnOffer.Body.Bytes(), &enOfferRes)
	if enOfferRes.Version.PriceAmount != 2900 {
		t.Fatalf("expected canonical stored price 2900 cents, got %d", enOfferRes.Version.PriceAmount)
	}

	// 3. Create Persian Localized Offer (290,000 Toman / month with Toman denomination and ratio = 10)
	faOfferReq := map[string]interface{}{
		"locale":               "fa",
		"market":               "ir",
		"name":                 "اشتراک سازمانی پالس",
		"description":          "راهکار جامع پایش و پایداری وب‌سایت برای آژانس‌های دیجیتال ایرانی.",
		"features_summary":     []string{"۲۵ مشتری مستقل", "۲۵۰ مانیتور پایش زنده", "برندینگ اختصاصی"},
		"display_price":        290000, // 290,000 Toman
		"currency":             "IRR",
		"display_denomination": "toman",
		"denomination_ratio":   10, // 1 Toman = 10 IRR Rials
		"billing_interval":     "monthly",
	}
	wFaOffer := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans/"+planID+"/offers", faOfferReq, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wFaOffer.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for FA offer, got %d: %s", wFaOffer.Code, wFaOffer.Body.String())
	}
	var faOfferRes struct {
		Offer   repository.SubscriptionOffer        `json:"offer"`
		Version repository.SubscriptionOfferVersion `json:"version"`
	}
	_ = json.Unmarshal(wFaOffer.Body.Bytes(), &faOfferRes)

	// Verify exact integer conversion: 290,000 Toman * 10 = 2,900,000 IRR Rials
	if faOfferRes.Version.PriceAmount != 2900000 {
		t.Fatalf("expected stored canonical IRR 2900000, got %d", faOfferRes.Version.PriceAmount)
	}
	if faOfferRes.Version.DisplayPrice != 290000 {
		t.Fatalf("expected display price 290000 Toman, got %d", faOfferRes.Version.DisplayPrice)
	}

	// 4. Test Negative Price Rejection
	negReq := map[string]interface{}{
		"locale":               "fa",
		"market":               "test",
		"name":                 "Negative Test",
		"display_price":        -500,
		"currency":             "IRR",
		"display_denomination": "toman",
		"denomination_ratio":   10,
	}
	wNeg := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans/"+planID+"/offers", negReq, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wNeg.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for negative price, got %d", wNeg.Code)
	}

	// 5. Publish Persian Offer (Becomes Immutable)
	wPubFa := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/offers/"+faOfferRes.Offer.ID+"/publish", nil, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wPubFa.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for publish offer, got %d: %s", wPubFa.Code, wPubFa.Body.String())
	}
	var pubFaRes struct {
		Offer   repository.SubscriptionOffer        `json:"offer"`
		Version repository.SubscriptionOfferVersion `json:"version"`
	}
	_ = json.Unmarshal(wPubFa.Body.Bytes(), &pubFaRes)
	if pubFaRes.Offer.Status != "active" || pubFaRes.Version.Status != "published" {
		t.Fatalf("expected published status, got offer: %s, ver: %s", pubFaRes.Offer.Status, pubFaRes.Version.Status)
	}

	// 6. Immutability Verification: Direct DB UPDATE on published offer version must trigger SQL exception
	_, dbErr := env.dbPool.Pool.Exec(ctx, "UPDATE subscription_offer_versions SET price_amount = 9999999 WHERE id = $1", pubFaRes.Version.ID)
	if dbErr == nil {
		t.Fatalf("expected DB trigger to reject mutation of published offer version, but update succeeded")
	}

	// 7. Updating an active offer creates an incremented draft version (v2)
	upFaReq := map[string]interface{}{
		"name":                 "اشتراک سازمانی پالس ویرایش ۲",
		"description":          pubFaRes.Offer.Description,
		"features_summary":     pubFaRes.Offer.FeaturesSummary,
		"display_price":        350000, // 350,000 Toman
		"currency":             "IRR",
		"display_denomination": "toman",
		"denomination_ratio":   10,
		"billing_interval":     "monthly",
	}
	wUpFa := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/offers/"+faOfferRes.Offer.ID, upFaReq, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wUpFa.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for updating offer, got %d: %s", wUpFa.Code, wUpFa.Body.String())
	}
	var upFaRes struct {
		Offer   repository.SubscriptionOffer        `json:"offer"`
		Version repository.SubscriptionOfferVersion `json:"version"`
	}
	_ = json.Unmarshal(wUpFa.Body.Bytes(), &upFaRes)
	if upFaRes.Version.Version != 2 || upFaRes.Version.Status != "draft" {
		t.Fatalf("expected new draft version 2, got v%d (%s)", upFaRes.Version.Version, upFaRes.Version.Status)
	}
	if upFaRes.Version.PriceAmount != 3500000 {
		t.Fatalf("expected 3500000 stored IRR for v2, got %d", upFaRes.Version.PriceAmount)
	}
}

func TestPlatformAdmin_SubscriptionAssignment_And_HistoricalPreservation(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()
	ctx := context.Background()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	adminUser := createTestUserHelper(t, env, "pa-sub-"+suffix+"@test.org", "Admin "+suffix, "AdminOrg "+suffix)
	clientOrg := createTestUserHelper(t, env, "client-"+suffix+"@test.org", "Client "+suffix, "ClientOrg "+suffix)

	_, err := env.adminRepo.GrantAdmin(ctx, adminUser.User.ID, nil, "Admin grant")
	if err != nil {
		t.Fatalf("failed to grant admin: %v", err)
	}

	// 1. Create Base Plan
	planReq := map[string]interface{}{
		"name":                       "Historical Pro " + suffix,
		"slug":                       "hist-plan-" + suffix,
		"description":                "Shared technical capabilities",
		"price_amount":               0,
		"currency":                   "USD",
		"billing_interval":           "monthly",
		"max_clients":                10,
		"max_assets":                 50,
		"max_monitors":               100,
		"max_heartbeats":             20,
		"max_members":                5,
		"min_check_interval_seconds": 60,
		"retention_days":             30,
		"custom_branding_enabled":    false,
		"public_reports_enabled":     true,
	}
	wPlan := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans", planReq, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	var planRes struct {
		Plan    repository.SubscriptionPlan        `json:"plan"`
		Version repository.SubscriptionPlanVersion `json:"version"`
	}
	_ = json.Unmarshal(wPlan.Body.Bytes(), &planRes)
	planID := planRes.Plan.ID

	// Publish plan
	_ = doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans/"+planID+"/publish", nil, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)

	// Fetch published plan version ID
	_, versions, _ := env.planRepo.GetPlanByID(ctx, planID)
	publishedPlanVersionID := versions[0].ID

	// 2. Create & Publish Persian Offer at 150,000 Toman (1,500,000 IRR)
	faOfferReq := map[string]interface{}{
		"locale":               "fa",
		"market":               "ir",
		"name":                 "پلن حرفه‌ای فارسی",
		"description":          "پکیج حرفه‌ای برای آژانس‌های ایرانی",
		"features_summary":     []string{"۱۰ مشتری", "۱۰۰ مانیتور"},
		"display_price":        150000,
		"currency":             "IRR",
		"display_denomination": "toman",
		"denomination_ratio":   10,
		"billing_interval":     "monthly",
	}
	wOffer := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans/"+planID+"/offers", faOfferReq, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	var offerRes struct {
		Offer   repository.SubscriptionOffer        `json:"offer"`
		Version repository.SubscriptionOfferVersion `json:"version"`
	}
	_ = json.Unmarshal(wOffer.Body.Bytes(), &offerRes)

	wPub := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/offers/"+offerRes.Offer.ID+"/publish", nil, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wPub.Code != http.StatusOK {
		t.Fatalf("failed to publish offer: %s", wPub.Body.String())
	}
	var pubRes struct {
		Offer   repository.SubscriptionOffer        `json:"offer"`
		Version repository.SubscriptionOfferVersion `json:"version"`
	}
	_ = json.Unmarshal(wPub.Body.Bytes(), &pubRes)
	firstOfferVersionID := pubRes.Version.ID

	// 3. Assign subscription using the Persian offer version
	assignReq := map[string]interface{}{
		"organization_id":     clientOrg.Organization.ID,
		"plan_version_id":     publishedPlanVersionID,
		"offer_version_id":    firstOfferVersionID,
		"administrative_note": "Initial purchase at 150,000 Toman",
	}
	wAssign := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/subscriptions/assign", assignReq, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wAssign.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for subscription assign, got %d: %s", wAssign.Code, wAssign.Body.String())
	}
	var assignRes struct {
		Subscription repository.OrganizationSubscription `json:"subscription"`
	}
	_ = json.Unmarshal(wAssign.Body.Bytes(), &assignRes)

	if assignRes.Subscription.OfferVersionID == nil || *assignRes.Subscription.OfferVersionID != firstOfferVersionID {
		t.Fatalf("expected subscription to link to offer_version_id %s", firstOfferVersionID)
	}
	if *assignRes.Subscription.PriceAmount != 1500000 {
		t.Fatalf("expected price_amount 1500000 IRR, got %v", *assignRes.Subscription.PriceAmount)
	}
	if *assignRes.Subscription.DisplayPrice != 150000 {
		t.Fatalf("expected display_price 150000 Toman, got %v", *assignRes.Subscription.DisplayPrice)
	}

	// 4. Update the offer price for future customers to 220,000 Toman (creates v2) and publish v2
	upReq := map[string]interface{}{
		"name":                 "پلن حرفه‌ای فارسی (نرخ جدید)",
		"description":          "نرخ جدید سالیانه",
		"features_summary":     []string{"۱۰ مشتری", "۱۰۰ مانیتور"},
		"display_price":        220000,
		"currency":             "IRR",
		"display_denomination": "toman",
		"denomination_ratio":   10,
		"billing_interval":     "monthly",
	}
	_ = doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/offers/"+offerRes.Offer.ID, upReq, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	_ = doJSONReq(env.srv, "POST", "/api/v1/platform-admin/offers/"+offerRes.Offer.ID+"/publish", nil, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)

	// 5. Verify Historical Subscription Price Preservation:
	// The client's active subscription MUST still show 150,000 Toman (1,500,000 IRR)
	activeSub, err := env.subscriptionRepo.GetActiveSubscriptionForOrg(ctx, clientOrg.Organization.ID)
	if err != nil {
		t.Fatalf("failed to get active subscription: %v", err)
	}
	if *activeSub.PriceAmount != 1500000 {
		t.Fatalf("historical price violated: expected 1500000 IRR, got %d", *activeSub.PriceAmount)
	}
	if *activeSub.DisplayPrice != 150000 {
		t.Fatalf("historical display price violated: expected 150000 Toman, got %d", *activeSub.DisplayPrice)
	}
	if *activeSub.OfferVersionID != firstOfferVersionID {
		t.Fatalf("historical offer version altered: expected %s, got %s", firstOfferVersionID, *activeSub.OfferVersionID)
	}

	// 6. Immutability: Cannot delete offer version referenced by subscriptions
	_, delErr := env.dbPool.Pool.Exec(ctx, "DELETE FROM subscription_offer_versions WHERE id = $1", firstOfferVersionID)
	if delErr == nil {
		t.Fatalf("expected DB trigger to reject deletion of offer version referenced by active subscription")
	}
}

func TestPublicCatalog_LocaleFallbackAndDeactivation(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()
	ctx := context.Background()

	// Ensure clean baseline and reset on cleanup
	_, _ = env.dbPool.Pool.Exec(ctx, "UPDATE platform_localization_settings SET enabled_locales = ARRAY['en', 'fa'], default_locale = 'en' WHERE id = 1")
	defer func() {
		_, _ = env.dbPool.Pool.Exec(context.Background(), "UPDATE platform_localization_settings SET enabled_locales = ARRAY['en', 'fa'], default_locale = 'en' WHERE id = 1")
	}()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	adminUser := createTestUserHelper(t, env, "pa-fall-"+suffix+"@test.org", "Admin "+suffix, "AdminOrg "+suffix)
	_, _ = env.adminRepo.GrantAdmin(ctx, adminUser.User.ID, nil, "Admin grant")

	// Ensure both 'en' and 'fa' are enabled, with 'en' as default
	_, _ = env.localizationRepo.UpdateLocalizationConfig(ctx, adminUser.User.ID, []string{"en", "fa"}, "en")

	// Create base plan
	planReq := map[string]interface{}{
		"name":                       "Catalog Plan " + suffix,
		"slug":                       "cat-plan-" + suffix,
		"description":                "Public catalog test plan",
		"price_amount":               0,
		"currency":                   "USD",
		"billing_interval":           "monthly",
		"max_clients":                5,
		"max_assets":                 10,
		"max_monitors":               20,
		"max_heartbeats":             5,
		"max_members":                3,
		"min_check_interval_seconds": 60,
		"retention_days":             30,
		"custom_branding_enabled":    false,
		"public_reports_enabled":     true,
	}
	wPlan := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans", planReq, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	var planRes struct {
		Plan repository.SubscriptionPlan `json:"plan"`
	}
	_ = json.Unmarshal(wPlan.Body.Bytes(), &planRes)
	_ = doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans/"+planRes.Plan.ID+"/publish", nil, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)

	// Create and publish English offer
	enOfferReq := map[string]interface{}{
		"locale":               "en",
		"market":               "global",
		"name":                 "English Offer " + suffix,
		"description":          "Public English offer",
		"display_price":        19,
		"currency":             "USD",
		"display_denomination": "usd",
		"denomination_ratio":   100,
		"billing_interval":     "monthly",
	}
	wEn := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans/"+planRes.Plan.ID+"/offers", enOfferReq, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	var enOfferRes struct {
		Offer repository.SubscriptionOffer `json:"offer"`
	}
	_ = json.Unmarshal(wEn.Body.Bytes(), &enOfferRes)
	_ = doJSONReq(env.srv, "POST", "/api/v1/platform-admin/offers/"+enOfferRes.Offer.ID+"/publish", nil, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)

	// Create and publish Persian offer
	faOfferReq := map[string]interface{}{
		"locale":               "fa",
		"market":               "ir",
		"name":                 "پیشنهاد فارسی " + suffix,
		"description":          "پیشنهاد عمومی فارسی",
		"display_price":        190000,
		"currency":             "IRR",
		"display_denomination": "toman",
		"denomination_ratio":   10,
		"billing_interval":     "monthly",
	}
	wFa := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans/"+planRes.Plan.ID+"/offers", faOfferReq, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	var faOfferRes struct {
		Offer repository.SubscriptionOffer `json:"offer"`
	}
	_ = json.Unmarshal(wFa.Body.Bytes(), &faOfferRes)
	_ = doJSONReq(env.srv, "POST", "/api/v1/platform-admin/offers/"+faOfferRes.Offer.ID+"/publish", nil, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)

	// 1. Query catalog for English: returns English offer
	wCatEn := doJSONReq(env.srv, "GET", "/api/v1/public/catalog?locale=en", nil, "", "", "", "")
	if wCatEn.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", wCatEn.Code, wCatEn.Body.String())
	}
	var catEnRes struct {
		Locale string                         `json:"locale"`
		Offers []repository.SubscriptionOffer `json:"offers"`
	}
	_ = json.Unmarshal(wCatEn.Body.Bytes(), &catEnRes)
	if catEnRes.Locale != "en" {
		t.Fatalf("expected locale en, got %s", catEnRes.Locale)
	}

	// 2. Query catalog for Persian: returns Persian offer
	wCatFa := doJSONReq(env.srv, "GET", "/api/v1/public/catalog?locale=fa", nil, "", "", "", "")
	if wCatFa.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", wCatFa.Code, wCatFa.Body.String())
	}
	var catFaRes struct {
		Locale string                         `json:"locale"`
		Offers []repository.SubscriptionOffer `json:"offers"`
	}
	_ = json.Unmarshal(wCatFa.Body.Bytes(), &catFaRes)
	if catFaRes.Locale != "fa" {
		t.Fatalf("expected locale fa, got %s", catFaRes.Locale)
	}

	// 3. Disable Persian in platform settings (only English enabled)
	_, err := env.localizationRepo.UpdateLocalizationConfig(ctx, adminUser.User.ID, []string{"en"}, "en")
	if err != nil {
		t.Fatalf("failed to disable Persian: %v", err)
	}

	// 4. Query public catalog with ?locale=fa:
	// Because Persian is disabled, server must gracefully fall back to default locale (English)
	wCatDisabled := doJSONReq(env.srv, "GET", "/api/v1/public/catalog?locale=fa", nil, "", "", "", "")
	if wCatDisabled.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with graceful fallback, got %d: %s", wCatDisabled.Code, wCatDisabled.Body.String())
	}
	var catDisabledRes struct {
		Locale string                         `json:"locale"`
		Offers []repository.SubscriptionOffer `json:"offers"`
	}
	_ = json.Unmarshal(wCatDisabled.Body.Bytes(), &catDisabledRes)
	if catDisabledRes.Locale != "en" {
		t.Fatalf("expected disabled locale to fall back to 'en', got %s", catDisabledRes.Locale)
	}
}
