# ClientPulse Architecture Overview

## 1. System Topology

ClientPulse is designed as a **Modular Monolith** with a decoupled background worker process, communicating over a centralized PostgreSQL data store.

```mermaid
graph TD
    User([Agency User / Client Browser]) -->|HTTPS / UI| Web[Next.js Frontend (apps/web)]
    User -->|REST API / HTTPS| Caddy[Caddy / Reverse Proxy]
    Caddy -->|/api/* & /health/*| API[Go API Service (services/api/cmd/api)]
    Caddy -->|/*| Web
    
    API -->|Read / Write Pool / Tenant Context| Postgres[(PostgreSQL 16)]
    Worker[Go Worker Process (services/api/cmd/worker)] -->|Heartbeats / Future Job Loop| Postgres
    Worker -->|Outbound Health / SSL / DNS Checks| ExternalSites[Target Websites / Endpoints]
```

---

## 2. Repository Layout

```text
clientpulse/
├── apps/
│   └── web/                     # Next.js 16+ (App Router), React 19.2, TypeScript, Tailwind CSS
│       ├── src/app/             # Pages: /login, /signup, /onboarding, /organizations, /settings/*
│       └── src/lib/             # api client, auth-context, i18n
├── services/
│   └── api/                     # Go 1.27+ Modular Monolith (API & Worker)
│       ├── cmd/
│       │   ├── api/             # HTTP API entrypoint
│       │   ├── worker/          # Background worker entrypoint
│       │   └── migrate/         # Database migration CLI
│       └── internal/
│           ├── auth/            # Argon2id hashing, CSPRNG tokens, SHA-256 peppered hashing
│           ├── config/          # Environment configuration loading & validation
│           ├── database/        # Connection pool (pgxpool) & ping
│           ├── health/          # Live/Ready/Worker health handlers
│           ├── logger/          # Structured JSON logging with slog & correlation ID
│           ├── mail/            # Email sender interface & SMTP / Mailpit adapter
│           ├── migrate/         # SQL migration engine with advisory locking
│           ├── rate/            # Sliding-window rate limiter
│           ├── repository/      # Models, tenant transactions, RLS integration & repositories
│           ├── server/          # HTTP server, routing, auth & tenant middlewares, handlers
│           ├── service/         # Auth & Organization business domain logic
│           └── tenant/          # TenantContext, UserContext, and RBAC role helpers
├── migrations/                  # Versioned SQL migrations (.up.sql / .down.sql)
├── deploy/                      # Dockerfiles for api, worker, and web
├── docs/                        # Architecture, ADRs, API, security, operations
├── task/                        # Implementation task tracking
├── task-report/                 # Phase execution reports
├── docker-compose.yml           # Local dev orchestrator
├── .env.example                 # Environment variables specification
└── README.md                    # Setup and execution guide
```

---

## 3. Core Modules (services/api/internal/)

| Module | Responsibility |
|---|---|
| `auth` | Argon2id password hashing, CSPRNG tokens, HMAC-SHA256 peppered hashing |
| `config` | Type-safe environment loading and validation |
| `database` | PostgreSQL connection pooling via `jackc/pgx/v5/pgxpool` |
| `health` | Live, Ready, and Worker heartbeat health check probes |
| `logger` | Structured JSON logging using standard `log/slog` with correlation ID |
| `mail` | `EmailSender` interface, SMTP provider (Mailpit in dev), MockMailer for tests |
| `migrate` | Versioned schema migration runner protected by session advisory lock |
| `rate` | Thread-safe sliding window rate limiter |
| `repository` | PostgreSQL data access with `WithTenantTx` and Row-Level Security (RLS) enforcement |
| `server` | HTTP routing, session cookie auth, tenant resolution, RBAC, CSRF, rate limit middlewares |
| `service` | Business logic for Authentication, User Accounts, Organizations, Memberships, and Roles |
| `ssrf` | Multi-layer outbound SSRF protection, IP classification, DNS rebinding defense & socket pinning |
| `checker` | Protocol check engine (HTTP GET/HEAD, SSL certificate lifecycle, JSON Pointer, DNS) |
| `tenant` | TenantContext, UserContext, and RBAC permission evaluation (`HasRole`) |
| `worker` | Monitoring scheduler daemon, atomic job claiming, drift prevention & check persistence |

---

## 4. Monitoring Engine & Scheduler Architecture

ClientPulse implements a distributed, drift-free scheduler using PostgreSQL as the single source of truth:

```mermaid
sequenceDiagram
    autonumber
    participant W as Worker Daemon (services/api/cmd/worker)
    participant DB as PostgreSQL (claim_due_monitors)
    participant SG as SSRF Guard & Dialer
    participant Target as Monitored Target Server

    loop Every 2 Seconds
        W->>DB: Call claim_due_monitors(worker_id, batch_size=10, lease_seconds=60)
        Note over DB: Atomically claims due monitors<br/>FOR UPDATE OF m SKIP LOCKED<br/>Sets lease_expires_at & active worker_id
        DB-->>W: Returns batch of ClaimedMonitorJobs
        
        par For each claimed monitor in worker pool
            W->>SG: Resolve DNS and validate public IP
            alt Prohibited IP or Private Range
                SG-->>W: SSRF Blocked Error
            else Public IP Validated
                SG->>Target: Outbound HTTP/HTTPS check (Pin IP, keep SNI)
                Target-->>SG: HTTP Response + TLS Peer Certificates
                SG-->>W: Check Execution Result (Latency, Status, TLS, Body)
            end
            W->>DB: Persist CheckResult & Update next_check_at (Drift-free)
        end
    end
```

### 4.1 Atomic Job Claiming (`claim_due_monitors`)
- Stored function defined with `SECURITY DEFINER` and search path restriction (`pg_catalog, public, pg_temp`), owned by restricted non-login role `clientpulse_definer`.
- `EXECUTE` privilege is revoked from `PUBLIC` and `clientpulse_app`, and granted strictly to dedicated login role `clientpulse_worker`.
- Validates parameters strictly inside the function (`1 <= p_batch_size <= 100`, `5 <= p_lease_timeout_seconds <= 3600`, non-empty `p_worker_id`).
- Employs `SELECT ... FOR UPDATE OF m SKIP LOCKED` to allow multiple concurrent worker instances to scale horizontally without duplicate executions.
- Sets an execution lease (`lease_acquired_at = NOW()`, `lease_worker_id = p_worker_id`). Abandoned or crashed jobs whose leases expire are automatically reclaimed by the next available worker.
- Short-lived database transaction: The database connection is held only during the atomic claim query. All network I/O, DNS resolution, and outbound HTTP checks execute outside database transactions.

### 4.2 Drift Prevention & Misfire Policy
- Scheduled check intervals must not drift forward based on slow network latency. The scheduler calculates `next_check_at = scheduled_at + interval`.
- **Misfire Skip Policy:** If a monitor's scheduled time fell significantly behind while the worker was offline or restarting (`next_check_at < NOW()`), the scheduler advances `next_check_at = NOW() + interval` instead of scheduling a stampede of historical catch-up runs.

### 4.3 SSRF Guard & Outbound Connection Pinning
- **DNS Rebinding Defense:** Target hostnames are resolved before connection initiation. Every resolved A and AAAA record is inspected against a comprehensive blacklist of private, loopback, link-local, multicast, CGNAT, and cloud metadata CIDRs.
- **Socket IP Pinning:** The exact public IP address validated by the SSRF guard is passed to `net.Dialer.DialContext`. The outbound TCP connection connects directly to that IP, preventing DNS rebinding race conditions.
- **SNI Preservation:** The original target hostname is preserved in `tls.Config.ServerName` and the HTTP `Host` header for accurate virtual hosting and TLS certificate verification.
- **Redirect Isolation:** The HTTP client validates every HTTP redirect target through the full SSRF pipeline before following. Redirects to internal or prohibited addresses are rejected immediately.
- **Proxy Neutralization:** Custom HTTP transports set `Proxy = nil` to guarantee environment proxy variables do not bypass security controls.

### 4.4 Monitor State Machine & Consecutive Failure Thresholds
- **States:** `pending` (initial), `up` (healthy), `suspected_down` (1 failure), `down` (2+ consecutive failures), `paused` (intentionally disabled), `unknown` (insufficient data).
- **Threshold Rules:**
  - Scheduled check failure with 0 previous failures transitions monitor to `suspected_down` (`consecutive_failures = 1`).
  - Scheduled check failure with 1+ previous failures transitions monitor to `down` (`consecutive_failures = consecutive_failures + 1`).
  - Successful scheduled check transitions monitor to `up` and resets `consecutive_failures = 0`.
  - Manual "Test Now" executions record results as `execution_type = 'manual'` but do not mutate `consecutive_failures` or change scheduled threshold status.

### 4.5 Check Results Retention & Hourly Aggregations
- **Raw Check Results Retention:** Raw entries in `check_results` are retained for a configurable period (`CHECK_RESULT_RETENTION_DAYS`, default 90 days).
- **Bounded Batch Pruning:** Expired check results are deleted in bounded batches (`LIMIT 500`) to avoid table locks and long-running transactions.
- **Hourly Metric Aggregations:** Stored in `check_aggregates_hourly` with rollups for `total_checks`, `successful_checks`, `total_latency_ms`, `min_latency_ms`, and `max_latency_ms`. Aggregates are retained for at least 13 months (400 days) to support historical dashboard reporting and uptime calculation.

---

## 5. Incident Management Architecture

### 5.1 Relational Architecture & Anti-Polymorphism
Incidents avoid untyped polymorphic foreign keys. Each incident explicitly references its target type through nullable, mutually-exclusive foreign keys (`monitor_id`, `heartbeat_monitor_id`) enforced by database check constraint:
```sql
CONSTRAINT chk_incident_target_exclusive CHECK (
    (monitor_id IS NOT NULL AND heartbeat_monitor_id IS NULL) OR
    (monitor_id IS NULL AND heartbeat_monitor_id IS NOT NULL)
)
```
Tenant consistency is strictly enforced at the database level via composite foreign keys:
- `(organization_id, monitor_id) REFERENCES monitors(organization_id, id)`
- `(organization_id, heartbeat_monitor_id) REFERENCES heartbeat_monitors(organization_id, id)`
- `(organization_id, first_failure_result_id) REFERENCES check_results(organization_id, id)`
- `(organization_id, recovery_result_id) REFERENCES check_results(organization_id, id)`

### 5.2 Single Active Incident Constraint & Flapping Prevention
Only one unresolved incident (`status = 'open'`) can exist for a given monitor or heartbeat target at any time. Partial unique indexes enforce this at the storage layer:
- `CREATE UNIQUE INDEX uq_open_incident_monitor ON incidents (monitor_id) WHERE (status = 'open');`
- `CREATE UNIQUE INDEX uq_open_incident_heartbeat ON incidents (heartbeat_monitor_id) WHERE (status = 'open');`

When subsequent failures occur while an incident is already open:
1. The existing open incident is updated with latest diagnostic context.
2. No duplicate incidents are created.
3. No duplicate `incident.opened` notification events are dispatched.

### 5.3 Incident State Transitions & Timestamps
- **Threshold to Open:** 1st consecutive scheduled failure moves monitor to `suspected_down` without opening an incident. The 2nd consecutive failure confirms `down` and atomically creates an incident.
- **Timestamps:**
  - `first_failure_at`: Preserves the exact timestamp of the initial failure cycle.
  - `confirmed_at`: Records when the threshold was confirmed down.
  - `resolved_at`: Recorded on recovery check.
  - `duration_seconds`: Calculated as `EXTRACT(EPOCH FROM (resolved_at - first_failure_at))`.
- **Acknowledgement:** Independent from status. Acknowledging sets `acknowledged_at` and `acknowledged_by` but retains `status = 'open'`. Only successful recovery check sets `status = 'resolved'`.

### 5.4 Persistent Notification Events
Notification events are immutable and append-only (`notification_events` table). Each event uses a deterministic `idempotency_key`:
- `incident_open:<incident_id>`
- `inc_ack:<incident_id>:<timestamp>`
- `incident_resolve:<incident_id>`
- `hb_missed:<heartbeat_id>:<timestamp>`
- `hb_recovered:<heartbeat_id>:<timestamp>`

---

## 6. Heartbeat Monitoring Architecture

### 6.1 Cryptographic Token Security
- **Generation:** 32 bytes of cryptographically secure random bytes generated via `crypto/rand` (64-character hexadecimal token).
- **Storage:** Only the SHA-256 digest (`token_hash`) is stored in PostgreSQL. Plaintext tokens are never stored in the database, never logged, and never returned in GET endpoints.
- **Exposure:** The raw secret token and ready-to-copy snippets are displayed to the user only once immediately following creation or token rotation.
- **Rotation / Revocation:** Rotation creates a new token and immediately replaces the hash, instantly invalidating the previous token. Revocation zeroes the hash and marks the record `revoked`.

### 6.2 Public Ping Endpoint (`POST /api/v1/heartbeat-pings/{token}`)
- Unauthenticated, rate-limited public endpoint.
- Constant-time SHA-256 computation and database lookup via SECURITY DEFINER function `lookup_heartbeat_by_token_hash`.
- Caller-supplied tenant IDs are neither requested nor trusted; the tenant is resolved purely from the matched record.
- Invalid tokens return HTTP 404 to prevent timing and existence oracle enumeration.

### 6.3 Missed Heartbeat Detection & Leases
- Background worker executes periodic scan calling `claim_missed_heartbeats(worker_id, batch_size, lease_seconds)`.
- Condition: `last_received_at + interval + grace_period < NOW()`.
- Claims rows using `FOR UPDATE OF hm SKIP LOCKED` with lease timeout.
- Sets `status = 'missed'` and creates an open incident.
- When a valid ping is received for a missed heartbeat, the heartbeat atomically updates to `healthy`, updates `last_received_at` and `next_expected_at`, resolves the open incident, and emits `heartbeat.recovered`.
