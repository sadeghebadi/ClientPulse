package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPlanNotFound         = errors.New("subscription plan not found")
	ErrPlanVersionNotFound  = errors.New("subscription plan version not found")
	ErrPlanSlugAlreadyExists = errors.New("a plan with this slug already exists")
	ErrPlanAlreadyPublished = errors.New("plan is already published; cannot modify an active version directly")
	ErrPlanArchived         = errors.New("plan is archived and cannot be published or assigned")
	ErrInvalidCurrency      = errors.New("invalid currency code: must be a 3-letter ISO 4217 uppercase code")
	ErrInvalidIntervalSec   = errors.New("minimum check interval must be at least 60 seconds")
	ErrInvalidPlanStatus    = errors.New("invalid plan status transition")
)

var currencyRegex = regexp.MustCompile(`^[A-Z]{3}$`)

// CreatePlanVersionParams holds limit and pricing inputs for creating or updating plan versions.
type CreatePlanVersionParams struct {
	PriceAmount             *int64                 `json:"price_amount"` // nil if unconfigured
	Currency                *string                `json:"currency"`     // nil if unconfigured
	BillingInterval         *string                `json:"billing_interval"`
	MaxClients              int                    `json:"max_clients"`
	MaxAssets               int                    `json:"max_assets"`
	MaxMonitors             int                    `json:"max_monitors"`
	MaxHeartbeats           int                    `json:"max_heartbeats"`
	MaxMembers              int                    `json:"max_members"`
	MinCheckIntervalSeconds int                    `json:"min_check_interval_seconds"`
	RetentionDays           int                    `json:"retention_days"`
	CustomBrandingEnabled   bool                   `json:"custom_branding_enabled"`
	PublicReportsEnabled    bool                   `json:"public_reports_enabled"`
	Features                map[string]interface{} `json:"features"`
}

// PlanRepository manages commercial plan definitions and versioning.
type PlanRepository struct {
	pool *pgxpool.Pool
}

// NewPlanRepository creates a new plan repository.
func NewPlanRepository(pool *pgxpool.Pool) *PlanRepository {
	return &PlanRepository{pool: pool}
}

// ValidateParams validates pricing, currency, intervals, and structured limits.
// Does NOT force defaults on PriceAmount, Currency, or BillingInterval.
func ValidateParams(p *CreatePlanVersionParams) error {
	if p.Currency != nil {
		c := strings.ToUpper(strings.TrimSpace(*p.Currency))
		if c == "" {
			p.Currency = nil
		} else {
			if !currencyRegex.MatchString(c) {
				return ErrInvalidCurrency
			}
			p.Currency = &c
		}
	}

	if p.BillingInterval != nil {
		bi := strings.ToLower(strings.TrimSpace(*p.BillingInterval))
		if bi == "" {
			p.BillingInterval = nil
		} else {
			if bi != "monthly" && bi != "yearly" && bi != "quarterly" && bi != "one_time" {
				return fmt.Errorf("invalid billing_interval: must be monthly, yearly, quarterly, or one_time")
			}
			p.BillingInterval = &bi
		}
	}

	if p.MinCheckIntervalSeconds < 60 {
		p.MinCheckIntervalSeconds = 60
	}
	if p.RetentionDays <= 0 {
		p.RetentionDays = 30
	}
	if p.PriceAmount != nil && *p.PriceAmount < 0 {
		return fmt.Errorf("price_amount cannot be negative")
	}
	if p.Features == nil {
		p.Features = make(map[string]interface{})
	}
	return nil
}

// ListPlans lists all subscription plans along with their latest version and active subscription counts.
func (r *PlanRepository) ListPlans(ctx context.Context) ([]SubscriptionPlan, error) {
	query := `
		SELECT 
			p.id, p.name, p.slug, p.description, p.status, p.created_at, p.updated_at, p.created_by, p.updated_by,
			(SELECT COUNT(*) FROM organization_subscriptions os JOIN subscription_plan_versions spv ON spv.id = os.plan_version_id WHERE spv.plan_id = p.id AND os.status = 'active') AS active_subs,
			(SELECT COUNT(*) FROM subscription_plan_versions spv WHERE spv.plan_id = p.id) AS versions_count
		FROM subscription_plans p
		ORDER BY p.created_at ASC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query plans: %w", err)
	}
	defer rows.Close()

	var plans []SubscriptionPlan
	for rows.Next() {
		var p SubscriptionPlan
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Slug, &p.Description, &p.Status, &p.CreatedAt, &p.UpdatedAt,
			&p.CreatedBy, &p.UpdatedBy, &p.ActiveSubscriptionsCount, &p.VersionsCount,
		); err != nil {
			return nil, fmt.Errorf("scan plan: %w", err)
		}

		// Retrieve latest version
		var v SubscriptionPlanVersion
		var featuresJSON []byte
		vErr := r.pool.QueryRow(ctx, `
			SELECT 
				id, plan_id, version, status, price_amount, currency, billing_interval,
				max_clients, max_assets, max_monitors, max_heartbeats, max_members,
				min_check_interval_seconds, retention_days, custom_branding_enabled, public_reports_enabled,
				features, created_at, updated_at, published_at, published_by
			FROM subscription_plan_versions
			WHERE plan_id = $1
			ORDER BY version DESC
			LIMIT 1
		`, p.ID).Scan(
			&v.ID, &v.PlanID, &v.Version, &v.Status, &v.PriceAmount, &v.Currency, &v.BillingInterval,
			&v.MaxClients, &v.MaxAssets, &v.MaxMonitors, &v.MaxHeartbeats, &v.MaxMembers,
			&v.MinCheckIntervalSeconds, &v.RetentionDays, &v.CustomBrandingEnabled, &v.PublicReportsEnabled,
			&featuresJSON, &v.CreatedAt, &v.UpdatedAt, &v.PublishedAt, &v.PublishedBy,
		)
		if vErr == nil {
			_ = json.Unmarshal(featuresJSON, &v.Features)
			p.LatestVersion = &v
		}

		plans = append(plans, p)
	}

	return plans, nil
}

// GetPlanByID retrieves a plan and all its historical and current versions.
func (r *PlanRepository) GetPlanByID(ctx context.Context, planID string) (*SubscriptionPlan, []SubscriptionPlanVersion, error) {
	var p SubscriptionPlan
	err := r.pool.QueryRow(ctx, `
		SELECT 
			p.id, p.name, p.slug, p.description, p.status, p.created_at, p.updated_at, p.created_by, p.updated_by,
			(SELECT COUNT(*) FROM organization_subscriptions os JOIN subscription_plan_versions spv ON spv.id = os.plan_version_id WHERE spv.plan_id = p.id AND os.status = 'active') AS active_subs,
			(SELECT COUNT(*) FROM subscription_plan_versions spv WHERE spv.plan_id = p.id) AS versions_count
		FROM subscription_plans p
		WHERE p.id = $1
	`, planID).Scan(
		&p.ID, &p.Name, &p.Slug, &p.Description, &p.Status, &p.CreatedAt, &p.UpdatedAt,
		&p.CreatedBy, &p.UpdatedBy, &p.ActiveSubscriptionsCount, &p.VersionsCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrPlanNotFound
		}
		return nil, nil, fmt.Errorf("query plan: %w", err)
	}

	vRows, err := r.pool.Query(ctx, `
		SELECT 
			id, plan_id, version, status, price_amount, currency, billing_interval,
			max_clients, max_assets, max_monitors, max_heartbeats, max_members,
			min_check_interval_seconds, retention_days, custom_branding_enabled, public_reports_enabled,
			features, created_at, updated_at, published_at, published_by
		FROM subscription_plan_versions
		WHERE plan_id = $1
		ORDER BY version DESC
	`, planID)
	if err != nil {
		return nil, nil, fmt.Errorf("query plan versions: %w", err)
	}
	defer vRows.Close()

	var versions []SubscriptionPlanVersion
	for vRows.Next() {
		var v SubscriptionPlanVersion
		var featuresJSON []byte
		if err := vRows.Scan(
			&v.ID, &v.PlanID, &v.Version, &v.Status, &v.PriceAmount, &v.Currency, &v.BillingInterval,
			&v.MaxClients, &v.MaxAssets, &v.MaxMonitors, &v.MaxHeartbeats, &v.MaxMembers,
			&v.MinCheckIntervalSeconds, &v.RetentionDays, &v.CustomBrandingEnabled, &v.PublicReportsEnabled,
			&featuresJSON, &v.CreatedAt, &v.UpdatedAt, &v.PublishedAt, &v.PublishedBy,
		); err != nil {
			return nil, nil, fmt.Errorf("scan version: %w", err)
		}
		_ = json.Unmarshal(featuresJSON, &v.Features)
		versions = append(versions, v)
	}

	if len(versions) > 0 {
		p.LatestVersion = &versions[0]
	}

	return &p, versions, nil
}

// CreateDraftPlan creates a new draft plan and its initial draft version (v1).
func (r *PlanRepository) CreateDraftPlan(ctx context.Context, name, slug, description string, createdBy string, vParams CreatePlanVersionParams) (*SubscriptionPlan, *SubscriptionPlanVersion, error) {
	name = strings.TrimSpace(name)
	slug = strings.ToLower(strings.TrimSpace(slug))
	description = strings.TrimSpace(description)

	if name == "" {
		return nil, nil, fmt.Errorf("plan name is required")
	}
	if slug == "" {
		return nil, nil, fmt.Errorf("plan slug is required")
	}
	if err := ValidateParams(&vParams); err != nil {
		return nil, nil, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	var plan SubscriptionPlan
	err = tx.QueryRow(ctx, `
		INSERT INTO subscription_plans (name, slug, description, status, created_by, updated_by)
		VALUES ($1, $2, $3, 'draft', $4, $4)
		RETURNING id, name, slug, description, status, created_at, updated_at, created_by, updated_by
	`, name, slug, description, createdBy).Scan(
		&plan.ID, &plan.Name, &plan.Slug, &plan.Description, &plan.Status,
		&plan.CreatedAt, &plan.UpdatedAt, &plan.CreatedBy, &plan.UpdatedBy,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, nil, ErrPlanSlugAlreadyExists
		}
		return nil, nil, fmt.Errorf("insert plan: %w", err)
	}

	featuresJSON, _ := json.Marshal(vParams.Features)

	var version SubscriptionPlanVersion
	err = tx.QueryRow(ctx, `
		INSERT INTO subscription_plan_versions (
			plan_id, version, status, price_amount, currency, billing_interval,
			max_clients, max_assets, max_monitors, max_heartbeats, max_members,
			min_check_interval_seconds, retention_days, custom_branding_enabled, public_reports_enabled,
			features
		) VALUES (
			$1, 1, 'draft', $2, $3, $4,
			$5, $6, $7, $8, $9,
			$10, $11, $12, $13,
			$14::jsonb
		)
		RETURNING 
			id, plan_id, version, status, price_amount, currency, billing_interval,
			max_clients, max_assets, max_monitors, max_heartbeats, max_members,
			min_check_interval_seconds, retention_days, custom_branding_enabled, public_reports_enabled,
			features, created_at, updated_at, published_at, published_by
	`,
		plan.ID, vParams.PriceAmount, vParams.Currency, vParams.BillingInterval,
		vParams.MaxClients, vParams.MaxAssets, vParams.MaxMonitors, vParams.MaxHeartbeats, vParams.MaxMembers,
		vParams.MinCheckIntervalSeconds, vParams.RetentionDays, vParams.CustomBrandingEnabled, vParams.PublicReportsEnabled,
		string(featuresJSON),
	).Scan(
		&version.ID, &version.PlanID, &version.Version, &version.Status, &version.PriceAmount, &version.Currency, &version.BillingInterval,
		&version.MaxClients, &version.MaxAssets, &version.MaxMonitors, &version.MaxHeartbeats, &version.MaxMembers,
		&version.MinCheckIntervalSeconds, &version.RetentionDays, &version.CustomBrandingEnabled, &version.PublicReportsEnabled,
		&featuresJSON, &version.CreatedAt, &version.UpdatedAt, &version.PublishedAt, &version.PublishedBy,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("insert initial plan version: %w", err)
	}

	_ = json.Unmarshal(featuresJSON, &version.Features)
	plan.LatestVersion = &version
	plan.VersionsCount = 1

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit plan creation: %w", err)
	}

	return &plan, &version, nil
}

// UpdateDraftPlan updates a draft plan's general details and its draft version limits.
// If the latest version is already published, a new draft version is automatically created!
func (r *PlanRepository) UpdateDraftPlan(ctx context.Context, planID string, name, description string, updatedBy string, vParams CreatePlanVersionParams) (*SubscriptionPlan, *SubscriptionPlanVersion, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" {
		return nil, nil, fmt.Errorf("plan name cannot be empty")
	}
	if err := ValidateParams(&vParams); err != nil {
		return nil, nil, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	var plan SubscriptionPlan
	err = tx.QueryRow(ctx, `
		UPDATE subscription_plans
		SET name = $1, description = $2, updated_by = $3, updated_at = NOW()
		WHERE id = $4 AND status != 'archived'
		RETURNING id, name, slug, description, status, created_at, updated_at, created_by, updated_by
	`, name, description, updatedBy, planID).Scan(
		&plan.ID, &plan.Name, &plan.Slug, &plan.Description, &plan.Status,
		&plan.CreatedAt, &plan.UpdatedAt, &plan.CreatedBy, &plan.UpdatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrPlanNotFound
		}
		return nil, nil, fmt.Errorf("update plan: %w", err)
	}

	// Fetch latest version
	var latestVer SubscriptionPlanVersion
	err = tx.QueryRow(ctx, `
		SELECT id, version, status
		FROM subscription_plan_versions
		WHERE plan_id = $1
		ORDER BY version DESC
		LIMIT 1
	`, planID).Scan(&latestVer.ID, &latestVer.Version, &latestVer.Status)
	if err != nil {
		return nil, nil, fmt.Errorf("query latest version: %w", err)
	}

	featuresJSON, _ := json.Marshal(vParams.Features)
	var finalVersion SubscriptionPlanVersion

	if latestVer.Status == "draft" {
		// Update existing draft version in place
		err = tx.QueryRow(ctx, `
			UPDATE subscription_plan_versions
			SET 
				price_amount = $1, currency = $2, billing_interval = $3,
				max_clients = $4, max_assets = $5, max_monitors = $6, max_heartbeats = $7, max_members = $8,
				min_check_interval_seconds = $9, retention_days = $10,
				custom_branding_enabled = $11, public_reports_enabled = $12,
				features = $13::jsonb, updated_at = NOW()
			WHERE id = $14
			RETURNING 
				id, plan_id, version, status, price_amount, currency, billing_interval,
				max_clients, max_assets, max_monitors, max_heartbeats, max_members,
				min_check_interval_seconds, retention_days, custom_branding_enabled, public_reports_enabled,
				features, created_at, updated_at, published_at, published_by
		`,
			vParams.PriceAmount, vParams.Currency, vParams.BillingInterval,
			vParams.MaxClients, vParams.MaxAssets, vParams.MaxMonitors, vParams.MaxHeartbeats, vParams.MaxMembers,
			vParams.MinCheckIntervalSeconds, vParams.RetentionDays,
			vParams.CustomBrandingEnabled, vParams.PublicReportsEnabled,
			string(featuresJSON), latestVer.ID,
		).Scan(
			&finalVersion.ID, &finalVersion.PlanID, &finalVersion.Version, &finalVersion.Status,
			&finalVersion.PriceAmount, &finalVersion.Currency, &finalVersion.BillingInterval,
			&finalVersion.MaxClients, &finalVersion.MaxAssets, &finalVersion.MaxMonitors, &finalVersion.MaxHeartbeats, &finalVersion.MaxMembers,
			&finalVersion.MinCheckIntervalSeconds, &finalVersion.RetentionDays,
			&finalVersion.CustomBrandingEnabled, &finalVersion.PublicReportsEnabled,
			&featuresJSON, &finalVersion.CreatedAt, &finalVersion.UpdatedAt, &finalVersion.PublishedAt, &finalVersion.PublishedBy,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("update draft version: %w", err)
		}
	} else {
		// Already published! Automatically create new draft version (v + 1)
		newVersionNum := latestVer.Version + 1
		err = tx.QueryRow(ctx, `
			INSERT INTO subscription_plan_versions (
				plan_id, version, status, price_amount, currency, billing_interval,
				max_clients, max_assets, max_monitors, max_heartbeats, max_members,
				min_check_interval_seconds, retention_days, custom_branding_enabled, public_reports_enabled,
				features
			) VALUES (
				$1, $2, 'draft', $3, $4, $5,
				$6, $7, $8, $9, $10,
				$11, $12, $13, $14,
				$15::jsonb
			)
			RETURNING 
				id, plan_id, version, status, price_amount, currency, billing_interval,
				max_clients, max_assets, max_monitors, max_heartbeats, max_members,
				min_check_interval_seconds, retention_days, custom_branding_enabled, public_reports_enabled,
				features, created_at, updated_at, published_at, published_by
		`,
			planID, newVersionNum, vParams.PriceAmount, vParams.Currency, vParams.BillingInterval,
			vParams.MaxClients, vParams.MaxAssets, vParams.MaxMonitors, vParams.MaxHeartbeats, vParams.MaxMembers,
			vParams.MinCheckIntervalSeconds, vParams.RetentionDays,
			vParams.CustomBrandingEnabled, vParams.PublicReportsEnabled,
			string(featuresJSON),
		).Scan(
			&finalVersion.ID, &finalVersion.PlanID, &finalVersion.Version, &finalVersion.Status,
			&finalVersion.PriceAmount, &finalVersion.Currency, &finalVersion.BillingInterval,
			&finalVersion.MaxClients, &finalVersion.MaxAssets, &finalVersion.MaxMonitors, &finalVersion.MaxHeartbeats, &finalVersion.MaxMembers,
			&finalVersion.MinCheckIntervalSeconds, &finalVersion.RetentionDays,
			&finalVersion.CustomBrandingEnabled, &finalVersion.PublicReportsEnabled,
			&featuresJSON, &finalVersion.CreatedAt, &finalVersion.UpdatedAt, &finalVersion.PublishedAt, &finalVersion.PublishedBy,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("create incremented draft version: %w", err)
		}
	}

	_ = json.Unmarshal(featuresJSON, &finalVersion.Features)
	plan.LatestVersion = &finalVersion

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit plan update: %w", err)
	}

	return &plan, &finalVersion, nil
}

// PublishPlan publishes the latest draft version, creating an immutable published version, and sets plan status to active.
func (r *PlanRepository) PublishPlan(ctx context.Context, planID, publishedBy string) (*SubscriptionPlan, *SubscriptionPlanVersion, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	// Find the latest draft version and validate completeness
	var draftVerID string
	var draftVerNum int
	var priceAmount *int64
	var currency, billingInterval *string
	err = tx.QueryRow(ctx, `
		SELECT id, version, price_amount, currency, billing_interval
		FROM subscription_plan_versions
		WHERE plan_id = $1 AND status = 'draft'
		ORDER BY version DESC
		LIMIT 1
	`, planID).Scan(&draftVerID, &draftVerNum, &priceAmount, &currency, &billingInterval)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, fmt.Errorf("no draft version found to publish for plan %s", planID)
		}
		return nil, nil, fmt.Errorf("query draft version: %w", err)
	}

	if priceAmount == nil {
		return nil, nil, fmt.Errorf("cannot publish plan version: price_amount is required")
	}
	if currency == nil || *currency == "" {
		return nil, nil, fmt.Errorf("cannot publish plan version: currency is required")
	}
	if billingInterval == nil || *billingInterval == "" {
		return nil, nil, fmt.Errorf("cannot publish plan version: billing_interval is required")
	}

	// Update draft version to published
	var publishedVer SubscriptionPlanVersion
	var featuresJSON []byte
	err = tx.QueryRow(ctx, `
		UPDATE subscription_plan_versions
		SET status = 'published', published_at = NOW(), published_by = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING 
			id, plan_id, version, status, price_amount, currency, billing_interval,
			max_clients, max_assets, max_monitors, max_heartbeats, max_members,
			min_check_interval_seconds, retention_days, custom_branding_enabled, public_reports_enabled,
			features, created_at, updated_at, published_at, published_by
	`, publishedBy, draftVerID).Scan(
		&publishedVer.ID, &publishedVer.PlanID, &publishedVer.Version, &publishedVer.Status,
		&publishedVer.PriceAmount, &publishedVer.Currency, &publishedVer.BillingInterval,
		&publishedVer.MaxClients, &publishedVer.MaxAssets, &publishedVer.MaxMonitors, &publishedVer.MaxHeartbeats, &publishedVer.MaxMembers,
		&publishedVer.MinCheckIntervalSeconds, &publishedVer.RetentionDays,
		&publishedVer.CustomBrandingEnabled, &publishedVer.PublicReportsEnabled,
		&featuresJSON, &publishedVer.CreatedAt, &publishedVer.UpdatedAt, &publishedVer.PublishedAt, &publishedVer.PublishedBy,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("publish version: %w", err)
	}
	_ = json.Unmarshal(featuresJSON, &publishedVer.Features)

	// Update parent plan status to active
	var plan SubscriptionPlan
	err = tx.QueryRow(ctx, `
		UPDATE subscription_plans
		SET status = 'active', updated_by = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, name, slug, description, status, created_at, updated_at, created_by, updated_by
	`, publishedBy, planID).Scan(
		&plan.ID, &plan.Name, &plan.Slug, &plan.Description, &plan.Status,
		&plan.CreatedAt, &plan.UpdatedAt, &plan.CreatedBy, &plan.UpdatedBy,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("update plan status to active: %w", err)
	}

	plan.LatestVersion = &publishedVer

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit plan publication: %w", err)
	}

	return &plan, &publishedVer, nil
}

// ArchivePlan archives a plan so it cannot be assigned to new organizations, preserving all existing assignments.
func (r *PlanRepository) ArchivePlan(ctx context.Context, planID, updatedBy string) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE subscription_plans
		SET status = 'archived', updated_by = $1, updated_at = NOW()
		WHERE id = $2 AND status != 'archived'
	`, updatedBy, planID)
	if err != nil {
		return fmt.Errorf("archive plan: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrPlanNotFound
	}
	return nil
}

// DeleteDraftPlan deletes a draft plan and its unreferenced draft version.
// If the plan has any published versions or any subscriptions, deletion is rejected.
func (r *PlanRepository) DeleteDraftPlan(ctx context.Context, planID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	// Check if plan exists and is in draft status
	var status string
	err = tx.QueryRow(ctx, "SELECT status FROM subscription_plans WHERE id = $1", planID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPlanNotFound
		}
		return fmt.Errorf("query plan: %w", err)
	}
	if status != "draft" {
		return fmt.Errorf("only draft plans can be deleted; current status is %s", status)
	}

	// Check if any versions are published
	var publishedCount int
	err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM subscription_plan_versions WHERE plan_id = $1 AND (status = 'published' OR published_at IS NOT NULL)", planID).Scan(&publishedCount)
	if err != nil {
		return fmt.Errorf("check published versions: %w", err)
	}
	if publishedCount > 0 {
		return fmt.Errorf("cannot delete plan with published versions")
	}

	// Check if any subscriptions reference versions of this plan
	var subCount int
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM organization_subscriptions os
		JOIN subscription_plan_versions spv ON spv.id = os.plan_version_id
		WHERE spv.plan_id = $1
	`, planID).Scan(&subCount)
	if err != nil {
		return fmt.Errorf("check subscriptions: %w", err)
	}
	if subCount > 0 {
		return fmt.Errorf("cannot delete plan referenced by organization subscriptions")
	}

	// Delete draft versions explicitly to trigger any checks
	_, err = tx.Exec(ctx, "DELETE FROM subscription_plan_versions WHERE plan_id = $1", planID)
	if err != nil {
		return fmt.Errorf("delete plan versions: %w", err)
	}

	// Delete plan
	_, err = tx.Exec(ctx, "DELETE FROM subscription_plans WHERE id = $1", planID)
	if err != nil {
		return fmt.Errorf("delete plan: %w", err)
	}

	return tx.Commit(ctx)
}

// GetPlanAssignments lists all organization subscriptions assigned to any version of this plan.
func (r *PlanRepository) GetPlanAssignments(ctx context.Context, planID string) ([]OrganizationSubscription, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT 
			os.id, os.organization_id, o.name, os.plan_version_id, p.id, p.name, p.slug, spv.version,
			os.status, os.started_at, os.ends_at, os.assigned_by, os.administrative_note,
			os.created_at, os.updated_at, spv.price_amount, spv.currency, spv.billing_interval,
			spv.max_clients, spv.max_assets, spv.max_monitors, spv.max_heartbeats, spv.max_members,
			spv.min_check_interval_seconds, spv.retention_days, spv.custom_branding_enabled, spv.public_reports_enabled
		FROM organization_subscriptions os
		JOIN organizations o ON o.id = os.organization_id
		JOIN subscription_plan_versions spv ON spv.id = os.plan_version_id
		JOIN subscription_plans p ON p.id = spv.plan_id
		WHERE p.id = $1
		ORDER BY os.started_at DESC
	`, planID)
	if err != nil {
		return nil, fmt.Errorf("query plan assignments: %w", err)
	}
	defer rows.Close()

	var subs []OrganizationSubscription
	for rows.Next() {
		var sub OrganizationSubscription
		var limits PlanLimits
		if err := rows.Scan(
			&sub.ID, &sub.OrganizationID, &sub.OrganizationName, &sub.PlanVersionID, &sub.PlanID, &sub.PlanName, &sub.PlanSlug, &sub.PlanVersion,
			&sub.Status, &sub.StartedAt, &sub.EndsAt, &sub.AssignedBy, &sub.AdministrativeNote,
			&sub.CreatedAt, &sub.UpdatedAt, &sub.PriceAmount, &sub.Currency, &sub.BillingInterval,
			&limits.MaxClients, &limits.MaxAssets, &limits.MaxMonitors, &limits.MaxHeartbeats, &limits.MaxMembers,
			&limits.MinCheckIntervalSeconds, &limits.RetentionDays, &limits.CustomBrandingEnabled, &limits.PublicReportsEnabled,
		); err != nil {
			return nil, fmt.Errorf("scan assignment: %w", err)
		}
		sub.Limits = limits
		subs = append(subs, sub)
	}
	return subs, nil
}

