-- Migration: 000010_notifications_hardening.up.sql
-- Description: Notification security and delivery reliability hardening

-- 1. Update notification_destinations status check to include 'pending_approval'
ALTER TABLE notification_destinations DROP CONSTRAINT IF EXISTS chk_notif_dest_status;
ALTER TABLE notification_destinations ADD CONSTRAINT chk_notif_dest_status
    CHECK (status IN ('active', 'disconnected', 'pending', 'pending_approval'));

-- 2. Add email verification and group authorization columns
ALTER TABLE notification_destinations
    ADD COLUMN IF NOT EXISTS verified_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS verification_token_hash VARCHAR(255) NULL,
    ADD COLUMN IF NOT EXISTS verification_sent_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS pending_group_approval BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS pending_group_chat_id BIGINT NULL,
    ADD COLUMN IF NOT EXISTS pending_group_title VARCHAR(255) NULL,
    ADD COLUMN IF NOT EXISTS authorized_by_user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL;

-- 3. Unique partial index to prevent cross-tenant group chat hijacking
DELETE FROM notification_destinations a 
USING notification_destinations b 
WHERE a.telegram_chat_id = b.telegram_chat_id 
  AND a.destination_type = 'telegram_group' 
  AND a.created_at < b.created_at;

CREATE UNIQUE INDEX IF NOT EXISTS uq_active_telegram_group
    ON notification_destinations (telegram_chat_id)
    WHERE destination_type = 'telegram_group' AND status IN ('active', 'pending_approval') AND telegram_chat_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_notif_dest_verification_hash
    ON notification_destinations (verification_token_hash)
    WHERE verification_token_hash IS NOT NULL;

-- 4. Update notification_deliveries status check to include 'uncertain'
ALTER TABLE notification_deliveries DROP CONSTRAINT IF EXISTS chk_notif_deliv_status;
ALTER TABLE notification_deliveries ADD CONSTRAINT chk_notif_deliv_status
    CHECK (status IN ('queued', 'sending', 'sent', 'retrying', 'failed', 'cancelled', 'uncertain'));

-- 5. Add delivery_attempts JSONB audit array to notification_deliveries
ALTER TABLE notification_deliveries
    ADD COLUMN IF NOT EXISTS delivery_attempts JSONB NOT NULL DEFAULT '[]'::jsonb;

-- 6. Telegram Webhook Replay Protection Table
CREATE TABLE IF NOT EXISTS telegram_webhook_updates (
    update_id BIGINT PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for pruning old updates if needed
CREATE INDEX IF NOT EXISTS idx_tg_webhook_updates_processed ON telegram_webhook_updates (processed_at);

-- Permissions for telegram_webhook_updates
GRANT SELECT, INSERT ON telegram_webhook_updates TO clientpulse_app;
GRANT SELECT, INSERT ON telegram_webhook_updates TO clientpulse_worker;
GRANT SELECT, INSERT ON telegram_webhook_updates TO clientpulse_definer;

-- 7. Secure Definer Functions with hardened search_path = public, pg_temp

CREATE OR REPLACE FUNCTION consume_notification_connection_token(
    p_token_hash VARCHAR
)
RETURNS TABLE (
    id UUID,
    organization_id UUID,
    user_id UUID,
    destination_type VARCHAR,
    label VARCHAR
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    RETURN QUERY
    UPDATE public.notification_connection_tokens nct
    SET consumed_at = NOW()
    WHERE nct.token_hash = p_token_hash
      AND nct.consumed_at IS NULL
      AND nct.expires_at > NOW()
    RETURNING nct.id, nct.organization_id, nct.user_id, nct.destination_type, nct.label;
END;
$$;

ALTER FUNCTION consume_notification_connection_token(VARCHAR) OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION consume_notification_connection_token(VARCHAR) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION consume_notification_connection_token(VARCHAR) TO clientpulse_app;
GRANT EXECUTE ON FUNCTION consume_notification_connection_token(VARCHAR) TO clientpulse_worker;

CREATE OR REPLACE FUNCTION claim_due_notification_deliveries(
    p_worker_id VARCHAR,
    p_limit INT DEFAULT 20
)
RETURNS TABLE (
    id UUID,
    organization_id UUID,
    event_id UUID,
    destination_id UUID,
    destination_type VARCHAR,
    recipient VARCHAR,
    event_type VARCHAR,
    status VARCHAR,
    retry_count INT,
    max_retries INT,
    payload JSONB
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    RETURN QUERY
    WITH candidate AS (
        SELECT nd.id
        FROM public.notification_deliveries nd
        WHERE nd.status IN ('queued', 'retrying')
          AND nd.next_attempt_at <= NOW()
        ORDER BY nd.next_attempt_at ASC
        LIMIT p_limit
        FOR UPDATE SKIP LOCKED
    )
    UPDATE public.notification_deliveries nd
    SET status = 'sending',
        last_attempt_at = NOW(),
        updated_at = NOW()
    FROM candidate
    WHERE nd.id = candidate.id
    RETURNING
        nd.id,
        nd.organization_id,
        nd.event_id,
        nd.destination_id,
        nd.destination_type,
        nd.recipient,
        nd.event_type,
        nd.status,
        nd.retry_count,
        nd.max_retries,
        nd.payload;
END;
$$;

ALTER FUNCTION claim_due_notification_deliveries(VARCHAR, INT) OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION claim_due_notification_deliveries(VARCHAR, INT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION claim_due_notification_deliveries(VARCHAR, INT) TO clientpulse_app;
GRANT EXECUTE ON FUNCTION claim_due_notification_deliveries(VARCHAR, INT) TO clientpulse_worker;

-- 8. Atomic Email Verification Consumption Function
CREATE OR REPLACE FUNCTION consume_email_verification_token(
    p_token_hash VARCHAR
)
RETURNS TABLE (
    id UUID,
    organization_id UUID,
    email_address VARCHAR
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    RETURN QUERY
    UPDATE public.notification_destinations nd
    SET status = 'active',
        verified_at = NOW(),
        last_verified_at = NOW(),
        verification_token_hash = NULL,
        updated_at = NOW()
    WHERE nd.verification_token_hash = p_token_hash
      AND nd.status = 'pending'
      AND nd.destination_type = 'email'
      AND nd.verification_sent_at > (NOW() - INTERVAL '24 hours')
    RETURNING nd.id, nd.organization_id, nd.email_address;
END;
$$;

ALTER FUNCTION consume_email_verification_token(VARCHAR) OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION consume_email_verification_token(VARCHAR) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION consume_email_verification_token(VARCHAR) TO clientpulse_app;
GRANT EXECUTE ON FUNCTION consume_email_verification_token(VARCHAR) TO clientpulse_worker;

-- 9. Cross-Tenant Group Chat Lookup Function (Security Definer)
CREATE OR REPLACE FUNCTION find_active_group_destination_by_chat_id(
    p_chat_id BIGINT
)
RETURNS TABLE (
    id UUID,
    organization_id UUID,
    destination_type VARCHAR,
    name VARCHAR,
    telegram_chat_id BIGINT,
    telegram_user_id BIGINT,
    telegram_username VARCHAR,
    email_address VARCHAR,
    user_id UUID,
    status VARCHAR,
    is_enabled BOOLEAN,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    last_verified_at TIMESTAMPTZ,
    verified_at TIMESTAMPTZ,
    verification_sent_at TIMESTAMPTZ,
    pending_group_approval BOOLEAN,
    pending_group_chat_id BIGINT,
    pending_group_title VARCHAR,
    authorized_by_user_id UUID
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    RETURN QUERY
    SELECT nd.id, nd.organization_id, nd.destination_type, nd.name,
           nd.telegram_chat_id, nd.telegram_user_id, nd.telegram_username,
           nd.email_address, nd.user_id, nd.status, nd.is_enabled,
           nd.created_at, nd.updated_at, nd.last_verified_at, nd.verified_at,
           nd.verification_sent_at, nd.pending_group_approval,
           nd.pending_group_chat_id, nd.pending_group_title, nd.authorized_by_user_id
    FROM public.notification_destinations nd
    WHERE nd.telegram_chat_id = p_chat_id
      AND nd.destination_type = 'telegram_group'
      AND nd.status IN ('active', 'pending_approval')
    LIMIT 1;
END;
$$;

ALTER FUNCTION find_active_group_destination_by_chat_id(BIGINT) OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION find_active_group_destination_by_chat_id(BIGINT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION find_active_group_destination_by_chat_id(BIGINT) TO clientpulse_app;
GRANT EXECUTE ON FUNCTION find_active_group_destination_by_chat_id(BIGINT) TO clientpulse_worker;
