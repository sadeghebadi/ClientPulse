package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"clientpulse/services/api/internal/entitlement"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/tenant"
)

type ClientHandler struct {
	clientRepo   *repository.ClientRepository
	auditRepo    *repository.AuditRepository
	entitlements *entitlement.Service
}

func NewClientHandler(clientRepo *repository.ClientRepository, auditRepo *repository.AuditRepository) *ClientHandler {
	return &ClientHandler{
		clientRepo: clientRepo,
		auditRepo:  auditRepo,
	}
}

// WithEntitlements attaches the entitlement service for quota enforcement.
func (h *ClientHandler) WithEntitlements(svc *entitlement.Service) *ClientHandler {
	h.entitlements = svc
	return h
}

type CreateClientRequest struct {
	Name         string  `json:"name"`
	ContactName  *string `json:"contact_name,omitempty"`
	ContactEmail *string `json:"contact_email,omitempty"`
	Website      *string `json:"website,omitempty"`
	Notes        *string `json:"notes,omitempty"`
}

type UpdateClientRequest struct {
	Name         string  `json:"name"`
	ContactName  *string `json:"contact_name,omitempty"`
	ContactEmail *string `json:"contact_email,omitempty"`
	Website      *string `json:"website,omitempty"`
	Notes        *string `json:"notes,omitempty"`
}

func (h *ClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())

	var req CreateClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		sendJSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Client name is required")
		return
	}

	if h.entitlements != nil {
		if err := h.entitlements.CheckClientCreation(r.Context(), tc.OrganizationID); err != nil {
			sendJSONError(w, http.StatusForbidden, "LIMIT_EXCEEDED", err.Error())
			return
		}
	}

	client, err := h.clientRepo.CreateClient(r.Context(), tc.OrganizationID, req.Name, req.ContactName, req.ContactEmail, req.Website, req.Notes)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create client")
		return
	}

	// Record audit event
	if h.auditRepo != nil {
		meta, _ := json.Marshal(map[string]interface{}{"client_name": client.Name})
		_ = h.auditRepo.RecordEvent(r.Context(), &tc.OrganizationID, &uc.UserID, "client.created", "client", client.ID, meta, r.RemoteAddr)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"client": client,
	})
}

func (h *ClientHandler) Get(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	clientID := strings.TrimSpace(r.PathValue("id"))
	if clientID == "" {
		sendJSONError(w, http.StatusBadRequest, "INVALID_CLIENT_ID", "Client ID is required")
		return
	}

	client, err := h.clientRepo.GetClientByID(r.Context(), tc.OrganizationID, clientID)
	if err != nil {
		if errors.Is(err, repository.ErrClientNotFound) {
			sendJSONError(w, http.StatusNotFound, "CLIENT_NOT_FOUND", "Client not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch client")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"client": client,
	})
}

func (h *ClientHandler) List(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())

	q := r.URL.Query()
	search := q.Get("search")
	status := q.Get("status")
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	clients, total, err := h.clientRepo.ListClients(r.Context(), tc.OrganizationID, search, status, limit, offset)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list clients")
		return
	}

	if clients == nil {
		clients = []*repository.Client{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"clients": clients,
		"total":   total,
	})
}

func (h *ClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())
	clientID := strings.TrimSpace(r.PathValue("id"))

	var req UpdateClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		sendJSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Client name is required")
		return
	}

	client, err := h.clientRepo.UpdateClient(r.Context(), tc.OrganizationID, clientID, req.Name, req.ContactName, req.ContactEmail, req.Website, req.Notes)
	if err != nil {
		if errors.Is(err, repository.ErrClientNotFound) {
			sendJSONError(w, http.StatusNotFound, "CLIENT_NOT_FOUND", "Client not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update client")
		return
	}

	if h.auditRepo != nil {
		meta, _ := json.Marshal(map[string]interface{}{"client_name": client.Name})
		_ = h.auditRepo.RecordEvent(r.Context(), &tc.OrganizationID, &uc.UserID, "client.updated", "client", client.ID, meta, r.RemoteAddr)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"client": client,
	})
}

func (h *ClientHandler) Archive(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())
	clientID := strings.TrimSpace(r.PathValue("id"))

	err := h.clientRepo.ArchiveClient(r.Context(), tc.OrganizationID, clientID)
	if err != nil {
		if errors.Is(err, repository.ErrClientNotFound) {
			sendJSONError(w, http.StatusNotFound, "CLIENT_NOT_FOUND", "Client not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to archive client")
		return
	}

	if h.auditRepo != nil {
		_ = h.auditRepo.RecordEvent(r.Context(), &tc.OrganizationID, &uc.UserID, "client.archived", "client", clientID, []byte("{}"), r.RemoteAddr)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "archived",
		"message": "Client and related assets successfully archived",
	})
}

func (h *ClientHandler) Restore(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())
	clientID := strings.TrimSpace(r.PathValue("id"))

	err := h.clientRepo.RestoreClient(r.Context(), tc.OrganizationID, clientID)
	if err != nil {
		if errors.Is(err, repository.ErrClientNotFound) {
			sendJSONError(w, http.StatusNotFound, "CLIENT_NOT_FOUND", "Client not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to restore client")
		return
	}

	if h.auditRepo != nil {
		_ = h.auditRepo.RecordEvent(r.Context(), &tc.OrganizationID, &uc.UserID, "client.restored", "client", clientID, []byte("{}"), r.RemoteAddr)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "active",
		"message": "Client successfully restored to active",
	})
}

func (h *ClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())
	clientID := strings.TrimSpace(r.PathValue("id"))

	err := h.clientRepo.DeleteClient(r.Context(), tc.OrganizationID, clientID)
	if err != nil {
		if errors.Is(err, repository.ErrClientNotFound) {
			sendJSONError(w, http.StatusNotFound, "CLIENT_NOT_FOUND", "Client not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete client")
		return
	}

	if h.auditRepo != nil {
		_ = h.auditRepo.RecordEvent(r.Context(), &tc.OrganizationID, &uc.UserID, "client.deleted", "client", clientID, []byte("{}"), r.RemoteAddr)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "deleted",
		"message": "Client permanently deleted",
	})
}
