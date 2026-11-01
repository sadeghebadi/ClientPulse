# ADR 0002: Authentication Architecture, Session Security, and Multi-Tenant Isolation

**Date:** 2026-10-31  
**Status:** Accepted  
**Deciders:** Core Engineering Team  
**Consulted:** PRD.md (Sections 3.5, 6.1, 6.2, 8.1, 9.7, 12.1, 12.2, 12.4, 16.3)

---

## 1. Context

ClientPulse is a multi-tenant application where multiple agencies, each managing confidential client systems and sensitive uptime reports, share the same database infrastructure. Data leakage across tenants would severely compromise customer trust and data privacy.

Furthermore, user authentication must follow modern security best practices: secure password hashing, protection against credential stuffing, session hijacking, CSRF, and user enumeration.

---

## 2. Key Decisions

### 2.1 Password Hashing: Argon2id
- **Decision:** Use **Argon2id** (the hybrid version of Argon2 combining Argon2i's side-channel resistance with Argon2d's GPU/ASIC cracking resistance) per OWASP recommendations.
- **Parameters:**
  - Memory: 64 MiB (`65536` KiB)
  - Iterations (Time): 1 pass
  - Parallelism (Threads): 4
  - Salt: 16 cryptographically secure random bytes (`crypto/rand`)
  - Key length: 32 bytes
- **Encoding:** Standard PHC string format `$argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>`. Verification uses `crypto/subtle.ConstantTimeCompare` to avoid timing side-channels.

### 2.2 Session Security Model
- **Token Generation:** 32 cryptographically secure random bytes (`crypto/rand`), encoded as URL-safe base64.
- **Database Storage:** Raw session tokens are **never** stored in the database. Instead, only the SHA-256 hash combined with a server-side secret pepper (`TOKEN_HASH_PEPPER`) is stored in the `sessions` table (`token_hash`). If the database is compromised, active session tokens cannot be derived.
- **Transport via HttpOnly Cookie:**
  - Cookie name: `clientpulse_session`
  - Flags: `HttpOnly=true`, `SameSite=Lax`, `Path=/`
  - In production (`APP_ENV=production`): `Secure=true`.
  - Fallback: Bearer authentication in `Authorization` header is supported for headless API clients.
- **Revocation:** Explicit logout marks the session `revoked_at = NOW()`. Password changes or user-initiated "Revoke all sessions" invalidate all sessions associated with that user ID.
- **Active Organization:** The session tracks `active_organization_id`. However, on every request, the user's active membership in that organization is validated against the `organization_memberships` table. A user cannot access an organization they do not belong to.

### 2.3 CSRF Protection
- **Decision:** Double-Submit Cookie pattern paired with custom header validation for cookie-authenticated mutating requests (`POST`, `PUT`, `PATCH`, `DELETE`).
- **Mechanism:**
  - On authentication, a random CSRF token is issued in a readable cookie `clientpulse_csrf` (`SameSite=Lax`, `Path=/`).
  - Mutating requests authenticated via cookies must present this token in the `X-CSRF-Token` header.
  - Safe methods (`GET`, `HEAD`, `OPTIONS`) are exempt.
  - Bearer token requests authenticated via `Authorization: Bearer ...` are exempt because cross-origin form submissions cannot inject custom headers without CORS preflight.

### 2.4 Multi-Tenant Isolation Strategy (Defense in Depth)
Isolation is enforced across four distinct layers:

1. **Server-Side Membership Validation:**
   - Every tenant-scoped request resolves the active tenant ID from the authenticated user's verified memberships.
   - Any client-supplied tenant ID (e.g., via `X-Organization-ID` or request payload) is checked against the user's explicit memberships. If the user is not an active member, the request is rejected with `403 Forbidden` / `404 Not Found`.

2. **Explicit Repository Scoping:**
   - All repository methods interacting with tenant records require an explicit `orgID uuid.UUID` parameter and execute SQL queries with `WHERE organization_id = $org_id`.

3. **PostgreSQL Row-Level Security (RLS):**
   - Tenant-scoped tables (`organization_memberships`, `audit_logs`, and future tenant tables) have RLS enabled.
   - Policy: `USING (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid)`.
   - Even if application code omitted an `organization_id` filter due to a bug, PostgreSQL enforces isolation at the engine level.

4. **Transaction-Scoped Tenant Context (`SET LOCAL`):**
   - Because `pgxpool` reuses physical connections across requests, setting a global session variable (`SET app.current_organization_id`) risks leaking context to subsequent queries using that connection.
   - Therefore, tenant context is set using `SET LOCAL app.current_organization_id = $1` inside an explicit database transaction (`pgx.Tx`).
   - In PostgreSQL, `SET LOCAL` is automatically cleared upon transaction `COMMIT` or `ROLLBACK`.
   - In addition, non-transactional connection helpers guarantee setting cleanup in a `defer` block.

### 2.5 Role-Based Access Control (RBAC)
- Roles defined in PRD Section 3.5:
  - **Owner:** Full organization access, member invitations, role updates, organization deletion/transfer.
  - **Admin:** Operational and configuration access; cannot transfer ownership or remove/demote the Owner.
  - **Member:** Standard operational access; cannot manage members or modify organization settings.
  - **Viewer:** Read-only access.
- Role checks are performed by backend middleware (`RequireRole(RoleAdmin)`) and service logic, preventing unauthorized privilege escalation.

### 2.6 User Enumeration Mitigation
- `POST /api/v1/auth/forgot-password` returns `200 OK` with a generic message ("If the email is registered, instructions have been sent") regardless of whether the email exists in the database.
- Registration returns clean generic feedback without leaking sensitive account metadata.
- Rate limiting prevents automated credential stuffing or enumeration scans.

