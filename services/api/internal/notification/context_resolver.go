package notification

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"clientpulse/services/api/internal/appurl"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Helper extracting string from payload
func getString(m map[string]interface{}, key, def string) string {
	if val, ok := m[key]; ok && val != nil {
		if s, ok := val.(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
		if s, ok := val.(fmt.Stringer); ok {
			return s.String()
		}
	}
	return def
}

// Helper extracting int from payload
func getInt(m map[string]interface{}, key string, def int) int {
	if val, ok := m[key]; ok && val != nil {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return def
}

// Helper extracting time from payload (RFC3339 string)
func getTime(m map[string]interface{}, key string) *time.Time {
	if val, ok := m[key]; ok && val != nil {
		if s, ok := val.(string); ok && s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return &t
			}
			if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
				return &t
			}
		}
	}
	return nil
}

// ParseContextFromPayload converts raw event payload into a typed NotificationContext.
func ParseContextFromPayload(eventType string, p map[string]interface{}, appBaseURL, destLocale string) *NotificationContext {
	builder, _ := appurl.NewLinkBuilder(appBaseURL, false)

	// Locale precedence: destination override -> payload locale -> "en"
	locale := "en"
	if destLocale != "" && (destLocale == "en" || destLocale == "fa") {
		locale = destLocale
	} else if loc := getString(p, "locale", ""); loc == "en" || loc == "fa" {
		locale = loc
	}

	timezone := getString(p, "timezone", getString(p, "org_timezone", "UTC"))
	incID := getString(p, "incident_id", "")
	targetURL := getString(p, "target_url", getString(p, "url", ""))

	ctx := &NotificationContext{
		EventType:            eventType,
		IncidentID:           incID,
		IncidentRef:          FormatIncidentReference(incID),
		OrganizationID:       getString(p, "organization_id", ""),
		OrgName:              getString(p, "organization_name", "ClientPulse"),
		Timezone:             timezone,
		Locale:               locale,
		ClientID:             getString(p, "client_id", ""),
		ClientName:           getString(p, "client_name", "Unknown Client"),
		AssetID:              getString(p, "asset_id", ""),
		AssetName:            getString(p, "asset_name", ""),
		AssetType:            getString(p, "asset_type", "website"),
		AssetURL:             getString(p, "asset_url", ""),
		SafeHostname:         getString(p, "safe_hostname", ExtractSafeHostname(targetURL)),
		AssetStatus:          getString(p, "asset_status", "active"),
		MonitorID:            getString(p, "monitor_id", ""),
		MonitorName:          getString(p, "monitor_name", ""),
		MonitorType:          getString(p, "monitor_type", "HTTP"),
		TargetURL:            targetURL,
		SanitizedTarget:      SanitizeTargetURL(targetURL),
		CheckIntervalSeconds: getInt(p, "interval_seconds", 60),
		ExpectedStatusMin:    getInt(p, "expected_status_min", 200),
		ExpectedStatusMax:    getInt(p, "expected_status_max", 399),
		HTTPStatusCode:       getInt(p, "http_status_code", 0),
		FailureCategory:      getString(p, "failure_category", "service_error"),
		FailureReason:        getString(p, "failure_reason", getString(p, "summary", "")),
		RecoveryType:         getString(p, "recovery_type", "scheduled_check_pass"),
		RecoveryStatus:       getString(p, "recovery_status", "HTTP 200 OK"),
		FirstFailureAt:       getTime(p, "first_failure_at"),
		ConfirmedAt:          getTime(p, "confirmed_at"),
		AcknowledgedAt:       getTime(p, "acknowledged_at"),
		AcknowledgedByName:   getString(p, "acknowledged_by_name", getString(p, "acknowledged_by", "On-call Engineer")),
		AcknowledgedByEmail:  getString(p, "acknowledged_by_email", ""),
		ResolvedAt:           getTime(p, "resolved_at"),
		HeartbeatID:          getString(p, "heartbeat_id", ""),
		HeartbeatName:        getString(p, "heartbeat_name", "Heartbeat Monitor"),
		LastPingAt:           getTime(p, "last_ping_at"),
		ExpectedIntervalSeconds: getInt(p, "expected_interval_seconds", 300),
		GracePeriodSeconds:      getInt(p, "grace_period_seconds", 120),
		ExpectedDeadlineAt:      getTime(p, "missed_at"),
		ReportID:             getString(p, "report_id", ""),
		ReportTitle:          getString(p, "report_title", "Care & Maintenance Report"),
		PeriodStart:          getString(p, "period_start", ""),
		PeriodEnd:            getString(p, "period_end", ""),
	}

	// Navigation link using verified canonical routes
	if builder != nil && builder.IsConfigured() {
		if incID != "" {
			ctx.ViewURL = builder.IncidentURL(incID)
		} else if ctx.HeartbeatID != "" {
			ctx.ViewURL = builder.HeartbeatURL(ctx.HeartbeatID)
		} else if ctx.ReportID != "" {
			ctx.ViewURL = builder.ReportURL(ctx.ReportID)
		} else {
			ctx.ViewURL = builder.DashboardURL()
		}
	} else {
		ctx.ViewURL = ""
	}

	// Fallback naming
	if ctx.MonitorName == "" {
		if ctx.AssetName != "" {
			ctx.MonitorName = fmt.Sprintf("%s %s Monitor", ctx.AssetName, strings.ToUpper(ctx.AssetType))
		} else {
			ctx.MonitorName = "Website Monitor"
		}
	}

	// Fallback asset name display
	if ctx.AssetName == "" {
		if locale == "fa" {
			ctx.AssetName = "بدون دارایی متصل"
		} else {
			ctx.AssetName = "No asset linked"
		}
	}

	// Dynamic Duration Calculation
	now := time.Now().UTC()

	// If resolved_at is already present in payload:
	if ctx.ResolvedAt != nil {
		ctx.IsAlreadyResolved = true
		if ctx.FirstFailureAt != nil {
			dur := ctx.ResolvedAt.Sub(*ctx.FirstFailureAt).Seconds()
			if dur < 0 {
				dur = 0
			}
			ctx.DurationSeconds = int64(dur)
		} else {
			ctx.DurationSeconds = int64(getInt(p, "duration_sec", 0))
		}
	} else if ctx.FirstFailureAt != nil {
		// Ongoing incident: calculate duration from start to now
		dur := now.Sub(*ctx.FirstFailureAt).Seconds()
		if dur < 0 {
			dur = 0
		}
		ctx.DurationSeconds = int64(dur)
	} else if ctx.ConfirmedAt != nil {
		dur := now.Sub(*ctx.ConfirmedAt).Seconds()
		if dur < 0 {
			dur = 0
		}
		ctx.DurationSeconds = int64(dur)
	}

	// Heartbeat overdue duration
	if eventType == "heartbeat.missed" {
		if ctx.ExpectedDeadlineAt != nil {
			overdue := now.Sub(*ctx.ExpectedDeadlineAt).Seconds()
			if overdue < 0 {
				overdue = 0
			}
			ctx.OverdueSeconds = int64(overdue)
		} else {
			ctx.OverdueSeconds = int64(getInt(p, "overdue_sec", 0))
		}
	} else if eventType == "heartbeat.recovered" {
		if durSec := getInt(p, "duration_sec", 0); durSec > 0 {
			ctx.OverdueSeconds = int64(durSec)
		} else if ctx.ResolvedAt != nil && ctx.ExpectedDeadlineAt != nil {
			overdue := ctx.ResolvedAt.Sub(*ctx.ExpectedDeadlineAt).Seconds()
			if overdue < 0 {
				overdue = 0
			}
			ctx.OverdueSeconds = int64(overdue)
		}
	}

	// Time to acknowledge
	if eventType == "incident.acknowledged" && ctx.AcknowledgedAt != nil && ctx.FirstFailureAt != nil {
		tta := ctx.AcknowledgedAt.Sub(*ctx.FirstFailureAt).Seconds()
		if tta < 0 {
			tta = 0
		}
		ctx.TimeToAcknowledgeSeconds = int64(tta)
	}

	return ctx
}

// NotificationContextResolver enriches the notification context at dispatch/delivery time
// by performing tenant-scoped database lookups when required, especially checking if an
// ongoing incident was already resolved while queued.
type NotificationContextResolver struct {
	pool *pgxpool.Pool
}

// NewNotificationContextResolver creates a resolver.
func NewNotificationContextResolver(pool *pgxpool.Pool) *NotificationContextResolver {
	return &NotificationContextResolver{pool: pool}
}

// ResolveAndRefreshContext resolves the context and verifies current incident lifecycle state.
func (r *NotificationContextResolver) ResolveAndRefreshContext(
	ctx context.Context,
	orgID string,
	eventType string,
	payload map[string]interface{},
	appBaseURL string,
	destLocale string,
) *NotificationContext {
	notifCtx := ParseContextFromPayload(eventType, payload, appBaseURL, destLocale)

	// If no database pool available, return parsed payload context
	if r.pool == nil || orgID == "" {
		return notifCtx
	}

	// 1. Fetch organization timezone and locale to ensure accurate localized formatting
	var orgTimezone, orgLocale string
	orgErr := r.pool.QueryRow(ctx, `
		SELECT COALESCE(timezone, 'UTC'), COALESCE(locale, 'en')
		FROM organizations
		WHERE id = $1
	`, orgID).Scan(&orgTimezone, &orgLocale)
	if orgErr == nil {
		if orgTimezone != "" && (notifCtx.Timezone == "" || notifCtx.Timezone == "UTC") {
			notifCtx.Timezone = orgTimezone
		}
		if destLocale == "" && getString(payload, "locale", "") == "" && orgLocale != "" {
			if orgLocale == "fa" || orgLocale == "en" {
				notifCtx.Locale = orgLocale
			}
		}
	}

	// 2. If this is an incident event (opened / acknowledged), verify current incident state in DB
	if (eventType == "incident.opened" || eventType == "incident.acknowledged") && notifCtx.IncidentID != "" {
		var status string
		var resolvedAt *time.Time
		var durSec *int
		var ackAt *time.Time
		var ackBy *string
		var ackName *string

		// Tenant-scoped query ensuring no cross-tenant leakage
		err := r.pool.QueryRow(ctx, `
			SELECT i.status, i.resolved_at, i.duration_seconds, i.acknowledged_at, i.acknowledged_by,
			       u.display_name
			FROM incidents i
			LEFT JOIN users u ON u.id = i.acknowledged_by
			WHERE i.id = $1 AND i.organization_id = $2
		`, notifCtx.IncidentID, orgID).Scan(&status, &resolvedAt, &durSec, &ackAt, &ackBy, &ackName)

		if err == nil {
			if status == "resolved" || resolvedAt != nil {
				notifCtx.IsAlreadyResolved = true
				notifCtx.ResolvedAt = resolvedAt
				if durSec != nil && *durSec > 0 {
					notifCtx.DurationSeconds = int64(*durSec)
				} else if resolvedAt != nil && notifCtx.FirstFailureAt != nil {
					notifCtx.DurationSeconds = int64(resolvedAt.Sub(*notifCtx.FirstFailureAt).Seconds())
				}
			}

			if ackAt != nil && notifCtx.AcknowledgedAt == nil {
				notifCtx.AcknowledgedAt = ackAt
				if ackName != nil && *ackName != "" {
					notifCtx.AcknowledgedByName = *ackName
				}
			}
		}
	}

	return notifCtx
}

// HTMLEscape escapes user controlled strings for safe Telegram HTML and Email HTML rendering.
func HTMLEscape(s string) string {
	return html.EscapeString(s)
}

