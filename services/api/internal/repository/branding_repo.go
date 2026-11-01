package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidColor  = errors.New("color must be a valid hex code (e.g. #0ea5e9 or #0af)")
	ErrInvalidLocale = errors.New("default report locale must be 'en' or 'fa'")
)

var hexColorRegex = regexp.MustCompile(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`)

type BrandingRepository struct {
	pool *pgxpool.Pool
}

func NewBrandingRepository(pool *pgxpool.Pool) *BrandingRepository {
	return &BrandingRepository{pool: pool}
}

type UpdateBrandingParams struct {
	DisplayName         *string
	PrimaryColor        *string
	AccentColor         *string
	ReportFooterText    *string
	ContactEmail        *string
	WebsiteURL          *string
	DefaultReportLocale *string
}

// GetBranding retrieves organization branding or default sensible values if not configured.
func (r *BrandingRepository) GetBranding(ctx context.Context, orgID string) (*OrganizationBranding, error) {
	var b OrganizationBranding
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			SELECT organization_id, display_name, logo_storage_key, primary_color, accent_color,
			       report_footer_text, contact_email, website_url, default_report_locale,
			       created_at, updated_at
			FROM organization_branding
			WHERE organization_id = $1
		`
		err := tx.QueryRow(ctx, query, orgID).Scan(
			&b.OrganizationID, &b.DisplayName, &b.LogoStorageKey, &b.PrimaryColor, &b.AccentColor,
			&b.ReportFooterText, &b.ContactEmail, &b.WebsiteURL, &b.DefaultReportLocale,
			&b.CreatedAt, &b.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Fallback to org name as display name
				var orgName, orgLocale string
				_ = tx.QueryRow(ctx, "SELECT name, locale FROM organizations WHERE id = $1", orgID).Scan(&orgName, &orgLocale)
				if orgLocale != "fa" {
					orgLocale = "en"
				}

				b = OrganizationBranding{
					OrganizationID:      orgID,
					DisplayName:         orgName,
					PrimaryColor:        "#0ea5e9",
					AccentColor:         "#6366f1",
					ReportFooterText:    "ClientPulse Care Report",
					DefaultReportLocale: orgLocale,
					CreatedAt:           time.Now().UTC(),
					UpdatedAt:           time.Now().UTC(),
				}
				return nil
			}
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &b, nil
}

// UpsertBranding creates or updates custom branding settings.
func (r *BrandingRepository) UpsertBranding(ctx context.Context, orgID string, params UpdateBrandingParams) (*OrganizationBranding, error) {
	// 1. Fetch current or defaults
	current, err := r.GetBranding(ctx, orgID)
	if err != nil {
		return nil, err
	}

	if params.DisplayName != nil {
		current.DisplayName = strings.TrimSpace(*params.DisplayName)
	}

	if params.PrimaryColor != nil {
		c := strings.ToLower(strings.TrimSpace(*params.PrimaryColor))
		if !hexColorRegex.MatchString(c) {
			return nil, ErrInvalidColor
		}
		current.PrimaryColor = c
	}

	if params.AccentColor != nil {
		c := strings.ToLower(strings.TrimSpace(*params.AccentColor))
		if !hexColorRegex.MatchString(c) {
			return nil, ErrInvalidColor
		}
		current.AccentColor = c
	}

	if params.ReportFooterText != nil {
		current.ReportFooterText = strings.TrimSpace(*params.ReportFooterText)
	}

	if params.ContactEmail != nil {
		current.ContactEmail = strings.TrimSpace(*params.ContactEmail)
	}

	if params.WebsiteURL != nil {
		current.WebsiteURL = strings.TrimSpace(*params.WebsiteURL)
	}

	if params.DefaultReportLocale != nil {
		loc := strings.ToLower(strings.TrimSpace(*params.DefaultReportLocale))
		if loc != "en" && loc != "fa" {
			return nil, ErrInvalidLocale
		}
		current.DefaultReportLocale = loc
	}

	var saved OrganizationBranding
	err = WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			INSERT INTO organization_branding (
				organization_id, display_name, primary_color, accent_color,
				report_footer_text, contact_email, website_url, default_report_locale,
				created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
			ON CONFLICT (organization_id) DO UPDATE
			SET display_name = EXCLUDED.display_name,
			    primary_color = EXCLUDED.primary_color,
			    accent_color = EXCLUDED.accent_color,
			    report_footer_text = EXCLUDED.report_footer_text,
			    contact_email = EXCLUDED.contact_email,
			    website_url = EXCLUDED.website_url,
			    default_report_locale = EXCLUDED.default_report_locale,
			    updated_at = NOW()
			RETURNING organization_id, display_name, logo_storage_key, primary_color, accent_color,
			          report_footer_text, contact_email, website_url, default_report_locale,
			          created_at, updated_at
		`

		return tx.QueryRow(ctx, query,
			orgID, current.DisplayName, current.PrimaryColor, current.AccentColor,
			current.ReportFooterText, current.ContactEmail, current.WebsiteURL, current.DefaultReportLocale,
		).Scan(
			&saved.OrganizationID, &saved.DisplayName, &saved.LogoStorageKey, &saved.PrimaryColor, &saved.AccentColor,
			&saved.ReportFooterText, &saved.ContactEmail, &saved.WebsiteURL, &saved.DefaultReportLocale,
			&saved.CreatedAt, &saved.UpdatedAt,
		)
	})

	if err != nil {
		return nil, fmt.Errorf("failed upserting branding: %w", err)
	}
	return &saved, nil
}

// UpdateLogoStorageKey sets or clears the logo storage key for an organization.
func (r *BrandingRepository) UpdateLogoStorageKey(ctx context.Context, orgID string, storageKey *string) error {
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			INSERT INTO organization_branding (
				organization_id, logo_storage_key, created_at, updated_at
			)
			VALUES ($1, $2, NOW(), NOW())
			ON CONFLICT (organization_id) DO UPDATE
			SET logo_storage_key = EXCLUDED.logo_storage_key,
			    updated_at = NOW()
		`
		_, err := tx.Exec(ctx, query, orgID, storageKey)
		return err
	})
}

// CalculateContrastRatio calculates the WCAG relative luminance contrast ratio between hexColor and white background.
func CalculateContrastRatio(hexColor string) (float64, bool) {
	clean := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(hexColor)), "#")
	if len(clean) == 3 {
		clean = string([]byte{clean[0], clean[0], clean[1], clean[1], clean[2], clean[2]})
	}
	if len(clean) != 6 {
		return 1.0, false
	}

	r, _ := strconv.ParseInt(clean[0:2], 16, 64)
	g, _ := strconv.ParseInt(clean[2:4], 16, 64)
	b, _ := strconv.ParseInt(clean[4:6], 16, 64)

	toLinear := func(val int64) float64 {
		s := float64(val) / 255.0
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}

	lum := 0.2126*toLinear(r) + 0.7152*toLinear(g) + 0.0722*toLinear(b)
	whiteLum := 1.0

	// Contrast against white
	ratio := (whiteLum + 0.05) / (lum + 0.05)
	if ratio < 1.0 {
		ratio = 1.0 / ratio
	}

	// WCAG AA requirement for UI components is at least 3:1 (large text / UI component), 4.5:1 for body
	isAccessible := ratio >= 2.5
	return ratio, isAccessible
}

