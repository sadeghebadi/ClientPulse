# TASK-DEPLOY-002: Production Release Version Propagation Report

**Project:** ClientPulse  
**Priority:** Critical  
**Scope:** Local deployment script and rollback tool  
**Targets:** `deploy/scripts/deploy-production.sh`, `deploy/scripts/rollback-production.sh`  
**Production Changes:** None (Local validation only)

---

## 1. Root Cause Analysis

In `deploy/scripts/deploy-production.sh`, the local preflight inspection (Stage A) correctly evaluated Docker Compose using `VERSION="${VERSION_TAG}"`. However, during remote SSH execution stages:
1. Stage G (Database initialization & safe migrations) and Stage H (Application startup & health verification) only passed `"${REMOTE_DIR}"` as an argument to `ssh bash -s`.
2. The remote Docker Compose commands relied on `--env-file "${REMOTE_DIR}/.env"`.
3. The production `.env` intentionally does not define `VERSION` (to avoid persistent version pinning or stale configuration files across releases).
4. `deploy/docker-compose.production.yml` specifies image tags with a fallback:
   - `clientpulse-api:${VERSION:-v1.0.0}`
   - `clientpulse-worker:${VERSION:-v1.0.0}`
   - `clientpulse-web:${VERSION:-v1.0.0}`
5. Because `VERSION` was not exported into the remote environment and `sudo` resets environment variables, Compose silently defaulted to `v1.0.0` for all remote operations. Consequently, migrations and application containers ran against `v1.0.0` even after newer release images (e.g. `v1.1.1`) were built, transferred, and loaded.

---

## 2. Files Changed

1. **`deploy/scripts/deploy-production.sh`**:
   - Updated remote SSH invocations in Stage G and Stage H to pass `"${VERSION_TAG}"` as argument `$2`.
   - Updated all remote `docker compose` commands (PostgreSQL startup, `ps -q` health checks, migration `run`, application `up -d`, and service status `ps`) to explicitly supply `sudo env VERSION="${VERSION}" docker compose ...`.

2. **`deploy/scripts/rollback-production.sh`**:
   - Normalized `TARGET_VERSION` to `VERSION_TAG="v${TARGET_VERSION#v}"` so target releases are consistently identified with their archive directories and image tags.
   - Updated remote `docker compose` commands in `rollback-production.sh` to explicitly supply `sudo env VERSION="${VERSION}" docker compose ...`, preventing rollback from falling back to `v1.0.0` or assuming any unrelated release version.

3. **`deploy/scripts/test-version-propagation.sh`** (New Test Suite):
   - Comprehensive offline test harness verifying release version propagation across all deployment stages without connecting to production.

---

## 3. Version Propagation Strategy

### 3.1 Parameter Passing across SSH
Rather than modifying `.env` on disk (which would violate safety principles and introduce persistence risks), the normalized release tag (`VERSION_TAG="v${VERSION#v}"`) is passed as an explicit positional parameter across the SSH boundary:

```bash
ssh -p "${PORT}" "${USER}@${HOST}" bash -s -- "${REMOTE_DIR}" "${VERSION_TAG}" << 'EOF'
set -euo pipefail
REMOTE_DIR="$1"
VERSION="$2"
```

### 3.2 Sudo Environment Isolation Bypass
Because `sudo` typically cleanses non-whitelisted environment variables via `env_reset`, calling `VERSION="${VERSION}" sudo docker compose` can lose the variable. Instead, the verified safe pattern:

```bash
sudo env VERSION="${VERSION}" docker compose \
    -p clientpulse \
    -f "${REMOTE_DIR}/docker-compose.production.yml" \
    --env-file "${REMOTE_DIR}/.env" \
    ...
```

explicitly defines `VERSION` within `sudo`'s execution environment.

---

## 4. Remote Compose Stages Covered

| Stage | Step | Invocation Pattern | Purpose |
|---|---|---|---|
| **Stage G** | 1 | `sudo env VERSION="${VERSION}" docker compose ... up -d postgres` | Starts isolated PostgreSQL database container |
| **Stage G** | 2 | `$(sudo env VERSION="${VERSION}" docker compose ... ps -q postgres)` | Retrieves postgres container ID for health checks |
| **Stage G** | 3 | `sudo env VERSION="${VERSION}" docker compose ... run --rm --no-deps --entrypoint /app/migrate api up` | Executes database migrations using release `api` image |
| **Stage G** | 4 | `$(sudo env VERSION="${VERSION}" docker compose ... ps -q postgres)` | Retrieves postgres container ID for role password sync |
| **Stage H** | 1 | `sudo env VERSION="${VERSION}" docker compose ... up -d` | Starts `api`, `worker`, and `web` runtime containers |
| **Stage H** | 2 | `sudo env VERSION="${VERSION}" docker compose ... ps` | Verifies running status of all ClientPulse services |
| **Rollback** | 4 | `sudo env VERSION="${VERSION}" docker compose ... up -d --force-recreate` | Restores containers to targeted rollback release image |
| **Rollback** | 5 | `sudo env VERSION="${VERSION}" docker compose ... ps` | Verifies running status of rolled-back services |

---

## 5. Test Cases and Verification Results

All test suites were executed locally in isolated sandboxes with zero remote connections.

### 5.1 Test Suite: `deploy/scripts/test-version-propagation.sh`
- **Test 1: Static Audit of Remote Docker Compose Invocations**  
  *Result: PASS.* Confirmed every remote Compose invocation in Stage G and Stage H includes `sudo env VERSION="${VERSION}"`.
- **Test 2: Missing Version Flag (`-v`) Fails Early**  
  *Result: PASS.* Confirmed omission of `-v` halts deployment before any network or remote actions.
- **Test 3: Compose Interpolation Without `VERSION` in `.env`**  
  *Result: PASS.* Validated that `env VERSION="v1.1.1" docker compose config` correctly resolves `clientpulse-api:v1.1.1`, `clientpulse-worker:v1.1.1`, and `clientpulse-web:v1.1.1` without falling back to `v1.0.0`.
- **Test 4: Mocked Execution of `-v 1.1.1`**  
  *Result: PASS.* Verified normalization to `v1.1.1`, SSH argument passing, and propagation to all Stage G (migration) and Stage H (application) Compose operations.
- **Test 5: Mocked Execution of `-v v1.1.2`**  
  *Result: PASS.* Verified handling of pre-normalized tags (`v1.1.2`).
- **Test 6: Sudo Environment Isolation Safety**  
  *Result: PASS.* Verified that `sudo env VERSION="${VERSION}"` survives environment scrubbing (`env_reset`).
- **Test 7: Rollback Script Version Propagation**  
  *Result: PASS.* Verified that `rollback-production.sh` passes the explicit normalized target version and does not fall back to `v1.0.0`.
- **Test 8: Regression Check: TASK-DEPLOY-001 Port Ownership Validation**  
  *Result: PASS.* All 16 port ownership test scenarios in `test-port-validation.sh` remain passing.

### 5.2 Syntax and Lint Validation
```bash
bash -n deploy/scripts/deploy-production.sh    # Exit 0
bash -n deploy/scripts/rollback-production.sh  # Exit 0
(cd services/api && go test ./...)             # Exit 0
(cd apps/web && npm run lint)                  # Exit 0
```

---

## 6. Safety & Operational Constraints Confirmation

- **Zero Production Interaction**: No SSH connections were initiated to `37.152.191.170:2222`.
- **Zero Production Modifications**: No containers, databases, volumes, firewall rules, or certificates were altered.
- **Protected Containers Intact**: All 8 protected host containers remain strictly guarded by preflight checks.
- **Production `.env` Preserved**: No `VERSION` variable was added to `.env`.

---

## 7. Remaining Risks & Recommendations

1. **Local Build Architecture**: Ensure local Docker builds continue targeting `--platform linux/amd64` to match VPS host architecture.
2. **First Release vs Upgrades**: Because `docker compose down` is prohibited on production to protect host stability, `docker compose up -d` handles container recreation seamlessly when image tags change.

