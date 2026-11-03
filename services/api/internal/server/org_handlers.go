package server

import (
	"encoding/json"
	"net/http"

	"clientpulse/services/api/internal/entitlement"
	"clientpulse/services/api/internal/service"
	"clientpulse/services/api/internal/tenant"
)

type OrgHandler struct {
	orgService   *service.OrgService
	entitlements *entitlement.Service
}

func NewOrgHandler(orgService *service.OrgService) *OrgHandler {
	return &OrgHandler{orgService: orgService}
}

// WithEntitlements attaches the entitlement service for member limit checks.
func (h *OrgHandler) WithEntitlements(svc *entitlement.Service) *OrgHandler {
	h.entitlements = svc
	return h
}

// ListUserOrganizations returns all organizations the authenticated user belongs to.
func (h *OrgHandler) ListUserOrganizations(w http.ResponseWriter, r *http.Request) {
	u := tenant.GetUserContext(r.Context())
	if u == nil {
		sendJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required.")
		return
	}

	memberships, err := h.orgService.ListUserOrganizations(r.Context(), u.UserID)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve organizations.")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"organizations": memberships,
	})
}

// CreateOrganization provisions a new organization for the user.
func (h *OrgHandler) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	u := tenant.GetUserContext(r.Context())
	if u == nil {
		sendJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required.")
		return
	}

	var req service.CreateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload format.")
		return
	}

	org, err := h.orgService.CreateOrganization(r.Context(), u.UserID, req)
	if err != nil {
		sendJSONError(w, http.StatusBadRequest, "CREATE_ORG_FAILED", err.Error())
		return
	}

	// Also switch session to this new org
	if u.SessionID != "" {
		_, _ = h.orgService.SwitchOrganization(r.Context(), u.SessionID, u.UserID, org.ID)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(org)
}

// SelectOrganization switches active organization for the current session.
func (h *OrgHandler) SelectOrganization(w http.ResponseWriter, r *http.Request) {
	u := tenant.GetUserContext(r.Context())
	if u == nil {
		sendJSONError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required.")
		return
	}

	var req struct {
		OrganizationID string `json:"organization_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload format.")
		return
	}

	org, err := h.orgService.SwitchOrganization(r.Context(), u.SessionID, u.UserID, req.OrganizationID)
	if err != nil {
		sendJSONError(w, http.StatusForbidden, "SELECTION_FAILED", "User is not a member of this organization.")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Active organization switched successfully.",
		"organization": org,
	})
}

// GetCurrent returns the active organization details and caller's role.
func (h *OrgHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	t := tenant.GetTenantContext(r.Context())
	u := tenant.GetUserContext(r.Context())
	if t == nil || u == nil {
		sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "No active organization.")
		return
	}

	org, role, err := h.orgService.GetCurrentOrganization(r.Context(), t.OrganizationID, u.UserID)
	if err != nil {
		sendJSONError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"organization": org,
		"role":         role,
	})
}

// UpdateCurrent updates active organization settings (Owner or Admin only).
func (h *OrgHandler) UpdateCurrent(w http.ResponseWriter, r *http.Request) {
	t := tenant.GetTenantContext(r.Context())
	u := tenant.GetUserContext(r.Context())
	if t == nil || u == nil {
		sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "No active organization.")
		return
	}

	var req service.UpdateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload format.")
		return
	}

	org, err := h.orgService.UpdateOrganization(r.Context(), t.OrganizationID, u.UserID, req)
	if err != nil {
		sendJSONError(w, http.StatusForbidden, "UPDATE_FAILED", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(org)
}

// ListMembers lists members in active organization.
func (h *OrgHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	t := tenant.GetTenantContext(r.Context())
	u := tenant.GetUserContext(r.Context())
	if t == nil || u == nil {
		sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "No active organization.")
		return
	}

	members, err := h.orgService.ListMembers(r.Context(), t.OrganizationID, u.UserID)
	if err != nil {
		sendJSONError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"members": members,
	})
}

// AddMember invites/adds a member to active organization.
func (h *OrgHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	t := tenant.GetTenantContext(r.Context())
	u := tenant.GetUserContext(r.Context())
	if t == nil || u == nil {
		sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "No active organization.")
		return
	}

	var req service.AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload format.")
		return
	}

	if h.entitlements != nil {
		if err := h.entitlements.CheckMemberAddition(r.Context(), t.OrganizationID); err != nil {
			sendJSONError(w, http.StatusForbidden, "LIMIT_EXCEEDED", err.Error())
			return
		}
	}

	membership, err := h.orgService.AddMember(r.Context(), t.OrganizationID, u.UserID, req)
	if err != nil {
		sendJSONError(w, http.StatusBadRequest, "ADD_MEMBER_FAILED", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(membership)
}

// UpdateMemberRole changes role of a member (Owner only).
func (h *OrgHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	t := tenant.GetTenantContext(r.Context())
	u := tenant.GetUserContext(r.Context())
	if t == nil || u == nil {
		sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "No active organization.")
		return
	}

	targetUserID := r.PathValue("userID")
	if targetUserID == "" {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Target user ID is required.")
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request payload format.")
		return
	}

	if err := h.orgService.UpdateMemberRole(r.Context(), t.OrganizationID, u.UserID, targetUserID, req.Role); err != nil {
		sendJSONError(w, http.StatusBadRequest, "ROLE_UPDATE_FAILED", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Member role updated successfully.",
	})
}

// RemoveMember removes a member from active organization.
func (h *OrgHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	t := tenant.GetTenantContext(r.Context())
	u := tenant.GetUserContext(r.Context())
	if t == nil || u == nil {
		sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "No active organization.")
		return
	}

	targetUserID := r.PathValue("userID")
	if targetUserID == "" {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Target user ID is required.")
		return
	}

	if err := h.orgService.RemoveMember(r.Context(), t.OrganizationID, u.UserID, targetUserID); err != nil {
		sendJSONError(w, http.StatusBadRequest, "REMOVE_MEMBER_FAILED", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Member removed successfully.",
	})
}

