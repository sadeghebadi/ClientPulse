package worker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"clientpulse/services/api/internal/notification/telegram"
	"clientpulse/services/api/internal/repository"
)

func TestNotificationWorker_DisabledTransports(t *testing.T) {
	pool := getTestPool(t)
	repo := repository.NewNotificationDestRepository(pool)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	ctx := context.Background()
	slug := fmt.Sprintf("org-test-disabled-%d", time.Now().UnixNano())
	var orgID, userID string

	// Create test org and user in DB
	err := pool.QueryRow(ctx, "INSERT INTO organizations (name, slug) VALUES ('Test Org', $1) RETURNING id", slug).Scan(&orgID)
	if err != nil {
		t.Fatalf("failed seeding test org: %v", err)
	}
	defer pool.Exec(ctx, "DELETE FROM organizations WHERE id = $1", orgID)

	userEmail := fmt.Sprintf("user-%d@example.com", time.Now().UnixNano())
	err = pool.QueryRow(ctx, "INSERT INTO users (email, password_hash, display_name) VALUES ($1, 'hash', 'Test User') RETURNING id", userEmail).Scan(&userID)
	if err != nil {
		t.Fatalf("failed seeding test user: %v", err)
	}
	defer pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)

	// 1. Create a destination for telegram and email
	var tgDestID, emailDestID string
	err = pool.QueryRow(ctx, `
		INSERT INTO notification_destinations (organization_id, user_id, destination_type, name, telegram_chat_id, status)
		VALUES ($1, $2, 'telegram_private', 'Telegram Dest', 12345678, 'active')
		RETURNING id
	`, orgID, userID).Scan(&tgDestID)
	if err != nil {
		t.Fatalf("failed creating test tg dest: %v", err)
	}

	err = pool.QueryRow(ctx, `
		INSERT INTO notification_destinations (organization_id, user_id, destination_type, name, email_address, status)
		VALUES ($1, $2, 'email', 'Email Dest', 'admin@example.com', 'active')
		RETURNING id
	`, orgID, userID).Scan(&emailDestID)
	if err != nil {
		t.Fatalf("failed creating test email dest: %v", err)
	}

	// Create test deliveries
	delivTG, err := repo.CreateDelivery(
		ctx,
		orgID,
		nil,
		tgDestID,
		"telegram_private",
		"12345678",
		"incident.triggered",
		fmt.Sprintf("test-tg-%d", time.Now().UnixNano()),
		map[string]interface{}{"title": "Incident Alert"},
	)
	if err != nil {
		t.Fatalf("failed creating telegram test delivery: %v", err)
	}

	delivEmail, err := repo.CreateDelivery(
		ctx,
		orgID,
		nil,
		emailDestID,
		"email",
		"admin@example.com",
		"incident.triggered",
		fmt.Sprintf("test-email-%d", time.Now().UnixNano()),
		map[string]interface{}{"title": "Incident Alert"},
	)
	if err != nil {
		t.Fatalf("failed creating email test delivery: %v", err)
	}

	// Create mock worker configured with disabled transports
	mockBot := telegram.NewMockSender()
	w := NewNotificationWorker(
		repo,
		mockBot,
		nil, // nil email sender
		"https://mindu.ir",
		"worker-test-disabled",
		2,
		logger,
	).WithProviders("disabled", "disabled")

	// Deliver Telegram item
	if err := w.Deliver(ctx, delivTG); err != nil {
		t.Fatalf("expected Deliver to return nil when transport is disabled, got %v", err)
	}

	// Deliver Email item
	if err := w.Deliver(ctx, delivEmail); err != nil {
		t.Fatalf("expected Deliver to return nil when transport is disabled, got %v", err)
	}

	// Verify both deliveries in DB have status 'cancelled'
	var tgStatus, tgErrMsg string
	err = pool.QueryRow(ctx, "SELECT status, error_message FROM notification_deliveries WHERE id = $1", delivTG.ID).Scan(&tgStatus, &tgErrMsg)
	if err != nil {
		t.Fatalf("failed querying telegram delivery: %v", err)
	}
	if tgStatus != "cancelled" {
		t.Errorf("expected telegram delivery status 'cancelled', got '%s'", tgStatus)
	}
	if tgErrMsg == "" {
		t.Errorf("expected non-empty error/reason message on cancelled telegram delivery")
	}

	var emailStatus, emailErrMsg string
	err = pool.QueryRow(ctx, "SELECT status, error_message FROM notification_deliveries WHERE id = $1", delivEmail.ID).Scan(&emailStatus, &emailErrMsg)
	if err != nil {
		t.Fatalf("failed querying email delivery: %v", err)
	}
	if emailStatus != "cancelled" {
		t.Errorf("expected email delivery status 'cancelled', got '%s'", emailStatus)
	}
	if emailErrMsg == "" {
		t.Errorf("expected non-empty error/reason message on cancelled email delivery")
	}

	// Verify mockBot never received a message
	if len(mockBot.SentMessages) != 0 {
		t.Errorf("expected 0 messages sent through mockBot, got %d", len(mockBot.SentMessages))
	}
}
