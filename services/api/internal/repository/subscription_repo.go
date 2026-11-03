package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSubscriptionNotFound = errors.New("organization subscription not found")
	ErrCannotAssignArchived = errors.New("cannot assign an archived plan version to an organization")
	ErrCannotAssignDraft    = errors.New("cannot assign an unpublished draft plan version; plan must be published first")
)

// SubscriptionRepository manages organization subscription assignments and status lifecycles.
type SubscriptionRepository struct {
	pool *pgxpool.Pool
}

// NewSubscriptionRepository creates a new subscription repository.
func NewSubscriptionRepository(pool *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{pool: pool}
}

// AssignPlan assigns a published plan version to an organization, replacing any existing active subscription.
// If existing usage exceeds the newly assigned plan limits, warnings are generated but resources are preserved.
func (r *SubscriptionRepository) AssignPlan(ctx context.Context, orgID, planVersionID, assignedBy, adminNote string) (*OrganizationSubscription, []string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	// 1. Lock the organization state to prevent concurrent plan assignments on the same tenant
	var orgName string
	err = tx.QueryRow(ctx, "SELECT name FROM organizations WHERE id = $1 AND deleted_at IS NULL FOR UPDATE", orgID).Scan(&orgName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrOrgNotFound
		}
		return nil, nil, fmt.Errorf("query org: %w", err)
	}

	// 2. Verify plan version is published and plan is active
	var planID, planName, planSlug, versionStatus, planStatus string
	var versionNum int
	var priceAmount *int64
	var currency, billingInterval *string
	var limits PlanLimits

	err = tx.QueryRow(ctx, `
		SELECT 
			p.id, p.name, p.slug, p.status, spv.version, spv.status,
			spv.price_amount, spv.currency, spv.billing_interval,
			spv.max_clients, spv.max_assets, spv.max_monitors, spv.max_heartbeats, spv.max_members,
			spv.min_check_interval_seconds, spv.retention_days, spv.custom_branding_enabled, spv.public_reports_enabled
		FROM subscription_plan_versions spv
		JOIN subscription_plans p ON p.id = spv.plan_id
		WHERE spv.id = $1
	`, planVersionID).Scan(
		&planID, &planName, &planSlug, &planStatus, &versionNum, &versionStatus,
		&priceAmount, &currency, &billingInterval,
		&limits.MaxClients, &limits.MaxAssets, &limits.MaxMonitors, &limits.MaxHeartbeats, &limits.MaxMembers,
		&limits.MinCheckIntervalSeconds, &limits.RetentionDays, &limits.CustomBrandingEnabled, &limits.PublicReportsEnabled,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrPlanVersionNotFound
		}
		return nil, nil, fmt.Errorf("query plan version: %w", err)
	}

	if versionStatus != "published" {
		return nil, nil, ErrCannotAssignDraft
	}
	if planStatus != "active" {
		return nil, nil, fmt.Errorf("cannot assign plan with status %s: plan must be active", planStatus)
	}

	// 3. Mark any existing active subscription as canceled/replaced
	_, err = tx.Exec(ctx, `
		UPDATE organization_subscriptions
		SET status = 'canceled', ends_at = NOW(), updated_at = NOW()
		WHERE organization_id = $1 AND status = 'active'
	`, orgID)
	if err != nil {
		return nil, nil, fmt.Errorf("deactivate current subscription: %w", err)
	}

	// 4. Insert new active subscription
	var sub OrganizationSubscription
	err = tx.QueryRow(ctx, `
		INSERT INTO organization_subscriptions (
			organization_id, plan_version_id, status, started_at, assigned_by, administrative_note
		) VALUES (
			$1, $2, 'active', NOW(), $3, $4
		)
		RETURNING id, organization_id, plan_version_id, status, started_at, ends_at, assigned_by, administrative_note, created_at, updated_at
	`, orgID, planVersionID, assignedBy, adminNote).Scan(
		&sub.ID, &sub.OrganizationID, &sub.PlanVersionID, &sub.Status, &sub.StartedAt, &sub.EndsAt,
		&sub.AssignedBy, &sub.AdministrativeNote, &sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("insert subscription: %w", err)
	}

	sub.OrganizationName = orgName
	sub.PlanID = planID
	sub.PlanName = planName
	sub.PlanSlug = planSlug
	sub.PlanVersion = versionNum
	sub.PriceAmount = priceAmount
	sub.Currency = currency
	sub.BillingInterval = billingInterval
	sub.Limits = limits

	// 5. Query current usage using the definer function to detect over-limit warnings
	var usage PlatformOrganizationUsage
	err = tx.QueryRow(ctx, "SELECT * FROM get_platform_organization_usage($1)", orgID).Scan(
		&usage.ClientCount,
		&usage.AssetCount,
		&usage.MonitorCount,
		&usage.HeartbeatCount,
		&usage.MemberCount,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("evaluate usage: %w", err)
	}

	var warnings []string
	if limits.MaxClients >= 0 && usage.ClientCount > int64(limits.MaxClients) {
		warnings = append(warnings, fmt.Sprintf("Organization currently has %d clients, exceeding the new limit of %d.", usage.ClientCount, limits.MaxClients))
	}
	if limits.MaxAssets >= 0 && usage.AssetCount > int64(limits.MaxAssets) {
		warnings = append(warnings, fmt.Sprintf("Organization currently has %d assets, exceeding the new limit of %d.", usage.AssetCount, limits.MaxAssets))
	}
	if limits.MaxMonitors >= 0 && usage.MonitorCount > int64(limits.MaxMonitors) {
		warnings = append(warnings, fmt.Sprintf("Organization currently has %d monitors, exceeding the new limit of %d.", usage.MonitorCount, limits.MaxMonitors))
	}
	if limits.MaxHeartbeats >= 0 && usage.HeartbeatCount > int64(limits.MaxHeartbeats) {
		warnings = append(warnings, fmt.Sprintf("Organization currently has %d heartbeats, exceeding the new limit of %d.", usage.HeartbeatCount, limits.MaxHeartbeats))
	}
	if limits.MaxMembers >= 0 && usage.MemberCount > int64(limits.MaxMembers) {
		warnings = append(warnings, fmt.Sprintf("Organization currently has %d members, exceeding the new limit of %d.", usage.MemberCount, limits.MaxMembers))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit subscription assignment: %w", err)
	}

	return &sub, warnings, nil
}

// CheckOverLimitWarnings checks whether an organization's current usage exceeds a plan version's limits.
func (r *SubscriptionRepository) CheckOverLimitWarnings(ctx context.Context, orgID, planVersionID string) ([]string, error) {
	var limits PlanLimits
	err := r.pool.QueryRow(ctx, `
		SELECT 
			max_clients, max_assets, max_monitors, max_heartbeats, max_members,
			min_check_interval_seconds, retention_days, custom_branding_enabled, public_reports_enabled
		FROM subscription_plan_versions
		WHERE id = $1
	`, planVersionID).Scan(
		&limits.MaxClients, &limits.MaxAssets, &limits.MaxMonitors, &limits.MaxHeartbeats, &limits.MaxMembers,
		&limits.MinCheckIntervalSeconds, &limits.RetentionDays, &limits.CustomBrandingEnabled, &limits.PublicReportsEnabled,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPlanVersionNotFound
		}
		return nil, fmt.Errorf("query plan version limits: %w", err)
	}

	var usage PlatformOrganizationUsage
	err = r.pool.QueryRow(ctx, "SELECT * FROM get_platform_organization_usage($1)", orgID).Scan(
		&usage.ClientCount,
		&usage.AssetCount,
		&usage.MonitorCount,
		&usage.HeartbeatCount,
		&usage.MemberCount,
	)
	if err != nil {
		return nil, fmt.Errorf("evaluate usage: %w", err)
	}

	var warnings []string
	if limits.MaxClients >= 0 && usage.ClientCount > int64(limits.MaxClients) {
		warnings = append(warnings, fmt.Sprintf("Organization currently has %d clients, exceeding the new limit of %d.", usage.ClientCount, limits.MaxClients))
	}
	if limits.MaxAssets >= 0 && usage.AssetCount > int64(limits.MaxAssets) {
		warnings = append(warnings, fmt.Sprintf("Organization currently has %d assets, exceeding the new limit of %d.", usage.AssetCount, limits.MaxAssets))
	}
	if limits.MaxMonitors >= 0 && usage.MonitorCount > int64(limits.MaxMonitors) {
		warnings = append(warnings, fmt.Sprintf("Organization currently has %d monitors, exceeding the new limit of %d.", usage.MonitorCount, limits.MaxMonitors))
	}
	if limits.MaxHeartbeats >= 0 && usage.HeartbeatCount > int64(limits.MaxHeartbeats) {
		warnings = append(warnings, fmt.Sprintf("Organization currently has %d heartbeats, exceeding the new limit of %d.", usage.HeartbeatCount, limits.MaxHeartbeats))
	}
	if limits.MaxMembers >= 0 && usage.MemberCount > int64(limits.MaxMembers) {
		warnings = append(warnings, fmt.Sprintf("Organization currently has %d members, exceeding the new limit of %d.", usage.MemberCount, limits.MaxMembers))
	}

	return warnings, nil
}

// GetActiveSubscriptionForOrg retrieves the active plan version and limits for an organization.
// Returns nil if no active subscription is assigned (allowing fallback to legacy unconstrained behavior).
func (r *SubscriptionRepository) GetActiveSubscriptionForOrg(ctx context.Context, orgID string) (*OrganizationSubscription, error) {
	var sub OrganizationSubscription
	var subPlanID, subPlanName, subPlanSlug string
	var subPlanVersion int
	var subPriceAmount *int64
	var subCurrency, subBillingInterval *string
	var limits PlanLimits

	err := r.pool.QueryRow(ctx, `
		SELECT 
			s.id, s.organization_id, s.plan_version_id, s.status, s.started_at, s.ends_at,
			s.assigned_by, s.administrative_note, s.created_at, s.updated_at,
			p.id, p.name, p.slug, pv.version, pv.price_amount, pv.currency, pv.billing_interval,
			pv.max_clients, pv.max_assets, pv.max_monitors, pv.max_heartbeats, pv.max_members,
			pv.min_check_interval_seconds, pv.retention_days, pv.custom_branding_enabled, pv.public_reports_enabled
		FROM organization_subscriptions s
		JOIN subscription_plan_versions pv ON pv.id = s.plan_version_id
		JOIN subscription_plans p ON p.id = pv.plan_id
		WHERE s.organization_id = $1 AND s.status = 'active'
		ORDER BY s.started_at DESC
		LIMIT 1
	`, orgID).Scan(
		&sub.ID, &sub.OrganizationID, &sub.PlanVersionID, &sub.Status, &sub.StartedAt, &sub.EndsAt,
		&sub.AssignedBy, &sub.AdministrativeNote, &sub.CreatedAt, &sub.UpdatedAt,
		&subPlanID, &subPlanName, &subPlanSlug, &subPlanVersion, &subPriceAmount, &subCurrency, &subBillingInterval,
		&limits.MaxClients, &limits.MaxAssets, &limits.MaxMonitors, &limits.MaxHeartbeats, &limits.MaxMembers,
		&limits.MinCheckIntervalSeconds, &limits.RetentionDays, &limits.CustomBrandingEnabled, &limits.PublicReportsEnabled,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query active subscription: %w", err)
	}

	sub.PlanID = subPlanID
	sub.PlanName = subPlanName
	sub.PlanSlug = subPlanSlug
	sub.PlanVersion = subPlanVersion
	sub.PriceAmount = subPriceAmount
	sub.Currency = subCurrency
	sub.BillingInterval = subBillingInterval
	sub.Limits = limits

	return &sub, nil
}

// GetLatestSubscriptionForOrg retrieves the most recent subscription assigned to the organization, regardless of status.
// Returns (nil, nil) if the organization has never had a subscription assigned (legacy unassigned organization).
func (r *SubscriptionRepository) GetLatestSubscriptionForOrg(ctx context.Context, orgID string) (*OrganizationSubscription, error) {
	var sub OrganizationSubscription
	var subPlanID, subPlanName, subPlanSlug string
	var subPlanVersion int
	var subPriceAmount *int64
	var subCurrency, subBillingInterval *string
	var limits PlanLimits

	err := r.pool.QueryRow(ctx, `
		SELECT 
			s.id, s.organization_id, s.plan_version_id, s.status, s.started_at, s.ends_at,
			s.assigned_by, s.administrative_note, s.created_at, s.updated_at,
			p.id, p.name, p.slug, pv.version, pv.price_amount, pv.currency, pv.billing_interval,
			pv.max_clients, pv.max_assets, pv.max_monitors, pv.max_heartbeats, pv.max_members,
			pv.min_check_interval_seconds, pv.retention_days, pv.custom_branding_enabled, pv.public_reports_enabled
		FROM organization_subscriptions s
		JOIN subscription_plan_versions pv ON pv.id = s.plan_version_id
		JOIN subscription_plans p ON p.id = pv.plan_id
		WHERE s.organization_id = $1
		ORDER BY s.started_at DESC, s.created_at DESC
		LIMIT 1
	`, orgID).Scan(
		&sub.ID, &sub.OrganizationID, &sub.PlanVersionID, &sub.Status, &sub.StartedAt, &sub.EndsAt,
		&sub.AssignedBy, &sub.AdministrativeNote, &sub.CreatedAt, &sub.UpdatedAt,
		&subPlanID, &subPlanName, &subPlanSlug, &subPlanVersion, &subPriceAmount, &subCurrency, &subBillingInterval,
		&limits.MaxClients, &limits.MaxAssets, &limits.MaxMonitors, &limits.MaxHeartbeats, &limits.MaxMembers,
		&limits.MinCheckIntervalSeconds, &limits.RetentionDays, &limits.CustomBrandingEnabled, &limits.PublicReportsEnabled,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query latest subscription: %w", err)
	}

	sub.PlanID = subPlanID
	sub.PlanName = subPlanName
	sub.PlanSlug = subPlanSlug
	sub.PlanVersion = subPlanVersion
	sub.PriceAmount = subPriceAmount
	sub.Currency = subCurrency
	sub.BillingInterval = subBillingInterval
	sub.Limits = limits

	return &sub, nil
}

// ListSubscriptions retrieves subscriptions with optional filters on orgID and status.
func (r *SubscriptionRepository) ListSubscriptions(ctx context.Context, orgID, status string, limit, offset int) ([]OrganizationSubscription, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var conditions []string
	var args []interface{}
	idx := 1

	if orgID != "" {
		conditions = append(conditions, fmt.Sprintf("os.organization_id = $%d", idx))
		args = append(args, orgID)
		idx++
	}
	if status != "" {
		conditions = append(conditions, fmt.Sprintf("os.status = $%d", idx))
		args = append(args, status)
		idx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM organization_subscriptions os %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count subscriptions: %w", err)
	}

	selectQuery := fmt.Sprintf(`
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
		%s
		ORDER BY os.started_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, idx, idx+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("select subscriptions: %w", err)
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
			return nil, 0, fmt.Errorf("scan subscription: %w", err)
		}
		sub.Limits = limits
		subs = append(subs, sub)
	}

	return subs, total, nil
}

// UpdateSubscriptionStatus changes subscription status ('active', 'suspended', 'canceled').
func (r *SubscriptionRepository) UpdateSubscriptionStatus(ctx context.Context, subID, newStatus, actorID, note string) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE organization_subscriptions
		SET status = $1, administrative_note = COALESCE($2, administrative_note), updated_at = NOW()
		WHERE id = $3
	`, newStatus, note, subID)
	if err != nil {
		return fmt.Errorf("update subscription status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrSubscriptionNotFound
	}
	return nil
}

