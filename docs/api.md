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

---

## 6. Client Management Endpoints

All Client endpoints require authentication and membership in the active organization. Mutating operations (`POST`, `PATCH`, `DELETE`) require verified email and `admin` or `owner` role.

### List Clients
`GET /api/v1/clients?search=&status=`
- **Query Parameters:**
  - `search` (optional): Filter clients by name, contact name, contact email, or website.
  - `status` (optional): `active` (default) or `archived`.
- **Response `200 OK`:**
```json
{
  "clients": [
    {
      "id": "c1f7a022-8b4e-4f51-b84e-289c029143ef",
      "organization_id": "7b79a528-7fd8-4d5c-bd8d-7108170c9159",
      "name": "Acme Corp",
      "contact_name": "Alice Smith",
      "contact_email": "alice@acme.com",
      "website": "https://acme.com",
      "notes": "Premium enterprise tier client",
      "status": "active",
      "created_at": "2026-11-01T05:00:00Z",
      "updated_at": "2026-11-01T05:00:00Z",
      "archived_at": null
    }
  ]
}
```

### Create Client
`POST /api/v1/clients`
- Requires `admin` or `owner` role. Requires verified email.
- **Body:**
```json
{
  "name": "Acme Corp",
  "contact_name": "Alice Smith",
  "contact_email": "alice@acme.com",
  "website": "https://acme.com",
  "notes": "Notes about the client"
}
```
- **Responses:**
  - `201 Created`: Returns created client object.
  - `400 Bad Request`: Validation failure (e.g. empty name, invalid website URL).

### Get Client Details
`GET /api/v1/clients/{clientID}`
- Returns client object along with its associated assets and recent monitoring summary.
- **Response `200 OK`:**
```json
{
  "client": { ... },
  "assets": [ ... ],
  "monitors_summary": {
    "total": 3,
    "up": 2,
    "down": 1,
    "paused": 0
  }
}
```

### Update Client
`PATCH /api/v1/clients/{clientID}`
- Requires `admin` or `owner` role. Requires verified email.
- **Body:** Partial update fields (`name`, `contact_name`, `contact_email`, `website`, `notes`).
- **Response `200 OK`:** Returns updated client object.

### Archive Client
`DELETE /api/v1/clients/{clientID}`
- Implements archive-first soft deletion. Marks client status as `archived` with `archived_at = NOW()`.
- Automatically deactivates all child assets and pauses all associated monitors.
- **Response `200 OK`:** `{"status": "archived"}`.

### Restore Client
`POST /api/v1/clients/{clientID}/restore`
- Restores an archived client back to `active` status (`archived_at = NULL`).
- **Response `200 OK`:** Returns restored client object.

---

## 7. Asset Management Endpoints

Assets represent monitored websites, web apps, or APIs belonging to a Client. Mutating operations require verified email and `admin` or `owner` role.

### List Assets for Client
`GET /api/v1/clients/{clientID}/assets`
- Returns all assets belonging to the client within the active organization.

### Create Asset
`POST /api/v1/clients/{clientID}/assets`
- Server-side validates parent client ownership within active organization.
- Performs SSRF pre-validation on the asset URL. Rejects private, loopback, or cloud metadata addresses.
- Detects duplicate URLs within the active organization and returns a non-blocking `warning` in the response payload.
- **Body:**
```json
{
  "name": "Primary Production Website",
  "url": "https://example.com",
  "type": "website",
  "environment": "production",
  "description": "Public storefront"
}
```
- **Response `201 Created`:**
```json
{
  "asset": {
    "id": "e4a1b023-9c5e-4a62-c95f-390d139254fa",
    "organization_id": "7b79a528-7fd8-4d5c-bd8d-7108170c9159",
    "client_id": "c1f7a022-8b4e-4f51-b84e-289c029143ef",
    "name": "Primary Production Website",
    "url": "https://example.com",
    "type": "website",
    "environment": "production",
    "description": "Public storefront",
    "is_active": true,
    "created_at": "2026-11-01T05:00:00Z",
    "updated_at": "2026-11-01T05:00:00Z"
  },
  "warning": null
}
```

### Get Asset Details
`GET /api/v1/assets/{assetID}`
- Returns asset entity, associated monitors, and latest check results.

### Update Asset
`PATCH /api/v1/assets/{assetID}`
- Requires `admin` or `owner` role. Requires verified email.
- If `url` is changed, performs complete revalidation through the SSRF guard before persisting.
- **Body:** Partial update fields (`name`, `url`, `type`, `environment`, `description`, `is_active`).

### Archive Asset
`DELETE /api/v1/assets/{assetID}`
- Soft-archives asset (`is_active = false`, `archived_at = NOW()`).
- Halts and pauses all associated scheduled monitoring jobs.

---

## 8. Monitor Configuration & Execution Endpoints

Monitors perform automated availability, latency, SSL, and HTTP assertions against assets.

### List Monitors for Asset
`GET /api/v1/assets/{assetID}/monitors`
- Returns all configured monitors for the asset.

### List Organization Monitors
`GET /api/v1/monitors`
- Tenant-wide listing with latest check result and current status.
- Monitor statuses: `pending`, `up`, `suspected_down` (1 scheduled failure), `down` (2+ scheduled failures), `paused`, `unknown`. Contains `consecutive_failures` count.

### Create Monitor
`POST /api/v1/assets/{assetID}/monitors`
- Requires `admin` or `owner` role. Requires verified email.
- Server-side validates interval (min 60s, default 300s), timeout (3–30s, default 10s), HTTP methods (`GET` or `HEAD`), redirect limit (max 5), and assertion rules.
- Performs SSRF validation on target URL.
- Computes initial `next_check_at = NOW()`.
- **Body:**
```json
{
  "name": "Homepage HTTPS Check",
  "url_override": "",
  "method": "GET",
  "interval_seconds": 300,
  "timeout_seconds": 10,
  "expected_status_low": 200,
  "expected_status_high": 399,
  "follow_redirects": true,
  "max_redirects": 5,
  "expected_text": "Welcome to Acme",
  "json_pointer": "",
  "is_active": true
}
```

### Get Monitor Details
`GET /api/v1/monitors/{monitorID}`
- Returns monitor configuration, parent asset and client details, latest check result, and remaining SSL certificate validity.

### Update Monitor
`PATCH /api/v1/monitors/{monitorID}`
- Requires `admin` or `owner` role. Requires verified email.
- Server-side validates updated parameters. If URL override changes, executes SSRF revalidation.

### Pause Monitor
`POST /api/v1/monitors/{monitorID}/pause`
- Pauses scheduling (`is_active = false`, `status = paused`). Releases any active worker lease.

### Resume Monitor
`POST /api/v1/monitors/{monitorID}/resume`
- Resumes scheduling (`is_active = true`, `status = pending`). Sets `next_check_at = NOW()`.

### Delete Monitor
`DELETE /api/v1/monitors/{monitorID}`
- Soft-deletes monitor (`is_active = false`, `archived_at = NOW()`).

### Test Now (On-Demand Execution)
`POST /api/v1/monitors/{monitorID}/test`
- Authenticated, organization-scoped, rate-limited (max 5 requests / min / IP).
- Executes live check synchronously through the SSRF-guarded engine.
- Records result with `execution_type = manual`. Manual checks are excluded from ordinary scheduled uptime metrics.
- Returns comprehensive execution diagnostics: HTTP status code, latency, SSL validity days, DNS resolution time, and matched assertions.

### Execution History
`GET /api/v1/monitors/{monitorID}/history?limit=50&offset=0`
- Returns paginated check execution history for the monitor with TLS metadata and safe error classifications.

---

## 9. Dashboard Endpoints

### Agency Overview Stats
`GET /api/v1/dashboard/stats`
- Returns high-level metrics for the active organization:
  - `total_clients`
  - `total_assets`
  - `total_monitors`
  - `active_monitors`
  - `up_monitors`
  - `suspected_down_monitors` (1 consecutive scheduled failure)
  - `down_monitors` (2+ consecutive scheduled failures)
  - `paused_monitors`
  - `unknown_monitors`
  - `ssl_expiring_soon` (certificates expiring within 30 days)
  - `open_incidents`: Count of currently unresolved incidents
  - `total_heartbeats`, `heartbeats_healthy`, `heartbeats_missed`, `heartbeats_pending`
  - `recent_results`: Last 10 check results across all tenant monitors.

---

## 10. Heartbeat Monitoring Endpoints

### Public Heartbeat Ping
`POST /api/v1/heartbeat-pings/{token}`
- **Unauthenticated** public endpoint for cron jobs, backup scripts, and background daemons.
- Caller identity and organization are resolved strictly from the SHA-256 hash digest of `{token}`.
- Max payload 1 KB (empty or ignored); request timeout 5 seconds.
- IP rate-limited (max 120 pings / min / IP).
- Invalid, revoked, or archived tokens return `404 Not Found` to prevent timing enumeration.
- When pinged while in `missed` status, automatically resolves any associated open incident and restores status to `healthy`.

### List Heartbeat Monitors
`GET /api/v1/heartbeats?client_id={id}&status={status}&search={query}&limit=50&offset=0`
- Authenticated, organization-scoped.
- Filterable by client, status (`healthy`, `missed`, `late`, `pending`, `paused`, `revoked`), and name query.

### Create Heartbeat Monitor
`POST /api/v1/heartbeats`
- Authenticated, requires `member`, `admin`, or `owner` role.
- Parameters: `client_id` (required), `asset_id` (optional, must belong to same client), `name` (required), `description`, `expected_interval_seconds` (60 to 2,592,000), `grace_period_seconds` (optional, defaults to 20% of interval with minimum 60s).
- Generates 32-byte cryptographically secure token. Plaintext token and copyable snippets (`bash_snippet`, `powershell_snippet`) are returned **only once** upon creation.

### Get Heartbeat Monitor Details
`GET /api/v1/heartbeats/{heartbeatID}`
- Returns full monitor status, timing metadata (`last_received_at`, `next_expected_at`), client name, and asset info. Never exposes token plaintext or hash.

### Update Heartbeat Monitor
`PUT /api/v1/heartbeats/{heartbeatID}`
- Requires `member`, `admin`, or `owner` role.
- Updates name, description, expected interval, and grace period.

### Rotate Heartbeat Token
`POST /api/v1/heartbeats/{heartbeatID}/rotate-token`
- Requires `admin` or `owner` role.
- Generates a new 32-byte cryptographic token and immediately invalidates the previous token digest.
- Returns the new secret token and updated ready-to-copy snippets once.

### Revoke Heartbeat Token
`POST /api/v1/heartbeats/{heartbeatID}/revoke-token`
- Requires `admin` or `owner` role.
- Sets token hash to empty string and sets status to `revoked`. Future pings are immediately rejected.

### Pause Heartbeat Monitor
`POST /api/v1/heartbeats/{heartbeatID}/pause`
- Pauses missed-heartbeat scheduling (`is_active = false`, `status = paused`).

### Resume Heartbeat Monitor
`POST /api/v1/heartbeats/{heartbeatID}/resume`
- Resumes monitoring (`is_active = true`, `status = pending`). Sets `next_expected_at = NOW() + expected_interval`.

### Delete Heartbeat Monitor
`DELETE /api/v1/heartbeats/{heartbeatID}`
- Soft-deletes monitor (`is_active = false`, `archived_at = NOW()`).

---

## 11. Incident Management Endpoints

### List Incidents
`GET /api/v1/incidents?status={status}&target_type={monitor|heartbeat}&client_id={id}&limit=50&offset=0`
- Returns paginated incidents filtered by status (`open`, `resolved`), target type, client, or acknowledgement state.
- Includes joined target names, client details, downtime durations, and acknowledgement metadata.

### Get Incident Details
`GET /api/v1/incidents/{incidentID}`
- Returns full incident entity with diagnostic data:
  - Initial failure check result (HTTP status, error message, latency, TLS info)
  - Recovery check result (HTTP status, latency, recovery timestamp)
  - Append-only event timeline (`notification_events`) sorted chronologically.

### Acknowledge Incident
`POST /api/v1/incidents/{incidentID}/acknowledge`
- Requires `member`, `admin`, or `owner` role.
- Records acknowledging user ID and timestamp (`acknowledged_at`).
- **Important:** Acknowledging an incident confirms human review but **does not resolve** the incident. Status remains `open` until recovery criteria are satisfied.
- Generates append-only `incident.acknowledged` notification event.

### Calculate Operational Uptime
`GET /api/v1/monitors/{monitorID}/uptime?range=30d`
- Calculates true operational uptime percentage over the requested window.
- Excludes manual test checks and paused maintenance intervals.
- Accounts for partial outages from `first_failure_at` through `resolved_at`.

---

## 12. Maintenance Activity Endpoints

### List Maintenance Activities
`GET /api/v1/maintenance-activities?client_id={id}&asset_id={id}&category={cat}&from={iso}&to={iso}&limit=50&offset=0`
- Returns paginated list of maintenance work items.
- Filterable by client, asset, category (`update`, `backup`, `bug_fix`, `performance`, `security_review`, `content_change`, `configuration`, `other`), and date range.

### Create Maintenance Activity
`POST /api/v1/maintenance-activities`
- Requires `member`, `admin`, or `owner` role.
- Body:
  ```json
  {
    "client_id": "uuid",
    "asset_id": "uuid (optional)",
    "title": "Security update applied",
    "description": "Updated WordPress core and WooCommerce plugins",
    "category": "update",
    "performed_at": "2026-10-15T10:00:00Z",
    "duration_minutes": 45
  }
  ```
- Validates client existence, optional asset ownership, non-negative duration, and title length.

### Get Maintenance Activity
`GET /api/v1/maintenance-activities/{id}`
- Returns a single activity by ID with client, asset, and creator details.

### Update Maintenance Activity
`PATCH /api/v1/maintenance-activities/{id}`
- Requires `member`, `admin`, or `owner` role.
- If the activity falls within the period of an existing generated report, automatically creates an audit log entry (`maintenance_activity.updated_after_report`).
- Historical report snapshots remain immutable.

### Delete Maintenance Activity
`DELETE /api/v1/maintenance-activities/{id}`
- Requires `member`, `admin`, or `owner` role.
- Removes activity record from database.

---

## 13. Agency White-Label Branding Endpoints

### Get Branding Settings
`GET /api/v1/settings/branding`
- Returns active organization's branding configuration: `display_name`, `logo_storage_key`, `primary_color`, `accent_color`, `report_footer_text`, `contact_email`, `website_url`, and `default_report_locale`.

### Update Branding Settings
`PATCH /api/v1/settings/branding`
- Requires `admin` or `owner` role.
- Validates hex colors (`#RRGGBB`), contact email format, website URL, and report locale (`en` or `fa`).

### Upload Agency Logo
`POST /api/v1/settings/branding/logo`
- Requires `admin` or `owner` role (`multipart/form-data`).
- Validates 2MB max size and inspects file magic bytes for `image/png`, `image/jpeg`, or `image/webp`.
- Explicitly rejects SVG uploads to eliminate script injection / XSS vectors.
- Stores under random tenant-isolated keys.

### Delete Agency Logo
`DELETE /api/v1/settings/branding/logo`
- Removes stored logo and nullifies `logo_storage_key`.

---

## 14. Monthly Care Reports & Snapshot Endpoints

### Calculate Report Preview
`POST /api/v1/reports/preview`
- Body:
  ```json
  {
    "client_id": "uuid",
    "period_start": "2026-10-01T00:00:00Z (optional)",
    "period_end": "2026-10-31T23:59:59Z (optional)",
    "locale": "en | fa"
  }
  ```
- Deterministically calculates real-time report metrics without persisting to database.

### Generate & Seal Report Version
`POST /api/v1/reports`
- Requires `member`, `admin`, or `owner` role.
- Deterministically computes metrics and persists an immutable snapshot in `reports` with monotonically incrementing `version_number`.

### List Reports
`GET /api/v1/reports?client_id={id}&report_type={type}&limit=50&offset=0`
- Returns paginated list of generated reports with metadata, client details, version number, and PDF storage status.

### Get Report Details & Snapshot
`GET /api/v1/reports/{id}`
- Returns full immutable `snapshot_json` along with report version and metadata.

### Generate Report PDF
`POST /api/v1/reports/{id}/generate-pdf`
- Requires `member`, `admin`, or `owner` role.
- Spawns headless Chromium to render the sealed snapshot into a selectable-text vector PDF with embedded fonts.
- Stores PDF under `reports/<org_id>/<report_id>_v<version>.pdf`.

### Download Report PDF
`GET /api/v1/reports/{id}/pdf`
- Returns PDF binary stream (`application/pdf`) with `Content-Disposition: attachment`. Generates on demand if not yet cached.

---

## 15. Secure Report Sharing Endpoints

### Create Share Link
`POST /api/v1/reports/{id}/share`
- Requires `member`, `admin`, or `owner` role.
- Body: `{"expires_in_days": 30}`
- Generates 32 bytes of cryptographically secure random material (`cp_sh_<hex>`), stores only SHA-256 hash in `report_share_tokens`.
- Returns full public URL once.

### List Active Shares
`GET /api/v1/reports/{id}/shares`
- Returns list of unrevoked share tokens with expiry dates and creation timestamps.

### Revoke Share Link
`POST /api/v1/reports/{id}/revoke-share`
- Body: `{"share_id": "uuid"}`
- Immediately invalidates share link (`revoked_at = NOW()`).

### Public Read-Only Report View (Unauthenticated)
`GET /public/reports/{token}`
- Rate-limited per client IP (60 req/min).
- Redacts raw token in server logs (`/public/reports/[REDACTED]`).
- Looks up report snapshot via stored procedure `lookup_report_by_share_token_hash`.
- Returns generic 404 if invalid, expired, or revoked.

### Public Read-Only PDF Download (Unauthenticated)
`GET /public/reports/{token}/pdf`
- Rate-limited per client IP.
- Streams downloadable PDF corresponding to the shared snapshot.
