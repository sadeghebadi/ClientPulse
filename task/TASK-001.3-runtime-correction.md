# TASK-001.3: Runtime Versions Correction

**Status:** COMPLETED  
**Assignee:** Antigravity  
**Created:** 2026-10-31  
**Target:** Phase 0 Runtime Version Modernization  

---

## 1. Goal and Scope

Correct the runtime version statements and update the codebase configurations to use the official, current stable releases:
- **Go:** 1.27.3 (released October 19, 2026). Builder image: `golang:1.27.3-alpine3.24`.
- **Alpine Linux:** 3.24.3. Runtime runner image: `alpine:3.24.3`.
- **Node.js:** 24.22.1 LTS. Builder and runner image: `node:24.22.1-alpine3.24`.
- **PostgreSQL:** 16.15 on Alpine 3.24. Image: `postgres:16.15-alpine3.24`.

Specific objectives:
1. **Dockerfiles & Compose:**
   - Update `deploy/docker/api.Dockerfile` to `golang:1.27.3-alpine3.24` and `alpine:3.24.3`.
   - Update `deploy/docker/worker.Dockerfile` to `golang:1.27.3-alpine3.24` and `alpine:3.24.3`.
   - Update `deploy/docker/web.Dockerfile` to `node:24.22.1-alpine3.24` for both builder and runner.
   - Update `docker-compose.yml` to use immutable pinned tag `postgres:16.15-alpine3.24`.
2. **Go Configuration:**
   - Update `services/api/go.mod` to `go 1.27.3`.
   - Validate Go build inside the actual `golang:1.27.3-alpine3.24` builder container.
3. **Registry Image Verification:**
   - Inspect official image manifests and digests using `docker buildx imagetools inspect`.
   - Record exact image digests and verify `linux/amd64` architecture compatibility.
4. **CI Workflow:**
   - Update `.github/workflows/ci.yml` with Go 1.27.3 and Node 24.
5. **Full Build & Runtime Execution:**
   - Run `go test -count=1 -race ./...` and `go vet ./...`.
   - Run `npm run lint` and `npm run build`.
   - Execute `docker compose build` and `docker compose up -d`.
   - Verify internal runtime versions (`cat /etc/os-release`, `go version`, `node -v`, `postgres --version`).
   - Verify health probes.
6. **Reporting:**
   - Document corrections, official references, digests, and outputs in `task-report/TASK-001.3-report.md`.

---

## 2. Acceptance Criteria

- [x] `deploy/docker/api.Dockerfile` updated to `golang:1.27.3-alpine3.24` and `alpine:3.24.3`.
- [x] `deploy/docker/worker.Dockerfile` updated to `golang:1.27.3-alpine3.24` and `alpine:3.24.3`.
- [x] `deploy/docker/web.Dockerfile` updated to `node:24.22.1-alpine3.24`.
- [x] `docker-compose.yml` updated to `postgres:16.15-alpine3.24`.
- [x] `services/api/go.mod` updated to `go 1.27.3`.
- [x] `.github/workflows/ci.yml` updated to Go 1.27.3 and Node 24.
- [x] Registry inspection executed via `docker buildx imagetools inspect` and digests documented.
- [x] Containers built and running with the verified images.
- [x] Running container runtime versions inspected and verified inside containers.
- [x] All automated tests pass (`go test -count=1 -race ./...`, `go vet ./...`, `npm run lint`, `npm run build`).
- [x] All health endpoints responding with HTTP 200.
- [x] Comprehensive report created in `task-report/TASK-001.3-report.md`.

