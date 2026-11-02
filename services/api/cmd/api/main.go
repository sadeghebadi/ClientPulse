package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"clientpulse/services/api/internal/checker"
	"clientpulse/services/api/internal/config"
	"clientpulse/services/api/internal/database"
	"clientpulse/services/api/internal/health"
	"clientpulse/services/api/internal/logger"
	"clientpulse/services/api/internal/mail"
	"clientpulse/services/api/internal/notification/email"
	"clientpulse/services/api/internal/notification/telegram"
	"clientpulse/services/api/internal/pdf"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/server"
	"clientpulse/services/api/internal/service"
	"clientpulse/services/api/internal/ssrf"
	"clientpulse/services/api/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
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

	// Assemble server dependencies
	var deps server.Dependencies
	deps.HealthHdr = healthHdr

	if db != nil && db.Pool != nil {
		userRepo := repository.NewUserRepository(db.Pool)
		sessionRepo := repository.NewSessionRepository(db.Pool)
		orgRepo := repository.NewOrganizationRepository(db.Pool)
		tokenRepo := repository.NewTokenRepository(db.Pool)
		auditRepo := repository.NewAuditRepository(db.Pool)
		clientRepo := repository.NewClientRepository(db.Pool)
		assetRepo := repository.NewAssetRepository(db.Pool)
		monitorRepo := repository.NewMonitorRepository(db.Pool)
		resultRepo := repository.NewCheckResultRepository(db.Pool)
		heartbeatRepo := repository.NewHeartbeatRepository(db.Pool)
		incidentRepo := repository.NewIncidentRepository(db.Pool)
		eventRepo := repository.NewNotificationEventRepository(db.Pool)
		ssrfGuard := ssrf.NewGuard(nil)
		chk := checker.NewChecker(ssrfGuard, "api-manual-worker")

		storageDir := os.Getenv("STORAGE_LOCAL_DIR")
		if storageDir == "" {
			storageDir = "./data/storage"
		}
		st, err := storage.NewLocalStorage(storageDir)
		if err != nil {
			log.Error("failed initializing local storage", "error", err)
			os.Exit(1)
		}

		pdfGen, err := pdf.NewGenerator()
		if err != nil {
			log.Warn("pdf generator initialized without chrome binary (generation will fail on demand)", "error", err)
		}

		maintRepo := repository.NewMaintenanceRepository(db.Pool, auditRepo)
		brandingRepo := repository.NewBrandingRepository(db.Pool)
		reportRepo := repository.NewReportRepository(db.Pool, brandingRepo)
		notifRepo := repository.NewNotificationDestRepository(db.Pool)

		telegramClient := telegram.NewClient(cfg.TelegramBotToken, "", nil)
		telegramUpdateHdr := telegram.NewUpdateHandler(notifRepo, telegramClient, cfg.TelegramBotUsername, log)
		emailClient := email.NewClient(email.SMTPConfig{
			Host:      cfg.SMTPHost,
			Port:      cfg.SMTPPort,
			Username:  cfg.SMTPUsername,
			Password:  cfg.SMTPPassword,
			FromEmail: cfg.EmailFrom,
			FromName:  cfg.SMTPFromName,
			TLSMode:   cfg.SMTPTLSMode,
		})

		var mailer mail.EmailSender
		if cfg.SMTPHost != "" {
			mailer = mail.NewSMTPMailer(mail.SMTPConfig{
				Host:     cfg.SMTPHost,
				Port:     cfg.SMTPPort,
				Username: cfg.SMTPUsername,
				Password: cfg.SMTPPassword,
				From:     cfg.EmailFrom,
			})
		} else {
			mailer = mail.NewMockMailer()
		}

		authService := service.NewAuthService(userRepo, sessionRepo, orgRepo, tokenRepo, auditRepo, mailer, cfg)
		orgService := service.NewOrgService(orgRepo, userRepo, sessionRepo, auditRepo, cfg)

		deps.AuthHdr = server.NewAuthHandler(authService, cfg.IsProduction())
		deps.OrgHdr = server.NewOrgHandler(orgService)
		deps.ClientHdr = server.NewClientHandler(clientRepo, auditRepo)
		deps.AssetHdr = server.NewAssetHandler(assetRepo, auditRepo, ssrfGuard, cfg.AppEnv)
		deps.MonitorHdr = server.NewMonitorHandler(monitorRepo, resultRepo, assetRepo, auditRepo, chk, ssrfGuard)
		deps.HeartbeatHdr = server.NewHeartbeatHandler(heartbeatRepo, cfg, log)
		deps.IncidentHdr = server.NewIncidentHandler(incidentRepo, eventRepo, log)
		deps.MaintenanceHdr = server.NewMaintenanceHandler(maintRepo, log)
		deps.BrandingHdr = server.NewBrandingHandler(brandingRepo, st, log)
		deps.ReportHdr = server.NewReportHandler(reportRepo, brandingRepo, st, pdfGen, log)
		deps.NotificationHdr = server.NewNotificationHandler(notifRepo, telegramUpdateHdr, telegramClient, emailClient, cfg, log)
		deps.AuthMiddleware = server.NewAuthMiddleware(sessionRepo, userRepo, cfg)
		deps.TenantMiddleware = server.NewTenantMiddleware(orgRepo, sessionRepo)
	}

	// Initialize HTTP server
	srv := server.New(cfg, log, deps)

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
