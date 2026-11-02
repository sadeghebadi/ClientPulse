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
	ErrIncidentNotFound          = errors.New("incident not found")
	ErrIncidentAlreadyResolved   = errors.New("incident is already resolved")
	ErrIncidentAlreadyAcked      = errors.New("incident is already acknowledged")
	ErrInvalidIncidentTransition = errors.New("invalid incident transition")
)

type IncidentFilter struct {
	Status             string // "open", "resolved", or "" for all
	ClientID           string
	MonitorID          string
	HeartbeatMonitorID string
	FailureCategory    string
	From               *time.Time
	To                 *time.Time
	Limit              int
	Offset             int
}

type UptimeReport struct {
	UptimePercentage     *float64  `json:"uptime_percentage"`
	CoveragePercentage   float64   `json:"coverage_percentage"`
	InsufficientData     bool      `json:"insufficient_data"`
	TotalSeconds         int64     `json:"total_seconds"`
	OutageSeconds        int64     `json:"outage_seconds"`
	IncidentCount        int       `json:"incident_count"`
	TotalScheduledChecks int       `json:"total_scheduled_checks"`
	SuccessfulChecks     int       `json:"successful_checks"`
	FailedChecks         int       `json:"failed_checks"`
	CoverageStart        time.Time `json:"coverage_start"`
	CoverageEnd          time.Time `json:"coverage_end"`
	LimitationNotice     string    `json:"limitation_notice"`
}

type IncidentRepository struct {
	pool *pgxpool.Pool
}

func NewIncidentRepository(pool *pgxpool.Pool) *IncidentRepository {
	return &IncidentRepository{pool: pool}
}

// CreateMonitorIncidentTx inserts an open incident for a monitor within an active tenant transaction.
// If an open incident already exists for this monitor, it returns the existing incident without duplicating.
func (r *IncidentRepository) CreateMonitorIncidentTx(
	ctx context.Context,
	tx pgx.Tx,
	orgID, monitorID string,
	firstFailureAt, confirmedAt time.Time,
	firstResultID *string,
	failureCategory, summary string,
) (*Incident, bool, error) {
	// First check if an open incident already exists for this monitor
	var existing Incident
	err := tx.QueryRow(ctx, `
		SELECT id, organization_id, monitor_id, heartbeat_monitor_id, first_failure_at,
		       confirmed_at, resolved_at, status, failure_category, first_failure_result_id,
		       recovery_result_id, acknowledged_at, acknowledged_by, notification_state,
		       duration_seconds, summary, created_at, updated_at
		FROM incidents
		WHERE organization_id = $1 AND monitor_id = $2 AND status = 'open'
		FOR UPDATE
	`, orgID, monitorID).Scan(
		&existing.ID, &existing.OrganizationID, &existing.MonitorID, &existing.HeartbeatMonitorID,
		&existing.FirstFailureAt, &existing.ConfirmedAt, &existing.ResolvedAt, &existing.Status,
		&existing.FailureCategory, &existing.FirstFailureResultID, &existing.RecoveryResultID,
		&existing.AcknowledgedAt, &existing.AcknowledgedBy, &existing.NotificationState,
		&existing.DurationSeconds, &existing.Summary, &existing.CreatedAt, &existing.UpdatedAt,
	)
	if err == nil {
		// Existing open incident found - do not create a duplicate
		return &existing, false, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, fmt.Errorf("failed to check for existing open incident: %w", err)
	}

	// Create new incident
	var inc Incident
	err = tx.QueryRow(ctx, `
		INSERT INTO incidents (
			organization_id, monitor_id, first_failure_at, confirmed_at, status,
			failure_category, first_failure_result_id, notification_state, summary,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, 'open', $5, $6, 'pending', $7, NOW(), NOW())
		RETURNING id, organization_id, monitor_id, heartbeat_monitor_id, first_failure_at,
		          confirmed_at, resolved_at, status, failure_category, first_failure_result_id,
		          recovery_result_id, acknowledged_at, acknowledged_by, notification_state,
		          duration_seconds, summary, created_at, updated_at
	`, orgID, monitorID, firstFailureAt, confirmedAt, failureCategory, firstResultID, summary).Scan(
		&inc.ID, &inc.OrganizationID, &inc.MonitorID, &inc.HeartbeatMonitorID,
		&inc.FirstFailureAt, &inc.ConfirmedAt, &inc.ResolvedAt, &inc.Status,
		&inc.FailureCategory, &inc.FirstFailureResultID, &inc.RecoveryResultID,
		&inc.AcknowledgedAt, &inc.AcknowledgedBy, &inc.NotificationState,
		&inc.DurationSeconds, &inc.Summary, &inc.CreatedAt, &inc.UpdatedAt,
	)
	if err != nil {
		return nil, false, fmt.Errorf("failed to insert monitor incident: %w", err)
	}

	return &inc, true, nil
}

// CreateHeartbeatIncidentTx inserts an open incident for a missed heartbeat within an active tenant transaction.
func (r *IncidentRepository) CreateHeartbeatIncidentTx(
	ctx context.Context,
	tx pgx.Tx,
	orgID, heartbeatID string,
	firstFailureAt, confirmedAt time.Time,
	failureCategory, summary string,
) (*Incident, bool, error) {
	var existing Incident
	err := tx.QueryRow(ctx, `
		SELECT id, organization_id, monitor_id, heartbeat_monitor_id, first_failure_at,
		       confirmed_at, resolved_at, status, failure_category, first_failure_result_id,
		       recovery_result_id, acknowledged_at, acknowledged_by, notification_state,
		       duration_seconds, summary, created_at, updated_at
		FROM incidents
		WHERE organization_id = $1 AND heartbeat_monitor_id = $2 AND status = 'open'
		FOR UPDATE
	`, orgID, heartbeatID).Scan(
		&existing.ID, &existing.OrganizationID, &existing.MonitorID, &existing.HeartbeatMonitorID,
		&existing.FirstFailureAt, &existing.ConfirmedAt, &existing.ResolvedAt, &existing.Status,
		&existing.FailureCategory, &existing.FirstFailureResultID, &existing.RecoveryResultID,
		&existing.AcknowledgedAt, &existing.AcknowledgedBy, &existing.NotificationState,
		&existing.DurationSeconds, &existing.Summary, &existing.CreatedAt, &existing.UpdatedAt,
	)
	if err == nil {
		return &existing, false, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, fmt.Errorf("failed to check for existing open heartbeat incident: %w", err)
	}

	var inc Incident
	err = tx.QueryRow(ctx, `
		INSERT INTO incidents (
			organization_id, heartbeat_monitor_id, first_failure_at, confirmed_at, status,
			failure_category, notification_state, summary, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, 'open', $5, 'pending', $6, NOW(), NOW())
		RETURNING id, organization_id, monitor_id, heartbeat_monitor_id, first_failure_at,
		          confirmed_at, resolved_at, status, failure_category, first_failure_result_id,
		          recovery_result_id, acknowledged_at, acknowledged_by, notification_state,
		          duration_seconds, summary, created_at, updated_at
	`, orgID, heartbeatID, firstFailureAt, confirmedAt, failureCategory, summary).Scan(
		&inc.ID, &inc.OrganizationID, &inc.MonitorID, &inc.HeartbeatMonitorID,
		&inc.FirstFailureAt, &inc.ConfirmedAt, &inc.ResolvedAt, &inc.Status,
		&inc.FailureCategory, &inc.FirstFailureResultID, &inc.RecoveryResultID,
		&inc.AcknowledgedAt, &inc.AcknowledgedBy, &inc.NotificationState,
		&inc.DurationSeconds, &inc.Summary, &inc.CreatedAt, &inc.UpdatedAt,
	)
	if err != nil {
		return nil, false, fmt.Errorf("failed to insert heartbeat incident: %w", err)
	}

	return &inc, true, nil
}

// ResolveIncidentTx marks an open incident as resolved within a transaction.
// It calculates actual duration and associates the recovery check result.
func (r *IncidentRepository) ResolveIncidentTx(
	ctx context.Context,
	tx pgx.Tx,
	orgID, incidentID string,
	resolvedAt time.Time,
	recoveryResultID *string,
) (*Incident, error) {
	var inc Incident
	var confirmedAt time.Time

	err := tx.QueryRow(ctx, `
		SELECT id, confirmed_at, status
		FROM incidents
		WHERE id = $1 AND organization_id = $2
		FOR UPDATE
	`, incidentID, orgID).Scan(&inc.ID, &confirmedAt, &inc.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrIncidentNotFound
		}
		return nil, err
	}

	if inc.Status == "resolved" {
		return nil, ErrIncidentAlreadyResolved
	}

	durationSec := int(resolvedAt.Sub(confirmedAt).Seconds())
	if durationSec < 0 {
		durationSec = 0
	}

	err = tx.QueryRow(ctx, `
		UPDATE incidents
		SET status = 'resolved',
		    resolved_at = $1,
		    recovery_result_id = $2,
		    duration_seconds = $3,
		    updated_at = NOW()
		WHERE id = $4 AND organization_id = $5
		RETURNING id, organization_id, monitor_id, heartbeat_monitor_id, first_failure_at,
		          confirmed_at, resolved_at, status, failure_category, first_failure_result_id,
		          recovery_result_id, acknowledged_at, acknowledged_by, notification_state,
		          duration_seconds, summary, created_at, updated_at
	`, resolvedAt, recoveryResultID, durationSec, incidentID, orgID).Scan(
		&inc.ID, &inc.OrganizationID, &inc.MonitorID, &inc.HeartbeatMonitorID,
		&inc.FirstFailureAt, &inc.ConfirmedAt, &inc.ResolvedAt, &inc.Status,
		&inc.FailureCategory, &inc.FirstFailureResultID, &inc.RecoveryResultID,
		&inc.AcknowledgedAt, &inc.AcknowledgedBy, &inc.NotificationState,
		&inc.DurationSeconds, &inc.Summary, &inc.CreatedAt, &inc.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve incident: %w", err)
	}

	return &inc, nil
}

// AcknowledgeIncident records acknowledgement by a user without resolving the incident.
func (r *IncidentRepository) AcknowledgeIncident(ctx context.Context, orgID, incidentID, userID string) (*Incident, error) {
	var inc Incident
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		var status string
		var ackedAt *time.Time
		err := tx.QueryRow(ctx, `
			SELECT status, acknowledged_at
			FROM incidents
			WHERE id = $1 AND organization_id = $2
			FOR UPDATE
		`, incidentID, orgID).Scan(&status, &ackedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrIncidentNotFound
			}
			return err
		}

		if ackedAt != nil {
			return ErrIncidentAlreadyAcked
		}

		now := time.Now().UTC()
		err = tx.QueryRow(ctx, `
			UPDATE incidents
			SET acknowledged_at = $1,
			    acknowledged_by = $2,
			    updated_at = NOW()
			WHERE id = $3 AND organization_id = $4
			RETURNING id, organization_id, monitor_id, heartbeat_monitor_id, first_failure_at,
			          confirmed_at, resolved_at, status, failure_category, first_failure_result_id,
			          recovery_result_id, acknowledged_at, acknowledged_by, notification_state,
			          duration_seconds, summary, created_at, updated_at
		`, now, userID, incidentID, orgID).Scan(
			&inc.ID, &inc.OrganizationID, &inc.MonitorID, &inc.HeartbeatMonitorID,
			&inc.FirstFailureAt, &inc.ConfirmedAt, &inc.ResolvedAt, &inc.Status,
			&inc.FailureCategory, &inc.FirstFailureResultID, &inc.RecoveryResultID,
			&inc.AcknowledgedAt, &inc.AcknowledgedBy, &inc.NotificationState,
			&inc.DurationSeconds, &inc.Summary, &inc.CreatedAt, &inc.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to acknowledge incident: %w", err)
		}

		// Insert notification event for acknowledgement
		// Enrich payload with rich context via JOIN (immutable snapshot at event time).
		idempKey := fmt.Sprintf("inc_ack:%s:%s", inc.ID, now.Format(time.RFC3339Nano))
		ackPayload := map[string]interface{}{
			"incident_id":      inc.ID,
			"acknowledged_by":  userID,
			"acknowledged_at":  now.Format(time.RFC3339),
			"first_failure_at": inc.FirstFailureAt.Format(time.RFC3339),
			"confirmed_at":     inc.ConfirmedAt.Format(time.RFC3339),
			"failure_category": inc.FailureCategory,
		}
		if inc.Summary != nil {
			ackPayload["failure_reason"] = *inc.Summary
		}
		var userName, userEmail string
		if err := tx.QueryRow(ctx, `SELECT display_name, email FROM users WHERE id = $1`, userID).Scan(&userName, &userEmail); err == nil {
			ackPayload["acknowledged_by_name"] = userName
			ackPayload["acknowledged_by_email"] = userEmail
		}
		if inc.MonitorID != nil {
			EnrichMonitorPayload(ctx, tx, *inc.MonitorID, orgID, ackPayload)
		} else if inc.HeartbeatMonitorID != nil {
			EnrichHeartbeatPayload(ctx, tx, *inc.HeartbeatMonitorID, orgID, ackPayload)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO notification_events (
				organization_id, event_type, incident_id, monitor_id, heartbeat_monitor_id,
				idempotency_key, payload, status, created_at
			)
			VALUES ($1, 'incident.acknowledged', $2, $3, $4, $5, $6, 'pending', NOW())
			ON CONFLICT (idempotency_key) DO NOTHING
		`, orgID, inc.ID, inc.MonitorID, inc.HeartbeatMonitorID, idempKey, ackPayload)
		return err
	})

	if err != nil {
		return nil, err
	}
	return &inc, nil
}

// GetIncidentByID retrieves an incident with joined target, client, user, and check result details.
func (r *IncidentRepository) GetIncidentByID(ctx context.Context, orgID, incidentID string) (*Incident, error) {
	var inc Incident
	var clientID, clientName, targetName, targetType string
	var assetID, assetName, ackByName, ackByEmail *string

	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			SELECT i.id, i.organization_id, i.monitor_id, i.heartbeat_monitor_id, i.first_failure_at,
			       i.confirmed_at, i.resolved_at, i.status, i.failure_category, i.first_failure_result_id,
			       i.recovery_result_id, i.acknowledged_at, i.acknowledged_by, i.notification_state,
			       i.duration_seconds, i.summary, i.created_at, i.updated_at,
			       CASE
			           WHEN i.monitor_id IS NOT NULL THEN 'monitor'
			           ELSE 'heartbeat'
			       END AS target_type,
			       COALESCE(a.name, hm.name, 'Unknown Target') AS target_name,
			       COALESCE(c.id, hmc.id) AS client_id,
			       COALESCE(c.name, hmc.name, 'Unknown Client') AS client_name,
			       a.id AS asset_id,
			       a.name AS asset_name,
			       u.display_name AS ack_by_name,
			       u.email AS ack_by_email
			FROM incidents i
			LEFT JOIN monitors m ON i.monitor_id = m.id
			LEFT JOIN assets a ON m.asset_id = a.id
			LEFT JOIN clients c ON a.client_id = c.id
			LEFT JOIN heartbeat_monitors hm ON i.heartbeat_monitor_id = hm.id
			LEFT JOIN clients hmc ON hm.client_id = hmc.id
			LEFT JOIN users u ON i.acknowledged_by = u.id
			WHERE i.id = $1 AND i.organization_id = $2
		`
		err := tx.QueryRow(ctx, query, incidentID, orgID).Scan(
			&inc.ID, &inc.OrganizationID, &inc.MonitorID, &inc.HeartbeatMonitorID,
			&inc.FirstFailureAt, &inc.ConfirmedAt, &inc.ResolvedAt, &inc.Status,
			&inc.FailureCategory, &inc.FirstFailureResultID, &inc.RecoveryResultID,
			&inc.AcknowledgedAt, &inc.AcknowledgedBy, &inc.NotificationState,
			&inc.DurationSeconds, &inc.Summary, &inc.CreatedAt, &inc.UpdatedAt,
			&targetType, &targetName, &clientID, &clientName,
			&assetID, &assetName, &ackByName, &ackByEmail,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrIncidentNotFound
			}
			return fmt.Errorf("failed to scan incident: %w", err)
		}

		inc.TargetType = targetType
		inc.TargetName = targetName
		inc.ClientID = clientID
		inc.ClientName = clientName
		inc.AssetID = assetID
		inc.AssetName = assetName
		inc.AcknowledgedByName = ackByName
		inc.AcknowledgedByEmail = ackByEmail

		// If first_failure_result_id exists, fetch check result
		if inc.FirstFailureResultID != nil {
			var cr CheckResult
			err = tx.QueryRow(ctx, `
				SELECT id, organization_id, monitor_id, scheduled_at, started_at, completed_at,
				       latency_ms, result, http_status_code, error_category, error_message,
				       tls_version, tls_cipher, tls_cert_issuer, tls_cert_subject, tls_cert_not_before,
				       tls_cert_not_after, tls_days_remaining, tls_is_valid, resolved_ip, worker_id,
				       execution_type, created_at
				FROM check_results
				WHERE id = $1 AND organization_id = $2
			`, *inc.FirstFailureResultID, orgID).Scan(
				&cr.ID, &cr.OrganizationID, &cr.MonitorID, &cr.ScheduledAt, &cr.StartedAt, &cr.CompletedAt,
				&cr.LatencyMs, &cr.Result, &cr.HTTPStatusCode, &cr.ErrorCategory, &cr.ErrorMessage,
				&cr.TLSVersion, &cr.TLSCipher, &cr.TLSCertIssuer, &cr.TLSCertSubject, &cr.TLSCertNotBefore,
				&cr.TLSCertNotAfter, &cr.TLSDaysRemaining, &cr.TLSIsValid, &cr.ResolvedIP, &cr.WorkerID,
				&cr.ExecutionType, &cr.CreatedAt,
			)
			if err == nil {
				inc.FirstFailureResult = &cr
			}
		}

		// If recovery_result_id exists, fetch check result
		if inc.RecoveryResultID != nil {
			var cr CheckResult
			err = tx.QueryRow(ctx, `
				SELECT id, organization_id, monitor_id, scheduled_at, started_at, completed_at,
				       latency_ms, result, http_status_code, error_category, error_message,
				       tls_version, tls_cipher, tls_cert_issuer, tls_cert_subject, tls_cert_not_before,
				       tls_cert_not_after, tls_days_remaining, tls_is_valid, resolved_ip, worker_id,
				       execution_type, created_at
				FROM check_results
				WHERE id = $1 AND organization_id = $2
			`, *inc.RecoveryResultID, orgID).Scan(
				&cr.ID, &cr.OrganizationID, &cr.MonitorID, &cr.ScheduledAt, &cr.StartedAt, &cr.CompletedAt,
				&cr.LatencyMs, &cr.Result, &cr.HTTPStatusCode, &cr.ErrorCategory, &cr.ErrorMessage,
				&cr.TLSVersion, &cr.TLSCipher, &cr.TLSCertIssuer, &cr.TLSCertSubject, &cr.TLSCertNotBefore,
				&cr.TLSCertNotAfter, &cr.TLSDaysRemaining, &cr.TLSIsValid, &cr.ResolvedIP, &cr.WorkerID,
				&cr.ExecutionType, &cr.CreatedAt,
			)
			if err == nil {
				inc.RecoveryResult = &cr
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return &inc, nil
}

// ListIncidents retrieves a paginated list of incidents with filters.
func (r *IncidentRepository) ListIncidents(ctx context.Context, orgID string, filter IncidentFilter) ([]*Incident, int, error) {
	var incidents []*Incident
	var totalCount int

	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		var whereClauses []string
		var args []interface{}
		argIdx := 1

		whereClauses = append(whereClauses, fmt.Sprintf("i.organization_id = $%d", argIdx))
		args = append(args, orgID)
		argIdx++

		if filter.Status != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("i.status = $%d", argIdx))
			args = append(args, filter.Status)
			argIdx++
		}

		if filter.MonitorID != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("i.monitor_id = $%d", argIdx))
			args = append(args, filter.MonitorID)
			argIdx++
		}

		if filter.HeartbeatMonitorID != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("i.heartbeat_monitor_id = $%d", argIdx))
			args = append(args, filter.HeartbeatMonitorID)
			argIdx++
		}

		if filter.FailureCategory != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("i.failure_category = $%d", argIdx))
			args = append(args, filter.FailureCategory)
			argIdx++
		}

		if filter.ClientID != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("(c.id = $%d OR hmc.id = $%d)", argIdx, argIdx))
			args = append(args, filter.ClientID)
			argIdx++
		}

		if filter.From != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("i.created_at >= $%d", argIdx))
			args = append(args, *filter.From)
			argIdx++
		}

		if filter.To != nil {
			whereClauses = append(whereClauses, fmt.Sprintf("i.created_at <= $%d", argIdx))
			args = append(args, *filter.To)
			argIdx++
		}

		whereSQL := strings.Join(whereClauses, " AND ")

		countQuery := fmt.Sprintf(`
			SELECT COUNT(*)
			FROM incidents i
			LEFT JOIN monitors m ON i.monitor_id = m.id
			LEFT JOIN assets a ON m.asset_id = a.id
			LEFT JOIN clients c ON a.client_id = c.id
			LEFT JOIN heartbeat_monitors hm ON i.heartbeat_monitor_id = hm.id
			LEFT JOIN clients hmc ON hm.client_id = hmc.id
			WHERE %s
		`, whereSQL)

		if err := tx.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
			return fmt.Errorf("failed to count incidents: %w", err)
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
			SELECT i.id, i.organization_id, i.monitor_id, i.heartbeat_monitor_id, i.first_failure_at,
			       i.confirmed_at, i.resolved_at, i.status, i.failure_category, i.first_failure_result_id,
			       i.recovery_result_id, i.acknowledged_at, i.acknowledged_by, i.notification_state,
			       i.duration_seconds, i.summary, i.created_at, i.updated_at,
			       CASE
			           WHEN i.monitor_id IS NOT NULL THEN 'monitor'
			           ELSE 'heartbeat'
			       END AS target_type,
			       COALESCE(a.name, hm.name, 'Unknown Target') AS target_name,
			       COALESCE(c.id, hmc.id) AS client_id,
			       COALESCE(c.name, hmc.name, 'Unknown Client') AS client_name,
			       a.id AS asset_id,
			       a.name AS asset_name,
			       u.display_name AS ack_by_name,
			       u.email AS ack_by_email
			FROM incidents i
			LEFT JOIN monitors m ON i.monitor_id = m.id
			LEFT JOIN assets a ON m.asset_id = a.id
			LEFT JOIN clients c ON a.client_id = c.id
			LEFT JOIN heartbeat_monitors hm ON i.heartbeat_monitor_id = hm.id
			LEFT JOIN clients hmc ON hm.client_id = hmc.id
			LEFT JOIN users u ON i.acknowledged_by = u.id
			WHERE %s
			ORDER BY i.created_at DESC
			LIMIT $%d OFFSET $%d
		`, whereSQL, argIdx, argIdx+1)

		args = append(args, limit, offset)

		rows, err := tx.Query(ctx, listQuery, args...)
		if err != nil {
			return fmt.Errorf("failed to query incidents: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var inc Incident
			var clientID, clientName, targetName, targetType string
			var assetID, assetName, ackByName, ackByEmail *string

			if err := rows.Scan(
				&inc.ID, &inc.OrganizationID, &inc.MonitorID, &inc.HeartbeatMonitorID,
				&inc.FirstFailureAt, &inc.ConfirmedAt, &inc.ResolvedAt, &inc.Status,
				&inc.FailureCategory, &inc.FirstFailureResultID, &inc.RecoveryResultID,
				&inc.AcknowledgedAt, &inc.AcknowledgedBy, &inc.NotificationState,
				&inc.DurationSeconds, &inc.Summary, &inc.CreatedAt, &inc.UpdatedAt,
				&targetType, &targetName, &clientID, &clientName,
				&assetID, &assetName, &ackByName, &ackByEmail,
			); err != nil {
				return fmt.Errorf("failed to scan incident row: %w", err)
			}

			inc.TargetType = targetType
			inc.TargetName = targetName
			inc.ClientID = clientID
			inc.ClientName = clientName
			inc.AssetID = assetID
			inc.AssetName = assetName
			inc.AcknowledgedByName = ackByName
			inc.AcknowledgedByEmail = ackByEmail

			incidents = append(incidents, &inc)
		}

		return rows.Err()
	})

	if err != nil {
		return nil, 0, err
	}
	return incidents, totalCount, nil
}

// CalculateMonitorUptime calculates operational uptime for a monitor over the specified duration window.
func (r *IncidentRepository) CalculateMonitorUptime(ctx context.Context, orgID, monitorID string, window time.Duration) (*UptimeReport, error) {
	var report UptimeReport
	now := time.Now().UTC()
	windowStart := now.Add(-window)

	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		var monitorCreatedAt time.Time
		var monitorStatus string
		var monitorIsActive bool

		err := tx.QueryRow(ctx, `
			SELECT created_at, status, is_active
			FROM monitors
			WHERE id = $1 AND organization_id = $2
		`, monitorID, orgID).Scan(&monitorCreatedAt, &monitorStatus, &monitorIsActive)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrMonitorNotFound
			}
			return err
		}

		totalRequestedSec := int64(window.Seconds())
		if totalRequestedSec <= 0 {
			totalRequestedSec = 1
		}

		coverageStart := windowStart
		if monitorCreatedAt.After(coverageStart) {
			coverageStart = monitorCreatedAt
		}
		coverageEnd := now

		var totalSec int64
		if coverageEnd.After(coverageStart) {
			totalSec = int64(coverageEnd.Sub(coverageStart).Seconds())
		}
		if totalSec <= 0 {
			totalSec = 1
		}

		coveragePct := (float64(totalSec) / float64(totalRequestedSec)) * 100.0
		if coveragePct > 100.0 {
			coveragePct = 100.0
		}
		if coveragePct < 0.0 {
			coveragePct = 0.0
		}

		// 1. Query ONLY scheduled checks (strictly excluding manual checks)
		var totalChecks, successChecks, failureChecks int
		err = tx.QueryRow(ctx, `
			SELECT 
				COUNT(*),
				COUNT(*) FILTER (WHERE result = 'success'),
				COUNT(*) FILTER (WHERE result IN ('failure', 'timeout', 'blocked', 'error'))
			FROM check_results
			WHERE monitor_id = $1 AND organization_id = $2
			  AND execution_type = 'scheduled'
			  AND created_at >= $3 AND created_at <= $4
		`, monitorID, orgID, coverageStart, coverageEnd).Scan(&totalChecks, &successChecks, &failureChecks)
		if err != nil {
			return fmt.Errorf("failed to query scheduled check results: %w", err)
		}

		// If no scheduled checks exist in the window, do NOT fabricate or assume 100% uptime.
		if totalChecks == 0 {
			report = UptimeReport{
				UptimePercentage:     nil,
				CoveragePercentage:   coveragePct,
				InsufficientData:     true,
				TotalSeconds:         totalSec,
				OutageSeconds:        0,
				IncidentCount:        0,
				TotalScheduledChecks: 0,
				SuccessfulChecks:     0,
				FailedChecks:         0,
				CoverageStart:        coverageStart,
				CoverageEnd:          coverageEnd,
				LimitationNotice:     "Insufficient monitoring data. No scheduled check results exist for the requested window. Manual checks are excluded.",
			}
			return nil
		}

		// 2. Query incidents intersecting with the coverage window
		rows, err := tx.Query(ctx, `
			SELECT confirmed_at, resolved_at
			FROM incidents
			WHERE monitor_id = $1 AND organization_id = $2
			  AND confirmed_at <= $3
			  AND (resolved_at IS NULL OR resolved_at >= $4)
			ORDER BY confirmed_at ASC
		`, monitorID, orgID, coverageEnd, coverageStart)
		if err != nil {
			return fmt.Errorf("failed to query incidents for uptime: %w", err)
		}
		defer rows.Close()

		var totalOutageSec int64
		var incCount int

		for rows.Next() {
			var confAt time.Time
			var resAt *time.Time
			if err := rows.Scan(&confAt, &resAt); err != nil {
				return err
			}

			incCount++
			outageStart := confAt
			if outageStart.Before(coverageStart) {
				outageStart = coverageStart
			}

			outageEnd := now
			if resAt != nil && resAt.Before(now) {
				outageEnd = *resAt
			}

			if outageEnd.After(outageStart) {
				totalOutageSec += int64(outageEnd.Sub(outageStart).Seconds())
			}
		}

		if err := rows.Err(); err != nil {
			return err
		}

		if totalOutageSec > totalSec {
			totalOutageSec = totalSec
		}

		uptimeSec := totalSec - totalOutageSec
		percentage := (float64(uptimeSec) / float64(totalSec)) * 100.0
		if percentage > 100.0 {
			percentage = 100.0
		}
		if percentage < 0.0 {
			percentage = 0.0
		}

		report = UptimeReport{
			UptimePercentage:     &percentage,
			CoveragePercentage:   coveragePct,
			InsufficientData:     false,
			TotalSeconds:         totalSec,
			OutageSeconds:        totalOutageSec,
			IncidentCount:        incCount,
			TotalScheduledChecks: totalChecks,
			SuccessfulChecks:     successChecks,
			FailedChecks:         failureChecks,
			CoverageStart:        coverageStart,
			CoverageEnd:          coverageEnd,
			LimitationNotice:     "Calculated from confirmed incident detections and scheduled check history. Outage start is estimated at first confirmed failure. Manual checks and paused periods are excluded.",
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &report, nil
}



