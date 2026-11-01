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
	"clientpulse/services/api/internal/checker"
	"clientpulse/services/api/internal/mail"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/service"
	"clientpulse/services/api/internal/ssrf"
	"clientpulse/services/api/internal/testhelper"
)

func setupPhase2Server(t *testing.T) (*Server, *service.AuthService, *repository.UserRepository, func()) {
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
	ssrfGuard := ssrf.NewGuard(nil)
	chk := checker.NewChecker(ssrfGuard, "test-api-worker")

	mockMailer := mail.NewMockMailer()
	authService := service.NewAuthService(userRepo, sessionRepo, orgRepo, tokenRepo, auditRepo, mockMailer, cfg)
	orgService := service.NewOrgService(orgRepo, userRepo, sessionRepo, auditRepo, cfg)

	deps := Dependencies{
		AuthHdr:          NewAuthHandler(authService, false),
		OrgHdr:           NewOrgHandler(orgService),
		ClientHdr:        NewClientHandler(clientRepo, auditRepo),
		AssetHdr:         NewAssetHandler(assetRepo, auditRepo, ssrfGuard, cfg.AppEnv),
		MonitorHdr:       NewMonitorHandler(monitorRepo, resultRepo, assetRepo, auditRepo, chk, ssrfGuard),
		AuthMiddleware:   NewAuthMiddleware(sessionRepo, userRepo, cfg),
		TenantMiddleware: NewTenantMiddleware(orgRepo, sessionRepo),
	}

	testLog := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(cfg, testLog, deps)

	cleanup := func() {
		db.Close()
	}

	return srv, authService, userRepo, cleanup
}

func TestPhase2APIClientAndAssetWorkflow(t *testing.T) {
	srv, authService, userRepo, cleanup := setupPhase2Server(t)
	defer cleanup()

	ctx := context.Background()

	// 1. Create User A (Agency Alpha)
	randA, _ := auth.GenerateSecureToken(4)
	resA, err := authService.Signup(ctx, service.SignupRequest{
		Email:            fmt.Sprintf("alpha-%s@example.com", randA),
		Password:         "Secure-Password-12345!",
		DisplayName:      "Alice Alpha",
		OrganizationName: "Agency Alpha",
	})
	if err != nil {
		t.Fatalf("failed to signup user A: %v", err)
	}

	// 2. Create User B (Agency Beta) - unverified initially
	randB, _ := auth.GenerateSecureToken(4)
	resB, err := authService.Signup(ctx, service.SignupRequest{
		Email:            fmt.Sprintf("beta-%s@example.com", randB),
		Password:         "Secure-Password-12345!",
		DisplayName:      "Bob Beta",
		OrganizationName: "Agency Beta",
	})
	if err != nil {
		t.Fatalf("failed to signup user B: %v", err)
	}

	// Unverified User B attempts sensitive operation (Create Client) -> 403 EMAIL_NOT_VERIFIED (PRD FR-AUTH-01)
	createClientBody := []byte(`{"name":"Beta Client"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/clients", bytes.NewReader(createClientBody))
	req.Header.Set("X-Organization-ID", resB.Organization.ID)
	req.Header.Set("X-CSRF-Token", resB.RawCSRF)
	req.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: resB.RawSession})
	req.AddCookie(&http.Cookie{Name: "clientpulse_csrf", Value: resB.RawCSRF})
	w := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for unverified user, got %d (body: %s)", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("EMAIL_NOT_VERIFIED")) {
		t.Fatalf("expected EMAIL_NOT_VERIFIED error code, got %s", w.Body.String())
	}

	// Verify User A email
	if err := userRepo.SetEmailVerified(ctx, resA.User.ID); err != nil {
		t.Fatalf("failed to verify user A email: %v", err)
	}

	// 3. User A creates Client
	createClientBodyA := []byte(`{"name":"Acme Corp","website":"https://acme.com"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/clients", bytes.NewReader(createClientBodyA))
	req.Header.Set("X-Organization-ID", resA.Organization.ID)
	req.Header.Set("X-CSRF-Token", resA.RawCSRF)
	req.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: resA.RawSession})
	req.AddCookie(&http.Cookie{Name: "clientpulse_csrf", Value: resA.RawCSRF})
	w = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for client, got %d (body: %s)", w.Code, w.Body.String())
	}

	var clientResp struct {
		Client struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"client"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &clientResp)
	clientID := clientResp.Client.ID
	if clientID == "" {
		t.Fatal("expected client ID to be non-empty")
	}

	// 4. User A attempts to create Asset with prohibited SSRF URL -> Rejected with 400
	ssrfAssetBody := []byte(fmt.Sprintf(`{"client_id":%q,"name":"Internal Admin","url":"http://169.254.169.254/latest/meta-data","type":"website","environment":"production"}`, clientID))
	req = httptest.NewRequest(http.MethodPost, "/api/v1/assets", bytes.NewReader(ssrfAssetBody))
	req.Header.Set("X-Organization-ID", resA.Organization.ID)
	req.Header.Set("X-CSRF-Token", resA.RawCSRF)
	req.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: resA.RawSession})
	req.AddCookie(&http.Cookie{Name: "clientpulse_csrf", Value: resA.RawCSRF})
	w = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for SSRF URL, got %d (body: %s)", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("SSRF_PROHIBITED_URL")) {
		t.Fatalf("expected SSRF_PROHIBITED_URL, got %s", w.Body.String())
	}

	// 5. User A creates valid Asset
	validAssetBody := []byte(fmt.Sprintf(`{"client_id":%q,"name":"Acme Public Website","url":"https://acme.example.com","type":"website","environment":"production"}`, clientID))
	req = httptest.NewRequest(http.MethodPost, "/api/v1/assets", bytes.NewReader(validAssetBody))
	req.Header.Set("X-Organization-ID", resA.Organization.ID)
	req.Header.Set("X-CSRF-Token", resA.RawCSRF)
	req.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: resA.RawSession})
	req.AddCookie(&http.Cookie{Name: "clientpulse_csrf", Value: resA.RawCSRF})
	w = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for valid asset, got %d (body: %s)", w.Code, w.Body.String())
	}

	var assetResp struct {
		Asset struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"asset"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &assetResp)
	assetID := assetResp.Asset.ID

	// 6. User A creates Monitor for Asset
	createMonBody := []byte(fmt.Sprintf(`{"asset_id":%q,"method":"GET","interval_seconds":60,"timeout_seconds":10,"expected_status_min":200,"expected_status_max":399}`, assetID))
	req = httptest.NewRequest(http.MethodPost, "/api/v1/monitors", bytes.NewReader(createMonBody))
	req.Header.Set("X-Organization-ID", resA.Organization.ID)
	req.Header.Set("X-CSRF-Token", resA.RawCSRF)
	req.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: resA.RawSession})
	req.AddCookie(&http.Cookie{Name: "clientpulse_csrf", Value: resA.RawCSRF})
	w = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for monitor, got %d (body: %s)", w.Code, w.Body.String())
	}

	var monResp struct {
		Monitor struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"monitor"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &monResp)
	monitorID := monResp.Monitor.ID

	// 7. User A tests Pause & Resume Monitor
	pauseReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/monitors/%s/pause", monitorID), bytes.NewReader([]byte("{}")))
	pauseReq.Header.Set("X-Organization-ID", resA.Organization.ID)
	pauseReq.Header.Set("X-CSRF-Token", resA.RawCSRF)
	pauseReq.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: resA.RawSession})
	pauseReq.AddCookie(&http.Cookie{Name: "clientpulse_csrf", Value: resA.RawCSRF})
	w = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, pauseReq)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for pause, got %d", w.Code)
	}

	resumeReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/monitors/%s/resume", monitorID), bytes.NewReader([]byte("{}")))
	resumeReq.Header.Set("X-Organization-ID", resA.Organization.ID)
	resumeReq.Header.Set("X-CSRF-Token", resA.RawCSRF)
	resumeReq.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: resA.RawSession})
	resumeReq.AddCookie(&http.Cookie{Name: "clientpulse_csrf", Value: resA.RawCSRF})
	w = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, resumeReq)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for resume, got %d", w.Code)
	}

	// 8. Cross-Tenant Attack Test: User B attempts to access User A's client
	_ = userRepo.SetEmailVerified(ctx, resB.User.ID)

	crossReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/clients/%s", clientID), nil)
	crossReq.Header.Set("X-Organization-ID", resB.Organization.ID)
	crossReq.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: resB.RawSession})
	w = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, crossReq)

	if w.Code != http.StatusNotFound {
		t.Fatalf("SECURITY VIOLATION: Cross-tenant fetch returned %d instead of 404 (body: %s)", w.Code, w.Body.String())
	}

	// 9. Dashboard stats check
	dashReq := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/stats", nil)
	dashReq.Header.Set("X-Organization-ID", resA.Organization.ID)
	dashReq.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: resA.RawSession})
	w = httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(w, dashReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for dashboard stats, got %d", w.Code)
	}

	var stats map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &stats)
	if stats["total_clients"].(float64) < 1 {
		t.Fatalf("expected at least 1 total client, got %v", stats["total_clients"])
	}
	if stats["total_assets"].(float64) < 1 {
		t.Fatalf("expected at least 1 total asset, got %v", stats["total_assets"])
	}
}
