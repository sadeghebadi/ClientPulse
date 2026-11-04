-- Migration: 000020_locale_settings_and_localized_catalogs.up.sql
-- Description: Centralized language activation settings, locale-aware subscription offers, immutable offer versions, and Toman monetary support.

-- 1. Centralized Platform Localization Settings
CREATE TABLE IF NOT EXISTS platform_localization_settings (
    id INT PRIMARY KEY DEFAULT 1,
    enabled_locales TEXT[] NOT NULL DEFAULT ARRAY['en', 'fa']::TEXT[],
    default_locale VARCHAR(16) NOT NULL DEFAULT 'en',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT chk_localization_singleton CHECK (id = 1),
    CONSTRAINT chk_at_least_one_enabled CHECK (cardinality(enabled_locales) > 0),
    CONSTRAINT chk_default_in_enabled CHECK (default_locale = ANY(enabled_locales))
);

INSERT INTO platform_localization_settings (id, enabled_locales, default_locale)
VALUES (1, ARRAY['en', 'fa']::TEXT[], 'en')
ON CONFLICT (id) DO NOTHING;

-- 2. Localized Subscription Offers (Bound to a base plan and target locale/market)
CREATE TABLE IF NOT EXISTS subscription_offers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL REFERENCES subscription_plans(id) ON DELETE CASCADE,
    locale VARCHAR(16) NOT NULL,
    market VARCHAR(32) NOT NULL DEFAULT 'global',
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    features_summary JSONB NOT NULL DEFAULT '[]'::jsonb,
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    updated_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT uq_plan_locale_market UNIQUE (plan_id, locale, market),
    CONSTRAINT chk_offer_status CHECK (status IN ('draft', 'active', 'archived')),
    CONSTRAINT chk_offer_locale CHECK (locale IN ('en', 'fa'))
);

CREATE INDEX IF NOT EXISTS idx_subscription_offers_plan ON subscription_offers (plan_id);
CREATE INDEX IF NOT EXISTS idx_subscription_offers_locale ON subscription_offers (locale, market);
CREATE INDEX IF NOT EXISTS idx_subscription_offers_status ON subscription_offers (status);

-- 3. Localized Subscription Offer Versions (Immutable published snapshots with exact monetary data)
CREATE TABLE IF NOT EXISTS subscription_offer_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    offer_id UUID NOT NULL REFERENCES subscription_offers(id) ON DELETE CASCADE,
    plan_version_id UUID NULL REFERENCES subscription_plan_versions(id) ON DELETE SET NULL,
    version INT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    price_amount BIGINT NOT NULL DEFAULT 0, -- Canonical stored monetary amount in minor units
    currency VARCHAR(3) NOT NULL DEFAULT 'USD', -- ISO 4217 uppercase 3-letter code (e.g. USD, IRR)
    display_denomination VARCHAR(32) NOT NULL DEFAULT 'standard', -- e.g. 'usd', 'toman', 'rial', 'standard'
    denomination_ratio BIGINT NOT NULL DEFAULT 1, -- Conversion factor: 1 display unit = N canonical units (e.g. 1 Toman = 10 IRR)
    billing_interval VARCHAR(32) NOT NULL DEFAULT 'monthly', -- 'monthly', 'yearly', 'quarterly', 'one_time'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ NULL,
    published_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT uq_offer_version UNIQUE (offer_id, version),
    CONSTRAINT chk_offer_version_status CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT chk_offer_version_currency CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_offer_version_price CHECK (price_amount >= 0),
    CONSTRAINT chk_offer_version_ratio CHECK (denomination_ratio >= 1),
    CONSTRAINT chk_offer_version_billing_interval CHECK (billing_interval IN ('monthly', 'yearly', 'quarterly', 'one_time')),
    CONSTRAINT chk_offer_version_published CHECK (status != 'published' OR published_at IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_offer_versions_offer_id ON subscription_offer_versions (offer_id);
CREATE INDEX IF NOT EXISTS idx_offer_versions_status ON subscription_offer_versions (status);

-- 4. Organization Subscriptions extension (Links to exact published localized offer version)
ALTER TABLE organization_subscriptions 
    ADD COLUMN IF NOT EXISTS offer_version_id UUID NULL REFERENCES subscription_offer_versions(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_org_subscriptions_offer_version ON organization_subscriptions (offer_version_id);

-- 5. Immutability Trigger for Offer Versions
CREATE OR REPLACE FUNCTION prevent_offer_version_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    -- Protect against DELETE
    IF TG_OP = 'DELETE' THEN
        IF OLD.status = 'published' OR OLD.published_at IS NOT NULL THEN
            RAISE EXCEPTION 'Cannot delete published offer version % (id: %)', OLD.version, OLD.id;
        END IF;
        IF EXISTS (SELECT 1 FROM organization_subscriptions WHERE offer_version_id = OLD.id) THEN
            RAISE EXCEPTION 'Cannot delete offer version % (id: %) referenced by subscriptions', OLD.version, OLD.id;
        END IF;
        RETURN OLD;
    END IF;

    -- Protect against UPDATE
    IF TG_OP = 'UPDATE' THEN
        IF OLD.status = 'published' OR OLD.published_at IS NOT NULL THEN
            RAISE EXCEPTION 'Cannot modify published offer version % (id: %)', OLD.version, OLD.id;
        END IF;

        IF NEW.status = 'published' THEN
            IF NEW.published_at IS NULL THEN
                NEW.published_at := NOW();
            END IF;
        END IF;

        RETURN NEW;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_prevent_offer_version_mutation ON subscription_offer_versions;
CREATE TRIGGER trg_prevent_offer_version_mutation
    BEFORE UPDATE OR DELETE ON subscription_offer_versions
    FOR EACH ROW
    EXECUTE FUNCTION prevent_offer_version_mutation();

-- 6. Re-create get_platform_overview_stats() with published_offers
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
    unassigned_organizations BIGINT,
    published_offers BIGINT
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
        ))::BIGINT AS unassigned_organizations,
        (SELECT COUNT(*) FROM subscription_offer_versions WHERE status = 'published')::BIGINT AS published_offers;
END;
$$;

ALTER FUNCTION get_platform_overview_stats() OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION get_platform_overview_stats() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION get_platform_overview_stats() TO clientpulse_app;

-- 7. Grant Permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON platform_localization_settings, subscription_offers, subscription_offer_versions TO clientpulse_app;
GRANT SELECT ON platform_localization_settings, subscription_offers, subscription_offer_versions TO clientpulse_definer;

