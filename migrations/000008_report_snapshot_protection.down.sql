-- 000008_report_snapshot_protection.down.sql

DROP TRIGGER IF EXISTS trg_prevent_report_deletion ON public.reports;
DROP FUNCTION IF EXISTS prevent_report_deletion();

DROP TRIGGER IF EXISTS trg_prevent_report_snapshot_mutation ON public.reports;
DROP FUNCTION IF EXISTS prevent_report_snapshot_mutation();

GRANT DELETE ON public.reports TO clientpulse_app;

