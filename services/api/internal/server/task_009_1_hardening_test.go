package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"clientpulse/services/api/internal/auth"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/service"
)

func createTestUser(t *testing.T, env *testAdminEnv, email, displayName, orgName string) *service.AuthResult {
	t.Helper()
	ctx := context.Background()
	signupRes, err := env.authService.Signup(ctx, service.SignupRequest{
		Email:            email,
		Password:         "Password123!",
		DisplayName:      displayName,
		OrganizationName: orgName,
	})
	if err != nil {
		t.Fatalf("signup failed for %s: %v", email, err)
	}
	if err := env.userRepo.SetEmailVerified(ctx, signupRes.User.ID); err != nil {
		t.Fatalf("email verify failed for %s: %v", email, err)
	}
	return signupRes
}

func TestTask009_1_PlanVersioningAndImmutability(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()

	suffix, _ := auth.GenerateSecureToken(4)
	adminSignup := createTestUser(t, env, fmt.Sprintf("admin-%s@pulse.test", suffix), "Admin User", "Pulse Admin")
	_, err := env.adminRepo.GrantAdmin(context.Background(), adminSignup.User.ID, nil, "Test setup")
	if err != nil {
		t.Fatalf("grant admin failed: %v", err)
	}

	// 1. Create a Draft plan with NO price (unconfigured)
	planSlug := fmt.Sprintf("draft-plan-%d", time.Now().UnixNano())
	createReq := map[string]interface{}{
		"name":        "Flexible Enterprise",
		"slug":        planSlug,
		"description": "Draft unconfigured plan",
		"max_clients": 10,
	}
	wCreate := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans", createReq, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for draft with unconfigured price, got %d: %s", wCreate.Code, wCreate.Body.String())
	}

	var planResp struct {
		Plan    repository.SubscriptionPlan        `json:"plan"`
		Version repository.SubscriptionPlanVersion `json:"version"`
	}
	_ = json.Unmarshal(wCreate.Body.Bytes(), &planResp)
	planID := planResp.Plan.ID
	versionID := planResp.Version.ID

	if planResp.Version.PriceAmount != nil {
		t.Fatalf("expected price_amount to be nil for unconfigured draft, got: %v", *planResp.Version.PriceAmount)
	}
	if planResp.Version.PublishedAt != nil {
		t.Fatalf("draft version must have published_at == nil")
	}

	// 2. Edit Draft in place
	updateReq := map[string]interface{}{
		"name":         "Flexible Enterprise Renamed",
		"description":  "Updated description",
		"price_amount": 5000,
		"currency":     "EUR",
	}
	wUpdate := doJSONReq(env.srv, "PUT", fmt.Sprintf("/api/v1/platform-admin/plans/%s", planID), updateReq, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wUpdate.Code != http.StatusOK {
		t.Fatalf("expected 200 OK updating draft plan, got %d: %s", wUpdate.Code, wUpdate.Body.String())
	}

	// 3. Reject publish if billing interval is missing
	wPublishIncomplete := doJSONReq(env.srv, "POST", fmt.Sprintf("/api/v1/platform-admin/plans/%s/publish", planID), nil, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wPublishIncomplete.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request when publishing plan without billing interval, got %d: %s", wPublishIncomplete.Code, wPublishIncomplete.Body.String())
	}

	// Now configure billing interval
	updateReq["billing_interval"] = "monthly"
	wUpdate2 := doJSONReq(env.srv, "PUT", fmt.Sprintf("/api/v1/platform-admin/plans/%s", planID), updateReq, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wUpdate2.Code != http.StatusOK {
		t.Fatalf("expected 200 OK configuring billing interval, got %d: %s", wUpdate2.Code, wUpdate2.Body.String())
	}

	// 4. Publish Draft
	wPublish := doJSONReq(env.srv, "POST", fmt.Sprintf("/api/v1/platform-admin/plans/%s/publish", planID), nil, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wPublish.Code != http.StatusOK {
		t.Fatalf("expected 200 OK publishing plan, got %d: %s", wPublish.Code, wPublish.Body.String())
	}

	// 5. Verify Published Version is immutable via direct SQL UPDATE and DELETE
	ctx := context.Background()
	_, err = env.dbPool.Pool.Exec(ctx, "UPDATE subscription_plan_versions SET price_amount = 9999 WHERE id = $1", versionID)
	if err == nil {
		t.Fatalf("IMMUTABILITY BREACH: direct SQL UPDATE on published version succeeded!")
	}

	_, err = env.dbPool.Pool.Exec(ctx, "DELETE FROM subscription_plan_versions WHERE id = $1", versionID)
	if err == nil {
		t.Fatalf("IMMUTABILITY BREACH: direct SQL DELETE on published version succeeded!")
	}

	// 6. Delete Draft Plan endpoint check: Published plan rejection
	wDeletePublished := doJSONReq(env.srv, "DELETE", fmt.Sprintf("/api/v1/platform-admin/plans/%s", planID), nil, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wDeletePublished.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request when deleting published plan, got %d: %s", wDeletePublished.Code, wDeletePublished.Body.String())
	}

	// 7. Create another pure draft and delete it cleanly
	draftOnlySlug := fmt.Sprintf("draft-only-%d", time.Now().UnixNano())
	wCreateDraftOnly := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans", map[string]interface{}{
		"name": "To Be Deleted",
		"slug": draftOnlySlug,
	}, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wCreateDraftOnly.Code != http.StatusCreated {
		t.Fatalf("failed creating draft only plan: %s", wCreateDraftOnly.Body.String())
	}
	var draftOnlyResp struct {
		Plan repository.SubscriptionPlan `json:"plan"`
	}
	_ = json.Unmarshal(wCreateDraftOnly.Body.Bytes(), &draftOnlyResp)

	wDeleteDraft := doJSONReq(env.srv, "DELETE", fmt.Sprintf("/api/v1/platform-admin/plans/%s", draftOnlyResp.Plan.ID), nil, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wDeleteDraft.Code != http.StatusOK {
		t.Fatalf("expected 200 OK deleting unused draft plan, got %d: %s", wDeleteDraft.Code, wDeleteDraft.Body.String())
	}
}

func TestTask009_1_SubscriptionLifecycleAndUnassignedLegacy(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()

	suffix, _ := auth.GenerateSecureToken(4)
	adminSignup := createTestUser(t, env, fmt.Sprintf("admin-%s@pulse.test", suffix), "Admin User", "Pulse Admin")
	_, err := env.adminRepo.GrantAdmin(context.Background(), adminSignup.User.ID, nil, "Test setup")
	if err != nil {
		t.Fatalf("grant admin failed: %v", err)
	}

	// User creates Org
	tenantSignup := createTestUser(t, env, fmt.Sprintf("tenant-%s@pulse.test", suffix), "Tenant Owner", "Tenant Org")
	orgID := tenantSignup.Organization.ID

	// 1. Unassigned legacy organization has nil active subscription but unlimited entitlement
	err = env.entitlementSvc.CheckClientCreation(context.Background(), orgID)
	if err != nil {
		t.Fatalf("expected unassigned organization to have unconstrained legacy entitlement, got: %v", err)
	}

	// 2. Create and publish a plan with max_clients = 2
	planSlug := fmt.Sprintf("starter-sub-%d", time.Now().UnixNano())
	createReq := map[string]interface{}{
		"name":             "Starter Sub",
		"slug":             planSlug,
		"price_amount":     1500,
		"currency":         "USD",
		"billing_interval": "monthly",
		"max_clients":      2,
	}
	wCreate := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans", createReq, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	var planResp struct {
		Plan    repository.SubscriptionPlan        `json:"plan"`
		Version repository.SubscriptionPlanVersion `json:"version"`
	}
	_ = json.Unmarshal(wCreate.Body.Bytes(), &planResp)
	doJSONReq(env.srv, "POST", fmt.Sprintf("/api/v1/platform-admin/plans/%s/publish", planResp.Plan.ID), nil, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)

	// 3. Assign subscription
	assignReq := map[string]string{
		"organization_id":     orgID,
		"plan_version_id":     planResp.Version.ID,
		"administrative_note": "Initial onboarding",
	}
	wAssign := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/subscriptions/assign", assignReq, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wAssign.Code != http.StatusOK {
		t.Fatalf("failed to assign subscription: %s", wAssign.Body.String())
	}

	var assignResp struct {
		Subscription repository.OrganizationSubscription `json:"subscription"`
	}
	_ = json.Unmarshal(wAssign.Body.Bytes(), &assignResp)
	subID := assignResp.Subscription.ID

	// 4. Suspend subscription
	wSuspend := doJSONReq(env.srv, "POST", fmt.Sprintf("/api/v1/platform-admin/subscriptions/%s/suspend", subID), map[string]string{"reason": "Non-payment"}, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wSuspend.Code != http.StatusOK {
		t.Fatalf("failed to suspend subscription: %s", wSuspend.Body.String())
	}

	// 5. Entitlement check must NOT fall back to legacy unassigned; it must explicitly return ErrSubscriptionInactive
	err = env.entitlementSvc.CheckClientCreation(context.Background(), orgID)
	if err == nil || !strings.Contains(err.Error(), "inactive") {
		t.Fatalf("SECURITY VIOLATION: suspended subscription was treated as unassigned legacy organization! err: %v", err)
	}

	// 6. Reactivate subscription
	wReactivate := doJSONReq(env.srv, "POST", fmt.Sprintf("/api/v1/platform-admin/subscriptions/%s/reactivate", subID), map[string]string{"reason": "Payment received"}, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	if wReactivate.Code != http.StatusOK {
		t.Fatalf("failed to reactivate subscription: %s", wReactivate.Body.String())
	}

	// Active again -> allowed under limit
	err = env.entitlementSvc.CheckClientCreation(context.Background(), orgID)
	if err != nil {
		t.Fatalf("expected active subscription to allow creation under limit, got: %v", err)
	}
}

func TestTask009_1_PlatformAdminGrants_ConcurrencyAndProtection(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()

	suffix, _ := auth.GenerateSecureToken(4)
	u1 := createTestUser(t, env, fmt.Sprintf("pa-race-1-%s@pulse.test", suffix), "Admin 1", "Pulse 1")
	u2 := createTestUser(t, env, fmt.Sprintf("pa-race-2-%s@pulse.test", suffix), "Admin 2", "Pulse 2")

	_, err := env.adminRepo.GrantAdmin(context.Background(), u1.User.ID, nil, "Setup admin 1")
	if err != nil {
		t.Fatalf("grant admin 1 failed: %v", err)
	}
	_, err = env.adminRepo.GrantAdmin(context.Background(), u2.User.ID, nil, "Setup admin 2")
	if err != nil {
		t.Fatalf("grant admin 2 failed: %v", err)
	}

	// Clean up any test grants so total active count is strictly 2
	_, _ = env.dbPool.Pool.Exec(context.Background(), "DELETE FROM platform_admin_grants WHERE user_id NOT IN ($1, $2)", u1.User.ID, u2.User.ID)

	// Test 1: Duplicate active grant is rejected/idempotent
	grantID2, err := env.adminRepo.GrantAdmin(context.Background(), u1.User.ID, nil, "Duplicate attempt")
	if err != nil {
		t.Fatalf("duplicate grant failed unexpectedly: %v", err)
	}
	if grantID2 == "" {
		t.Fatalf("expected valid grant ID for idempotent grant")
	}

	// Verify only 1 active grant for u1 in DB
	var activeCount int
	err = env.dbPool.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM platform_admin_grants WHERE user_id = $1 AND role = 'platform_admin' AND revoked_at IS NULL", u1.User.ID).Scan(&activeCount)
	if err != nil || activeCount != 1 {
		t.Fatalf("expected exactly 1 active grant, got %d (err: %v)", activeCount, err)
	}

	// Verify total active admins is exactly 2 (u1 and u2)
	var totalActive int
	err = env.dbPool.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM platform_admin_grants WHERE role = 'platform_admin' AND revoked_at IS NULL").Scan(&totalActive)
	if err != nil || totalActive != 2 {
		t.Fatalf("expected total active admins to be 2 before race test, got %d", totalActive)
	}

	// Test 2: Concurrent revocation race - both admins try to revoke their own grant simultaneously
	var wg sync.WaitGroup
	errs := make([]error, 2)
	wg.Add(2)

	go func() {
		defer wg.Done()
		errs[0] = env.adminRepo.RevokeAdmin(context.Background(), u1.User.ID, &u1.User.ID, "Revoking self")
	}()

	go func() {
		defer wg.Done()
		errs[1] = env.adminRepo.RevokeAdmin(context.Background(), u2.User.ID, &u2.User.ID, "Revoking self")
	}()

	wg.Wait()

	// Exactly one must succeed and one must fail due to last-admin protection!
	successes := 0
	failures := 0
	for _, e := range errs {
		if e == nil {
			successes++
		} else {
			failures++
			if !strings.Contains(e.Error(), "last-administrator protection active") {
				t.Errorf("unexpected revocation error: %v", e)
			}
		}
	}

	if successes != 1 || failures != 1 {
		t.Fatalf("CONCURRENCY FAILURE: expected 1 success and 1 last-admin failure, got %d successes and %d failures", successes, failures)
	}

	// Test 3: User deletion restriction - User with Platform Admin grant history cannot be cascade-deleted
	var remainingAdminID string
	if errs[0] == nil {
		remainingAdminID = u2.User.ID
	} else {
		remainingAdminID = u1.User.ID
	}

	_, err = env.dbPool.Pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", remainingAdminID)
	if err == nil {
		t.Fatalf("SECURITY VIOLATION: deleting user with platform admin history succeeded without restriction!")
	}
}

func TestTask009_1_EntitlementConcurrency(t *testing.T) {
	env := setupTestPlatformAdminServer(t)
	defer env.cleanup()

	suffix, _ := auth.GenerateSecureToken(4)
	adminSignup := createTestUser(t, env, fmt.Sprintf("admin-%s@pulse.test", suffix), "Admin User", "Pulse Admin")
	_, err := env.adminRepo.GrantAdmin(context.Background(), adminSignup.User.ID, nil, "Test setup")
	if err != nil {
		t.Fatalf("grant admin failed: %v", err)
	}

	tenantSignup := createTestUser(t, env, fmt.Sprintf("tenant-%s@pulse.test", suffix), "Tenant Owner", "Tenant Org")
	orgID := tenantSignup.Organization.ID

	// Create and assign a plan with max_clients = 2
	planSlug := fmt.Sprintf("quota-race-%d", time.Now().UnixNano())
	createReq := map[string]interface{}{
		"name":             "Quota Race Plan",
		"slug":             planSlug,
		"price_amount":     9900,
		"currency":         "USD",
		"billing_interval": "monthly",
		"max_clients":      2,
		"max_assets":       2,
		"max_monitors":     2,
		"max_heartbeats":   2,
		"max_members":      2,
	}
	wCreate := doJSONReq(env.srv, "POST", "/api/v1/platform-admin/plans", createReq, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)
	var planResp struct {
		Plan    repository.SubscriptionPlan        `json:"plan"`
		Version repository.SubscriptionPlanVersion `json:"version"`
	}
	_ = json.Unmarshal(wCreate.Body.Bytes(), &planResp)
	doJSONReq(env.srv, "POST", fmt.Sprintf("/api/v1/platform-admin/plans/%s/publish", planResp.Plan.ID), nil, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)

	doJSONReq(env.srv, "POST", "/api/v1/platform-admin/subscriptions/assign", map[string]string{
		"organization_id": orgID,
		"plan_version_id": planResp.Version.ID,
	}, adminSignup.RawSession, adminSignup.RawCSRF, "", adminSignup.RawCSRF)

	// Create 1 client initially
	_, err = env.clientRepo.CreateClient(context.Background(), orgID, "Client 1", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed creating client 1: %v", err)
	}

	// Now 1 spot remains (limit is 2, currently 1).
	// Run 5 concurrent requests attempting to create a client
	concurrency := 5
	var wg sync.WaitGroup
	wg.Add(concurrency)

	allowedCount := 0
	var mu sync.Mutex

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			w := doJSONReq(env.srv, "POST", "/api/v1/clients", map[string]string{
				"name": fmt.Sprintf("Concurrent Client %d", idx),
			}, tenantSignup.RawSession, tenantSignup.RawCSRF, orgID, tenantSignup.RawCSRF)

			if w.Code == http.StatusCreated {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	if allowedCount != 1 {
		t.Fatalf("QUOTA RACE: expected exactly 1 concurrent client creation to succeed, got %d", allowedCount)
	}

	// Verify total client count in DB is exactly 2
	var totalClients int
	_ = env.dbPool.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM clients WHERE organization_id = $1 AND archived_at IS NULL", orgID).Scan(&totalClients)
	if totalClients != 2 {
		t.Fatalf("expected exactly 2 clients in DB, got %d", totalClients)
	}
}
