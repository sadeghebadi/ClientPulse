-- Migration: 000020_locale_settings_and_localized_catalogs.down.sql

ALTER TABLE organization_subscriptions DROP COLUMN IF EXISTS offer_version_id;

DROP TRIGGER IF EXISTS trg_prevent_offer_version_mutation ON subscription_offer_versions;
DROP FUNCTION IF EXISTS prevent_offer_version_mutation();

DROP TABLE IF EXISTS subscription_offer_versions;
DROP TABLE IF EXISTS subscription_offers;
DROP TABLE IF EXISTS platform_localization_settings;

DROP FUNCTION IF EXISTS get_platform_overview_stats();

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

