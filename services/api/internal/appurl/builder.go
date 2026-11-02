package appurl

import (
	"fmt"
	"net"
	"net/url"
	"path"
	"strings"
)

// LinkBuilder constructs verified canonical application URLs for notifications and emails.
type LinkBuilder struct {
	baseURL *url.URL
	baseStr string
}

// NewLinkBuilder parses, validates, and initializes a LinkBuilder.
// If rawURL is empty, LinkBuilder is unconfigured (IsConfigured() == false)
// and all route methods return empty string without errors.
func NewLinkBuilder(rawURL string, isProduction bool) (*LinkBuilder, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return &LinkBuilder{baseURL: nil, baseStr: ""}, nil
	}

	if err := ValidatePublicAppURL(trimmed, isProduction); err != nil {
		return nil, err
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PUBLIC_APP_URL: %w", err)
	}

	// Normalize base URL: strip trailing slashes from path, preserve non-empty base path safely.
	cleanPath := path.Clean(parsed.Path)
	if cleanPath == "." || cleanPath == "/" {
		cleanPath = ""
	} else if strings.HasSuffix(parsed.Path, "/") && !strings.HasSuffix(cleanPath, "/") {
		// Clean removes trailing slashes
	}
	cleanPath = strings.TrimRight(cleanPath, "/")

	// Construct normalized base string: scheme://host[:port][/basepath]
	baseStr := fmt.Sprintf("%s://%s%s", strings.ToLower(parsed.Scheme), parsed.Host, cleanPath)

	return &LinkBuilder{
		baseURL: parsed,
		baseStr: baseStr,
	}, nil
}

// ValidatePublicAppURL validates the application URL according to strict security rules.
func ValidatePublicAppURL(rawURL string, isProduction bool) error {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("PUBLIC_APP_URL is not a valid URL: %w", err)
	}

	// Must be an absolute URL with scheme
	if !parsed.IsAbs() || parsed.Scheme == "" {
		return fmt.Errorf("PUBLIC_APP_URL must be an absolute URL containing both scheme and host (e.g. https://mindu.ir)")
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("PUBLIC_APP_URL scheme must be http or https, got %q", parsed.Scheme)
	}

	// Must include host
	if parsed.Host == "" {
		return fmt.Errorf("PUBLIC_APP_URL must be an absolute URL containing both scheme and host (e.g. https://mindu.ir)")
	}

	// Reject credentials (username / password)
	if parsed.User != nil {
		return fmt.Errorf("PUBLIC_APP_URL must not contain credentials (username or password)")
	}

	// Reject query string
	if parsed.RawQuery != "" {
		return fmt.Errorf("PUBLIC_APP_URL must not contain query parameters")
	}

	// Reject fragments
	if parsed.Fragment != "" {
		return fmt.Errorf("PUBLIC_APP_URL must not contain a URL fragment")
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("PUBLIC_APP_URL must include a valid hostname")
	}

	if isProduction {
		if scheme != "https" {
			return fmt.Errorf("production PUBLIC_APP_URL must use https scheme, got %q", scheme)
		}

		lowerHost := strings.ToLower(hostname)
		if lowerHost == "localhost" || lowerHost == "127.0.0.1" || lowerHost == "::1" || lowerHost == "0.0.0.0" || strings.HasSuffix(lowerHost, ".local") {
			return fmt.Errorf("production PUBLIC_APP_URL cannot be localhost or loopback address: %s", hostname)
		}

		// Ensure hostname contains at least one dot or is a valid IP that is not loopback/private
		ip := net.ParseIP(lowerHost)
		if ip != nil {
			if ip.IsLoopback() || ip.IsUnspecified() || ip.IsPrivate() {
				return fmt.Errorf("production PUBLIC_APP_URL cannot use loopback or private IP: %s", hostname)
			}
		} else {
			if !strings.Contains(lowerHost, ".") {
				return fmt.Errorf("production PUBLIC_APP_URL must have a fully-qualified domain name: %s", hostname)
			}
		}
	}

	return nil
}

// IsConfigured reports whether a valid PUBLIC_APP_URL was configured.
func (b *LinkBuilder) IsConfigured() bool {
	return b != nil && b.baseURL != nil && b.baseStr != ""
}

// BaseURL returns the canonical base URL string or empty string if not configured.
func (b *LinkBuilder) BaseURL() string {
	if !b.IsConfigured() {
		return ""
	}
	return b.baseStr
}

// IncidentURL returns the canonical route to an incident detail page,
// or list page if incidentID is empty. Returns empty string if unconfigured.
func (b *LinkBuilder) IncidentURL(incidentID string) string {
	if !b.IsConfigured() {
		return ""
	}
	trimmed := strings.TrimSpace(incidentID)
	if trimmed == "" {
		return fmt.Sprintf("%s/incidents", b.baseStr)
	}
	return fmt.Sprintf("%s/incidents/%s", b.baseStr, url.PathEscape(trimmed))
}

// HeartbeatURL returns the canonical route to a heartbeat monitor detail page,
// or list page if heartbeatID is empty. Returns empty string if unconfigured.
func (b *LinkBuilder) HeartbeatURL(heartbeatID string) string {
	if !b.IsConfigured() {
		return ""
	}
	trimmed := strings.TrimSpace(heartbeatID)
	if trimmed == "" {
		return fmt.Sprintf("%s/heartbeats", b.baseStr)
	}
	return fmt.Sprintf("%s/heartbeats/%s", b.baseStr, url.PathEscape(trimmed))
}

// ReportURL returns the canonical route to a Care & Maintenance report detail page,
// or list page if reportID is empty. Returns empty string if unconfigured.
func (b *LinkBuilder) ReportURL(reportID string) string {
	if !b.IsConfigured() {
		return ""
	}
	trimmed := strings.TrimSpace(reportID)
	if trimmed == "" {
		return fmt.Sprintf("%s/reports", b.baseStr)
	}
	return fmt.Sprintf("%s/reports/%s", b.baseStr, url.PathEscape(trimmed))
}

// DashboardURL returns the canonical route to the main dashboard.
// Returns empty string if unconfigured.
func (b *LinkBuilder) DashboardURL() string {
	if !b.IsConfigured() {
		return ""
	}
	return b.baseStr
}

// EmailVerificationURL returns the email verification route.
// Returns empty string if unconfigured.
func (b *LinkBuilder) EmailVerificationURL(token string) string {
	if !b.IsConfigured() {
		return ""
	}
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return fmt.Sprintf("%s/settings/notifications", b.baseStr)
	}
	return fmt.Sprintf("%s/settings/notifications?verify_email=%s", b.baseStr, url.QueryEscape(trimmed))
}
