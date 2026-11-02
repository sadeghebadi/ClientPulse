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
	ErrMonitorNotFound  = errors.New("monitor not found")
	ErrAssetNotInOrg    = errors.New("asset does not belong to the active organization")
	ErrInvalidMethod    = errors.New("invalid HTTP method: only GET and HEAD supported")
	ErrInvalidTimeout   = errors.New("invalid timeout: must be between 3 and 30 seconds")
	ErrInvalidInterval  = errors.New("invalid interval: must be at least 60 seconds")
	ErrInvalidStatusRng = errors.New("invalid status range: expected_status_max must be >= expected_status_min")
)

type MonitorRepository struct {
	pool *pgxpool.Pool
}

func NewMonitorRepository(pool *pgxpool.Pool) *MonitorRepository {
	return &MonitorRepository{pool: pool}
}

type CreateMonitorParams struct {
	AssetID           string
	Method            string
	URLOverride       *string
	IntervalSeconds   int
	TimeoutSeconds    int
	ExpectedStatusMin int
	ExpectedStatusMax int
	FollowRedirects   bool
	MaxRedirects      int
	ExpectedText      *string
	JSONPointer       *string
	JSONExpectedValue *string
}

func (r *MonitorRepository) CreateMonitor(ctx context.Context, orgID string, params CreateMonitorParams) (*Monitor, error) {
	if params.Method == "" {
		params.Method = "GET"
	}
	params.Method = strings.ToUpper(strings.TrimSpace(params.Method))
	if params.Method != "GET" && params.Method != "HEAD" {
		return nil, ErrInvalidMethod
	}
	if params.IntervalSeconds <= 0 {
		params.IntervalSeconds = 300
	}
	if params.IntervalSeconds < 60 {
		return nil, ErrInvalidInterval
	}
	if params.TimeoutSeconds <= 0 {
		params.TimeoutSeconds = 10
	}
	if params.TimeoutSeconds < 3 || params.TimeoutSeconds > 30 {
		return nil, ErrInvalidTimeout
	}
	if params.ExpectedStatusMin <= 0 {
		params.ExpectedStatusMin = 200
	}
	if params.ExpectedStatusMax <= 0 {
		params.ExpectedStatusMax = 399
	}
	if params.ExpectedStatusMax < params.ExpectedStatusMin {
		return nil, ErrInvalidStatusRng
	}
	if params.MaxRedirects < 0 || params.MaxRedirects > 10 {
		params.MaxRedirects = 5
	}

	var m Monitor
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		// Strict parent-child tenant ownership check: verify asset exists in this org
		var assetExists bool
		err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1 AND organization_id = $2)", params.AssetID, orgID).Scan(&assetExists)
		if err != nil {
			return err
		}
		if !assetExists {
			return ErrAssetNotInOrg
		}

		query := `
			INSERT INTO monitors (
				organization_id, asset_id, method, url_override, interval_seconds, timeout_seconds,
				expected_status_min, expected_status_max, follow_redirects, max_redirects,
				expected_text, json_pointer, json_expected_value, status, is_active,
				next_check_at, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, 'pending', true, NOW(), NOW(), NOW())
			RETURNING id, organization_id, asset_id, method, url_override, interval_seconds, timeout_seconds,
			          expected_status_min, expected_status_max, follow_redirects, max_redirects,
			          expected_text, json_pointer, json_expected_value, status, is_active,
			          next_check_at, last_checked_at, lease_acquired_at, lease_worker_id, created_at, updated_at,
			          consecutive_failures
		`
		return tx.QueryRow(ctx, query,
			orgID, params.AssetID, params.Method, params.URLOverride, params.IntervalSeconds, params.TimeoutSeconds,
			params.ExpectedStatusMin, params.ExpectedStatusMax, params.FollowRedirects, params.MaxRedirects,
			params.ExpectedText, params.JSONPointer, params.JSONExpectedValue,
		).Scan(
			&m.ID, &m.OrganizationID, &m.AssetID, &m.Method, &m.URLOverride, &m.IntervalSeconds, &m.TimeoutSeconds,
			&m.ExpectedStatusMin, &m.ExpectedStatusMax, &m.FollowRedirects, &m.MaxRedirects,
			&m.ExpectedText, &m.JSONPointer, &m.JSONExpectedValue, &m.Status, &m.IsActive,
			&m.NextCheckAt, &m.LastCheckedAt, &m.LeaseAcquiredAt, &m.LeaseWorkerID, &m.CreatedAt, &m.UpdatedAt,
			&m.ConsecutiveFailures,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create monitor: %w", err)
	}

	return &m, nil
}

func (r *MonitorRepository) GetMonitorByID(ctx context.Context, orgID, monitorID string) (*Monitor, error) {
	var m Monitor
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			SELECT m.id, m.organization_id, m.asset_id, m.method, m.url_override, m.interval_seconds, m.timeout_seconds,
			       m.expected_status_min, m.expected_status_max, m.follow_redirects, m.max_redirects,
			       m.expected_text, m.json_pointer, m.json_expected_value, m.status, m.is_active,
			       m.next_check_at, m.last_checked_at, m.lease_acquired_at, m.lease_worker_id, m.created_at, m.updated_at,
			       m.consecutive_failures,
			       a.name AS asset_name, a.url AS asset_url,
			       c.name AS client_name, c.id AS client_id,
			       COALESCE(m.url_override, a.url) AS effective_url
			FROM monitors m
			JOIN assets a ON m.asset_id = a.id
			JOIN clients c ON a.client_id = c.id
			WHERE m.id = $1 AND m.organization_id = $2
		`
		err := tx.QueryRow(ctx, query, monitorID, orgID).Scan(
			&m.ID, &m.OrganizationID, &m.AssetID, &m.Method, &m.URLOverride, &m.IntervalSeconds, &m.TimeoutSeconds,
			&m.ExpectedStatusMin, &m.ExpectedStatusMax, &m.FollowRedirects, &m.MaxRedirects,
			&m.ExpectedText, &m.JSONPointer, &m.JSONExpectedValue, &m.Status, &m.IsActive,
			&m.NextCheckAt, &m.LastCheckedAt, &m.LeaseAcquiredAt, &m.LeaseWorkerID, &m.CreatedAt, &m.UpdatedAt,
			&m.ConsecutiveFailures,
			&m.AssetName, &m.AssetURL, &m.ClientName, &m.ClientID, &m.EffectiveURL,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMonitorNotFound
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MonitorRepository) ListMonitors(ctx context.Context, orgID string, assetID, clientID, status string, limit, offset int) ([]*Monitor, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var monitors []*Monitor
	var total int

	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		baseFilter := "WHERE m.organization_id = $1"
		args := []interface{}{orgID}
		argIdx := 2

		if assetID != "" {
			baseFilter += fmt.Sprintf(" AND m.asset_id = $%d", argIdx)
			args = append(args, assetID)
			argIdx++
		}
		if clientID != "" {
			baseFilter += fmt.Sprintf(" AND a.client_id = $%d", argIdx)
			args = append(args, clientID)
			argIdx++
		}
		if status != "" && status != "all" {
			baseFilter += fmt.Sprintf(" AND m.status = $%d", argIdx)
			args = append(args, status)
			argIdx++
		}

		countQuery := fmt.Sprintf(`
			SELECT COUNT(*)
			FROM monitors m
			JOIN assets a ON m.asset_id = a.id
			%s
		`, baseFilter)
		if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
			return err
		}

		query := fmt.Sprintf(`
			SELECT m.id, m.organization_id, m.asset_id, m.method, m.url_override, m.interval_seconds, m.timeout_seconds,
			       m.expected_status_min, m.expected_status_max, m.follow_redirects, m.max_redirects,
			       m.expected_text, m.json_pointer, m.json_expected_value, m.status, m.is_active,
			       m.next_check_at, m.last_checked_at, m.lease_acquired_at, m.lease_worker_id, m.created_at, m.updated_at,
			       m.consecutive_failures,
			       a.name AS asset_name, a.url AS asset_url,
			       c.name AS client_name, c.id AS client_id,
			       COALESCE(m.url_override, a.url) AS effective_url
			FROM monitors m
			JOIN assets a ON m.asset_id = a.id
			JOIN clients c ON a.client_id = c.id
			%s
			ORDER BY m.created_at DESC
			LIMIT $%d OFFSET $%d
		`, baseFilter, argIdx, argIdx+1)

		args = append(args, limit, offset)
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var m Monitor
			if err := rows.Scan(
				&m.ID, &m.OrganizationID, &m.AssetID, &m.Method, &m.URLOverride, &m.IntervalSeconds, &m.TimeoutSeconds,
				&m.ExpectedStatusMin, &m.ExpectedStatusMax, &m.FollowRedirects, &m.MaxRedirects,
				&m.ExpectedText, &m.JSONPointer, &m.JSONExpectedValue, &m.Status, &m.IsActive,
				&m.NextCheckAt, &m.LastCheckedAt, &m.LeaseAcquiredAt, &m.LeaseWorkerID, &m.CreatedAt, &m.UpdatedAt,
				&m.ConsecutiveFailures,
				&m.AssetName, &m.AssetURL, &m.ClientName, &m.ClientID, &m.EffectiveURL,
			); err != nil {
				return err
			}
			monitors = append(monitors, &m)
		}
		return rows.Err()
	})

	if err != nil {
		return nil, 0, err
	}
	return monitors, total, nil
}

type UpdateMonitorParams struct {
	Method            string
	URLOverride       *string
	IntervalSeconds   int
	TimeoutSeconds    int
	ExpectedStatusMin int
	ExpectedStatusMax int
	FollowRedirects   bool
	MaxRedirects      int
	ExpectedText      *string
	JSONPointer       *string
	JSONExpectedValue *string
}

func (r *MonitorRepository) UpdateMonitor(ctx context.Context, orgID, monitorID string, params UpdateMonitorParams) (*Monitor, error) {
	params.Method = strings.ToUpper(strings.TrimSpace(params.Method))
	if params.Method != "GET" && params.Method != "HEAD" {
		return nil, ErrInvalidMethod
	}
	if params.IntervalSeconds < 60 {
		return nil, ErrInvalidInterval
	}
	if params.TimeoutSeconds < 3 || params.TimeoutSeconds > 30 {
		return nil, ErrInvalidTimeout
	}
	if params.ExpectedStatusMax < params.ExpectedStatusMin {
		return nil, ErrInvalidStatusRng
	}
	if params.MaxRedirects < 0 || params.MaxRedirects > 10 {
		params.MaxRedirects = 5
	}

	var m Monitor
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			UPDATE monitors
			SET method = $1,
			    url_override = $2,
			    interval_seconds = $3,
			    timeout_seconds = $4,
			    expected_status_min = $5,
			    expected_status_max = $6,
			    follow_redirects = $7,
			    max_redirects = $8,
			    expected_text = $9,
			    json_pointer = $10,
			    json_expected_value = $11,
			    updated_at = NOW()
			WHERE id = $12 AND organization_id = $13
			RETURNING id, organization_id, asset_id, method, url_override, interval_seconds, timeout_seconds,
			          expected_status_min, expected_status_max, follow_redirects, max_redirects,
			          expected_text, json_pointer, json_expected_value, status, is_active,
			          next_check_at, last_checked_at, lease_acquired_at, lease_worker_id, created_at, updated_at,
			          consecutive_failures
		`
		err := tx.QueryRow(ctx, query,
			params.Method, params.URLOverride, params.IntervalSeconds, params.TimeoutSeconds,
			params.ExpectedStatusMin, params.ExpectedStatusMax, params.FollowRedirects, params.MaxRedirects,
			params.ExpectedText, params.JSONPointer, params.JSONExpectedValue,
			monitorID, orgID,
		).Scan(
			&m.ID, &m.OrganizationID, &m.AssetID, &m.Method, &m.URLOverride, &m.IntervalSeconds, &m.TimeoutSeconds,
			&m.ExpectedStatusMin, &m.ExpectedStatusMax, &m.FollowRedirects, &m.MaxRedirects,
			&m.ExpectedText, &m.JSONPointer, &m.JSONExpectedValue, &m.Status, &m.IsActive,
			&m.NextCheckAt, &m.LastCheckedAt, &m.LeaseAcquiredAt, &m.LeaseWorkerID, &m.CreatedAt, &m.UpdatedAt,
			&m.ConsecutiveFailures,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMonitorNotFound
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *MonitorRepository) PauseMonitor(ctx context.Context, orgID, monitorID string) error {
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		res, err := tx.Exec(ctx, `
			UPDATE monitors
			SET status = 'paused', updated_at = NOW()
			WHERE id = $1 AND organization_id = $2
		`, monitorID, orgID)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return ErrMonitorNotFound
		}
		return nil
	})
}

func (r *MonitorRepository) ResumeMonitor(ctx context.Context, orgID, monitorID string) error {
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		res, err := tx.Exec(ctx, `
			UPDATE monitors
			SET status = 'pending', is_active = true, next_check_at = NOW(), updated_at = NOW()
			WHERE id = $1 AND organization_id = $2
		`, monitorID, orgID)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return ErrMonitorNotFound
		}
		return nil
	})
}

func (r *MonitorRepository) DeleteMonitor(ctx context.Context, orgID, monitorID string) error {
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		res, err := tx.Exec(ctx, `
			DELETE FROM monitors
			WHERE id = $1 AND organization_id = $2
		`, monitorID, orgID)
		if err != nil {
			return err
		}
		if res.RowsAffected() == 0 {
			return ErrMonitorNotFound
		}
		return nil
	})
}

// ClaimDueMonitors executes the atomic PostgreSQL function claim_due_monitors to lease jobs.
func (r *MonitorRepository) ClaimDueMonitors(ctx context.Context, workerID string, batchSize, leaseTimeoutSeconds int) ([]*ClaimedMonitorJob, error) {
	if batchSize <= 0 {
		batchSize = 10
	}
	if leaseTimeoutSeconds <= 0 {
		leaseTimeoutSeconds = 60
	}

	query := `SELECT id, organization_id, asset_id, target_url, method, interval_seconds, timeout_seconds,
	                 expected_status_min, expected_status_max, follow_redirects, max_redirects,
	                 expected_text, json_pointer, json_expected_value, scheduled_at
	          FROM claim_due_monitors($1, $2, $3)`

	rows, err := r.pool.Query(ctx, query, workerID, batchSize, leaseTimeoutSeconds)
	if err != nil {
		return nil, fmt.Errorf("failed to claim due monitors: %w", err)
	}
	defer rows.Close()

	var jobs []*ClaimedMonitorJob
	for rows.Next() {
		var job ClaimedMonitorJob
		if err := rows.Scan(
			&job.ID, &job.OrganizationID, &job.AssetID, &job.TargetURL, &job.Method,
			&job.IntervalSeconds, &job.TimeoutSeconds, &job.ExpectedStatusMin, &job.ExpectedStatusMax,
			&job.FollowRedirects, &job.MaxRedirects, &job.ExpectedText, &job.JSONPointer,
			&job.JSONExpectedValue, &job.ScheduledAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan claimed monitor job: %w", err)
		}
		jobs = append(jobs, &job)
	}

	return jobs, rows.Err()
}

// ScheduledCheckOutcomeParams defines input parameters for recording the outcome of a scheduled check.
type ScheduledCheckOutcomeParams struct {
	OrganizationID  string
	MonitorID       string
	CheckSuccess    bool
	CompletedAt     time.Time
	ScheduledAt     time.Time
	IntervalSeconds int
	CheckResultID   *string
	ErrorCategory   *string
	ErrorMessage    *string
	// Optional enrichment fields (passed from scheduler, used for richer notification payloads)
	TargetURL      string
	HTTPStatusCode *int
}

// RecordScheduledCheckOutcome updates monitor status based on check success/failure, tracks consecutive failures,
// transitions monitor state, atomically opens/resolves incidents, and prevents duplicate alerts.
func (r *MonitorRepository) RecordScheduledCheckOutcome(ctx context.Context, orgID, monitorID string, checkSuccess bool, completedAt, scheduledAt time.Time, intervalSeconds int) (string, error) {
	status, _, err := r.RecordScheduledCheckOutcomeWithResult(ctx, ScheduledCheckOutcomeParams{
		OrganizationID:  orgID,
		MonitorID:       monitorID,
		CheckSuccess:    checkSuccess,
		CompletedAt:     completedAt,
		ScheduledAt:     scheduledAt,
		IntervalSeconds: intervalSeconds,
	})
	return status, err
}

// RecordScheduledCheckOutcomeWithResult updates monitor status and manages incident lifecycle and notification events.
func (r *MonitorRepository) RecordScheduledCheckOutcomeWithResult(ctx context.Context, params ScheduledCheckOutcomeParams) (string, *Incident, error) {
	now := time.Now().UTC()

	nextCheck := params.ScheduledAt.Add(time.Duration(params.IntervalSeconds) * time.Second)
	if nextCheck.Before(now) {
		nextCheck = now.Add(time.Duration(params.IntervalSeconds) * time.Second)
	}

	var calculatedStatus string
	var resultingIncident *Incident

	err := WithTenantTx(ctx, r.pool, params.OrganizationID, func(tx pgx.Tx) error {
		var currentStatus string
		var consecutiveFailures int
		var isActive bool

		err := tx.QueryRow(ctx, `
			SELECT status, consecutive_failures, is_active
			FROM monitors
			WHERE id = $1 AND organization_id = $2
			FOR UPDATE
		`, params.MonitorID, params.OrganizationID).Scan(&currentStatus, &consecutiveFailures, &isActive)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrMonitorNotFound
			}
			return err
		}

		// Paused monitors stay paused
		if currentStatus == "paused" || !isActive {
			calculatedStatus = "paused"
			_, err = tx.Exec(ctx, `
				UPDATE monitors
				SET last_checked_at = $1,
				    next_check_at = $2,
				    lease_acquired_at = NULL,
				    lease_worker_id = NULL,
				    updated_at = NOW()
				WHERE id = $3 AND organization_id = $4
			`, params.CompletedAt, nextCheck, params.MonitorID, params.OrganizationID)
			return err
		}

		newFailures := consecutiveFailures
		if params.CheckSuccess {
			newFailures = 0
			calculatedStatus = "up"

			// If there was an open incident for this monitor, resolve it atomically
			var openIncID string
			var confirmedAt time.Time
			err := tx.QueryRow(ctx, `
				SELECT id, confirmed_at
				FROM incidents
				WHERE monitor_id = $1 AND organization_id = $2 AND status = 'open'
				FOR UPDATE
			`, params.MonitorID, params.OrganizationID).Scan(&openIncID, &confirmedAt)

			if err == nil {
				durationSec := int(params.CompletedAt.Sub(confirmedAt).Seconds())
				if durationSec < 0 {
					durationSec = 0
				}

				var inc Incident
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
				`, params.CompletedAt, params.CheckResultID, durationSec, openIncID, params.OrganizationID).Scan(
					&inc.ID, &inc.OrganizationID, &inc.MonitorID, &inc.HeartbeatMonitorID,
					&inc.FirstFailureAt, &inc.ConfirmedAt, &inc.ResolvedAt, &inc.Status,
					&inc.FailureCategory, &inc.FirstFailureResultID, &inc.RecoveryResultID,
					&inc.AcknowledgedAt, &inc.AcknowledgedBy, &inc.NotificationState,
					&inc.DurationSeconds, &inc.Summary, &inc.CreatedAt, &inc.UpdatedAt,
				)
				if err != nil {
					return fmt.Errorf("failed to resolve incident: %w", err)
				}
				resultingIncident = &inc

				// Generate persistent notification event: incident.resolved
				// Enrich payload with rich context via JOIN (immutable snapshot at event time).
				idempKey := fmt.Sprintf("inc_res:%s:%d", inc.ID, params.CompletedAt.Unix())
				resolvedPayload := map[string]interface{}{
					"incident_id":      inc.ID,
					"monitor_id":       params.MonitorID,
					"first_failure_at": inc.FirstFailureAt.Format(time.RFC3339),
					"confirmed_at":     inc.ConfirmedAt.Format(time.RFC3339),
					"resolved_at":      params.CompletedAt.Format(time.RFC3339),
					"duration_sec":     durationSec,
					"recovery_type":    "scheduled_check_pass",
					"failure_category": inc.FailureCategory,
					"target_url":       params.TargetURL,
					"interval_seconds": params.IntervalSeconds,
				}
				if inc.Summary != nil {
					resolvedPayload["failure_reason"] = *inc.Summary
				}
				enrichMonitorPayload(ctx, tx, params.MonitorID, params.OrganizationID, resolvedPayload)
				_, err = tx.Exec(ctx, `
					INSERT INTO notification_events (
						organization_id, event_type, incident_id, monitor_id,
						idempotency_key, payload, status, created_at
					)
					VALUES ($1, 'incident.resolved', $2, $3, $4, $5, 'pending', NOW())
					ON CONFLICT (idempotency_key) DO NOTHING
				`, params.OrganizationID, inc.ID, params.MonitorID, idempKey, resolvedPayload)
				if err != nil {
					return fmt.Errorf("failed to create incident.resolved event: %w", err)
				}
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		} else {
			newFailures = consecutiveFailures + 1
			if newFailures == 1 {
				calculatedStatus = "suspected_down"
			} else {
				calculatedStatus = "down"

				// Check if an open incident already exists to prevent duplicate open incidents
				var existingIncID string
				err := tx.QueryRow(ctx, `
					SELECT id
					FROM incidents
					WHERE monitor_id = $1 AND organization_id = $2 AND status = 'open'
				`, params.MonitorID, params.OrganizationID).Scan(&existingIncID)

				if errors.Is(err, pgx.ErrNoRows) {
					// No open incident exists - create one atomically
					// Determine first failure timestamp and first failure check result
					firstFailureAt := params.CompletedAt.Add(-time.Duration(params.IntervalSeconds) * time.Second)
					firstResultID := params.CheckResultID

					var prevFailAt time.Time
					var prevResultID string
					err := tx.QueryRow(ctx, `
						SELECT completed_at, id
						FROM check_results
						WHERE monitor_id = $1 AND organization_id = $2 AND result != 'success' AND execution_type = 'scheduled'
						ORDER BY created_at DESC
						LIMIT 1 OFFSET 1
					`, params.MonitorID, params.OrganizationID).Scan(&prevFailAt, &prevResultID)
					if err == nil {
						firstFailureAt = prevFailAt
						firstResultID = &prevResultID
					}

					failCat := "http_failure"
					if params.ErrorCategory != nil && *params.ErrorCategory != "" {
						failCat = *params.ErrorCategory
					}
					var summary string
					if params.ErrorMessage != nil {
						summary = *params.ErrorMessage
					}

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
					`, params.OrganizationID, params.MonitorID, firstFailureAt, params.CompletedAt, failCat, firstResultID, summary).Scan(
						&inc.ID, &inc.OrganizationID, &inc.MonitorID, &inc.HeartbeatMonitorID,
						&inc.FirstFailureAt, &inc.ConfirmedAt, &inc.ResolvedAt, &inc.Status,
						&inc.FailureCategory, &inc.FirstFailureResultID, &inc.RecoveryResultID,
						&inc.AcknowledgedAt, &inc.AcknowledgedBy, &inc.NotificationState,
						&inc.DurationSeconds, &inc.Summary, &inc.CreatedAt, &inc.UpdatedAt,
					)
					if err != nil {
						return fmt.Errorf("failed to create incident: %w", err)
					}
					resultingIncident = &inc

					// Generate persistent notification event: incident.opened
					// Enrich payload with rich context via JOIN (immutable snapshot at event time).
					idempKey := fmt.Sprintf("inc_open:%s:%d", inc.ID, params.CompletedAt.Unix())
					openedPayload := map[string]interface{}{
						"incident_id":      inc.ID,
						"monitor_id":       params.MonitorID,
						"first_failure_at": firstFailureAt.Format(time.RFC3339),
						"confirmed_at":     params.CompletedAt.Format(time.RFC3339),
						"failure_category": failCat,
						"failure_reason":   summary,
						"target_url":       params.TargetURL,
						"interval_seconds": params.IntervalSeconds,
					}
					if params.HTTPStatusCode != nil {
						openedPayload["http_status_code"] = *params.HTTPStatusCode
					}
					enrichMonitorPayload(ctx, tx, params.MonitorID, params.OrganizationID, openedPayload)
					_, err = tx.Exec(ctx, `
						INSERT INTO notification_events (
							organization_id, event_type, incident_id, monitor_id,
							idempotency_key, payload, status, created_at
						)
						VALUES ($1, 'incident.opened', $2, $3, $4, $5, 'pending', NOW())
						ON CONFLICT (idempotency_key) DO NOTHING
					`, params.OrganizationID, inc.ID, params.MonitorID, idempKey, openedPayload)
					if err != nil {
						return fmt.Errorf("failed to create incident.opened event: %w", err)
					}
				} else if err != nil {
					return err
				}
			}
		}

		query := `
			UPDATE monitors
			SET status = $1,
			    consecutive_failures = $2,
			    last_checked_at = $3,
			    next_check_at = $4,
			    lease_acquired_at = NULL,
			    lease_worker_id = NULL,
			    updated_at = NOW()
			WHERE id = $5 AND organization_id = $6
		`
		_, err = tx.Exec(ctx, query, calculatedStatus, newFailures, params.CompletedAt, nextCheck, params.MonitorID, params.OrganizationID)
		return err
	})

	if err != nil {
		return "", nil, err
	}
	return calculatedStatus, resultingIncident, nil
}

// UpdateAfterCheck updates monitor status directly (for backward compatibility).
func (r *MonitorRepository) UpdateAfterCheck(ctx context.Context, orgID, monitorID, newStatus string, completedAt, scheduledAt time.Time, intervalSeconds int) error {
	isSuccess := newStatus == "up"
	_, err := r.RecordScheduledCheckOutcome(ctx, orgID, monitorID, isSuccess, completedAt, scheduledAt, intervalSeconds)
	return err
}


// EnrichMonitorPayload does a single JOIN query within an existing transaction to add
// asset_name, asset_url, client_id, client_name, org_name, org_timezone, locale,
// and monitor_name to the provided payload map. Errors are silently ignored so that
// notification events are always created even if enrichment fails (best-effort).
func EnrichMonitorPayload(ctx context.Context, tx pgx.Tx, monitorID, orgID string, payload map[string]interface{}) {
	var assetName, assetType, assetURL, effectiveURL, clientID, clientName, orgName, orgTimezone, orgLocale string
	err := tx.QueryRow(ctx, `
		SELECT a.name, a.type, COALESCE(a.url, ''), COALESCE(m.url_override, a.url, ''),
		       c.id, c.name, o.name, o.timezone, o.locale
		FROM monitors m
		JOIN assets a ON a.id = m.asset_id AND a.organization_id = m.organization_id
		JOIN clients c ON c.id = a.client_id AND c.organization_id = a.organization_id
		JOIN organizations o ON o.id = m.organization_id
		WHERE m.id = $1 AND m.organization_id = $2
	`, monitorID, orgID).Scan(&assetName, &assetType, &assetURL, &effectiveURL, &clientID, &clientName, &orgName, &orgTimezone, &orgLocale)
	if err != nil {
		return
	}
	payload["asset_name"] = assetName
	payload["asset_type"] = assetType
	payload["asset_url"] = assetURL
	payload["monitor_name"] = fmt.Sprintf("%s Monitor", assetName)
	payload["client_id"] = clientID
	payload["client_name"] = clientName
	payload["org_name"] = orgName
	payload["organization_name"] = orgName
	payload["org_timezone"] = orgTimezone
	payload["timezone"] = orgTimezone
	payload["locale"] = orgLocale
	if _, ok := payload["target_url"]; !ok || payload["target_url"] == "" {
		payload["target_url"] = effectiveURL
	}
}

func enrichMonitorPayload(ctx context.Context, tx pgx.Tx, monitorID, orgID string, payload map[string]interface{}) {
	EnrichMonitorPayload(ctx, tx, monitorID, orgID, payload)
}
