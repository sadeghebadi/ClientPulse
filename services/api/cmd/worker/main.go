package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"clientpulse/services/api/internal/config"
	"clientpulse/services/api/internal/database"
	"clientpulse/services/api/internal/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		println("fatal: worker failed to load configuration:", err.Error())
		os.Exit(1)
	}

	log := logger.New(cfg.AppEnv, os.Stdout)
	log.Info("initializing ClientPulse Worker service",
		"env", cfg.AppEnv,
		"version", "1.0.0",
		"concurrency", cfg.WorkerConcurrency,
	)

	// Connect to database pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := database.New(ctx, database.Config{
		DatabaseURL: cfg.DatabaseURL,
	})
	if err != nil {
		log.Error("worker failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Info("worker database pool established")

	// Set up cancellation context for graceful shutdown
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Worker heartbeat loop
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Info("worker daemon started, waiting for jobs")

	running := true
	for running {
		select {
		case sig := <-sigChan:
			log.Info("worker received termination signal", "signal", sig.String())
			running = false
			workerCancel()
		case <-ticker.C:
			// Foundation heartbeat: verify database connection
			pingCtx, pingCancel := context.WithTimeout(workerCtx, 2*time.Second)
			if err := db.Ping(pingCtx); err != nil {
				log.Warn("worker database ping failed", "error", err)
			} else {
				log.Debug("worker active, queue polling ready")
			}
			pingCancel()
		}
	}

	log.Info("worker process gracefully stopped")
}

