package notification

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// NotificationContext represents the standard, fully resolved contextual data for rendering alerts.
type NotificationContext struct {
	EventType      string `json:"event_type"`
	IncidentID     string `json:"incident_id,omitempty"`
	IncidentRef    string `json:"incident_ref,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`
	OrgName        string `json:"organization_name,omitempty"`
	Timezone       string `json:"timezone,omitempty"`
	Locale         string `json:"locale,omitempty"`

	// Client
	ClientID   string `json:"client_id,omitempty"`
	ClientName string `json:"client_name,omitempty"`

	// Asset
	AssetID      string `json:"asset_id,omitempty"`
	AssetName    string `json:"asset_name,omitempty"`
	AssetType    string `json:"asset_type,omitempty"`
	AssetURL     string `json:"asset_url,omitempty"`
	SafeHostname string `json:"safe_hostname,omitempty"`
	AssetStatus  string `json:"asset_status,omitempty"`

	// Monitor
	MonitorID            string `json:"monitor_id,omitempty"`
	MonitorName          string `json:"monitor_name,omitempty"`
	MonitorType          string `json:"monitor_type,omitempty"`
	TargetURL            string `json:"target_url,omitempty"`
	SanitizedTarget      string `json:"sanitized_target,omitempty"`
	CheckIntervalSeconds int    `json:"check_interval_seconds,omitempty"`
	ExpectedStatusMin    int    `json:"expected_status_min,omitempty"`
	ExpectedStatusMax    int    `json:"expected_status_max,omitempty"`
	MonitorStatus        string `json:"monitor_status,omitempty"`

	// Failure / Check Result Info
	HTTPStatusCode int    `json:"http_status_code,omitempty"`
	FailureCategory string `json:"failure_category,omitempty"`
	FailureReason   string `json:"failure_reason,omitempty"`
	RecoveryType    string `json:"recovery_type,omitempty"`
	RecoveryStatus  string `json:"recovery_status,omitempty"`

	// Timing & Duration
	FirstFailureAt           *time.Time `json:"first_failure_at,omitempty"`
	ConfirmedAt              *time.Time `json:"confirmed_at,omitempty"`
	AcknowledgedAt           *time.Time `json:"acknowledged_at,omitempty"`
	AcknowledgedByName       string     `json:"acknowledged_by_name,omitempty"`
	AcknowledgedByEmail      string     `json:"acknowledged_by_email,omitempty"`
	TimeToAcknowledgeSeconds int64      `json:"time_to_acknowledge_seconds,omitempty"`
	ResolvedAt               *time.Time `json:"resolved_at,omitempty"`
	DurationSeconds          int64      `json:"duration_seconds,omitempty"`
	IsAlreadyResolved        bool       `json:"is_already_resolved,omitempty"`

	// Heartbeat Specifics
	HeartbeatID             string     `json:"heartbeat_id,omitempty"`
	HeartbeatName           string     `json:"heartbeat_name,omitempty"`
	LastPingAt              *time.Time `json:"last_ping_at,omitempty"`
	ExpectedIntervalSeconds int        `json:"expected_interval_seconds,omitempty"`
	GracePeriodSeconds      int        `json:"grace_period_seconds,omitempty"`
	ExpectedDeadlineAt      *time.Time `json:"expected_deadline_at,omitempty"`
	OverdueSeconds          int64      `json:"overdue_seconds,omitempty"`

	// Reports
	ReportID    string `json:"report_id,omitempty"`
	ReportTitle string `json:"report_title,omitempty"`
	PeriodStart string `json:"period_start,omitempty"`
	PeriodEnd   string `json:"period_end,omitempty"`

	// Navigation
	ViewURL string `json:"view_url,omitempty"`
}

// FormatIncidentReference creates a standard concise human-readable reference like #INC-3B8D2DE3
func FormatIncidentReference(incidentID string) string {
	clean := strings.ReplaceAll(incidentID, "-", "")
	if len(clean) >= 8 {
		return fmt.Sprintf("#INC-%s", strings.ToUpper(clean[:8]))
	}
	if len(incidentID) > 0 {
		return fmt.Sprintf("#INC-%s", strings.ToUpper(incidentID))
	}
	return ""
}

// SanitizeTargetURL removes sensitive credentials, userinfo, and token parameters from target URLs.
func SanitizeTargetURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		// Fallback: return raw string if not a parseable URL
		return rawURL
	}

	// Remove userinfo (e.g. username:password@)
	u.User = nil

	// Strip URL fragments which might contain access tokens or auth state
	u.Fragment = ""

	// Sanitize sensitive query parameters
	if u.RawQuery != "" {
		q := u.Query()
		sensitiveKeys := []string{
			"token", "api_key", "apikey", "key", "secret", "password",
			"auth", "access_token", "refresh_token", "jwt", "code",
			"session", "sig", "signature", "client_secret",
		}
		modified := false
		for _, key := range sensitiveKeys {
			for qKey := range q {
				if strings.EqualFold(qKey, key) {
					q.Del(qKey)
					modified = true
				}
			}
		}
		if modified {
			u.RawQuery = q.Encode()
		}
	}

	return u.String()
}

// ExtractSafeHostname extracts a clean domain or IP from an asset URL.
func ExtractSafeHostname(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	// Fallback to strip protocol manually
	trimmed := strings.TrimPrefix(rawURL, "https://")
	trimmed = strings.TrimPrefix(trimmed, "http://")
	parts := strings.Split(trimmed, "/")
	return parts[0]
}

// FormatDuration converts seconds into human-readable compact notation (e.g. 45s, 8m 42s, 2h 15m, 1d 4h).
func FormatDuration(seconds int64, isPersian bool) string {
	if seconds < 0 {
		seconds = 0
	}

	if isPersian {
		if seconds < 60 {
			return fmt.Sprintf("%s ثانیه", ToPersianDigits(fmt.Sprintf("%d", seconds)))
		}
		if seconds < 3600 {
			m := seconds / 60
			s := seconds % 60
			if s > 0 {
				return fmt.Sprintf("%s دقیقه و %s ثانیه", ToPersianDigits(fmt.Sprintf("%d", m)), ToPersianDigits(fmt.Sprintf("%d", s)))
			}
			return fmt.Sprintf("%s دقیقه", ToPersianDigits(fmt.Sprintf("%d", m)))
		}
		if seconds < 86400 {
			h := seconds / 3600
			m := (seconds % 3600) / 60
			if m > 0 {
				return fmt.Sprintf("%s ساعت و %s دقیقه", ToPersianDigits(fmt.Sprintf("%d", h)), ToPersianDigits(fmt.Sprintf("%d", m)))
			}
			return fmt.Sprintf("%s ساعت", ToPersianDigits(fmt.Sprintf("%d", h)))
		}
		d := seconds / 86400
		h := (seconds % 86400) / 3600
		if h > 0 {
			return fmt.Sprintf("%s روز و %s ساعت", ToPersianDigits(fmt.Sprintf("%d", d)), ToPersianDigits(fmt.Sprintf("%d", h)))
		}
		return fmt.Sprintf("%s روز", ToPersianDigits(fmt.Sprintf("%d", d)))
	}

	// English format
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		m := seconds / 60
		s := seconds % 60
		if s > 0 {
			return fmt.Sprintf("%dm %ds", m, s)
		}
		return fmt.Sprintf("%dm", m)
	}
	if seconds < 86400 {
		h := seconds / 3600
		m := (seconds % 3600) / 60
		if m > 0 {
			return fmt.Sprintf("%dh %dm", h, m)
		}
		return fmt.Sprintf("%dh", h)
	}
	d := seconds / 86400
	h := (seconds % 86400) / 3600
	if h > 0 {
		return fmt.Sprintf("%dd %dh", d, h)
	}
	return fmt.Sprintf("%dd", d)
}

// FormatTimestamp converts a timestamp to organization timezone and locale.
func FormatTimestamp(t time.Time, timezone string, isPersian bool) string {
	if t.IsZero() {
		return ""
	}

	loc := time.UTC
	if timezone != "" {
		if l, err := time.LoadLocation(timezone); err == nil {
			loc = l
		}
	}

	inLoc := t.In(loc)
	formatted := inLoc.Format("2006-01-02 15:04:05 MST")

	if isPersian {
		return ToPersianDigits(formatted)
	}
	return formatted
}

// FormatTimeOnly formats only the clock portion with timezone (e.g. 15:04:05 UTC)
func FormatTimeOnly(t time.Time, timezone string, isPersian bool) string {
	if t.IsZero() {
		return ""
	}

	loc := time.UTC
	if timezone != "" {
		if l, err := time.LoadLocation(timezone); err == nil {
			loc = l
		}
	}

	inLoc := t.In(loc)
	formatted := inLoc.Format("15:04:05 MST")

	if isPersian {
		return ToPersianDigits(formatted)
	}
	return formatted
}

// ToPersianDigits converts ASCII digits 0-9 to Persian digits ۰-۹.
func ToPersianDigits(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch r {
		case '0':
			sb.WriteRune('۰')
		case '1':
			sb.WriteRune('۱')
		case '2':
			sb.WriteRune('۲')
		case '3':
			sb.WriteRune('۳')
		case '4':
			sb.WriteRune('۴')
		case '5':
			sb.WriteRune('۵')
		case '6':
			sb.WriteRune('۶')
		case '7':
			sb.WriteRune('۷')
		case '8':
			sb.WriteRune('۸')
		case '9':
			sb.WriteRune('۹')
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

