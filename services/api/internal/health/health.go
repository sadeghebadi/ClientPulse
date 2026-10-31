package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Pinger defines an interface for components capable of health verification.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Handler provides HTTP endpoints for liveness, readiness, and worker probes.
type Handler struct {
	db      Pinger
	version string
}

// NewHandler constructs a new health handler.
func NewHandler(db Pinger, version string) *Handler {
	if version == "" {
		version = "1.0.0"
	}
	return &Handler{
		db:      db,
		version: version,
	}
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
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
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

// Worker handles GET /health/worker.
func (h *Handler) Worker(w http.ResponseWriter, r *http.Request) {
	resp := WorkerResponse{
		Status:    "ok",
		Service:   "clientpulse-worker",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

