-- Migration: 000003_auth_and_multitenancy.down.sql
-- Description: Revert authentication and multi-tenancy tables

DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS email_verification_tokens CASCADE;
DROP TABLE IF EXISTS password_reset_tokens CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS organization_memberships CASCADE;
DROP TABLE IF EXISTS organizations CASCADE;
DROP TABLE IF EXISTS users CASCADE;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'clientpulse_app') THEN
        REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM clientpulse_app;
        REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM clientpulse_app;
        REVOKE ALL PRIVILEGES ON SCHEMA public FROM clientpulse_app;
        DROP OWNED BY clientpulse_app;
        DROP ROLE clientpulse_app;
    END IF;
END $$;
