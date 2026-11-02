# TASK-007.B: ClientPulse First Production Deployment Report

**Project:** ClientPulse  
**Target Domain:** https://mindu.ir  
**Target VPS:** 37.152.191.170:2222 (`ubuntu-g1-small5-foroogh-1`)  
**Deployment Date:** 2026-11-02  
**Status:** ✅ Successfully Deployed & Fully Operational  
**Zero-Disruption Guarantee:** 100% Verified (0 restarts across all protected services)  

---

## 1. Version Tag and Git Commit Hash

- **Version Tag:** `v1.0.0`
- **Git Commit Hash:** `a7acbda7645f37ac8b76a9f262f76d82101c64ed`
- **Commit Message:** `fix(notifications): complete TASK-006.3.1 link centralization, timeline verification and telegram polish`
- **Branch:** `main`
- **Tree State:** Clean (TASK-006.3.1 and TASK-007.A artifacts preserved)

---

## 2. Image Archive Metadata

All images were compiled locally on the developer machine for `linux/amd64` architecture, saved into a single archive, and compressed with maximum gzip:

| Attribute | Value |
|---|---|
| **Archive File** | `clientpulse-images-v1.0.0.tar.gz` |
| **File Size** | `550,473,083 bytes` (525 MB) |
| **SHA-256 Checksum** | `d4c9e1c482ee1e1cfc6dfba97db11d2c7f4ce41be6854a60208683b0bd9fe719` |
| **Integrity Verification** | Checked on VPS prior to loading via `sha256sum -c` (MATCH) |

### Included Docker Images

1. `clientpulse-api:v1.0.0` (ID: `93309e45e526`) — Go 1.27.3 statically linked API service binary, Chromium 155 headless, DejaVu/Noto Arabic fonts, and migration tooling.
2. `clientpulse-worker:v1.0.0` (ID: `93309e45e526`) — Shared runner image executing `/app/worker`.
3. `clientpulse-web:v1.0.0` (ID: `e52cf871ae81`) — Next.js 15 Standalone Node 20 runtime with `NEXT_PUBLIC_API_BASE_URL=https://mindu.ir` baked in.
4. `postgres:16.15-alpine3.24` (ID: `827aa19fb66c`) — Isolated relational database engine.

---

## 3. Target Environment

- **VPS Host:** `37.152.191.170`
- **SSH User:** `ubuntu`
- **SSH Port:** `2222`
- **Remote Project Directory:** `/home/ubuntu/clientpulse`
- **Release Directory:** `/home/ubuntu/clientpulse/releases/v1.0.0`
- **Compose Project Name:** `clientpulse`
- **Operating System:** Ubuntu 22.04.4 LTS (GNU/Linux 5.15.0-105-generic x86_64)

---

## 4. Running Containers, Status, and Healthchecks

All services are running under Compose project `clientpulse`:

```text
NAME                     IMAGE                       COMMAND                  STATUS                    PORTS
clientpulse-api-1        clientpulse-api:v1.0.0      "/app/api"               Up (healthy)              127.0.0.1:8090->8080/tcp
clientpulse-postgres-1   postgres:16.15-alpine3.24   "docker-entrypoint.s…"   Up (healthy)              (isolated, no host ports)
clientpulse-web-1        clientpulse-web:v1.0.0      "docker-entrypoint.s…"   Up (healthy)              127.0.0.1:3100->3000/tcp
clientpulse-worker-1     clientpulse-worker:v1.0.0   "/app/worker"            Up (healthy)              (internal only)
```

### Healthcheck Configurations

- **API:** `curl -f http://localhost:8080/health/live || exit 1` (Interval: 10s, Timeout: 3s, Retries: 3)
- **Web:** `wget --no-verbose --tries=1 --spider http://127.0.0.1:3000/ || exit 1` (Interval: 10s, Timeout: 3s, Retries: 3)
- **Worker:** `pidof worker || exit 1` (Interval: 10s, Timeout: 3s, Retries: 3)
- **PostgreSQL:** `pg_isready -U clientpulse_admin` (Interval: 10s, Timeout: 3s, Retries: 5)

---

## 5. Port Bindings and Network Isolation

### Loopback Enforcement

The host interface is protected against external direct container exposure:

```text
LISTEN 0      4096       127.0.0.1:3100       0.0.0.0:*    users:(("docker-proxy",pid=1798536,fd=4))
LISTEN 0      4096       127.0.0.1:8090       0.0.0.0:*    users:(("docker-proxy",pid=1799670,fd=4))
```

- Port `3100` is bound strictly to `127.0.0.1` (Next.js SSR Frontend).
- Port `8090` is bound strictly to `127.0.0.1` (Go Backend API).
- No service listens on `0.0.0.0` or public interface `37.152.191.170`.

### Database and Volume Isolation

- **Zero Host Exposure:** PostgreSQL has NO published host ports (`ports:` omitted in Compose). It cannot be reached via `localhost:5432` on the VPS host or from the public internet.
- **Network Segmentation:**
  - `clientpulse_data_internal`: Strict `internal: true` bridge network. Contains `postgres`, `api`, and `worker`. Cannot communicate with the outside internet.
  - `clientpulse_app_internal`: Bridge network linking `api` and `web`, published to loopback `127.0.0.1`.
  - `clientpulse_egress`: Separate bridge allowing the background `worker` outbound egress solely to perform HTTP/TCP uptime checks against monitored websites.
- **Dedicated Persistent Volume:** `clientpulse_postgres_data` is a dedicated named volume; existing databases (Mattermost, RocketChat) are completely untouched.

---

## 6. Applied Database Migrations

Database migrations were executed using `/app/migrate up` from the compiled migration container. All 12 schema migrations executed cleanly:

| Migration File | Description | Execution Status |
|---|---|---|
| `000001_create_users_table.up.sql` | Users and authentication tables | Applied (OK) |
| `000002_create_organizations_table.up.sql` | Multi-tenancy and organization membership | Applied (OK) |
| `000003_create_database_roles.up.sql` | `clientpulse_app` and `clientpulse_worker` roles | Applied (OK) |
| `000004_create_rls_policies.up.sql` | PostgreSQL Row Level Security (RLS) policies | Applied (OK) |
| `000005_grant_worker_role_permissions.up.sql` | Worker role least-privilege grants | Applied (OK) |
| `000006_create_monitors_table.up.sql` | HTTP/TCP monitors and SSL tracking | Applied (OK) |
| `000007_create_check_results_table.up.sql` | Monitoring check results partition/tables | Applied (OK) |
| `000008_create_heartbeats_table.up.sql` | Cron heartbeat monitoring endpoints | Applied (OK) |
| `000009_create_incidents_table.up.sql` | Incident management and state transitions | Applied (OK) |
| `000010_create_notification_destinations.up.sql` | Channels, tokens, and delivery records | Applied (OK) |
| `000011_create_maintenance_windows.up.sql` | Maintenance schedules and suppressions | Applied (OK) |
| `000012_create_reports_and_branding.up.sql` | Care reports, branding, and PDF storage | Applied (OK) |

---

## 7. Database User Roles and Privilege Verification

Least-privilege role separation is strictly enforced in PostgreSQL:

1. **`clientpulse_admin`:**
   - Superuser/schema owner used exclusively for running database migrations and backup dumps.
   - Credentials not used by runtime API or worker processes.
2. **`clientpulse_app`:**
   - Application runtime role.
   - Limited strictly to `SELECT`, `INSERT`, `UPDATE`, `DELETE` on tenant application tables.
   - Bound by Row-Level Security (`current_setting('app.current_tenant_id')`).
   - No DDL permissions (`DROP TABLE`, `ALTER TABLE` forbidden).
   - Password synchronized to production `.env`.
3. **`clientpulse_worker`:**
   - Background monitoring and maintenance role.
   - Granted access only to monitor targets, check results, heartbeats, incidents, worker heartbeats, and notification queues.
   - Cannot mutate core tenant administrative data.
   - Password synchronized to production `.env`.

---

## 8. TLS Certificate Details

- **Target Domain:** `mindu.ir`
- **Certificate Provider:** Let's Encrypt Authority
- **Certificate Path:** `/etc/letsencrypt/live/mindu.ir/fullchain.pem`
- **Private Key Path:** `/etc/letsencrypt/live/mindu.ir/privkey.pem`
- **Certificate Validity:** Verified active through **January 29, 2027**
- **TLS Configuration:** Modern TLS 1.2 and TLS 1.3 with secure ciphers, session resumption, and HTTP/2 enabled.

---

## 9. Nginx Configuration Status and Backup Location

- **Authorized Virtual Host File:** `/etc/nginx/sites-available/mindu.ir`
- **Enabled Symlink:** `/etc/nginx/sites-enabled/mindu.ir`
- **Backup Created Before Mutation:**  
  `/home/ubuntu/clientpulse/backups/nginx/mindu.ir_20261102_164006.bak`
- **Syntax Validation:** `sudo nginx -t` (Syntax OK, test successful)
- **Reload Status:** Graceful reload applied via `sudo systemctl reload nginx`.

### Routing Architecture

- `/api/` ➔ Proxy to `http://127.0.0.1:8090` (Go API)
- `/health/` ➔ Proxy to `http://127.0.0.1:8090` (Public and internal health probes)
- `/public/` ➔ Proxy to `http://127.0.0.1:8090` (Public client care reports and shared assets)
- `/` ➔ Proxy to `http://127.0.0.1:3100` (Next.js Web frontend)

### Privacy and Security Hardening

- **Access Logging Disabled:** Sensitive query strings and tokens in `/api/v1/heartbeat-pings` and `/public/reports` have access logging disabled (`access_log off;`) to protect client token privacy.
- **Cache Invalidation:** Dynamic API and public report endpoints have upstream proxy caching explicitly disabled (`proxy_no_cache 1; proxy_cache_bypass 1;`).
- **Security Headers:** Upstream Next.js and Go API headers preserved (`X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: strict-origin-when-cross-origin`).

---

## 10. Public Smoke Test Results

All smoke tests executed from an external host against `https://mindu.ir`:

| Endpoint Tested | Method | Expected | Actual Status | Details |
|---|---|---|---|---|
| `https://mindu.ir/` | `GET` | `200 OK` | `200 OK` | Full Next.js SSR HTML rendered (`Cache-Control: s-maxage=31536000`) |
| `https://mindu.ir/login` | `GET` | `200 OK` | `200 OK` | English & Persian localized auth page rendered |
| `https://mindu.ir/health/live` | `GET` | `200 OK` | `200 OK` | `{"status":"ok","service":"clientpulse-api","version":"1.0.0"}` |
| `https://mindu.ir/health/ready` | `GET` | `200 OK` | `200 OK` | `{"status":"ok","service":"clientpulse-api","database":"connected"}` |
| `https://mindu.ir/health/worker` | `GET` | `200 OK` | `200 OK` | `{"status":"ok","service":"clientpulse-worker","age_seconds":0.94}` |
| `https://mindu.ir/api/v1/info` | `GET` | `200 OK` | `200 OK` | `{"environment":"production","version":"1.0.0"}` |
| `https://mindu.ir/api/v1/auth/signup` | `POST` | `201 Created` | `201 Created` | Successfully registered user; returned `HttpOnly; Secure; SameSite=Lax` cookies |
| `https://mindu.ir/api/v1/auth/me` | `GET` | `200 OK` | `200 OK` | Verified user session & tenant organization in database |
| `https://mindu.ir/api/v1/notifications/destinations` | `GET` | `200 OK` | `200 OK` | Correctly reports `email_configured: false`, `telegram_configured: false` |
| `https://mindu.ir/api/v1/clients` | `POST` | `403 Forbidden` | `403 Forbidden` | Correctly verifies CSRF double-cookie and email verification requirements |
| `https://mindu.ir/public/reports/invalid` | `GET` | `404 Not Found` | `404 Not Found` | Handled cleanly by API router |
| `https://mindu.ir/api/v1/heartbeat-pings/test` | `POST` | `404 Not Found` | `404 Not Found` | Clean handling, sensitive access logging bypassed |

---

## 11. Localhost Upstream Test Results

Direct loopback connectivity verified from within the VPS host:

- `http://127.0.0.1:8090/health/live` ➔ `200 OK` (0 ms latency)
- `http://127.0.0.1:8090/health/ready` ➔ `200 OK` (`"database":"connected"`)
- `http://127.0.0.1:3100/` ➔ `200 OK`

---

## 12. Host and Container Resource Utilization

### Host Memory & Disk

| Metric | Before Deployment | After Deployment | Status |
|---|---|---|---|
| **Host Total RAM** | 3.8 GiB (3917 MB) | 3.8 GiB (3917 MB) | Verified |
| **Host Used RAM** | 1.80 GiB (1740 MB) | 1.80 GiB (1806 MB) | Minimal footprint (+66 MB) |
| **Host Available RAM** | 1.88 GiB (1840 MB) | 1.74 GiB (1741 MB) | Excellent headroom (> 1.7 GiB free) |
| **Host Swap** | 0 B (No swap created) | 0 B (No swap created) | Policy preserved |
| **Host Root Disk Free** | 17 GiB free | 15 GiB free (78% used) | Healthy headroom |

### ClientPulse Container Resource Footprint

| Service / Container | Memory Usage | Memory Limit | Memory % | CPU % |
|---|---|---|---|---|
| `clientpulse-worker-1` | `6.9 MiB` | `192 MiB` | `3.6%` | `0.15%` |
| `clientpulse-api-1` | `166.7 MiB` | `512 MiB` | `32.5%` | `0.02%` |
| `clientpulse-web-1` | `43.5 MiB` | `256 MiB` | `16.9%` | `0.03%` |
| `clientpulse-postgres-1` | `54.0 MiB` | `256 MiB` | `21.1%` | `1.11%` |
| **Total ClientPulse** | **~271 MiB** | **1,216 MiB Cap** | **22.3%** | **~1.3%** |

---

## 13. Memory Limits and Reservation Compliance

Strict memory quotas and reservations are specified in `docker-compose.production.yml`:

- `api`: Limit `512M`, Reservation `128M`
- `web`: Limit `256M`, Reservation `64M`
- `worker`: Limit `192M`, Reservation `64M`
- `postgres`: Limit `256M`, Reservation `64M`
- **Aggregate Max Limit:** `1,216 MiB` (guaranteeing that ClientPulse cannot exhaust host memory and cause OOM-kills to protected services).

---

## 14. Telegram Integration Status: DEFERRED

As instructed for the first production release, Telegram delivery is intentionally disabled:

- `TELEGRAM_MODE=disabled` in production `.env`.
- No Bot API token or webhook configured.
- No polling worker or outbound network requests to Telegram servers.
- `api` returns `telegram_configured: false`.
- Any queued domain notifications are transitioned cleanly to status `cancelled` with reason `telegram_transport_disabled`.

---

## 15. Email Integration Status: DEFERRED

As instructed for the first production release, Email delivery is intentionally disabled:

- `EMAIL_PROVIDER=disabled` in production `.env`.
- No SMTP host, credentials, or outbound SMTP connections configured.
- No local postfix or MTA installed on the VPS.
- `api` returns `email_configured: false`.
- In-memory mock mailer records events safely without external I/O.
- Frontend alerts user in Persian & English that Telegram and Email are deferred in this deployment.

---

## 16. Worker Heartbeat and Incident Detection Verification

- **Worker Service:** Active in `clientpulse-worker-1`.
- **Heartbeat Registry:** Periodically writes timestamps to the `worker_heartbeats` table in PostgreSQL.
- **Probe Verification:** `/health/worker` query confirms the latest heartbeat is `0.94` seconds old (well within the 10-second healthy timeout threshold).
- **Scheduler Active:** Periodic monitor evaluation, SSL expiration checks, and incident state machines are actively running.

---

## 17. PDF Report Generation Verification

- **Runtime Environment:** Alpine Linux 3.24.3 runner includes `chromium 155.0.8059.84` and system fonts (`font-noto-arabic`, `font-dejavu`).
- **Binary Detection:** Automatically discovered at `/usr/bin/chromium`.
- **Execution Flags:** Configured with `--headless --disable-gpu --no-sandbox --disable-dev-shm-usage --disable-software-rasterizer` for sandboxed headless report printing.
- **Unit & Local Tests:** Verified with `TestGenerateAndValidateEnglishReportPDF` (PASS in 0.63s).

---

## 18. Backup Tool Verification

- **Tool Path:** `/home/ubuntu/clientpulse/bin/backup-production.sh` (mode `755`)
- **Execution Test:** Executed on the VPS; completed in 2 seconds with exit code `0`.
- **Created Backups:**
  - Database: `/home/ubuntu/clientpulse/backups/database/clientpulse_db_20261102_164358.sql.gz` (16 KB, permissions `600`, owned by `ubuntu:ubuntu`).
  - Storage Volume: `/home/ubuntu/clientpulse/backups/storage/clientpulse_storage_20261102_164358.tar.gz` (4 KB, permissions `600`, owned by `ubuntu:ubuntu`).
- **Permissions:** Restrictive permissions (`chmod 600`, directory `700`) verified.
- **Pruning:** Automated retention policy prunes backups older than 7 days.

---

## 19. Off-Site Backup Limitation Documentation

> [!WARNING] Critical Operational Notice: Local Backups Only
> All backups created by `backup-production.sh` reside exclusively on local VPS disk storage at `/home/ubuntu/clientpulse/backups/`.
> 
> In the event of catastrophic server disk loss, VPS provider failure, or hardware corruption, local backups will NOT protect against data loss.
> 
> **Prerequisite Before Formal Public Launch:**  
> A dedicated off-site backup transport must be implemented (such as scheduled encrypted uploads to Cloudflare R2, AWS S3, or an external encrypted rsync destination).

---

## 20. Scoped Rollback Tool Verification and Documentation

- **Tool Path:** `deploy/scripts/rollback-production.sh` (and `/home/ubuntu/clientpulse/bin/rollback-production.sh`)
- **Safety Properties:**
  - Strictly scoped to `-p clientpulse`.
  - Preserves `/home/ubuntu/clientpulse/.env` and named database volumes.
  - Never runs automatic down migrations or drops tables.
  - Automatically loads the targeted previous image archive from `/home/ubuntu/clientpulse/releases/<TAG>/clientpulse-images.tar.gz`.
  - Optional flag `--restore-nginx` restores the backed-up Nginx configuration.

### Rollback Usage Example

```bash
# From local machine:
./deploy/scripts/rollback-production.sh -v v1.0.0 --restore-nginx

# Or directly on VPS:
/home/ubuntu/clientpulse/bin/rollback-production.sh -v v1.0.0
```

---

## 21. Protected Containers Audit

Before and after deployment, all 8 critical containers were inspected for container ID, image reference, creation time, start time, and restart count:

| Protected Container | Image Reference | Created At | Started At | Status | Restarts | Disruption |
|---|---|---|---|---|---|---|
| `rocketchat-rocketchat-1` | `mirror-docker.runflare.com/rocketchat/rocket.chat:7.13.3` | `2026-11-02 02:00:05 UTC` | `2026-11-02 02:00:06 UTC` | `Up 15 hours` | `0` | **NONE** |
| `rocketchat-mongodb-1` | `docker.io/bitnami/mongodb:5.0` | `2026-11-02 02:00:05 UTC` | `2026-11-02 02:00:05 UTC` | `Up 15 hours` | `0` | **NONE** |
| `rustdesk-hbbs` | `rustdesk/rustdesk-server:latest` | `2026-10-24 09:39:36 UTC` | `2026-10-24 09:39:37 UTC` | `Up 9 days` | `0` | **NONE** |
| `rustdesk-hbbr` | `rustdesk/rustdesk-server:latest` | `2026-10-24 09:39:36 UTC` | `2026-10-24 09:39:36 UTC` | `Up 9 days` | `0` | **NONE** |
| `fchat-mattermost-1` | `docker.arvancloud.ir/mattermost/mattermost-enterprise-edition:10.11.2` | `2026-03-23 09:15:02 UTC` | `2026-07-17 00:53:21 UTC` | `Up 3 months` | `0` | **NONE** |
| `fchat-postgres-1` | `docker.arvancloud.ir/postgres:13-alpine` | `2026-03-23 09:15:02 UTC` | `2026-07-17 00:53:23 UTC` | `Up 3 months` | `0` | **NONE** |
| `docker-main-mattermost-1` | `docker.arvancloud.ir/mattermost/mattermost-enterprise-edition` | `2026-01-17 20:20:36 UTC` | `2026-07-17 00:53:22 UTC` | `Up 3 months` | `0` | **NONE** |
| `docker-main-postgres-1` | `docker.arvancloud.ir/postgres:13-alpine` | `2026-01-17 20:20:36 UTC` | `2026-07-17 00:53:23 UTC` | `Up 3 months` | `0` | **NONE** |

---

## 22. Zero-Disruption Certification for Protected Services

> [!IMPORTANT] Formal Certification of Zero Disruption
> We certify that during the entirety of TASK-007.B deployment:
> 1. None of the 8 protected containers were restarted, stopped, recreated, or modified in any manner.
> 2. `docker compose down` was never executed globally.
> 3. No Docker prune commands (`system prune`, `volume prune`, `network prune`, `image prune`) were executed.
> 4. Existing PostgreSQL databases for Mattermost and Rocket.Chat MongoDB were never accessed, linked, or altered.
> 5. No Nginx virtual host configuration outside `/etc/nginx/sites-available/mindu.ir` was read or modified.
> 6. Host SSH configurations, firewall rules, and sudo policies were preserved without change.

---

## 23. Operational Runbook & Cheat Sheet

### Common Maintenance Commands (Run as `ubuntu` on VPS)

```bash
# 1. Check ClientPulse container status
sudo docker compose -p clientpulse -f /home/ubuntu/clientpulse/docker-compose.production.yml ps

# 2. View live logs for API or Worker
sudo docker compose -p clientpulse -f /home/ubuntu/clientpulse/docker-compose.production.yml logs -f --tail=100 api
sudo docker compose -p clientpulse -f /home/ubuntu/clientpulse/docker-compose.production.yml logs -f --tail=100 worker

# 3. Check memory & CPU consumption
sudo docker stats --no-stream --format "table {{.Name}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.CPUPerc}}" | grep -E 'NAME|clientpulse'

# 4. Trigger manual backup
/home/ubuntu/clientpulse/bin/backup-production.sh

# 5. Run verification suite from local machine
./deploy/scripts/verify-production.sh

# 6. Restart ClientPulse services gracefully (scoped)
sudo docker compose -p clientpulse -f /home/ubuntu/clientpulse/docker-compose.production.yml restart

# 7. Check upstream localhost health
curl -s http://127.0.0.1:8090/health/ready
curl -s http://127.0.0.1:8090/health/worker
```

---

*Report prepared and certified on 2026-11-02.*

