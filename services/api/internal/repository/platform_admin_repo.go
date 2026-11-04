package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrPlatformAdminRequired = errors.New("platform administrator privilege required")
)

// PlatformAdminRepository handles all data access for global platform administration.
type PlatformAdminRepository struct {
	pool *pgxpool.Pool
}

// NewPlatformAdminRepository creates a new platform admin repository.
func NewPlatformAdminRepository(pool *pgxpool.Pool) *PlatformAdminRepository {
	return &PlatformAdminRepository{pool: pool}
}

// IsUserPlatformAdmin determines if a user holds an active, unrevoked global platform admin grant.
func (r *PlatformAdminRepository) IsUserPlatformAdmin(ctx context.Context, userID string) (bool, error) {
	if userID == "" {
		return false, nil
	}

	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM platform_admin_grants
			WHERE user_id = $1 AND role = 'platform_admin' AND revoked_at IS NULL
		)
	`, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check platform admin grant: %w", err)
	}

	return exists, nil
}

const PlatformAdminLockID = 982347103

// GrantAdmin grants global platform admin privileges to a user.
// Enforces role validation ('platform_admin'), acquires an advisory lock to prevent races,
// and ensures duplicate active grants are not created.
func (r *PlatformAdminRepository) GrantAdmin(ctx context.Context, userID string, grantedBy *string, reason string) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	// Acquire advisory lock to serialize grant modifications
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", PlatformAdminLockID); err != nil {
		return "", fmt.Errorf("acquire admin lock: %w", err)
	}

	// Verify user exists
	var email string
	err = tx.QueryRow(ctx, "SELECT email FROM users WHERE id = $1", userID).Scan(&email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("user %s does not exist", userID)
		}
		return "", fmt.Errorf("query user: %w", err)
	}

	// Check if already actively granted
	var activeID string
	err = tx.QueryRow(ctx, "SELECT id FROM platform_admin_grants WHERE user_id = $1 AND role = 'platform_admin' AND revoked_at IS NULL", userID).Scan(&activeID)
	if err == nil {
		return activeID, nil // idempotent existing active grant
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("check active grant: %w", err)
	}

	var grantID string
	err = tx.QueryRow(ctx, `
		INSERT INTO platform_admin_grants (user_id, role, granted_at, granted_by, reason)
		VALUES ($1, 'platform_admin', NOW(), $2, $3)
		RETURNING id
	`, userID, grantedBy, reason).Scan(&grantID)
	if err != nil {
		return "", fmt.Errorf("grant platform admin: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit grant: %w", err)
	}

	return grantID, nil
}

// RevokeAdmin revokes active platform admin privileges for a user.
// Uses pg_advisory_xact_lock to strictly protect against concurrent last-admin revocations.
func (r *PlatformAdminRepository) RevokeAdmin(ctx context.Context, userID string, revokedBy *string, reason string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	// Serialize with transaction advisory lock
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", PlatformAdminLockID); err != nil {
		return fmt.Errorf("acquire admin lock: %w", err)
	}

	// Count remaining active platform admins
	var activeCount int
	err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM platform_admin_grants WHERE role = 'platform_admin' AND revoked_at IS NULL").Scan(&activeCount)
	if err != nil {
		return fmt.Errorf("count active admins: %w", err)
	}

	// Check if target user has active grant
	var grantID string
	err = tx.QueryRow(ctx, "SELECT id FROM platform_admin_grants WHERE user_id = $1 AND role = 'platform_admin' AND revoked_at IS NULL", userID).Scan(&grantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("user %s does not have an active platform_admin grant", userID)
		}
		return fmt.Errorf("query active grant: %w", err)
	}

	if activeCount <= 1 {
		return fmt.Errorf("CANNOT REVOKE: last-administrator protection active (only %d platform administrator remains)", activeCount)
	}

	_, err = tx.Exec(ctx, `
		UPDATE platform_admin_grants
		SET revoked_at = NOW(), revoked_by = $2, reason = COALESCE(reason || ' | Revocation: ' || $3, $3), updated_at = NOW()
		WHERE id = $1
	`, grantID, revokedBy, reason)
	if err != nil {
		return fmt.Errorf("revoke platform admin: %w", err)
	}

	return tx.Commit(ctx)
}

// GetOverviewStats retrieves platform-wide aggregates via the hardened definer function.
func (r *PlatformAdminRepository) GetOverviewStats(ctx context.Context) (*PlatformOverviewStats, error) {
	stats := &PlatformOverviewStats{}
	err := r.pool.QueryRow(ctx, "SELECT * FROM get_platform_overview_stats()").Scan(
		&stats.TotalUsers,
		&stats.ActiveUsers,
		&stats.SuspendedUsers,
		&stats.TotalOrganizations,
		&stats.ActiveOrganizations,
		&stats.SuspendedOrganizations,
		&stats.PublishedPlans,
		&stats.ActiveSubscriptions,
		&stats.UnassignedOrganizations,
		&stats.PublishedOffers,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query overview stats: %w", err)
	}
	return stats, nil
}

// ListUsers queries registered users across all organizations with search, status filtering, and pagination.
func (r *PlatformAdminRepository) ListUsers(ctx context.Context, search, status string, limit, offset int) ([]PlatformUserItem, int, error) {
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

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("u.status = $%d", idx))
		args = append(args, status)
		idx++
	}

	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(u.email ILIKE $%d OR u.display_name ILIKE $%d)", idx, idx))
		args = append(args, "%"+search+"%")
		idx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users u %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	selectQuery := fmt.Sprintf(`
		SELECT 
			u.id, 
			u.email, 
			u.display_name, 
			u.status, 
			(u.email_verified_at IS NOT NULL) AS email_verified, 
			u.created_at,
			COALESCE((SELECT COUNT(*) FROM organization_memberships om WHERE om.user_id = u.id), 0) AS org_count,
			EXISTS (SELECT 1 FROM platform_admin_grants pag WHERE pag.user_id = u.id AND pag.revoked_at IS NULL) AS is_admin
		FROM users u
		%s
		ORDER BY u.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, idx, idx+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("select users: %w", err)
	}
	defer rows.Close()

	var users []PlatformUserItem
	for rows.Next() {
		var item PlatformUserItem
		if err := rows.Scan(
			&item.ID,
			&item.Email,
			&item.DisplayName,
			&item.Status,
			&item.EmailVerified,
			&item.CreatedAt,
			&item.OrganizationsCount,
			&item.IsPlatformAdmin,
		); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, item)
	}

	return users, total, nil
}

// GetUserDetail retrieves full profile details for a user, including organizations and platform admin status.
func (r *PlatformAdminRepository) GetUserDetail(ctx context.Context, userID string) (*PlatformUserDetail, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, display_name, email_verified_at, status, created_at, updated_at
		FROM users
		WHERE id = $1
	`, userID).Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&user.EmailVerifiedAt,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("query user: %w", err)
	}

	isAdmin, err := r.IsUserPlatformAdmin(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Retrieve memberships via definer function
	rows, err := r.pool.Query(ctx, "SELECT * FROM get_platform_user_memberships($1)", userID)
	if err != nil {
		return nil, fmt.Errorf("query user memberships: %w", err)
	}
	defer rows.Close()

	var memberships []PlatformUserMembership
	for rows.Next() {
		var m PlatformUserMembership
		if err := rows.Scan(
			&m.OrganizationID,
			&m.OrganizationName,
			&m.OrganizationSlug,
			&m.OrganizationStatus,
			&m.Role,
			&m.JoinedAt,
		); err != nil {
			return nil, fmt.Errorf("scan membership: %w", err)
		}
		memberships = append(memberships, m)
	}

	return &PlatformUserDetail{
		User:            user,
		IsPlatformAdmin: isAdmin,
		Memberships:     memberships,
	}, nil
}

// SetUserStatus updates user status and revokes active sessions if suspended.
func (r *PlatformAdminRepository) SetUserStatus(ctx context.Context, userID, status string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // nolint:errcheck

	result, err := tx.Exec(ctx, `
		UPDATE users 
		SET status = $1, updated_at = NOW() 
		WHERE id = $2
	`, status, userID)
	if err != nil {
		return fmt.Errorf("update user status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	// If deactivated, revoke all active user sessions immediately
	if status == "deactivated" {
		_, err = tx.Exec(ctx, `
			UPDATE sessions 
			SET revoked_at = NOW() 
			WHERE user_id = $1 AND revoked_at IS NULL
		`, userID)
		if err != nil {
			return fmt.Errorf("revoke sessions: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// ListOrganizations retrieves paginated organizations using the hardened definer function.
func (r *PlatformAdminRepository) ListOrganizations(ctx context.Context, search, status string, limit, offset int) ([]PlatformOrganizationItem, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := r.pool.Query(ctx, "SELECT * FROM get_platform_organizations_list($1, $2, $3, $4)", search, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query organizations list: %w", err)
	}
	defer rows.Close()

	var orgs []PlatformOrganizationItem
	var totalCount int64

	for rows.Next() {
		var item PlatformOrganizationItem
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Slug,
			&item.Status,
			&item.Locale,
			&item.Timezone,
			&item.CreatedAt,
			&item.OwnerID,
			&item.OwnerEmail,
			&item.OwnerName,
			&item.MemberCount,
			&item.ClientCount,
			&item.AssetCount,
			&item.MonitorCount,
			&item.HeartbeatCount,
			&item.PlanName,
			&item.PlanSlug,
			&item.PlanVersion,
			&item.SubscriptionStatus,
			&totalCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan organization item: %w", err)
		}
		orgs = append(orgs, item)
	}

	return orgs, int(totalCount), nil
}

// GetOrganizationDetail retrieves single organization details, members, usage metrics, and current subscription.
func (r *PlatformAdminRepository) GetOrganizationDetail(ctx context.Context, orgID string) (*PlatformOrganizationDetail, error) {
	var org Organization
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, locale, timezone, status, created_at, updated_at, deleted_at
		FROM organizations
		WHERE id = $1 AND deleted_at IS NULL
	`, orgID).Scan(
		&org.ID,
		&org.Name,
		&org.Slug,
		&org.Locale,
		&org.Timezone,
		&org.Status,
		&org.CreatedAt,
		&org.UpdatedAt,
		&org.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrgNotFound
		}
		return nil, fmt.Errorf("query org: %w", err)
	}

	// Members via definer function
	memberRows, err := r.pool.Query(ctx, "SELECT * FROM get_platform_org_members($1)", orgID)
	if err != nil {
		return nil, fmt.Errorf("query org members: %w", err)
	}
	defer memberRows.Close()

	var members []PlatformOrgMember
	var owner *PlatformOrgMember
	for memberRows.Next() {
		var m PlatformOrgMember
		if err := memberRows.Scan(&m.UserID, &m.Email, &m.DisplayName, &m.UserStatus, &m.Role, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan org member: %w", err)
		}
		members = append(members, m)
		if m.Role == "owner" && owner == nil {
			mCopy := m
			owner = &mCopy
		}
	}

	// Usage via definer function
	var usage PlatformOrganizationUsage
	err = r.pool.QueryRow(ctx, "SELECT * FROM get_platform_organization_usage($1)", orgID).Scan(
		&usage.ClientCount,
		&usage.AssetCount,
		&usage.MonitorCount,
		&usage.HeartbeatCount,
		&usage.MemberCount,
	)
	if err != nil {
		return nil, fmt.Errorf("query org usage: %w", err)
	}

	// Active Subscription
	var sub OrganizationSubscription
	var subPlanID, subPlanName, subPlanSlug string
	var subPlanVersion int
	var subPriceAmount *int64
	var subCurrency, subBillingInterval *string
	var limits PlanLimits
	var featuresJSON []byte

	err = r.pool.QueryRow(ctx, `
		SELECT 
			s.id, s.organization_id, s.plan_version_id, s.status, s.started_at, s.ends_at,
			s.assigned_by, s.administrative_note, s.created_at, s.updated_at,
			p.id, p.name, p.slug, pv.version, pv.price_amount, pv.currency, pv.billing_interval,
			pv.max_clients, pv.max_assets, pv.max_monitors, pv.max_heartbeats, pv.max_members,
			pv.min_check_interval_seconds, pv.retention_days, pv.custom_branding_enabled, pv.public_reports_enabled,
			pv.features
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
		&featuresJSON,
	)

	var activeSub *OrganizationSubscription
	var overLimits []string

	if err == nil {
		sub.PlanID = subPlanID
		sub.PlanName = subPlanName
		sub.PlanSlug = subPlanSlug
		sub.PlanVersion = subPlanVersion
		sub.PriceAmount = subPriceAmount
		sub.Currency = subCurrency
		sub.BillingInterval = subBillingInterval
		sub.Limits = limits
		activeSub = &sub

		// Evaluate over-limit conditions
		if limits.MaxClients >= 0 && usage.ClientCount > int64(limits.MaxClients) {
			overLimits = append(overLimits, fmt.Sprintf("Clients: %d used of %d allowed", usage.ClientCount, limits.MaxClients))
		}
		if limits.MaxAssets >= 0 && usage.AssetCount > int64(limits.MaxAssets) {
			overLimits = append(overLimits, fmt.Sprintf("Assets: %d used of %d allowed", usage.AssetCount, limits.MaxAssets))
		}
		if limits.MaxMonitors >= 0 && usage.MonitorCount > int64(limits.MaxMonitors) {
			overLimits = append(overLimits, fmt.Sprintf("Monitors: %d used of %d allowed", usage.MonitorCount, limits.MaxMonitors))
		}
		if limits.MaxHeartbeats >= 0 && usage.HeartbeatCount > int64(limits.MaxHeartbeats) {
			overLimits = append(overLimits, fmt.Sprintf("Heartbeats: %d used of %d allowed", usage.HeartbeatCount, limits.MaxHeartbeats))
		}
		if limits.MaxMembers >= 0 && usage.MemberCount > int64(limits.MaxMembers) {
			overLimits = append(overLimits, fmt.Sprintf("Members: %d joined of %d allowed", usage.MemberCount, limits.MaxMembers))
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("query active subscription: %w", err)
	}

	return &PlatformOrganizationDetail{
		Organization: org,
		Owner:        owner,
		Members:      members,
		Usage:        usage,
		Subscription: activeSub,
		OverLimits:   overLimits,
	}, nil
}

// SetOrganizationStatus updates an organization's status ('active' or 'suspended').
func (r *PlatformAdminRepository) SetOrganizationStatus(ctx context.Context, orgID, status string) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE organizations 
		SET status = $1, updated_at = NOW() 
		WHERE id = $2 AND deleted_at IS NULL
	`, status, orgID)
	if err != nil {
		return fmt.Errorf("update org status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrOrgNotFound
	}
	return nil
}

// RecordAuditLog records an immutable administrative event into platform_admin_audit_logs.
func (r *PlatformAdminRepository) RecordAuditLog(ctx context.Context, actorUserID, action, targetType, targetID, reason string, metadata map[string]interface{}, ip, userAgent string) error {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		metaJSON = []byte("{}")
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO platform_admin_audit_logs (actor_user_id, action, target_type, target_id, reason, metadata_json, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8)
	`, actorUserID, action, targetType, targetID, reason, string(metaJSON), ip, userAgent)
	if err != nil {
		return fmt.Errorf("insert platform admin audit log: %w", err)
	}
	return nil
}

// ListAuditLogs retrieves paginated audit logs with optional filters.
func (r *PlatformAdminRepository) ListAuditLogs(ctx context.Context, actorID, action, targetType, targetID string, fromDate, toDate *time.Time, limit, offset int) ([]PlatformAdminAuditLog, int, error) {
	if limit <= 0 {
		limit = 25
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

	if actorID != "" {
		conditions = append(conditions, fmt.Sprintf("al.actor_user_id = $%d", idx))
		args = append(args, actorID)
		idx++
	}
	if action != "" {
		conditions = append(conditions, fmt.Sprintf("al.action = $%d", idx))
		args = append(args, action)
		idx++
	}
	if targetType != "" {
		conditions = append(conditions, fmt.Sprintf("al.target_type = $%d", idx))
		args = append(args, targetType)
		idx++
	}
	if targetID != "" {
		conditions = append(conditions, fmt.Sprintf("al.target_id = $%d", idx))
		args = append(args, targetID)
		idx++
	}
	if fromDate != nil {
		conditions = append(conditions, fmt.Sprintf("al.created_at >= $%d", idx))
		args = append(args, *fromDate)
		idx++
	}
	if toDate != nil {
		conditions = append(conditions, fmt.Sprintf("al.created_at <= $%d", idx))
		args = append(args, *toDate)
		idx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM platform_admin_audit_logs al %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	selectQuery := fmt.Sprintf(`
		SELECT 
			al.id, al.actor_user_id, u.email, u.display_name,
			al.action, al.target_type, al.target_id, al.reason,
			al.metadata_json, al.ip_address, al.user_agent, al.created_at
		FROM platform_admin_audit_logs al
		JOIN users u ON u.id = al.actor_user_id
		%s
		ORDER BY al.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, idx, idx+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("select audit logs: %w", err)
	}
	defer rows.Close()

	var logs []PlatformAdminAuditLog
	for rows.Next() {
		var l PlatformAdminAuditLog
		var rawMeta []byte
		if err := rows.Scan(
			&l.ID,
			&l.ActorUserID,
			&l.ActorEmail,
			&l.ActorName,
			&l.Action,
			&l.TargetType,
			&l.TargetID,
			&l.Reason,
			&rawMeta,
			&l.IPAddress,
			&l.UserAgent,
			&l.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan audit log: %w", err)
		}
		if len(rawMeta) > 0 {
			_ = json.Unmarshal(rawMeta, &l.MetadataJSON)
		}
		logs = append(logs, l)
	}

	return logs, total, nil
}



// OverviewRecentSignup represents recent user registration for platform overview.
type OverviewRecentSignup struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// OverviewSubscriptionDistribution represents subscription breakdown counters.
type OverviewSubscriptionDistribution struct {
	Category string `json:"category"`
	Count    int64  `json:"count"`
}

// PlatformOverviewResponse aggregates KPI counters and real operational metrics.
type PlatformOverviewResponse struct {
	Stats                          *PlatformOverviewStats             `json:"stats"`
	RecentSignups                  []OverviewRecentSignup             `json:"recent_signups"`
	RecentActivity                 []PlatformAdminAuditLog            `json:"recent_activity"`
	SubscriptionDistributionLocale []OverviewSubscriptionDistribution `json:"subscription_distribution_locale"`
	SubscriptionDistributionPlan   []OverviewSubscriptionDistribution `json:"subscription_distribution_plan"`
	RecentSecurityEvents           []PlatformAdminAuditLog            `json:"recent_security_events"`

	// Flat fields for backward-compatibility with existing tests
	TotalUsers              int64 `json:"total_users"`
	ActiveUsers             int64 `json:"active_users"`
	SuspendedUsers          int64 `json:"suspended_users"`
	TotalOrganizations      int64 `json:"total_organizations"`
	ActiveOrganizations     int64 `json:"active_organizations"`
	SuspendedOrganizations  int64 `json:"suspended_organizations"`
	PublishedPlans          int64 `json:"published_plans"`
	ActiveSubscriptions     int64 `json:"active_subscriptions"`
	UnassignedOrganizations int64 `json:"unassigned_organizations"`
	PublishedOffers         int64 `json:"published_offers"`
}

// GetPlatformOverview collects real KPI stats and operational platform data.
func (r *PlatformAdminRepository) GetPlatformOverview(ctx context.Context) (*PlatformOverviewResponse, error) {
	stats, err := r.GetOverviewStats(ctx)
	if err != nil {
		return nil, err
	}

	resp := &PlatformOverviewResponse{
		Stats:                   stats,
		TotalUsers:              stats.TotalUsers,
		ActiveUsers:             stats.ActiveUsers,
		SuspendedUsers:          stats.SuspendedUsers,
		TotalOrganizations:      stats.TotalOrganizations,
		ActiveOrganizations:     stats.ActiveOrganizations,
		SuspendedOrganizations:  stats.SuspendedOrganizations,
		PublishedPlans:          stats.PublishedPlans,
		ActiveSubscriptions:     stats.ActiveSubscriptions,
		UnassignedOrganizations: stats.UnassignedOrganizations,
		PublishedOffers:         stats.PublishedOffers,
		RecentSignups:           []OverviewRecentSignup{},
		RecentActivity:          []PlatformAdminAuditLog{},
		SubscriptionDistributionLocale: []OverviewSubscriptionDistribution{},
		SubscriptionDistributionPlan:   []OverviewSubscriptionDistribution{},
		RecentSecurityEvents:           []PlatformAdminAuditLog{},
	}

	// 1. Recent Signups
	userRows, err := r.pool.Query(ctx, `
		SELECT id, email, display_name, status, created_at
		FROM users
		ORDER BY created_at DESC
		LIMIT 5
	`)
	if err == nil {
		defer userRows.Close()
		for userRows.Next() {
			var u OverviewRecentSignup
			if err := userRows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Status, &u.CreatedAt); err == nil {
				resp.RecentSignups = append(resp.RecentSignups, u)
			}
		}
	}

	// 2. Recent Administrative Activity
	recentLogs, _, err := r.ListAuditLogs(ctx, "", "", "", "", nil, nil, 6, 0)
	if err == nil {
		resp.RecentActivity = recentLogs
	}

	// 3. Subscription Distribution by Locale
	locRows, err := r.pool.Query(ctx, `
		SELECT COALESCE(NULLIF(so.locale, ''), 'unspecified') AS cat, COUNT(*)
		FROM organization_subscriptions os
		LEFT JOIN subscription_offer_versions sov ON sov.id = os.offer_version_id
		LEFT JOIN subscription_offers so ON so.id = sov.offer_id
		WHERE os.status = 'active'
		GROUP BY cat
		ORDER BY COUNT(*) DESC
	`)
	if err == nil {
		defer locRows.Close()
		for locRows.Next() {
			var dist OverviewSubscriptionDistribution
			if err := locRows.Scan(&dist.Category, &dist.Count); err == nil {
				resp.SubscriptionDistributionLocale = append(resp.SubscriptionDistributionLocale, dist)
			}
		}
	}

	// 4. Subscription Distribution by Plan
	planRows, err := r.pool.Query(ctx, `
		SELECT p.name, COUNT(*)
		FROM organization_subscriptions os
		JOIN subscription_plan_versions spv ON spv.id = os.plan_version_id
		JOIN subscription_plans p ON p.id = spv.plan_id
		WHERE os.status = 'active'
		GROUP BY p.name
		ORDER BY COUNT(*) DESC
		LIMIT 5
	`)
	if err == nil {
		defer planRows.Close()
		for planRows.Next() {
			var dist OverviewSubscriptionDistribution
			if err := planRows.Scan(&dist.Category, &dist.Count); err == nil {
				resp.SubscriptionDistributionPlan = append(resp.SubscriptionDistributionPlan, dist)
			}
		}
	}

	// 5. Recent Security Events
	secRows, err := r.pool.Query(ctx, `
		SELECT 
			al.id, al.actor_user_id, u.email, u.display_name,
			al.action, al.target_type, al.target_id, al.reason,
			al.metadata_json, al.ip_address, al.user_agent, al.created_at
		FROM platform_admin_audit_logs al
		JOIN users u ON u.id = al.actor_user_id
		WHERE al.action ILIKE '%suspend%' OR al.action ILIKE '%reactivate%' OR al.action ILIKE '%grant%' OR al.action ILIKE '%security%' OR al.action ILIKE '%localization%'
		ORDER BY al.created_at DESC
		LIMIT 5
	`)
	if err == nil {
		defer secRows.Close()
		for secRows.Next() {
			var l PlatformAdminAuditLog
			var rawMeta []byte
			if err := secRows.Scan(
				&l.ID, &l.ActorUserID, &l.ActorEmail, &l.ActorName,
				&l.Action, &l.TargetType, &l.TargetID, &l.Reason,
				&rawMeta, &l.IPAddress, &l.UserAgent, &l.CreatedAt,
			); err == nil {
				if len(rawMeta) > 0 {
					_ = json.Unmarshal(rawMeta, &l.MetadataJSON)
				}
				resp.RecentSecurityEvents = append(resp.RecentSecurityEvents, l)
			}
		}
	}

	return resp, nil
}
