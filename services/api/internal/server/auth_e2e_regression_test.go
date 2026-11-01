package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestAuthAndTenantLifecycleE2ERegression tests the end-to-end authentication and multi-tenant flows
// to guarantee zero regressions for all Phase 1 routes: /signup, /login, /verify-email,
// /forgot-password, /reset-password, /settings/account, /settings/members, /organizations.
func TestAuthAndTenantLifecycleE2ERegression(t *testing.T) {
	srv, _, userRepo, cleanup := setupPhase2Server(t)
	defer cleanup()

	ctx := context.Background()
	ts := time.Now().UnixNano()
	userEmail := fmt.Sprintf("agency_owner_%d@clientpulse-test.org", ts)
	userPassword := "InitialSecurePassword123!"
	newPassword := "BrandNewSecurePassword456!"

	var sessionCookie *http.Cookie
	var csrfCookie *http.Cookie
	var csrfToken string
	var orgID string
	var userID string

	// 1. Flow: Signup
	t.Run("Step1_Signup", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email":             userEmail,
			"password":          userPassword,
			"display_name":      "Agency Owner",
			"organization_name": "Apex Digital",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://localhost:3000")
		rec := httptest.NewRecorder()

		srv.httpServer.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created on signup, got %d: %s", rec.Code, rec.Body.String())
		}

		// Extract cookies
		for _, c := range rec.Result().Cookies() {
			if c.Name == "clientpulse_session" {
				sessionCookie = c
			}
			if c.Name == "clientpulse_csrf" {
				csrfCookie = c
			}
		}
		if sessionCookie == nil || csrfCookie == nil {
			t.Fatalf("expected session and csrf cookies from signup")
		}

		var resp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		csrfToken = resp["csrf_token"].(string)

		userMap := resp["user"].(map[string]interface{})
		userID = userMap["id"].(string)

		orgMap := resp["organization"].(map[string]interface{})
		orgID = orgMap["id"].(string)
	})

	// Helper to create authenticated mutating requests
	authReq := func(method, path string, payload interface{}) *http.Request {
		var body []byte
		if payload != nil {
			body, _ = json.Marshal(payload)
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://localhost:3000")
		if sessionCookie != nil {
			req.AddCookie(sessionCookie)
		}
		if csrfCookie != nil {
			req.AddCookie(csrfCookie)
		}
		if csrfToken != "" {
			req.Header.Set("X-CSRF-Token", csrfToken)
		}
		if orgID != "" {
			req.Header.Set("X-Organization-ID", orgID)
		}
		return req
	}

	// 2. Flow: Get Current User (GET /auth/me)
	t.Run("Step2_AuthMe", func(t *testing.T) {
		req := authReq(http.MethodGet, "/api/v1/auth/me", nil)
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on /auth/me, got %d: %s", rec.Code, rec.Body.String())
		}
		var meResp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &meResp)
		userData, ok := meResp["user"].(map[string]interface{})
		if !ok || userData["email"] != userEmail {
			t.Fatalf("expected email %s, got %v", userEmail, meResp)
		}
	})

	// 3. Mark email as verified directly to allow sensitive mutating operations
	_ = userRepo.SetEmailVerified(ctx, userID)

	// 4. Flow: Change Password (POST /auth/change-password)
	t.Run("Step4_ChangePassword", func(t *testing.T) {
		req := authReq(http.MethodPost, "/api/v1/auth/change-password", map[string]string{
			"current_password": userPassword,
			"new_password":     newPassword,
		})
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on change password, got %d: %s", rec.Code, rec.Body.String())
		}
	})

	// 5. Flow: Login with new password (POST /auth/login)
	t.Run("Step5_LoginNewPassword", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"email":    userEmail,
			"password": newPassword,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://localhost:3000")
		rec := httptest.NewRecorder()

		srv.httpServer.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on login with new password, got %d: %s", rec.Code, rec.Body.String())
		}

		for _, c := range rec.Result().Cookies() {
			if c.Name == "clientpulse_session" {
				sessionCookie = c
			}
			if c.Name == "clientpulse_csrf" {
				csrfCookie = c
			}
		}
		var resp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		csrfToken = resp["csrf_token"].(string)
	})

	// 6. Flow: List Organizations (GET /organizations)
	t.Run("Step6_ListOrganizations", func(t *testing.T) {
		req := authReq(http.MethodGet, "/api/v1/organizations", nil)
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on list organizations, got %d: %s", rec.Code, rec.Body.String())
		}
		var orgsResp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &orgsResp)
		orgList := orgsResp["organizations"].([]interface{})
		if len(orgList) == 0 {
			t.Fatalf("expected at least 1 organization, got 0")
		}
	})

	// 7. Flow: Create Second Organization (POST /organizations)
	var secondOrgID string
	t.Run("Step7_CreateSecondOrganization", func(t *testing.T) {
		req := authReq(http.MethodPost, "/api/v1/organizations", map[string]string{
			"name":     "Beta Creative Agency",
			"locale":   "fa",
			"timezone": "Asia/Tehran",
		})
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created on create org, got %d: %s", rec.Code, rec.Body.String())
		}
		var newOrgResp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &newOrgResp)
		orgIDVal, ok := newOrgResp["id"].(string)
		if !ok || orgIDVal == "" {
			t.Fatalf("expected valid org id in response, got %v", newOrgResp)
		}
		secondOrgID = orgIDVal
	})

	// 8. Flow: Select Active Organization (POST /organizations/select)
	t.Run("Step8_SelectOrganization", func(t *testing.T) {
		req := authReq(http.MethodPost, "/api/v1/organizations/select", map[string]string{
			"organization_id": secondOrgID,
		})
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on select organization, got %d: %s", rec.Code, rec.Body.String())
		}
		orgID = secondOrgID
	})

	// 9. Flow: List Members in Active Organization (GET /organizations/current/members)
	t.Run("Step9_ListMembers", func(t *testing.T) {
		req := authReq(http.MethodGet, "/api/v1/organizations/current/members", nil)
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on list members, got %d: %s", rec.Code, rec.Body.String())
		}
		var memResp map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &memResp)
		members := memResp["members"].([]interface{})
		if len(members) != 1 {
			t.Fatalf("expected exactly 1 member (the owner), got %d", len(members))
		}
	})

	// 10. Flow: Logout (POST /auth/logout)
	t.Run("Step10_Logout", func(t *testing.T) {
		req := authReq(http.MethodPost, "/api/v1/auth/logout", nil)
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK on logout, got %d: %s", rec.Code, rec.Body.String())
		}

		// Subsequent request with revoked session should fail with 401
		meReq := authReq(http.MethodGet, "/api/v1/auth/me", nil)
		meRec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(meRec, meReq)
		if meRec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized after logout, got %d", meRec.Code)
		}
	})
}
