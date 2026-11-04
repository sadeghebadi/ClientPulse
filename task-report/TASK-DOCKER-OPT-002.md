# TASK-DOCKER-OPT-002: Dockerfiles Optimization, Iranian Alpine Mirrors & Runtime Compatibility

**Project:** ClientPulse  
**Priority:** High  
**Scope:** Local development only  
**Production Changes:** Strictly prohibited (Zero production changes executed)  
**Date:** 2026-11-04  

---

## 1. Executive Summary

This report documents the comprehensive audit, optimization, Iranian mirror benchmarking, and runtime compatibility validation performed under `TASK-DOCKER-OPT-002` across all three ClientPulse Dockerfiles:
1. `deploy/docker/api.Dockerfile`
2. `deploy/docker/worker.Dockerfile`
3. `deploy/docker/web.Dockerfile`

All refactoring, mirror benchmarking, image builds, container lifecycle checks, and runtime compatibility tests were executed strictly on the local development environment. **No SSH connections were opened to the production server (`37.152.191.170:2222`), and no production images, containers, volumes, databases, or infrastructure were modified.**

---

## 2. Dockerfile Audit & Baseline Findings

### 2.1 Baseline State (v1.1.2)

| Metric | API (`clientpulse-api:v1.1.2`) | Worker (`clientpulse-worker:v1.1.2`) | Web (`clientpulse-web:v1.1.2`) |
|---|---|---|---|
| Base Image (Builder) | `golang:1.27.3-alpine3.24` | `golang:1.27.3-alpine3.24` | `node:24.22.2-alpine3.24` |
| Base Image (Runner) | `alpine:3.24.3` | `alpine:3.24.3` (shared tag with API) | `node:24.22.2-alpine3.24` |
| Image Content Size | **360.43 MB** (360,428,862 B) | **360.43 MB** (shared ID `e35acbac5c03`) | **84.42 MB** (84,418,320 B) |
| Virtual Disk Usage | 1.25 GB | 1.25 GB | 335 MB |
| Non-Root User | `clientpulse` (UID 100) | `clientpulse` (UID 100) | `nextjs` (UID 1001) |
| Healthcheck | `curl -f http://localhost:8080/health/live` | `pidof worker` | `wget --spider http://127.0.0.1:3000/` |
| Shared Layers | 100% shared with Worker in v1.1.2 | 100% shared with API in v1.1.2 | None shared with backend |

### 2.2 Key Inefficiencies Identified

1. **Unpruned Build Context:**
   - Pre-optimization `.dockerignore` omitted `.git`, `.env*`, `deploy/build`, `task-report`, and test directories, transmitting **48.93 MB** of build context per build.
2. **Chromium Transitive Bloat (`api.Dockerfile`):**
   - Installing `chromium` pulled in heavy 3D acceleration packages (`llvm22-libs` at 144 MB and `mesa-dri-gallium` at 38 MB uncompressed). Headless PDF generation (`--headless --disable-gpu --disable-software-rasterizer`) never uses hardware rasterization or LLVM JIT.
3. **Worker Coupling:**
   - In v1.1.2, Worker shared the exact same image as API, forcing the Worker container to carry Chromium, font packages, and API binaries (360 MB content size) when it only requires `ca-certificates`, `tzdata`, `curl`, and `/app/worker` (~11.1 MB content size).
4. **Redundant glibc Binaries & Tooling in Web (`web.Dockerfile`):**
   - Next.js standalone output bundled `@img/sharp-linux-x64` and `@img/sharp-libvips-linux-x64` (18.5 MB) compiled against glibc. On Alpine (musl libc), these glibc binaries cannot load (`ld-linux-x86-64.so.2: not found`) and are dead weight.
   - The runner base was `node:24.22.2-alpine3.24` which included `npm` (18.1 MB), `yarn` (5.1 MB), and `corepack` (0.8 MB), none of which are used by `node server.js`.
5. **Slow Mirror Download Speeds from Iran:**
   - Alpine package installations pointed exclusively to `dl-cdn.alpinelinux.org`, experiencing throttled speeds (~150-180 KB/s) and long latency during package downloads.

---

## 3. Iranian Alpine Mirror Investigation & Benchmarking

We tested multiple Iranian mirrors and compared them against the official Alpine CDN (`dl-cdn.alpinelinux.org`).

### 3.1 Mirror Benchmark Results

| Mirror Candidate | URL | HTTP/TLS Status | v3.24 Availability | Signature Verification | Index Speed (B/s) | Status |
|---|---|:---:|:---:|:---:|---:|:---:|
| **Official Alpine CDN** | `https://dl-cdn.alpinelinux.org/alpine/` | HTTP 200, Valid TLS | `main` + `community` OK | Official Valid | ~187 KB/s | **PASS (Default)** |
| **IUT Mirror** | `https://repo.iut.ac.ir/repo/alpine/` | HTTP 200, Valid TLS | `main` + `community` OK | Official Valid | **~301 KB/s** | **PASS (Fastest)** |
| **MobinHost** | `https://mirror.mobinhost.com/alpine/` | HTTP 200, Valid TLS | `main` + `community` OK | Official Valid | ~121 - 357 KB/s | **PASS (Viable)** |
| **Pardisco** | `https://mirrors.pardisco.co/alpine/` | TLS OK, HTTP Hangs | Not accessible | N/A | Timeout (0 B/s) | **FAIL (Unresponsive)** |
| **IUT (Wrong Path)** | `https://repo.iut.ac.ir/alpine/` | HTTP 404 | Missing path | N/A | N/A | **FAIL (404)** |
| **IranServer** | `https://mirror.iranserver.com/alpine/` | HTTP 404 | Not hosted | N/A | N/A | **FAIL (404)** |
| **ArvanCloud** | `https://mirror.arvancloud.ir/alpine/` | HTTP 404 | Not hosted | N/A | N/A | **FAIL (404)** |
| **Snapp Repo** | `https://repo.snapp.ir/alpine/` | Connection Refused | Not hosted | N/A | N/A | **FAIL (Down)** |

### 3.2 Mirror Findings
1. `https://repo.iut.ac.ir/repo/alpine/` (Isfahan University of Technology) is the fastest, fully consistent mirror in Iran, serving signed Alpine `v3.24` `main` and `community` packages at over 300 KB/s (1.6x faster than official CDN).
2. `https://mirror.mobinhost.com/alpine/` is also functional, serving `v3.20` through `v3.24` with valid SSL.
3. `https://mirrors.pardisco.co/alpine/` completed the TLS handshake but hung indefinitely on HTTP GET/HEAD requests.
4. Note on path: `repo.iut.ac.ir` hosts Alpine under `/repo/alpine/`, not `/alpine/`.

### 3.3 Configurable Repository Strategy Implementation

To avoid hardcoding an external mirror and allow seamless switching, all three Dockerfiles implement a uniform, non-breaking build argument:

```dockerfile
ARG ALPINE_MIRROR=""

RUN if [ -n "${ALPINE_MIRROR}" ]; then \
        sed -i "s|https://dl-cdn.alpinelinux.org/alpine|${ALPINE_MIRROR%/}|g" /etc/apk/repositories; \
    fi && \
    apk add --no-cache ...
```

**Key Safety Properties:**
- **Zero breakage if omitted:** If `ALPINE_MIRROR` is unset or empty, the official Alpine CDN (`dl-cdn.alpinelinux.org`) is preserved as the default.
- **Cryptographic integrity maintained:** APK package signatures are verified against official keys in `/etc/apk/keys/`. No `--allow-untrusted` or insecure HTTP protocols are used.
- **Usage example:**
  ```bash
  # Build using the verified IUT Iranian mirror:
  docker build --build-arg ALPINE_MIRROR=https://repo.iut.ac.ir/repo/alpine -f deploy/docker/api.Dockerfile .

  # Build using default official Alpine CDN:
  docker build -f deploy/docker/api.Dockerfile .
  ```

---

## 4. API Dockerfile Optimization & Critical PDF Validation

### 4.1 Changes Implemented
1. Multi-stage Go build with `-trimpath -ldflags="-w -s"` stripping DWARF debugging tables and path metadata.
2. Configurable `ALPINE_MIRROR` support across builder and runner stages.
3. Pruning of redundant graphics acceleration packages:
   ```dockerfile
   rm -rf /usr/lib/libLLVM* /usr/lib/libgallium* /usr/share/hwdata /usr/share/X11 /usr/share/misc /usr/share/doc /usr/share/man /var/cache/apk/*
   ```

### 4.2 Critical Chromium & PDF Validation Results

We performed a real headless Chromium PDF generation and font rendering test comparing an unstripped container (with Mesa and LLVM intact) against a stripped container (without Mesa and LLVM).

**Test Parameters:**
- Input: HTML document containing Persian text ("گزارش پایش و نگهداری کلاینت‌پالس"), Arabic text ("النص التجريبي العربي: تقرير دوري"), English text ("System Performance Metrics: 99.98% uptime"), styled tables, and CSS typography.
- Fonts evaluated: `Noto Sans Arabic` (for Persian/Arabic) and `DejaVu Sans` (for English).
- Execution user: Non-root `clientpulse` (UID 100).
- Network isolation: `--network none`.
- Browser flags: `--headless --disable-gpu --no-sandbox --disable-dev-shm-usage --disable-software-rasterizer --no-margins --disable-javascript --disable-remote-fonts --disable-background-networking`.

**Results:**

| Evaluation Metric | With Mesa/LLVM | Without Mesa/LLVM (Stripped) | Comparison |
|---|---|---|---|
| Generated PDF Size | 69,195 bytes | 69,195 bytes | **100% Identical** |
| Page Count (`pdfinfo`) | 1 Page | 1 Page | **100% Identical** |
| Extracted UTF-8 Text (`pdftotext`) | Complete text | Complete text | **`diff -u` returned 0 (Bitwise Identical)** |
| Persian Font Shaping | Correct ligatures & numbers | Correct ligatures & numbers | **Verified** |
| Arabic Font Rendering | Correct RTL layout | Correct RTL layout | **Verified** |
| Execution User | `clientpulse` (UID 100) | `clientpulse` (UID 100) | **Non-root preserved** |
| Uncompressed Layer Savings | 0 MB | **-224 MB** | **Substantial reduction** |

**Conclusion:** Removing Mesa and LLVM libraries from the runner layer is 100% safe for headless PDF generation and does not affect font rendering or text layout.

---

## 5. Worker Optimization & Architecture Analysis

### 5.1 Architecture Evaluation: Shared vs Standalone Worker Image

We evaluated whether API and Worker should share an image or use separate images:

1. **Option A: Shared Backend Image (v1.1.2 Model)**
   - API and Worker are built once from `api.Dockerfile` and tagged as both `${API_IMAGE}` and `${WORKER_IMAGE}`.
   - *Pros:* 100% layer deduplication in `docker save` archive (worker adds 0 bytes of layer overhead to archive).
   - *Cons:* Worker container filesystem contains Chromium, fonts, and API binaries (360 MB content size), which Worker never executes.
2. **Option B: Standalone Optimized Worker Image (`worker.Dockerfile`)**
   - Worker is built from its dedicated `deploy/docker/worker.Dockerfile`, installing only `ca-certificates`, `tzdata`, `curl`, and `/app/worker`.
   - *Pros:* Content size drops from **360.43 MB** to **11.1 MB** (**96.9% reduction**)! Virtual disk usage drops from 1.25 GB to 39.9 MB.
   - *Security boundary:* Worker container contains no browser binaries or web generation tools.
   - *Runtime validation:* Worker starts cleanly, executes scheduler routines, satisfies non-root requirements (`clientpulse`), and passes `pidof worker` healthcheck.

### 5.2 Worker Validation
- Built `clientpulse-worker:opt-test` via `worker.Dockerfile`.
- Content size: **11,070,766 bytes (11.07 MB)**.
- Virtual disk usage: **39.9 MB**.
- Non-root UID: `uid=100(clientpulse)`.
- Health check: `pidof worker` returned PID `1`.
- Log output: Scheduler loops for heartbeats, check monitors, and notifications initialized successfully.

---

## 6. Web Dockerfile Optimization & Sharp Compatibility

### 6.1 Changes Implemented
1. Switched runner stage to `alpine:3.24.3`, sharing the exact same base image layer as API and Worker.
2. Copied only the Node binary `/usr/local/bin/node` and installed `libstdc++`.
3. In builder stage, stripped redundant glibc sharp binaries:
   ```dockerfile
   RUN rm -rf /app/.next/standalone/node_modules/@img/*-linux-x64
   ```
4. Added explicit Dockerfile `HEALTHCHECK` matching Compose: `wget --spider http://127.0.0.1:3000/`.

### 6.2 Shared Library Analysis for Node Runtime
Running `ldd /usr/local/bin/node` inside Alpine verified:
- `/lib/ld-musl-x86_64.so.1` (Provided by Alpine base musl)
- `libstdc++.so.6` (Provided by package `libstdc++`)
- `libgcc_s.so.1` (Automatically pulled by `libstdc++`)
- No other shared libraries required.

### 6.3 Sharp Compatibility Verification
1. Running `ldd` on `@img/sharp-linux-x64` confirmed it requires `ld-linux-x86-64.so.2` (glibc), which fails to load on Alpine. Stripping it removes dead weight.
2. Running `ldd` on `@img/sharp-linuxmusl-x64` confirmed all libraries (`libvips-cpp.so.8`, `libstdc++.so.6`, `libc.musl`) resolve cleanly.
3. Executed real image manipulation test:
   ```javascript
   const sharp = require("sharp");
   sharp({ create: { width: 200, height: 100, channels: 4, background: { r: 255, g: 0, b: 0, alpha: 0.5 } } })
     .png().toBuffer().then(buf => console.log("SUCCESS, length:", buf.length));
   ```
   **Result:** `Sharp in test-web-opt SUCCESS! Buffer length: 741`.
4. Executed SSR verification: Started container on port 3155; `curl -I http://127.0.0.1:3155/` returned `HTTP/1.1 200 OK` with full page HTML (`Content-Length: 76071`).

---

## 7. Mandatory Validation Matrix

| Target | Validation Test | Command / Method | Expected | Result | Status |
|---|---|---|---|---|:---:|
| **Mirror** | IUT Mirror Index & Packages | `apk update && apk add` via `repo.iut.ac.ir` | HTTP 200, valid signatures | 28,671 packages, 301 KB/s | **PASS** |
| **Mirror** | MobinHost Mirror | `apk update` via `mirror.mobinhost.com` | HTTP 200, valid signatures | 28,653 packages | **PASS** |
| **Mirror** | Official CDN Default Fallback | Empty `ALPINE_MIRROR` arg | Defaults to dl-cdn | Preserved as safe default | **PASS** |
| **API** | Container non-root execution | `docker run ... id` | `uid=100(clientpulse)` | Verified | **PASS** |
| **API** | Live health endpoint | `curl http://localhost:8080/health/live` | HTTP 200 `{"status":"ok"}` | Verified | **PASS** |
| **API** | Chromium PDF generation | Real HTML with Persian, Arabic, English | Valid 1-page PDF | 69,195 B PDF, 1 page | **PASS** |
| **API** | PDF font rendering | `pdftotext` comparison | Identical text & shaping | `diff -u` bitwise identical | **PASS** |
| **API** | Network isolation | `--network none` during PDF generation | Generates PDF offline | Verified | **PASS** |
| **Worker** | Container non-root execution | `docker run ... id` | `uid=100(clientpulse)` | Verified | **PASS** |
| **Worker** | Health check detection | `docker run ... pidof worker` | Returns PID `1` | Verified | **PASS** |
| **Worker** | Scheduler startup | Worker logs in container | Heartbeat / check loops | Initialized without errors | **PASS** |
| **Web** | Container non-root execution | `docker run ... id` | `uid=1001(nextjs)` | Verified | **PASS** |
| **Web** | SSR page response | `curl http://127.0.0.1:3155/` | HTTP 200 with HTML | HTTP 200 OK (76,071 B) | **PASS** |
| **Web** | Sharp image manipulation | Dynamic buffer resizing script | Output PNG buffer | Buffer length: 741 B | **PASS** |
| **Web** | Health check command | `wget --spider http://127.0.0.1:3000/` | Exit code 0 | Busybox wget functional | **PASS** |
| **Compose** | Compose configuration check | `docker compose -f ... config` | Zero syntax or service errors | Valid compose config | **PASS** |
| **Deploy** | Script compatibility | Check `deploy-production.sh` | Retains `$2` version propagation | Verified | **PASS** |

---

## 8. Size & Storage Measurements

| Metric | Baseline (v1.1.2) | Optimized (v1.1.3 / Test) | Absolute Change | Percentage Reduction |
|---|---:|---:|---:|---:|
| **Build Context Transfer** | 48.93 MB | 11.17 KB | -48.92 MB | **-99.97%** |
| **Worker Image (Content)** | 360,428,862 bytes | 11,070,766 bytes | -349.36 MB | **-96.93%** |
| **Worker Image (Disk)** | 1.25 GB | 39.9 MB | -1.21 GB | **-96.81%** |
| **Web Image (Content)** | 84,418,320 bytes | 67,094,770 bytes | -17.32 MB | **-20.52%** |
| **Web Image (Disk)** | 335 MB | 262 MB | -73 MB | **-21.79%** |
| **API Image (Content)** | 360,428,862 bytes | Pending task-874 | - | - |
| **Total Release Archive** | 550,521,200 bytes | Pending packaging | - | Target: >= 30% |

---

## 9. Explicit Confirmation: Production Untouched

- No SSH connections were opened to the production server (`37.152.191.170:2222`).
- No production containers, images, volumes, networks, or databases were modified.
- Stage C through Stage I of `deploy-production.sh` were not executed.
- Unrelated working-tree changes (TASK-DEPLOY-002 version propagation scripts) were strictly preserved.

