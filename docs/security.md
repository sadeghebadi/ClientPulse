# ClientPulse Security Architecture & Controls

This document details the security architecture and defensive controls implemented in ClientPulse according to PRD.md requirements.

---

## 1. Multi-Tenant Data Isolation (Defense in Depth)

Multi-tenant isolation is the highest-priority security requirement of ClientPulse. Defense in depth is enforced across four distinct architectural layers:

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
| - Repository methods require organization_id.                           |
| - SQL queries enforce WHERE organization_id = $1.                       |
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
| Layer 4: PostgreSQL Row-Level Security (RLS)                            |
| - Tenant tables (audit_logs, organization_memberships, etc.) enforce    |
|   FORCE ROW LEVEL SECURITY.                                             |
| - Application role clientpulse_app has NOBYPASSRLS.                     |
| - Policy restricts rows to NULLIF(current_setting(...), '')::uuid.      |
+-------------------------------------------------------------------------+
```

### Connection Pool Safety
Because `pgxpool` reuses physical connections across HTTP requests:
1. Setting global session variables (`SET app.current_organization_id`) is strictly prohibited.
2. The repository layer executes tenant transactions using `WithTenantTx(ctx, pool, orgID, fn)`.
3. Inside the transaction, `SET LOCAL ROLE clientpulse_app` and `set_config(..., ..., true)` are executed.
4. When the transaction commits or rolls back, PostgreSQL automatically purges the transaction-local setting and restores the pool connection state.
5. Automated tests (`TestTenantContextCleanupAndConnectionPoolSafety` and `TestConcurrentTenantQueries`) continuously verify that 0 context leaks occur across pooled connections and concurrent goroutines.

---

## 2. Authentication & Session Security Model

### 2.1 Password Hashing: Argon2id
- **Algorithm:** Argon2id (hybrid memory-hard algorithm)
- **Parameters (OWASP Recommended):**
  - Memory: 64 MiB (`65536` KiB)
  - Iterations (Time): 1 pass
  - Parallelism: 4 threads
  - Salt: 16 cryptographically secure random bytes (`crypto/rand`)
  - Key length: 32 bytes
- **Comparison:** `crypto/subtle.ConstantTimeCompare` protects against timing attacks.
- **Enforcement:** Passwords must be at least 12 characters. Plaintext passwords and hashes are never logged or exposed in JSON responses.

### 2.2 Token Generation & Hashing
- **Raw Generation:** Generated with CSPRNG (`crypto/rand`), 32 bytes URL-safe base64.
- **Storage:** Raw tokens (sessions, password reset tokens, email verification tokens) are **never** stored in the database.
- **Hash with Pepper:** Tokens are stored as HMAC-SHA256 hashes keyed with a server-side secret (`TOKEN_HASH_PEPPER`). If the database is compromised, active session or reset tokens cannot be forged or extracted.

### 2.3 Cookie Security
- **Session Cookie:** `clientpulse_session`
  - `HttpOnly: true` (inaccessible to JavaScript, immune to XSS token theft)
  - `SameSite: Lax`
  - `Secure: true` in production environments
  - `Path: /`
- **Revocation:** Explicit logout marks `revoked_at = NOW()`. Password changes or user-initiated "Revoke all sessions" invalidate all user sessions server-side.

### 2.4 CSRF Protection
- **Mechanism:** Double-Submit Cookie pattern paired with custom header verification (`X-CSRF-Token`).
- **Operation:**
  - Upon authentication, a random token is issued in readable cookie `clientpulse_csrf` (`SameSite=Lax`).
  - All mutating requests (`POST`, `PUT`, `PATCH`, `DELETE`) authenticated via cookie must present the `X-CSRF-Token` header.
  - Safe methods (`GET`, `HEAD`, `OPTIONS`) are exempt.
  - API requests using `Authorization: Bearer ...` are exempt because cross-origin browser requests cannot inject custom authorization headers without CORS preflight.

### 2.5 User Enumeration Mitigation
- `POST /api/v1/auth/forgot-password` returns `200 OK` with a generic message ("If this email is registered, password reset instructions have been sent") regardless of whether the email exists.
- Login failures return generic `INVALID_CREDENTIALS` for both non-existent users and bad passwords.
- Rate limiting prevents automated enumeration scans.

### 2.6 Rate Limiting
- In-memory sliding window rate limiter protects sensitive endpoints:
  - `/api/v1/auth/login`: 10 requests / minute / IP
  - `/api/v1/auth/forgot-password`: 5 requests / minute / IP
- Returns `429 Too Many Requests` when limits are exceeded.

---

## 3. Role-Based Access Control (RBAC)

Roles are enforced strictly on the backend:

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

---

## 4. HTTP Security Headers

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Access-Control-Allow-Origin`: Explicit whitelist (`APP_BASE_URL`).
- `Access-Control-Allow-Credentials: true`
- `Access-Control-Allow-Headers: Content-Type, Authorization, X-Request-ID, X-CSRF-Token, X-Organization-ID`
