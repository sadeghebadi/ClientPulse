# ClientPulse Architecture Overview

## 1. System Topology

ClientPulse is designed as a **Modular Monolith** with a decoupled background worker process, communicating over a centralized PostgreSQL data store.

```mermaid
graph TD
    User([Agency User / Client Browser]) -->|HTTPS / Static UI| Web[Next.js Frontend (apps/web)]
    User -->|REST API / HTTPS| Caddy[Caddy / Reverse Proxy]
    Caddy -->|/api/* & /health/*| API[Go API Service (services/api/cmd/api)]
    Caddy -->|/*| Web
    
    API -->|Read / Write Pool| Postgres[(PostgreSQL 16)]
    Worker[Go Worker Process (services/api/cmd/worker)] -->|Poll Jobs / FOR UPDATE SKIP LOCKED| Postgres
    Worker -->|Outbound Health / SSL / DNS Checks| ExternalSites[Target Websites / Endpoints]
    Cron[External Client Crons / Backup Scripts] -->|POST /api/v1/heartbeat-pings/:token| API
```

## 2. Repository Layout

```text
clientpulse/
├── apps/
│   └── web/                     # Next.js 14+ (App Router), TypeScript, Tailwind CSS
├── services/
│   └── api/                     # Go 1.23+ Modular Monolith (API & Worker)
│       ├── cmd/
│       │   ├── api/             # HTTP API entrypoint
│       │   ├── worker/          # Background worker entrypoint
│       │   └── migrate/         # Database migration CLI
│       └── internal/
│           ├── config/          # Environment configuration loading & validation
│           ├── database/        # Connection pool (pgxpool) & ping
│           ├── health/          # Live/Ready/Worker health handlers
│           ├── logger/          # Structured JSON logging with slog
│           ├── migrate/         # SQL migration engine
│           └── server/          # HTTP server, routing, security middlewares
├── migrations/                  # Versioned SQL migrations (.up.sql / .down.sql)
├── deploy/
│   ├── docker/                  # Dockerfiles for api, worker, and web
│   ├── caddy/                   # Caddy reverse proxy configuration
│   └── scripts/                 # Utility scripts (dev, migrate, backup)
├── docs/                        # Architecture, ADRs, API, security, operations
├── task/                        # Implementation task tracking
├── task-report/                 # Phase execution reports
├── docker-compose.yml           # Local dev orchestrator
├── .env.example                 # Environment variables specification
└── README.md                    # Setup and execution guide
```

## 3. Core Modules (services/api/internal/)

| Module | Responsibility |
|---|---|
| `config` | Type-safe environment loading and validation |
| `logger` | Structured JSON logging using standard `log/slog` with correlation ID |
| `database` | PostgreSQL connection pooling via `jackc/pgx/v5` |
| `migrate` | Versioned schema migration runner |
| `server` | HTTP server lifecycle, routing, and standard middlewares |
| `health` | Liveness (`/health/live`), Readiness (`/health/ready`), Worker (`/health/worker`) probes |
| *(Phases 1-5)* | `auth`, `tenancy`, `clients`, `assets`, `monitors`, `incidents`, `heartbeats`, `maintenance`, `reports`, `notifications`, `billing`, `admin` |

## 4. Multi-Tenant Data Isolation Strategy

- Every tenant-scoped entity contains an `organization_id` foreign key.
- Tenant context is strictly determined by verified backend session credentials, never accepted from client-supplied request bodies.
- Cross-tenant queries are blocked at the repository layer through mandatory organization context parameters.

