package config

import (
	"os"
	"testing"
)

func TestConfigLoadDefaults(t *testing.T) {
	// Clear any overrides for test
	os.Unsetenv("APP_ENV")
	os.Unsetenv("APP_BASE_URL")
	os.Unsetenv("API_PORT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config to load successfully, got error: %v", err)
	}

	if cfg.AppEnv != "development" {
		t.Errorf("expected AppEnv 'development', got '%s'", cfg.AppEnv)
	}

	if cfg.Port != 8080 {
		t.Errorf("expected default Port 8080, got %d", cfg.Port)
	}

	if cfg.IsProduction() {
		t.Errorf("expected IsProduction to be false in development")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{
			name: "valid config",
			mutate: func(c *Config) {
				// No change
			},
			wantErr: false,
		},
		{
			name: "invalid base url",
			mutate: func(c *Config) {
				c.AppBaseURL = "invalid-url"
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			mutate: func(c *Config) {
				c.Port = 70000
			},
			wantErr: true,
		},
		{
			name: "missing database url",
			mutate: func(c *Config) {
				c.DatabaseURL = ""
			},
			wantErr: true,
		},
		{
			name: "invalid worker heartbeat timeout",
			mutate: func(c *Config) {
				c.WorkerHeartbeatTimeoutSeconds = 0
			},
			wantErr: true,
		},
		{
			name: "production with insecure secret",
			mutate: func(c *Config) {
				c.AppEnv = "production"
				c.SessionSecret = "dev-session-secret-short"
			},
			wantErr: true,
		},
		{
			name: "production with valid secure secret",
			mutate: func(c *Config) {
				c.AppEnv = "production"
				c.SessionSecret = "a-very-long-production-session-secret-that-is-at-least-32-chars"
				c.TokenHashPepper = "a-very-long-production-token-pepper-that-is-at-least-32-chars"
			},
			wantErr: false,
		},
		{
			name: "valid production PublicAppURL",
			mutate: func(c *Config) {
				c.AppEnv = "production"
				c.SessionSecret = "a-very-long-production-session-secret-that-is-at-least-32-chars"
				c.TokenHashPepper = "a-very-long-production-token-pepper-that-is-at-least-32-chars"
				c.PublicAppURL = "https://mindu.ir"
			},
			wantErr: false,
		},
		{
			name: "invalid production PublicAppURL (http rejected)",
			mutate: func(c *Config) {
				c.AppEnv = "production"
				c.SessionSecret = "a-very-long-production-session-secret-that-is-at-least-32-chars"
				c.TokenHashPepper = "a-very-long-production-token-pepper-that-is-at-least-32-chars"
				c.PublicAppURL = "http://mindu.ir"
			},
			wantErr: true,
		},
		{
			name: "invalid PublicAppURL with credentials",
			mutate: func(c *Config) {
				c.PublicAppURL = "https://user:pass@mindu.ir"
			},
			wantErr: true,
		},
		{
			name: "production with EMAIL_PROVIDER=disabled without smtp credentials",
			mutate: func(c *Config) {
				c.AppEnv = "production"
				c.SessionSecret = "a-very-long-production-session-secret-that-is-at-least-32-chars"
				c.TokenHashPepper = "a-very-long-production-token-pepper-that-is-at-least-32-chars"
				c.PublicAppURL = "https://mindu.ir"
				c.EmailProvider = "disabled"
				c.TelegramMode = "disabled"
				c.SMTPHost = ""
			},
			wantErr: false,
		},
		{
			name: "invalid EMAIL_PROVIDER rejected",
			mutate: func(c *Config) {
				c.EmailProvider = "unsupported_provider"
			},
			wantErr: true,
		},
		{
			name: "valid TelegramProxyURL with http",
			mutate: func(c *Config) {
				c.TelegramProxyURL = "http://proxy.example.com:8080"
			},
			wantErr: false,
		},
		{
			name: "valid TelegramProxyURL with https",
			mutate: func(c *Config) {
				c.TelegramProxyURL = "https://proxy.example.com:8443"
			},
			wantErr: false,
		},
		{
			name: "valid TelegramProxyURL with socks5",
			mutate: func(c *Config) {
				c.TelegramProxyURL = "socks5://127.0.0.1:1080"
			},
			wantErr: false,
		},
		{
			name: "valid TelegramProxyURL with socks5 and auth",
			mutate: func(c *Config) {
				c.TelegramProxyURL = "socks5://proxyuser:proxypass@proxy.example.com:1080"
			},
			wantErr: false,
		},
		{
			name: "invalid TelegramProxyURL with socks5h rejected",
			mutate: func(c *Config) {
				c.TelegramProxyURL = "socks5h://127.0.0.1:1080"
			},
			wantErr: true,
		},
		{
			name: "invalid TelegramProxyURL with unsupported scheme ftp rejected",
			mutate: func(c *Config) {
				c.TelegramProxyURL = "ftp://proxy.example.com:21"
			},
			wantErr: true,
		},
		{
			name: "invalid TelegramProxyURL missing host rejected",
			mutate: func(c *Config) {
				c.TelegramProxyURL = "http://"
			},
			wantErr: true,
		},
		{
			name: "invalid TelegramProxyURL missing scheme rejected",
			mutate: func(c *Config) {
				c.TelegramProxyURL = "proxy.example.com:8080"
			},
			wantErr: true,
		},
		{
			name: "invalid TelegramProxyURL malformed rejected",
			mutate: func(c *Config) {
				c.TelegramProxyURL = "://bad-url"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load()
			if err != nil {
				t.Fatalf("failed to load base config: %v", err)
			}
			tt.mutate(cfg)
			err = cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeProxyURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"http://proxy.example.com:8080", "http://proxy.example.com:8080"},
		{"socks5://user:secret123@proxy.example.com:1080", "socks5://REDACTED@proxy.example.com:1080"},
		{"https://admin:pass@127.0.0.1:8443", "https://REDACTED@127.0.0.1:8443"},
	}

	for _, tt := range tests {
		got := SanitizeProxyURL(tt.input)
		if got != tt.expected {
			t.Errorf("SanitizeProxyURL(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

