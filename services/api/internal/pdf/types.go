package pdf

import "time"

// ReportData represents all snapshot data needed to generate a Monthly Website Care Report PDF.
type ReportData struct {
	ID                   string                   `json:"id,omitempty"`
	ReportType           string                   `json:"report_type"`
	Locale               string                   `json:"locale"` // "en" or "fa"
	Title                string                   `json:"title"`
	PeriodStart          time.Time                `json:"period_start"`
	PeriodEnd            time.Time                `json:"period_end"`
	OrganizationTimezone string                   `json:"organization_timezone"`
	GeneratedAt          time.Time                `json:"generated_at"`
	GeneratedByEmail     string                   `json:"generated_by_email,omitempty"`
	VersionNumber        int                      `json:"version_number"`
	TemplateVersion      string                   `json:"template_version"`
	Branding             BrandingData             `json:"branding"`
	Client               ClientData               `json:"client"`
	Summary              SummaryData              `json:"summary"`
	Assets               []AssetReportData        `json:"assets"`
	Monitors             []MonitorReportData      `json:"monitors"`
	Incidents            []IncidentReportData     `json:"incidents"`
	Heartbeats           []HeartbeatReportData    `json:"heartbeats"`
	Maintenance          []MaintenanceReportData  `json:"maintenance"`
	Recommendations      []RecommendationData     `json:"recommendations"`
	LimitationNotice     string                   `json:"limitation_notice"`
}

type BrandingData struct {
	DisplayName     string `json:"display_name"`
	LogoDataURI     string `json:"logo_data_uri,omitempty"` // data:image/png;base64,...
	PrimaryColor    string `json:"primary_color"`
	AccentColor     string `json:"accent_color"`
	FooterText      string `json:"footer_text"`
	ContactEmail    string `json:"contact_email"`
	WebsiteURL      string `json:"website_url"`
}

type ClientData struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug,omitempty"`
	Contact   string `json:"contact,omitempty"`
	Email     string `json:"email,omitempty"`
}

type SummaryData struct {
	TotalMonitors       int      `json:"total_monitors"`
	AverageUptime       *float64 `json:"average_uptime"`
	TotalIncidents      int      `json:"total_incidents"`
	TotalDowntimeSec    int64    `json:"total_downtime_sec"`
	TotalActivities     int      `json:"total_activities"`
	TotalMaintenanceMin int      `json:"total_maintenance_min"`
	OverallStatus       string   `json:"overall_status"`
	ExecutiveStatement  string   `json:"executive_statement"`
}

type AssetReportData struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	URL            string `json:"url"`
	Environment    string `json:"environment"`
	Status         string `json:"status"`
	SSLStatus      string `json:"ssl_status,omitempty"`
	SSLExpiryDate  string `json:"ssl_expiry_date,omitempty"`
}

type MonitorReportData struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	AssetName           string   `json:"asset_name"`
	URL                 string   `json:"url"`
	UptimePercentage    *float64 `json:"uptime_percentage"`
	CoveragePercentage  float64  `json:"coverage_percentage"`
	InsufficientData    bool     `json:"insufficient_data"`
	AvgResponseTimeMs   *int64   `json:"avg_response_time_ms,omitempty"`
	MedianResponseTimeMs *int64  `json:"median_response_time_ms,omitempty"`
	TotalChecks         int      `json:"total_checks"`
	FailedChecks        int      `json:"failed_checks"`
	IncidentCount       int      `json:"incident_count"`
	EligibleDurationSec int64    `json:"eligible_duration_sec"`
	ObservedDurationSec int64    `json:"observed_duration_sec"`
	UnknownDurationSec  int64    `json:"unknown_duration_sec"`
	LimitationNotice    string   `json:"limitation_notice"`
}

type IncidentReportData struct {
	ID              string  `json:"id"`
	TargetName      string  `json:"target_name"`
	TargetType      string  `json:"target_type"` // "monitor" or "heartbeat"
	ConfirmedAt     string  `json:"confirmed_at"`
	ResolvedAt      string  `json:"resolved_at,omitempty"`
	DurationMinutes int64   `json:"duration_minutes"`
	FailureCategory string  `json:"failure_category"`
	Summary         string  `json:"summary"`
}

type HeartbeatReportData struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	AssetName        string `json:"asset_name,omitempty"`
	Status           string `json:"status"` // "healthy", "missed", "pending", "paused"
	ExpectedInterval string `json:"expected_interval"`
	LastReceivedAt   string `json:"last_received_at,omitempty"`
	LimitationNotice string `json:"limitation_notice"`
}

type MaintenanceReportData struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Category        string `json:"category"`
	PerformedAt     string `json:"performed_at"`
	DurationMinutes *int   `json:"duration_minutes,omitempty"`
	AssetName       string `json:"asset_name,omitempty"`
	Description     string `json:"description"`
}

type RecommendationData struct {
	Severity    string `json:"severity"` // "info", "warning", "critical"
	Title       string `json:"title"`
	Description string `json:"description"`
}

