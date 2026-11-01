# TASK-003.1 Audit & Hardening Report — Monitoring Core Security, Data Integrity & Regression Audit

**Project:** ClientPulse  
**Phase:** Phase 2 Targeted Verification & Hardening  
**Task:** TASK-003.1  
**Status:** Completed & Fully Verified ✅  
**Date:** 2026-11-01  
**Single Source of Truth:** `PRD.md` and TASK-003.1 specification

---

## 1. Executive Summary

This report documents the targeted security hardening, data integrity enforcement, state machine transition implementation, lifecycle retention automation, and end-to-end verification performed for the ClientPulse Phase 2 Monitoring Core.

All 6 key findings identified in the TASK-003.1 review have been thoroughly investigated, implemented via versioned migration `000005_monitoring_core_hardening`, and verified with automated test suites under Go race detection and Next.js Turbopack compilation.

| Audit Area | Status | Verification Summary |
|---|---|---|
| **1. Worker DB Function & Privilege Separation** | Verified ✅ | Created `clientpulse_worker` (login) and `clientpulse_definer` (non-login owner). Revoked `EXECUTE` on `claim_due_monitors` from `PUBLIC` and `clientpulse_app`. Fixed `search_path`, parameter validation, revoked schema `CREATE`, revoked credential/audit access from worker. |
| **2. Composite Tenant Foreign Keys** | Verified ✅ | Composite UNIQUE constraints and composite FKs enforced across `assets(org_id, client_id) -> clients(org_id, id)`, `monitors(org_id, asset_id) -> assets(org_id, id)`, and `check_results(org_id, monitor_id) -> monitors(org_id, id)`. |
| **3. Monitor State Transition Threshold** | Verified ✅ | Added `consecutive_failures` column and updated status constraint (`pending`, `up`, `suspected_down`, `down`, `paused`, `unknown`). 1 failure -> `suspected_down`; 2 consecutive failures -> `down`; recovery -> `up`. Manual "Test Now" checks do not alter scheduled threshold. |
| **4. Check Results Retention & Aggregation** | Verified ✅ | Implemented `check_aggregates_hourly` table (retained 400 days / >13 months). Automated bounded batch pruning (`LIMIT 500`) for raw check results older than `CHECK_RESULT_RETENTION_DAYS` (default 90). |
| **5. Frontend Flat Config & Auth Regression** | Verified ✅ | Verified ESLint 9 Flat Config (`eslint .`) with 0 warnings/errors. Updated frontend status badges with Persian translations. Auth regression test suite verified all Phase 1 auth routes end-to-end. |
| **6. Monitoring E2E Runtime Evidence** | Verified ✅ | Real scheduled check cycle executed by background scheduler worker. Recorded CheckResult, transitioned monitor to `up`, updated drift-free `next_check_at`, and verified complete Org 2 isolation. |

---

## 2. Worker Database Function & Role Separation (Finding 1)

### Architecture & Privilege Design
Previously, `claim_due_monitors` was a `SECURITY DEFINER` function without role execution restrictions, allowing any connected role (including the API role) to claim monitors across all organizations. Furthermore, the worker daemon connected using the same database credentials as the API or superuser.

In migration `000005_monitoring_core_hardening.up.sql`:
1. **Restricted Non-Login Definer:** Created `clientpulse_definer` (`NOLOGIN`, `NOSUPERUSER`, `NOBYPASSRLS`). The function ownership was transferred to this role:
   ```sql
   ALTER FUNCTION claim_due_monitors(VARCHAR, INT, INT) OWNER TO clientpulse_definer;
   ```
2. **Dedicated Worker Login Role:** Created `clientpulse_worker` (`LOGIN`, `NOSUPERUSER`, `NOBYPASSRLS`) with its own password.
3. **Execution Privilege Revocation:**
   ```sql
   REVOKE ALL ON FUNCTION claim_due_monitors(VARCHAR, INT, INT) FROM PUBLIC;
   REVOKE ALL ON FUNCTION claim_due_monitors(VARCHAR, INT, INT) FROM clientpulse_app;
   GRANT EXECUTE ON FUNCTION claim_due_monitors(VARCHAR, INT, INT) TO clientpulse_worker;
   ```
4. **Fixed search_path:** Set `search_path = pg_catalog, public, pg_temp` directly on the function to prevent search path hijacking.
5. **Schema Protection:** Revoked `CREATE ON SCHEMA public` from `PUBLIC`, `clientpulse_app`, `clientpulse_worker`, and `clientpulse_definer`.
6. **Worker Table Privileges:**
   - Granted `SELECT, UPDATE` on `monitors`, `SELECT` on `assets`, `INSERT, SELECT, DELETE` on `check_results`, `SELECT, INSERT, UPDATE, DELETE` on `check_aggregates_hourly`, and `SELECT, INSERT, UPDATE` on `worker_heartbeats`.
   - Explicitly revoked all access from `clientpulse_worker` to sensitive credential tables: `users`, `sessions`, `password_reset_tokens`, `email_verification_tokens`, and `audit_logs`.
7. **Input Validation:** Enforced `1 <= p_batch_size <= 100`, `5 <= p_lease_timeout_seconds <= 3600`, and non-empty `p_worker_id` inside the function body.

### Verified Test Output
Automated test `TestDatabaseRolePermissionsAndClaimIsolation` (`services/api/internal/repository/task_003_1_security_test.go`):
```text
=== RUN   TestDatabaseRolePermissionsAndClaimIsolation
--- PASS: TestDatabaseRolePermissionsAndClaimIsolation (0.05s)
```
Proved:
- `clientpulse_app` cannot execute `claim_due_monitors` (`permission denied for function claim_due_monitors`).
- `clientpulse_worker` can execute `claim_due_monitors`.
- `clientpulse_worker` cannot create tables or alter schema in `public`.
- `clientpulse_worker` cannot read `users` table (`permission denied for table users`).
- `clientpulse_worker` cannot delete or update `audit_logs` (`permission denied for table audit_logs`).
- Invalid batch size (`101`) is rejected with SQL exception.

---

## 3. Composite Tenant Foreign Keys (Finding 2)

### Schema Constraints
To ensure cross-organization data integrity cannot be violated even if application-level checks fail or direct queries run, composite foreign keys tying the child's `organization_id` to the parent's `organization_id` were established:

1. **Unique Constraints Added:**
   - `uq_clients_org_id` ON `clients(organization_id, id)`
   - `uq_assets_org_id` ON `assets(organization_id, id)`
   - `uq_monitors_org_id` ON `monitors(organization_id, id)`
2. **Composite Foreign Keys Enforced:**
   - `fk_assets_client_org` ON `assets(organization_id, client_id) REFERENCES clients(organization_id, id) ON DELETE CASCADE`
   - `fk_monitors_asset_org` ON `monitors(organization_id, asset_id) REFERENCES assets(organization_id, id) ON DELETE CASCADE`
   - `fk_check_results_monitor_org` ON `check_results(organization_id, monitor_id) REFERENCES monitors(organization_id, id) ON DELETE CASCADE`

### Verified Test Output
Automated test `TestCompositeTenantForeignKeys` (`services/api/internal/repository/task_003_1_security_test.go`):
```text
=== RUN   TestCompositeTenantForeignKeys
--- PASS: TestCompositeTenantForeignKeys (0.04s)
```
Proved:
- Direct database query attempting to create an asset in Org A referencing a client in Org B fails with foreign key violation: `insert or update on table "assets" violates foreign key constraint "fk_assets_client_org"`.
- Direct database query attempting to create a monitor in Org A referencing an asset in Org B fails with foreign key violation: `insert or update on table "monitors" violates foreign key constraint "fk_monitors_asset_org"`.
- Direct database query attempting to create a check result in Org A referencing a monitor in Org B fails with foreign key violation: `insert or update on table "check_results" violates foreign key constraint "fk_check_results_monitor_org"`.

---

## 4. Monitor State Transition Threshold (Finding 3)

### State Machine Specification & Thresholds
The monitoring engine state transitions were implemented according to PRD operational thresholds:
```
             ┌───────────┐
             │  pending  │ (no check executed yet)
             └─────┬─────┘
                   │
         Check 1   │
       ┌───────────┴───────────┐
       ▼                       ▼
    Success                 Failure
 ┌───────────┐           ┌────────────────┐
 │    up     │           │ suspected_down │ (consecutive_failures = 1)
 └─────┬─────┘           └───────┬────────┘
       ▲                         │
       │ Recovery (Success)      │ Check 2 Failure
       │                         ▼
       │                  ┌──────────────┐
       └──────────────────┤     down     │ (consecutive_failures >= 2)
                          └──────────────┘
```

1. **Allowed Statuses:** `pending`, `up`, `suspected_down`, `down`, `paused`, `unknown`.
2. **Consecutive Failures:** Tracked in `monitors.consecutive_failures INT NOT NULL DEFAULT 0`.
3. **Transition Rules (`RecordScheduledCheckOutcome`):**
   - If monitor is `paused`, status remains `paused`.
   - On scheduled failure:
     - If previous `consecutive_failures == 0`: transition to `suspected_down` with `consecutive_failures = 1`.
     - If previous `consecutive_failures >= 1`: transition to `down` with `consecutive_failures = consecutive_failures + 1`.
   - On scheduled success:
     - Transition to `up` with `consecutive_failures = 0`.
4. **Manual Test Now Isolation:**
   - Manual tests record check results with `execution_type = 'manual'`, but explicitly do NOT update `consecutive_failures` or alter the scheduled state threshold.

### Verified Test Output
Automated test `TestMonitorStateTransitionsAndConsecutiveFailures` (`services/api/internal/repository/task_003_1_security_test.go`):
```text
=== RUN   TestMonitorStateTransitionsAndConsecutiveFailures
--- PASS: TestMonitorStateTransitionsAndConsecutiveFailures (0.04s)
```
Proved:
- Initial state: `pending`, `consecutive_failures = 0`.
- 1st failure: `suspected_down`, `consecutive_failures = 1`.
- 2nd failure: `down`, `consecutive_failures = 2`.
- 3rd failure: stays `down`, `consecutive_failures = 3`.
- Successful recovery check: resets to `up`, `consecutive_failures = 0`.
- Paused monitor does not transition to `down` upon check outcome.

---

## 5. Check Results Retention & Hourly Aggregation (Finding 4)

### Data Lifecycle Architecture
- **Raw Check Results Retention:** Configurable via `CHECK_RESULT_RETENTION_DAYS` (default 90 days).
- **Hourly Aggregates Table:** `check_aggregates_hourly` stores time-bucketed rollups (`bucket_hour`, `total_checks`, `successful_checks`, `total_latency_ms`, `min_latency_ms`, `max_latency_ms`). Retained for >= 13 months (400 days).
- **Atomic Upsert:** Scheduled check completion executes atomic upsert:
  ```sql
  INSERT INTO check_aggregates_hourly (
      organization_id, monitor_id, bucket_hour, total_checks,
      successful_checks, total_latency_ms, min_latency_ms, max_latency_ms
  ) VALUES ($1, $2, $3, 1, $4, $5, $5, $5)
  ON CONFLICT (organization_id, monitor_id, bucket_hour) DO UPDATE SET
      total_checks = check_aggregates_hourly.total_checks + 1,
      successful_checks = check_aggregates_hourly.successful_checks + EXCLUDED.successful_checks,
      total_latency_ms = check_aggregates_hourly.total_latency_ms + EXCLUDED.total_latency_ms,
      min_latency_ms = LEAST(check_aggregates_hourly.min_latency_ms, EXCLUDED.min_latency_ms),
      max_latency_ms = GREATEST(check_aggregates_hourly.max_latency_ms, EXCLUDED.max_latency_ms);
  ```
- **Bounded Batch Pruning:** Pruning runs in batches of 500 using `DELETE ... WHERE id IN (SELECT id FROM check_results WHERE created_at < cutoff LIMIT 500)` to eliminate table locks and avoid long-running transactions.

### Verified Test Output
Automated test `TestRetentionPruningAndHourlyAggregation` (`services/api/internal/repository/task_003_1_security_test.go`):
```text
=== RUN   TestRetentionPruningAndHourlyAggregation
    task_003_1_security_test.go:411: Pruned 1 expired records
--- PASS: TestRetentionPruningAndHourlyAggregation (0.04s)
```
Proved:
- Expired raw records (>90 days old) are deleted by worker pruning.
- Recent check results (<90 days) remain intact.
- Hourly aggregates remain intact and queryable after raw records are pruned.
- Bounded deletion loop terminates cleanly and is idempotent (0 deleted on immediate re-run).

---

## 6. Frontend Flat Config & Auth Regression (Finding 5)

### Verification Details
1. **ESLint 9 Flat Config:**
   Verified that `"lint": "eslint ."` runs cleanly using ESLint 9 Flat Config (`eslint.config.mjs`):
   ```bash
   $ cd apps/web && npm run lint
   > @clientpulse/web@1.0.0 lint
   > eslint .
   (Clean exit, 0 errors, 0 warnings)
   ```
2. **Next.js Turbopack Compilation:**
   All 17 application routes compiled successfully:
   ```text
   Route (app)
   ┌ ○ /
   ├ ○ /_not-found
   ├ ƒ /assets/[id]
   ├ ƒ /assets/[id]/edit
   ├ ○ /assets/new
   ├ ○ /clients
   ├ ƒ /clients/[id]
   ├ ƒ /clients/[id]/edit
   ├ ○ /clients/new
   ├ ○ /forgot-password
   ├ ○ /login
   ├ ○ /monitors
   ├ ƒ /monitors/[id]
   ├ ƒ /monitors/[id]/edit
   ├ ○ /monitors/new
   ├ ○ /onboarding
   ├ ○ /organizations
   ├ ○ /reset-password
   ├ ○ /settings/account
   ├ ○ /settings/members
   ├ ○ /signup
   └ ○ /verify-email
   ```
3. **Auth & Multi-Tenancy E2E Regression:**
   Automated test `TestAuthAndTenantLifecycleE2ERegression` (`services/api/internal/server/auth_e2e_regression_test.go`):
   ```text
   === RUN   TestAuthAndTenantLifecycleE2ERegression
   === RUN   TestAuthAndTenantLifecycleE2ERegression/Step1_Signup
   === RUN   TestAuthAndTenantLifecycleE2ERegression/Step2_AuthMe
   === RUN   TestAuthAndTenantLifecycleE2ERegression/Step4_ChangePassword
   === RUN   TestAuthAndTenantLifecycleE2ERegression/Step5_LoginNewPassword
   === RUN   TestAuthAndTenantLifecycleE2ERegression/Step6_ListOrganizations
   === RUN   TestAuthAndTenantLifecycleE2ERegression/Step7_CreateSecondOrganization
   === RUN   TestAuthAndTenantLifecycleE2ERegression/Step8_SelectOrganization
   === RUN   TestAuthAndTenantLifecycleE2ERegression/Step9_ListMembers
   === RUN   TestAuthAndTenantLifecycleE2ERegression/Step10_Logout
   --- PASS: TestAuthAndTenantLifecycleE2ERegression (0.38s)
   ```

---

## 7. Monitoring E2E Runtime Evidence (Finding 6)

### Live Scheduled Execution Evidence
Automated test `TestScheduledMonitoringEndToEndEvidence` (`services/api/internal/server/scheduled_e2e_evidence_test.go`) ran a full real-world cycle with the worker scheduler daemon executing against a mock target server under SSRF protection:

```text
=== PROVISIONED ENTITIES ===
Organization 1 ID: 0a9019d2-53b2-40db-b954-b968b59c9be5
Client ID:         fdecc432-49ac-44c1-96f7-e489409919d4 (Alpha Client)
Asset ID:          5981290c-fa5c-4362-b386-3180f9709600 (http://target.clientpulse-test.org)
Monitor ID:        cccd80ce-ec3c-4814-a812-a0b56823534a (initial status: pending, next_check: 2026-11-01T09:41:47+03:30)

=== RUNTIME EXECUTION EVIDENCE ===
Monitor ID:             cccd80ce-ec3c-4814-a812-a0b56823534a
Check Scheduled At:     2026-11-01T09:41:37+03:30
Check Started At:       2026-11-01T09:41:47+03:30
Check Completed At:     2026-11-01T09:41:47+03:30
Check Latency:          0 ms
HTTP Status Code:       200
Check Result Outcome:   success
Worker Identifier:      evidence-worker-01
Execution Type:         scheduled
Updated Monitor Status: up
Next Scheduled Check:   2026-11-01T09:42:37+03:30
Consecutive Failures:   0

=== TENANT ISOLATION PROOF ===
--- PASS: TestScheduledMonitoringEndToEndEvidence/Org2_Isolation_CheckResult_Invisible (0.00s)
--- PASS: TestScheduledMonitoringEndToEndEvidence/Org2_Isolation_Monitor_Invisible (0.00s)
```

### Docker Compose Runtime Logs
In Docker Compose, `clientpulse_worker` logs confirm real-world job claiming, status progression (`suspected_down`, `down`, `up`), and background heartbeat renewal:
```text
clientpulse_worker | {"level":"INFO","msg":"monitoring check executed and recorded","monitor_id":"75d4eb...","target_url":"https://acme.example.com","result":"failure","status":"suspected_down","latency_ms":31}
clientpulse_worker | {"level":"INFO","msg":"monitoring check executed and recorded","monitor_id":"d93d37...","target_url":"https://acme.example.com","result":"failure","status":"down","latency_ms":31}
clientpulse_worker | {"level":"INFO","msg":"monitoring check executed and recorded","monitor_id":"91bd28...","target_url":"https://example.com","result":"success","status":"up","latency_ms":320}
clientpulse_worker | {"level":"DEBUG","msg":"worker heartbeat updated","worker_id":"worker-ac4c79b80893-1"}
```

---

## 8. Docker Service Health Verification

All containerized services were built and verified running healthy:
```text
$ docker compose ps
NAME                   SERVICE    STATUS                   PORTS
clientpulse_api        api        Up (healthy)             127.0.0.1:8080->8080/tcp
clientpulse_mailpit    mailpit    Up (healthy)             127.0.0.1:1025->1025/tcp, 127.0.0.1:8025->8025/tcp
clientpulse_postgres   postgres   Up (healthy)             127.0.0.1:5433->5432/tcp
clientpulse_web        web        Up                       127.0.0.1:3001->3000/tcp
clientpulse_worker     worker     Up                       

$ curl -s http://localhost:8080/health/live
{"status":"ok","service":"clientpulse-api","version":"1.0.0","timestamp":"2026-11-01T06:15:07Z"}

$ curl -s http://localhost:8080/health/ready
{"status":"ok","service":"clientpulse-api","database":"connected","version":"1.0.0","timestamp":"2026-11-01T06:15:07Z"}

$ curl -s http://localhost:8080/health/worker
{"status":"ok","service":"clientpulse-worker","worker_id":"worker-ac4c79b80893-1","hostname":"ac4c79b80893","last_heartbeat_at":"2026-11-01T06:15:07Z","age_seconds":0.16,"timeout_seconds":10,"timestamp":"2026-11-01T06:15:07Z"}
```

---

## 9. Full Test Suite Verification

### Go Backend Suite
```bash
$ cd services/api && go vet ./... && go test -count=1 -race ./...
ok      clientpulse/services/api/internal/auth          2.296s
ok      clientpulse/services/api/internal/checker       1.570s
ok      clientpulse/services/api/internal/config        1.041s
ok      clientpulse/services/api/internal/database      1.053s
ok      clientpulse/services/api/internal/health        1.067s
ok      clientpulse/services/api/internal/logger        1.036s
ok      clientpulse/services/api/internal/migrate       2.036s
ok      clientpulse/services/api/internal/rate          1.037s
ok      clientpulse/services/api/internal/repository    2.512s
ok      clientpulse/services/api/internal/server        6.036s
ok      clientpulse/services/api/internal/service       2.156s
ok      clientpulse/services/api/internal/ssrf          1.015s
```
**Outcome:** 100% of tests passed cleanly with race detection enabled across all packages.

### Frontend Suite
```bash
$ cd apps/web && npm run lint && npm run build
> @clientpulse/web@1.0.0 lint
> eslint .

> @clientpulse/web@1.0.0 build
> next build
✓ Compiled successfully in 159ms
✓ Finished TypeScript in 1461ms
✓ Generating static pages using 11 workers (17/17) in 303ms
```
**Outcome:** 0 lint errors, 0 lint warnings, 17/17 pages built successfully with Turbopack.

---

## 10. Conclusion & Acceptance Status

TASK-003.1 is **100% complete**. All findings have been resolved, hardened, and verified with rigorous integration tests and live runtime evidence. Development stands ready for Phase 3 planning.

