package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/tenant"
)

type TenantMiddleware struct {
	orgRepo     *repository.OrganizationRepository
	sessionRepo *repository.SessionRepository
}

func NewTenantMiddleware(orgRepo *repository.OrganizationRepository, sessionRepo *repository.SessionRepository) *TenantMiddleware {
	return &TenantMiddleware{
		orgRepo:     orgRepo,
		sessionRepo: sessionRepo,
	}
}

// ResolveTenant extracts and validates tenant context for an authenticated user.
// It prioritizes the X-Organization-ID header, then falls back to session active org or first membership.
// Crucially, it ALWAYS validates membership in the database to prevent cross-tenant access.
func (m *TenantMiddleware) ResolveTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := tenant.GetUserContext(r.Context())
		if u == nil {
			next.ServeHTTP(w, r)
			return
		}

		targetOrgID := strings.TrimSpace(r.Header.Get("X-Organization-ID"))

		// If no header, check session's active org
		if targetOrgID == "" && u.SessionID != "" {
			sessions, err := m.sessionRepo.ListUserSessions(r.Context(), u.UserID)
			if err == nil {
				for _, s := range sessions {
					if s.ID == u.SessionID && s.ActiveOrganizationID != nil {
						targetOrgID = *s.ActiveOrganizationID
						break
					}
				}
			}
		}

		// Fallback to first available membership
		if targetOrgID == "" {
			memberships, err := m.orgRepo.ListUserMemberships(r.Context(), u.UserID)
			if err == nil && len(memberships) > 0 {
				targetOrgID = memberships[0].OrganizationID
			}
		}

		if targetOrgID != "" {
			// Mandatory server-side membership validation
			membership, err := m.orgRepo.GetMembership(r.Context(), targetOrgID, u.UserID)
			if err == nil {
				org, err := m.orgRepo.GetOrganizationByID(r.Context(), targetOrgID)
				if err == nil {
					tenantCtx := &tenant.TenantContext{
						OrganizationID: org.ID,
						Slug:           org.Slug,
						Name:           org.Name,
						Role:           membership.Role,
						Status:         org.Status,
					}
					ctx := tenant.WithTenantContext(r.Context(), tenantCtx)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// RequireTenant ensures that the request context has an active, validated organization.
func RequireTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := tenant.GetTenantContext(r.Context())
		if t == nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "NO_ACTIVE_ORGANIZATION",
					"message": "No active organization found or membership is invalid.",
				},
			})
			return
		}
		if t.Status == "suspended" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "ORGANIZATION_SUSPENDED",
					"message": "This organization has been suspended by platform administration. Access to organization resources is restricted.",
				},
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole ensures that the authenticated user possesses at least the specified role in the active organization.
func RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t := tenant.GetTenantContext(r.Context())
			if t == nil || !tenant.HasRole(t.Role, requiredRole) {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "FORBIDDEN",
						"message": "You do not have the necessary permissions to perform this action.",
					},
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

