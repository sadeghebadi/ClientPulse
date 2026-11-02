package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SchedulerInstanceState represents the persisted state of a single scheduler instance.
type SchedulerInstanceState struct {
	InstanceID           string     `json:"instance_id"`
	WorkerID             string     `json:"worker_id"`
	StartedAt            time.Time  `json:"started_at"`
	LastPollStartedAt    *time.Time `json:"last_poll_started_at"`
	LastPollCompletedAt  *time.Time `json:"last_poll_completed_at"`
	LastSuccessfulPollAt *time.Time `json:"last_successful_poll_at"`
	PollDurationMs       *int64     `json:"poll_duration_ms"`
	DueMonitorsCount     int        `json:"due_monitors_count"`
	ClaimedMonitorsCount int        `json:"claimed_monitors_count"`
	CompletedChecksCount int        `json:"completed_checks_count"`
	FailedChecksCount    int        `json:"failed_checks_count"`
	LastErrorCategory    *string    `json:"last_error_category"`
	LastErrorMessage     *string    `json:"last_error_message"`
	ConsecutiveErrors    int        `json:"consecutive_errors"`
	Status               string     `json:"status"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// SchedulerCycleHistory represents an audit log entry of a completed scheduler polling cycle.
type SchedulerCycleHistory struct {
	ID                   int64     `json:"id"`
	InstanceID           string    `json:"instance_id"`
	PollStartedAt        time.Time `json:"poll_started_at"`
	PollCompletedAt      time.Time `json:"poll_completed_at"`
	DurationMs           int64     `json:"duration_ms"`
	DueMonitorsCount     int       `json:"due_monitors_count"`
	ClaimedMonitorsCount int       `json:"claimed_monitors_count"`
	CompletedChecksCount int       `json:"completed_checks_count"`
	FailedChecksCount    int       `json:"failed_checks_count"`
	MismatchDetected     bool      `json:"mismatch_detected"`
	ErrorCategory        *string   `json:"error_category,omitempty"`
	ErrorMessage         *string   `json:"error_message,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}

// RecordCycleParams holds data for recording a finished polling cycle.
type RecordCycleParams struct {
	InstanceID           string
	WorkerID             string
	StartedAt            time.Time
	PollStartedAt        time.Time
	PollCompletedAt      time.Time
	DurationMs           int64
	DueMonitorsCount     int
	ClaimedMonitorsCount int
	CompletedChecksCount int
	FailedChecksCount    int
	MismatchDetected     bool
	ErrorCategory        *string
	ErrorMessage         *string
	IsSuccess            bool
}

// SchedulingStateDiagnostic represents aggregate counts from the authoritative SQL check.
type SchedulingStateDiagnostic struct {
	TotalCandidates    int `json:"total_candidates"`
	EligibleJoined     int `json:"eligible_joined"`
	LeasedCount        int `json:"leased_count"`
	MismatchCandidates int `json:"mismatch_candidates"`
}

// StaleMonitorItem represents an overdue monitor identified by detect_stale_monitors.
type StaleMonitorItem struct {
	MonitorID                  string     `json:"monitor_id"`
	OrganizationID             string     `json:"organization_id"`
	Status                     string     `json:"status"`
	IntervalSeconds            int        `json:"interval_seconds"`
	NextCheckAt                time.Time  `json:"next_check_at"`
	LastCheckedAt              *time.Time `json:"last_checked_at"`
	OverdueSeconds             int        `json:"overdue_seconds"`
	EffectiveThresholdSeconds  int        `json:"effective_threshold_seconds"`
	IsFirstCheck               bool       `json:"is_first_check"`
}

// SchedulerAggregateHealth represents aggregate health across all running worker instances.
type SchedulerAggregateHealth struct {
	TotalActiveInstances   int        `json:"total_active_instances"`
	HealthyInstances       int        `json:"healthy_instances"`
	StaleInstances         int        `json:"stale_instances"`
	DegradedInstances      int        `json:"degraded_instances"`
	LatestSuccessfulPollAt *time.Time `json:"latest_successful_poll_at"`
	LatestPollCompletedAt  *time.Time `json:"latest_poll_completed_at"`
	AggregateState         string     `json:"aggregate_state"`
}

// SchedulerTelemetryRepository manages persistence and queries for scheduler telemetry.
type SchedulerTelemetryRepository struct {
	pool *pgxpool.Pool
}

// NewSchedulerTelemetryRepository creates a new repository instance.
func NewSchedulerTelemetryRepository(pool *pgxpool.Pool) *SchedulerTelemetryRepository {
	return &SchedulerTelemetryRepository{pool: pool}
}

// RegisterInstance registers or reactivates a scheduler instance state.
func (r *SchedulerTelemetryRepository) RegisterInstance(ctx context.Context, instanceID, workerID string, startedAt time.Time) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	query := `
		INSERT INTO scheduler_instances (
			instance_id, worker_id, started_at, status, updated_at
		)
		VALUES ($1, $2, $3, 'active', NOW())
		ON CONFLICT (instance_id) DO UPDATE SET
			worker_id = EXCLUDED.worker_id,
			started_at = EXCLUDED.started_at,
			status = 'active',
			updated_at = NOW();
	`
	_, err := r.pool.Exec(ctx, query, instanceID, workerID, startedAt)
	if err != nil {
		return fmt.Errorf("failed to register scheduler instance: %w", err)
	}
	return nil
}

// StopInstance marks a scheduler instance as stopped upon shutdown.
func (r *SchedulerTelemetryRepository) StopInstance(ctx context.Context, instanceID string) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	query := `
		UPDATE scheduler_instances
		SET status = 'stopped', updated_at = NOW()
		WHERE instance_id = $1;
	`
	_, err := r.pool.Exec(ctx, query, instanceID)
	if err != nil {
		return fmt.Errorf("failed to stop scheduler instance: %w", err)
	}
	return nil
}

// RecordPollCycle updates the per-instance state table and optionally records historical telemetry.
func (r *SchedulerTelemetryRepository) RecordPollCycle(ctx context.Context, p RecordCycleParams, recordHistory bool) error {
	if r == nil || r.pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	// 1. Update or Insert scheduler_instances
	var updateQuery string
	if p.IsSuccess {
		updateQuery = `
			INSERT INTO scheduler_instances (
				instance_id, worker_id, started_at,
				last_poll_started_at, last_poll_completed_at, last_successful_poll_at,
				poll_duration_ms, due_monitors_count, claimed_monitors_count,
				completed_checks_count, failed_checks_count,
				last_error_category, last_error_message,
				consecutive_errors, status, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $5, $6, $7, $8, $9, $10, NULL, NULL, 0, 'active', NOW())
			ON CONFLICT (instance_id) DO UPDATE SET
				worker_id = EXCLUDED.worker_id,
				last_poll_started_at = EXCLUDED.last_poll_started_at,
				last_poll_completed_at = EXCLUDED.last_poll_completed_at,
				last_successful_poll_at = EXCLUDED.last_poll_completed_at,
				poll_duration_ms = EXCLUDED.poll_duration_ms,
				due_monitors_count = EXCLUDED.due_monitors_count,
				claimed_monitors_count = EXCLUDED.claimed_monitors_count,
				completed_checks_count = EXCLUDED.completed_checks_count,
				failed_checks_count = EXCLUDED.failed_checks_count,
				last_error_category = NULL,
				last_error_message = NULL,
				consecutive_errors = 0,
				status = 'active',
				updated_at = NOW();
		`
		_, err := r.pool.Exec(ctx, updateQuery,
			p.InstanceID, p.WorkerID, p.StartedAt,
			p.PollStartedAt, p.PollCompletedAt,
			p.DurationMs, p.DueMonitorsCount, p.ClaimedMonitorsCount,
			p.CompletedChecksCount, p.FailedChecksCount,
		)
		if err != nil {
			return fmt.Errorf("failed to update scheduler instance state (success): %w", err)
		}
	} else {
		updateQuery = `
			INSERT INTO scheduler_instances (
				instance_id, worker_id, started_at,
				last_poll_started_at, last_poll_completed_at,
				poll_duration_ms, due_monitors_count, claimed_monitors_count,
				completed_checks_count, failed_checks_count,
				last_error_category, last_error_message,
				consecutive_errors, status, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 1, 'active', NOW())
			ON CONFLICT (instance_id) DO UPDATE SET
				worker_id = EXCLUDED.worker_id,
				last_poll_started_at = EXCLUDED.last_poll_started_at,
				last_poll_completed_at = EXCLUDED.last_poll_completed_at,
				poll_duration_ms = EXCLUDED.poll_duration_ms,
				due_monitors_count = EXCLUDED.due_monitors_count,
				claimed_monitors_count = EXCLUDED.claimed_monitors_count,
				completed_checks_count = EXCLUDED.completed_checks_count,
				failed_checks_count = EXCLUDED.failed_checks_count,
				last_error_category = EXCLUDED.last_error_category,
				last_error_message = EXCLUDED.last_error_message,
				consecutive_errors = scheduler_instances.consecutive_errors + 1,
				status = 'active',
				updated_at = NOW();
		`
		_, err := r.pool.Exec(ctx, updateQuery,
			p.InstanceID, p.WorkerID, p.StartedAt,
			p.PollStartedAt, p.PollCompletedAt,
			p.DurationMs, p.DueMonitorsCount, p.ClaimedMonitorsCount,
			p.CompletedChecksCount, p.FailedChecksCount,
			p.ErrorCategory, p.ErrorMessage,
		)
		if err != nil {
			return fmt.Errorf("failed to update scheduler instance state (failure): %w", err)
		}
	}

	// 2. Optionally record bounded cycle history
	if recordHistory {
		historyQuery := `
			INSERT INTO scheduler_cycle_history (
				instance_id, poll_started_at, poll_completed_at, duration_ms,
				due_monitors_count, claimed_monitors_count, completed_checks_count, failed_checks_count,
				mismatch_detected, error_category, error_message, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW());
		`
		_, err := r.pool.Exec(ctx, historyQuery,
			p.InstanceID, p.PollStartedAt, p.PollCompletedAt, p.DurationMs,
			p.DueMonitorsCount, p.ClaimedMonitorsCount, p.CompletedChecksCount, p.FailedChecksCount,
			p.MismatchDetected, p.ErrorCategory, p.ErrorMessage,
		)
		if err != nil {
			return fmt.Errorf("failed to record scheduler cycle history: %w", err)
		}
	}

	return nil
}

// DiagnoseSchedulingState executes the security-definer diagnostic function.
func (r *SchedulerTelemetryRepository) DiagnoseSchedulingState(ctx context.Context, leaseTimeoutSeconds int) (*SchedulingStateDiagnostic, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `SELECT total_candidates, eligible_joined, leased_count, mismatch_candidates FROM diagnose_monitor_scheduling_state($1)`
	row := r.pool.QueryRow(ctx, query, leaseTimeoutSeconds)

	var d SchedulingStateDiagnostic
	err := row.Scan(&d.TotalCandidates, &d.EligibleJoined, &d.LeasedCount, &d.MismatchCandidates)
	if err != nil {
		return nil, fmt.Errorf("failed to query scheduling state diagnosis: %w", err)
	}
	return &d, nil
}

// DetectStaleMonitors executes the security-definer stale monitor diagnostic function with dynamic per-monitor thresholds and authoritative lease timeout.
func (r *SchedulerTelemetryRepository) DetectStaleMonitors(ctx context.Context, intervalMultiplier, minToleranceSeconds, leaseTimeoutSeconds, limit int) ([]*StaleMonitorItem, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	if intervalMultiplier <= 0 {
		intervalMultiplier = 3
	}
	if minToleranceSeconds <= 0 {
		minToleranceSeconds = 180
	}
	if leaseTimeoutSeconds <= 0 {
		leaseTimeoutSeconds = 60
	}
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT monitor_id, organization_id, status, interval_seconds,
		       next_check_at, last_checked_at, overdue_seconds, effective_threshold_seconds, is_first_check
		FROM detect_stale_monitors($1, $2, $3, $4)
	`
	rows, err := r.pool.Query(ctx, query, intervalMultiplier, minToleranceSeconds, leaseTimeoutSeconds, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query stale monitors: %w", err)
	}
	defer rows.Close()

	var items []*StaleMonitorItem
	for rows.Next() {
		var item StaleMonitorItem
		if err := rows.Scan(
			&item.MonitorID, &item.OrganizationID, &item.Status, &item.IntervalSeconds,
			&item.NextCheckAt, &item.LastCheckedAt, &item.OverdueSeconds, &item.EffectiveThresholdSeconds, &item.IsFirstCheck,
		); err != nil {
			return nil, fmt.Errorf("failed to scan stale monitor item: %w", err)
		}
		items = append(items, &item)
	}
	return items, rows.Err()
}

// GetSchedulerAggregateHealth calls the security-definer function get_scheduler_aggregate_health.
func (r *SchedulerTelemetryRepository) GetSchedulerAggregateHealth(ctx context.Context, staleThresholdSeconds int) (*SchedulerAggregateHealth, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	if staleThresholdSeconds <= 0 {
		staleThresholdSeconds = 20
	}

	query := `
		SELECT total_active_instances, healthy_instances, stale_instances, degraded_instances,
		       latest_successful_poll_at, latest_poll_completed_at, aggregate_state
		FROM get_scheduler_aggregate_health($1);
	`
	row := r.pool.QueryRow(ctx, query, staleThresholdSeconds)

	var h SchedulerAggregateHealth
	err := row.Scan(
		&h.TotalActiveInstances,
		&h.HealthyInstances,
		&h.StaleInstances,
		&h.DegradedInstances,
		&h.LatestSuccessfulPollAt,
		&h.LatestPollCompletedAt,
		&h.AggregateState,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduler aggregate health: %w", err)
	}
	return &h, nil
}

// GetLatestActiveInstance retrieves the most recently active scheduler instance state.
func (r *SchedulerTelemetryRepository) GetLatestActiveInstance(ctx context.Context) (*SchedulerInstanceState, error) {
	if r == nil || r.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := `
		SELECT instance_id, worker_id, started_at, last_poll_started_at,
		       last_poll_completed_at, last_successful_poll_at, poll_duration_ms,
		       due_monitors_count, claimed_monitors_count, completed_checks_count, failed_checks_count,
		       last_error_category, last_error_message, consecutive_errors, status, updated_at
		FROM scheduler_instances
		WHERE status = 'active'
		ORDER BY last_poll_completed_at DESC NULLS LAST, updated_at DESC
		LIMIT 1;
	`
	row := r.pool.QueryRow(ctx, query)
	var s SchedulerInstanceState
	err := row.Scan(
		&s.InstanceID, &s.WorkerID, &s.StartedAt, &s.LastPollStartedAt,
		&s.LastPollCompletedAt, &s.LastSuccessfulPollAt, &s.PollDurationMs,
		&s.DueMonitorsCount, &s.ClaimedMonitorsCount, &s.CompletedChecksCount, &s.FailedChecksCount,
		&s.LastErrorCategory, &s.LastErrorMessage, &s.ConsecutiveErrors, &s.Status, &s.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest scheduler instance: %w", err)
	}
	return &s, nil
}

// PruneCycleHistory removes history rows older than the specified retention duration.
func (r *SchedulerTelemetryRepository) PruneCycleHistory(ctx context.Context, retentionDays int) (int64, error) {
	if r == nil || r.pool == nil {
		return 0, fmt.Errorf("database pool is nil")
	}
	if retentionDays <= 0 {
		retentionDays = 7
	}

	query := `
		DELETE FROM scheduler_cycle_history
		WHERE created_at < NOW() - make_interval(days => $1::int);
	`
	tag, err := r.pool.Exec(ctx, query, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("failed to prune scheduler cycle history: %w", err)
	}
	return tag.RowsAffected(), nil
}
