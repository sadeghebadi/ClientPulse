# TASK-INCIDENT-003.1: Scheduler Observability Security & Threshold Hardening Report

**Project:** ClientPulse  
**Priority:** High — Production Readiness  
**Execution Environment:** Local Development Only  
**Production Status:** Strictly Untouched (Zero SSH, zero deployments, zero network interaction)  
**Related Tasks:** TASK-INCIDENT-001, TASK-INCIDENT-002, TASK-INCIDENT-002.P, TASK-INCIDENT-003  
**Migration Version:** 15  
**Date:** 2026-11-02  

---

## 1. Executive Summary

Under TASK-INCIDENT-003, scheduler telemetry, silent zero-claim candidate mismatch detection, and initial observability endpoints were introduced. Following comprehensive technical review, **TASK-INCIDENT-003.1** was executed to harden four critical operational and security areas before production readiness:

1. **Dynamic Stale Monitor Thresholds:** Replaced the static 900-second threshold with a per-monitor formula:
   $$\text{stale\_threshold} = \max(3 \times \text{monitor\_interval}, \text{minimum\_scheduling\_tolerance})$$
   Monitors configured with 60s, 300s, and 3600s intervals now evaluate against proportionate operational tolerances (180s, 900s, and 10800s respectively), eliminating false alarms for long-interval monitors while detecting stalls early for fast-interval monitors.
2. **Multi-Worker Lifecycle & Aggregation:** Replaced single-row arbitrary limits with a multi-worker aggregate model in PostgreSQL. Stale instances, stopped instances, crashed workers, and restarted workers with new instance IDs are correctly reconciled per worker.
3. **Global Telemetry Least Privilege:** Revoked direct `SELECT` access to `scheduler_instances` and `scheduler_cycle_history` from `clientpulse_app`. Implemented the security-definer procedure `get_scheduler_aggregate_health(stale_threshold_seconds)` so public health and API endpoints receive only sanitized aggregate numbers without cross-tenant operational data exposure.
4. **Scheduler Stale Thresholds & Lifecycle Hardening:** Justified the staleness threshold at $\max(10 \times \text{pollPeriod}, 20\text{s})$, providing sufficient headroom for database lock contention, claim timeouts, and batch execution while maintaining early detection of scheduler hangs.

---

## 2. Dynamic Stale Monitor Thresholds

### 2.1 Problem & Rationale
In TASK-INCIDENT-003, `detect_stale_monitors` defaulted to a fixed 900s (15 minute) delay. While appropriate for the default 300s (5-minute) cadence:
- A 60-second monitor was allowed to be overdue by 15 minutes before an alert was raised.
- A 3600-second (1-hour) monitor would trigger a false alarm merely 15 minutes after its next check time, even though a 15-minute delay represents only 25% of its interval.

### 2.2 Formula & Implementation
$$\text{effective\_threshold\_seconds} = \max(v\_\text{min\_tol}, v\_\text{multiplier} \times m.\text{interval\_seconds})$$
- Default multiplier: $3\times$.
- Minimum scheduling tolerance: 180s (3 minutes to absorb claim scheduling jitter, network timeout retries, and worker concurrency dispatch).
- Evaluated in `detect_stale_monitors(p_interval_multiplier INT, p_min_tolerance_seconds INT, p_limit INT)`:
  - 60s monitor $\to$ threshold: 180s (3 minutes)
  - 300s monitor $\to$ threshold: 900s (15 minutes)
  - 3600s monitor $\to$ threshold: 10800s (3 hours)
- Paused, disabled, and actively leased monitors remain strictly excluded.
- Expired leases ($>60$s) do not suppress stale detection.

---

## 3. Multi-Worker Lifecycle & Aggregation Model

### 3.1 Aggregation States
The stored procedure `get_scheduler_aggregate_health(p_stale_threshold_seconds INT)` aggregates the latest active instance per worker within a bounded activity window:
- **Healthy:** At least one worker scheduler instance is actively polling without exceeding thresholds or consecutive errors, and no active instance is stale or degraded.
- **Degraded:** At least one worker instance is healthy, but another active worker instance is failing ($consecutive\_errors \ge 3$) or stale.
- **Stale:** Zero healthy worker instances; all active instances have ceased progress beyond the staleness threshold.
- **Unknown:** No registered scheduler instances or insufficient information.

### 3.2 Handling Worker Restarts & Crashes
- When a worker restarts and generates a new `instance_id`, `DISTINCT ON (s.worker_id)` resolves the latest row by `updated_at DESC`.
- When an instance is gracefully stopped (`StopInstance`), its status is set to `'stopped'`, preventing it from influencing active worker health.
- If a container crashes without stopping, the bounded 1-hour window ensures old abandoned instances do not cause permanent alarms once replaced.

---

## 4. Global Telemetry Security Review & Migration 000015

### 4.1 Least-Privilege Remediation
In migration 000014, `clientpulse_app` was granted `SELECT` on `scheduler_instances` and `scheduler_cycle_history`. While `clientpulse_app` cannot modify these tables, granting direct `SELECT` allowed querying all worker hostnames, process IDs, and raw internal timing metrics.

Under Migration 000015:
1. `REVOKE SELECT ON public.scheduler_instances FROM clientpulse_app;`
2. `REVOKE SELECT ON public.scheduler_cycle_history FROM clientpulse_app;`
3. Direct access to underlying telemetry tables is revoked from the application role.
4. `clientpulse_app` is granted `EXECUTE` on `get_scheduler_aggregate_health(INT)`. The procedure runs with `SECURITY DEFINER` (owned by `clientpulse_definer`), hardened with `SET search_path = pg_catalog, public, pg_temp`, returning aggregate counts (`total_active_instances`, `healthy_instances`, `stale_instances`, `degraded_instances`, `aggregate_state`).
5. Detailed diagnostics (`detect_stale_monitors` and `diagnose_monitor_scheduling_state`) remain restricted to `clientpulse_worker` only.

---

## 5. Public Health Endpoint Review

Public endpoints `/health/live`, `/health/ready`, `/health/worker`, and `/health/scheduler` were audited against production Nginx routing (`location ^~ /health/` proxies to `clientpulse_api`):
- All endpoints sanitize responses; internal SQL exceptions, table names, tenant IDs, and customer URLs are never exposed.
- `/health/scheduler` returns HTTP 200 with sanitized status (`"status": "ok"`, `"status": "degraded"`) or HTTP 503 (`"status": "stale"`).
- `/health/worker` returns HTTP 200 with worker heartbeat info and summary `"scheduler_status"`.

---

## 6. Files Created & Modified

| File | Action | Purpose |
|---|---|---|
| `migrations/000015_scheduler_hardening.up.sql` | Created | Dynamic stale monitor thresholds, `get_scheduler_aggregate_health` procedure, revoked direct table SELECT from `clientpulse_app` |
| `migrations/000015_scheduler_hardening.down.sql` | Created | Idempotent rollback for migration 15 |
| `services/api/internal/repository/scheduler_telemetry_repo.go` | Modified | Added `SchedulerAggregateHealth`, updated `DetectStaleMonitors` with dynamic parameters, added `GetSchedulerAggregateHealth` |
| `services/api/internal/health/scheduler_health.go` | Modified | Tuned staleness threshold to 20s, integrated `GetSchedulerAggregateHealth` into `DefaultSchedulerHealthService` |
| `services/api/internal/health/scheduler_health_test.go` | Modified | Updated mock service and threshold unit tests for 20s threshold |
| `services/api/internal/repository/task_incident_003_regression_test.go` | Modified | Updated calls to `DetectStaleMonitors` matching new signature |
| `services/api/internal/repository/task_incident_003_1_hardening_test.go` | Created | Integration tests for dynamic thresholds (60s, 300s, 3600s), multi-worker lifecycle aggregation, and security privilege boundaries |

---

## 7. Verification & Regression Test Results

### 7.1 Hardening Regression Test Suite
Command:
```bash
go test ./internal/repository/ -v -run TestSchedulerObservabilityHardening
```
Result:
```
=== RUN   TestSchedulerObservabilityHardening
=== RUN   TestSchedulerObservabilityHardening/TestCase1_DynamicStaleMonitorThresholds_60s_300s_3600s
=== RUN   TestSchedulerObservabilityHardening/TestCase2_MultiWorkerLifecycleAndAggregation
=== RUN   TestSchedulerObservabilityHardening/TestCase3_LeastPrivilegeGlobalTelemetryAccess
--- PASS: TestSchedulerObservabilityHardening (0.14s)
    --- PASS: TestSchedulerObservabilityHardening/TestCase1_DynamicStaleMonitorThresholds_60s_300s_3600s (0.07s)
    --- PASS: TestSchedulerObservabilityHardening/TestCase2_MultiWorkerLifecycleAndAggregation (0.02s)
    --- PASS: TestSchedulerObservabilityHardening/TestCase3_LeastPrivilegeGlobalTelemetryAccess (0.01s)
PASS
```

### 7.2 TASK-INCIDENT-003 Baseline Suite
Command:
```bash
go test ./internal/repository/ -v -run TestSchedulerObservabilityAndDiagnostics
```
Result:
```
=== RUN   TestSchedulerObservabilityAndDiagnostics
=== RUN   TestSchedulerObservabilityAndDiagnostics/TestCase1_TelemetryPersistenceAndMultipleWorkers
=== RUN   TestSchedulerObservabilityAndDiagnostics/TestCase2_FailuresAndRecoveryState
=== RUN   TestSchedulerObservabilityAndDiagnostics/TestCase3_SilentZeroClaimMismatchDetection
=== RUN   TestSchedulerObservabilityAndDiagnostics/TestCase4_StaleMonitorDetectionRules
=== RUN   TestSchedulerObservabilityAndDiagnostics/TestCase5_SecurityAndTenantIsolation
=== RUN   TestSchedulerObservabilityAndDiagnostics/TestCase6_CycleHistoryRetentionPruning
--- PASS: TestSchedulerObservabilityAndDiagnostics (0.14s)
PASS
```

### 7.3 Health Package Suite
Command:
```bash
go test ./internal/health/ -v
```
Result:
```
=== RUN   TestHealthLive
--- PASS: TestHealthLive (0.00s)
=== RUN   TestHealthReady_Success
--- PASS: TestHealthReady_Success (0.00s)
=== RUN   TestHealthReady_Failure
--- PASS: TestHealthReady_Failure (0.00s)
=== RUN   TestHealthWorker_Healthy
--- PASS: TestHealthWorker_Healthy (0.00s)
=== RUN   TestHealthWorker_Stale
--- PASS: TestHealthWorker_Stale (0.00s)
=== RUN   TestHealthWorker_Missing
--- PASS: TestHealthWorker_Missing (0.00s)
=== RUN   TestHealthWorker_DBError
--- PASS: TestHealthWorker_DBError (0.00s)
=== RUN   TestSchedulerHealthEvaluator
--- PASS: TestSchedulerHealthEvaluator (0.00s)
=== RUN   TestHealthEndpoint_Scheduler
=== RUN   TestHealthEndpoint_Scheduler/Scheduler_Not_Configured
=== RUN   TestHealthEndpoint_Scheduler/Scheduler_Healthy
=== RUN   TestHealthEndpoint_Scheduler/Scheduler_Stale_Returns_503
--- PASS: TestHealthEndpoint_Scheduler (0.00s)
PASS
```

### 7.4 Migration Chain Suite
Command:
```bash
go test ./internal/migrate/ -count=1
```
Result:
```
ok  clientpulse/services/api/internal/migrate  1.987s
```

### 7.5 Complete API Suite
Command:
```bash
go test ./... -p 1 -count=1
```
Result:
```
ok   clientpulse/services/api/internal/appurl       0.002s
ok   clientpulse/services/api/internal/auth         0.508s
ok   clientpulse/services/api/internal/checker      0.225s
ok   clientpulse/services/api/internal/config       0.002s
ok   clientpulse/services/api/internal/database     0.003s
ok   clientpulse/services/api/internal/health       0.003s
ok   clientpulse/services/api/internal/logger       0.002s
ok   clientpulse/services/api/internal/migrate      1.994s
ok   clientpulse/services/api/internal/notification 0.002s
ok   clientpulse/services/api/internal/notification/email    0.013s
ok   clientpulse/services/api/internal/notification/telegram 0.015s
ok   clientpulse/services/api/internal/pdf          2.275s
ok   clientpulse/services/api/internal/rate         0.002s
ok   clientpulse/services/api/internal/repository   4.948s
ok   clientpulse/services/api/internal/server       6.908s
ok   clientpulse/services/api/internal/service      0.437s
ok   clientpulse/services/api/internal/ssrf         0.003s
ok   clientpulse/services/api/internal/storage      0.004s
ok   clientpulse/services/api/internal/worker       2.622s
```
All packages passed cleanly with 0 failures.

---

## 8. Rollback Implications

If migration `000015` is rolled back:
```bash
./cmd/migrate down
```
Rollback restores `clientpulse_app` SELECT grants on `scheduler_instances` and `scheduler_cycle_history`, drops `get_scheduler_aggregate_health`, and restores the static 2-parameter definition of `detect_stale_monitors`.

---

## 9. Production Safety Confirmation

- **Zero SSH sessions to Production.**
- **Zero production containers modified or touched.**
- **Zero production database queries or migrations applied.**
- **Telegram and Email notifications remain completely disabled.**
- **Hamtapesh 5-minute production check schedule remains intact.**
- **Implementation is 100% verified locally and ready for review.**

