-- Migration: 000013_assets_scheduler_rls_hardening.up.sql
-- Description: Establish least-privilege SELECT RLS policy on assets for clientpulse_definer (TASK-INCIDENT-001/002)

-- Drop any existing policy with the target name (including emergency hotfix 'FOR ALL' or temporary variants)
DROP POLICY IF EXISTS assets_definer_policy ON public.assets;

-- Reconcile and clean up any dev/staging alias if present
DROP POLICY IF EXISTS assets_definer_select ON public.assets;

-- Create least-privilege SELECT-only policy for security definer background procedures (e.g. claim_due_monitors)
CREATE POLICY assets_definer_policy ON public.assets
    FOR SELECT
    TO clientpulse_definer
    USING (true);

