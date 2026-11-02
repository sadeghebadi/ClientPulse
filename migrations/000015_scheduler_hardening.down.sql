-- Migration: 000015_scheduler_hardening.down.sql
-- Description: Rollback scheduler hardening (restore migration 14 definitions)

-- 1. Restore clientpulse_app SELECT on telemetry tables
GRANT SELECT ON public.scheduler_instances TO clientpulse_app;
GRANT SELECT ON public.scheduler_cycle_history TO clientpulse_app;

-- 2. Drop get_scheduler_aggregate_health
REVOKE EXECUTE ON FUNCTION get_scheduler_aggregate_health(INT) FROM clientpulse_app;
REVOKE EXECUTE ON FUNCTION get_scheduler_aggregate_health(INT) FROM clientpulse_worker;
DROP FUNCTION IF EXISTS get_scheduler_aggregate_health(INT);

-- 3. Restore migration 14 detect_stale_monitors definition
DROP FUNCTION IF EXISTS detect_stale_monitors(INT, INT, INT);

CREATE OR REPLACE FUNCTION detect_stale_monitors(
    p_threshold_seconds INT DEFAULT 900,
    p_limit INT DEFAULT 100
)
RETURNS TABLE (
    monitor_id UUID,
    organization_id UUID,
    status VARCHAR,
    interval_seconds INT,
    next_check_at TIMESTAMPTZ,
    last_checked_at TIMESTAMPTZ,
    overdue_seconds INT,
    is_first_check BOOLEAN
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public, pg_temp
AS $$
DECLARE
    v_thresh INT;
    v_limit INT;
BEGIN
    v_thresh := COALESCE(p_threshold_seconds, 900);
    IF v_thresh < 60 THEN
        v_thresh := 60;
    END IF;

    v_limit := COALESCE(p_limit, 100);
    IF v_limit < 1 OR v_limit > 500 THEN
        v_limit := 100;
    END IF;

    RETURN QUERY
    SELECT 
        m.id AS monitor_id,
        m.organization_id,
        m.status::VARCHAR,
        m.interval_seconds,
        m.next_check_at,
        m.last_checked_at,
        GREATEST(0, EXTRACT(EPOCH FROM (NOW() - m.next_check_at)))::INT AS overdue_seconds,
        (m.last_checked_at IS NULL) AS is_first_check
    FROM public.monitors m
    WHERE m.is_active = true
      AND m.status != 'paused'
      AND m.next_check_at <= NOW() - (v_thresh || ' seconds')::INTERVAL
      AND (
          m.lease_acquired_at IS NULL
          OR m.lease_acquired_at < NOW() - INTERVAL '60 seconds'
      )
    ORDER BY m.next_check_at ASC
    LIMIT v_limit;
END;
$$;

ALTER FUNCTION detect_stale_monitors(INT, INT) OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION detect_stale_monitors(INT, INT) FROM PUBLIC;
REVOKE ALL ON FUNCTION detect_stale_monitors(INT, INT) FROM clientpulse_app;
GRANT EXECUTE ON FUNCTION detect_stale_monitors(INT, INT) TO clientpulse_worker;

