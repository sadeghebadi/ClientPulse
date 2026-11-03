-- Migration: 000018_platform_admin_and_subscriptions.down.sql
-- Description: Revert platform admin grants, subscription plans, versioning, organization subscriptions, audit logs, and definer functions

DROP FUNCTION IF EXISTS get_platform_organizations_list(TEXT, TEXT, INT, INT);
DROP FUNCTION IF EXISTS get_platform_org_members(UUID);
DROP FUNCTION IF EXISTS get_platform_user_memberships(UUID);
DROP FUNCTION IF EXISTS get_platform_organization_usage(UUID);
DROP FUNCTION IF EXISTS get_platform_overview_stats();

DROP POLICY IF EXISTS org_memberships_definer_policy ON public.organization_memberships;
REVOKE SELECT ON public.organization_memberships FROM clientpulse_definer;

DROP POLICY IF EXISTS clients_definer_policy ON public.clients;
REVOKE SELECT ON public.clients FROM clientpulse_definer;

REVOKE SELECT ON public.users, public.organizations, public.subscription_plans, public.subscription_plan_versions, public.organization_subscriptions FROM clientpulse_definer;

DROP TABLE IF EXISTS platform_admin_audit_logs;
DROP TABLE IF EXISTS organization_subscriptions;
DROP TABLE IF EXISTS subscription_plan_versions;
DROP TABLE IF EXISTS subscription_plans;
DROP TABLE IF EXISTS platform_admin_grants;
