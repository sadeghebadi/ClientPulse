-- Migration: 000014_scheduler_telemetry_and_diagnostics.down.sql
-- Description: Rollback scheduler telemetry tracking, cycle history, and diagnostic functions

-- Revoke and drop diagnostic functions
REVOKE EXECUTE ON FUNCTION detect_stale_monitors(INT, INT) FROM clientpulse_worker;
DROP FUNCTION IF EXISTS detect_stale_monitors(INT, INT);

REVOKE EXECUTE ON FUNCTION diagnose_monitor_scheduling_state(INT) FROM clientpulse_worker;
DROP FUNCTION IF EXISTS diagnose_monitor_scheduling_state(INT);

-- Drop tables
DROP TABLE IF EXISTS scheduler_cycle_history CASCADE;
DROP TABLE IF EXISTS scheduler_instances CASCADE;

