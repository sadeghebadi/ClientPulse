package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	tokenRegex = regexp.MustCompile(`/bot[0-9]+:[a-zA-Z0-9_\-]+/`)
)

// SanitizeTelegramURL redacts bot token from any Telegram API URL.
func SanitizeTelegramURL(raw string) string {
	return tokenRegex.ReplaceAllString(raw, "/bot[REDACTED]/")
}

// RateLimitError represents an HTTP 429 from Telegram Bot API with Retry-After information.
type RateLimitError struct {
	RetryAfter time.Duration
	Message    string
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("telegram rate limited: %s (retry after %v)", e.Message, e.RetryAfter)
}

// PermanentError represents non-retryable errors (e.g. 400 Chat not found, 403 Bot was blocked).
type PermanentError struct {
	StatusCode int
	Message    string
}

func (e *PermanentError) Error() string {
	return fmt.Sprintf("telegram permanent error (HTTP %d): %s", e.StatusCode, e.Message)
}

// TelegramUser represents a Telegram user or bot identity.
type TelegramUser struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

// TelegramChat represents a Telegram chat.
type TelegramChat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"` // "private", "group", "supergroup", "channel"
	Title string `json:"title,omitempty"`
}

// TelegramMessage represents an incoming or outgoing Telegram message.
type TelegramMessage struct {
	MessageID int           `json:"message_id"`
	From      *TelegramUser `json:"from,omitempty"`
	Chat      TelegramChat  `json:"chat"`
	Date      int64         `json:"date"`
	Text      string        `json:"text,omitempty"`
}

// TelegramUpdate represents a webhook update or getUpdates payload.
type TelegramUpdate struct {
	UpdateID int              `json:"update_id"`
	Message  *TelegramMessage `json:"message,omitempty"`
}

type apiResponse struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result,omitempty"`
	ErrorCode   int             `json:"error_code,omitempty"`
	Description string          `json:"description,omitempty"`
	Parameters  *struct {
		RetryAfter int `json:"retry_after,omitempty"`
	} `json:"parameters,omitempty"`
}

// TelegramSender is the interface for sending messages and querying bot identity.
type TelegramSender interface {
	SendMessage(ctx context.Context, chatID int64, text string, parseMode string) error
	GetMe(ctx context.Context) (*TelegramUser, error)
}

// Client delivers messages using Telegram Bot API.
type Client struct {
	botToken   string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a Telegram API client.
func NewClient(botToken string, baseURL string, httpClient *http.Client) *Client {
	if baseURL == "" {
		baseURL = "https://api.telegram.org"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		botToken:   botToken,
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func (c *Client) apiURL(method string) string {
	return fmt.Sprintf("%s/bot%s/%s", c.baseURL, c.botToken, method)
}

// SendMessage sends an HTML or Markdown formatted message to a Telegram chat.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string, parseMode string) error {
	if c.botToken == "" {
		return errors.New("telegram bot token is not configured")
	}

	if parseMode == "" {
		parseMode = "HTML"
	}

	body := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": parseMode,
	}

	payloadBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL("sendMessage"), bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram request failed: %s", SanitizeTelegramURL(err.Error()))
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return fmt.Errorf("invalid telegram response (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	if !apiResp.OK {
		// Rate limit 429
		if resp.StatusCode == http.StatusTooManyRequests || apiResp.ErrorCode == 429 {
			retrySec := 5
			if apiResp.Parameters != nil && apiResp.Parameters.RetryAfter > 0 {
				retrySec = apiResp.Parameters.RetryAfter
			}
			return &RateLimitError{
				RetryAfter: time.Duration(retrySec) * time.Second,
				Message:    apiResp.Description,
			}
		}

		// Permanent failures: 400 chat not found, 403 bot blocked / not member
		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
			return &PermanentError{
				StatusCode: resp.StatusCode,
				Message:    apiResp.Description,
			}
		}

		return fmt.Errorf("telegram API error (HTTP %d): %s", resp.StatusCode, apiResp.Description)
	}

	return nil
}

// GetMe queries the bot's own profile.
func (c *Client) GetMe(ctx context.Context) (*TelegramUser, error) {
	if c.botToken == "" {
		return nil, errors.New("telegram bot token is not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL("getMe"), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram getMe failed: %s", SanitizeTelegramURL(err.Error()))
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("invalid telegram getMe response: %w", err)
	}

	if !apiResp.OK {
		return nil, fmt.Errorf("telegram getMe error: %s", apiResp.Description)
	}

	var user TelegramUser
	if err := json.Unmarshal(apiResp.Result, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

