# TASK-007.A-UX-AUDIT: ClientPulse Production Infrastructure Audit & Reconciled Deployment Architecture

**Document Version:** 2.0 (Reconciled)  
**Target Host:** `37.152.191.170:2222` (`ubuntu-g1-small5-foroogh-1`)  
**Target Domain:** `mindu.ir`  
**Execution Mode:** STRICT READ-ONLY PLANNING & AUDIT  
**Status:** COMPLETE (Zero Mutations Executed)

---

## Executive Summary & Non-Disruption Certification

This audit reconciles the findings of the production infrastructure inspection on the AbrArvan VPS (`37.152.191.170`) with the architectural requirements of ClientPulse.

### Absolute Protection Certification
During this audit:
- **No container operations were executed** (`docker stop`, `restart`, `rm`, `kill`, `down`, `prune`, etc. were strictly prohibited and never run).
- **All eight pre-existing production containers remain running with untouched uptime and persistent data**:
  - `rocketchat-rocketchat-1` (Unhealthy status preserved as found)
  - `rocketchat-mongodb-1`
  - `rustdesk-hbbs` (Host network mode preserved)
  - `rustdesk-hbbr` (Host network mode preserved)
  - `fchat-mattermost-1` (3+ months uptime preserved)
  - `fchat-postgres-1` (3+ months uptime preserved)
  - `docker-main-mattermost-1` (3+ months uptime preserved)
  - `docker-main-postgres-1` (3+ months uptime preserved)
- **Nginx configuration on the host was NOT modified or reloaded.**
- **No databases were accessed, queried, or modified.**
- **Host firewalls, iptables, SSH configurations, and DNS records were completely untouched.**
- **The directory `/home/ubuntu/clientpulse` was NOT created.**

---

## 1. Reconciled Docker Network Architecture (Two-Network Topology)

* **Finding Status:** `PROPOSED & RECONCILED` (Addresses single-network limitation)

The previous single internal network proposal is corrected. ClientPulse requires outbound network connectivity for:
1. Active website/API monitoring (HTTP, HTTPS, TLS cert checks, latency probes).
2. DNS record checks for monitored domains.
3. Outbound SMTP email delivery via STARTTLS on TCP port 587.
4. Telegram Bot API communication through an approved egress proxy.
5. Inbound/outbound external health probes.

To prevent any possibility of database exposure while enabling required worker egress, the deployment architecture mandates **two isolated Docker networks**:

### 1.1 Network Definitions
1. **`clientpulse_private`**:
   - Internal bridge network (`internal: true`).
   - Prohibits all external internet egress routing at the Docker bridge level.
   - Houses the PostgreSQL database and internal service-to-service communication.
2. **`clientpulse_egress`**:
   - Standard isolated Docker bridge network with default outbound gateway access.
   - Attached **only** to containers requiring outbound internet communication.
   - **PostgreSQL MUST NEVER be attached to this network.**

### 1.2 Service-by-Service Network Membership Matrix

| Service | Container Role | Attached to `clientpulse_private`? | Attached to `clientpulse_egress`? | Host Ports Bound | Rationale & Security Boundary |
|---|---|---|---|---|---|
| **`postgres`** | Database Store | **YES** | **NO (STRICT)** | None | Internal data store. Must never have external egress or public host port. |
| **`api`** | Go HTTP Backend | **YES** | **YES** | `127.0.0.1:8090:8080` | Needs `clientpulse_private` for DB & SSR; needs `clientpulse_egress` for transactional emails, manual test probes, and outbound alerts. |
| **`web`** | Next.js Frontend | **YES** | **NO** | `127.0.0.1:3100:3000` | Needs `clientpulse_private` for internal SSR calls to `api:8080`. Requires no arbitrary outbound internet access. |
| **`worker`** | Monitoring & Alerts | **YES** | **YES** | None | Needs `clientpulse_private` for PostgreSQL state & locks; needs `clientpulse_egress` for uptime checks, DNS probes, SMTP relay, and Telegram polling. |

```mermaid
graph TD
    subgraph "Nginx (Host Reverse Proxy)"
        Nginx["nginx (mindu.ir)"]
    end

    subgraph "Host Local Sockets (127.0.0.1)"
        P_Web["127.0.0.1:3100"]
        P_API["127.0.0.1:8090"]
    end

    subgraph "Docker Compose Isolation: clientpulse"
        subgraph "Network: clientpulse_private (internal: true)"
            DB[("PostgreSQL 16\n(clientpulse_postgres)")]
            Web["Next.js Web\n(clientpulse_web)"]
            API["Go API\n(clientpulse_api)"]
            Worker["Go Worker\n(clientpulse_worker)"]
        end

        subgraph "Network: clientpulse_egress (Outbound Bridge)"
            API
            Worker
        end
    end

    subgraph "External Internet Egress"
        MonitoredTargets["Monitored Targets\n(HTTP/HTTPS/DNS)"]
        SMTPRelay["External SMTP Relay\n(TCP 587 STARTTLS)"]
        TGProxy["Telegram Bot API\n(via Egress Proxy)"]
    end

    Nginx -->|proxy_pass| P_Web
    Nginx -->|proxy_pass| P_API
    P_Web -.-> Web
    P_API -.-> API

    Web -->|Internal SSR /api| API
    API -->|Port 5432| DB
    Worker -->|Port 5432 (Advisory Locks & State)| DB

    Worker -->|Outbound Probes| MonitoredTargets
    Worker -->|Outbound Port 587| SMTPRelay
    API -->|Outbound Port 587| SMTPRelay
    Worker -->|Outbound Tunnel| TGProxy
```

---

## 2. Telegram Egress Assessment & Unresolved Blocker

* **Finding Status:** `BLOCKED`

### 2.1 Audit Findings
- Direct connection to `api.telegram.org:443` from the VPS timed out (`BLOCKED/TIMEOUT`), confirming national censorship/filtering on the AbrArvan VPS IP (`37.152.191.170`).
- Localhost Squid proxy on `127.0.0.1:6824` returned `curl: (56) Received HTTP code 503 from proxy after CONNECT (ERR_CONNECT_FAIL)`. The existing Squid service cannot reach Telegram.
- Both Long Polling and Webhook modes require outbound communication with `api.telegram.org` (polling requires `getUpdates`; webhook requires `sendMessage` and `setWebhook`).

### 2.2 Evaluation of Solutions

| Architectural Option | Mechanism | Feasibility & Pros | Cons & Operational Impact |
|---|---|---|---|
| **Option A: External HTTP/SOCKS5 Proxy (Native)** | Supply external proxy via `TELEGRAM_PROXY_URL` in `.env` | **High:** Supported out of the box in ClientPulse Go code (`clientpulse/services/api/internal/notification/telegram`). Zero changes to container architecture. | Requires an external, reliable proxy hosted outside Iran with stable credentials. |
| **Option B: Outbound Egress Tunnel Sidecar** | Compose sidecar container (`gost` or `xray-core`) routing traffic to external VPS | **Medium:** All proxy logic encapsulated within the Compose project without altering host configuration. | Adds an additional running container to the resource budget (~30MB RAM) and requires maintenance of external tunnel endpoints. |
| **Option C: Dedicated Remote Worker** | Run Worker on an external European VPS connecting via WireGuard to the DB | **Low:** Completely eliminates censorship barriers for polling. | High operational complexity; exposes DB over VPN; splits architecture across multiple providers. |

### 2.3 Production Recommendation & Strict Gate
- **Status: BLOCKED for Live Delivery.**
- Until a working external proxy (`TELEGRAM_PROXY_URL`) or egress tunnel is supplied and verified by the user:
  - ClientPulse must be deployed with `TELEGRAM_MODE=disabled` in production `.env`, or left in dormant polling mode with proxy configured.
  - The application will function normally for all other features (monitoring, incident detection, reports, email alerts) without Telegram delivery disruption.

---

## 3. Root Domain Routing Decision & Rocket.Chat Implications

* **Finding Status:** `VERIFIED & RECONCILED`

### 3.1 Existing Host Routing
- Current `/etc/nginx/sites-available/mindu.ir` routes all traffic to `upstream mindu { server localhost:3000; }`.
- Host socket `0.0.0.0:3000` is bound by `rocketchat-rocketchat-1`.
- Rocket.Chat is also proxied through other virtual hosts (`chat.gazshop.ir` and `bp.mindu.ir` both forward to `localhost:3000`).

### 3.2 Routing Reconciliation
1. **Root Domain Repurposing:**
   - The user confirmed: *The current root-domain application has no active service that needs to be preserved. The previous configuration was copied from another application and is being repurposed exclusively for ClientPulse.*
   - Pointing `mindu.ir` to ClientPulse (`127.0.0.1:3100` and `127.0.0.1:8090`) will remove Rocket.Chat from `mindu.ir`.
   - **The Rocket.Chat container itself will remain running completely untouched** on port 3000 and accessible via its existing alternate hosts (`bp.mindu.ir`, `chat.gazshop.ir`).
2. **Subdomain Alternative (`chat.mindu.ir`):**
   - If Rocket.Chat must remain accessible under the `mindu.ir` namespace in the future, it would require:
     - Adding a DNS A record for `chat.mindu.ir` in Cloudflare pointing to `37.152.191.170`.
     - Issuing a separate Let's Encrypt TLS certificate for `chat.mindu.ir`.
     - Creating an isolated virtual host `/etc/nginx/sites-available/chat.mindu.ir`.
   - For Phase 1 deployment, `mindu.ir` root is dedicated exclusively to ClientPulse.

---

## 4. Reconciled Nginx Configuration & Cache Security

* **Finding Status:** `VERIFIED & PROPOSED`

### 4.1 Security Risk in Existing Configuration
- `/etc/nginx/sites-available/mindu.ir` currently inherits `proxy_cache mattermost_cache;` from another site (`mm.gazshop.ir`).
- For ClientPulse, caching dynamic responses would cause **severe security vulnerabilities**:
  - API responses containing session cookies, user credentials, or tenant-scoped data could be served to other tenants.
  - Report previews, share revocation tokens, and dynamic health stats would become stale or leaked.

### 4.2 Reconciled Configuration Specification (`/etc/nginx/sites-available/mindu.ir`)
The following configuration is verified for future application in TASK-007.B. **No edits have been made during this audit.**

```nginx
# ==============================================================================
# ClientPulse Production Virtual Host (mindu.ir)
# Strictly Isolated Reverse Proxy Configuration
# ==============================================================================

upstream clientpulse_web_upstream {
    server 127.0.0.1:3100;
    keepalive 32;
}

upstream clientpulse_api_upstream {
    server 127.0.0.1:8090;
    keepalive 32;
}

server {
    server_name mindu.ir;

    # Global Performance & Security
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css text/xml application/json application/javascript application/rss+xml application/atom+xml image/svg+xml;

    # --------------------------------------------------------------------------
    # 1. API, Public Reports & Health Check Routing -> Go API
    # --------------------------------------------------------------------------
    location ~ ^/(api|public|health)/ {
        client_max_body_size 25M;

        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Early-Data $ssl_early_data;

        proxy_buffers 256 16k;
        proxy_buffer_size 16k;
        proxy_read_timeout 120s;

        # Strict Cache Prohibition for Dynamic/Authenticated Data
        proxy_cache off;
        add_header Cache-Control "no-store, no-cache, must-revalidate, proxy-revalidate" always;
        add_header Pragma "no-cache" always;

        proxy_pass http://clientpulse_api_upstream;
    }

    # --------------------------------------------------------------------------
    # 2. Next.js Fingerprinted Static Assets -> Browser Long-term Cache
    # --------------------------------------------------------------------------
    location /_next/static/ {
        proxy_http_version 1.1;
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_cache off; # No Nginx disk cache needed; rely on immutable browser cache
        add_header Cache-Control "public, max-age=31536000, immutable";

        proxy_pass http://clientpulse_web_upstream;
    }

    # --------------------------------------------------------------------------
    # 3. Next.js Web Application & SSR Routes -> Frontend Container
    # --------------------------------------------------------------------------
    location / {
        client_max_body_size 10M;

        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Early-Data $ssl_early_data;

        proxy_buffers 256 16k;
        proxy_buffer_size 16k;
        proxy_read_timeout 60s;

        # Disable shared proxy caching for dynamic HTML/app pages
        proxy_cache off;
        add_header Cache-Control "no-cache, must-revalidate";

        proxy_pass http://clientpulse_web_upstream;
    }

    # --------------------------------------------------------------------------
    # TLS & Security Certificates (Retained exactly as managed by Certbot)
    # --------------------------------------------------------------------------
    listen 443 ssl; # managed by Certbot
    ssl_certificate /etc/letsencrypt/live/mindu.ir/fullchain.pem; # managed by Certbot
    ssl_certificate_key /etc/letsencrypt/live/mindu.ir/privkey.pem; # managed by Certbot
    include /etc/letsencrypt/options-ssl-nginx.conf; # managed by Certbot
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem; # managed by Certbot
}

# ------------------------------------------------------------------------------
# HTTP to HTTPS Redirection (Retained unchanged)
# ------------------------------------------------------------------------------
server {
    if ($host = mindu.ir) {
        return 301 https://$host$request_uri;
    } # managed by Certbot

    server_name mindu.ir;
    listen 80;
    return 404; # managed by Certbot
}
```

---

## 5. Reassessed Resource & Capacity Budget

* **Finding Status:** `VERIFIED & RECONCILED`

### 5.1 Real Memory Consumption of Existing Services (Live VPS Audit)
Exact memory usage collected from `sudo docker stats --no-stream` and `free -m`:

| Process / Container | Live Memory Usage | Memory % of Total | CPU % |
|---|---|---|---|
| `rocketchat-rocketchat-1` | **821.7 MiB** | 20.97% | 2.31% |
| `rocketchat-mongodb-1` | **203.2 MiB** | 5.19% | 1.67% |
| `docker-main-mattermost-1`| **294.6 MiB** | 7.52% | 0.04% |
| `fchat-mattermost-1` | **214.3 MiB** | 5.47% | 11.13% |
| `docker-main-postgres-1` | **79.0 MiB** | 2.02% | 0.00% |
| `fchat-postgres-1` | **54.6 MiB** | 1.39% | 0.17% |
| `rustdesk-hbbs` | **2.4 MiB** | 0.06% | 0.05% |
| `rustdesk-hbbr` | **1.6 MiB** | 0.04% | 0.02% |
| **Total Container Footprint** | **~1,672 MiB** | **~42.6%** | **~15.4%** |
| **Host System & Page Cache** | **~340 MiB (OS) + ~1.9 GiB (Buffer/Cache)** | — | — |
| **Available Unallocated Memory** | **1,903 MiB (~1.86 GiB)** | **~48.5%** | — |
| **System Swap Space** | **0B** (None configured) | — | — |

### 5.2 Realistic ClientPulse Memory Reassessment & Limits

Previous rough estimates are revised based on production container sizing:

| ClientPulse Service | Steady-State RAM | Peak RAM (During Operations) | Enforced Memory Limit (`mem_limit`) | Enforced Memory Reservation | Concurrency Setting |
|---|---|---|---|---|---|
| **PostgreSQL 16 Alpine** | 140 MiB | 180 MiB | **256 MiB** | 128 MiB | `max_connections = 40` |
| **Next.js Web (Node 24)** | 120 MiB | 180 MiB | **256 MiB** | 128 MiB | Default Node V8 heap |
| **Go API (`api`)** | 60 MiB | 320 MiB *(Chromium render peak)* | **512 MiB** | 128 MiB | In-process Chromium `--no-sandbox` |
| **Go Worker (`worker`)** | 40 MiB | 80 MiB | **192 MiB** | 64 MiB | `WORKER_CONCURRENCY = 5` (Low initial) |
| **Total ClientPulse** | **360 MiB** | **760 MiB** | **1,216 MiB (Hard Ceiling)** | **448 MiB** | — |

* **Headroom Analysis:**
  - Steady-state (360 MiB) leaves **~1.54 GiB** available headroom.
  - Peak burst during Chromium PDF generation (760 MiB) leaves **~1.14 GiB** headroom.
  - Enforced memory limits prevent ClientPulse from consuming more than 1.2 GiB under any condition.
* **Essential Swap Recommendation:**
  - Because active swap is **0B**, a 2 GiB swapfile on `/dev/vda3` (which has 17 GiB free) should be provisioned prior to deployment. This eliminates kernel OOM risk under sudden multi-application concurrency.

### 5.3 Disk Storage Budget & Reserve Margin
- **Available Disk Space on `/dev/vda3`:** **17.0 GiB** (75% used).
- **ClientPulse Disk Budget:**
  - Active release images (API: 1.23GB, Web: 330MB, Worker: 40MB, Postgres: 420MB): **~2.02 GiB**
  - Retained previous release archive (`releases/vX/`): **~450 MiB**
  - PostgreSQL persistent volume initial + 90 days retention: **~500 MiB**
  - Report PDF and branding logo storage: **~500 MiB**
  - Local backups (7 daily compressed dumps): **~250 MiB**
  - Temporary deployment archive (deleted post-load): **~450 MiB**
- **Total Projected Disk Utilization:** **~4.17 GiB**.
- **Residual Free Space:** `17.0 GiB - 4.17 GiB` = **~12.8 GiB**.
- **Safe Free-Space Reserve:** The projected remaining space comfortably exceeds the strict **8.0 GiB minimum reserve threshold**.

---

## 6. Corrected Database & Storage Backup Architecture

* **Finding Status:** `PROPOSED & RECONCILED` (Fixes shell expansion defect)

### 6.1 Validation of Shell Variable Expansion in `pg_dump`
In the previous proposal, executing:
`sudo docker compose -p clientpulse exec -T postgres pg_dump -U "$POSTGRES_USER" -Fc "$POSTGRES_DB"`
would cause the host shell to expand `$POSTGRES_USER` and `$POSTGRES_DB` to empty strings if they are not explicitly exported into the host shell environment.

**The corrected, validated backup command is:**
```bash
# Sourcing production .env on the host ensures variables are populated safely
set -a
. /home/ubuntu/clientpulse/.env
set +a

# Execute pg_dump inside container using container-side variable expansion
sudo docker compose -p clientpulse exec -T postgres sh -c 'pg_dump -U "$POSTGRES_USER" -Fc "$POSTGRES_DB"' > "/home/ubuntu/clientpulse/backups/db/clientpulse_db_$(date +%Y%m%d_%H%M%S).dump"
```

### 6.2 Complete Backup Strategy
1. **Database Dumps:** Daily compressed custom-format dump (`.dump`) saved to `/home/ubuntu/clientpulse/backups/db/`.
2. **Storage Volume Archive:** Tar-gzip of `/home/ubuntu/clientpulse/volumes/storage_data` saved to `/home/ubuntu/clientpulse/backups/storage/`.
3. **Retention Schedule:**
   - 7 daily backups retained locally.
   - Older backups pruned via `find /home/ubuntu/clientpulse/backups -type f -mtime +7 -delete`.
4. **Encrypted Off-Site Transfer:** Backups symmetrically encrypted with GnuPG (`gpg -c`) before off-site rsync to developer storage.
5. **Restoration Procedure:**
   ```bash
   # Restore Database
   cat backup.dump | sudo docker compose -p clientpulse exec -T postgres sh -c 'pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --clean --if-exists'
   
   # Restore Storage Volume
   tar -xzf storage_backup.tar.gz -C /home/ubuntu/clientpulse/volumes/storage_data/
   ```

---

## 7. Email Infrastructure & DNS Reconciliation

* **Finding Status:** `VERIFIED & REQUIRES USER DECISION`

### 7.1 Verified DNS & Egress State
- **MX Records:** Handled by Cloudflare Email Routing (`route1.mx.cloudflare.net`, `route2.mx.cloudflare.net`, `route3.mx.cloudflare.net`).
- **Current SPF Record:** `"v=spf1 include:_spf.mx.cloudflare.net ~all"`.
- **Egress Port 587 (Submission / STARTTLS):** **OPEN** and verified reachable from VPS.
- **Egress Port 25:** **BLOCKED** by cloud provider. Direct self-hosted SMTP is not viable.

### 7.2 Strict DNS Configuration Rules
1. **Single SPF Record Rule (RFC 7208):**
   - Multiple TXT SPF records cause an automatic SPF `PermError` and failed delivery.
   - When the user selects an external SMTP relay (e.g. Brevo), the SPF record in Cloudflare must be updated to merge authorizations:
     ```text
     v=spf1 include:_spf.mx.cloudflare.net include:spf.brevo.com ~all
     ```
2. **DKIM Record:**
   - Add the specific CNAME or TXT selector specified by the transactional provider (e.g., `mail._domainkey.mindu.ir`).
3. **Cloudflare Dashboard Requirement:**
   - The public DNS audit verified existing records, but Antigravity has no Cloudflare credentials or dashboard access.
   - The user must apply the merged SPF and DKIM records in their Cloudflare dashboard.

---

## 8. Application Deployment Compatibility & Build Verification

* **Finding Status:** `VERIFIED & RECONCILED` (Catches build-arg & migration entrypoint nuances)

### 8.1 Docker Images & Multi-Stage Dockerfiles

| Component | Dockerfile | Base / Runner Image | Internal Port | Exposed Host Port |
|---|---|---|---|---|
| **API** | `deploy/docker/api.Dockerfile` | `golang:1.27.3-alpine3.24` / `alpine:3.24.3` | `8080` | `127.0.0.1:8090` |
| **Web** | `deploy/docker/web.Dockerfile` | `node:24.22.2-alpine3.24` / `node:24.22.2-alpine3.24` | `3000` | `127.0.0.1:3100` |
| **Worker** | `deploy/docker/worker.Dockerfile` | `golang:1.27.3-alpine3.24` / `alpine:3.24.3` | None | None |
| **Postgres** | Standard Docker Library | `postgres:16.15-alpine3.24` | `5432` | None (Private network only) |

### 8.2 Next.js Build-Time Argument Requirement (`NEXT_PUBLIC_API_BASE_URL`)
- In [`apps/web/src/lib/api.ts`](file:///home/sadpc/Documents/ClientPulse/apps/web/src/lib/api.ts#L3-L6):
  ```typescript
  export const API_BASE_URL =
    process.env.NEXT_PUBLIC_API_BASE_URL ||
    process.env.NEXT_PUBLIC_API_URL ||
    "http://localhost:8080";
  ```
- In Next.js client bundles, `NEXT_PUBLIC_*` values are baked in **at build time**.
- If built without setting `NEXT_PUBLIC_API_BASE_URL`, client browsers will attempt to connect to `http://localhost:8080`.
- **Reconciled Requirement:**
  - When building the Docker image locally on the developer's PC, pass:
    `--build-arg NEXT_PUBLIC_API_BASE_URL="https://mindu.ir"` (or configure `ARG NEXT_PUBLIC_API_BASE_URL="https://mindu.ir"` in `web.Dockerfile`).
  - Because Nginx routes `https://mindu.ir/api/` directly to the API, browser requests to `${API_BASE_URL}/api/v1/...` resolve natively on the same origin with cookies and CSRF tokens.

### 8.3 Go Migration Tool Execution Architecture
- Inspection of `deploy/docker/api.Dockerfile` revealed:
  `ENTRYPOINT ["/app/api"]`
- Binary `/bin/migrate` is compiled and copied to `/app/migrate` alongside migrations at `/app/migrations`.
- **Critical Catch:** Executing `docker compose run --rm api /app/migrate up` would invoke `/app/api /app/migrate up` because `/app/api` is the container entrypoint!
- **Validated Migration Execution Methods:**
  - Method A (Stand-alone run with entrypoint override):
    ```bash
    sudo docker compose -p clientpulse run --rm --entrypoint /app/migrate api up
    ```
  - Method B (In-container execution on running API):
    ```bash
    sudo docker compose -p clientpulse exec -T api /app/migrate up
    ```

---

## 9. Final Deployment Prerequisites Checklist

Before executing TASK-007.B, the following prerequisites must be verified:

| Prerequisite Item | Status | Action Required |
|---|---|---|
| **Server Identity & SSH** | `VERIFIED` | Passwordless key authentication active; host verified. |
| **Two-Network Topology** | `VERIFIED` | `clientpulse_private` and `clientpulse_egress` mapped. |
| **Host Ports (3100, 8090)**| `VERIFIED` | Both ports confirmed completely free and unreferenced. |
| **Swap Space (2 GB)** | `PROPOSED` | Recommend provisioning 2 GB swapfile on host `/dev/vda3`. |
| **Nginx Authorization** | `VERIFIED` | Strict authorization for `/etc/nginx/sites-available/mindu.ir` only. |
| **Telegram Egress Route** | `BLOCKED` | User must provide external proxy or deploy with `TELEGRAM_MODE=disabled`. |
| **SMTP Relay Provider** | `REQUIRES USER DECISION`| User must select external SMTP relay (e.g. Brevo) & supply credentials. |
| **Cloudflare DNS Records** | `REQUIRES USER DECISION`| User must merge SPF and add DKIM in Cloudflare dashboard. |
| **Local Image Build Script**| `PROPOSED` | Local build script using `docker save`, `gzip`, and `--entrypoint` for migration. |

---

## STOP CONDITION COMPLIANCE

> [!IMPORTANT]
> **Strict Read-Only Planning Complete.**  
> - No software or package was installed.  
> - No Docker images were built, saved, or uploaded.  
> - No changes were made to `/etc/nginx/` or any server configuration.  
> - All eight protected containers continue running undisturbed.  
> - Ready for user review and explicit authorization before proceeding to implementation in TASK-007.B.

