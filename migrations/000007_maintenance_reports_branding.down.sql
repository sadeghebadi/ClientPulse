-- ==============================================================================
-- Migration: 000007_maintenance_reports_branding.down.sql
-- Description: Rollback maintenance activities, reports, branding, and share tokens
-- ==============================================================================

DROP POLICY IF EXISTS reports_definer_policy ON reports;
DROP POLICY IF EXISTS report_share_definer_policy ON report_share_tokens;
DROP FUNCTION IF EXISTS lookup_report_by_share_token_hash(VARCHAR);

DROP TABLE IF EXISTS report_share_tokens CASCADE;
DROP TABLE IF EXISTS reports CASCADE;
DROP TABLE IF EXISTS organization_branding CASCADE;
DROP TABLE IF EXISTS maintenance_activities CASCADE;
DROP INDEX IF EXISTS idx_audit_logs_entity;

-- Revert claim_missed_heartbeats to exclude pending
CREATE OR REPLACE FUNCTION claim_missed_heartbeats(
    p_worker_id VARCHAR,
    p_batch_size INT,
    p_lease_timeout_seconds INT DEFAULT 60
)
RETURNS TABLE (
    id UUID,
    organization_id UUID,
    client_id UUID,
    asset_id UUID,
    name VARCHAR,
    expected_interval_seconds INT,
    grace_period_seconds INT,
    status VARCHAR,
    last_received_at TIMESTAMPTZ,
    next_expected_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public, pg_temp
AS $$
BEGIN
    RETURN QUERY
    WITH overdue_heartbeats AS (
        SELECT hm.id AS hm_id
        FROM public.heartbeat_monitors hm
        WHERE hm.is_active = true
          AND hm.archived_at IS NULL
          AND hm.status IN ('healthy', 'late')
          AND hm.next_expected_at IS NOT NULL
          AND hm.next_expected_at <= NOW()
          AND (
              hm.lease_acquired_at IS NULL
              OR hm.lease_acquired_at < NOW() - (p_lease_timeout_seconds || ' seconds')::INTERVAL
          )
        ORDER BY hm.next_expected_at ASC
        LIMIT p_batch_size
        FOR UPDATE OF hm SKIP LOCKED
    ),
    claimed AS (
        UPDATE public.heartbeat_monitors hm
        SET lease_acquired_at = NOW(),
            lease_worker_id = p_worker_id,
            status = 'missed',
            updated_at = NOW()
        FROM overdue_heartbeats oh
        WHERE hm.id = oh.hm_id
        RETURNING hm.id, hm.organization_id, hm.client_id, hm.asset_id, hm.name,
                  hm.expected_interval_seconds, hm.grace_period_seconds, hm.status,
                  hm.last_received_at, hm.next_expected_at
    )
    SELECT c.id, c.organization_id, c.client_id, c.asset_id, c.name,
           c.expected_interval_seconds, c.grace_period_seconds, c.status,
           c.last_received_at, c.next_expected_at
    FROM claimed c;
END;
$$;
