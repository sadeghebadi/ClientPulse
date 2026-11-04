package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/service"
)

func createTestLangUser(t *testing.T, env *testAdminEnv, email, name, orgName string) *service.AuthResult {
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

func TestLanguageManagement_Availability(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()
	ctx := context.Background()

	// Ensure clean baseline state
	_, _ = env.dbPool.Pool.Exec(ctx, "UPDATE platform_localization_settings SET enabled_locales = ARRAY['en', 'fa'], default_locale = 'en' WHERE id = 1")
	defer func() {
		_, _ = env.dbPool.Pool.Exec(context.Background(), "UPDATE platform_localization_settings SET enabled_locales = ARRAY['en', 'fa'], default_locale = 'en' WHERE id = 1")
	}()

	adminUser := createTestLangUser(t, env, fmt.Sprintf("pa-lang-avail-%d@pulse.test", time.Now().UnixNano()%1000000), "Lang Admin", "Lang Admin Org")
	_, err := env.adminRepo.GrantAdmin(ctx, adminUser.User.ID, nil, "Language availability tests")
	if err != nil {
		t.Fatalf("failed to grant platform admin: %v", err)
	}

	// 1. Both languages enabled
	bothPayload := map[string]interface{}{
		"enabled_locales": []string{"en", "fa"},
		"default_locale":  "en",
		"reason":          "Activating both English and Persian",
	}
	wBoth := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/settings/localization", bothPayload, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wBoth.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for activating both languages, got %d: %s", wBoth.Code, wBoth.Body.String())
	}
	var bothRes repository.LocalizationConfig
	if err := json.Unmarshal(wBoth.Body.Bytes(), &bothRes); err != nil {
		t.Fatalf("failed to unmarshal both config: %v", err)
	}
	if len(bothRes.EnabledLocales) != 2 || bothRes.DefaultLocale != "en" {
		t.Fatalf("unexpected both config: %+v", bothRes)
	}

	// Verify public endpoint returns both enabled
	wPubBoth := doJSONReq(env.srv, "GET", "/api/v1/public/localization", nil, "", "", "", "")
	if wPubBoth.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for public localization, got %d", wPubBoth.Code)
	}
	var pubBoth repository.PublicLocalizationConfig
	_ = json.Unmarshal(wPubBoth.Body.Bytes(), &pubBoth)
	if len(pubBoth.EnabledLocales) != 2 {
		t.Fatalf("expected 2 enabled locales publicly, got %v", pubBoth.EnabledLocales)
	}

	// 2. English only (Persian disabled)
	enOnlyPayload := map[string]interface{}{
		"enabled_locales": []string{"en"},
		"default_locale":  "en",
		"reason":          "Disabling Persian, English only",
	}
	wEnOnly := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/settings/localization", enOnlyPayload, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wEnOnly.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for English only, got %d: %s", wEnOnly.Code, wEnOnly.Body.String())
	}
	var enRes repository.LocalizationConfig
	_ = json.Unmarshal(wEnOnly.Body.Bytes(), &enRes)
	if len(enRes.EnabledLocales) != 1 || enRes.EnabledLocales[0] != "en" {
		t.Fatalf("expected only en enabled, got %v", enRes.EnabledLocales)
	}

	// Check supported_locales metadata flags
	for _, meta := range enRes.SupportedLocales {
		if meta.Code == "en" && !meta.Enabled {
			t.Fatalf("expected en to be enabled in metadata")
		}
		if meta.Code == "fa" && meta.Enabled {
			t.Fatalf("expected fa to be disabled in metadata")
		}
	}

	// 3. Persian only (English disabled)
	faOnlyPayload := map[string]interface{}{
		"enabled_locales": []string{"fa"},
		"default_locale":  "fa",
		"reason":          "Disabling English, Persian only",
	}
	wFaOnly := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/settings/localization", faOnlyPayload, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wFaOnly.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for Persian only, got %d: %s", wFaOnly.Code, wFaOnly.Body.String())
	}
	var faRes repository.LocalizationConfig
	_ = json.Unmarshal(wFaOnly.Body.Bytes(), &faRes)
	if len(faRes.EnabledLocales) != 1 || faRes.EnabledLocales[0] != "fa" || faRes.DefaultLocale != "fa" {
		t.Fatalf("expected only fa enabled with fa default, got %+v", faRes)
	}

	// 4. Reject empty enabled-language array
	emptyPayload := map[string]interface{}{
		"enabled_locales": []string{},
		"default_locale":  "en",
	}
	wEmpty := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/settings/localization", emptyPayload, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wEmpty.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for empty enabled_locales, got %d: %s", wEmpty.Code, wEmpty.Body.String())
	}

	// 5. Reject unsupported locale (e.g. 'ar')
	unsupportedPayload := map[string]interface{}{
		"enabled_locales": []string{"en", "ar"},
		"default_locale":  "en",
	}
	wUnsupported := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/settings/localization", unsupportedPayload, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wUnsupported.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for unsupported locale 'ar', got %d: %s", wUnsupported.Code, wUnsupported.Body.String())
	}

	// 6. Cannot set disabled locale as default
	disabledDefPayload := map[string]interface{}{
		"enabled_locales": []string{"en"},
		"default_locale":  "fa", // 'fa' is not enabled
	}
	wDisabledDef := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/settings/localization", disabledDefPayload, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wDisabledDef.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request when setting disabled locale as default, got %d: %s", wDisabledDef.Code, wDisabledDef.Body.String())
	}
}

func TestLanguageManagement_DefaultTransitions(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()
	ctx := context.Background()

	// Ensure clean baseline state
	_, _ = env.dbPool.Pool.Exec(ctx, "UPDATE platform_localization_settings SET enabled_locales = ARRAY['en', 'fa'], default_locale = 'en' WHERE id = 1")
	defer func() {
		_, _ = env.dbPool.Pool.Exec(context.Background(), "UPDATE platform_localization_settings SET enabled_locales = ARRAY['en', 'fa'], default_locale = 'en' WHERE id = 1")
	}()

	adminUser := createTestLangUser(t, env, fmt.Sprintf("pa-lang-trans-%d@pulse.test", time.Now().UnixNano()%1000000), "Trans Admin", "Trans Admin Org")
	_, err := env.adminRepo.GrantAdmin(ctx, adminUser.User.ID, nil, "Default transitions test")
	if err != nil {
		t.Fatalf("failed to grant platform admin: %v", err)
	}

	// 1. Change default from English to Persian while both are enabled
	toFaDef := map[string]interface{}{
		"enabled_locales": []string{"en", "fa"},
		"default_locale":  "fa",
		"reason":          "Switching default fallback to Persian",
	}
	wToFa := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/settings/localization", toFaDef, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wToFa.Code != http.StatusOK {
		t.Fatalf("expected 200 OK changing default to fa, got %d: %s", wToFa.Code, wToFa.Body.String())
	}
	var cfgFa repository.LocalizationConfig
	_ = json.Unmarshal(wToFa.Body.Bytes(), &cfgFa)
	if cfgFa.DefaultLocale != "fa" {
		t.Fatalf("expected default_locale 'fa', got %s", cfgFa.DefaultLocale)
	}

	// 2. Change default from Persian to English
	toEnDef := map[string]interface{}{
		"enabled_locales": []string{"en", "fa"},
		"default_locale":  "en",
		"reason":          "Switching default fallback to English",
	}
	wToEn := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/settings/localization", toEnDef, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wToEn.Code != http.StatusOK {
		t.Fatalf("expected 200 OK changing default to en, got %d: %s", wToEn.Code, wToEn.Body.String())
	}
	var cfgEn repository.LocalizationConfig
	_ = json.Unmarshal(wToEn.Body.Bytes(), &cfgEn)
	if cfgEn.DefaultLocale != "en" {
		t.Fatalf("expected default_locale 'en', got %s", cfgEn.DefaultLocale)
	}

	// 3. Atomic transition: Change default from English to Persian AND disable English in a single operation
	atomicTransition := map[string]interface{}{
		"enabled_locales": []string{"fa"},
		"default_locale":  "fa",
		"reason":          "Atomic transition: make Persian default and disable English",
	}
	wAtomic := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/settings/localization", atomicTransition, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wAtomic.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for atomic transition, got %d: %s", wAtomic.Code, wAtomic.Body.String())
	}
	var cfgAtomic repository.LocalizationConfig
	_ = json.Unmarshal(wAtomic.Body.Bytes(), &cfgAtomic)
	if cfgAtomic.DefaultLocale != "fa" || len(cfgAtomic.EnabledLocales) != 1 || cfgAtomic.EnabledLocales[0] != "fa" {
		t.Fatalf("expected fa default and fa only enabled, got %+v", cfgAtomic)
	}

	// 4. Atomic re-enable English and restore English default
	atomicRestore := map[string]interface{}{
		"enabled_locales": []string{"en", "fa"},
		"default_locale":  "en",
		"reason":          "Re-enabling English and setting English default",
	}
	wRestore := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/settings/localization", atomicRestore, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wRestore.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for atomic restore, got %d: %s", wRestore.Code, wRestore.Body.String())
	}

	// 5. Verify audit log trail records each action
	wAudit := doJSONReq(env.srv, "GET", "/api/v1/platform-admin/audit-logs?action=localization.updated", nil, adminUser.RawSession, adminUser.RawCSRF, "", "")
	if wAudit.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for audit logs, got %d", wAudit.Code)
	}
	var auditRes struct {
		AuditLogs []repository.PlatformAdminAuditLog `json:"audit_logs"`
	}
	_ = json.Unmarshal(wAudit.Body.Bytes(), &auditRes)
	if len(auditRes.AuditLogs) < 4 {
		t.Fatalf("expected at least 4 localization audit records, got %d", len(auditRes.AuditLogs))
	}
}

func TestLanguageManagement_ConcurrentUpdates(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()
	ctx := context.Background()

	// Ensure clean baseline state
	_, _ = env.dbPool.Pool.Exec(ctx, "UPDATE platform_localization_settings SET enabled_locales = ARRAY['en', 'fa'], default_locale = 'en' WHERE id = 1")
	defer func() {
		_, _ = env.dbPool.Pool.Exec(context.Background(), "UPDATE platform_localization_settings SET enabled_locales = ARRAY['en', 'fa'], default_locale = 'en' WHERE id = 1")
	}()

	adminUser := createTestLangUser(t, env, fmt.Sprintf("pa-concurrent-%d@pulse.test", time.Now().UnixNano()%1000000), "Conc Admin", "Conc Admin Org")
	_, err := env.adminRepo.GrantAdmin(ctx, adminUser.User.ID, nil, "Concurrent update tests")
	if err != nil {
		t.Fatalf("failed to grant platform admin: %v", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 10)

	// Launch 10 concurrent requests alternating between valid states
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var payload map[string]interface{}
			if idx%2 == 0 {
				payload = map[string]interface{}{
					"enabled_locales": []string{"en", "fa"},
					"default_locale":  "en",
					"reason":          fmt.Sprintf("Concurrent update %d (both)", idx),
				}
			} else {
				payload = map[string]interface{}{
					"enabled_locales": []string{"fa"},
					"default_locale":  "fa",
					"reason":          fmt.Sprintf("Concurrent update %d (fa only)", idx),
				}
			}
			w := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/settings/localization", payload, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
			if w.Code != http.StatusOK {
				errCh <- fmt.Errorf("concurrent update %d failed with status %d: %s", idx, w.Code, w.Body.String())
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatal(err)
	}

	// Verify database state remains completely valid and consistent
	cfg, err := env.localizationRepo.GetLocalizationConfig(ctx)
	if err != nil {
		t.Fatalf("failed to query localization config after concurrent writes: %v", err)
	}
	if len(cfg.EnabledLocales) == 0 {
		t.Fatalf("invariant violated: enabled_locales is empty")
	}
	foundDef := false
	for _, l := range cfg.EnabledLocales {
		if l == cfg.DefaultLocale {
			foundDef = true
			break
		}
	}
	if !foundDef {
		t.Fatalf("invariant violated: default_locale '%s' not in enabled_locales %v", cfg.DefaultLocale, cfg.EnabledLocales)
	}
}

func TestLanguageManagement_PublicCatalogGracefulFallback(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()
	ctx := context.Background()

	// Ensure clean baseline state
	_, _ = env.dbPool.Pool.Exec(ctx, "UPDATE platform_localization_settings SET enabled_locales = ARRAY['en', 'fa'], default_locale = 'en' WHERE id = 1")
	defer func() {
		_, _ = env.dbPool.Pool.Exec(context.Background(), "UPDATE platform_localization_settings SET enabled_locales = ARRAY['en', 'fa'], default_locale = 'en' WHERE id = 1")
	}()

	adminUser := createTestLangUser(t, env, fmt.Sprintf("pa-cat-fb-%d@pulse.test", time.Now().UnixNano()%1000000), "FB Admin", "FB Admin Org")
	_, err := env.adminRepo.GrantAdmin(ctx, adminUser.User.ID, nil, "Catalog fallback test")
	if err != nil {
		t.Fatalf("failed to grant platform admin: %v", err)
	}

	// 1. Disable Persian (English only)
	enOnlyPayload := map[string]interface{}{
		"enabled_locales": []string{"en"},
		"default_locale":  "en",
		"reason":          "Persian disabled",
	}
	wEn := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/settings/localization", enOnlyPayload, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wEn.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", wEn.Code)
	}

	// Public catalog requested with ?locale=fa -> must gracefully fall back to 'en'
	wCatFa := doJSONReq(env.srv, "GET", "/api/v1/public/catalog?locale=fa", nil, "", "", "", "")
	if wCatFa.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for public catalog, got %d: %s", wCatFa.Code, wCatFa.Body.String())
	}
	var catFaRes struct {
		Locale string `json:"locale"`
	}
	_ = json.Unmarshal(wCatFa.Body.Bytes(), &catFaRes)
	if catFaRes.Locale != "en" {
		t.Fatalf("expected disabled 'fa' to fall back to 'en', got '%s'", catFaRes.Locale)
	}

	// 2. Disable English (Persian only)
	faOnlyPayload := map[string]interface{}{
		"enabled_locales": []string{"fa"},
		"default_locale":  "fa",
		"reason":          "English disabled",
	}
	wFa := doJSONReq(env.srv, "PUT", "/api/v1/platform-admin/settings/localization", faOnlyPayload, adminUser.RawSession, adminUser.RawCSRF, "", adminUser.RawCSRF)
	if wFa.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", wFa.Code)
	}

	// Public catalog requested with ?locale=en -> must gracefully fall back to 'fa'
	wCatEn := doJSONReq(env.srv, "GET", "/api/v1/public/catalog?locale=en", nil, "", "", "", "")
	if wCatEn.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for public catalog, got %d: %s", wCatEn.Code, wCatEn.Body.String())
	}
	var catEnRes struct {
		Locale string `json:"locale"`
	}
	_ = json.Unmarshal(wCatEn.Body.Bytes(), &catEnRes)
	if catEnRes.Locale != "fa" {
		t.Fatalf("expected disabled 'en' to fall back to 'fa', got '%s'", catEnRes.Locale)
	}
}
