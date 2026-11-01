package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"clientpulse/services/api/internal/pdf"
	"clientpulse/services/api/internal/rate"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/storage"
	"clientpulse/services/api/internal/tenant"
)

type ReportHandler struct {
	repo         *repository.ReportRepository
	brandingRepo *repository.BrandingRepository
	storage      storage.Storage
	pdfGen       *pdf.Generator
	logger       *slog.Logger
	pubLimiter   *rate.Limiter
}

func NewReportHandler(
	repo *repository.ReportRepository,
	brandingRepo *repository.BrandingRepository,
	st storage.Storage,
	pdfGen *pdf.Generator,
	logger *slog.Logger,
) *ReportHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &ReportHandler{
		repo:         repo,
		brandingRepo: brandingRepo,
		storage:      st,
		pdfGen:       pdfGen,
		logger:       logger,
		pubLimiter:   rate.NewLimiter(30, 1*time.Minute), // 30 requests per minute per IP for public endpoints
	}
}

// DefaultPreviousMonth calculates the start and end of previous calendar month in the specified timezone.
func DefaultPreviousMonth(tzName string) (time.Time, time.Time, string) {
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		loc = time.UTC
		tzName = "UTC"
	}

	now := time.Now().In(loc)
	firstOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	endOfPrevMonth := firstOfCurrentMonth.Add(-time.Nanosecond)
	firstOfPrevMonth := time.Date(endOfPrevMonth.Year(), endOfPrevMonth.Month(), 1, 0, 0, 0, 0, loc)

	return firstOfPrevMonth.UTC(), endOfPrevMonth.UTC(), tzName
}

// Preview handles POST /api/v1/reports/preview (Tenant-scoped)
func (h *ReportHandler) Preview(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())

	var req struct {
		ClientID    string `json:"client_id"`
		PeriodStart string `json:"period_start,omitempty"`
		PeriodEnd   string `json:"period_end,omitempty"`
		Locale      string `json:"locale,omitempty"`
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

	var pStart, pEnd time.Time
	var tzName string

	if req.PeriodStart != "" && req.PeriodEnd != "" {
		s, err1 := time.Parse(time.RFC3339, req.PeriodStart)
		e, err2 := time.Parse(time.RFC3339, req.PeriodEnd)
		if err1 != nil || err2 != nil {
			sendJSONError(w, http.StatusBadRequest, "INVALID_DATE_RANGE", "period_start and period_end must be RFC3339 timestamps")
			return
		}
		pStart = s.UTC()
		pEnd = e.UTC()
		tzName = "UTC"
	} else {
		pStart, pEnd, tzName = DefaultPreviousMonth("UTC")
	}

	if !pEnd.After(pStart) {
		sendJSONError(w, http.StatusBadRequest, "INVALID_PERIOD", "period_end must be strictly after period_start")
		return
	}

	data, err := h.repo.CalculateReportData(r.Context(), tc.OrganizationID, req.ClientID, pStart, pEnd, tzName, req.Locale)
	if err != nil {
		if errors.Is(err, repository.ErrClientNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Client not found in active organization")
			return
		}
		h.logger.Error("failed calculating report preview", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to calculate report preview")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"preview": data,
	})
}

// Create handles POST /api/v1/reports (Tenant-scoped, RoleMember)
func (h *ReportHandler) Create(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())

	var req struct {
		ClientID    string `json:"client_id"`
		PeriodStart string `json:"period_start,omitempty"`
		PeriodEnd   string `json:"period_end,omitempty"`
		Locale      string `json:"locale,omitempty"`
		Title       string `json:"title,omitempty"`
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

	var pStart, pEnd time.Time
	var tzName string

	if req.PeriodStart != "" && req.PeriodEnd != "" {
		s, err1 := time.Parse(time.RFC3339, req.PeriodStart)
		e, err2 := time.Parse(time.RFC3339, req.PeriodEnd)
		if err1 != nil || err2 != nil {
			sendJSONError(w, http.StatusBadRequest, "INVALID_DATE_RANGE", "period_start and period_end must be RFC3339 timestamps")
			return
		}
		pStart = s.UTC()
		pEnd = e.UTC()
		tzName = "UTC"
	} else {
		pStart, pEnd, tzName = DefaultPreviousMonth("UTC")
	}

	if !pEnd.After(pStart) {
		sendJSONError(w, http.StatusBadRequest, "INVALID_PERIOD", "period_end must be strictly after period_start")
		return
	}

	// 1. Deterministically calculate report snapshot
	data, err := h.repo.CalculateReportData(r.Context(), tc.OrganizationID, req.ClientID, pStart, pEnd, tzName, req.Locale)
	if err != nil {
		if errors.Is(err, repository.ErrClientNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Client not found in active organization")
			return
		}
		h.logger.Error("failed calculating report data for persistence", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to calculate report data")
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = data.Title
	} else {
		data.Title = title
	}

	// 2. Persist new immutable report version
	rep, err := h.repo.CreateReport(
		r.Context(), tc.OrganizationID, req.ClientID,
		"monthly_care", pStart, pEnd, tzName, data.Locale,
		title, *data, uc.UserID,
	)
	if err != nil {
		h.logger.Error("failed persisting report version", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create report")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"report": rep,
	})
}

// List handles GET /api/v1/reports (Tenant-scoped)
func (h *ReportHandler) List(w http.ResponseWriter, r *http.Request) {
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

	filter := repository.ReportFilter{
		Limit:  limit,
		Offset: offset,
	}

	if cid := r.URL.Query().Get("client_id"); cid != "" {
		filter.ClientID = &cid
	}
	if rt := r.URL.Query().Get("report_type"); rt != "" {
		filter.ReportType = &rt
	}

	reports, total, err := h.repo.ListReports(r.Context(), tc.OrganizationID, filter)
	if err != nil {
		h.logger.Error("failed listing reports", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list reports")
		return
	}

	if reports == nil {
		reports = []*repository.Report{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"reports": reports,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// Get handles GET /api/v1/reports/{id} (Tenant-scoped)
func (h *ReportHandler) Get(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	rep, err := h.repo.GetReport(r.Context(), tc.OrganizationID, id)
	if err != nil {
		if errors.Is(err, repository.ErrReportNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Report not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get report")
		return
	}

	var snap pdf.ReportData
	_ = json.Unmarshal(rep.SnapshotJSON, &snap)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"report":        rep,
		"snapshot":      snap,
		"has_pdf":       rep.PDFStorageKey != nil && *rep.PDFStorageKey != "",
	})
}

// GeneratePDF handles POST /api/v1/reports/{id}/generate-pdf (Tenant-scoped, RoleMember)
func (h *ReportHandler) GeneratePDF(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	rep, err := h.repo.GetReport(r.Context(), tc.OrganizationID, id)
	if err != nil {
		if errors.Is(err, repository.ErrReportNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Report not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get report")
		return
	}

	var snap pdf.ReportData
	if err := json.Unmarshal(rep.SnapshotJSON, &snap); err != nil {
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Corrupt report snapshot")
		return
	}

	// Embed logo data-uri if organization has logo
	b, err := h.brandingRepo.GetBranding(r.Context(), tc.OrganizationID)
	if err == nil && b.LogoStorageKey != nil && *b.LogoStorageKey != "" {
		rc, mime, err := h.storage.Get(r.Context(), *b.LogoStorageKey)
		if err == nil {
			logoBytes, _ := io.ReadAll(rc)
			_ = rc.Close()
			if len(logoBytes) > 0 {
				snap.Branding.LogoDataURI = fmt.Sprintf("data:%s;base64,%s", mime, base64.StdEncoding.EncodeToString(logoBytes))
			}
		}
	}

	// Render PDF
	pdfBytes, err := h.pdfGen.GeneratePDF(r.Context(), snap)
	if err != nil {
		h.logger.Error("failed generating pdf", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "PDF_GENERATION_FAILED", "Failed to render PDF report")
		return
	}

	// Save to storage
	storageKey := fmt.Sprintf("reports/%s/%s_v%d.pdf", tc.OrganizationID, rep.ID, rep.VersionNumber)
	if err := h.storage.Save(r.Context(), storageKey, bytes.NewReader(pdfBytes), "application/pdf"); err != nil {
		h.logger.Error("failed saving pdf to storage", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to save PDF to storage")
		return
	}

	// Update DB
	_ = h.repo.UpdateReportPDFStorageKey(r.Context(), tc.OrganizationID, rep.ID, storageKey)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "generated",
		"storage_key": storageKey,
		"size_bytes":  len(pdfBytes),
	})
}

// DownloadPDF handles GET /api/v1/reports/{id}/pdf (Tenant-scoped)
func (h *ReportHandler) DownloadPDF(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	rep, err := h.repo.GetReport(r.Context(), tc.OrganizationID, id)
	if err != nil {
		sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Report not found")
		return
	}

	var pdfRc io.ReadCloser
	var mime string

	if rep.PDFStorageKey != nil && *rep.PDFStorageKey != "" {
		rc, m, err := h.storage.Get(r.Context(), *rep.PDFStorageKey)
		if err == nil {
			pdfRc = rc
			mime = m
		}
	}

	if pdfRc == nil {
		// Generate on demand
		var snap pdf.ReportData
		_ = json.Unmarshal(rep.SnapshotJSON, &snap)
		pdfBytes, err := h.pdfGen.GeneratePDF(r.Context(), snap)
		if err != nil {
			sendJSONError(w, http.StatusInternalServerError, "PDF_FAILED", "Failed to render PDF")
			return
		}
		key := fmt.Sprintf("reports/%s/%s_v%d.pdf", tc.OrganizationID, rep.ID, rep.VersionNumber)
		_ = h.storage.Save(r.Context(), key, bytes.NewReader(pdfBytes), "application/pdf")
		_ = h.repo.UpdateReportPDFStorageKey(r.Context(), tc.OrganizationID, rep.ID, key)
		pdfRc = io.NopCloser(bytes.NewReader(pdfBytes))
		mime = "application/pdf"
	}
	defer pdfRc.Close()

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"report-%s-v%d.pdf\"", rep.ID[:8], rep.VersionNumber))
	_, _ = io.Copy(w, pdfRc)
}

// CreateShare handles POST /api/v1/reports/{id}/share (Tenant-scoped, RoleMember)
func (h *ReportHandler) CreateShare(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	var req struct {
		ExpiresInDays *int `json:"expires_in_days"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	_ = json.NewDecoder(r.Body).Decode(&req)

	var expiresAt *time.Time
	if req.ExpiresInDays != nil && *req.ExpiresInDays > 0 {
		exp := time.Now().UTC().Add(time.Duration(*req.ExpiresInDays) * 24 * time.Hour)
		expiresAt = &exp
	}

	tokenPlaintext, shareRec, err := h.repo.CreateShareToken(r.Context(), tc.OrganizationID, id, expiresAt, uc.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrReportNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Report not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create share link")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"share_token": tokenPlaintext,
		"share_url":   fmt.Sprintf("/public/reports/%s", tokenPlaintext),
		"expires_at":  shareRec.ExpiresAt,
		"share_id":    shareRec.ID,
	})
}

// ListShares handles GET /api/v1/reports/{id}/shares (Tenant-scoped)
func (h *ReportHandler) ListShares(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	shares, err := h.repo.ListShareTokens(r.Context(), tc.OrganizationID, id)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list share links")
		return
	}

	if shares == nil {
		shares = []*repository.ReportShareToken{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"shares": shares,
	})
}

// RevokeShare handles POST /api/v1/reports/{id}/revoke-share (Tenant-scoped, RoleMember)
func (h *ReportHandler) RevokeShare(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	id := strings.TrimSpace(r.PathValue("id"))

	var req struct {
		ShareID string `json:"share_id"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.ShareID) == "" {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "share_id is required")
		return
	}

	err := h.repo.RevokeShareToken(r.Context(), tc.OrganizationID, req.ShareID)
	if err != nil {
		if errors.Is(err, repository.ErrShareTokenNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Share link not found or already revoked")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to revoke share link")
		return
	}

	_ = id
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "revoked",
	})
}

// PublicView handles GET /public/reports/{token} (Unauthenticated, Rate-limited)
func (h *ReportHandler) PublicView(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	if !h.pubLimiter.Allow("pub:" + clientIP) {
		sendJSONError(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Rate limit exceeded. Please retry later.")
		return
	}

	token := strings.TrimSpace(r.PathValue("token"))
	if len(token) < 20 {
		sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Report not found or link has expired")
		return
	}

	view, _, err := h.repo.LookupReportByShareToken(r.Context(), token)
	if err != nil {
		sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Report not found or link has expired")
		return
	}

	var snap pdf.ReportData
	_ = json.Unmarshal(view.SnapshotJSON, &snap)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"report": map[string]interface{}{
			"report_type":           view.ReportType,
			"period_start":          view.PeriodStart,
			"period_end":            view.PeriodEnd,
			"organization_timezone": view.OrganizationTimezone,
			"locale":                view.Locale,
			"version_number":        view.VersionNumber,
			"template_version":      view.TemplateVersion,
			"title":                 view.Title,
			"generated_at":          view.GeneratedAt,
			"expires_at":            view.ExpiresAt,
			"has_pdf":               view.HasPDF,
		},
		"snapshot": snap,
	})
}

// PublicPDF handles GET /public/reports/{token}/pdf (Unauthenticated, Rate-limited)
func (h *ReportHandler) PublicPDF(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	if !h.pubLimiter.Allow("pub:" + clientIP) {
		sendJSONError(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Rate limit exceeded. Please retry later.")
		return
	}

	token := strings.TrimSpace(r.PathValue("token"))
	view, storageKey, err := h.repo.LookupReportByShareToken(r.Context(), token)
	if err != nil {
		sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Report not found or link has expired")
		return
	}

	var pdfRc io.ReadCloser
	var mime string

	if storageKey != "" {
		rc, m, err := h.storage.Get(r.Context(), storageKey)
		if err == nil {
			pdfRc = rc
			mime = m
		}
	}

	if pdfRc == nil {
		var snap pdf.ReportData
		_ = json.Unmarshal(view.SnapshotJSON, &snap)
		pdfBytes, err := h.pdfGen.GeneratePDF(r.Context(), snap)
		if err != nil {
			sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to render PDF")
			return
		}
		pdfRc = io.NopCloser(bytes.NewReader(pdfBytes))
		mime = "application/pdf"
	}
	defer pdfRc.Close()

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"report-v%d.pdf\"", view.VersionNumber))
	_, _ = io.Copy(w, pdfRc)
}
