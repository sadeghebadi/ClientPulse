# TASK-003.1: Monitoring Core Security, Data Integrity & Regression Audit

**Phase:** Phase 2 Targeted Verification & Hardening  
**Status:** Completed  
**Completed Date:** 2026-11-01  
**Single Source of Truth:** `PRD.md` and TASK-003.1 specification

---

## 1. Audit Scope & Objectives

Investigate, harden, and verify the Monitoring Core against critical security, data integrity, and operational requirements:
1. **Worker Database Function & Role Separation:** Revoke `EXECUTE` on `claim_due_monitors` from `PUBLIC` and `clientpulse_app`. Create dedicated Worker login role (`clientpulse_worker`) and restricted non-login function owner (`clientpulse_definer`). Secure fixed `search_path`, schema-qualify objects, validate `p_batch_size` (1-100) and `p_lease_seconds` (5-3600). Scope cross-tenant RLS policies for worker. Update configs, compose, and environment files.
2. **Composite Tenant Foreign Keys:** Enforce cross-tenant isolation at the database constraint level across `assets`, `monitors`, and `check_results`.
3. **Monitor State Transitions:** Implement state machine: `pending`, `up`, `suspected_down` (1 consecutive failure), `down` (2 consecutive failures), `paused`, `unknown`. Add `consecutive_failures` column. Ensure manual checks do not alter scheduled failure thresholds.
4. **Retention & Hourly Aggregations:** Implement `check_aggregates_hourly` table (retained >= 13 months / 400 days). Add configurable `CHECK_RESULT_RETENTION_DAYS` (default 90). Implement bounded batch pruning in worker (`LIMIT 500`).
5. **Frontend Regression & Flat Config:** Verify ESLint 9 Flat Config (`eslint .`) and Phase 1 auth routes (`/login`, `/signup`, `/forgot-password`, `/reset-password`, `/verify-email`, `/settings/account`, `/settings/members`, `/organizations`).
6. **Monitoring E2E Runtime Evidence:** Capture full real-cycle execution evidence proving scheduling, execution, persistence, state transitions, and Org 2 RLS isolation.

---

## 2. Work Breakdown & Status Checklist

- [x] **1. Security Critical — Worker Database Function & Role Separation:**
  - [x] Created `clientpulse_definer` (NOLOGIN, non-superuser, NOBYPASSRLS) as function owner.
  - [x] Created `clientpulse_worker` (LOGIN, non-superuser, NOBYPASSRLS) as dedicated worker role.
  - [x] Hardened schema permissions: `REVOKE CREATE ON SCHEMA public FROM PUBLIC, clientpulse_app, clientpulse_worker, clientpulse_definer`.
  - [x] Revoked `EXECUTE` on `claim_due_monitors` from `PUBLIC` and `clientpulse_app`.
  - [x] Granted `EXECUTE` on `claim_due_monitors` strictly to `clientpulse_worker`.
  - [x] Fixed `search_path = pg_catalog, public, pg_temp` inside `claim_due_monitors`.
  - [x] Enforced parameter validation: `p_batch_size` (1-100), `p_lease_timeout_seconds` (5-3600), non-empty `p_worker_id`.
  - [x] Scoped cross-tenant worker RLS policies for background claim, deletion, and aggregation.
  - [x] Revoked access from `clientpulse_worker` to sensitive credential and audit tables (`users`, `sessions`, `password_reset_tokens`, `email_verification_tokens`, `audit_logs`).
  - [x] Updated configuration (`WORKER_DATABASE_URL`), `docker-compose.yml`, `.env`, `.env.example`.
  - [x] Automated test: `TestDatabaseRolePermissionsAndClaimIsolation`.

- [x] **2. Critical — Composite Tenant Foreign Keys:**
  - [x] Added composite UNIQUE constraints: `clients(organization_id, id)`, `assets(organization_id, id)`, `monitors(organization_id, id)`.
  - [x] Added composite foreign keys:
    - `assets(organization_id, client_id) REFERENCES clients(organization_id, id) ON DELETE CASCADE`
    - `monitors(organization_id, asset_id) REFERENCES assets(organization_id, id) ON DELETE CASCADE`
    - `check_results(organization_id, monitor_id) REFERENCES monitors(organization_id, id) ON DELETE CASCADE`
  - [x] Automated test: `TestCompositeTenantForeignKeys` (verifies direct SQL attempts to insert cross-tenant relationships fail with FK constraint violations).

- [x] **3. Monitor State Transitions & Thresholds:**
  - [x] Added `consecutive_failures INT NOT NULL DEFAULT 0` column to `monitors`.
  - [x] Updated `chk_monitors_status` constraint to support: `pending`, `up`, `suspected_down`, `down`, `paused`, `unknown`.
  - [x] Implemented state transition logic:
    - Initial: `pending`
    - 1st failure: `suspected_down` (`consecutive_failures = 1`)
    - 2nd failure: `down` (`consecutive_failures >= 2`)
    - Recovery: `up` (`consecutive_failures = 0`)
    - Paused: preserved unchanged.
  - [x] Ensured manual "Test Now" executions do not mutate `consecutive_failures` or scheduled status.
  - [x] Automated test: `TestMonitorStateTransitionsAndConsecutiveFailures`.

- [x] **4. Check Results Retention & Hourly Aggregation:**
  - [x] Created `check_aggregates_hourly` table with composite tenant foreign key and RLS.
  - [x] Upsert scheduled check metrics into hourly aggregates (`total_checks`, `successful_checks`, `total_latency_ms`, `min_latency_ms`, `max_latency_ms`).
  - [x] Configured `CHECK_RESULT_RETENTION_DAYS` (default 90).
  - [x] Implemented bounded batch pruning (`LIMIT 500`) for check results older than retention period.
  - [x] Implemented aggregate retention pruning (retained for 400 days / >13 months).
  - [x] Automated test: `TestRetentionPruningAndHourlyAggregation`.

- [x] **5. Frontend Modernization & Auth Regression Verification:**
  - [x] Verified ESLint 9 Flat Config: `npm run lint` executes `eslint .` with 0 warnings and 0 errors.
  - [x] Updated TypeScript interfaces and status badges in frontend (`suspected_down`, `unknown`) with RTL Persian translations.
  - [x] Verified Phase 1 Auth/Tenant E2E flow in `TestAuthAndTenantLifecycleE2ERegression`: signup, me, change-password, login, list orgs, create second org, switch org, list members, logout.
  - [x] Verified Next.js Turbopack production build: all 17 routes compiled successfully.

- [x] **6. Monitoring E2E Runtime Evidence:**
  - [x] Automated test `TestScheduledMonitoringEndToEndEvidence`:
    - Provisioned Org 1, Client, Asset, Monitor.
    - Executed scheduled check cycle through worker scheduler daemon.
    - Verified check result creation, status update to `up`, and `next_check_at` drift-free update.
    - Proved Org 2 cannot read Org 1 check results or monitors under RLS.
  - [x] Verified real worker container logs in Docker Compose environment.

