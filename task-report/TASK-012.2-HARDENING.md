# TASK-012.2: Notification Engine Hardening & SMTP Unification Completion Report

**Project:** ClientPulse  
**Priority:** Critical  
**Execution Mode:** Local Development Only  
**Scope:** Backend Code (`services/api`) + Unit/Integration Tests  
**Status:** Completed & Verified  

---

## 1. Files Inspected and Modified

### Inspected Files
- `services/api/internal/notification/email/smtp.go`
- `services/api/internal/mail/mailer.go`
- `services/api/internal/worker/notification_worker.go`
- `services/api/internal/config/config.go`
- `services/api/internal/service/auth_service.go`
- `services/api/internal/server/auth_handlers.go`
- `services/api/internal/notification/telegram/client.go`
- `services/api/internal/repository/notification_dest_repo.go`

### Modified & Added Files
1. `services/api/internal/notification/email/smtp.go`:
   - Enforced bounded execution across all SMTP operations.
   - Added context cancellation and active connection deadlines (`applyConnDeadlines`, `net.Conn.SetDeadline`, `context.AfterFunc`).
   - Unified SMTP transaction workflow across `sendPlain`, `sendSTARTTLS`, and `sendDirectTLS` via `executeSMTPTransaction`.
   - Added `Timeout time.Duration` field to `SMTPConfig` (defaulting to 15s).

2. `services/api/internal/notification/email/smtp_hardening_test.go` *(New File)*:
   - Added local mock SMTP unit tests covering:
     - TCP connection timeout.
     - SMTP server accepting TCP handshake but stalling on greeting banner.
     - SMTP server stalling during message body transmission (`DATA`).
     - Context cancellation during in-flight network transmission.
     - STARTTLS negotiation failure (server rejection and unsupported command).
     - SMTP port auto-detection (465 -> tls, 1025 -> none, 587/25 -> starttls).

3. `services/api/internal/mail/mailer.go`:
   - Removed duplicate, unmaintained `smtp.SendMail` logic.
   - Converted `mail.SMTPMailer` into an adapter that delegates directly to `email.EmailSender` (`internal/notification/email.Client`).
   - Retained the `mail.EmailSender` interface and `mail.MockMailer` for dependency injection and tests.
   - Added `NewSMTPMailerWithSender(client email.EmailSender)` for clean DI.

4. `services/api/internal/mail/mailer_test.go` *(New File)*:
   - Unit tests verifying adapter delegation, error propagation, verification email templating, password reset email templating, nil client safety, and config constructor.

5. `services/api/cmd/api/main.go`:
   - Updated dependency wiring to instantiate a single unified `email.EmailSender` and pass it to both `AuthService` (via `mail.NewSMTPMailerWithSender`) and `NotificationHandler`.

6. `services/api/internal/worker/notification_worker.go`:
   - Enforced a 15-second bounded per-delivery timeout (`WithDeliveryTimeout`, defaulting to 15s) in `Deliver`.
   - Propagated delivery context to both Telegram (`telegramSender.SendMessage`) and SMTP (`emailSender.SendEmail`) transports.
   - Decoupled delivery transport context from database outcome recording, ensuring delivery timeouts successfully update status in PostgreSQL.
   - Upgraded `isPermanentEmailError` to parse typed `*textproto.Error` codes, recognize permanent SMTP failure codes (`535`, `501`, `502`, `503`, `504`, `550`, `554`), protect transient codes (`421`, `450`, `451`, `452`), and classify network timeouts/connection drops as retriable.

7. `services/api/internal/worker/notification_worker_hardening_test.go` *(New File)*:
   - Added table-driven tests for permanent, transient, timeout, and unknown email errors.
   - Added delivery worker timeout tests ensuring blocked email or Telegram providers cannot hold worker goroutines past the 15-second delivery deadline.

8. `services/api/internal/config/config.go`:
   - Added validation for `TELEGRAM_PROXY_URL` in `Config.Validate()`.
   - Restricted allowed schemes strictly to `http`, `https`, and `socks5`.
   - Explicitly documented and rejected `socks5h` because standard Go `net/http` does not support remote DNS resolution over SOCKS5 without custom dialers.
   - Added `SanitizeProxyURL` to redact credentials from log and error messages.

9. `services/api/internal/config/config_test.go`:
   - Added test table covering valid (`http`, `https`, `socks5`, with/without user auth) and invalid (`socks5h`, `ftp`, missing host, missing scheme, malformed) proxy configurations.
   - Added unit test verifying credential redaction in `SanitizeProxyURL`.

10. `services/api/internal/server/security_test.go`:
    - Added `TestForgotPasswordAntiEnumerationAndSMTPFailure` verifying that password reset endpoints maintain identical HTTP 200 responses and anti-enumeration behavior even when the underlying SMTP relay fails.

---

## 2. Previous vs. New SMTP Architecture

```
Previous Architecture (Dual & Divergent):
==========================================

   [AuthService]  ------------------>  [mail.SMTPMailer]
                                             │
                                             ▼
                                     net/smtp.SendMail (Raw TCP)
                                     - No context support
                                     - No implicit TLS (port 465 fails)
                                     - Unbounded execution time
                                     - Independent multipart formatting

   [NotificationWorker] ------------>  [email.Client]
                                             │
                                             ▼
                                     net.DialTimeout (Partial)
                                     - Ignored ctx parameter
                                     - Post-dial read/write unbounded
                                     - Duplicate connection logic

New Hardened & Unified Architecture:
====================================

   [AuthService]                        [NotificationWorker]
         │                                       │
         ▼                                       │
  [mail.SMTPMailer] (Adapter)                    │
         │                                       │
         └───────────────┬───────────────────────┘
                         │
                         ▼
             [email.Client] (Unified Engine)
             - Respects caller ctx
             - Enforces 15s bounded deadline
             - Supports STARTTLS, Direct TLS (465), Plain (1025)
             - Full context cancellation via applyConnDeadlines
                         │
                         ▼
             [SMTP Relay / Mailpit / SES]
```

---

## 3. Context Cancellation and Connection Deadlines

The hardened `email.Client` now enforces strict time bounds at every stage of the network lifecycle:

1. **Context Deadline Binding:**
   - If the caller supplies a context with an active deadline, the shortest deadline between caller context and `cfg.Timeout` (default 15s) is enforced.
   - If no deadline is set, `context.WithTimeout(ctx, 15*time.Second)` is applied.
2. **TCP Dial Phase:**
   - `dialer.DialContext(ctx, "tcp", addr)` guarantees TCP handshake does not block past the deadline or after context cancellation.
3. **Active Connection Deadlines (`applyConnDeadlines`):**
   - Directly sets `conn.SetDeadline(dl)` on the underlying `net.Conn`.
   - Uses `context.AfterFunc(ctx, func() { conn.SetDeadline(time.Now()) })` to instantly abort in-flight I/O if the context is cancelled early.
4. **TLS Handshake Safety:**
   - In Direct TLS (`sendDirectTLS`), `tlsConn.HandshakeContext(ctx)` is called with explicit context propagation.
   - In STARTTLS (`sendSTARTTLS`), deadlines set on the underlying TCP socket remain active during TLS upgrade, preventing hangs during TLS negotiation.
5. **SMTP Commands & Transmission (`executeSMTPTransaction`):**
   - Reading server banners (220), `EHLO`, `AUTH`, `MAIL FROM`, `RCPT TO`, `DATA`, streaming body bytes, and waiting for the final dot acknowledgment `w.Close()` are all protected by the connection deadline.

---

## 4. SMTP Error Classification Table

| Category | SMTP Code / Error Pattern | Handled As | Reason / Behavior |
| :--- | :--- | :--- | :--- |
| **Permanent** | `535` | Permanent (`true`) | Authentication credentials invalid. Fail permanently to prevent lockouts. |
| **Permanent** | `500`, `501`, `502`, `503`, `504` | Permanent (`true`) | Syntax errors, unrecognized or unimplemented commands. |
| **Permanent** | `550`, `551`, `552`, `553`, `554`, `555`, `556` | Permanent (`true`) | Mailbox unavailable, user unknown, mailbox name disallowed, transaction rejected. |
| **Permanent** | Phrases: `"mailbox unavailable"`, `"user unknown"`, `"no such user"`, `"invalid recipient"`, `"credentials rejected"` | Permanent (`true`) | Semantic match for non-retryable recipient or configuration rejections. |
| **Transient** | `421` | Transient (`false`) | Service not available / transmission channel closing. Requeue with backoff. |
| **Transient** | `450` | Transient (`false`) | Mailbox busy or temporarily blocked. Requeue with backoff. |
| **Transient** | `451` | Transient (`false`) | Local error in processing. Requeue with backoff. |
| **Transient** | `452` | Transient (`false`) | Insufficient system storage. Requeue with backoff. |
| **Transient** | `454` | Transient (`false`) | TLS temporarily not available. Requeue with backoff. |
| **Transient** | `context.DeadlineExceeded`, `context.Canceled` | Transient (`false`) | Execution timed out or interrupted. Retry delivery. |
| **Transient** | `net.Error` (Timeout=true) | Transient (`false`) | Network timeout. Requeue with exponential backoff. |
| **Transient** | Connection refused, reset by peer, broken pipe, i/o timeout | Transient (`false`) | Network transport glitch. Requeue with backoff. |
| **Unknown** | Any unspecified error | Transient (`false`) | Safe default: transient retry prevents message drop. |

---

## 5. Telegram Proxy Schemes & Transport Support

### Go Standard Library Transport Support
Go's standard `net/http.Transport.Proxy` function evaluates the URL scheme using `http.ProxyURL`:
- `http://` — Supported (HTTP CONNECT tunnel for TLS, plain HTTP proxy for non-TLS).
- `https://` — Supported (TLS-encrypted HTTP proxy connection).
- `socks5://` — Supported (Native SOCKS5 client built into Go 1.10+ `net/http`).
- `socks5h://` — **NOT Supported by Go standard library `net/http`**. Go's standard proxy resolver treats `socks5h` as an unsupported scheme.

### Configuration Enforcement
In `internal/config/config.go`:
- `SupportedTelegramProxySchemes` explicitly allows `http`, `https`, and `socks5`.
- Requests with `socks5h://` are rejected with:  
  `"unsupported TELEGRAM_PROXY_URL scheme \"socks5h\": Go net/http transport does not support socks5h (use socks5, http, or https)"`.
- Empty or missing hostnames (e.g. `http://`) and malformed URLs are rejected.
- Credential safety: `SanitizeProxyURL` replaces user/pass with `[REDACTED]` in all error messages and logs.

---

## 6. Password Reset Reliability & Anti-Enumeration

### Why Delivery Semantics Were Preserved Synchronously
Per TASK-012.2.6 requirements:
- Password reset email delivery was **NOT** moved into an unmanaged background goroutine. Untracked goroutines would risk dropped emails if the API process restarts or terminates under load.
- Asynchronous queuing was not introduced because database migrations and new queue tables are prohibited in this scope.
- Instead, bounded execution guarantees that synchronous delivery will complete or time out within 15 seconds.

### Anti-Enumeration Protection
- Regardless of whether the user exists, whether token generation succeeds, or whether the unified SMTP sender encounters an error, `AuthHandler.ForgotPassword` returns an identical HTTP 200 response:
  ```json
  {
    "message": "If your email is registered, you will receive password reset instructions shortly."
  }
  ```
- This guarantees zero side-channel information leakage regarding account existence.

---

## 7. Tests Executed & Exact Results

| Test Package | Test Suite / Focus | Result |
| :--- | :--- | :--- |
| `internal/mail` | `TestSMTPMailer_*` (delegation, errors, templates, nil check) | **PASS** (0.003s) |
| `internal/notification/email` | `TestSMTP_*` (TCP timeout, stalled greeting, stalled transmission, cancellation, STARTTLS failures, port detection) | **PASS** (0.447s) |
| `internal/worker` | `TestEmailErrorClassification` (32 table-driven error cases) | **PASS** (0.000s) |
| `internal/worker` | `TestNotificationWorker_DeliveryTimeout_*` (Email & Telegram 15s bounds) | **PASS** (0.120s) |
| `internal/config` | `TestConfigValidation` (Proxy schemes, malformed URLs, credentials) & `TestSanitizeProxyURL` | **PASS** (0.003s) |
| `internal/server` | `TestForgotPasswordAntiEnumerationAndSMTPFailure` | **PASS** (0.075s) |
| `internal/server` | `TestNotificationServerAndWorkerEndToEnd` | **PASS** (0.200s) |
| `internal/server` | `TestAuthAndTenantLifecycleE2ERegression` | **PASS** (0.260s) |
| `internal/repository` | `task_006_notifications_test.go` (tokens, binding, fan-out, deduplication, retry, RLS) | **PASS** (0.910s) |

### Pre-Existing Test Status Note
When running the full repository test suite concurrently against the local development PostgreSQL database (`docker-compose.yml`), `TestTask009_1_EntitlementConcurrency` and `TestScheduledMonitoringEndToEndEvidence` intermittently encounter quota race conditions or monitor lease contention from the background Docker worker container (`clientpulse_worker` container ID `854495db8eb3`). Running these tests in isolation confirms they are pre-existing test suite behaviors and completely unrelated to the notification engine changes.

---

## 8. Remaining Limitations

1. **Transactional Outbox for Password Resets:**
   - While password-reset delivery now has bounded timeouts and anti-enumeration protection, true asynchronous reliability without goroutine loss requires a transactional Outbox pattern (writing reset tokens and delivery intents in the same database transaction, claimed by a dedicated worker). This requires database migrations, which were prohibited in Phase 1.
2. **SOCKS5 DNS Resolution:**
   - Standard Go `net/http` SOCKS5 client resolves hostnames locally rather than on the proxy (`socks5h`). If remote DNS resolution via SOCKS5 proxy is strictly required in the future, a custom `net.Dialer` using `golang.org/x/net/proxy` will need to be configured.

---

## 9. Strict Scope & Production Boundary Confirmation

- **Production VPS:** Untouched. No SSH connections established.
- **Production Environment (`.env`):** Untouched.
- **Production Compose (`deploy/docker-compose.production.yml`):** Untouched.
- **Database Migrations:** None executed or created.
- **Production Containers:** No production containers were restarted or modified.
- **DNS, Nginx, Proxies:** Untouched.
- **Credentials:** No production credentials or real emails/messages were used or created.

---

## 10. Recommended Next Task

**TASK-012.3: Notification Delivery Observability & Outbox Preparation**
- Design an Outbox schema for transactional authentication emails.
- Add OpenTelemetry metrics around SMTP delivery latencies and per-provider error codes.
- Prepare Phase 2 notification engine migration plan.

