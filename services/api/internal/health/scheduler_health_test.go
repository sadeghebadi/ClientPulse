package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"clientpulse/services/api/internal/repository"
)

type mockSchedulerService struct {
	snap *SchedulerTelemetrySnapshot
	err  error
}

func (m *mockSchedulerService) GetSchedulerStatus(ctx context.Context) (SchedulerTelemetrySnapshot, error) {
	if m.err != nil {
		return SchedulerTelemetrySnapshot{}, m.err
	}
	if m.snap != nil {
		return *m.snap, nil
	}
	return SchedulerTelemetrySnapshot{State: SchedulerStateUnknown}, nil
}

func (m *mockSchedulerService) GetSchedulerAggregateHealth(ctx context.Context) (*repository.SchedulerAggregateHealth, error) {
	if m.err != nil {
		return nil, m.err
	}
	state := "Unknown"
	if m.snap != nil {
		state = string(m.snap.State)
	}
	return &repository.SchedulerAggregateHealth{
		AggregateState: state,
	}, nil
}

func (m *mockSchedulerService) GetStaleMonitors(ctx context.Context, intervalMultiplier, minToleranceSeconds, leaseTimeoutSeconds, limit int) ([]*repository.StaleMonitorItem, error) {
	return nil, nil
}

func (m *mockSchedulerService) DiagnoseSchedulingState(ctx context.Context) (*repository.SchedulingStateDiagnostic, error) {
	return nil, nil
}

func TestSchedulerHealthEvaluator(t *testing.T) {
	evaluator := NewSchedulerHealthEvaluator(1 * time.Second)
	now := time.Now().UTC()

	// 1. Nil instance
	snap := evaluator.Evaluate(nil, now)
	if snap.State != SchedulerStateUnknown {
		t.Fatalf("expected state Unknown, got %s", snap.State)
	}

	// 2. Newly started, no poll yet, within threshold -> Unknown
	inst := &repository.SchedulerInstanceState{
		InstanceID: "test-inst",
		StartedAt:  now.Add(-2 * time.Second),
	}
	snap = evaluator.Evaluate(inst, now)
	if snap.State != SchedulerStateUnknown {
		t.Fatalf("expected state Unknown for newly started scheduler, got %s", snap.State)
	}

	// 3. Started long ago, never polled beyond threshold (20s) -> Stale
	inst.StartedAt = now.Add(-25 * time.Second)
	snap = evaluator.Evaluate(inst, now)
	if snap.State != SchedulerStateStale {
		t.Fatalf("expected state Stale for unstarted scheduler, got %s", snap.State)
	}

	// 4. Polled successfully just now (zero claimed monitors is healthy!)
	lastPoll := now.Add(-1 * time.Second)
	inst.LastPollCompletedAt = &lastPoll
	inst.LastSuccessfulPollAt = &lastPoll
	inst.ClaimedMonitorsCount = 0
	inst.ConsecutiveErrors = 0
	snap = evaluator.Evaluate(inst, now)
	if snap.State != SchedulerStateHealthy {
		t.Fatalf("expected state Healthy for recent poll, got %s", snap.State)
	}

	// 5. Polled successfully in past, but has stopped for > 20s (stale threshold) -> Stale
	lastOldPoll := now.Add(-25 * time.Second)
	inst.LastPollCompletedAt = &lastOldPoll
	inst.LastSuccessfulPollAt = &lastOldPoll
	snap = evaluator.Evaluate(inst, now)
	if snap.State != SchedulerStateStale {
		t.Fatalf("expected state Stale for delayed poll, got %s", snap.State)
	}

	// 6. Polling actively (recent poll) but 3 consecutive errors -> Degraded
	recentPoll := now.Add(-1 * time.Second)
	inst.LastPollCompletedAt = &recentPoll
	inst.ConsecutiveErrors = 3
	snap = evaluator.Evaluate(inst, now)
	if snap.State != SchedulerStateDegraded {
		t.Fatalf("expected state Degraded for consecutive errors, got %s", snap.State)
	}

	// 7. Recovers after consecutive errors reset -> Healthy
	inst.ConsecutiveErrors = 0
	snap = evaluator.Evaluate(inst, now)
	if snap.State != SchedulerStateHealthy {
		t.Fatalf("expected state Healthy after recovery, got %s", snap.State)
	}
}

func TestHealthEndpoint_Scheduler(t *testing.T) {
	t.Run("Scheduler Not Configured", func(t *testing.T) {
		h := NewHandler(nil, nil, 10*time.Second, "1.0.0")
		req := httptest.NewRequest(http.MethodGet, "/health/scheduler", nil)
		rr := httptest.NewRecorder()
		h.Scheduler(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d", rr.Code)
		}
	})

	t.Run("Scheduler Healthy", func(t *testing.T) {
		mockSched := &mockSchedulerService{
			snap: &SchedulerTelemetrySnapshot{
				State:                SchedulerStateHealthy,
				InstanceID:           "test-sched-1",
				ClaimedMonitorsCount: 5,
				Summary:              "scheduler is polling actively",
			},
		}
		h := NewHandler(nil, nil, 10*time.Second, "1.0.0").WithSchedulerChecker(mockSched)
		req := httptest.NewRequest(http.MethodGet, "/health/scheduler", nil)
		rr := httptest.NewRecorder()
		h.Scheduler(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", rr.Code)
		}

		var resp SchedulerResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed decoding json: %v", err)
		}
		if resp.Status != "ok" || resp.Scheduler == nil || resp.Scheduler.State != SchedulerStateHealthy {
			t.Fatalf("unexpected response: %+v", resp)
		}
	})

	t.Run("Scheduler Stale Returns 503", func(t *testing.T) {
		mockSched := &mockSchedulerService{
			snap: &SchedulerTelemetrySnapshot{
				State:   SchedulerStateStale,
				Summary: "scheduler has not completed a poll cycle in 20s",
			},
		}
		h := NewHandler(nil, nil, 10*time.Second, "1.0.0").WithSchedulerChecker(mockSched)
		req := httptest.NewRequest(http.MethodGet, "/health/scheduler", nil)
		rr := httptest.NewRecorder()
		h.Scheduler(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503 Service Unavailable, got %d", rr.Code)
		}
	})

	t.Run("Scheduler Degraded Returns 503", func(t *testing.T) {
		mockSched := &mockSchedulerService{
			snap: &SchedulerTelemetrySnapshot{
				State:   SchedulerStateDegraded,
				Summary: "scheduler is experiencing repeated errors",
			},
		}
		h := NewHandler(nil, nil, 10*time.Second, "1.0.0").WithSchedulerChecker(mockSched)
		req := httptest.NewRequest(http.MethodGet, "/health/scheduler", nil)
		rr := httptest.NewRecorder()
		h.Scheduler(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503 Service Unavailable for Degraded state, got %d", rr.Code)
		}
		var resp SchedulerResponse
		_ = json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp.Status != "degraded" {
			t.Fatalf("expected status degraded, got %s", resp.Status)
		}
	})

	t.Run("Scheduler Unknown Returns 503", func(t *testing.T) {
		mockSched := &mockSchedulerService{
			snap: &SchedulerTelemetrySnapshot{
				State:   SchedulerStateUnknown,
				Summary: "no active scheduler instances registered",
			},
		}
		h := NewHandler(nil, nil, 10*time.Second, "1.0.0").WithSchedulerChecker(mockSched)
		req := httptest.NewRequest(http.MethodGet, "/health/scheduler", nil)
		rr := httptest.NewRecorder()
		h.Scheduler(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503 Service Unavailable for Unknown state, got %d", rr.Code)
		}
		var resp SchedulerResponse
		_ = json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp.Status != "unknown" {
			t.Fatalf("expected status unknown, got %s", resp.Status)
		}
	})
}

