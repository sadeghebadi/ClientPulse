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

	"clientpulse/services/api/internal/config"
	"clientpulse/services/api/internal/notification/email"
	"clientpulse/services/api/internal/notification/telegram"
	"clientpulse/services/api/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupHardeningTestEnv(t *testing.T) (*pgxpool.Pool, *pgxpool.Pool, string, string, *NotificationHandler, *telegram.UpdateHandler, *telegram.MockSender, *config.Config) {
	ctx := context.Background()

	adminPool, err := pgxpool.New(ctx, "postgres://clientpulse:clientpulse_dev_password@localhost:5433/clientpulse?sslmode=disable")
	if err != nil {
		t.Fatalf("failed connecting admin pool: %v", err)
	}

	appPool, err := pgxpool.New(ctx, "postgres://clientpulse_app:clientpulse_app_dev_password@localhost:5433/clientpulse?sslmode=disable")
	if err != nil {
		adminPool.Close()
		t.Fatalf("failed connecting app pool: %v", err)
	}

	slug := fmt.Sprintf("hard-test-org-%d", time.Now().UnixNano())
	var orgID string
	err = adminPool.QueryRow(ctx, "INSERT INTO organizations (name, slug) VALUES ('Hardening Test Org', $1) RETURNING id", slug).Scan(&orgID)
	if err != nil {
		adminPool.Close()
		appPool.Close()
		t.Fatalf("failed creating org: %v", err)
	}

	var userID string
	userEmail := fmt.Sprintf("admin-%d@hardening.test", time.Now().UnixNano())
	err = adminPool.QueryRow(ctx, "INSERT INTO users (email, password_hash, display_name, status, email_verified_at) VALUES ($1, 'hash', 'Admin', 'active', NOW()) RETURNING id", userEmail).Scan(&userID)
	if err != nil {
		adminPool.Close()
		appPool.Close()
		t.Fatalf("failed creating user: %v", err)
	}

	_, _ = adminPool.Exec(ctx, "INSERT INTO organization_memberships (organization_id, user_id, role) VALUES ($1, $2, 'admin')", orgID, userID)

	notifRepo := repository.NewNotificationDestRepository(appPool)
	mockTelegram := telegram.NewMockSender()
	mockEmail := email.NewClient(email.SMTPConfig{
		Host:      "localhost",
		Port:      1025,
		FromEmail: "alerts@clientpulse.local",
		FromName:  "ClientPulse Alerts",
		TLSMode:   "none",
	})

	cfg := &config.Config{
		AppEnv:                "development",
		AppBaseURL:            "http://localhost:3000",
		APIBaseURL:            "http://localhost:8080",
		TelegramBotToken:      "mock-token",
		TelegramBotUsername:   "ClientPulseHardenedBot",
		TelegramWebhookSecret: "super-secret-webhook-token-32-chars",
	}

	updateHdr := telegram.NewUpdateHandler(notifRepo, mockTelegram, cfg.TelegramBotUsername, nil)
	notifHdr := NewNotificationHandler(notifRepo, updateHdr, mockTelegram, mockEmail, cfg, nil)

	return adminPool, appPool, orgID, userID, notifHdr, updateHdr, mockTelegram, cfg
}

// ---------------------------------------------------------------------------
// 1. Email Destination Ownership Verification API Integration Tests
// ---------------------------------------------------------------------------

func TestEmailDestination_OwnershipVerificationAPI(t *testing.T) {
	adminPool, appPool, orgID, userID, notifHdr, _, _, _ := setupHardeningTestEnv(t)
	defer adminPool.Close()
	defer appPool.Close()
	defer adminPool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1", orgID)

	// 1. Create email destination via API
	emailPayload := fmt.Sprintf(`{"email": "verify-me-%d@corp.com", "name": "Alerts Mailbox"}`, time.Now().UnixNano())
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/destinations/email", bytes.NewBufferString(emailPayload))
	req = req.WithContext(injectTestTenant(req.Context(), orgID, userID, "admin"))

	notifHdr.CreateEmailDestination(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var createResp struct {
		Destination repository.NotificationDestination `json:"destination"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("failed decoding json: %v", err)
	}

	destID := createResp.Destination.ID
	if createResp.Destination.Status != "pending" {
		t.Errorf("expected destination status 'pending', got '%s'", createResp.Destination.Status)
	}
	if createResp.Destination.VerifiedAt != nil {
		t.Errorf("expected VerifiedAt to be nil for new email destination")
	}

	// 2. Extract verification token from database (as user would receive via email)
	var tokenHash string
	err := adminPool.QueryRow(context.Background(), `
		SELECT verification_token_hash FROM notification_destinations WHERE id = $1
	`, destID).Scan(&tokenHash)
	if err != nil || tokenHash == "" {
		t.Fatalf("verification_token_hash was not stored in database: %v", err)
	}

	// Resend verification: Test rate limiting cooldown
	wResend := httptest.NewRecorder()
	reqResend := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/notifications/destinations/%s/resend-verification", destID), nil)
	reqResend.SetPathValue("id", destID)
	reqResend = reqResend.WithContext(injectTestTenant(reqResend.Context(), orgID, userID, "admin"))

	notifHdr.ResendEmailVerification(wResend, reqResend)
	if wResend.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 Too Many Requests within 60s cooldown, got %d: %s", wResend.Code, wResend.Body.String())
	}

	// 3. Test Invalid Token verification
	wInvalid := httptest.NewRecorder()
	reqInvalid := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/destinations/verify-email", bytes.NewBufferString(`{"token": "invalid-random-token"}`))
	notifHdr.VerifyEmailDestination(wInvalid, reqInvalid)
	if wInvalid.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for invalid token, got %d", wInvalid.Code)
	}

	// 4. Test Valid Token verification via newly generated token
	// Reset verification_sent_at to allow resend test
	_, _ = adminPool.Exec(context.Background(), `UPDATE notification_destinations SET verification_sent_at = NOW() - INTERVAL '2 minutes' WHERE id = $1`, destID)
	notifRepo := repository.NewNotificationDestRepository(appPool)
	rawValidToken, err := notifRepo.ResendEmailVerificationToken(context.Background(), orgID, destID)
	if err != nil {
		t.Fatalf("failed resending token after cooldown: %v", err)
	}

	wVerify := httptest.NewRecorder()
	reqVerify := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/destinations/verify-email", bytes.NewBufferString(fmt.Sprintf(`{"token": "%s"}`, rawValidToken)))
	notifHdr.VerifyEmailDestination(wVerify, reqVerify)
	if wVerify.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid verification, got %d: %s", wVerify.Code, wVerify.Body.String())
	}

	var verifyResp struct {
		Status      string                             `json:"status"`
		Destination repository.NotificationDestination `json:"destination"`
	}
	if err := json.Unmarshal(wVerify.Body.Bytes(), &verifyResp); err != nil {
		t.Fatalf("failed decoding verify response: %v", err)
	}
	if verifyResp.Destination.Status != "active" {
		t.Errorf("expected destination status 'active' after verification, got '%s'", verifyResp.Destination.Status)
	}

	// 5. Test Token Replay: Verifying with consumed token again MUST fail
	wReplay := httptest.NewRecorder()
	reqReplay := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/destinations/verify-email", bytes.NewBufferString(fmt.Sprintf(`{"token": "%s"}`, rawValidToken)))
	notifHdr.VerifyEmailDestination(wReplay, reqReplay)
	if wReplay.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request on replayed verification token, got %d", wReplay.Code)
	}
}

// ---------------------------------------------------------------------------
// 2. Telegram Group Approval & Replay Protection Integration Tests
// ---------------------------------------------------------------------------

func TestTelegramWebhook_ReplayProtectionAndSecretValidation(t *testing.T) {
	adminPool, appPool, orgID, _, notifHdr, _, _, _ := setupHardeningTestEnv(t)
	defer adminPool.Close()
	defer appPool.Close()
	defer adminPool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1", orgID)

	// 1. Invalid Webhook Secret Token Header -> 401 Unauthorized
	wAuthFail := httptest.NewRecorder()
	reqAuthFail := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/telegram/webhook", bytes.NewBufferString(`{}`))
	reqAuthFail.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong-secret")
	notifHdr.TelegramWebhook(wAuthFail, reqAuthFail)
	if wAuthFail.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for wrong secret token, got %d", wAuthFail.Code)
	}

	// 2. Valid Webhook Request with UpdateID
	updateID := int64(777123000 + time.Now().UnixNano()%100000)
	webhookPayload := fmt.Sprintf(`{
		"update_id": %d,
		"message": {
			"message_id": 1,
			"chat": {"id": 12345, "type": "private"},
			"from": {"id": 12345, "first_name": "Test"},
			"text": "/start"
		}
	}`, updateID)

	wFirst := httptest.NewRecorder()
	reqFirst := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/telegram/webhook", bytes.NewBufferString(webhookPayload))
	reqFirst.Header.Set("X-Telegram-Bot-Api-Secret-Token", "super-secret-webhook-token-32-chars")
	notifHdr.TelegramWebhook(wFirst, reqFirst)
	if wFirst.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid webhook, got %d: %s", wFirst.Code, wFirst.Body.String())
	}

	// 3. Replay of same UpdateID -> Handled idempotently (returns 200 OK without re-processing)
	wReplay := httptest.NewRecorder()
	reqReplay := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/telegram/webhook", bytes.NewBufferString(webhookPayload))
	reqReplay.Header.Set("X-Telegram-Bot-Api-Secret-Token", "super-secret-webhook-token-32-chars")
	notifHdr.TelegramWebhook(wReplay, reqReplay)
	if wReplay.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on replayed update, got %d", wReplay.Code)
	}
}

func TestTelegramGroup_ApprovalAndCancellationAPI(t *testing.T) {
	adminPool, appPool, orgID, adminUserID, notifHdr, _, _, _ := setupHardeningTestEnv(t)
	defer adminPool.Close()
	defer appPool.Close()
	defer adminPool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1", orgID)

	notifRepo := repository.NewNotificationDestRepository(appPool)

	// Create a pending group destination (e.g. initiated by unlinked Telegram user)
	groupChatID := -int64(4000000000 + time.Now().UnixNano()%100000000)
	dest, err := notifRepo.CreatePendingGroupDestination(context.Background(), orgID, adminUserID, "Infrastructure Alerts", groupChatID, 999111, nil)
	if err != nil {
		t.Fatalf("failed creating pending group: %v", err)
	}

	// Non-admin user attempting to approve -> 403 Forbidden
	nonAdminUserID := ""
	_ = adminPool.QueryRow(context.Background(), "INSERT INTO users (email, password_hash, display_name, status) VALUES ('member@test.org', 'hash', 'Member', 'active') RETURNING id").Scan(&nonAdminUserID)
	wForbid := httptest.NewRecorder()
	reqForbid := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/notifications/destinations/%s/approve-group", dest.ID), nil)
	reqForbid.SetPathValue("id", dest.ID)
	reqForbid = reqForbid.WithContext(injectTestTenant(reqForbid.Context(), orgID, nonAdminUserID, "member"))

	notifHdr.ApproveGroupDestination(wForbid, reqForbid)
	if wForbid.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for non-admin group approval, got %d", wForbid.Code)
	}

	// Admin approves group -> 200 OK, transitions to active
	wApprove := httptest.NewRecorder()
	reqApprove := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/notifications/destinations/%s/approve-group", dest.ID), nil)
	reqApprove.SetPathValue("id", dest.ID)
	reqApprove = reqApprove.WithContext(injectTestTenant(reqApprove.Context(), orgID, adminUserID, "admin"))

	notifHdr.ApproveGroupDestination(wApprove, reqApprove)
	if wApprove.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for admin group approval, got %d: %s", wApprove.Code, wApprove.Body.String())
	}

	var approveResp struct {
		Status      string                             `json:"status"`
		Destination repository.NotificationDestination `json:"destination"`
	}
	if err := json.Unmarshal(wApprove.Body.Bytes(), &approveResp); err != nil {
		t.Fatalf("failed decoding response: %v", err)
	}
	if approveResp.Destination.Status != "active" {
		t.Errorf("expected destination status 'active', got: %s", approveResp.Destination.Status)
	}

	// Create another pending group and test cancellation
	groupChatID2 := -int64(4100000000 + time.Now().UnixNano()%100000000)
	dest2, err := notifRepo.CreatePendingGroupDestination(context.Background(), orgID, adminUserID, "Cancelled Alerts", groupChatID2, 999222, nil)
	if err != nil {
		t.Fatalf("failed creating pending group 2: %v", err)
	}

	wCancel := httptest.NewRecorder()
	reqCancel := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/notifications/destinations/%s/cancel-group", dest2.ID), nil)
	reqCancel.SetPathValue("id", dest2.ID)
	reqCancel = reqCancel.WithContext(injectTestTenant(reqCancel.Context(), orgID, adminUserID, "admin"))

	notifHdr.CancelGroupDestination(wCancel, reqCancel)
	if wCancel.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for cancellation, got %d: %s", wCancel.Code, wCancel.Body.String())
	}
}

// ---------------------------------------------------------------------------
// 3. SMTP Production Configuration Safety Tests
// ---------------------------------------------------------------------------

func TestSMTPConfigurationSafety_ProductionValidation(t *testing.T) {
	// Base valid production config
	validProdCfg := func() *config.Config {
		return &config.Config{
			AppEnv:                        "production",
			AppBaseURL:                    "https://app.clientpulse.io",
			APIBaseURL:                    "https://api.clientpulse.io",
			Port:                          8080,
			DatabaseURL:                   "postgres://clientpulse:secret@db.internal:5432/cp",
			WorkerConcurrency:             10,
			WorkerHeartbeatTimeoutSeconds: 10,
			SessionSecret:                 "a-cryptographically-secure-production-secret-min-32-chars",
			TokenHashPepper:               "a-cryptographically-secure-production-pepper-min-32-chars",
			EmailProvider:                 "smtp",
			SMTPHost:                      "smtp.sendgrid.net",
			SMTPPort:                      587,
			SMTPUsername:                  "apikey",
			SMTPPassword:                  "SG.real-secret-key",
			SMTPFromEmail:                 "alerts@clientpulse.io",
			SMTPTLSMode:                   "starttls",
		}
	}

	// 1. Valid production config -> passes validation
	cfg := validProdCfg()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid production config to pass, got: %v", err)
	}

	// 2. Production with SMTPTLSMode="none" -> MUST FAIL
	cfgInsecureTLS := validProdCfg()
	cfgInsecureTLS.SMTPTLSMode = "none"
	if err := cfgInsecureTLS.Validate(); err == nil {
		t.Fatalf("SECURITY VIOLATION: Production allowed SMTPTLSMode='none'!")
	}

	// 3. Production with SMTPHost="mailpit" or "localhost" -> MUST FAIL
	cfgMailpit := validProdCfg()
	cfgMailpit.SMTPHost = "mailpit"
	if err := cfgMailpit.Validate(); err == nil {
		t.Fatalf("SECURITY VIOLATION: Production allowed SMTPHost='mailpit'!")
	}

	// 4. Production missing SMTP credentials -> MUST FAIL
	cfgMissingCreds := validProdCfg()
	cfgMissingCreds.SMTPPassword = ""
	if err := cfgMissingCreds.Validate(); err == nil {
		t.Fatalf("expected error on missing production SMTPPassword, got nil")
	}

	// 5. Production invalid SMTPFromEmail -> MUST FAIL
	cfgBadFrom := validProdCfg()
	cfgBadFrom.SMTPFromEmail = "invalid-email"
	if err := cfgBadFrom.Validate(); err == nil {
		t.Fatalf("expected error on invalid production SMTPFromEmail, got nil")
	}
}

// ---------------------------------------------------------------------------
// 4. Report-Ready Event Verification Tests
// ---------------------------------------------------------------------------

func TestReportReadyEvent_EmittedIdempotently(t *testing.T) {
	adminPool, appPool, orgID, _, _, _, _, _ := setupHardeningTestEnv(t)
	defer adminPool.Close()
	defer appPool.Close()
	defer adminPool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1", orgID)

	ctx := context.Background()
	eventRepo := repository.NewNotificationEventRepository(appPool)

	var clientID string
	err := adminPool.QueryRow(ctx, "INSERT INTO clients (organization_id, name) VALUES ($1, 'Acme Corp') RETURNING id", orgID).Scan(&clientID)
	if err != nil {
		t.Fatalf("failed creating client: %v", err)
	}

	reportID := fmt.Sprintf("rep-%d", time.Now().UnixNano())
	versionNumber := 1

	idempKey := fmt.Sprintf("report.ready:%s:%s:v%d", orgID, reportID, versionNumber)
	payload := map[string]interface{}{
		"report_id":   reportID,
		"title":       "Monthly Care Report",
		"client_id":   clientID,
		"version":     versionNumber,
		"storage_key": fmt.Sprintf("reports/%s/%s_v%d.pdf", orgID, reportID, versionNumber),
		"size_bytes":  45230,
	}

	// 1. Emit report.ready event for the first time -> MUST SUCCEED
	ev1, created, err := eventRepo.CreateEvent(
		ctx,
		orgID,
		"report.ready",
		nil, nil, nil, &clientID,
		idempKey,
		payload,
	)
	if err != nil {
		t.Fatalf("failed creating report.ready event: %v", err)
	}
	if !created || ev1 == nil {
		t.Fatalf("expected first event creation to succeed with created=true")
	}
	if ev1.EventType != "report.ready" {
		t.Errorf("expected event_type 'report.ready', got '%s'", ev1.EventType)
	}

	// 2. Regeneration of the same PDF version -> MUST BE IDEMPOTENT (no duplicate event created)
	ev2, created2, err := eventRepo.CreateEvent(
		ctx,
		orgID,
		"report.ready",
		nil, nil, nil, &clientID,
		idempKey,
		payload,
	)
	if err != nil {
		t.Fatalf("unexpected error on regenerated report.ready event: %v", err)
	}
	if created2 {
		t.Fatalf("DUPLICATE EVENT: Regenerated PDF created a duplicate report.ready event!")
	}
	if ev2 != nil {
		t.Errorf("expected ev2 to be nil when row already exists")
	}

	// Verify only 1 event exists in database for this report version
	var count int
	err = adminPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM notification_events WHERE idempotency_key = $1
	`, idempKey).Scan(&count)
	if err != nil || count != 1 {
		t.Errorf("expected exactly 1 event row for idempotency key, found %d (err: %v)", count, err)
	}
}
