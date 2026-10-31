# ADR 0001: Modular Monolith Foundation and Technology Stack

**Date:** 2026-10-31  
**Status:** Accepted  
**Deciders:** Core Engineering Team  
**Consulted:** PRD.md (v1.0)  

---

## Context

ClientPulse is a multi-tenant SaaS platform designed for web agencies to monitor client websites, track maintenance activities, manage incidents, and generate branded care reports.

The project requires high reliability, multi-tenant security, clean architecture, low operating costs ($10–20/month initial budget), and maintainability without introducing unnecessary distributed systems complexity during the initial phases.

## Decisions

### 1. Architectural Pattern: Modular Monolith with Separate Worker Process
- **Decision:** Build the backend as a **Modular Monolith** in Go. Both the REST API server and the background worker share domain logic and data access packages within `services/api/`, but run as separate processes / entrypoints (`cmd/api` and `cmd/worker`).
- **Rationale:** Microservices introduce excessive operational overhead, network latency, deployment complexity, and distributed transaction challenges for early-stage SaaS. A modular monolith provides clear module boundaries while allowing single-repository simplicity. Running the worker as an independent process ensures intensive monitoring jobs or PDF generation do not starve HTTP request processing.
- **Alternatives Considered:**
  - *Microservices:* Rejected per PRD Section 9.1 ("در MVP از Microservices استفاده نکن").
  - *Unified single process (API + Worker in one runtime):* While simpler, it risks CPU/memory spikes during batch checks or PDF generation impacting API latency and availability.

### 2. Backend Language: Go (1.23+)
- **Decision:** Standardize backend services and background monitoring workers on **Go**.
- **Rationale:**
  - Excellent concurrency primitives (goroutines and channels) essential for concurrent HTTP/SSL/DNS checks and heartbeat monitoring.
  - Native compilation to lightweight, statically linked binaries with minimal container footprint and memory consumption.
  - Built-in structured logging (`log/slog`) and HTTP routing improvements in recent Go releases.
  - High performance, fast startup time, and strict type safety.

### 3. Frontend Framework: Next.js (TypeScript) & Tailwind CSS
- **Decision:** Next.js (App Router) with TypeScript, Tailwind CSS, and built-in i18n support.
- **Rationale:**
  - Full TypeScript type safety across UI components, forms, and API clients.
  - Server-side rendering (SSR) and client-side transitions for an optimal agency dashboard experience.
  - Native support for internationalization (English & Persian) with proper bidirectional (RTL) styling.

### 4. Database & Queue Strategy: PostgreSQL with `FOR UPDATE SKIP LOCKED`
- **Decision:** Use **PostgreSQL 16** as the single source of truth for persistent data, and leverage PostgreSQL's `FOR UPDATE SKIP LOCKED` for task dispatching in the initial phase.
- **Rationale:**
  - PRD Section 9.2: Redis is not mandatory for MVP. Scheduling jobs via PostgreSQL simplifies infrastructure, reduces VPS hosting costs, and provides transactional integrity between data updates and queue state.
  - PostgreSQL row-level locking with `SKIP LOCKED` scales effectively to hundreds of concurrent jobs without queue worker contention.
  - The job queue interface is abstracted so Redis (e.g., Asynq) can be introduced seamlessly if load testing indicates a database queue bottleneck in future phases.

### 5. Migration Strategy: Versioned SQL Migrations with Connection-Scoped Advisory Locking
- **Decision:** Employ a dedicated SQL migration runner executing versioned `.up.sql` and `.down.sql` scripts, tracking executed versions in `schema_migrations`, protected by connection-scoped PostgreSQL session advisory locking (`pg_advisory_lock(982347102)`).
- **Locking Strategy Details:**
  - Because `pgxpool` manages a dynamic pool of underlying database connections and PostgreSQL advisory locks are session-scoped (tied to a specific physical connection session), the migration runner explicitly calls `pool.Acquire(ctx)` to reserve a single dedicated connection (`*pgxpool.Conn`).
  - Both `pg_advisory_lock` and `pg_advisory_unlock` are executed on that exact same acquired connection session.
  - All transactional migration DDL/DML statements (`conn.Begin(ctx)`) are executed through that same acquired connection.
  - Unlock and connection release are guaranteed via `defer` using a detached background context to ensure lock cleanup cannot be bypassed by client context cancellation or migration failure.
  - Serializes concurrent migration attempts and prevents duplicate execution or partial schema state.

### 6. Observability: Structured JSON Logging & Semantic Health Checks
- **Decision:** Go standard library `log/slog` outputting JSON with contextual Request ID injection. Standardized health endpoints:
  - `/health/live`: Process liveness probe.
  - `/health/ready`: Database connectivity readiness probe.
  - `/health/worker`: Background worker loop probe.
- **Rationale:** Standard cloud-native health patterns prevent premature traffic routing during startup and enable reliable container orchestrator restarts. Sensitive data (tokens, passwords, response bodies) is strictly excluded from logs.

---

## Consequences

- **Positive:**
  - Fast developer feedback loop and low resource utilization.
  - Simple single-command local development with Docker Compose.
  - Production-ready security foundations (SSRF guard, tenant context scoping, secret isolation).
  - Clear evolutionary path to scale components independently if needed.
- **Negative / Trade-offs:**
  - Developers must adhere to module boundaries within `services/api/` and avoid circular package dependencies.
  - High worker throughput must be monitored to ensure PostgreSQL query performance remains optimal before scaling to a dedicated broker.

