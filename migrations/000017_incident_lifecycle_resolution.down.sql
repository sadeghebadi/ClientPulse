-- Migration: 000017_incident_lifecycle_resolution.down.sql
-- Description: Rollback incident lifecycle resolution columns, constraints, and restore original cascade FK
-- Rollback Implications:
-- Rolling back this migration drops resolved_by, resolution_type, and resolution_note from the incidents table.
-- Any attribution of manual resolutions or source monitor deletions will be lost upon rollback.

-- 1. Drop indexes
DROP INDEX IF EXISTS idx_incidents_org_res_type;
DROP INDEX IF EXISTS idx_incidents_org_ack_status;

-- 2. Restore original cascade foreign key for heartbeat monitors
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_incidents_heartbeat_org'
    ) THEN
        ALTER TABLE incidents DROP CONSTRAINT fk_incidents_heartbeat_org;
    END IF;

    ALTER TABLE incidents
        ADD CONSTRAINT fk_incidents_heartbeat_org
        FOREIGN KEY (organization_id, heartbeat_monitor_id)
        REFERENCES heartbeat_monitors(organization_id, id)
        ON DELETE CASCADE;
END $$;

-- 3. Drop check constraint and columns
ALTER TABLE incidents DROP CONSTRAINT IF EXISTS chk_incident_resolution_type;
ALTER TABLE incidents
    DROP COLUMN IF EXISTS resolution_note,
    DROP COLUMN IF EXISTS resolution_type,
    DROP COLUMN IF EXISTS resolved_by;
