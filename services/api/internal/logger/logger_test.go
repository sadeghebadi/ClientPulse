package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

func TestStructuredLoggingWithContext(t *testing.T) {
	var buf bytes.Buffer
	log := New("development", &buf)

	ctx := context.Background()
	ctx = WithRequestID(ctx, "test-req-12345")

	log.InfoContext(ctx, "test message", "client_ip", "127.0.0.1")

	output := buf.String()
	if output == "" {
		t.Fatal("expected log output, got empty string")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("failed to parse json log: %v, raw output: %s", err, output)
	}

	if parsed["msg"] != "test message" {
		t.Errorf("expected msg 'test message', got '%v'", parsed["msg"])
	}

	if parsed["request_id"] != "test-req-12345" {
		t.Errorf("expected request_id 'test-req-12345', got '%v'", parsed["request_id"])
	}

	if parsed["client_ip"] != "127.0.0.1" {
		t.Errorf("expected client_ip '127.0.0.1', got '%v'", parsed["client_ip"])
	}
}

func TestSensitiveDataRedaction(t *testing.T) {
	var buf bytes.Buffer
	log := New("development", &buf)

	log.Info("testing redaction", "user_password", "super-secret-123", "api_token", "abc-xyz-token")

	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("failed to parse json log: %v", err)
	}

	if parsed["user_password"] != "[REDACTED]" {
		t.Errorf("expected user_password to be redacted, got '%v'", parsed["user_password"])
	}

	if parsed["api_token"] != "[REDACTED]" {
		t.Errorf("expected api_token to be redacted, got '%v'", parsed["api_token"])
	}
}

