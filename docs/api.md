# ClientPulse API Specification & Guidelines

## 1. Conventions

- **Base URL:** `/api/v1` for business APIs; `/health/*` for infrastructure probes.
- **Protocol:** HTTP/1.1 or HTTP/2 over TLS in production.
- **Request / Response Format:** `application/json; charset=utf-8`.
- **Date/Time:** ISO 8601 UTC format (e.g. `2026-10-31T14:00:00Z`).
- **Correlation ID:** Every request receives or generates an `X-Request-ID` header, returned in responses and attached to structured log entries.
- **Authentication:** HttpOnly, SameSite=Lax, Secure (prod) session cookies (`clientpulse_session`), or `Authorization: Bearer <token>`.
- **CSRF Protection:** State-mutating requests (`POST`, `PUT`, `PATCH`, `DELETE`) authenticated via session cookie must present the `X-CSRF-Token` header matching the `clientpulse_csrf` cookie value. The token contains a cryptographic HMAC-SHA256 signature bound to the active session ID.
- **Cross-Site & Origin Defense:** Mutating requests must have a whitelisted `Origin` (or `Referer`) matching `APP_BASE_URL` or `API_BASE_URL`, and must not specify `Sec-Fetch-Site: cross-site`.
- **Tenant Context:** Tenant ID is resolved by backend middleware from verified memberships, or explicitly passed via `X-Organization-ID` header (validated on every request against memberships).

---

## 2. Standard Error Response

All API errors return a standardized JSON structure:

```json
{
  "error": {
    "code": "ERROR_CODE_CONSTANT",
    "message": "Human readable error description",
    "request_id": "c1f7a022-8b4e-4f51-b84e-289c029143ef",
    "details": {}
  }
}
```

Common error codes:
- `INVALID_REQUEST` / `VALIDATION_FAILED` (400)
- `INVALID_CREDENTIALS` (401)
- `UNAUTHORIZED` (401)
- `FORBIDDEN` / `CSRF_TOKEN_MISSING` / `CSRF_MISMATCH` / `CSRF_TOKEN_INVALID` / `CSRF_ORIGIN_INVALID` / `CSRF_CROSS_SITE_BLOCKED` (403)
- `NOT_FOUND` (404)
- `RATE_LIMIT_EXCEEDED` (429)
- `INTERNAL_ERROR` (500)
- `SERVICE_UNAVAILABLE` (503)

---

## 3. Infrastructure & Health Endpoints

### Liveness Probe
`GET /health/live`
- Returns 200 OK when the HTTP server process is running.

### Readiness Probe
`GET /health/ready`
- Checks connectivity to the PostgreSQL database pool. Returns 200 OK when healthy, 503 when disconnected.

### Worker Liveness Probe
`GET /health/worker`
- Verifies background worker heartbeats in the database. Returns 200 OK when worker heartbeat is within timeout threshold.

### Service Info
`GET /api/v1/info`
- Returns service identity, environment, and current phase.

---

## 4. Authentication Endpoints

### Register User & Provision Agency
`POST /api/v1/auth/signup`
- **Rate Limit:** 10 requests / minute / IP.
- **Body:**
```json
{
  "email": "user@agency.com",
  "password": "SecurePassword123!",
  "display_name": "Jane Doe",
  "organization_name": "Apex Agency"
}
```
- **Responses:**
  - `201 Created`: Sets `clientpulse_session` and `clientpulse_csrf` cookies. Returns user object, initial organization, role (`owner`), and CSRF token.
  - `400 Bad Request`: Validation failure or duplicate email.

### User Login
`POST /api/v1/auth/login`
- **Rate Limit:** 10 requests / minute / IP.
- **Body:**
```json
{
  "email": "user@agency.com",
  "password": "SecurePassword123!"
}
```
- **Responses:**
  - `200 OK`: Sets `clientpulse_session` and `clientpulse_csrf` cookies. Returns user, active organization, role, and CSRF token.
  - `401 Unauthorized`: Invalid credentials (anti-enumeration: generic message).

### User Logout
`POST /api/v1/auth/logout`
- Revokes active session in PostgreSQL and clears cookies (`MaxAge: -1`).
- **Response:** `200 OK`.

### Current Authenticated User & Workspace
`GET /api/v1/auth/me`
- Requires authentication.
- Returns user identity, active organization, caller role, and list of memberships.

### Verify Email
`POST /api/v1/auth/verify-email`
- **Rate Limit:** 15 requests / minute / IP.
- **Body:** `{"token": "raw-verification-token"}` (or `?token=...`).
- Atomically consumes verification token and marks user's email as verified.

### Resend Verification Email
`POST /api/v1/auth/resend-verification`
- **Rate Limit:** 5 requests / minute / IP.
- Requires authentication. Generates a fresh verification token and dispatches email via Mailpit/SMTP.

### Forgot Password
`POST /api/v1/auth/forgot-password`
- **Rate Limit:** 5 requests / minute / IP.
- **Body:** `{"email": "user@agency.com"}`.
- Generates a 30-minute single-use token and sends reset instructions. Always returns generic success to prevent email enumeration.

### Reset Password
`POST /api/v1/auth/reset-password`
- **Rate Limit:** 10 requests / minute / IP.
- **Body:** `{"token": "raw-token", "password": "NewSecurePassword123!"}`.
- Validates password strength (>= 12 chars), atomically consumes token, updates password hash using Argon2id, and invalidates all active sessions.

### Change Password
`POST /api/v1/auth/change-password`
- Requires authentication.
- **Body:** `{"current_password": "...", "new_password": "..."}`.
- Verifies current password, updates to new Argon2id hash, and revokes prior sessions.

---

## 5. Multi-Tenant Organization Endpoints

### List User Organizations
`GET /api/v1/organizations`
- Returns all organizations the user belongs to.

### Create Organization
`POST /api/v1/organizations`
- **Body:** `{"name": "New Agency", "slug": "new-agency", "locale": "en", "timezone": "UTC"}`.
- Provisions new tenant and assigns creator as `owner`.

### Select Active Organization
`POST /api/v1/organizations/select`
- **Body:** `{"organization_id": "<uuid>"}`.
- Validates user membership in target organization. Updates session's active organization.

### Get Current Organization Settings
`GET /api/v1/organizations/current`
- Tenant-scoped. Returns organization details and caller's role.

### Update Organization Settings
`PATCH /api/v1/organizations/current`
- Requires `admin` or `owner` role.
- **Body:** `{"name": "Updated Name", "locale": "fa", "timezone": "Asia/Tehran"}`.

### List Organization Members
`GET /api/v1/organizations/current/members`
- Tenant-scoped. Returns all members with their roles and user info.

### Add Member to Organization
`POST /api/v1/organizations/current/members`
- Requires `admin` or `owner` role. Admin cannot grant `owner` role.
- **Body:** `{"email": "teammate@agency.com", "role": "member"}`.
- Adds an existing registered user to the organization.
- *Note:* True pending email invitation flows with token links (PRD FR-TEN-03) will be implemented in a subsequent phase.

### Update Member Role
`PATCH /api/v1/organizations/current/members/{userID}`
- Requires `owner` role. Cannot demote the last remaining owner.
- **Body:** `{"role": "admin"}`.

### Remove Member
`DELETE /api/v1/organizations/current/members/{userID}`
- Requires `admin` or `owner` role (or self-removal). Admin cannot remove an Owner. Cannot remove the last remaining owner.
