package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"clientpulse/services/api/internal/auth"
	"clientpulse/services/api/internal/checker"
	"clientpulse/services/api/internal/database"
	"clientpulse/services/api/internal/entitlement"
	"clientpulse/services/api/internal/mail"
	"clientpulse/services/api/internal/pdf"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/service"
	"clientpulse/services/api/internal/ssrf"
	"clientpulse/services/api/internal/storage"
	"clientpulse/services/api/internal/testhelper"
)

type testAdminEnv struct {
	srv              *Server
	authService      *service.AuthService
	userRepo         *repository.UserRepository
	orgRepo          *repository.OrganizationRepository
	clientRepo       *repository.ClientRepository
	adminRepo        *repository.PlatformAdminRepository
	planRepo         *repository.PlanRepository
	subscriptionRepo *repository.SubscriptionRepository
	entitlementSvc   *entitlement.Service
	dbPool           *database.DB
	cleanup          func()
}

func setupTestPlatformAdminServer(t *testing.T) *testAdminEnv {
	t.Helper()
	db := testhelper.GetTestDB(t)
	cfg := testhelper.GetTestConfig()

	userRepo := repository.NewUserRepository(db.Pool)
	sessionRepo := repository.NewSessionRepository(db.Pool)
	orgRepo := repository.NewOrganizationRepository(db.Pool)
	tokenRepo := repository.NewTokenRepository(db.Pool)
	auditRepo := repository.NewAuditRepository(db.Pool)
	clientRepo := repository.NewClientRepository(db.Pool)
	assetRepo := repository.NewAssetRepository(db.Pool)
	monitorRepo := repository.NewMonitorRepository(db.Pool)
	resultRepo := repository.NewCheckResultRepository(db.Pool)
	heartbeatRepo := repository.NewHeartbeatRepository(db.Pool)
	incidentRepo := repository.NewIncidentRepository(db.Pool)
	eventRepo := repository.NewNotificationEventRepository(db.Pool)
	ssrfGuard := ssrf.NewGuard(nil)
	chk := checker.NewChecker(ssrfGuard, "test-platform-admin-worker")

	st, err := storage.NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create temp storage: %v", err)
	}

	brandingRepo := repository.NewBrandingRepository(db.Pool)
	reportRepo := repository.NewReportRepository(db.Pool, brandingRepo)
	pdfGen, _ := pdf.NewGenerator()

	platformAdminRepo := repository.NewPlatformAdminRepository(db.Pool)
	planRepo := repository.NewPlanRepository(db.Pool)
	subscriptionRepo := repository.NewSubscriptionRepository(db.Pool)
	entitlementSvc := entitlement.NewService(db.Pool, subscriptionRepo)

	mockMailer := mail.NewMockMailer()
	authService := service.NewAuthService(userRepo, sessionRepo, orgRepo, tokenRepo, auditRepo, mockMailer, cfg).WithPlatformAdminRepo(platformAdminRepo)
	orgService := service.NewOrgService(orgRepo, userRepo, sessionRepo, auditRepo, cfg)
	testLog := slog.New(slog.NewTextHandler(io.Discard, nil))

	deps := Dependencies{
		AuthHdr:          NewAuthHandler(authService, false),
		OrgHdr:           NewOrgHandler(orgService).WithEntitlements(entitlementSvc),
		ClientHdr:        NewClientHandler(clientRepo, auditRepo).WithEntitlements(entitlementSvc),
		AssetHdr:         NewAssetHandler(assetRepo, auditRepo, ssrfGuard, cfg.AppEnv).WithEntitlements(entitlementSvc),
		MonitorHdr:       NewMonitorHandler(monitorRepo, resultRepo, assetRepo, auditRepo, chk, ssrfGuard).WithEntitlements(entitlementSvc),
		HeartbeatHdr:     NewHeartbeatHandler(heartbeatRepo, cfg, testLog).WithEntitlements(entitlementSvc),
		IncidentHdr:      NewIncidentHandler(incidentRepo, eventRepo, testLog),
		BrandingHdr:      NewBrandingHandler(brandingRepo, st, testLog).WithEntitlements(entitlementSvc),
		ReportHdr:        NewReportHandler(reportRepo, brandingRepo, eventRepo, st, pdfGen, testLog),
		PlatformAdminHdr: NewPlatformAdminHandler(platformAdminRepo, planRepo, subscriptionRepo, testLog),
		AuthMiddleware:   NewAuthMiddleware(sessionRepo, userRepo, cfg).WithPlatformAdminRepo(platformAdminRepo),
		TenantMiddleware: NewTenantMiddleware(orgRepo, sessionRepo),
	}

	srv := New(cfg, testLog, deps)

	cleanup := func() {
		db.Close()
	}

	return &testAdminEnv{
		srv:              srv,
		authService:      authService,
		userRepo:         userRepo,
		orgRepo:          orgRepo,
		clientRepo:       clientRepo,
		adminRepo:        platformAdminRepo,
		planRepo:         planRepo,
		subscriptionRepo: subscriptionRepo,
		entitlementSvc:   entitlementSvc,
		dbPool:           db,
		cleanup:          cleanup,
	}
}

func doJSONReq(srv *Server, method, path string, body interface{}, sessionCookie, csrfCookie, orgID, csrfHeader string) *httptest.ResponseRecorder {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reqBody)
	if sessionCookie != "" {
		req.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: sessionCookie})
	}
	if csrfCookie != "" {
		req.AddCookie(&http.Cookie{Name: "clientpulse_csrf", Value: csrfCookie})
	}
	if csrfHeader != "" {
		req.Header.Set("X-CSRF-Token", csrfHeader)
	}
	if orgID != "" {
		req.Header.Set("X-Organization-ID", orgID)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)
	return w
}

func TestPlatformAdmin_AuthorizationAndIndependence(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()
	ctx := context.Background()

	// 1. Create standard Organization Owner account (e.g. simulated admin@mindu.ir)
	suffix, _ := auth.GenerateSecureToken(4)
	ownerEmail := fmt.Sprintf("admin-%s@mindu.ir", suffix)
	signupRes, err := env.authService.Signup(ctx, service.SignupRequest{
		Email:            ownerEmail,
		Password:         "SecurePassword123!",
		DisplayName:      "Mindu Org Owner",
		OrganizationName: "Mindu Technologies",
	})
	if err != nil {
		t.Fatalf("signup failed: %v", err)
	}
	if err := env.userRepo.SetEmailVerified(ctx, signupRes.User.ID); err != nil {
		t.Fatalf("email verify failed: %v", err)
	}

	// 2. Unauthenticated caller gets 401 Unauthorized
	wUnauth := doJSONReq(env.srv, "GET", "/api/v1/platform-admin/overview", nil, "", "", "", "")
	if wUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated request, got %d: %s", wUnauth.Code, wUnauth.Body.String())
	}

	// 3. Org Owner receives 403 Forbidden on platform admin endpoints (even though email has admin@mindu.ir prefix!)
	wOwnerAdmin := doJSONReq(env.srv, "GET", "/api/v1/platform-admin/overview", nil, signupRes.RawSession, signupRes.RawCSRF, "", "")
	if wOwnerAdmin.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for Org Owner on platform admin route, got %d: %s", wOwnerAdmin.Code, wOwnerAdmin.Body.String())
	}
	if !strings.Contains(wOwnerAdmin.Body.String(), "Platform administrator privileges required") {
		t.Errorf("expected platform administrator privileges message, got: %s", wOwnerAdmin.Body.String())
	}

	// 4. Verify privilege endpoint for Org Owner
	wOwnerPriv := doJSONReq(env.srv, "GET", "/api/v1/platform-admin/privilege", nil, signupRes.RawSession, signupRes.RawCSRF, "", "")
	if wOwnerPriv.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on privilege check endpoint, got %d", wOwnerPriv.Code)
	}
	var privMap map[string]interface{}
	if err := json.Unmarshal(wOwnerPriv.Body.Bytes(), &privMap); err != nil {
		t.Fatalf("failed to unmarshal privilege response: %v", err)
	}
	if privMap["is_platform_admin"] != false {
		t.Fatalf("expected is_platform_admin=false for org owner, got %v", privMap["is_platform_admin"])
	}

	// 5. Create a dedicated platform admin user and grant privileges directly in DB
	adminEmail := fmt.Sprintf("superadmin-%s@pulse.test", suffix)
	adminSignup, err := env.authService.Signup(ctx, service.SignupRequest{
		Email:            adminEmail,
		Password:         "SuperAdminPassword123!",
		DisplayName:      "Global Administrator",
		OrganizationName: "Pulse HQ",
	})
	if err != nil {
		t.Fatalf("admin signup failed: %v", err)
	}
	if err := env.userRepo.SetEmailVerified(ctx, adminSignup.User.ID); err != nil {
		t.Fatalf("admin email verify failed: %v", err)
	}

	// Grant platform admin via repository
	_, err = env.adminRepo.GrantAdmin(ctx, adminSignup.User.ID, nil, "TASK-009 test bootstrap grant")
	if err != nil {
		t.Fatalf("failed to grant platform admin: %v", err)
	}

	// 6. Platform Admin can now access /api/v1/platform-admin/overview
	wAdminOverview := doJSONReq(env.srv, "GET", "/api/v1/platform-admin/overview", nil, adminSignup.RawSession, adminSignup.RawCSRF, "", "")
	if wAdminOverview.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for platform admin, got %d: %s", wAdminOverview.Code, wAdminOverview.Body.String())
	}

	// Check privilege endpoint
	wAdminPriv := doJSONReq(env.srv, "GET", "/api/v1/platform-admin/privilege", nil, adminSignup.RawSession, adminSignup.RawCSRF, "", "")
	if wAdminPriv.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for platform admin privilege check, got %d", wAdminPriv.Code)
	}
	var adminPrivMap map[string]interface{}
	_ = json.Unmarshal(wAdminPriv.Body.Bytes(), &adminPrivMap)
	if adminPrivMap["is_platform_admin"] != true {
		t.Fatalf("expected is_platform_admin=true for platform admin, got %v", adminPrivMap["is_platform_admin"])
	}
}

func TestPlatformAdmin_UserLifecycle_SuspensionAndReactivation(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()
	ctx := context.Background()

	suffix, _ := auth.GenerateSecureToken(4)
	// Platform admin
	adminSignup, _ := env.authService.Signup(ctx, service.SignupRequest{
		Email: fmt.Sprintf("pa-%s@pulse.test", suffix), Password: "Password123!", DisplayName: "Admin", OrganizationName: "Pulse HQ",
	})
	_ = env.userRepo.SetEmailVerified(ctx, adminSignup.User.ID)
	_, _ = env.adminRepo.GrantAdmin(ctx, adminSignup.User.ID, nil, "test")

	// Target user
	targetEmail := fmt.Sprintf("target-%s@agency.test", suffix)
	targetSignup, _ := env.authService.Signup(ctx, service.SignupRequest{
		Email: targetEmail, Password: "TargetPassword123!", DisplayName: "Target User", OrganizationName: "Target Org",
	})
	_ = env.userRepo.SetEmailVerified(ctx, targetSignup.User.ID)

	// Target user can make requests normally
	wBefore := doJSONReq(env.srv, "GET", "/api/v1/clients", nil, targetSignup.RawSession, targetSignup.RawCSRF, targetSignup.Organization.ID, "")
	if wBefore.Code != http.StatusOK {
		t.Fatalf("target user client list before suspension failed with %d: %s", wBefore.Code, wBefore.Body.String())
	}

	// Platform Admin suspends target user
	suspendBody := map[string]string{"reason": "Terms of Service violation investigation"}
	wSuspend := doJSONReq(env.srv, "POST", fmt.Sprintf("/api/v1/platform-admin/users/%s/suspend", targetSignup.User.ID), suspendBody, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wSuspend.Code != http.StatusOK {
		t.Fatalf("suspend user failed: %d: %s", wSuspend.Code, wSuspend.Body.String())
	}

	// Target user attempts to make request with existing session -> 403 USER_SUSPENDED
	wSuspendedReq := doJSONReq(env.srv, "GET", "/api/v1/clients", nil, targetSignup.RawSession, targetSignup.RawCSRF, targetSignup.Organization.ID, "")
	if wSuspendedReq.Code != http.StatusForbidden {
		t.Fatalf("expected 403 USER_SUSPENDED for suspended user, got %d: %s", wSuspendedReq.Code, wSuspendedReq.Body.String())
	}
	if !strings.Contains(wSuspendedReq.Body.String(), "USER_SUSPENDED") {
		t.Errorf("expected USER_SUSPENDED error code, got: %s", wSuspendedReq.Body.String())
	}

	// Target user attempts to log in -> 403 USER_SUSPENDED
	loginBody := map[string]string{"email": targetEmail, "password": "TargetPassword123!"}
	wLogin := doJSONReq(env.srv, "POST", "/api/v1/auth/login", loginBody, "", "", "", "")
	if wLogin.Code != http.StatusForbidden {
		t.Fatalf("expected 403 USER_SUSPENDED on login, got %d: %s", wLogin.Code, wLogin.Body.String())
	}

	// Platform Admin reactivates target user
	reactivateBody := map[string]string{"reason": "Investigation completed; account restored"}
	wReactivate := doJSONReq(env.srv, "POST", fmt.Sprintf("/api/v1/platform-admin/users/%s/reactivate", targetSignup.User.ID), reactivateBody, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wReactivate.Code != http.StatusOK {
		t.Fatalf("reactivate user failed: %d: %s", wReactivate.Code, wReactivate.Body.String())
	}

	// Target user can log in again
	wLoginAfter := doJSONReq(env.srv, "POST", "/api/v1/auth/login", loginBody, "", "", "", "")
	if wLoginAfter.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on login after reactivation, got %d: %s", wLoginAfter.Code, wLoginAfter.Body.String())
	}
}

func TestPlatformAdmin_OrganizationLifecycle_SuspensionAndReactivation(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()
	ctx := context.Background()

	suffix, _ := auth.GenerateSecureToken(4)
	adminSignup, _ := env.authService.Signup(ctx, service.SignupRequest{
		Email: fmt.Sprintf("pa2-%s@pulse.test", suffix), Password: "Password123!", DisplayName: "Admin", OrganizationName: "Pulse HQ",
	})
	_ = env.userRepo.SetEmailVerified(ctx, adminSignup.User.ID)
	_, _ = env.adminRepo.GrantAdmin(ctx, adminSignup.User.ID, nil, "test")

	orgUser, _ := env.authService.Signup(ctx, service.SignupRequest{
		Email: fmt.Sprintf("orgowner-%s@agency.test", suffix), Password: "Password123!", DisplayName: "Org Owner", OrganizationName: "Suspension Org",
	})
	_ = env.userRepo.SetEmailVerified(ctx, orgUser.User.ID)

	// Org request works normally
	wBefore := doJSONReq(env.srv, "GET", "/api/v1/clients", nil, orgUser.RawSession, orgUser.RawCSRF, orgUser.Organization.ID, "")
	if wBefore.Code != http.StatusOK {
		t.Fatalf("org user client list before suspension failed with %d: %s", wBefore.Code, wBefore.Body.String())
	}

	// Platform Admin suspends organization
	suspendBody := map[string]string{"reason": "Payment delinquency & Terms violation"}
	wSuspend := doJSONReq(env.srv, "POST", fmt.Sprintf("/api/v1/platform-admin/organizations/%s/suspend", orgUser.Organization.ID), suspendBody, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wSuspend.Code != http.StatusOK {
		t.Fatalf("suspend org failed: %d: %s", wSuspend.Code, wSuspend.Body.String())
	}

	// Org user tries to access org route -> 403 ORGANIZATION_SUSPENDED
	wOrgBlocked := doJSONReq(env.srv, "GET", "/api/v1/clients", nil, orgUser.RawSession, orgUser.RawCSRF, orgUser.Organization.ID, "")
	if wOrgBlocked.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for suspended org, got %d: %s", wOrgBlocked.Code, wOrgBlocked.Body.String())
	}
	if !strings.Contains(wOrgBlocked.Body.String(), "ORGANIZATION_SUSPENDED") {
		t.Errorf("expected ORGANIZATION_SUSPENDED error code, got: %s", wOrgBlocked.Body.String())
	}

	// Platform Admin reactivates organization
	reactivateBody := map[string]string{"reason": "Account resolved"}
	wReactivate := doJSONReq(env.srv, "POST", fmt.Sprintf("/api/v1/platform-admin/organizations/%s/reactivate", orgUser.Organization.ID), reactivateBody, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wReactivate.Code != http.StatusOK {
		t.Fatalf("reactivate org failed: %d: %s", wReactivate.Code, wReactivate.Body.String())
	}

	// Org user can access org route again
	wOrgRestored := doJSONReq(env.srv, "GET", "/api/v1/clients", nil, orgUser.RawSession, orgUser.RawCSRF, orgUser.Organization.ID, "")
	if wOrgRestored.Code != http.StatusOK {
		t.Fatalf("expected 200 OK after org reactivation, got %d: %s", wOrgRestored.Code, wOrgRestored.Body.String())
	}
}

func TestPlatformAdmin_PlanManagement_VersioningAndImmutability(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()
	ctx := context.Background()

	suffix, _ := auth.GenerateSecureToken(4)
	adminSignup, _ := env.authService.Signup(ctx, service.SignupRequest{
		Email: fmt.Sprintf("pa3-%s@pulse.test", suffix), Password: "Password123!", DisplayName: "Admin", OrganizationName: "Pulse HQ",
	})
	_ = env.userRepo.SetEmailVerified(ctx, adminSignup.User.ID)
	_, _ = env.adminRepo.GrantAdmin(ctx, adminSignup.User.ID, nil, "test")

	planSlug := fmt.Sprintf("starter-%s", strings.ToLower(suffix))
	createReq := map[string]interface{}{
		"name":           "Starter Tier",
		"slug":           planSlug,
		"description":    "Starter plan for boutique agencies",
		"price_amount":   4900,
		"currency":       "USD",
		"billing_period": "monthly",
		"limits": map[string]interface{}{
			"max_clients":           3,
			"max_assets":            5,
			"max_monitors":          10,
			"max_heartbeats":        5,
			"max_team_members":      3,
			"min_interval_seconds":  60,
			"history_retention_days": 30,
			"features": map[string]bool{
				"custom_branding": false,
				"care_reports":    true,
			},
		},
	}

	// 1. Create draft plan
	wCreate := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans", createReq, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("create plan failed: %d: %s", wCreate.Code, wCreate.Body.String())
	}
	var createdPlan map[string]interface{}
	_ = json.Unmarshal(wCreate.Body.Bytes(), &createdPlan)
	planObj := createdPlan["plan"].(map[string]interface{})
	planVersionObj := createdPlan["version"].(map[string]interface{})
	planID := planObj["id"].(string)

	// Verify initial status is draft and version is 1
	if planObj["status"] != "draft" || planVersionObj["version"].(float64) != 1 {
		t.Fatalf("expected status=draft version=1, got status=%v version=%v", planObj["status"], planVersionObj["version"])
	}

	// 2. Publish plan
	wPublish := doJSONReq(env.srv, "POST", fmt.Sprintf("/api/v1/platform-admin/plans/%s/publish", planID), nil, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wPublish.Code != http.StatusOK {
		t.Fatalf("publish plan failed: %d: %s", wPublish.Code, wPublish.Body.String())
	}

	// 3. Update published plan -> must trigger new version creation!
	updateReq := map[string]interface{}{
		"name":           "Starter Tier v2",
		"description":    "Updated starter plan with more assets",
		"price_amount":   5900,
		"currency":       "USD",
		"billing_period": "monthly",
		"limits": map[string]interface{}{
			"max_clients":           5,
			"max_assets":            10,
			"max_monitors":          15,
			"max_heartbeats":        10,
			"max_team_members":      5,
			"min_interval_seconds":  60,
			"history_retention_days": 60,
			"features": map[string]bool{
				"custom_branding": true,
				"care_reports":    true,
			},
		},
	}
	wUpdate := doJSONReq(env.srv, "PUT", fmt.Sprintf("/api/v1/platform-admin/plans/%s", planID), updateReq, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wUpdate.Code != http.StatusOK {
		t.Fatalf("update published plan failed: %d: %s", wUpdate.Code, wUpdate.Body.String())
	}
	var updatedPlan map[string]interface{}
	_ = json.Unmarshal(wUpdate.Body.Bytes(), &updatedPlan)
	updatedVersionObj := updatedPlan["version"].(map[string]interface{})

	// Verify a new draft version 2 was created
	if updatedVersionObj["version"].(float64) != 2 || updatedVersionObj["status"] != "draft" {
		t.Fatalf("expected new draft version 2, got version=%v status=%v", updatedVersionObj["version"], updatedVersionObj["status"])
	}

	// 4. Archive plan
	wArchive := doJSONReq(env.srv, "POST", fmt.Sprintf("/api/v1/platform-admin/plans/%s/archive", planID), nil, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wArchive.Code != http.StatusOK {
		t.Fatalf("archive plan failed: %d: %s", wArchive.Code, wArchive.Body.String())
	}
}

func TestPlatformAdmin_SubscriptionAssignment_AndQuotaEnforcement(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()
	ctx := context.Background()

	suffix, _ := auth.GenerateSecureToken(4)
	adminSignup, _ := env.authService.Signup(ctx, service.SignupRequest{
		Email: fmt.Sprintf("pa4-%s@pulse.test", suffix), Password: "Password123!", DisplayName: "Admin", OrganizationName: "Pulse HQ",
	})
	_ = env.userRepo.SetEmailVerified(ctx, adminSignup.User.ID)
	_, _ = env.adminRepo.GrantAdmin(ctx, adminSignup.User.ID, nil, "test")

	// Create and publish a plan with strict limits: max_clients = 1, min_interval_seconds = 120, custom_branding = false
	planSlug := fmt.Sprintf("strict-%s", strings.ToLower(suffix))
	createReq := map[string]interface{}{
		"name":           "Strict Plan",
		"slug":           planSlug,
		"description":    "Plan with tight limits for testing",
		"price_amount":   1900,
		"currency":       "USD",
		"billing_period": "monthly",
		"limits": map[string]interface{}{
			"max_clients":           1,
			"max_assets":            2,
			"max_monitors":          2,
			"max_heartbeats":        1,
			"max_team_members":      2,
			"min_interval_seconds":  120,
			"history_retention_days": 14,
			"features": map[string]bool{
				"custom_branding": false,
				"care_reports":    false,
			},
		},
	}
	wCreate := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans", createReq, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("create plan failed: %d: %s", wCreate.Code, wCreate.Body.String())
	}
	var createdPlan map[string]interface{}
	_ = json.Unmarshal(wCreate.Body.Bytes(), &createdPlan)
	planObj := createdPlan["plan"].(map[string]interface{})
	planID := planObj["id"].(string)

	wPublish := doJSONReq(env.srv, "POST", fmt.Sprintf("/api/v1/platform-admin/plans/%s/publish", planID), nil, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wPublish.Code != http.StatusOK {
		t.Fatalf("publish plan failed: %d: %s", wPublish.Code, wPublish.Body.String())
	}

	// Create agency org
	agencySignup, _ := env.authService.Signup(ctx, service.SignupRequest{
		Email: fmt.Sprintf("agency-%s@agency.test", suffix), Password: "Password123!", DisplayName: "Agency Owner", OrganizationName: "Agency Org",
	})
	_ = env.userRepo.SetEmailVerified(ctx, agencySignup.User.ID)

	// Assign plan to agency org
	assignReq := map[string]interface{}{
		"organization_id": agencySignup.Organization.ID,
		"plan_id":         planID,
	}
	wAssign := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/subscriptions", assignReq, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wAssign.Code != http.StatusOK {
		t.Fatalf("assign plan failed: %d: %s", wAssign.Code, wAssign.Body.String())
	}

	// 1. Quota check: Client 1 succeeds
	client1Req := map[string]string{"name": "Client 1"}
	wClient1 := doJSONReq(env.srv, "POST", "/api/v1/clients", client1Req, agencySignup.RawSession, agencySignup.RawCSRF, agencySignup.Organization.ID, agencySignup.RawCSRF)
	if wClient1.Code != http.StatusCreated {
		t.Fatalf("client 1 creation failed: %d: %s", wClient1.Code, wClient1.Body.String())
	}

	// 2. Quota check: Client 2 fails (max_clients = 1 exceeded!)
	client2Req := map[string]string{"name": "Client 2"}
	wClient2 := doJSONReq(env.srv, "POST", "/api/v1/clients", client2Req, agencySignup.RawSession, agencySignup.RawCSRF, agencySignup.Organization.ID, agencySignup.RawCSRF)
	if wClient2.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on exceeding max_clients quota, got %d: %s", wClient2.Code, wClient2.Body.String())
	}
	if !strings.Contains(wClient2.Body.String(), "LIMIT_EXCEEDED") {
		t.Errorf("expected LIMIT_EXCEEDED error code, got: %s", wClient2.Body.String())
	}

	// 3. Feature check: Custom branding fails (custom_branding = false)
	brandingReq := map[string]string{"company_name": "New Brand"}
	wBranding := doJSONReq(env.srv, "PATCH", "/api/v1/settings/branding", brandingReq, agencySignup.RawSession, agencySignup.RawCSRF, agencySignup.Organization.ID, agencySignup.RawCSRF)
	if wBranding.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden on custom branding feature check, got %d: %s", wBranding.Code, wBranding.Body.String())
	}
	if !strings.Contains(wBranding.Body.String(), "FEATURE_NOT_AVAILABLE") {
		t.Errorf("expected FEATURE_NOT_AVAILABLE error code, got: %s", wBranding.Body.String())
	}

	// 4. Over-limit warning check endpoint
	checkLimitsReq := map[string]interface{}{
		"organization_id": agencySignup.Organization.ID,
		"plan_id":         planID,
	}
	wCheckLimits := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/subscriptions/check-limits", checkLimitsReq, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wCheckLimits.Code != http.StatusOK {
		t.Fatalf("check limits failed: %d: %s", wCheckLimits.Code, wCheckLimits.Body.String())
	}
}

func TestPlatformAdmin_AuditLogRecording(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()
	ctx := context.Background()

	suffix, _ := auth.GenerateSecureToken(4)
	adminSignup, _ := env.authService.Signup(ctx, service.SignupRequest{
		Email: fmt.Sprintf("pa5-%s@pulse.test", suffix), Password: "Password123!", DisplayName: "Admin", OrganizationName: "Pulse HQ",
	})
	_ = env.userRepo.SetEmailVerified(ctx, adminSignup.User.ID)
	_, _ = env.adminRepo.GrantAdmin(ctx, adminSignup.User.ID, nil, "test")

	// Perform an admin action (e.g. create a plan)
	planSlug := fmt.Sprintf("audit-plan-%s", strings.ToLower(suffix))
	createReq := map[string]interface{}{
		"name":           "Audit Plan",
		"slug":           planSlug,
		"price_amount":   1000,
		"currency":       "USD",
		"billing_period": "monthly",
		"limits": map[string]interface{}{
			"max_clients":          5,
			"min_interval_seconds": 60,
		},
	}
	wCreate := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans", createReq, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("create plan failed: %d: %s", wCreate.Code, wCreate.Body.String())
	}

	// Query audit logs
	wAudit := doJSONReq(env.srv, "GET", "/api/v1/platform-admin/audit-logs", nil, adminSignup.RawSession, adminSignup.RawCSRF, "", "")
	if wAudit.Code != http.StatusOK {
		t.Fatalf("get audit logs failed: %d: %s", wAudit.Code, wAudit.Body.String())
	}
	var auditResp map[string]interface{}
	_ = json.Unmarshal(wAudit.Body.Bytes(), &auditResp)
	logs := auditResp["audit_logs"].([]interface{})
	if len(logs) == 0 {
		t.Fatalf("expected audit log entries to be recorded, got 0")
	}

	// Verify latest log is plan.created
	latest := logs[0].(map[string]interface{})
	if latest["action"] != "plan.created" {
		t.Errorf("expected latest action to be plan.created, got %v", latest["action"])
	}
	if latest["actor_user_id"] != adminSignup.User.ID {
		t.Errorf("expected actor_user_id to match admin ID %s, got %v", adminSignup.User.ID, latest["actor_user_id"])
	}
}

func TestPlatformAdmin_MinCheckIntervalEnforcement(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()
	ctx := context.Background()

	suffix, _ := auth.GenerateSecureToken(4)
	adminSignup, _ := env.authService.Signup(ctx, service.SignupRequest{
		Email: fmt.Sprintf("pa6-%s@pulse.test", suffix), Password: "Password123!", DisplayName: "Admin", OrganizationName: "Pulse HQ",
	})
	_ = env.userRepo.SetEmailVerified(ctx, adminSignup.User.ID)
	_, _ = env.adminRepo.GrantAdmin(ctx, adminSignup.User.ID, nil, "test")

	// Create and publish a plan with min_check_interval_seconds = 300 (5 minutes)
	planSlug := fmt.Sprintf("interval-plan-%s", strings.ToLower(suffix))
	createReq := map[string]interface{}{
		"name":           "5-Min Plan",
		"slug":           planSlug,
		"price_amount":   2900,
		"currency":       "USD",
		"billing_period": "monthly",
		"limits": map[string]interface{}{
			"max_clients":          5,
			"max_assets":           5,
			"max_monitors":         5,
			"min_interval_seconds": 300,
		},
	}
	wCreate := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans", createReq, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("create plan failed: %d: %s", wCreate.Code, wCreate.Body.String())
	}
	var createdPlan map[string]interface{}
	_ = json.Unmarshal(wCreate.Body.Bytes(), &createdPlan)
	planID := createdPlan["plan"].(map[string]interface{})["id"].(string)

	_ = doJSONReq(env.srv, "POST", fmt.Sprintf("/api/v1/platform-admin/plans/%s/publish", planID), nil, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)

	// Create client organization and assign plan
	orgSignup, _ := env.authService.Signup(ctx, service.SignupRequest{
		Email: fmt.Sprintf("agency2-%s@agency.test", suffix), Password: "Password123!", DisplayName: "Agency Owner", OrganizationName: "Interval Agency",
	})
	_ = env.userRepo.SetEmailVerified(ctx, orgSignup.User.ID)

	assignReq := map[string]interface{}{
		"organization_id": orgSignup.Organization.ID,
		"plan_id":         planID,
	}
	_ = doJSONReq(env.srv, "POST", "/api/v1/platform-admin/subscriptions", assignReq, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)

	// Create a client first
	clientReq := map[string]string{"name": "Client for Interval Test"}
	wClient := doJSONReq(env.srv, "POST", "/api/v1/clients", clientReq, orgSignup.RawSession, orgSignup.RawCSRF, orgSignup.Organization.ID, orgSignup.RawCSRF)
	if wClient.Code != http.StatusCreated {
		t.Fatalf("client creation failed: %d: %s", wClient.Code, wClient.Body.String())
	}
	var clientResp map[string]interface{}
	_ = json.Unmarshal(wClient.Body.Bytes(), &clientResp)
	clientID := clientResp["client"].(map[string]interface{})["id"].(string)

	// Create an asset
	assetReq := map[string]interface{}{
		"client_id": clientID,
		"name":      "Production API",
		"type":      "api",
		"url":       "https://example.com/api",
	}
	wAsset := doJSONReq(env.srv, "POST", "/api/v1/assets", assetReq, orgSignup.RawSession, orgSignup.RawCSRF, orgSignup.Organization.ID, orgSignup.RawCSRF)
	if wAsset.Code != http.StatusCreated {
		t.Fatalf("asset creation failed: %d: %s", wAsset.Code, wAsset.Body.String())
	}
	var assetResp map[string]interface{}
	_ = json.Unmarshal(wAsset.Body.Bytes(), &assetResp)
	assetID := assetResp["asset"].(map[string]interface{})["id"].(string)

	// Attempt to create a monitor with interval 60s (< 300s permitted!) -> should be rejected!
	monitorReq := map[string]interface{}{
		"asset_id":        assetID,
		"name":            "Fast Monitor",
		"type":            "http",
		"target":          "https://example.com/api/health",
		"interval_seconds": 60,
	}
	wMonitor := doJSONReq(env.srv, "POST", "/api/v1/monitors", monitorReq, orgSignup.RawSession, orgSignup.RawCSRF, orgSignup.Organization.ID, orgSignup.RawCSRF)
	if wMonitor.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for interval below plan minimum, got %d: %s", wMonitor.Code, wMonitor.Body.String())
	}
	if !strings.Contains(wMonitor.Body.String(), "LIMIT_EXCEEDED") {
		t.Errorf("expected LIMIT_EXCEEDED error code, got: %s", wMonitor.Body.String())
	}
}
