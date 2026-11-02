# TASK-INCIDENT-003: Scheduler Observability & Stale Monitor Detection Report

**Project:** ClientPulse  
**Priority:** Very High — Core Monitoring Reliability  
**Execution Environment:** Local Development Only  
**Production Status:** Strictly Untouched (Zero SSH, zero deployments, zero network interaction)  
**Related Tasks:** TASK-INCIDENT-001, TASK-INCIDENT-002, TASK-INCIDENT-002.P  
**Date:** 2026-11-02  

---

## 1. Executive Summary

During production incident TASK-INCIDENT-001, automatic website and API monitoring ceased while the Worker container remained green and healthy. Because the Worker process was actively running and updating its heartbeat record every two seconds, `/health/worker` reported `"status": "ok"` despite zero monitors being executed. The underlying root cause was an RLS policy omission on the joined `assets` table that prevented `claim_due_monitors()` from returning rows.

The purpose of **TASK-INCIDENT-003** is to establish end-to-end observability and early detection for the automatic monitoring scheduler so that:
1. **Worker Process Health and Scheduler Health are decoupled.** A worker heartbeat alone never implies that monitoring is functioning. If the scheduler loop stops or hangs, it transitions to `Stale` and triggers service alerts.
2. **Silent zero-claim anomalies are reliably distinguished from idle periods.** A normal idle cycle with 0 due monitors remains `Healthy`. If due monitor candidates exist but cannot be claimed because of query, join, or RLS defects, a mismatch diagnostic detects and logs the anomaly.
3. **Stale monitors are detected across tenants without violating tenant isolation.** Monitors overdue beyond threshold (default 15 minutes) are identified via a hardened `SECURITY DEFINER` diagnostic function that returns minimal sanitized metadata.
4. **Platform Admin Panel readiness (`TASK-009.B`).** Clean server-side service and repository read interfaces have been established without exposing global telemetry to tenant endpoints or granting unverified platform permissions.

---

## 2. Root Cause & Architectural Coverage

| Failure Mode | Prior Behavior | Hardened TASK-INCIDENT-003 Behavior |
|---|---|---|
| **Worker alive, Scheduler frozen** | `/health/worker` returned HTTP 200 OK because worker heartbeat looped every 2s. | Scheduler telemetry tracks `last_poll_completed_at`. If inactive past threshold, scheduler state transitions to `Stale`, and `/health/scheduler` returns HTTP 503. `/health/worker` also includes `scheduler_status`. |
| **Silent RLS / Join Failure** | `claim_due_monitors` returned 0 rows with no SQL error; system remained silently stalled. | Authoritative SQL procedure `diagnose_monitor_scheduling_state(60)` evaluates `mismatch_candidates` (`total_due` minus `eligible_joined` and `leased`). Any candidate blocked by join/RLS immediately logs an error and records telemetry. |
| **Legitimately Idle System** | Polling with 0 due monitors. | Evaluated as healthy: `consecutive_errors = 0`, `state = Healthy`, no false alarms. |
| **Lock Contention / Concurrency** | Another worker holds lease (`FOR UPDATE SKIP LOCKED`). | Diagnostic explicitly accounts for `leased_count` (`lease_acquired_at >= NOW() - timeout`) and does not count active concurrent leases as mismatches or errors. |
| **Overdue / Stuck Monitors** | Stuck monitors in `Pending First Check` remained undetected. | `detect_stale_monitors(threshold_seconds, limit)` identifies un-checked monitors overdue beyond 15m. Paused, disabled, and actively leased monitors are safely excluded. |

---

## 3. Database Architecture & Migrations

### Migration `000014_scheduler_telemetry_and_diagnostics`

Applied forward migration: `migrations/000014_scheduler_telemetry_and_diagnostics.up.sql`  
Rollback migration: `migrations/000014_scheduler_telemetry_and_diagnostics.down.sql`

#### 3.1 Schema Components

1. **`scheduler_instances` (Table)**
   - Tracks per-instance state: `instance_id` (PK), `worker_id`, `started_at`, `last_poll_started_at`, `last_poll_completed_at`, `last_successful_poll_at`, `poll_duration_ms`, `due_monitors_count`, `claimed_monitors_count`, `completed_checks_count`, `failed_checks_count`, `last_error_category`, `last_error_message`, `consecutive_errors`, `status`, `updated_at`.
   - Updated each polling tick (low database write overhead: 1 single row `ON CONFLICT UPDATE` per worker).

2. **`scheduler_cycle_history` (Table)**
   - Historical audit log of significant polling cycles: recorded on non-empty claims, errors, silent mismatches, and sampled once every 60 cycles (~1 minute).
   - Indexed on `created_at` and `mismatch_detected`.
   - Automated bounded retention: pruned every 6 hours to retain 7 days of historical cycles.

3. **`diagnose_monitor_scheduling_state(p_lease_timeout_seconds INT)` (Function)**
   - `SECURITY DEFINER` owned by `clientpulse_definer`, `search_path = pg_catalog, public, pg_temp`.
   - Returns aggregate counts: `total_candidates`, `eligible_joined`, `leased_count`, `mismatch_candidates`.
   - `REVOKE ALL FROM PUBLIC; REVOKE ALL FROM clientpulse_app; GRANT EXECUTE TO clientpulse_worker;`
   - Zero sensitive customer URLs or data exposed.

4. **`detect_stale_monitors(p_threshold_seconds INT, p_limit INT)` (Function)**
   - `SECURITY DEFINER` owned by `clientpulse_definer`, `search_path = pg_catalog, public, pg_temp`.
   - Detects monitors with `is_active = true`, `status != 'paused'`, `next_check_at <= NOW() - threshold`, not leased.
   - Returns sanitized columns: `monitor_id`, `organization_id`, `status`, `interval_seconds`, `next_check_at`, `last_checked_at`, `overdue_seconds`, `is_first_check`.
   - `REVOKE ALL FROM PUBLIC; REVOKE ALL FROM clientpulse_app; GRANT EXECUTE TO clientpulse_worker;`

---

## 4. Scheduler Health State Model

The health model introduces four discrete states evaluated by `SchedulerHealthEvaluator`:

```
                      +-------------------+
                      |      Unknown      | (Startup / < 10s grace period)
                      +-------------------+
                                |
                                v
+-------------------->+-------------------+<--------------------+
| (Errors cleared)    |      Healthy      |                     |
|                     +-------------------+                     | (Polls resume)
|                               |                               |
|                     (3+ failed cycles)                        |
|                               v                               |
|                     +-------------------+                     |
+---------------------+     Degraded      |                     |
                      +-------------------+                     |
                                |                               |
                       (No poll in > 10s)                       |
                                v                               |
                      +-------------------+---------------------+
                      |       Stale       | (HTTP 503 Service Unavailable)
                      +-------------------+
```

- **Healthy:** Completed successful polling cycles within `staleThreshold = max(5 * pollPeriod, 10s)`. Claiming 0 monitors is completely healthy.
- **Degraded:** Active polling but `consecutive_errors >= 3`.
- **Stale:** Has not completed a poll cycle within `staleThreshold` (e.g. 10s for 1s polling cadence).
- **Unknown:** Process just started, first poll pending.

### Stale Monitor Threshold Rationale
- Standard check interval in production: 300s (5 minutes).
- Preliminary stale detection threshold: 900s (15 minutes = 3x interval).
- Minimum threshold enforced by SQL: 60s.
- First-check tolerance: 900s gives ample headroom for queue processing and transient delays without prematurely alarming on normal job registration.

---

## 5. Security & Tenant Isolation Review

1. **Preservation of Migrations 000006 and 000013:**
   - `assets_definer_policy` on `public.assets` remains strictly `FOR SELECT TO clientpulse_definer USING (true)`.
   - Tenant isolation policies on `monitors`, `assets`, and check results remain 100% intact and enforced via `FORCE ROW LEVEL SECURITY`.
2. **Principle of Least Privilege:**
   - Both `diagnose_monitor_scheduling_state` and `detect_stale_monitors` are explicitly revoked from `PUBLIC` and `clientpulse_app`. Only `clientpulse_worker` can execute them.
   - `clientpulse_app` has `SELECT` permission only on `scheduler_instances` and `scheduler_cycle_history` for future admin read interfaces. It cannot insert, update, delete, or bypass tenant isolation.
   - Global scheduler diagnostics expose no customer URLs, passwords, tokens, or query bodies.

---

## 6. Files Created & Modified

| File | Action | Description |
|---|---|---|
| `migrations/000014_scheduler_telemetry_and_diagnostics.up.sql` | Created | Tables, permissions, indexes, and diagnostic procedures |
| `migrations/000014_scheduler_telemetry_and_diagnostics.down.sql` | Created | Idempotent drop and revoke statements for rollback |
| `services/api/internal/repository/scheduler_telemetry_repo.go` | Created | Telemetry persistence, diagnostic wrappers, cycle pruning |
| `services/api/internal/health/scheduler_health.go` | Created | Health evaluator, state model, SchedulerHealthService |
| `services/api/internal/health/scheduler_health_test.go` | Created | Unit tests for health states and `/health/scheduler` endpoint |
| `services/api/internal/health/health.go` | Modified | Added `SchedulerHealthService` integration to `/health/worker` and `/health/scheduler` |
| `services/api/internal/server/server.go` | Modified | Registered `GET /health/scheduler` endpoint |
| `services/api/cmd/api/main.go` | Modified | Wired `SchedulerHealthService` into API dependency container |
| `services/api/internal/worker/scheduler.go` | Modified | Cycle duration timing, telemetry recording, silent mismatch detection, cycle history pruning |
| `services/api/internal/repository/task_incident_003_regression_test.go` | Created | 6 end-to-end integration test suites covering all required scenarios |
| `services/api/internal/migrate/task_incident_002_reproducibility_test.go` | Modified | Adjusted multi-migration chain test to support version 14 |

---

## 7. Verification & Regression Test Results

### 7.1 Regression Test Execution

Command:
```bash
go test ./internal/repository/ -v -run TestSchedulerObservabilityAndDiagnostics
```

Results:
```
=== RUN   TestSchedulerObservabilityAndDiagnostics
=== RUN   TestSchedulerObservabilityAndDiagnostics/TestCase1_TelemetryPersistenceAndMultipleWorkers
=== RUN   TestSchedulerObservabilityAndDiagnostics/TestCase2_FailuresAndRecoveryState
=== RUN   TestSchedulerObservabilityAndDiagnostics/TestCase3_SilentZeroClaimMismatchDetection
    task_incident_003_regression_test.go:213: Baseline diagnosis: total=1, eligible=1, leased=0, mismatch=0
    task_incident_003_regression_test.go:231: After asset deactivated: total=1, eligible=0, leased=0, mismatch=1
=== RUN   TestSchedulerObservabilityAndDiagnostics/TestCase4_StaleMonitorDetectionRules
=== RUN   TestSchedulerObservabilityAndDiagnostics/TestCase5_SecurityAndTenantIsolation
=== RUN   TestSchedulerObservabilityAndDiagnostics/TestCase6_CycleHistoryRetentionPruning
--- PASS: TestSchedulerObservabilityAndDiagnostics (0.14s)
    --- PASS: TestSchedulerObservabilityAndDiagnostics/TestCase1_TelemetryPersistenceAndMultipleWorkers (0.02s)
    --- PASS: TestSchedulerObservabilityAndDiagnostics/TestCase2_FailuresAndRecoveryState (0.01s)
    --- PASS: TestSchedulerObservabilityAndDiagnostics/TestCase3_SilentZeroClaimMismatchDetection (0.04s)
    --- PASS: TestSchedulerObservabilityAndDiagnostics/TestCase4_StaleMonitorDetectionRules (0.02s)
    --- PASS: TestSchedulerObservabilityAndDiagnostics/TestCase5_SecurityAndTenantIsolation (0.02s)
    --- PASS: TestSchedulerObservabilityAndDiagnostics/TestCase6_CycleHistoryRetentionPruning (0.01s)
PASS
```

### 7.2 Health Evaluator & Endpoint Unit Tests

Command:
```bash
go test ./internal/health/ -v
```

Results:
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

### 7.3 Full Repository Test Suite

Command:
```bash
go test ./... -p 1 -count=1
```

Results:
```
ok   clientpulse/services/api/internal/appurl       0.002s
ok   clientpulse/services/api/internal/auth         0.479s
ok   clientpulse/services/api/internal/checker      0.163s
ok   clientpulse/services/api/internal/config       0.002s
ok   clientpulse/services/api/internal/database     0.003s
ok   clientpulse/services/api/internal/health       0.003s
ok   clientpulse/services/api/internal/logger       0.002s
ok   clientpulse/services/api/internal/migrate      1.861s
ok   clientpulse/services/api/internal/notification 0.002s
ok   clientpulse/services/api/internal/notification/email    0.013s
ok   clientpulse/services/api/internal/notification/telegram 0.016s
ok   clientpulse/services/api/internal/pdf          3.213s
ok   clientpulse/services/api/internal/rate         0.002s
ok   clientpulse/services/api/internal/repository   4.991s
ok   clientpulse/services/api/internal/server       3.611s
ok   clientpulse/services/api/internal/service      0.427s
ok   clientpulse/services/api/internal/ssrf         0.003s
ok   clientpulse/services/api/internal/storage      0.003s
ok   clientpulse/services/api/internal/worker       2.632s
```
All packages compile and pass cleanly.

---

## 8. Rollback Implications

If migration `000014` needs to be rolled back in any environment:
```sql
-- Down migration command:
./cmd/migrate down
```
Rollback removes `detect_stale_monitors`, `diagnose_monitor_scheduling_state`, and drops `scheduler_instances` and `scheduler_cycle_history`. The worker code gracefully handles a missing telemetry repo (if `pool` or table is unavailable, it continues core claim and execution without failing).

---

## 9. Platform Admin Panel Integration Readiness (`TASK-009.B`)

The server-side service interface `health.SchedulerHealthService` provides the exact read methods needed by the upcoming admin panel:
```go
type SchedulerHealthService interface {
    GetSchedulerStatus(ctx context.Context) (SchedulerTelemetrySnapshot, error)
    GetStaleMonitors(ctx context.Context, thresholdSeconds int, limit int) ([]*repository.StaleMonitorItem, error)
    DiagnoseSchedulingState(ctx context.Context) (*repository.SchedulingStateDiagnostic, error)
}
```
When `TASK-009.B` implements Platform Admin authorization, this service can be injected into the admin handler without modifying database procedures or compromising tenant boundaries.

---

## 10. Production Safety Confirmation

- **Zero SSH sessions established.**
- **Zero production containers modified or restarted.**
- **Zero production database connections opened.**
- **Zero production migrations executed.**
- **Telegram and Email notifications remain disabled.**
- **Hamtapesh monitor and production cadence remain untouched.**
- **Implementation is 100% verified locally and ready for review.**

