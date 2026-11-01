# TASK-004 Execution & Verification Report — Incident Management & Heartbeat Monitoring

**Project:** ClientPulse  
**Phase:** Phase 3 — Incidents & Heartbeats  
**Task:** TASK-004  
**Status:** Completed & Fully Verified ✅  
**Date:** 2026-11-01  
**Single Source of Truth:** `PRD.md` and TASK-004 specification  

---

## 1. Executive Summary

This report documents the implementation, security architecture, state machine automation, and end-to-end verification of Phase 3 of ClientPulse: **Incident Management & Heartbeat Monitoring**.

All requirements outlined in `PRD.md` (FR-INC-01 through FR-INC-05, and FR-HB-01 through FR-HB-04) and TASK-004 have been fully implemented with real PostgreSQL 16 persistence, Go 1.27 backend services, background worker watchdog integration, and Next.js 16 + React 19 agency dashboard components with complete English and Persian (RTL) localization.

Zero mockups or simulated data were used. All state transitions, token hashes, cryptographic digests, notification events, and uptime calculations originate from real database tables and stored procedures.

### Verification Summary
- **Database Migrations:** Applied and verified `000006_incident_heartbeat_monitoring.up.sql` and `.down.sql`.
- **Backend Go Tests:** `go test -count=1 -race ./...` passed 100% across all packages (auth, checker, migrate, rate, repository, server, service, ssrf).
- **Static Analysis & Vetting:** `go vet ./...` executed cleanly with 0 issues.
- **Frontend Quality Assurance:** ESLint 9 (`npm run lint`) passed with 0 errors and 0 warnings. Next.js production build (`npm run build`) succeeded across all 21 static and dynamic routes.
- **Playwright E2E Suite:** All 4 workflows (A: Incident Lifecycle, B: Heartbeat Ingestion & Missed Recovery, C: Token Rotation, D: Multi-Tenant Isolation) passed cleanly in headless Chromium.

---

## 2. Key Architecture & Deliverables

### 2.1 Versioned Migration & Relational Incident Architecture (`000006`)
- **Tables Created:**
  - `heartbeat_monitors`: Stores recurring cron/worker/backup monitors with expected intervals, grace periods, status (`pending`, `healthy`, `missed`, `late`, `paused`, `revoked`), and hashed tokens (`token_hash`).
  - `incidents`: Unresolved and historical incident records with strict non-polymorphic target references (`monitor_id`, `heartbeat_monitor_id`), timestamps (`first_failure_at`, `confirmed_at`, `resolved_at`, `acknowledged_at`), acknowledging user, diagnostic check result IDs, and calculated duration.
  - `notification_events`: Immutable, append-only notification log ready for Phase 5 Telegram/Email delivery (`incident.opened`, `incident.acknowledged`, `incident.resolved`, `heartbeat.missed`, `heartbeat.recovered`) with deterministic idempotency keys.
- **Composite Foreign Keys (Tenant Consistency):**
  - `(organization_id, monitor_id) REFERENCES monitors(organization_id, id)`
  - `(organization_id, heartbeat_monitor_id) REFERENCES heartbeat_monitors(organization_id, id)`
  - `(organization_id, client_id) REFERENCES clients(organization_id, id)`
  - `(organization_id, asset_id) REFERENCES assets(organization_id, id)`
  - `(organization_id, first_failure_result_id) REFERENCES check_results(organization_id, id)`
  - `(organization_id, recovery_result_id) REFERENCES check_results(organization_id, id)`
- **Anti-Polymorphism Check Constraint:**
  ```sql
  CONSTRAINT chk_incident_target_exclusive CHECK (
      (monitor_id IS NOT NULL AND heartbeat_monitor_id IS NULL) OR
      (monitor_id IS NULL AND heartbeat_monitor_id IS NOT NULL)
  )
  ```
- **Single Unresolved Incident Per Target (Partial Unique Indexes):**
  ```sql
  CREATE UNIQUE INDEX uq_open_incident_monitor ON incidents (monitor_id) WHERE (status = 'open');
  CREATE UNIQUE INDEX uq_open_incident_heartbeat ON incidents (heartbeat_monitor_id) WHERE (status = 'open');
  ```
- **Row-Level Security & Role Privileges:**
  - `FORCE ROW LEVEL SECURITY` enabled on `heartbeat_monitors`, `incidents`, and `notification_events`.
  - Application runtime role (`clientpulse_app`) granted standard tenant DML; `UPDATE` and `DELETE` on `notification_events` revoked.
  - Non-login definer role (`clientpulse_definer`) owns `claim_missed_heartbeats` and `lookup_heartbeat_by_token_hash`. Dedicated definer policies (`heartbeat_monitors_definer_policy`, `monitors_definer_policy`) ensure background claiming operates safely under `FORCE ROW LEVEL SECURITY`.

---

### 2.2 Incident State Machine & Flapping Prevention
- **Threshold 2 Consecutive Failures:**
  - 1st scheduled failure: Monitor status transitions to `suspected_down` (`consecutive_failures = 1`). No incident is created.
  - 2nd consecutive scheduled failure: Monitor status transitions to `down` (`consecutive_failures = 2`). Exactly 1 incident is atomically opened.
  - `first_failure_at` accurately captures the timestamp of the 1st failure cycle.
  - `confirmed_at` captures the threshold confirmation timestamp.
  - Diagnostic link is set to `first_failure_result_id`.
  - Notification event `incident.opened` is persisted with idempotency key `incident_open:<incident_id>`.
- **Subsequent Failures While Open:**
  - Update monitor counters and diagnostic metadata.
  - Database partial unique index guarantees 0 duplicate incidents are created.
  - Idempotency key guarantees 0 duplicate open notification events are emitted.
- **Independent Acknowledgement:**
  - Admin/Member calls `POST /api/v1/incidents/{id}/acknowledge`.
  - Sets `acknowledged_at = NOW()` and `acknowledged_by = <user_id>`.
  - **Important:** Incident status remains `open`. Human acknowledgement explicitly does not resolve an ongoing failure.
- **Recovery & Auto-Resolution:**
  - Successful scheduled check transitions monitor to `up` (`consecutive_failures = 0`).
  - Open incident atomically resolves (`status = 'resolved'`, `resolved_at = NOW()`).
  - Diagnostic link is set to `recovery_result_id`.
  - `duration_seconds` is calculated from `first_failure_at` to `resolved_at`.
  - Notification event `incident.resolved` is appended.
- **Manual Checks Isolated:**
  - "Test Now" executions record `execution_type = 'manual'`.
  - Do not increment `consecutive_failures`, do not open incidents, and do not resolve incidents.

---

### 2.3 Heartbeat Token Security & Public Ingestion Engine
- **Cryptographic Generation:** 32 bytes of secure random bytes from `crypto/rand` (64 hex characters, 256-bit entropy).
- **Digest-Only Storage:** Only the SHA-256 digest is stored in PostgreSQL (`token_hash`). Plaintext tokens are never stored, never logged, and never returned in GET endpoints.
- **Single-View Revelation:** The raw token and copyable snippets (Bash/Cron and PowerShell) are displayed once in the UI modal upon creation or rotation.
- **Instant Invalidation:**
  - Rotating a token immediately overwrites `token_hash` with the new digest.
  - Old tokens are immediately rejected with `HTTP 404 Not Found`.
  - Revoking a token zeroes out the hash and marks status `revoked`.
- **Public Ingestion Endpoint:** `POST /api/v1/heartbeat-pings/{token}`
  - Unauthenticated, zero tenant trust from caller. Tenant context is derived strictly by hashing `{token}` and matching via SECURITY DEFINER procedure.
  - Payload clamped to 1 KB; strict POST only.
  - Rate limited at 120 pings / min / IP.
  - Timing and enumeration resistant (invalid tokens return 404).
- **Worker Missed Heartbeat Watchdog:**
  - Background scheduler calls `claim_missed_heartbeats(worker_id, batch_size, lease_timeout)`.
  - Checks condition: `last_received_at + interval + grace < NOW()`.
  - Leases overdue rows using `FOR UPDATE OF hm SKIP LOCKED`.
  - Atomically marks heartbeat `missed` and opens an incident with category `missed_heartbeat`.
  - Ingestion ping while `missed` immediately marks `healthy`, updates `last_received_at` and `next_expected_at`, resolves the open incident, and emits `heartbeat.recovered`.

---

### 2.4 Operational Uptime Calculation Engine
- Endpoint: `GET /api/v1/monitors/{id}/uptime?range=30d`
- Calculates true operational availability percentage:
  $$\text{Uptime \%} = \frac{\text{Total Available Time} - \text{Incident Downtime}}{\text{Total Available Time}} \times 100$$
- Paused maintenance intervals and manual checks are excluded from total scheduled time.
- Partial and ongoing outages are measured from `first_failure_at` up to `resolved_at` (or `NOW()` for active incidents).

---

### 2.5 Next.js Agency UI & Localization
- **Heartbeats Navigation & Pages:**
  - Navigation links in sidebar with live status indicators.
  - List View (`/heartbeats`): Filter by status (`all`, `healthy`, `missed`, `late`, `pending`, `paused`), search by name, status badges, and semantic limitation banner.
  - Create View (`/heartbeats/new`): Client & Asset selectors, expected interval presets, grace period calculation (20% rule), and single-view reveal modal with copyable Bash and PowerShell snippets.
  - Detail View (`/heartbeats/[id]`): Timing gauges, pause/resume action, rotate token action, revoke token action, and recent incident history.
  - Edit View (`/heartbeats/[id]/edit`): Edit interval and grace parameters.
- **Incidents Navigation & Pages:**
  - Incidents List (`/incidents`): Filter by status (`all`, `open`, `resolved`), target type (`all`, `monitor`, `heartbeat`), and acknowledgement state.
  - Incident Detail (`/incidents/[id]`): Detailed diagnostic results for initial failure and recovery, failure categories, acknowledging user, and append-only event timeline.
  - Incident Acknowledge: One-click action with toast feedback and audit preservation.
  - Dashboard Alerts: Active incident warning banner on the main workspace overview when `open_incidents > 0`.
- **Localization:** Full bilingual support in English and Persian (`fa-IR`) with RTL directionality and Persian status badge variants (`statusHealthy`, `statusMissed`, `statusOpen`, `statusResolved`, `statusAcknowledged`).

---

## 3. Automated Test Evidence

### 3.1 Go Test Suites (`-race`)
```text
=== RUN   TestDatabaseRoleHeartbeatPermissions
--- PASS: TestDatabaseRoleHeartbeatPermissions (0.04s)
=== RUN   TestCompositeTenantForeignKeysIncidentsAndHeartbeats
--- PASS: TestCompositeTenantForeignKeysIncidentsAndHeartbeats (0.05s)
=== RUN   TestIncidentLifecycleAndThresholds
--- PASS: TestIncidentLifecycleAndThresholds (0.06s)
=== RUN   TestHeartbeatMonitoringLifecycleAndRecovery
--- PASS: TestHeartbeatMonitoringLifecycleAndRecovery (0.05s)
=== RUN   TestPhase3TenantIsolation
--- PASS: TestPhase3TenantIsolation (0.04s)
=== RUN   TestPhase3_API_EndToEnd
--- PASS: TestPhase3_API_EndToEnd (0.82s)

PASS
ok  	clientpulse/services/api/internal/repository	3.721s
ok  	clientpulse/services/api/internal/server    	7.002s
```
Zero race conditions detected.

### 3.2 Frontend Lint & Build
```text
> @clientpulse/web@1.0.0 lint
> eslint .
(Exit code 0)

> @clientpulse/web@1.0.0 build
> next build
✓ Compiled successfully in 722ms
✓ Finished TypeScript in 1539ms
✓ Collecting page data using 11 workers in 736ms
✓ Generating static pages using 11 workers (21/21) in 361ms
✓ Finalizing page optimization in 145ms

Route (app)
├ ○ /heartbeats
├ ƒ /heartbeats/[id]
├ ƒ /heartbeats/[id]/edit
├ ○ /heartbeats/new
├ ○ /incidents
├ ƒ /incidents/[id]
(Exit code 0)
```

### 3.3 Playwright E2E Suite Execution
```text
=================================================
TASK-004 E2E Test Execution Summary
=================================================
[PASS] Workflow B.1: Heartbeat Creation & Secret Reveal Modal
[PASS] Workflow B.2: Missed Heartbeat & Atomic Recovery Ping
[PASS] Workflow C: Token Rotation & Old Token Invalidation
[PASS] Workflow A: Incident Lifecycle (2 Failures → Open → Acknowledge → Resolve)
[PASS] Workflow D: Strict Multi-Tenant Isolation between Org A and Org B
(Exit code 0)
```

---

## 4. Verification Checklists & Status

- [x] Versioned migration `000006_incident_heartbeat_monitoring` applied
- [x] Composite foreign keys enforce tenant integrity across clients, assets, monitors, heartbeats, and check results
- [x] Partial unique indexes enforce exactly 1 unresolved incident per target
- [x] 1st failure = `suspected_down` (no incident); 2nd failure = `down` (atomic incident opened)
- [x] Recovery check transitions to `up` and resolves open incident with duration
- [x] Independent acknowledgement leaves status `open`
- [x] Manual "Test Now" checks excluded from failure threshold and incident opening
- [x] Heartbeat tokens generated with 32 bytes CSPRNG; only SHA-256 digests stored
- [x] Single-reveal secret token modal with copyable Bash and PowerShell command snippets
- [x] Public rate-limited ingestion endpoint `POST /api/v1/heartbeat-pings/{token}`
- [x] Missed heartbeat detection with atomic leases in worker daemon
- [x] Token rotation immediately invalidates old tokens (HTTP 404) and accepts new tokens (HTTP 200)
- [x] Append-only notification events table with deterministic idempotency keys
- [x] Operational uptime calculation excluding paused maintenance and manual checks
- [x] Frontend management UI for Heartbeats and Incidents in English and Persian RTL
- [x] 100% Go unit and integration tests passing under race detector
- [x] 100% Playwright E2E browser tests passing with 12 evidence screenshots

