package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepository struct {
	pool *pgxpool.Pool
}

func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

// RecordEvent records an audit log entry.
func (r *AuditRepository) RecordEvent(ctx context.Context, orgID, actorUserID *string, action, entityType, entityID string, metadata []byte, ipHash string) error {
	query := `
		INSERT INTO audit_logs (organization_id, actor_user_id, action, entity_type, entity_id, metadata_json, ip_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	if len(metadata) == 0 {
		metadata = []byte("{}")
	}

	_, err := r.pool.Exec(ctx, query, orgID, actorUserID, action, entityType, entityID, metadata, ipHash)
	if err != nil {
		return fmt.Errorf("failed to record audit event: %w", err)
	}
	return nil
}

// ListByOrganization retrieves recent audit events for an organization.
// Protected by RLS or explicit WHERE clause.
func (r *AuditRepository) ListByOrganization(ctx context.Context, orgID string, limit int) ([]AuditLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT id, organization_id, actor_user_id, action, entity_type, entity_id, metadata_json, ip_hash, created_at
		FROM audit_logs
		WHERE organization_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var a AuditLog
		err := rows.Scan(
			&a.ID,
			&a.OrganizationID,
			&a.ActorUserID,
			&a.Action,
			&a.EntityType,
			&a.EntityID,
			&a.MetadataJSON,
			&a.IPHash,
			&a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, a)
	}
	return logs, nil
}

