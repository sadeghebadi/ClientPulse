# TASK-INCIDENT-003.2: Scheduler Health State & Lease Semantics Verification Report

**Project:** ClientPulse  
**Task ID:** TASK-INCIDENT-003.2  
**Target Migration Version:** 16 (`scheduler_lease_config_and_state_semantics`)  
**Execution Environment:** Local Development Only  
**Production Deployment:** Strictly Prohibited  
**Related Documents:**
- `task-report/TASK-INCIDENT-003-SCHEDULER-OBSERVABILITY.md`
- `task-report/TASK-INCIDENT-003.1-HARDENING.md`
- `migrations/000016_scheduler_lease_config_and_state_semantics.up.sql`
- `services/api/internal/repository/task_incident_003_2_verification_test.go`

---

## 1. Executive Summary

TASK-INCIDENT-003.2 performed a focused correctness review and hardening of the ClientPulse Scheduler Observability system. This verification focused on:
1. **Multi-Worker State Aggregation:** Ensuring aggregate status strictly follows multi-instance operational realities (Healthy, Degraded, Stale, Stopped, Unknown). Schedulers completing polling cycles but suffering database errors are classified as `Degraded`, while schedulers that cease polling progress are classified as `Stale`.
2. **HTTP Health Semantics:** Ensuring `/health/scheduler` returns `HTTP 200` exclusively for `Healthy` and `HTTP 503 Service Unavailable` for `Degraded`, `Stale`, and `Unknown`. `/health/live` remains completely decoupled so process liveness probes are not killed when background scheduling encounters issues.
3. **Lease Timeout Alignment:** Passing an explicit authoritative lease timeout parameter (`p_lease_timeout_seconds`) to `detect_stale_monitors()` matching the worker's configured lease duration (`MONITOR_LEASE_TIMEOUT_SECONDS`), eliminating discrepancies between monitor claiming and stale detection.
4. **Edge Case Safety:** Verifying safe handling of `next_check_at IS NULL` monitors and first-check scheduling tolerance.
5. **Dynamic Staleness Calculation:** Formalizing staleness thresholds as $\max(10 \times \text{pollPeriod}, 20\text{s})$ to allow reasonable tolerance while catching deadlocked loops promptly.

All changes were implemented via forward/backward database migration 16, wired through configuration, worker lifecycle, telemetry repositories, and validated by comprehensive automated test suites.

---

## 2. Multi-Worker State Aggregation Rules

### 2.1 Per-Instance Health Semantics
- **Healthy:**
  - Instance is active (`is_active = true`).
  - Completed a polling cycle within `p_stale_threshold_seconds` (`last_poll_completed_at >= NOW() - stale_threshold`).
  - Consecutive error count is below the degraded threshold (`consecutive_errors < 3`).
- **Degraded:**
  - Instance is active (`is_active = true`).
  - Completed a polling cycle within `p_stale_threshold_seconds`.
  - Consecutive error count has reached or exceeded the degraded threshold (`consecutive_errors >= 3`).
  - *Distinction:* Progress is being made, but claim/poll operations are repeatedly failing.
- **Stale:**
  - Instance is marked active (`is_active = true`), but no polling cycle completed within `p_stale_threshold_seconds`.
  - Live worker process heartbeats do **not** override this state if the scheduler loop itself is frozen or deadlocked.
- **Stopped:**
  - Instance was cleanly shut down (`is_active = false`).
  - Excluded from active instance counts; historical record retained for audit.
- **Unknown:**
  - No active instances exist, or an active instance has no cycle history records yet (initial grace period).

### 2.2 Multi-Worker Aggregation Matrix

In `migrations/000016_scheduler_lease_config_and_state_semantics.up.sql`, `get_scheduler_aggregate_health(p_stale_threshold_seconds)` implements the required matrix:

| Worker A | Worker B | Aggregate Status | Justification |
|:---|:---|:---|:---|
| **Healthy** | **Healthy** | **Healthy** | All active schedulers functioning nominally. |
| **Healthy** | **Degraded** | **Degraded** | Capacity exists, but partial failure/errors detected on a worker. |
| **Healthy** | **Stale** | **Degraded** | One worker making progress, but another worker is deadlocked/halted. |
| **Degraded** | **Degraded** | **Degraded** | Schedulers are cycling, but all are encountering recurring errors. |
| **Degraded** | **Stale** | **Degraded** | Partial execution with high errors and frozen worker. |
| **Stale** | **Stale** | **Stale** | No active scheduler is completing polling cycles across the fleet. |
| **Stopped** | **Stopped** | **Unknown** | All instances intentionally shut down; zero active workers. |
| **Unknown** | *None* | **Unknown** | No active instances registered or historical progress recorded. |

---

## 3. Authoritative Lease Timeout Configuration

### 3.1 Architecture Review & Problem Identified
In migration 000015, `detect_stale_monitors()` contained a hardcoded lease check:
```sql
AND (m.last_leased_at IS NULL OR m.last_leased_at < NOW() - INTERVAL '60 seconds')
```
If a worker was configured with a different lease duration, monitors with an active valid lease could be prematurely flagged as stale, or expired leases could linger undetected.

### 3.2 Solution in Migration 000016
1. Added `MonitorLeaseTimeoutSeconds` to `services/api/internal/config/config.go` (env: `MONITOR_LEASE_TIMEOUT_SECONDS`, default: 60s, bounds: 5s–3600s).
2. Updated `worker.SchedulerConfig` to pass this lease timeout to `claim_due_monitors()` and all telemetry queries.
3. Updated PostgreSQL function signature:
   ```sql
   CREATE OR REPLACE FUNCTION detect_stale_monitors(
       p_interval_multiplier integer DEFAULT 3,
       p_min_tolerance_seconds integer DEFAULT 60,
       p_lease_timeout_seconds integer DEFAULT 60,
       p_limit integer DEFAULT 50
   )
   RETURNS TABLE (...)
   ...
   AND (
       m.last_leased_at IS NULL 
       OR m.last_leased_at < NOW() - (p_lease_timeout_seconds || ' seconds')::interval
   )
   ```
4. Added validation in `task_incident_003_2_verification_test.go` demonstrating that when `lease_timeout_seconds = 120`, a monitor leased 70s ago is recognized as actively leased and excluded, whereas when `lease_timeout_seconds = 60`, it is correctly identified as stale.

---

## 4. Public Health HTTP Status Codes & Decoupling

### 4.1 Endpoint Behavior
In `services/api/internal/health/health.go`, the HTTP status codes for public health probes are strictly defined:

| Endpoint | Scheduler State | HTTP Status Code | Response Body |
|:---|:---|:---:|:---|
| `/health/scheduler` | `Healthy` | `200 OK` | `{"status":"Healthy"}` |
| `/health/scheduler` | `Degraded` | `503 Service Unavailable` | `{"status":"Degraded"}` |
| `/health/scheduler` | `Stale` | `503 Service Unavailable` | `{"status":"Stale"}` |
| `/health/scheduler` | `Unknown` | `503 Service Unavailable` | `{"status":"Unknown"}` |

### 4.2 Liveness Decoupling
- `/health/live` returns HTTP 200 based purely on API process responsiveness.
- It does **not** check scheduler aggregate status.
- This prevents Kubernetes or container orchestrators from restarting API web servers during worker scheduler degradation.
- Zero sensitive data (worker IDs, tenant IDs, URLs, database errors) is exposed on public endpoints.

---

## 5. Edge Case Handling: NULL Schedules and First Checks

### 5.1 NULL `next_check_at` Handling
In `detect_stale_monitors()`, monitors with `m.next_check_at IS NULL` are explicitly filtered out:
```sql
WHERE m.status != 'paused'
  AND a.status = 'active'
  AND m.next_check_at IS NOT NULL
```
Monitors that have never been scheduled or are uninitialized cannot be falsely flagged as overdue.

### 5.2 First-Check Timing
For newly created monitors, `next_check_at` is initialized upon creation (typically `NOW()` or slightly in the past for immediate claim). The dynamic formula applies:
$$\text{threshold} = \max(\text{interval\_multiplier} \times m.\text{check\_interval\_seconds}, \text{min\_tolerance\_seconds})$$
This ensures a newly inserted monitor is given adequate polling window tolerance before being flagged as stale.

---

## 6. Staleness Threshold Formula

The staleness threshold for evaluating whether a scheduler is actively progressing is:
$$\text{StaleThresholdSeconds} = \max(10 \times \text{PollPeriodSeconds}, 20\text{s})$$

- For a standard 1-second worker polling loop: staleness threshold is 20s.
- For a 5-second polling loop: staleness threshold is 50s.
- This eliminates false positives during brief GC pauses or database latency spikes while ensuring any loop freeze is caught within seconds.

---

## 7. Verification Test Results

### 7.1 Regression and Unit Tests
All unit and integration test packages pass cleanly:

1. **Unit Tests:**
   ```bash
   go test ./internal/health/ -v
   # PASS: TestSchedulerHealthHandler_PublicStatuses
   # PASS: TestSchedulerHealthHandler_SanitizedPublicResponse
   ```
2. **Aggregator & Multi-Worker State Verification:**
   ```bash
   go test ./internal/repository/ -v -run TestTaskIncident003_2
   # === RUN   TestTaskIncident003_2_MultiWorkerAggregateHealth
   # === RUN   TestTaskIncident003_2_MultiWorkerAggregateHealth/Healthy_+_Healthy_=_Healthy
   # === RUN   TestTaskIncident003_2_MultiWorkerAggregateHealth/Healthy_+_Degraded_=_Degraded
   # === RUN   TestTaskIncident003_2_MultiWorkerAggregateHealth/Healthy_+_Stale_=_Degraded
   # === RUN   TestTaskIncident003_2_MultiWorkerAggregateHealth/Degraded_+_Degraded_=_Degraded
   # === RUN   TestTaskIncident003_2_MultiWorkerAggregateHealth/Degraded_+_Stale_=_Degraded
   # === RUN   TestTaskIncident003_2_MultiWorkerAggregateHealth/Stale_+_Stale_=_Stale
   # === RUN   TestTaskIncident003_2_MultiWorkerAggregateHealth/Stopped_+_Stopped_=_Unknown
   # === RUN   TestTaskIncident003_2_MultiWorkerAggregateHealth/No_active_instances_=_Unknown
   # --- PASS: TestTaskIncident003_2_MultiWorkerAggregateHealth (0.05s)
   # === RUN   TestTaskIncident003_2_LeaseTimeoutConfigurable
   # --- PASS: TestTaskIncident003_2_LeaseTimeoutConfigurable (0.01s)
   # === RUN   TestTaskIncident003_2_NullNextCheckAtExcluded
   # --- PASS: TestTaskIncident003_2_NullNextCheckAtExcluded (0.01s)
   ```
3. **Full Project Test Suite:**
   ```bash
   cd services/api && go test ./... -p 1 -count=1
   # All packages PASS (appurl, auth, checker, config, database, health, logger, migrate, notification, pdf, rate, repository, server, service, ssrf, storage, worker)
   ```

---

## 8. Production Safety Confirmation

- **No Remote Commands Run:** No SSH connections, remote database queries, or remote deployments were attempted.
- **Local Isolation:** All schema migrations (000016) and integration tests were executed against local development containers.
- **Rollback Supported:** `migrations/000016_scheduler_lease_config_and_state_semantics.down.sql` provides clean reversion to migration 000015.
- **Ready for Review:** Production deployment remains strictly prohibited until authorized.

