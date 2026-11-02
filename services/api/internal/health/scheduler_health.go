package health

import (
	"context"
	"fmt"
	"time"

	"clientpulse/services/api/internal/repository"
)

// SchedulerHealthState represents the discrete operational state of the scheduler.
type SchedulerHealthState string

const (
	SchedulerStateUnknown  SchedulerHealthState = "Unknown"
	SchedulerStateHealthy  SchedulerHealthState = "Healthy"
	SchedulerStateDegraded SchedulerHealthState = "Degraded"
	SchedulerStateStale    SchedulerHealthState = "Stale"
)

// SchedulerTelemetrySnapshot holds an aggregated, sanitized view of scheduler health.
type SchedulerTelemetrySnapshot struct {
	State                SchedulerHealthState `json:"state"`
	InstanceID           string               `json:"instance_id,omitempty"`
	WorkerID             string               `json:"worker_id,omitempty"`
	StartedAt            *time.Time           `json:"started_at,omitempty"`
	LastPollStartedAt    *time.Time           `json:"last_poll_started_at,omitempty"`
	LastPollCompletedAt  *time.Time           `json:"last_poll_completed_at,omitempty"`
	LastSuccessfulPollAt *time.Time           `json:"last_successful_poll_at,omitempty"`
	PollDurationMs       *int64               `json:"poll_duration_ms,omitempty"`
	DueMonitorsCount     int                  `json:"due_monitors_count"`
	ClaimedMonitorsCount int                  `json:"claimed_monitors_count"`
	CompletedChecksCount int                  `json:"completed_checks_count"`
	FailedChecksCount    int                  `json:"failed_checks_count"`
	ConsecutiveErrors    int                  `json:"consecutive_errors"`
	LastErrorCategory    *string              `json:"last_error_category,omitempty"`
	LastErrorMessage     *string              `json:"last_error_message,omitempty"`
	Summary              string               `json:"summary"`
}

// SchedulerHealthEvaluator calculates the current health status of the scheduler.
type SchedulerHealthEvaluator struct {
	pollPeriod  time.Duration
	staleFactor int // multiplier of pollPeriod before considering stale (e.g. 10x)
}

// NewSchedulerHealthEvaluator creates a new evaluator based on configured poll period.
func NewSchedulerHealthEvaluator(pollPeriod time.Duration) *SchedulerHealthEvaluator {
	if pollPeriod <= 0 {
		pollPeriod = 1 * time.Second
	}
	return &SchedulerHealthEvaluator{
		pollPeriod:  pollPeriod,
		staleFactor: 10,
	}
}

// StaleThreshold returns the calculated threshold duration.
func (e *SchedulerHealthEvaluator) StaleThreshold() time.Duration {
	return e.staleThreshold()
}

func (e *SchedulerHealthEvaluator) staleThreshold() time.Duration {
	staleThreshold := time.Duration(e.staleFactor) * e.pollPeriod
	if staleThreshold < 20*time.Second {
		staleThreshold = 20 * time.Second
	}
	return staleThreshold
}

// Evaluate determines the health state based on the current instance telemetry.
func (e *SchedulerHealthEvaluator) Evaluate(inst *repository.SchedulerInstanceState, now time.Time) SchedulerTelemetrySnapshot {
	if inst == nil {
		return SchedulerTelemetrySnapshot{
			State:   SchedulerStateUnknown,
			Summary: "no active scheduler telemetry found",
		}
	}

	snap := SchedulerTelemetrySnapshot{
		InstanceID:           inst.InstanceID,
		WorkerID:             inst.WorkerID,
		StartedAt:            &inst.StartedAt,
		LastPollStartedAt:    inst.LastPollStartedAt,
		LastPollCompletedAt:  inst.LastPollCompletedAt,
		LastSuccessfulPollAt: inst.LastSuccessfulPollAt,
		PollDurationMs:       inst.PollDurationMs,
		DueMonitorsCount:     inst.DueMonitorsCount,
		ClaimedMonitorsCount: inst.ClaimedMonitorsCount,
		CompletedChecksCount: inst.CompletedChecksCount,
		FailedChecksCount:    inst.FailedChecksCount,
		ConsecutiveErrors:    inst.ConsecutiveErrors,
		LastErrorCategory:    inst.LastErrorCategory,
		LastErrorMessage:     inst.LastErrorMessage,
	}

	// Stale threshold derived from configured pollPeriod:
	staleThreshold := e.staleThreshold()

	// 1. If it has never completed a poll since starting, check how long it has been running
	if inst.LastSuccessfulPollAt == nil && inst.LastPollCompletedAt == nil {
		if now.Sub(inst.StartedAt) > staleThreshold {
			snap.State = SchedulerStateStale
			snap.Summary = "scheduler initialized but has not completed any poll cycles within threshold"
			return snap
		}
		snap.State = SchedulerStateUnknown
		snap.Summary = "scheduler initialized, first poll cycle pending"
		return snap
	}

	// 2. Check for staleness based on LastSuccessfulPollAt or LastPollCompletedAt
	lastActivity := inst.StartedAt
	if inst.LastPollCompletedAt != nil {
		lastActivity = *inst.LastPollCompletedAt
	}

	timeSinceLastActivity := now.Sub(lastActivity)
	if timeSinceLastActivity > staleThreshold {
		snap.State = SchedulerStateStale
		snap.Summary = fmt.Sprintf("scheduler has not completed a poll cycle in %v (threshold: %v)", timeSinceLastActivity.Truncate(time.Second), staleThreshold)
		return snap
	}

	// 3. Repeated consecutive failures (> 3) classify as Degraded
	if inst.ConsecutiveErrors >= 3 {
		snap.State = SchedulerStateDegraded
		snap.Summary = fmt.Sprintf("scheduler experiencing repeated errors (%d consecutive errors)", inst.ConsecutiveErrors)
		return snap
	}

	// 4. Otherwise, successfully polling -> Healthy (empty polls are healthy!)
	snap.State = SchedulerStateHealthy
	snap.Summary = "scheduler is polling actively"
	return snap
}

// SchedulerHealthService provides an interface for both API health check and future Platform Admin service.
type SchedulerHealthService interface {
	GetSchedulerStatus(ctx context.Context) (SchedulerTelemetrySnapshot, error)
	GetSchedulerAggregateHealth(ctx context.Context) (*repository.SchedulerAggregateHealth, error)
	GetStaleMonitors(ctx context.Context, intervalMultiplier, minToleranceSeconds, leaseTimeoutSeconds, limit int) ([]*repository.StaleMonitorItem, error)
	DiagnoseSchedulingState(ctx context.Context) (*repository.SchedulingStateDiagnostic, error)
}

// DefaultSchedulerHealthService implements SchedulerHealthService using the telemetry repository.
type DefaultSchedulerHealthService struct {
	repo      *repository.SchedulerTelemetryRepository
	evaluator *SchedulerHealthEvaluator
}

// NewSchedulerHealthService constructs a new SchedulerHealthService.
func NewSchedulerHealthService(repo *repository.SchedulerTelemetryRepository, pollPeriod time.Duration) *DefaultSchedulerHealthService {
	return &DefaultSchedulerHealthService{
		repo:      repo,
		evaluator: NewSchedulerHealthEvaluator(pollPeriod),
	}
}

// GetSchedulerStatus fetches latest state and evaluates health using the least-privilege procedure.
func (s *DefaultSchedulerHealthService) GetSchedulerStatus(ctx context.Context) (SchedulerTelemetrySnapshot, error) {
	staleSec := int(s.evaluator.staleThreshold().Seconds())
	agg, err := s.repo.GetSchedulerAggregateHealth(ctx, staleSec)
	if err != nil {
		return SchedulerTelemetrySnapshot{
			State:   SchedulerStateUnknown,
			Summary: "failed to retrieve scheduler aggregate health: " + err.Error(),
		}, err
	}

	snap := SchedulerTelemetrySnapshot{
		State:                SchedulerHealthState(agg.AggregateState),
		LastSuccessfulPollAt: agg.LatestSuccessfulPollAt,
		LastPollCompletedAt:  agg.LatestPollCompletedAt,
		Summary:              fmt.Sprintf("aggregate: %s (%d active, %d healthy, %d stale, %d degraded)", agg.AggregateState, agg.TotalActiveInstances, agg.HealthyInstances, agg.StaleInstances, agg.DegradedInstances),
	}
	return snap, nil
}

// GetSchedulerAggregateHealth retrieves aggregate multi-worker metrics.
func (s *DefaultSchedulerHealthService) GetSchedulerAggregateHealth(ctx context.Context) (*repository.SchedulerAggregateHealth, error) {
	staleSec := int(s.evaluator.staleThreshold().Seconds())
	return s.repo.GetSchedulerAggregateHealth(ctx, staleSec)
}

// GetStaleMonitors delegates to repository detect_stale_monitors with dynamic per-monitor thresholds and authoritative lease timeout.
func (s *DefaultSchedulerHealthService) GetStaleMonitors(ctx context.Context, intervalMultiplier, minToleranceSeconds, leaseTimeoutSeconds, limit int) ([]*repository.StaleMonitorItem, error) {
	return s.repo.DetectStaleMonitors(ctx, intervalMultiplier, minToleranceSeconds, leaseTimeoutSeconds, limit)
}

// DiagnoseSchedulingState delegates to repository diagnose_monitor_scheduling_state.
func (s *DefaultSchedulerHealthService) DiagnoseSchedulingState(ctx context.Context) (*repository.SchedulingStateDiagnostic, error) {
	return s.repo.DiagnoseSchedulingState(ctx, 60)
}

