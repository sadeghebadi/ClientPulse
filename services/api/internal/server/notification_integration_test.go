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
	"clientpulse/services/api/internal/tenant"
	"clientpulse/services/api/internal/worker"
	"github.com/jackc/pgx/v5/pgxpool"
)

func injectTestTenant(ctx context.Context, orgID, userID, role string) context.Context {
	ctx = tenant.WithUserContext(ctx, &tenant.UserContext{
		UserID: userID,
		Email:  "test@example.com",
	})
	return tenant.WithTenantContext(ctx, &tenant.TenantContext{
		OrganizationID: orgID,
		Role:           role,
	})
}

func TestNotificationServerAndWorkerEndToEnd(t *testing.T) {
	ctx := context.Background()

	adminPool, err := pgxpool.New(ctx, "postgres://clientpulse:clientpulse_dev_password@localhost:5433/clientpulse?sslmode=disable")
	if err != nil {
		t.Fatalf("failed connecting admin pool: %v", err)
	}
	defer adminPool.Close()

	appPool, err := pgxpool.New(ctx, "postgres://clientpulse_app:clientpulse_app_dev_password@localhost:5433/clientpulse?sslmode=disable")
	if err != nil {
		t.Fatalf("failed connecting app pool: %v", err)
	}
	defer appPool.Close()

	workerPool, err := pgxpool.New(ctx, "postgres://clientpulse_worker:clientpulse_worker_dev_password@localhost:5433/clientpulse?sslmode=disable")
	if err != nil {
		t.Fatalf("failed connecting worker pool: %v", err)
	}
	defer workerPool.Close()

	// 1. Provision Test Organization and Admin User
	orgID := ""
	slug := fmt.Sprintf("notif-e2e-org-%d", time.Now().UnixNano())
	err = adminPool.QueryRow(ctx, "INSERT INTO organizations (name, slug) VALUES ('Notif E2E Org', $1) RETURNING id", slug).Scan(&orgID)
	if err != nil {
		t.Fatalf("failed creating test org: %v", err)
	}
	defer adminPool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgID)

	userID := ""
	userEmail := fmt.Sprintf("admin-%d@notif-e2e.test", time.Now().UnixNano())
	err = adminPool.QueryRow(ctx, "INSERT INTO users (email, password_hash, display_name, status, email_verified_at) VALUES ($1, 'hash', 'Admin', 'active', NOW()) RETURNING id", userEmail).Scan(&userID)
	if err != nil {
		t.Fatalf("failed creating test user: %v", err)
	}
	defer adminPool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)

	_, err = adminPool.Exec(ctx, "INSERT INTO organization_memberships (organization_id, user_id, role) VALUES ($1, $2, 'admin')", orgID, userID)
	if err != nil {
		t.Fatalf("failed adding org membership: %v", err)
	}

	sessionToken := fmt.Sprintf("session-%d", time.Now().UnixNano())
	tokenHash := repository.HashToken(sessionToken)
	_, err = adminPool.Exec(ctx, "INSERT INTO sessions (user_id, token_hash, expires_at) VALUES ($1, $2, NOW() + INTERVAL '1 day')", userID, tokenHash)
	if err != nil {
		t.Fatalf("failed creating session: %v", err)
	}

	// 2. Set up repositories and mocks
	notifRepo := repository.NewNotificationDestRepository(appPool)
	workerNotifRepo := repository.NewNotificationDestRepository(workerPool)
	mockTelegram := telegram.NewMockSender()
	mockEmail := email.NewClient(email.SMTPConfig{
		Host:      "localhost",
		Port:      1025,
		FromEmail: "alerts@clientpulse.local",
		FromName:  "ClientPulse Alerts",
		TLSMode:   "none",
	})

	cfg := &config.Config{
		AppEnv:                 "development",
		AppBaseURL:             "http://localhost:3000",
		APIBaseURL:             "http://localhost:8080",
		Port:                   8080,
		TelegramBotToken:       "mock-token",
		TelegramBotUsername:    "ClientPulseMockBot",
		TelegramWebhookSecret: "mock-secret-123",
	}

	updateHdr := telegram.NewUpdateHandler(notifRepo, mockTelegram, cfg.TelegramBotUsername, nil)
	notifHdr := NewNotificationHandler(notifRepo, updateHdr, mockTelegram, mockEmail, cfg, nil)

	// 3. Test Generating Telegram Connect Token via API Handler
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"label": "Admin iPhone"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/destinations/telegram/connect-token", body)
	req = req.WithContext(injectTestTenant(req.Context(), orgID, userID, "admin"))

	notifHdr.CreateTelegramConnectToken(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateTelegramConnectToken returned status %d: %s", w.Code, w.Body.String())
	}

	var tokenResp struct {
		Token       string `json:"token"`
		DeepLink    string `json:"deep_link"`
		BotUsername string `json:"bot_username"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &tokenResp); err != nil {
		t.Fatalf("failed decoding token response: %v", err)
	}
	if tokenResp.Token == "" || tokenResp.BotUsername != "ClientPulseMockBot" {
		t.Errorf("invalid token response: %+v", tokenResp)
	}

	// 4. Simulate Telegram Private /start Update received by Bot
	mockChatID := int64(888123456)
	update := telegram.TelegramUpdate{
		UpdateID: 1001,
		Message: &telegram.TelegramMessage{
			MessageID: 55,
			From: &telegram.TelegramUser{
				ID:        999001,
				FirstName: "Alice",
				Username:  "alice_alerts",
			},
			Chat: telegram.TelegramChat{
				ID:   mockChatID,
				Type: "private",
			},
			Text: fmt.Sprintf("/start %s", tokenResp.Token),
		},
	}

	err = updateHdr.ProcessUpdate(ctx, &update)
	if err != nil {
		t.Fatalf("ProcessUpdate private failed: %v", err)
	}

	// Verify confirmation message was sent to Telegram user
	lastMsg := mockTelegram.LastMessage()
	if lastMsg == nil || lastMsg.ChatID != mockChatID {
		t.Fatalf("expected confirmation message to chat %d, got %+v", mockChatID, lastMsg)
	}

	// 5. Verify Destination in database
	destinations, err := notifRepo.ListDestinations(ctx, orgID)
	if err != nil || len(destinations) != 1 {
		t.Fatalf("expected 1 destination for org, got %d (err: %v)", len(destinations), err)
	}
	privDest := destinations[0]
	if privDest.DestinationType != "telegram_private" || *privDest.TelegramChatID != mockChatID {
		t.Errorf("destination details mismatch: %+v", privDest)
	}

	// 6. Test Group Connect flow
	wGroup := httptest.NewRecorder()
	reqGroup := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/destinations/telegram/group-token", bytes.NewBufferString(`{"label": "DevOps Alerts"}`))
	reqGroup = reqGroup.WithContext(injectTestTenant(reqGroup.Context(), orgID, userID, "admin"))

	notifHdr.CreateTelegramGroupToken(wGroup, reqGroup)
	if wGroup.Code != http.StatusCreated {
		t.Fatalf("CreateTelegramGroupToken failed: %d %s", wGroup.Code, wGroup.Body.String())
	}
	var groupTokenResp struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(wGroup.Body.Bytes(), &groupTokenResp)

	mockGroupChatID := int64(-1009876543210)
	groupUpdate := telegram.TelegramUpdate{
		UpdateID: 1002,
		Message: &telegram.TelegramMessage{
			MessageID: 56,
			From: &telegram.TelegramUser{
				ID:        999001,
				FirstName: "Alice",
			},
			Chat: telegram.TelegramChat{
				ID:    mockGroupChatID,
				Type:  "supergroup",
				Title: "Production SRE On-Call",
			},
			Text: fmt.Sprintf("/connect@ClientPulseMockBot %s", groupTokenResp.Code),
		},
	}

	err = updateHdr.ProcessUpdate(ctx, &groupUpdate)
	if err != nil {
		t.Fatalf("ProcessUpdate group failed: %v", err)
	}

	destinations, err = notifRepo.ListDestinations(ctx, orgID)
	if err != nil || len(destinations) != 2 {
		t.Fatalf("expected 2 destinations, got %d", len(destinations))
	}

	// 7. Test Immediate Test Notification Action
	wTest := httptest.NewRecorder()
	reqTest := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/notifications/destinations/%s/test", privDest.ID), nil)
	reqTest.SetPathValue("id", privDest.ID)
	reqTest = reqTest.WithContext(injectTestTenant(reqTest.Context(), orgID, userID, "admin"))

	notifHdr.TestDestination(wTest, reqTest)
	if wTest.Code != http.StatusOK {
		t.Fatalf("TestDestination failed: %d %s", wTest.Code, wTest.Body.String())
	}

	// 8. Test Background Worker processing: Fan out event to destinations
	workerInstance := worker.NewNotificationWorker(
		workerNotifRepo,
		mockTelegram,
		mockEmail,
		"http://localhost:3000",
		"test-e2e-worker",
		2,
		nil,
	)

	// Trigger incident.opened event
	idemp := fmt.Sprintf("incident-opened-e2e-%d", time.Now().UnixNano())
	_, err = adminPool.Exec(ctx, `
		INSERT INTO notification_events (organization_id, event_type, idempotency_key, payload, status)
		VALUES ($1, 'incident.opened', $2, '{"monitor_name": "API Gateway", "failure_category": "502_bad_gateway"}'::jsonb, 'pending')
	`, orgID, idemp)
	if err != nil {
		t.Fatalf("failed inserting event: %v", err)
	}

	// Dispatch
	dispatched, err := workerNotifRepo.DispatchPendingEvents(ctx, 10)
	if err != nil || dispatched == 0 {
		t.Fatalf("failed dispatching event: %v (count: %d)", err, dispatched)
	}

	// Claim due deliveries
	claimed, err := workerNotifRepo.ClaimDueDeliveries(ctx, "test-e2e-worker", 10)
	if err != nil || len(claimed) == 0 {
		t.Fatalf("failed claiming deliveries: %v (len: %d)", err, len(claimed))
	}

	// Deliver each claimed delivery
	for _, d := range claimed {
		if d.OrganizationID == orgID {
			if err := workerInstance.Deliver(ctx, d); err != nil {
				t.Fatalf("delivery failed: %v", err)
			}
		}
	}

	// Verify Delivery History
	deliveries, total, err := notifRepo.ListDeliveries(ctx, orgID, 10, 0, "", "")
	if err != nil {
		t.Fatalf("ListDeliveries failed: %v", err)
	}
	if total < 2 {
		t.Fatalf("expected at least 2 delivery records (1 test + dispatched), got %d", total)
	}

	for _, d := range deliveries {
		if d.Status != "sent" {
			t.Errorf("expected delivery status 'sent', got '%s'", d.Status)
		}
		if d.SentAt == nil {
			t.Errorf("expected sent_at timestamp to be populated")
		}
	}

	// 9. Disconnect Destination and verify future events are suppressed for it
	wDel := httptest.NewRecorder()
	reqDel := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/notifications/destinations/%s", privDest.ID), nil)
	reqDel.SetPathValue("id", privDest.ID)
	reqDel = reqDel.WithContext(injectTestTenant(reqDel.Context(), orgID, userID, "admin"))

	notifHdr.DeleteDestination(wDel, reqDel)
	if wDel.Code != http.StatusOK {
		t.Fatalf("DeleteDestination failed: %d %s", wDel.Code, wDel.Body.String())
	}

	destinationsAfter, _ := notifRepo.ListDestinations(ctx, orgID)
	if len(destinationsAfter) != 1 {
		t.Errorf("expected 1 active destination after disconnect, got %d", len(destinationsAfter))
	}
}
