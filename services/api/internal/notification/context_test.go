package notification

import (
	"strings"
	"testing"
	"time"
)

func TestFormatIncidentReference(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"3b8d2de3-92f5-4e7a-9a99-b1d61a0f9fa4", "#INC-3B8D2DE3"},
		{"short", "#INC-SHORT"},
		{"1234567890", "#INC-12345678"},
		{"test-inc-12345678-abcd", "#INC-TESTINC1"},
	}

	for _, tc := range tests {
		actual := FormatIncidentReference(tc.input)
		if actual != tc.expected {
			t.Errorf("FormatIncidentReference(%q) = %q; expected %q", tc.input, actual, tc.expected)
		}
	}
}

func TestSanitizeTargetURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "clean https url",
			input:    "https://api.example.com/v1/health",
			expected: "https://api.example.com/v1/health",
		},
		{
			name:     "url with credentials",
			input:    "https://user:secret123@api.example.com/data",
			expected: "https://api.example.com/data",
		},
		{
			name:     "url with sensitive query params and fragment deleted",
			input:    "https://example.com/auth?token=my-secret-token&api_key=key123&page=2#section",
			expected: "https://example.com/auth?page=2",
		},
		{
			name:     "empty url",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := SanitizeTargetURL(tc.input)
			if actual != tc.expected {
				t.Errorf("SanitizeTargetURL(%q) = %q; expected %q", tc.input, actual, tc.expected)
			}
		})
	}
}

func TestExtractSafeHostname(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://sub.domain.com:8443/path", "sub.domain.com"},
		{"http://192.168.1.1:8080/health", "192.168.1.1"},
		{"api.clientpulse.io", "api.clientpulse.io"},
		{"", ""},
	}

	for _, tc := range tests {
		actual := ExtractSafeHostname(tc.input)
		if actual != tc.expected {
			t.Errorf("ExtractSafeHostname(%q) = %q; expected %q", tc.input, actual, tc.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds int64
		isFA    bool
		want    string
	}{
		{0, false, "0s"},
		{0, true, "۰ ثانیه"},
		{45, false, "45s"},
		{45, true, "۴۵ ثانیه"},
		{522, false, "8m 42s"},
		{522, true, "۸ دقیقه و ۴۲ ثانیه"},
		{8100, false, "2h 15m"},
		{8100, true, "۲ ساعت و ۱۵ دقیقه"},
		{100800, false, "1d 4h"},
		{100800, true, "۱ روز و ۴ ساعت"},
	}

	for _, tc := range tests {
		actual := FormatDuration(tc.seconds, tc.isFA)
		if actual != tc.want {
			t.Errorf("FormatDuration(%d, %v) = %q; want %q", tc.seconds, tc.isFA, actual, tc.want)
		}
	}
}

func TestToPersianDigits(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1234567890", "۱۲۳۴۵۶۷۸۹۰"},
		{"HTTP 500 Error", "HTTP ۵۰۰ Error"},
		{"No digits here", "No digits here"},
	}

	for _, tc := range tests {
		actual := ToPersianDigits(tc.input)
		if actual != tc.want {
			t.Errorf("ToPersianDigits(%q) = %q; want %q", tc.input, actual, tc.want)
		}
	}
}

func TestFormatTimestamp(t *testing.T) {
	fixedTime := time.Date(2026, 11, 2, 14, 30, 0, 0, time.UTC)

	en := FormatTimestamp(fixedTime, "UTC", false)
	if !strings.Contains(en, "2026-11-02") || !strings.Contains(en, "14:30") || !strings.Contains(en, "UTC") {
		t.Errorf("expected standard format for EN timestamp, got: %s", en)
	}

	fa := FormatTimestamp(fixedTime, "UTC", true)
	if !strings.Contains(fa, "۱۴:۳۰") {
		t.Errorf("expected Persian digits in FA timestamp, got: %s", fa)
	}
}

func TestParseContextFromPayload_PrecedenceAndCalculations(t *testing.T) {
	now := time.Now().UTC()
	firstFail := now.Add(-15 * time.Minute)
	confirmed := now.Add(-13 * time.Minute)

	payload := map[string]interface{}{
		"incident_id":       "test-inc-12345678-abcd",
		"client_name":       "Acme Corp",
		"asset_name":        "Main Portal",
		"monitor_name":      "API Health Check",
		"target_url":        "https://api.acme.com/ping",
		"first_failure_at":  firstFail.Format(time.RFC3339),
		"confirmed_at":      confirmed.Format(time.RFC3339),
		"failure_category":  "http_5xx",
		"http_status_code":  503,
		"interval_seconds":  60,
		"organization_name": "Digital Dynamics",
		"timezone":          "America/New_York",
		"locale":            "en",
	}

	// 1. Test destination locale override
	ctxFA := ParseContextFromPayload("incident.opened", payload, "https://app.example.com", "fa")
	if ctxFA.Locale != "fa" {
		t.Errorf("expected destination locale override 'fa', got: %s", ctxFA.Locale)
	}
	if ctxFA.ClientName != "Acme Corp" {
		t.Errorf("expected ClientName 'Acme Corp', got: %s", ctxFA.ClientName)
	}
	if ctxFA.IncidentRef != "#INC-TESTINC1" {
		t.Errorf("expected IncidentRef '#INC-TESTINC1', got: %s", ctxFA.IncidentRef)
	}
	if ctxFA.HTTPStatusCode != 503 {
		t.Errorf("expected HTTPStatusCode 503, got: %d", ctxFA.HTTPStatusCode)
	}
	// Duration ongoing should be around 15m (900s)
	if ctxFA.DurationSeconds < 850 || ctxFA.DurationSeconds > 950 {
		t.Errorf("expected ongoing duration ~900s, got: %d", ctxFA.DurationSeconds)
	}

	// 2. Test resolved payload duration
	resolvedAt := now.Add(-5 * time.Minute)
	payload["resolved_at"] = resolvedAt.Format(time.RFC3339)
	payload["duration_sec"] = 600

	ctxResolved := ParseContextFromPayload("incident.resolved", payload, "https://app.example.com", "")
	if !ctxResolved.IsAlreadyResolved {
		t.Errorf("expected IsAlreadyResolved = true")
	}
	if ctxResolved.DurationSeconds != 600 {
		t.Errorf("expected DurationSeconds 600, got: %d", ctxResolved.DurationSeconds)
	}

	// 3. Test heartbeat overdue calculation
	hbMissedAt := now.Add(-10 * time.Minute)
	hbPayload := map[string]interface{}{
		"heartbeat_id":              "hb-uuid-1",
		"heartbeat_name":            "Daily DB Backup",
		"client_name":               "Starlight",
		"missed_at":                 hbMissedAt.Format(time.RFC3339),
		"expected_interval_seconds": 86400,
		"grace_period_seconds":      3600,
	}

	ctxHB := ParseContextFromPayload("heartbeat.missed", hbPayload, "https://app.example.com", "en")
	if ctxHB.HeartbeatName != "Daily DB Backup" {
		t.Errorf("expected HeartbeatName 'Daily DB Backup', got: %s", ctxHB.HeartbeatName)
	}
	if ctxHB.OverdueSeconds < 550 || ctxHB.OverdueSeconds > 650 {
		t.Errorf("expected OverdueSeconds ~600, got: %d", ctxHB.OverdueSeconds)
	}
}
