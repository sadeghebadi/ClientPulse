# TASK-001.2: Final Foundation Audit

**Status:** COMPLETED  
**Assignee:** Antigravity  
**Created:** 2026-10-31  
**Target:** Phase 0 Final Audit & Hardening  

---

## 1. Goal and Scope

Execute the final pre-Phase 1 foundation audit and hardening tasks for **ClientPulse** as defined in the product specification and engineering standards:

1. **PostgreSQL Advisory Lock (Connection-Scoped):**
   - Audit `services/api/internal/migrate/` and `services/api/internal/database/`.
   - Ensure `pg_advisory_lock` is acquired on a dedicated physical connection acquired via `pgxpool.Acquire()`.
   - Ensure all migration queries and DDL statements execute through that exact acquired connection.
   - Ensure `pg_advisory_unlock` and connection release execute on the same connection, resilient to failure and context cancellation.
   - Add integration tests for concurrent migration serialization, failure unlock, and subsequent execution.
   - Document the locking architecture in ADR 0001.

2. **Real Worker SIGKILL Crash Test:**
   - Execute an unclean worker process termination via `docker compose kill -s SIGKILL worker`.
   - Do not manually alter database records.
   - Wait past the configured `WORKER_HEARTBEAT_TIMEOUT_SECONDS` (10s).
   - Verify `/health/worker` genuinely returns HTTP 503 due to stale heartbeat.
   - Restart the worker container with `docker compose start worker`.
   - Verify `/health/worker` automatically recovers to HTTP 200.

3. **Runtime & Base Image Modernization Audit:**
   - Review versions for Go toolchain, Alpine Linux, Node.js, and Docker base images.
   - Verify image tag compatibility, stability, and upstream registry status.
   - Document any version constraints and Docker Hub rate limiting realities.

4. **Quality & Regression Testing:**
   - Run `go test -count=1 -race ./...` across all Go packages.
   - Run `go vet ./...`.
   - Run `npm run lint` and `npm run build` on Next.js web application.
   - Validate `docker compose config`.
   - Verify running service endpoints.

5. **Strict Boundary:**
   - Zero Phase 1 features (no authentication, tenant onboarding, or business tables).
   - All tests executed live with real command outputs.

---

## 2. Acceptance Criteria

- [x] Dedicated connection acquired via `pgxpool.Acquire()` for the entire duration of `Up()` and `Down()`.
- [x] Advisory lock acquired and released on the same connection session.
- [x] All migration statements run on the acquired connection.
- [x] Deferred unlock executes with a detached background context to prevent lock leaks on context cancellation.
- [x] Concurrent migration test confirms serialization and zero duplicate executions.
- [x] Advisory lock release verified after intentional migration failure.
- [x] Connection-scoped locking strategy documented in ADR 0001.
- [x] Unclean SIGKILL termination executed on worker container.
- [x] Stale heartbeat (>10s) verified via `/health/worker` returning HTTP 503 without manual DB alteration.
- [x] Worker restart restores `/health/worker` to HTTP 200.
- [x] Comprehensive runtime version audit documented.
- [x] All automated regression tests pass with race detector enabled (`go test -count=1 -race ./...`).
- [x] Frontend lint and build pass with zero warnings or errors (`npm run lint`, `npm run build`).
- [x] All host ports confirmed bound to `127.0.0.1`.
- [x] Comprehensive report created in `task-report/TASK-001.2-report.md`.

