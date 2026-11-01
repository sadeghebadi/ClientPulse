-- Migration: 000006_incident_heartbeat_monitoring.up.sql
-- Description: Incident management, Heartbeat monitoring, and persistent notification events with strict composite foreign keys, RLS, and security definer claiming

-- 1. Prerequisites: Composite constraints on existing tables for strict tenant validation
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_assets_org_client_id'
    ) THEN
        ALTER TABLE assets ADD CONSTRAINT uq_assets_org_client_id UNIQUE (organization_id, client_id, id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'uq_check_results_org_id'
    ) THEN
        ALTER TABLE check_results ADD CONSTRAINT uq_check_results_org_id UNIQUE (organization_id, id);
    END IF;
END $$;

-- 2. Heartbeat Monitors Table
CREATE TABLE IF NOT EXISTS heartbeat_monitors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    client_id UUID NOT NULL,
    asset_id UUID NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NULL,
    expected_interval_seconds INT NOT NULL DEFAULT 3600,
    grace_period_seconds INT NOT NULL DEFAULT 720,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    is_active BOOLEAN NOT NULL DEFAULT true,
    token_hash VARCHAR(64) NOT NULL,
    last_received_at TIMESTAMPTZ NULL,
    next_expected_at TIMESTAMPTZ NULL,
    lease_acquired_at TIMESTAMPTZ NULL,
    lease_worker_id VARCHAR(64) NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ NULL,
    CONSTRAINT chk_heartbeats_interval CHECK (expected_interval_seconds >= 60 AND expected_interval_seconds <= 2592000),
    CONSTRAINT chk_heartbeats_grace CHECK (grace_period_seconds >= 60 AND grace_period_seconds <= 86400),
    CONSTRAINT chk_heartbeats_status CHECK (status IN ('pending', 'healthy', 'late', 'missed', 'paused', 'revoked')),
    CONSTRAINT uq_heartbeat_monitors_org_id UNIQUE (organization_id, id),
    CONSTRAINT uq_heartbeat_monitors_token_hash UNIQUE (token_hash),
    CONSTRAINT fk_heartbeats_client_org FOREIGN KEY (organization_id, client_id) REFERENCES clients(organization_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_heartbeats_asset_org FOREIGN KEY (organization_id, client_id, asset_id) REFERENCES assets(organization_id, client_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_heartbeats_org_status ON heartbeat_monitors (organization_id, status);
CREATE INDEX IF NOT EXISTS idx_heartbeats_org_client ON heartbeat_monitors (organization_id, client_id);
CREATE INDEX IF NOT EXISTS idx_heartbeats_org_created ON heartbeat_monitors (organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_heartbeats_next_expected ON heartbeat_monitors (next_expected_at)
    WHERE is_active = true AND status NOT IN ('paused', 'revoked') AND archived_at IS NULL;

-- 3. Incidents Table
CREATE TABLE IF NOT EXISTS incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    monitor_id UUID NULL,
    heartbeat_monitor_id UUID NULL,
    first_failure_at TIMESTAMPTZ NOT NULL,
    confirmed_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'open',
    failure_category VARCHAR(64) NOT NULL DEFAULT 'unknown',
    first_failure_result_id UUID NULL,
    recovery_result_id UUID NULL,
    acknowledged_at TIMESTAMPTZ NULL,
    acknowledged_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    notification_state VARCHAR(32) NOT NULL DEFAULT 'pending',
    duration_seconds INT NULL,
    summary TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_incident_status CHECK (status IN ('open', 'resolved')),
    CONSTRAINT chk_incident_single_target CHECK (
        (monitor_id IS NOT NULL AND heartbeat_monitor_id IS NULL) OR
        (monitor_id IS NULL AND heartbeat_monitor_id IS NOT NULL)
    ),
    CONSTRAINT uq_incidents_org_id UNIQUE (organization_id, id),
    CONSTRAINT fk_incidents_monitor_org FOREIGN KEY (organization_id, monitor_id) REFERENCES monitors(organization_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_incidents_heartbeat_org FOREIGN KEY (organization_id, heartbeat_monitor_id) REFERENCES heartbeat_monitors(organization_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_incidents_first_result_org FOREIGN KEY (organization_id, first_failure_result_id) REFERENCES check_results(organization_id, id) ON DELETE SET NULL,
    CONSTRAINT fk_incidents_recovery_result_org FOREIGN KEY (organization_id, recovery_result_id) REFERENCES check_results(organization_id, id) ON DELETE SET NULL
);

-- Guarantee exactly one open/unresolved incident per target at database constraint level
CREATE UNIQUE INDEX IF NOT EXISTS uq_open_incident_monitor ON incidents (monitor_id) WHERE status = 'open';
CREATE UNIQUE INDEX IF NOT EXISTS uq_open_incident_heartbeat ON incidents (heartbeat_monitor_id) WHERE status = 'open';

CREATE INDEX IF NOT EXISTS idx_incidents_org_status ON incidents (organization_id, status);
CREATE INDEX IF NOT EXISTS idx_incidents_org_created ON incidents (organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_incidents_monitor_created ON incidents (monitor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_incidents_heartbeat_created ON incidents (heartbeat_monitor_id, created_at DESC);

-- 4. Notification Events Table (Persistent, Append-Only)
CREATE TABLE IF NOT EXISTS notification_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    event_type VARCHAR(64) NOT NULL,
    incident_id UUID NULL,
    monitor_id UUID NULL,
    heartbeat_monitor_id UUID NULL,
    client_id UUID NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_notif_events_type CHECK (event_type IN ('incident.opened', 'incident.acknowledged', 'incident.resolved', 'heartbeat.missed', 'heartbeat.recovered')),
    CONSTRAINT chk_notif_events_status CHECK (status IN ('pending', 'delivered', 'failed', 'suppressed')),
    CONSTRAINT uq_notif_events_idempotency UNIQUE (idempotency_key),
    CONSTRAINT fk_notif_events_incident_org FOREIGN KEY (organization_id, incident_id) REFERENCES incidents(organization_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_notif_events_org_created ON notification_events (organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notif_events_pending ON notification_events (status, created_at ASC) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_notif_events_incident ON notification_events (incident_id);

-- 5. Row-Level Security (RLS) Enforcement
ALTER TABLE heartbeat_monitors ENABLE ROW LEVEL SECURITY;
ALTER TABLE heartbeat_monitors FORCE ROW LEVEL SECURITY;

CREATE POLICY heartbeat_monitors_tenant_isolation ON heartbeat_monitors
    FOR ALL
    USING (
        organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid
    )
    WITH CHECK (
        organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid
    );

-- Allow clientpulse_definer role to select and claim across tenants for security definer functions
CREATE POLICY heartbeat_monitors_definer_policy ON heartbeat_monitors
    FOR ALL
    TO clientpulse_definer
    USING (true)
    WITH CHECK (true);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies WHERE policyname = 'monitors_definer_policy' AND tablename = 'monitors'
    ) THEN
        CREATE POLICY monitors_definer_policy ON monitors
            FOR ALL
            TO clientpulse_definer
            USING (true)
            WITH CHECK (true);
    END IF;
END $$;

ALTER TABLE incidents ENABLE ROW LEVEL SECURITY;
ALTER TABLE incidents FORCE ROW LEVEL SECURITY;

CREATE POLICY incidents_tenant_isolation ON incidents
    FOR ALL
    USING (
        organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid
    )
    WITH CHECK (
        organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid
    );

ALTER TABLE notification_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_events FORCE ROW LEVEL SECURITY;

CREATE POLICY notification_events_tenant_isolation ON notification_events
    FOR ALL
    USING (
        organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid
    )
    WITH CHECK (
        organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid
    );

-- 6. Atomic Heartbeat Claiming Stored Procedure (SECURITY DEFINER)
CREATE OR REPLACE FUNCTION claim_missed_heartbeats(
    p_worker_id VARCHAR,
    p_batch_size INT,
    p_lease_timeout_seconds INT DEFAULT 60
)
RETURNS TABLE (
    id UUID,
    organization_id UUID,
    client_id UUID,
    asset_id UUID,
    name VARCHAR,
    expected_interval_seconds INT,
    grace_period_seconds INT,
    status VARCHAR,
    last_received_at TIMESTAMPTZ,
    next_expected_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public, pg_temp
AS $$
BEGIN
    IF p_batch_size IS NULL OR p_batch_size < 1 OR p_batch_size > 100 THEN
        RAISE EXCEPTION 'p_batch_size must be between 1 and 100 (got %)', p_batch_size;
    END IF;

    IF p_lease_timeout_seconds IS NULL OR p_lease_timeout_seconds < 5 OR p_lease_timeout_seconds > 3600 THEN
        RAISE EXCEPTION 'p_lease_timeout_seconds must be between 5 and 3600 (got %)', p_lease_timeout_seconds;
    END IF;

    IF p_worker_id IS NULL OR length(trim(p_worker_id)) = 0 THEN
        RAISE EXCEPTION 'p_worker_id cannot be empty';
    END IF;

    RETURN QUERY
    WITH overdue_heartbeats AS (
        SELECT hm.id AS hm_id
        FROM public.heartbeat_monitors hm
        WHERE hm.is_active = true
          AND hm.archived_at IS NULL
          AND hm.status IN ('healthy', 'late')
          AND hm.next_expected_at IS NOT NULL
          AND hm.next_expected_at <= NOW()
          AND (
              hm.lease_acquired_at IS NULL
              OR hm.lease_acquired_at < NOW() - (p_lease_timeout_seconds || ' seconds')::INTERVAL
          )
        ORDER BY hm.next_expected_at ASC
        LIMIT p_batch_size
        FOR UPDATE OF hm SKIP LOCKED
    ),
    claimed AS (
        UPDATE public.heartbeat_monitors hm
        SET lease_acquired_at = NOW(),
            lease_worker_id = p_worker_id,
            status = 'missed',
            updated_at = NOW()
        FROM overdue_heartbeats oh
        WHERE hm.id = oh.hm_id
        RETURNING hm.id, hm.organization_id, hm.client_id, hm.asset_id, hm.name,
                  hm.expected_interval_seconds, hm.grace_period_seconds, hm.status,
                  hm.last_received_at, hm.next_expected_at
    )
    SELECT c.id, c.organization_id, c.client_id, c.asset_id, c.name,
           c.expected_interval_seconds, c.grace_period_seconds, c.status,
           c.last_received_at, c.next_expected_at
    FROM claimed c;
END;
$$;

-- Security Definer ownership and execution permissions
ALTER FUNCTION claim_missed_heartbeats(VARCHAR, INT, INT) OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION claim_missed_heartbeats(VARCHAR, INT, INT) FROM PUBLIC;
REVOKE ALL ON FUNCTION claim_missed_heartbeats(VARCHAR, INT, INT) FROM clientpulse_app;
GRANT EXECUTE ON FUNCTION claim_missed_heartbeats(VARCHAR, INT, INT) TO clientpulse_worker;

-- 7. Unauthenticated Token Lookup Stored Procedure (SECURITY DEFINER)
-- Allows resolving heartbeat target and tenant ID safely without leaking tenant data
CREATE OR REPLACE FUNCTION lookup_heartbeat_by_token_hash(p_token_hash VARCHAR)
RETURNS TABLE (
    id UUID,
    organization_id UUID,
    token_hash VARCHAR,
    status VARCHAR,
    archived_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public, pg_temp
AS $$
BEGIN
    IF p_token_hash IS NULL OR length(trim(p_token_hash)) = 0 THEN
        RETURN;
    END IF;

    RETURN QUERY
    SELECT hm.id, hm.organization_id, hm.token_hash, hm.status, hm.archived_at
    FROM public.heartbeat_monitors hm
    WHERE hm.token_hash = p_token_hash
    LIMIT 1;
END;
$$;

ALTER FUNCTION lookup_heartbeat_by_token_hash(VARCHAR) OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION lookup_heartbeat_by_token_hash(VARCHAR) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION lookup_heartbeat_by_token_hash(VARCHAR) TO clientpulse_app;

-- Grant definer role table access so SECURITY DEFINER functions can select/update heartbeat_monitors
GRANT SELECT, UPDATE ON public.heartbeat_monitors TO clientpulse_definer;

-- 8. Grant Permissions to Application and Worker Runtime Roles
GRANT SELECT, INSERT, UPDATE, DELETE ON public.heartbeat_monitors TO clientpulse_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.incidents TO clientpulse_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.notification_events TO clientpulse_app;

GRANT SELECT, UPDATE ON public.heartbeat_monitors TO clientpulse_worker;
GRANT SELECT, INSERT, UPDATE ON public.incidents TO clientpulse_worker;
GRANT SELECT, INSERT, UPDATE ON public.notification_events TO clientpulse_worker;
