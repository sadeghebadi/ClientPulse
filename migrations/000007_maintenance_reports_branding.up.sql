-- ==============================================================================
-- Migration: 000007_maintenance_reports_branding.up.sql
-- Description: Maintenance activities, reports & snapshots, branding, share tokens,
--              audit entity indexing, and first-deadline heartbeat claim update.
-- Phase: 4 (Maintenance Activity, Reports, PDF & White Label)
-- ==============================================================================

-- 1. Indexing on existing audit_logs for entity lookups
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity ON audit_logs (organization_id, entity_type, entity_id);

-- 2. Maintenance Activities
CREATE TABLE IF NOT EXISTS maintenance_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    client_id UUID NOT NULL,
    asset_id UUID NULL,
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(50) NOT NULL,
    performed_at TIMESTAMPTZ NOT NULL,
    duration_minutes INT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_maintenance_category CHECK (category IN (
        'update', 'backup', 'bug_fix', 'performance',
        'security_review', 'content_change', 'configuration', 'other'
    )),
    CONSTRAINT chk_maintenance_duration CHECK (duration_minutes IS NULL OR (duration_minutes >= 0 AND duration_minutes <= 14400)),
    CONSTRAINT uq_maintenance_activities_org_id UNIQUE (organization_id, id),
    CONSTRAINT fk_maintenance_client_org FOREIGN KEY (organization_id, client_id) 
        REFERENCES clients(organization_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_maintenance_asset_org FOREIGN KEY (organization_id, asset_id) 
        REFERENCES assets(organization_id, id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_maintenance_org_client_perf ON maintenance_activities (organization_id, client_id, performed_at DESC);
CREATE INDEX IF NOT EXISTS idx_maintenance_org_asset_perf ON maintenance_activities (organization_id, asset_id, performed_at DESC);
CREATE INDEX IF NOT EXISTS idx_maintenance_org_cat ON maintenance_activities (organization_id, category);

ALTER TABLE maintenance_activities ENABLE ROW LEVEL SECURITY;
ALTER TABLE maintenance_activities FORCE ROW LEVEL SECURITY;

CREATE POLICY maintenance_activities_tenant_isolation ON maintenance_activities
    FOR ALL
    USING (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID)
    WITH CHECK (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID);

-- 3. Organization Branding
CREATE TABLE IF NOT EXISTS organization_branding (
    organization_id UUID PRIMARY KEY REFERENCES organizations(id) ON DELETE CASCADE,
    display_name VARCHAR(150) NOT NULL DEFAULT '',
    logo_storage_key VARCHAR(255) NULL,
    primary_color VARCHAR(30) NOT NULL DEFAULT '#0ea5e9',
    accent_color VARCHAR(30) NOT NULL DEFAULT '#6366f1',
    report_footer_text TEXT NOT NULL DEFAULT '',
    contact_email VARCHAR(255) NOT NULL DEFAULT '',
    website_url VARCHAR(255) NOT NULL DEFAULT '',
    default_report_locale VARCHAR(10) NOT NULL DEFAULT 'en',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_branding_locale CHECK (default_report_locale IN ('en', 'fa')),
    CONSTRAINT chk_branding_primary_color CHECK (primary_color ~ '^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$'),
    CONSTRAINT chk_branding_accent_color CHECK (accent_color ~ '^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$')
);

ALTER TABLE organization_branding ENABLE ROW LEVEL SECURITY;
ALTER TABLE organization_branding FORCE ROW LEVEL SECURITY;

CREATE POLICY organization_branding_tenant_isolation ON organization_branding
    FOR ALL
    USING (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID)
    WITH CHECK (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID);

-- 4. Reports & Persistent Immutable Snapshots
CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    client_id UUID NOT NULL,
    report_type VARCHAR(50) NOT NULL DEFAULT 'monthly_care',
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    organization_timezone VARCHAR(64) NOT NULL DEFAULT 'UTC',
    locale VARCHAR(10) NOT NULL DEFAULT 'en',
    version_number INT NOT NULL DEFAULT 1,
    template_version VARCHAR(20) NOT NULL DEFAULT 'v1.0',
    title VARCHAR(255) NOT NULL,
    snapshot_json JSONB NOT NULL,
    pdf_storage_key VARCHAR(255) NULL,
    generated_by UUID NOT NULL REFERENCES users(id),
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_reports_locale CHECK (locale IN ('en', 'fa')),
    CONSTRAINT chk_reports_period CHECK (period_end > period_start),
    CONSTRAINT uq_reports_org_id UNIQUE (organization_id, id),
    CONSTRAINT uq_reports_org_client_period_version UNIQUE (organization_id, client_id, period_start, period_end, version_number),
    CONSTRAINT fk_reports_client_org FOREIGN KEY (organization_id, client_id) 
        REFERENCES clients(organization_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_reports_org_client_gen ON reports (organization_id, client_id, generated_at DESC);
CREATE INDEX IF NOT EXISTS idx_reports_org_period ON reports (organization_id, period_start, period_end);

ALTER TABLE reports ENABLE ROW LEVEL SECURITY;
ALTER TABLE reports FORCE ROW LEVEL SECURITY;

CREATE POLICY reports_tenant_isolation ON reports
    FOR ALL
    USING (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID)
    WITH CHECK (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID);

-- 5. Report Share Tokens (Public Read-Only Access via Hashed Tokens)
CREATE TABLE IF NOT EXISTS report_share_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    report_id UUID NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NULL,
    revoked_at TIMESTAMPTZ NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_report_share_report_org FOREIGN KEY (organization_id, report_id)
        REFERENCES reports(organization_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_report_share_tokens_lookup ON report_share_tokens (token_hash) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_report_share_tokens_org_report ON report_share_tokens (organization_id, report_id);

ALTER TABLE report_share_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE report_share_tokens FORCE ROW LEVEL SECURITY;

CREATE POLICY report_share_tokens_tenant_isolation ON report_share_tokens
    FOR ALL
    USING (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID)
    WITH CHECK (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::UUID);

-- 6. SECURITY DEFINER Procedure for Public Report Share Resolution
CREATE OR REPLACE FUNCTION lookup_report_by_share_token_hash(p_token_hash VARCHAR)
RETURNS TABLE (
    share_id UUID,
    report_id UUID,
    organization_id UUID,
    client_id UUID,
    report_type VARCHAR,
    period_start TIMESTAMPTZ,
    period_end TIMESTAMPTZ,
    organization_timezone VARCHAR,
    locale VARCHAR,
    version_number INT,
    template_version VARCHAR,
    title VARCHAR,
    snapshot_json JSONB,
    pdf_storage_key VARCHAR,
    generated_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
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
    SELECT 
        st.id AS share_id,
        r.id AS report_id,
        r.organization_id,
        r.client_id,
        r.report_type,
        r.period_start,
        r.period_end,
        r.organization_timezone,
        r.locale,
        r.version_number,
        r.template_version,
        r.title,
        r.snapshot_json,
        r.pdf_storage_key,
        r.generated_at,
        st.expires_at,
        st.revoked_at
    FROM public.report_share_tokens st
    JOIN public.reports r ON r.id = st.report_id AND r.organization_id = st.organization_id
    WHERE st.token_hash = p_token_hash
      AND st.revoked_at IS NULL
      AND (st.expires_at IS NULL OR st.expires_at > NOW())
    LIMIT 1;
END;
$$;

ALTER FUNCTION lookup_report_by_share_token_hash(VARCHAR) OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION lookup_report_by_share_token_hash(VARCHAR) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION lookup_report_by_share_token_hash(VARCHAR) TO clientpulse_app;

-- Definer permissions to query reports and share tokens under SECURITY DEFINER
GRANT SELECT ON public.report_share_tokens TO clientpulse_definer;
GRANT SELECT ON public.reports TO clientpulse_definer;

-- Definer RLS policies for report share resolution
DROP POLICY IF EXISTS report_share_definer_policy ON report_share_tokens;
CREATE POLICY report_share_definer_policy ON report_share_tokens
    FOR SELECT TO clientpulse_definer USING (true);

DROP POLICY IF EXISTS reports_definer_policy ON reports;
CREATE POLICY reports_definer_policy ON reports
    FOR SELECT TO clientpulse_definer USING (true);

-- 7. Update claim_missed_heartbeats Stored Procedure to include first-deadline 'pending' heartbeats
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
          AND hm.status IN ('healthy', 'late', 'pending')
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

-- 8. Runtime Role Grants for application
GRANT SELECT, INSERT, UPDATE, DELETE ON public.maintenance_activities TO clientpulse_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.organization_branding TO clientpulse_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.reports TO clientpulse_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.report_share_tokens TO clientpulse_app;
