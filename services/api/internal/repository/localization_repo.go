package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrDefaultLocaleNotEnabled  = errors.New("default locale must be included in enabled locales")
	ErrAtLeastOneLocaleRequired = errors.New("at least one locale must remain enabled")
)

// LocaleMetadata describes a supported system language and its display properties.
type LocaleMetadata struct {
	Code        string `json:"code"`
	Label       string `json:"label"`
	NativeLabel string `json:"native_label"`
	Dir         string `json:"dir"`
	Enabled     bool   `json:"enabled"`
	IsDefault   bool   `json:"is_default"`
}

// LocalizationConfig represents the complete centralized platform language settings.
type LocalizationConfig struct {
	EnabledLocales   []string         `json:"enabled_locales"`
	DefaultLocale    string           `json:"default_locale"`
	SupportedLocales []LocaleMetadata `json:"supported_locales"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// PublicLocalizationConfig returns only public-safe localization settings.
type PublicLocalizationConfig struct {
	EnabledLocales   []string         `json:"enabled_locales"`
	DefaultLocale    string           `json:"default_locale"`
	SupportedLocales []LocaleMetadata `json:"supported_locales"`
}

// AllSupportedLocales defines the baseline platform languages.
var AllSupportedLocales = []LocaleMetadata{
	{
		Code:        "en",
		Label:       "English",
		NativeLabel: "English",
		Dir:         "ltr",
	},
	{
		Code:        "fa",
		Label:       "Persian",
		NativeLabel: "فارسی",
		Dir:         "rtl",
	},
}

// LocalizationRepository manages centralized language enablement and persistence.
type LocalizationRepository struct {
	pool       *pgxpool.Pool
	mu         sync.RWMutex
	cachedConf *PublicLocalizationConfig
}

// NewLocalizationRepository initializes a new localization repository.
func NewLocalizationRepository(pool *pgxpool.Pool) *LocalizationRepository {
	return &LocalizationRepository{pool: pool}
}

// IsValidSupportedLocale checks if a locale code is known to the system.
func IsValidSupportedLocale(code string) bool {
	for _, l := range AllSupportedLocales {
		if l.Code == code {
			return true
		}
	}
	return false
}

// buildConfigFromRow constructs the structured config from DB row values.
func buildConfigFromRow(enabled []string, def string, updatedAt time.Time) *LocalizationConfig {
	enabledMap := make(map[string]bool)
	for _, l := range enabled {
		enabledMap[l] = true
	}

	supported := make([]LocaleMetadata, len(AllSupportedLocales))
	for i, l := range AllSupportedLocales {
		meta := l
		meta.Enabled = enabledMap[l.Code]
		meta.IsDefault = (l.Code == def)
		supported[i] = meta
	}

	return &LocalizationConfig{
		EnabledLocales:   enabled,
		DefaultLocale:    def,
		SupportedLocales: supported,
		UpdatedAt:        updatedAt,
	}
}

// GetLocalizationConfig retrieves the current language settings from PostgreSQL.
func (r *LocalizationRepository) GetLocalizationConfig(ctx context.Context) (*LocalizationConfig, error) {
	var enabledLocales []string
	var defaultLocale string
	var updatedAt time.Time

	err := r.pool.QueryRow(ctx, `
		SELECT enabled_locales, default_locale, updated_at
		FROM platform_localization_settings
		WHERE id = 1
	`).Scan(&enabledLocales, &defaultLocale, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Fallback defaults if table is empty
			return buildConfigFromRow([]string{"en", "fa"}, "en", time.Now().UTC()), nil
		}
		return nil, fmt.Errorf("query localization settings: %w", err)
	}

	return buildConfigFromRow(enabledLocales, defaultLocale, updatedAt), nil
}

// GetPublicLocalizationConfig retrieves public-safe configuration, utilizing a cached copy.
func (r *LocalizationRepository) GetPublicLocalizationConfig(ctx context.Context) (*PublicLocalizationConfig, error) {
	r.mu.RLock()
	cached := r.cachedConf
	r.mu.RUnlock()

	if cached != nil {
		return cached, nil
	}

	cfg, err := r.GetLocalizationConfig(ctx)
	if err != nil {
		return nil, err
	}

	pub := &PublicLocalizationConfig{
		EnabledLocales:   cfg.EnabledLocales,
		DefaultLocale:    cfg.DefaultLocale,
		SupportedLocales: cfg.SupportedLocales,
	}

	r.mu.Lock()
	r.cachedConf = pub
	r.mu.Unlock()

	return pub, nil
}

// UpdateLocalizationConfig atomically validates and persists new language settings.
func (r *LocalizationRepository) UpdateLocalizationConfig(ctx context.Context, actorID string, enabledLocales []string, defaultLocale string) (*LocalizationConfig, error) {
	if len(enabledLocales) == 0 {
		return nil, ErrAtLeastOneLocaleRequired
	}

	// Validate supported locales
	hasDefault := false
	uniqueEnabled := make([]string, 0, len(enabledLocales))
	seen := make(map[string]bool)

	for _, loc := range enabledLocales {
		if !IsValidSupportedLocale(loc) {
			return nil, fmt.Errorf("%w: '%s'", ErrInvalidLocale, loc)
		}
		if !seen[loc] {
			seen[loc] = true
			uniqueEnabled = append(uniqueEnabled, loc)
		}
		if loc == defaultLocale {
			hasDefault = true
		}
	}

	if !hasDefault {
		return nil, ErrDefaultLocaleNotEnabled
	}

	// Atomic update with row lock
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	var currentID int
	err = tx.QueryRow(ctx, `
		SELECT id FROM platform_localization_settings
		WHERE id = 1
		FOR UPDATE
	`).Scan(&currentID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lock localization settings: %w", err)
	}

	var updatedAt time.Time
	err = tx.QueryRow(ctx, `
		INSERT INTO platform_localization_settings (id, enabled_locales, default_locale, updated_at, updated_by)
		VALUES (1, $1, $2, NOW(), $3)
		ON CONFLICT (id) DO UPDATE
		SET enabled_locales = EXCLUDED.enabled_locales,
		    default_locale = EXCLUDED.default_locale,
		    updated_at = NOW(),
		    updated_by = EXCLUDED.updated_by
		RETURNING updated_at
	`, uniqueEnabled, defaultLocale, actorID).Scan(&updatedAt)
	if err != nil {
		return nil, fmt.Errorf("persist localization settings: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit localization settings: %w", err)
	}

	// Invalidate cache
	r.mu.Lock()
	r.cachedConf = nil
	r.mu.Unlock()

	return buildConfigFromRow(uniqueEnabled, defaultLocale, updatedAt), nil
}
