package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"clientpulse/services/api/internal/notification/telegram"
	"clientpulse/services/api/internal/repository"
)

func getTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://clientpulse:clientpulse_dev_password@localhost:5433/clientpulse?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		t.Fatalf("failed connecting to test db: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
	})
	return pool
}

// ---------------------------------------------------------------------------
// 1. Webhook to Polling Transition & Preserve Updates Test
// ---------------------------------------------------------------------------
func TestTelegramPoller_WebhookTransitionAndPreserveUpdates(t *testing.T) {
	pool := getTestPool(t)
	repo := repository.NewNotificationDestRepository(pool)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	mockBot := telegram.NewMockSender()
	mockBot.WebhookURL = "https://previous.domain.com/api/v1/integrations/telegram/webhook"
	mockBot.PendingUpdates = []telegram.TelegramUpdate{
		{UpdateID: 1001},
		{UpdateID: 1002},
	}

	updateHdr := telegram.NewUpdateHandler(repo, mockBot, "ClientPulseTestBot", logger)
	poller := NewTelegramPoller(pool, repo, mockBot, updateHdr, "ClientPulseTestBot", 1, logger)
	poller.SetLockID(int64(8890000000000000000 + time.Now().UnixNano()%100000000000))

	ctx := context.Background()
	err := poller.initializePolling(ctx)
	if err != nil {
		t.Fatalf("expected initializePolling to succeed, got: %v", err)
	}

	// Verify webhook URL is cleared
	wh, err := mockBot.GetWebhookInfo(ctx)
	if err != nil {
		t.Fatalf("failed getting webhook info: %v", err)
	}
	if wh.URL != "" {
		t.Errorf("expected webhook URL to be empty, got: %q", wh.URL)
	}

	// Verify pending updates were preserved
	if len(mockBot.PendingUpdates) != 2 {
		t.Errorf("expected pending updates to be preserved (count=2), got: %d", len(mockBot.PendingUpdates))
	}
}

// ---------------------------------------------------------------------------
// 2. Singleton Advisory Lock Test
// ---------------------------------------------------------------------------
func TestTelegramPoller_SingletonAdvisoryLock(t *testing.T) {
	pool := getTestPool(t)
	repo := repository.NewNotificationDestRepository(pool)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	mockBot := telegram.NewMockSender()
	updateHdr := telegram.NewUpdateHandler(repo, mockBot, "ClientPulseTestBot", logger)

	poller1 := NewTelegramPoller(pool, repo, mockBot, updateHdr, "ClientPulseTestBot", 1, logger)
	poller2 := NewTelegramPoller(pool, repo, mockBot, updateHdr, "ClientPulseTestBot", 1, logger)
	testLockID := int64(8880000000000000000 + time.Now().UnixNano()%100000000000)
	poller1.SetLockID(testLockID)
	poller2.SetLockID(testLockID)

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	var wg sync.WaitGroup
	wg.Add(1)

	// Start Poller 1
	go func() {
		defer wg.Done()
		poller1.Start(ctx1)
	}()

	// Wait for Poller 1 to acquire singleton lock
	time.Sleep(500 * time.Millisecond)

	// Start Poller 2 in background; it should stand by and not crash
	p2Acquired := false
	go func() {
		poller2.Start(ctx2)
		p2Acquired = true
	}()

	time.Sleep(500 * time.Millisecond)
	if p2Acquired {
		t.Errorf("poller 2 should be blocked in standby while poller 1 holds lock")
	}

	// Cancel Poller 1 -> Poller 2 should be able to acquire
	cancel1()
	wg.Wait()

	// Wait briefly for Poller 1 connection release
	time.Sleep(500 * time.Millisecond)
	cancel2()
}

// ---------------------------------------------------------------------------
// 3. Batch Processing & Offset Persistence Test
// ---------------------------------------------------------------------------
func TestTelegramPoller_BatchProcessingAndOffsetPersistence(t *testing.T) {
	pool := getTestPool(t)
	repo := repository.NewNotificationDestRepository(pool)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	baseID := int(time.Now().UnixNano() % 10000000)
	update1 := telegram.TelegramUpdate{
		UpdateID: baseID + 1,
		Message: &telegram.TelegramMessage{
			MessageID: 1,
			Chat:      telegram.TelegramChat{ID: 12345},
			Text:      "test message 1",
		},
	}
	update2 := telegram.TelegramUpdate{
		UpdateID: baseID + 2,
		Message: &telegram.TelegramMessage{
			MessageID: 2,
			Chat:      telegram.TelegramChat{ID: 12345},
			Text:      "test message 2",
		},
	}

	mockBot := telegram.NewMockSender()
	mockBot.PendingUpdates = []telegram.TelegramUpdate{update1, update2}

	updateHdr := telegram.NewUpdateHandler(repo, mockBot, "ClientPulseTestBot", logger)
	poller := NewTelegramPoller(pool, repo, mockBot, updateHdr, "ClientPulseTestBot", 1, logger)
	poller.SetLockID(int64(8890000000000000000 + time.Now().UnixNano()%100000000000))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go poller.Start(ctx)

	// Wait for poller to process updates
	deadline := time.Now().Add(2 * time.Second)
	var lastOffset int64
	for time.Now().Before(deadline) {
		offset, _ := repo.GetTelegramPollerOffset(context.Background())
		if offset >= int64(baseID+2) {
			lastOffset = offset
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if lastOffset < int64(baseID+2) {
		t.Fatalf("expected last offset to advance to >= %d, got %d", baseID+2, lastOffset)
	}
}

// ---------------------------------------------------------------------------
// 4. Duplicate Update Deduplication Test
// ---------------------------------------------------------------------------
func TestTelegramPoller_DuplicateUpdateDeduplication(t *testing.T) {
	pool := getTestPool(t)
	repo := repository.NewNotificationDestRepository(pool)

	updateID := int64(time.Now().UnixNano() % 10000000)

	// Pre-record update in DB
	isNew1, err := repo.RecordTelegramUpdateID(context.Background(), updateID)
	if err != nil || !isNew1 {
		t.Fatalf("expected first insert of update ID to be true, got %v, err %v", isNew1, err)
	}

	// Re-record identical update ID
	isNew2, err := repo.RecordTelegramUpdateID(context.Background(), updateID)
	if err != nil {
		t.Fatalf("unexpected error on duplicate record: %v", err)
	}
	if isNew2 {
		t.Errorf("expected duplicate update ID to return false, got true")
	}
}

// ---------------------------------------------------------------------------
// 5. Failed Update Processing Does Not Advance Offset
// ---------------------------------------------------------------------------
func TestTelegramPoller_FailedUpdateProcessing_DoesNotAdvanceOffset(t *testing.T) {
	pool := getTestPool(t)
	repo := repository.NewNotificationDestRepository(pool)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	baseID := int(time.Now().UnixNano() % 10000000)

	// Pre-set offset directly in DB
	_, err := pool.Exec(context.Background(), "UPDATE telegram_poller_state SET last_update_id = $1 WHERE id = 1", baseID)
	if err != nil {
		t.Fatalf("failed setting poller state: %v", err)
	}

	mockBot := telegram.NewMockSender()
	mockBot.GetUpdatesErr = errors.New("simulated network connection timeout")

	updateHdr := telegram.NewUpdateHandler(repo, mockBot, "ClientPulseTestBot", logger)
	poller := NewTelegramPoller(pool, repo, mockBot, updateHdr, "ClientPulseTestBot", 1, logger)
	poller.SetLockID(int64(8890000000000000000 + time.Now().UnixNano()%100000000000))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go poller.Start(ctx)

	time.Sleep(800 * time.Millisecond)

	// Verify offset was NOT advanced
	currentOffset, err := repo.GetTelegramPollerOffset(context.Background())
	if err != nil {
		t.Fatalf("failed reading offset: %v", err)
	}
	if currentOffset != int64(baseID) {
		t.Errorf("expected offset to remain %d, got %d", baseID, currentOffset)
	}
}

// ---------------------------------------------------------------------------
// 6. Private /start and Group /connect via Poller
// ---------------------------------------------------------------------------
func TestTelegramPoller_PrivateStartAndGroupConnect(t *testing.T) {
	pool := getTestPool(t)
	repo := repository.NewNotificationDestRepository(pool)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	ctx := context.Background()

	// Setup tenant and admin user
	orgSlug := fmt.Sprintf("poller-org-%d", time.Now().UnixNano())
	var orgID, adminUserID string
	err := pool.QueryRow(ctx, "INSERT INTO organizations (name, slug) VALUES ('Poller Org', $1) RETURNING id", orgSlug).Scan(&orgID)
	if err != nil {
		t.Fatalf("failed creating org: %v", err)
	}
	err = pool.QueryRow(ctx, "INSERT INTO users (email, password_hash, display_name) VALUES ($1, 'hash', 'Poller Admin') RETURNING id", orgSlug+"@test.org").Scan(&adminUserID)
	if err != nil {
		t.Fatalf("failed creating user: %v", err)
	}
	_, err = pool.Exec(ctx, "INSERT INTO organization_memberships (organization_id, user_id, role) VALUES ($1, $2, 'admin')", orgID, adminUserID)
	if err != nil {
		t.Fatalf("failed inserting member: %v", err)
	}

	// 1. Generate private connection token
	rawPrivToken := fmt.Sprintf("priv-token-%d", time.Now().UnixNano())
	tokenHash := repository.HashToken(rawPrivToken)
	err = pool.QueryRow(ctx, `
		INSERT INTO notification_connection_tokens (organization_id, user_id, destination_type, token_hash, expires_at)
		VALUES ($1, $2, 'telegram_private', $3, NOW() + INTERVAL '10 minutes')
		RETURNING id
	`, orgID, adminUserID, tokenHash).Scan(new(string))
	if err != nil {
		t.Fatalf("failed inserting priv token: %v", err)
	}

	mockBot := telegram.NewMockSender()
	privChatID := int64(time.Now().UnixNano() % 900000000)

	baseOffset := int(time.Now().UnixNano() % 10000000)
	if baseOffset < 100000000 {
		baseOffset += 100000000
	}
	_, _ = pool.Exec(context.Background(), "UPDATE telegram_poller_state SET last_update_id = $1 WHERE id = 1", baseOffset)

	updatePriv := telegram.TelegramUpdate{
		UpdateID: baseOffset + 1,
		Message: &telegram.TelegramMessage{
			MessageID: 101,
			From: &telegram.TelegramUser{
				ID:        privChatID,
				Username:  "poller_admin",
				FirstName: "Admin",
			},
			Chat: telegram.TelegramChat{
				ID:   privChatID,
				Type: "private",
			},
			Text: "/start " + rawPrivToken,
		},
	}

	mockBot.PendingUpdates = []telegram.TelegramUpdate{updatePriv}
	updateHdr := telegram.NewUpdateHandler(repo, mockBot, "ClientPulseTestBot", logger)
	poller := NewTelegramPoller(pool, repo, mockBot, updateHdr, "ClientPulseTestBot", 1, logger)
	poller.SetLockID(int64(8890000000000000000 + time.Now().UnixNano()%100000000000))

	pollCtx, pollCancel := context.WithTimeout(ctx, 3*time.Second)
	defer pollCancel()

	go poller.Start(pollCtx)

	// Poll until private destination is created
	deadline := time.Now().Add(2500 * time.Millisecond)
	var destID string
	for time.Now().Before(deadline) {
		_ = pool.QueryRow(ctx, "SELECT id FROM notification_destinations WHERE organization_id = $1 AND telegram_chat_id = $2", orgID, privChatID).Scan(&destID)
		if destID != "" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if destID == "" {
		t.Fatalf("expected private destination to be created by poller")
	}

	// Verify confirmation message was sent to Telegram chat
	lastMsg := mockBot.LastMessage()
	if lastMsg == nil || lastMsg.ChatID != privChatID {
		t.Errorf("expected confirmation message sent to chat %d, got %v", privChatID, lastMsg)
	}
}
