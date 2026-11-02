package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"clientpulse/services/api/internal/config"
	"clientpulse/services/api/internal/notification/email"
	"clientpulse/services/api/internal/notification/telegram"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/tenant"
)

type NotificationHandler struct {
	repo              *repository.NotificationDestRepository
	telegramUpdateHdr *telegram.UpdateHandler
	telegramSender    telegram.TelegramSender
	emailSender       email.EmailSender
	cfg               *config.Config
	logger            *slog.Logger
}

func NewNotificationHandler(
	repo *repository.NotificationDestRepository,
	telegramUpdateHdr *telegram.UpdateHandler,
	telegramSender telegram.TelegramSender,
	emailSender email.EmailSender,
	cfg *config.Config,
	logger *slog.Logger,
) *NotificationHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &NotificationHandler{
		repo:              repo,
		telegramUpdateHdr: telegramUpdateHdr,
		telegramSender:    telegramSender,
		emailSender:       emailSender,
		cfg:               cfg,
		logger:            logger,
	}
}

// ListDestinations handles GET /api/v1/notifications/destinations
func (h *NotificationHandler) ListDestinations(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	destinations, err := h.repo.ListDestinations(r.Context(), tc.OrganizationID)
	if err != nil {
		h.logger.Error("failed listing destinations", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list destinations")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"destinations": destinations,
		"bot_username": strings.TrimPrefix(h.cfg.TelegramBotUsername, "@"),
	})
}

// CreateTelegramConnectToken handles POST /api/v1/notifications/destinations/telegram/connect-token
func (h *NotificationHandler) CreateTelegramConnectToken(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())

	var req struct {
		Label string `json:"label"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	ttl := 15 * time.Minute
	rawToken, expiresAt, err := h.repo.CreateConnectionToken(
		r.Context(),
		tc.OrganizationID,
		uc.UserID,
		"telegram_private",
		req.Label,
		ttl,
	)
	if err != nil {
		h.logger.Error("failed creating telegram connection token", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create connection token")
		return
	}

	botUsername := strings.TrimPrefix(h.cfg.TelegramBotUsername, "@")
	deepLink := fmt.Sprintf("https://t.me/%s?start=%s", botUsername, rawToken)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"token":        rawToken,
		"deep_link":    deepLink,
		"bot_username": botUsername,
		"expires_at":   expiresAt.Format(time.RFC3339),
	})
}

// CreateTelegramGroupToken handles POST /api/v1/notifications/destinations/telegram/group-token
func (h *NotificationHandler) CreateTelegramGroupToken(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())

	var req struct {
		Label string `json:"label"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	ttl := 15 * time.Minute
	rawCode, expiresAt, err := h.repo.CreateConnectionToken(
		r.Context(),
		tc.OrganizationID,
		uc.UserID,
		"telegram_group",
		req.Label,
		ttl,
	)
	if err != nil {
		h.logger.Error("failed creating telegram group connection code", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create group connection code")
		return
	}

	botUsername := strings.TrimPrefix(h.cfg.TelegramBotUsername, "@")

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"code":         rawCode,
		"bot_username": botUsername,
		"expires_at":   expiresAt.Format(time.RFC3339),
	})
}

// CreateEmailDestination handles POST /api/v1/notifications/destinations/email
func (h *NotificationHandler) CreateEmailDestination(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	uc := tenant.GetUserContext(r.Context())

	var req struct {
		Email       string          `json:"email"`
		Name        string          `json:"name"`
		Preferences map[string]bool `json:"preferences"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid JSON request payload")
		return
	}

	emailAddr := strings.TrimSpace(strings.ToLower(req.Email))
	if emailAddr == "" || !strings.Contains(emailAddr, "@") || !strings.Contains(emailAddr, ".") {
		sendJSONError(w, http.StatusBadRequest, "INVALID_EMAIL", "A valid email address is required")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = emailAddr
	}

	dest, err := h.repo.CreateEmailDestination(r.Context(), tc.OrganizationID, uc.UserID, emailAddr, name, req.Preferences)
	if err != nil {
		h.logger.Error("failed creating email destination", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create email destination")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"destination": dest,
	})
}

// UpdateDestination handles PATCH /api/v1/notifications/destinations/{id}
func (h *NotificationHandler) UpdateDestination(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	destID := r.PathValue("id")
	if destID == "" {
		sendJSONError(w, http.StatusBadRequest, "MISSING_ID", "Destination ID is required")
		return
	}

	var req struct {
		Name        *string         `json:"name"`
		IsEnabled   *bool           `json:"is_enabled"`
		Preferences map[string]bool `json:"preferences"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	dest, err := h.repo.UpdateDestination(r.Context(), tc.OrganizationID, destID, req.Name, req.IsEnabled, req.Preferences)
	if err != nil {
		if errors.Is(err, repository.ErrDestinationNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Destination not found")
			return
		}
		h.logger.Error("failed updating destination", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update destination")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"destination": dest,
	})
}

// DeleteDestination handles DELETE /api/v1/notifications/destinations/{id}
func (h *NotificationHandler) DeleteDestination(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	destID := r.PathValue("id")
	if destID == "" {
		sendJSONError(w, http.StatusBadRequest, "MISSING_ID", "Destination ID is required")
		return
	}

	err := h.repo.DisconnectDestination(r.Context(), tc.OrganizationID, destID)
	if err != nil {
		if errors.Is(err, repository.ErrDestinationNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Destination not found")
			return
		}
		h.logger.Error("failed disconnecting destination", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to disconnect destination")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Destination disconnected successfully",
	})
}

// TestDestination handles POST /api/v1/notifications/destinations/{id}/test
func (h *NotificationHandler) TestDestination(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())
	destID := r.PathValue("id")
	if destID == "" {
		sendJSONError(w, http.StatusBadRequest, "MISSING_ID", "Destination ID is required")
		return
	}

	dest, err := h.repo.GetDestinationByID(r.Context(), tc.OrganizationID, destID)
	if err != nil {
		if errors.Is(err, repository.ErrDestinationNotFound) {
			sendJSONError(w, http.StatusNotFound, "NOT_FOUND", "Destination not found")
			return
		}
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve destination")
		return
	}

	var recipient string
	var sendErr error

	testPayload := map[string]interface{}{
		"org_name": tc.OrganizationID,
		"test_at":  time.Now().UTC().Format(time.RFC3339),
	}

	switch dest.DestinationType {
	case "telegram_private", "telegram_group":
		if dest.TelegramChatID == nil {
			sendJSONError(w, http.StatusBadRequest, "UNCONFIGURED_CHAT", "Destination has no Telegram chat linked")
			return
		}
		recipient = strconv.FormatInt(*dest.TelegramChatID, 10)
		if h.telegramSender == nil {
			sendJSONError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Telegram Bot service is not configured")
			return
		}
		msg := "🔔 <b>ClientPulse Test Notification</b>\n\nYour Telegram destination is connected and verified successfully!"
		sendErr = h.telegramSender.SendMessage(r.Context(), *dest.TelegramChatID, msg, "HTML")

	case "email":
		if dest.EmailAddress == nil || *dest.EmailAddress == "" {
			sendJSONError(w, http.StatusBadRequest, "UNCONFIGURED_EMAIL", "Destination has no email address configured")
			return
		}
		recipient = *dest.EmailAddress
		if h.emailSender == nil {
			sendJSONError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Email service is not configured")
			return
		}
		rendered := email.RenderNotificationEmail("test.notification", "en", testPayload, h.cfg.AppBaseURL)
		sendErr = h.emailSender.SendEmail(r.Context(), recipient, rendered.Subject, rendered.HTMLBody, rendered.TextBody)

	default:
		sendJSONError(w, http.StatusBadRequest, "UNSUPPORTED_TYPE", "Unsupported destination type")
		return
	}

	// Record delivery attempt in notification_deliveries for audit history
	idempKey := fmt.Sprintf("test:%s:%d", dest.ID, time.Now().UnixNano())
	deliv, _ := h.repo.CreateDelivery(r.Context(), tc.OrganizationID, nil, dest.ID, dest.DestinationType, recipient, "test.notification", idempKey, testPayload)

	if sendErr != nil {
		if deliv != nil {
			_ = h.repo.MarkDeliveryFailed(r.Context(), deliv.ID, sendErr.Error(), 0, true)
		}
		h.logger.Warn("test notification failed", "dest_id", dest.ID, "error", sendErr)
		sendJSONError(w, http.StatusBadGateway, "DELIVERY_FAILED", fmt.Sprintf("Test delivery failed: %v", sendErr))
		return
	}

	if deliv != nil {
		_ = h.repo.MarkDeliverySent(r.Context(), deliv.ID)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Test notification delivered successfully",
	})
}

// ListHistory handles GET /api/v1/notifications/history
func (h *NotificationHandler) ListHistory(w http.ResponseWriter, r *http.Request) {
	tc := tenant.GetTenantContext(r.Context())

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 100 {
			limit = val
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	statusFilter := r.URL.Query().Get("status")
	destIDFilter := r.URL.Query().Get("destination_id")

	deliveries, total, err := h.repo.ListDeliveries(r.Context(), tc.OrganizationID, limit, offset, statusFilter, destIDFilter)
	if err != nil {
		h.logger.Error("failed listing delivery history", "error", err)
		sendJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve notification history")
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"deliveries": deliveries,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}

// TelegramWebhook handles POST /api/v1/integrations/telegram/webhook
func (h *NotificationHandler) TelegramWebhook(w http.ResponseWriter, r *http.Request) {
	if h.cfg.TelegramWebhookSecret != "" {
		secretHeader := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
		if secretHeader != h.cfg.TelegramWebhookSecret {
			h.logger.Warn("unauthorized telegram webhook request", "header", secretHeader)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var update telegram.TelegramUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if h.telegramUpdateHdr != nil {
		go func(up telegram.TelegramUpdate) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := h.telegramUpdateHdr.ProcessUpdate(ctx, &up); err != nil {
				h.logger.Error("failed processing telegram update", "error", err)
			}
		}(update)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
