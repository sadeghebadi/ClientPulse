# TASK-INCIDENT-004: Incident Lifecycle, Acknowledgement, Resolution & Heartbeat Deletion

**Project:** ClientPulse  
**Priority:** High — Core Monitoring & Data Integrity  
**Execution Environment:** Local Development Only  
**Production Deployment:** Strictly Prohibited  
**Base Release:** v1.0.2  
**Database Migration:** `000017_incident_lifecycle_resolution`  
**Related Tasks:** TASK-INCIDENT-001, TASK-INCIDENT-002, TASK-INCIDENT-003, TASK-008.B.2, TASK-008.B.3  
**Status:** Completed & Fully Verified  

---

## 1. Existing Lifecycle Audit

Before making code alterations, an exhaustive inspection of the ClientPulse codebase and historical database migrations (000001 through 000016) was performed:

### Findings
1. **Schema Deficiencies (`000006_create_incidents.up.sql`):**
   - The `incidents` table had `status VARCHAR(32) NOT NULL DEFAULT 'open'` checked against `('open', 'resolved')`.
   - Columns existed for `acknowledged_at TIMESTAMPTZ` and `acknowledged_by UUID REFERENCES users(id)`.
   - **Critical Gap:** The schema lacked columns to record *who* resolved an incident (`resolved_by`), *how* or *why* it was resolved (`resolution_type`), or any contextual notes (`resolution_note`).
2. **Conflation of Acknowledgement and Resolution:**
   - In earlier iterations, operators lacked a distinct workflow for manual closure. Acknowledging an incident recorded an operator ID, but there was no standardized way to manually mark an incident as resolved without relying on the automated checker.
3. **Heartbeat Deletion Semantics & Cascade Risk:**
   - Inspection of `heartbeat_repo.go` revealed that heartbeat monitors utilize soft deletion (`archived_at TIMESTAMPTZ`), never raw `DELETE`.
   - However, in `000006_create_incidents.up.sql`, the foreign key `fk_incidents_heartbeat_org` was configured as:
     `FOREIGN KEY (organization_id, heartbeat_monitor_id) REFERENCES heartbeat_monitors(organization_id, id) ON DELETE CASCADE`.
     If a hard delete was ever triggered or foreign key cascaded, all historical incident records would be destroyed.
   - Furthermore, when `ArchiveHeartbeat` soft-deleted a monitor (`archived_at = NOW()`), it did **not** resolve any active open incidents referencing that heartbeat. As a consequence, open incidents remained indefinitely in the database, pointing to an archived monitor and corrupting dashboard outage counters.
4. **Dashboard Aggregates:**
   - `check_result_repo.go` aggregated incidents simply as `COUNT(*) FILTER (WHERE status = 'open') AS open_incidents`.
   - It did not differentiate website outages from missed heartbeats, nor did it track unacknowledged vs. acknowledged counts.

---

## 2. Original Acknowledgement Behavior and Its Correction

### Original Problem
- When an operator acknowledged an incident, there was ambiguity regarding whether the incident was considered "closed" or remained actively failing.
- Several client queries and UI filters conflated acknowledged incidents with closed incidents.

### Corrected Behavior
- **Acknowledge != Resolve:**
  - An acknowledgement means: *"An operator has acknowledged responsibility for tracking this incident."*
  - It does **not** imply that the target has recovered or that tracking has ceased.
- When an operator acknowledges an open incident:
  1. `acknowledged_at` is set to `NOW()`.
  2. `acknowledged_by` is set to the authenticated user's ID.
  3. `status` remains strictly `'open'`.
  4. Failure state, duration calculation, and diagnostic snapshots remain intact.
  5. The incident is included in **Open Incident** counts and excluded from **Unacknowledged** counts.
- **Idempotency & Guardrails:**
  - Repeated acknowledgement returns `ErrIncidentAlreadyAcked` without mutating the original `acknowledged_at` timestamp.
  - Attempting to acknowledge an already resolved incident returns `ErrIncidentAlreadyResolved` (HTTP 400 `ALREADY_RESOLVED`).

---

## 3. Manual Resolution Behavior

### Implementation
- Added explicit manual resolution API and repository capability:
  - Repository: `ResolveIncidentManually(ctx, orgID, incidentID, resolvedBy, note)`
  - HTTP Handler: `POST /api/v1/incidents/{id}/resolve`
- **Mutations:**
  - `status = 'resolved'`
  - `resolved_at = NOW()`
  - `resolved_by = resolvedBy` (Authenticated operator)
  - `resolution_type = 'manual'`
  - `resolution_note = TRIM(note)` (Optional, sanitized string)
- **Persistent Notification Event & Audit Trail:**
  - Emits notification event `incident.resolved` with payload:
    `{"resolution_type": "manual", "resolved_by": resolvedBy, "note": note}`.
  - Appends an audit log entry with action `incident.resolve_manual` and actor attribution.
- **Handling Subsequent Check Failures:**
  - If a website or API monitor continues to fail on its next scheduled check after being manually resolved, the monitor check engine (`RecordScheduledCheckOutcomeWithResult`) recognizes that no open incident currently exists for that monitor.
  - Because `consecutive_failures` remains `>= threshold`, the system automatically opens a **new** incident with its own distinct lifecycle and links to diagnostic check results. The manual resolution history is preserved without masking persistent downtime.

---

## 4. Automatic Website Recovery Behavior

### Implementation in `RecordScheduledCheckOutcomeWithResult`
1. When a scheduled monitor check satisfies all configured assertions and passes:
   - The monitor state machine resets `consecutive_failures = 0`.
   - If an active open incident exists (`status = 'open'`), it is atomically resolved within the tenant transaction:
     - `status = 'resolved'`
     - `resolved_at = completed_at` (actual check completion timestamp)
     - `resolved_by = NULL` (system action)
     - `resolution_type = 'automatic_recovery'`
     - `recovery_result_id = result.ID`
2. **Idempotency & Concurrency:**
   - Resolving uses `WHERE organization_id = $1 AND monitor_id = $2 AND status = 'open'`.
   - If two checks complete concurrently, row-level locking ensures only one transition occurs.
   - Emits `incident.resolved` notification event and records `incident.resolve_automatic` in audit logs.

---

## 5. Automatic Heartbeat Recovery Behavior

### Implementation in `RecordHeartbeatPing`
1. When a heartbeat ping is received:
   - Validates the secret token against stored `token_hash` using constant-time HMAC comparison.
   - Updates heartbeat status to `'healthy'` and records `last_received_at`.
   - Locates any open incident for this heartbeat monitor (`heartbeat_monitor_id = $1 AND status = 'open'`).
   - Atomically updates:
     - `status = 'resolved'`
     - `resolved_at = received_at`
     - `resolved_by = NULL` (system action)
     - `resolution_type = 'automatic_recovery'`
2. **Security & Isolation:**
   - Unauthenticated or invalid token requests return HTTP 404/401 and **never** resolve incidents.
   - Pings for one heartbeat monitor can never affect another monitor's incidents due to strict composite tenant and monitor ID scoping.

---

## 6. Heartbeat Deletion Behavior

### Implementation in `ArchiveHeartbeatWithActor`
1. **Soft-Delete Mechanism:**
   - When a user deletes a heartbeat monitor via `DELETE /api/v1/heartbeats/{id}`:
     - Acquires `FOR UPDATE` lock on `heartbeat_monitors`.
     - Sets `archived_at = NOW()`.
2. **Atomic Resolution of Associated Open Incidents:**
   - Queries all open incidents for the target heartbeat within the same transaction:
     `SELECT id, first_failure_at FROM incidents WHERE organization_id = $1 AND heartbeat_monitor_id = $2 AND status = 'open' FOR UPDATE`.
   - For each open incident:
     - `status = 'resolved'`
     - `resolved_at = NOW()`
     - `resolved_by = actorUserID`
     - `resolution_type = 'source_monitor_deleted'`
     - `resolution_note = 'Automatically resolved upon source monitor deletion'`
     - `duration_seconds = EXTRACT(EPOCH FROM (NOW() - first_failure_at))`
   - Emits `incident.resolved` event with `resolution_type: "source_monitor_deleted"`.
   - Writes audit log entries:
     - `incident.resolve_source_deleted` for each resolved incident.
     - `heartbeat.archive` for the monitor.
3. **Repeated / Concurrent Deletion:**
   - Calling delete on an already archived monitor is idempotent: returns `0` resolved incidents and does not corrupt state.

---

## 7. Referential Integrity and Transaction Design

### Foreign Key Hardening
- Previously, `fk_incidents_heartbeat_org` had `ON DELETE CASCADE`.
- In Migration `000017`:
  ```sql
  ALTER TABLE incidents DROP CONSTRAINT IF EXISTS fk_incidents_heartbeat_org;
  ALTER TABLE incidents ADD CONSTRAINT fk_incidents_heartbeat_org
      FOREIGN KEY (organization_id, heartbeat_monitor_id)
      REFERENCES heartbeat_monitors(organization_id, id)
      ON DELETE RESTRICT;
  ```
- `ON DELETE RESTRICT` guarantees that even if a manual SQL operation attempts a hard delete on `heartbeat_monitors`, Postgres blocks it, preventing catastrophic cascading deletion of historical incident records.
- All lifecycle operations execute within `WithTenantTx(ctx, pool, orgID, ...)` ensuring strict atomic commits.

---

## 8. Resolution Metadata and Audit Events

### Schema Columns in `incidents` (Migration 000017)
| Column Name | Type | Constraints | Purpose |
|---|---|---|---|
| `resolved_by` | `UUID` | `REFERENCES users(id) ON DELETE SET NULL` | Operator who resolved or deleted the monitor |
| `resolution_type` | `VARCHAR(64)` | `chk_incident_resolution_type` | `'automatic_recovery'`, `'manual'`, `'source_monitor_deleted'` |
| `resolution_note` | `TEXT` | `NULL` | Optional operator reason or diagnostic details |

### Check Constraint
```sql
ALTER TABLE incidents ADD CONSTRAINT chk_incident_resolution_type
    CHECK (resolution_type IS NULL OR resolution_type IN (
        'automatic_recovery',
        'manual',
        'source_monitor_deleted'
    ));
```

### Audit Log Schema & Events
- `incident.resolve_manual`: Records `incident_id`, `resolved_by`, `resolution_type = manual`, and `note`.
- `incident.resolve_automatic`: Records `incident_id`, `resolution_type = automatic_recovery`, and `source_type`.
- `incident.resolve_source_deleted`: Records `incident_id`, `heartbeat_monitor_id`, `deleted_by`, and `resolution_type = source_monitor_deleted`.

---

## 9. Incident Count and Aggregate Semantics

### SQL Query in `GetDashboardStats` (`check_result_repo.go`)
```sql
SELECT
    COUNT(*) FILTER (WHERE status = 'open') AS open_incidents,
    COUNT(*) FILTER (WHERE status = 'open' AND acknowledged_at IS NULL) AS unacknowledged_incidents,
    COUNT(*) FILTER (WHERE status = 'open' AND acknowledged_at IS NOT NULL) AS acknowledged_incidents,
    COUNT(*) FILTER (WHERE status = 'resolved') AS resolved_incidents,
    COUNT(*) FILTER (WHERE status = 'open' AND monitor_id IS NOT NULL) AS website_outage_incidents,
    COUNT(*) FILTER (WHERE status = 'open' AND heartbeat_monitor_id IS NOT NULL) AS heartbeat_missed_incidents
FROM incidents
WHERE organization_id = $1
```

### Aggregates Truth Table
- $\text{open\_incidents} = \text{unacknowledged\_incidents} + \text{acknowledged\_incidents}$
- $\text{open\_incidents} = \text{website\_outage\_incidents} + \text{heartbeat\_missed\_incidents}$
- Incidents resolved via `source_monitor_deleted` immediately drop $\text{open\_incidents}$ and $\text{heartbeat\_missed\_incidents}$ to 0 while incrementing $\text{resolved\_incidents}$.

---

## 10. API Contract Changes

### 1. `POST /api/v1/incidents/{id}/resolve` (New Endpoint)
- **Role Requirement:** `RoleMember` (Member, Admin, Owner)
- **Request Body (Optional):**
  ```json
  {
    "note": "Operator manually verified server recovery."
  }
  ```
- **Success Response (200 OK):**
  ```json
  {
    "status": "success",
    "incident": {
      "id": "c1f7b920-...",
      "status": "resolved",
      "resolution_type": "manual",
      "resolution_note": "Operator manually verified server recovery.",
      "resolved_by": "u-456-...",
      "resolved_at": "2026-11-03T10:15:00Z"
    }
  }
  ```
- **Error Responses:**
  - `400 Bad Request`: `ALREADY_RESOLVED` if incident is not open.
  - `403 Forbidden`: `FORBIDDEN` if viewer role or cross-tenant.
  - `404 Not Found`: `NOT_FOUND` if incident does not exist.

### 2. `POST /api/v1/incidents/{id}/acknowledge` (Updated)
- Now returns `400 Bad Request` with `ALREADY_RESOLVED` if called on an already resolved incident.
- Retains `status = "open"`.

### 3. `GET /api/v1/incidents` (Updated Query Parameters)
- Supported filters:
  - `status`: `all`, `open`, `resolved`
  - `target_type`: `all`, `monitor`, `heartbeat`
  - `ack_state`: `all`, `unack`, `ack`
- Response includes `resolved_by`, `resolved_by_name`, `resolution_type`, `resolution_note`, and `heartbeat_archived_at`.

### 4. `GET /api/v1/incidents/{id}` (Updated Detail Payload)
- Includes full resolution metadata: `resolved_by`, `resolved_by_name`, `resolved_by_email`, `resolution_type`, `resolution_note`, and `heartbeat_archived_at`.

---

## 11. Files Created and Modified

### Created Files
1. `migrations/000017_incident_lifecycle_resolution.up.sql` (45 lines) — Migration adding resolution columns, check constraints, indexes, and hardened FK.
2. `migrations/000017_incident_lifecycle_resolution.down.sql` (32 lines) — Rollback script restoring cascade and dropping added columns.
3. `services/api/internal/repository/task_incident_004_lifecycle_test.go` (465 lines) — 10 lifecycle test suites (A through J) testing all edge cases.
4. `services/api/internal/server/task_incident_004_server_test.go` (220 lines) — HTTP endpoint tests for resolution, heartbeat deletion, and filters.
5. `apps/web/tests/task-incident-004-screenshots.js` (160 lines) — Playwright visual test script generating all report screenshots.
6. `task-report/TASK-INCIDENT-004-IMPLEMENTATION.md` (this report).

### Modified Files
1. `services/api/internal/repository/models.go` — Added resolution fields to `Incident` and expanded `DashboardStats`.
2. `services/api/internal/repository/incident_repo.go` — Added `ResolveIncidentManually`, updated `AcknowledgeIncident`, `ResolveIncidentTx`, `GetIncidentByID`, `ListIncidents`.
3. `services/api/internal/repository/heartbeat_repo.go` — Added `ArchiveHeartbeatWithActor` with atomic incident resolution.
4. `services/api/internal/repository/monitor_repo.go` — Updated automatic recovery in `RecordScheduledCheckOutcomeWithResult`.
5. `services/api/internal/repository/check_result_repo.go` — Updated `GetDashboardStats` with distinct incident aggregates.
6. `services/api/internal/server/incident_handlers.go` — Added `Resolve` endpoint handler and updated `List` and `Acknowledge`.
7. `services/api/internal/server/heartbeat_handlers.go` — Updated `Delete` handler to pass actor ID to `ArchiveHeartbeatWithActor`.
8. `services/api/internal/server/server.go` — Registered route `POST /api/v1/incidents/:id/resolve`.
9. `services/api/internal/health/task_incident_003_3_audit_test.go` — Adjusted migration 16/17 rollback test assertions.
10. `apps/web/src/lib/i18n.tsx` — Added bilingual translations for acknowledgement, resolution, and monitor deletion notices.
11. `apps/web/src/components/ui/Badge.tsx` — Added badge variants for `source_monitor_deleted`, `monitor_deleted`, `manual`, and `automatic_recovery`.
12. `apps/web/src/app/incidents/page.tsx` — Added manual resolution modal, resolution badges, deleted monitor indicator, and server query filters.
13. `apps/web/src/app/incidents/[id]/page.tsx` — Added manual resolution button & modal, deleted monitor banner, resolution note display, and safe links.
14. `apps/web/src/app/heartbeats/[id]/page.tsx` — Updated delete confirmation modal with clear bilingual explanation.

---

## 12. Database Migrations

### Forward Migration: `migrations/000017_incident_lifecycle_resolution.up.sql`
```sql
ALTER TABLE incidents
    ADD COLUMN IF NOT EXISTS resolved_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS resolution_type VARCHAR(64),
    ADD COLUMN IF NOT EXISTS resolution_note TEXT;

ALTER TABLE incidents DROP CONSTRAINT IF EXISTS chk_incident_resolution_type;
ALTER TABLE incidents ADD CONSTRAINT chk_incident_resolution_type
    CHECK (resolution_type IS NULL OR resolution_type IN (
        'automatic_recovery',
        'manual',
        'source_monitor_deleted'
    ));

ALTER TABLE incidents DROP CONSTRAINT IF EXISTS fk_incidents_heartbeat_org;
ALTER TABLE incidents ADD CONSTRAINT fk_incidents_heartbeat_org
    FOREIGN KEY (organization_id, heartbeat_monitor_id)
    REFERENCES heartbeat_monitors(organization_id, id)
    ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_incidents_resolution_type ON incidents(resolution_type);
CREATE INDEX IF NOT EXISTS idx_incidents_resolved_by ON incidents(resolved_by);
CREATE INDEX IF NOT EXISTS idx_incidents_open_heartbeats ON incidents(organization_id, heartbeat_monitor_id) WHERE status = 'open';
```

### Rollback Migration: `migrations/000017_incident_lifecycle_resolution.down.sql`
```sql
ALTER TABLE incidents DROP CONSTRAINT IF EXISTS fk_incidents_heartbeat_org;
ALTER TABLE incidents ADD CONSTRAINT fk_incidents_heartbeat_org
    FOREIGN KEY (organization_id, heartbeat_monitor_id)
    REFERENCES heartbeat_monitors(organization_id, id)
    ON DELETE CASCADE;

DROP INDEX IF EXISTS idx_incidents_open_heartbeats;
DROP INDEX IF EXISTS idx_incidents_resolved_by;
DROP INDEX IF EXISTS idx_incidents_resolution_type;

ALTER TABLE incidents DROP CONSTRAINT IF EXISTS chk_incident_resolution_type;

ALTER TABLE incidents
    DROP COLUMN IF EXISTS resolution_note,
    DROP COLUMN IF EXISTS resolution_type,
    DROP COLUMN IF EXISTS resolved_by;
```

---

## 13. Authorization and RLS Review

1. **Role Enforcement:**
   - Both `POST /api/v1/incidents/:id/acknowledge` and `POST /api/v1/incidents/:id/resolve` are wrapped with `sensitiveTenantScoped(..., tenant.RoleMember)`.
   - Owners, Admins, and Members are permitted to acknowledge and resolve.
   - Viewers (`RoleViewer`) receive HTTP 403 Forbidden.
2. **Tenant Isolation:**
   - Every database query in `incident_repo.go` filters strictly by `organization_id = $1`.
   - Cross-tenant requests for incident ID across another organization return HTTP 404 (preventing ID enumeration).
3. **Database RLS Consistency:**
   - The Postgres RLS helper `WithTenantTx(ctx, pool, orgID, ...)` executes `SET LOCAL app.current_tenant_id = orgID`.
   - Security definer functions such as `assets_definer_policy` established in TASK-INCIDENT-001/002 remain completely unaffected.

---

## 14. Regression Tests and Actual Results

Ten exhaustive test scenarios were authored in `services/api/internal/repository/task_incident_004_lifecycle_test.go`:

| Test Suite | Scenario | Verifications | Result |
|---|---|---|---|
| **Test A** | Acknowledge Semantics | Status remains `open`, sets `acknowledged_at` & `acknowledged_by`, idempotent repeated ack, rejects ack on resolved incident | **PASS** |
| **Test B** | Manual Resolution | Sets status `resolved`, `resolution_type = manual`, `resolved_by`, optional note, rejects repeated resolve, records audit log | **PASS** |
| **Test C** | Automatic Website Recovery | Consecutive failures reset, resolves active incident with `automatic_recovery`, records `recovery_result_id`, records audit log | **PASS** |
| **Test D** | Automatic Heartbeat Recovery | Valid ping resolves open heartbeat incident with `automatic_recovery`, preserves history, records audit log | **PASS** |
| **Test E** | Heartbeat Monitor Deletion | Soft delete archives monitor, atomically resolves open incident as `source_monitor_deleted`, records deleting user | **PASS** |
| **Test F** | Multiple Incidents Handling | Monitor with multiple open incidents has all open incidents resolved upon deletion | **PASS** |
| **Test G** | Repeated & Concurrent Deletion | Repeated archiving is safe and idempotent, returns 0 resolved, does not corrupt data | **PASS** |
| **Test H** | RBAC & Cross-Tenant Isolation | Cross-tenant access to resolve returns not found, viewer role restriction enforced | **PASS** |
| **Test I** | Dashboard Aggregates | Verifies separate counters for `unacknowledged`, `acknowledged`, `website_outage`, and `heartbeat_missed` | **PASS** |
| **Test J** | Historical Compatibility | Historical incidents with `NULL` resolution fields load cleanly with graceful fallbacks | **PASS** |

In addition, `services/api/internal/server/task_incident_004_server_test.go` verified HTTP endpoints, serialization, error codes, and query filtering.

---

## 15. Full Backend Test Results

Full execution across all backend packages (`go test ./... -p 1 -count=1`):
```text
ok      clientpulse/services/api/internal/appurl        0.002s
ok      clientpulse/services/api/internal/auth          0.479s
ok      clientpulse/services/api/internal/checker       0.213s
ok      clientpulse/services/api/internal/config        0.002s
ok      clientpulse/services/api/internal/database      0.003s
ok      clientpulse/services/api/internal/health        1.613s
ok      clientpulse/services/api/internal/logger        0.002s
ok      clientpulse/services/api/internal/migrate       2.103s
ok      clientpulse/services/api/internal/notification  0.003s
ok      clientpulse/services/api/internal/notification/email    0.012s
ok      clientpulse/services/api/internal/notification/telegram 0.016s
ok      clientpulse/services/api/internal/pdf           2.234s
ok      clientpulse/services/api/internal/rate          0.002s
ok      clientpulse/services/api/internal/repository    5.788s
ok      clientpulse/services/api/internal/server        3.942s
ok      clientpulse/services/api/internal/service       0.473s
ok      clientpulse/services/api/internal/ssrf          0.003s
ok      clientpulse/services/api/internal/storage       0.004s
ok      clientpulse/services/api/internal/worker        2.530s
```
**Backend Test Result:** 100% Passed (0 Failures).

---

## 16. Frontend Test Results

Validation suite executed in `apps/web`:
1. **TypeScript Typecheck:** `npx tsc --noEmit` — Exit code `0` (No errors).
2. **ESLint:** `npm run lint` — Exit code `0` (Clean).
3. **Production Build:** `npm run build` — Exit code `0` (All 25 static & dynamic pages successfully generated).

---

## 17. Screenshots Inventory

All screenshots have been generated and archived in `task-report/screenshots/task-incident-004/`:

| File Name | Dimensions | Description |
|---|---|---|
| `01-incident-list-lifecycle-en.png` | 1440x900 | Incidents list in English showing open, acknowledged, and resolved incidents with distinct badges (`open`, `acknowledged`, `manual`, `source_monitor_deleted`). |
| `02-incident-detail-manual-resolution-en.png` | 1440x900 | Incident detail view showing explicit manual resolution by operator, resolution note, and event timeline. |
| `03-incident-detail-source-deleted-en.png` | 1440x900 | Incident detail view showing the informational notice: *"Closed because the associated heartbeat monitor was deleted"* with preserved failure diagnostics and safe non-broken links. |
| `04-heartbeat-delete-modal-en.png` | 1440x900 | Heartbeat monitor delete confirmation dialog explaining that active open incidents will be atomically resolved as `source_monitor_deleted` while preserving historical records. |
| `05-incident-list-lifecycle-fa-rtl.png` | 1440x900 | Incidents list in Persian (RTL) showing exact required terminology: `تأیید و پذیرش پیگیری`, `رفع و بستن رخداد`, and `بسته شده به دلیل حذف مانیتور`. |

---

## 18. Known Limitations & Edge Cases Handled

1. **Legacy Incidents Without Resolution Metadata:**
   - Pre-migration incidents have `NULL` in `resolved_by`, `resolution_type`, and `resolution_note`.
   - The UI and API gracefully handle `NULL` without crashing, falling back to *"Automatic"* / *"Auto Recovery"*.
2. **Post-Manual-Resolution Outages:**
   - If an operator manually resolves an incident while an endpoint is still failing, the monitor check worker detects continued failures and opens a new incident on the next check. Downtime cannot be permanently hidden.
3. **Deleted Source Navigation:**
   - If an incident's heartbeat monitor is archived, the UI detects `heartbeat_archived_at` and avoids rendering a broken navigation link to `/heartbeats/{id}`. Instead, it displays the past monitor name accompanied by a `Monitor Deleted` badge.

---

## 19. Dashboard Integration Contract

In accordance with TASK-008.B.3 rules, `apps/web/src/app/page.tsx` was **not** modified during this task. Instead, the backend API response for `GET /api/v1/dashboard/stats` was expanded with backward-compatible fields:
```json
{
  "total_clients": 5,
  "total_monitors": 12,
  "active_monitors": 10,
  "paused_monitors": 2,
  "up_monitors": 9,
  "down_monitors": 1,
  "open_incidents": 2,
  "unacknowledged_incidents": 1,
  "acknowledged_incidents": 1,
  "resolved_incidents": 18,
  "website_outage_incidents": 1,
  "heartbeat_missed_incidents": 1,
  "active_heartbeats": 4
}
```
The Dashboard team can immediately consume `unacknowledged_incidents`, `acknowledged_incidents`, `website_outage_incidents`, and `heartbeat_missed_incidents` without any breaking changes.

---

## 20. Production Deployment Confirmation

> [!IMPORTANT]
> **Production Deployment Status: STRICTLY PROHIBITED**
> - Production is operating on Release `v1.0.2` with Migration `000016`.
> - All work in this task was developed and validated strictly in the local containerized development environment.
> - No deployment to production or staging servers was executed.
> - The repository is left in a clean, tested, and reviewable state.

---

## Verification Checklist (Section 19)

- [x] **Acknowledge does not resolve an incident** (Verified: Status remains `open`).
- [x] **Acknowledgement actor and timestamp are preserved** (Verified: Recorded in `acknowledged_by` and `acknowledged_at`).
- [x] **Acknowledged incidents remain included in Open Incident counts** (Verified in SQL filter & stats test).
- [x] **Unacknowledged counts exclude acknowledged incidents** (Verified: Counted via `acknowledged_at IS NULL`).
- [x] **Manual resolution is a distinct operation** (Verified: `POST /api/v1/incidents/:id/resolve`).
- [x] **Manual resolution has actor and timestamp** (Verified: `resolved_by` and `resolved_at`).
- [x] **Manual resolution is not misrepresented as automatic recovery** (Verified: `resolution_type = 'manual'`).
- [x] **Successful website recovery resolves the appropriate incident** (Verified in Test C).
- [x] **Valid heartbeat recovery resolves its corresponding incident** (Verified in Test D).
- [x] **Invalid heartbeat requests never resolve incidents** (Verified: 404/401 on bad token).
- [x] **Deleting a Heartbeat Monitor resolves associated open incidents** (Verified in Test E & F).
- [x] **Deleted-source resolution is distinct from automatic recovery** (Verified: `resolution_type = 'source_monitor_deleted'`).
- [x] **Historical incidents remain available after source deletion** (Verified: Soft-delete + query retains row).
- [x] **Deletion and incident resolution are transactionally consistent** (Verified: Executed in single `pgx.Tx`).
- [x] **Repeated and concurrent deletion is safe** (Verified in Test G).
- [x] **Multi-tenant authorization remains intact** (Verified in Test H).
- [x] **Website outage and heartbeat incident counts are separated** (Verified in Test I).
- [x] **Legacy incidents remain readable** (Verified in Test J).
- [x] **Existing Scheduler migrations remain intact** (Verified: Migrations 000014–000016 untouched).
- [x] **All new regression tests pass** (Verified: 10/10 suites passed).
- [x] **Full backend and frontend validations pass** (Verified: Go tests + Next.js build clean).
- [x] **Production remains untouched** (Verified: Local development only).
- [x] **Full report is saved** (Saved to `task-report/TASK-INCIDENT-004-IMPLEMENTATION.md`).

