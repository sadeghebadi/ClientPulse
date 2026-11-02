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
		IncidentID:               "test-incident-uuid-123456",
		IncidentRef:              "#INC-TEST1234",
		OrganizationID:           "org-uuid-1",
		OrgName:                  "Acme Digital",
		ClientID:                 "client-uuid-1",
		ClientName:               "Globex Corporation",
		AssetID:                  "asset-uuid-1",
		AssetName:                "E-Commerce Store",
		AssetType:                "website",
		MonitorID:                "mon-uuid-1",
		MonitorName:              "Checkout API Check",
		MonitorType:              "HTTP",
		TargetURL:                "https://store.globex.com/api/checkout",
		SanitizedTarget:          "https://store.globex.com/api/checkout",
		FirstFailureAt:           &firstFail,
		ConfirmedAt:              &firstFail,
		ResolvedAt:               &resolved,
		AcknowledgedAt:           &ack,
		AcknowledgedByName:       "Sara Connor",
		DurationSeconds:          480,
		TimeToAcknowledgeSeconds: 240,
		CheckIntervalSeconds:     60,
		HTTPStatusCode:           502,
		FailureCategory:          "http_5xx",
		FailureReason:            "502 Bad Gateway from upstream",
		RecoveryType:             "scheduled_check_pass",
		RecoveryStatus:           "HTTP 200 OK (passed in 142ms)",
		HeartbeatID:              "hb-uuid-1",
		HeartbeatName:            "Midnight DB Backup",
		LastPingAt:               &firstFail,
		ExpectedIntervalSeconds:  86400,
		GracePeriodSeconds:       3600,
		OverdueSeconds:           1200,
		ReportID:                 "rep-uuid-1",
		ReportTitle:              "Monthly Performance Report",
		PeriodStart:              "2026-10-01",
		PeriodEnd:                "2026-10-31",
		ViewURL:                  "https://mindu.ir/incidents/test-incident-uuid-123456",
		Timezone:                 "UTC",
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
			if TelegramMessageLength(msg) > TelegramMaxMessageChars {
				t.Errorf("rendered message exceeds Telegram 4096 character limit (%d UTF-16 units) for %s (EN)", TelegramMessageLength(msg), ev)
			}
			if ev != "test.notification" && !strings.Contains(msg, "Globex Corporation") {
				t.Errorf("expected client name in message for %s (EN)", ev)
			}
			assertValidHTMLTags(t, msg)
		})

		t.Run(ev+"_FA", func(t *testing.T) {
			ctx := *baseCtx
			ctx.EventType = ev
			ctx.Locale = "fa"

			msg := RenderTelegramMessage(&ctx)
			if msg == "" {
				t.Fatalf("rendered empty message for %s (FA)", ev)
			}
			if TelegramMessageLength(msg) > TelegramMaxMessageChars {
				t.Errorf("rendered message exceeds Telegram 4096 character limit (%d UTF-16 units) for %s (FA)", TelegramMessageLength(msg), ev)
			}
			if ev != "test.notification" && !strings.Contains(msg, "Globex Corporation") {
				t.Errorf("expected client name in message for %s (FA)", ev)
			}
			assertValidHTMLTags(t, msg)
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
		ViewURL:         "https://mindu.ir/incidents/1",
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
	assertValidHTMLTags(t, msg)
}

func TestRenderTelegramMessage_LongPersianText(t *testing.T) {
	persianParagraph := "این یک پیام طولانی به زبان فارسی است که شامل کلمات و جملات مکرر جهت تست پایداری و صحت انکودینگ کاراکترهای یونیکد در پیام‌رسان تلگرام می‌باشد. "
	hugePersian := strings.Repeat(persianParagraph, 50) // ~10,000 characters

	ctx := &notification.NotificationContext{
		EventType:       "incident.opened",
		Locale:          "fa",
		ClientName:      "شرکت توسعه نرم‌افزار آرمان داده پرداز پویا",
		AssetName:       "سامانه یکپارچه تجارت الکترونیک",
		MonitorName:     "سرویس درگاه پرداخت بانکی",
		FailureReason:   hugePersian,
		SanitizedTarget: "https://pay.example.ir/gateway",
		ViewURL:         "https://mindu.ir/incidents/fa-test",
	}

	msg := RenderTelegramMessage(ctx)
	length := TelegramMessageLength(msg)
	if length > TelegramMaxMessageChars {
		t.Errorf("long Persian message length (%d) exceeded Telegram 4096 character limit", length)
	}
	if !strings.Contains(msg, "شرکت توسعه نرم‌افزار آرمان داده پرداز پویا") {
		t.Errorf("expected Persian client name to be preserved")
	}
	assertValidHTMLTags(t, msg)
}

func TestRenderTelegramMessage_LongEnglishText(t *testing.T) {
	hugeEnglish := strings.Repeat("Critical database deadlock detected on primary replica table user_sessions with PID 89412 waiting on lock. ", 100)
	ctx := &notification.NotificationContext{
		EventType:     "incident.opened",
		Locale:        "en",
		ClientName:    "Big Enterprise Systems International",
		AssetName:     "Global Cloud Infrastructure",
		MonitorName:   "Primary Replica Node",
		FailureReason: hugeEnglish,
		ViewURL:       "https://mindu.ir/incidents/en-test",
	}

	msg := RenderTelegramMessage(ctx)
	length := TelegramMessageLength(msg)
	if length > TelegramMaxMessageChars {
		t.Errorf("long English message length (%d) exceeded Telegram 4096 character limit", length)
	}
	assertValidHTMLTags(t, msg)
}

func TestRenderTelegramMessage_EmojiAndNonBMPUnicode(t *testing.T) {
	// Emojis that require surrogate pairs in UTF-16 (> U+FFFF)
	emojiString := "🚨🔥⚠️🛠️⚡️💥👩‍💻👨‍💻🚀🔒🌐" + strings.Repeat("🎉🎈✨", 200)

	ctx := &notification.NotificationContext{
		EventType:     "incident.opened",
		Locale:        "en",
		ClientName:    "Emoji Client " + emojiString,
		AssetName:     "Asset 🚀",
		MonitorName:   "Monitor ⚡️",
		FailureReason: "Error occurred with emoji 🔥 " + emojiString,
		ViewURL:       "https://mindu.ir/incidents/emoji",
	}

	msg := RenderTelegramMessage(ctx)
	length := TelegramMessageLength(msg)
	if length > TelegramMaxMessageChars {
		t.Errorf("emoji message length (%d UTF-16 units) exceeded Telegram 4096 character limit", length)
	}
	assertValidHTMLTags(t, msg)
}

func TestRenderTelegramMessage_ExtremeFieldLengths(t *testing.T) {
	giantName := strings.Repeat("UltraLongEntityName-", 300) // 6000 runes
	ctx := &notification.NotificationContext{
		EventType:       "incident.opened",
		Locale:          "en",
		ClientName:      giantName,
		AssetName:       giantName,
		MonitorName:     giantName,
		SanitizedTarget: "https://example.com/" + giantName,
		FailureReason:   giantName,
		ViewURL:         "https://mindu.ir/incidents/extreme",
	}

	msg := RenderTelegramMessage(ctx)
	length := TelegramMessageLength(msg)
	if length > TelegramMaxMessageChars {
		t.Errorf("extreme field message length (%d UTF-16 units) exceeded 4096 characters", length)
	}
	// Verify critical context is preserved
	if !strings.Contains(msg, "WEBSITE DOWN") {
		t.Errorf("expected critical event type header in message")
	}
	if !strings.Contains(msg, "Client:") {
		t.Errorf("expected Client label in message")
	}
	assertValidHTMLTags(t, msg)
}

func TestRenderTelegramMessage_MalformedUTF8(t *testing.T) {
	malformed := "Bad bytes: \xff\xfe\xfd here and \xc0\xaf trailing"
	ctx := &notification.NotificationContext{
		EventType:     "incident.opened",
		Locale:        "en",
		ClientName:    malformed,
		FailureReason: malformed,
	}

	msg := RenderTelegramMessage(ctx)
	if strings.Contains(msg, "\xff") || strings.Contains(msg, "\xfe") {
		t.Errorf("malformed UTF-8 bytes were not sanitized")
	}
	assertValidHTMLTags(t, msg)
}

func TestRenderTelegramMessage_PlainTextFallback(t *testing.T) {
	ctx := &notification.NotificationContext{
		EventType:            "incident.opened",
		Locale:               "en",
		ClientName:           "Acme Corp",
		AssetName:            "Website",
		MonitorName:          "HTTP Monitor",
		CheckIntervalSeconds: 60,
		FailureReason:        "Connection reset by peer",
		ViewURL:              "https://mindu.ir/incidents/123",
	}

	plain := RenderTelegramMessagePlainText(ctx)
	if strings.Contains(plain, "<b>") || strings.Contains(plain, "</b>") || strings.Contains(plain, "<a href=") {
		t.Errorf("plain text message should not contain HTML tags: %s", plain)
	}
	if !strings.Contains(plain, "WEBSITE DOWN") {
		t.Errorf("plain text should contain event type: %s", plain)
	}
	if !strings.Contains(plain, "Acme Corp") {
		t.Errorf("plain text should contain client name: %s", plain)
	}
	if !strings.Contains(plain, "https://mindu.ir/incidents/123") {
		t.Errorf("plain text should include ViewURL: %s", plain)
	}
}

func TestEnsureValidTelegramHTML_TagBalancing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxChars int
		wantEnd  string
	}{
		{
			name:     "already balanced HTML",
			input:    "<b>Bold</b> and <i>Italic</i>",
			maxChars: 100,
			wantEnd:  "</i>",
		},
		{
			name:     "unclosed bold tag gets closed",
			input:    "<b>Unclosed bold text",
			maxChars: 100,
			wantEnd:  "</b>",
		},
		{
			name:     "nested unclosed tags get closed in reverse order",
			input:    "<b>Bold <code>Code <a href=\"#\">Link",
			maxChars: 100,
			wantEnd:  "</a></code></b>",
		},
		{
			name:     "truncation inside tag closes open tags safely",
			input:    "<b>" + strings.Repeat("word ", 50) + "</b>",
			maxChars: 30,
			wantEnd:  "...</b>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnsureValidTelegramHTML(tt.input, tt.maxChars)
			if !strings.HasSuffix(got, tt.wantEnd) {
				t.Errorf("EnsureValidTelegramHTML() = %q, expected suffix %q", got, tt.wantEnd)
			}
			assertValidHTMLTags(t, got)
		})
	}
}

// Helper to assert that all opened tags in a message are properly closed and matched.
func assertValidHTMLTags(t *testing.T, htmlStr string) {
	t.Helper()
	var stack []string
	tokens := strings.Split(htmlStr, "<")
	for i, token := range tokens {
		if i == 0 {
			continue
		}
		gtIdx := strings.Index(token, ">")
		if gtIdx == -1 {
			t.Errorf("unclosed tag angle bracket in: %s", token)
			continue
		}
		tagContent := token[:gtIdx]
		if strings.HasPrefix(tagContent, "/") {
			// Closing tag
			tagName := strings.ToLower(tagContent[1:])
			if len(stack) == 0 {
				t.Errorf("unexpected closing tag </%s> with empty stack in: %s", tagName, htmlStr)
				return
			}
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if top != tagName {
				t.Errorf("mismatched closing tag: got </%s>, expected </%s> in: %s", tagName, top, htmlStr)
				return
			}
		} else {
			// Opening tag
			fields := strings.Fields(tagContent)
			if len(fields) > 0 {
				tagName := strings.ToLower(fields[0])
				// Ignore self-closing tags if any
				if !strings.HasSuffix(tagContent, "/") {
					stack = append(stack, tagName)
				}
			}
		}
	}
	if len(stack) > 0 {
		t.Errorf("unclosed tags remaining in stack: %v in: %s", stack, htmlStr)
	}
}
