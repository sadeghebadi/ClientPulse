-- Migration: 000014_scheduler_telemetry_and_diagnostics.up.sql
-- Description: Scheduler telemetry tracking, cycle history, and aggregate diagnostics for silent claim failures

-- 1. Scheduler Instance State (per-instance current state)
CREATE TABLE IF NOT EXISTS scheduler_instances (
    instance_id VARCHAR(64) PRIMARY KEY,
    worker_id VARCHAR(64) NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    last_poll_started_at TIMESTAMPTZ,
    last_poll_completed_at TIMESTAMPTZ,
    last_successful_poll_at TIMESTAMPTZ,
    poll_duration_ms BIGINT,
    due_monitors_count INT NOT NULL DEFAULT 0,
    claimed_monitors_count INT NOT NULL DEFAULT 0,
    completed_checks_count INT NOT NULL DEFAULT 0,
    failed_checks_count INT NOT NULL DEFAULT 0,
    last_error_category VARCHAR(64),
    last_error_message VARCHAR(512),
    consecutive_errors INT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scheduler_instances_status 
    ON scheduler_instances (status, last_poll_completed_at DESC);

-- 2. Scheduler Cycle History (lightweight bounded history of cycles, sampled or state changes / diagnostics)
CREATE TABLE IF NOT EXISTS scheduler_cycle_history (
    id BIGSERIAL PRIMARY KEY,
    instance_id VARCHAR(64) NOT NULL,
    poll_started_at TIMESTAMPTZ NOT NULL,
    poll_completed_at TIMESTAMPTZ NOT NULL,
    duration_ms BIGINT NOT NULL,
    due_monitors_count INT NOT NULL DEFAULT 0,
    claimed_monitors_count INT NOT NULL DEFAULT 0,
    completed_checks_count INT NOT NULL DEFAULT 0,
    failed_checks_count INT NOT NULL DEFAULT 0,
    mismatch_detected BOOLEAN NOT NULL DEFAULT false,
    error_category VARCHAR(64),
    error_message VARCHAR(512),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scheduler_cycle_history_created 
    ON scheduler_cycle_history (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_scheduler_cycle_history_mismatch 
    ON scheduler_cycle_history (mismatch_detected, created_at DESC);

-- Grants for scheduler telemetry tables
-- Worker needs full access to record and prune telemetry
GRANT SELECT, INSERT, UPDATE, DELETE ON public.scheduler_instances TO clientpulse_worker;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.scheduler_cycle_history TO clientpulse_worker;
GRANT USAGE, SELECT ON SEQUENCE scheduler_cycle_history_id_seq TO clientpulse_worker;

-- clientpulse_app (future platform admin read interface) gets read-only access (NO write, NO tenant bypass)
GRANT SELECT ON public.scheduler_instances TO clientpulse_app;
GRANT SELECT ON public.scheduler_cycle_history TO clientpulse_app;

-- 3. Stored Diagnostic Function: diagnose_monitor_scheduling_state
-- Calculates aggregate counts:
-- - total_candidates: active, unpaused, due now
-- - eligible_joined: active, unpaused, due now AND active asset (eligible for claim)
-- - leased_count: due candidates currently under active unexpired lease
-- - mismatch_candidates: candidates whose asset is inactive, missing, or blocked by RLS/join
CREATE OR REPLACE FUNCTION diagnose_monitor_scheduling_state(
    p_lease_timeout_seconds INT DEFAULT 60
)
RETURNS TABLE (
    total_candidates INT,
    eligible_joined INT,
    leased_count INT,
    mismatch_candidates INT
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = pg_catalog, public, pg_temp
AS $$
DECLARE
    v_lease_timeout INT;
    v_total INT;
    v_eligible INT;
    v_leased INT;
    v_mismatch INT;
BEGIN
    v_lease_timeout := COALESCE(p_lease_timeout_seconds, 60);
    IF v_lease_timeout < 5 THEN
        v_lease_timeout := 5;
    ELSIF v_lease_timeout > 3600 THEN
        v_lease_timeout := 3600;
    END IF;

    -- Total due candidates across all tenants (m.is_active = true, status != paused, next_check_at <= NOW)
    SELECT COUNT(*) INTO v_total
    FROM public.monitors m
    WHERE m.is_active = true
      AND m.status != 'paused'
      AND m.next_check_at <= NOW();

    -- Due candidates that are currently under an unexpired lease
    SELECT COUNT(*) INTO v_leased
    FROM public.monitors m
    WHERE m.is_active = true
      AND m.status != 'paused'
      AND m.next_check_at <= NOW()
      AND m.lease_acquired_at IS NOT NULL
      AND m.lease_acquired_at >= NOW() - (v_lease_timeout || ' seconds')::INTERVAL;

    -- Eligible joined monitors (matching what claim_due_monitors queries)
    SELECT COUNT(*) INTO v_eligible
    FROM public.monitors m
    JOIN public.assets a ON m.asset_id = a.id
    WHERE m.is_active = true
      AND a.is_active = true
      AND m.status != 'paused'
      AND m.next_check_at <= NOW()
      AND (
          m.lease_acquired_at IS NULL
          OR m.lease_acquired_at < NOW() - (v_lease_timeout || ' seconds')::INTERVAL
      );

    -- Mismatch count:
    -- Due candidates that are NOT currently actively leased, but NOT eligible in the asset JOIN!
    -- This isolates the exact silent RLS or missing/inactive asset defect.
    SELECT COUNT(*) INTO v_mismatch
    FROM public.monitors m
    WHERE m.is_active = true
      AND m.status != 'paused'
      AND m.next_check_at <= NOW()
      AND (
          m.lease_acquired_at IS NULL
          OR m.lease_acquired_at < NOW() - (v_lease_timeout || ' seconds')::INTERVAL
      )
      AND NOT EXISTS (
          SELECT 1 FROM public.assets a
          WHERE a.id = m.asset_id
            AND a.is_active = true
      );

    RETURN QUERY SELECT v_total, v_eligible, v_leased, v_mismatch;
END;
$$;

ALTER FUNCTION diagnose_monitor_scheduling_state(INT) OWNER TO clientpulse_definer;
REVOKE ALL ON FUNCTION diagnose_monitor_scheduling_state(INT) FROM PUBLIC;
REVOKE ALL ON FUNCTION diagnose_monitor_scheduling_state(INT) FROM clientpulse_app;
GRANT EXECUTE ON FUNCTION diagnose_monitor_scheduling_state(INT) TO clientpulse_worker;

-- 4. Stored Diagnostic Function: detect_stale_monitors
-- Identifies monitors that are active, unpaused, not leased, and overdue by more than threshold_seconds.
-- Returns strictly sanitized diagnostic fields (no target URLs, no tenant secrets, aggregate / ID references).
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

