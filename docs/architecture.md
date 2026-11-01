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
| `tenant` | TenantContext, UserContext, and RBAC permission evaluation (`HasRole`) |
