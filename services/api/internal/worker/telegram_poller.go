package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"clientpulse/services/api/internal/notification/telegram"
	"clientpulse/services/api/internal/repository"
)

// TelegramPollerAdvisoryLockID is the global PostgreSQL advisory lock identifier for the singleton poller.
// 0x54454c454752414d is ASCII for "TELEGRAM" (6072213898144467277)
const TelegramPollerAdvisoryLockID int64 = 6072213898144467277

// TelegramPoller manages long-polling of Telegram updates as a distributed singleton.
type TelegramPoller struct {
	pool        *pgxpool.Pool
	repo        *repository.NotificationDestRepository
	botAPI      telegram.TelegramBotAPI
	updateHdr   *telegram.UpdateHandler
	botUsername string
	pollTimeout int
	logger      *slog.Logger
	lockID      int64
}

// NewTelegramPoller creates a new long-polling manager.
func NewTelegramPoller(
	pool *pgxpool.Pool,
	repo *repository.NotificationDestRepository,
	botAPI telegram.TelegramBotAPI,
	updateHdr *telegram.UpdateHandler,
	botUsername string,
	pollTimeoutSeconds int,
	logger *slog.Logger,
) *TelegramPoller {
	if logger == nil {
		logger = slog.Default()
	}
	if pollTimeoutSeconds <= 0 {
		pollTimeoutSeconds = 50
	}
	return &TelegramPoller{
		pool:        pool,
		repo:        repo,
		botAPI:      botAPI,
		updateHdr:   updateHdr,
		botUsername: botUsername,
		pollTimeout: pollTimeoutSeconds,
		logger:      logger,
		lockID:      TelegramPollerAdvisoryLockID,
	}
}

// SetLockID overrides the advisory lock ID (useful for isolated integration testing).
func (p *TelegramPoller) SetLockID(id int64) {
	p.lockID = id
}

// Start runs the singleton long-polling loop until ctx is cancelled.
func (p *TelegramPoller) Start(ctx context.Context) {
	p.logger.Info("starting telegram long-polling manager", "bot_username", p.botUsername, "poll_timeout", p.pollTimeout)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("telegram poller shutting down due to context cancellation")
			return
		default:
		}

		// 1. Acquire singleton advisory lock on dedicated connection
		conn, err := p.pool.Acquire(ctx)
		if err != nil {
			p.logger.Warn("failed acquiring db connection for poller lock; retrying", "error", err)
			time.Sleep(3 * time.Second)
			continue
		}

		var acquired bool
		err = conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", p.lockID).Scan(&acquired)
		if err != nil || !acquired {
			conn.Release()
			p.logger.Info("another telegram polling instance is active; standing by as standby replica")
			// Wait before attempting to acquire singleton lock again
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Second):
				continue
			}
		}

		p.logger.Info("acquired telegram poller singleton advisory lock; initializing polling lifecycle")

		// 2. Perform Webhook-to-Polling transition & identity check
		if err := p.initializePolling(ctx); err != nil {
			p.logger.Error("failed initializing telegram polling lifecycle", "error", err)
			_, _ = conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", p.lockID)
			conn.Release()
			time.Sleep(5 * time.Second)
			continue
		}

		// 3. Run active polling loop holding the advisory lock
		pollErr := p.runPollLoop(ctx)
		if pollErr != nil && !errors.Is(pollErr, context.Canceled) {
			p.logger.Error("telegram polling loop interrupted with error", "error", pollErr)
		}

		// Unlock advisory lock before retrying or exiting
		_, _ = conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", p.lockID)
		conn.Release()

		if errors.Is(pollErr, context.Canceled) || ctx.Err() != nil {
			p.logger.Info("telegram poller cleanly exited")
			return
		}

		// Brief delay before re-evaluating lock acquisition
		time.Sleep(2 * time.Second)
	}
}

// initializePolling verifies bot identity and ensures webhooks are safely cleared without dropping pending updates.
func (p *TelegramPoller) initializePolling(ctx context.Context) error {
	// A. Verify Bot Identity
	me, err := p.botAPI.GetMe(ctx)
	if err != nil {
		return fmt.Errorf("bot identity verification failed: %w", err)
	}
	p.logger.Info("verified telegram bot identity", "bot_id", me.ID, "username", me.Username)

	if p.botUsername != "" && !strings.EqualFold(me.Username, p.botUsername) {
		p.logger.Warn("bot username does not match configured username", "actual", me.Username, "configured", p.botUsername)
	}

	// B. Inspect Current Webhook Configuration
	whInfo, err := p.botAPI.GetWebhookInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed inspecting current webhook state: %w", err)
	}

	if whInfo.URL != "" {
		p.logger.Info("existing webhook detected; transitioning to polling mode without dropping pending updates",
			"webhook_url", whInfo.URL,
			"pending_updates", whInfo.PendingUpdateCount,
		)
		// Delete webhook with drop_pending_updates = false
		if err := p.botAPI.DeleteWebhook(ctx, false); err != nil {
			return fmt.Errorf("failed deleting outgoing webhook: %w", err)
		}

		// Confirm webhook is deleted
		postDeleteInfo, err := p.botAPI.GetWebhookInfo(ctx)
		if err == nil && postDeleteInfo.URL != "" {
			return fmt.Errorf("webhook URL still active after deleteWebhook: %s", postDeleteInfo.URL)
		}
		p.logger.Info("outgoing webhook successfully cleared; pending updates preserved for polling")
	} else {
		p.logger.Info("confirmed webhook URL is empty; ready for long polling", "pending_updates", whInfo.PendingUpdateCount)
	}

	return nil
}

// runPollLoop executes the persistent getUpdates cycle.
func (p *TelegramPoller) runPollLoop(ctx context.Context) error {
	// Load persisted offset from database
	lastProcessedID, err := p.repo.GetTelegramPollerOffset(ctx)
	if err != nil {
		return fmt.Errorf("failed loading initial poller offset: %w", err)
	}

	offset := int64(0)
	if lastProcessedID > 0 {
		offset = lastProcessedID + 1
	}
	p.logger.Info("resuming telegram polling stream", "initial_offset", offset, "last_processed_update_id", lastProcessedID)

	backoff := 1 * time.Second
	const maxBackoff = 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Long-poll getUpdates
		updates, err := p.botAPI.GetUpdates(ctx, offset, 100, p.pollTimeout, []string{"message"})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}

			// Handle RateLimitError
			var rateErr *telegram.RateLimitError
			if errors.As(err, &rateErr) {
				p.logger.Warn("telegram rate limited polling request", "retry_after", rateErr.RetryAfter)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(rateErr.RetryAfter):
					continue
				}
			}

			// Handle PermanentError (e.g. 401 Unauthorized)
			var permErr *telegram.PermanentError
			if errors.As(err, &permErr) {
				p.logger.Error("permanent error received during polling; pausing poller", "status_code", permErr.StatusCode, "message", permErr.Message)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(30 * time.Second):
					continue
				}
			}

			// Transient network/connection error: exponential backoff with jitter
			jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
			sleepDuration := backoff + jitter
			p.logger.Warn("transient error during telegram getUpdates; backing off", "error", err, "sleep", sleepDuration)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(sleepDuration):
			}

			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// Reset backoff upon successful request
		backoff = 1 * time.Second

		if len(updates) == 0 {
			// Clean timeout with no new updates, loop immediately
			continue
		}

		p.logger.Info("received telegram updates batch", "count", len(updates), "first_id", updates[0].UpdateID, "last_id", updates[len(updates)-1].UpdateID)

		// Process updates in strict ascending update ID order
		for _, u := range updates {
			updateID := int64(u.UpdateID)

			// 1. Durable deduplication check
			isNew, err := p.repo.RecordTelegramUpdateID(ctx, updateID)
			if err != nil {
				p.logger.Error("failed checking update deduplication in db; halting batch to avoid skipping", "update_id", updateID, "error", err)
				break
			}

			if isNew {
				// 2. Delegate to shared business logic processor
				if err := p.updateHdr.ProcessUpdate(ctx, &u); err != nil {
					p.logger.Error("error processing update in handler", "update_id", updateID, "error", err)
					// If update failed to process due to a transient database issue, do not advance offset
					// so that it can be retried cleanly.
					break
				}
			} else {
				p.logger.Debug("skipping duplicate or already processed update_id", "update_id", updateID)
			}

			// 3. Durably advance offset in PostgreSQL
			if err := p.repo.SaveTelegramPollerOffset(ctx, updateID); err != nil {
				p.logger.Error("failed updating persisted poller offset", "update_id", updateID, "error", err)
				break
			}

			offset = updateID + 1
		}
	}
}
