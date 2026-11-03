package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrHeartbeatNotFound = errors.New("heartbeat monitor not found")
	ErrHeartbeatRevoked  = errors.New("heartbeat monitor is revoked")
	ErrHeartbeatArchived = errors.New("heartbeat monitor is archived")
	ErrHeartbeatPaused   = errors.New("heartbeat monitor is paused")
)

type CreateHeartbeatParams struct {
	ClientID                string
	AssetID                 *string
	Name                    string
	Description             *string
	ExpectedIntervalSeconds int
	GracePeriodSeconds      int
}

type UpdateHeartbeatParams struct {
	Name                    string
	Description             *string
	AssetID                 *string
	ExpectedIntervalSeconds int
	GracePeriodSeconds      int
}

type HeartbeatFilter struct {
	ClientID string
	AssetID  string
	Status   string
	Search   string
	Limit    int
	Offset   int
}

type HeartbeatRepository struct {
	pool *pgxpool.Pool
}

func NewHeartbeatRepository(pool *pgxpool.Pool) *HeartbeatRepository {
	return &HeartbeatRepository{pool: pool}
}

// GenerateSecureToken creates a cryptographically secure 32-byte token and its SHA-256 hex digest.
func GenerateSecureToken() (plaintextToken string, tokenHash string, err error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("failed to generate secure random bytes: %w", err)
	}
	plaintextToken = hex.EncodeToString(bytes)
	hash := sha256.Sum256([]byte(plaintextToken))
	tokenHash = hex.EncodeToString(hash[:])
	return plaintextToken, tokenHash, nil
}

// HashToken computes the SHA-256 hex digest of a plaintext token.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// CreateHeartbeat creates a new heartbeat monitor and returns the generated plaintext token once.
func (r *HeartbeatRepository) CreateHeartbeat(ctx context.Context, orgID string, params CreateHeartbeatParams) (*HeartbeatMonitor, string, error) {
	if strings.TrimSpace(params.Name) == "" {
		return nil, "", errors.New("heartbeat name cannot be empty")
	}
	if params.ExpectedIntervalSeconds < 60 || params.ExpectedIntervalSeconds > 2592000 {
		return nil, "", errors.New("expected interval must be between 60 and 2,592,000 seconds (30 days)")
	}

	// PRD grace period: default 20% of interval with minimum 60 seconds
	grace := params.GracePeriodSeconds
	if grace <= 0 {
		grace = (params.ExpectedIntervalSeconds * 20) / 100
		if grace < 60 {
			grace = 60
		}
	}
	if grace < 60 || grace > 86400 {
		return nil, "", errors.New("grace period must be between 60 and 86,400 seconds (24 hours)")
	}

	plaintextToken, tokenHash, err := GenerateSecureToken()
	if err != nil {
		return nil, "", err
	}

	var hb HeartbeatMonitor
	err = WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		// Validate client belongs to org
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

		// If asset_id is provided, validate it belongs to client and org
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

		initialDeadline := time.Now().UTC().Add(time.Duration(params.ExpectedIntervalSeconds+grace) * time.Second)

		query := `
			INSERT INTO heartbeat_monitors (
				organization_id, client_id, asset_id, name, description,
				expected_interval_seconds, grace_period_seconds, status,
				is_active, token_hash, next_expected_at, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending', true, $8, $9, NOW(), NOW())
			RETURNING id, organization_id, client_id, asset_id, name, description,
			          expected_interval_seconds, grace_period_seconds, status,
			          is_active, token_hash, last_received_at, next_expected_at,
			          created_at, updated_at, archived_at
		`

		return tx.QueryRow(ctx, query,
			orgID, params.ClientID, params.AssetID, params.Name, params.Description,
			params.ExpectedIntervalSeconds, grace, tokenHash, initialDeadline,
		).Scan(
			&hb.ID, &hb.OrganizationID, &hb.ClientID, &hb.AssetID, &hb.Name, &hb.Description,
			&hb.ExpectedIntervalSeconds, &hb.GracePeriodSeconds, &hb.Status,
			&hb.IsActive, &hb.TokenHash, &hb.LastReceivedAt, &hb.NextExpectedAt,
			&hb.CreatedAt, &hb.UpdatedAt, &hb.ArchivedAt,
		)
	})

	if err != nil {
		return nil, "", err
	}
	return &hb, plaintextToken, nil
}

// RotateToken generates a new cryptographically secure token, invalidating the previous one immediately.
func (r *HeartbeatRepository) RotateToken(ctx context.Context, orgID, heartbeatID string) (string, error) {
	plaintextToken, tokenHash, err := GenerateSecureToken()
	if err != nil {
		return "", err
	}

	err = WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		var status string
		var isArchived bool
		err := tx.QueryRow(ctx, `
			SELECT status, (archived_at IS NOT NULL)
			FROM heartbeat_monitors
			WHERE id = $1 AND organization_id = $2
			FOR UPDATE
		`, heartbeatID, orgID).Scan(&status, &isArchived)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrHeartbeatNotFound
			}
			return err
		}

		if isArchived {
			return ErrHeartbeatArchived
		}

		_, err = tx.Exec(ctx, `
			UPDATE heartbeat_monitors
			SET token_hash = $1,
			    updated_at = NOW()
			WHERE id = $2 AND organization_id = $3
		`, tokenHash, heartbeatID, orgID)
		return err
	})

	if err != nil {
		return "", err
	}
	return plaintextToken, nil
}

// RevokeToken immediately revokes the heartbeat token and marks it revoked.
func (r *HeartbeatRepository) RevokeToken(ctx context.Context, orgID, heartbeatID string) error {
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		res, err := tx.Exec(ctx, `
			UPDATE heartbeat_monitors
			SET status = 'revoked',
			    is_active = false,
			    updated_at = NOW()
			WHERE id = $1 AND organization_id = $2 AND archived_at IS NULL
		`, heartbeatID, orgID)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return ErrHeartbeatNotFound
		}
		return nil
	})
}

// PauseHeartbeat pauses the heartbeat monitor.
func (r *HeartbeatRepository) PauseHeartbeat(ctx context.Context, orgID, heartbeatID string) error {
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		res, err := tx.Exec(ctx, `
			UPDATE heartbeat_monitors
			SET status = 'paused',
			    updated_at = NOW()
			WHERE id = $1 AND organization_id = $2 AND archived_at IS NULL AND status != 'revoked'
		`, heartbeatID, orgID)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return ErrHeartbeatNotFound
		}
		return nil
	})
}

// ResumeHeartbeat resumes a paused heartbeat monitor.
func (r *HeartbeatRepository) ResumeHeartbeat(ctx context.Context, orgID, heartbeatID string) error {
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		var lastReceived *time.Time
		var interval, grace int
		err := tx.QueryRow(ctx, `
			SELECT last_received_at, expected_interval_seconds, grace_period_seconds
			FROM heartbeat_monitors
			WHERE id = $1 AND organization_id = $2 AND archived_at IS NULL AND status = 'paused'
			FOR UPDATE
		`, heartbeatID, orgID).Scan(&lastReceived, &interval, &grace)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrHeartbeatNotFound
			}
			return err
		}

		now := time.Now().UTC()
		newStatus := "pending"
		var nextExpected *time.Time

		if lastReceived != nil {
			deadline := lastReceived.Add(time.Duration(interval+grace) * time.Second)
			if deadline.After(now) {
				newStatus = "healthy"
				nextExpected = &deadline
			}
		}

		_, err = tx.Exec(ctx, `
			UPDATE heartbeat_monitors
			SET status = $1,
			    next_expected_at = $2,
			    updated_at = NOW()
			WHERE id = $3 AND organization_id = $4
		`, newStatus, nextExpected, heartbeatID, orgID)
		return err
	})
}

// ArchiveHeartbeat soft-deletes the heartbeat monitor and resolves any associated open incidents
// with resolution_type 'source_monitor_deleted' while preserving historical records and audit logs.
func (r *HeartbeatRepository) ArchiveHeartbeat(ctx context.Context, orgID, heartbeatID string) error {
	return r.ArchiveHeartbeatWithActor(ctx, orgID, heartbeatID, "")
}

// ArchiveHeartbeatWithActor soft-deletes the heartbeat monitor and resolves any associated open incidents
// attributing the action to the acting user.
func (r *HeartbeatRepository) ArchiveHeartbeatWithActor(ctx context.Context, orgID, heartbeatID, userID string) error {
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		var hbName string
		var clientID string
		err := tx.QueryRow(ctx, `
			SELECT name, client_id
			FROM heartbeat_monitors
			WHERE id = $1 AND organization_id = $2 AND archived_at IS NULL
			FOR UPDATE
		`, heartbeatID, orgID).Scan(&hbName, &clientID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrHeartbeatNotFound
			}
			return err
		}

		now := time.Now().UTC()

		res, err := tx.Exec(ctx, `
			UPDATE heartbeat_monitors
			SET archived_at = $1,
			    is_active = false,
			    updated_at = NOW()
			WHERE id = $2 AND organization_id = $3 AND archived_at IS NULL
		`, now, heartbeatID, orgID)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return ErrHeartbeatNotFound
		}

		// Identify all open incidents belonging to that exact Heartbeat Monitor
		rows, err := tx.Query(ctx, `
			SELECT id, confirmed_at
			FROM incidents
			WHERE heartbeat_monitor_id = $1 AND organization_id = $2 AND status = 'open'
			FOR UPDATE
		`, heartbeatID, orgID)
		if err != nil {
			return fmt.Errorf("failed to find open incidents for heartbeat: %w", err)
		}
		defer rows.Close()

		type openInc struct {
			id          string
			confirmedAt time.Time
		}
		var openIncs []openInc
		for rows.Next() {
			var oi openInc
			if err := rows.Scan(&oi.id, &oi.confirmedAt); err != nil {
				return err
			}
			openIncs = append(openIncs, oi)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		var actorUserPtr *string
		if userID != "" {
			actorUserPtr = &userID
		}

		var resolvedIncIDs []string
		for _, oi := range openIncs {
			durationSec := int(now.Sub(oi.confirmedAt).Seconds())
			if durationSec < 0 {
				durationSec = 0
			}

			resolutionNote := "Closed because the associated heartbeat monitor was deleted."
			_, err = tx.Exec(ctx, `
				UPDATE incidents
				SET status = 'resolved',
				    resolved_at = $1,
				    resolved_by = $2,
				    resolution_type = 'source_monitor_deleted',
				    resolution_note = $3,
				    duration_seconds = $4,
				    updated_at = NOW()
				WHERE id = $5 AND organization_id = $6
			`, now, actorUserPtr, resolutionNote, durationSec, oi.id, orgID)
			if err != nil {
				return fmt.Errorf("failed to resolve incident %s on heartbeat deletion: %w", oi.id, err)
			}
			resolvedIncIDs = append(resolvedIncIDs, oi.id)

			// Record notification event for each resolved incident
			idempKey := fmt.Sprintf("inc_res_del:%s:%d", oi.id, now.Unix())
			resPayload := map[string]interface{}{
				"incident_id":     oi.id,
				"heartbeat_id":    heartbeatID,
				"heartbeat_name":  hbName,
				"client_id":       clientID,
				"resolution_type": "source_monitor_deleted",
				"resolved_at":     now.Format(time.RFC3339),
				"duration_sec":    durationSec,
				"reason":          "source_monitor_deleted",
			}
			if actorUserPtr != nil {
				resPayload["resolved_by"] = *actorUserPtr
			}
			_, _ = tx.Exec(ctx, `
				INSERT INTO notification_events (
					organization_id, event_type, incident_id, heartbeat_monitor_id,
					client_id, idempotency_key, payload, status, created_at
				)
				VALUES ($1, 'incident.resolved', $2, $3, $4, $5, $6, 'pending', NOW())
				ON CONFLICT (idempotency_key) DO NOTHING
			`, orgID, oi.id, heartbeatID, clientID, idempKey, resPayload)

			// Record audit log entry for this incident resolution
			auditMeta := map[string]interface{}{
				"incident_id":      oi.id,
				"heartbeat_id":     heartbeatID,
				"heartbeat_name":   hbName,
				"resolution_type":  "source_monitor_deleted",
				"resolution_note":  resolutionNote,
				"resolved_at":      now.Format(time.RFC3339),
				"duration_seconds": durationSec,
			}
			if actorUserPtr != nil {
				auditMeta["resolved_by"] = *actorUserPtr
			}
			metaJSON, _ := json.Marshal(auditMeta)
			_, _ = tx.Exec(ctx, `
				INSERT INTO audit_logs (
					organization_id, actor_user_id, action, entity_type, entity_id, metadata_json, ip_hash, created_at
				)
				VALUES ($1, $2, 'incident.resolved', 'incident', $3, $4, '', NOW())
			`, orgID, actorUserPtr, oi.id, metaJSON)
		}

		// Record audit log for heartbeat monitor deletion
		hbAuditMeta := map[string]interface{}{
			"heartbeat_id":          heartbeatID,
			"heartbeat_name":        hbName,
			"client_id":             clientID,
			"archived_at":           now.Format(time.RFC3339),
			"resolved_incident_ids": resolvedIncIDs,
		}
		if actorUserPtr != nil {
			hbAuditMeta["deleted_by"] = *actorUserPtr
		}
		metaJSON, _ := json.Marshal(hbAuditMeta)
		_, _ = tx.Exec(ctx, `
			INSERT INTO audit_logs (
				organization_id, actor_user_id, action, entity_type, entity_id, metadata_json, ip_hash, created_at
			)
			VALUES ($1, $2, 'heartbeat.deleted', 'heartbeat_monitor', $3, $4, '', NOW())
		`, orgID, actorUserPtr, heartbeatID, metaJSON)

		return nil
	})
}

// UpdateHeartbeat modifies configurable parameters for a heartbeat monitor.
func (r *HeartbeatRepository) UpdateHeartbeat(ctx context.Context, orgID, heartbeatID string, params UpdateHeartbeatParams) (*HeartbeatMonitor, error) {
	if strings.TrimSpace(params.Name) == "" {
		return nil, errors.New("heartbeat name cannot be empty")
	}
	if params.ExpectedIntervalSeconds < 60 || params.ExpectedIntervalSeconds > 2592000 {
		return nil, errors.New("expected interval must be between 60 and 2,592,000 seconds (30 days)")
	}

	grace := params.GracePeriodSeconds
	if grace <= 0 {
		grace = (params.ExpectedIntervalSeconds * 20) / 100
		if grace < 60 {
			grace = 60
		}
	}
	if grace < 60 || grace > 86400 {
		return nil, errors.New("grace period must be between 60 and 86,400 seconds (24 hours)")
	}

	var hb HeartbeatMonitor
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		var clientID string
		err := tx.QueryRow(ctx, `
			SELECT client_id FROM heartbeat_monitors WHERE id = $1 AND organization_id = $2 AND archived_at IS NULL
		`, heartbeatID, orgID).Scan(&clientID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrHeartbeatNotFound
			}
			return err
		}

		if params.AssetID != nil && *params.AssetID != "" {
			var assetValid bool
			err := tx.QueryRow(ctx, `
				SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1 AND client_id = $2 AND organization_id = $3 AND archived_at IS NULL)
			`, *params.AssetID, clientID, orgID).Scan(&assetValid)
			if err != nil {
				return err
			}
			if !assetValid {
				return errors.New("asset does not belong to client and organization")
			}
		} else {
			params.AssetID = nil
		}

		query := `
			UPDATE heartbeat_monitors
			SET name = $1,
			    description = $2,
			    asset_id = $3,
			    expected_interval_seconds = $4,
			    grace_period_seconds = $5,
			    updated_at = NOW()
			WHERE id = $6 AND organization_id = $7
			RETURNING id, organization_id, client_id, asset_id, name, description,
			          expected_interval_seconds, grace_period_seconds, status,
			          is_active, token_hash, last_received_at, next_expected_at,
			          created_at, updated_at, archived_at
		`
		return tx.QueryRow(ctx, query,
			params.Name, params.Description, params.AssetID,
			params.ExpectedIntervalSeconds, grace, heartbeatID, orgID,
		).Scan(
			&hb.ID, &hb.OrganizationID, &hb.ClientID, &hb.AssetID, &hb.Name, &hb.Description,
			&hb.ExpectedIntervalSeconds, &hb.GracePeriodSeconds, &hb.Status,
			&hb.IsActive, &hb.TokenHash, &hb.LastReceivedAt, &hb.NextExpectedAt,
			&hb.CreatedAt, &hb.UpdatedAt, &hb.ArchivedAt,
		)
	})

	if err != nil {
		return nil, err
	}
	return &hb, nil
}

// GetHeartbeatByID retrieves a heartbeat monitor with joined client and asset metadata.
func (r *HeartbeatRepository) GetHeartbeatByID(ctx context.Context, orgID, heartbeatID string) (*HeartbeatMonitor, error) {
	var hb HeartbeatMonitor
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			SELECT hm.id, hm.organization_id, hm.client_id, hm.asset_id, hm.name, hm.description,
			       hm.expected_interval_seconds, hm.grace_period_seconds, hm.status,
			       hm.is_active, hm.token_hash, hm.last_received_at, hm.next_expected_at,
			       hm.created_at, hm.updated_at, hm.archived_at,
			       c.name AS client_name, a.name AS asset_name
			FROM heartbeat_monitors hm
			JOIN clients c ON hm.client_id = c.id
			LEFT JOIN assets a ON hm.asset_id = a.id
			WHERE hm.id = $1 AND hm.organization_id = $2
		`
		err := tx.QueryRow(ctx, query, heartbeatID, orgID).Scan(
			&hb.ID, &hb.OrganizationID, &hb.ClientID, &hb.AssetID, &hb.Name, &hb.Description,
			&hb.ExpectedIntervalSeconds, &hb.GracePeriodSeconds, &hb.Status,
			&hb.IsActive, &hb.TokenHash, &hb.LastReceivedAt, &hb.NextExpectedAt,
			&hb.CreatedAt, &hb.UpdatedAt, &hb.ArchivedAt,
			&hb.ClientName, &hb.AssetName,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrHeartbeatNotFound
			}
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &hb, nil
}

// ListHeartbeats retrieves paginated heartbeat monitors with joined metadata.
func (r *HeartbeatRepository) ListHeartbeats(ctx context.Context, orgID string, filter HeartbeatFilter) ([]*HeartbeatMonitor, int, error) {
	var list []*HeartbeatMonitor
	var totalCount int

	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		var whereClauses []string
		var args []interface{}
		argIdx := 1

		whereClauses = append(whereClauses, fmt.Sprintf("hm.organization_id = $%d AND hm.archived_at IS NULL", argIdx))
		args = append(args, orgID)
		argIdx++

		if filter.ClientID != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("hm.client_id = $%d", argIdx))
			args = append(args, filter.ClientID)
			argIdx++
		}

		if filter.AssetID != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("hm.asset_id = $%d", argIdx))
			args = append(args, filter.AssetID)
			argIdx++
		}

		if filter.Status != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("hm.status = $%d", argIdx))
			args = append(args, filter.Status)
			argIdx++
		}

		if filter.Search != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("(hm.name ILIKE $%d OR c.name ILIKE $%d)", argIdx, argIdx))
			args = append(args, "%"+filter.Search+"%")
			argIdx++
		}

		whereSQL := strings.Join(whereClauses, " AND ")

		countQuery := fmt.Sprintf(`
			SELECT COUNT(*)
			FROM heartbeat_monitors hm
			JOIN clients c ON hm.client_id = c.id
			LEFT JOIN assets a ON hm.asset_id = a.id
			WHERE %s
		`, whereSQL)

		if err := tx.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
			return fmt.Errorf("failed to count heartbeats: %w", err)
		}

		limit := filter.Limit
		if limit <= 0 || limit > 100 {
			limit = 50
		}
		offset := filter.Offset
		if offset < 0 {
			offset = 0
		}

		listQuery := fmt.Sprintf(`
			SELECT hm.id, hm.organization_id, hm.client_id, hm.asset_id, hm.name, hm.description,
			       hm.expected_interval_seconds, hm.grace_period_seconds, hm.status,
			       hm.is_active, hm.token_hash, hm.last_received_at, hm.next_expected_at,
			       hm.created_at, hm.updated_at, hm.archived_at,
			       c.name AS client_name, a.name AS asset_name
			FROM heartbeat_monitors hm
			JOIN clients c ON hm.client_id = c.id
			LEFT JOIN assets a ON hm.asset_id = a.id
			WHERE %s
			ORDER BY hm.created_at DESC
			LIMIT $%d OFFSET $%d
		`, whereSQL, argIdx, argIdx+1)

		args = append(args, limit, offset)

		rows, err := tx.Query(ctx, listQuery, args...)
		if err != nil {
			return fmt.Errorf("failed to query heartbeats: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var hb HeartbeatMonitor
			if err := rows.Scan(
				&hb.ID, &hb.OrganizationID, &hb.ClientID, &hb.AssetID, &hb.Name, &hb.Description,
				&hb.ExpectedIntervalSeconds, &hb.GracePeriodSeconds, &hb.Status,
				&hb.IsActive, &hb.TokenHash, &hb.LastReceivedAt, &hb.NextExpectedAt,
				&hb.CreatedAt, &hb.UpdatedAt, &hb.ArchivedAt,
				&hb.ClientName, &hb.AssetName,
			); err != nil {
				return fmt.Errorf("failed to scan heartbeat row: %w", err)
			}
			list = append(list, &hb)
		}
		return rows.Err()
	})

	if err != nil {
		return nil, 0, err
	}
	return list, totalCount, nil
}

// RecordHeartbeatPing processes an unauthenticated public ping via SHA-256 token hash lookup.
// If valid and active, it records last_received_at, schedules next_expected_at, marks healthy,
// and atomically resolves any open incident for this heartbeat.
func (r *HeartbeatRepository) RecordHeartbeatPing(ctx context.Context, plaintextToken string) (*HeartbeatMonitor, *Incident, error) {
	if len(plaintextToken) < 16 {
		return nil, nil, ErrHeartbeatNotFound
	}

	incomingHash := HashToken(plaintextToken)

	// Direct lookup by hash to identify target without trusting user tenant parameters
	var hb HeartbeatMonitor
	var storedHash string

	// Use the SECURITY DEFINER function to lookup the heartbeat by its token hash across tenants
	// without requiring the caller to supply any tenant identifiers or credentials.
	err := r.pool.QueryRow(ctx, `
		SELECT id, organization_id, token_hash, status, archived_at
		FROM lookup_heartbeat_by_token_hash($1)
	`, incomingHash).Scan(&hb.ID, &hb.OrganizationID, &storedHash, &hb.Status, &hb.ArchivedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrHeartbeatNotFound
		}
		return nil, nil, ErrHeartbeatNotFound
	}

	// Constant-time check on token hash
	if subtle.ConstantTimeCompare([]byte(storedHash), []byte(incomingHash)) != 1 {
		return nil, nil, ErrHeartbeatNotFound
	}

	if hb.ArchivedAt != nil {
		return nil, nil, ErrHeartbeatArchived
	}
	if hb.Status == "revoked" {
		return nil, nil, ErrHeartbeatRevoked
	}

	var resolvedIncident *Incident
	now := time.Now().UTC()

	// Perform all updates inside the identified tenant's transaction
	err = WithTenantTx(ctx, r.pool, hb.OrganizationID, func(tx pgx.Tx) error {
		var interval, grace int
		err := tx.QueryRow(ctx, `
			SELECT expected_interval_seconds, grace_period_seconds
			FROM heartbeat_monitors
			WHERE id = $1 AND organization_id = $2
			FOR UPDATE
		`, hb.ID, hb.OrganizationID).Scan(&interval, &grace)
		if err != nil {
			return err
		}

		nextExpected := now.Add(time.Duration(interval+grace) * time.Second)

		query := `
			UPDATE heartbeat_monitors
			SET status = 'healthy',
			    last_received_at = $1,
			    next_expected_at = $2,
			    lease_acquired_at = NULL,
			    lease_worker_id = NULL,
			    updated_at = NOW()
			WHERE id = $3 AND organization_id = $4
			RETURNING id, organization_id, client_id, asset_id, name, description,
			          expected_interval_seconds, grace_period_seconds, status,
			          is_active, token_hash, last_received_at, next_expected_at,
			          created_at, updated_at, archived_at
		`
		err = tx.QueryRow(ctx, query, now, nextExpected, hb.ID, hb.OrganizationID).Scan(
			&hb.ID, &hb.OrganizationID, &hb.ClientID, &hb.AssetID, &hb.Name, &hb.Description,
			&hb.ExpectedIntervalSeconds, &hb.GracePeriodSeconds, &hb.Status,
			&hb.IsActive, &hb.TokenHash, &hb.LastReceivedAt, &hb.NextExpectedAt,
			&hb.CreatedAt, &hb.UpdatedAt, &hb.ArchivedAt,
		)
		if err != nil {
			return err
		}

		// Check if an open incident exists for this heartbeat
		var openIncID string
		var confirmedAt time.Time
		err = tx.QueryRow(ctx, `
			SELECT id, confirmed_at
			FROM incidents
			WHERE heartbeat_monitor_id = $1 AND organization_id = $2 AND status = 'open'
			FOR UPDATE
		`, hb.ID, hb.OrganizationID).Scan(&openIncID, &confirmedAt)

		if err == nil {
			// Resolve open incident
			durationSec := int(now.Sub(confirmedAt).Seconds())
			if durationSec < 0 {
				durationSec = 0
			}

			var inc Incident
			err = tx.QueryRow(ctx, `
				UPDATE incidents
				SET status = 'resolved',
				    resolved_at = $1,
				    resolution_type = 'automatic_recovery',
				    duration_seconds = $2,
				    updated_at = NOW()
				WHERE id = $3 AND organization_id = $4
				RETURNING id, organization_id, monitor_id, heartbeat_monitor_id, first_failure_at,
				          confirmed_at, resolved_at, status, failure_category, first_failure_result_id,
				          recovery_result_id, acknowledged_at, acknowledged_by, resolved_by,
				          resolution_type, resolution_note, notification_state,
				          duration_seconds, summary, created_at, updated_at
			`, now, durationSec, openIncID, hb.OrganizationID).Scan(
				&inc.ID, &inc.OrganizationID, &inc.MonitorID, &inc.HeartbeatMonitorID,
				&inc.FirstFailureAt, &inc.ConfirmedAt, &inc.ResolvedAt, &inc.Status,
				&inc.FailureCategory, &inc.FirstFailureResultID, &inc.RecoveryResultID,
				&inc.AcknowledgedAt, &inc.AcknowledgedBy, &inc.ResolvedBy,
				&inc.ResolutionType, &inc.ResolutionNote, &inc.NotificationState,
				&inc.DurationSeconds, &inc.Summary, &inc.CreatedAt, &inc.UpdatedAt,
			)
			if err != nil {
				return fmt.Errorf("failed to resolve heartbeat incident: %w", err)
			}
			resolvedIncident = &inc

			// Record audit log for automatic recovery
			auditMeta := map[string]interface{}{
				"incident_id":      inc.ID,
				"heartbeat_id":     hb.ID,
				"heartbeat_name":   hb.Name,
				"resolution_type":  "automatic_recovery",
				"resolved_at":      now.Format(time.RFC3339),
				"duration_seconds": durationSec,
			}
			metaJSON, _ := json.Marshal(auditMeta)
			_, _ = tx.Exec(ctx, `
				INSERT INTO audit_logs (
					organization_id, actor_user_id, action, entity_type, entity_id, metadata_json, ip_hash, created_at
				)
				VALUES ($1, NULL, 'incident.resolved', 'incident', $2, $3, '', NOW())
			`, hb.OrganizationID, inc.ID, metaJSON)

			// Generate persistent notification event: heartbeat.recovered
			// Enrich payload with rich context via JOIN (immutable snapshot at event time).
			idempKey := fmt.Sprintf("hb_rec:%s:%d", hb.ID, now.Unix())
			recPayload := map[string]interface{}{
				"heartbeat_id":   hb.ID,
				"heartbeat_name": hb.Name,
				"client_id":      hb.ClientID,
				"incident_id":    inc.ID,
				"recovered_at":   now.Format(time.RFC3339),
				"duration_sec":   durationSec,
			}
			enrichHeartbeatPayload(ctx, tx, hb.ID, hb.OrganizationID, recPayload)
			_, err = tx.Exec(ctx, `
				INSERT INTO notification_events (
					organization_id, event_type, incident_id, heartbeat_monitor_id,
					client_id, idempotency_key, payload, status, created_at
				)
				VALUES ($1, 'heartbeat.recovered', $2, $3, $4, $5, $6, 'pending', NOW())
				ON CONFLICT (idempotency_key) DO NOTHING
			`, hb.OrganizationID, inc.ID, hb.ID, hb.ClientID, idempKey, recPayload)
			if err != nil {
				return fmt.Errorf("failed to create heartbeat recovery event: %w", err)
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}
	return &hb, resolvedIncident, nil
}

// ClaimMissedHeartbeats calls the atomic SECURITY DEFINER stored procedure.
func (r *HeartbeatRepository) ClaimMissedHeartbeats(ctx context.Context, workerID string, batchSize, leaseTimeoutSeconds int) ([]*ClaimedHeartbeatJob, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, organization_id, client_id, asset_id, name,
		       expected_interval_seconds, grace_period_seconds, status,
		       last_received_at, next_expected_at
		FROM claim_missed_heartbeats($1, $2, $3)
	`, workerID, batchSize, leaseTimeoutSeconds)
	if err != nil {
		return nil, fmt.Errorf("failed to claim missed heartbeats: %w", err)
	}
	defer rows.Close()

	var jobs []*ClaimedHeartbeatJob
	for rows.Next() {
		var j ClaimedHeartbeatJob
		if err := rows.Scan(
			&j.ID, &j.OrganizationID, &j.ClientID, &j.AssetID, &j.Name,
			&j.ExpectedIntervalSeconds, &j.GracePeriodSeconds, &j.Status,
			&j.LastReceivedAt, &j.NextExpectedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan claimed heartbeat job: %w", err)
		}
		jobs = append(jobs, &j)
	}
	return jobs, rows.Err()
}

// EnrichHeartbeatPayload adds heartbeat_name, client_name, org_name, org_timezone,
// locale, expected_interval_seconds, grace_period_seconds, and asset_name (if linked)
// to the payload map using a single query within an existing transaction.
func EnrichHeartbeatPayload(ctx context.Context, tx pgx.Tx, heartbeatID, orgID string, payload map[string]interface{}) {
	var hbName, clientID, clientName, orgName, orgTimezone, orgLocale string
	var intervalSec, graceSec int
	var assetName *string
	err := tx.QueryRow(ctx, `
		SELECT hm.name, hm.expected_interval_seconds, hm.grace_period_seconds,
		       c.id, c.name, COALESCE(a.name, NULL), o.name, o.timezone, o.locale
		FROM heartbeat_monitors hm
		JOIN clients c ON c.id = hm.client_id AND c.organization_id = hm.organization_id
		LEFT JOIN assets a ON a.id = hm.asset_id AND a.organization_id = hm.organization_id
		JOIN organizations o ON o.id = hm.organization_id
		WHERE hm.id = $1 AND hm.organization_id = $2
	`, heartbeatID, orgID).Scan(&hbName, &intervalSec, &graceSec, &clientID, &clientName, &assetName, &orgName, &orgTimezone, &orgLocale)
	if err != nil {
		return
	}
	payload["heartbeat_name"] = hbName
	payload["expected_interval_seconds"] = intervalSec
	payload["grace_period_seconds"] = graceSec
	payload["client_id"] = clientID
	payload["client_name"] = clientName
	payload["org_name"] = orgName
	payload["organization_name"] = orgName
	payload["org_timezone"] = orgTimezone
	payload["timezone"] = orgTimezone
	payload["locale"] = orgLocale
	if assetName != nil {
		payload["asset_name"] = *assetName
	}
}

func enrichHeartbeatPayload(ctx context.Context, tx pgx.Tx, heartbeatID, orgID string, payload map[string]interface{}) {
	EnrichHeartbeatPayload(ctx, tx, heartbeatID, orgID, payload)
}
