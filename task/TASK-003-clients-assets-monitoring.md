# TASK-003: Clients, Assets & Monitoring Core

**Project:** ClientPulse  
**Phase:** Phase 2  
**Status:** Completed ✅  
**Started:** 2026-11-01  
**Completed:** 2026-11-01  

---

## 1. Scope & Objectives

Implement the core operational foundation of ClientPulse:
1. **Client Management:** Multi-tenant Client CRUD, search/filtering, archive/restore behavior, ownership enforcement.
2. **Asset Management:** Website/API Asset CRUD, environment classification, duplicate URL warnings, revalidation on change.
3. **SSRF Guard & Network Safety:** Dedicated multi-layer SSRF protection with DNS resolution inspection, TCP IP pinning, custom dialer, SNI preservation, redirect revalidation, and response size limits.
4. **Monitoring Engine & Scheduler:** PostgreSQL-backed atomic job claiming (`FOR UPDATE SKIP LOCKED`), lease management, misfire policy, drift-free `next_check_at`.
5. **Checks Implementation:** Real HTTP (GET/HEAD, status ranges, body text matching, JSON Pointer), SSL/TLS (chain validation, expiry thresholds), and DNS (A/AAAA).
6. **Check Results & History:** Result persistence, manual vs. scheduled check tracking, retention policy.
7. **Monitoring Controls:** Activate, Pause, Resume, Edit, Archive, and Rate-Limited "Test Now".
8. **Frontend Interface:** Client list/detail/form, Asset list/detail/form, Monitor list/detail/form, check history, and real data Agency Dashboard in English and Persian (RTL).
9. **Security & Isolation:** PostgreSQL RLS with `FORCE ROW LEVEL SECURITY` on all new tables, `clientpulse_app` runtime role privileges, email verification restrictions on sensitive operations.
10. **Comprehensive Verification:** Automated test suite (`go test -race ./...`, `npm run lint`, `npm run build`), end-to-end scheduled check verification, documentation.

---

## 2. Work Breakdown & Status Checklist

- [x] **Step 1: Database Schema & Migrations**
  - [x] Migration `000004_clients_assets_monitoring.up.sql`
  - [x] Migration `000004_clients_assets_monitoring.down.sql`
  - [x] RLS policies and runtime role grants (`clientpulse_app`)
  - [x] Test migration execution and rollback
- [x] **Step 2: SSRF Guard Implementation & Tests**
  - [x] URL validation (scheme, userinfo, ports)
  - [x] IP classification (private, loopback, link-local, multicast, metadata, CGNAT, IPv6)
  - [x] DNS resolution inspection and connection IP pinning
  - [x] Safe HTTP Transport, dialer, and redirect validator
  - [x] SSRF unit tests covering all Part 5 test cases
- [x] **Step 3: Repositories & Data Access**
  - [x] Client Repository with tenant RLS & search/filter
  - [x] Asset Repository with client validation & URL duplicate check
  - [x] Monitor Repository with atomic scheduler claims and leases
  - [x] CheckResult Repository with history queries and retention pruning
  - [x] Multi-tenant RLS isolation tests
- [x] **Step 4: Monitoring Check Engine**
  - [x] DNS checker (A / AAAA, duration, failure categorization)
  - [x] SSL/TLS checker (chain validation, NotAfter/NotBefore, warning thresholds)
  - [x] HTTP checker (GET/HEAD, expected status, text matching, JSON pointer, 256KB limit)
  - [x] Check executor unit tests
- [x] **Step 5: Worker Scheduler Service**
  - [x] Integrate monitoring execution into Go worker daemon
  - [x] Atomic job claiming loop (`SELECT FOR UPDATE SKIP LOCKED`)
  - [x] Misfire and drift-prevention policy
  - [x] Stale lease recovery
  - [x] Graceful worker shutdown
- [x] **Step 6: API Services & HTTP Handlers**
  - [x] RequireVerifiedEmail middleware for sensitive mutating operations
  - [x] Client CRUD handlers (`/api/v1/clients`)
  - [x] Asset CRUD handlers (`/api/v1/assets`)
  - [x] Monitor CRUD handlers & controls (`/api/v1/monitors`, `/pause`, `/resume`, `/test`, `/history`)
  - [x] Dashboard statistics handler (`/api/v1/dashboard/stats`)
  - [x] API integration tests
- [x] **Step 7: Frontend Pages & Components**
  - [x] Dashboard page with real metrics and recent checks
  - [x] Client management (List, Create, Edit, Detail)
  - [x] Asset management (Create, Edit, Detail)
  - [x] Monitor management (Create, Edit, Detail, Test Now, History)
  - [x] English and Persian translations with RTL support
  - [x] Frontend build and lint verification
- [x] **Step 8: End-to-End Verification & Reporting**
  - [x] Real scheduled check execution against controlled test target
  - [x] Verify check result persistence and dashboard update
  - [x] Documentation update (`docs/api.md`, `docs/architecture.md`, `docs/security.md`, `README.md`)
  - [x] Final report `task-report/TASK-003-report.md`
