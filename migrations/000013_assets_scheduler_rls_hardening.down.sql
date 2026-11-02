-- Migration: 000013_assets_scheduler_rls_hardening.down.sql
-- Description: Revert least-privilege SELECT RLS policy on assets for clientpulse_definer
-- WARNING / ROLLBACK CONSEQUENCE:
-- Reverting this migration removes the assets policy for clientpulse_definer.
-- In a database running the background monitor scheduler, this will cause claim_due_monitors()
-- to return 0 rows for all monitors joining assets under background worker connections without
-- an HTTP tenant context, reproducing the TASK-INCIDENT-001 monitoring failure.

DROP POLICY IF EXISTS assets_definer_policy ON public.assets;

