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

// PlatformAdminHandler handles all endpoints under /api/v1/platform-admin/.
type PlatformAdminHandler struct {
	adminRepo        *repository.PlatformAdminRepository
	planRepo         *repository.PlanRepository
	subscriptionRepo *repository.SubscriptionRepository
	logger           *slog.Logger
}

// NewPlatformAdminHandler constructs a configured PlatformAdminHandler.
func NewPlatformAdminHandler(
	adminRepo *repository.PlatformAdminRepository,
	planRepo *repository.PlanRepository,
	subscriptionRepo *repository.SubscriptionRepository,
	logger *slog.Logger,
) *PlatformAdminHandler {
	return &PlatformAdminHandler{
		adminRepo:        adminRepo,
		planRepo:         planRepo,
		subscriptionRepo: subscriptionRepo,
		logger:           logger,
	}
}

// RequirePlatformAdmin ensures that the request context holds an authenticated user with verified platform admin privilege.
func RequirePlatformAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := tenant.GetUserContext(r.Context())
		if u == nil {
			sendJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
			return
		}
		if u.Status == "suspended" {
			sendJSONError(w, http.StatusForbidden, "USER_SUSPENDED", "Your account has been suspended by platform administration.")
			return
		}
		if !u.IsPlatformAdmin {
			sendJSONError(w, http.StatusForbidden, "FORBIDDEN", "Platform administrator privileges required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CheckPrivilege returns status indicating active platform administrator status.
func (h *PlatformAdminHandler) CheckPrivilege(w http.ResponseWriter, r *http.Request) {
	u := tenant.GetUserContext(r.Context())
	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"is_platform_admin": u != nil && u.IsPlatformAdmin,
		"user_id":           u.UserID,
		"email":             u.Email,
	})
}

// GetOverview handles GET /api/v1/platform-admin/overview.
func (h *PlatformAdminHandler) GetOverview(w http.ResponseWriter, r *http.Request) {
	stats, err := h.adminRepo.GetOverviewStats(r.Context())
	if err != nil {
		h.logger.Error("failed to get overview stats", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve overview statistics")
		return
	}
	sendJSONResponse(w, http.StatusOK, stats)
}

// ListUsers handles GET /api/v1/platform-admin/users.
func (h *PlatformAdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	search := strings.TrimSpace(query.Get("search"))
	status := strings.TrimSpace(query.Get("status"))

	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	users, total, err := h.adminRepo.ListUsers(r.Context(), search, status, limit, offset)
	if err != nil {
		h.logger.Error("failed to list platform users", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list users")
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetUser handles GET /api/v1/platform-admin/users/{id}.
func (h *PlatformAdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "User ID is required")
		return
	}

	detail, err := h.adminRepo.GetUserDetail(r.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
			return
		}
		h.logger.Error("failed to get user detail", "user_id", userID, "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve user details")
		return
	}

	sendJSONResponse(w, http.StatusOK, detail)
}

// SuspendUser handles POST /api/v1/platform-admin/users/{id}/suspend.
func (h *PlatformAdminHandler) SuspendUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	actor := tenant.GetUserContext(r.Context())

	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		reason = "Administrative user suspension"
	}

	if err := h.adminRepo.SetUserStatus(r.Context(), userID, "suspended"); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
			return
		}
		h.logger.Error("failed to suspend user", "user_id", userID, "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to suspend user")
		return
	}

	_ = h.adminRepo.RecordAuditLog(r.Context(), actor.UserID, "user.suspended", "user", userID, reason, nil, r.RemoteAddr, r.UserAgent())
	sendJSONResponse(w, http.StatusOK, map[string]interface{}{"status": "suspended", "user_id": userID})
}

// ReactivateUser handles POST /api/v1/platform-admin/users/{id}/reactivate.
func (h *PlatformAdminHandler) ReactivateUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	actor := tenant.GetUserContext(r.Context())

	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		reason = "Administrative user reactivation"
	}

	if err := h.adminRepo.SetUserStatus(r.Context(), userID, "active"); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
			return
		}
		h.logger.Error("failed to reactivate user", "user_id", userID, "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to reactivate user")
		return
	}

	_ = h.adminRepo.RecordAuditLog(r.Context(), actor.UserID, "user.reactivated", "user", userID, reason, nil, r.RemoteAddr, r.UserAgent())
	sendJSONResponse(w, http.StatusOK, map[string]interface{}{"status": "active", "user_id": userID})
}

// ListOrganizations handles GET /api/v1/platform-admin/organizations.
func (h *PlatformAdminHandler) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	search := strings.TrimSpace(query.Get("search"))
	status := strings.TrimSpace(query.Get("status"))

	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	orgs, total, err := h.adminRepo.ListOrganizations(r.Context(), search, status, limit, offset)
	if err != nil {
		h.logger.Error("failed to list organizations", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list organizations")
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"organizations": orgs,
		"total":         total,
		"page":          page,
		"limit":         limit,
	})
}

// GetOrganization handles GET /api/v1/platform-admin/organizations/{id}.
func (h *PlatformAdminHandler) GetOrganization(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Organization ID is required")
		return
	}

	detail, err := h.adminRepo.GetOrganizationDetail(r.Context(), orgID)
	if err != nil {
		if errors.Is(err, repository.ErrOrgNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Organization not found")
			return
		}
		h.logger.Error("failed to get organization detail", "org_id", orgID, "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve organization details")
		return
	}

	sendJSONResponse(w, http.StatusOK, detail)
}

// SuspendOrganization handles POST /api/v1/platform-admin/organizations/{id}/suspend.
func (h *PlatformAdminHandler) SuspendOrganization(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	actor := tenant.GetUserContext(r.Context())

	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		reason = "Administrative organization suspension"
	}

	if err := h.adminRepo.SetOrganizationStatus(r.Context(), orgID, "suspended"); err != nil {
		if errors.Is(err, repository.ErrOrgNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Organization not found")
			return
		}
		h.logger.Error("failed to suspend organization", "org_id", orgID, "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to suspend organization")
		return
	}

	_ = h.adminRepo.RecordAuditLog(r.Context(), actor.UserID, "organization.suspended", "organization", orgID, reason, nil, r.RemoteAddr, r.UserAgent())
	sendJSONResponse(w, http.StatusOK, map[string]interface{}{"status": "suspended", "organization_id": orgID})
}

// ReactivateOrganization handles POST /api/v1/platform-admin/organizations/{id}/reactivate.
func (h *PlatformAdminHandler) ReactivateOrganization(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	actor := tenant.GetUserContext(r.Context())

	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		reason = "Administrative organization reactivation"
	}

	if err := h.adminRepo.SetOrganizationStatus(r.Context(), orgID, "active"); err != nil {
		if errors.Is(err, repository.ErrOrgNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Organization not found")
			return
		}
		h.logger.Error("failed to reactivate organization", "org_id", orgID, "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to reactivate organization")
		return
	}

	_ = h.adminRepo.RecordAuditLog(r.Context(), actor.UserID, "organization.reactivated", "organization", orgID, reason, nil, r.RemoteAddr, r.UserAgent())
	sendJSONResponse(w, http.StatusOK, map[string]interface{}{"status": "active", "organization_id": orgID})
}

// ListPlans handles GET /api/v1/platform-admin/plans.
func (h *PlatformAdminHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.planRepo.ListPlans(r.Context())
	if err != nil {
		h.logger.Error("failed to list plans", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list subscription plans")
		return
	}
	sendJSONResponse(w, http.StatusOK, map[string]interface{}{"plans": plans})
}

type PlanInputRequest struct {
	Name                    string                 `json:"name"`
	Slug                    string                 `json:"slug"`
	Description             string                 `json:"description"`
	PriceAmount             *int64                 `json:"price_amount"`
	Currency                *string                `json:"currency"`
	BillingInterval         *string                `json:"billing_interval"`
	BillingPeriod           *string                `json:"billing_period,omitempty"`
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
	Limits                  map[string]interface{} `json:"limits,omitempty"`
}

func (req *PlanInputRequest) ToVersionParams() repository.CreatePlanVersionParams {
	billingInterval := req.BillingInterval
	if (billingInterval == nil || *billingInterval == "") && req.BillingPeriod != nil && *req.BillingPeriod != "" {
		billingInterval = req.BillingPeriod
	}

	params := repository.CreatePlanVersionParams{
		PriceAmount:             req.PriceAmount,
		Currency:                req.Currency,
		BillingInterval:         billingInterval,
		MaxClients:              req.MaxClients,
		MaxAssets:               req.MaxAssets,
		MaxMonitors:             req.MaxMonitors,
		MaxHeartbeats:           req.MaxHeartbeats,
		MaxMembers:              req.MaxMembers,
		MinCheckIntervalSeconds: req.MinCheckIntervalSeconds,
		RetentionDays:           req.RetentionDays,
		CustomBrandingEnabled:   req.CustomBrandingEnabled,
		PublicReportsEnabled:    req.PublicReportsEnabled,
		Features:                req.Features,
	}

	if req.Limits != nil {
		if v, ok := req.Limits["max_clients"].(float64); ok && params.MaxClients == 0 {
			params.MaxClients = int(v)
		}
		if v, ok := req.Limits["max_assets"].(float64); ok && params.MaxAssets == 0 {
			params.MaxAssets = int(v)
		}
		if v, ok := req.Limits["max_monitors"].(float64); ok && params.MaxMonitors == 0 {
			params.MaxMonitors = int(v)
		}
		if v, ok := req.Limits["max_heartbeats"].(float64); ok && params.MaxHeartbeats == 0 {
			params.MaxHeartbeats = int(v)
		}
		if v, ok := req.Limits["max_team_members"].(float64); ok && params.MaxMembers == 0 {
			params.MaxMembers = int(v)
		}
		if v, ok := req.Limits["min_interval_seconds"].(float64); ok && params.MinCheckIntervalSeconds == 0 {
			params.MinCheckIntervalSeconds = int(v)
		}
		if v, ok := req.Limits["history_retention_days"].(float64); ok && params.RetentionDays == 0 {
			params.RetentionDays = int(v)
		}
		if feat, ok := req.Limits["features"].(map[string]interface{}); ok {
			if params.Features == nil {
				params.Features = feat
			}
			if cb, ok := feat["custom_branding"].(bool); ok {
				params.CustomBrandingEnabled = cb
			}
			if pr, ok := feat["care_reports"].(bool); ok {
				params.PublicReportsEnabled = pr
			}
		}
	}

	if params.MinCheckIntervalSeconds < 60 {
		params.MinCheckIntervalSeconds = 60
	}

	return params
}

// CreateDraftPlan handles POST /api/v1/platform-admin/plans.
func (h *PlatformAdminHandler) CreateDraftPlan(w http.ResponseWriter, r *http.Request) {
	actor := tenant.GetUserContext(r.Context())

	var req PlanInputRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON payload")
		return
	}

	vParams := req.ToVersionParams()
	plan, version, err := h.planRepo.CreateDraftPlan(r.Context(), req.Name, req.Slug, req.Description, actor.UserID, vParams)
	if err != nil {
		if errors.Is(err, repository.ErrPlanSlugAlreadyExists) {
			sendJSONError(w, http.StatusConflict, "CONFLICT", "A subscription plan with this slug already exists")
			return
		}
		if errors.Is(err, repository.ErrInvalidCurrency) {
			sendJSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		h.logger.Error("failed to create draft plan", "error", err)
		sendJSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	_ = h.adminRepo.RecordAuditLog(r.Context(), actor.UserID, "plan.created", "plan", plan.ID, "Created initial draft plan", map[string]interface{}{
		"name": plan.Name,
		"slug": plan.Slug,
	}, r.RemoteAddr, r.UserAgent())

	sendJSONResponse(w, http.StatusCreated, map[string]interface{}{
		"plan":    plan,
		"version": version,
	})
}

// GetPlan handles GET /api/v1/platform-admin/plans/{id}.
func (h *PlatformAdminHandler) GetPlan(w http.ResponseWriter, r *http.Request) {
	planID := r.PathValue("id")
	plan, versions, err := h.planRepo.GetPlanByID(r.Context(), planID)
	if err != nil {
		if errors.Is(err, repository.ErrPlanNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Plan not found")
			return
		}
		h.logger.Error("failed to get plan", "plan_id", planID, "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve plan")
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"plan":     plan,
		"versions": versions,
	})
}

// UpdatePlan handles PUT /api/v1/platform-admin/plans/{id}.
func (h *PlatformAdminHandler) UpdatePlan(w http.ResponseWriter, r *http.Request) {
	planID := r.PathValue("id")
	actor := tenant.GetUserContext(r.Context())

	var req PlanInputRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON payload")
		return
	}

	vParams := req.ToVersionParams()
	plan, version, err := h.planRepo.UpdateDraftPlan(r.Context(), planID, req.Name, req.Description, actor.UserID, vParams)
	if err != nil {
		if errors.Is(err, repository.ErrPlanNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Plan not found")
			return
		}
		h.logger.Error("failed to update plan", "plan_id", planID, "error", err)
		sendJSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	_ = h.adminRepo.RecordAuditLog(r.Context(), actor.UserID, "plan.updated", "plan", plan.ID, "Updated plan configuration", map[string]interface{}{
		"version": version.Version,
		"status":  version.Status,
	}, r.RemoteAddr, r.UserAgent())

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"plan":    plan,
		"version": version,
	})
}

// PublishPlan handles POST /api/v1/platform-admin/plans/{id}/publish.
func (h *PlatformAdminHandler) PublishPlan(w http.ResponseWriter, r *http.Request) {
	planID := r.PathValue("id")
	actor := tenant.GetUserContext(r.Context())

	plan, version, err := h.planRepo.PublishPlan(r.Context(), planID, actor.UserID)
	if err != nil {
		h.logger.Error("failed to publish plan", "plan_id", planID, "error", err)
		sendJSONError(w, http.StatusBadRequest, "PUBLISH_FAILED", err.Error())
		return
	}

	_ = h.adminRepo.RecordAuditLog(r.Context(), actor.UserID, "plan.published", "plan", plan.ID, "Published plan version", map[string]interface{}{
		"version": version.Version,
	}, r.RemoteAddr, r.UserAgent())

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"plan":    plan,
		"version": version,
	})
}

// ArchivePlan handles POST /api/v1/platform-admin/plans/{id}/archive.
func (h *PlatformAdminHandler) ArchivePlan(w http.ResponseWriter, r *http.Request) {
	planID := r.PathValue("id")
	actor := tenant.GetUserContext(r.Context())

	if err := h.planRepo.ArchivePlan(r.Context(), planID, actor.UserID); err != nil {
		if errors.Is(err, repository.ErrPlanNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Plan not found")
			return
		}
		h.logger.Error("failed to archive plan", "plan_id", planID, "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to archive plan")
		return
	}

	_ = h.adminRepo.RecordAuditLog(r.Context(), actor.UserID, "plan.archived", "plan", planID, "Archived subscription plan", nil, r.RemoteAddr, r.UserAgent())
	sendJSONResponse(w, http.StatusOK, map[string]interface{}{"status": "archived", "plan_id": planID})
}

// DeleteDraftPlan handles DELETE /api/v1/platform-admin/plans/{id}.
func (h *PlatformAdminHandler) DeleteDraftPlan(w http.ResponseWriter, r *http.Request) {
	planID := r.PathValue("id")
	actor := tenant.GetUserContext(r.Context())

	if err := h.planRepo.DeleteDraftPlan(r.Context(), planID); err != nil {
		if errors.Is(err, repository.ErrPlanNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Plan not found")
			return
		}
		h.logger.Error("failed to delete draft plan", "plan_id", planID, "error", err)
		sendJSONError(w, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}

	_ = h.adminRepo.RecordAuditLog(r.Context(), actor.UserID, "plan.deleted", "plan", planID, "Deleted draft subscription plan", nil, r.RemoteAddr, r.UserAgent())
	sendJSONResponse(w, http.StatusOK, map[string]interface{}{"status": "deleted", "plan_id": planID})
}

// GetPlanAssignments handles GET /api/v1/platform-admin/plans/{id}/assignments.
func (h *PlatformAdminHandler) GetPlanAssignments(w http.ResponseWriter, r *http.Request) {
	planID := r.PathValue("id")
	assignments, err := h.planRepo.GetPlanAssignments(r.Context(), planID)
	if err != nil {
		h.logger.Error("failed to get plan assignments", "plan_id", planID, "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve assignments")
		return
	}
	sendJSONResponse(w, http.StatusOK, map[string]interface{}{"assignments": assignments})
}

// ListSubscriptions handles GET /api/v1/platform-admin/subscriptions.
func (h *PlatformAdminHandler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	orgID := strings.TrimSpace(query.Get("organization_id"))
	status := strings.TrimSpace(query.Get("status"))

	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	subs, total, err := h.subscriptionRepo.ListSubscriptions(r.Context(), orgID, status, limit, offset)
	if err != nil {
		h.logger.Error("failed to list subscriptions", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list subscriptions")
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"subscriptions": subs,
		"total":         total,
		"page":          page,
		"limit":         limit,
	})
}

// AssignSubscription handles POST /api/v1/platform-admin/subscriptions/assign.
func (h *PlatformAdminHandler) AssignSubscription(w http.ResponseWriter, r *http.Request) {
	actor := tenant.GetUserContext(r.Context())

	var req struct {
		OrganizationID     string `json:"organization_id"`
		PlanID             string `json:"plan_id"`
		PlanVersionID      string `json:"plan_version_id"`
		AdministrativeNote string `json:"administrative_note"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON payload")
		return
	}

	if req.PlanVersionID == "" && req.PlanID != "" {
		_, versions, err := h.planRepo.GetPlanByID(r.Context(), req.PlanID)
		if err == nil {
			for _, v := range versions {
				if v.Status == "published" {
					req.PlanVersionID = v.ID
					break
				}
			}
		}
	}

	if strings.TrimSpace(req.OrganizationID) == "" || strings.TrimSpace(req.PlanVersionID) == "" {
		sendJSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Both organization_id and plan_version_id are required")
		return
	}

	sub, warnings, err := h.subscriptionRepo.AssignPlan(r.Context(), req.OrganizationID, req.PlanVersionID, actor.UserID, req.AdministrativeNote)
	if err != nil {
		if errors.Is(err, repository.ErrOrgNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Organization not found")
			return
		}
		if errors.Is(err, repository.ErrPlanVersionNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Plan version not found")
			return
		}
		if errors.Is(err, repository.ErrCannotAssignDraft) || errors.Is(err, repository.ErrCannotAssignArchived) {
			sendJSONError(w, http.StatusBadRequest, "INVALID_PLAN_STATE", err.Error())
			return
		}
		h.logger.Error("failed to assign plan", "org_id", req.OrganizationID, "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to assign plan")
		return
	}

	_ = h.adminRepo.RecordAuditLog(r.Context(), actor.UserID, "subscription.assigned", "subscription", sub.ID, req.AdministrativeNote, map[string]interface{}{
		"organization_id": sub.OrganizationID,
		"plan_name":       sub.PlanName,
		"plan_version":    sub.PlanVersion,
		"warnings":        warnings,
	}, r.RemoteAddr, r.UserAgent())

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"subscription": sub,
		"warnings":     warnings,
	})
}

// CheckLimits handles POST /api/v1/platform-admin/subscriptions/check-limits.
func (h *PlatformAdminHandler) CheckLimits(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrganizationID string `json:"organization_id"`
		PlanID         string `json:"plan_id"`
		PlanVersionID  string `json:"plan_version_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON payload")
		return
	}

	if req.PlanVersionID == "" && req.PlanID != "" {
		_, versions, err := h.planRepo.GetPlanByID(r.Context(), req.PlanID)
		if err == nil {
			for _, v := range versions {
				if v.Status == "published" {
					req.PlanVersionID = v.ID
					break
				}
			}
		}
	}

	if strings.TrimSpace(req.OrganizationID) == "" || strings.TrimSpace(req.PlanVersionID) == "" {
		sendJSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Both organization_id and plan_version_id are required")
		return
	}

	warnings, err := h.subscriptionRepo.CheckOverLimitWarnings(r.Context(), req.OrganizationID, req.PlanVersionID)
	if err != nil {
		if errors.Is(err, repository.ErrPlanVersionNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Plan version not found")
			return
		}
		h.logger.Error("failed to check limits", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to check limits")
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"warnings":     warnings,
		"has_warnings": len(warnings) > 0,
	})
}

// SuspendSubscription handles POST /api/v1/platform-admin/subscriptions/{id}/suspend.
func (h *PlatformAdminHandler) SuspendSubscription(w http.ResponseWriter, r *http.Request) {
	subID := r.PathValue("id")
	actor := tenant.GetUserContext(r.Context())

	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	if err := h.subscriptionRepo.UpdateSubscriptionStatus(r.Context(), subID, "suspended", actor.UserID, body.Reason); err != nil {
		if errors.Is(err, repository.ErrSubscriptionNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Subscription not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to suspend subscription")
		return
	}

	_ = h.adminRepo.RecordAuditLog(r.Context(), actor.UserID, "subscription.suspended", "subscription", subID, body.Reason, nil, r.RemoteAddr, r.UserAgent())
	sendJSONResponse(w, http.StatusOK, map[string]interface{}{"status": "suspended", "subscription_id": subID})
}

// ReactivateSubscription handles POST /api/v1/platform-admin/subscriptions/{id}/reactivate.
func (h *PlatformAdminHandler) ReactivateSubscription(w http.ResponseWriter, r *http.Request) {
	subID := r.PathValue("id")
	actor := tenant.GetUserContext(r.Context())

	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	if err := h.subscriptionRepo.UpdateSubscriptionStatus(r.Context(), subID, "active", actor.UserID, body.Reason); err != nil {
		if errors.Is(err, repository.ErrSubscriptionNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Subscription not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to reactivate subscription")
		return
	}

	_ = h.adminRepo.RecordAuditLog(r.Context(), actor.UserID, "subscription.reactivated", "subscription", subID, body.Reason, nil, r.RemoteAddr, r.UserAgent())
	sendJSONResponse(w, http.StatusOK, map[string]interface{}{"status": "active", "subscription_id": subID})
}

// CancelSubscription handles POST /api/v1/platform-admin/subscriptions/{id}/cancel.
func (h *PlatformAdminHandler) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	subID := r.PathValue("id")
	actor := tenant.GetUserContext(r.Context())

	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	if err := h.subscriptionRepo.UpdateSubscriptionStatus(r.Context(), subID, "canceled", actor.UserID, body.Reason); err != nil {
		if errors.Is(err, repository.ErrSubscriptionNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Subscription not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to cancel subscription")
		return
	}

	_ = h.adminRepo.RecordAuditLog(r.Context(), actor.UserID, "subscription.canceled", "subscription", subID, body.Reason, nil, r.RemoteAddr, r.UserAgent())
	sendJSONResponse(w, http.StatusOK, map[string]interface{}{"status": "canceled", "subscription_id": subID})
}

// ListAuditLogs handles GET /api/v1/platform-admin/audit-logs.
func (h *PlatformAdminHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	actorID := strings.TrimSpace(query.Get("actor_id"))
	action := strings.TrimSpace(query.Get("action"))
	targetType := strings.TrimSpace(query.Get("target_type"))
	targetID := strings.TrimSpace(query.Get("target_id"))

	var fromDate, toDate *time.Time
	if fromStr := strings.TrimSpace(query.Get("from")); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			fromDate = &t
		}
	}
	if toStr := strings.TrimSpace(query.Get("to")); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			toDate = &t
		}
	}

	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	offset := (page - 1) * limit

	logs, total, err := h.adminRepo.ListAuditLogs(r.Context(), actorID, action, targetType, targetID, fromDate, toDate, limit, offset)
	if err != nil {
		h.logger.Error("failed to list audit logs", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve audit logs")
		return
	}

	sendJSONResponse(w, http.StatusOK, map[string]interface{}{
		"audit_logs": logs,
		"total":      total,
		"page":       page,
		"limit":      limit,
	})
}

