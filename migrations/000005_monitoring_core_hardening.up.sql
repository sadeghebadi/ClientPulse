-- Migration: 000005_monitoring_core_hardening.up.sql
-- Description: Security hardening for Worker DB roles, SECURITY DEFINER claiming, composite tenant foreign keys, consecutive failures, and hourly aggregations

-- 1. Create dedicated database roles
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clientpulse_definer') THEN
        CREATE ROLE clientpulse_definer NOLOGIN;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clientpulse_worker') THEN
        CREATE ROLE clientpulse_worker WITH LOGIN PASSWORD 'clientpulse_worker_dev_password'
            NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
    END IF;
END $$;

-- 2. Schema security: revoke CREATE in public schema to prevent object tampering
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
REVOKE CREATE ON SCHEMA public FROM clientpulse_app;
REVOKE CREATE ON SCHEMA public FROM clientpulse_worker;
REVOKE CREATE ON SCHEMA public FROM clientpulse_definer;
GRANT USAGE ON SCHEMA public TO clientpulse_app, clientpulse_worker, clientpulse_definer;

-- 3. Composite Tenant Foreign Keys
-- Ensure no existing cross-tenant inconsistencies exist
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM assets a
        JOIN clients c ON a.client_id = c.id
        WHERE a.organization_id != c.organization_id
    ) THEN
        RAISE EXCEPTION 'Data integrity violation: asset organization does not match client organization';
    END IF;

    IF EXISTS (
        SELECT 1 FROM monitors m
        JOIN assets a ON m.asset_id = a.id
        WHERE m.organization_id != a.organization_id
    ) THEN
        RAISE EXCEPTION 'Data integrity violation: monitor organization does not match asset organization';
    END IF;

    IF EXISTS (
        SELECT 1 FROM check_results cr
        JOIN monitors m ON cr.monitor_id = m.id
        WHERE cr.organization_id != m.organization_id
    ) THEN
        RAISE EXCEPTION 'Data integrity violation: check_result organization does not match monitor organization';
    END IF;
END $$;

-- Composite UNIQUE constraints on parent tables (organization_id, id)
ALTER TABLE clients ADD CONSTRAINT uq_clients_org_id UNIQUE (organization_id, id);
ALTER TABLE assets ADD CONSTRAINT uq_assets_org_id UNIQUE (organization_id, id);
ALTER TABLE monitors ADD CONSTRAINT uq_monitors_org_id UNIQUE (organization_id, id);

-- Replace simple FKs with composite foreign keys
ALTER TABLE assets DROP CONSTRAINT assets_client_id_fkey;
ALTER TABLE assets ADD CONSTRAINT fk_assets_client_org 
    FOREIGN KEY (organization_id, client_id) REFERENCES clients(organization_id, id) ON DELETE CASCADE;

ALTER TABLE monitors DROP CONSTRAINT monitors_asset_id_fkey;
ALTER TABLE monitors ADD CONSTRAINT fk_monitors_asset_org 
    FOREIGN KEY (organization_id, asset_id) REFERENCES assets(organization_id, id) ON DELETE CASCADE;

ALTER TABLE check_results DROP CONSTRAINT check_results_monitor_id_fkey;
ALTER TABLE check_results ADD CONSTRAINT fk_check_results_monitor_org 
    FOREIGN KEY (organization_id, monitor_id) REFERENCES monitors(organization_id, id) ON DELETE CASCADE;

-- 4. Monitor State Transitions & Consecutive Failures
ALTER TABLE monitors ADD COLUMN IF NOT EXISTS consecutive_failures INT NOT NULL DEFAULT 0;

ALTER TABLE monitors DROP CONSTRAINT IF EXISTS chk_monitors_status;
ALTER TABLE monitors ADD CONSTRAINT chk_monitors_status 
    CHECK (status IN ('pending', 'up', 'suspected_down', 'down', 'paused', 'unknown'));

-- 5. Hourly Check Results Aggregates Table
CREATE TABLE IF NOT EXISTS check_aggregates_hourly (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    bucket_hour TIMESTAMPTZ NOT NULL,
    total_checks INT NOT NULL DEFAULT 0,
    successful_checks INT NOT NULL DEFAULT 0,
    failed_checks INT NOT NULL DEFAULT 0,
    total_latency_ms BIGINT NOT NULL DEFAULT 0,
    min_latency_ms BIGINT NOT NULL DEFAULT 0,
    max_latency_ms BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_check_aggregates_hourly UNIQUE (organization_id, monitor_id, bucket_hour),
    CONSTRAINT fk_check_aggregates_monitor_org 
        FOREIGN KEY (organization_id, monitor_id) REFERENCES monitors(organization_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_check_aggregates_monitor_bucket ON check_aggregates_hourly (monitor_id, bucket_hour DESC);
CREATE INDEX IF NOT EXISTS idx_check_aggregates_org_bucket ON check_aggregates_hourly (organization_id, bucket_hour DESC);

ALTER TABLE check_aggregates_hourly ENABLE ROW LEVEL SECURITY;
ALTER TABLE check_aggregates_hourly FORCE ROW LEVEL SECURITY;

CREATE POLICY check_aggregates_tenant_isolation ON check_aggregates_hourly
    FOR ALL
    USING (
        organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid
    )
    WITH CHECK (
        organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid
    );

-- 6. Hardened claim_due_monitors function
GRANT SELECT, UPDATE ON public.monitors TO clientpulse_definer;
GRANT SELECT ON public.assets TO clientpulse_definer;

CREATE OR REPLACE FUNCTION claim_due_monitors(
    p_worker_id VARCHAR,
    p_batch_size INT,
    p_lease_timeout_seconds INT DEFAULT 60
)
RETURNS TABLE (
    id UUID,
    organization_id UUID,
    asset_id UUID,
    target_url VARCHAR,
    method VARCHAR,
    interval_seconds INT,
    timeout_seconds INT,
    expected_status_min INT,
    expected_status_max INT,
    follow_redirects BOOLEAN,
    max_redirects INT,
    expected_text TEXT,
    json_pointer VARCHAR,
    json_expected_value TEXT,
    scheduled_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public, pg_temp
AS $$
BEGIN
    -- Validate p_batch_size
    IF p_batch_size IS NULL OR p_batch_size < 1 OR p_batch_size > 100 THEN
        RAISE EXCEPTION 'p_batch_size must be between 1 and 100 (got %)', p_batch_size;
    END IF;

    -- Validate p_lease_timeout_seconds
    IF p_lease_timeout_seconds IS NULL OR p_lease_timeout_seconds < 5 OR p_lease_timeout_seconds > 3600 THEN
        RAISE EXCEPTION 'p_lease_timeout_seconds must be between 5 and 3600 (got %)', p_lease_timeout_seconds;
    END IF;

    -- Validate worker id
    IF p_worker_id IS NULL OR length(trim(p_worker_id)) = 0 THEN
        RAISE EXCEPTION 'p_worker_id cannot be empty';
    END IF;

    RETURN QUERY
    WITH due_monitors AS (
        SELECT m.id AS m_id, m.next_check_at AS m_sched
        FROM public.monitors m
        JOIN public.assets a ON m.asset_id = a.id
        WHERE m.is_active = true
          AND a.is_active = true
          AND m.status != 'paused'
          AND m.next_check_at <= NOW()
          AND (
              m.lease_acquired_at IS NULL
              OR m.lease_acquired_at < NOW() - (p_lease_timeout_seconds || ' seconds')::INTERVAL
          )
        ORDER BY m.next_check_at ASC
        LIMIT p_batch_size
        FOR UPDATE OF m SKIP LOCKED
    ),
    updated AS (
        UPDATE public.monitors m
        SET lease_acquired_at = NOW(),
            lease_worker_id = p_worker_id
        FROM due_monitors dm
        WHERE m.id = dm.m_id
        RETURNING m.id, m.organization_id, m.asset_id, m.method, m.url_override,
                  m.interval_seconds, m.timeout_seconds, m.expected_status_min,
                  m.expected_status_max, m.follow_redirects, m.max_redirects,
                  m.expected_text, m.json_pointer, m.json_expected_value, dm.m_sched
    )
    SELECT u.id, u.organization_id, u.asset_id,
           COALESCE(u.url_override, a.url)::VARCHAR AS target_url,
           u.method, u.interval_seconds, u.timeout_seconds,
           u.expected_status_min, u.expected_status_max,
           u.follow_redirects, u.max_redirects,
           u.expected_text, u.json_pointer, u.json_expected_value,
           u.m_sched AS scheduled_at
    FROM updated u
    JOIN public.assets a ON u.asset_id = a.id;
END;
$$;

-- Transfer function ownership to non-login restricted definer role
ALTER FUNCTION claim_due_monitors(VARCHAR, INT, INT) OWNER TO clientpulse_definer;

-- Revoke execute from PUBLIC and API role
REVOKE ALL ON FUNCTION claim_due_monitors(VARCHAR, INT, INT) FROM PUBLIC;
REVOKE ALL ON FUNCTION claim_due_monitors(VARCHAR, INT, INT) FROM clientpulse_app;

-- Grant EXECUTE exclusively to clientpulse_worker
GRANT EXECUTE ON FUNCTION claim_due_monitors(VARCHAR, INT, INT) TO clientpulse_worker;

-- 7. Configure Worker Role Table Permissions
GRANT SELECT, UPDATE ON public.monitors TO clientpulse_worker;
GRANT SELECT ON public.assets TO clientpulse_worker;
GRANT INSERT, SELECT, DELETE ON public.check_results TO clientpulse_worker;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.check_aggregates_hourly TO clientpulse_worker;
GRANT SELECT, INSERT, UPDATE ON public.worker_heartbeats TO clientpulse_worker;

-- Worker role must not have access to user credentials or modify audit logs
REVOKE ALL ON public.audit_logs FROM clientpulse_worker;
REVOKE ALL ON public.users FROM clientpulse_worker;
REVOKE ALL ON public.sessions FROM clientpulse_worker;
REVOKE ALL ON public.password_reset_tokens FROM clientpulse_worker;
REVOKE ALL ON public.email_verification_tokens FROM clientpulse_worker;

-- Grant app role access to read and update aggregates
GRANT SELECT, INSERT, UPDATE ON public.check_aggregates_hourly TO clientpulse_app;

-- 8. Worker-specific cross-tenant RLS policies for background cleanup & aggregation
CREATE POLICY check_results_worker_cleanup_select ON check_results
    FOR SELECT
    TO clientpulse_worker
    USING (true);

CREATE POLICY check_results_worker_cleanup_delete ON check_results
    FOR DELETE
    TO clientpulse_worker
    USING (true);

CREATE POLICY check_aggregates_worker_cleanup_select ON check_aggregates_hourly
    FOR SELECT
    TO clientpulse_worker
    USING (true);

CREATE POLICY check_aggregates_worker_cleanup_delete ON check_aggregates_hourly
    FOR DELETE
    TO clientpulse_worker
    USING (true);
