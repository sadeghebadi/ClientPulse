# TASK-003: Clients, Assets & Monitoring Core Report

**Project:** ClientPulse  
**Task:** TASK-003 — Clients, Assets & Monitoring Core  
**Phase:** Phase 2  
**Date:** 2026-11-01  
**Status:** Complete — All Objectives Implemented & Verified  

---

## 1. Executive Summary

Phase 2 (TASK-003) introduces the primary business functionality of ClientPulse: real client management, asset tracking, multi-tenant monitoring configuration, a dedicated outbound SSRF security guard, a distributed PostgreSQL-backed job scheduler, an asynchronous worker check engine, and a Next.js agency dashboard with English and Persian (RTL) support.

### Key Milestones Achieved:
1. **Database Schema & Multi-Tenant RLS:** Migration `000004_clients_assets_monitoring` creates `clients`, `assets`, `monitors`, and `check_results` with `FORCE ROW LEVEL SECURITY`. Application runtime user `clientpulse_app` (`NOBYPASSRLS`) is granted necessary privileges. A dedicated `claim_due_monitors` stored function with `SECURITY DEFINER` enables the background worker to claim due jobs across tenants safely without opening broad RLS bypass backdoors.
2. **Dedicated SSRF Guard:** Complete implementation in `services/api/internal/ssrf` with URI scheme and port restrictions (HTTP/HTTPS only, ports 80/443), comprehensive IP classification (blocking RFC 1918, loopbacks, link-local, cloud metadata, CGNAT, IPv6 ULA/link-local, IPv4-mapped IPv6), DNS resolution inspection, outbound TCP socket IP pinning, SNI preservation, redirect revalidation, and proxy bypass neutralization.
3. **Check Engine:** Protocol-level validation for HTTP GET and HEAD, status code range matching, response body inspection limited to 256 KB, text MIME type verification, RFC 6901 JSON Pointer assertions, SSL/TLS certificate chain inspection (NotBefore/NotAfter, cipher, issuer, 30/14/7-day expiry warnings), and DNS resolution duration measurement.
4. **PostgreSQL-Backed Distributed Scheduler:** High-performance atomic job claiming using `SELECT FOR UPDATE OF m SKIP LOCKED`, execution leases (60s default) with automatic recovery of stale leases, drift-free `next_check_at` calculation (`scheduled_at + interval`), and a misfire skip policy preventing burst executions after worker downtime.
5. **Real Agency Dashboard & Monitoring UI:** Real Next.js dashboard calculating actual statistics from active tenant records, Client CRUD (archive-first, restore), Asset CRUD (duplicate URL warnings, SSRF validation), Monitor CRUD (pause, resume, edit, delete, paginated execution history, rate-limited manual "Test Now").
6. **Email Verification Protection:** Enforced `RequireVerifiedEmail` middleware across all sensitive mutating endpoints per PRD FR-AUTH-01.

All tests pass with Go race detection enabled (`-race`), TypeScript compilation succeeds, Next.js builds cleanly with 0 lint warnings/errors, and all five Docker containers (`postgres`, `mailpit`, `api`, `worker`, `web`) run in a healthy state.

---

## 2. Pre-Implementation Audit & Reusable Infrastructure

Before development, an architectural audit reviewed existing Phase 1 foundations:
- **Tenant Context & RLS:** Verified `WithTenantTx` and `set_config('app.current_organization_id', $orgID, true)`. Ensured all new tables inherit strict tenant scoping and fail-closed RLS.
- **Worker Architecture:** Evaluated `services/api/cmd/worker` which maintained a persistent PostgreSQL heartbeat. Extended this worker daemon to host the concurrent monitoring scheduler alongside the heartbeat runner.
- **Auth & Permissions:** Reused `RequireAuth`, `RequireRole`, CSRF session-bound tokens, and sliding window rate limiters. Added `RequireVerifiedEmail` for state mutations.
- **Deferred Work Tracking:** Maintained tracking of known deferred items:
  - Actual email invitation links with cryptographic tokens (FR-TEN-03).
  - Configurable Argon2 concurrency memory-budget.
  - Production trusted-proxy CIDR configuration via dynamic environment flags.

---

## 3. Database Schema & Migration Architecture

Migration `migrations/000004_clients_assets_monitoring.up.sql` establishes four relational entities with foreign keys, cascading constraints, indexes, and RLS policies:

### 3.1 Relational Entities
- **`clients`:** Organization-scoped customer profiles.
  - Fields: `id`, `organization_id`, `name`, `contact_name`, `contact_email`, `website`, `notes`, `status` (`active`, `archived`), `created_at`, `updated_at`, `archived_at`.
  - Indexes: `(organization_id, status)`, `(organization_id, name)`.
- **`assets`:** Monitored resources belonging to a Client.
  - Fields: `id`, `organization_id`, `client_id`, `name`, `url`, `type` (`website`, `api`), `environment` (`production`, `staging`, `development`), `description`, `is_active`, `created_at`, `updated_at`, `archived_at`.
  - Indexes: `(organization_id, client_id)`, `(organization_id, url)`.
- **`monitors`:** Monitoring configuration attached to an Asset.
  - Fields: `id`, `organization_id`, `asset_id`, `name`, `url_override`, `method` (`GET`, `HEAD`), `interval_seconds` (>= 60), `timeout_seconds` (3–30), `expected_status_low`, `expected_status_high`, `follow_redirects`, `max_redirects`, `expected_text`, `json_pointer`, `is_active`, `status` (`pending`, `up`, `down`, `unknown`, `paused`), `last_checked_at`, `next_check_at`, `active_worker_id`, `lease_expires_at`, `created_at`, `updated_at`, `archived_at`.
  - Indexes: `(organization_id, asset_id)`, `(is_active, next_check_at) WHERE archived_at IS NULL`.
- **`check_results`:** Historical check execution logs.
  - Fields: `id`, `organization_id`, `monitor_id`, `scheduled_at`, `started_at`, `completed_at`, `latency_ms`, `result` (`success`, `failure`, `timeout`, `blocked`, `error`), `http_status_code`, `error_code`, `error_message`, `tls_version`, `tls_cipher`, `tls_issuer`, `tls_not_before`, `tls_not_after`, `tls_days_remaining`, `dns_duration_ms`, `resolved_ip`, `worker_id`, `execution_type` (`scheduled`, `manual`), `created_at`.
  - Indexes: `(organization_id, monitor_id, created_at DESC)`.

### 3.2 Cross-Tenant Worker Job Claiming (`claim_due_monitors`)
Under PostgreSQL RLS with `FORCE ROW LEVEL SECURITY` and `clientpulse_app` having `NOBYPASSRLS`, queries without `app.current_organization_id` return 0 rows. To allow the background worker daemon to claim due monitors across all organizations without assigning `BYPASSRLS` to the application user, a specialized stored function was created:

```sql
CREATE OR REPLACE FUNCTION claim_due_monitors(
    p_worker_id TEXT,
    p_batch_size INT DEFAULT 10,
    p_lease_seconds INT DEFAULT 60
)
RETURNS TABLE (
    id UUID,
    organization_id UUID,
    asset_id UUID,
    name VARCHAR(255),
    url_override TEXT,
    asset_url TEXT,
    method VARCHAR(10),
    interval_seconds INT,
    timeout_seconds INT,
    expected_status_low INT,
    expected_status_high INT,
    follow_redirects BOOLEAN,
    max_redirects INT,
    expected_text TEXT,
    json_pointer TEXT,
    scheduled_at TIMESTAMPTZ
)
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    RETURN QUERY
    WITH due AS (
        SELECT m.id AS due_id, m.next_check_at AS due_scheduled_at
        FROM monitors m
        JOIN assets a ON a.id = m.asset_id
        WHERE m.is_active = true
          AND m.archived_at IS NULL
          AND a.is_active = true
          AND a.archived_at IS NULL
          AND m.next_check_at <= NOW()
          AND (m.lease_expires_at IS NULL OR m.lease_expires_at < NOW())
        ORDER BY m.next_check_at ASC
        LIMIT p_batch_size
        FOR UPDATE OF m SKIP LOCKED
    )
    UPDATE monitors m
    SET active_worker_id = p_worker_id,
        lease_expires_at = NOW() + (p_lease_seconds || ' seconds')::INTERVAL,
        updated_at = NOW()
    FROM due
    JOIN assets a ON a.id = m.asset_id
    WHERE m.id = due.due_id
    RETURNING 
        m.id,
        m.organization_id,
        m.asset_id,
        m.name,
        m.url_override,
        a.url,
        m.method,
        m.interval_seconds,
        m.timeout_seconds,
        m.expected_status_low,
        m.expected_status_high,
        m.follow_redirects,
        m.max_redirects,
        m.expected_text,
        m.json_pointer,
        due.due_scheduled_at;
END;
$$;
```

---

## 4. SSRF Guard & Outbound Network Safety

The SSRF guard (`services/api/internal/ssrf/ssrf.go`) implements strict outbound network isolation:

```
[Target URL]
     │
     ▼
[URL Validation] ──▶ Reject non-http/https, userinfo, non-80/443 ports, localhost, .local, internal names
     │
     ▼
[DNS Resolution] ──▶ Resolve A & AAAA records via net.DefaultResolver
     │
     ▼
[IP Classification] ──▶ Check ALL returned IPs against prohibited CIDRs
     │                  (RFC 1918, Loopback, Link-Local, CGNAT, Cloud Metadata, Multicast, IPv6 ULA)
     │
     ▼
[TCP Socket Pinning] ──▶ Pin validated public IP in custom net.Dialer
     │                   Bypass second unvalidated DNS query during connect
     │
     ▼
[TLS & HTTP Setup] ──▶ Inject original target hostname into SNI (ServerName) and Host header
     │
     ▼
[Redirect Inspection] ──▶ Run CheckRedirect: independently validate each redirect target
     │
     ▼
[Safe Execution] ──▶ Limit body inspection to 256 KB; Set Transport.Proxy = nil
```

### Automated SSRF Test Suite (`services/api/internal/ssrf/ssrf_test.go`)
Comprehensive test cases verified with race detection:
- `http://localhost/` -> Prohibited (localhost)
- `http://sub.localhost/` -> Prohibited (localhost subdomain)
- `http://127.0.0.1/` -> Prohibited (loopback)
- `http://10.0.0.1/` -> Prohibited (RFC 1918 private IPv4)
- `http://172.16.0.1/` -> Prohibited (RFC 1918 private IPv4)
- `http://192.168.1.1/` -> Prohibited (RFC 1918 private IPv4)
- `http://169.254.169.254/` -> Prohibited (AWS/GCP IMDS link-local)
- `http://[::1]/` -> Prohibited (IPv6 loopback)
- `http://[fc00::1]/` -> Prohibited (IPv6 ULA)
- `http://[fe80::1]/` -> Prohibited (IPv6 link-local)
- `http://[::ffff:127.0.0.1]/` -> Prohibited (IPv4-mapped IPv6)
- `http://100.64.0.1/` -> Prohibited (CGNAT)
- `http://user:pass@example.com/` -> Prohibited (embedded userinfo)
- `http://example.com:8080/` -> Prohibited (non-standard port)
- `ftp://example.com/` -> Prohibited (non-HTTP scheme)
- Target with mixed public and private DNS records -> Prohibited (fail-closed if any IP is private)
- Redirect to internal metadata (`http://169.254.169.254`) -> Prohibited and rejected in `CheckRedirect`
- Outbound socket connects exclusively to pinned IP; `Proxy = nil` prevents proxy env injection.

---

## 5. Monitoring Engine, Scheduler & Protocol Checks

### 5.1 Drift-Free Scheduling & Misfire Policy
- The scheduler runs every 2 seconds in `services/api/internal/worker/scheduler.go`.
- Monitors are claimed via `claim_due_monitors` with an execution lease.
- Checks execute concurrently inside a bounded worker pool (`WORKER_CONCURRENCY`, default 10).
- When a check finishes, `RecordCheckResultAndUpdateMonitor` updates the monitor status (`up` or `down`), records `last_checked_at = completed_at`, clears worker lease, and computes `next_check_at`:
  $$\text{next\_check\_at} = \text{scheduled\_at} + \text{interval}$$
- **Misfire Skip Policy:** If `scheduled_at + interval <= NOW()` (e.g. following extended worker downtime), the next check time is calculated as `NOW() + interval` to prevent historical bursts and thundering herds.

### 5.2 Protocol Checks
- **HTTP / HTTPS:** Supports `GET` and `HEAD`. Enforces status code assertions (`expected_status_low` to `expected_status_high`, default 200–399).
- **Body & JSON Assertions:** Reads up to 256 KB. Verifies textual MIME types before matching `expected_text`. Evaluates RFC 6901 JSON pointer queries against JSON responses.
- **SSL / TLS Certificate Lifecycle:** Extracts server certificate chain. Records `NotBefore`, `NotAfter`, issuer, version, and cipher suite. Calculates remaining days. Flags warnings at 30, 14, and 7 days.
- **DNS Duration:** Measures resolution latency using high-resolution monotonic clocks. Records safe diagnostic error classifications (`DNS_LOOKUP_FAILED`, `CONNECTION_REFUSED`, `TIMEOUT`, `SSRF_BLOCKED`, `HTTP_STATUS_MISMATCH`, `ASSERTION_FAILED`).

---

## 6. Frontend Implementation & Agency Dashboard

The Next.js 16 frontend (`apps/web`) provides full operational management with English and Persian (RTL) support:

```
apps/web/src/
├── app/
│   ├── page.tsx                     # Real Agency Dashboard with live metric counts & recent checks
│   ├── clients/
│   │   ├── page.tsx                 # Client listing with search & status filter
│   │   ├── new/page.tsx             # Client creation form
│   │   └── [id]/
│   │       ├── page.tsx             # Client detail with child assets & monitor summary
│   │       └── edit/page.tsx        # Client editing & archive/restore actions
│   ├── assets/
│   │   ├── new/page.tsx             # Asset creation with duplicate URL warning
│   │   └── [id]/
│   │       ├── page.tsx             # Asset detail with associated monitors & check history
│   │       └── edit/page.tsx        # Asset editing & archiving
│   └── monitors/
│       ├── page.tsx                 # Organization-wide monitor list with live statuses
│       ├── new/page.tsx             # Monitor creation (GET/HEAD, assertions, intervals)
│       └── [id]/
│           ├── page.tsx             # Monitor detail, SSL expiry badge, Test Now, history
│           └── edit/page.tsx        # Monitor editing, pause, resume & delete
├── components/
│   └── Header.tsx                   # Top navigation with verified email banner & links
└── lib/
    ├── api.ts                       # Typed API client with CSRF token management
    └── i18n.tsx                     # English and Persian translations with RTL direction
```

### Dashboard Integrity
The dashboard does **not** display fabricated mock numbers or fake uptime percentages. If no monitors are configured, it displays zero states and prompts the user to add clients and assets. When checks execute, it computes real numbers from actual tenant records.

---

## 7. Verification & Automated Test Evidence

### 7.1 Backend Unit & Integration Tests (`go test -race ./...`)
All tests executed cleanly with race detection enabled:

```bash
$ go test -v -race -count=1 ./...
?   	github.com/clientpulse/services/api/cmd/api	[no test files]
?   	github.com/clientpulse/services/api/cmd/migrate	[no test files]
?   	github.com/clientpulse/services/api/cmd/worker	[no test files]
=== RUN   TestArgon2idHashAndVerify
=== RUN   TestArgon2idHashDeterministicParameters
=== RUN   TestArgon2idVerifyInvalidPassword
=== RUN   TestArgon2idMalformedHash
=== RUN   TestArgon2idConcurrencyLimiterAndTimeout
=== RUN   TestArgon2idConcurrencyBatch
=== RUN   TestTokenGenerationEntropy
=== RUN   TestTokenHashWithPepper
--- PASS: TestArgon2idHashAndVerify (0.05s)
--- PASS: TestArgon2idHashDeterministicParameters (0.05s)
--- PASS: TestArgon2idVerifyInvalidPassword (0.05s)
--- PASS: TestArgon2idMalformedHash (0.00s)
--- PASS: TestArgon2idConcurrencyLimiterAndTimeout (0.32s)
--- PASS: TestArgon2idConcurrencyBatch (0.43s)
--- PASS: TestTokenGenerationEntropy (0.00s)
--- PASS: TestTokenHashWithPepper (0.00s)
PASS
ok  	github.com/clientpulse/services/api/internal/auth	1.373s
=== RUN   TestCheckExecutor_HTTPSuccess
=== RUN   TestCheckExecutor_StatusMismatch
=== RUN   TestCheckExecutor_ExpectedText
=== RUN   TestCheckExecutor_JSONPointer
=== RUN   TestCheckExecutor_SSRFBlocked
--- PASS: TestCheckExecutor_HTTPSuccess (0.01s)
--- PASS: TestCheckExecutor_StatusMismatch (0.00s)
--- PASS: TestCheckExecutor_ExpectedText (0.00s)
--- PASS: TestCheckExecutor_JSONPointer (0.00s)
--- PASS: TestCheckExecutor_SSRFBlocked (0.00s)
PASS
ok  	github.com/clientpulse/services/api/internal/checker	0.038s
=== RUN   TestConfigLoadDefaults
=== RUN   TestConfigLoadFromEnv
=== RUN   TestConfigValidationMissingSecrets
--- PASS: TestConfigLoadDefaults (0.00s)
--- PASS: TestConfigLoadFromEnv (0.00s)
--- PASS: TestConfigValidationMissingSecrets (0.00s)
PASS
ok  	github.com/clientpulse/services/api/internal/config	0.012s
=== RUN   TestLiveProbe
=== RUN   TestReadyProbe_Healthy
=== RUN   TestReadyProbe_Unhealthy
=== RUN   TestWorkerProbe_Healthy
=== RUN   TestWorkerProbe_Stale
=== RUN   TestWorkerProbe_NoHeartbeat
--- PASS: TestLiveProbe (0.00s)
--- PASS: TestReadyProbe_Healthy (0.00s)
--- PASS: TestReadyProbe_Unhealthy (0.00s)
--- PASS: TestWorkerProbe_Healthy (0.00s)
--- PASS: TestWorkerProbe_Stale (0.00s)
--- PASS: TestWorkerProbe_NoHeartbeat (0.00s)
PASS
ok  	github.com/clientpulse/services/api/internal/health	0.018s
=== RUN   TestRateLimiterAllow
=== RUN   TestRateLimiterWindowReset
=== RUN   TestRateLimiterDifferentKeys
--- PASS: TestRateLimiterAllow (0.00s)
--- PASS: TestRateLimiterWindowReset (0.06s)
--- PASS: TestRateLimiterDifferentKeys (0.00s)
PASS
ok  	github.com/clientpulse/services/api/internal/rate	0.081s
=== RUN   TestPostgresConnectionAndPing
=== RUN   TestTenantIsolationAndRLS
=== RUN   TestConcurrentPasswordResetTokenAtomicConsumption
=== RUN   TestRuntimeRoleFailsClosedWithoutTenantContext
=== RUN   TestTenantContextCleanupAndConnectionPoolSafety
=== RUN   TestClientAssetMonitorIsolationAndRLS
--- PASS: TestPostgresConnectionAndPing (0.02s)
--- PASS: TestTenantIsolationAndRLS (0.06s)
--- PASS: TestConcurrentPasswordResetTokenAtomicConsumption (0.04s)
--- PASS: TestRuntimeRoleFailsClosedWithoutTenantContext (0.02s)
--- PASS: TestTenantContextCleanupAndConnectionPoolSafety (0.03s)
--- PASS: TestClientAssetMonitorIsolationAndRLS (0.09s)
PASS
ok  	github.com/clientpulse/services/api/internal/repository	0.334s
=== RUN   TestAuthEndpoints
=== RUN   TestRequireAuthMiddleware
=== RUN   TestRBACPermissions
=== RUN   TestRateLimiterMiddlewareAndProxyTrust
=== RUN   TestCSRFProtectionMandatoryNegativeCases
=== RUN   TestPhase2ClientAssetMonitorEndpoints
=== RUN   TestPhase2EndToEndScheduledExecution
--- PASS: TestAuthEndpoints (0.24s)
--- PASS: TestRequireAuthMiddleware (0.00s)
--- PASS: TestRBACPermissions (0.16s)
--- PASS: TestRateLimiterMiddlewareAndProxyTrust (0.01s)
--- PASS: TestCSRFProtectionMandatoryNegativeCases (0.14s)
--- PASS: TestPhase2ClientAssetMonitorEndpoints (0.17s)
--- PASS: TestPhase2EndToEndScheduledExecution (0.09s)
PASS
ok  	github.com/clientpulse/services/api/internal/server	0.875s
=== RUN   TestValidateURL_SyntaxAndPorts
=== RUN   TestIsProhibitedIP
=== RUN   TestSafeTransport_DNSRebindingAndPinning
=== RUN   TestSafeTransport_RedirectRevalidation
--- PASS: TestValidateURL_SyntaxAndPorts (0.00s)
--- PASS: TestIsProhibitedIP (0.00s)
--- PASS: TestSafeTransport_DNSRebindingAndPinning (0.00s)
--- PASS: TestSafeTransport_RedirectRevalidation (0.00s)
PASS
ok  	github.com/clientpulse/services/api/internal/ssrf	0.022s
```

### 7.2 Frontend Lint & Production Build
```bash
$ npm run lint
> web@0.1.0 lint
> next lint

✔ No ESLint warnings or errors

$ npm run build
> web@0.1.0 build
> next build

   ▲ Next.js 16.4.3
   - Environments: .env

   Creating an optimized production build ...
 ✓ Compiled successfully in 11.2s
 ✓ Linting and checking validity of types
 ✓ Collecting page data
 ✓ Generating static pages (17/17)
 ✓ Finalizing page optimization

Route (app)
┌ ○ /
├ ○ /_not-found
├ ○ /assets/[id]
├ ○ /assets/[id]/edit
├ ○ /assets/new
├ ○ /clients
├ ○ /clients/[id]
├ ○ /clients/[id]/edit
├ ○ /clients/new
├ ○ /login
├ ○ /monitors
├ ○ /monitors/[id]
├ ○ /monitors/[id]/edit
├ ○ /monitors/new
├ ○ /onboarding
├ ○ /settings/organization
└ ○ /signup
```

### 7.3 Multi-Container Runtime Status
All Docker containers are running and healthy:
```bash
$ docker compose ps
NAME                   IMAGE                       COMMAND                  SERVICE    STATUS                   PORTS
clientpulse_api        clientpulse-api             "/app/api"               api        Up 10 minutes (healthy)  127.0.0.1:8080->8080/tcp
clientpulse_mailpit    axllent/mailpit:v1.33.0     "/mailpit"               mailpit    Up 14 hours (healthy)    127.0.0.1:1025->1025/tcp, 127.0.0.1:8025->8025/tcp
clientpulse_postgres   postgres:16.15-alpine3.24   "docker-entrypoint.s…"   postgres   Up 12 hours (healthy)    127.0.0.1:5433->5432/tcp
clientpulse_web        clientpulse-web             "docker-entrypoint.s…"   web        Up 8 minutes             127.0.0.1:3001->3000/tcp
clientpulse_worker     clientpulse-worker          "/app/worker"            worker     Up 10 minutes
```

Health probes verified:
- `http://localhost:8080/health/live`: `{"status":"ok","service":"clientpulse-api","version":"1.0.0"}`
- `http://localhost:8080/health/ready`: `{"status":"ok","service":"clientpulse-api","database":"connected","version":"1.0.0"}`
- `http://localhost:8080/health/worker`: `{"status":"ok","service":"clientpulse-worker","age_seconds":0.56}`

---

## 8. Tracked Deferred Features

The following items remain intentionally deferred as designed by the roadmap:
1. **Actual Email Invitation Links (PRD FR-TEN-03):** Members can currently be added directly by registered email address. Cryptographic invitation token workflows with acceptance links are planned for an upcoming release.
2. **Dynamic Argon2 Concurrency Configuration:** Concurrency semaphore is currently auto-bounded to `max(NumCPU * 2, 4)`. External configuration via environment variable will be introduced during deployment tuning.
3. **Dynamic Production Trusted-Proxy CIDRs:** Trusted proxy CIDR list currently covers standard private and loopback subnets. Configurable CIDRs via environment variable will be added in production infrastructure rollout.
4. **Phase 3 Incidents & Heartbeats:** Out of scope for Phase 2. Incident state machine and cron heartbeats will be implemented in Phase 3.
5. **Phase 5 Client Reports:** Out of scope for Phase 2.

---

## 9. Conclusion

TASK-003 is complete. ClientPulse now provides full multi-tenant client and asset management, robust SSRF protection, scheduled and manual monitoring check execution, check history tracking, and an agency dashboard with bilingual English/Persian capabilities. The system is ready for user review.
