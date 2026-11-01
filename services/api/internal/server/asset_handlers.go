package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/ssrf"
	"clientpulse/services/api/internal/tenant"
)

type AssetHandler struct {
	assetRepo *repository.AssetRepository
	auditRepo *repository.AuditRepository
	ssrfGuard *ssrf.Guard
	appEnv    string
}

func NewAssetHandler(assetRepo *repository.AssetRepository, auditRepo *repository.AuditRepository, ssrfGuard *ssrf.Guard, appEnv string) *AssetHandler {
	if ssrfGuard == nil {
		ssrfGuard = ssrf.NewGuard(nil)
	}
	return &AssetHandler{
		assetRepo: assetRepo,
		auditRepo: auditRepo,
		ssrfGuard: ssrfGuard,
		appEnv:    appEnv,
	}
}

type CreateAssetRequest struct {
	ClientID    string  `json:"client_id"`
	Name        string  `json:"name"`
	URL         string  `json:"url"`
	Type        string  `json:"type"`
	Environment string  `json:"environment"`
	Description *string `json:"description,omitempty"`
}

type UpdateAssetRequest struct {
	Name        string  `json:"name"`
	URL         string  `json:"url"`
	Type        string  `json:"type"`
	Environment string  `json:"environment"`
	Description *string `json:"description,omitempty"`
}

func (h *AssetHandler) Create(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())

	var req CreateAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON request body")
		return
	}

	req.ClientID = strings.TrimSpace(req.ClientID)
	req.Name = strings.TrimSpace(req.Name)
	req.URL = strings.TrimSpace(req.URL)

	if req.ClientID == "" {
		sendJSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Client ID is required")
		return
	}
	if req.Name == "" {
		sendJSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Asset name is required")
		return
	}
	if req.URL == "" {
		sendJSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Asset URL is required")
		return
	}

	// 1. SSRF Validation
	if _, err := h.ssrfGuard.ValidateURL(req.URL); err != nil {
		sendJSONError(w, http.StatusBadRequest, "SSRF_PROHIBITED_URL", err.Error())
		return
	}

	// 2. Reject private development URLs in hosted environments
	if req.Environment == "development" && h.appEnv != "development" {
		sendJSONError(w, http.StatusBadRequest, "INVALID_ENVIRONMENT", "Private development environments cannot be monitored in hosted product")
		return
	}

	// 3. Duplicate URL check (warn without blocking)
	isDuplicate, _ := h.assetRepo.CheckDuplicateURL(r.Context(), tc.OrganizationID, req.URL, "")

	// 4. Create asset
	asset, err := h.assetRepo.CreateAsset(r.Context(), tc.OrganizationID, req.ClientID, req.Name, req.URL, req.Type, req.Environment, req.Description)
	if err != nil {
		if errors.Is(err, repository.ErrClientNotInOrg) {
			sendJSONError(w, http.StatusBadRequest, "CLIENT_NOT_IN_ORGANIZATION", "The specified client does not belong to your organization")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create asset")
		return
	}

	if h.auditRepo != nil {
		meta, _ := json.Marshal(map[string]interface{}{"asset_name": asset.Name, "url": asset.URL})
		_ = h.auditRepo.RecordEvent(r.Context(), &tc.OrganizationID, &uc.UserID, "asset.created", "asset", asset.ID, meta, r.RemoteAddr)
	}

	response := map[string]interface{}{
		"asset": asset,
	}
	if isDuplicate {
		response["warning"] = "DUPLICATE_URL"
		response["warning_message"] = "Another asset in your organization already monitors this URL."
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *AssetHandler) Get(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	assetID := strings.TrimSpace(r.PathValue("id"))
	if assetID == "" {
		sendJSONError(w, http.StatusBadRequest, "INVALID_ASSET_ID", "Asset ID is required")
		return
	}

	asset, err := h.assetRepo.GetAssetByID(r.Context(), tc.OrganizationID, assetID)
	if err != nil {
		if errors.Is(err, repository.ErrAssetNotFound) {
			sendJSONError(w, http.StatusNotFound, "ASSET_NOT_FOUND", "Asset not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch asset")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"asset": asset,
	})
}

func (h *AssetHandler) List(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())

	q := r.URL.Query()
	clientID := q.Get("client_id")
	assetType := q.Get("type")
	environment := q.Get("environment")
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	var isActive *bool
	if activeStr := q.Get("is_active"); activeStr != "" {
		val := activeStr == "true" || activeStr == "1"
		isActive = &val
	}

	assets, total, err := h.assetRepo.ListAssets(r.Context(), tc.OrganizationID, clientID, assetType, environment, isActive, limit, offset)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list assets")
		return
	}

	if assets == nil {
		assets = []*repository.Asset{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"assets": assets,
		"total":  total,
	})
}

func (h *AssetHandler) Update(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())
	assetID := strings.TrimSpace(r.PathValue("id"))

	var req UpdateAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.URL = strings.TrimSpace(req.URL)

	if req.Name == "" {
		sendJSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Asset name is required")
		return
	}
	if req.URL == "" {
		sendJSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Asset URL is required")
		return
	}

	// SSRF Revalidation on URL update
	if _, err := h.ssrfGuard.ValidateURL(req.URL); err != nil {
		sendJSONError(w, http.StatusBadRequest, "SSRF_PROHIBITED_URL", err.Error())
		return
	}

	// Duplicate URL check
	isDuplicate, _ := h.assetRepo.CheckDuplicateURL(r.Context(), tc.OrganizationID, req.URL, assetID)

	asset, err := h.assetRepo.UpdateAsset(r.Context(), tc.OrganizationID, assetID, req.Name, req.URL, req.Type, req.Environment, req.Description)
	if err != nil {
		if errors.Is(err, repository.ErrAssetNotFound) {
			sendJSONError(w, http.StatusNotFound, "ASSET_NOT_FOUND", "Asset not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update asset")
		return
	}

	if h.auditRepo != nil {
		meta, _ := json.Marshal(map[string]interface{}{"asset_name": asset.Name, "url": asset.URL})
		_ = h.auditRepo.RecordEvent(r.Context(), &tc.OrganizationID, &uc.UserID, "asset.updated", "asset", asset.ID, meta, r.RemoteAddr)
	}

	response := map[string]interface{}{
		"asset": asset,
	}
	if isDuplicate {
		response["warning"] = "DUPLICATE_URL"
		response["warning_message"] = "Another asset in your organization already monitors this URL."
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *AssetHandler) Archive(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())
	assetID := strings.TrimSpace(r.PathValue("id"))

	err := h.assetRepo.ArchiveAsset(r.Context(), tc.OrganizationID, assetID)
	if err != nil {
		if errors.Is(err, repository.ErrAssetNotFound) {
			sendJSONError(w, http.StatusNotFound, "ASSET_NOT_FOUND", "Asset not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to archive asset")
		return
	}

	if h.auditRepo != nil {
		_ = h.auditRepo.RecordEvent(r.Context(), &tc.OrganizationID, &uc.UserID, "asset.archived", "asset", assetID, []byte("{}"), r.RemoteAddr)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "archived",
		"message": "Asset and associated monitors successfully archived",
	})
}

func (h *AssetHandler) Restore(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())
	assetID := strings.TrimSpace(r.PathValue("id"))

	err := h.assetRepo.RestoreAsset(r.Context(), tc.OrganizationID, assetID)
	if err != nil {
		if errors.Is(err, repository.ErrAssetNotFound) {
			sendJSONError(w, http.StatusNotFound, "ASSET_NOT_FOUND", "Asset not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to restore asset")
		return
	}

	if h.auditRepo != nil {
		_ = h.auditRepo.RecordEvent(r.Context(), &tc.OrganizationID, &uc.UserID, "asset.restored", "asset", assetID, []byte("{}"), r.RemoteAddr)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "active",
		"message": "Asset and monitors successfully restored to active",
	})
}
