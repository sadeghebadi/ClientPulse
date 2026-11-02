-- Migration: 000009_notifications_telegram_email.down.sql
-- Description: Rollback notification infrastructure

DROP FUNCTION IF EXISTS claim_due_notification_deliveries(VARCHAR, INT);
DROP FUNCTION IF EXISTS consume_notification_connection_token(VARCHAR);

DROP TABLE IF EXISTS notification_deliveries CASCADE;
DROP TABLE IF EXISTS notification_connection_tokens CASCADE;
DROP TABLE IF EXISTS notification_preferences CASCADE;
DROP TABLE IF EXISTS notification_destinations CASCADE;

-- Revert notification_events constraint
ALTER TABLE notification_events DROP CONSTRAINT IF EXISTS chk_notif_events_type;
ALTER TABLE notification_events ADD CONSTRAINT chk_notif_events_type
    CHECK (event_type IN ('incident.opened', 'incident.acknowledged', 'incident.resolved', 'heartbeat.missed', 'heartbeat.recovered'));

