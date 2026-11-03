package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"clientpulse/services/api/internal/auth"
	"clientpulse/services/api/internal/config"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/tenant"
)

const (
	SessionCookieName = "clientpulse_session"
	CSRFCookieName    = "clientpulse_csrf"
)

type AuthMiddleware struct {
	sessionRepo       *repository.SessionRepository
	userRepo          *repository.UserRepository
	platformAdminRepo *repository.PlatformAdminRepository
	cfg               *config.Config
}

func NewAuthMiddleware(sessionRepo *repository.SessionRepository, userRepo *repository.UserRepository, cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{
		sessionRepo: sessionRepo,
		userRepo:    userRepo,
		cfg:         cfg,
	}
}

// WithPlatformAdminRepo configures platform admin repository for global permission checks.
func (m *AuthMiddleware) WithPlatformAdminRepo(repo *repository.PlatformAdminRepository) *AuthMiddleware {
	m.platformAdminRepo = repo
	return m
}

// Authenticate extracts session token from cookie or Authorization header,
// resolves the user, and attaches UserContext to the request context.
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawToken := ""

		// 1. Check HttpOnly cookie
		if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
			rawToken = cookie.Value
		}

		// 2. Fallback to Authorization: Bearer <token>
		if rawToken == "" {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				rawToken = strings.TrimSpace(authHeader[7:])
			}
		}

		if rawToken == "" {
			next.ServeHTTP(w, r)
			return
		}

		tokenHash := auth.HashToken(rawToken, m.cfg.TokenHashPepper)
		session, err := m.sessionRepo.GetSessionByTokenHash(r.Context(), tokenHash)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		user, err := m.userRepo.GetUserByID(r.Context(), session.UserID)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		// Update last seen
		_ = m.sessionRepo.TouchSession(r.Context(), session.ID)

		isAdmin := false
		if m.platformAdminRepo != nil {
			isAdmin, _ = m.platformAdminRepo.IsUserPlatformAdmin(r.Context(), user.ID)
		}

		userCtx := &tenant.UserContext{
			UserID:          user.ID,
			Email:           user.Email,
			DisplayName:     user.DisplayName,
			EmailVerified:   user.EmailVerifiedAt != nil,
			SessionID:       session.ID,
			IsPlatformAdmin: isAdmin,
			Status:          user.Status,
		}

		ctx := tenant.WithUserContext(r.Context(), userCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuth middleware ensures the request has an authenticated user.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := tenant.GetUserContext(r.Context())
		if u == nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "UNAUTHORIZED",
					"message": "Authentication is required to access this resource.",
				},
			})
			return
		}
		if u.Status == "suspended" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "USER_SUSPENDED",
					"message": "Your account has been suspended by platform administration.",
				},
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireVerifiedEmail ensures that the authenticated user has verified their email address
// before executing sensitive operational actions (PRD FR-AUTH-01).
func RequireVerifiedEmail(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := tenant.GetUserContext(r.Context())
		if u == nil {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "UNAUTHORIZED",
					"message": "Authentication is required to access this resource.",
				},
			})
			return
		}

		if !u.EmailVerified {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "EMAIL_NOT_VERIFIED",
					"message": "Email address must be verified before performing sensitive operations.",
				},
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}
