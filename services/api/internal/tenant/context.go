package tenant

import (
	"context"
)

// Role constants
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleViewer = "viewer"
)

// UserContext carries the authenticated user's identity.
type UserContext struct {
	UserID          string
	Email           string
	DisplayName     string
	EmailVerified   bool
	SessionID       string
	IsPlatformAdmin bool
	Status          string
}

// TenantContext carries the verified active organization and user's role within it.
type TenantContext struct {
	OrganizationID string
	Slug           string
	Name           string
	Role           string
	Status         string
}

type userContextKeyType struct{}
type tenantContextKeyType struct{}

var (
	userContextKey   = userContextKeyType{}
	tenantContextKey = tenantContextKeyType{}
)

// WithUserContext returns a context enriched with user identity.
func WithUserContext(ctx context.Context, u *UserContext) context.Context {
	return context.WithValue(ctx, userContextKey, u)
}

// GetUserContext extracts the user identity from context, or nil if unauthenticated.
func GetUserContext(ctx context.Context) *UserContext {
	if val, ok := ctx.Value(userContextKey).(*UserContext); ok {
		return val
	}
	return nil
}

// WithTenantContext returns a context enriched with verified tenant identity and role.
func WithTenantContext(ctx context.Context, t *TenantContext) context.Context {
	return context.WithValue(ctx, tenantContextKey, t)
}

// GetTenantContext extracts the tenant context, or nil if outside tenant scope.
func GetTenantContext(ctx context.Context) *TenantContext {
	if val, ok := ctx.Value(tenantContextKey).(*TenantContext); ok {
		return val
	}
	return nil
}

// HasRole checks whether the role has at least the permissions of the required role.
// Hierarchy: owner > admin > member > viewer
func HasRole(actualRole, requiredRole string) bool {
	weight := map[string]int{
		RoleViewer: 1,
		RoleMember: 2,
		RoleAdmin:  3,
		RoleOwner:  4,
	}

	actualWeight, ok1 := weight[actualRole]
	reqWeight, ok2 := weight[requiredRole]
	if !ok1 || !ok2 {
		return false
	}

	return actualWeight >= reqWeight
}

