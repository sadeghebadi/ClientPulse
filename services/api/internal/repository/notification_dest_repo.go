package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTokenInvalidOrExpired = errors.New("connection token is invalid, expired, or already used")
	ErrDestinationNotFound   = errors.New("notification destination not found")
	ErrDuplicateDestination  = errors.New("destination is already connected")
)

type NotificationDestination struct {
	ID               string          `json:"id"`
	OrganizationID   string          `json:"organization_id"`
	DestinationType  string          `json:"destination_type"`
	Name             string          `json:"name"`
	TelegramChatID   *int64          `json:"telegram_chat_id,omitempty"`
	TelegramUserID   *int64          `json:"telegram_user_id,omitempty"`
	TelegramUsername *string         `json:"telegram_username,omitempty"`
	EmailAddress     *string         `json:"email_address,omitempty"`
	UserID           *string         `json:"user_id,omitempty"`
	Status           string          `json:"status"`
	IsEnabled        bool            `json:"is_enabled"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	LastVerifiedAt   *time.Time      `json:"last_verified_at,omitempty"`
	Preferences      map[string]bool `json:"preferences,omitempty"`
}

type NotificationDelivery struct {
	ID              string                 `json:"id"`
	OrganizationID  string                 `json:"organization_id"`
	EventID         *string                `json:"event_id,omitempty"`
	DestinationID   string                 `json:"destination_id"`
	DestinationType string                 `json:"destination_type"`
	Recipient       string                 `json:"recipient"`
	EventType       string                 `json:"event_type"`
	Status          string                 `json:"status"`
	RetryCount      int                    `json:"retry_count"`
	MaxRetries      int                    `json:"max_retries"`
	NextAttemptAt   time.Time              `json:"next_attempt_at"`
	IdempotencyKey  string                 `json:"idempotency_key"`
	Payload         map[string]interface{} `json:"payload"`
	ErrorMessage    *string                `json:"error_message,omitempty"`
	LastAttemptAt   *time.Time             `json:"last_attempt_at,omitempty"`
	SentAt          *time.Time             `json:"sent_at,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type ConsumedTokenInfo struct {
	ID              string
	OrganizationID  string
	UserID          string
	DestinationType string
	Label           string
}

type NotificationDestRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationDestRepository(pool *pgxpool.Pool) *NotificationDestRepository {
	return &NotificationDestRepository{pool: pool}
}

// GenerateRandomToken generates cryptographically secure random bytes in hex.
func GenerateRandomToken(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateConnectionToken creates a single-use token with short expiration for connecting Telegram.
func (r *NotificationDestRepository) CreateConnectionToken(
	ctx context.Context,
	orgID, userID, destType, label string,
	ttl time.Duration,
) (string, time.Time, error) {
	rawToken, err := GenerateRandomToken(24) // 48 hex characters
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed generating token: %w", err)
	}

	tokenHash := HashToken(rawToken)
	expiresAt := time.Now().UTC().Add(ttl)

	var labelVal *string
	if label != "" {
		labelVal = &label
	}

	err = WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO notification_connection_tokens (
				organization_id, user_id, destination_type, token_hash, label, expires_at, created_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, NOW())
		`, orgID, userID, destType, tokenHash, labelVal, expiresAt)
		return err
	})

	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed storing connection token: %w", err)
	}

	return rawToken, expiresAt, nil
}

// ConsumeConnectionToken consumes a token atomically across tenants (invoked by Telegram bot update handler).
func (r *NotificationDestRepository) ConsumeConnectionToken(ctx context.Context, rawToken string) (*ConsumedTokenInfo, error) {
	tokenHash := HashToken(rawToken)

	var info ConsumedTokenInfo
	var labelVal *string

	err := r.pool.QueryRow(ctx, `
		SELECT id, organization_id, user_id, destination_type, COALESCE(label, '')
		FROM consume_notification_connection_token($1)
	`, tokenHash).Scan(
		&info.ID,
		&info.OrganizationID,
		&info.UserID,
		&info.DestinationType,
		&labelVal,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTokenInvalidOrExpired
		}
		return nil, fmt.Errorf("failed consuming token: %w", err)
	}

	if labelVal != nil {
		info.Label = *labelVal
	}
	return &info, nil
}

// ConnectTelegramDestination binds a verified Telegram chat as a destination for the organization.
func (r *NotificationDestRepository) ConnectTelegramDestination(
	ctx context.Context,
	orgID, userID, destType, name string,
	chatID, telegramUserID int64,
	username *string,
) (*NotificationDestination, error) {
	var dest NotificationDestination
	now := time.Now().UTC()

	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			INSERT INTO notification_destinations (
				organization_id, destination_type, name, telegram_chat_id, telegram_user_id,
				telegram_username, user_id, status, is_enabled, created_at, updated_at, last_verified_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, 'active', true, $8, $8, $8)
			ON CONFLICT (organization_id, id) DO NOTHING
			RETURNING id, organization_id, destination_type, name, telegram_chat_id, telegram_user_id,
			          telegram_username, email_address, user_id, status, is_enabled, created_at, updated_at, last_verified_at
		`
		err := tx.QueryRow(ctx, query,
			orgID, destType, name, chatID, telegramUserID,
			username, userID, now,
		).Scan(
			&dest.ID, &dest.OrganizationID, &dest.DestinationType, &dest.Name,
			&dest.TelegramChatID, &dest.TelegramUserID, &dest.TelegramUsername,
			&dest.EmailAddress, &dest.UserID, &dest.Status, &dest.IsEnabled,
			&dest.CreatedAt, &dest.UpdatedAt, &dest.LastVerifiedAt,
		)
		if err != nil {
			return fmt.Errorf("failed inserting telegram destination: %w", err)
		}

		// Insert default preferences (all active)
		defaultEvents := []string{
			"incident.opened", "incident.acknowledged", "incident.resolved",
			"heartbeat.missed", "heartbeat.recovered", "report.ready",
		}
		for _, ev := range defaultEvents {
			_, err = tx.Exec(ctx, `
				INSERT INTO notification_preferences (organization_id, destination_id, event_type, is_enabled)
				VALUES ($1, $2, $3, true)
				ON CONFLICT (destination_id, event_type) DO UPDATE SET is_enabled = true
			`, orgID, dest.ID, ev)
			if err != nil {
				return fmt.Errorf("failed inserting default preferences: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	dest.Preferences = map[string]bool{
		"incident.opened":       true,
		"incident.acknowledged": true,
		"incident.resolved":     true,
		"heartbeat.missed":      true,
		"heartbeat.recovered":   true,
		"report.ready":          true,
	}
	return &dest, nil
}

// CreateEmailDestination creates or enables an email destination.
func (r *NotificationDestRepository) CreateEmailDestination(
	ctx context.Context,
	orgID, userID, email, name string,
	preferences map[string]bool,
) (*NotificationDestination, error) {
	var dest NotificationDestination
	now := time.Now().UTC()

	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			INSERT INTO notification_destinations (
				organization_id, destination_type, name, email_address, user_id,
				status, is_enabled, created_at, updated_at, last_verified_at
			)
			VALUES ($1, 'email', $2, $3, $4, 'active', true, $5, $5, $5)
			RETURNING id, organization_id, destination_type, name, telegram_chat_id, telegram_user_id,
			          telegram_username, email_address, user_id, status, is_enabled, created_at, updated_at, last_verified_at
		`
		err := tx.QueryRow(ctx, query, orgID, name, email, userID, now).Scan(
			&dest.ID, &dest.OrganizationID, &dest.DestinationType, &dest.Name,
			&dest.TelegramChatID, &dest.TelegramUserID, &dest.TelegramUsername,
			&dest.EmailAddress, &dest.UserID, &dest.Status, &dest.IsEnabled,
			&dest.CreatedAt, &dest.UpdatedAt, &dest.LastVerifiedAt,
		)
		if err != nil {
			return fmt.Errorf("failed inserting email destination: %w", err)
		}

		allEvents := []string{
			"incident.opened", "incident.acknowledged", "incident.resolved",
			"heartbeat.missed", "heartbeat.recovered", "report.ready",
		}
		for _, ev := range allEvents {
			enabled := true
			if preferences != nil {
				if val, exists := preferences[ev]; exists {
					enabled = val
				}
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO notification_preferences (organization_id, destination_id, event_type, is_enabled)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (destination_id, event_type) DO UPDATE SET is_enabled = EXCLUDED.is_enabled
			`, orgID, dest.ID, ev, enabled)
			if err != nil {
				return fmt.Errorf("failed inserting preference for %s: %w", ev, err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	dest.Preferences = preferences
	return &dest, nil
}

// ListDestinations lists all destinations for an organization with their event preferences.
func (r *NotificationDestRepository) ListDestinations(ctx context.Context, orgID string) ([]*NotificationDestination, error) {
	var results []*NotificationDestination

	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, organization_id, destination_type, name, telegram_chat_id, telegram_user_id,
			       telegram_username, email_address, user_id, status, is_enabled, created_at, updated_at, last_verified_at
			FROM notification_destinations
			WHERE organization_id = $1 AND status != 'disconnected'
			ORDER BY created_at DESC
		`, orgID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var d NotificationDestination
			if err := rows.Scan(
				&d.ID, &d.OrganizationID, &d.DestinationType, &d.Name,
				&d.TelegramChatID, &d.TelegramUserID, &d.TelegramUsername,
				&d.EmailAddress, &d.UserID, &d.Status, &d.IsEnabled,
				&d.CreatedAt, &d.UpdatedAt, &d.LastVerifiedAt,
			); err != nil {
				return err
			}
			d.Preferences = make(map[string]bool)
			results = append(results, &d)
		}

		if len(results) == 0 {
			return nil
		}

		// Fetch preferences for all destinations in org
		prefRows, err := tx.Query(ctx, `
			SELECT destination_id, event_type, is_enabled
			FROM notification_preferences
			WHERE organization_id = $1
		`, orgID)
		if err != nil {
			return err
		}
		defer prefRows.Close()

		destMap := make(map[string]*NotificationDestination)
		for _, d := range results {
			destMap[d.ID] = d
		}

		for prefRows.Next() {
			var destID, ev string
			var enabled bool
			if err := prefRows.Scan(&destID, &ev, &enabled); err != nil {
				return err
			}
			if d, ok := destMap[destID]; ok {
				d.Preferences[ev] = enabled
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed listing destinations: %w", err)
	}

	return results, nil
}

// GetDestinationByID returns a single destination by ID.
func (r *NotificationDestRepository) GetDestinationByID(ctx context.Context, orgID, destID string) (*NotificationDestination, error) {
	var d NotificationDestination
	d.Preferences = make(map[string]bool)

	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			SELECT id, organization_id, destination_type, name, telegram_chat_id, telegram_user_id,
			       telegram_username, email_address, user_id, status, is_enabled, created_at, updated_at, last_verified_at
			FROM notification_destinations
			WHERE organization_id = $1 AND id = $2 AND status != 'disconnected'
		`
		err := tx.QueryRow(ctx, query, orgID, destID).Scan(
			&d.ID, &d.OrganizationID, &d.DestinationType, &d.Name,
			&d.TelegramChatID, &d.TelegramUserID, &d.TelegramUsername,
			&d.EmailAddress, &d.UserID, &d.Status, &d.IsEnabled,
			&d.CreatedAt, &d.UpdatedAt, &d.LastVerifiedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrDestinationNotFound
			}
			return err
		}

		rows, err := tx.Query(ctx, `
			SELECT event_type, is_enabled
			FROM notification_preferences
			WHERE organization_id = $1 AND destination_id = $2
		`, orgID, destID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var ev string
			var enabled bool
			if err := rows.Scan(&ev, &enabled); err != nil {
				return err
			}
			d.Preferences[ev] = enabled
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return &d, nil
}

// UpdateDestination updates mutable fields (name, is_enabled, preferences).
func (r *NotificationDestRepository) UpdateDestination(
	ctx context.Context,
	orgID, destID string,
	name *string,
	isEnabled *bool,
	preferences map[string]bool,
) (*NotificationDestination, error) {
	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		if name != nil || isEnabled != nil {
			updateQuery := `
				UPDATE notification_destinations
				SET name = COALESCE($1, name),
				    is_enabled = COALESCE($2, is_enabled),
				    updated_at = NOW()
				WHERE organization_id = $3 AND id = $4 AND status != 'disconnected'
			`
			ct, err := tx.Exec(ctx, updateQuery, name, isEnabled, orgID, destID)
			if err != nil {
				return err
			}
			if ct.RowsAffected() == 0 {
				return ErrDestinationNotFound
			}
		}

		if preferences != nil {
			for ev, enabled := range preferences {
				_, err := tx.Exec(ctx, `
					INSERT INTO notification_preferences (organization_id, destination_id, event_type, is_enabled)
					VALUES ($1, $2, $3, $4)
					ON CONFLICT (destination_id, event_type) DO UPDATE SET is_enabled = EXCLUDED.is_enabled, updated_at = NOW()
				`, orgID, destID, ev, enabled)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return r.GetDestinationByID(ctx, orgID, destID)
}

// DisconnectDestination marks a destination as disconnected so no future notifications are sent.
func (r *NotificationDestRepository) DisconnectDestination(ctx context.Context, orgID, destID string) error {
	return WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx, `
			UPDATE notification_destinations
			SET status = 'disconnected', is_enabled = false, updated_at = NOW()
			WHERE organization_id = $1 AND id = $2 AND status != 'disconnected'
		`, orgID, destID)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return ErrDestinationNotFound
		}
		return nil
	})
}

// CreateDelivery queues a single notification delivery.
func (r *NotificationDestRepository) CreateDelivery(
	ctx context.Context,
	orgID string,
	eventID *string,
	destID, destType, recipient, eventType, idempotencyKey string,
	payload map[string]interface{},
) (*NotificationDelivery, error) {
	if payload == nil {
		payload = make(map[string]interface{})
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	var deliv NotificationDelivery
	var rawPayload []byte

	err = WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		query := `
			INSERT INTO notification_deliveries (
				organization_id, event_id, destination_id, destination_type,
				recipient, event_type, status, idempotency_key, payload, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, 'queued', $7, $8, NOW(), NOW())
			ON CONFLICT (idempotency_key) DO UPDATE
			SET updated_at = NOW()
			RETURNING id, organization_id, event_id, destination_id, destination_type,
			          recipient, event_type, status, retry_count, max_retries, next_attempt_at,
			          idempotency_key, payload, error_message, last_attempt_at, sent_at, created_at, updated_at
		`
		return tx.QueryRow(ctx, query,
			orgID, eventID, destID, destType, recipient, eventType, idempotencyKey, payloadBytes,
		).Scan(
			&deliv.ID, &deliv.OrganizationID, &deliv.EventID, &deliv.DestinationID,
			&deliv.DestinationType, &deliv.Recipient, &deliv.EventType, &deliv.Status,
			&deliv.RetryCount, &deliv.MaxRetries, &deliv.NextAttemptAt, &deliv.IdempotencyKey,
			&rawPayload, &deliv.ErrorMessage, &deliv.LastAttemptAt, &deliv.SentAt,
			&deliv.CreatedAt, &deliv.UpdatedAt,
		)
	})

	if err != nil {
		return nil, fmt.Errorf("failed queuing delivery: %w", err)
	}

	_ = json.Unmarshal(rawPayload, &deliv.Payload)
	return &deliv, nil
}

// DispatchPendingEvents finds pending notification_events and fans out to matching destinations.
func (r *NotificationDestRepository) DispatchPendingEvents(ctx context.Context, limit int) (int, error) {
	query := `
		SELECT ne.id, ne.organization_id, ne.event_type, ne.payload
		FROM notification_events ne
		WHERE ne.status = 'pending'
		ORDER BY ne.created_at ASC
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return 0, fmt.Errorf("failed querying pending events: %w", err)
	}
	defer rows.Close()

	type pendingEvent struct {
		ID        string
		OrgID     string
		EventType string
		Payload   map[string]interface{}
	}
	var events []pendingEvent

	for rows.Next() {
		var pe pendingEvent
		var rawPayload []byte
		if err := rows.Scan(&pe.ID, &pe.OrgID, &pe.EventType, &rawPayload); err != nil {
			return 0, err
		}
		_ = json.Unmarshal(rawPayload, &pe.Payload)
		events = append(events, pe)
	}
	rows.Close()

	if len(events) == 0 {
		return 0, nil
	}

	dispatchedCount := 0
	for _, ev := range events {
		err := WithTenantTx(ctx, r.pool, ev.OrgID, func(tx pgx.Tx) error {
			// Find all active destinations that have preference enabled for this event_type
			destRows, err := tx.Query(ctx, `
				SELECT nd.id, nd.destination_type,
				       COALESCE(nd.telegram_chat_id::text, nd.email_address, '') as recipient
				FROM notification_destinations nd
				JOIN notification_preferences np
				  ON np.destination_id = nd.id AND np.event_type = $2 AND np.is_enabled = true
				WHERE nd.organization_id = $1
				  AND nd.status = 'active'
				  AND nd.is_enabled = true
			`, ev.OrgID, ev.EventType)
			if err != nil {
				return err
			}
			defer destRows.Close()

			type targetDest struct {
				ID        string
				Type      string
				Recipient string
			}
			var targets []targetDest
			for destRows.Next() {
				var td targetDest
				if err := destRows.Scan(&td.ID, &td.Type, &td.Recipient); err != nil {
					return err
				}
				if td.Recipient != "" {
					targets = append(targets, td)
				}
			}
			destRows.Close()

			payloadBytes, _ := json.Marshal(ev.Payload)
			for _, td := range targets {
				idempKey := fmt.Sprintf("deliv:%s:%s", ev.ID, td.ID)
				_, err = tx.Exec(ctx, `
					INSERT INTO notification_deliveries (
						organization_id, event_id, destination_id, destination_type,
						recipient, event_type, status, idempotency_key, payload, created_at, updated_at
					)
					VALUES ($1, $2, $3, $4, $5, $6, 'queued', $7, $8, NOW(), NOW())
					ON CONFLICT (idempotency_key) DO NOTHING
				`, ev.OrgID, ev.ID, td.ID, td.Type, td.Recipient, ev.EventType, idempKey, payloadBytes)
				if err != nil {
					return err
				}
			}

			newStatus := "delivered"
			if len(targets) == 0 {
				newStatus = "suppressed"
			}
			_, err = tx.Exec(ctx, `
				UPDATE notification_events
				SET status = $1
				WHERE id = $2 AND organization_id = $3
			`, newStatus, ev.ID, ev.OrgID)
			return err
		})

		if err == nil {
			dispatchedCount++
		}
	}

	return dispatchedCount, nil
}

// ClaimDueDeliveries claims due notification deliveries across tenants using FOR UPDATE SKIP LOCKED.
func (r *NotificationDestRepository) ClaimDueDeliveries(ctx context.Context, workerID string, limit int) ([]*NotificationDelivery, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, organization_id, event_id, destination_id, destination_type,
		       recipient, event_type, status, retry_count, max_retries, payload
		FROM claim_due_notification_deliveries($1, $2)
	`, workerID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed claiming due deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []*NotificationDelivery
	for rows.Next() {
		var d NotificationDelivery
		var rawPayload []byte
		if err := rows.Scan(
			&d.ID, &d.OrganizationID, &d.EventID, &d.DestinationID, &d.DestinationType,
			&d.Recipient, &d.EventType, &d.Status, &d.RetryCount, &d.MaxRetries, &rawPayload,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(rawPayload, &d.Payload)
		deliveries = append(deliveries, &d)
	}

	return deliveries, nil
}

// MarkDeliverySent updates delivery to 'sent'.
func (r *NotificationDestRepository) MarkDeliverySent(ctx context.Context, deliveryID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE notification_deliveries
		SET status = 'sent',
		    sent_at = NOW(),
		    error_message = NULL,
		    updated_at = NOW()
		WHERE id = $1
	`, deliveryID)
	return err
}

// MarkDeliveryFailed updates delivery to 'retrying' with backoff or terminal 'failed'.
func (r *NotificationDestRepository) MarkDeliveryFailed(
	ctx context.Context,
	deliveryID string,
	errMsg string,
	retryAfter time.Duration,
	permanent bool,
) error {
	if permanent {
		_, err := r.pool.Exec(ctx, `
			UPDATE notification_deliveries
			SET status = 'failed',
			    error_message = $2,
			    updated_at = NOW()
			WHERE id = $1
		`, deliveryID, errMsg)
		return err
	}

	nextAttempt := time.Now().UTC().Add(retryAfter)
	_, err := r.pool.Exec(ctx, `
		UPDATE notification_deliveries
		SET status = CASE WHEN retry_count + 1 >= max_retries THEN 'failed' ELSE 'retrying' END,
		    retry_count = retry_count + 1,
		    next_attempt_at = $2,
		    error_message = $3,
		    updated_at = NOW()
		WHERE id = $1
	`, deliveryID, nextAttempt, errMsg)
	return err
}

// RequeueStaleDeliveries re-queues any deliveries stuck in 'sending' for longer than the timeout.
func (r *NotificationDestRepository) RequeueStaleDeliveries(ctx context.Context, staleThreshold time.Duration) (int64, error) {
	threshold := time.Now().UTC().Add(-staleThreshold)
	ct, err := r.pool.Exec(ctx, `
		UPDATE notification_deliveries
		SET status = 'retrying',
		    updated_at = NOW()
		WHERE status = 'sending' AND last_attempt_at < $1
	`, threshold)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

// ListDeliveries retrieves paginated delivery history for an organization.
func (r *NotificationDestRepository) ListDeliveries(
	ctx context.Context,
	orgID string,
	limit, offset int,
	statusFilter, destIDFilter string,
) ([]*NotificationDelivery, int, error) {
	var deliveries []*NotificationDelivery
	var totalCount int

	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		countQuery := `
			SELECT COUNT(*)
			FROM notification_deliveries
			WHERE organization_id = $1
			  AND ($2 = '' OR status = $2)
			  AND ($3 = '' OR destination_id = $3::uuid)
		`
		err := tx.QueryRow(ctx, countQuery, orgID, statusFilter, destIDFilter).Scan(&totalCount)
		if err != nil {
			return err
		}

		query := `
			SELECT id, organization_id, event_id, destination_id, destination_type,
			       recipient, event_type, status, retry_count, max_retries, next_attempt_at,
			       idempotency_key, payload, error_message, last_attempt_at, sent_at, created_at, updated_at
			FROM notification_deliveries
			WHERE organization_id = $1
			  AND ($2 = '' OR status = $2)
			  AND ($3 = '' OR destination_id = $3::uuid)
			ORDER BY created_at DESC
			LIMIT $4 OFFSET $5
		`
		rows, err := tx.Query(ctx, query, orgID, statusFilter, destIDFilter, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var d NotificationDelivery
			var rawPayload []byte
			if err := rows.Scan(
				&d.ID, &d.OrganizationID, &d.EventID, &d.DestinationID,
				&d.DestinationType, &d.Recipient, &d.EventType, &d.Status,
				&d.RetryCount, &d.MaxRetries, &d.NextAttemptAt, &d.IdempotencyKey,
				&rawPayload, &d.ErrorMessage, &d.LastAttemptAt, &d.SentAt,
				&d.CreatedAt, &d.UpdatedAt,
			); err != nil {
				return err
			}
			_ = json.Unmarshal(rawPayload, &d.Payload)
			deliveries = append(deliveries, &d)
		}
		return nil
	})

	if err != nil {
		return nil, 0, fmt.Errorf("failed listing deliveries: %w", err)
	}

	return deliveries, totalCount, nil
}
