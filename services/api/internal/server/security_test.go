package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"clientpulse/services/api/internal/mail"
	"clientpulse/services/api/internal/rate"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/service"
	"clientpulse/services/api/internal/testhelper"
)

func TestCSRFProtectionRejection(t *testing.T) {
	db := testhelper.GetTestDB(t)
	defer db.Close()
	cfg := testhelper.GetTestConfig()

	userRepo := repository.NewUserRepository(db.Pool)
	sessionRepo := repository.NewSessionRepository(db.Pool)
	orgRepo := repository.NewOrganizationRepository(db.Pool)
	tokenRepo := repository.NewTokenRepository(db.Pool)
	auditRepo := repository.NewAuditRepository(db.Pool)
	mockMailer := mail.NewMockMailer()

	authService := service.NewAuthService(userRepo, sessionRepo, orgRepo, tokenRepo, auditRepo, mockMailer, cfg)
	orgService := service.NewOrgService(orgRepo, userRepo, sessionRepo, auditRepo, cfg)

	deps := Dependencies{
		AuthHdr:          NewAuthHandler(authService, false),
		OrgHdr:           NewOrgHandler(orgService),
		AuthMiddleware:   NewAuthMiddleware(sessionRepo, userRepo, cfg),
		TenantMiddleware: NewTenantMiddleware(orgRepo, sessionRepo),
	}

	testLog := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, testLog, deps)

	// Create user with active session
	user, orgID, rawSession := createTestUserAndSession(t, authService, "CSRF Victim", "csrf")
	_ = user

	// 1. Mutating request with valid session cookie but NO CSRF token -> MUST FAIL (403)
	payload, _ := json.Marshal(map[string]string{"name": "Attempt Without CSRF"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/current", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-ID", orgID)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: rawSession})
	rec := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when CSRF token is missing, got: %d", rec.Code)
	}

	// 2. Mutating request with mismatched CSRF header -> MUST FAIL (403)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/current", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-ID", orgID)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: rawSession})
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "valid-cookie-token"})
	req.Header.Set("X-CSRF-Token", "attacker-forged-token")
	rec = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden when CSRF tokens mismatch, got: %d", rec.Code)
	}

	// 3. Mutating request with matching CSRF cookie and header -> SUCCEEDS (200)
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/current", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Organization-ID", orgID)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: rawSession})
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "legitimate-token-123"})
	req.Header.Set("X-CSRF-Token", "legitimate-token-123")
	rec = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK when valid CSRF token provided, got: %d (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestRateLimiterMiddleware(t *testing.T) {
	limiter := rate.NewLimiter(3, 100*time.Millisecond)
	handler := RateLimit(limiter, "test-route")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	// 3 requests allowed
	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected request %d to be allowed (200), got %d", i+1, rec.Code)
		}
	}

	// 4th request must be 429
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 Too Many Requests, got: %d", rec.Code)
	}
}
