# TASK-001: Phase 0 — Repository Audit & Foundation

**Status:** IN PROGRESS  
**Assignee:** Antigravity  
**Created:** 2026-10-31  
**Target Phase:** Phase 0 — Repository Audit & Foundation  

---

## 1. Goal and Scope

Establish the production-oriented technical and architectural foundation for **ClientPulse** as defined in `PRD.md`. This task prepares the repository structure, Docker environment, Go backend and worker foundations, Next.js frontend foundation, PostgreSQL database integration, migration framework, structured logging, health checks, automated tests, CI workflow, and initial documentation.

No Phase 1 features (such as user authentication, tenant onboarding, client CRUD, or live monitor checks) will be implemented prematurely in this task.

---

## 2. Deliverables

1. **Repository Structure:**
   - `apps/web/`: Next.js + TypeScript frontend foundation with Tailwind CSS and RTL/i18n foundation.
   - `services/api/`: Go backend and worker foundation (Go 1.23).
     - `cmd/api/`: REST API entrypoint.
     - `cmd/worker/`: Background worker entrypoint.
     - `cmd/migrate/`: Database migration CLI.
     - `internal/config/`: Configuration loading with environment validation.
     - `internal/logger/`: Structured JSON logging (`log/slog`) with Request ID propagation.
     - `internal/database/`: PostgreSQL connection pool (`jackc/pgx/v5`).
     - `internal/server/`: HTTP server, routing, security middlewares.
     - `internal/health/`: Liveness, readiness, and worker health endpoints.
     - `internal/migrate/`: Versioned SQL migration runner.
   - `migrations/`: SQL migration files (`.up.sql` and `.down.sql`).
   - `deploy/`:
     - `deploy/docker/`: Multi-stage Dockerfiles for API, Worker, and Web.
     - `deploy/caddy/`: Caddyfile reverse proxy configuration.
     - `deploy/scripts/`: Helper scripts for development and migrations.
   - `docs/`: Architecture Decision Record (`docs/adr/0001-modular-monolith-foundation.md`), architecture overview, API guide, security, and operations documentation.
   - `.github/workflows/ci.yml`: Basic GitHub Actions CI workflow for linting, building, and running automated tests.

2. **Docker Compose & Environment:**
   - `docker-compose.yml`: Local multi-service environment (`postgres`, `api`, `worker`, `web`, `mailpit`).
   - `.env.example`: Non-secret environment template covering all services with sensible defaults.
   - Safe port configuration avoiding conflicts with host services (PostgreSQL mapped to `5433:5432` on host).

3. **Database & Migration Framework:**
   - PostgreSQL schema version tracking (`schema_migrations`).
   - Up and Down migration CLI and programmatic execution.
   - Initial foundational migration (`000001_foundation.up.sql`).

4. **Health Endpoints & Observability:**
   - `GET /health/live`: Process liveness check.
   - `GET /health/ready`: Database connectivity check.
   - `GET /health/worker`: Worker status check.
   - Structured JSON logging with request tracing and sensitive data exclusion.

5. **Automated Tests:**
   - Go unit and integration tests for configuration, logging, database ping, migrations, middlewares, and health endpoints.
   - Frontend build and lint verification.

6. **Documentation & Task Report:**
   - `README.md` with verified, step-by-step setup and execution instructions.
   - `task-report/TASK-001-report.md` capturing all executed commands, test outputs, verified completion criteria, and next steps.

---

## 3. Acceptance Criteria

- [ ] Directory layout matches PRD Section 9.3.
- [ ] Architecture Decision Record (ADR) created.
- [ ] Go backend and worker compile cleanly.
- [ ] Next.js frontend builds and lints cleanly.
- [ ] Database migration framework applies and rolls back migrations successfully.
- [ ] Docker Compose starts all services (`postgres`, `api`, `worker`, `web`).
- [ ] PostgreSQL container reports healthy.
- [ ] Backend `/health/live` and `/health/ready` endpoints return HTTP 200 with valid JSON.
- [ ] All automated tests pass.
- [ ] README.md documents actual working commands.
- [ ] Completion report filed in `task-report/TASK-001-report.md`.

