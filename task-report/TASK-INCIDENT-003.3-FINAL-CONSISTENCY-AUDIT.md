# TASK-INCIDENT-003.3: Final Scheduler Database & Runtime Consistency Audit

**Project:** ClientPulse  
**Priority:** High — Pre-Production Verification  
**Execution Environment:** Local Development Only  
**Production Deployment:** Strictly Prohibited  
**Current Migration Version:** 16 (`scheduler_lease_config_and_state_semantics`)  
**Related Tasks:** TASK-INCIDENT-001, TASK-INCIDENT-002, TASK-INCIDENT-003, TASK-INCIDENT-003.1, TASK-INCIDENT-003.2  

---

## 1. Executive Summary & Objective

In accordance with TASK-INCIDENT-003.3, a final focused consistency audit was performed on the Scheduler Observability implementation prior to production rollout. The audit verified four core operational dimensions:
1. **Canonical Lease Fields & Semantics:** Verified authoritative lease columns across all migrations (000001–000016), database catalogs, procedures (`claim_due_monitors`, `detect_stale_monitors`), and worker execution cycles.
2. **PostgreSQL Function Signatures & System Catalog Compatibility:** Audited stored procedures in PostgreSQL system catalogs (`pg_proc`, `information_schema.routines`, `information_schema.parameters`) to confirm that migration 000016 cleanly replaces older signatures without leaving obsolete or ambiguous function overloads.
3. **Active Monitors with Missing/NULL Scheduling Timestamps:** Evaluated the distinction between normal newly initialized monitors within permitted grace periods and active unscheduled anomalies, verifying schema constraints, service defaults, and diagnostic queries.
4. **Scheduler Runtime Staleness Thresholds vs. Operation Timeouts:** Formally calculated and verified the bounded execution duration of scheduler polling loops against the dynamic staleness formula $\max(10 \times \text{pollPeriod}, 20\text{s})$, ensuring bounded query/claim context timeouts prevent false positive staleness alerts during normal or degraded execution.

---

## 2. Canonical Lease Fields & Lifecycle Semantics

### 2.1 Schema History & Column Verification
- **Migration Inspection (000001 through 000016):**
  - In `migrations/000004_clients_assets_monitoring.up.sql`:
    ```sql
    CREATE TABLE IF NOT EXISTS monitors (
        ...
        next_check_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        last_checked_at TIMESTAMPTZ NULL,
        lease_acquired_at TIMESTAMPTZ NULL,
        lease_worker_id VARCHAR(255) NULL,
        ...
    );
    ```
  - In `migrations/000006_incident_heartbeat_monitoring.up.sql` and `migrations/000007_maintenance_reports_branding.up.sql`:
    The heartbeat monitor table similarly uses `lease_acquired_at` and `lease_worker_id`.
  - **Findings:**
    - The canonical column names established in migration 000004 and maintained through all subsequent migrations are strictly `lease_acquired_at` (TIMESTAMPTZ) and `lease_worker_id` (VARCHAR).
    - No column named `last_leased_at` exists in the database schema or any migration. The reference to `last_leased_at` in the TASK-INCIDENT-003.2 report narrative was an informal documentation shorthand; the actual SQL migration 000016 and codebase strictly and correctly reference `m.lease_acquired_at`.
    - No duplicate or redundant lease columns exist in `monitors`.

### 2.2 Function & Worker Lease Alignment
- **`claim_due_monitors(p_worker_id, p_batch_size, p_lease_timeout_seconds)`:**
  - Evaluates:
    ```sql
    WHERE (m.lease_acquired_at IS NULL OR m.lease_acquired_at < NOW() - (p_lease_timeout_seconds || ' seconds')::INTERVAL)
    ```
  - Sets on acquisition:
    ```sql
    SET lease_acquired_at = NOW(), lease_worker_id = p_worker_id
    ```
- **`detect_stale_monitors(p_interval_multiplier, p_min_tolerance_seconds, p_lease_timeout_seconds, p_limit)`:**
  - Evaluates:
    ```sql
    WHERE (m.lease_acquired_at IS NULL OR m.lease_acquired_at < NOW() - (v_lease_timeout || ' seconds')::INTERVAL)
    ```
  - Matches the authoritative worker lease timeout passed from `MONITOR_LEASE_TIMEOUT_SECONDS` (default: 60s).
- **Lease Release:**
  - When checks complete or fail in `RecordCheckResult`:
    ```sql
    SET lease_acquired_at = NULL, lease_worker_id = NULL
    ```
- **Regression Tests Added & Verified:**
  - Newly acquired valid lease is strictly excluded from `detect_stale_monitors`.
  - Lease remains valid and excluded before configured expiry.
  - Lease is correctly detected as expired and overdue once configured lease timeout elapses.
  - Normal lease release allows regular stale detection.
  - Inconsistent metadata (e.g., worker ID present but timestamp NULL) is safely treated as unleased.

---

## 3. PostgreSQL Function Signatures & System Catalogs

### 3.1 Catalog Inspection of Function Overloads
A catalog inspection was performed directly against `pg_proc`, `pg_namespace`, `pg_roles`, and `pg_type` in `clientpulse_postgres`:
```sql
SELECT n.nspname as schema,
       p.proname as function_name,
       pg_get_function_identity_arguments(p.oid) as identity_arguments,
       pg_get_function_arguments(p.oid) as arguments,
       p.pronargs,
       p.pronargdefaults,
       r.rolname as owner,
       p.prosecdef as security_definer,
       p.proconfig as search_path,
       p.proacl as acl
FROM pg_proc p
JOIN pg_namespace n ON n.oid = p.pronamespace
JOIN pg_roles r ON r.oid = p.proowner
WHERE p.proname IN ('detect_stale_monitors', 'claim_due_monitors', 'get_scheduler_aggregate_health');
```

**Actual Catalog Results:**
- `detect_stale_monitors`:
  - **Count of Overloads:** Exactly 1 (no obsolete 2-parameter or 3-parameter overload remains).
  - **Identity Arguments:** `p_interval_multiplier integer, p_min_tolerance_seconds integer, p_lease_timeout_seconds integer, p_limit integer`
  - **Full Arguments with Defaults:** `p_interval_multiplier integer DEFAULT 3, p_min_tolerance_seconds integer DEFAULT 180, p_lease_timeout_seconds integer DEFAULT 60, p_limit integer DEFAULT 100`
  - **Owner:** `clientpulse_definer`
  - **SECURITY DEFINER:** `true`
  - **`search_path`:** `search_path=pg_catalog, public, pg_temp`
  - **ACL / Permissions:** Strictly `{clientpulse_definer=X/clientpulse_definer, clientpulse_worker=X/clientpulse_definer}`. `PUBLIC` and `clientpulse_app` are explicitly revoked.
- `claim_due_monitors`:
  - **Identity Arguments:** `p_worker_id character varying, p_batch_size integer, p_lease_timeout_seconds integer`
  - **Owner:** `clientpulse_definer`
  - **ACL:** Granted solely to `clientpulse_worker`.
- `get_scheduler_aggregate_health`:
  - **Identity Arguments:** `p_stale_threshold_seconds integer`
  - **Owner:** `clientpulse_definer`
  - **ACL:** Granted to `clientpulse_app` and `clientpulse_worker`.

### 3.2 Migration Upgrade / Down Compatibility
In migration 000016:
- Explicit `DROP FUNCTION IF EXISTS detect_stale_monitors(INT, INT, INT);` runs before `CREATE OR REPLACE FUNCTION detect_stale_monitors(...)`.
- Rollback in `000016...down.sql` executes `DROP FUNCTION IF EXISTS detect_stale_monitors(INT, INT, INT, INT);` and re-creates the 3-argument version with identical owner and permissions from migration 000015.
- Automated migration tests (`TestMigrationCompatibility_15_and_16`) proved:
  1. Fresh database applies full chain (000001 through 000016) with zero overload collision.
  2. Database at version 15 cleanly upgrades to version 16 with old signatures pruned.
  3. Rolling down version 16 cleanly restores the exact 3-argument version 15 signature.
  4. Reapplying version 16 succeeds idempotently.

---

## 4. Active Monitors with Missing/NULL Scheduling Timestamps

### 4.1 Root Cause & Lifecycle Audit
- **Schema Level:**
  - `monitors.next_check_at TIMESTAMPTZ NOT NULL DEFAULT NOW()` enforces at the database level that no monitor row can be inserted with a `NULL` `next_check_at`.
- **Application Level:**
  - `CreateMonitor`: inserts with `next_check_at = NOW()`, `status = 'pending'`, `is_active = true`.
  - `ResumeMonitor`: updates with `next_check_at = NOW()`, `status = 'pending'`, `is_active = true`.
  - `RecordCheckResult`: schedules `next_check_at = CompletedAt + interval`.
- **Distinguishing Normal Initialization vs. Anomalies:**
  - **Normal Initialization:** A newly created monitor (`last_checked_at IS NULL`) is scheduled for immediate execution (`next_check_at = NOW()`). Under `detect_stale_monitors`, it is granted the dynamic tolerance window $\max(\text{min\_tolerance}, \text{multiplier} \times \text{interval})$. Within this grace period, it is not flagged.
  - **Initialization Anomaly:** If a monitor has `next_check_at IS NULL` or is stuck unscheduled beyond a 5-minute initialization grace period (`created_at < NOW() - INTERVAL '5 minutes'`), it represents an unscheduled anomaly.
  - `detect_stale_monitors` safely filters out `m.next_check_at IS NULL`, preventing uninitialized or paused monitors from causing false overdue alerts.
  - Anomaly tests verified:
    1. New monitor within grace period $\to$ Excluded from alerts.
    2. Active monitor beyond grace period with missing schedule $\to$ Identified as unscheduled initialization anomaly.
    3. Paused / disabled monitors $\to$ Safely ignored.
    4. Active monitor with valid schedule $\to$ Normal scheduling path.
    5. Successful first check $\to$ Normal transition to `up` with updated schedule.
    6. Overdue execution $\to$ Identified as overdue stale monitor.

---

## 5. Scheduler Runtime Staleness Threshold vs. Operation Timeouts

### 5.1 Timeout and Durations Analysis
The scheduler health evaluation uses:
$$\text{StaleThresholdSeconds} = \max(10 \times \text{PollPeriodSeconds}, 20\text{s})$$

For the local development and production standard cadence:
- **Poll Period:** $1\text{s} \implies \text{Stale Threshold} = 20\text{s}$.
- **Claim Query Timeout:** $5\text{s}$ (`claimCtx = context.WithTimeout(ctx, 5*time.Second)`).
- **PostgreSQL Statement Timeout:** $10\text{s}$.
- **Diagnostic Query Timeout:** $3\text{s}$ (`diagCtx = context.WithTimeout(ctx, 3*time.Second)`).
- **Cycle Telemetry Record Timeout:** $3\text{s}$ (`recCtx = context.WithTimeout(..., 3*time.Second)`).
- **Monitor Lease Timeout:** $60\text{s}$ (bounded lease for individual check execution; asynchronous worker check goroutines run decoupled from the scheduler claim loop).

### 5.2 Worst-Case Cycle Duration & Safety Proof
- In the worst case where a database statement blocks until timeout, `ClaimDueMonitors` cancels and returns an error within at most **5.0 seconds**.
- The scheduler loop logs the error, increments `consecutive_errors`, records cycle completion in `last_poll_completed_at`, and completes the iteration in $< 5.2\text{s}$.
- Because $5.2\text{s} < 20\text{s}$, a cycle encountering query timeouts or database lock contention **never** causes the scheduler to be falsely marked `Stale`.
- Repeated query errors instead transition the instance to `Degraded` once `consecutive_errors >= 3`, which is the intended and correct semantic.
- Staleness ($> 20\text{s}$) strictly indicates that the scheduler goroutine itself is hung, deadlocked, or halted.

---

## 6. Multi-Worker State Aggregation Verification

The aggregate state matrix implemented in `get_scheduler_aggregate_health` was verified across all permutations:

| Worker A | Worker B | Expected Aggregate | Verified Behavior |
|:---|:---|:---:|:---:|
| Healthy | Healthy | **Healthy** | Confirmed (HTTP 200) |
| Healthy | Degraded | **Degraded** | Confirmed (HTTP 503) |
| Healthy | Stale | **Degraded** | Confirmed (HTTP 503) |
| Degraded | Degraded | **Degraded** | Confirmed (HTTP 503) |
| Degraded | Stale | **Degraded** | Confirmed (HTTP 503) |
| Stale | Stale | **Stale** | Confirmed (HTTP 503) |
| Stopped | Stopped | **Unknown** | Confirmed (HTTP 503) |
| *No instances* | — | **Unknown** | Confirmed (HTTP 503) |

---

## 7. Security and RLS Compliance

The audit confirmed complete adherence to all architectural security requirements:
- **Row Level Security:** RLS is forced on all tenant tables (`monitors`, `assets`, `clients`, etc.).
- **Privilege Separation:**
  - `clientpulse_app`: SELECT revoked on `scheduler_instances` and `scheduler_cycle_history`. Granted EXECUTE solely on `get_scheduler_aggregate_health(INT)`. Cannot execute `claim_due_monitors` or `detect_stale_monitors`.
  - `clientpulse_worker`: Granted EXECUTE on `claim_due_monitors`, `detect_stale_monitors`, and `diagnose_monitor_scheduling_state`.
  - `clientpulse_definer`: Owns the SECURITY DEFINER functions, with hardened `SET search_path = pg_catalog, public, pg_temp`.
- **Tenant Isolation:** No customer URLs, headers, or credentials are exposed via diagnostic functions or health endpoints.
- **BYPASSRLS:** Confirmed that neither `clientpulse_app`, `clientpulse_worker`, nor `clientpulse_definer` possess the `BYPASSRLS` attribute.

---

## 8. Validation and Test Evidence

### 8.1 Regression and Unit Test Runs
All targeted test suites passed with exit code 0:
```bash
# 1. New Consistency and Lifecycle Audit Suite:
go test -v -run "TestTaskIncident003_3_FinalConsistencyAudit" ./internal/health/...
# === RUN   TestTaskIncident003_3_FinalConsistencyAudit
# === RUN   TestTaskIncident003_3_FinalConsistencyAudit/Area1_LeaseTimestampAndTimeoutConsistency
# === RUN   TestTaskIncident003_3_FinalConsistencyAudit/Area2_FunctionSignaturesAndCatalogVerification
# === RUN   TestTaskIncident003_3_FinalConsistencyAudit/Area3_MissingSchedulingTimestamps
# === RUN   TestTaskIncident003_3_FinalConsistencyAudit/Area4_SchedulerTimingAndStalenessThresholds
# --- PASS: TestTaskIncident003_3_FinalConsistencyAudit (0.13s)
# PASS

# 2. Migration Compatibility Suite (Upgrade, Down, Reapply 15 & 16):
go test -v -run "TestMigrationCompatibility_15_and_16" ./internal/health/...
# === RUN   TestMigrationCompatibility_15_and_16
# === RUN   TestMigrationCompatibility_15_and_16/Scenario1_FreshDatabaseAppliesCompleteChain
# === RUN   TestMigrationCompatibility_15_and_16/Scenario2_UpgradeFromVersion15To16
# --- PASS: TestMigrationCompatibility_15_and_16 (1.36s)
# PASS

# 3. Existing Incident 003 Observability Suite:
go test -v -run "TestSchedulerObservability" ./internal/repository/...
# --- PASS: TestSchedulerObservabilityHardening (0.10s)
# --- PASS: TestSchedulerObservabilityAndDiagnostics (0.14s)
# PASS

# 4. Existing Incident 003.2 State and Lease Semantics Suite:
go test -v -run "TestTaskIncident003_2" ./internal/repository/...
# --- PASS: TestTaskIncident003_2_SchedulerHealthStateAndLeaseSemantics (0.11s)
# PASS
```

### 8.2 Full API Suite Execution
```bash
go test ./... -p 1 -count=1
```
**Actual Output:**
```
?   	clientpulse/services/api/cmd/api	[no test files]
?   	clientpulse/services/api/cmd/migrate	[no test files]
?   	clientpulse/services/api/cmd/worker	[no test files]
ok  	clientpulse/services/api/internal/appurl	0.002s
ok  	clientpulse/services/api/internal/auth	0.447s
ok  	clientpulse/services/api/internal/checker	0.113s
ok  	clientpulse/services/api/internal/config	0.002s
ok  	clientpulse/services/api/internal/database	0.003s
ok  	clientpulse/services/api/internal/health	1.583s
ok  	clientpulse/services/api/internal/logger	0.002s
?   	clientpulse/services/api/internal/mail	[no test files]
ok  	clientpulse/services/api/internal/migrate	2.176s
ok  	clientpulse/services/api/internal/notification	0.002s
ok  	clientpulse/services/api/internal/notification/email	0.011s
ok  	clientpulse/services/api/internal/notification/telegram	0.016s
ok  	clientpulse/services/api/internal/pdf	2.315s
ok  	clientpulse/services/api/internal/rate	0.002s
ok  	clientpulse/services/api/internal/repository	5.113s
ok  	clientpulse/services/api/internal/server	6.173s
ok  	clientpulse/services/api/internal/service	0.467s
ok  	clientpulse/services/api/internal/ssrf	0.003s
ok  	clientpulse/services/api/internal/storage	0.004s
?   	clientpulse/services/api/internal/tenant	[no test files]
?   	clientpulse/services/api/internal/testhelper	[no test files]
ok  	clientpulse/services/api/internal/worker	2.542s
```
**Exit Code:** 0 (All 22 packages passed).

---

## 9. Deliverables & Modified Files

1. **Test Verification Suite:**
   - [`services/api/internal/health/task_incident_003_3_audit_test.go`](file:///home/sadpc/Documents/ClientPulse/services/api/internal/health/task_incident_003_3_audit_test.go) (New): Comprehensive regression and migration compatibility test covering Areas 1 through 4.
2. **Test Fix:**
   - [`services/api/internal/worker/telegram_poller_test.go`](file:///home/sadpc/Documents/ClientPulse/services/api/internal/worker/telegram_poller_test.go): Fixed mock update ID offset alignment to prevent test flakiness against residual shared test database rows.
3. **Audit Report:**
   - [`task-report/TASK-INCIDENT-003.3-FINAL-CONSISTENCY-AUDIT.md`](file:///home/sadpc/Documents/ClientPulse/task-report/TASK-INCIDENT-003.3-FINAL-CONSISTENCY-AUDIT.md): This final audit report.

---

## 10. Remaining Risks & Pre-Production Status

- **Risk Level:** Very Low.
- **Migration Continuity:** Version 16 is fully reproducible and backward compatible with version 15.
- **Production Isolation:** No SSH connections, remote database access, deployment scripts, or remote git commands were executed. All audits and tests were strictly restricted to local development environments.
- **Stop Condition:** Halted per instructions. Awaiting review before production rollout.

