package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/tenant"
)

type MaintenanceHandler struct {
	repo   *repository.MaintenanceRepository
	logger *slog.Logger
}

func NewMaintenanceHandler(repo *repository.MaintenanceRepository, logger *slog.Logger) *MaintenanceHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &MaintenanceHandler{
		repo:   repo,
		logger: logger,
	}
}

// List handles GET /api/v1/maintenance-activities (Tenant-scoped)
func (h *MaintenanceHandler) List(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	filter := repository.MaintenanceFilter{
		Limit:  limit,
		Offset: offset,
	}

	if cid := r.URL.Query().Get("client_id"); cid != "" {
		filter.ClientID = &cid
	}
	if aid := r.URL.Query().Get("asset_id"); aid != "" {
		filter.AssetID = &aid
	}
	if cat := r.URL.Query().Get("category"); cat != "" {
		filter.Category = &cat
	}
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if parsed, err := time.Parse(time.RFC3339, fromStr); err == nil {
			filter.From = &parsed
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if parsed, err := time.Parse(time.RFC3339, toStr); err == nil {
			filter.To = &parsed
		}
	}

	activities, totalCount, err := h.repo.ListMaintenanceActivities(r.Context(), tc.OrganizationID, filter)
	if err != nil {
		h.logger.Error("failed to list maintenance activities", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list maintenance activities")
		return
	}

	if activities == nil {
		activities = []*repository.MaintenanceActivity{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"activities": activities,
		"total":      totalCount,
		"limit":      limit,
		"offset":     offset,
	})
}

// Create handles POST /api/v1/maintenance-activities (Tenant-scoped, RoleMember)
func (h *MaintenanceHandler) Create(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())

	var req struct {
		ClientID        string  `json:"client_id"`
		AssetID         *string `json:"asset_id"`
		Title           string  `json:"title"`
		Description     string  `json:"description"`
		Category        string  `json:"category"`
		PerformedAt     string  `json:"performed_at"`
		DurationMinutes *int    `json:"duration_minutes"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if strings.TrimSpace(req.ClientID) == "" {
		sendJSONError(w, http.StatusBadRequest, "INVALID_CLIENT", "Client ID is required")
		return
	}

	var perfTime time.Time
	if req.PerformedAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.PerformedAt)
		if err != nil {
			sendJSONError(w, http.StatusBadRequest, "INVALID_TIME", "performed_at must be an RFC3339 timestamp")
			return
		}
		perfTime = parsed
	} else {
		perfTime = time.Now().UTC()
	}

	act, err := h.repo.CreateMaintenanceActivity(r.Context(), tc.OrganizationID, repository.CreateMaintenanceParams{
		ClientID:        req.ClientID,
		AssetID:         req.AssetID,
		Title:           req.Title,
		Description:     req.Description,
		Category:        req.Category,
		PerformedAt:     perfTime,
		DurationMinutes: req.DurationMinutes,
		CreatedBy:       uc.UserID,
	})
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrInvalidTitle):
			sendJSONError(w, http.StatusBadRequest, "INVALID_TITLE", err.Error())
		case errors.Is(err, repository.ErrInvalidDescription):
			sendJSONError(w, http.StatusBadRequest, "INVALID_DESCRIPTION", err.Error())
		case errors.Is(err, repository.ErrInvalidCategory):
			sendJSONError(w, http.StatusBadRequest, "INVALID_CATEGORY", "Category must be one of: update, backup, bug_fix, performance, security_review, content_change, configuration, other")
		case errors.Is(err, repository.ErrInvalidDuration):
			sendJSONError(w, http.StatusBadRequest, "INVALID_DURATION", err.Error())
		default:
			sendJSONError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"activity": act,
	})
}

// Get handles GET /api/v1/maintenance-activities/{id} (Tenant-scoped)
func (h *MaintenanceHandler) Get(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	act, err := h.repo.GetMaintenanceActivity(r.Context(), tc.OrganizationID, id)
	if err != nil {
		if errors.Is(err, repository.ErrMaintenanceNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Maintenance activity not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get maintenance activity")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"activity": act,
	})
}

// Update handles PATCH /api/v1/maintenance-activities/{id} (Tenant-scoped, RoleMember)
func (h *MaintenanceHandler) Update(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	var req struct {
		AssetID         *string `json:"asset_id"`
		Title           *string `json:"title"`
		Description     *string `json:"description"`
		Category        *string `json:"category"`
		PerformedAt     *string `json:"performed_at"`
		DurationMinutes *int    `json:"duration_minutes"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	var parsedPerfTime *time.Time
	if req.PerformedAt != nil {
		parsed, err := time.Parse(time.RFC3339, *req.PerformedAt)
		if err != nil {
			sendJSONError(w, http.StatusBadRequest, "INVALID_TIME", "performed_at must be an RFC3339 timestamp")
			return
		}
		parsedPerfTime = &parsed
	}

	act, err := h.repo.UpdateMaintenanceActivity(r.Context(), tc.OrganizationID, id, repository.UpdateMaintenanceParams{
		AssetID:         req.AssetID,
		Title:           req.Title,
		Description:     req.Description,
		Category:        req.Category,
		PerformedAt:     parsedPerfTime,
		DurationMinutes: req.DurationMinutes,
		UpdatedBy:       uc.UserID,
	})
	if err != nil {
		if errors.Is(err, repository.ErrMaintenanceNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Maintenance activity not found")
			return
		}
		sendJSONError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"activity": act,
	})
}

// Delete handles DELETE /api/v1/maintenance-activities/{id} (Tenant-scoped, RoleMember)
func (h *MaintenanceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	err := h.repo.DeleteMaintenanceActivity(r.Context(), tc.OrganizationID, id, uc.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrMaintenanceNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Maintenance activity not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete maintenance activity")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "deleted",
	})
}
