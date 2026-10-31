package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"clientpulse/services/api/internal/config"
	"clientpulse/services/api/internal/database"
	"clientpulse/services/api/internal/health"
	"clientpulse/services/api/internal/logger"
	"clientpulse/services/api/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		// Cannot use structured logger yet if config load fails
		println("fatal: failed to load configuration:", err.Error())
		os.Exit(1)
	}

	log := logger.New(cfg.AppEnv, os.Stdout)
	log.Info("initializing ClientPulse API service",
		"env", cfg.AppEnv,
		"version", "1.0.0",
		"port", cfg.Port,
	)

	// Connect to PostgreSQL database pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.New(ctx, database.Config{
		DatabaseURL: cfg.DatabaseURL,
	})
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		// We still start the server so /health/live works and /health/ready accurately reports 503
	} else {
		defer db.Close()
	}

	var workerChecker health.WorkerHeartbeatChecker
	if db != nil && db.Pool != nil {
		workerChecker = health.NewPGWorkerHeartbeat(db.Pool)
		log.Info("database connection pool initialized")
	}

	// Initialize health handler with persistent worker heartbeat checker
	workerTimeout := time.Duration(cfg.WorkerHeartbeatTimeoutSeconds) * time.Second
	healthHdr := health.NewHandler(db, workerChecker, workerTimeout, "1.0.0")

	// Initialize HTTP server
	srv := server.New(cfg, log, healthHdr)

	// Run server in background goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrChan <- err
		}
	}()

	// Wait for OS shutdown signal or server error
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrChan:
		log.Error("server encountered fatal error", "error", err)
		os.Exit(1)
	case sig := <-sigChan:
		log.Info("received termination signal, initiating graceful shutdown", "signal", sig.String())
	}

	// Graceful shutdown context
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	log.Info("API service shutdown completed successfully")
}

