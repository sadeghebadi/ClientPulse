package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"clientpulse/services/api/internal/checker"
	"clientpulse/services/api/internal/entitlement"
	"clientpulse/services/api/internal/rate"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/ssrf"
	"clientpulse/services/api/internal/tenant"
)

type MonitorHandler struct {
	monitorRepo    *repository.MonitorRepository
	resultRepo     *repository.CheckResultRepository
	assetRepo      *repository.AssetRepository
	auditRepo      *repository.AuditRepository
	checker        *checker.Checker
	ssrfGuard      *ssrf.Guard
	entitlements   *entitlement.Service
	testNowLimiter *rate.Limiter
}

func NewMonitorHandler(
	monitorRepo *repository.MonitorRepository,
	resultRepo *repository.CheckResultRepository,
	assetRepo *repository.AssetRepository,
	auditRepo *repository.AuditRepository,
	chk *checker.Checker,
	ssrfGuard *ssrf.Guard,
) *MonitorHandler {
	if ssrfGuard == nil {
		ssrfGuard = ssrf.NewGuard(nil)
	}
	if chk == nil {
		chk = checker.NewChecker(ssrfGuard, "api-manual-worker")
	}

	return &MonitorHandler{
		monitorRepo:    monitorRepo,
		resultRepo:     resultRepo,
		assetRepo:      assetRepo,
		auditRepo:      auditRepo,
		checker:        chk,
		ssrfGuard:      ssrfGuard,
		testNowLimiter: rate.NewLimiter(10, 1*time.Minute), // Rate limit: max 10 Test Now checks per minute
	}
}

// WithEntitlements configures entitlement enforcement for monitor creation and interval restrictions.
func (h *MonitorHandler) WithEntitlements(svc *entitlement.Service) *MonitorHandler {
	h.entitlements = svc
	return h
}

type CreateMonitorRequest struct {
	AssetID           string  `json:"asset_id"`
	Method            string  `json:"method"`
	URLOverride       *string `json:"url_override,omitempty"`
	IntervalSeconds   int     `json:"interval_seconds"`
	TimeoutSeconds    int     `json:"timeout_seconds"`
	ExpectedStatusMin int     `json:"expected_status_min"`
	ExpectedStatusMax int     `json:"expected_status_max"`
	FollowRedirects   *bool   `json:"follow_redirects,omitempty"`
	MaxRedirects      int     `json:"max_redirects"`
	ExpectedText      *string `json:"expected_text,omitempty"`
	JSONPointer       *string `json:"json_pointer,omitempty"`
	JSONExpectedValue *string `json:"json_expected_value,omitempty"`
}

type UpdateMonitorRequest struct {
	Method            string  `json:"method"`
	URLOverride       *string `json:"url_override,omitempty"`
	IntervalSeconds   int     `json:"interval_seconds"`
	TimeoutSeconds    int     `json:"timeout_seconds"`
	ExpectedStatusMin int     `json:"expected_status_min"`
	ExpectedStatusMax int     `json:"expected_status_max"`
	FollowRedirects   *bool   `json:"follow_redirects,omitempty"`
	MaxRedirects      int     `json:"max_redirects"`
	ExpectedText      *string `json:"expected_text,omitempty"`
	JSONPointer       *string `json:"json_pointer,omitempty"`
	JSONExpectedValue *string `json:"json_expected_value,omitempty"`
}

func (h *MonitorHandler) Create(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())

	var req CreateMonitorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON request body")
		return
	}

	req.AssetID = strings.TrimSpace(req.AssetID)
	if req.AssetID == "" {
		sendJSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Asset ID is required")
		return
	}

	// Validate SSRF on URL override if present
	if req.URLOverride != nil && *req.URLOverride != "" {
		trimmedURL := strings.TrimSpace(*req.URLOverride)
		if _, err := h.ssrfGuard.ValidateURL(trimmedURL); err != nil {
			sendJSONError(w, http.StatusBadRequest, "SSRF_PROHIBITED_URL", err.Error())
			return
		}
		req.URLOverride = &trimmedURL
	}

	if h.entitlements != nil {
		if err := h.entitlements.CheckMonitorCreation(r.Context(), tc.OrganizationID, req.IntervalSeconds); err != nil {
			sendJSONError(w, http.StatusForbidden, "LIMIT_EXCEEDED", err.Error())
			return
		}
	}

	followRedirects := true
	if req.FollowRedirects != nil {
		followRedirects = *req.FollowRedirects
	}

	monitor, err := h.monitorRepo.CreateMonitor(r.Context(), tc.OrganizationID, repository.CreateMonitorParams{
		AssetID:           req.AssetID,
		Method:            req.Method,
		URLOverride:       req.URLOverride,
		IntervalSeconds:   req.IntervalSeconds,
		TimeoutSeconds:    req.TimeoutSeconds,
		ExpectedStatusMin: req.ExpectedStatusMin,
		ExpectedStatusMax: req.ExpectedStatusMax,
		FollowRedirects:   followRedirects,
		MaxRedirects:      req.MaxRedirects,
		ExpectedText:      req.ExpectedText,
		JSONPointer:       req.JSONPointer,
		JSONExpectedValue: req.JSONExpectedValue,
	})
	if err != nil {
		if errors.Is(err, repository.ErrAssetNotInOrg) {
			sendJSONError(w, http.StatusBadRequest, "ASSET_NOT_IN_ORGANIZATION", "The specified asset does not belong to your organization")
			return
		}
		if errors.Is(err, repository.ErrInvalidMethod) || errors.Is(err, repository.ErrInvalidTimeout) || errors.Is(err, repository.ErrInvalidInterval) || errors.Is(err, repository.ErrInvalidStatusRng) {
			sendJSONError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create monitor")
		return
	}

	if h.auditRepo != nil {
		meta, _ := json.Marshal(map[string]interface{}{"monitor_id": monitor.ID, "asset_id": monitor.AssetID})
		_ = h.auditRepo.RecordEvent(r.Context(), &tc.OrganizationID, &uc.UserID, "monitor.created", "monitor", monitor.ID, meta, r.RemoteAddr)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"monitor": monitor,
	})
}

func (h *MonitorHandler) Get(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	monitorID := strings.TrimSpace(r.PathValue("id"))
	if monitorID == "" {
		sendJSONError(w, http.StatusBadRequest, "INVALID_MONITOR_ID", "Monitor ID is required")
		return
	}

	monitor, err := h.monitorRepo.GetMonitorByID(r.Context(), tc.OrganizationID, monitorID)
	if err != nil {
		if errors.Is(err, repository.ErrMonitorNotFound) {
			sendJSONError(w, http.StatusNotFound, "MONITOR_NOT_FOUND", "Monitor not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch monitor")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"monitor": monitor,
	})
}

func (h *MonitorHandler) List(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())

	q := r.URL.Query()
	assetID := q.Get("asset_id")
	clientID := q.Get("client_id")
	status := q.Get("status")
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	monitors, total, err := h.monitorRepo.ListMonitors(r.Context(), tc.OrganizationID, assetID, clientID, status, limit, offset)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list monitors")
		return
	}

	if monitors == nil {
		monitors = []*repository.Monitor{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"monitors": monitors,
		"total":    total,
	})
}

func (h *MonitorHandler) Update(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())
	monitorID := strings.TrimSpace(r.PathValue("id"))

	var req UpdateMonitorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON request body")
		return
	}

	if req.URLOverride != nil && *req.URLOverride != "" {
		trimmedURL := strings.TrimSpace(*req.URLOverride)
		if _, err := h.ssrfGuard.ValidateURL(trimmedURL); err != nil {
			sendJSONError(w, http.StatusBadRequest, "SSRF_PROHIBITED_URL", err.Error())
			return
		}
		req.URLOverride = &trimmedURL
	}

	if h.entitlements != nil {
		if err := h.entitlements.CheckMonitorInterval(r.Context(), tc.OrganizationID, req.IntervalSeconds); err != nil {
			sendJSONError(w, http.StatusForbidden, "LIMIT_EXCEEDED", err.Error())
			return
		}
	}

	followRedirects := true
	if req.FollowRedirects != nil {
		followRedirects = *req.FollowRedirects
	}

	monitor, err := h.monitorRepo.UpdateMonitor(r.Context(), tc.OrganizationID, monitorID, repository.UpdateMonitorParams{
		Method:            req.Method,
		URLOverride:       req.URLOverride,
		IntervalSeconds:   req.IntervalSeconds,
		TimeoutSeconds:    req.TimeoutSeconds,
		ExpectedStatusMin: req.ExpectedStatusMin,
		ExpectedStatusMax: req.ExpectedStatusMax,
		FollowRedirects:   followRedirects,
		MaxRedirects:      req.MaxRedirects,
		ExpectedText:      req.ExpectedText,
		JSONPointer:       req.JSONPointer,
		JSONExpectedValue: req.JSONExpectedValue,
	})
	if err != nil {
		if errors.Is(err, repository.ErrMonitorNotFound) {
			sendJSONError(w, http.StatusNotFound, "MONITOR_NOT_FOUND", "Monitor not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update monitor")
		return
	}

	if h.auditRepo != nil {
		meta, _ := json.Marshal(map[string]interface{}{"monitor_id": monitor.ID})
		_ = h.auditRepo.RecordEvent(r.Context(), &tc.OrganizationID, &uc.UserID, "monitor.updated", "monitor", monitor.ID, meta, r.RemoteAddr)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"monitor": monitor,
	})
}

func (h *MonitorHandler) Pause(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())
	monitorID := strings.TrimSpace(r.PathValue("id"))

	err := h.monitorRepo.PauseMonitor(r.Context(), tc.OrganizationID, monitorID)
	if err != nil {
		if errors.Is(err, repository.ErrMonitorNotFound) {
			sendJSONError(w, http.StatusNotFound, "MONITOR_NOT_FOUND", "Monitor not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to pause monitor")
		return
	}

	if h.auditRepo != nil {
		_ = h.auditRepo.RecordEvent(r.Context(), &tc.OrganizationID, &uc.UserID, "monitor.paused", "monitor", monitorID, []byte("{}"), r.RemoteAddr)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "paused",
		"message": "Monitor paused successfully",
	})
}

func (h *MonitorHandler) Resume(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())
	monitorID := strings.TrimSpace(r.PathValue("id"))

	err := h.monitorRepo.ResumeMonitor(r.Context(), tc.OrganizationID, monitorID)
	if err != nil {
		if errors.Is(err, repository.ErrMonitorNotFound) {
			sendJSONError(w, http.StatusNotFound, "MONITOR_NOT_FOUND", "Monitor not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to resume monitor")
		return
	}

	if h.auditRepo != nil {
		_ = h.auditRepo.RecordEvent(r.Context(), &tc.OrganizationID, &uc.UserID, "monitor.resumed", "monitor", monitorID, []byte("{}"), r.RemoteAddr)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "pending",
		"message": "Monitor resumed successfully and scheduled for check",
	})
}

func (h *MonitorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())
	monitorID := strings.TrimSpace(r.PathValue("id"))

	err := h.monitorRepo.DeleteMonitor(r.Context(), tc.OrganizationID, monitorID)
	if err != nil {
		if errors.Is(err, repository.ErrMonitorNotFound) {
			sendJSONError(w, http.StatusNotFound, "MONITOR_NOT_FOUND", "Monitor not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete monitor")
		return
	}

	if h.auditRepo != nil {
		_ = h.auditRepo.RecordEvent(r.Context(), &tc.OrganizationID, &uc.UserID, "monitor.deleted", "monitor", monitorID, []byte("{}"), r.RemoteAddr)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "deleted",
		"message": "Monitor permanently deleted",
	})
}

// TestNow performs an authenticated, rate-limited, manual check on-demand.
func (h *MonitorHandler) TestNow(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())
	monitorID := strings.TrimSpace(r.PathValue("id"))

	// Rate limit check: max 10 tests/min per user
	limitKey := "test-now:" + uc.UserID
	if !h.testNowLimiter.Allow(limitKey) {
		sendJSONError(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Test Now rate limit exceeded. Please wait a moment before trying again.")
		return
	}

	monitor, err := h.monitorRepo.GetMonitorByID(r.Context(), tc.OrganizationID, monitorID)
	if err != nil {
		if errors.Is(err, repository.ErrMonitorNotFound) {
			sendJSONError(w, http.StatusNotFound, "MONITOR_NOT_FOUND", "Monitor not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch monitor")
		return
	}

	// Target URL resolution
	targetURL := monitor.AssetURL
	if monitor.URLOverride != nil && *monitor.URLOverride != "" {
		targetURL = *monitor.URLOverride
	}

	// Execute manual check
	execResult := h.checker.Execute(r.Context(), checker.CheckConfig{
		TargetURL:         targetURL,
		Method:            monitor.Method,
		TimeoutSeconds:    monitor.TimeoutSeconds,
		ExpectedStatusMin: monitor.ExpectedStatusMin,
		ExpectedStatusMax: monitor.ExpectedStatusMax,
		FollowRedirects:   monitor.FollowRedirects,
		MaxRedirects:      monitor.MaxRedirects,
		ExpectedText:      monitor.ExpectedText,
		JSONPointer:       monitor.JSONPointer,
		JSONExpectedValue: monitor.JSONExpectedValue,
		ScheduledAt:       time.Now().UTC(),
	})

	// Persist result with execution_type = "manual"
	savedResult, err := h.resultRepo.CreateCheckResult(r.Context(), repository.CreateCheckResultParams{
		OrganizationID:   tc.OrganizationID,
		MonitorID:        monitor.ID,
		ScheduledAt:      execResult.StartedAt,
		StartedAt:        execResult.StartedAt,
		CompletedAt:      execResult.CompletedAt,
		LatencyMs:        execResult.LatencyMs,
		Result:           execResult.Result,
		HTTPStatusCode:   execResult.HTTPStatusCode,
		ErrorCategory:    execResult.ErrorCategory,
		ErrorMessage:     execResult.ErrorMessage,
		TLSVersion:       execResult.TLSVersion,
		TLSCipher:        execResult.TLSCipher,
		TLSCertIssuer:    execResult.TLSCertIssuer,
		TLSCertSubject:   execResult.TLSCertSubject,
		TLSCertNotBefore: execResult.TLSCertNotBefore,
		TLSCertNotAfter:  execResult.TLSCertNotAfter,
		TLSDaysRemaining: execResult.TLSDaysRemaining,
		TLSIsValid:       execResult.TLSIsValid,
		ResolvedIP:       execResult.ResolvedIP,
		WorkerID:         "manual-api",
		ExecutionType:    "manual",
	})
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to store test result")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"result": savedResult,
	})
}

// History returns check execution history for a monitor.
func (h *MonitorHandler) History(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	monitorID := strings.TrimSpace(r.PathValue("id"))

	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	results, total, err := h.resultRepo.ListMonitorResults(r.Context(), tc.OrganizationID, monitorID, limit, offset)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch monitor history")
		return
	}

	if results == nil {
		results = []*repository.CheckResult{}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
		"total":   total,
	})
}

// DashboardStats returns aggregated workspace statistics.
func (h *MonitorHandler) DashboardStats(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())

	stats, err := h.resultRepo.GetDashboardStats(r.Context(), tc.OrganizationID)
	if err != nil {
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch dashboard stats")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(stats)
}
