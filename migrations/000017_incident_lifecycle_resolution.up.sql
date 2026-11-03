-- Migration: 000017_incident_lifecycle_resolution.up.sql
-- Description: Incident lifecycle enhancements: manual resolution, resolution type attribution, actor tracking, and heartbeat monitor referential integrity hardening

-- 1. Add resolution metadata columns to incidents table
ALTER TABLE incidents
    ADD COLUMN IF NOT EXISTS resolved_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS resolution_type VARCHAR(64) NULL,
    ADD COLUMN IF NOT EXISTS resolution_note TEXT NULL;

-- 2. Enforce allowed resolution types via check constraint
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_incident_resolution_type'
    ) THEN
        ALTER TABLE incidents
            ADD CONSTRAINT chk_incident_resolution_type
            CHECK (resolution_type IS NULL OR resolution_type IN ('automatic_recovery', 'manual', 'source_monitor_deleted'));
    END IF;
END $$;

-- 3. Referential integrity hardening:
-- Prevent hard-deletion of heartbeat monitors from cascading to historical incidents.
-- Heartbeat monitors use soft-deletion (archived_at), but if hard deletion is attempted,
-- RESTRICT ensures historical incidents are not destroyed.
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
        ON DELETE RESTRICT;
END $$;

-- 4. Create performance indexes for incident queries and resolution filtering
CREATE INDEX IF NOT EXISTS idx_incidents_org_res_type ON incidents (organization_id, resolution_type)
    WHERE resolution_type IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_incidents_org_ack_status ON incidents (organization_id, status, acknowledged_at);
