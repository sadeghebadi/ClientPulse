-- 000008_report_snapshot_protection.up.sql
-- Database-level immutability enforcement for report snapshots and versioning metadata

-- 1. Trigger function to enforce immutability of report snapshots and version/period/identity columns
CREATE OR REPLACE FUNCTION prevent_report_snapshot_mutation()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.id IS DISTINCT FROM OLD.id
       OR NEW.organization_id IS DISTINCT FROM OLD.organization_id
       OR NEW.client_id IS DISTINCT FROM OLD.client_id
       OR NEW.report_type IS DISTINCT FROM OLD.report_type
       OR NEW.period_start IS DISTINCT FROM OLD.period_start
       OR NEW.period_end IS DISTINCT FROM OLD.period_end
       OR NEW.organization_timezone IS DISTINCT FROM OLD.organization_timezone
       OR NEW.locale IS DISTINCT FROM OLD.locale
       OR NEW.version_number IS DISTINCT FROM OLD.version_number
       OR NEW.template_version IS DISTINCT FROM OLD.template_version
       OR NEW.title IS DISTINCT FROM OLD.title
       OR NEW.snapshot_json IS DISTINCT FROM OLD.snapshot_json
       OR NEW.generated_by IS DISTINCT FROM OLD.generated_by
       OR NEW.generated_at IS DISTINCT FROM OLD.generated_at
       OR NEW.created_at IS DISTINCT FROM OLD.created_at
    THEN
        RAISE EXCEPTION 'report snapshot, version, period and identity columns are strictly immutable (attempted mutation on report %)', OLD.id
            USING ERRCODE = '23514'; -- check_violation
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_report_snapshot_mutation ON public.reports;
CREATE TRIGGER trg_prevent_report_snapshot_mutation
    BEFORE UPDATE ON public.reports
    FOR EACH ROW
    EXECUTE FUNCTION prevent_report_snapshot_mutation();

-- 2. Trigger function to enforce non-deletability of historical reports
CREATE OR REPLACE FUNCTION prevent_report_deletion()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'historical reports cannot be deleted (attempted deletion on report %)', OLD.id
        USING ERRCODE = '23514';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_report_deletion ON public.reports;
CREATE TRIGGER trg_prevent_report_deletion
    BEFORE DELETE ON public.reports
    FOR EACH ROW
    EXECUTE FUNCTION prevent_report_deletion();

-- 3. Revoke DELETE permission from application role for defense-in-depth
REVOKE DELETE ON public.reports FROM clientpulse_app;

