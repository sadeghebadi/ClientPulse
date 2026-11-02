-- Migration: 000015_scheduler_hardening.up.sql
-- Description: Dynamic stale monitor thresholds, least-privilege health summary procedure, and multi-worker aggregation support

-- 1. Redefine detect_stale_monitors with Dynamic Per-Monitor Thresholds
-- stale_threshold = GREATEST(p_min_tolerance_seconds, p_interval_multiplier * m.interval_seconds)
-- Default multiplier: 3x, Default minimum tolerance: 180s (3 minutes to absorb claim delay / jitter)
-- Drop old signature of detect_stale_monitors to avoid signature confusion
DROP FUNCTION IF EXISTS detect_stale_monitors(INT, INT);

CREATE OR REPLACE FUNCTION detect_stale_monitors(
    p_interval_multiplier INT DEFAULT 3,
    p_min_tolerance_seconds INT DEFAULT 180,
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
    effective_threshold_seconds INT,
    is_first_check BOOLEAN
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public, pg_temp
AS $$
DECLARE
    v_mult INT;
    v_min_tol INT;
    v_limit INT;
BEGIN
    v_mult := COALESCE(p_interval_multiplier, 3);
    IF v_mult < 1 THEN
        v_mult := 1;
    ELSIF v_mult > 10 THEN
        v_mult := 10;
    END IF;

    v_min_tol := COALESCE(p_min_tolerance_seconds, 180);
    IF v_min_tol < 60 THEN
        v_min_tol := 60;
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
        GREATEST(v_min_tol, v_mult * m.interval_seconds)::INT AS effective_threshold_seconds,
        (m.last_checked_at IS NULL) AS is_first_check
    FROM public.monitors m
    WHERE m.is_active = true
      AND m.status != 'paused'
      AND m.next_check_at <= NOW() - (GREATEST(v_min_tol, v_mult * m.interval_seconds) || ' seconds')::INTERVAL
      AND (
          m.lease_acquired_at IS NULL
          OR m.lease_acquired_at < NOW() - INTERVAL '60 seconds'
      )
    ORDER BY m.next_check_at ASC
    LIMIT v_limit;
END;
$$;

ALTER FUNCTION detect_stale_monitors(INT, INT, INT) OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION detect_stale_monitors(INT, INT, INT) FROM PUBLIC;
REVOKE ALL ON FUNCTION detect_stale_monitors(INT, INT, INT) FROM clientpulse_app;
GRANT EXECUTE ON FUNCTION detect_stale_monitors(INT, INT, INT) TO clientpulse_worker;

-- 2. Stored Procedure for Public/API Aggregate Scheduler Health: get_scheduler_aggregate_health
-- Allows clientpulse_app to read ONLY sanitized aggregate health without querying scheduler_instances directly
CREATE OR REPLACE FUNCTION get_scheduler_aggregate_health(
    p_stale_threshold_seconds INT DEFAULT 20
)
RETURNS TABLE (
    total_active_instances INT,
    healthy_instances INT,
    stale_instances INT,
    degraded_instances INT,
    latest_successful_poll_at TIMESTAMPTZ,
    latest_poll_completed_at TIMESTAMPTZ,
    aggregate_state VARCHAR(32)
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public, pg_temp
AS $$
DECLARE
    v_stale_thresh INT;
    v_total INT := 0;
    v_healthy INT := 0;
    v_stale INT := 0;
    v_degraded INT := 0;
    v_latest_success TIMESTAMPTZ;
    v_latest_poll TIMESTAMPTZ;
    v_agg_state VARCHAR(32);
    rec RECORD;
BEGIN
    v_stale_thresh := COALESCE(p_stale_threshold_seconds, 20);
    IF v_stale_thresh < 10 THEN
        v_stale_thresh := 10;
    END IF;

    -- Evaluate distinct latest active instance per worker within recent activity window (1 hour)
    FOR rec IN 
        SELECT DISTINCT ON (s.worker_id)
               s.instance_id, s.worker_id, s.started_at, s.last_poll_started_at,
               s.last_poll_completed_at, s.last_successful_poll_at, s.consecutive_errors,
               s.status, s.updated_at
        FROM public.scheduler_instances s
        WHERE s.status = 'active'
          AND (
              (s.last_poll_completed_at IS NOT NULL AND s.last_poll_completed_at > NOW() - INTERVAL '1 hour')
              OR (s.last_poll_completed_at IS NULL AND s.started_at > NOW() - (v_stale_thresh || ' seconds')::INTERVAL)
          )
        ORDER BY s.worker_id, s.updated_at DESC
    LOOP
        v_total := v_total + 1;
        
        IF v_latest_success IS NULL OR (rec.last_successful_poll_at IS NOT NULL AND rec.last_successful_poll_at > v_latest_success) THEN
            v_latest_success := rec.last_successful_poll_at;
        END IF;
        IF v_latest_poll IS NULL OR (rec.last_poll_completed_at IS NOT NULL AND rec.last_poll_completed_at > v_latest_poll) THEN
            v_latest_poll := rec.last_poll_completed_at;
        END IF;

        -- Evaluate per-instance health
        IF (rec.last_poll_completed_at IS NULL AND rec.last_successful_poll_at IS NULL) THEN
            IF NOW() - rec.started_at > (v_stale_thresh || ' seconds')::INTERVAL THEN
                v_stale := v_stale + 1;
            END IF;
        ELSIF (NOW() - COALESCE(rec.last_poll_completed_at, rec.started_at)) > (v_stale_thresh || ' seconds')::INTERVAL THEN
            v_stale := v_stale + 1;
        ELSIF rec.consecutive_errors >= 3 THEN
            v_degraded := v_degraded + 1;
        ELSE
            v_healthy := v_healthy + 1;
        END IF;
    END LOOP;

    -- Determine aggregate state
    IF v_total = 0 THEN
        v_agg_state := 'Unknown';
    ELSIF v_healthy > 0 AND v_stale = 0 AND v_degraded = 0 THEN
        v_agg_state := 'Healthy';
    ELSIF v_healthy > 0 AND (v_stale > 0 OR v_degraded > 0) THEN
        v_agg_state := 'Degraded';
    ELSIF v_healthy = 0 AND (v_stale > 0 OR v_degraded > 0) THEN
        v_agg_state := 'Stale';
    ELSE
        v_agg_state := 'Unknown';
    END IF;

    RETURN QUERY SELECT 
        v_total,
        v_healthy,
        v_stale,
        v_degraded,
        v_latest_success,
        v_latest_poll,
        v_agg_state;
END;
$$;

ALTER FUNCTION get_scheduler_aggregate_health(INT) OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION get_scheduler_aggregate_health(INT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION get_scheduler_aggregate_health(INT) TO clientpulse_app;
GRANT EXECUTE ON FUNCTION get_scheduler_aggregate_health(INT) TO clientpulse_worker;

-- 3. Restrict Direct Access of clientpulse_app on scheduler_instances and scheduler_cycle_history
-- App role no longer needs direct SELECT on underlying tables
REVOKE SELECT ON public.scheduler_instances FROM clientpulse_app;
REVOKE SELECT ON public.scheduler_cycle_history FROM clientpulse_app;

-- Ensure clientpulse_definer has SELECT on scheduler_instances for get_scheduler_aggregate_health
GRANT SELECT ON public.scheduler_instances TO clientpulse_definer;
