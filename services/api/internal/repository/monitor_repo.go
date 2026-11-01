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

// RecordScheduledCheckOutcome updates monitor status based on check success/failure, tracks consecutive failures,
// and schedules next_check_at with drift prevention & misfire policy.
// State transitions:
// - pending -> up (on first success)
// - up -> suspected_down (on 1st consecutive failure)
// - suspected_down -> down (on 2nd consecutive failure)
// - down -> up (on recovery / success)
// - paused -> remains paused (paused monitors are not marked down due to missing checks)
func (r *MonitorRepository) RecordScheduledCheckOutcome(ctx context.Context, orgID, monitorID string, checkSuccess bool, completedAt, scheduledAt time.Time, intervalSeconds int) (string, error) {
	now := time.Now().UTC()

	nextCheck := scheduledAt.Add(time.Duration(intervalSeconds) * time.Second)
	if nextCheck.Before(now) {
		nextCheck = now.Add(time.Duration(intervalSeconds) * time.Second)
	}

	var calculatedStatus string
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		var currentStatus string
		var consecutiveFailures int
		var isActive bool

		err := tx.QueryRow(ctx, `
			SELECT status, consecutive_failures, is_active
			FROM monitors
			WHERE id = $1 AND organization_id = $2
			FOR UPDATE
		`, monitorID, orgID).Scan(&currentStatus, &consecutiveFailures, &isActive)
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
			`, completedAt, nextCheck, monitorID, orgID)
			return err
		}

		newFailures := consecutiveFailures
		if checkSuccess {
			newFailures = 0
			calculatedStatus = "up"
		} else {
			newFailures = consecutiveFailures + 1
			if newFailures == 1 {
				calculatedStatus = "suspected_down"
			} else {
				calculatedStatus = "down"
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
		_, err = tx.Exec(ctx, query, calculatedStatus, newFailures, completedAt, nextCheck, monitorID, orgID)
		return err
	})

	if err != nil {
		return "", err
	}
	return calculatedStatus, nil
}

// UpdateAfterCheck updates monitor status directly (for backward compatibility).
func (r *MonitorRepository) UpdateAfterCheck(ctx context.Context, orgID, monitorID, newStatus string, completedAt, scheduledAt time.Time, intervalSeconds int) error {
	isSuccess := newStatus == "up"
	_, err := r.RecordScheduledCheckOutcome(ctx, orgID, monitorID, isSuccess, completedAt, scheduledAt, intervalSeconds)
	return err
}

