# TASK-001.1: Foundation Verification & Hardening

**Status:** IN PROGRESS  
**Assignee:** Antigravity  
**Created:** 2026-10-31  
**Target:** Phase 0 Hardening & Verification  

---

## 1. Goal and Scope

Perform final technical verification and security/operational hardening for Phase 0 of **ClientPulse** before requesting authorization for Phase 1.

Key objectives:
1. **Genuine Worker Health Verification:** Replace static API worker health response with a persistent PostgreSQL-backed heartbeat mechanism, configurable stale threshold, automated tests for all states, and real container shutdown/restart validation.
2. **Docker Network Security:** Bind host-exposed ports (PostgreSQL, Mailpit, API, Web) strictly to `127.0.0.1` to prevent unauthorized external network access.
3. **Secrets & Environment Audit:** Verify `.env` isolation, credential absence in `.env.example`, absence of hardcoded secrets, and sensitive field log redaction.
4. **Docker Reproducibility:** Pin explicit image versions across `docker-compose.yml` and Dockerfiles (eliminate unpinned `:latest` tags).
5. **Database Migration Robustness:** Verify transactional rollback on failed migrations, migration locking, and clean connection pool management.
6. **Automated Quality Checks:** Execute `go test -race ./...`, `go vet ./...`, `npm run lint`, `npm run build`, and docker compose runtime validation.
7. **Reporting:** Record all actual commands, outputs, and runtime test results in `task-report/TASK-001.1-report.md`.

No Phase 1 features (auth, organizations, tenant isolation logic, or monitors) will be implemented.

---

## 2. Acceptance Criteria

- [ ] Worker updates a persistent heartbeat in PostgreSQL (`worker_heartbeats`).
- [ ] API `/health/worker` queries the database for active worker heartbeats.
- [ ] API returns HTTP 200 with heartbeat metadata when worker is active.
- [ ] API returns HTTP 503 with clear diagnostic info when worker heartbeat is stale or missing.
- [ ] Automated tests cover healthy, stale, and missing worker heartbeat states.
- [ ] Real runtime test executed: worker healthy -> worker container stopped -> heartbeat expires -> API reports 503 -> worker container restarted -> API reports 200.
- [ ] `docker-compose.yml` binds host ports to `127.0.0.1` only.
- [ ] Docker image references are pinned to explicit tags.
- [ ] Migration engine rollback and atomicity verified by automated test.
- [ ] `go test -race ./...` and `go vet ./...` pass with zero failures or race conditions.
- [ ] `npm run lint` and `npm run build` pass with zero errors.
- [ ] Complete report documented in `task-report/TASK-001.1-report.md`.

