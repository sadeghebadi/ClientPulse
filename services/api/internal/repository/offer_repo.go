package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrOfferNotFound         = errors.New("subscription offer not found")
	ErrOfferVersionNotFound  = errors.New("subscription offer version not found")
	ErrOfferAlreadyPublished = errors.New("offer is already published")
	ErrOfferArchived         = errors.New("offer is archived and cannot be published or modified")
	ErrOfferVersionImmutable = errors.New("published offer versions are immutable and cannot be modified")
	ErrNegativePrice         = errors.New("price cannot be negative")
	ErrPriceOverflow         = errors.New("price calculation resulted in integer overflow")
	ErrInvalidDenomination   = errors.New("invalid display denomination for currency")
	ErrInvalidRatio          = errors.New("denomination conversion ratio must be at least 1")
)

// SubscriptionOffer represents a localized presentation and commercial offer for a base plan.
type SubscriptionOffer struct {
	ID              string                    `json:"id"`
	PlanID          string                    `json:"plan_id"`
	PlanName        string                    `json:"plan_name,omitempty"`
	PlanSlug        string                    `json:"plan_slug,omitempty"`
	Locale          string                    `json:"locale"`
	Market          string                    `json:"market"`
	Name            string                    `json:"name"`
	Description     string                    `json:"description"`
	FeaturesSummary []string                  `json:"features_summary"`
	Status          string                    `json:"status"` // 'draft', 'active', 'archived'
	CreatedAt       time.Time                 `json:"created_at"`
	UpdatedAt       time.Time                 `json:"updated_at"`
	CreatedBy       *string                   `json:"created_by,omitempty"`
	UpdatedBy       *string                   `json:"updated_by,omitempty"`
	LatestVersion   *SubscriptionOfferVersion `json:"latest_version,omitempty"`
	VersionsCount   int                       `json:"versions_count"`
	// Base plan technical limits (from latest plan version)
	Limits *PlanLimits `json:"limits,omitempty"`
}

// SubscriptionOfferVersion represents an immutable published snapshot of localized pricing.
type SubscriptionOfferVersion struct {
	ID                  string     `json:"id"`
	OfferID             string     `json:"offer_id"`
	PlanVersionID       *string    `json:"plan_version_id,omitempty"`
	Version             int        `json:"version"`
	Status              string     `json:"status"` // 'draft', 'published', 'archived'
	PriceAmount         int64      `json:"price_amount"` // Canonical stored monetary amount in minor units
	Currency            string     `json:"currency"`     // ISO 4217, e.g. 'IRR', 'USD'
	DisplayDenomination string     `json:"display_denomination"` // 'toman', 'rial', 'usd', 'standard'
	DenominationRatio   int64      `json:"denomination_ratio"`   // e.g. 10 for Toman (1 Toman = 10 IRR stored units)
	BillingInterval     string     `json:"billing_interval"`     // 'monthly', 'yearly', 'quarterly', 'one_time'
	PublishedAt         *time.Time `json:"published_at,omitempty"`
	PublishedBy         *string    `json:"published_by,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	// Helper computed fields for UI display
	DisplayPrice   int64  `json:"display_price"`
	FormattedPrice string `json:"formatted_price"`
}

// CreateOfferParams defines inputs for creating a localized offer.
type CreateOfferParams struct {
	PlanID              string   `json:"plan_id"`
	Locale              string   `json:"locale"`
	Market              string   `json:"market"`
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	FeaturesSummary     []string `json:"features_summary"`
	DisplayPrice        int64    `json:"display_price"`
	Currency            string   `json:"currency"`
	DisplayDenomination string   `json:"display_denomination"`
	DenominationRatio   int64    `json:"denomination_ratio"`
	BillingInterval     string   `json:"billing_interval"`
}

// UpdateOfferParams defines inputs for updating a localized offer.
type UpdateOfferParams struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	FeaturesSummary     []string `json:"features_summary"`
	DisplayPrice        int64    `json:"display_price"`
	Currency            string   `json:"currency"`
	DisplayDenomination string   `json:"display_denomination"`
	DenominationRatio   int64    `json:"denomination_ratio"`
	BillingInterval     string   `json:"billing_interval"`
}

// OfferRepository handles commercial localized offers and versioning.
type OfferRepository struct {
	pool *pgxpool.Pool
}

// NewOfferRepository creates a new offer repository.
func NewOfferRepository(pool *pgxpool.Pool) *OfferRepository {
	return &OfferRepository{pool: pool}
}

// CalculateCanonicalPrice converts user display price into canonical stored price units using exact integer arithmetic.
func CalculateCanonicalPrice(displayPrice int64, currency, denomination string, customRatio int64) (canonicalPrice int64, ratio int64, err error) {
	if displayPrice < 0 {
		return 0, 0, ErrNegativePrice
	}

	currency = strings.ToUpper(strings.TrimSpace(currency))
	if !IsValidCurrency(currency) {
		return 0, 0, ErrInvalidCurrency
	}

	denom := strings.ToLower(strings.TrimSpace(denomination))
	if denom == "" {
		denom = "standard"
	}

	ratio = customRatio
	if ratio <= 0 {
		if denom == "toman" {
			ratio = 10 // 1 Toman = 10 Rials
		} else if denom == "usd" || currency == "USD" {
			ratio = 100 // 1 USD = 100 cents
		} else {
			ratio = 1
		}
	}

	if denom == "toman" && currency != "IRR" {
		return 0, 0, fmt.Errorf("%w: Toman is only applicable to IRR", ErrInvalidDenomination)
	}

	// Check multiplication overflow
	if displayPrice > 0 && math.MaxInt64/ratio < displayPrice {
		return 0, 0, ErrPriceOverflow
	}

	canonicalPrice = displayPrice * ratio
	return canonicalPrice, ratio, nil
}

// FormatDisplayPrice computes the human-facing display price and string representation deterministically.
func FormatDisplayPrice(canonicalPrice int64, currency, denomination string, ratio int64, locale string) (displayPrice int64, formatted string) {
	if ratio <= 0 {
		ratio = 1
	}
	displayPrice = canonicalPrice / ratio
	denom := strings.ToLower(strings.TrimSpace(denomination))

	if denom == "toman" {
		formatted = fmt.Sprintf("%d تومان", displayPrice)
	} else if currency == "USD" {
		formatted = fmt.Sprintf("$%d", displayPrice)
	} else {
		formatted = fmt.Sprintf("%d %s", displayPrice, currency)
	}

	return displayPrice, formatted
}

// ListOffersByPlan retrieves all localized offers and their latest versions for a base plan.
func (r *OfferRepository) ListOffersByPlan(ctx context.Context, planID string) ([]SubscriptionOffer, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT 
			o.id, o.plan_id, o.locale, o.market, o.name, o.description, o.features_summary, o.status,
			o.created_at, o.updated_at, o.created_by, o.updated_by,
			(SELECT COUNT(*) FROM subscription_offer_versions sov WHERE sov.offer_id = o.id) AS versions_count
		FROM subscription_offers o
		WHERE o.plan_id = $1
		ORDER BY o.locale ASC, o.market ASC
	`, planID)
	if err != nil {
		return nil, fmt.Errorf("query offers: %w", err)
	}
	defer rows.Close()

	var offers []SubscriptionOffer
	for rows.Next() {
		var o SubscriptionOffer
		var featsJSON []byte
		if err := rows.Scan(
			&o.ID, &o.PlanID, &o.Locale, &o.Market, &o.Name, &o.Description, &featsJSON,
			&o.Status, &o.CreatedAt, &o.UpdatedAt, &o.CreatedBy, &o.UpdatedBy, &o.VersionsCount,
		); err != nil {
			return nil, fmt.Errorf("scan offer: %w", err)
		}
		_ = json.Unmarshal(featsJSON, &o.FeaturesSummary)

		// Fetch latest offer version
		var v SubscriptionOfferVersion
		vErr := r.pool.QueryRow(ctx, `
			SELECT 
				id, offer_id, plan_version_id, version, status, price_amount, currency,
				display_denomination, denomination_ratio, billing_interval,
				created_at, updated_at, published_at, published_by
			FROM subscription_offer_versions
			WHERE offer_id = $1
			ORDER BY version DESC
			LIMIT 1
		`, o.ID).Scan(
			&v.ID, &v.OfferID, &v.PlanVersionID, &v.Version, &v.Status, &v.PriceAmount, &v.Currency,
			&v.DisplayDenomination, &v.DenominationRatio, &v.BillingInterval,
			&v.CreatedAt, &v.UpdatedAt, &v.PublishedAt, &v.PublishedBy,
		)
		if vErr == nil {
			v.DisplayPrice, v.FormattedPrice = FormatDisplayPrice(v.PriceAmount, v.Currency, v.DisplayDenomination, v.DenominationRatio, o.Locale)
			o.LatestVersion = &v
		}

		offers = append(offers, o)
	}

	return offers, nil
}

// GetOfferByID returns an offer and all its historical versions.
func (r *OfferRepository) GetOfferByID(ctx context.Context, offerID string) (*SubscriptionOffer, []SubscriptionOfferVersion, error) {
	var o SubscriptionOffer
	var featsJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT 
			o.id, o.plan_id, p.name AS plan_name, p.slug AS plan_slug,
			o.locale, o.market, o.name, o.description, o.features_summary, o.status,
			o.created_at, o.updated_at, o.created_by, o.updated_by,
			(SELECT COUNT(*) FROM subscription_offer_versions sov WHERE sov.offer_id = o.id) AS versions_count
		FROM subscription_offers o
		JOIN subscription_plans p ON p.id = o.plan_id
		WHERE o.id = $1
	`, offerID).Scan(
		&o.ID, &o.PlanID, &o.PlanName, &o.PlanSlug,
		&o.Locale, &o.Market, &o.Name, &o.Description, &featsJSON,
		&o.Status, &o.CreatedAt, &o.UpdatedAt, &o.CreatedBy, &o.UpdatedBy, &o.VersionsCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrOfferNotFound
		}
		return nil, nil, fmt.Errorf("query offer: %w", err)
	}
	_ = json.Unmarshal(featsJSON, &o.FeaturesSummary)

	vRows, err := r.pool.Query(ctx, `
		SELECT 
			id, offer_id, plan_version_id, version, status, price_amount, currency,
			display_denomination, denomination_ratio, billing_interval,
			created_at, updated_at, published_at, published_by
		FROM subscription_offer_versions
		WHERE offer_id = $1
		ORDER BY version DESC
	`, offerID)
	if err != nil {
		return nil, nil, fmt.Errorf("query offer versions: %w", err)
	}
	defer vRows.Close()

	var versions []SubscriptionOfferVersion
	for vRows.Next() {
		var v SubscriptionOfferVersion
		if err := vRows.Scan(
			&v.ID, &v.OfferID, &v.PlanVersionID, &v.Version, &v.Status, &v.PriceAmount, &v.Currency,
			&v.DisplayDenomination, &v.DenominationRatio, &v.BillingInterval,
			&v.CreatedAt, &v.UpdatedAt, &v.PublishedAt, &v.PublishedBy,
		); err != nil {
			return nil, nil, fmt.Errorf("scan offer version: %w", err)
		}
		v.DisplayPrice, v.FormattedPrice = FormatDisplayPrice(v.PriceAmount, v.Currency, v.DisplayDenomination, v.DenominationRatio, o.Locale)
		versions = append(versions, v)
	}

	if len(versions) > 0 {
		o.LatestVersion = &versions[0]
	}

	return &o, versions, nil
}

// CreateDraftOffer creates a new localized offer and its initial draft version.
func (r *OfferRepository) CreateDraftOffer(ctx context.Context, actorID string, p CreateOfferParams) (*SubscriptionOffer, *SubscriptionOfferVersion, error) {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return nil, nil, fmt.Errorf("offer name is required")
	}
	p.Locale = strings.TrimSpace(p.Locale)
	if p.Locale != "en" && p.Locale != "fa" {
		return nil, nil, fmt.Errorf("locale must be 'en' or 'fa'")
	}
	p.Market = strings.TrimSpace(p.Market)
	if p.Market == "" {
		p.Market = "global"
	}
	p.Currency = strings.ToUpper(strings.TrimSpace(p.Currency))
	if p.Currency == "" {
		if p.Locale == "fa" {
			p.Currency = "IRR"
		} else {
			p.Currency = "USD"
		}
	}
	if p.DisplayDenomination == "" {
		if p.Locale == "fa" {
			p.DisplayDenomination = "toman"
		} else {
			p.DisplayDenomination = "standard"
		}
	}
	p.BillingInterval = strings.ToLower(strings.TrimSpace(p.BillingInterval))
	if p.BillingInterval == "" {
		p.BillingInterval = "monthly"
	}

	canonicalPrice, ratio, err := CalculateCanonicalPrice(p.DisplayPrice, p.Currency, p.DisplayDenomination, p.DenominationRatio)
	if err != nil {
		return nil, nil, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	// Find latest published or draft plan_version_id for base plan
	var planVersionID *string
	_ = tx.QueryRow(ctx, `
		SELECT id FROM subscription_plan_versions
		WHERE plan_id = $1
		ORDER BY version DESC
		LIMIT 1
	`, p.PlanID).Scan(&planVersionID)

	featsJSON, _ := json.Marshal(p.FeaturesSummary)

	var offer SubscriptionOffer
	err = tx.QueryRow(ctx, `
		INSERT INTO subscription_offers (plan_id, locale, market, name, description, features_summary, status, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, 'draft', $7, $7)
		RETURNING id, plan_id, locale, market, name, description, features_summary, status, created_at, updated_at, created_by, updated_by
	`, p.PlanID, p.Locale, p.Market, p.Name, p.Description, string(featsJSON), actorID).Scan(
		&offer.ID, &offer.PlanID, &offer.Locale, &offer.Market, &offer.Name, &offer.Description,
		&featsJSON, &offer.Status, &offer.CreatedAt, &offer.UpdatedAt, &offer.CreatedBy, &offer.UpdatedBy,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("insert offer: %w", err)
	}
	_ = json.Unmarshal(featsJSON, &offer.FeaturesSummary)

	var version SubscriptionOfferVersion
	err = tx.QueryRow(ctx, `
		INSERT INTO subscription_offer_versions (
			offer_id, plan_version_id, version, status, price_amount, currency,
			display_denomination, denomination_ratio, billing_interval
		) VALUES (
			$1, $2, 1, 'draft', $3, $4, $5, $6, $7
		)
		RETURNING 
			id, offer_id, plan_version_id, version, status, price_amount, currency,
			display_denomination, denomination_ratio, billing_interval,
			created_at, updated_at, published_at, published_by
	`,
		offer.ID, planVersionID, canonicalPrice, p.Currency,
		p.DisplayDenomination, ratio, p.BillingInterval,
	).Scan(
		&version.ID, &version.OfferID, &version.PlanVersionID, &version.Version, &version.Status,
		&version.PriceAmount, &version.Currency, &version.DisplayDenomination, &version.DenominationRatio,
		&version.BillingInterval, &version.CreatedAt, &version.UpdatedAt, &version.PublishedAt, &version.PublishedBy,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("insert initial offer version: %w", err)
	}

	version.DisplayPrice, version.FormattedPrice = FormatDisplayPrice(version.PriceAmount, version.Currency, version.DisplayDenomination, version.DenominationRatio, offer.Locale)
	offer.LatestVersion = &version
	offer.VersionsCount = 1

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit offer creation: %w", err)
	}

	return &offer, &version, nil
}

// UpdateDraftOffer updates a draft offer or creates a new draft version if already published.
func (r *OfferRepository) UpdateDraftOffer(ctx context.Context, actorID, offerID string, p UpdateOfferParams) (*SubscriptionOffer, *SubscriptionOfferVersion, error) {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return nil, nil, fmt.Errorf("offer name cannot be empty")
	}
	p.Currency = strings.ToUpper(strings.TrimSpace(p.Currency))
	p.BillingInterval = strings.ToLower(strings.TrimSpace(p.BillingInterval))
	if p.BillingInterval == "" {
		p.BillingInterval = "monthly"
	}

	canonicalPrice, ratio, err := CalculateCanonicalPrice(p.DisplayPrice, p.Currency, p.DisplayDenomination, p.DenominationRatio)
	if err != nil {
		return nil, nil, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	featsJSON, _ := json.Marshal(p.FeaturesSummary)

	var offer SubscriptionOffer
	err = tx.QueryRow(ctx, `
		UPDATE subscription_offers
		SET name = $1, description = $2, features_summary = $3::jsonb, updated_by = $4, updated_at = NOW()
		WHERE id = $5 AND status != 'archived'
		RETURNING id, plan_id, locale, market, name, description, features_summary, status, created_at, updated_at, created_by, updated_by
	`, p.Name, p.Description, string(featsJSON), actorID, offerID).Scan(
		&offer.ID, &offer.PlanID, &offer.Locale, &offer.Market, &offer.Name, &offer.Description,
		&featsJSON, &offer.Status, &offer.CreatedAt, &offer.UpdatedAt, &offer.CreatedBy, &offer.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrOfferNotFound
		}
		return nil, nil, fmt.Errorf("update offer: %w", err)
	}
	_ = json.Unmarshal(featsJSON, &offer.FeaturesSummary)

	// Fetch latest offer version
	var latestVer SubscriptionOfferVersion
	err = tx.QueryRow(ctx, `
		SELECT id, version, status
		FROM subscription_offer_versions
		WHERE offer_id = $1
		ORDER BY version DESC
		LIMIT 1
	`, offerID).Scan(&latestVer.ID, &latestVer.Version, &latestVer.Status)
	if err != nil {
		return nil, nil, fmt.Errorf("query latest offer version: %w", err)
	}

	// Fetch latest plan version for base plan
	var planVerID *string
	_ = tx.QueryRow(ctx, `
		SELECT id FROM subscription_plan_versions
		WHERE plan_id = $1
		ORDER BY version DESC
		LIMIT 1
	`, offer.PlanID).Scan(&planVerID)

	var finalVersion SubscriptionOfferVersion
	if latestVer.Status == "draft" {
		err = tx.QueryRow(ctx, `
			UPDATE subscription_offer_versions
			SET price_amount = $1, currency = $2, display_denomination = $3,
			    denomination_ratio = $4, billing_interval = $5, plan_version_id = $6, updated_at = NOW()
			WHERE id = $7
			RETURNING 
				id, offer_id, plan_version_id, version, status, price_amount, currency,
				display_denomination, denomination_ratio, billing_interval,
				created_at, updated_at, published_at, published_by
		`, canonicalPrice, p.Currency, p.DisplayDenomination, ratio, p.BillingInterval, planVerID, latestVer.ID).Scan(
			&finalVersion.ID, &finalVersion.OfferID, &finalVersion.PlanVersionID, &finalVersion.Version,
			&finalVersion.Status, &finalVersion.PriceAmount, &finalVersion.Currency, &finalVersion.DisplayDenomination,
			&finalVersion.DenominationRatio, &finalVersion.BillingInterval,
			&finalVersion.CreatedAt, &finalVersion.UpdatedAt, &finalVersion.PublishedAt, &finalVersion.PublishedBy,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("update draft offer version: %w", err)
		}
	} else {
		// Create new draft version (v + 1)
		newVersionNum := latestVer.Version + 1
		err = tx.QueryRow(ctx, `
			INSERT INTO subscription_offer_versions (
				offer_id, plan_version_id, version, status, price_amount, currency,
				display_denomination, denomination_ratio, billing_interval
			) VALUES (
				$1, $2, $3, 'draft', $4, $5, $6, $7, $8
			)
			RETURNING 
				id, offer_id, plan_version_id, version, status, price_amount, currency,
				display_denomination, denomination_ratio, billing_interval,
				created_at, updated_at, published_at, published_by
		`, offer.ID, planVerID, newVersionNum, canonicalPrice, p.Currency, p.DisplayDenomination, ratio, p.BillingInterval).Scan(
			&finalVersion.ID, &finalVersion.OfferID, &finalVersion.PlanVersionID, &finalVersion.Version,
			&finalVersion.Status, &finalVersion.PriceAmount, &finalVersion.Currency, &finalVersion.DisplayDenomination,
			&finalVersion.DenominationRatio, &finalVersion.BillingInterval,
			&finalVersion.CreatedAt, &finalVersion.UpdatedAt, &finalVersion.PublishedAt, &finalVersion.PublishedBy,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("create incremented draft offer version: %w", err)
		}
	}

	finalVersion.DisplayPrice, finalVersion.FormattedPrice = FormatDisplayPrice(finalVersion.PriceAmount, finalVersion.Currency, finalVersion.DisplayDenomination, finalVersion.DenominationRatio, offer.Locale)
	offer.LatestVersion = &finalVersion

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit offer update: %w", err)
	}

	return &offer, &finalVersion, nil
}

// PublishOffer publishes the latest draft offer version and marks the offer as active.
func (r *OfferRepository) PublishOffer(ctx context.Context, actorID, offerID string) (*SubscriptionOffer, *SubscriptionOfferVersion, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	var draftVerID string
	err = tx.QueryRow(ctx, `
		SELECT id
		FROM subscription_offer_versions
		WHERE offer_id = $1 AND status = 'draft'
		ORDER BY version DESC
		LIMIT 1
	`, offerID).Scan(&draftVerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, fmt.Errorf("no draft version found to publish for offer %s", offerID)
		}
		return nil, nil, fmt.Errorf("query draft offer version: %w", err)
	}

	var pubVersion SubscriptionOfferVersion
	err = tx.QueryRow(ctx, `
		UPDATE subscription_offer_versions
		SET status = 'published', published_at = NOW(), published_by = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING 
			id, offer_id, plan_version_id, version, status, price_amount, currency,
			display_denomination, denomination_ratio, billing_interval,
			created_at, updated_at, published_at, published_by
	`, actorID, draftVerID).Scan(
		&pubVersion.ID, &pubVersion.OfferID, &pubVersion.PlanVersionID, &pubVersion.Version,
		&pubVersion.Status, &pubVersion.PriceAmount, &pubVersion.Currency, &pubVersion.DisplayDenomination,
		&pubVersion.DenominationRatio, &pubVersion.BillingInterval,
		&pubVersion.CreatedAt, &pubVersion.UpdatedAt, &pubVersion.PublishedAt, &pubVersion.PublishedBy,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("publish offer version: %w", err)
	}

	var offer SubscriptionOffer
	var featsJSON []byte
	err = tx.QueryRow(ctx, `
		UPDATE subscription_offers
		SET status = 'active', updated_by = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, plan_id, locale, market, name, description, features_summary, status, created_at, updated_at, created_by, updated_by
	`, actorID, offerID).Scan(
		&offer.ID, &offer.PlanID, &offer.Locale, &offer.Market, &offer.Name, &offer.Description,
		&featsJSON, &offer.Status, &offer.CreatedAt, &offer.UpdatedAt, &offer.CreatedBy, &offer.UpdatedBy,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("update offer to active: %w", err)
	}
	_ = json.Unmarshal(featsJSON, &offer.FeaturesSummary)

	pubVersion.DisplayPrice, pubVersion.FormattedPrice = FormatDisplayPrice(pubVersion.PriceAmount, pubVersion.Currency, pubVersion.DisplayDenomination, pubVersion.DenominationRatio, offer.Locale)
	offer.LatestVersion = &pubVersion

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit offer publish: %w", err)
	}

	return &offer, &pubVersion, nil
}

// ArchiveOffer archives an offer.
func (r *OfferRepository) ArchiveOffer(ctx context.Context, actorID, offerID string) error {
	res, err := r.pool.Exec(ctx, `
		UPDATE subscription_offers
		SET status = 'archived', updated_by = $1, updated_at = NOW()
		WHERE id = $2
	`, actorID, offerID)
	if err != nil {
		return fmt.Errorf("archive offer: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrOfferNotFound
	}
	return nil
}

// ListCatalogOffers retrieves customer-facing published offers for the specified locale and market.
func (r *OfferRepository) ListCatalogOffers(ctx context.Context, locale, market string) ([]SubscriptionOffer, error) {
	if market == "" {
		market = "global"
	}

	rows, err := r.pool.Query(ctx, `
		SELECT 
			o.id, o.plan_id, p.name AS plan_name, p.slug AS plan_slug,
			o.locale, o.market, o.name, o.description, o.features_summary, o.status,
			o.created_at, o.updated_at,
			sov.id, sov.plan_version_id, sov.version, sov.status, sov.price_amount, sov.currency,
			sov.display_denomination, sov.denomination_ratio, sov.billing_interval,
			sov.published_at,
			spv.max_clients, spv.max_assets, spv.max_monitors, spv.max_heartbeats, spv.max_members,
			spv.min_check_interval_seconds, spv.retention_days, spv.custom_branding_enabled, spv.public_reports_enabled
		FROM subscription_offers o
		JOIN subscription_plans p ON p.id = o.plan_id AND p.status = 'active'
		JOIN LATERAL (
			SELECT * FROM subscription_offer_versions v
			WHERE v.offer_id = o.id AND v.status = 'published'
			ORDER BY v.version DESC
			LIMIT 1
		) sov ON true
		LEFT JOIN subscription_plan_versions spv ON spv.id = sov.plan_version_id
		WHERE o.status = 'active' AND o.locale = $1
		ORDER BY p.created_at ASC
	`, locale)
	if err != nil {
		return nil, fmt.Errorf("query catalog offers: %w", err)
	}
	defer rows.Close()

	var offers []SubscriptionOffer
	for rows.Next() {
		var o SubscriptionOffer
		var v SubscriptionOfferVersion
		var limits PlanLimits
		var featsJSON []byte
		var planVerID *string

		if err := rows.Scan(
			&o.ID, &o.PlanID, &o.PlanName, &o.PlanSlug,
			&o.Locale, &o.Market, &o.Name, &o.Description, &featsJSON, &o.Status,
			&o.CreatedAt, &o.UpdatedAt,
			&v.ID, &planVerID, &v.Version, &v.Status, &v.PriceAmount, &v.Currency,
			&v.DisplayDenomination, &v.DenominationRatio, &v.BillingInterval,
			&v.PublishedAt,
			&limits.MaxClients, &limits.MaxAssets, &limits.MaxMonitors, &limits.MaxHeartbeats, &limits.MaxMembers,
			&limits.MinCheckIntervalSeconds, &limits.RetentionDays, &limits.CustomBrandingEnabled, &limits.PublicReportsEnabled,
		); err != nil {
			return nil, fmt.Errorf("scan catalog offer: %w", err)
		}

		_ = json.Unmarshal(featsJSON, &o.FeaturesSummary)
		v.OfferID = o.ID
		v.PlanVersionID = planVerID
		v.DisplayPrice, v.FormattedPrice = FormatDisplayPrice(v.PriceAmount, v.Currency, v.DisplayDenomination, v.DenominationRatio, o.Locale)
		o.LatestVersion = &v
		o.Limits = &limits

		offers = append(offers, o)
	}

	return offers, nil
}

// ListAllAdminCatalogs retrieves all offers across all catalogs with base plan status.
func (r *OfferRepository) ListAllAdminCatalogs(ctx context.Context) ([]SubscriptionOffer, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT 
			o.id, o.plan_id, p.name AS plan_name, p.slug AS plan_slug,
			o.locale, o.market, o.name, o.description, o.features_summary, o.status,
			o.created_at, o.updated_at, o.created_by, o.updated_by,
			(SELECT COUNT(*) FROM subscription_offer_versions sov WHERE sov.offer_id = o.id) AS versions_count
		FROM subscription_offers o
		JOIN subscription_plans p ON p.id = o.plan_id
		ORDER BY o.locale ASC, p.created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query all admin catalogs: %w", err)
	}
	defer rows.Close()

	var offers []SubscriptionOffer
	for rows.Next() {
		var o SubscriptionOffer
		var featsJSON []byte
		if err := rows.Scan(
			&o.ID, &o.PlanID, &o.PlanName, &o.PlanSlug,
			&o.Locale, &o.Market, &o.Name, &o.Description, &featsJSON,
			&o.Status, &o.CreatedAt, &o.UpdatedAt, &o.CreatedBy, &o.UpdatedBy, &o.VersionsCount,
		); err != nil {
			return nil, fmt.Errorf("scan offer: %w", err)
		}
		_ = json.Unmarshal(featsJSON, &o.FeaturesSummary)

		// Fetch latest version
		var v SubscriptionOfferVersion
		vErr := r.pool.QueryRow(ctx, `
			SELECT 
				id, offer_id, plan_version_id, version, status, price_amount, currency,
				display_denomination, denomination_ratio, billing_interval,
				created_at, updated_at, published_at, published_by
			FROM subscription_offer_versions
			WHERE offer_id = $1
			ORDER BY version DESC
			LIMIT 1
		`, o.ID).Scan(
			&v.ID, &v.OfferID, &v.PlanVersionID, &v.Version, &v.Status, &v.PriceAmount, &v.Currency,
			&v.DisplayDenomination, &v.DenominationRatio, &v.BillingInterval,
			&v.CreatedAt, &v.UpdatedAt, &v.PublishedAt, &v.PublishedBy,
		)
		if vErr == nil {
			v.DisplayPrice, v.FormattedPrice = FormatDisplayPrice(v.PriceAmount, v.Currency, v.DisplayDenomination, v.DenominationRatio, o.Locale)
			o.LatestVersion = &v
		}

		offers = append(offers, o)
	}

	return offers, nil
}

