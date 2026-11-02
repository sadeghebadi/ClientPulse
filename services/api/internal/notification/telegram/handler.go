package telegram

import (
	"context"
	"errors"
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

	// 1. Cross-tenant group hijacking prevention
	existingDest, err := h.repo.GetGroupDestinationByChatID(ctx, msg.Chat.ID)
	if err != nil {
		h.logger.Error("failed checking existing group destinations", "chat_id", msg.Chat.ID, "error", err)
	}
	if existingDest != nil {
		if existingDest.OrganizationID != tokenInfo.OrganizationID {
			failMsg := "❌ <b>Cross-Tenant Conflict</b>\n\nThis Telegram group is already registered with another ClientPulse organization.\n\nA Telegram group cannot be bound to multiple organizations simultaneously."
			_ = h.sender.SendMessage(ctx, msg.Chat.ID, failMsg, "HTML")
			return errors.New("group already connected to another organization")
		}
		if existingDest.Status == "active" {
			infoMsg := "ℹ️ <b>Group Already Active</b>\n\nThis Telegram group is already actively receiving notifications for your organization."
			_ = h.sender.SendMessage(ctx, msg.Chat.ID, infoMsg, "HTML")
			return nil
		}
	}

	groupName := "Telegram Group"
	if msg.Chat.Title != "" {
		groupName = msg.Chat.Title
	}
	if tokenInfo.Label != "" {
		groupName = tokenInfo.Label
	}

	var fromUserID int64
	var fromUsername *string
	if msg.From != nil {
		fromUserID = msg.From.ID
		if msg.From.Username != "" {
			u := msg.From.Username
			fromUsername = &u
		}
	}

	// 2. Sender Identity Verification:
	// A random code alone is NOT sufficient proof of sender identity.
	// Check whether the sender's Telegram identity is already linked & verified for the initiating admin.
	isVerifiedAdmin, err := h.repo.IsUserVerifiedTelegramIdentity(ctx, tokenInfo.OrganizationID, tokenInfo.UserID, fromUserID)
	if err != nil {
		h.logger.Error("failed checking sender verified telegram identity", "error", err)
	}

	if isVerifiedAdmin {
		// Sender is proven to be the initiating admin!
		// Verify bot send permissions in this group before committing to database.
		successMsg := fmt.Sprintf("✅ <b>ClientPulse Group Connected!</b>\n\nGroup <b>%s</b> is now active and authorized by your administrator.", html.EscapeString(groupName))
		if sendErr := h.sender.SendMessage(ctx, msg.Chat.ID, successMsg, "HTML"); sendErr != nil {
			h.logger.Error("bot lacks message sending permissions in group", "chat_id", msg.Chat.ID, "error", sendErr)
			return fmt.Errorf("bot cannot send messages to group: %w", sendErr)
		}

		dest, err := h.repo.ConnectTelegramDestination(
			ctx,
			tokenInfo.OrganizationID,
			tokenInfo.UserID,
			"telegram_group",
			groupName,
			msg.Chat.ID,
			fromUserID,
			fromUsername,
		)
		if err != nil {
			h.logger.Error("failed binding authorized telegram group destination", "error", err)
			return err
		}
		h.logger.Info("authorized telegram group connected immediately", "destination_id", dest.ID, "group_id", msg.Chat.ID)
		return nil
	}

	// 3. Sender is UNLINKED or belongs to a different identity:
	// Do NOT automatically activate the destination!
	// Place into pending approval, requiring explicit authenticated confirmation in the dashboard.
	dest, err := h.repo.CreatePendingGroupDestination(
		ctx,
		tokenInfo.OrganizationID,
		tokenInfo.UserID,
		groupName,
		msg.Chat.ID,
		fromUserID,
		fromUsername,
	)
	if err != nil {
		h.logger.Error("failed creating pending telegram group destination", "error", err)
		failMsg := "❌ <b>Group Connection Error</b>\n\nCould not register group connection request. Please try again."
		_ = h.sender.SendMessage(ctx, msg.Chat.ID, failMsg, "HTML")
		return err
	}

	pendingNotice := fmt.Sprintf("⚠️ <b>ClientPulse Group Connection Pending Approval</b>\n\nA connection was requested by Telegram user <code>%d</code>.\n\nBecause this Telegram user account is not yet linked to an organization administrator, an admin must approve this connection in the <b>ClientPulse Notification Settings</b> dashboard before notifications are sent.", fromUserID)
	if sendErr := h.sender.SendMessage(ctx, msg.Chat.ID, pendingNotice, "HTML"); sendErr != nil {
		h.logger.Error("bot lacks message sending permissions in group during pending connection", "chat_id", msg.Chat.ID, "error", sendErr)
		_ = h.repo.CancelPendingGroupDestination(ctx, tokenInfo.OrganizationID, dest.ID)
		return fmt.Errorf("bot cannot send messages to group: %w", sendErr)
	}

	h.logger.Info("telegram group connection placed into pending approval", "destination_id", dest.ID, "group_id", msg.Chat.ID, "sender_id", fromUserID)
	return nil
}

