package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"clientpulse/services/api/internal/config"
	"clientpulse/services/api/internal/health"
)

// Server encapsulates the HTTP server, configuration, and dependencies.
type Server struct {
	httpServer *http.Server
	cfg        *config.Config
	logger     *slog.Logger
	healthHdr  *health.Handler
}

// New constructs a configured Server instance.
func New(cfg *config.Config, log *slog.Logger, healthHdr *health.Handler) *Server {
	s := &Server{
		cfg:       cfg,
		logger:    log,
		healthHdr: healthHdr,
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	// Build middleware chain
	var handler http.Handler = mux
	handler = CORS([]string{cfg.AppBaseURL, "http://localhost:3000"})(handler)
	handler = SecurityHeaders(handler)
	handler = RequestLogger(log)(handler)
	handler = Recovery(log)(handler)
	handler = RequestID(handler)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Health and probe endpoints
	mux.HandleFunc("GET /health/live", s.healthHdr.Live)
	mux.HandleFunc("GET /health/ready", s.healthHdr.Ready)
	mux.HandleFunc("GET /health/worker", s.healthHdr.Worker)

	// API v1 base info endpoint (foundation verification)
	mux.HandleFunc("GET /api/v1/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"service":     "ClientPulse API",
			"version":     "1.0.0",
			"phase":       "Phase 0 Foundation",
			"environment": s.cfg.AppEnv,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		})
	})
}

// Start listens and serves incoming HTTP requests.
func (s *Server) Start() error {
	s.logger.Info("starting HTTP API server",
		"port", s.cfg.Port,
		"env", s.cfg.AppEnv,
		"base_url", s.cfg.APIBaseURL,
	)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server failure: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down HTTP API server")
	return s.httpServer.Shutdown(ctx)
}

