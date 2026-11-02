package health

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"time"
)

// Pinger defines an interface for components capable of health verification.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Handler provides HTTP endpoints for liveness, readiness, worker, and scheduler probes.
type Handler struct {
	db               Pinger
	workerChecker    WorkerHeartbeatChecker
	workerTimeout    time.Duration
	schedulerChecker SchedulerHealthService
	version          string
}

// NewHandler constructs a new health handler.
func NewHandler(db Pinger, workerChecker WorkerHeartbeatChecker, workerTimeout time.Duration, version string) *Handler {
	if version == "" {
		version = "1.0.0"
	}
	if workerTimeout <= 0 {
		workerTimeout = 10 * time.Second
	}
	return &Handler{
		db:            db,
		workerChecker: workerChecker,
		workerTimeout: workerTimeout,
		version:       version,
	}
}

// WithSchedulerChecker sets the optional scheduler health service.
func (h *Handler) WithSchedulerChecker(sc SchedulerHealthService) *Handler {
	h.schedulerChecker = sc
	return h
}

// LiveResponse defines the payload for liveness check.
type LiveResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

// ReadyResponse defines the payload for readiness check.
type ReadyResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Database  string `json:"database"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
	Error     string `json:"error,omitempty"`
}

// WorkerResponse defines the payload for worker status check.
type WorkerResponse struct {
	Status          string   `json:"status"`
	Service         string   `json:"service"`
	WorkerID        string   `json:"worker_id,omitempty"`
	Hostname        string   `json:"hostname,omitempty"`
	LastHeartbeatAt string   `json:"last_heartbeat_at,omitempty"`
	AgeSeconds      *float64 `json:"age_seconds,omitempty"`
	TimeoutSeconds  int      `json:"timeout_seconds"`
	Timestamp       string   `json:"timestamp"`
	SchedulerStatus string   `json:"scheduler_status,omitempty"`
	Error           string   `json:"error,omitempty"`
}

// Live handles GET /health/live.
func (h *Handler) Live(w http.ResponseWriter, r *http.Request) {
	resp := LiveResponse{
		Status:    "ok",
		Service:   "clientpulse-api",
		Version:   h.version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	writeJSON(w, http.StatusOK, resp)
}

// Ready handles GET /health/ready.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	resp := ReadyResponse{
		Service:   "clientpulse-api",
		Version:   h.version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if h.db == nil {
		resp.Status = "unavailable"
		resp.Database = "not_configured"
		writeJSON(w, http.StatusServiceUnavailable, resp)
		return
	}

	if err := h.db.Ping(ctx); err != nil {
		resp.Status = "unavailable"
		resp.Database = "disconnected"
		resp.Error = "database connection ping failed"
		writeJSON(w, http.StatusServiceUnavailable, resp)
		return
	}

	resp.Status = "ok"
	resp.Database = "connected"
	writeJSON(w, http.StatusOK, resp)
}

// Worker handles GET /health/worker by genuinely checking persistent worker heartbeats.
func (h *Handler) Worker(w http.ResponseWriter, r *http.Request) {
	timeoutSec := int(h.workerTimeout.Seconds())
	resp := WorkerResponse{
		Service:        "clientpulse-worker",
		TimeoutSeconds: timeoutSec,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}

	if h.workerChecker == nil {
		resp.Status = "unhealthy"
		resp.Error = "worker heartbeat checker not configured"
		writeJSON(w, http.StatusServiceUnavailable, resp)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	hb, err := h.workerChecker.GetLatestHeartbeat(ctx)
	if err != nil {
		resp.Status = "unhealthy"
		resp.Error = "failed querying worker heartbeat: " + err.Error()
		writeJSON(w, http.StatusServiceUnavailable, resp)
		return
	}

	if hb == nil {
		resp.Status = "unhealthy"
		resp.Error = "no active worker heartbeats found in database"
		writeJSON(w, http.StatusServiceUnavailable, resp)
		return
	}

	age := time.Since(hb.LastHeartbeatAt)
	ageSec := math.Round(age.Seconds()*100) / 100
	resp.WorkerID = hb.WorkerID
	resp.Hostname = hb.Hostname
	resp.LastHeartbeatAt = hb.LastHeartbeatAt.UTC().Format(time.RFC3339)
	resp.AgeSeconds = &ageSec

	if age > h.workerTimeout {
		resp.Status = "unhealthy"
		resp.Error = "worker heartbeat is stale"
		writeJSON(w, http.StatusServiceUnavailable, resp)
		return
	}

	resp.Status = "ok"
	if h.schedulerChecker != nil {
		schedSnap, _ := h.schedulerChecker.GetSchedulerStatus(ctx)
		resp.SchedulerStatus = string(schedSnap.State)
	}
	writeJSON(w, http.StatusOK, resp)
}

// SchedulerResponse defines the payload for scheduler status check.
type SchedulerResponse struct {
	Status    string                      `json:"status"`
	Service   string                      `json:"service"`
	Scheduler *SchedulerTelemetrySnapshot `json:"scheduler,omitempty"`
	Timestamp string                      `json:"timestamp"`
	Error     string                      `json:"error,omitempty"`
}

// Scheduler handles GET /health/scheduler.
func (h *Handler) Scheduler(w http.ResponseWriter, r *http.Request) {
	resp := SchedulerResponse{
		Service:   "clientpulse-scheduler",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if h.schedulerChecker == nil {
		resp.Status = "unavailable"
		resp.Error = "scheduler health checker not configured"
		writeJSON(w, http.StatusServiceUnavailable, resp)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	snap, err := h.schedulerChecker.GetSchedulerStatus(ctx)
	if err != nil {
		resp.Status = "unhealthy"
		resp.Error = "failed to evaluate scheduler status: " + err.Error()
		writeJSON(w, http.StatusServiceUnavailable, resp)
		return
	}

	resp.Scheduler = &snap
	switch snap.State {
	case SchedulerStateHealthy:
		resp.Status = "ok"
		writeJSON(w, http.StatusOK, resp)
	case SchedulerStateDegraded:
		resp.Status = "degraded"
		writeJSON(w, http.StatusOK, resp)
	case SchedulerStateStale:
		resp.Status = "stale"
		writeJSON(w, http.StatusServiceUnavailable, resp)
	default:
		resp.Status = "unknown"
		writeJSON(w, http.StatusOK, resp)
	}
}

func writeJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}
