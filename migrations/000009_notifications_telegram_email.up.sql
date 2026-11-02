-- Migration: 000009_notifications_telegram_email.up.sql
-- Description: Notification infrastructure supporting Telegram Bot and Email multi-tenant delivery

-- 1. Extend notification_events event_type constraint to support report.ready
ALTER TABLE notification_events DROP CONSTRAINT IF EXISTS chk_notif_events_type;
ALTER TABLE notification_events ADD CONSTRAINT chk_notif_events_type
    CHECK (event_type IN ('incident.opened', 'incident.acknowledged', 'incident.resolved', 'heartbeat.missed', 'heartbeat.recovered', 'report.ready'));

-- 2. Notification Destinations Table
CREATE TABLE IF NOT EXISTS notification_destinations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    destination_type VARCHAR(32) NOT NULL,
    name VARCHAR(255) NOT NULL,
    telegram_chat_id BIGINT NULL,
    telegram_user_id BIGINT NULL,
    telegram_username VARCHAR(255) NULL,
    email_address VARCHAR(255) NULL,
    user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_verified_at TIMESTAMPTZ NULL,
    CONSTRAINT chk_notif_dest_type CHECK (destination_type IN ('telegram_private', 'telegram_group', 'email')),
    CONSTRAINT chk_notif_dest_status CHECK (status IN ('active', 'disconnected', 'pending')),
    CONSTRAINT uq_notif_dest_org_id UNIQUE (organization_id, id)
);

CREATE INDEX IF NOT EXISTS idx_notif_dest_org_status ON notification_destinations (organization_id, status);
CREATE INDEX IF NOT EXISTS idx_notif_dest_telegram_chat ON notification_destinations (organization_id, telegram_chat_id) WHERE telegram_chat_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_notif_dest_email ON notification_destinations (organization_id, LOWER(email_address)) WHERE email_address IS NOT NULL;

-- 3. Notification Preferences Table (Per-destination event filtering)
CREATE TABLE IF NOT EXISTS notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    destination_id UUID NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_notif_pref_dest FOREIGN KEY (organization_id, destination_id)
        REFERENCES notification_destinations(organization_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_notif_pref_dest_event UNIQUE (destination_id, event_type),
    CONSTRAINT chk_notif_pref_event CHECK (event_type IN ('incident.opened', 'incident.acknowledged', 'incident.resolved', 'heartbeat.missed', 'heartbeat.recovered', 'report.ready'))
);

CREATE INDEX IF NOT EXISTS idx_notif_pref_dest ON notification_preferences (destination_id);
CREATE INDEX IF NOT EXISTS idx_notif_pref_org ON notification_preferences (organization_id);

-- 4. Single-Use Connection Tokens Table (For Telegram private and group chat binding)
CREATE TABLE IF NOT EXISTS notification_connection_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    destination_type VARCHAR(32) NOT NULL,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    label VARCHAR(255) NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_notif_token_type CHECK (destination_type IN ('telegram_private', 'telegram_group'))
);

CREATE INDEX IF NOT EXISTS idx_notif_tokens_hash ON notification_connection_tokens (token_hash) WHERE consumed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_notif_tokens_org ON notification_connection_tokens (organization_id, created_at DESC);

-- 5. Notification Deliveries Table (Durable delivery queue and auditable history)
CREATE TABLE IF NOT EXISTS notification_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    event_id UUID NULL REFERENCES notification_events(id) ON DELETE SET NULL,
    destination_id UUID NOT NULL,
    destination_type VARCHAR(32) NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'queued',
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 5,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT NULL,
    last_attempt_at TIMESTAMPTZ NULL,
    sent_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_notif_deliv_dest FOREIGN KEY (organization_id, destination_id)
        REFERENCES notification_destinations(organization_id, id) ON DELETE CASCADE,
    CONSTRAINT chk_notif_deliv_status CHECK (status IN ('queued', 'sending', 'sent', 'retrying', 'failed', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_notif_deliv_queue ON notification_deliveries (status, next_attempt_at) WHERE status IN ('queued', 'retrying');
CREATE INDEX IF NOT EXISTS idx_notif_deliv_org_created ON notification_deliveries (organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notif_deliv_event ON notification_deliveries (event_id);

-- 6. Row-Level Security (RLS) Enforcement
ALTER TABLE notification_destinations ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_destinations FORCE ROW LEVEL SECURITY;

ALTER TABLE notification_preferences ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_preferences FORCE ROW LEVEL SECURITY;

ALTER TABLE notification_connection_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_connection_tokens FORCE ROW LEVEL SECURITY;

ALTER TABLE notification_deliveries ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_deliveries FORCE ROW LEVEL SECURITY;

-- Tenant Isolation Policies for Application Context
CREATE POLICY notif_dest_tenant_isolation ON notification_destinations
    FOR ALL
    USING (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid)
    WITH CHECK (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid);

CREATE POLICY notif_pref_tenant_isolation ON notification_preferences
    FOR ALL
    USING (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid)
    WITH CHECK (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid);

CREATE POLICY notif_tokens_tenant_isolation ON notification_connection_tokens
    FOR ALL
    USING (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid)
    WITH CHECK (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid);

CREATE POLICY notif_deliv_tenant_isolation ON notification_deliveries
    FOR ALL
    USING (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid)
    WITH CHECK (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid);

-- Cross-Tenant Policies for Background Worker & Definer Roles
CREATE POLICY notif_events_worker_policy ON notification_events
    FOR ALL
    TO clientpulse_worker
    USING (true)
    WITH CHECK (true);

CREATE POLICY notif_dest_worker_policy ON notification_destinations
    FOR SELECT
    TO clientpulse_worker
    USING (true);

CREATE POLICY notif_pref_worker_policy ON notification_preferences
    FOR SELECT
    TO clientpulse_worker
    USING (true);

CREATE POLICY notif_deliv_worker_policy ON notification_deliveries
    FOR ALL
    TO clientpulse_worker
    USING (true)
    WITH CHECK (true);

CREATE POLICY notif_dest_definer_policy ON notification_destinations
    FOR ALL
    TO clientpulse_definer
    USING (true)
    WITH CHECK (true);

CREATE POLICY notif_pref_definer_policy ON notification_preferences
    FOR ALL
    TO clientpulse_definer
    USING (true)
    WITH CHECK (true);

CREATE POLICY notif_tokens_definer_policy ON notification_connection_tokens
    FOR ALL
    TO clientpulse_definer
    USING (true)
    WITH CHECK (true);

CREATE POLICY notif_deliv_definer_policy ON notification_deliveries
    FOR ALL
    TO clientpulse_definer
    USING (true)
    WITH CHECK (true);

-- 7. Security Definer Functions

-- Atomic consumption of single-use connection token
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
SET search_path = public
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

-- Atomic claim of due notification deliveries with FOR UPDATE SKIP LOCKED
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
SET search_path = public
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

-- Function owners and permissions
ALTER FUNCTION consume_notification_connection_token(VARCHAR) OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION consume_notification_connection_token(VARCHAR) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION consume_notification_connection_token(VARCHAR) TO clientpulse_app;
GRANT EXECUTE ON FUNCTION consume_notification_connection_token(VARCHAR) TO clientpulse_worker;

ALTER FUNCTION claim_due_notification_deliveries(VARCHAR, INT) OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION claim_due_notification_deliveries(VARCHAR, INT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION claim_due_notification_deliveries(VARCHAR, INT) TO clientpulse_app;
GRANT EXECUTE ON FUNCTION claim_due_notification_deliveries(VARCHAR, INT) TO clientpulse_worker;

-- 8. Permissions on Tables and Sequences
GRANT SELECT, INSERT, UPDATE, DELETE ON public.notification_destinations TO clientpulse_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.notification_preferences TO clientpulse_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.notification_connection_tokens TO clientpulse_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.notification_deliveries TO clientpulse_app;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.notification_destinations TO clientpulse_worker;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.notification_preferences TO clientpulse_worker;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.notification_connection_tokens TO clientpulse_worker;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.notification_deliveries TO clientpulse_worker;

GRANT SELECT, INSERT, UPDATE, DELETE ON public.notification_destinations TO clientpulse_definer;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.notification_preferences TO clientpulse_definer;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.notification_connection_tokens TO clientpulse_definer;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.notification_deliveries TO clientpulse_definer;
