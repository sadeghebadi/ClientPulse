package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"clientpulse/services/api/internal/config"
	"clientpulse/services/api/internal/health"
	"clientpulse/services/api/internal/rate"
	"clientpulse/services/api/internal/tenant"
)

// Dependencies bundles all injected dependencies for the HTTP server.
type Dependencies struct {
	HealthHdr        *health.Handler
	AuthHdr          *AuthHandler
	OrgHdr           *OrgHandler
	ClientHdr        *ClientHandler
	AssetHdr         *AssetHandler
	MonitorHdr       *MonitorHandler
	HeartbeatHdr     *HeartbeatHandler
	IncidentHdr      *IncidentHandler
	AuthMiddleware   *AuthMiddleware
	TenantMiddleware *TenantMiddleware
}

// Server encapsulates the HTTP server, configuration, and dependencies.
type Server struct {
	httpServer *http.Server
	cfg        *config.Config
	logger     *slog.Logger
	deps       Dependencies
}

// New constructs a configured Server instance.
func New(cfg *config.Config, log *slog.Logger, deps Dependencies) *Server {
	s := &Server{
		cfg:    cfg,
		logger: log,
		deps:   deps,
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	// Build global middleware chain
	var handler http.Handler = mux

	allowedOrigins := []string{cfg.AppBaseURL, "http://localhost:3000", "http://localhost:3001"}

	// CSRF validation for mutating cookie-authenticated requests (runs after AuthMiddleware)
	handler = CSRFProtection(cfg, allowedOrigins)(handler)

	// Global session authentication so user context is available for CSRF validation and route handlers
	if s.deps.AuthMiddleware != nil {
		handler = s.deps.AuthMiddleware.Authenticate(handler)
	}

	// CORS handler
	handler = CORS(allowedOrigins)(handler)
	handler = SecurityHeaders(handler)
	handler = RequestLogger(log)(handler)
	handler = Recovery(log)(handler)
	handler = RequestID(handler)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Rate limiters for sensitive endpoints
	signupLimiter := rate.NewLimiter(10, 1*time.Minute)
	loginLimiter := rate.NewLimiter(10, 1*time.Minute)
	forgotPasswordLimiter := rate.NewLimiter(5, 1*time.Minute)
	resetPasswordLimiter := rate.NewLimiter(10, 1*time.Minute)
	verifyEmailLimiter := rate.NewLimiter(15, 1*time.Minute)
	resendVerificationLimiter := rate.NewLimiter(5, 1*time.Minute)

	// 1. Health and probe endpoints (public)
	if s.deps.HealthHdr != nil {
		mux.HandleFunc("GET /health/live", s.deps.HealthHdr.Live)
		mux.HandleFunc("GET /health/ready", s.deps.HealthHdr.Ready)
		mux.HandleFunc("GET /health/worker", s.deps.HealthHdr.Worker)
	}

	// 2. Base info endpoint
	mux.HandleFunc("GET /api/v1/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"service":     "ClientPulse API",
			"version":     "1.0.0",
			"phase":       "Phase 1 Authentication & Multi-Tenancy",
			"environment": s.cfg.AppEnv,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	if s.deps.AuthHdr == nil || s.deps.OrgHdr == nil || s.deps.AuthMiddleware == nil || s.deps.TenantMiddleware == nil {
		return
	}

	// 3. Public Authentication Routes
	mux.Handle("POST /api/v1/auth/signup", RateLimit(signupLimiter, "signup")(http.HandlerFunc(s.deps.AuthHdr.Signup)))
	mux.Handle("POST /api/v1/auth/login", RateLimit(loginLimiter, "login")(http.HandlerFunc(s.deps.AuthHdr.Login)))
	mux.Handle("POST /api/v1/auth/forgot-password", RateLimit(forgotPasswordLimiter, "forgot-password")(http.HandlerFunc(s.deps.AuthHdr.ForgotPassword)))
	mux.Handle("POST /api/v1/auth/reset-password", RateLimit(resetPasswordLimiter, "reset-password")(http.HandlerFunc(s.deps.AuthHdr.ResetPassword)))
	mux.Handle("POST /api/v1/auth/verify-email", RateLimit(verifyEmailLimiter, "verify-email")(http.HandlerFunc(s.deps.AuthHdr.VerifyEmail)))

	// Helper wrapper for authenticated routes
	authed := func(h http.HandlerFunc) http.Handler {
		return s.deps.AuthMiddleware.Authenticate(RequireAuth(h))
	}

	// 4. Authenticated User Routes
	mux.Handle("POST /api/v1/auth/logout", authed(s.deps.AuthHdr.Logout))
	mux.Handle("GET /api/v1/auth/me", authed(s.deps.AuthHdr.Me))
	mux.Handle("POST /api/v1/auth/resend-verification", RateLimit(resendVerificationLimiter, "resend-verification")(authed(s.deps.AuthHdr.ResendVerification)))
	mux.Handle("POST /api/v1/auth/change-password", authed(s.deps.AuthHdr.ChangePassword))

	// 5. Organization Management (User Scope)
	mux.Handle("GET /api/v1/organizations", authed(s.deps.OrgHdr.ListUserOrganizations))
	mux.Handle("POST /api/v1/organizations", authed(s.deps.OrgHdr.CreateOrganization))
	mux.Handle("POST /api/v1/organizations/select", authed(s.deps.OrgHdr.SelectOrganization))

	// Helper wrapper for active tenant-scoped routes
	tenantScoped := func(h http.HandlerFunc, minRole string) http.Handler {
		chain := http.Handler(h)
		if minRole != "" {
			chain = RequireRole(minRole)(chain)
		}
		chain = RequireTenant(chain)
		chain = s.deps.TenantMiddleware.ResolveTenant(chain)
		chain = RequireAuth(chain)
		return s.deps.AuthMiddleware.Authenticate(chain)
	}

	// Helper wrapper for sensitive mutating tenant routes (requires email verification)
	sensitiveTenantScoped := func(h http.HandlerFunc, minRole string) http.Handler {
		chain := http.Handler(h)
		chain = RequireVerifiedEmail(chain)
		if minRole != "" {
			chain = RequireRole(minRole)(chain)
		}
		chain = RequireTenant(chain)
		chain = s.deps.TenantMiddleware.ResolveTenant(chain)
		chain = RequireAuth(chain)
		return s.deps.AuthMiddleware.Authenticate(chain)
	}

	// 6. Active Organization Scoped Routes
	mux.Handle("GET /api/v1/organizations/current", tenantScoped(s.deps.OrgHdr.GetCurrent, ""))
	mux.Handle("PATCH /api/v1/organizations/current", sensitiveTenantScoped(s.deps.OrgHdr.UpdateCurrent, tenant.RoleAdmin))
	mux.Handle("GET /api/v1/organizations/current/members", tenantScoped(s.deps.OrgHdr.ListMembers, ""))
	mux.Handle("POST /api/v1/organizations/current/members", sensitiveTenantScoped(s.deps.OrgHdr.AddMember, tenant.RoleAdmin))
	mux.Handle("PATCH /api/v1/organizations/current/members/{userID}", sensitiveTenantScoped(s.deps.OrgHdr.UpdateMemberRole, tenant.RoleOwner))
	mux.Handle("DELETE /api/v1/organizations/current/members/{userID}", sensitiveTenantScoped(s.deps.OrgHdr.RemoveMember, tenant.RoleAdmin))

	// 7. Client Management Routes
	if s.deps.ClientHdr != nil {
		mux.Handle("GET /api/v1/clients", tenantScoped(s.deps.ClientHdr.List, ""))
		mux.Handle("POST /api/v1/clients", sensitiveTenantScoped(s.deps.ClientHdr.Create, tenant.RoleMember))
		mux.Handle("GET /api/v1/clients/{id}", tenantScoped(s.deps.ClientHdr.Get, ""))
		mux.Handle("PATCH /api/v1/clients/{id}", sensitiveTenantScoped(s.deps.ClientHdr.Update, tenant.RoleMember))
		mux.Handle("POST /api/v1/clients/{id}/archive", sensitiveTenantScoped(s.deps.ClientHdr.Archive, tenant.RoleAdmin))
		mux.Handle("POST /api/v1/clients/{id}/restore", sensitiveTenantScoped(s.deps.ClientHdr.Restore, tenant.RoleAdmin))
		mux.Handle("DELETE /api/v1/clients/{id}", sensitiveTenantScoped(s.deps.ClientHdr.Delete, tenant.RoleOwner))
	}

	// 8. Asset Management Routes
	if s.deps.AssetHdr != nil {
		mux.Handle("GET /api/v1/assets", tenantScoped(s.deps.AssetHdr.List, ""))
		mux.Handle("POST /api/v1/assets", sensitiveTenantScoped(s.deps.AssetHdr.Create, tenant.RoleMember))
		mux.Handle("GET /api/v1/assets/{id}", tenantScoped(s.deps.AssetHdr.Get, ""))
		mux.Handle("PATCH /api/v1/assets/{id}", sensitiveTenantScoped(s.deps.AssetHdr.Update, tenant.RoleMember))
		mux.Handle("POST /api/v1/assets/{id}/archive", sensitiveTenantScoped(s.deps.AssetHdr.Archive, tenant.RoleAdmin))
		mux.Handle("POST /api/v1/assets/{id}/restore", sensitiveTenantScoped(s.deps.AssetHdr.Restore, tenant.RoleAdmin))
	}

	// 9. Monitor Management & Dashboard Routes
	if s.deps.MonitorHdr != nil {
		mux.Handle("GET /api/v1/monitors", tenantScoped(s.deps.MonitorHdr.List, ""))
		mux.Handle("POST /api/v1/monitors", sensitiveTenantScoped(s.deps.MonitorHdr.Create, tenant.RoleMember))
		mux.Handle("GET /api/v1/monitors/{id}", tenantScoped(s.deps.MonitorHdr.Get, ""))
		mux.Handle("PATCH /api/v1/monitors/{id}", sensitiveTenantScoped(s.deps.MonitorHdr.Update, tenant.RoleMember))
		mux.Handle("POST /api/v1/monitors/{id}/pause", sensitiveTenantScoped(s.deps.MonitorHdr.Pause, tenant.RoleMember))
		mux.Handle("POST /api/v1/monitors/{id}/resume", sensitiveTenantScoped(s.deps.MonitorHdr.Resume, tenant.RoleMember))
		mux.Handle("DELETE /api/v1/monitors/{id}", sensitiveTenantScoped(s.deps.MonitorHdr.Delete, tenant.RoleAdmin))
		mux.Handle("POST /api/v1/monitors/{id}/test", sensitiveTenantScoped(s.deps.MonitorHdr.TestNow, tenant.RoleMember))
		mux.Handle("GET /api/v1/monitors/{id}/history", tenantScoped(s.deps.MonitorHdr.History, ""))
		mux.Handle("GET /api/v1/dashboard/stats", tenantScoped(s.deps.MonitorHdr.DashboardStats, ""))
	}

	// 10. Heartbeat Public Ping Endpoint (Unauthenticated, rate limited)
	if s.deps.HeartbeatHdr != nil {
		mux.HandleFunc("POST /api/v1/heartbeat-pings/{token}", s.deps.HeartbeatHdr.Ping)
	}

	// 11. Heartbeat Management Routes (Tenant-scoped)
	if s.deps.HeartbeatHdr != nil {
		mux.Handle("GET /api/v1/heartbeats", tenantScoped(s.deps.HeartbeatHdr.List, ""))
		mux.Handle("POST /api/v1/heartbeats", sensitiveTenantScoped(s.deps.HeartbeatHdr.Create, tenant.RoleMember))
		mux.Handle("GET /api/v1/heartbeats/{id}", tenantScoped(s.deps.HeartbeatHdr.Get, ""))
		mux.Handle("PATCH /api/v1/heartbeats/{id}", sensitiveTenantScoped(s.deps.HeartbeatHdr.Update, tenant.RoleMember))
		mux.Handle("POST /api/v1/heartbeats/{id}/pause", sensitiveTenantScoped(s.deps.HeartbeatHdr.Pause, tenant.RoleMember))
		mux.Handle("POST /api/v1/heartbeats/{id}/resume", sensitiveTenantScoped(s.deps.HeartbeatHdr.Resume, tenant.RoleMember))
		mux.Handle("POST /api/v1/heartbeats/{id}/rotate-token", sensitiveTenantScoped(s.deps.HeartbeatHdr.RotateToken, tenant.RoleAdmin))
		mux.Handle("POST /api/v1/heartbeats/{id}/revoke-token", sensitiveTenantScoped(s.deps.HeartbeatHdr.RevokeToken, tenant.RoleAdmin))
		mux.Handle("DELETE /api/v1/heartbeats/{id}", sensitiveTenantScoped(s.deps.HeartbeatHdr.Delete, tenant.RoleAdmin))
	}

	// 12. Incident Management Routes (Tenant-scoped)
	if s.deps.IncidentHdr != nil {
		mux.Handle("GET /api/v1/incidents", tenantScoped(s.deps.IncidentHdr.List, ""))
		mux.Handle("GET /api/v1/incidents/{id}", tenantScoped(s.deps.IncidentHdr.Get, ""))
		mux.Handle("POST /api/v1/incidents/{id}/acknowledge", sensitiveTenantScoped(s.deps.IncidentHdr.Acknowledge, tenant.RoleMember))
		mux.Handle("GET /api/v1/monitors/{id}/uptime", tenantScoped(s.deps.IncidentHdr.Uptime, ""))
	}
}

// Start listens and serves incoming HTTP requests.
func (s *Server) Start() error {
	s.logger.Info("starting HTTP API server",
		"port", s.cfg.Port,
		"env", s.cfg.AppEnv,
		"base_url", s.cfg.APIBaseURL,
	)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server failure: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down HTTP API server")
	return s.httpServer.Shutdown(ctx)
}
