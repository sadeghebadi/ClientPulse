package repository

import (
	"time"
)

// User represents a user account.
type User struct {
	ID              string     `json:"id"`
	Email           string     `json:"email"`
	PasswordHash    string     `json:"-"`
	DisplayName     string     `json:"display_name"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Organization represents a multi-tenant agency workspace.
type Organization struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Slug      string     `json:"slug"`
	Locale    string     `json:"locale"`
	Timezone  string     `json:"timezone"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// Membership links a user to an organization with a specific role.
type Membership struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	UserID         string    `json:"user_id"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	// Joined fields
	UserEmail       string `json:"user_email,omitempty"`
	UserDisplayName string `json:"user_display_name,omitempty"`
	OrgName         string `json:"org_name,omitempty"`
	OrgSlug         string `json:"org_slug,omitempty"`
}

// Session represents an authenticated user session.
type Session struct {
	ID                   string     `json:"id"`
	UserID               string     `json:"user_id"`
	TokenHash            string     `json:"-"`
	ActiveOrganizationID *string    `json:"active_organization_id,omitempty"`
	IPAddress            string     `json:"ip_address"`
	UserAgent            string     `json:"user_agent"`
	ExpiresAt            time.Time  `json:"expires_at"`
	RevokedAt            *time.Time `json:"revoked_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	LastSeenAt           time.Time  `json:"last_seen_at"`
}

// TokenRecord represents a single-use token (password reset or email verification).
type TokenRecord struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// AuditLog represents an immutable audit entry.
type AuditLog struct {
	ID             string    `json:"id"`
	OrganizationID *string   `json:"organization_id,omitempty"`
	ActorUserID    *string   `json:"actor_user_id,omitempty"`
	Action         string    `json:"action"`
	EntityType     string    `json:"entity_type"`
	EntityID       string    `json:"entity_id"`
	MetadataJSON   []byte    `json:"metadata_json"`
	IPHash         string    `json:"ip_hash"`
	CreatedAt      time.Time `json:"created_at"`
}

// Client represents an agency client.
type Client struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	Name           string     `json:"name"`
	ContactName    *string    `json:"contact_name,omitempty"`
	ContactEmail   *string    `json:"contact_email,omitempty"`
	Website        *string    `json:"website,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ArchivedAt     *time.Time `json:"archived_at,omitempty"`

	// Aggregated metadata
	AssetCount   int `json:"asset_count,omitempty"`
	MonitorCount int `json:"monitor_count,omitempty"`
}

// Asset represents a website or API belonging to a client.
type Asset struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	ClientID       string     `json:"client_id"`
	Name           string     `json:"name"`
	URL            string     `json:"url"`
	Type           string     `json:"type"`
	Environment    string     `json:"environment"`
	Description    *string    `json:"description,omitempty"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ArchivedAt     *time.Time `json:"archived_at,omitempty"`

	// Joined metadata
	ClientName    string     `json:"client_name,omitempty"`
	MonitorStatus *string    `json:"monitor_status,omitempty"`
	MonitorID     *string    `json:"monitor_id,omitempty"`
	LastCheckedAt *time.Time `json:"last_checked_at,omitempty"`
}

// Monitor represents an automated health check definition for an asset.
type Monitor struct {
	ID                string     `json:"id"`
	OrganizationID    string     `json:"organization_id"`
	AssetID           string     `json:"asset_id"`
	Method            string     `json:"method"`
	URLOverride       *string    `json:"url_override,omitempty"`
	IntervalSeconds   int        `json:"interval_seconds"`
	TimeoutSeconds    int        `json:"timeout_seconds"`
	ExpectedStatusMin int        `json:"expected_status_min"`
	ExpectedStatusMax int        `json:"expected_status_max"`
	FollowRedirects   bool       `json:"follow_redirects"`
	MaxRedirects      int        `json:"max_redirects"`
	ExpectedText      *string    `json:"expected_text,omitempty"`
	JSONPointer       *string    `json:"json_pointer,omitempty"`
	JSONExpectedValue *string    `json:"json_expected_value,omitempty"`
	Status            string     `json:"status"`
	IsActive          bool       `json:"is_active"`
	NextCheckAt       time.Time  `json:"next_check_at"`
	LastCheckedAt     *time.Time `json:"last_checked_at,omitempty"`
	LeaseAcquiredAt     *time.Time `json:"lease_acquired_at,omitempty"`
	LeaseWorkerID       *string    `json:"lease_worker_id,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	ConsecutiveFailures int        `json:"consecutive_failures"`

	// Joined metadata
	AssetName    string `json:"asset_name,omitempty"`
	AssetURL     string `json:"asset_url,omitempty"`
	ClientName   string `json:"client_name,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	EffectiveURL string `json:"effective_url,omitempty"`
}

// CheckResult represents the outcome of an executed check.
type CheckResult struct {
	ID               string     `json:"id"`
	OrganizationID   string     `json:"organization_id"`
	MonitorID        string     `json:"monitor_id"`
	ScheduledAt      time.Time  `json:"scheduled_at"`
	StartedAt        time.Time  `json:"started_at"`
	CompletedAt      time.Time  `json:"completed_at"`
	LatencyMs        int64      `json:"latency_ms"`
	Result           string     `json:"result"`
	HTTPStatusCode   *int       `json:"http_status_code,omitempty"`
	ErrorCategory    *string    `json:"error_category,omitempty"`
	ErrorMessage     *string    `json:"error_message,omitempty"`
	TLSVersion       *string    `json:"tls_version,omitempty"`
	TLSCipher        *string    `json:"tls_cipher,omitempty"`
	TLSCertIssuer    *string    `json:"tls_cert_issuer,omitempty"`
	TLSCertSubject   *string    `json:"tls_cert_subject,omitempty"`
	TLSCertNotBefore *time.Time `json:"tls_cert_not_before,omitempty"`
	TLSCertNotAfter  *time.Time `json:"tls_cert_not_after,omitempty"`
	TLSDaysRemaining *int       `json:"tls_days_remaining,omitempty"`
	TLSIsValid       *bool      `json:"tls_is_valid,omitempty"`
	ResolvedIP       *string    `json:"resolved_ip,omitempty"`
	WorkerID         string     `json:"worker_id"`
	ExecutionType    string     `json:"execution_type"`
	CreatedAt        time.Time  `json:"created_at"`
}

// ClaimedMonitorJob represents a monitor claimed by the scheduler for execution.
type ClaimedMonitorJob struct {
	ID                string     `json:"id"`
	OrganizationID    string     `json:"organization_id"`
	AssetID           string     `json:"asset_id"`
	TargetURL         string     `json:"target_url"`
	Method            string     `json:"method"`
	IntervalSeconds   int        `json:"interval_seconds"`
	TimeoutSeconds    int        `json:"timeout_seconds"`
	ExpectedStatusMin int        `json:"expected_status_min"`
	ExpectedStatusMax int        `json:"expected_status_max"`
	FollowRedirects   bool       `json:"follow_redirects"`
	MaxRedirects      int        `json:"max_redirects"`
	ExpectedText      *string    `json:"expected_text,omitempty"`
	JSONPointer       *string    `json:"json_pointer,omitempty"`
	JSONExpectedValue *string    `json:"json_expected_value,omitempty"`
	ScheduledAt       time.Time  `json:"scheduled_at"`
}

// DashboardStats aggregates high-level metrics for the active workspace.
type DashboardStats struct {
	TotalClients          int            `json:"total_clients"`
	TotalAssets           int            `json:"total_assets"`
	MonitorsUp            int            `json:"monitors_up"`
	MonitorsDown          int            `json:"monitors_down"`
	MonitorsSuspectedDown int            `json:"monitors_suspected_down"`
	MonitorsPending       int            `json:"monitors_pending"`
	MonitorsPaused        int            `json:"monitors_paused"`
	MonitorsUnknown       int            `json:"monitors_unknown"`
	OpenIncidents         int            `json:"open_incidents"`
	TotalHeartbeats       int            `json:"total_heartbeats"`
	HeartbeatsHealthy     int            `json:"heartbeats_healthy"`
	HeartbeatsMissed      int            `json:"heartbeats_missed"`
	HeartbeatsPending     int            `json:"heartbeats_pending"`
	RecentChecks          []*CheckResult `json:"recent_checks"`
}

// CheckAggregateHourly represents hourly aggregated check metrics.
type CheckAggregateHourly struct {
	ID               string    `json:"id"`
	OrganizationID   string    `json:"organization_id"`
	MonitorID        string    `json:"monitor_id"`
	BucketHour       time.Time `json:"bucket_hour"`
	TotalChecks      int       `json:"total_checks"`
	SuccessfulChecks int       `json:"successful_checks"`
	FailedChecks     int       `json:"failed_checks"`
	TotalLatencyMS   int64     `json:"total_latency_ms"`
	MinLatencyMS     int64     `json:"min_latency_ms"`
	MaxLatencyMS     int64     `json:"max_latency_ms"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// HeartbeatMonitor represents a cron, background worker, or backup heartbeat monitor.
type HeartbeatMonitor struct {
	ID                      string     `json:"id"`
	OrganizationID          string     `json:"organization_id"`
	ClientID                string     `json:"client_id"`
	AssetID                 *string    `json:"asset_id,omitempty"`
	Name                    string     `json:"name"`
	Description             *string    `json:"description,omitempty"`
	ExpectedIntervalSeconds int        `json:"expected_interval_seconds"`
	GracePeriodSeconds      int        `json:"grace_period_seconds"`
	Status                  string     `json:"status"`
	IsActive                bool       `json:"is_active"`
	TokenHash               string     `json:"-"`
	LastReceivedAt          *time.Time `json:"last_received_at,omitempty"`
	NextExpectedAt          *time.Time `json:"next_expected_at,omitempty"`
	LeaseAcquiredAt         *time.Time `json:"-"`
	LeaseWorkerID           *string    `json:"-"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
	ArchivedAt              *time.Time `json:"archived_at,omitempty"`

	// Joined metadata
	ClientName string  `json:"client_name,omitempty"`
	AssetName  *string `json:"asset_name,omitempty"`
}

// ClaimedHeartbeatJob represents an overdue heartbeat claimed by the scheduler for incident handling.
type ClaimedHeartbeatJob struct {
	ID                      string     `json:"id"`
	OrganizationID          string     `json:"organization_id"`
	ClientID                string     `json:"client_id"`
	AssetID                 *string    `json:"asset_id,omitempty"`
	Name                    string     `json:"name"`
	ExpectedIntervalSeconds int        `json:"expected_interval_seconds"`
	GracePeriodSeconds      int        `json:"grace_period_seconds"`
	Status                  string     `json:"status"`
	LastReceivedAt          *time.Time `json:"last_received_at,omitempty"`
	NextExpectedAt          *time.Time `json:"next_expected_at,omitempty"`
}

// Incident represents an operational outage or missed heartbeat.
type Incident struct {
	ID                   string     `json:"id"`
	OrganizationID       string     `json:"organization_id"`
	MonitorID            *string    `json:"monitor_id,omitempty"`
	HeartbeatMonitorID   *string    `json:"heartbeat_monitor_id,omitempty"`
	FirstFailureAt       time.Time  `json:"first_failure_at"`
	ConfirmedAt          time.Time  `json:"confirmed_at"`
	ResolvedAt           *time.Time `json:"resolved_at,omitempty"`
	Status               string     `json:"status"`
	FailureCategory      string     `json:"failure_category"`
	FirstFailureResultID *string    `json:"first_failure_result_id,omitempty"`
	RecoveryResultID     *string    `json:"recovery_result_id,omitempty"`
	AcknowledgedAt       *time.Time `json:"acknowledged_at,omitempty"`
	AcknowledgedBy       *string    `json:"acknowledged_by,omitempty"`
	NotificationState    string     `json:"notification_state"`
	DurationSeconds      *int       `json:"duration_seconds,omitempty"`
	Summary              *string    `json:"summary,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`

	// Joined metadata
	TargetType             string       `json:"target_type"` // "monitor" or "heartbeat"
	TargetName             string       `json:"target_name,omitempty"`
	ClientID               string       `json:"client_id,omitempty"`
	ClientName             string       `json:"client_name,omitempty"`
	AssetID                *string      `json:"asset_id,omitempty"`
	AssetName              *string      `json:"asset_name,omitempty"`
	AcknowledgedByName     *string      `json:"acknowledged_by_name,omitempty"`
	AcknowledgedByEmail    *string      `json:"acknowledged_by_email,omitempty"`
	FirstFailureResult     *CheckResult `json:"first_failure_result,omitempty"`
	RecoveryResult         *CheckResult `json:"recovery_result,omitempty"`
}

// NotificationEvent represents an immutable, persistent event ready for notification delivery.
type NotificationEvent struct {
	ID                 string                 `json:"id"`
	OrganizationID     string                 `json:"organization_id"`
	EventType          string                 `json:"event_type"`
	IncidentID         *string                `json:"incident_id,omitempty"`
	MonitorID          *string                `json:"monitor_id,omitempty"`
	HeartbeatMonitorID *string                `json:"heartbeat_monitor_id,omitempty"`
	ClientID           *string                `json:"client_id,omitempty"`
	IdempotencyKey     string                 `json:"idempotency_key"`
	Payload            map[string]interface{} `json:"payload"`
	Status             string                 `json:"status"`
	CreatedAt          time.Time              `json:"created_at"`
}

// MaintenanceActivity represents actual work performed by the agency for a client.
type MaintenanceActivity struct {
	ID              string    `json:"id"`
	OrganizationID  string    `json:"organization_id"`
	ClientID        string    `json:"client_id"`
	AssetID         *string   `json:"asset_id,omitempty"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Category        string    `json:"category"`
	PerformedAt     time.Time `json:"performed_at"`
	DurationMinutes *int      `json:"duration_minutes,omitempty"`
	CreatedBy       string    `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Joined metadata
	ClientName     string  `json:"client_name,omitempty"`
	AssetName      *string `json:"asset_name,omitempty"`
	CreatedByName  string  `json:"created_by_name,omitempty"`
	CreatedByEmail string  `json:"created_by_email,omitempty"`
}

// OrganizationBranding represents custom branding configurations for client care reports.
type OrganizationBranding struct {
	OrganizationID      string    `json:"organization_id"`
	DisplayName         string    `json:"display_name"`
	LogoStorageKey      *string   `json:"logo_storage_key,omitempty"`
	PrimaryColor        string    `json:"primary_color"`
	AccentColor         string    `json:"accent_color"`
	ReportFooterText    string    `json:"report_footer_text"`
	ContactEmail        string    `json:"contact_email"`
	WebsiteURL          string    `json:"website_url"`
	DefaultReportLocale string    `json:"default_report_locale"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// Report represents an immutable, versioned monthly care report snapshot.
type Report struct {
	ID                   string    `json:"id"`
	OrganizationID       string    `json:"organization_id"`
	ClientID             string    `json:"client_id"`
	ReportType           string    `json:"report_type"`
	PeriodStart          time.Time `json:"period_start"`
	PeriodEnd            time.Time `json:"period_end"`
	OrganizationTimezone string    `json:"organization_timezone"`
	Locale               string    `json:"locale"`
	VersionNumber        int       `json:"version_number"`
	TemplateVersion      string    `json:"template_version"`
	Title                string    `json:"title"`
	SnapshotJSON         []byte    `json:"snapshot_json"`
	PDFStorageKey        *string   `json:"pdf_storage_key,omitempty"`
	GeneratedBy          string    `json:"generated_by"`
	GeneratedAt          time.Time `json:"generated_at"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`

	// Joined metadata
	ClientName       string `json:"client_name,omitempty"`
	GeneratedByName  string `json:"generated_by_name,omitempty"`
	GeneratedByEmail string `json:"generated_by_email,omitempty"`
}

// ReportShareToken represents an unauthenticated, read-only token to securely view a report snapshot.
type ReportShareToken struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	ReportID       string     `json:"report_id"`
	TokenHash      string     `json:"-"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	CreatedBy      string     `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`

	// Metadata
	IsActive bool `json:"is_active"`
}

// PublicReportView represents the safe public projection returned when resolving a valid share token.
type PublicReportView struct {
	ReportID             string     `json:"report_id"`
	ReportType           string     `json:"report_type"`
	PeriodStart          time.Time  `json:"period_start"`
	PeriodEnd            time.Time  `json:"period_end"`
	OrganizationTimezone string     `json:"organization_timezone"`
	Locale               string     `json:"locale"`
	VersionNumber        int        `json:"version_number"`
	TemplateVersion      string     `json:"template_version"`
	Title                string     `json:"title"`
	SnapshotJSON         []byte     `json:"snapshot_json"`
	HasPDF               bool       `json:"has_pdf"`
	GeneratedAt          time.Time  `json:"generated_at"`
	ExpiresAt            *time.Time `json:"expires_at,omitempty"`
}



