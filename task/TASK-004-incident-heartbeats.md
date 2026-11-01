# TASK-004: Incident Management & Heartbeat Monitoring

**Project:** ClientPulse  
**Phase:** Phase 3  
**Status:** Completed ✅  
**Started:** 2026-11-01  
**Completed:** 2026-11-01  

---

## 1. Scope & Objectives

Implement the complete Incident Management and Heartbeat Monitoring functionality of ClientPulse:
1. **Incident Data Architecture:** Relational incident model with composite tenant foreign keys, strict tenant isolation, partial unique indexes for single unresolved incident per target, and auditability.
2. **Incident State Machine:** Automatic transitions (`open`, `resolved`), failure thresholds (2 consecutive failures = down -> open incident), recovery thresholds (1 success = up -> resolve incident), and independent user acknowledgement (`acknowledged_at`, `acknowledged_by`).
3. **Flapping & Duplicate Prevention:** Exactly one unresolved incident per target, event deduplication, throttling, and documented cooldown policy.
4. **Heartbeat Monitoring Data Model:** Heartbeat monitors bound to Clients (required) and Assets (optional), composite foreign key enforcement, configurable expected intervals and grace periods (PRD default 20% with min 60s).
5. **Heartbeat Token Security:** Cryptographically secure 32-byte tokens, SHA-256 digests stored in PostgreSQL (never plaintext), zero token logging, single-view secret revelation upon creation/rotation, instant revocation.
6. **Public Heartbeat Ingestion Endpoint:** `POST /api/v1/heartbeat-pings/{token}` with strict abuse protection, rate limiting, constant-time token lookup, zero caller-supplied tenant trust, and minimal safe response.
7. **Worker Missed Heartbeat Detection:** Background worker loop with atomic leases (`FOR UPDATE SKIP LOCKED`), discovering overdue heartbeats (`last_received_at + interval + grace < NOW()`), opening missed heartbeat incidents, and automatic recovery upon ping.
8. **Persistent Notification Event Generation:** Persistent, append-only notification events (`incident.opened`, `incident.acknowledged`, `incident.resolved`, `heartbeat.missed`, `heartbeat.recovered`) with stable idempotency keys, atomic persistence, and zero secret leakage (ready for Phase 5 delivery).
9. **Operational Uptime Calculation:** Dynamic calculation based on scheduled monitoring coverage and incident durations, excluding paused intervals and manual Test Now runs.
10. **Frontend Dashboard & UI:** Comprehensive management interfaces for Incidents (list, detail, filters, acknowledge) and Heartbeats (list, create, detail, edit, pause/resume, rotate, revoke, copyable Bash/Cron and PowerShell snippets) in English and Persian (RTL).
11. **Comprehensive Verification:** Unit, multi-tenant isolation, database security definer, worker race detection, frontend lint/build, and Playwright E2E browser tests.

---

## 2. Work Breakdown & Status Checklist

- [x] **Step 1: Database Schema & Versioned Migrations**
  - [x] Migration `000006_incident_heartbeat_monitoring.up.sql`
  - [x] Migration `000006_incident_heartbeat_monitoring.down.sql`
  - [x] Tables: `heartbeat_monitors`, `incidents`, `notification_events`
  - [x] Composite foreign keys:
    - `clients(organization_id, id)`
    - `assets(organization_id, client_id, id)`
    - `monitors(organization_id, id)`
    - `heartbeat_monitors(organization_id, id)`
    - `check_results(organization_id, id)`
    - `incidents(organization_id, id)`
  - [x] Partial unique indexes for single unresolved incident per target
  - [x] Unique constraints on heartbeat token hashes
  - [x] RLS policies (`FORCE ROW LEVEL SECURITY`) and grants for `clientpulse_app` & `clientpulse_worker`
  - [x] SECURITY DEFINER function `claim_missed_heartbeats(...)` owned by `clientpulse_definer`
- [x] **Step 2: Backend Models & Repositories**
  - [x] `HeartbeatMonitorRepository`: CRUD, token hash generation, rotation, revocation, lease claiming, ping recording
  - [x] `IncidentRepository`: Create/find open, acknowledge, resolve, duration calculation, filterable queries
  - [x] `NotificationEventRepository`: Idempotent atomic event creation with structured payloads
  - [x] Integrate incident transitions into `MonitorRepository.RecordScheduledCheckOutcome` (threshold 2 -> open, 1 success -> resolve)
  - [x] Uptime calculation engine based on scheduled checks, incidents, and paused intervals
- [x] **Step 3: Public Heartbeat Ping Handler & Rate Limiter**
  - [x] `POST /api/v1/heartbeat-pings/{token}`
  - [x] Token SHA-256 hash lookup in database
  - [x] Dedicated in-memory / IP rate limiter (distinct from user auth)
  - [x] Strict payload limit (max 1KB, empty or ignored)
  - [x] Atomic heartbeat recording & automatic incident recovery
- [x] **Step 4: Worker Heartbeat Watchdog**
  - [x] Missed heartbeat detector loop in worker scheduler
  - [x] Atomic claiming using `claim_missed_heartbeats`
  - [x] Transition status to `missed` and create incident atomically
  - [x] Lease timeout and recovery
- [x] **Step 5: API Endpoints & Tenant Middlewares**
  - [x] Heartbeat API routes:
    - `GET /api/v1/heartbeats`
    - `POST /api/v1/heartbeats`
    - `GET /api/v1/heartbeats/{id}`
    - `PUT /api/v1/heartbeats/{id}`
    - `DELETE /api/v1/heartbeats/{id}` (archive)
    - `POST /api/v1/heartbeats/{id}/pause`
    - `POST /api/v1/heartbeats/{id}/resume`
    - `POST /api/v1/heartbeats/{id}/rotate-token`
    - `POST /api/v1/heartbeats/{id}/revoke-token`
  - [x] Incident API routes:
    - `GET /api/v1/incidents`
    - `GET /api/v1/incidents/{id}`
    - `POST /api/v1/incidents/{id}/acknowledge`
  - [x] Uptime API metrics / client & monitor incident histories
- [x] **Step 6: Frontend Incident & Heartbeat Interfaces**
  - [x] Navigation updates (Incidents, Heartbeats links in AppLayout with badges)
  - [x] Heartbeats List (`/heartbeats`) with status, interval, last received, search/filter
  - [x] Create Heartbeat (`/heartbeats/new`) with client/asset selector and secret token modal
  - [x] Heartbeat Details (`/heartbeats/[id]`) with copyable Bash/Cron & PowerShell snippets, history
  - [x] Edit Heartbeat (`/heartbeats/[id]/edit`)
  - [x] Incidents List (`/incidents`) with status, target, duration, failure category filters
  - [x] Incident Details (`/incidents/[id]`) with timeline, check results, acknowledge action
  - [x] English and Persian (fa-IR) translations & RTL support
- [x] **Step 7: Automated Tests & Verification**
  - [x] Go unit tests & race detector: `go test -count=1 -race ./...`
  - [x] Go vet: `go vet ./...`
  - [x] Database isolation tests with 2 organizations (Agency A vs. Agency B)
  - [x] Token security and hashing verification
  - [x] State machine threshold tests (1 failure != incident, 2 failures = incident, recovery = resolve)
  - [x] Playwright E2E browser tests for Workflows A, B, C, D
  - [x] Frontend lint & build: `npm run lint`, `npm run build`
- [x] **Step 8: Documentation & Reports**
  - [x] `task-report/TASK-004-report.md`
  - [x] `task-report/incident-e2e-results.md`
  - [x] Update `docs/api.md`, `docs/architecture.md`, `docs/security.md`, `README.md`
