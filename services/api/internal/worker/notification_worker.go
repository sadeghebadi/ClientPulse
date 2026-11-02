package worker

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"clientpulse/services/api/internal/notification/email"
	"clientpulse/services/api/internal/notification/telegram"
	"clientpulse/services/api/internal/repository"
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
	repo           *repository.NotificationDestRepository
	telegramSender telegram.TelegramSender
	emailSender    email.EmailSender
	appBaseURL     string
	workerID       string
	logger         *slog.Logger
	concurrency    int
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
		repo:           repo,
		telegramSender: telegramSender,
		emailSender:    emailSender,
		appBaseURL:     strings.TrimRight(appBaseURL, "/"),
		workerID:       workerID,
		concurrency:    concurrency,
		logger:         logger,
	}
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
	switch d.DestinationType {
	case "telegram_private", "telegram_group":
		return w.deliverTelegram(ctx, d)
	case "email":
		return w.deliverEmail(ctx, d)
	default:
		return w.repo.MarkDeliveryFailed(ctx, d.ID, w.workerID, fmt.Sprintf("unsupported destination type: %s", d.DestinationType), 0, true)
	}
}

func (w *NotificationWorker) deliverTelegram(ctx context.Context, d *repository.NotificationDelivery) error {
	if w.telegramSender == nil {
		err := fmt.Errorf("telegram sender adapter is not configured")
		_ = w.repo.MarkDeliveryFailed(ctx, d.ID, w.workerID, err.Error(), 1*time.Minute, false)
		return err
	}

	chatID, err := strconv.ParseInt(d.Recipient, 10, 64)
	if err != nil {
		_ = w.repo.MarkDeliveryFailed(ctx, d.ID, w.workerID, fmt.Sprintf("invalid telegram chat ID '%s': %v", d.Recipient, err), 0, true)
		return err
	}

	text := formatTelegramMessage(d.EventType, d.Payload, w.appBaseURL)

	sendErr := w.telegramSender.SendMessage(ctx, chatID, text, "HTML")
	if sendErr == nil {
		return w.repo.MarkDeliverySent(ctx, d.ID, w.workerID)
	}

	// Classify error
	var rateErr *telegram.RateLimitError
	var permErr *telegram.PermanentError

	if isRateLimit(sendErr, &rateErr) {
		jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
		backoff := rateErr.RetryAfter + jitter
		_ = w.repo.MarkDeliveryFailed(ctx, d.ID, w.workerID, rateErr.Error(), backoff, false)
		return sendErr
	}

	if isPermanentTelegram(sendErr, &permErr) {
		_ = w.repo.MarkDeliveryFailed(ctx, d.ID, w.workerID, permErr.Error(), 0, true)
		return sendErr
	}

	// Temporary network / internal error: exponential backoff + jitter
	backoff := calculateBackoff(d.RetryCount)
	_ = w.repo.MarkDeliveryFailed(ctx, d.ID, w.workerID, sendErr.Error(), backoff, false)
	return sendErr
}

func (w *NotificationWorker) deliverEmail(ctx context.Context, d *repository.NotificationDelivery) error {
	if w.emailSender == nil {
		err := fmt.Errorf("email sender adapter is not configured")
		_ = w.repo.MarkDeliveryFailed(ctx, d.ID, w.workerID, err.Error(), 1*time.Minute, false)
		return err
	}

	locale := "en"
	if loc, ok := d.Payload["locale"].(string); ok && loc != "" {
		locale = loc
	}

	rendered := email.RenderNotificationEmail(d.EventType, locale, d.Payload, w.appBaseURL)

	sendErr := w.emailSender.SendEmail(ctx, d.Recipient, rendered.Subject, rendered.HTMLBody, rendered.TextBody)
	if sendErr == nil {
		return w.repo.MarkDeliverySent(ctx, d.ID, w.workerID)
	}

	permanent := isPermanentEmailError(sendErr)
	backoff := calculateBackoff(d.RetryCount)
	_ = w.repo.MarkDeliveryFailed(ctx, d.ID, w.workerID, sendErr.Error(), backoff, permanent)
	return sendErr
}

func formatTelegramMessage(eventType string, p map[string]interface{}, appBaseURL string) string {
	if appBaseURL == "" {
		appBaseURL = "http://localhost:3000"
	}
	appBaseURL = strings.TrimRight(appBaseURL, "/")

	switch eventType {
	case "incident.opened":
		name := html.EscapeString(getString(p, "monitor_name", getString(p, "summary", "Monitor")))
		cat := html.EscapeString(getString(p, "failure_category", "service_error"))
		incID := getString(p, "incident_id", "")
		url := fmt.Sprintf("%s/incidents/%s", appBaseURL, incID)
		return fmt.Sprintf("🚨 <b>INCIDENT OPENED: %s is DOWN</b>\n\n<b>Category:</b> <code>%s</code>\n<b>Confirmed At:</b> %s\n\n👉 <a href=\"%s\">View Incident in ClientPulse</a>", name, cat, time.Now().UTC().Format("15:04:05 UTC"), url)

	case "incident.acknowledged":
		incID := getString(p, "incident_id", "")
		url := fmt.Sprintf("%s/incidents/%s", appBaseURL, incID)
		return fmt.Sprintf("👁️ <b>INCIDENT ACKNOWLEDGED</b>\n\nIncident <code>%s</code> has been acknowledged by on-call team.\n\n👉 <a href=\"%s\">View Incident Details</a>", incID[:8], url)

	case "incident.resolved":
		name := html.EscapeString(getString(p, "monitor_name", "Service"))
		incID := getString(p, "incident_id", "")
		url := fmt.Sprintf("%s/incidents/%s", appBaseURL, incID)
		return fmt.Sprintf("✅ <b>INCIDENT RESOLVED: %s is back UP</b>\n\nService health checks are passing normally.\n\n👉 <a href=\"%s\">View Resolution Report</a>", name, url)

	case "heartbeat.missed":
		name := html.EscapeString(getString(p, "heartbeat_name", "Heartbeat"))
		url := fmt.Sprintf("%s/heartbeats", appBaseURL)
		return fmt.Sprintf("⚠️ <b>HEARTBEAT MISSED: %s</b>\n\nExpected heartbeat ping was not received before deadline.\n\n👉 <a href=\"%s\">Check Heartbeats</a>", name, url)

	case "heartbeat.recovered":
		name := html.EscapeString(getString(p, "heartbeat_name", "Heartbeat"))
		url := fmt.Sprintf("%s/heartbeats", appBaseURL)
		return fmt.Sprintf("✅ <b>HEARTBEAT RECOVERED: %s</b>\n\nPing received successfully, status is now healthy.\n\n👉 <a href=\"%s\">Check Heartbeats</a>", name, url)

	case "report.ready":
		title := html.EscapeString(getString(p, "report_title", "Maintenance Report"))
		repID := getString(p, "report_id", "")
		url := fmt.Sprintf("%s/reports/%s", appBaseURL, repID)
		return fmt.Sprintf("📊 <b>REPORT READY: %s</b>\n\nYour care and maintenance report is ready for download.\n\n👉 <a href=\"%s\">View Report</a>", title, url)

	default:
		return "🔔 <b>ClientPulse Notification Test</b>\n\nThis is a test notification confirming that your Telegram channel is active and receiving alerts."
	}
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
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "invalid") ||
		strings.Contains(errMsg, "mailbox unavailable") ||
		strings.Contains(errMsg, "no such user") ||
		strings.Contains(errMsg, "user unknown") ||
		strings.Contains(errMsg, "550") ||
		strings.Contains(errMsg, "554") {
		return true
	}
	return false
}

func getString(m map[string]interface{}, key, def string) string {
	if m == nil {
		return def
	}
	if v, ok := m[key]; ok && v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return def
}
