package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"clientpulse/services/api/internal/config"
	"clientpulse/services/api/internal/rate"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/tenant"
)

type HeartbeatHandler struct {
	repo        *repository.HeartbeatRepository
	cfg         *config.Config
	logger      *slog.Logger
	pingLimiter *rate.Limiter
}

func NewHeartbeatHandler(repo *repository.HeartbeatRepository, cfg *config.Config, logger *slog.Logger) *HeartbeatHandler {
	if logger == nil {
		logger = slog.Default()
	}
	// 60 pings/min per IP limit for public endpoint
	return &HeartbeatHandler{
		repo:        repo,
		cfg:         cfg,
		logger:      logger,
		pingLimiter: rate.NewLimiter(60, 1*time.Minute),
	}
}

// Ping processes an unauthenticated public heartbeat ping: POST /api/v1/heartbeat-pings/{token}
func (h *HeartbeatHandler) Ping(w http.ResponseWriter, r *http.Request) {
	// Security: Strict POST only
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		sendJSONError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Only POST is allowed for heartbeat pings")
		return
	}

	// Abuse protection: IP rate limiting
	clientIP := getClientIP(r)
	if !h.pingLimiter.Allow("ping:" + clientIP) {
		h.logger.Warn("heartbeat ping rate limit exceeded", "ip", clientIP)
		sendJSONError(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Rate limit exceeded. Please retry later.")
		return
	}

	// Payload restriction: max 1KB, empty or ignored
	r.Body = http.MaxBytesReader(w, r.Body, 1024)

	token := strings.TrimSpace(r.PathValue("token"))
	if len(token) < 16 {
		sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Heartbeat target not found")
		return
	}

	ctx, cancel := contextWithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	hb, resolvedInc, err := h.repo.RecordHeartbeatPing(ctx, token)
	if err != nil {
		if errors.Is(err, repository.ErrHeartbeatNotFound) ||
			errors.Is(err, repository.ErrHeartbeatRevoked) ||
			errors.Is(err, repository.ErrHeartbeatArchived) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Heartbeat target not found or inactive")
			return
		}
		h.logger.Error("failed to record heartbeat ping", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to process heartbeat ping")
		return
	}

	if resolvedInc != nil {
		h.logger.Info("heartbeat recovered and incident resolved",
			"heartbeat_id", hb.ID,
			"incident_id", resolvedInc.ID,
			"duration_sec", resolvedInc.DurationSeconds,
		)
	}

	// Non-cacheable, minimal response without exposing sensitive tenant information
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// Create handles POST /api/v1/heartbeats (Tenant-scoped, RoleMember)
func (h *HeartbeatHandler) Create(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())

	var req struct {
		ClientID                string  `json:"client_id"`
		AssetID                 *string `json:"asset_id"`
		Name                    string  `json:"name"`
		Description             *string `json:"description"`
		ExpectedIntervalSeconds int     `json:"expected_interval_seconds"`
		GracePeriodSeconds      int     `json:"grace_period_seconds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON payload")
		return
	}

	if strings.TrimSpace(req.ClientID) == "" {
		sendJSONError(w, http.StatusBadRequest, "VALIDATION_FAILED", "client_id is required")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		sendJSONError(w, http.StatusBadRequest, "VALIDATION_FAILED", "name is required")
		return
	}
	if req.ExpectedIntervalSeconds <= 0 {
		req.ExpectedIntervalSeconds = 3600 // 1 hour default
	}

	hb, plaintextToken, err := h.repo.CreateHeartbeat(r.Context(), tc.OrganizationID, repository.CreateHeartbeatParams{
		ClientID:                req.ClientID,
		AssetID:                 req.AssetID,
		Name:                    req.Name,
		Description:             req.Description,
		ExpectedIntervalSeconds: req.ExpectedIntervalSeconds,
		GracePeriodSeconds:      req.GracePeriodSeconds,
	})
	if err != nil {
		sendJSONError(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}

	pingURL := fmt.Sprintf("%s/api/v1/heartbeat-pings/%s", h.cfg.APIBaseURL, plaintextToken)
	bashSnippet := fmt.Sprintf("curl --fail --silent --show-error --max-time 10 -X POST \"%s\"", pingURL)
	powershellSnippet := fmt.Sprintf("Invoke-RestMethod -Uri \"%s\" -Method Post -TimeoutSec 10", pingURL)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"heartbeat":          hb,
		"token":              plaintextToken,
		"ping_url":           pingURL,
		"bash_snippet":       bashSnippet,
		"powershell_snippet": powershellSnippet,
	})
}

// List handles GET /api/v1/heartbeats (Tenant-scoped)
func (h *HeartbeatHandler) List(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())

	filter := repository.HeartbeatFilter{
		ClientID: r.URL.Query().Get("client_id"),
		AssetID:  r.URL.Query().Get("asset_id"),
		Status:   r.URL.Query().Get("status"),
		Search:   r.URL.Query().Get("search"),
	}

	list, total, err := h.repo.ListHeartbeats(r.Context(), tc.OrganizationID, filter)
	if err != nil {
		h.logger.Error("failed to list heartbeats", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve heartbeats")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"heartbeats": list,
		"total":      total,
	})
}

// Get handles GET /api/v1/heartbeats/{id} (Tenant-scoped)
func (h *HeartbeatHandler) Get(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	hb, err := h.repo.GetHeartbeatByID(r.Context(), tc.OrganizationID, id)
	if err != nil {
		if errors.Is(err, repository.ErrHeartbeatNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Heartbeat monitor not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve heartbeat")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"heartbeat": hb,
	})
}

// Update handles PATCH /api/v1/heartbeats/{id} (Tenant-scoped, RoleMember)
func (h *HeartbeatHandler) Update(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	var req struct {
		Name                    string  `json:"name"`
		Description             *string `json:"description"`
		AssetID                 *string `json:"asset_id"`
		ExpectedIntervalSeconds int     `json:"expected_interval_seconds"`
		GracePeriodSeconds      int     `json:"grace_period_seconds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON payload")
		return
	}

	hb, err := h.repo.UpdateHeartbeat(r.Context(), tc.OrganizationID, id, repository.UpdateHeartbeatParams{
		Name:                    req.Name,
		Description:             req.Description,
		AssetID:                 req.AssetID,
		ExpectedIntervalSeconds: req.ExpectedIntervalSeconds,
		GracePeriodSeconds:      req.GracePeriodSeconds,
	})
	if err != nil {
		if errors.Is(err, repository.ErrHeartbeatNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Heartbeat monitor not found")
			return
		}
		sendJSONError(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"heartbeat": hb,
	})
}

// RotateToken handles POST /api/v1/heartbeats/{id}/rotate-token (Tenant-scoped, RoleAdmin)
func (h *HeartbeatHandler) RotateToken(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	newToken, err := h.repo.RotateToken(r.Context(), tc.OrganizationID, id)
	if err != nil {
		if errors.Is(err, repository.ErrHeartbeatNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Heartbeat monitor not found")
			return
		}
		if errors.Is(err, repository.ErrHeartbeatArchived) {
			sendJSONError(w, http.StatusBadRequest, "ARCHIVED", "Cannot rotate token for archived heartbeat")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to rotate token")
		return
	}

	pingURL := fmt.Sprintf("%s/api/v1/heartbeat-pings/%s", h.cfg.APIBaseURL, newToken)
	bashSnippet := fmt.Sprintf("curl --fail --silent --show-error --max-time 10 -X POST \"%s\"", pingURL)
	powershellSnippet := fmt.Sprintf("Invoke-RestMethod -Uri \"%s\" -Method Post -TimeoutSec 10", pingURL)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"token":              newToken,
		"ping_url":           pingURL,
		"bash_snippet":       bashSnippet,
		"powershell_snippet": powershellSnippet,
	})
}

// RevokeToken handles POST /api/v1/heartbeats/{id}/revoke-token (Tenant-scoped, RoleAdmin)
func (h *HeartbeatHandler) RevokeToken(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	if err := h.repo.RevokeToken(r.Context(), tc.OrganizationID, id); err != nil {
		if errors.Is(err, repository.ErrHeartbeatNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Heartbeat monitor not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to revoke token")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "revoked",
	})
}

// Pause handles POST /api/v1/heartbeats/{id}/pause (Tenant-scoped, RoleMember)
func (h *HeartbeatHandler) Pause(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	if err := h.repo.PauseHeartbeat(r.Context(), tc.OrganizationID, id); err != nil {
		if errors.Is(err, repository.ErrHeartbeatNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Heartbeat monitor not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to pause heartbeat")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "paused"})
}

// Resume handles POST /api/v1/heartbeats/{id}/resume (Tenant-scoped, RoleMember)
func (h *HeartbeatHandler) Resume(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	if err := h.repo.ResumeHeartbeat(r.Context(), tc.OrganizationID, id); err != nil {
		if errors.Is(err, repository.ErrHeartbeatNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Heartbeat monitor not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to resume heartbeat")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "resumed"})
}

// Delete handles DELETE /api/v1/heartbeats/{id} (Tenant-scoped, RoleAdmin)
func (h *HeartbeatHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	if err := h.repo.ArchiveHeartbeatWithActor(r.Context(), tc.OrganizationID, id, uc.UserID); err != nil {
		if errors.Is(err, repository.ErrHeartbeatNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Heartbeat monitor not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to archive heartbeat")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "archived"})
}

func contextWithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, d)
}
