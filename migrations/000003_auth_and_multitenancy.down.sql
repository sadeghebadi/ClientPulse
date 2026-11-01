-- Migration: 000003_auth_and_multitenancy.down.sql
-- Description: Revert authentication and multi-tenancy tables

DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS email_verification_tokens CASCADE;
DROP TABLE IF EXISTS password_reset_tokens CASCADE;
DROP TABLE IF EXISTS sessions CASCADE;
DROP TABLE IF EXISTS organization_memberships CASCADE;
DROP TABLE IF EXISTS organizations CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP ROLE IF EXISTS clientpulse_app;
