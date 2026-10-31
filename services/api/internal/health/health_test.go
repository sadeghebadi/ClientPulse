package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(ctx context.Context) error {
	return m.err
}

type mockWorkerChecker struct {
	hb  *WorkerHeartbeat
	err error
}

func (m *mockWorkerChecker) GetLatestHeartbeat(ctx context.Context) (*WorkerHeartbeat, error) {
	return m.hb, m.err
}

func TestHealthLive(t *testing.T) {
	handler := NewHandler(nil, nil, 10*time.Second, "1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rr := httptest.NewRecorder()

	handler.Live(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp LiveResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" || resp.Service != "clientpulse-api" {
		t.Errorf("unexpected live response: %+v", resp)
	}
}

func TestHealthReady_Success(t *testing.T) {
	mockDB := &mockPinger{err: nil}
	handler := NewHandler(mockDB, nil, 10*time.Second, "1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rr := httptest.NewRecorder()

	handler.Ready(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp ReadyResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" || resp.Database != "connected" {
		t.Errorf("unexpected ready response: %+v", resp)
	}
}

func TestHealthReady_Failure(t *testing.T) {
	mockDB := &mockPinger{err: errors.New("connection refused")}
	handler := NewHandler(mockDB, nil, 10*time.Second, "1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rr := httptest.NewRecorder()

	handler.Ready(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", rr.Code)
	}

	var resp ReadyResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "unavailable" || resp.Database != "disconnected" {
		t.Errorf("unexpected ready failure response: %+v", resp)
	}
}

func TestHealthWorker_Healthy(t *testing.T) {
	mockChecker := &mockWorkerChecker{
		hb: &WorkerHeartbeat{
			WorkerID:        "worker-test-1",
			Hostname:        "test-host",
			PID:             1234,
			StartedAt:       time.Now().Add(-1 * time.Hour),
			LastHeartbeatAt: time.Now().Add(-2 * time.Second), // fresh (< 10s)
			Status:          "active",
		},
		err: nil,
	}

	handler := NewHandler(nil, mockChecker, 10*time.Second, "1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/health/worker", nil)
	rr := httptest.NewRecorder()

	handler.Worker(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200 for fresh heartbeat, got %d", rr.Code)
	}

	var resp WorkerResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode worker response: %v", err)
	}

	if resp.Status != "ok" || resp.WorkerID != "worker-test-1" {
		t.Errorf("unexpected healthy worker response: %+v", resp)
	}
}

func TestHealthWorker_Stale(t *testing.T) {
	mockChecker := &mockWorkerChecker{
		hb: &WorkerHeartbeat{
			WorkerID:        "worker-stale-1",
			Hostname:        "test-host",
			PID:             1234,
			StartedAt:       time.Now().Add(-1 * time.Hour),
			LastHeartbeatAt: time.Now().Add(-25 * time.Second), // stale (> 10s)
			Status:          "active",
		},
		err: nil,
	}

	handler := NewHandler(nil, mockChecker, 10*time.Second, "1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/health/worker", nil)
	rr := httptest.NewRecorder()

	handler.Worker(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 for stale heartbeat, got %d", rr.Code)
	}

	var resp WorkerResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode worker response: %v", err)
	}

	if resp.Status != "unhealthy" || resp.Error != "worker heartbeat is stale" {
		t.Errorf("unexpected stale worker response: %+v", resp)
	}
}

func TestHealthWorker_Missing(t *testing.T) {
	mockChecker := &mockWorkerChecker{
		hb:  nil, // No records
		err: nil,
	}

	handler := NewHandler(nil, mockChecker, 10*time.Second, "1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/health/worker", nil)
	rr := httptest.NewRecorder()

	handler.Worker(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 for missing heartbeat, got %d", rr.Code)
	}

	var resp WorkerResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode worker response: %v", err)
	}

	if resp.Status != "unhealthy" || resp.Error != "no active worker heartbeats found in database" {
		t.Errorf("unexpected missing worker response: %+v", resp)
	}
}

func TestHealthWorker_DBError(t *testing.T) {
	mockChecker := &mockWorkerChecker{
		hb:  nil,
		err: errors.New("database disk full"),
	}

	handler := NewHandler(nil, mockChecker, 10*time.Second, "1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/health/worker", nil)
	rr := httptest.NewRecorder()

	handler.Worker(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 for database error, got %d", rr.Code)
	}

	var resp WorkerResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode worker response: %v", err)
	}

	if resp.Status != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got '%s'", resp.Status)
	}
}
