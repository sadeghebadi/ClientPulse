# ClientPulse

> Multi-tenant website monitoring and client care reporting SaaS for web agencies.

Based on **PRD v1.0** (`PRD.md`).

---

## 1. Architecture Overview

ClientPulse is built as a **Modular Monolith** in Go with a decoupled background worker and a Next.js (TypeScript) agency dashboard, backed by PostgreSQL.

- **Frontend:** Next.js 16 (App Router), React 19.2, TypeScript, Tailwind CSS, English & Persian (RTL) foundation (`apps/web`) running on Node.js 24 LTS (`node:24.22.2-alpine3.24`).
- **Backend API:** Go 1.27+ modular REST API service (`services/api/cmd/api`) built on `golang:1.27.3-alpine3.24` and running on `alpine:3.24.3`.
- **Worker:** Go 1.27+ background worker daemon for monitoring checks and report generation (`services/api/cmd/worker`) running on `alpine:3.24.3`.
- **Database:** PostgreSQL 16.15 (`postgres:16.15-alpine3.24`) with versioned transactional migrations (`migrations/`).
- **Dev Mailer:** Mailpit for local email testing without external credentials (`axllent/mailpit:v1.33.0`).

Full architectural details:
- [Architecture Overview](docs/architecture.md)
- [ADR 0001: Modular Monolith Foundation](docs/adr/0001-modular-monolith-foundation.md)
- [API Specification](docs/api.md)
- [Security Guidelines](docs/security.md)
- [Operations & Runbook](docs/operations.md)

---

## 2. Prerequisites

- **Docker:** Engine 24+ and Docker Compose v2.20+
- **Go:** 1.27+ (or Go 1.23+ with toolchain auto-switching for local host builds and tests)
- **Node.js:** 24+ LTS (or Node 22+) and npm (for frontend development)
- **curl:** for health endpoint verification

---

## 3. Quickstart (Docker Compose)

### Step 1: Clone and Configure Environment

```bash
git clone <repo-url> clientpulse
cd clientpulse

# Copy development environment variables template
cp .env.example .env
```

### Step 2: Start All Services

```bash
docker compose up -d --build
```

This starts:
- `clientpulse_postgres` on host port `5433` (mapped internally from `5432` to avoid host conflicts)
- `clientpulse_api` on `http://localhost:8080`
- `clientpulse_worker` (background daemon)
- `clientpulse_web` on `http://localhost:3000`
- `clientpulse_mailpit` on `http://localhost:8025` (Web UI) and port `1025` (SMTP)

### Step 3: Run Database Migrations

```bash
./deploy/scripts/migrate.sh up
```

To verify migration status:
```bash
./deploy/scripts/migrate.sh status
```

To roll back the last migration:
```bash
./deploy/scripts/migrate.sh down
```

### Step 4: Verify Health Probes

```bash
# Liveness probe
curl -i http://localhost:8080/health/live

# Readiness probe (verifies database connectivity)
curl -i http://localhost:8080/health/ready

# Worker probe
curl -i http://localhost:8080/health/worker

# Base API v1 info
curl -i http://localhost:8080/api/v1/info
```

Open the web frontend in your browser:
👉 [http://localhost:3000](http://localhost:3000)

Mailpit inbox for local emails:
👉 [http://localhost:8025](http://localhost:8025)

---

## 4. Local Development (Without Docker for API/Web)

If running services directly on your host machine:

### Backend API & Worker
```bash
cd services/api

# Run unit tests
go test -v ./...

# Run API server locally
go run ./cmd/api

# Run background worker locally
go run ./cmd/worker

# Run migration CLI locally
go run ./cmd/migrate up
```

### Frontend Web
```bash
cd apps/web

# Install dependencies
npm install

# Run linter
npm run lint

# Build standalone production bundle
npm run build

# Start local dev server
npm run dev
```

---

## 5. Automated Tests & CI

GitHub Actions workflow is located at `.github/workflows/ci.yml`.

Run tests locally:
```bash
# Backend tests
(cd services/api && go test -v -race ./...)

# Frontend lint & build
(cd apps/web && npm run lint && npm run build)

# Validate Docker Compose configuration
docker compose config
```

---

## 6. Directory Structure

```text
clientpulse/
├── apps/
│   └── web/                     # Next.js 16 frontend (TypeScript, Tailwind, EN/FA RTL)
├── services/
│   └── api/                     # Go backend & worker
│       ├── cmd/
│       │   ├── api/             # HTTP API entrypoint
│       │   ├── worker/          # Monitoring execution & heartbeat daemon
│       │   └── migrate/         # Migration runner CLI
│       └── internal/
│           ├── auth/            # Argon2id password hashing, tokens, HMAC pepper
│           ├── checker/         # HTTP, SSL, JSON Pointer, DNS check engine
│           ├── config/          # Environment configuration loading & validation
│           ├── database/        # PostgreSQL connection pool (pgxpool)
│           ├── health/          # Liveness, readiness, worker health handlers
│           ├── logger/          # Structured JSON logging (slog) with Request ID
│           ├── migrate/         # SQL migration engine
│           ├── repository/      # RLS-enforced repositories (Client, Asset, Monitor, Results, Heartbeats, Incidents, Notification Events)
│           ├── server/          # HTTP server, routing, RBAC, CSRF, auth & verified email middlewares
│           ├── service/         # Auth & Organization domain logic
│           ├── ssrf/            # Dedicated SSRF guard, DNS inspection & IP pinning
│           ├── tenant/          # Tenant and user context helpers
│           └── worker/          # Distributed scheduler, atomic job claiming & drift prevention
├── migrations/                  # Versioned SQL migrations (.up.sql / .down.sql)
├── deploy/
│   ├── docker/                  # Multi-stage Dockerfiles (api, worker, web)
│   ├── caddy/                   # Caddy reverse proxy configuration
│   └── scripts/                 # dev.sh and migrate.sh helpers
├── docs/                        # Architecture, ADRs, API, security, operations
├── task/                        # Phase task definition documents
├── task-report/                 # Phase execution and completion reports
├── docker-compose.yml           # Local dev multi-container environment
├── .env.example                 # Non-secret environment specification
└── README.md                    # Setup and execution guide
```

---

## 7. Current Project Phase

- **Phase 0 — Repository Audit & Foundation:** ✅ COMPLETE
- **Phase 1 — Auth & Multi-Tenancy:** ✅ COMPLETE
- **Phase 2 — Clients, Assets & Monitoring Core (TASK-003 & TASK-003.1):** ✅ COMPLETE & AUDITED
- **Phase 3 — Incidents & Heartbeats (TASK-004):** ✅ COMPLETE & AUDITED
- **Phase 4 — Maintenance Activity, Reports, PDF & White Label (TASK-005):** ✅ COMPLETE & VERIFIED
- **Phase 5 — Notifications & Alerts:** ⏳ NEXT UPCOMING PHASE

