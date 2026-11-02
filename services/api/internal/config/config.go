package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"clientpulse/services/api/internal/appurl"
)

// Config holds all configuration parameters for ClientPulse API and Worker.
type Config struct {
	AppEnv          string
	AppBaseURL      string
	PublicAppURL    string
	APIBaseURL      string
	Port            int
	DefaultLocale   string
	DefaultTimezone string

	// Database
	DatabaseURL       string
	WorkerDatabaseURL string

	// Security
	SessionSecret   string
	TokenHashPepper string

	// Integrations
	TelegramBotToken           string
	TelegramBotUsername        string
	TelegramWebhookSecret      string
	TelegramMode               string
	TelegramPollTimeoutSeconds int
	TelegramProxyURL           string

	// Email
	EmailProvider string
	SMTPHost      string
	SMTPPort      int
	SMTPUsername  string
	SMTPPassword  string
	SMTPFromEmail string
	EmailFrom     string // Deprecated: alias for SMTPFromEmail
	SMTPFromName  string
	SMTPTLSMode   string

	// Storage
	StorageProvider  string
	StorageLocalPath string
	S3Endpoint       string
	S3Bucket         string
	S3Region         string
	S3AccessKey      string
	S3SecretKey      string

	// Billing
	PaymentProvider      string
	PaymentWebhookSecret string

	// Worker & Monitoring
	WorkerConcurrency             int
	WorkerHeartbeatTimeoutSeconds int
	MonitorDefaultIntervalSeconds int
	MonitorDefaultTimeoutSeconds  int
	MonitorLeaseTimeoutSeconds    int
	CheckResultRetentionDays      int

	// Admin Seed
	PlatformAdminEmail    string
	PlatformAdminPassword string
}

// Load loads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:                       getEnv("APP_ENV", "development"),
		AppBaseURL:                   getEnv("APP_BASE_URL", "http://localhost:3000"),
		PublicAppURL:                 getEnv("PUBLIC_APP_URL", ""),
		APIBaseURL:                   getEnv("API_BASE_URL", "http://localhost:8080"),
		Port:                         getEnvAsInt("API_PORT", 8080),
		DefaultLocale:                getEnv("DEFAULT_LOCALE", "en"),
		DefaultTimezone:              getEnv("DEFAULT_TIMEZONE", "UTC"),
		DatabaseURL:                  getEnv("DATABASE_URL", "postgres://clientpulse:clientpulse_dev_password@localhost:5433/clientpulse?sslmode=disable"),
		WorkerDatabaseURL:            getEnv("WORKER_DATABASE_URL", getEnv("DATABASE_URL", "postgres://clientpulse_worker:clientpulse_worker_dev_password@localhost:5433/clientpulse?sslmode=disable")),
		SessionSecret:                getEnv("SESSION_SECRET", "dev-session-secret-replace-with-cryptographic-random-value-in-prod"),
		TokenHashPepper:              getEnv("TOKEN_HASH_PEPPER", "dev-token-pepper-replace-with-cryptographic-random-value-in-prod"),
		TelegramBotToken:             getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramBotUsername:          getEnv("TELEGRAM_BOT_USERNAME", ""),
		TelegramWebhookSecret:        getEnv("TELEGRAM_WEBHOOK_SECRET", ""),
		TelegramMode:                 getEnv("TELEGRAM_MODE", "disabled"),
		TelegramPollTimeoutSeconds:   getEnvAsInt("TELEGRAM_POLL_TIMEOUT_SECONDS", 50),
		TelegramProxyURL:             getEnv("TELEGRAM_PROXY_URL", ""),
		EmailProvider:                getEnv("EMAIL_PROVIDER", "none"),
		SMTPHost:                     getEnv("SMTP_HOST", "mailpit"),
		SMTPPort:                     getEnvAsInt("SMTP_PORT", 1025),
		SMTPUsername:                 getEnv("SMTP_USERNAME", ""),
		SMTPPassword:                 getEnv("SMTP_PASSWORD", ""),
		SMTPFromEmail:                getEnv("SMTP_FROM_EMAIL", getEnv("EMAIL_FROM", "notifications@clientpulse.local")),
		EmailFrom:                    getEnv("SMTP_FROM_EMAIL", getEnv("EMAIL_FROM", "notifications@clientpulse.local")),
		SMTPFromName:                 getEnv("SMTP_FROM_NAME", "ClientPulse Alerts"),
		SMTPTLSMode:                  getEnv("SMTP_TLS_MODE", "none"),
		StorageProvider:              getEnv("STORAGE_PROVIDER", "local"),
		StorageLocalPath:             getEnv("STORAGE_LOCAL_PATH", "/data/storage"),
		S3Endpoint:                   getEnv("S3_ENDPOINT", ""),
		S3Bucket:                     getEnv("S3_BUCKET", ""),
		S3Region:                     getEnv("S3_REGION", ""),
		S3AccessKey:                  getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:                  getEnv("S3_SECRET_KEY", ""),
		PaymentProvider:               getEnv("PAYMENT_PROVIDER", "manual"),
		PaymentWebhookSecret:          getEnv("PAYMENT_WEBHOOK_SECRET", ""),
		WorkerConcurrency:             getEnvAsInt("WORKER_CONCURRENCY", 10),
		WorkerHeartbeatTimeoutSeconds:  getEnvAsInt("WORKER_HEARTBEAT_TIMEOUT_SECONDS", 10),
		MonitorDefaultIntervalSeconds: getEnvAsInt("MONITOR_DEFAULT_INTERVAL_SECONDS", 300),
		MonitorDefaultTimeoutSeconds:  getEnvAsInt("MONITOR_DEFAULT_TIMEOUT_SECONDS", 10),
		MonitorLeaseTimeoutSeconds:    getEnvAsInt("MONITOR_LEASE_TIMEOUT_SECONDS", 60),
		CheckResultRetentionDays:     getEnvAsInt("CHECK_RESULT_RETENTION_DAYS", 90),
		PlatformAdminEmail:           getEnv("PLATFORM_ADMIN_EMAIL", "admin@clientpulse.local"),
		PlatformAdminPassword:        getEnv("PLATFORM_ADMIN_PASSWORD", "dev-admin-password-change-me"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation error: %w", err)
	}

	return cfg, nil
}

// Validate verifies mandatory configuration rules.
func (c *Config) Validate() error {
	if c.AppEnv == "" {
		return fmt.Errorf("APP_ENV cannot be empty")
	}

	if _, err := url.ParseRequestURI(c.AppBaseURL); err != nil {
		return fmt.Errorf("invalid APP_BASE_URL: %w", err)
	}

	if err := appurl.ValidatePublicAppURL(c.PublicAppURL, c.IsProduction()); err != nil {
		return fmt.Errorf("invalid PUBLIC_APP_URL: %w", err)
	}

	if _, err := url.ParseRequestURI(c.APIBaseURL); err != nil {
		return fmt.Errorf("invalid API_BASE_URL: %w", err)
	}

	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("API_PORT must be between 1 and 65535")
	}

	if c.WorkerConcurrency <= 0 {
		return fmt.Errorf("WORKER_CONCURRENCY must be greater than 0")
	}

	if c.WorkerHeartbeatTimeoutSeconds <= 0 {
		return fmt.Errorf("WORKER_HEARTBEAT_TIMEOUT_SECONDS must be greater than 0")
	}

	if c.MonitorLeaseTimeoutSeconds < 5 || c.MonitorLeaseTimeoutSeconds > 3600 {
		c.MonitorLeaseTimeoutSeconds = 60
	}

	if c.TelegramPollTimeoutSeconds <= 0 {
		c.TelegramPollTimeoutSeconds = 50
	}
	switch strings.ToLower(c.TelegramMode) {
	case "polling", "webhook", "disabled", "mock", "":
		// valid modes
	default:
		return fmt.Errorf("invalid TELEGRAM_MODE: %q (must be 'polling', 'webhook', or 'disabled')", c.TelegramMode)
	}

	switch strings.ToLower(c.EmailProvider) {
	case "smtp", "none", "disabled", "mock", "":
		// valid modes
	default:
		return fmt.Errorf("invalid EMAIL_PROVIDER: %q (must be 'disabled', 'none', or 'smtp')", c.EmailProvider)
	}

	if c.AppEnv == "production" {
		if strings.Contains(c.SessionSecret, "dev-session-secret") || len(c.SessionSecret) < 32 {
			return fmt.Errorf("production SESSION_SECRET must be at least 32 characters and cannot use default dev secret")
		}
		if strings.Contains(c.TokenHashPepper, "dev-token-pepper") || len(c.TokenHashPepper) < 32 {
			return fmt.Errorf("production TOKEN_HASH_PEPPER must be at least 32 characters and cannot use default dev pepper")
		}
		if c.EmailProvider == "smtp" {
			if strings.ToLower(c.SMTPTLSMode) == "none" {
				return fmt.Errorf("production SMTP_TLS_MODE cannot be 'none'; must be 'starttls' or 'tls'")
			}
			if c.SMTPHost == "" || c.SMTPHost == "mailpit" || c.SMTPHost == "localhost" || c.SMTPHost == "127.0.0.1" {
				return fmt.Errorf("production SMTP_HOST must be a valid external SMTP server (not %s)", c.SMTPHost)
			}
			if c.SMTPPort <= 0 {
				return fmt.Errorf("production SMTP_PORT must be a valid port number")
			}
			if c.SMTPUsername == "" {
				return fmt.Errorf("production SMTP_USERNAME is required")
			}
			if c.SMTPPassword == "" {
				return fmt.Errorf("production SMTP_PASSWORD is required")
			}
			if c.SMTPFromEmail == "" || !strings.Contains(c.SMTPFromEmail, "@") {
				return fmt.Errorf("production SMTP_FROM_EMAIL must be a valid email address")
			}
		}
	}

	return nil
}

// IsProduction returns true if current environment is production.
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.AppEnv) == "production"
}

// Helper functions for reading environment variables with defaults.
func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := getEnv(key, "")
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	valStr := getEnv(key, "")
	if valStr == "" {
		return defaultVal
	}
	dur, err := time.ParseDuration(valStr)
	if err != nil {
		return defaultVal
	}
	return dur
}

