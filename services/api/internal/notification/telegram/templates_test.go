package telegram

import (
	"strings"
	"testing"
	"time"

	"clientpulse/services/api/internal/notification"
)

func TestRenderTelegramMessage_AllEventTypes_BothLocales(t *testing.T) {
	now := time.Now().UTC()
	firstFail := now.Add(-10 * time.Minute)
	resolved := now.Add(-2 * time.Minute)
	ack := now.Add(-6 * time.Minute)

	baseCtx := &notification.NotificationContext{
		IncidentID:           "test-incident-uuid-123456",
		IncidentRef:          "#INC-TEST1234",
		OrganizationID:       "org-uuid-1",
		OrgName:              "Acme Digital",
		ClientID:             "client-uuid-1",
		ClientName:           "Globex Corporation",
		AssetID:              "asset-uuid-1",
		AssetName:            "E-Commerce Store",
		AssetType:            "website",
		MonitorID:            "mon-uuid-1",
		MonitorName:          "Checkout API Check",
		MonitorType:          "HTTP",
		TargetURL:            "https://store.globex.com/api/checkout",
		SanitizedTarget:      "https://store.globex.com/api/checkout",
		FirstFailureAt:       &firstFail,
		ConfirmedAt:          &firstFail,
		ResolvedAt:           &resolved,
		AcknowledgedAt:       &ack,
		AcknowledgedByName:   "Sara Connor",
		DurationSeconds:      480,
		TimeToAcknowledgeSeconds: 240,
		CheckIntervalSeconds: 60,
		HTTPStatusCode:       502,
		FailureCategory:      "http_5xx",
		FailureReason:        "502 Bad Gateway from upstream",
		RecoveryType:         "scheduled_check_pass",
		RecoveryStatus:       "HTTP 200 OK (passed in 142ms)",
		HeartbeatID:          "hb-uuid-1",
		HeartbeatName:        "Midnight DB Backup",
		LastPingAt:           &firstFail,
		ExpectedIntervalSeconds: 86400,
		GracePeriodSeconds:      3600,
		OverdueSeconds:          1200,
		ReportID:             "rep-uuid-1",
		ReportTitle:          "Monthly Performance Report",
		PeriodStart:          "2026-10-01",
		PeriodEnd:            "2026-10-31",
		ViewURL:              "https://app.clientpulse.io/incidents/test",
		Timezone:             "UTC",
	}

	eventTypes := []string{
		"incident.opened",
		"incident.resolved",
		"incident.acknowledged",
		"heartbeat.missed",
		"heartbeat.recovered",
		"report.ready",
		"test.notification",
	}

	for _, ev := range eventTypes {
		t.Run(ev+"_EN", func(t *testing.T) {
			ctx := *baseCtx
			ctx.EventType = ev
			ctx.Locale = "en"

			msg := RenderTelegramMessage(&ctx)
			if msg == "" {
				t.Fatalf("rendered empty message for %s (EN)", ev)
			}
			if len(msg) > 4096 {
				t.Errorf("rendered message exceeds 4096 bytes (%d bytes) for %s (EN)", len(msg), ev)
			}
			// Verify client name is present (except test notification)
			if ev != "test.notification" && !strings.Contains(msg, "Globex Corporation") {
				t.Errorf("expected client name in message for %s (EN)", ev)
			}
		})

		t.Run(ev+"_FA", func(t *testing.T) {
			ctx := *baseCtx
			ctx.EventType = ev
			ctx.Locale = "fa"

			msg := RenderTelegramMessage(&ctx)
			if msg == "" {
				t.Fatalf("rendered empty message for %s (FA)", ev)
			}
			if len(msg) > 4096 {
				t.Errorf("rendered message exceeds 4096 bytes (%d bytes) for %s (FA)", len(msg), ev)
			}
			if ev != "test.notification" && !strings.Contains(msg, "Globex Corporation") {
				t.Errorf("expected client name in message for %s (FA)", ev)
			}
		})
	}
}

func TestRenderTelegramMessage_HTMLEscaping(t *testing.T) {
	ctx := &notification.NotificationContext{
		EventType:       "incident.opened",
		Locale:          "en",
		ClientName:      "<script>alert('pwned')</script>",
		AssetName:       "Bob & Alice's Store",
		MonitorName:     "API <Health> & Status",
		SanitizedTarget: "https://example.com/api?a=1&b=2",
		FailureReason:   "Error: <nil> pointer & crash",
		ViewURL:         "https://app.clientpulse.io/incidents/1",
	}

	msg := RenderTelegramMessage(ctx)
	if strings.Contains(msg, "<script>") {
		t.Errorf("raw <script> tag was not escaped in Telegram message: %s", msg)
	}
	if !strings.Contains(msg, "&lt;script&gt;") {
		t.Errorf("expected escaped &lt;script&gt; in Telegram message: %s", msg)
	}
	if !strings.Contains(msg, "Bob &amp; Alice&#39;s Store") && !strings.Contains(msg, "Bob &amp; Alice's Store") {
		t.Errorf("expected escaped ampersand in Telegram message: %s", msg)
	}
}

func TestRenderTelegramMessage_LengthSafeguard(t *testing.T) {
	hugeString := strings.Repeat("A very long error message that keeps going and going. ", 200)
	ctx := &notification.NotificationContext{
		EventType:     "incident.opened",
		Locale:        "en",
		ClientName:    "Big Enterprise",
		AssetName:     "Core Cluster",
		MonitorName:   "Heavy Ingestion Node",
		FailureReason: hugeString,
		ViewURL:       "https://app.clientpulse.io/incidents/1",
	}

	msg := RenderTelegramMessage(ctx)
	if len(msg) > 4096 {
		t.Errorf("message length %d exceeded 4096 bytes safeguard", len(msg))
	}
}

func TestRenderTelegramMessage_IsAlreadyResolved(t *testing.T) {
	ctx := &notification.NotificationContext{
		EventType:         "incident.opened",
		Locale:            "en",
		ClientName:        "Fast Recovery Corp",
		AssetName:         "Auth Service",
		MonitorName:       "Login Monitor",
		DurationSeconds:   120,
		IsAlreadyResolved: true,
		ViewURL:           "https://app.clientpulse.io/incidents/1",
	}

	msg := RenderTelegramMessage(ctx)
	if !strings.Contains(msg, "RESOLVED") {
		t.Errorf("expected message to indicate incident was already resolved: %s", msg)
	}
}

func TestRenderTelegramMessage_MissingFieldsFallback(t *testing.T) {
	// Completely empty context should not panic and produce reasonable fallback
	ctx := &notification.NotificationContext{
		EventType: "incident.opened",
		Locale:    "en",
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RenderTelegramMessage panicked on empty context: %v", r)
		}
	}()

	msg := RenderTelegramMessage(ctx)
	if msg == "" {
		t.Errorf("expected non-empty message for empty context")
	}
}
