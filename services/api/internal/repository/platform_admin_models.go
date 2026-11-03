package repository

import (
	"time"
)

// PlatformAdminGrant represents a verified global administrative authorization.
type PlatformAdminGrant struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Role      string     `json:"role"`
	GrantedAt time.Time  `json:"granted_at"`
	GrantedBy *string    `json:"granted_by,omitempty"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	RevokedBy *string    `json:"revoked_by,omitempty"`
	Reason    *string    `json:"reason,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// SubscriptionPlan represents a high-level commercial or operational plan definition.
type SubscriptionPlan struct {
	ID                        string                   `json:"id"`
	Name                      string                   `json:"name"`
	Slug                      string                   `json:"slug"`
	Description               string                   `json:"description"`
	Status                    string                   `json:"status"` // 'draft', 'active', 'archived'
	CreatedAt                 time.Time                `json:"created_at"`
	UpdatedAt                 time.Time                `json:"updated_at"`
	CreatedBy                 *string                  `json:"created_by,omitempty"`
	UpdatedBy                 *string                  `json:"updated_by,omitempty"`
	LatestVersion             *SubscriptionPlanVersion `json:"latest_version,omitempty"`
	ActiveSubscriptionsCount int                      `json:"active_subscriptions_count"`
	VersionsCount             int                      `json:"versions_count"`
}

// PlanLimits holds structured limits and entitlements for a specific plan version.
type PlanLimits struct {
	MaxClients              int  `json:"max_clients"`
	MaxAssets               int  `json:"max_assets"`
	MaxMonitors             int  `json:"max_monitors"`
	MaxHeartbeats           int  `json:"max_heartbeats"`
	MaxMembers              int  `json:"max_members"`
	MinCheckIntervalSeconds int  `json:"min_check_interval_seconds"`
	RetentionDays           int  `json:"retention_days"`
	CustomBrandingEnabled   bool `json:"custom_branding_enabled"`
	PublicReportsEnabled    bool `json:"public_reports_enabled"`
}

// SubscriptionPlanVersion represents an immutable snapshot of plan limits and pricing.
type SubscriptionPlanVersion struct {
	ID                      string                 `json:"id"`
	PlanID                  string                 `json:"plan_id"`
	Version                 int                    `json:"version"`
	Status                  string                 `json:"status"` // 'draft', 'published', 'archived'
	PriceAmount             int64                  `json:"price_amount"` // minor currency units
	Currency                string                 `json:"currency"`     // ISO 4217 uppercase 3-letter code
	BillingInterval         string                 `json:"billing_interval"`
	MaxClients              int                    `json:"max_clients"`
	MaxAssets               int                    `json:"max_assets"`
	MaxMonitors             int                    `json:"max_monitors"`
	MaxHeartbeats           int                    `json:"max_heartbeats"`
	MaxMembers              int                    `json:"max_members"`
	MinCheckIntervalSeconds int                    `json:"min_check_interval_seconds"`
	RetentionDays           int                    `json:"retention_days"`
	CustomBrandingEnabled   bool                   `json:"custom_branding_enabled"`
	PublicReportsEnabled    bool                   `json:"public_reports_enabled"`
	Features                map[string]interface{} `json:"features"`
	CreatedAt               time.Time              `json:"created_at"`
	UpdatedAt               time.Time              `json:"updated_at"`
	PublishedAt             *time.Time             `json:"published_at,omitempty"`
	PublishedBy             *string                `json:"published_by,omitempty"`
}

// OrganizationSubscription represents a tenant's subscription to an exact plan version.
type OrganizationSubscription struct {
	ID                 string     `json:"id"`
	OrganizationID     string     `json:"organization_id"`
	OrganizationName   string     `json:"organization_name,omitempty"`
	PlanVersionID      string     `json:"plan_version_id"`
	PlanID             string     `json:"plan_id,omitempty"`
	PlanName           string     `json:"plan_name,omitempty"`
	PlanSlug           string     `json:"plan_slug,omitempty"`
	PlanVersion        int        `json:"plan_version,omitempty"`
	Status             string     `json:"status"` // 'active', 'suspended', 'canceled', 'expired'
	StartedAt          time.Time  `json:"started_at"`
	EndsAt             *time.Time `json:"ends_at,omitempty"`
	AssignedBy         *string    `json:"assigned_by,omitempty"`
	AdministrativeNote string     `json:"administrative_note"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	PriceAmount        int64      `json:"price_amount,omitempty"`
	Currency           string     `json:"currency,omitempty"`
	BillingInterval    string     `json:"billing_interval,omitempty"`
	Limits             PlanLimits `json:"limits"`
}

// PlatformOverviewStats represents platform-wide aggregated counters.
type PlatformOverviewStats struct {
	TotalUsers              int64 `json:"total_users"`
	ActiveUsers             int64 `json:"active_users"`
	SuspendedUsers          int64 `json:"suspended_users"`
	TotalOrganizations      int64 `json:"total_organizations"`
	ActiveOrganizations     int64 `json:"active_organizations"`
	SuspendedOrganizations  int64 `json:"suspended_organizations"`
	PublishedPlans          int64 `json:"published_plans"`
	ActiveSubscriptions     int64 `json:"active_subscriptions"`
	UnassignedOrganizations int64 `json:"unassigned_organizations"`
}

// PlatformUserItem represents a user in the Platform Admin user list.
type PlatformUserItem struct {
	ID                 string    `json:"id"`
	Email              string    `json:"email"`
	DisplayName        string    `json:"display_name"`
	Status             string    `json:"status"`
	EmailVerified      bool      `json:"email_verified"`
	CreatedAt          time.Time `json:"created_at"`
	OrganizationsCount int       `json:"organizations_count"`
	IsPlatformAdmin    bool      `json:"is_platform_admin"`
}

// PlatformUserMembership represents an organization relationship for a user.
type PlatformUserMembership struct {
	OrganizationID     string    `json:"organization_id"`
	OrganizationName   string    `json:"organization_name"`
	OrganizationSlug   string    `json:"organization_slug"`
	OrganizationStatus string    `json:"organization_status"`
	Role               string    `json:"role"`
	JoinedAt           time.Time `json:"joined_at"`
}

// PlatformUserDetail contains a user profile and all organizational memberships.
type PlatformUserDetail struct {
	User            User                     `json:"user"`
	IsPlatformAdmin bool                     `json:"is_platform_admin"`
	Memberships     []PlatformUserMembership `json:"memberships"`
}

// PlatformOrganizationItem represents an organization in the platform admin table.
type PlatformOrganizationItem struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Slug               string    `json:"slug"`
	Status             string    `json:"status"`
	Locale             string    `json:"locale"`
	Timezone           string    `json:"timezone"`
	CreatedAt          time.Time `json:"created_at"`
	OwnerID            *string   `json:"owner_id,omitempty"`
	OwnerEmail         *string   `json:"owner_email,omitempty"`
	OwnerName          *string   `json:"owner_name,omitempty"`
	MemberCount        int64     `json:"member_count"`
	ClientCount        int64     `json:"client_count"`
	AssetCount         int64     `json:"asset_count"`
	MonitorCount       int64     `json:"monitor_count"`
	HeartbeatCount     int64     `json:"heartbeat_count"`
	PlanName           *string   `json:"plan_name,omitempty"`
	PlanSlug           *string   `json:"plan_slug,omitempty"`
	PlanVersion        *int      `json:"plan_version,omitempty"`
	SubscriptionStatus *string   `json:"subscription_status,omitempty"`
}

// PlatformOrgMember represents a member within an organization for the admin panel.
type PlatformOrgMember struct {
	UserID      string    `json:"user_id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	UserStatus  string    `json:"user_status"`
	Role        string    `json:"role"`
	JoinedAt    time.Time `json:"joined_at"`
}

// PlatformOrganizationUsage represents actual resource consumption.
type PlatformOrganizationUsage struct {
	ClientCount    int64 `json:"client_count"`
	AssetCount     int64 `json:"asset_count"`
	MonitorCount   int64 `json:"monitor_count"`
	HeartbeatCount int64 `json:"heartbeat_count"`
	MemberCount    int64 `json:"member_count"`
}

// PlatformOrganizationDetail combines org metadata, members, usage, and subscription.
type PlatformOrganizationDetail struct {
	Organization Organization              `json:"organization"`
	Owner        *PlatformOrgMember        `json:"owner,omitempty"`
	Members      []PlatformOrgMember       `json:"members"`
	Usage        PlatformOrganizationUsage `json:"usage"`
	Subscription *OrganizationSubscription `json:"subscription,omitempty"`
	OverLimits   []string                  `json:"over_limits,omitempty"`
}

// PlatformAdminAuditLog represents an immutable record in the administrative log.
type PlatformAdminAuditLog struct {
	ID           string                 `json:"id"`
	ActorUserID  string                 `json:"actor_user_id"`
	ActorEmail   string                 `json:"actor_email"`
	ActorName    string                 `json:"actor_name"`
	Action       string                 `json:"action"`
	TargetType   string                 `json:"target_type"`
	TargetID     string                 `json:"target_id"`
	Reason       string                 `json:"reason"`
	MetadataJSON map[string]interface{} `json:"metadata"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	CreatedAt    time.Time              `json:"created_at"`
}
