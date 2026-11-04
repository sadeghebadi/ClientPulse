# TASK-012.1: Notification Infrastructure Audit & Delivery Readiness

**Project:** ClientPulse  
**Priority:** Critical  
**Audit Scope:** Notification Engine, Telegram Long Polling / Webhook, SMTP Relays, VPS Network Egress, and Security Guarantees  
**Execution Mode:** Local Codebase Analysis + Read-only Production Diagnostics  
**Status:** Completed & Verified  
**Date:** 2026-11-04  

---

## 1. Existing Architecture & Delivery Flow

### 1.1 Architecture Diagram

```mermaid
flowchart TD
    subgraph Trigger["Domain Event Trigger"]
        M[Monitor Check Failure / Recovery] --> NE[(notification_events)]
        H[Heartbeat Missed / Recovered] --> NE
        I[Incident Opened / Ack / Resolved] --> NE
        R[Maintenance Report Ready] --> NE
    end

    subgraph Dispatcher["Worker Dispatcher (1s loop)"]
        NE -->|DispatchPendingEvents| DPE[Event Dispatcher & Preferences Filter]
        ND[(notification_destinations)] -->|Join Preferences & RLS| DPE
        DPE -->|Idempotent Fan-out\nON CONFLICT DO NOTHING| NDeliv[(notification_deliveries\nstatus='queued')]
        DPE -->|Update Event status='delivered'| NE
    end

    subgraph DeliveryWorker["Delivery Worker (Concurrency=5..10)"]
        NDeliv -->|claim_due_notification_deliveries\nFOR UPDATE SKIP LOCKED\nstatus='sending'| CW[Claimed Deliveries]
        CW --> Route{Destination Type?}
        
        Route -->|telegram_private\ntelegram_group| TG[Telegram Delivery Engine]
        Route -->|email| EM[SMTP Delivery Engine]
        
        TG -->|HTTP 200 OK| SentTG[MarkDeliverySent\nstatus='sent']
        EM -->|SMTP 250 OK| SentEM[MarkDeliverySent\nstatus='sent']
        
        TG -->|HTTP 429 / 5xx| RetryTG[MarkDeliveryFailed\nstatus='retrying'\nExponential Backoff]
        EM -->|SMTP 4xx / Timeout| RetryEM[MarkDeliveryFailed\nstatus='retrying'\nExponential Backoff]
        
        TG -->|HTTP 400 / 403 / Max Retries| FailTG[MarkDeliveryFailed\nstatus='failed'\nPermanent]
        EM -->|SMTP 550 / 554 / Max Retries| FailEM[MarkDeliveryFailed\nstatus='failed'\nPermanent]
        
        SentTG --> NDeliv
        SentEM --> NDeliv
        RetryTG --> NDeliv
        RetryEM --> NDeliv
        FailTG --> NDeliv
        FailEM --> NDeliv
    end

    subgraph StaleRecovery["Stale Recovery Loop (1m)"]
        NDeliv -.->|RequeueStaleDeliveries\nstatus='sending' > 5m| Rec{retry_count > 0?}
        Rec -->|Yes| Uncert[status='uncertain'\nPrevent Duplicate Replay]
        Rec -->|No| Requeued[status='retrying']
        Uncert -.-> NDeliv
        Requeued -.-> NDeliv
    end
```

### 1.2 Production Network Topology & Isolation Diagram

```mermaid
flowchart LR
    subgraph Host["Production VPS (37.152.191.170:2222)"]
        subgraph PreExisting["Protected Containers (Zero Disruption)"]
            RC[rocketchat-rocketchat-1]
            MM[fchat-mattermost-1 / docker-main]
            RD[rustdesk-hbbs / hbbr]
        end

        NginxHost["Host Nginx\n(mindu.ir:80, 443)"]
        
        subgraph CPCompose["ClientPulse Docker Compose"]
            WebContainer["clientpulse-web\n(:3000)"]
            APIContainer["clientpulse-api\n(:8080)"]
            WorkerContainer["clientpulse-worker\n(scheduler & notifications)"]
            PostgresContainer["clientpulse-postgres\n(PostgreSQL 16)"]
        end
    end

    subgraph Networks["Docker Network Segmentation"]
        AppNet["clientpulse_app_internal\n(bridge, private)"]
        DataNet["clientpulse_data_internal\n(internal: true, NO egress)"]
        EgressNet["clientpulse_egress\n(outbound NAT bridge)"]
    end

    subgraph External["External Services"]
        TGApi["Telegram Bot API\n(api.telegram.org:443)\n[Censored in Iran / Poisoned DNS]"]
        ExtProxy["External HTTP/SOCKS5 Proxy\n(Hosted outside Iran)"]
        SMTPRelay["SMTP Relays (Port 587)\n(Brevo, AWS SES, Gmail)"]
    end

    NginxHost -->|Proxy Pass 127.0.0.1:3100| WebContainer
    NginxHost -->|Proxy Pass 127.0.0.1:8090| APIContainer

    WebContainer --- AppNet
    APIContainer --- AppNet
    APIContainer --- DataNet
    WorkerContainer --- DataNet
    PostgresContainer --- DataNet
    WorkerContainer --- EgressNet

    WorkerContainer -.->|Encrypted Tunnel| ExtProxy
    ExtProxy -.->|Unfiltered Outbound| TGApi
    WorkerContainer -->|Direct Port 587 STARTTLS| SMTPRelay
```

---

## 2. Current Implementation Findings

### 2.1 Event Generation and Dispatch Flow
- **Generation:** Domain events are created upon state changes in `services/api/internal/server` and `services/api/internal/worker/scheduler.go`:
  - `incident.opened`, `incident.acknowledged`, `incident.resolved`
  - `heartbeat.missed`, `heartbeat.recovered`
  - `report.ready` (emitted only after PDF compilation and storage persistence succeed)
- **Dispatching:**
  - Background goroutine in `NotificationWorker` queries `notification_events` where `status = 'pending'` every 1 second in batches of up to 50.
  - Matches active destinations in `notification_destinations` via join on `notification_preferences` for the given `event_type`.
  - **Ownership Gate:** Excludes unverified email destinations via `AND (d.destination_type != 'email' OR d.verified_at IS NOT NULL)`.
  - **Fan-Out Idempotency:** Each delivery record is inserted with `idempotency_key = fmt.Sprintf("deliv:%s:%s", event.ID, dest.ID)` with `ON CONFLICT (idempotency_key) DO NOTHING`.
  - Marks event status as `delivered` (or `suppressed` if no destinations were enabled).

### 2.2 Provider Interfaces and Implementations
- **Telegram:**
  - Interface `telegram.TelegramSender` defines `SendMessage(ctx, chatID, text, parseMode)` and `GetMe(ctx)`.
  - Extended interface `telegram.TelegramBotAPI` adds `GetWebhookInfo`, `DeleteWebhook`, `GetUpdates`.
  - Client implementation: `services/api/internal/notification/telegram/client.go` (`NewClientWithProxy`).
  - Supports configurable timeouts, bot token redaction via `SanitizeTelegramURL`, and proxy configuration.
- **Email:**
  - **Architectural Discrepancy Identified:** Two separate email modules exist in the codebase:
    1. `services/api/internal/notification/email/smtp.go` (`email.Client`): Implements `email.EmailSender`. Supports `starttls`, `tls`, and `none`. Used by `NotificationWorker` for alerts and `NotificationHandler` for destination verification and manual test emails.
    2. `services/api/internal/mail/mailer.go` (`mail.SMTPMailer`): Implements `mail.EmailSender`. Uses Go's standard library `net/smtp.SendMail`. Does NOT support explicit TLS (port 465), ignores `context.Context`, lacks connection deadlines, and is used solely by `auth_service.go` for signup email verification and password resets.
  - **Recommendation:** Unify all email transmission through a single, robust client engine (`internal/notification/email`) with strict context timeouts.

### 2.3 Queue, Concurrency, and Stale Recovery
- **Queue Mechanics:**
  - Deliveries are persisted in `notification_deliveries` (`status = 'queued'`).
  - Workers claim batches using PostgreSQL security-definer function `claim_due_notification_deliveries(worker_id, limit)` which executes `SELECT ... FOR UPDATE SKIP LOCKED` and transitions status to `'sending'`.
  - Processing concurrency is bounded by a semaphore (`WorkerConcurrency`, default 5 to 10).
- **Stale Worker Recovery:**
  - `RequeueStaleDeliveries` runs every 1 minute, checking deliveries in `sending` status with `last_attempt_at < NOW() - 5m`.
  - If `retry_count > 0` (indicating external transmission may have started prior to a crash), the status transitions to `uncertain` with a recovery audit note in `delivery_attempts` JSONB. It is NOT automatically re-sent, preventing duplicate external deliveries across network partitions.
  - If `retry_count == 0`, it transitions to `retrying`.

### 2.4 Backoff, Retries, and Error Classification
- **Exponential Backoff:** `calculateBackoff(retryCount)` produces $2 \times 2^{\text{retryCount}} + \text{jitter}$ (seconds: ~2s, ~4s, ~8s, ~16s, ~32s, capped at 300s).
- **Telegram Error Classification:**
  - HTTP 429: extracts `parameters.retry_after` and sleeps/backs off accurately.
  - HTTP 400 / 403 / 404: classified as `PermanentError` (`permanent = true`), immediately marking the delivery `failed`.
- **Email Error Classification:**
  - `isPermanentEmailError` checks for strings: `invalid`, `mailbox unavailable`, `no such user`, `user unknown`, `550`, `554`.
  - **Gap Identified:** SMTP `535` (Authentication Credentials Invalid) is currently not in the permanent error list, leading to redundant retries on bad credentials.

### 2.5 Multi-Tenant Isolation & Anti-Hijacking
- Row-Level Security (RLS) is enabled and forced on all four notification tables: `notification_destinations`, `notification_preferences`, `notification_deliveries`, `notification_connection_tokens`.
- **Telegram Group Hijacking Prevention:**
  - Schema migration `000010` created partial unique index:
    ```sql
    CREATE UNIQUE INDEX uq_active_telegram_group
    ON notification_destinations (telegram_chat_id)
    WHERE destination_type = 'telegram_group'
      AND status IN ('active', 'pending_approval')
      AND telegram_chat_id IS NOT NULL;
    ```
  - Unlinked group connection requests (`/connect CODE`) require authenticated administrator approval from the dashboard (`pending_group_approval = true`).
- **Email Destination Ownership:**
  - Created with `status = 'pending'`, `is_enabled = false`, and `verified_at = NULL`.
  - Single-use 32-byte cryptographic token hashed with SHA-256 (`verification_token_hash`).
  - Cooldown: 60-second rate limit on verification resend requests.

### 2.6 Environment Configuration Gap in Production Compose
- **Critical Finding in `deploy/docker-compose.production.yml`:**
  - The production Compose manifest currently hardcodes:
    ```yaml
    EMAIL_PROVIDER: disabled
    TELEGRAM_MODE: disabled
    ```
  - Neither `SMTP_*` nor `TELEGRAM_*` environment variables are mapped under `services.api.environment` or `services.worker.environment`.
  - Even if credentials are added to `/home/ubuntu/clientpulse/.env`, production containers will NOT receive them until `docker-compose.production.yml` is updated to interpolate these environment variables.

---

## 3. Telegram Infrastructure Investigation

### 3.1 Known VPS Outbound Issue: Root Cause Diagnosis
Read-only diagnostics executed on production VPS (`37.152.191.170`) identified the exact mechanism causing outbound Telegram connections to time out or refuse connection:

1. **DNS Poisoning / Sinkholing:**
   - Command: `dig +short api.telegram.org`
   - Result: `10.10.34.35` (and IPv6 `2001:4188:2:600:10:10:34:35`).
   - Querying upstream public DNS (`dig @8.8.8.8 +short api.telegram.org`) returns the identical sinkhole IP `10.10.34.35`.
   - **Conclusion:** Upstream ISP DPI (Deep Packet Inspection) in Iran intercepts all UDP/TCP port 53 DNS queries and returns national filtering sinkhole addresses.
2. **TCP Connection Refusal / Drop:**
   - Probing `api.telegram.org:443` resolves to `10.10.34.35:443`, which immediately returns:
     `curl: (7) Failed to connect to api.telegram.org port 443 after 10 ms: Connection refused`.
   - Probing genuine Telegram IP ranges directly is dropped or rejected at the datacenter gateway.
3. **Existing Host Squid Proxy (`127.0.0.1:6824`):**
   - Probing through the host's Squid service returned `curl: (56) Received HTTP code 503 from proxy after CONNECT (ERR_CONNECT_FAIL 101)`.
   - Squid uses the host's DNS and local network routing; it cannot reach Telegram.

### 3.2 Outbound Protocol Requirements
- **Required Domain:** `api.telegram.org`
- **Required Port:** `443` (HTTPS)
- **Traffic Direction:** Outbound only (when running in `TELEGRAM_MODE=polling`). In polling mode, the worker establishes outbound HTTPS long-poll connections (`getUpdates`) and outbound `sendMessage` requests. No inbound firewall ports or public webhook endpoints are required.

### 3.3 HTTP & SOCKS5 Proxy Architecture
The ClientPulse Telegram client natively supports proxy configuration via `TELEGRAM_PROXY_URL`:
- Go's `net/http` standard library transport natively supports `http://`, `https://`, `socks5://`, and `socks5h://` schemes.
- Both HTTP CONNECT and SOCKS5 proxies perform **remote DNS resolution** on the proxy server. This completely bypasses local DNS poisoning and ISP routing restrictions.
- **Proxy Safety & Isolation:**
  - `TELEGRAM_PROXY_URL` is injected strictly into the application's Telegram HTTP client.
  - It does NOT set global OS environment variables (`HTTP_PROXY`, `ALL_PROXY`).
  - It does NOT alter VPS iptables, routing tables, systemd, or Nginx.
  - Protected containers (`rocketchat`, `mattermost`, `rustdesk`) remain completely isolated and unaffected.

### 3.4 Independent Verification Methodology
Telegram connectivity can and MUST be verified independently from the ClientPulse application without modifying production services:

```bash
# 1. Independent Probe (No Bot Token required) - Verifies DNS, TCP, TLS through proxy
curl -x "${TELEGRAM_PROXY_URL}" -s -I "https://api.telegram.org" | head -n 5
# Expected: HTTP/2 302 or HTTP/1.1 302 Found

# 2. Independent Bot Token & Identity Verification (Redacted token in subshell)
curl -x "${TELEGRAM_PROXY_URL}" -s "https://api.telegram.org/bot${BOT_TOKEN}/getMe" | jq '{ok: .ok, id: .result.id, username: .result.username}'
# Expected: {"ok": true, "id": ..., "username": "ClientPulseAlertsBot"}
```

### 3.5 Recommended Connectivity Architecture
Two verified architectural options are available:

| Architecture Option | Configuration | Mechanism | Safety & Complexity |
|---|---|---|---|
| **Option A: External Proxy (Recommended)** | `TELEGRAM_PROXY_URL=http://user:pass@proxy-host:port`<br>or `socks5://user:pass@proxy-host:port` | ClientPulse worker dials an external HTTP/SOCKS5 proxy server hosted outside Iran. | **Zero host impact.** Zero container changes. Only requires setting one environment variable. |
| **Option B: Compose Sidecar Proxy** | Add a lightweight sidecar container (e.g. `gost`, `xray-core`, or `shadowsocks`) to `docker-compose.production.yml` | Worker connects to `http://telegram-proxy:1080` over the internal Docker network. | Scoped 100% inside `clientpulse` Compose project. No host ports exposed. Adds ~30MB RAM overhead. |

---

## 4. Email / SMTP Investigation

### 4.1 Production VPS Egress Diagnostics
Read-only TCP connectivity probes were executed from both the VPS host and from within the running `clientpulse-worker-1` and `clientpulse-api-1` containers.

| Target SMTP Provider | Destination Host & Port | Connection Status | Diagnostic Finding |
|---|---|---|---|
| **Brevo (Sendinblue)** | `smtp-relay.sendinblue.com:587` | **OPEN (Instant)** | Verified working from host and worker container. |
| **AWS SES (EU)** | `email-smtp.eu-west-1.amazonaws.com:587` | **OPEN (Instant)** | Verified working from host and worker container. |
| **Google Workspace / Gmail** | `smtp.gmail.com:587` | **OPEN (Instant)** | Verified working from host and worker container. |
| **SendGrid** | `smtp.sendgrid.net:587` | **OPEN (Instant)** | Verified working from host and worker container. |
| **Mailjet** | `in-v3.mailjet.com:587` | **TIMED OUT** | Destination IP / routing unreachable from VPS ISP. |
| **Any Provider (Port 465)** | `smtp.gmail.com:465` / `smtp.sendgrid.net:465` | **TIMED OUT** | Port 465 (SMTPS direct TLS) is blocked upstream by VPS ISP. |
| **Any Provider (Port 25)** | Standard SMTP port 25 | **BLOCKED** | Standard hosting provider port 25 egress filter. |

> [!IMPORTANT]
> **Production Rule:** All SMTP integrations on this VPS MUST use **Port 587 with STARTTLS** (`SMTP_PORT=587`, `SMTP_TLS_MODE=starttls`). Port 465 (direct TLS) and Port 25 will fail due to datacenter network restrictions.

### 4.2 SMTP Provider Comparison

| Provider | Free / Base Tier | Egress Status | Setup & Verification Requirements | Recommendation |
|---|---|---|---|---|
| **Brevo (Sendinblue)** | 300 emails/day free | **Verified Open (587)** | Requires API key as SMTP password, SPF `include:spf.sendinblue.com`, DKIM TXT record. | **Recommended (Primary)** for initial launch. Generous free tier, reliable European delivery. |
| **Google Workspace SMTP Relay** | 2,000 emails/day | **Verified Open (587)** | Requires Google Workspace account for `mindu.ir` and 16-character App Password. | **Recommended (Alternative)** if Workspace is already licensed for `mindu.ir`. |
| **Amazon SES** | $0.10 / 1,000 emails | **Verified Open (587)** | Requires AWS account, sandbox exit request, domain TXT verification. | Highly reliable long-term, requires AWS management setup. |
| **SendGrid** | 100 emails/day free | **Verified Open (587)** | Requires Twilio/SendGrid account, API Key auth. | Feasible, but web portal occasionally throttles from Iranian IPs. |
| **Mailjet** | 200 emails/day free | **Blocked / Timeout** | Incompatible with current datacenter routing. | **Not viable.** |

### 4.3 Proposed Sender Domain & DNS Records (`mindu.ir`)
Live DNS query results on `mindu.ir`:
- **Current SPF:** `"v=spf1 include:_spf.mx.cloudflare.net ~all"` (Configured for Cloudflare inbound email forwarding).
- **Current DMARC:** `"v=DMARC1; p=none; rua=mailto:3d9bd16034cf44ddaf5344cc3cab3dbe@dmarc-reports.cloudflare.net"`
- **Current MX:** Cloudflare Email Routing (`route1.mx.cloudflare.net`, etc.).

#### Required DNS Updates for Outbound Delivery:
Because Cloudflare Email Routing only handles inbound mail, sending outbound email requires authorization for the outbound SMTP relay:

1. **Option 1: Subdomain Delegation (Recommended):**
   Use `notifications@pulse.mindu.ir` or `notifications@mail.mindu.ir`.
   - Leaves root domain `mindu.ir` SPF/MX completely untouched.
   - Separate sender reputation protects primary domain deliverability.
   - Records required on subdomain:
     - TXT `pulse.mindu.ir`: `"v=spf1 include:spf.sendinblue.com ~all"` (if Brevo)
     - CNAME / TXT `mail._domainkey.pulse.mindu.ir`: `<DKIM-KEY-FROM-PROVIDER>`
     - TXT `_dmarc.pulse.mindu.ir`: `"v=DMARC1; p=none; sp=none; rua=mailto:dmarc@mindu.ir"`

2. **Option 2: Apex Domain Integration (`mindu.ir`):**
   Use `notifications@mindu.ir`.
   - Update existing TXT SPF record on `mindu.ir` to append the provider:
     `"v=spf1 include:_spf.mx.cloudflare.net include:spf.sendinblue.com ~all"`
   - Add provider-generated DKIM TXT record:
     `mail._domainkey.mindu.ir` -> Provider public key.

---

## 5. Security & Reliability Gaps Identified

| ID | Component | Gap Description | Severity | Remediation Plan |
|---|---|---|---|---|
| **GAP-01** | `deploy/docker-compose.production.yml` | `EMAIL_PROVIDER` and `TELEGRAM_MODE` are hardcoded to `disabled`; no `SMTP_*` or `TELEGRAM_*` variables are forwarded into containers. | **Critical** | Update Compose file to parameterize integration variables with sensible `${VAR}` defaults. |
| **GAP-02** | `internal/mail/mailer.go` vs `internal/notification/email/smtp.go` | Two disparate email implementations. `mail.SMTPMailer` ignores context, has no connection timeouts, and lacks TLS options. | **High** | Unify `auth_service` and `notification_worker` under the hardened `notification/email` client. |
| **GAP-03** | `internal/notification/email/smtp.go` | `sendSTARTTLS` sets a 10s TCP dial timeout but does not set deadlines on the underlying connection (`net.Conn.SetDeadline`). A hanging `DATA` or `MAIL` command can block indefinitely. | **High** | Add connection deadlines respecting the caller's `context.Context`. |
| **GAP-04** | `internal/worker/notification_worker.go` | `w.Deliver(ctx, deliv)` receives the parent `workerCtx` (process lifetime) with no per-delivery timeout. | **Medium** | Enforce a bounded 15-second per-delivery context timeout (`context.WithTimeout(ctx, 15*time.Second)`). |
| **GAP-05** | `internal/worker/notification_worker.go` | `isPermanentEmailError` does not classify SMTP `535 Authentication failed` or `501 Syntax error` as permanent. | **Medium** | Expand error parsing to treat 535 and 501/502/503/504 as non-retryable terminal errors. |
| **GAP-06** | `internal/service/auth_service.go` | `RequestPasswordReset` sends email synchronously inside the HTTP handler. If SMTP is slow, the HTTP request latency spikes. | **Low** | Move password reset email dispatch to a non-blocking background goroutine with bounded context. |
| **GAP-07** | `internal/config/config.go` | `TELEGRAM_PROXY_URL` syntax and scheme are not validated during `Validate()`. Malformed URLs fail silently at runtime. | **Low** | Add URL validation (`url.ParseRequestURI` and scheme checks for `http`, `https`, `socks5`, `socks5h`). |

---

## 6. Required Environment Variables (Names Only)

The following variable names must be supported across `.env.production` and `docker-compose.production.yml`. **No secrets or credentials are included.**

### 6.1 Email / SMTP Variables
```env
# Provider toggle: "disabled", "smtp"
EMAIL_PROVIDER

# SMTP Relay Connection
SMTP_HOST
SMTP_PORT
SMTP_USERNAME
SMTP_PASSWORD

# Sender Identity & TLS Mode
SMTP_FROM_EMAIL
SMTP_FROM_NAME
SMTP_TLS_MODE
```

### 6.2 Telegram Bot Variables
```env
# Mode toggle: "disabled", "polling", "webhook"
TELEGRAM_MODE

# Bot Identity
TELEGRAM_BOT_TOKEN
TELEGRAM_BOT_USERNAME

# Egress Proxy Configuration
TELEGRAM_PROXY_URL

# Polling Tuning
TELEGRAM_POLL_TIMEOUT_SECONDS

# Webhook Secret (if webhook mode used in future)
TELEGRAM_WEBHOOK_SECRET
```

### 6.3 Centralized Links
```env
# Central canonical application URL for links in alerts and verification emails
PUBLIC_APP_URL
```

---

## 7. Proposed Implementation Phases (TASK-012 Roadmap)

```mermaid
flowchart TD
    P1["Phase 1: Code Hardening & Unification\n(Local Development & Unit Tests)"]
    P2["Phase 2: Configuration & Compose Manifest Alignment\n(Local Compose Verification)"]
    P3["Phase 3: Out-of-Band Preflight Verification\n(Independent VPS Connectivity Tests)"]
    P4["Phase 4: Staged Production Activation\n(Email First, Then Telegram with Proxy)"]

    P1 --> P2 --> P3 --> P4
```

### Phase 1: Local Code Hardening & Mailer Unification
1. Replace `internal/mail/mailer.go` with an adapter wrapping `internal/notification/email.Client`.
2. Add connection read/write deadlines to `email/smtp.go` using `context.Deadline()`.
3. Add per-delivery context timeout (15s) in `NotificationWorker.Deliver()`.
4. Add SMTP 535 and 50x classification to `isPermanentEmailError`.
5. Add URL and scheme validation for `TELEGRAM_PROXY_URL` in `config.Validate()`.
6. Run comprehensive regression test suite (`go test ./internal/...`).

### Phase 2: Configuration & Manifest Alignment
1. Update `deploy/docker-compose.production.yml` to map all `SMTP_*` and `TELEGRAM_*` variables from `.env` into `api` and `worker` services.
2. Verify local Docker Compose launch with Mailpit and Mock Telegram.

### Phase 3: Out-of-Band Production Preflight Diagnostics
1. Execute standalone SMTP probe script to verify authentication and delivery on port 587 without touching ClientPulse containers:
   - Verify `smtp-relay.sendinblue.com:587` with test credentials.
2. Execute standalone Telegram probe script through the target proxy:
   - Verify `curl -x $TELEGRAM_PROXY_URL https://api.telegram.org/bot$TOKEN/getMe`.
3. Verify DNS SPF and DKIM records for `mindu.ir` / `pulse.mindu.ir`.

### Phase 4: Staged Production Activation
1. **Stage 4A (Email Delivery):**
   - Inject verified SMTP credentials into production `/home/ubuntu/clientpulse/.env`.
   - Keep `TELEGRAM_MODE=disabled`.
   - Redeploy ClientPulse release.
   - Verify user signup verification email and manual destination test email via Dashboard.
2. **Stage 4B (Telegram Delivery):**
   - Inject `TELEGRAM_MODE=polling`, `TELEGRAM_BOT_TOKEN`, and `TELEGRAM_PROXY_URL`.
   - Restart worker service.
   - Verify poller acquires singleton advisory lock and connects cleanly.
   - Verify private `/start` and group `/connect` flows via Telegram bot.
   - Monitor `notification_deliveries` table for clean status transitions (`sent`).

---

## 8. Exact Production Deployment Prerequisites

Before modifying production configuration or deploying TASK-012:

- [ ] **Egress Proxy Available:** A verified external HTTP or SOCKS5 proxy running outside Iran with stable uptime and credentials.
- [ ] **Proxy Validated Independently:** `curl -x <PROXY_URL> https://api.telegram.org` returns HTTP 302 from the production VPS shell.
- [ ] **SMTP Provider Account Ready:** Brevo, AWS SES, or Google Workspace account created with SMTP credentials generated.
- [ ] **SMTP Port 587 Confirmed:** Connection to target SMTP host on port 587 verified using netcat from the VPS shell.
- [ ] **DNS SPF Record Published:** Sender domain TXT record contains provider include (e.g. `include:spf.sendinblue.com`).
- [ ] **DNS DKIM Record Published:** Sender domain DKIM selector published and passing validation.
- [ ] **Zero Disruption to Host Services:** Confirmed that `docker compose -p clientpulse` is used exclusively, protecting all 8 pre-existing containers.

---

## 9. Risks and Rollback Strategy

### 9.1 Risk Matrix

| Risk Scenario | Impact | Mitigation Strategy |
|---|---|---|
| **Egress Proxy Disconnection** | Telegram notifications fail to deliver; getUpdates logs transient errors. | `NotificationWorker` retries with exponential backoff. Status transitions cleanly to `retrying` and `uncertain` after 5m without data loss or message flooding. |
| **SMTP Provider Rejection (535 Auth / 550 Blocked)** | Email notifications fail to deliver. | Permanent error detection marks deliveries `failed` immediately. Error message logged in `delivery_attempts` JSONB. |
| **Worker Crash During Send** | Delivery status stuck in `sending`. | `RequeueStaleDeliveries` marks records `uncertain` after 5 minutes, preventing duplicate re-transmission. |
| **Host Service Disruption** | Protected container or host port affected. | Strictly prohibited. ClientPulse operates strictly on internal docker bridges and localhost-bound ports (8090, 3100). |

### 9.2 Emergency Rollback Procedure
If any unexpected delivery or connectivity failure occurs upon activation:

1. **Immediate Disable via Environment:**
   Edit `/home/ubuntu/clientpulse/.env` on the production VPS:
   ```env
   EMAIL_PROVIDER=disabled
   TELEGRAM_MODE=disabled
   ```
2. **Graceful Service Restart:**
   ```bash
   sudo docker compose -p clientpulse -f /home/ubuntu/clientpulse/docker-compose.production.yml up -d --no-deps api worker
   ```
3. **Verify Clean Quiescence:**
   Deliveries created during the outage transition to `cancelled` with reason `"transport disabled"`. Monitoring checks, heartbeat processing, and web dashboard continue operating without disruption.
4. **Inspect Delivery Audit Trail:**
   Query `notification_deliveries` to inspect failure reasons:
   ```sql
   SELECT id, destination_type, status, retry_count, error_message, last_attempt_at
   FROM notification_deliveries
   ORDER BY created_at DESC LIMIT 10;
   ```

