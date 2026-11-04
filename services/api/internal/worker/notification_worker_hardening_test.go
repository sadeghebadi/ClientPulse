package worker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"testing"
	"time"

	"clientpulse/services/api/internal/notification/telegram"
	"clientpulse/services/api/internal/repository"
)

type customNetTimeoutError struct{}

var _ net.Error = (*customNetTimeoutError)(nil)

func (e *customNetTimeoutError) Error() string   { return "custom network timeout" }
func (e *customNetTimeoutError) Timeout() bool   { return true }
func (e *customNetTimeoutError) Temporary() bool { return true }

func TestEmailErrorClassification(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		permanent bool
	}{
		// Permanent 5xx errors
		{
			name:      "535 Authentication failure string",
			err:       errors.New("535 5.7.8 Authentication credentials invalid"),
			permanent: true,
		},
		{
			name:      "535 Authentication failure textproto.Error",
			err:       &textproto.Error{Code: 535, Msg: "Authentication credentials invalid"},
			permanent: true,
		},
		{
			name:      "535 Wrapped textproto.Error",
			err:       fmt.Errorf("SMTP auth failed: %w", &textproto.Error{Code: 535, Msg: "Auth invalid"}),
			permanent: true,
		},
		{
			name:      "501 Syntax error in parameters",
			err:       &textproto.Error{Code: 501, Msg: "Syntax error in parameters or arguments"},
			permanent: true,
		},
		{
			name:      "502 Command not implemented",
			err:       errors.New("502 Command not implemented"),
			permanent: true,
		},
		{
			name:      "503 Bad sequence of commands",
			err:       &textproto.Error{Code: 503, Msg: "Bad sequence of commands"},
			permanent: true,
		},
		{
			name:      "504 Command parameter not implemented",
			err:       errors.New("504 Command parameter not implemented"),
			permanent: true,
		},
		{
			name:      "550 Mailbox unavailable",
			err:       &textproto.Error{Code: 550, Msg: "Requested action not taken: mailbox unavailable"},
			permanent: true,
		},
		{
			name:      "554 Transaction failed",
			err:       errors.New("554 Transaction failed"),
			permanent: true,
		},
		{
			name:      "Phrase user unknown",
			err:       errors.New("recipient address: user unknown"),
			permanent: true,
		},
		{
			name:      "Phrase no such user",
			err:       errors.New("550 No such user here"),
			permanent: true,
		},
		{
			name:      "Phrase invalid recipient",
			err:       errors.New("server rejected: invalid recipient address"),
			permanent: true,
		},
		{
			name:      "Phrase credentials rejected",
			err:       errors.New("SMTP credentials rejected by server"),
			permanent: true,
		},

		// Transient 4xx errors
		{
			name:      "421 Service not available textproto.Error",
			err:       &textproto.Error{Code: 421, Msg: "Service not available, closing transmission channel"},
			permanent: false,
		},
		{
			name:      "421 Service not available string",
			err:       errors.New("421 Service not available"),
			permanent: false,
		},
		{
			name:      "450 Mailbox busy textproto.Error",
			err:       &textproto.Error{Code: 450, Msg: "Requested mail action not taken: mailbox busy"},
			permanent: false,
		},
		{
			name:      "451 Local error in processing textproto.Error",
			err:       &textproto.Error{Code: 451, Msg: "Requested action aborted: local error in processing"},
			permanent: false,
		},
		{
			name:      "452 Insufficient storage string",
			err:       errors.New("452 4.2.2 Insufficient system storage"),
			permanent: false,
		},
		{
			name:      "454 TLS not available string",
			err:       errors.New("454 TLS not available due to temporary reason"),
			permanent: false,
		},
		{
			name:      "Wrapped 450 transient error",
			err:       fmt.Errorf("send failed: %w", &textproto.Error{Code: 450, Msg: "Mailbox busy"}),
			permanent: false,
		},

		// Timeouts and network errors (transient)
		{
			name:      "context.DeadlineExceeded",
			err:       context.DeadlineExceeded,
			permanent: false,
		},
		{
			name:      "Wrapped context.DeadlineExceeded",
			err:       fmt.Errorf("SMTP write data aborted: %w", context.DeadlineExceeded),
			permanent: false,
		},
		{
			name:      "context.Canceled",
			err:       context.Canceled,
			permanent: false,
		},
		{
			name:      "net.Error timeout",
			err:       &customNetTimeoutError{},
			permanent: false,
		},
		{
			name:      "Connection refused string",
			err:       errors.New("dial tcp 127.0.0.1:25: connect: connection refused"),
			permanent: false,
		},
		{
			name:      "Connection reset by peer string",
			err:       errors.New("read tcp 10.0.0.1:587: read: connection reset by peer"),
			permanent: false,
		},
		{
			name:      "i/o timeout string",
			err:       errors.New("read tcp 127.0.0.1:1025: i/o timeout"),
			permanent: false,
		},
		{
			name:      "broken pipe string",
			err:       errors.New("write tcp 127.0.0.1:1025: write: broken pipe"),
			permanent: false,
		},
		{
			name:      "unexpected EOF",
			err:       errors.New("unexpected EOF"),
			permanent: false,
		},

		// Unknown / generic errors (transient)
		{
			name:      "Unknown error string",
			err:       errors.New("internal system glitch occurred"),
			permanent: false,
		},
		{
			name:      "Nil error",
			err:       nil,
			permanent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPermanentEmailError(tt.err)
			if got != tt.permanent {
				t.Errorf("isPermanentEmailError(%v) = %v; want %v", tt.err, got, tt.permanent)
			}
		})
	}
}

// Blocking mock email sender for per-delivery timeout testing
type blockingEmailSender struct {
	blockedCtxDone chan struct{}
}

func (m *blockingEmailSender) SendEmail(ctx context.Context, to, subject, htmlBody, textBody string) error {
	select {
	case <-ctx.Done():
		close(m.blockedCtxDone)
		return ctx.Err()
	case <-time.After(10 * time.Second):
		return nil
	}
}

// Blocking mock telegram sender for per-delivery timeout testing
type blockingTelegramSender struct {
	blockedCtxDone chan struct{}
}

func (m *blockingTelegramSender) SendMessage(ctx context.Context, chatID int64, text, parseMode string) error {
	select {
	case <-ctx.Done():
		close(m.blockedCtxDone)
		return ctx.Err()
	case <-time.After(10 * time.Second):
		return nil
	}
}

func (m *blockingTelegramSender) GetMe(ctx context.Context) (*telegram.TelegramUser, error) {
	return &telegram.TelegramUser{ID: 12345, Username: "testbot"}, nil
}

// TestNotificationWorker_DeliveryTimeout_Email verifies that a blocked email provider
// is cancelled at the per-delivery timeout deadline and cannot hold the worker indefinitely.
func TestNotificationWorker_DeliveryTimeout_Email(t *testing.T) {
	blockSignal := make(chan struct{})
	emailSender := &blockingEmailSender{blockedCtxDone: blockSignal}

	worker := NewNotificationWorker(
		nil,
		nil,
		emailSender,
		"https://app.clientpulse.io",
		"test-worker-timeout",
		1,
		nil,
	).WithProviders("disabled", "smtp").WithDeliveryTimeout(60 * time.Millisecond)

	deliv := &repository.NotificationDelivery{
		ID:              "deliv-email-timeout-1",
		OrganizationID:  "org-1",
		EventType:       "incident.opened",
		DestinationType: "email",
		Recipient:       "alice@example.com",
		Payload: map[string]interface{}{
			"monitor_name": "API Gateway",
		},
	}

	start := time.Now()
	err := worker.Deliver(context.Background(), deliv)
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
	if elapsed > 1*time.Second {
		t.Fatalf("expected blocked email to abort in ~60ms, took %v", elapsed)
	}

	select {
	case <-blockSignal:
		// Succeeded: cancellation propagated
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected emailSender to detect cancellation")
	}
}

// TestNotificationWorker_DeliveryTimeout_Telegram verifies that a blocked telegram provider
// is cancelled at the per-delivery timeout deadline.
func TestNotificationWorker_DeliveryTimeout_Telegram(t *testing.T) {
	blockSignal := make(chan struct{})
	tgSender := &blockingTelegramSender{blockedCtxDone: blockSignal}

	worker := NewNotificationWorker(
		nil,
		tgSender,
		nil,
		"https://app.clientpulse.io",
		"test-worker-timeout",
		1,
		nil,
	).WithProviders("polling", "none").WithDeliveryTimeout(60 * time.Millisecond)

	deliv := &repository.NotificationDelivery{
		ID:              "deliv-tg-timeout-1",
		OrganizationID:  "org-1",
		EventType:       "incident.opened",
		DestinationType: "telegram_private",
		Recipient:       "12345678",
		Payload: map[string]interface{}{
			"monitor_name": "API Gateway",
		},
	}

	start := time.Now()
	err := worker.Deliver(context.Background(), deliv)
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
	if elapsed > 1*time.Second {
		t.Fatalf("expected blocked telegram to abort in ~60ms, took %v", elapsed)
	}

	select {
	case <-blockSignal:
		// Succeeded
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected tgSender to detect cancellation")
	}
}
