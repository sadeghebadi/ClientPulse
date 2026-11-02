-- Migration: 000010_notifications_hardening.down.sql

DROP FUNCTION IF EXISTS consume_email_verification_token(VARCHAR);

DROP TABLE IF EXISTS telegram_webhook_updates;

ALTER TABLE notification_deliveries DROP COLUMN IF EXISTS delivery_attempts;

ALTER TABLE notification_deliveries DROP CONSTRAINT IF EXISTS chk_notif_deliv_status;
ALTER TABLE notification_deliveries ADD CONSTRAINT chk_notif_deliv_status
    CHECK (status IN ('queued', 'sending', 'sent', 'retrying', 'failed', 'cancelled'));

DROP INDEX IF EXISTS idx_notif_dest_verification_hash;
DROP INDEX IF EXISTS uq_active_telegram_group;

ALTER TABLE notification_destinations
    DROP COLUMN IF EXISTS verified_at,
    DROP COLUMN IF EXISTS verification_token_hash,
    DROP COLUMN IF EXISTS verification_sent_at,
    DROP COLUMN IF EXISTS pending_group_approval,
    DROP COLUMN IF EXISTS pending_group_chat_id,
    DROP COLUMN IF EXISTS pending_group_title,
    DROP COLUMN IF EXISTS authorized_by_user_id;

ALTER TABLE notification_destinations DROP CONSTRAINT IF EXISTS chk_notif_dest_status;
ALTER TABLE notification_destinations ADD CONSTRAINT chk_notif_dest_status
    CHECK (status IN ('active', 'disconnected', 'pending'));

