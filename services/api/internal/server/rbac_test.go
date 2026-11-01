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
	"testing"

	"clientpulse/services/api/internal/auth"
	"clientpulse/services/api/internal/mail"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/service"
	"clientpulse/services/api/internal/testhelper"
)

func setupTestServer(t *testing.T) (*Server, *service.AuthService, *service.OrgService, *repository.SessionRepository, func()) {
	t.Helper()
	db := testhelper.GetTestDB(t)
	cfg := testhelper.GetTestConfig()

	userRepo := repository.NewUserRepository(db.Pool)
	sessionRepo := repository.NewSessionRepository(db.Pool)
	orgRepo := repository.NewOrganizationRepository(db.Pool)
	tokenRepo := repository.NewTokenRepository(db.Pool)
	auditRepo := repository.NewAuditRepository(db.Pool)
	mockMailer := mail.NewMockMailer()

	authService := service.NewAuthService(userRepo, sessionRepo, orgRepo, tokenRepo, auditRepo, mockMailer, cfg)
	orgService := service.NewOrgService(orgRepo, userRepo, sessionRepo, auditRepo, cfg)

	authHdr := NewAuthHandler(authService, false)
	orgHdr := NewOrgHandler(orgService)
	authMiddleware := NewAuthMiddleware(sessionRepo, userRepo, cfg)
	tenantMiddleware := NewTenantMiddleware(orgRepo, sessionRepo)

	deps := Dependencies{
		AuthHdr:          authHdr,
		OrgHdr:           orgHdr,
		AuthMiddleware:   authMiddleware,
		TenantMiddleware: tenantMiddleware,
	}

	testLog := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, testLog, deps)

	cleanup := func() {
		db.Close()
	}

	return srv, authService, orgService, sessionRepo, cleanup
}

func createTestUserAndSession(t *testing.T, authService *service.AuthService, displayName, prefix string) (*repository.User, string, string, string) {
	t.Helper()
	randID, _ := auth.GenerateSecureToken(4)
	email := fmt.Sprintf("%s-%s@example.com", prefix, randID)
	password := "Secure-Password-12345!"

	res, err := authService.Signup(context.Background(), service.SignupRequest{
		Email:            email,
		Password:         password,
		DisplayName:      displayName,
		OrganizationName: fmt.Sprintf("%s Org", displayName),
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	// Mark user verified for RBAC/CSRF test operations
	db := testhelper.GetTestDB(t)
	_, _ = db.Pool.Exec(context.Background(), "UPDATE users SET email_verified_at = NOW() WHERE id = $1", res.User.ID)
	db.Close()

	return res.User, res.Organization.ID, res.RawSession, res.RawCSRF
}

func TestRBACPermissions(t *testing.T) {
	srv, authService, orgService, _, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// 1. Create Owner user and organization
	ownerUser, orgID, ownerSession, ownerCSRF := createTestUserAndSession(t, authService, "Alice Owner", "owner")

	// 2. Create Admin user
	adminUser, _, adminSession, adminCSRF := createTestUserAndSession(t, authService, "Bob Admin", "admin")
	// Add Bob as Admin to Alice's Org
	_, err := orgService.AddMember(ctx, orgID, ownerUser.ID, service.AddMemberRequest{
		Email: adminUser.Email,
		Role:  "admin",
	})
	if err != nil {
		t.Fatalf("failed to add admin member: %v", err)
	}

	// 3. Create Member user
	memberUser, _, memberSession, memberCSRF := createTestUserAndSession(t, authService, "Charlie Member", "member")
	_, err = orgService.AddMember(ctx, orgID, ownerUser.ID, service.AddMemberRequest{
		Email: memberUser.Email,
		Role:  "member",
	})
	if err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	// Helper to execute request with session cookie and target org header
	execReq := func(method, path, sessionToken, csrfToken, targetOrgID string, body interface{}) *httptest.ResponseRecorder {
		var reqBody []byte
		if body != nil {
			reqBody, _ = json.Marshal(body)
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		if sessionToken != "" {
			req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: sessionToken})
			if csrfToken != "" {
				req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: csrfToken})
				req.Header.Set("X-CSRF-Token", csrfToken)
			}
		}
		if targetOrgID != "" {
			req.Header.Set("X-Organization-ID", targetOrgID)
		}
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)
		return rec
	}

	// Test Case 1: Member attempts to update organization settings -> MUST BE 403
	rec := execReq(http.MethodPatch, "/api/v1/organizations/current", memberSession, memberCSRF, orgID, map[string]string{
		"name":     "Hacked Organization Name",
		"locale":   "fa",
		"timezone": "Asia/Tehran",
	})
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden when Member tries to update org settings, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	// Test Case 2: Admin can update organization settings -> MUST BE 200
	rec = execReq(http.MethodPatch, "/api/v1/organizations/current", adminSession, adminCSRF, orgID, map[string]string{
		"name":     "Alice Agency Updated by Admin",
		"locale":   "en",
		"timezone": "UTC",
	})
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK when Admin updates org settings, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	// Test Case 3: Admin attempts to change member role (Owner-only endpoint) -> MUST BE 403
	rec = execReq(http.MethodPatch, fmt.Sprintf("/api/v1/organizations/current/members/%s", memberUser.ID), adminSession, adminCSRF, orgID, map[string]string{
		"role": "admin",
	})
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden when Admin tries to change member roles, got %d", rec.Code)
	}

	// Test Case 4: Owner can change member role -> MUST BE 200
	rec = execReq(http.MethodPatch, fmt.Sprintf("/api/v1/organizations/current/members/%s", memberUser.ID), ownerSession, ownerCSRF, orgID, map[string]string{
		"role": "admin",
	})
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK when Owner updates member role, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	// Test Case 5: Owner cannot demote themselves if they are the only Owner -> MUST BE 400 with error
	rec = execReq(http.MethodPatch, fmt.Sprintf("/api/v1/organizations/current/members/%s", ownerUser.ID), ownerSession, ownerCSRF, orgID, map[string]string{
		"role": "member",
	})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request when single Owner demotes self, got %d", rec.Code)
	}

	// Test Case 6: Unauthenticated request to protected endpoint -> MUST BE 401
	rec = execReq(http.MethodGet, "/api/v1/organizations/current", "", "", orgID, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for unauthenticated request, got %d", rec.Code)
	}
}
