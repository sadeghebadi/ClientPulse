package email

import (
	"context"
	"strings"
	"testing"
	"time"

	"clientpulse/services/api/internal/notification"
)

func TestRenderNotificationEmail_Templates(t *testing.T) {
	events := []string{
		"incident.opened",
		"incident.acknowledged",
		"incident.resolved",
		"heartbeat.missed",
		"heartbeat.recovered",
		"report.ready",
		"test.notification",
	}

	payload := map[string]interface{}{
		"monitor_name":     "Production API",
		"incident_id":      "c19bb8d2-7901-443b-85ef-b1fba5432109",
		"failure_category": "http_5xx",
		"heartbeat_name":   "Database Backup Cron",
		"report_title":     "October 2026 Care Report",
		"report_id":        "r99bb8d2-7901-443b-85ef-b1fba5432109",
	}

	for _, ev := range events {
		// English
		en := RenderNotificationEmail(ev, "en", payload, "https://app.clientpulse.io")
		if en.Subject == "" {
			t.Errorf("expected non-empty English subject for event %s", ev)
		}
		if !strings.Contains(en.HTMLBody, "dir=\"ltr\"") {
			t.Errorf("expected English HTML to be LTR for %s", ev)
		}
		if en.TextBody == "" {
			t.Errorf("expected non-empty plain text for event %s", ev)
		}

		// Persian
		fa := RenderNotificationEmail(ev, "fa", payload, "https://app.clientpulse.io")
		if fa.Subject == "" {
			t.Errorf("expected non-empty Persian subject for event %s", ev)
		}
		if !strings.Contains(fa.HTMLBody, "dir=\"rtl\"") {
			t.Errorf("expected Persian HTML to be RTL for %s", ev)
		}
		if fa.TextBody == "" {
			t.Errorf("expected non-empty plain text for event %s", ev)
		}
	}
}

func TestSMTPClient_MailpitDelivery(t *testing.T) {
	client := NewClient(SMTPConfig{
		Host:      "localhost",
		Port:      1025,
		FromEmail: "alerts@clientpulse.local",
		FromName:  "ClientPulse Alerts",
		TLSMode:   "none",
	})

	err := client.SendEmail(
		context.Background(),
		"test-user@clientpulse.local",
		"Test Integration Alert",
		"<p>This is a test notification</p>",
		"This is a test notification",
	)
	if err != nil {
		t.Fatalf("Mailpit SMTP delivery failed: %v", err)
	}
}


func TestRenderNotificationEmail_RichContext(t *testing.T) {
	now := time.Now().UTC()
	firstFail := now.Add(-12 * time.Minute)
	resolved := now.Add(-3 * time.Minute)

	notifCtx := &notification.NotificationContext{
		EventType:            "incident.opened",
		Locale:               "en",
		IncidentID:           "test-inc-1",
		IncidentRef:          "#INC-TEST1",
		ClientName:           "Umbrella Corp",
		AssetName:            "Primary Web App",
		MonitorName:          "Frontend Health",
		SanitizedTarget:      "https://umbrella.com",
		FirstFailureAt:       &firstFail,
		ConfirmedAt:          &firstFail,
		DurationSeconds:      720,
		CheckIntervalSeconds: 60,
		FailureCategory:      "timeout",
		FailureReason:        "Connection timed out after 10s",
		HTTPStatusCode:       504,
		ViewURL:              "https://app.clientpulse.io/incidents/test-inc-1",
	}

	// 1. English incident.opened
	en := RenderNotificationEmailFromContext(notifCtx, "https://app.clientpulse.io")
	if !strings.Contains(en.Subject, "Umbrella Corp") {
		t.Errorf("expected client name in subject, got: %s", en.Subject)
	}
	if !strings.Contains(en.HTMLBody, "Primary Web App") {
		t.Errorf("expected asset name in HTML body")
	}
	if !strings.Contains(en.HTMLBody, "504") {
		t.Errorf("expected HTTP status 504 in HTML body")
	}
	if !strings.Contains(en.TextBody, "Umbrella Corp") {
		t.Errorf("expected client name in text body")
	}

	// 2. Persian incident.opened
	notifCtx.Locale = "fa"
	fa := RenderNotificationEmailFromContext(notifCtx, "https://app.clientpulse.io")
	if !strings.Contains(fa.Subject, "Umbrella Corp") {
		t.Errorf("expected client name in Persian subject, got: %s", fa.Subject)
	}
	if !strings.Contains(fa.HTMLBody, "dir=\"rtl\"") {
		t.Errorf("expected dir=rtl in Persian HTML")
	}
	if !strings.Contains(fa.HTMLBody, "lang=\"fa\"") {
		t.Errorf("expected lang=fa in Persian HTML")
	}

	// 3. IsAlreadyResolved banner
	notifCtx.IsAlreadyResolved = true
	notifCtx.ResolvedAt = &resolved
	resolvedEmail := RenderNotificationEmailFromContext(notifCtx, "https://app.clientpulse.io")
	if !strings.Contains(resolvedEmail.HTMLBody, "برطرف شد") {
		t.Errorf("expected resolved banner in Persian HTML")
	}
}

func TestRenderEmailVerification_BothLocales(t *testing.T) {
	en := RenderEmailVerification("token-xyz-123", "https://app.clientpulse.io", "en")
	if !strings.Contains(en.Subject, "Verify") {
		t.Errorf("expected 'Verify' in English subject, got: %s", en.Subject)
	}
	if !strings.Contains(en.HTMLBody, "token-xyz-123") {
		t.Errorf("expected token in English verification link")
	}
	if !strings.Contains(en.HTMLBody, "dir=\"ltr\"") {
		t.Errorf("expected LTR in English verification email")
	}

	fa := RenderEmailVerification("token-xyz-123", "https://app.clientpulse.io", "fa")
	if !strings.Contains(fa.Subject, "تایید") {
		t.Errorf("expected 'تایید' in Persian subject, got: %s", fa.Subject)
	}
	if !strings.Contains(fa.HTMLBody, "token-xyz-123") {
		t.Errorf("expected token in Persian verification link")
	}
	if !strings.Contains(fa.HTMLBody, "dir=\"rtl\"") {
		t.Errorf("expected RTL in Persian verification email")
	}
}
