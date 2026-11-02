package telegram

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strings"

	"clientpulse/services/api/internal/repository"
)

type UpdateHandler struct {
	repo        *repository.NotificationDestRepository
	sender      TelegramSender
	botUsername string
	logger      *slog.Logger
}

func NewUpdateHandler(
	repo *repository.NotificationDestRepository,
	sender TelegramSender,
	botUsername string,
	logger *slog.Logger,
) *UpdateHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &UpdateHandler{
		repo:        repo,
		sender:      sender,
		botUsername: strings.TrimPrefix(botUsername, "@"),
		logger:      logger,
	}
}

// ProcessUpdate handles an incoming Telegram update from webhook or polling.
func (h *UpdateHandler) ProcessUpdate(ctx context.Context, update *TelegramUpdate) error {
	if update == nil || update.Message == nil {
		return nil
	}

	msg := update.Message
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return nil
	}

	switch msg.Chat.Type {
	case "private":
		return h.handlePrivateMessage(ctx, msg, text)
	case "group", "supergroup":
		return h.handleGroupMessage(ctx, msg, text)
	default:
		return nil
	}
}

func (h *UpdateHandler) handlePrivateMessage(ctx context.Context, msg *TelegramMessage, text string) error {
	if !strings.HasPrefix(text, "/start") {
		return nil
	}

	parts := strings.Fields(text)
	if len(parts) < 2 {
		// Plain /start without token
		welcome := "👋 <b>Welcome to ClientPulse Bot!</b>\n\nTo connect this bot to your ClientPulse organization, go to your <b>Notification Settings</b> in the ClientPulse dashboard and click <b>Connect Telegram</b>."
		_ = h.sender.SendMessage(ctx, msg.Chat.ID, welcome, "HTML")
		return nil
	}

	rawToken := strings.TrimSpace(parts[1])
	tokenInfo, err := h.repo.ConsumeConnectionToken(ctx, rawToken)
	if err != nil {
		h.logger.Warn("invalid or expired telegram private connection token", "chat_id", msg.Chat.ID, "error", err)
		failMsg := "❌ <b>Connection Failed</b>\n\nThis connection link is invalid, expired, or has already been used.\n\nPlease generate a fresh connection link in your ClientPulse Notification Settings."
		_ = h.sender.SendMessage(ctx, msg.Chat.ID, failMsg, "HTML")
		return nil
	}

	if tokenInfo.DestinationType != "telegram_private" {
		failMsg := "❌ <b>Invalid Connection Type</b>\n\nThis code was generated for a Telegram group, not a private chat."
		_ = h.sender.SendMessage(ctx, msg.Chat.ID, failMsg, "HTML")
		return nil
	}

	// Prepare user display name
	name := "Telegram Account"
	if msg.From != nil {
		if msg.From.FirstName != "" {
			name = msg.From.FirstName
		}
		if msg.From.Username != "" {
			name = fmt.Sprintf("%s (@%s)", name, msg.From.Username)
		}
	}
	if tokenInfo.Label != "" {
		name = tokenInfo.Label
	}

	var usernamePtr *string
	var fromUserID int64
	if msg.From != nil {
		fromUserID = msg.From.ID
		if msg.From.Username != "" {
			u := msg.From.Username
			usernamePtr = &u
		}
	}

	dest, err := h.repo.ConnectTelegramDestination(
		ctx,
		tokenInfo.OrganizationID,
		tokenInfo.UserID,
		"telegram_private",
		name,
		msg.Chat.ID,
		fromUserID,
		usernamePtr,
	)
	if err != nil {
		h.logger.Error("failed binding telegram private destination", "error", err)
		failMsg := "❌ <b>Connection Error</b>\n\nCould not bind your Telegram destination to the workspace. Please try again."
		_ = h.sender.SendMessage(ctx, msg.Chat.ID, failMsg, "HTML")
		return err
	}

	successMsg := fmt.Sprintf("✅ <b>ClientPulse Successfully Connected!</b>\n\nYour account is now linked to receive real-time incident and monitoring alerts.\nDestination ID: <code>%s</code>", dest.ID)
	_ = h.sender.SendMessage(ctx, msg.Chat.ID, successMsg, "HTML")
	h.logger.Info("telegram private destination bound successfully", "destination_id", dest.ID, "org_id", tokenInfo.OrganizationID)
	return nil
}

func (h *UpdateHandler) handleGroupMessage(ctx context.Context, msg *TelegramMessage, text string) error {
	// Look for /connect or /connect@botname
	if !strings.HasPrefix(text, "/connect") {
		return nil
	}

	parts := strings.Fields(text)
	cmd := parts[0]
	// Verify command target if it contains @
	if atIdx := strings.Index(cmd, "@"); atIdx != -1 {
		targetBot := cmd[atIdx+1:]
		if h.botUsername != "" && !strings.EqualFold(targetBot, h.botUsername) {
			// Targeted to a different bot in the group
			return nil
		}
	}

	if len(parts) < 2 {
		helpMsg := "ℹ️ <b>Connect Group to ClientPulse</b>\n\nUsage: <code>/connect CONNECTION_CODE</code>\n\nYou can generate a connection code from your organization's <b>Notification Settings</b>."
		_ = h.sender.SendMessage(ctx, msg.Chat.ID, helpMsg, "HTML")
		return nil
	}

	rawCode := strings.TrimSpace(parts[1])
	tokenInfo, err := h.repo.ConsumeConnectionToken(ctx, rawCode)
	if err != nil {
		h.logger.Warn("invalid or expired telegram group connection code", "chat_id", msg.Chat.ID, "error", err)
		failMsg := "❌ <b>Group Connection Failed</b>\n\nThe connection code is invalid, expired, or has already been used.\n\nPlease generate a new code from your ClientPulse Notification Settings."
		_ = h.sender.SendMessage(ctx, msg.Chat.ID, failMsg, "HTML")
		return nil
	}

	if tokenInfo.DestinationType != "telegram_group" {
		failMsg := "❌ <b>Invalid Connection Type</b>\n\nThis code was generated for a private chat, not a group chat."
		_ = h.sender.SendMessage(ctx, msg.Chat.ID, failMsg, "HTML")
		return nil
	}

	groupName := "Telegram Group"
	if msg.Chat.Title != "" {
		groupName = msg.Chat.Title
	}
	if tokenInfo.Label != "" {
		groupName = tokenInfo.Label
	}

	var fromUserID int64
	if msg.From != nil {
		fromUserID = msg.From.ID
	}

	dest, err := h.repo.ConnectTelegramDestination(
		ctx,
		tokenInfo.OrganizationID,
		tokenInfo.UserID,
		"telegram_group",
		groupName,
		msg.Chat.ID,
		fromUserID,
		nil,
	)
	if err != nil {
		h.logger.Error("failed binding telegram group destination", "error", err)
		failMsg := "❌ <b>Group Connection Error</b>\n\nCould not bind group destination. Please verify bot permissions and try again."
		_ = h.sender.SendMessage(ctx, msg.Chat.ID, failMsg, "HTML")
		return err
	}

	successMsg := fmt.Sprintf("✅ <b>ClientPulse Group Connected!</b>\n\nGroup <b>%s</b> will now receive monitoring and incident notifications.\nDestination ID: <code>%s</code>", html.EscapeString(groupName), dest.ID)
	_ = h.sender.SendMessage(ctx, msg.Chat.ID, successMsg, "HTML")
	h.logger.Info("telegram group destination bound successfully", "destination_id", dest.ID, "group_id", msg.Chat.ID)
	return nil
}

