# ClientPulse Security Architecture & Controls

This document details the security architecture, threat model, and defensive controls implemented and audited in ClientPulse Phase 1 according to `PRD.md` and TASK-002.1 security audit requirements.

---

## 1. Multi-Tenant Data Isolation & Database Role Separation (Defense in Depth)

Multi-tenant isolation is the highest-priority security requirement of ClientPulse. Defense in depth is enforced across five distinct architectural layers:

```
+-------------------------------------------------------------------------+
| Layer 1: Server-Side Membership Resolution & Validation                 |
| - Tenant context is never trusted blindly from client input.             |
| - Header X-Organization-ID is validated against user's DB memberships.   |
+-------------------------------------------------------------------------+
                                    |
                                    v
+-------------------------------------------------------------------------+
| Layer 2: Explicit Repository Scoping                                    |
| - Repository methods require organization_id parameter.                 |
| - SQL queries explicitly enforce WHERE organization_id = $1.            |
+-------------------------------------------------------------------------+
                                    |
                                    v
+-------------------------------------------------------------------------+
| Layer 3: Transaction-Scoped Connection Context (SET LOCAL)              |
| - Database transactions execute:                                        |
|   SET LOCAL ROLE clientpulse_app;                                       |
|   SELECT set_config('app.current_organization_id', $orgID, true);      |
| - is_local = true guarantees setting cleanup on COMMIT or ROLLBACK.     |
| - Prevents tenant context leakage across pooled pgx connections.        |
+-------------------------------------------------------------------------+
                                    |
                                    v
+-------------------------------------------------------------------------+
| Layer 4: PostgreSQL Row-Level Security (FORCE ROW LEVEL SECURITY)        |
| - Tenant tables (audit_logs, organization_memberships, etc.) enforce    |
|   FORCE ROW LEVEL SECURITY.                                             |
| - RLS fails closed (returns 0 rows) when tenant context is unset.       |
| - Direct WHERE clauses cannot bypass tenant RLS policy.                 |
+-------------------------------------------------------------------------+
                                    |
                                    v
+-------------------------------------------------------------------------+
| Layer 5: Database Role Privilege Separation (Least Privilege)           |
| - Migration user (clientpulse): superuser/table owner for migrations.   |
| - Runtime user (clientpulse_app): NOSUPERUSER, NOBYPASSRLS.             |
| - Runtime user does not own tables; cannot ALTER, DROP, or disable RLS. |
| - audit_logs is append-only: UPDATE and DELETE are revoked.             |
+-------------------------------------------------------------------------+
```

### 1.1 Database Role Separation (Least Privilege Architecture)
Database access is divided across four specialized PostgreSQL roles:
- **Migration Role (`clientpulse`):** Used strictly during deployment and migration execution (`cmd/migrate`) to apply DDL schema changes. Configured via `MIGRATION_DATABASE_URL`.
- **Application Runtime Role (`clientpulse_app`):** Used exclusively by the runtime API service.
  - Defined with `NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS`.
  - Non-owner of database tables; cannot execute `ALTER TABLE`, `DROP TABLE`, or disable RLS.
  - Grants:
    - Business tables (`users`, `organizations`, `organization_memberships`, `sessions`, `password_reset_tokens`, `email_verification_tokens`, `clients`, `assets`, `monitors`, `check_results`, `check_aggregates_hourly`): `SELECT, INSERT, UPDATE, DELETE` (subject to tenant RLS).
    - Table (`audit_logs`): `SELECT, INSERT` ONLY. `UPDATE` and `DELETE` are revoked at the SQL grant level.
    - Explicitly revoked from executing `claim_due_monitors`.
- **Worker Runtime Role (`clientpulse_worker`):** Used exclusively by the monitoring worker daemon.
  - Defined with `LOGIN`, `NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS`.
  - Separate credentials configured via `WORKER_DATABASE_URL`.
  - Grants:
    - Only tables required for monitoring: `SELECT, UPDATE` on `monitors`; `SELECT` on `assets`; `INSERT, SELECT, DELETE` on `check_results`; `SELECT, INSERT, UPDATE, DELETE` on `check_aggregates_hourly`; `SELECT, INSERT, UPDATE` on `worker_heartbeats`.
    - Revoked from all credential and audit tables: `users`, `sessions`, `password_reset_tokens`, `email_verification_tokens`, `audit_logs`.
    - Sole role granted `EXECUTE ON FUNCTION claim_due_monitors`.
    - Scoped worker RLS policies allow cross-tenant bounded cleanup and aggregation.
- **Restricted Function Owner Role (`clientpulse_definer`):**
  - Defined with `NOLOGIN`, `NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS`.
  - Owns `claim_due_monitors` with explicit `search_path = pg_catalog, public, pg_temp`.
  - `CREATE ON SCHEMA public` is revoked to prevent untrusted object creation.

### 1.2 Composite Tenant Foreign Keys
Cross-tenant parent-child data tampering is impossible even if application validation or RLS were bypassed, enforced by composite database constraints:
- `clients(organization_id, id)` UNIQUE
- `assets(organization_id, id)` UNIQUE
- `monitors(organization_id, id)` UNIQUE
- `assets(organization_id, client_id) REFERENCES clients(organization_id, id) ON DELETE CASCADE`
- `monitors(organization_id, asset_id) REFERENCES assets(organization_id, id) ON DELETE CASCADE`
- `check_results(organization_id, monitor_id) REFERENCES monitors(organization_id, id) ON DELETE CASCADE`
- `check_aggregates_hourly(organization_id, monitor_id) REFERENCES monitors(organization_id, id) ON DELETE CASCADE`

Cross-organization relationships fail unconditionally with PostgreSQL foreign key constraint errors.

### 1.3 Connection Pool Safety & Fail-Closed RLS
- Because `pgxpool` multiplexes physical connections across incoming HTTP requests, global session variables (`SET app.current_organization_id`) are strictly prohibited.
- Repository operations use `WithTenantTx(ctx, pool, orgID, fn)` which binds `set_config('app.current_organization_id', $orgID, true)`. The third parameter `is_local = true` ensures that PostgreSQL automatically clears the tenant context immediately upon transaction `COMMIT` or `ROLLBACK`.
- Automated regression test `TestTenantContextCleanupAndConnectionPoolSafety` verifies that physical pooled connections released to the pool retain an empty string `""` for `app.current_organization_id`.
- Automated regression test `TestRuntimeRoleFailsClosedWithoutTenantContext` verifies that queries executed without setting `app.current_organization_id` return 0 rows (fail closed).

---

## 2. Authentication & Session Security Model

### 2.1 Password Hashing: Argon2id with Bounded Concurrency
- **Algorithm:** Argon2id (hybrid memory-hard algorithm, OWASP recommended).
- **Parameters:**
  - Memory: 64 MiB (`65536` KiB)
  - Iterations (Time): 1 pass
  - Parallelism: 4 threads
  - Salt: 16 cryptographically secure random bytes (`crypto/rand`)
  - Key length: 32 bytes
- **Benchmark Measurements (Development Container / x86_64 CPU):**
  - Hash latency: **38.7 ms**
  - Verification latency: **45.5 ms**
  - Memory allocation: **~64 MiB/op** (`67,117,466 B/op`)
  - Concurrent batch latency (16 parallel operations): **349.8 ms** with 0 errors.
- **Bounded Concurrency Semaphore:**
  - Because 64 MiB Argon2id allocations could exhaust container memory under distributed login flood attacks, the cryptographic engine enforces a process-wide concurrency semaphore bounded to `max(runtime.NumCPU() * 2, 4)`.
  - Excessive requests that cannot acquire a semaphore slot within 3 seconds are shed with `ErrCryptoBusy` (`503 Service Unavailable`), preventing memory exhaustion (OOM) and CPU starvation.

### 2.2 Token Generation, Hashing & Atomic Consumption
- **Token Entropy:** Generated using CSPRNG (`crypto/rand`), 32 bytes URL-safe base64 (256 bits of entropy).
- **Hash with Pepper:** Raw tokens (sessions, password reset tokens, email verification tokens) are **never** stored in plaintext. They are hashed using SHA-256 combined with a 32+ byte server-side pepper (`TOKEN_HASH_PEPPER`).
- **Atomic Single-Use Token Consumption:**
  - Password reset tokens and email verification tokens are consumed atomically using a single SQL statement:
    ```sql
    UPDATE password_reset_tokens
    SET used_at = NOW()
    WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
    RETURNING id, user_id, expires_at, created_at, used_at;
    ```
  - This eliminates check-then-act race conditions.
  - Automated concurrency test `TestConcurrentPasswordResetTokenAtomicConsumption` verifies that out of 10 simultaneous goroutines attempting to consume the same token, exactly 1 succeeds and 9 fail.

### 2.3 Cookie Security & Storage
- **Session Cookie:** `clientpulse_session`
  - `HttpOnly: true` (inaccessible to JavaScript, immune to XSS token theft)
  - `SameSite: Lax`
  - `Secure: true` in production environments (`false` in local development)
  - `Path: /`
- **Revocation:** Explicit logout marks `revoked_at = NOW()`. Password changes or user-initiated revocation invalidate all user sessions server-side.
- **Client Storage Policy:** The Next.js frontend never stores raw session secrets or credentials in `localStorage` or `sessionStorage`. All authenticated state is tracked via HttpOnly session cookies.

### 2.4 CSRF Protection: Session-Bound Cryptographic Signatures
- **Architecture:** Session-Bound Signed Double-Submit Cookie pattern with defense-in-depth origin validation.
- **Token Composition:**
  - `CSRFToken = Base64URL(Nonce) + "." + Base64URL(HMAC-SHA256(Nonce + ":" + SessionID, SessionSecret))`
- **Verification Pipeline:**
  1. **Cross-Site Blocking:** Requests bearing `Sec-Fetch-Site: cross-site` on mutating methods are rejected immediately (`403 CSRF_CROSS_SITE_BLOCKED`).
  2. **Strict Origin & Referer Verification:** Mutating requests must have an `Origin` (or fallback `Referer`) strictly matching the whitelisted application origins (`APP_BASE_URL` or `API_BASE_URL`).
  3. **Double-Submit Matching:** The `clientpulse_csrf` cookie value and the `X-CSRF-Token` HTTP header value must match in constant time (`subtle.ConstantTimeCompare`).
  4. **Session Binding Cryptographic Verification:** The token's HMAC is verified against the authenticated user's `SessionID`. Replaying a valid CSRF token from Session B against Session A is rejected (`403 CSRF_TOKEN_INVALID`).
  5. **Session Revocation Guard:** If a session is revoked or expired, the request fails with `401 Unauthorized` before any state-changing handler can execute.
- **Mandatory Negative Regression Tests (`TestCSRFProtectionMandatoryNegativeCases`):**
  - Missing CSRF token -> 403 Forbidden.
  - Mismatched CSRF header vs cookie -> 403 Forbidden.
  - Forged CSRF cookie -> 403 Forbidden.
  - Token from another session -> 403 Forbidden.
  - Cross-origin state-changing request -> 403 Forbidden.
  - `Sec-Fetch-Site: cross-site` -> 403 Forbidden.
  - Revoked session with previously valid token -> 401 Unauthorized.

### 2.5 Rate Limiting & Proxy Trust
- In-memory sliding window rate limiters protect all sensitive authentication endpoints:
  - `POST /api/v1/auth/signup`: 10 requests / min / IP
  - `POST /api/v1/auth/login`: 10 requests / min / IP
  - `POST /api/v1/auth/forgot-password`: 5 requests / min / IP
  - `POST /api/v1/auth/reset-password`: 10 requests / min / IP
  - `POST /api/v1/auth/verify-email`: 15 requests / min / IP
  - `POST /api/v1/auth/resend-verification`: 5 requests / min / IP
- **Proxy Trust CIDR Validation:**
  - Untrusted clients cannot bypass IP rate limiting by spoofing the `X-Forwarded-For` header.
  - `X-Forwarded-For` is only inspected if `RemoteAddr` belongs to a trusted reverse proxy CIDR (`127.0.0.0/8`, `::1/128`, `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`).
  - Automated test `TestRateLimiterMiddlewareAndProxyTrust` verifies that spoofed headers from untrusted remote IPs are discarded.

---

## 3. Role-Based Access Control (RBAC)

RBAC is strictly enforced on the API backend via middleware (`RequireRole(minRole)`):

| Capability | Owner | Admin | Member | Viewer |
| :--- | :---: | :---: | :---: | :---: |
| View Dashboards & Monitors | Yes | Yes | Yes | Yes |
| Log Maintenance Activity | Yes | Yes | Yes | No |
| Manage Monitors & Assets | Yes | Yes | No | No |
| Update Organization Settings | Yes | Yes | No | No |
| Invite / Add Members | Yes | Yes | No | No |
| Remove Members | Yes | Yes (cannot remove Owner) | No | No |
| Update Member Roles | Yes | No | No | No |
| Transfer / Delete Organization | Yes | No | No | No |

- An organization must retain at least one active Owner. Attempting to demote or remove the sole Owner is rejected with `400 Bad Request`.
- Self-promotion is prevented: Admins cannot promote themselves or others to Owner.

> [!NOTE]
> **Email Invitations Status:** In Phase 1, `POST /api/v1/organizations/current/members` allows adding already registered users directly by email address. True pending email invitation links with cryptographic invitation tokens (PRD FR-TEN-03) will be implemented as an explicit feature in a subsequent phase.

---

## 4. HTTP Security Headers

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Access-Control-Allow-Origin`: Explicit whitelist (`APP_BASE_URL`).
- `Access-Control-Allow-Credentials: true`
- `Access-Control-Allow-Headers: Content-Type, Authorization, X-Request-ID, X-CSRF-Token, X-Organization-ID`

---

## 5. Server-Side Request Forgery (SSRF) Protection

Because ClientPulse allows users to configure arbitrary HTTP/HTTPS endpoints for automated uptime monitoring, SSRF is a critical threat vector. A defense-in-depth security guard (`services/api/internal/ssrf`) is enforced across both the API server (at creation/update time) and the monitoring worker (at execution time).

### 5.1 Scheme and Port Restrictions
- **Allowed Schemes:** `http://` and `https://` only. All other URI schemes (`file://`, `gopher://`, `ftp://`, `ldap://`, `dict://`) are rejected.
- **Allowed Ports:** Standard ports `80` (HTTP) and `443` (HTTPS). Non-standard ports (e.g. `22`, `3306`, `5432`, `6379`, `8080`, `27017`) are blocked.
- **Prohibited Hostnames:**
  - `localhost` and all `*.localhost` subdomains.
  - `.local`, `.internal`, `.lan`, `.home.arpa` internal zones.
  - Metadata hostnames: `metadata.google.internal`, `instance-data`.
  - Non-FQDN hostnames without valid public dots.
  - URLs containing embedded credentials (userinfo `user:pass@host`).

### 5.2 Prohibited IP Address Classification
Every resolved IP address is classified against an explicit blocklist of non-public CIDR ranges:

| Classification | Blocked CIDR Range(s) | Threat Vector |
|---|---|---|
| IPv4 Private Networks | `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16` | RFC 1918 internal services & databases |
| IPv4 Loopback | `127.0.0.0/8` | Localhost services & IPC |
| IPv4 Link-Local / Cloud Metadata | `169.254.0.0/16` | AWS/GCP/Azure IMDS endpoints |
| IPv4 Carrier-Grade NAT | `100.64.0.0/10` | RFC 6598 ISP shared networks |
| IPv4 Multicast & Reserved | `224.0.0.0/4`, `240.0.0.0/4`, `0.0.0.0/8` | Broadcast / future use attacks |
| IPv6 Loopback | `::1/128` | Localhost loopback |
| IPv6 Unique Local (ULA) | `fc00::/7` | RFC 4193 private IPv6 services |
| IPv6 Link-Local | `fe80::/10` | Link-local host discovery |
| IPv6 Cloud Metadata | `fd00:ec2::254/128` | AWS Nitro IMDSv2 IPv6 |
| IPv4-Mapped IPv6 | `::ffff:0:0/96` (and `::ffff:127.0.0.1`) | Bypass via dual-stack encapsulation |

### 5.3 DNS Rebinding Defense & Socket Pinning
1. Target hostnames are resolved using `net.DefaultResolver.LookupIPAddr`.
2. If **any** returned IP address belongs to a prohibited CIDR range, the target is rejected immediately.
3. The first validated public IP is pinned directly to the outbound TCP socket via a custom `SafeDialContext`.
4. The HTTP transport bypasses secondary OS DNS lookups during TCP handshake (`dialer.DialContext(ctx, "tcp", net.JoinHostPort(pinnedIP.String(), port))`).
5. The original hostname is explicitly passed to `tls.Config.ServerName` and HTTP `Host` headers to guarantee valid TLS verification and HTTP vhost routing.
6. `http.Client.CheckRedirect` runs the full SSRF validation pipeline on every redirect URL before following. Redirects to private IPs or metadata endpoints are blocked.
7. `http.Transport.Proxy` is explicitly set to `nil` to prevent environment proxy variables (`HTTP_PROXY`, `ALL_PROXY`) from intercepting or bypassing SSRF guards.

### 5.4 Response Safety Limits
- **Maximum Response Body Inspection:** 256 KB (`io.LimitReader(resp.Body, 256*1024)`). Prevents memory exhaustion attacks via streaming responses.
- **Content-Type Validation:** Expected text searches require textual MIME types (`text/`, `application/json`, `application/xml`). Binary responses are never matched.
- **Sensitive Data Redaction:** Raw response bodies are never logged or stored in PostgreSQL. Only latency, HTTP status code, TLS metadata, and safe error codes are persisted.

---

## 6. Email Verification Requirement on Sensitive Operations

As mandated by PRD **FR-AUTH-01**, unverified accounts have restricted access:
- Authentication, logging in, changing password, and viewing dashboard metrics are allowed.
- All state-mutating agency operations are protected by `RequireVerifiedEmail` middleware:
  - Client creation, update, and archiving (`POST/PATCH/DELETE /api/v1/clients/*`)
  - Asset creation, update, and archiving (`POST/PATCH/DELETE /api/v1/assets/*`)
  - Monitor creation, update, deletion, pause, resume, and on-demand testing (`POST/PATCH/DELETE /api/v1/monitors/*`)
  - Organization member additions and role changes (`POST/PATCH/DELETE /api/v1/organizations/current/members/*`)
- Unverified requests to these endpoints are rejected with `403 Forbidden` (`code: EMAIL_NOT_VERIFIED`). The UI displays an email verification banner guiding the user to check their email.

---

## 7. Incident & Heartbeat Monitoring Security Controls

### 7.1 Heartbeat Token Protection & Secret Hygiene
1. **Cryptographic Entropy:** All heartbeat tokens are generated using 32 bytes of cryptographic randomness (`crypto/rand`), formatted as a 64-character lowercase hexadecimal string (256 bits of entropy).
2. **Digest-Only Storage:** Plaintext tokens are NEVER stored in PostgreSQL. Only the SHA-256 digest (`token_hash`) is persisted.
3. **Log Sanitization:** Raw tokens are strictly excluded from all application logs, database query parameters, error reports, and APM telemetry.
4. **Single-Reveal Display:** The UI presents the raw secret token only once during creation or token rotation. Once dismissed, the token cannot be retrieved via any API endpoint.
5. **Immediate Invalidation on Rotation / Revocation:** Rotating a token immediately calculates a new hash and replaces the stored digest in the database. Any subsequent ping using the prior token is immediately rejected. Revoking zeroes out the digest and marks the status as `revoked`.

### 7.2 Public Ping Endpoint Hardening (`POST /api/v1/heartbeat-pings/{token}`)
1. **Unauthenticated by Design:** Accepts pings without user session cookies to support unattended cron jobs and backup scripts. Authentication is purely cryptographic via token verification.
2. **Untrusted Caller Identifiers:** The caller cannot provide an organization ID or client ID. Tenant context is derived strictly by matching the SHA-256 digest via SECURITY DEFINER function `lookup_heartbeat_by_token_hash`.
3. **Payload Clamping & Strict Method:** Rejects any HTTP method other than `POST` with `405 Method Not Allowed`. Enforces a strict 1 KB body limit (`http.MaxBytesReader`) to thwart denial-of-service attempts.
4. **Independent Sliding-Window Rate Limiting:** Rate-limited independently of web session endpoints at 120 pings per minute per source IP address.
5. **Enumeration Resistance:** Invalid, revoked, or non-existent tokens return identical generic `404 Not Found` responses to prevent token existence or rotation state discovery.

### 7.3 Multi-Tenant Incident Model Integrity
1. **Anti-Polymorphism:** Incidents do not rely on untyped `target_type` + `target_id` pairs. Strong foreign keys (`monitor_id`, `heartbeat_monitor_id`) are enforced with database check constraints.
2. **Composite Tenant Foreign Keys:** Cross-tenant target reference is impossible at the database engine level via composite keys:
   - `(organization_id, monitor_id) REFERENCES monitors(organization_id, id)`
   - `(organization_id, heartbeat_monitor_id) REFERENCES heartbeat_monitors(organization_id, id)`
   - `(organization_id, first_failure_result_id) REFERENCES check_results(organization_id, id)`
   - `(organization_id, recovery_result_id) REFERENCES check_results(organization_id, id)`
3. **RLS Definer Policy Coexistence:** To prevent bypass of `FORCE ROW LEVEL SECURITY` while allowing atomic background worker claims across tenants, definer functions (`claim_missed_heartbeats`, `lookup_heartbeat_by_token_hash`) are owned by `clientpulse_definer` with explicit RLS definer policies (`heartbeat_monitors_definer_policy`, `monitors_definer_policy`). Runtime users cannot assume this definer role.
4. **Append-Only Immutable Event Log:** `notification_events` rows are strictly append-only. UPDATE and DELETE privileges are revoked from the application role. Deterministic idempotency keys prevent duplicate notification spam during flapping or repeated checks.

