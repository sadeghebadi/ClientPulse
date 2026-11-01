-- Migration: 000004_clients_assets_monitoring.up.sql
-- Description: Core schema for Clients, Website/API Assets, Monitors, and Check Results with Row-Level Security

-- 1. Clients table
CREATE TABLE IF NOT EXISTS clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    contact_name VARCHAR(255) NULL,
    contact_email VARCHAR(255) NULL,
    website VARCHAR(512) NULL,
    notes TEXT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ NULL,
    CONSTRAINT chk_clients_status CHECK (status IN ('active', 'archived'))
);

CREATE INDEX IF NOT EXISTS idx_clients_org_status ON clients (organization_id, status);
CREATE INDEX IF NOT EXISTS idx_clients_org_created ON clients (organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_clients_org_name ON clients (organization_id, LOWER(name));

-- 2. Assets table
CREATE TABLE IF NOT EXISTS assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(1024) NOT NULL,
    type VARCHAR(32) NOT NULL DEFAULT 'website',
    environment VARCHAR(32) NOT NULL DEFAULT 'production',
    description TEXT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ NULL,
    CONSTRAINT chk_assets_type CHECK (type IN ('website', 'api')),
    CONSTRAINT chk_assets_environment CHECK (environment IN ('production', 'staging', 'development'))
);

CREATE INDEX IF NOT EXISTS idx_assets_org_client ON assets (organization_id, client_id);
CREATE INDEX IF NOT EXISTS idx_assets_org_active ON assets (organization_id, is_active);
CREATE INDEX IF NOT EXISTS idx_assets_org_url ON assets (organization_id, LOWER(url));
CREATE INDEX IF NOT EXISTS idx_assets_org_created ON assets (organization_id, created_at DESC);

-- 3. Monitors table
CREATE TABLE IF NOT EXISTS monitors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    method VARCHAR(16) NOT NULL DEFAULT 'GET',
    url_override VARCHAR(1024) NULL,
    interval_seconds INT NOT NULL DEFAULT 300,
    timeout_seconds INT NOT NULL DEFAULT 10,
    expected_status_min INT NOT NULL DEFAULT 200,
    expected_status_max INT NOT NULL DEFAULT 399,
    follow_redirects BOOLEAN NOT NULL DEFAULT true,
    max_redirects INT NOT NULL DEFAULT 5,
    expected_text TEXT NULL,
    json_pointer VARCHAR(255) NULL,
    json_expected_value TEXT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    is_active BOOLEAN NOT NULL DEFAULT true,
    next_check_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_checked_at TIMESTAMPTZ NULL,
    lease_acquired_at TIMESTAMPTZ NULL,
    lease_worker_id VARCHAR(255) NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_monitors_method CHECK (method IN ('GET', 'HEAD')),
    CONSTRAINT chk_monitors_interval CHECK (interval_seconds >= 60),
    CONSTRAINT chk_monitors_timeout CHECK (timeout_seconds BETWEEN 3 AND 30),
    CONSTRAINT chk_monitors_status_min CHECK (expected_status_min >= 100 AND expected_status_min <= 599),
    CONSTRAINT chk_monitors_status_max CHECK (expected_status_max >= 100 AND expected_status_max <= 599 AND expected_status_max >= expected_status_min),
    CONSTRAINT chk_monitors_max_redirects CHECK (max_redirects BETWEEN 0 AND 10),
    CONSTRAINT chk_monitors_status CHECK (status IN ('pending', 'up', 'down', 'paused'))
);

CREATE INDEX IF NOT EXISTS idx_monitors_org_asset ON monitors (organization_id, asset_id);
CREATE INDEX IF NOT EXISTS idx_monitors_org_status ON monitors (organization_id, status);
CREATE INDEX IF NOT EXISTS idx_monitors_scheduler ON monitors (next_check_at ASC) 
    WHERE is_active = true AND status != 'paused';

-- 4. Check Results table
CREATE TABLE IF NOT EXISTS check_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    scheduled_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL,
    latency_ms BIGINT NOT NULL,
    result VARCHAR(32) NOT NULL,
    http_status_code INT NULL,
    error_category VARCHAR(64) NULL,
    error_message TEXT NULL,
    tls_version VARCHAR(32) NULL,
    tls_cipher VARCHAR(64) NULL,
    tls_cert_issuer VARCHAR(255) NULL,
    tls_cert_subject VARCHAR(255) NULL,
    tls_cert_not_before TIMESTAMPTZ NULL,
    tls_cert_not_after TIMESTAMPTZ NULL,
    tls_days_remaining INT NULL,
    tls_is_valid BOOLEAN NULL,
    resolved_ip VARCHAR(64) NULL,
    worker_id VARCHAR(255) NOT NULL,
    execution_type VARCHAR(32) NOT NULL DEFAULT 'scheduled',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_check_results_result CHECK (result IN ('success', 'failure', 'timeout', 'blocked', 'error')),
    CONSTRAINT chk_check_results_type CHECK (execution_type IN ('scheduled', 'manual'))
);

CREATE INDEX IF NOT EXISTS idx_check_results_monitor_created ON check_results (monitor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_check_results_org_created ON check_results (organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_check_results_retention ON check_results (created_at);

-- 5. Atomic scheduler claim function (Security Definer for cross-tenant worker claiming)
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
AS $$
BEGIN
    RETURN QUERY
    WITH due_monitors AS (
        SELECT m.id AS m_id, m.next_check_at AS m_sched
        FROM monitors m
        JOIN assets a ON m.asset_id = a.id
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
        UPDATE monitors m
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
    JOIN assets a ON u.asset_id = a.id;
END;
$$;

-- 6. Row-Level Security (RLS) enforcement
ALTER TABLE clients ENABLE ROW LEVEL SECURITY;
ALTER TABLE clients FORCE ROW LEVEL SECURITY;

CREATE POLICY clients_tenant_isolation ON clients
    FOR ALL
    USING (
        organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid
    )
    WITH CHECK (
        organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid
    );

ALTER TABLE assets ENABLE ROW LEVEL SECURITY;
ALTER TABLE assets FORCE ROW LEVEL SECURITY;

CREATE POLICY assets_tenant_isolation ON assets
    FOR ALL
    USING (
        organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid
    )
    WITH CHECK (
        organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid
    );

ALTER TABLE monitors ENABLE ROW LEVEL SECURITY;
ALTER TABLE monitors FORCE ROW LEVEL SECURITY;

CREATE POLICY monitors_tenant_isolation ON monitors
    FOR ALL
    USING (
        organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid
    )
    WITH CHECK (
        organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid
    );

ALTER TABLE check_results ENABLE ROW LEVEL SECURITY;
ALTER TABLE check_results FORCE ROW LEVEL SECURITY;

CREATE POLICY check_results_tenant_isolation ON check_results
    FOR ALL
    USING (
        organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid
    )
    WITH CHECK (
        organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid
    );

-- 7. Application Runtime Role Grants
GRANT SELECT, INSERT, UPDATE, DELETE ON clients, assets, monitors, check_results TO clientpulse_app;
GRANT EXECUTE ON FUNCTION claim_due_monitors(VARCHAR, INT, INT) TO clientpulse_app;
