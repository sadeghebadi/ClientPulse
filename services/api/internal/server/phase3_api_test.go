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
	"time"

	"clientpulse/services/api/internal/auth"
	"clientpulse/services/api/internal/checker"
	"clientpulse/services/api/internal/mail"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/service"
	"clientpulse/services/api/internal/ssrf"
	"clientpulse/services/api/internal/testhelper"
	"github.com/jackc/pgx/v5"
)

func setupPhase3Server(t *testing.T) (*Server, *service.AuthService, *repository.UserRepository, *repository.HeartbeatRepository, *repository.IncidentRepository, func()) {
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
	chk := checker.NewChecker(ssrfGuard, "test-api-worker")

	mockMailer := mail.NewMockMailer()
	authService := service.NewAuthService(userRepo, sessionRepo, orgRepo, tokenRepo, auditRepo, mockMailer, cfg)
	orgService := service.NewOrgService(orgRepo, userRepo, sessionRepo, auditRepo, cfg)
	testLog := slog.New(slog.NewTextHandler(io.Discard, nil))

	deps := Dependencies{
		AuthHdr:          NewAuthHandler(authService, false),
		OrgHdr:           NewOrgHandler(orgService),
		ClientHdr:        NewClientHandler(clientRepo, auditRepo),
		AssetHdr:         NewAssetHandler(assetRepo, auditRepo, ssrfGuard, cfg.AppEnv),
		MonitorHdr:       NewMonitorHandler(monitorRepo, resultRepo, assetRepo, auditRepo, chk, ssrfGuard),
		HeartbeatHdr:     NewHeartbeatHandler(heartbeatRepo, cfg, testLog),
		IncidentHdr:      NewIncidentHandler(incidentRepo, eventRepo, testLog),
		AuthMiddleware:   NewAuthMiddleware(sessionRepo, userRepo, cfg),
		TenantMiddleware: NewTenantMiddleware(orgRepo, sessionRepo),
	}

	srv := New(cfg, testLog, deps)
	cleanup := func() {
		db.Close()
	}

	return srv, authService, userRepo, heartbeatRepo, incidentRepo, cleanup
}

func TestPhase3HeartbeatAndIncidentAPI(t *testing.T) {
	srv, authService, userRepo, _, incidentRepo, cleanup := setupPhase3Server(t)
	defer cleanup()

	ctx := context.Background()

	// 1. Create and verify user
	suffix, _ := auth.GenerateSecureToken(4)
	email := fmt.Sprintf("phase3-%s@testpulse.local", suffix)
	signupRes, err := authService.Signup(ctx, service.SignupRequest{
		Email:            email,
		Password:         "SecurePassword123!",
		DisplayName:      "Phase3 Owner",
		OrganizationName: "Phase3 Agency",
	})
	if err != nil {
		t.Fatalf("signup failed: %v", err)
	}

	// Verify email
	if err := userRepo.SetEmailVerified(ctx, signupRes.User.ID); err != nil {
		t.Fatalf("failed to verify email: %v", err)
	}

	doAuthedReq := func(method, path string, body interface{}) *httptest.ResponseRecorder {
		var reqBody io.Reader
		if body != nil {
			b, _ := json.Marshal(body)
			reqBody = bytes.NewReader(b)
		}
		req := httptest.NewRequest(method, path, reqBody)
		req.Header.Set("X-Organization-ID", signupRes.Organization.ID)
		req.Header.Set("X-CSRF-Token", signupRes.RawCSRF)
		req.AddCookie(&http.Cookie{Name: "clientpulse_session", Value: signupRes.RawSession})
		req.AddCookie(&http.Cookie{Name: "clientpulse_csrf", Value: signupRes.RawCSRF})
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(w, req)
		return w
	}

	// 2. Create a Client
	clientW := doAuthedReq("POST", "/api/v1/clients", map[string]interface{}{
		"name": "Acme Database Services",
	})
	if clientW.Code != http.StatusCreated {
		t.Fatalf("client creation failed (%d): %s", clientW.Code, clientW.Body.String())
	}
	var clientResp struct {
		Client struct {
			ID string `json:"id"`
		} `json:"client"`
	}
	_ = json.NewDecoder(clientW.Body).Decode(&clientResp)
	clientID := clientResp.Client.ID

	// 3. Create Heartbeat Monitor
	hbW := doAuthedReq("POST", "/api/v1/heartbeats", map[string]interface{}{
		"client_id":                 clientID,
		"name":                      "PostgreSQL Nightly Dump",
		"expected_interval_seconds": 3600,
		"grace_period_seconds":      720,
	})
	if hbW.Code != http.StatusCreated {
		t.Fatalf("heartbeat creation failed (%d): %s", hbW.Code, hbW.Body.String())
	}

	var hbCreated struct {
		Heartbeat struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"heartbeat"`
		Token             string `json:"token"`
		PingURL           string `json:"ping_url"`
		BashSnippet       string `json:"bash_snippet"`
		PowerShellSnippet string `json:"powershell_snippet"`
	}
	_ = json.NewDecoder(hbW.Body).Decode(&hbCreated)
	hbID := hbCreated.Heartbeat.ID
	token := hbCreated.Token

	if hbCreated.Heartbeat.Status != "pending" {
		t.Fatalf("expected pending heartbeat status, got %s", hbCreated.Heartbeat.Status)
	}
	if len(token) < 32 {
		t.Fatalf("token too short: %s", token)
	}
	if !strings.Contains(hbCreated.BashSnippet, "curl") || !strings.Contains(hbCreated.BashSnippet, token) {
		t.Fatalf("invalid bash snippet: %s", hbCreated.BashSnippet)
	}
	if !strings.Contains(hbCreated.PowerShellSnippet, "Invoke-RestMethod") || !strings.Contains(hbCreated.PowerShellSnippet, token) {
		t.Fatalf("invalid powershell snippet: %s", hbCreated.PowerShellSnippet)
	}

	// 4. Send Public Ping (Unauthenticated)
	pingReq := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/heartbeat-pings/%s", token), nil)
	pingW := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(pingW, pingReq)
	if pingW.Code != http.StatusOK {
		t.Fatalf("public ping failed (%d): %s", pingW.Code, pingW.Body.String())
	}
	var pingResp struct {
		Status string `json:"status"`
	}
	_ = json.NewDecoder(pingW.Body).Decode(&pingResp)
	if pingResp.Status != "ok" {
		t.Fatalf("expected status ok from ping, got %s", pingResp.Status)
	}

	// 5. Verify Heartbeat Details is now healthy
	getHbW := doAuthedReq("GET", fmt.Sprintf("/api/v1/heartbeats/%s", hbID), nil)
	if getHbW.Code != http.StatusOK {
		t.Fatalf("get heartbeat failed (%d): %s", getHbW.Code, getHbW.Body.String())
	}
	var getHbResp struct {
		Heartbeat struct {
			Status         string     `json:"status"`
			LastReceivedAt *time.Time `json:"last_received_at"`
		} `json:"heartbeat"`
	}
	_ = json.NewDecoder(getHbW.Body).Decode(&getHbResp)
	if getHbResp.Heartbeat.Status != "healthy" {
		t.Fatalf("expected healthy status, got %s", getHbResp.Heartbeat.Status)
	}

	// 6. Token Rotation
	rotateW := doAuthedReq("POST", fmt.Sprintf("/api/v1/heartbeats/%s/rotate-token", hbID), nil)
	if rotateW.Code != http.StatusOK {
		t.Fatalf("rotate token failed (%d): %s", rotateW.Code, rotateW.Body.String())
	}
	var rotResp struct {
		Token string `json:"token"`
	}
	_ = json.NewDecoder(rotateW.Body).Decode(&rotResp)
	newToken := rotResp.Token

	// Old token ping must now return 404
	oldPingReq := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/heartbeat-pings/%s", token), nil)
	oldPingW := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(oldPingW, oldPingReq)
	if oldPingW.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for old token, got %d", oldPingW.Code)
	}

	// New token ping succeeds
	newPingReq := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/heartbeat-pings/%s", newToken), nil)
	newPingW := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(newPingW, newPingReq)
	if newPingW.Code != http.StatusOK {
		t.Fatalf("expected 200 for new token ping, got %d", newPingW.Code)
	}

	// 7. Test Incident APIs: Create an open incident directly to verify List, Detail & Ack
	orgID := signupRes.Organization.ID

	incW := doAuthedReq("GET", "/api/v1/incidents", nil)
	if incW.Code != http.StatusOK {
		t.Fatalf("incidents list failed (%d): %s", incW.Code, incW.Body.String())
	}

	var testInc *repository.Incident
	err = repository.WithTenantTx(ctx, testhelper.GetTestDB(t).Pool, orgID, func(tx pgx.Tx) error {
		var created bool
		var err error
		testInc, created, err = incidentRepo.CreateHeartbeatIncidentTx(
			ctx, tx, orgID, hbID, time.Now().UTC().Add(-10*time.Minute), time.Now().UTC(),
			"heartbeat_missed", "Test heartbeat missed",
		)
		if !created {
			return fmt.Errorf("incident not created")
		}
		return err
	})
	if err != nil {
		t.Fatalf("failed to insert test incident: %v", err)
	}

	// Verify incident list returns it
	listW := doAuthedReq("GET", "/api/v1/incidents?status=open", nil)
	if listW.Code != http.StatusOK {
		t.Fatalf("failed to list open incidents (%d): %s", listW.Code, listW.Body.String())
	}
	var incListResp struct {
		Incidents []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"incidents"`
		Total int `json:"total"`
	}
	_ = json.NewDecoder(listW.Body).Decode(&incListResp)
	if incListResp.Total == 0 {
		t.Fatalf("expected at least 1 incident, got 0")
	}

	// Acknowledge incident
	ackW := doAuthedReq("POST", fmt.Sprintf("/api/v1/incidents/%s/acknowledge", testInc.ID), nil)
	if ackW.Code != http.StatusOK {
		t.Fatalf("acknowledge incident failed (%d): %s", ackW.Code, ackW.Body.String())
	}

	var ackResp struct {
		Incident struct {
			Status         string     `json:"status"`
			AcknowledgedAt *time.Time `json:"acknowledged_at"`
		} `json:"incident"`
	}
	_ = json.NewDecoder(ackW.Body).Decode(&ackResp)
	if ackResp.Incident.Status != "open" {
		t.Fatalf("acknowledgement must NOT change status to resolved; got %s", ackResp.Incident.Status)
	}
	if ackResp.Incident.AcknowledgedAt == nil {
		t.Fatalf("acknowledged_at was nil after acknowledge")
	}

	// Incident Detail endpoint
	detailW := doAuthedReq("GET", fmt.Sprintf("/api/v1/incidents/%s", testInc.ID), nil)
	if detailW.Code != http.StatusOK {
		t.Fatalf("get incident detail failed (%d): %s", detailW.Code, detailW.Body.String())
	}
	var detailResp struct {
		Incident struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"incident"`
		Events []struct {
			EventType string `json:"event_type"`
		} `json:"events"`
	}
	_ = json.NewDecoder(detailW.Body).Decode(&detailResp)
	if detailResp.Incident.ID != testInc.ID {
		t.Fatalf("detail incident ID mismatch: %s vs %s", detailResp.Incident.ID, testInc.ID)
	}
}
