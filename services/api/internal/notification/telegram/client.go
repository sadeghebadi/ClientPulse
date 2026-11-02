package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// WebhookInfo represents the current status of Telegram outgoing webhook.
type WebhookInfo struct {
	URL                  string   `json:"url"`
	HasCustomCertificate bool     `json:"has_custom_certificate"`
	PendingUpdateCount   int      `json:"pending_update_count"`
	LastErrorDate        *int64   `json:"last_error_date,omitempty"`
	LastErrorMessage     string   `json:"last_error_message,omitempty"`
	MaxConnections       int      `json:"max_connections,omitempty"`
	AllowedUpdates       []string `json:"allowed_updates,omitempty"`
}

// TelegramSender is the interface for sending messages and querying bot identity.
type TelegramSender interface {
	SendMessage(ctx context.Context, chatID int64, text string, parseMode string) error
	GetMe(ctx context.Context) (*TelegramUser, error)
}

// TelegramBotAPI extends TelegramSender to support long-polling and webhook lifecycle management.
type TelegramBotAPI interface {
	TelegramSender
	GetWebhookInfo(ctx context.Context) (*WebhookInfo, error)
	DeleteWebhook(ctx context.Context, dropPendingUpdates bool) error
	GetUpdates(ctx context.Context, offset int64, limit int, timeout int, allowedUpdates []string) ([]TelegramUpdate, error)
}

// Client delivers messages using Telegram Bot API.
type Client struct {
	botToken   string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a Telegram API client using default or provided HTTP client.
func NewClient(botToken string, baseURL string, httpClient *http.Client) *Client {
	if baseURL == "" {
		baseURL = "https://api.telegram.org"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if httpClient == nil {
		httpClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
			Timeout: 15 * time.Second,
		}
	}
	return &Client{
		botToken:   botToken,
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// NewClientWithProxy creates a Telegram API client with explicit proxy URL support (e.g. socks5:// or http://).
func NewClientWithProxy(botToken string, baseURL string, proxyURL string, timeout time.Duration) *Client {
	if baseURL == "" {
		baseURL = "https://api.telegram.org"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if timeout <= 0 {
		timeout = 65 * time.Second
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	if proxyURL != "" {
		if parsed, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(parsed)
		}
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   timeout,
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

// GetWebhookInfo retrieves the current status of Telegram webhook.
func (c *Client) GetWebhookInfo(ctx context.Context) (*WebhookInfo, error) {
	if c.botToken == "" {
		return nil, errors.New("telegram bot token is not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL("getWebhookInfo"), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram getWebhookInfo failed: %s", SanitizeTelegramURL(err.Error()))
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("invalid telegram getWebhookInfo response: %w", err)
	}
	if !apiResp.OK {
		return nil, fmt.Errorf("telegram getWebhookInfo error: %s", apiResp.Description)
	}

	var info WebhookInfo
	if err := json.Unmarshal(apiResp.Result, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// DeleteWebhook deletes the current outgoing webhook configuration, preserving pending updates if dropPendingUpdates is false.
func (c *Client) DeleteWebhook(ctx context.Context, dropPendingUpdates bool) error {
	if c.botToken == "" {
		return errors.New("telegram bot token is not configured")
	}

	body := map[string]interface{}{
		"drop_pending_updates": dropPendingUpdates,
	}
	payloadBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL("deleteWebhook"), bytes.NewReader(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram deleteWebhook failed: %s", SanitizeTelegramURL(err.Error()))
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return fmt.Errorf("invalid telegram deleteWebhook response: %w", err)
	}
	if !apiResp.OK {
		return fmt.Errorf("telegram deleteWebhook error: %s", apiResp.Description)
	}
	return nil
}

// GetUpdates polls Telegram for updates with offset, batch limit, long-poll timeout, and allowed update types.
func (c *Client) GetUpdates(ctx context.Context, offset int64, limit int, timeout int, allowedUpdates []string) ([]TelegramUpdate, error) {
	if c.botToken == "" {
		return nil, errors.New("telegram bot token is not configured")
	}

	if limit <= 0 {
		limit = 100
	}
	if len(allowedUpdates) == 0 {
		allowedUpdates = []string{"message"}
	}

	body := map[string]interface{}{
		"limit":           limit,
		"timeout":         timeout,
		"allowed_updates": allowedUpdates,
	}
	if offset > 0 {
		body["offset"] = offset
	}

	payloadBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL("getUpdates"), bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram getUpdates failed: %s", SanitizeTelegramURL(err.Error()))
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("invalid telegram getUpdates response: %w", err)
	}

	if !apiResp.OK {
		if resp.StatusCode == http.StatusTooManyRequests || apiResp.ErrorCode == 429 {
			retrySec := 5
			if apiResp.Parameters != nil && apiResp.Parameters.RetryAfter > 0 {
				retrySec = apiResp.Parameters.RetryAfter
			}
			return nil, &RateLimitError{
				RetryAfter: time.Duration(retrySec) * time.Second,
				Message:    apiResp.Description,
			}
		}
		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, &PermanentError{
				StatusCode: resp.StatusCode,
				Message:    apiResp.Description,
			}
		}
		return nil, fmt.Errorf("telegram getUpdates API error (HTTP %d): %s", resp.StatusCode, apiResp.Description)
	}

	var updates []TelegramUpdate
	if err := json.Unmarshal(apiResp.Result, &updates); err != nil {
		return nil, fmt.Errorf("failed unmarshaling updates: %w", err)
	}
	return updates, nil
}

