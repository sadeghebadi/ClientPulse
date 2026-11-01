package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationEventRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationEventRepository(pool *pgxpool.Pool) *NotificationEventRepository {
	return &NotificationEventRepository{pool: pool}
}

// CreateEventTx creates an immutable notification event within an existing transaction.
// It uses idempotency_key to prevent duplicate events. Returns (event, created, err).
func (r *NotificationEventRepository) CreateEventTx(
	ctx context.Context,
	tx pgx.Tx,
	orgID, eventType string,
	incidentID, monitorID, heartbeatID, clientID *string,
	idempotencyKey string,
	payload map[string]interface{},
) (*NotificationEvent, bool, error) {
	if payload == nil {
		payload = make(map[string]interface{})
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, false, fmt.Errorf("failed to marshal notification event payload: %w", err)
	}

	query := `
		INSERT INTO notification_events (
			organization_id, event_type, incident_id, monitor_id, heartbeat_monitor_id,
			client_id, idempotency_key, payload, status, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending', NOW())
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING id, organization_id, event_type, incident_id, monitor_id, heartbeat_monitor_id,
		          client_id, idempotency_key, payload, status, created_at
	`

	var event NotificationEvent
	var rawPayload []byte
	err = tx.QueryRow(ctx, query,
		orgID, eventType, incidentID, monitorID, heartbeatID,
		clientID, idempotencyKey, payloadJSON,
	).Scan(
		&event.ID, &event.OrganizationID, &event.EventType, &event.IncidentID,
		&event.MonitorID, &event.HeartbeatMonitorID, &event.ClientID,
		&event.IdempotencyKey, &rawPayload, &event.Status, &event.CreatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			// Duplicate idempotency key - event was already created
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to insert notification event: %w", err)
	}

	_ = json.Unmarshal(rawPayload, &event.Payload)
	return &event, true, nil
}

// GetEventsByIncidentID retrieves all notification events for an incident in chronological order.
func (r *NotificationEventRepository) GetEventsByIncidentID(ctx context.Context, orgID, incidentID string) ([]*NotificationEvent, error) {
	var events []*NotificationEvent

	err := WithTenantTx(ctx, r.pool, orgID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, organization_id, event_type, incident_id, monitor_id, heartbeat_monitor_id,
			       client_id, idempotency_key, payload, status, created_at
			FROM notification_events
			WHERE incident_id = $1 AND organization_id = $2
			ORDER BY created_at ASC
		`, incidentID, orgID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var ev NotificationEvent
			var rawPayload []byte
			if err := rows.Scan(
				&ev.ID, &ev.OrganizationID, &ev.EventType, &ev.IncidentID,
				&ev.MonitorID, &ev.HeartbeatMonitorID, &ev.ClientID,
				&ev.IdempotencyKey, &rawPayload, &ev.Status, &ev.CreatedAt,
			); err != nil {
				return err
			}
			_ = json.Unmarshal(rawPayload, &ev.Payload)
			events = append(events, &ev)
		}
		return rows.Err()
	})

	if err != nil {
		return nil, err
	}
	return events, nil
}
