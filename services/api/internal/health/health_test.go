package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(ctx context.Context) error {
	return m.err
}

func TestHealthLive(t *testing.T) {
	handler := NewHandler(nil, "1.0.0")

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
	handler := NewHandler(mockDB, "1.0.0")

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
	handler := NewHandler(mockDB, "1.0.0")

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

