-- Migration: 000018_platform_admin_and_subscriptions.up.sql
-- Description: Platform Admin grants, subscription plans, versioning, organization subscriptions, platform audit logs, and SECURITY DEFINER analytics

-- 1. Platform Admin Grants table (Explicit global administrative authorizations)
CREATE TABLE IF NOT EXISTS platform_admin_grants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(32) NOT NULL DEFAULT 'platform_admin',
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    revoked_at TIMESTAMPTZ NULL,
    revoked_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    reason TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_platform_admin_role CHECK (role IN ('platform_admin'))
);

CREATE INDEX IF NOT EXISTS idx_platform_admin_grants_user_id ON platform_admin_grants (user_id);
CREATE INDEX IF NOT EXISTS idx_platform_admin_grants_active ON platform_admin_grants (user_id) WHERE revoked_at IS NULL;

-- 2. Subscription Plans table
CREATE TABLE IF NOT EXISTS subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    updated_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT chk_plan_status CHECK (status IN ('draft', 'active', 'archived'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_subscription_plans_slug_lower ON subscription_plans (LOWER(slug));
CREATE INDEX IF NOT EXISTS idx_subscription_plans_status ON subscription_plans (status);

-- 3. Subscription Plan Versions table (Immutable versioning on publication)
CREATE TABLE IF NOT EXISTS subscription_plan_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL REFERENCES subscription_plans(id) ON DELETE CASCADE,
    version INT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    price_amount BIGINT NOT NULL DEFAULT 0, -- Minor currency units (e.g. cents/rials)
    currency VARCHAR(3) NOT NULL DEFAULT 'USD', -- ISO 4217 uppercase 3-letter code
    billing_interval VARCHAR(32) NOT NULL DEFAULT 'monthly', -- 'monthly', 'yearly', 'quarterly', 'one_time'
    max_clients INT NOT NULL DEFAULT -1, -- -1 = unlimited, 0 = disabled, > 0 = finite limit
    max_assets INT NOT NULL DEFAULT -1,
    max_monitors INT NOT NULL DEFAULT -1,
    max_heartbeats INT NOT NULL DEFAULT -1,
    max_members INT NOT NULL DEFAULT -1,
    min_check_interval_seconds INT NOT NULL DEFAULT 60,
    retention_days INT NOT NULL DEFAULT 30,
    custom_branding_enabled BOOLEAN NOT NULL DEFAULT false,
    public_reports_enabled BOOLEAN NOT NULL DEFAULT true,
    features JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ NULL,
    published_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT uq_plan_versions_plan_version UNIQUE (plan_id, version),
    CONSTRAINT chk_plan_version_status CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT chk_plan_version_currency CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_plan_version_billing_interval CHECK (billing_interval IN ('monthly', 'yearly', 'quarterly', 'one_time')),
    CONSTRAINT chk_plan_version_min_interval CHECK (min_check_interval_seconds >= 60)
);

CREATE INDEX IF NOT EXISTS idx_plan_versions_plan_id ON subscription_plan_versions (plan_id);
CREATE INDEX IF NOT EXISTS idx_plan_versions_status ON subscription_plan_versions (status);

-- 4. Organization Subscriptions table (Links tenant to exact plan version)
CREATE TABLE IF NOT EXISTS organization_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    plan_version_id UUID NOT NULL REFERENCES subscription_plan_versions(id) ON DELETE RESTRICT,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at TIMESTAMPTZ NULL,
    assigned_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    administrative_note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_org_subscription_status CHECK (status IN ('active', 'suspended', 'canceled', 'expired'))
);

CREATE INDEX IF NOT EXISTS idx_org_subscriptions_org_id ON organization_subscriptions (organization_id);
CREATE INDEX IF NOT EXISTS idx_org_subscriptions_plan_version_id ON organization_subscriptions (plan_version_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_org_subscriptions_active_unique ON organization_subscriptions (organization_id) WHERE status = 'active';

-- 5. Platform Admin Audit Logs table (Immutable security and admin action log)
CREATE TABLE IF NOT EXISTS platform_admin_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(64) NOT NULL,
    target_type VARCHAR(64) NOT NULL,
    target_id VARCHAR(64) NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip_address VARCHAR(45) NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_platform_admin_audit_actor ON platform_admin_audit_logs (actor_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_platform_admin_audit_target ON platform_admin_audit_logs (target_type, target_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_platform_admin_audit_action ON platform_admin_audit_logs (action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_platform_admin_audit_created ON platform_admin_audit_logs (created_at DESC);

-- 6. Permissions and Role Grants
GRANT SELECT, INSERT, UPDATE, DELETE ON platform_admin_grants, subscription_plans, subscription_plan_versions, organization_subscriptions TO clientpulse_app;
-- Audit logs: append-only (SELECT & INSERT). UPDATE and DELETE are strictly disallowed.
GRANT SELECT, INSERT ON platform_admin_audit_logs TO clientpulse_app;
REVOKE UPDATE, DELETE ON platform_admin_audit_logs FROM clientpulse_app;

-- 7. Grant clientpulse_definer read permissions for analytics functions
GRANT SELECT ON public.users, public.organizations, public.subscription_plans, public.subscription_plan_versions, public.organization_subscriptions TO clientpulse_definer;
GRANT SELECT ON public.clients TO clientpulse_definer;
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies 
        WHERE tablename = 'clients' AND policyname = 'clients_definer_policy'
    ) THEN
        CREATE POLICY clients_definer_policy ON public.clients FOR SELECT TO clientpulse_definer USING (true);
    END IF;
END $$;

GRANT SELECT ON public.organization_memberships TO clientpulse_definer;
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies 
        WHERE tablename = 'organization_memberships' AND policyname = 'org_memberships_definer_policy'
    ) THEN
        CREATE POLICY org_memberships_definer_policy ON public.organization_memberships FOR SELECT TO clientpulse_definer USING (true);
    END IF;
END $$;

-- 8. SECURITY DEFINER Analytics Functions (Owned by clientpulse_definer)

CREATE OR REPLACE FUNCTION get_platform_overview_stats()
RETURNS TABLE (
    total_users BIGINT,
    active_users BIGINT,
    suspended_users BIGINT,
    total_organizations BIGINT,
    active_organizations BIGINT,
    suspended_organizations BIGINT,
    published_plans BIGINT,
    active_subscriptions BIGINT,
    unassigned_organizations BIGINT
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public, pg_temp
AS $$
BEGIN
    RETURN QUERY
    SELECT
        (SELECT COUNT(*) FROM users)::BIGINT AS total_users,
        (SELECT COUNT(*) FROM users WHERE status = 'active')::BIGINT AS active_users,
        (SELECT COUNT(*) FROM users WHERE status = 'suspended')::BIGINT AS suspended_users,
        (SELECT COUNT(*) FROM organizations WHERE deleted_at IS NULL)::BIGINT AS total_organizations,
        (SELECT COUNT(*) FROM organizations WHERE status = 'active' AND deleted_at IS NULL)::BIGINT AS active_organizations,
        (SELECT COUNT(*) FROM organizations WHERE status = 'suspended' AND deleted_at IS NULL)::BIGINT AS suspended_organizations,
        (SELECT COUNT(DISTINCT plan_id) FROM subscription_plan_versions WHERE status = 'published')::BIGINT AS published_plans,
        (SELECT COUNT(*) FROM organization_subscriptions WHERE status = 'active')::BIGINT AS active_subscriptions,
        (SELECT COUNT(*) FROM organizations o WHERE o.deleted_at IS NULL AND NOT EXISTS (
            SELECT 1 FROM organization_subscriptions s WHERE s.organization_id = o.id AND s.status = 'active'
        ))::BIGINT AS unassigned_organizations;
END;
$$;

ALTER FUNCTION get_platform_overview_stats() OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION get_platform_overview_stats() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION get_platform_overview_stats() TO clientpulse_app;

CREATE OR REPLACE FUNCTION get_platform_organization_usage(p_org_id UUID)
RETURNS TABLE (
    client_count BIGINT,
    asset_count BIGINT,
    monitor_count BIGINT,
    heartbeat_count BIGINT,
    member_count BIGINT
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public, pg_temp
AS $$
BEGIN
    RETURN QUERY
    SELECT
        (SELECT COUNT(*) FROM clients WHERE organization_id = p_org_id AND archived_at IS NULL)::BIGINT AS client_count,
        (SELECT COUNT(*) FROM assets WHERE organization_id = p_org_id AND archived_at IS NULL)::BIGINT AS asset_count,
        (SELECT COUNT(*) FROM monitors WHERE organization_id = p_org_id)::BIGINT AS monitor_count,
        (SELECT COUNT(*) FROM heartbeat_monitors WHERE organization_id = p_org_id)::BIGINT AS heartbeat_count,
        (SELECT COUNT(*) FROM organization_memberships WHERE organization_id = p_org_id)::BIGINT AS member_count;
END;
$$;

ALTER FUNCTION get_platform_organization_usage(UUID) OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION get_platform_organization_usage(UUID) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION get_platform_organization_usage(UUID) TO clientpulse_app;

CREATE OR REPLACE FUNCTION get_platform_user_memberships(p_user_id UUID)
RETURNS TABLE (
    organization_id UUID,
    organization_name VARCHAR,
    organization_slug VARCHAR,
    organization_status VARCHAR,
    role VARCHAR,
    created_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public, pg_temp
AS $$
BEGIN
    RETURN QUERY
    SELECT
        o.id AS organization_id,
        o.name AS organization_name,
        o.slug AS organization_slug,
        o.status AS organization_status,
        m.role,
        m.created_at
    FROM organization_memberships m
    JOIN organizations o ON o.id = m.organization_id
    WHERE m.user_id = p_user_id AND o.deleted_at IS NULL
    ORDER BY m.created_at ASC;
END;
$$;

ALTER FUNCTION get_platform_user_memberships(UUID) OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION get_platform_user_memberships(UUID) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION get_platform_user_memberships(UUID) TO clientpulse_app;

CREATE OR REPLACE FUNCTION get_platform_org_members(p_org_id UUID)
RETURNS TABLE (
    user_id UUID,
    email VARCHAR,
    display_name VARCHAR,
    user_status VARCHAR,
    role VARCHAR,
    joined_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public, pg_temp
AS $$
BEGIN
    RETURN QUERY
    SELECT
        u.id AS user_id,
        u.email,
        u.display_name,
        u.status AS user_status,
        m.role,
        m.created_at AS joined_at
    FROM organization_memberships m
    JOIN users u ON u.id = m.user_id
    WHERE m.organization_id = p_org_id
    ORDER BY m.created_at ASC;
END;
$$;

ALTER FUNCTION get_platform_org_members(UUID) OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION get_platform_org_members(UUID) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION get_platform_org_members(UUID) TO clientpulse_app;

CREATE OR REPLACE FUNCTION get_platform_organizations_list(
    p_search TEXT,
    p_status TEXT,
    p_limit INT,
    p_offset INT
)
RETURNS TABLE (
    id UUID,
    name VARCHAR,
    slug VARCHAR,
    status VARCHAR,
    locale VARCHAR,
    timezone VARCHAR,
    created_at TIMESTAMPTZ,
    owner_id UUID,
    owner_email VARCHAR,
    owner_name VARCHAR,
    member_count BIGINT,
    client_count BIGINT,
    asset_count BIGINT,
    monitor_count BIGINT,
    heartbeat_count BIGINT,
    plan_name VARCHAR,
    plan_slug VARCHAR,
    plan_version INT,
    subscription_status VARCHAR,
    total_count BIGINT
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public, pg_temp
AS $$
DECLARE
    v_total BIGINT;
BEGIN
    SELECT COUNT(*)
    INTO v_total
    FROM organizations o
    WHERE o.deleted_at IS NULL
      AND (p_status IS NULL OR p_status = '' OR o.status = p_status)
      AND (p_search IS NULL OR p_search = '' OR o.name ILIKE '%' || p_search || '%' OR o.slug ILIKE '%' || p_search || '%');

    RETURN QUERY
    SELECT
        o.id,
        o.name,
        o.slug,
        o.status,
        o.locale,
        o.timezone,
        o.created_at,
        owner_u.id AS owner_id,
        owner_u.email AS owner_email,
        owner_u.display_name AS owner_name,
        COALESCE((SELECT COUNT(*) FROM organization_memberships om WHERE om.organization_id = o.id), 0)::BIGINT AS member_count,
        COALESCE((SELECT COUNT(*) FROM clients c WHERE c.organization_id = o.id AND c.archived_at IS NULL), 0)::BIGINT AS client_count,
        COALESCE((SELECT COUNT(*) FROM assets a WHERE a.organization_id = o.id AND a.archived_at IS NULL), 0)::BIGINT AS asset_count,
        COALESCE((SELECT COUNT(*) FROM monitors m WHERE m.organization_id = o.id), 0)::BIGINT AS monitor_count,
        COALESCE((SELECT COUNT(*) FROM heartbeat_monitors hm WHERE hm.organization_id = o.id), 0)::BIGINT AS heartbeat_count,
        p.name AS plan_name,
        p.slug AS plan_slug,
        pv.version AS plan_version,
        os.status AS subscription_status,
        v_total AS total_count
    FROM organizations o
    LEFT JOIN LATERAL (
        SELECT u.id, u.email, u.display_name
        FROM organization_memberships om
        JOIN users u ON u.id = om.user_id
        WHERE om.organization_id = o.id AND om.role = 'owner'
        LIMIT 1
    ) owner_u ON true
    LEFT JOIN LATERAL (
        SELECT s.plan_version_id, s.status
        FROM organization_subscriptions s
        WHERE s.organization_id = o.id AND s.status = 'active'
        ORDER BY s.started_at DESC
        LIMIT 1
    ) os ON true
    LEFT JOIN subscription_plan_versions pv ON pv.id = os.plan_version_id
    LEFT JOIN subscription_plans p ON p.id = pv.plan_id
    WHERE o.deleted_at IS NULL
      AND (p_status IS NULL OR p_status = '' OR o.status = p_status)
      AND (p_search IS NULL OR p_search = '' OR o.name ILIKE '%' || p_search || '%' OR o.slug ILIKE '%' || p_search || '%')
    ORDER BY o.created_at DESC
    LIMIT p_limit OFFSET p_offset;
END;
$$;

ALTER FUNCTION get_platform_organizations_list(TEXT, TEXT, INT, INT) OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION get_platform_organizations_list(TEXT, TEXT, INT, INT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION get_platform_organizations_list(TEXT, TEXT, INT, INT) TO clientpulse_app;
