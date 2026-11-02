package appurl

import (
	"strings"
	"testing"
)

func TestValidatePublicAppURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		isProduction bool
		wantErr      bool
		errMsg       string
	}{
		{
			name:         "empty URL permitted",
			url:          "",
			isProduction: false,
			wantErr:      false,
		},
		{
			name:         "empty URL permitted in production",
			url:          "   ",
			isProduction: true,
			wantErr:      false,
		},
		{
			name:         "valid development HTTP URL with port",
			url:          "http://localhost:3001",
			isProduction: false,
			wantErr:      false,
		},
		{
			name:         "valid development HTTP IP URL",
			url:          "http://127.0.0.1:3000",
			isProduction: false,
			wantErr:      false,
		},
		{
			name:         "valid HTTPS production URL",
			url:          "https://mindu.ir",
			isProduction: true,
			wantErr:      false,
		},
		{
			name:         "valid HTTPS production with base path",
			url:          "https://mindu.ir/cp",
			isProduction: true,
			wantErr:      false,
		},
		{
			name:         "valid HTTPS production with trailing slash",
			url:          "https://mindu.ir/",
			isProduction: true,
			wantErr:      false,
		},
		{
			name:         "invalid URL format",
			url:          "not-a-url",
			isProduction: false,
			wantErr:      true,
			errMsg:       "absolute URL",
		},
		{
			name:         "relative path rejected",
			url:          "/dashboard",
			isProduction: false,
			wantErr:      true,
			errMsg:       "absolute URL",
		},
		{
			name:         "contains credentials with password",
			url:          "https://user:pass@mindu.ir",
			isProduction: false,
			wantErr:      true,
			errMsg:       "credentials",
		},
		{
			name:         "contains credentials username only",
			url:          "https://user@mindu.ir",
			isProduction: false,
			wantErr:      true,
			errMsg:       "credentials",
		},
		{
			name:         "contains query parameters",
			url:          "https://mindu.ir?token=secret",
			isProduction: false,
			wantErr:      true,
			errMsg:       "query parameters",
		},
		{
			name:         "contains fragment",
			url:          "https://mindu.ir#dashboard",
			isProduction: false,
			wantErr:      true,
			errMsg:       "fragment",
		},
		{
			name:         "unsupported scheme ftp",
			url:          "ftp://mindu.ir",
			isProduction: false,
			wantErr:      true,
			errMsg:       "scheme must be http or https",
		},
		{
			name:         "unsupported scheme javascript",
			url:          "javascript:alert(1)",
			isProduction: false,
			wantErr:      true,
			errMsg:       "scheme must be http or https",
		},
		{
			name:         "production rejects HTTP",
			url:          "http://mindu.ir",
			isProduction: true,
			wantErr:      true,
			errMsg:       "production PUBLIC_APP_URL must use https scheme",
		},
		{
			name:         "production rejects localhost",
			url:          "https://localhost",
			isProduction: true,
			wantErr:      true,
			errMsg:       "cannot be localhost",
		},
		{
			name:         "production rejects 127.0.0.1",
			url:          "https://127.0.0.1",
			isProduction: true,
			wantErr:      true,
			errMsg:       "cannot be localhost or loopback address",
		},
		{
			name:         "production rejects private IP 192.168.1.1",
			url:          "https://192.168.1.1",
			isProduction: true,
			wantErr:      true,
			errMsg:       "cannot use loopback or private IP",
		},
		{
			name:         "production rejects bare hostname without dot",
			url:          "https://myhost",
			isProduction: true,
			wantErr:      true,
			errMsg:       "must have a fully-qualified domain name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePublicAppURL(tt.url, tt.isProduction)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePublicAppURL() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("error %q does not contain expected substring %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestLinkBuilder_RouteGeneration(t *testing.T) {
	t.Run("configured with standard root domain", func(t *testing.T) {
		b, err := NewLinkBuilder("https://mindu.ir", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !b.IsConfigured() {
			t.Fatalf("expected builder to be configured")
		}

		if got := b.DashboardURL(); got != "https://mindu.ir" {
			t.Errorf("DashboardURL() = %q, want https://mindu.ir", got)
		}
		if got := b.IncidentURL("inc-123"); got != "https://mindu.ir/incidents/inc-123" {
			t.Errorf("IncidentURL() = %q, want https://mindu.ir/incidents/inc-123", got)
		}
		if got := b.IncidentURL(""); got != "https://mindu.ir/incidents" {
			t.Errorf("IncidentURL('') = %q, want https://mindu.ir/incidents", got)
		}
		if got := b.HeartbeatURL("hb-456"); got != "https://mindu.ir/heartbeats/hb-456" {
			t.Errorf("HeartbeatURL() = %q, want https://mindu.ir/heartbeats/hb-456", got)
		}
		if got := b.HeartbeatURL(""); got != "https://mindu.ir/heartbeats" {
			t.Errorf("HeartbeatURL('') = %q, want https://mindu.ir/heartbeats", got)
		}
		if got := b.ReportURL("rep-789"); got != "https://mindu.ir/reports/rep-789" {
			t.Errorf("ReportURL() = %q, want https://mindu.ir/reports/rep-789", got)
		}
		if got := b.ReportURL(""); got != "https://mindu.ir/reports" {
			t.Errorf("ReportURL('') = %q, want https://mindu.ir/reports", got)
		}
		if got := b.EmailVerificationURL("tok_abc"); got != "https://mindu.ir/settings/notifications?verify_email=tok_abc" {
			t.Errorf("EmailVerificationURL() = %q, want https://mindu.ir/settings/notifications?verify_email=tok_abc", got)
		}
	})

	t.Run("configured with base path and trailing slash", func(t *testing.T) {
		b, err := NewLinkBuilder("https://mindu.ir/clientpulse/", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := b.DashboardURL(); got != "https://mindu.ir/clientpulse" {
			t.Errorf("DashboardURL() = %q, want https://mindu.ir/clientpulse", got)
		}
		if got := b.IncidentURL("inc-999"); got != "https://mindu.ir/clientpulse/incidents/inc-999" {
			t.Errorf("IncidentURL() = %q, want https://mindu.ir/clientpulse/incidents/inc-999", got)
		}
		if got := b.HeartbeatURL("hb-999"); got != "https://mindu.ir/clientpulse/heartbeats/hb-999" {
			t.Errorf("HeartbeatURL() = %q, want https://mindu.ir/clientpulse/heartbeats/hb-999", got)
		}
		if got := b.ReportURL("rep-999"); got != "https://mindu.ir/clientpulse/reports/rep-999" {
			t.Errorf("ReportURL() = %q, want https://mindu.ir/clientpulse/reports/rep-999", got)
		}
	})

	t.Run("unconfigured empty URL yields empty links", func(t *testing.T) {
		b, err := NewLinkBuilder("", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if b.IsConfigured() {
			t.Fatalf("expected builder to be unconfigured")
		}

		if got := b.DashboardURL(); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
		if got := b.IncidentURL("inc-123"); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
		if got := b.HeartbeatURL("hb-456"); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
		if got := b.ReportURL("rep-789"); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
		if got := b.EmailVerificationURL("token"); got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("injection prevention in path parameters", func(t *testing.T) {
		b, err := NewLinkBuilder("https://mindu.ir", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Attempt directory traversal or open redirect injection
		maliciousID := "../evil.com/redirect"
		got := b.IncidentURL(maliciousID)
		if strings.Contains(got, "/../") || !strings.HasPrefix(got, "https://mindu.ir/incidents/") {
			t.Errorf("vulnerable to path injection: %q", got)
		}
		// Expect url.PathEscape encoding of slashes
		if !strings.Contains(got, "%2F") {
			t.Errorf("slashes should be path escaped in parameter, got %q", got)
		}

		// Attempt query injection in verification token
		tokenInjection := "tok&evil=true"
		gotVerify := b.EmailVerificationURL(tokenInjection)
		if strings.Contains(gotVerify, "&evil=true") {
			t.Errorf("query parameter injection detected: %q", gotVerify)
		}
	})
}
