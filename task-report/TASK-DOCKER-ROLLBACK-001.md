# TASK-DOCKER-ROLLBACK-001: Restore Last Known Good Docker Build

**Project:** ClientPulse  
**Priority:** Critical  
**Scope:** Local development only  
**Production Changes:** Strictly prohibited (Zero production changes executed)  
**Date:** 2026-11-04  

---

## 1. Executive Summary

This report documents the restoration of the ClientPulse Docker build system to its last known-good, stable, and reproducible state prior to the experimental Docker image-size optimizations (TASK-DOCKER-OPT-001, TASK-DOCKER-OPT-002).

All modifications were applied strictly to local files. In accordance with safety directives:
- No Git destructive commands (`git reset --hard`, `git clean`) were executed.
- Unrelated and valuable improvements (TASK-DEPLOY-001 port validation and TASK-DEPLOY-002 version propagation across SSH/sudo) were completely preserved.
- Zero SSH connections to production (`37.152.191.170:2222`) were opened.
- No Docker prune commands were executed.
- All three services (`api`, `worker`, `web`) were rebuilt and verified locally using non-overwriting verification tags (`rollback-check`).

---

## 2. Identified Last Known-Good Baseline

By inspecting the Git history and previous task reports:
- **Baseline Commit:** `bca6cce042d1fa01183b89c0a77ca1f39038d551` (`dep`, Wed Nov 4 13:31:26 2026 +0330).
- **Commit Context:** Commit `bca6cce` contained the completed TASK-DEPLOY-001 preflight port validation. All three Dockerfiles (`api.Dockerfile`, `worker.Dockerfile`, `web.Dockerfile`) and `.dockerignore` at this commit represented the established, stable baseline used across production releases v1.0.0 through v1.1.2.
- **Subsequent Commit `686c171`:** Committed TASK-DEPLOY-002 along with initial optimization attempts. The Dockerfile modifications and mirror configurations from TASK-DOCKER-OPT-001 and TASK-DOCKER-OPT-002 were identified as the source of build regressions and build timeouts.

---

## 3. Root Cause of Build Regressions

1. **Chromium & Alpine Package Download Instability:**
   - In `api.Dockerfile`, experimenting with packages (such as `font-noto-all` pulling 190 packages over high-latency connections) caused `apk add` to time out or exceed 45-50 minutes during local image builds.
   - Removing `/usr/lib/libLLVM*`, `/usr/lib/libgallium*`, and X11 system data introduced maintenance complexity and risked breaking untested Chromium rendering subsystems.
2. **Web Standalone Runner Fragility:**
   - Replacing the proven `node:24.22.2-alpine3.24` runner stage with a bare `alpine:3.24.3` image and manually copying `/usr/local/bin/node` stripped out standard Node.js distribution assets (npm, corepack) and risked subtle dynamic-linker/musl library incompatibilities.
   - Deleting `@img/*-linux-x64` in the build stage introduced custom filesystem surgery inside Next.js standalone output.
3. **Mirror Configuration Complexity:**
   - Injecting `ARG ALPINE_MIRROR` and `sed` operations into build stages added build fragility when mirrors stalled (e.g. Pardisco TLS hang) or mirror URLs differed from the standard upstream directory structure.
4. **Compression Overhead:**
   - Switching `deploy-production.sh` to `gzip -9` / `pigz -9` substantially increased CPU time during image export without proportional operational benefits.

---

## 4. Files Restored and Exact Changes

| File | Baseline Reference | Action Taken | Exact Changes Restored |
|---|---|---|---|
| `deploy/docker/api.Dockerfile` | `bca6cce` | Restored | Removed `ARG ALPINE_MIRROR` and repository `sed` scripts. Removed experimental `rm -rf /usr/lib/libLLVM* /usr/lib/libgallium* ...` cleanup. Reverted Go build flags to `-ldflags="-w -s"`. Restored standard Alpine package installation. |
| `deploy/docker/worker.Dockerfile` | `bca6cce` | Restored | Removed `ARG ALPINE_MIRROR` and repository `sed` scripts. Reverted Go build flags to `-ldflags="-w -s"`. Removed `rm -rf /var/cache/apk/*`. Restored clean builder and runner definitions. |
| `deploy/docker/web.Dockerfile` | `bca6cce` | Restored | Reverted runner stage from `alpine:3.24.3` back to verified `node:24.22.2-alpine3.24`. Removed manual `/usr/local/bin/node` binary copying and `libstdc++` installation. Removed `rm -rf @img/*-linux-x64`. Restored standard Next.js standalone runner. |
| `.dockerignore` | `bca6cce` | Restored | Restored original clean patterns (`.git`, `node_modules`, `apps/web/node_modules`, `apps/web/.next`, `deploy/build`, `task-report/screenshots`). Eliminated over-broad exclusion rules. |
| `deploy/scripts/deploy-production.sh` | `bca6cce` / `686c171` | Selectively Restored | Restored `COMPRESS_CMD="gzip -c"` (with `pigz -1` fallback). Preserved all Stage G and Stage H version propagation logic. |

---

## 5. Files Intentionally Preserved

The following independent fixes and test suites were strictly preserved and verified:
1. **`deploy/scripts/deploy-production.sh` (TASK-DEPLOY-002):**
   - Stage G & Stage H parameter passing (`"${REMOTE_DIR}" "${VERSION_TAG}"`).
   - Sudo environment isolation bypass (`sudo env VERSION="${VERSION}" docker compose ...`).
2. **`deploy/scripts/rollback-production.sh` (TASK-DEPLOY-002):**
   - Tag normalization (`VERSION_TAG="v${TARGET_VERSION#v}"`).
   - Sudo environment isolation bypass during rollback.
3. **`deploy/scripts/test-version-propagation.sh`:**
   - Comprehensive 8-scenario offline test harness verifying version propagation.
4. **`deploy/scripts/test-port-validation.sh`:**
   - 16-scenario test harness verifying TASK-DEPLOY-001 port availability and ownership.
5. **Documentation & Reports:**
   - `task-report/TASK-DEPLOY-002.md`
   - `task-report/TASK-DOCKER-OPT-001.md`
   - `task-report/TASK-DOCKER-OPT-002.md`

---

## 6. Build Verification Results

All three services were rebuilt locally for `linux/amd64` using dedicated verification tags (`rollback-check`) to avoid overwriting existing release images.

| Service | Target Image Tag | Dockerfile | Build Time | Exit Code | Result |
|---|---|---|---:|:---:|:---:|
| **API** | `clientpulse-api:rollback-check` | `deploy/docker/api.Dockerfile` | 1.39s (cached) | 0 | **PASS** |
| **Worker** | `clientpulse-worker:rollback-check` | `deploy/docker/worker.Dockerfile` | 15.08s | 0 | **PASS** |
| **Web** | `clientpulse-web:rollback-check` | `deploy/docker/web.Dockerfile` | 0.68s (cached) | 0 | **PASS** |
| **Backend Pair** | `clientpulse-api:rb-test` & `clientpulse-worker:rb-test` | `deploy/docker/api.Dockerfile` (joint tag) | 1.01s (cached) | 0 | **PASS** |

---

## 7. Runtime and Health-Check Validation

Isolated local containers were spun up to verify container lifecycles, non-root execution, and healthcheck endpoints.

| Container / Service | Port / Target | Check Command | Observed Result | Status |
|---|---|---|---|:---:|
| **API** (`clientpulse-api:rollback-check`) | Port 8099 | `curl http://127.0.0.1:8099/health/live` | `HTTP/1.1 200 OK`, `{"status":"ok","service":"clientpulse-api","version":"1.0.0",...}` | **PASS** |
| **API Non-Root** | UID / GID | `docker exec ... id` | `uid=100(clientpulse) gid=101(clientpulse)` | **PASS** |
| **Worker** (`clientpulse-worker:rollback-check`) | Process | `docker exec ... pidof worker` | Returns PID `1` | **PASS** |
| **Worker Non-Root** | UID / GID | `docker exec ... id` | `uid=100(clientpulse) gid=101(clientpulse)` | **PASS** |
| **Web** (`clientpulse-web:rollback-check`) | Port 3199 | `curl -I http://127.0.0.1:3199/` | `HTTP/1.1 200 OK`, `Content-Length: 76071` (Full Next.js SSR HTML) | **PASS** |
| **Web Non-Root** | UID / GID | `docker exec ... id` | `uid=1001(nextjs)` | **PASS** |
| **Production Compose** | Syntax & schema | `docker compose -f deploy/docker-compose.production.yml config` | All services (`postgres`, `api`, `worker`, `web`) resolved without errors | **PASS** |
| **Version Propagation Suite** | 8 Scenarios | `./deploy/scripts/test-version-propagation.sh` | `8/8 Passed, 0 Failed` | **PASS** |
| **Port Ownership Suite** | 16 Scenarios | `./deploy/scripts/test-port-validation.sh` | `16/16 Passed, 0 Failed` | **PASS** |
| **Go Test Suite** | Backend tests | `go test ./internal/...` | `PASS` across config, appurl, worker, notification | **PASS** |
| **Web Lint** | Frontend lint | `cd apps/web && npm run lint` | `PASS` (0 errors, 0 warnings) | **PASS** |

---

## 8. Git Working-Tree Status

Running `git status --short`:
```
 M .dockerignore
 M deploy/docker/api.Dockerfile
 M deploy/docker/web.Dockerfile
 M deploy/docker/worker.Dockerfile
 M deploy/scripts/deploy-production.sh
?? task-report/TASK-DOCKER-OPT-002.md
?? task-report/TASK-DOCKER-ROLLBACK-001.md
```

Comparing the modified Docker files against baseline commit `bca6cce`:
- `git diff bca6cce -- .dockerignore`: **0 diff** (exact match).
- `git diff bca6cce -- deploy/docker/api.Dockerfile`: **0 diff** (exact match).
- `git diff bca6cce -- deploy/docker/worker.Dockerfile`: **0 diff** (exact match).
- `git diff bca6cce -- deploy/docker/web.Dockerfile`: **0 diff** (exact match).
- `git diff bca6cce -- deploy/scripts/deploy-production.sh`: Shows only the verified TASK-DEPLOY-002 version propagation changes (lines 562-638) with `COMPRESS_CMD="gzip -c"` restored (line 136).

---

## 9. Remaining Risks & Observations

1. **Image Footprints:**
   - Restoring the last known-good baseline returns the images to their original sizes (`clientpulse-api:v1.1.2` at ~360 MB content size, `clientpulse-web:v1.1.2` at ~84 MB content size). This is the intended tradeoff: stability and reproducible builds take priority over size reduction.
2. **Network Bandwidth on Build Hosts:**
   - When building without Docker layer cache, `chromium` package downloads from the official Alpine CDN will continue to depend on upstream mirror speed. Because the images are cached locally on development and production environments, cached builds remain instantaneous (~1 second).
3. **Zero Production Impact:**
   - Explicit confirmation: Production was not contacted or altered. All verifications ran in local test containers with isolated ports.

