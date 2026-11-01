package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"clientpulse/services/api/internal/logger"
)

func TestRequestIDMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := logger.GetRequestID(r.Context())
		if reqID == "" {
			t.Error("expected request ID in context, got empty string")
		}
		w.WriteHeader(http.StatusOK)
	})

	h := RequestID(testHandler)

	// Scenario 1: No incoming X-Request-ID
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	respReqID := rr.Header().Get("X-Request-ID")
	if respReqID == "" {
		t.Error("expected generated X-Request-ID in response header")
	}

	// Scenario 2: Preserves incoming X-Request-ID
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "custom-id-999")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Header().Get("X-Request-ID") != "custom-id-999" {
		t.Errorf("expected preserved X-Request-ID 'custom-id-999', got '%s'", rr.Header().Get("X-Request-ID"))
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := SecurityHeaders(testHandler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("expected X-Content-Type-Options: nosniff, got %s", rr.Header().Get("X-Content-Type-Options"))
	}
	if rr.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("expected X-Frame-Options: DENY, got %s", rr.Header().Get("X-Frame-Options"))
	}
	if rr.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Errorf("expected Referrer-Policy: strict-origin-when-cross-origin, got %s", rr.Header().Get("Referrer-Policy"))
	}
}

func TestPanicRecoveryMiddleware(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New("development", &buf)

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("simulated critical crash")
	})

	h := RequestID(Recovery(log)(panicHandler))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 on panic, got %d", rr.Code)
	}

	var errResp map[string]map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse json error response: %v", err)
	}

	if errResp["error"]["code"] != "INTERNAL_ERROR" {
		t.Errorf("expected error code 'INTERNAL_ERROR', got '%v'", errResp["error"]["code"])
	}
}

func TestCORSMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	allowedOrigins := []string{"http://localhost:3000", "https://app.clientpulse.local"}
	h := CORS(allowedOrigins)(testHandler)

	// Preflight OPTIONS request
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected status 204 for preflight, got %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("expected Access-Control-Allow-Origin 'http://localhost:3000', got '%s'", rr.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestHeartbeatTokenRedactionInRequestLogger(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New("development", &buf)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := RequestLogger(log)(testHandler)

	rawSecretToken := "hb_sec_9999abcdef0123456789deadbeefcafebabe"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/heartbeat-pings/"+rawSecretToken, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	logOutput := buf.String()

	// Prove that raw secret token never appears anywhere in recorded log
	if bytes.Contains(buf.Bytes(), []byte(rawSecretToken)) {
		t.Fatalf("SECURITY VIOLATION: raw heartbeat token was found in request log output: %s", logOutput)
	}

	// Verify that the sanitized path was logged with [REDACTED]
	if !bytes.Contains(buf.Bytes(), []byte("/api/v1/heartbeat-pings/[REDACTED]")) {
		t.Errorf("expected redacted path in log output, got: %s", logOutput)
	}
}

func TestPublicReportTokenRedactionInRequestLogger(t *testing.T) {
	var buf bytes.Buffer
	log := logger.New("development", &buf)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := RequestLogger(log)(testHandler)

	rawShareToken := "cp_share_super_secret_report_token_12345678"
	req := httptest.NewRequest(http.MethodGet, "/public/reports/"+rawShareToken+"/pdf", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	logOutput := buf.String()

	if bytes.Contains(buf.Bytes(), []byte(rawShareToken)) {
		t.Fatalf("SECURITY VIOLATION: raw report share token was found in request log output: %s", logOutput)
	}

	if !bytes.Contains(buf.Bytes(), []byte("/public/reports/[REDACTED]/pdf")) {
		t.Errorf("expected redacted report path in log output, got: %s", logOutput)
	}
}


