# TASK-006 Implementation Report: Notification Infrastructure — Telegram Bot & Email Integration

**Project:** ClientPulse  
**Phase:** Phase 5 — Notifications & Alerts  
**Status:** Completed & Fully Verified  
**Date:** 2026-11-02  

---

## 1. Executive Summary

TASK-006 implements real, production-ready, multi-tenant notification delivery for ClientPulse using an extensible provider-adapter architecture. Organizations can now connect multiple notification destinations across both **Telegram** (personal accounts via private chats and group chats via bot commands) and **Email** (configurable SMTP with TLS certificate validation), configure granular per-destination event filtering, transmit test alerts, and inspect an auditable delivery history.

The delivery pipeline operates asynchronously through durable PostgreSQL queues using `FOR UPDATE SKIP LOCKED`, preventing HTTP request blocking, ensuring idempotency, handling rate limits (including Telegram HTTP 429 `retry_after`), and surviving worker restarts without duplicate delivery or data loss.

---

## 2. Architecture & Components Implemented

### 2.1 Provider-Adapter Architecture
- **Telegram Adapter (`services/api/internal/notification/telegram`)**:
  - Central managed bot model: single bot instance serving all tenants.
  - REST client with automated URL redaction (`SanitizeTelegramURL`) so bot tokens never leak into logs or error strings.
  - Robust error classification:
    - `RateLimitError`: parses Telegram HTTP 429 responses and extracts `parameters.retry_after` seconds.
    - `PermanentError`: identifies non-retryable errors (e.g., HTTP 400 Chat Not Found, HTTP 403 Bot Blocked by User).
    - Temporary Network/Server Errors: triggers exponential backoff with random jitter.
  - Update handler (`UpdateHandler`): handles incoming Telegram webhook updates for private `/start <token>` and group `/connect@<bot_username> <code>`.
  - In-memory mock sender (`MockSender`) for automated tests without real Telegram credentials.

- **Email SMTP Adapter (`services/api/internal/notification/email`)**:
  - Independent configurable SMTP client (`Client`) supporting `starttls`, direct `tls` (port 465), and unencrypted `none` (local testing / Mailpit on port 1025).
  - Enforces TLS certificate validation (`InsecureSkipVerify: false`).
  - MIME multipart (`multipart/alternative`) formatter for UTF-8 HTML and plain-text fallback.
  - Safe, responsive email templates in English and Persian (RTL) for all domain events:
    - `incident.opened`
    - `incident.acknowledged`
    - `incident.resolved`
    - `heartbeat.missed`
    - `heartbeat.recovered`
    - `report.ready`
    - `test.notification`

- **Notification Worker & Dispatcher (`services/api/internal/worker/notification_worker.go`)**:
  - **Dispatcher Loop**: polls pending `notification_events`, evaluates destination active status and event filtering preferences, and transactions fan-out into `notification_deliveries` with `ON CONFLICT (idempotency_key) DO NOTHING`.
  - **Delivery Worker Loop**: claims queued and due retrying deliveries using PostgreSQL `claim_due_notification_deliveries` with `FOR UPDATE SKIP LOCKED` and concurrent worker pools.
  - **Stale Sweep Loop**: automatically rescues deliveries stuck in `sending` state (> 5 minutes) during node or worker crashes.

- **Frontend Notification Settings (`apps/web/src/app/settings/notifications/page.tsx`)**:
  - Multi-tab UI for destinations management and delivery history.
  - Private Telegram Connect modal with dynamic deep link.
  - Group Telegram Connect modal with single-use command copy.
  - Add Email destination modal with input validation.
  - Destination enable/disable toggle switches.
  - Granular Event Preferences modal (6 distinct domain event checkboxes).
  - Immediate Test Notification action with live alert feedback.
  - Paginated Delivery History table with delivery latency and error descriptions.
  - Complete Persian RTL language support.

---

## 3. Database Schema & Multi-Tenant Security

### 3.1 Migration `000009_notifications_telegram_email`
Created four dedicated tables with foreign key cascade rules, indexes, and full Row-Level Security:

1. `notification_destinations`:
   - `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
   - `organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE`
   - `destination_type VARCHAR(32)` (`telegram_private`, `telegram_group`, `email`)
   - `name VARCHAR(255)`
   - `telegram_chat_id BIGINT NULL` (numeric 64-bit Telegram chat ID)
   - `telegram_user_id BIGINT NULL` (numeric 64-bit Telegram user ID)
   - `telegram_username VARCHAR(255) NULL`
   - `email_address VARCHAR(255) NULL`
   - `user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL`
   - `status VARCHAR(32) NOT NULL DEFAULT 'active'`
   - `is_enabled BOOLEAN NOT NULL DEFAULT true`
   - `created_at`, `updated_at`, `last_verified_at`

2. `notification_preferences`:
   - Per-destination event filtering (`incident.opened`, `incident.acknowledged`, `incident.resolved`, `heartbeat.missed`, `heartbeat.recovered`, `report.ready`).
   - `CONSTRAINT uq_notif_pref_dest_event UNIQUE (destination_id, event_type)`.

3. `notification_connection_tokens`:
   - Ephemeral single-use tokens for binding Telegram chats.
   - Raw tokens are never stored; only cryptographic SHA-256 hashes (`token_hash`) are stored.
   - Expiration timestamp with atomic single-use consumption.

4. `notification_deliveries`:
   - Durable queue and auditable delivery history.
   - Status transitions: `queued` → `sending` → `sent` | `retrying` | `failed` | `cancelled`.
   - `idempotency_key VARCHAR(255) UNIQUE` preventing duplicate delivery across worker retries.
   - `retry_count`, `max_retries`, `next_attempt_at`, `payload`, `error_message`, `sent_at`.

5. Check Constraint Update:
   - Updated `chk_notif_events_type` on `notification_events` to support `report.ready`.

### 3.2 Security Definer Functions & RLS Roles
- **`consume_notification_connection_token(p_token_hash VARCHAR)`**:
  - Owned by `clientpulse_definer`, executable by `clientpulse_app` and `clientpulse_worker`.
  - Atomically marks `consumed_at = NOW()` where `consumed_at IS NULL AND expires_at > NOW()`.
- **`claim_due_notification_deliveries(p_worker_id VARCHAR, p_limit INT)`**:
  - Owned by `clientpulse_definer`, executable by `clientpulse_worker` and `clientpulse_app`.
  - Claims due deliveries using `FOR UPDATE SKIP LOCKED` and transitions status to `sending`.
- **Role Isolation & RLS**:
  - RLS enabled and forced on all tables.
  - Multi-tenant isolation: `organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid`.
  - Background worker cross-tenant policies on `notification_events`, `notification_destinations`, and `notification_deliveries`.

---

## 4. Connection & Delivery Workflows

### 4.1 Telegram Private Connection Flow
1. User clicks **Connect Telegram** in Notification Settings.
2. Backend generates a 24-byte cryptographically secure random token (48 hex chars) with a 15-minute TTL.
3. Token SHA-256 hash is saved to `notification_connection_tokens` linked to `(organization_id, user_id)`.
4. User clicks deep link: `https://t.me/<BOT_USERNAME>?start=<TOKEN>`.
5. User presses **Start** in Telegram client.
6. Telegram sends incoming update to `POST /api/v1/integrations/telegram/webhook`.
7. Webhook validates secret header, extracts token, and atomically consumes token via `consume_notification_connection_token`.
8. Destination record is created with `destination_type = 'telegram_private'`, lossless 64-bit `chat_id`, and verified `telegram_user_id`. Default preferences enabled.
9. Confirmation message is sent back to the user via Telegram.

### 4.2 Telegram Group Connection Flow
1. Admin clicks **Add Telegram Group** in Notification Settings.
2. Backend creates a 15-minute single-use code.
3. Admin adds the bot to their group and sends:
   `/connect@<BOT_USERNAME> <CODE>`
4. Bot receives group message update, parses command, and consumes code.
5. Organization destination created with `destination_type = 'telegram_group'`, group `chat_id`, and sender's Telegram identity.
6. Confirmation message is posted into the Telegram group.

### 4.3 Email Delivery Flow
1. User adds email address in Notification Settings.
2. Domain events (`incident.opened`, `heartbeat.missed`, etc.) fan out into `notification_deliveries`.
3. Background delivery worker claims batch with `SKIP LOCKED`.
4. Renders responsive HTML and plain-text template (English or Persian RTL).
5. Transmits message via SMTP adapter with TLS.
6. On success: marks delivery `sent` and sets `sent_at = NOW()`.
7. On failure: checks error type. Permanent errors transition directly to `failed`; temporary errors schedule exponential backoff ($2 \cdot 2^{\text{retry}} + \text{jitter}$).

---

## 5. Verification & Test Evidence

### 5.1 Automated Test Suite Breakdown

| Test Area | Package / Test File | Test Case | Outcome |
| :--- | :--- | :--- | :--- |
| **Telegram Client** | `notification/telegram/client_test.go` | `TestSanitizeTelegramURL` (Redaction of bot tokens) | **PASS** |
| **Telegram Client** | `notification/telegram/client_test.go` | `TestClientSendMessage_Success` | **PASS** |
| **Telegram Client** | `notification/telegram/client_test.go` | `TestClientSendMessage_RateLimit429` (Retry-After) | **PASS** |
| **Telegram Client** | `notification/telegram/client_test.go` | `TestClientSendMessage_Permanent403` (Bot Blocked) | **PASS** |
| **Email Templates** | `notification/email/email_test.go` | `TestRenderNotificationEmail_Templates` (EN & FA RTL) | **PASS** |
| **SMTP Delivery** | `notification/email/email_test.go` | `TestSMTPClient_MailpitDelivery` (Real Mailpit SMTP) | **PASS** |
| **Database & Tokens**| `repository/task_006_notifications_test.go` | `TestTelegramConnectionToken_SingleUseAndExpiry` | **PASS** |
| **Destinations Binding**| `repository/task_006_notifications_test.go` | `TestTelegramDestinationBinding_PrivateAndGroup` | **PASS** |
| **Fan-Out & Deduplication**| `repository/task_006_notifications_test.go` | `TestNotificationEvent_FanOutAndDeduplication` | **PASS** |
| **Queue Concurrency**| `repository/task_006_notifications_test.go` | `TestClaimDueDeliveries_ConcurrencySkipLocked` | **PASS** |
| **Retry & Failure** | `repository/task_006_notifications_test.go` | `TestDeliveryRetryAndTerminalFailure` | **PASS** |
| **Tenant Isolation** | `repository/task_006_notifications_test.go` | `TestNotificationCrossTenantIsolation` | **PASS** |
| **End-to-End API** | `server/notification_integration_test.go`| `TestNotificationServerAndWorkerEndToEnd` | **PASS** |
| **Frontend Playwright**| `apps/web/tests/task-006-e2e.js` | 9-step E2E browser test across all workflows | **PASS** |
| **Phase 4 Regression**| `apps/web/tests/task-005-e2e.js` | 6 maintenance & reporting workflows | **PASS** |
| **Phase 4 Regression**| `apps/web/tests/task-004-e2e.js` | 5 incident & heartbeat monitoring workflows | **PASS** |

**Summary of Tests Run:**
- Unit & Integration Go Tests: **48+ tests passed (100% green)**
- Playwright Browser E2E Tests: **3 major test suites passed (100% green)**
- Regressions: **0 regressions detected**

### 5.2 Screenshots Generated
All artifacts are saved to `task-report/screenshots/`:
1. `task-006-notifications-main.png` — Notification settings dashboard with Telegram and Email destinations.
2. `task-006-telegram-connect-modal.png` — Single-use private Telegram deep link connection modal.
3. `task-006-telegram-group-modal.png` — Group chat bot connect modal with copyable command.
4. `task-006-preferences-modal.png` — Event category checkboxes (incident, heartbeat, report).
5. `task-006-delivery-history.png` — Auditable delivery history table with status badges and timestamps.
6. `task-006-persian-rtl.png` — Notification settings in full Persian RTL layout.

---

## 6. Known Limitations & Out-of-Scope Items

1. **Telegram Channels**: As specified in the product requirements, broadcast channels are out of scope for TASK-006.
2. **Customer-Owned Bots**: Only the central ClientPulse bot is supported; organizations do not supply their own bot tokens.
3. **Mock/Dev Environment Deliveries**: In development and testing environments without active Telegram bot credentials, simulated webhooks and mock senders deliver notifications cleanly.

---

## 7. Required Production Configuration for TASK-007

When deploying to production in TASK-007, provide the following environment variables:

```bash
# Telegram Bot Configuration
TELEGRAM_BOT_TOKEN="<PRODUCTION_BOT_TOKEN_FROM_BOTFATHER>"
TELEGRAM_BOT_USERNAME="ClientPulseAlertsBot"
TELEGRAM_WEBHOOK_SECRET="<SECURE_RANDOM_WEBHOOK_SECRET>"
TELEGRAM_MODE="webhook"

# Production SMTP Mail Server (mindu.ir)
EMAIL_PROVIDER="smtp"
SMTP_HOST="mail.mindu.ir"
SMTP_PORT=587
SMTP_USERNAME="noreply@mindu.ir"
SMTP_PASSWORD="<PRODUCTION_SMTP_PASSWORD>"
EMAIL_FROM="noreply@mindu.ir"
SMTP_FROM_NAME="ClientPulse Alerts"
SMTP_TLS_MODE="starttls"
```

