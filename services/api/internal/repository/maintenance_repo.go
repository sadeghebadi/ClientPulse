package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrMaintenanceNotFound   = errors.New("maintenance activity not found")
	ErrInvalidCategory      = errors.New("invalid maintenance activity category")
	ErrInvalidDuration      = errors.New("duration must be between 0 and 14400 minutes (10 days)")
	ErrInvalidTitle         = errors.New("title is required and cannot exceed 200 characters")
	ErrInvalidDescription   = errors.New("description cannot be empty")
)

var ValidCategories = map[string]bool{
	"update":          true,
	"backup":          true,
	"bug_fix":         true,
	"performance":     true,
	"security_review": true,
	"content_change":  true,
	"configuration":   true,
	"other":           true,
}

type MaintenanceRepository struct {
	pool      *pgxpool.Pool
	auditRepo *AuditRepository
}

func NewMaintenanceRepository(pool *pgxpool.Pool, auditRepo *AuditRepository) *MaintenanceRepository {
	return &MaintenanceRepository{
		pool:      pool,
		auditRepo: auditRepo,
	}
}

type CreateMaintenanceParams struct {
	ClientID        string
	AssetID         *string
	Title           string
	Description     string
	Category        string
	PerformedAt     time.Time
	DurationMinutes *int
	CreatedBy       string
}

type UpdateMaintenanceParams struct {
	AssetID         *string
	Title           *string
	Description     *string
	Category        *string
	PerformedAt     *time.Time
	DurationMinutes *int
	UpdatedBy       string
}

type MaintenanceFilter struct {
	ClientID *string
	AssetID  *string
	Category *string
	From     *time.Time
	To       *time.Time
	Limit    int
	Offset   int
}

// CreateMaintenanceActivity records real maintenance work performed for a client.
func (r *MaintenanceRepository) CreateMaintenanceActivity(ctx context.Context, orgID string, params CreateMaintenanceParams) (*MaintenanceActivity, error) {
	title := strings.TrimSpace(params.Title)
	if title == "" || len(title) > 200 {
		return nil, ErrInvalidTitle
	}

	description := strings.TrimSpace(params.Description)
	if description == "" {
		return nil, ErrInvalidDescription
	}

	cat := strings.ToLower(strings.TrimSpace(params.Category))
	if !ValidCategories[cat] {
		return nil, ErrInvalidCategory
	}

	if params.DurationMinutes != nil {
		if *params.DurationMinutes < 0 || *params.DurationMinutes > 14400 {
			return nil, ErrInvalidDuration
		}
	}

	if params.PerformedAt.IsZero() {
		params.PerformedAt = time.Now().UTC()
	}

	var act MaintenanceActivity
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		// Validate client belongs to active org
		var clientExists bool
		err := tx.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM clients WHERE id = $1 AND organization_id = $2 AND archived_at IS NULL)
		`, params.ClientID, orgID).Scan(&clientExists)
		if err != nil {
			return err
		}
		if !clientExists {
			return errors.New("client does not belong to active organization or is archived")
		}

		// Validate asset if provided
		if params.AssetID != nil && *params.AssetID != "" {
			var assetValid bool
			err := tx.QueryRow(ctx, `
				SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1 AND client_id = $2 AND organization_id = $3 AND archived_at IS NULL)
			`, *params.AssetID, params.ClientID, orgID).Scan(&assetValid)
			if err != nil {
				return err
			}
			if !assetValid {
				return errors.New("asset does not belong to the selected client and organization")
			}
		} else {
			params.AssetID = nil
		}

		query := `
			INSERT INTO maintenance_activities (
				organization_id, client_id, asset_id, title, description,
				category, performed_at, duration_minutes, created_by, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
			RETURNING id, organization_id, client_id, asset_id, title, description,
			          category, performed_at, duration_minutes, created_by, created_at, updated_at
		`

		return tx.QueryRow(ctx, query,
			orgID, params.ClientID, params.AssetID, title, description,
			cat, params.PerformedAt, params.DurationMinutes, params.CreatedBy,
		).Scan(
			&act.ID, &act.OrganizationID, &act.ClientID, &act.AssetID, &act.Title, &act.Description,
			&act.Category, &act.PerformedAt, &act.DurationMinutes, &act.CreatedBy, &act.CreatedAt, &act.UpdatedAt,
		)
	})

	if err != nil {
		return nil, err
	}
	return &act, nil
}

// GetMaintenanceActivity fetches a single maintenance activity by ID.
func (r *MaintenanceRepository) GetMaintenanceActivity(ctx context.Context, orgID, activityID string) (*MaintenanceActivity, error) {
	var act MaintenanceActivity
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			SELECT ma.id, ma.organization_id, ma.client_id, ma.asset_id, ma.title, ma.description,
			       ma.category, ma.performed_at, ma.duration_minutes, ma.created_by, ma.created_at, ma.updated_at,
			       c.name AS client_name,
			       a.name AS asset_name,
			       u.display_name AS created_by_name,
			       u.email AS created_by_email
			FROM maintenance_activities ma
			JOIN clients c ON c.id = ma.client_id AND c.organization_id = ma.organization_id
			LEFT JOIN assets a ON a.id = ma.asset_id AND a.organization_id = ma.organization_id
			JOIN users u ON u.id = ma.created_by
			WHERE ma.id = $1 AND ma.organization_id = $2
		`

		return tx.QueryRow(ctx, query, activityID, orgID).Scan(
			&act.ID, &act.OrganizationID, &act.ClientID, &act.AssetID, &act.Title, &act.Description,
			&act.Category, &act.PerformedAt, &act.DurationMinutes, &act.CreatedBy, &act.CreatedAt, &act.UpdatedAt,
			&act.ClientName, &act.AssetName, &act.CreatedByName, &act.CreatedByEmail,
		)
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMaintenanceNotFound
		}
		return nil, err
	}
	return &act, nil
}

// ListMaintenanceActivities retrieves paginated maintenance activities according to filter criteria.
func (r *MaintenanceRepository) ListMaintenanceActivities(ctx context.Context, orgID string, filter MaintenanceFilter) ([]*MaintenanceActivity, int, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	var activities []*MaintenanceActivity
	var totalCount int

	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		conditions := []string{"ma.organization_id = $1"}
		args := []interface{}{orgID}
		argIdx := 2

		if filter.ClientID != nil && *filter.ClientID != "" {
			conditions = append(conditions, fmt.Sprintf("ma.client_id = $%d", argIdx))
			args = append(args, *filter.ClientID)
			argIdx++
		}

		if filter.AssetID != nil && *filter.AssetID != "" {
			conditions = append(conditions, fmt.Sprintf("ma.asset_id = $%d", argIdx))
			args = append(args, *filter.AssetID)
			argIdx++
		}

		if filter.Category != nil && *filter.Category != "" {
			conditions = append(conditions, fmt.Sprintf("ma.category = $%d", argIdx))
			args = append(args, strings.ToLower(*filter.Category))
			argIdx++
		}

		if filter.From != nil {
			conditions = append(conditions, fmt.Sprintf("ma.performed_at >= $%d", argIdx))
			args = append(args, *filter.From)
			argIdx++
		}

		if filter.To != nil {
			conditions = append(conditions, fmt.Sprintf("ma.performed_at <= $%d", argIdx))
			args = append(args, *filter.To)
			argIdx++
		}

		whereClause := strings.Join(conditions, " AND ")

		// 1. Total count
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM maintenance_activities ma WHERE %s", whereClause)
		if err := tx.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
			return fmt.Errorf("failed to count maintenance activities: %w", err)
		}

		// 2. Paginated rows
		query := fmt.Sprintf(`
			SELECT ma.id, ma.organization_id, ma.client_id, ma.asset_id, ma.title, ma.description,
			       ma.category, ma.performed_at, ma.duration_minutes, ma.created_by, ma.created_at, ma.updated_at,
			       c.name AS client_name,
			       a.name AS asset_name,
			       u.display_name AS created_by_name,
			       u.email AS created_by_email
			FROM maintenance_activities ma
			JOIN clients c ON c.id = ma.client_id AND c.organization_id = ma.organization_id
			LEFT JOIN assets a ON a.id = ma.asset_id AND a.organization_id = ma.organization_id
			JOIN users u ON u.id = ma.created_by
			WHERE %s
			ORDER BY ma.performed_at DESC
			LIMIT $%d OFFSET $%d
		`, whereClause, argIdx, argIdx+1)

		args = append(args, filter.Limit, filter.Offset)

		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed querying maintenance activities: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var act MaintenanceActivity
			if err := rows.Scan(
				&act.ID, &act.OrganizationID, &act.ClientID, &act.AssetID, &act.Title, &act.Description,
				&act.Category, &act.PerformedAt, &act.DurationMinutes, &act.CreatedBy, &act.CreatedAt, &act.UpdatedAt,
				&act.ClientName, &act.AssetName, &act.CreatedByName, &act.CreatedByEmail,
			); err != nil {
				return fmt.Errorf("failed scanning maintenance activity: %w", err)
			}
			activities = append(activities, &act)
		}
		return rows.Err()
	})

	if err != nil {
		return nil, 0, err
	}
	return activities, totalCount, nil
}

// UpdateMaintenanceActivity modifies an existing activity. If included in a generated report, creates an audit log entry.
func (r *MaintenanceRepository) UpdateMaintenanceActivity(ctx context.Context, orgID, activityID string, params UpdateMaintenanceParams) (*MaintenanceActivity, error) {
	var act MaintenanceActivity
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		// 1. Fetch current activity
		var current MaintenanceActivity
		err := tx.QueryRow(ctx, `
			SELECT id, organization_id, client_id, asset_id, title, description,
			       category, performed_at, duration_minutes, created_by, created_at, updated_at
			FROM maintenance_activities
			WHERE id = $1 AND organization_id = $2
			FOR UPDATE
		`, activityID, orgID).Scan(
			&current.ID, &current.OrganizationID, &current.ClientID, &current.AssetID, &current.Title, &current.Description,
			&current.Category, &current.PerformedAt, &current.DurationMinutes, &current.CreatedBy, &current.CreatedAt, &current.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrMaintenanceNotFound
			}
			return err
		}

		// Apply updates
		if params.Title != nil {
			t := strings.TrimSpace(*params.Title)
			if t == "" || len(t) > 200 {
				return ErrInvalidTitle
			}
			current.Title = t
		}

		if params.Description != nil {
			d := strings.TrimSpace(*params.Description)
			if d == "" {
				return ErrInvalidDescription
			}
			current.Description = d
		}

		if params.Category != nil {
			c := strings.ToLower(strings.TrimSpace(*params.Category))
			if !ValidCategories[c] {
				return ErrInvalidCategory
			}
			current.Category = c
		}

		if params.DurationMinutes != nil {
			if *params.DurationMinutes < 0 || *params.DurationMinutes > 14400 {
				return ErrInvalidDuration
			}
			current.DurationMinutes = params.DurationMinutes
		}

		if params.PerformedAt != nil {
			current.PerformedAt = *params.PerformedAt
		}

		if params.AssetID != nil {
			if *params.AssetID != "" {
				var assetValid bool
				err := tx.QueryRow(ctx, `
					SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1 AND client_id = $2 AND organization_id = $3 AND archived_at IS NULL)
				`, *params.AssetID, current.ClientID, orgID).Scan(&assetValid)
				if err != nil {
					return err
				}
				if !assetValid {
					return errors.New("asset does not belong to the selected client and organization")
				}
				current.AssetID = params.AssetID
			} else {
				current.AssetID = nil
			}
		}

		// 2. Audit check: Was this activity within the period of any generated report for this client?
		var includedInReport bool
		_ = tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM reports
				WHERE organization_id = $1 AND client_id = $2
				  AND period_start <= $3 AND period_end >= $3
			)
		`, orgID, current.ClientID, current.PerformedAt).Scan(&includedInReport)

		// 3. Update row
		err = tx.QueryRow(ctx, `
			UPDATE maintenance_activities
			SET asset_id = $1, title = $2, description = $3, category = $4,
			    performed_at = $5, duration_minutes = $6, updated_at = NOW()
			WHERE id = $7 AND organization_id = $8
			RETURNING id, organization_id, client_id, asset_id, title, description,
			          category, performed_at, duration_minutes, created_by, created_at, updated_at
		`, current.AssetID, current.Title, current.Description, current.Category,
			current.PerformedAt, current.DurationMinutes, activityID, orgID,
		).Scan(
			&act.ID, &act.OrganizationID, &act.ClientID, &act.AssetID, &act.Title, &act.Description,
			&act.Category, &act.PerformedAt, &act.DurationMinutes, &act.CreatedBy, &act.CreatedAt, &act.UpdatedAt,
		)
		if err != nil {
			return err
		}

		// 4. If included in a report, log audit
		if includedInReport && r.auditRepo != nil {
			actorID := params.UpdatedBy
			meta := map[string]interface{}{
				"activity_id":        activityID,
				"client_id":          current.ClientID,
				"performed_at":       current.PerformedAt.Format(time.RFC3339),
				"title":              current.Title,
				"included_in_report": true,
			}
			_ = r.auditRepo.RecordAuditLogTx(ctx, tx, orgID, &actorID, "maintenance_activity.updated_after_report", "maintenance_activity", activityID, meta, "")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return &act, nil
}

// DeleteMaintenanceActivity removes an activity. If included in a report, logs an audit entry.
func (r *MaintenanceRepository) DeleteMaintenanceActivity(ctx context.Context, orgID, activityID, deletedBy string) error {
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		var clientID string
		var performedAt time.Time
		var title string

		err := tx.QueryRow(ctx, `
			SELECT client_id, performed_at, title
			FROM maintenance_activities
			WHERE id = $1 AND organization_id = $2
		`, activityID, orgID).Scan(&clientID, &performedAt, &title)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrMaintenanceNotFound
			}
			return err
		}

		var includedInReport bool
		_ = tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM reports
				WHERE organization_id = $1 AND client_id = $2
				  AND period_start <= $3 AND period_end >= $3
			)
		`, orgID, clientID, performedAt).Scan(&includedInReport)

		tag, err := tx.Exec(ctx, `
			DELETE FROM maintenance_activities
			WHERE id = $1 AND organization_id = $2
		`, activityID, orgID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrMaintenanceNotFound
		}

		if includedInReport && r.auditRepo != nil {
			actorID := deletedBy
			meta := map[string]interface{}{
				"activity_id":        activityID,
				"client_id":          clientID,
				"performed_at":       performedAt.Format(time.RFC3339),
				"title":              title,
				"included_in_report": true,
			}
			_ = r.auditRepo.RecordAuditLogTx(ctx, tx, orgID, &actorID, "maintenance_activity.deleted_after_report", "maintenance_activity", activityID, meta, "")
		}

		return nil
	})
}
