package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"clientpulse/services/api/internal/notification"
	"clientpulse/services/api/internal/notification/email"
	"clientpulse/services/api/internal/notification/telegram"
	notiftg "clientpulse/services/api/internal/notification/telegram"
	"clientpulse/services/api/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	smtpTransientRegex = regexp.MustCompile(`\b4[0-9]{2}\b`)
	smtpPermanentRegex = regexp.MustCompile(`\b5[0-9]{2}\b`)
)

// Delivery Guarantee Contract:
// - Multi-tenant Durable Queue with At-Least-Once Delivery.
// - Fan-out and queue insertions are strictly idempotent per domain event and destination.
// - External delivery providers (Telegram Bot API, SMTP Relays) do not participate in two-phase distributed transactions.
// - Delivery states:
//     * queued: Persisted in PostgreSQL, awaiting worker claim.
//     * sending: Claimed by a worker instance using SELECT FOR UPDATE SKIP LOCKED.
//     * sent: External provider accepted transmission (HTTP 200 OK / SMTP 250 OK).
//             Note: Provider acceptance guarantees provider receipt, not inbox/device display.
//     * failed: Permanent error (invalid recipient, bot blocked) or exhausted retry budget.
//     * retrying: Transient error (rate limit, network timeout) with exponential backoff and jitter.
//     * uncertain: Worker process crashed or lost connectivity in-flight after transmitting;
//                  the outcome cannot be proven without external verification, avoiding silent duplicates.
type NotificationWorker struct {
	repo            *repository.NotificationDestRepository
	telegramSender  telegram.TelegramSender
	emailSender     email.EmailSender
	appBaseURL      string
	workerID        string
	logger          *slog.Logger
	concurrency     int
	contextResolver *notification.NotificationContextResolver
	telegramMode    string
	emailProvider   string
	deliveryTimeout time.Duration
}

func NewNotificationWorker(
	repo *repository.NotificationDestRepository,
	telegramSender telegram.TelegramSender,
	emailSender email.EmailSender,
	appBaseURL string,
	workerID string,
	concurrency int,
	logger *slog.Logger,
) *NotificationWorker {
	if logger == nil {
		logger = slog.Default()
	}
	if concurrency <= 0 {
		concurrency = 5
	}
	return &NotificationWorker{
		repo:            repo,
		telegramSender:  telegramSender,
		emailSender:     emailSender,
		appBaseURL:      strings.TrimRight(appBaseURL, "/"),
		workerID:        workerID,
		concurrency:     concurrency,
		logger:          logger,
		deliveryTimeout: 15 * time.Second,
	}
}

// WithDeliveryTimeout sets the bounded per-delivery timeout (default is 15s).
func (w *NotificationWorker) WithDeliveryTimeout(timeout time.Duration) *NotificationWorker {
	if timeout > 0 {
		w.deliveryTimeout = timeout
	}
	return w
}

// WithProviders sets the transport mode for Telegram and Email.
func (w *NotificationWorker) WithProviders(telegramMode, emailProvider string) *NotificationWorker {
	w.telegramMode = telegramMode
	w.emailProvider = emailProvider
	return w
}

// WithPool configures a database connection pool for rich runtime context resolution.
func (w *NotificationWorker) WithPool(pool *pgxpool.Pool) *NotificationWorker {
	if pool != nil {
		w.contextResolver = notification.NewNotificationContextResolver(pool)
	}
	return w
}

// Start runs the notification dispatcher and delivery loops until ctx is cancelled.
func (w *NotificationWorker) Start(ctx context.Context) {
	w.logger.Info("starting notification delivery worker", "worker_id", w.workerID, "concurrency", w.concurrency)

	var wg sync.WaitGroup
	wg.Add(3)

	// 1. Dispatcher loop: event -> deliveries fan-out
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				count, err := w.repo.DispatchPendingEvents(ctx, 50)
				if err != nil {
					w.logger.Error("failed dispatching notification events", "error", err)
				} else if count > 0 {
					w.logger.Debug("dispatched notification events", "count", count)
				}
			}
		}
	}()

	// 2. Delivery worker loop: claim due deliveries and send
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.pollAndDeliver(ctx)
			}
		}
	}()

	// 3. Stale deliveries recovery loop
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				requeued, err := w.repo.RequeueStaleDeliveries(ctx, 5*time.Minute)
				if err != nil {
					w.logger.Error("failed requeueing stale deliveries", "error", err)
				} else if requeued > 0 {
					w.logger.Warn("re-queued stuck notification deliveries", "count", requeued)
				}
			}
		}
	}()

	wg.Wait()
	w.logger.Info("notification delivery worker stopped")
}

func (w *NotificationWorker) pollAndDeliver(ctx context.Context) {
	deliveries, err := w.repo.ClaimDueDeliveries(ctx, w.workerID, w.concurrency*2)
	if err != nil {
		w.logger.Error("failed claiming due notification deliveries", "error", err)
		return
	}
	if len(deliveries) == 0 {
		return
	}

	sem := make(chan struct{}, w.concurrency)
	var wg sync.WaitGroup

	for _, d := range deliveries {
		wg.Add(1)
		sem <- struct{}{}

		go func(deliv *repository.NotificationDelivery) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := w.Deliver(ctx, deliv); err != nil {
				w.logger.Warn("notification delivery failed",
					"delivery_id", deliv.ID,
					"type", deliv.DestinationType,
					"recipient", deliv.Recipient,
					"error", err,
				)
			} else {
				w.logger.Info("notification delivery succeeded",
					"delivery_id", deliv.ID,
					"type", deliv.DestinationType,
					"event_type", deliv.EventType,
				)
			}
		}(d)
	}

	wg.Wait()
}

// Deliver processes and transmits a single notification delivery item.
func (w *NotificationWorker) Deliver(ctx context.Context, d *repository.NotificationDelivery) error {
	timeout := w.deliveryTimeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	deliveryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch d.DestinationType {
	case "telegram_private", "telegram_group":
		return w.deliverTelegram(ctx, deliveryCtx, d)
	case "email":
		return w.deliverEmail(ctx, deliveryCtx, d)
	default:
		if w.repo != nil {
			return w.repo.MarkDeliveryFailed(ctx, d.ID, w.workerID, fmt.Sprintf("unsupported destination type: %s", d.DestinationType), 0, true)
		}
		return fmt.Errorf("unsupported destination type: %s", d.DestinationType)
	}
}

func (w *NotificationWorker) deliverTelegram(ctx, deliveryCtx context.Context, d *repository.NotificationDelivery) error {
	if strings.EqualFold(w.telegramMode, "disabled") || w.telegramSender == nil {
		w.logger.Info("telegram delivery transport disabled; cancelling delivery", "delivery_id", d.ID)
		if w.repo != nil {
			return w.repo.MarkDeliveryCancelled(ctx, d.ID, w.workerID, "transport disabled: Telegram delivery is intentionally disabled in this deployment")
		}
		return nil
	}

	chatID, err := strconv.ParseInt(d.Recipient, 10, 64)
	if err != nil {
		if w.repo != nil {
			_ = w.repo.MarkDeliveryFailed(ctx, d.ID, w.workerID, fmt.Sprintf("invalid telegram chat ID '%s': %v", d.Recipient, err), 0, true)
		}
		return err
	}

	destLocale, _ := d.Payload["dest_locale"].(string)
	var notifCtx *notification.NotificationContext
	if w.contextResolver != nil {
		notifCtx = w.contextResolver.ResolveAndRefreshContext(deliveryCtx, d.OrganizationID, d.EventType, d.Payload, w.appBaseURL, destLocale)
	} else {
		notifCtx = notification.ParseContextFromPayload(d.EventType, d.Payload, w.appBaseURL, destLocale)
	}

	text := notiftg.RenderTelegramMessage(notifCtx)

	sendErr := w.telegramSender.SendMessage(deliveryCtx, chatID, text, "HTML")
	if sendErr == nil {
		if w.repo != nil {
			return w.repo.MarkDeliverySent(ctx, d.ID, w.workerID)
		}
		return nil
	}

	// If Telegram rejected message due to entity parsing, fallback to unformatted plain-text
	if strings.Contains(strings.ToLower(sendErr.Error()), "can't parse entities") {
		plainText := notiftg.RenderTelegramMessagePlainText(notifCtx)
		sendErr = w.telegramSender.SendMessage(deliveryCtx, chatID, plainText, "")
		if sendErr == nil {
			if w.repo != nil {
				return w.repo.MarkDeliverySent(ctx, d.ID, w.workerID)
			}
			return nil
		}
	}

	// Classify error
	var rateErr *telegram.RateLimitError
	var permErr *telegram.PermanentError

	if isRateLimit(sendErr, &rateErr) {
		jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
		backoff := rateErr.RetryAfter + jitter
		if w.repo != nil {
			_ = w.repo.MarkDeliveryFailed(ctx, d.ID, w.workerID, rateErr.Error(), backoff, false)
		}
		return sendErr
	}

	if isPermanentTelegram(sendErr, &permErr) {
		if w.repo != nil {
			_ = w.repo.MarkDeliveryFailed(ctx, d.ID, w.workerID, permErr.Error(), 0, true)
		}
		return sendErr
	}

	// Temporary network / internal error: exponential backoff + jitter
	backoff := calculateBackoff(d.RetryCount)
	if w.repo != nil {
		_ = w.repo.MarkDeliveryFailed(ctx, d.ID, w.workerID, sendErr.Error(), backoff, false)
	}
	return sendErr
}

func (w *NotificationWorker) deliverEmail(ctx, deliveryCtx context.Context, d *repository.NotificationDelivery) error {
	if strings.EqualFold(w.emailProvider, "disabled") || strings.EqualFold(w.emailProvider, "none") || w.emailSender == nil {
		w.logger.Info("email delivery transport disabled; cancelling delivery", "delivery_id", d.ID)
		if w.repo != nil {
			return w.repo.MarkDeliveryCancelled(ctx, d.ID, w.workerID, "transport disabled: Email delivery is intentionally disabled in this deployment")
		}
		return nil
	}

	destLocale, _ := d.Payload["dest_locale"].(string)
	var notifCtx *notification.NotificationContext
	if w.contextResolver != nil {
		notifCtx = w.contextResolver.ResolveAndRefreshContext(deliveryCtx, d.OrganizationID, d.EventType, d.Payload, w.appBaseURL, destLocale)
	} else {
		notifCtx = notification.ParseContextFromPayload(d.EventType, d.Payload, w.appBaseURL, destLocale)
	}

	rendered := email.RenderNotificationEmailFromContext(notifCtx, w.appBaseURL)

	sendErr := w.emailSender.SendEmail(deliveryCtx, d.Recipient, rendered.Subject, rendered.HTMLBody, rendered.TextBody)
	if sendErr == nil {
		if w.repo != nil {
			return w.repo.MarkDeliverySent(ctx, d.ID, w.workerID)
		}
		return nil
	}

	permanent := isPermanentEmailError(sendErr)
	backoff := calculateBackoff(d.RetryCount)
	if w.repo != nil {
		_ = w.repo.MarkDeliveryFailed(ctx, d.ID, w.workerID, sendErr.Error(), backoff, permanent)
	}
	return sendErr
}

func calculateBackoff(retryCount int) time.Duration {
	base := 2.0
	shift := 1 << uint(retryCount)
	secs := base * float64(shift) // 2s, 4s, 8s, 16s, 32s
	if secs > 300 {
		secs = 300
	}
	jitter := float64(rand.Intn(1000)) / 1000.0
	return time.Duration((secs + jitter) * float64(time.Second))
}

func isRateLimit(err error, target **telegram.RateLimitError) bool {
	if rl, ok := err.(*telegram.RateLimitError); ok {
		*target = rl
		return true
	}
	return false
}

func isPermanentTelegram(err error, target **telegram.PermanentError) bool {
	if pe, ok := err.(*telegram.PermanentError); ok {
		*target = pe
		return true
	}
	return false
}

func isPermanentEmailError(err error) bool {
	if err == nil {
		return false
	}

	// 1. Context cancellation / deadlines are transient timeouts
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// 2. Network timeouts or transient connection issues are NOT permanent
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "broken pipe") ||
		strings.Contains(errMsg, "network is unreachable") ||
		strings.Contains(errMsg, "i/o timeout") ||
		strings.Contains(errMsg, "deadline exceeded") ||
		strings.Contains(errMsg, "dial timeout") ||
		strings.Contains(errMsg, "unexpected eof") {
		return false
	}

	// 3. Inspect typed textproto.Error SMTP codes
	var tpErr *textproto.Error
	if errors.As(err, &tpErr) {
		if tpErr.Code >= 400 && tpErr.Code < 500 {
			// Transient SMTP failure (e.g. 421, 450, 451, 452, 454)
			return false
		}
		if tpErr.Code >= 500 && tpErr.Code < 600 {
			// Permanent SMTP failure (e.g. 501, 502, 503, 504, 535, 550, 554)
			return true
		}
	}

	// 4. Inspect status code patterns in error string (for wrapped/stringified errors)
	if smtpTransientRegex.MatchString(errMsg) {
		// Matches 4xx transient codes (e.g., 421, 450, 451, 452)
		return false
	}
	if smtpPermanentRegex.MatchString(errMsg) {
		// Matches 5xx permanent codes (e.g., 501, 502, 503, 504, 535, 550, 554)
		return true
	}

	// 5. Inspect well-known permanent semantic phrases
	if strings.Contains(errMsg, "mailbox unavailable") ||
		strings.Contains(errMsg, "no such user") ||
		strings.Contains(errMsg, "user unknown") ||
		strings.Contains(errMsg, "invalid recipient") ||
		strings.Contains(errMsg, "invalid address") ||
		strings.Contains(errMsg, "authentication failed") ||
		strings.Contains(errMsg, "auth failed") ||
		strings.Contains(errMsg, "credentials rejected") ||
		strings.Contains(errMsg, "recipient rejected") {
		return true
	}

	// Default: unknown errors treated as transient/retriable
	return false
}


