# ADR 0002: Authentication Architecture, Session Security, and Multi-Tenant Isolation

**Date:** 2026-10-31 (Updated 2026-11-01 per TASK-002.1 Security Audit)  
**Status:** Accepted & Hardened  
**Deciders:** Core Engineering Team  
**Consulted:** PRD.md (Sections 3.5, 6.1, 6.2, 8.1, 9.7, 12.1, 12.2, 12.4, 16.3)

---

## 1. Context

ClientPulse is a multi-tenant application where multiple agencies, each managing confidential client systems and sensitive uptime reports, share the same database infrastructure. Data leakage across tenants would severely compromise customer trust and data privacy.

Furthermore, user authentication must follow modern security best practices: secure password hashing, protection against credential stuffing, session hijacking, CSRF, and user enumeration.

Following TASK-002.1 security audit, the implementation underwent verification, benchmarking, and defensive hardening.

---

## 2. Key Decisions

### 2.1 Password Hashing: Argon2id with Bounded Concurrency
- **Algorithm:** Argon2id (OWASP recommended parameters: 64 MiB memory, 1 iteration, 4 parallelism, 16-byte random salt, 32-byte key).
- **Benchmark Evidence:** Benchmarked on x86_64 CPU in Docker container: single hash latency is **38.7 ms**, verification is **45.5 ms**, allocating **~64 MiB/op**. 16 concurrent operations complete in **349.8 ms**.
- **Bounded Concurrency:** To prevent memory exhaustion (OOM) and CPU starvation from concurrent authentication floods, a process-wide bounded semaphore limits concurrent Argon2id operations to `max(runtime.NumCPU() * 2, 4)` with a 3-second acquisition timeout, returning `503 Service Unavailable` (`ErrCryptoBusy`) under excessive load.

### 2.2 Token Generation, Pepper Hashing & Atomic Consumption
- **Entropy & Pepper:** 32-byte CSPRNG tokens are hashed with SHA-256 and a 32+ byte server-side pepper (`TOKEN_HASH_PEPPER`). Raw tokens are never stored in the database.
- **Atomic Single-Use Consumption:** Password reset and email verification tokens are consumed using single atomic SQL `UPDATE ... WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW() RETURNING ...` queries, eliminating check-then-act race conditions.
- **Cookie Security:** Sessions are tracked via `clientpulse_session` cookies (`HttpOnly=true`, `SameSite=Lax`, `Secure=true` in production). Frontend never stores tokens in `localStorage` or `sessionStorage`.

### 2.3 CSRF Protection: Session-Bound Cryptographic Signatures
- **Architecture:** Replaced naive double-submit cookie with a Session-Bound Signed Double-Submit Cookie pattern.
- **Signature:** `CSRFToken = Base64URL(Nonce) + "." + Base64URL(HMAC-SHA256(Nonce + ":" + SessionID, SessionSecret))`.
- **Validation Pipeline:**
  1. Blocks `Sec-Fetch-Site: cross-site` requests on mutating methods.
  2. Enforces strict `Origin` (or fallback `Referer`) matching against allowed origins (`APP_BASE_URL`, `API_BASE_URL`).
  3. Constant-time match between `clientpulse_csrf` cookie and `X-CSRF-Token` header.
  4. Cryptographic HMAC verification bound to the authenticated `SessionID`. Replaying tokens from another session is rejected (`403 CSRF_TOKEN_INVALID`).
  5. Session revocation immediately invalidates CSRF capability (`401 Unauthorized`).

### 2.4 Multi-Tenant Isolation Strategy & Database Role Separation
Isolation is enforced across five distinct layers:
1. **Server-Side Membership Validation:** Every tenant-scoped request resolves tenant ID from verified user memberships.
2. **Explicit Repository Scoping:** Repository methods enforce `WHERE organization_id = $org_id`.
3. **Database Role Separation:**
   - Dedicated migration superuser role `clientpulse` applies DDL migrations.
   - Dedicated application runtime role `clientpulse_app` (`NOSUPERUSER`, `NOBYPASSRLS`). The runtime role does not own tables and cannot run `ALTER TABLE`, `DROP TABLE`, or disable RLS.
   - `audit_logs` table permissions are strictly `SELECT, INSERT` (append-only); `UPDATE` and `DELETE` are revoked at SQL grant level.
4. **PostgreSQL Row-Level Security (FORCE ROW LEVEL SECURITY):**
   - Tenant tables enforce RLS: `USING (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid)`.
   - Missing tenant context fails closed (returns 0 rows).
5. **Transaction-Scoped Tenant Context (`SET LOCAL`):**
   - Tenant context is set via `set_config('app.current_organization_id', $1, true)` within an explicit transaction.
   - PostgreSQL purges setting automatically upon `COMMIT` or `ROLLBACK`, guaranteeing zero context leakage across pooled `pgx` connections.

### 2.5 Role-Based Access Control (RBAC)
- Enforces `Owner`, `Admin`, `Member`, `Viewer` roles strictly in backend middleware and repositories.
- An organization must retain at least one active Owner; sole Owner demotion or removal is blocked (`400 Bad Request`).
- Self-promotion to higher roles is blocked.
- *Phase 1 Note:* `POST /api/v1/organizations/current/members` attaches existing registered users directly. True token-based pending email invitation flows (PRD FR-TEN-03) are deferred to a subsequent phase.

### 2.6 Rate Limiting & Proxy Trust
- In-memory rate limiting applied to `/signup` (10/min), `/login` (10/min), `/forgot-password` (5/min), `/reset-password` (10/min), `/verify-email` (15/min), and `/resend-verification` (5/min).
- Client IP resolution enforces CIDR trust validation: `X-Forwarded-For` is only accepted from trusted reverse proxies (`127.0.0.0/8`, `::1/128`, `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`). Untrusted clients spoofing headers cannot bypass rate limits.

