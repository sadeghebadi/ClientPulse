-- Migration: 000004_clients_assets_monitoring.down.sql
-- Description: Rollback clients, assets, monitors, and check_results

DROP FUNCTION IF EXISTS claim_due_monitors(VARCHAR, INT, INT);

DROP TABLE IF EXISTS check_results CASCADE;
DROP TABLE IF EXISTS monitors CASCADE;
DROP TABLE IF EXISTS assets CASCADE;
DROP TABLE IF EXISTS clients CASCADE;

