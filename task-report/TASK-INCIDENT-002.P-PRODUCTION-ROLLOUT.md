# TASK-INCIDENT-002.P: Production Deployment Report

**Project:** ClientPulse  
**Environment:** Production (`https://mindu.ir`)  
**VPS Host:** `ubuntu-g1-small5-foroogh-1` (`37.152.191.170:2222`)  
**Execution Date:** 2026-11-02  
**Target Migration:** `000013_assets_scheduler_rls_hardening`  
**Status:** **SUCCESSFUL**

---

## A. Executive Summary

- **Deployment Status:** Successfully deployed to production.
- **Migration Status:** Version 13 applied and verified atomically. Schema migrations table records version 13.
- **RLS Hardening Result:** The emergency `FOR ALL` policy on `public.assets` for role `clientpulse_definer` was replaced with the least-privilege `FOR SELECT` policy (`assets_definer_policy`).
- **Monitoring Scheduler:** Functional and operating automatically. Three consecutive scheduled automatic monitoring runs were observed and verified in production with HTTP 200 responses.
- **Protected Infrastructure:** All 8 non-ClientPulse production containers remained 100% untouched with identical container IDs, image IDs, zero restart increments, and unchanged network topologies.

---

## B. Pre-Deployment Baseline

### System Resources
- **Host Uptime:** 108 days, 18 hours
- **Memory:** 3.8 GiB total, 1.8 GiB used, 1.7 GiB available
- **Disk:** 70 GiB total, 52 GiB used (78%), 15 GiB available
- **Docker Daemon:** Active

### Protected Containers Baseline
| Container Name | Container ID | Image ID | Restart Count | Status |
|---|---|---|---|---|
| `rocketchat-rocketchat-1` | `fee0c779ecaa6f59bb01683684fb4ce45ace42a0ed3ce4643badf48b6f9b7f68` | `sha256:bf8332ba9ffda7f75598de6875479823c427a5b8521c9a5c7f77807d36d0eb6d` | 0 | running |
| `rocketchat-mongodb-1` | `66dcfc37b5fe6342f7b329a5ea2e5b97ace6b7781437ec216c57bdcd340aa670` | `sha256:9d7f29f45afa6aa2a26539edfa969b8022e87a2b7bd8194f2275e78ab7545d70` | 0 | running |
| `rustdesk-hbbs` | `3b50e881dcfa6b220b062467c264e7c87fa955f0060836bfd2132f8716b5c560` | `sha256:ae1522abfa7f00e85c2dbadd3a1d68bab5f4f7a39ff742f05d22ea64a7c6529b` | 0 | running |
| `rustdesk-hbbr` | `cf8e3bbb029e655c6fa01738f557cbf6d850d26f46ee578ff15720249e9a4f1c` | `sha256:ae1522abfa7f00e85c2dbadd3a1d68bab5f4f7a39ff742f05d22ea64a7c6529b` | 0 | running |
| `fchat-mattermost-1` | `f6d1f3c1f425d84a0443f07311ed6638586071976d0cc3ec84e21a8ed74649ff` | `sha256:ce2fa5cdc5a2241e4c1ca33edef758a6157a9d59f2a1033741fe4a0267b086bd` | 0 | running |
| `fchat-postgres-1` | `9f6887895acc86417182f2b383a686dbdc3dc494804dfdc4673abd638d46650b` | `sha256:017c7689ab46e179fd5b3283b0eddb30d2a1cc811567560d7540e6a50231d6b8` | 0 | running |
| `docker-main-mattermost-1` | `d2a7e61f05331235b8c518d0e1a6f99e3753a6de7706b80de47b1d406ebffec1` | `sha256:e7076118debefca79c6e20307b0fdc37d5ea99b8b40bc1f8431477de5bf81933` | 0 | running |
| `docker-main-postgres-1` | `f3a9dd7cba487d37fc556707ba82a924c1bd221f338d1456cb05376d2acd3567` | `sha256:017c7689ab46e179fd5b3283b0eddb30d2a1cc811567560d7540e6a50231d6b8` | 0 | running |

### Database Baseline
- **Current Migration Version:** 12 (`notification_message_quality`)
- **RLS Policy on `public.assets`:**
  - `policyname`: `assets_definer_policy`
  - `cmd`: `ALL`
  - `roles`: `{clientpulse_definer}`
  - `qual`: `true`
  - `with_check`: `true`
- **Security flags:** `relrowsecurity: t`, `relforcerowsecurity: t` on both `assets` and `monitors`.

---

## C. Backup Evidence

Pre-rollout production backup was taken using `/home/ubuntu/clientpulse/bin/backup-production.sh`.

- **Backup Timestamp:** `20261102_193002`
- **Database Dump:**
  - Path: `/home/ubuntu/clientpulse/backups/database/clientpulse_db_20261102_193002.sql.gz`
  - Size: 18 KB
  - Permissions: `0600`
  - SHA-256: `fe791be2d65120ab54e39ed1206ba080ee2fc2d18fa9b993a0a827a9100731fa`
  - Verification: Inspected with `zcat`; contains full PostgreSQL 16.15 dump with complete table definitions and RLS policies.
- **Storage Dump:**
  - Path: `/home/ubuntu/clientpulse/backups/storage/clientpulse_storage_20261102_193002.tar.gz`
  - Size: 18 KB
  - Permissions: `0600`
  - SHA-256: `6730f386937c319ff0d8b2bafefcab35b17b633cb3f0930629d9ea7d83c16608`
  - Verification: Inspected with `tar -ztvf`; contains tenant logo assets.
- **Exit Status:** 0 (clean completion)

---

## D. Release Evidence

- **Source Commit:** `a60917c4578770c3c3d216b6f77621f0b2e6772b`
- **Release Tag:** `v1.0.1`
- **Built Image:** `clientpulse-api:v1.0.1`
- **Image ID:** `sha256:6071b7972a9262756358aad233fe93971c98e15cdec14d136c45ada3246c80a6`
- **Archive File:** `clientpulse-api-v1.0.1.tar.gz`
- **Local SHA-256:** `29b8547165c83278ba9ce1e08fada2f4de3bdb3e7af5e7f538202eca5e364478`
- **Remote SHA-256:** `29b8547165c83278ba9ce1e08fada2f4de3bdb3e7af5e7f538202eca5e364478` (Matches)
- **Transfer Result:** Transferred via SCP on port 2222; verified byte-exact; loaded via `sudo docker load`.

---

## E. Migration Execution

- **Migration Mechanism:** Application migration engine (`/app/migrate`) executed within `clientpulse-api:v1.0.1`.
- **Transaction Semantics & Locking:**
  - Uses connection-level PostgreSQL advisory lock `pg_advisory_lock(982347102)`.
  - Migration executed inside a single explicit `BEGIN ... COMMIT` database transaction on that exact acquired connection.
  - Advisory lock unconditionally released in `defer` block.
- **Execution Log:**
  ```json
  {"time":"2026-11-02T19:34:37.60801425Z","level":"INFO","msg":"migration CLI initialized","command":"up","migrations_dir":"/app/migrations"}
  {"time":"2026-11-02T19:34:37.699721306Z","level":"INFO","msg":"migrations applied successfully","applied_count":1}
  ```
- **Migration Status Output:**
  ```text
  Version 000012 | notification_message_quality | [APPLIED at 2026-11-02T16:35:47Z]
  Version 000013 | assets_scheduler_rls_hardening | [APPLIED at 2026-11-02T19:34:37Z]
  ```

---

## F. Security Verification

### `pg_policies` Query Result
```text
 schemaname | tablename |        policyname         | permissive |         roles         |  cmd   | qual | with_check 
------------+-----------+---------------------------+------------+-----------------------+--------+------+------------
 public     | assets    | assets_definer_policy     | PERMISSIVE | {clientpulse_definer} | SELECT | true | 
 public     | assets    | assets_tenant_isolation   | PERMISSIVE | {public}              | ALL    | ...  | ...
 public     | monitors  | monitors_definer_policy   | PERMISSIVE | {clientpulse_definer} | ALL    | true | true
 public     | monitors  | monitors_tenant_isolation | PERMISSIVE | {public}              | ALL    | ...  | ...
```

### Verification Findings
1. **Least Privilege:** `assets_definer_policy` command is strictly `SELECT` (no write/insert/update/delete permissions for definer on assets).
2. **Tenant Isolation:** `assets_tenant_isolation` and `monitors_tenant_isolation` remain active and unchanged.
3. **RLS Configuration:** `relrowsecurity = true` and `relforcerowsecurity = true` verified on both `public.assets` and `public.monitors`.
4. **Role Isolation:**
   - `clientpulse_worker` querying `assets` directly under RLS yields 0 rows.
   - `clientpulse_app` querying `assets` directly without tenant session variable yields 0 rows.
5. **Scheduler Function Privileges:**
   - Function `claim_due_monitors` runs with `SECURITY DEFINER` as owner `clientpulse_definer`.
   - `search_path` is secured to `pg_catalog, public, pg_temp`.
   - Execution grant is restricted to `clientpulse_worker`.

---

## G. Scheduler Verification

The production monitor `e1bb1e87-0efd-41ef-bbb3-23d0be5584e1` (target: `https://api.hamtapesh.com/api/blog`, interval: 300s) was monitored in real-time. Three consecutive automatic scheduled runs were observed post-migration:

### Real-Time Scheduled Checks
| Run | Result ID | Scheduled At | Completed At | Result | HTTP Status | Latency | Worker Log |
|---|---|---|---|---|---|---|---|
| 1 | `eda894c8-d150-465a-af56-f05ed9f7c910` | `2026-11-02 19:35:42.991 UTC` | `2026-11-02 19:35:43.681 UTC` | success | 200 | 292 ms | `{"msg":"monitoring check executed and recorded","status":"up","latency_ms":292}` |
| 2 | `288eb415-df8b-49f5-8eb4-0f5158142218` | `2026-11-02 19:40:42.991 UTC` | `2026-11-02 19:40:43.867 UTC` | success | 200 | 478 ms | `{"msg":"monitoring check executed and recorded","status":"up","latency_ms":478}` |
| 3 | `ad80ea4a-6554-4fb5-b644-ab82abcafadd` | `2026-11-02 19:45:42.991 UTC` | `2026-11-02 19:45:43.784 UTC` | success | 200 | 395 ms | `{"msg":"monitoring check executed and recorded","status":"up","latency_ms":395}` |

- **Execution Cadence:** Exactly every 300 seconds (5 minutes, matching the monitor configuration) without jitter or failure.
- **Worker Status:** Heartbeat active, zero RLS or SQL errors.

---

## H. Protected Services Comparison

| Container | Baseline ID | Post-Deploy ID | Baseline Restarts | Post-Deploy Restarts | Status |
|---|---|---|---|---|---|
| `rocketchat-rocketchat-1` | `fee0c779ecaa` | `fee0c779ecaa` | 0 | 0 | Match / Running |
| `rocketchat-mongodb-1` | `66dcfc37b5fe` | `66dcfc37b5fe` | 0 | 0 | Match / Running |
| `rustdesk-hbbs` | `3b50e881dcfa` | `3b50e881dcfa` | 0 | 0 | Match / Running |
| `rustdesk-hbbr` | `cf8e3bbb029e` | `cf8e3bbb029e` | 0 | 0 | Match / Running |
| `fchat-mattermost-1` | `f6d1f3c1f425` | `f6d1f3c1f425` | 0 | 0 | Match / Running |
| `fchat-postgres-1` | `9f6887895acc` | `9f6887895acc` | 0 | 0 | Match / Running |
| `docker-main-mattermost-1` | `d2a7e61f0533` | `d2a7e61f0533` | 0 | 0 | Match / Running |
| `docker-main-postgres-1` | `f3a9dd7cba48` | `f3a9dd7cba48` | 0 | 0 | Match / Running |

- **Nginx Config:** `/etc/nginx/sites-available/mindu.ir` was verified untouched (timestamp: `2026-11-02 16:40:06 UTC`, syntax valid, service active).
- **Service Scope:** Only `clientpulse-api-1` was cleanly recreated to mount the new binary/migration bundle; PostgreSQL, Worker, and Web services remained uninterrupted.

---

## I. Rollback Strategy & Warnings

> [!WARNING]
> **DO NOT EXECUTE MIGRATION 000013 DOWN MIGRATION.**
> Reverting migration 13 drops `assets_definer_policy`, which strips the definer role of read permissions on `assets` and will reproduce the original TASK-INCIDENT-001 monitoring failure.

If an application rollback is needed:
1. Keep migration 13 applied in the database.
2. The previous API image is tagged on the host as `clientpulse-api:v1.0.0-pre-incident002`.
3. To roll back the API image only:
   ```bash
   sudo docker tag clientpulse-api:v1.0.0-pre-incident002 clientpulse-api:v1.0.0
   sudo docker compose -p clientpulse -f /home/ubuntu/clientpulse/docker-compose.production.yml up -d --no-deps api
   ```

---

## J. Remaining Risks & Observations

1. **Backups are Local:** As documented in the architecture overview, database backups currently reside on the VPS disk. An offsite disaster recovery sync should be planned before onboarding external paying clients.
2. **Zero Inconsistencies:** The database schema is fully aligned with version control at version 13, and the scheduler is healthy.

