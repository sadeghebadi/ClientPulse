# TASK-INCIDENT-002: Persist Production Scheduler RLS Fix in Database Migrations

**Project:** ClientPulse  
**Priority:** Critical — Production Reliability & Database Security  
**Execution Environment:** Local Development Only (Zero Production Deployment / Zero Production Contact)  
**Related Incident:** TASK-INCIDENT-001  
**Reference Report:** `task-report/TASK-INCIDENT-001-MONITOR-SCHEDULER.md`  
**Execution Date:** 2026-11-02  

---

## 1. Executive Summary

During TASK-INCIDENT-001, a P0 production monitoring failure was diagnosed where automatic checks for newly registered monitors remained indefinitely in `Pending First Check`. The root cause was identified as a missing PostgreSQL Row Level Security (RLS) policy on `public.assets` for role `clientpulse_definer`, which prevented the background scheduler function `claim_due_monitors()` from discovering due monitors across tenants when running without an active tenant HTTP session.

An emergency hotfix was applied directly to the production database:
```sql
CREATE POLICY assets_definer_policy ON public.assets
    FOR ALL
    TO clientpulse_definer
    USING (true)
    WITH CHECK (true);
```
While this successfully restored monitoring in production, the emergency policy had two operational defects:
1. It was not persisted in version-controlled migrations (risking regression on future migrations, fresh deployments, or disaster recovery).
2. It was overly permissive (`FOR ALL` / `WITH CHECK (true)`), granting write capabilities when only read access was needed.

Under TASK-INCIDENT-002, this fix has been systematically reviewed, hardened to least-privilege `FOR SELECT`, captured in a new forward migration (`000013_assets_scheduler_rls_hardening`), reconciled idempotently against existing environments, and verified across automated regression and reproducibility test suites.

---

## 2. Pre-Implementation Inspection Baseline

### 2.1 Migration State
- **Highest repository migration version:** `000012_notification_message_quality` (both `.up.sql` and `.down.sql` present).
- **Highest applied version in test database:** Version `12`.
- **Target new migration version:** `000013_assets_scheduler_rls_hardening`.

### 2.2 Table RLS Policies on `assets` and `monitors`
| Table | Policy Name | Command | Role | Expression (`USING`) |
|---|---|---|---|---|
| `public.monitors` | `monitors_tenant_isolation` | ALL | `public` | `(organization_id = (NULLIF(current_setting('app.current_organization_id', true), ''))::uuid)` |
| `public.monitors` | `monitors_definer_policy` | ALL | `clientpulse_definer` | `true` |
| `public.assets` | `assets_tenant_isolation` | ALL | `public` | `(organization_id = (NULLIF(current_setting('app.current_organization_id', true), ''))::uuid)` |
| `public.assets` (Hotfix) | `assets_definer_policy` | ALL | `clientpulse_definer` | `true` (with `WITH CHECK (true)`) |

Both tables have `FORCE ROW LEVEL SECURITY` enabled.

---

## 3. Critical Security Review & Least-Privilege Policy

### 3.1 Security Definer Functions Audit
An audit of all 8 `SECURITY DEFINER` functions in the database was conducted:
1. `claim_due_monitors(p_worker_id VARCHAR, p_batch_size INT, p_lease_timeout_seconds INT)`:
   - **Table Access:** Reads `monitors` and `assets`, writes `monitors` (lease acquisition).
   - **Operation on `assets`:** `JOIN public.assets a ON m.asset_id = a.id` — purely `SELECT`.
   - **Privileges:** `REVOKE ALL FROM PUBLIC; GRANT EXECUTE TO clientpulse_worker;`
   - **`search_path`:** Hardened to `pg_catalog, public, pg_temp`.
   - **Owner:** `clientpulse_definer`.
2. `lookup_heartbeat_by_token_hash`: Reads `heartbeat_monitors`, `organizations`, `clients` (`search_path=pg_catalog, public, pg_temp`). Does not touch `assets`.
3. `lookup_report_by_share_token_hash`: Reads `report_share_tokens`, `reports`, `organizations` (`search_path=pg_catalog, public, pg_temp`). Does not touch `assets`.
4. `claim_missed_heartbeats`: Reads/updates `heartbeat_monitors`. Does not touch `assets`.
5. `claim_due_notification_deliveries`: Reads/updates `notification_deliveries`. Does not touch `assets`.
6. `consume_notification_connection_token`: Reads/updates `notification_connection_tokens`. Does not touch `assets`.
7. `consume_email_verification_token`: Reads/updates tokens. Does not touch `assets`.
8. `find_active_group_destination_by_chat_id`: Reads destinations. Does not touch `assets`.

### 3.2 Least-Privilege Determination
- **Audit Conclusion:** `claim_due_monitors()` is the **only** `SECURITY DEFINER` function that accesses `assets`, and it requires **only** `SELECT` access to resolve target URLs (`COALESCE(u.url_override, a.url)`).
- **Final Policy Definition:**
  ```sql
  CREATE POLICY assets_definer_policy ON public.assets
      FOR SELECT
      TO clientpulse_definer
      USING (true);
  ```
- **Security Boundary:**
  - `clientpulse_definer` has no `INSERT`, `UPDATE`, or `DELETE` capabilities on `assets`.
  - Normal application requests executing as `clientpulse_app` remain strictly isolated by `assets_tenant_isolation`.
  - Direct queries by `clientpulse_worker` without a tenant context still see 0 rows because `clientpulse_worker` itself has no bypass policy.
  - Cross-tenant access is restricted exclusively to execution within the security-definer function `claim_due_monitors()`.

---

## 4. Forward Migration Implementation

### 4.1 Up Migration
**File:** [migrations/000013_assets_scheduler_rls_hardening.up.sql](file:///home/sadpc/Documents/ClientPulse/migrations/000013_assets_scheduler_rls_hardening.up.sql)
```sql
-- Migration: 000013_assets_scheduler_rls_hardening.up.sql
-- Description: Establish least-privilege SELECT RLS policy on assets for clientpulse_definer (TASK-INCIDENT-001/002)

-- Drop any existing policy with the target name (including emergency hotfix 'FOR ALL' or temporary variants)
DROP POLICY IF EXISTS assets_definer_policy ON public.assets;

-- Reconcile and clean up any dev/staging alias if present
DROP POLICY IF EXISTS assets_definer_select ON public.assets;

-- Create least-privilege SELECT-only policy for security definer background procedures (e.g. claim_due_monitors)
CREATE POLICY assets_definer_policy ON public.assets
    FOR SELECT
    TO clientpulse_definer
    USING (true);
```

### 4.2 Down Migration & Rollback Warning
**File:** [migrations/000013_assets_scheduler_rls_hardening.down.sql](file:///home/sadpc/Documents/ClientPulse/migrations/000013_assets_scheduler_rls_hardening.down.sql)
```sql
-- Migration: 000013_assets_scheduler_rls_hardening.down.sql
-- Description: Revert least-privilege SELECT RLS policy on assets for clientpulse_definer
-- WARNING / ROLLBACK CONSEQUENCE:
-- Reverting this migration removes the assets policy for clientpulse_definer.
-- In a database running the background monitor scheduler, this will cause claim_due_monitors()
-- to return 0 rows for all monitors joining assets under background worker connections without
-- an HTTP tenant context, reproducing the TASK-INCIDENT-001 monitoring failure.

DROP POLICY IF EXISTS assets_definer_policy ON public.assets;
```

### 4.3 Safe Production Reconciliation
Because the live production database contains the emergency hotfix policy created during TASK-INCIDENT-001, migration `000013` uses:
1. `DROP POLICY IF EXISTS assets_definer_policy ON public.assets;`
2. `CREATE POLICY assets_definer_policy ON public.assets FOR SELECT TO clientpulse_definer USING (true);`

This guarantees that:
- In environments where the hotfix was already applied (Production), the policy is seamlessly downgraded from `FOR ALL` to least-privilege `FOR SELECT` without error.
- In fresh environments (new deployments, CI, local dev), the policy is created cleanly.

---

## 5. Automated Regression & Integration Tests

### 5.1 Test Suite 1: Regression Test Suite
**File:** [services/api/internal/repository/task_incident_002_regression_test.go](file:///home/sadpc/Documents/ClientPulse/services/api/internal/repository/task_incident_002_regression_test.go)

Includes all mandatory test cases:
- **Reproduction of Original Failure:** Inside an isolated transaction, dropping `assets_definer_policy` causes `claim_due_monitors()` to return 0 rows for due monitors.
- **Test A & Test B (Cross-Tenant Discovery):** Simulates background worker execution without `app.current_organization_id`. Proves that monitors belonging to `Org A` and `Org B` are both discovered, target URLs resolved, and claimed across tenant boundaries.
- **Test C (Normal Tenant Isolation):** Verifies that `clientpulse_app` without tenant context cannot see any assets (0 rows), and with `app.current_organization_id = Org A`, only Org A's assets are visible. Org B's assets are strictly inaccessible.
- **Test D (Direct Role Restrictions):** Verifies that `clientpulse_worker` direct queries on `assets` return 0 rows without tenant context, and verifies `assets_definer_policy` is restricted to `cmd = 'SELECT'`.
- **Test E (Scheduler Compatibility):** Verifies atomic lease acquisition, `SKIP LOCKED` concurrency safety (concurrent worker claims skip already leased monitors), and check outcome recording.

**Execution Result:**
```
=== RUN   TestIncident002_SchedulerRLSFix
=== RUN   TestIncident002_SchedulerRLSFix/Reproduction_Original_Failure_Without_Assets_Definer_Policy
    task_incident_002_regression_test.go:122: CONFIRMED: Without assets_definer_policy, claim_due_monitors returned 0 rows for due monitor b309cd93-605d-4f9b-a678-79bd35479493
=== RUN   TestIncident002_SchedulerRLSFix/TestA_and_TestB_CrossTenant_Scheduler_Claiming_Without_Tenant_Context
    task_incident_002_regression_test.go:159: Successfully claimed monitor b309cd93-605d-4f9b-a678-79bd35479493 for org 4b5c0610-2b25-4419-b901-f46cbfb527f5 (target=https://api-alpha.example.com)
    task_incident_002_regression_test.go:159: Successfully claimed monitor 1fd4b753-e62f-4c92-bc4f-b7afacbfcea9 for org a318354c-fb14-4dbe-b125-0d44e9edbe82 (target=https://api-beta.example.com)
=== RUN   TestIncident002_SchedulerRLSFix/TestC_Normal_Tenant_Isolation_Remains_Intact
=== RUN   TestIncident002_SchedulerRLSFix/TestD_Direct_Role_Restrictions
=== RUN   TestIncident002_SchedulerRLSFix/TestE_Scheduler_Compatibility_And_Lease_Acquisition
--- PASS: TestIncident002_SchedulerRLSFix (0.12s)
PASS
```

### 5.2 Test Suite 2: Migration Reproducibility Suite
**File:** [services/api/internal/migrate/task_incident_002_reproducibility_test.go](file:///home/sadpc/Documents/ClientPulse/services/api/internal/migrate/task_incident_002_reproducibility_test.go)

- **Scenario 1 (Fresh Database):** Creates a completely new database, applies the full migration chain (000001 through 000013), validates policy presence (`cmd='SELECT'`), verifies `assets_tenant_isolation` is preserved, executes rollback (Down to version 12), and verifies re-application (Up to version 13).
- **Scenario 2 (Existing Database with Emergency Policy):** Creates a database, applies migrations 1–12, injects the exact emergency policy `FOR ALL TO clientpulse_definer USING (true) WITH CHECK (true)`, applies migration 13, and verifies that the policy is seamlessly reconciled to `cmd='SELECT'` without duplication or failure.

**Execution Result:**
```
=== RUN   TestMigration0013_Reproducibility
=== RUN   TestMigration0013_Reproducibility/Scenario1_Fresh_Database_Migration_Chain
    task_incident_002_reproducibility_test.go:89: Fresh DB migrations applied: 13 migrations
=== RUN   TestMigration0013_Reproducibility/Scenario2_Existing_Database_With_Emergency_Policy
--- PASS: TestMigration0013_Reproducibility (1.23s)
    --- PASS: TestMigration0013_Reproducibility/Scenario1_Fresh_Database_Migration_Chain (0.65s)
    --- PASS: TestMigration0013_Reproducibility/Scenario2_Existing_Database_With_Emergency_Policy (0.58s)
PASS
```

### 5.3 Full Integration Suite Status
All 22 packages across `services/api` passed cleanly:
```
ok      clientpulse/services/api/internal/migrate       1.879s
ok      clientpulse/services/api/internal/repository    4.705s
ok      clientpulse/services/api/internal/server        (cached)
ok      clientpulse/services/api/internal/worker        (cached)
... All packages PASS
```

---

## 6. Strict Production Boundary Compliance

In accordance with TASK-INCIDENT-002 instructions:
- **Zero SSH / VPS connections made.**
- **No changes made to the live Production database or containers.**
- **The live production hotfix policy remains active and untouched.**
- **Zero deployments performed.**
- **Local working-tree state was preserved cleanly.**

---

## 7. Acceptance Criteria Verification

- [x] **The original RLS failure is reproduced by a regression test** (`TestIncident002_SchedulerRLSFix/Reproduction_Original_Failure_Without_Assets_Definer_Policy` passes).
- [x] **A new forward migration is created** (`000013_assets_scheduler_rls_hardening.up.sql` and `.down.sql`).
- [x] **Historical migrations remain unchanged** (000001 through 000012 are untouched).
- [x] **The migration handles the existing emergency policy safely** (Idempotent `DROP POLICY IF EXISTS ... CREATE POLICY ... FOR SELECT`).
- [x] **The final policy follows least privilege** (`FOR SELECT TO clientpulse_definer USING (true)`).
- [x] **All relevant SECURITY DEFINER functions are reviewed** (Audit of all 8 functions confirmed `claim_due_monitors()` is the sole function accessing `assets`).
- [x] **Function EXECUTE grants are appropriate** (`clientpulse_worker` granted, `clientpulse_app` denied, `PUBLIC` denied).
- [x] **Function `search_path` is hardened** (`pg_catalog, public, pg_temp`).
- [x] **Cross-tenant scheduled-monitor discovery works** (Verified across Org A and Org B without tenant session).
- [x] **Normal tenant isolation remains intact** (`clientpulse_app` isolated by `assets_tenant_isolation`).
- [x] **Fresh database migration chain succeeds** (Scenario 1 test passed).
- [x] **Existing-emergency-policy migration test succeeds** (Scenario 2 reconciliation test passed).
- [x] **All relevant database tests pass** (All repository and migration tests passed).
- [x] **No production infrastructure was modified** (Work confined strictly to local development).
- [x] **Full implementation report is saved** (Saved to `task-report/TASK-INCIDENT-002-RLS-MIGRATION.md`).

