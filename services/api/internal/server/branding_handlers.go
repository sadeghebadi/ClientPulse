package server

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"clientpulse/services/api/internal/entitlement"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/storage"
	"clientpulse/services/api/internal/tenant"
	_ "golang.org/x/image/webp"
)

type BrandingHandler struct {
	repo         *repository.BrandingRepository
	storage      storage.Storage
	logger       *slog.Logger
	entitlements *entitlement.Service
}

func NewBrandingHandler(repo *repository.BrandingRepository, st storage.Storage, logger *slog.Logger) *BrandingHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &BrandingHandler{
		repo:    repo,
		storage: st,
		logger:  logger,
	}
}

// WithEntitlements attaches the entitlement service for branding feature gating.
func (h *BrandingHandler) WithEntitlements(svc *entitlement.Service) *BrandingHandler {
	h.entitlements = svc
	return h
}

// Get handles GET /api/v1/settings/branding (Tenant-scoped)
func (h *BrandingHandler) Get(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())

	b, err := h.repo.GetBranding(r.Context(), tc.OrganizationID)
	if err != nil {
		h.logger.Error("failed to get branding", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get branding")
		return
	}

	ratio, isAccessible := repository.CalculateContrastRatio(b.PrimaryColor)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"branding":               b,
		"primary_contrast_ratio": ratio,
		"is_accessible":          isAccessible,
		"has_logo":               b.LogoStorageKey != nil && *b.LogoStorageKey != "",
	})
}

// Update handles PATCH /api/v1/settings/branding (Tenant-scoped, RoleAdmin)
func (h *BrandingHandler) Update(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())

	var req struct {
		DisplayName         *string `json:"display_name"`
		PrimaryColor        *string `json:"primary_color"`
		AccentColor         *string `json:"accent_color"`
		ReportFooterText    *string `json:"report_footer_text"`
		ContactEmail        *string `json:"contact_email"`
		WebsiteURL          *string `json:"website_url"`
		DefaultReportLocale *string `json:"default_report_locale"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if h.entitlements != nil {
		if err := h.entitlements.CheckCustomBranding(r.Context(), tc.OrganizationID); err != nil {
			sendJSONError(w, http.StatusForbidden, "FEATURE_NOT_AVAILABLE", "Custom branding is not available on your organization's subscription plan")
			return
		}
	}

	b, err := h.repo.UpsertBranding(r.Context(), tc.OrganizationID, repository.UpdateBrandingParams{
		DisplayName:         req.DisplayName,
		PrimaryColor:        req.PrimaryColor,
		AccentColor:         req.AccentColor,
		ReportFooterText:    req.ReportFooterText,
		ContactEmail:        req.ContactEmail,
		WebsiteURL:          req.WebsiteURL,
		DefaultReportLocale: req.DefaultReportLocale,
	})
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrInvalidColor):
			sendJSONError(w, http.StatusBadRequest, "INVALID_COLOR", err.Error())
		case errors.Is(err, repository.ErrInvalidLocale):
			sendJSONError(w, http.StatusBadRequest, "INVALID_LOCALE", err.Error())
		default:
			sendJSONError(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		}
		return
	}

	ratio, isAccessible := repository.CalculateContrastRatio(b.PrimaryColor)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"branding":               b,
		"primary_contrast_ratio": ratio,
		"is_accessible":          isAccessible,
	})
}

// UploadLogo handles POST /api/v1/settings/branding/logo (Tenant-scoped, RoleAdmin)
func (h *BrandingHandler) UploadLogo(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())

	// 1. Size restriction: 2MB max
	r.Body = http.MaxBytesReader(w, r.Body, 2*1024*1024)
	if err := r.ParseMultipartForm(2 * 1024 * 1024); err != nil {
		sendJSONError(w, http.StatusBadRequest, "FILE_TOO_LARGE", "Logo file size exceeds the 2MB limit")
		return
	}

	file, header, err := r.FormFile("logo")
	if err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_FILE", "Missing 'logo' form file")
		return
	}
	defer file.Close()

	// 2. Read first 512 bytes to sniff actual MIME type
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		sendJSONError(w, http.StatusBadRequest, "INVALID_FILE", "Failed reading file header")
		return
	}

	detectedMIME := http.DetectContentType(buf[:n])
	var ext string

	switch {
	case strings.HasPrefix(detectedMIME, "image/png"):
		ext = "png"
	case strings.HasPrefix(detectedMIME, "image/jpeg"):
		ext = "jpg"
	case strings.HasPrefix(detectedMIME, "image/webp"):
		ext = "webp"
	case strings.Contains(strings.ToLower(header.Filename), ".svg") || strings.Contains(detectedMIME, "svg") || strings.Contains(detectedMIME, "xml"):
		// SVG uploads remain strictly disabled for security
		sendJSONError(w, http.StatusBadRequest, "UNSUPPORTED_FORMAT", "SVG format is disabled for security reasons. Please upload PNG, JPEG or WebP.")
		return
	default:
		sendJSONError(w, http.StatusBadRequest, "UNSUPPORTED_FORMAT", "Unsupported image format. Allowed: PNG, JPEG, WebP")
		return
	}

	// 3. Image dimension & validity check
	fullReader := io.MultiReader(bytes.NewReader(buf[:n]), file)
	var fullBuf bytes.Buffer
	if _, err := io.Copy(&fullBuf, fullReader); err != nil {
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to buffer file")
		return
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(fullBuf.Bytes()))
	if err != nil {
		sendJSONError(w, http.StatusBadRequest, "CORRUPT_IMAGE", "File is not a valid or decodeable image")
		return
	}

	if cfg.Width > 2500 || cfg.Height > 2500 {
		sendJSONError(w, http.StatusBadRequest, "DIMENSIONS_TOO_LARGE", "Image dimensions exceed maximum allowed size (2500x2500 pixels)")
		return
	}

	// 4. Generate random storage key (never expose user paths or raw filenames)
	var randomBytes [16]byte
	_, _ = rand.Read(randomBytes[:])
	key := fmt.Sprintf("logos/%s/%s.%s", tc.OrganizationID, hex.EncodeToString(randomBytes[:]), ext)

	if err := h.storage.Save(r.Context(), key, bytes.NewReader(fullBuf.Bytes()), detectedMIME); err != nil {
		h.logger.Error("failed saving logo to storage", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to save logo to object storage")
		return
	}

	// 5. Update database record
	if err := h.repo.UpdateLogoStorageKey(r.Context(), tc.OrganizationID, &key); err != nil {
		_ = h.storage.Delete(r.Context(), key)
		h.logger.Error("failed updating logo storage key", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update branding logo record")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "uploaded",
		"storage_key": key,
		"mime_type":   detectedMIME,
		"width":       cfg.Width,
		"height":      cfg.Height,
	})
}

// DeleteLogo handles DELETE /api/v1/settings/branding/logo (Tenant-scoped, RoleAdmin)
func (h *BrandingHandler) DeleteLogo(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())

	b, err := h.repo.GetBranding(r.Context(), tc.OrganizationID)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get branding")
		return
	}

	if b.LogoStorageKey != nil && *b.LogoStorageKey != "" {
		_ = h.storage.Delete(r.Context(), *b.LogoStorageKey)
	}

	if err := h.repo.UpdateLogoStorageKey(r.Context(), tc.OrganizationID, nil); err != nil {
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to clear logo record")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "deleted",
	})
}

// GetLogoFile handles GET /api/v1/settings/branding/logo-file (Tenant-scoped)
func (h *BrandingHandler) GetLogoFile(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())

	b, err := h.repo.GetBranding(r.Context(), tc.OrganizationID)
	if err != nil || b.LogoStorageKey == nil || *b.LogoStorageKey == "" {
		sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "No logo configured")
		return
	}

	rc, mime, err := h.storage.Get(r.Context(), *b.LogoStorageKey)
	if err != nil {
		sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Logo file not found in storage")
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = io.Copy(w, rc)
}

