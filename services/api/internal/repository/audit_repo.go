package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditRepository handles appending audit logs for sensitive operations and report changes.
type AuditRepository struct {
	pool *pgxpool.Pool
}

// NewAuditRepository creates a new AuditRepository instance.
func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

// RecordEvent records an audit entry with raw JSON bytes.
func (r *AuditRepository) RecordEvent(ctx context.Context, orgID *string, actorUserID *string, action, entityType, entityID string, metadataJSON []byte, ipAddress string) error {
	if len(metadataJSON) == 0 {
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO audit_logs (
			organization_id, actor_user_id, action, entity_type, entity_id, metadata_json, ip_hash, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`

	_, err := r.pool.Exec(ctx, query, orgID, actorUserID, action, entityType, entityID, metadataJSON, ipAddress)
	if err != nil {
		return fmt.Errorf("failed inserting audit log event: %w", err)
	}
	return nil
}

// RecordAuditLogTx appends an audit entry within an existing tenant transaction.
func (r *AuditRepository) RecordAuditLogTx(ctx context.Context, tx pgx.Tx, orgID string, actorUserID *string, action, entityType, entityID string, metadata map[string]interface{}, ipHash string) error {
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		metaJSON = []byte("{}")
	}

	query := `
		INSERT INTO audit_logs (
			organization_id, actor_user_id, action, entity_type, entity_id, metadata_json, ip_hash, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`

	_, err = tx.Exec(ctx, query, orgID, actorUserID, action, entityType, entityID, metaJSON, ipHash)
	if err != nil {
		return fmt.Errorf("failed inserting audit log: %w", err)
	}
	return nil
}

// RecordAuditLog appends an audit entry from map metadata.
func (r *AuditRepository) RecordAuditLog(ctx context.Context, orgID string, actorUserID *string, action, entityType, entityID string, metadata map[string]interface{}, ipHash string) error {
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		return r.RecordAuditLogTx(ctx, tx, orgID, actorUserID, action, entityType, entityID, metadata, ipHash)
	})
}
