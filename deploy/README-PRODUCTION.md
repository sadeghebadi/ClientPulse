# ClientPulse Production Operations Guide

**Domain:** `https://mindu.ir`  
**Host:** `37.152.191.170:2222`  
**Compose Project:** `clientpulse`  
**Remote Directory:** `/home/ubuntu/clientpulse`  

---

## 1. Architecture Overview

ClientPulse runs in an isolated Docker Compose project (`clientpulse`) on the production VPS, interfaced through the existing host Nginx web server using Let's Encrypt TLS certificates.

### 1.1 Service Map
| Service | Image | Internal Port | Host Binding | Network Topology | Memory Limit |
|---|---|---|---|---|---|
| `postgres` | `postgres:16.15-alpine3.24` | `5432` | None (Private) | `clientpulse_data_internal` | 256 MiB |
| `api` | `clientpulse-api:<VERSION>` | `8080` | `127.0.0.1:8090` | `clientpulse_app_internal`, `clientpulse_data_internal` | 512 MiB |
| `worker` | `clientpulse-worker:<VERSION>` | None | None | `clientpulse_data_internal`, `clientpulse_egress` | 192 MiB |
| `web` | `clientpulse-web:<VERSION>` | `3000` | `127.0.0.1:3100` | `clientpulse_app_internal` | 256 MiB |

### 1.2 Network Topology
- **`clientpulse_app_internal`**: Internal bridge (`internal: true`). Only connects `web` and `api`.
- **`clientpulse_data_internal`**: Internal bridge (`internal: true`). Only connects `api`, `worker`, and `postgres`. No external egress routing.
- **`clientpulse_egress`**: Outbound bridge. Connects only `worker` to enable target URL and DNS monitoring checks.

### 1.3 Absolute Protection of Existing Host Services
The VPS hosts 8 pre-existing production containers that are strictly protected:
- `rocketchat-rocketchat-1`
- `rocketchat-mongodb-1`
- `rustdesk-hbbs`
- `rustdesk-hbbr`
- `fchat-mattermost-1`
- `fchat-postgres-1`
- `docker-main-mattermost-1`
- `docker-main-postgres-1`

All Docker operations MUST strictly use `-p clientpulse` and explicit compose files. Global prune commands (`docker system prune`, `volume prune`, etc.) and global restart commands are prohibited.

---

## 2. Deployment Procedures

### 2.1 Automated Deployment
Deploy a new release to production:

```bash
./deploy/scripts/deploy-production.sh -v 1.0.0
```

Workflow:
1. Runs local regression tests (Go tests and Next.js lint).
2. Builds `linux/amd64` Docker images locally.
3. Exports and compresses images to `clientpulse-images-v1.0.0.tar.gz`.
4. Computes SHA-256 checksum and generates release metadata.
5. Performs preflight checks on VPS (verifying protected containers and capacities).
6. Prompts for human approval before transfer.
7. Securely transfers archive and verifies checksum remotely.
8. Loads images via `sudo docker load`.
9. Starts `postgres` and executes `/app/migrate up`.
10. Starts application services and verifies localhost upstreams.
11. Backs up and updates `/etc/nginx/sites-available/mindu.ir`, followed by `sudo nginx -t` and graceful reload.
12. Performs public verification smoke tests.

---

## 3. Operations & Maintenance

### 3.1 Verification
To run health and infrastructure checks at any time:

```bash
./deploy/scripts/verify-production.sh
```

### 3.2 Backups
Run an on-demand database and storage backup:

```bash
ssh -p 2222 ubuntu@37.152.191.170 "/home/ubuntu/clientpulse/bin/backup-production.sh"
```

Backups are saved to:
- `/home/ubuntu/clientpulse/backups/database/clientpulse_db_<TIMESTAMP>.sql.gz`
- `/home/ubuntu/clientpulse/backups/storage/clientpulse_storage_<TIMESTAMP>.tar.gz`
- Retention: 7 days.

> [!WARNING]
> Backups are currently stored locally on the VPS disk. An off-site backup destination has not been configured yet. Do not onboard paying customers until an off-site backup and restoration strategy is verified.

### 3.3 Rollback
Rollback to a previous release:

```bash
./deploy/scripts/rollback-production.sh -v 1.0.0
```

To also restore the previous Nginx configuration:

```bash
./deploy/scripts/rollback-production.sh -v 1.0.0 --restore-nginx
```

---

## 4. Deferred Features (Initial Release)
- **Telegram Bot Delivery**: Disabled via `TELEGRAM_MODE=disabled`.
- **Email Delivery**: Disabled via `EMAIL_PROVIDER=disabled`.
- Core application functions (website monitoring, heartbeats, incidents, care reports, Persian/English UI) operate normally.

