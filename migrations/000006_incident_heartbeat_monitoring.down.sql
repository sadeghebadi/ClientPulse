-- Migration: 000006_incident_heartbeat_monitoring.down.sql
-- Description: Rollback incident management, heartbeat monitoring, and notification events

DROP FUNCTION IF EXISTS lookup_heartbeat_by_token_hash(VARCHAR);
DROP FUNCTION IF EXISTS claim_missed_heartbeats(VARCHAR, INT, INT);

DROP POLICY IF EXISTS monitors_definer_policy ON monitors;

DROP TABLE IF EXISTS notification_events CASCADE;
DROP TABLE IF EXISTS incidents CASCADE;
DROP TABLE IF EXISTS heartbeat_monitors CASCADE;

ALTER TABLE check_results DROP CONSTRAINT IF EXISTS uq_check_results_org_id;
ALTER TABLE assets DROP CONSTRAINT IF EXISTS uq_assets_org_client_id;
