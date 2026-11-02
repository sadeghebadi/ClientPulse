package email

import (
	"context"
	"strings"
	"testing"
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

