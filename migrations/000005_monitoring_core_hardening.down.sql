-- Migration: 000005_monitoring_core_hardening.down.sql
-- Description: Rollback of monitoring core hardening

-- Drop worker-specific RLS policies
DROP POLICY IF EXISTS check_results_worker_cleanup_select ON check_results;
DROP POLICY IF EXISTS check_results_worker_cleanup_delete ON check_results;
DROP POLICY IF EXISTS check_aggregates_worker_cleanup_select ON check_aggregates_hourly;
DROP POLICY IF EXISTS check_aggregates_worker_cleanup_delete ON check_aggregates_hourly;

-- Revert function permissions and ownership
ALTER FUNCTION claim_due_monitors(VARCHAR, INT, INT) OWNER TO clientpulse;
REVOKE EXECUTE ON FUNCTION claim_due_monitors(VARCHAR, INT, INT) FROM clientpulse_worker;
GRANT EXECUTE ON FUNCTION claim_due_monitors(VARCHAR, INT, INT) TO clientpulse_app;

-- Drop table check_aggregates_hourly
DROP TABLE IF EXISTS check_aggregates_hourly;

-- Revert monitors status and consecutive_failures
ALTER TABLE monitors DROP CONSTRAINT IF EXISTS chk_monitors_status;
ALTER TABLE monitors ADD CONSTRAINT chk_monitors_status 
    CHECK (status IN ('pending', 'up', 'down', 'paused'));
ALTER TABLE monitors DROP COLUMN IF EXISTS consecutive_failures;

-- Revert composite foreign keys back to simple foreign keys
ALTER TABLE check_results DROP CONSTRAINT IF EXISTS fk_check_results_monitor_org;
ALTER TABLE check_results ADD CONSTRAINT check_results_monitor_id_fkey 
    FOREIGN KEY (monitor_id) REFERENCES monitors(id) ON DELETE CASCADE;

ALTER TABLE monitors DROP CONSTRAINT IF EXISTS fk_monitors_asset_org;
ALTER TABLE monitors ADD CONSTRAINT monitors_asset_id_fkey 
    FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE;

ALTER TABLE assets DROP CONSTRAINT IF EXISTS fk_assets_client_org;
ALTER TABLE assets ADD CONSTRAINT assets_client_id_fkey 
    FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE;

-- Drop composite UNIQUE constraints
ALTER TABLE monitors DROP CONSTRAINT IF EXISTS uq_monitors_org_id;
ALTER TABLE assets DROP CONSTRAINT IF EXISTS uq_assets_org_id;
ALTER TABLE clients DROP CONSTRAINT IF EXISTS uq_clients_org_id;

-- Revert schema CREATE permission
GRANT CREATE ON SCHEMA public TO PUBLIC;
