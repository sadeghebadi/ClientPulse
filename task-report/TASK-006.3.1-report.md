# TASK-006.3.1: Notification Links, Timeline Verification & Telegram Formatting Polish Report

**Project:** ClientPulse  
**Date:** 2026-11-02  
**Scope:** Application-level corrections for centralized application links, incident timeline verification, Telegram formatting & length limits, and localization consistency.  
**Execution Status:** Completed successfully with zero failures and strict adherence to TASK-007.A read-only boundaries.

---

## 1. Executive Summary & Boundaries Confirmation

In strict compliance with the isolation rules:
- **No VPS Access**: No connection was made to the production VPS or production server.
- **No Infra/Config Modifications**: Nginx configuration, DNS, Cloudflare, Docker Compose, `deploy/`, and production `.env` files were left untouched.
- **No Audit Disruption**: The concurrent TASK-007.A infrastructure audit artifacts (`task-report/TASK-007.A-UX-AUDIT.md`) were completely preserved and unmodified.
- **Zero Hardcoded URLs**: All notification channels (Telegram, Email) now rely strictly on a centralized, validated `PUBLIC_APP_URL` configuration helper.
- **Timeline Mystery Resolved**: Event-to-delivery correlation verified that the supposed timestamp mismatch in TASK-006.3 was an artifact of comparing the delivery record of an *opened* event with the resolution timestamp of a subsequent *resolved* event.

---

## 2. Files Changed

| Component / Path | Nature of Change |
|---|---|
| `services/api/internal/appurl/builder.go` | **New File**: Central canonical `LinkBuilder` and `ValidatePublicAppURL` validator. |
| `services/api/internal/appurl/builder_test.go` | **New File**: 20 unit test cases verifying URL validation, security controls, and route generation. |
| `services/api/internal/config/config.go` | Added `PublicAppURL` configuration field and production/development validation in `Validate()`. |
| `services/api/internal/config/config_test.go` | Added unit tests verifying `PUBLIC_APP_URL` parsing and validation. |
| `services/api/internal/notification/telegram/formatter.go` | **New File**: Telegram UTF-16 code-unit counter, tag balancer `EnsureValidTelegramHTML`, rune truncation, URL sanitizer, and plain-text stripper. |
| `services/api/internal/notification/telegram/templates.go` | Refactored formatting to use progressive multi-stage degradation (`formatFull`, `formatCompact`, `formatMinimal`), HTML tag balancing, and plain-text fallback. |
| `services/api/internal/notification/telegram/templates_test.go` | Comprehensive test suite for long Persian, long English, emoji/non-BMP Unicode, extreme lengths, malformed UTF-8, tag balancing, and plain-text fallback. |
| `services/api/internal/notification/email/templates.go` | Refactored email templates to use `appurl.LinkBuilder`, escape attributes safely, and omit action buttons/links when URL is unconfigured. |
| `services/api/internal/notification/email/email_test.go` | Added tests verifying action button omission, attribute escaping, and zero hardcoded links. |
| `services/api/internal/notification/context_resolver.go` | Wired `LinkBuilder`, resolved org timezone (`Asia/Tehran`) & locale fallback, and resolved tenant RLS isolation on incident queries. |
| `services/api/internal/notification/context_test.go` | Added tests for `LinkBuilder` integration and timezone-aware timestamp rendering. |
| `services/api/internal/repository/task_006_notifications_test.go` | Added `TestIncidentTimeline_EventDeliveryCorrelationAndConsistency` regression test. |
| `services/api/internal/server/notification_handlers.go` | Switched verification and test notification generation to `PublicAppURL`. |
| `services/api/internal/service/auth_service.go` | Switched email verification and password reset link builders to `PublicAppURL` with proper query escaping. |
| `services/api/cmd/worker/main.go` | Passed `cfg.PublicAppURL` to `worker.NewNotificationWorker`. |
| `services/api/internal/worker/notification_worker.go` | Added automatic plain-text retry fallback when Telegram rejects HTML formatting. |

---

## 3. Central Link-Generation Implementation

### Canonical LinkBuilder (`internal/appurl`)

A reusable, concurrency-safe helper was introduced:
```go
package appurl

type LinkBuilder struct {
    baseURL string
}

func NewLinkBuilder(publicAppURL string) *LinkBuilder
func (b *LinkBuilder) IncidentURL(incidentID string) string
func (b *LinkBuilder) HeartbeatURL(heartbeatID string) string
func (b *LinkBuilder) ReportURL(reportID string) string
func (b *LinkBuilder) DashboardURL() string
func (b *LinkBuilder) EmailVerifyURL(token string) string
```

### Security & Sanitization Rules
1. **Absolute URL Check**: Requires scheme (`http` or `https`) and valid host.
2. **Credential Rejection**: Reject URLs containing username or password (`user:pass@host`).
3. **Query & Fragment Rejection**: Reject URLs with query parameters (`?query=...`) or anchors (`#hash`).
4. **Environment Constraints**:
   - **Production (`APP_ENV=production`)**: Strict HTTPS required; non-routable localhost/127.0.0.1/private IPs and bare hostnames without dots are rejected.
   - **Development**: Permits `http://localhost:3001`, `http://127.0.0.1:3000`, or local development ports.
5. **Path Joining & Base Path Preservation**: Correctly joins sub-paths (e.g. `https://myhost.org/pulse/` -> `https://myhost.org/pulse/incidents/123`) using clean path resolution without double slashes or path traversal.
6. **Graceful Degradation**: When `PUBLIC_APP_URL` is empty, **all** route methods return an empty string (`""`). No fallback to `localhost:3000` or fictional domains like `app.clientpulse.io` is ever performed.

---

## 4. Verified Frontend Routes

All Next.js application routes in `apps/web/src/app` were verified before wiring:

| Notification Target | Verified Frontend Next.js Route File | Generated Canonical Path | Fallback if ID Empty |
|---|---|---|---|
| Incident Detail | `apps/web/src/app/incidents/[id]/page.tsx` | `/incidents/{id}` | `/incidents` |
| Heartbeat Detail | `apps/web/src/app/heartbeats/[id]/page.tsx` | `/heartbeats/{id}` | `/heartbeats` |
| Care Report Detail | `apps/web/src/app/reports/[id]/page.tsx` | `/reports/{id}` | `/reports` |
| Dashboard | `apps/web/src/app/page.tsx` | `/` | `/` |
| Email Verification | `apps/web/src/app/settings/notifications/page.tsx` | `/settings/notifications?verify_email={token}` | `/settings/notifications` |

Both Telegram and Email notification templates use this exact same link generation logic.

---

## 5. Required `PUBLIC_APP_URL` Configuration

The configuration variable is:
```env
PUBLIC_APP_URL=http://localhost:3001
```

### Confirmation on Production Hostname
- **No production hostname has been hardcoded.**
- Neither `mindu.ir`, `app.mindu.ir`, nor any other domain was assumed as permanent.
- Codebase-wide grep confirms zero active occurrences of `app.clientpulse.io` or `localhost:3000` in notification templates.
- When `PUBLIC_APP_URL` is unset, templates omit the action buttons/links gracefully without rendering broken hyperlinks.

---

## 6. Incident Timeline and Delivery Evidence

### Investigation of TASK-006.3 Discrepancy

In the TASK-006.3 report, an apparent discrepancy was noted:
- Incident recovery time: `16:44:40 +03:30` (13:14:40 UTC)
- Delivery record timestamp: `13:10:24 UTC` (16:40:24 +03:30)

Direct inspection of the PostgreSQL event and delivery tables for the real test incident (`94277caa-f0f7-4a40-bbae-44884622b88f`) revealed the ground truth:

### Evidence Table (Sanitized)

| Stage | Record Type | Identifier / Column | Timestamp (UTC) | Timestamp (Asia/Tehran +03:30) | Notes |
|---|---|---|---|---|---|
| **Opened** | Incident Record | Incident ID: `94277caa-f0f7-...` | `2026-11-02 13:02:40.000000Z` | `16:32:40` | Incident triggered after consecutive failed checks |
| **Opened** | Event Record | Event ID: `460e815f-fa31-...` | `2026-11-02 13:10:21.897451Z` | `16:40:21` | `incident.opened` event generated |
| **Opened** | Delivery Record | Delivery ID: `cde0fcb7-6a1f-...` | `2026-11-02 13:10:22.487176Z` | `16:40:22` | Delivery row inserted (`status: queued`) |
| **Opened** | Delivery Sent | `sent_at` column | `2026-11-02 13:10:24.454102Z` | `16:40:24` | **Provider accepted telegram message** |
| **Resolved** | Incident Record | `first_failure_at` | `2026-11-02 13:02:40.000000Z` | `16:32:40` | Incident start |
| **Resolved** | Incident Record | `resolved_at` | `2026-11-02 13:14:40.000000Z` | `16:44:40` | Monitor recovered; incident resolved |
| **Resolved** | Event Record | Event ID: `9e72bde7-f752-...` | `2026-11-02 13:16:40.601835Z` | `16:46:40` | `incident.resolved` event generated |
| **Resolved** | Delivery Record | Delivery ID: `3dc1e52c-85a2-...` | `2026-11-02 13:16:41.487103Z` | `16:46:41` | Delivery row inserted (`status: queued`) |
| **Resolved** | Delivery Sent | `sent_at` column | `2026-11-02 13:16:43.271976Z` | `16:46:43` | **Provider accepted telegram message** |

### Findings & Correlation
1. **No Inversion / No Bug**: The `13:10:24 UTC` (`16:40:24 +03:30`) timestamp belonged to the **incident.opened** event delivery (`cde0fcb7`). The recovery occurred at `13:14:40 UTC` (`16:44:40 +03:30`), and its resolution notification was delivered at `13:16:43 UTC` (`16:46:43 +03:30`) under a distinct delivery row (`3dc1e52c`).
2. **Total Detected Downtime**: Calculated as `13:14:40 - 13:02:40 = 12m 0s`, accurately rendered in Persian as `۱۲ دقیقه ۰ ثانیه` and English as `12m 0s`.
3. **Delayed Delivery Safeguard**:
   If an `incident.opened` delivery is delayed (e.g., worker pause, network retry) until *after* the incident has already been resolved in PostgreSQL, `notification.ResolveAndRefreshContext` queries the live incident status. It detects `resolved_at IS NOT NULL` and flags `IsAlreadyResolved = true`.
   The rendered message explicitly informs the user that the incident is already resolved, preventing confusion.
4. **RLS Permission Bug Fixed**:
   During context resolution, querying `incidents` originally attempted a `LEFT JOIN users` which failed under the worker's database role (`clientpulse_worker` does not have read access to `users`). This was refactored to query `incidents` with `set_config('app.current_organization_id', orgID, true)` in a tenant transaction without joining `users`.

---

## 7. Telegram Message Length Handling

### Telegram Bot API Specification Compliance
Per [Telegram Bot API Documentation](https://core.telegram.org/bots/api#sendmessage), `sendMessage` limit is **1–4096 characters** (measured in UTF-16 code units / runes after parsing entities), not 4096 bytes.

### Multi-Stage Degradation & Truncation Engine
1. **UTF-16 Measurement (`TelegramMessageLength`)**:
   Correctly measures message length, counting surrogate pairs for non-BMP runes (e.g. emoji).
2. **Tag Balancing (`EnsureValidTelegramHTML`)**:
   Tracks open tags using a tag stack (`<b>`, `<i>`, `<code>`, `<pre>`, `<a>`). When truncating near the 4000-character safety margin, it cuts at a rune boundary outside of tags and automatically appends closing tags in reverse order (`</a></code></b>`). Raw HTML tags are never left open or malformed.
3. **Progressive Degradation Pipeline**:
   - **Stage 1 (Full)**: Renders all context fields, full URLs, error snippets, and links.
   - **Stage 2 (Compact)**: If > 4000 UTF-16 units, omits verbose failure reason snippets, shortens target URLs to hostnames, and truncates long identifiers.
   - **Stage 3 (Minimal)**: If still > 4000 UTF-16 units, preserves only mandatory fields: Event Type, Client, Asset, Monitor/Heartbeat, Timestamp, Duration, and Incident Reference.
   - **Fallback (Plain Text)**: `RenderTelegramMessagePlainText` strips all HTML tags and renders a clean plain-text version. Furthermore, if Telegram's API returns `400 Bad Request: can't parse entities`, `NotificationWorker` automatically falls back to plain-text mode without failing the delivery.

---

## 8. Localization and Timezone Consistency

- **Locale Precedence**:
  1. Destination locale override (`dest.Locale`).
  2. Organization locale fallback (`org.Locale`, default `"fa"`).
- **Timezone Precedence**:
  - Organization timezone (`org.Timezone`, e.g. `Asia/Tehran`).
  - Timestamps are formatted with their localized timezone offset (e.g. `+03:30` / `IRST`).
  - Persian digit conversion (`toPersianDigits`) applied across Persian notifications (`۱۴۰۵/۰۸/۱۱ ۱۶:۴۰:۲۴`).
  - English templates format timestamps in standard LTR (`2026-11-02 16:40:24 +03:30`).

---

## 9. Comprehensive Test Suite Results

All tests executed with zero cache (`-count=1`):

```bash
cd services/api && go test -p 1 -count=1 ./...
```

### Detailed Test Breakdown

| Test Package | Subtests / Test Cases | Failures | Status |
|---|---|---|---|
| `internal/appurl` | 20 subtests (valid/invalid URLs, production HTTPS, credentials, query/fragments, trailing slashes, base paths) | 0 | **PASS** |
| `internal/config` | 6 subtests (PublicAppURL validation in production and development) | 0 | **PASS** |
| `internal/notification` | 12 subtests (context resolver, link builder integration, timezone formatting, duration calculation, digit conversion) | 0 | **PASS** |
| `internal/notification/email` | 7 test suites (HTML attribute escaping, link omission, Mailpit integration, verification links) | 0 | **PASS** |
| `internal/notification/telegram` | 21 subtests (all event types in EN/FA, HTML escaping, 5000+ char Persian/English, emoji/non-BMP Unicode, extreme field lengths, malformed UTF-8, tag balancing, plain-text fallback) | 0 | **PASS** |
| `internal/repository` | Timeline & delivery correlation test (`TestIncidentTimeline_EventDeliveryCorrelationAndConsistency`) + full repository suite | 0 | **PASS** |
| `internal/server` | Notification handlers, token binding, email verification, test notifications | 0 | **PASS** |
| `internal/worker` | Notification worker, retry handling, plain-text fallback | 0 | **PASS** |
| **Entire API Suite** | **441 Total Test Passes across 18 packages** | **0** | **ALL PASS** |

---

## 10. Required Environment Wiring for TASK-007.B

When deploying or restarting Docker Compose in TASK-007.B, wire the following variable into `.env` and `docker-compose.yml`:

```yaml
services:
  api:
    environment:
      - PUBLIC_APP_URL=${PUBLIC_APP_URL:-http://localhost:3001}
  worker:
    environment:
      - PUBLIC_APP_URL=${PUBLIC_APP_URL:-http://localhost:3001}
```

- In production, set `PUBLIC_APP_URL=https://<final-domain>` once the domain decision from TASK-007.A/B is confirmed.
- No changes to `docker-compose.yml` or production `.env` were made in this task.
