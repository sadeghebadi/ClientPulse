package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"clientpulse/services/api/internal/config"
	"clientpulse/services/api/internal/database"
	"clientpulse/services/api/internal/health"
	"clientpulse/services/api/internal/logger"
	"clientpulse/services/api/internal/notification/email"
	"clientpulse/services/api/internal/notification/telegram"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/ssrf"
	"clientpulse/services/api/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		println("fatal: worker failed to load configuration:", err.Error())
		os.Exit(1)
	}

	log := logger.New(cfg.AppEnv, os.Stdout)

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "worker-host"
	}
	pid := os.Getpid()
	workerID := os.Getenv("WORKER_ID")
	if workerID == "" {
		workerID = fmt.Sprintf("worker-%s-%d", hostname, pid)
	}

	log.Info("initializing ClientPulse Worker service",
		"env", cfg.AppEnv,
		"version", "1.0.0",
		"worker_id", workerID,
		"hostname", hostname,
		"pid", pid,
		"concurrency", cfg.WorkerConcurrency,
	)

	// Connect to database pool using dedicated Worker role credentials
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workerDBURL := cfg.WorkerDatabaseURL
	if workerDBURL == "" {
		workerDBURL = cfg.DatabaseURL
	}

	db, err := database.New(ctx, database.Config{
		DatabaseURL: workerDBURL,
	})
	if err != nil {
		log.Error("worker failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Info("worker database pool established")

	// Set up worker heartbeat recorder
	hbRecorder := health.NewPGWorkerHeartbeat(db.Pool)
	regCtx, regCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := hbRecorder.Register(regCtx, workerID, hostname, pid); err != nil {
		log.Warn("initial worker heartbeat registration failed (migrations may be pending)", "error", err)
	} else {
		log.Info("worker heartbeat registered in database", "worker_id", workerID)
	}
	regCancel()

	// Set up monitoring repositories and scheduler
	monitorRepo := repository.NewMonitorRepository(db.Pool)
	resultRepo := repository.NewCheckResultRepository(db.Pool)
	heartbeatRepo := repository.NewHeartbeatRepository(db.Pool)
	incidentRepo := repository.NewIncidentRepository(db.Pool)
	ssrfGuard := ssrf.NewGuard(nil)

	scheduler := worker.NewScheduler(
		db.Pool,
		monitorRepo,
		resultRepo,
		heartbeatRepo,
		incidentRepo,
		ssrfGuard,
		worker.SchedulerConfig{
			WorkerID:      workerID,
			Concurrency:   cfg.WorkerConcurrency,
			PollPeriod:    1 * time.Second,
			RetentionDays: cfg.CheckResultRetentionDays,
		},
		log,
	)

	// Set up notification delivery worker
	notifRepo := repository.NewNotificationDestRepository(db.Pool)
	var botAPI telegram.TelegramBotAPI
	if cfg.TelegramBotToken != "" {
		botAPI = telegram.NewClientWithProxy(
			cfg.TelegramBotToken,
			"",
			cfg.TelegramProxyURL,
			time.Duration(cfg.TelegramPollTimeoutSeconds+15)*time.Second,
		)
	} else {
		log.Info("telegram bot token not configured; running in mock telegram mode")
		botAPI = telegram.NewMockSender()
	}
	emailClient := email.NewClient(email.SMTPConfig{
		Host:      cfg.SMTPHost,
		Port:      cfg.SMTPPort,
		Username:  cfg.SMTPUsername,
		Password:  cfg.SMTPPassword,
		FromEmail: cfg.SMTPFromEmail,
		FromName:  cfg.SMTPFromName,
		TLSMode:   cfg.SMTPTLSMode,
	})
	notifWorker := worker.NewNotificationWorker(
		notifRepo,
		botAPI,
		emailClient,
		cfg.AppBaseURL,
		workerID,
		cfg.WorkerConcurrency,
		log,
	)

	// Set up cancellation context for graceful shutdown
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	// Launch monitoring scheduler daemon
	schedulerErrChan := make(chan error, 1)
	go func() {
		if err := scheduler.Run(workerCtx); err != nil {
			log.Error("scheduler daemon error", "error", err)
			schedulerErrChan <- err
		}
	}()

	// Launch notification delivery daemon
	go func() {
		notifWorker.Start(workerCtx)
	}()

	// Launch Telegram Long-Polling daemon if configured
	if strings.EqualFold(cfg.TelegramMode, "polling") {
		if cfg.TelegramBotToken == "" {
			log.Warn("TELEGRAM_MODE is polling but TELEGRAM_BOT_TOKEN is empty; polling poller disabled")
		} else {
			telegramUpdateHdr := telegram.NewUpdateHandler(notifRepo, botAPI, cfg.TelegramBotUsername, log)
			telegramPoller := worker.NewTelegramPoller(
				db.Pool,
				notifRepo,
				botAPI,
				telegramUpdateHdr,
				cfg.TelegramBotUsername,
				cfg.TelegramPollTimeoutSeconds,
				log,
			)
			go func() {
				telegramPoller.Start(workerCtx)
			}()
		}
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Worker heartbeat loop (heartbeat every 2 seconds, well within 10s timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	log.Info("worker daemon started, queue polling and heartbeat active")

	running := true
	for running {
		select {
		case sig := <-sigChan:
			log.Info("worker received termination signal", "signal", sig.String())
			running = false
			workerCancel()
		case <-ticker.C:
			// Refresh persistent worker heartbeat in database
			hbCtx, hbCancel := context.WithTimeout(workerCtx, 2*time.Second)
			if err := hbRecorder.Heartbeat(hbCtx, workerID); err != nil {
				log.Warn("worker heartbeat refresh failed", "error", err)
			} else {
				log.Debug("worker heartbeat updated", "worker_id", workerID)
			}
			hbCancel()
		}
	}

	// Mark worker stopped in database upon graceful shutdown
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer stopCancel()
	if err := hbRecorder.Stop(stopCtx, workerID); err != nil {
		log.Warn("failed to mark worker stopped in database", "error", err)
	} else {
		log.Info("worker marked stopped in database")
	}

	log.Info("worker process gracefully stopped")
}
