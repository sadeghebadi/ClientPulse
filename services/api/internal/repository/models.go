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

