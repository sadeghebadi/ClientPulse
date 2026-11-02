package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSanitizeTelegramURL(t *testing.T) {
	urlWithToken := "https://api.telegram.org/bot123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11/sendMessage"
	sanitized := SanitizeTelegramURL(urlWithToken)
	if strings.Contains(sanitized, "123456:ABC-DEF") {
		t.Errorf("SanitizeTelegramURL failed to redact token: %s", sanitized)
	}
	expected := "https://api.telegram.org/bot[REDACTED]/sendMessage"
	if sanitized != expected {
		t.Errorf("expected %s, got %s", expected, sanitized)
	}
}

func TestClientSendMessage_Success(t *testing.T) {
	var receivedChatID int64
	var receivedText string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/sendMessage") {
			http.NotFound(w, r)
			return
		}

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if cid, ok := body["chat_id"].(float64); ok {
			receivedChatID = int64(cid)
		}
		if txt, ok := body["text"].(string); ok {
			receivedText = txt
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"result": map[string]interface{}{
				"message_id": 999,
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-token-123", server.URL, server.Client())
	err := client.SendMessage(context.Background(), 123456789, "Hello World", "HTML")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedChatID != 123456789 {
		t.Errorf("expected chat_id 123456789, got %d", receivedChatID)
	}
	if receivedText != "Hello World" {
		t.Errorf("expected text 'Hello World', got '%s'", receivedText)
	}
}

func TestClientSendMessage_RateLimit429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":          false,
			"error_code":  429,
			"description": "Too Many Requests: retry after 8",
			"parameters": map[string]interface{}{
				"retry_after": 8,
			},
		})
	}))
	defer server.Close()

	client := NewClient("secret-token", server.URL, server.Client())
	err := client.SendMessage(context.Background(), 123456, "Hello", "HTML")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var rateErr *RateLimitError
	if !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("expected rate limit error, got: %v", err)
	}
	// Verify it can be type asserted
	if rl, ok := err.(*RateLimitError); ok {
		rateErr = rl
	}
	if rateErr == nil || rateErr.RetryAfter != 8*time.Second {
		t.Errorf("expected retry after 8s, got %v", rateErr)
	}
}

func TestClientSendMessage_Permanent403(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":          false,
			"error_code":  403,
			"description": "Forbidden: bot was blocked by the user",
		})
	}))
	defer server.Close()

	client := NewClient("secret-token", server.URL, server.Client())
	err := client.SendMessage(context.Background(), 123456, "Hello", "HTML")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if _, ok := err.(*PermanentError); !ok {
		t.Errorf("expected *PermanentError, got: %T (%v)", err, err)
	}
}

