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

type IncidentHandler struct {
	repo      *repository.IncidentRepository
	eventRepo *repository.NotificationEventRepository
	logger    *slog.Logger
}

func NewIncidentHandler(repo *repository.IncidentRepository, eventRepo *repository.NotificationEventRepository, logger *slog.Logger) *IncidentHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &IncidentHandler{
		repo:      repo,
		eventRepo: eventRepo,
		logger:    logger,
	}
}

// List handles GET /api/v1/incidents (Tenant-scoped)
func (h *IncidentHandler) List(w http.ResponseWriter, r *http.Request) {
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

	filter := repository.IncidentFilter{
		Status:             r.URL.Query().Get("status"),
		ClientID:           r.URL.Query().Get("client_id"),
		MonitorID:          r.URL.Query().Get("monitor_id"),
		HeartbeatMonitorID: r.URL.Query().Get("heartbeat_id"),
		FailureCategory:    r.URL.Query().Get("failure_category"),
		TargetType:         r.URL.Query().Get("target_type"),
		AckState:           r.URL.Query().Get("ack_state"),
		Limit:              limit,
		Offset:             offset,
	}

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			filter.From = &t
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			filter.To = &t
		}
	}

	incidents, total, err := h.repo.ListIncidents(r.Context(), tc.OrganizationID, filter)
	if err != nil {
		h.logger.Error("failed to list incidents", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve incidents")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"incidents": incidents,
		"total":     total,
	})
}

// Get handles GET /api/v1/incidents/{id} (Tenant-scoped)
func (h *IncidentHandler) Get(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	inc, err := h.repo.GetIncidentByID(r.Context(), tc.OrganizationID, id)
	if err != nil {
		if errors.Is(err, repository.ErrIncidentNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Incident not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve incident")
		return
	}

	var events []*repository.NotificationEvent
	if h.eventRepo != nil {
		events, _ = h.eventRepo.GetEventsByIncidentID(r.Context(), tc.OrganizationID, id)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"incident": inc,
		"events":   events,
	})
}

// Acknowledge handles POST /api/v1/incidents/{id}/acknowledge (Tenant-scoped, RoleMember)
func (h *IncidentHandler) Acknowledge(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	inc, err := h.repo.AcknowledgeIncident(r.Context(), tc.OrganizationID, id, uc.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrIncidentNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Incident not found")
			return
		}
		if errors.Is(err, repository.ErrIncidentAlreadyResolved) {
			sendJSONError(w, http.StatusBadRequest, "ALREADY_RESOLVED", "Incident is already resolved")
			return
		}
		if errors.Is(err, repository.ErrIncidentAlreadyAcked) {
			sendJSONError(w, http.StatusBadRequest, "ALREADY_ACKNOWLEDGED", "Incident has already been acknowledged")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to acknowledge incident")
		return
	}

	h.logger.Info("incident acknowledged", "incident_id", id, "user_id", uc.UserID)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"incident": inc,
	})
}

// Resolve handles POST /api/v1/incidents/{id}/resolve (Tenant-scoped, RoleMember)
func (h *IncidentHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	var body struct {
		Note *string `json:"note"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}

	inc, err := h.repo.ResolveIncidentManually(r.Context(), tc.OrganizationID, id, uc.UserID, body.Note)
	if err != nil {
		if errors.Is(err, repository.ErrIncidentNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Incident not found")
			return
		}
		if errors.Is(err, repository.ErrIncidentAlreadyResolved) {
			sendJSONError(w, http.StatusBadRequest, "ALREADY_RESOLVED", "Incident is already resolved")
			return
		}
		h.logger.Error("failed to manually resolve incident", "incident_id", id, "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to resolve incident")
		return
	}

	h.logger.Info("incident manually resolved", "incident_id", id, "user_id", uc.UserID)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"incident": inc,
	})
}

// Uptime handles GET /api/v1/monitors/{id}/uptime (Tenant-scoped)
func (h *IncidentHandler) Uptime(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	monitorID := strings.TrimSpace(r.PathValue("id"))

	window := 30 * 24 * time.Hour
	switch r.URL.Query().Get("window") {
	case "24h":
		window = 24 * time.Hour
	case "7d":
		window = 7 * 24 * time.Hour
	case "30d":
		window = 30 * 24 * time.Hour
	case "90d":
		window = 90 * 24 * time.Hour
	}

	report, err := h.repo.CalculateMonitorUptime(r.Context(), tc.OrganizationID, monitorID, window)
	if err != nil {
		if errors.Is(err, repository.ErrMonitorNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Monitor not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to calculate uptime")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"uptime": report,
	})
}

