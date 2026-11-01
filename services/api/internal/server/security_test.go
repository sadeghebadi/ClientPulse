package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"clientpulse/services/api/internal/auth"
	"clientpulse/services/api/internal/mail"
	"clientpulse/services/api/internal/rate"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/service"
	"clientpulse/services/api/internal/testhelper"
)

func TestCSRFProtectionMandatoryNegativeCases(t *testing.T) {
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

	// Create User A with active session A and CSRF token A
	userA, orgIDA, rawSessionA, validCSRFA := createTestUserAndSession(t, authService, "Alice CSRF", "alice-csrf")
	_ = userA

	// Create User B with active session B and CSRF token B
	userB, _, rawSessionB, validCSRFB := createTestUserAndSession(t, authService, "Bob CSRF", "bob-csrf")
	_ = userB
	_ = rawSessionB

	payload, _ := json.Marshal(map[string]string{"name": "Updated Org Name"})

	// Helper to create base request
	newReq := func() *http.Request {
		r := httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/current", bytes.NewReader(payload))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Organization-ID", orgIDA)
		r.Header.Set("Origin", cfg.AppBaseURL)
		r.AddCookie(&http.Cookie{Name: SessionCookieName, Value: rawSessionA})
		return r
	}

	// 1. Missing CSRF token (no cookie, no header) -> MUST FAIL (403 CSRF_TOKEN_MISSING)
	{
		req := newReq()
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("Test 1 (Missing CSRF): expected 403 Forbidden, got %d", rec.Code)
		}
	}

	// 2. Incorrect CSRF header (cookie present, header mismatched) -> MUST FAIL (403 CSRF_MISMATCH)
	{
		req := newReq()
		req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: validCSRFA})
		req.Header.Set("X-CSRF-Token", "mismatched-header-value")
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("Test 2 (Mismatched CSRF): expected 403 Forbidden, got %d", rec.Code)
		}
	}

	// 3. Forged CSRF cookie (matching header, but forged invalid HMAC) -> MUST FAIL (403 CSRF_TOKEN_INVALID)
	{
		req := newReq()
		forgedToken := "forgedtoken123456789012345678901234.forgedhmac123456789012345678901234"
		req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: forgedToken})
		req.Header.Set("X-CSRF-Token", forgedToken)
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("Test 3 (Forged CSRF): expected 403 Forbidden, got %d", rec.Code)
		}
	}

	// 4. Correct token belonging to another session (Session B token sent with Session A) -> MUST FAIL (403 CSRF_TOKEN_INVALID)
	{
		req := newReq()
		// Send Session B's CSRF token while authenticated as Session A
		req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: validCSRFB})
		req.Header.Set("X-CSRF-Token", validCSRFB)
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("Test 4 (Token belonging to another session): expected 403 Forbidden, got %d", rec.Code)
		}
	}

	// 5. Cross-origin state-changing request (Origin: https://evil-attacker.com) -> MUST FAIL (403 CSRF_ORIGIN_INVALID)
	{
		req := newReq()
		req.Header.Set("Origin", "https://evil-attacker.com")
		req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: validCSRFA})
		req.Header.Set("X-CSRF-Token", validCSRFA)
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("Test 5 (Cross-origin mutation): expected 403 Forbidden, got %d", rec.Code)
		}
	}

	// 6. Sec-Fetch-Site cross-site attempt -> MUST FAIL (403 CSRF_CROSS_SITE_BLOCKED)
	{
		req := newReq()
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: validCSRFA})
		req.Header.Set("X-CSRF-Token", validCSRFA)
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("Test 6 (Sec-Fetch-Site cross-site): expected 403 Forbidden, got %d", rec.Code)
		}
	}

	// 7. Revoked session with previously valid CSRF token -> MUST FAIL (401 UNAUTHORIZED)
	{
		// Look up session ID from raw session token hash and revoke it
		hashedToken := auth.HashToken(rawSessionA, cfg.TokenHashPepper)
		sessA, err := sessionRepo.GetSessionByTokenHash(context.Background(), hashedToken)
		if err != nil {
			t.Fatalf("failed to retrieve session A: %v", err)
		}
		err = authService.Logout(context.Background(), sessA.ID)
		if err != nil {
			t.Fatalf("failed to revoke session A: %v", err)
		}

		req := newReq()
		req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: validCSRFA})
		req.Header.Set("X-CSRF-Token", validCSRFA)
		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("Test 7 (Revoked session): expected 401 Unauthorized, got %d", rec.Code)
		}
	}

	// 8. Positive Verification: Fresh session with matching session-bound CSRF token -> SUCCEEDS (200 OK)
	{
		userC, orgIDC, rawSessionC, validCSRFC := createTestUserAndSession(t, authService, "Charlie Valid", "charlie-csrf")
		_ = userC

		cPayload, _ := json.Marshal(map[string]string{"name": "Valid Renamed Org", "locale": "en", "timezone": "UTC"})
		req := httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/current", bytes.NewReader(cPayload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Organization-ID", orgIDC)
		req.Header.Set("Origin", cfg.AppBaseURL)
		req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: rawSessionC})
		req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: validCSRFC})
		req.Header.Set("X-CSRF-Token", validCSRFC)

		rec := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Test 8 (Valid Session-Bound CSRF): expected 200 OK, got %d (body: %s)", rec.Code, rec.Body.String())
		}
	}
}

func TestRateLimiterMiddlewareAndProxyTrust(t *testing.T) {
	limiter := rate.NewLimiter(3, 100*time.Millisecond)
	handler := RateLimit(limiter, "test-route")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Case A: Untrusted remote IP sending spoofed X-Forwarded-For
	// Attacker at 203.0.113.50 sends X-Forwarded-For: 1.2.3.4, then 1.2.3.5, etc.
	// The rate limiter must use RemoteAddr (203.0.113.50) because it is NOT in trusted CIDRs.
	attackerRemote := "203.0.113.50:55432"
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.RemoteAddr = attackerRemote
		req.Header.Set("X-Forwarded-For", "198.51.100.1")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected request %d to succeed, got %d", i+1, rec.Code)
		}
	}

	// 4th request from same attacker remote IP with different spoofed X-Forwarded-For -> MUST BE 429
	{
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.RemoteAddr = attackerRemote
		req.Header.Set("X-Forwarded-For", "198.51.100.2") // attempts to bypass rate limit
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("expected 429 Too Many Requests when untrusted client spoofs XFF, got: %d", rec.Code)
		}
	}

	// Case B: Trusted proxy (127.0.0.1:4000) forwarding traffic
	// Different clients behind trusted reverse proxy are differentiated by their XFF
	trustedProxyRemote := "127.0.0.1:4000"
	clientALimiter := rate.NewLimiter(2, 200*time.Millisecond)
	clientHandler := RateLimit(clientALimiter, "proxy-test")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Client 1 behind proxy makes 2 requests
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.RemoteAddr = trustedProxyRemote
		req.Header.Set("X-Forwarded-For", "203.0.113.1")
		rec := httptest.NewRecorder()
		clientHandler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected trusted proxy client 1 request %d to succeed, got %d", i+1, rec.Code)
		}
	}

	// 3rd request from Client 1 -> 429
	{
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.RemoteAddr = trustedProxyRemote
		req.Header.Set("X-Forwarded-For", "203.0.113.1")
		rec := httptest.NewRecorder()
		clientHandler.ServeHTTP(rec, req)
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("expected 429 for Client 1 behind trusted proxy, got %d", rec.Code)
		}
	}

	// Client 2 behind same proxy should NOT be rate limited
	{
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.RemoteAddr = trustedProxyRemote
		req.Header.Set("X-Forwarded-For", "203.0.113.2")
		rec := httptest.NewRecorder()
		clientHandler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected Client 2 behind trusted proxy to succeed (not affected by Client 1), got %d", rec.Code)
		}
	}
}
