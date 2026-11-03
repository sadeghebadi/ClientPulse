-- Migration: 000019_subscription_hardening_and_platform_security.up.sql
-- Description: Hardening triggers for immutable published plan versions, draft editing/deletion, currency validation, audit immutability, unique active admin grants, and subscription foreign key protections.

-- 1. Hardening platform_admin_grants:
-- Prevent cascading deletion of historical grants when users are deleted: switch foreign key to ON DELETE RESTRICT
ALTER TABLE platform_admin_grants DROP CONSTRAINT IF EXISTS platform_admin_grants_user_id_fkey;
ALTER TABLE platform_admin_grants ADD CONSTRAINT platform_admin_grants_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT;

-- Prevent duplicate active grants for the same user and role
CREATE UNIQUE INDEX IF NOT EXISTS uq_platform_admin_grants_active_user_role 
    ON platform_admin_grants (user_id, role) 
    WHERE revoked_at IS NULL;

-- 2. Hardening subscription_plan_versions:
-- Make price_amount, currency, billing_interval nullable for Draft versions
ALTER TABLE subscription_plan_versions ALTER COLUMN price_amount DROP NOT NULL;
ALTER TABLE subscription_plan_versions ALTER COLUMN price_amount DROP DEFAULT;

ALTER TABLE subscription_plan_versions ALTER COLUMN currency DROP NOT NULL;
ALTER TABLE subscription_plan_versions ALTER COLUMN currency DROP DEFAULT;

ALTER TABLE subscription_plan_versions ALTER COLUMN billing_interval DROP NOT NULL;
ALTER TABLE subscription_plan_versions ALTER COLUMN billing_interval DROP DEFAULT;

-- Update constraints so nulls are permitted in drafts, but valid when present or published
ALTER TABLE subscription_plan_versions DROP CONSTRAINT IF EXISTS chk_plan_version_currency;
ALTER TABLE subscription_plan_versions ADD CONSTRAINT chk_plan_version_currency 
    CHECK (currency IS NULL OR currency ~ '^[A-Z]{3}$');

ALTER TABLE subscription_plan_versions DROP CONSTRAINT IF EXISTS chk_plan_version_billing_interval;
ALTER TABLE subscription_plan_versions ADD CONSTRAINT chk_plan_version_billing_interval 
    CHECK (billing_interval IS NULL OR billing_interval IN ('monthly', 'yearly', 'quarterly', 'one_time'));

ALTER TABLE subscription_plan_versions DROP CONSTRAINT IF EXISTS chk_plan_version_published_config;
ALTER TABLE subscription_plan_versions ADD CONSTRAINT chk_plan_version_published_config
    CHECK (
        status != 'published' OR (
            published_at IS NOT NULL AND
            price_amount IS NOT NULL AND
            currency IS NOT NULL AND
            billing_interval IS NOT NULL
        )
    );

-- 3. Trigger for Plan Version Mutation Protection:
-- Published versions are strictly immutable (UPDATE and DELETE prohibited).
-- Draft versions are editable and deletable as long as they were never published and have no subscriptions.
-- Transition from draft (published_at IS NULL) to published (published_at IS NOT NULL) is allowed atomically.
CREATE OR REPLACE FUNCTION prevent_plan_version_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    -- Protect against DELETE
    IF TG_OP = 'DELETE' THEN
        IF OLD.status = 'published' OR OLD.published_at IS NOT NULL THEN
            RAISE EXCEPTION 'Cannot delete published plan version % (id: %)', OLD.version, OLD.id;
        END IF;
        -- Also check if referenced by subscriptions
        IF EXISTS (SELECT 1 FROM organization_subscriptions WHERE plan_version_id = OLD.id) THEN
            RAISE EXCEPTION 'Cannot delete plan version % (id: %) referenced by subscriptions', OLD.version, OLD.id;
        END IF;
        RETURN OLD;
    END IF;

    -- Protect against UPDATE
    IF TG_OP = 'UPDATE' THEN
        -- If already published, prohibit any modification
        IF OLD.status = 'published' OR OLD.published_at IS NOT NULL THEN
            RAISE EXCEPTION 'Cannot modify published plan version % (id: %)', OLD.version, OLD.id;
        END IF;

        -- If transitioning from draft to published, ensure published_at is being set
        IF NEW.status = 'published' THEN
            IF NEW.published_at IS NULL THEN
                NEW.published_at := NOW();
            END IF;
            -- Ensure required fields are configured
            IF NEW.price_amount IS NULL OR NEW.currency IS NULL OR NEW.billing_interval IS NULL THEN
                RAISE EXCEPTION 'Cannot publish plan version without price_amount, currency, and billing_interval';
            END IF;
        END IF;

        RETURN NEW;
    END IF;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_prevent_plan_version_mutation ON subscription_plan_versions;
CREATE TRIGGER trg_prevent_plan_version_mutation
    BEFORE UPDATE OR DELETE ON subscription_plan_versions
    FOR EACH ROW
    EXECUTE FUNCTION prevent_plan_version_mutation();

-- 4. Trigger for Audit Log Immutability:
-- Hardens platform_admin_audit_logs against UPDATE, DELETE, and TRUNCATE even if roles are misconfigured.
CREATE OR REPLACE FUNCTION prevent_platform_audit_log_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'Platform admin audit log records are immutable and cannot be updated, deleted, or truncated';
END;
$$;

DROP TRIGGER IF EXISTS trg_prevent_audit_log_mutation ON platform_admin_audit_logs;
CREATE TRIGGER trg_prevent_audit_log_mutation
    BEFORE UPDATE OR DELETE ON platform_admin_audit_logs
    FOR EACH ROW
    EXECUTE FUNCTION prevent_platform_audit_log_mutation();

-- Re-affirm role privileges on platform_admin_audit_logs
REVOKE ALL ON platform_admin_audit_logs FROM PUBLIC;
GRANT SELECT, INSERT ON platform_admin_audit_logs TO clientpulse_app;
REVOKE UPDATE, DELETE, TRUNCATE ON platform_admin_audit_logs FROM clientpulse_app;

