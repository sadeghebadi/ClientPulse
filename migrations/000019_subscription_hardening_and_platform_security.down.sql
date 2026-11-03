-- Migration: 000019_subscription_hardening_and_platform_security.down.sql
-- Description: Rollback of hardening triggers, constraints, and index additions.

DROP TRIGGER IF EXISTS trg_prevent_audit_log_mutation ON platform_admin_audit_logs;
DROP FUNCTION IF EXISTS prevent_platform_audit_log_mutation();

DROP TRIGGER IF EXISTS trg_prevent_plan_version_mutation ON subscription_plan_versions;
DROP FUNCTION IF EXISTS prevent_plan_version_mutation();

ALTER TABLE subscription_plan_versions DROP CONSTRAINT IF EXISTS chk_plan_version_published_config;

-- Restore constraints and default values
ALTER TABLE subscription_plan_versions DROP CONSTRAINT IF EXISTS chk_plan_version_currency;
ALTER TABLE subscription_plan_versions ADD CONSTRAINT chk_plan_version_currency CHECK (currency ~ '^[A-Z]{3}$');

ALTER TABLE subscription_plan_versions DROP CONSTRAINT IF EXISTS chk_plan_version_billing_interval;
ALTER TABLE subscription_plan_versions ADD CONSTRAINT chk_plan_version_billing_interval CHECK (billing_interval IN ('monthly', 'yearly', 'quarterly', 'one_time'));

UPDATE subscription_plan_versions SET price_amount = 0 WHERE price_amount IS NULL;
ALTER TABLE subscription_plan_versions ALTER COLUMN price_amount SET DEFAULT 0;
ALTER TABLE subscription_plan_versions ALTER COLUMN price_amount SET NOT NULL;

UPDATE subscription_plan_versions SET currency = 'USD' WHERE currency IS NULL;
ALTER TABLE subscription_plan_versions ALTER COLUMN currency SET DEFAULT 'USD';
ALTER TABLE subscription_plan_versions ALTER COLUMN currency SET NOT NULL;

UPDATE subscription_plan_versions SET billing_interval = 'monthly' WHERE billing_interval IS NULL;
ALTER TABLE subscription_plan_versions ALTER COLUMN billing_interval SET DEFAULT 'monthly';
ALTER TABLE subscription_plan_versions ALTER COLUMN billing_interval SET NOT NULL;

DROP INDEX IF EXISTS uq_platform_admin_grants_active_user_role;

ALTER TABLE platform_admin_grants DROP CONSTRAINT IF EXISTS platform_admin_grants_user_id_fkey;
ALTER TABLE platform_admin_grants ADD CONSTRAINT platform_admin_grants_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

