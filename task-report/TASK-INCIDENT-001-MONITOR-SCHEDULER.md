# TASK-INCIDENT-001: Production Monitor Scheduler Investigation & Repair Report

**Project:** ClientPulse  
**Priority:** P0 — Production Monitoring Failure  
**Environment:** Production  
**Execution Date:** 2026-11-02  
**Target Domain:** https://mindu.ir  
**Target Monitor ID:** `e1bb1e87-0efd-41ef-bbb3-23d0be5584e1` (`Hamtapesh Api`)  
**Target URL:** `https://api.hamtapesh.com/api/blog`

---

## A. Executive Summary

### Problem Description
Following production deployment, the monitor created for `https://api.hamtapesh.com/api/blog` (Monitor ID: `e1bb1e87-0efd-41ef-bbb3-23d0be5584e1`) remained stuck indefinitely in `Pending First Check`. Manual checks triggered from the web application succeeded, and the endpoint was reachable, but automatic checks never initiated despite the worker container reporting `healthy` status and actively writing heartbeats.

### Root Cause
**Classification: F. Database / Locking Failure (Missing Cross-Tenant RLS Policy for Security Definer Role on Joined Table)**.  
The automatic monitor scheduler executes the stored function `claim_due_monitors()`, which runs with `SECURITY DEFINER` privileges under the role `clientpulse_definer`. In this function:
```sql
SELECT ... FROM public.monitors m JOIN public.assets a ON m.asset_id = a.id
```
Both `monitors` and `assets` have `FORCE ROW LEVEL SECURITY` enabled. While migration `000006_incident_heartbeat_monitoring.up.sql` created the bypass policy `monitors_definer_policy` on `monitors` for `clientpulse_definer`, **no corresponding bypass policy existed on `assets` for `clientpulse_definer`**. Consequently, whenever `claim_due_monitors` joined against `assets`, PostgreSQL enforced the default tenant isolation policy `(organization_id = current_setting('app.current_organization_id'))`. Because background worker scheduler connections have no HTTP session or tenant context set, `assets` returned 0 rows, resulting in `claim_due_monitors` continuously returning 0 eligible monitors.

In contrast, manual checks executed directly in the API container within an authenticated tenant HTTP session, where `app.current_organization_id` was explicitly populated, bypassing the join exclusion.

### Repair Status
**Successfully Repaired**. A non-destructive, non-schema-altering RLS policy (`assets_definer_policy`) was created for `clientpulse_definer`. Automatic checks immediately commenced without requiring container restarts, downtime, image rebuilds, or database data modifications.

### Current Monitor Status
- **Status:** `up` (Transitioned from `pending` / `Pending First Check`)
- **Last Checked At:** `2026-11-02 19:09:43.906836 UTC`
- **Next Check At:** `2026-11-02 19:10:42.991968 UTC`
- **Automatic Cadence:** 4 verified automatic executions at precise ~60-second intervals (HTTP 200, ~260–580ms latency).

---

## B. Production Baseline

Recorded prior to remediation on `2026-11-02 19:03:50 UTC`:

- **Host Uptime:** 108 days, 18 hours (Load Average: 0.41, 0.26, 0.20)
- **Host Time / UTC:** `Mon Nov 2 19:03:50 UTC 2026`
- **Memory:** 3.8 GiB total, 1.8 GiB used, 1.7 GiB available (Swap: 0B)
- **Disk Usage:** 70 GiB total, 52 GiB used (78%), 15 GiB available on `/dev/vda3`
- **Docker Daemon:** Active & healthy

### ClientPulse Container Baseline
| Container Name | Compose Service | Status | Health | Restart Count | Memory Usage / Limit |
|---|---|---|---|---|---|
| `clientpulse-worker-1` | `worker` | Up 2 hours | healthy | 0 | 7.24 MiB / 192 MiB (3.77%) |
| `clientpulse-api-1` | `api` | Up 2 hours | healthy | 0 | 144.8 MiB / 512 MiB (28.29%) |
| `clientpulse-web-1` | `web` | Up 2 hours | healthy | 0 | 55.01 MiB / 256 MiB (21.49%) |
| `clientpulse-postgres-1` | `postgres` | Up 2 hours | healthy | 0 | 54.35 MiB / 256 MiB (21.23%) |

No containers had experienced OOM termination (`OOMKilled: false`), restart loops, or resource exhaustion.

---

## C. Investigation Evidence

### 1. Worker Startup & Scheduler Logs
Worker container logs (`sudo docker logs clientpulse-worker-1`) confirmed that the worker process started and initialized both its heartbeat loop and monitor scheduler:
```json
{"time":"2026-11-02T16:43:07.381121991Z","level":"INFO","msg":"starting monitoring scheduler daemon","worker_id":"worker-57513d56eff8-1","concurrency":5,"poll_period":"1s","retention_days":90}
{"time":"2026-11-02T16:43:07.381408997Z","level":"INFO","msg":"worker daemon started, queue polling and heartbeat active"}
```
Worker heartbeat was continuously refreshed in `worker_heartbeats` every 2 seconds, verifying database connectivity and active loop execution.

### 2. Database State of Target Monitor Prior to Repair
```sql
SELECT id, organization_id, asset_id, status, is_active, next_check_at, last_checked_at, lease_acquired_at, lease_worker_id 
FROM monitors WHERE id = 'e1bb1e87-0efd-41ef-bbb3-23d0be5584e1';
```
Result:
- `id`: `e1bb1e87-0efd-41ef-bbb3-23d0be5584e1`
- `organization_id`: `1f21de76-603e-4ea1-8254-400a3300d210`
- `asset_id`: `f778f317-499a-4e99-8964-67df5313087b`
- `status`: `pending`
- `is_active`: `t`
- `next_check_at`: `2026-11-02 19:03:11.451634+00` (Elapsed, overdue)
- `last_checked_at`: `NULL`
- `lease_acquired_at`: `NULL`
- `lease_worker_id`: `NULL`

### 3. Stored Procedure Analysis & RLS Policy Gap
The scheduler claims due jobs using:
```sql
SELECT * FROM claim_due_monitors($1, $2, $3);
```
Executing this query directly in PostgreSQL as `clientpulse_worker` returned 0 rows.
Testing inside PostgreSQL by setting the execution role:
```sql
SET ROLE clientpulse_definer;
SELECT count(*) FROM monitors; -- returned 1
SELECT count(*) FROM assets;   -- returned 0!
```
Inspection of `pg_policies` revealed:
- `monitors` had `monitors_definer_policy` granting `clientpulse_definer` access to all rows (`USING (true)`).
- `assets` had only `assets_tenant_isolation` (`USING (organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::uuid)`).
- Because `claim_due_monitors` joins `public.monitors m JOIN public.assets a ON m.asset_id = a.id`, RLS evaluated against `assets` resulted in 0 matched assets, causing the entire join to return 0 rows.

### 4. Outbound Network Connectivity from Worker
Direct testing from inside `clientpulse-worker-1`:
- DNS resolution for `api.hamtapesh.com`: resolved to `185.213.164.146`
- TCP Port 443 handshake: connected successfully
- HTTP GET response: `HTTP/1.1 200 OK` (Content-Type: `application/json; charset=utf-8`)
Worker outbound egress was fully operational and unrestricted.

---

## D. Root Cause Classification

**Primary Classification:** `F. Database / Locking Failure` (Missing RLS Policy for Security Definer Role).  
**Confidence Level:** 100% (Confirmed with database queries, role simulation, and immediate runtime resolution).

### Why Manual Execution Worked While Automatic Execution Failed
1. **Manual Check:** Executed by `clientpulse-api-1` within a tenant-scoped HTTP request. The API middleware calls `SET LOCAL app.current_organization_id = '<org_id>'`. Under this setting, the tenant isolation policy on `assets` evaluated to `TRUE`, allowing the API to read the asset record and execute the check.
2. **Automatic Scheduler:** Executed by `clientpulse-worker-1` via the background daemon calling `claim_due_monitors()`. Worker connections do not have `app.current_organization_id` set (as the worker handles all tenants concurrently). While `monitors` had a definer policy, `assets` did not, so RLS on `assets` evaluated to `NULL` / `FALSE`, hiding the asset record and producing an empty join.

---

## E. Remediation

### Operation Performed
Created the missing RLS definer policy on table `assets` in PostgreSQL:
```sql
CREATE POLICY assets_definer_policy ON assets
    FOR ALL
    TO clientpulse_definer
    USING (true)
    WITH CHECK (true);
```

### Safety & Impact
- **Containers Affected:** None restarted. No downtime.
- **Data Alteration:** Zero data rows modified or deleted. No schema or column modifications.
- **Rollback Instructions (if ever required):**
  ```sql
  DROP POLICY IF EXISTS assets_definer_policy ON assets;
  ```

---

## F. End-to-End Automatic Monitoring Verification

Immediately following the policy creation, the worker's next tick claimed the monitor, executed the check, persisted the result, and updated the status.

### 1. Worker Scheduler Logs
```json
{"time":"2026-11-02T19:06:43.004960358Z","level":"INFO","msg":"monitoring check executed and recorded","monitor_id":"e1bb1e87-0efd-41ef-bbb3-23d0be5584e1","target_url":"https://api.hamtapesh.com/api/blog","result":"success","status":"up","latency_ms":583}
{"time":"2026-11-02T19:07:43.664997209Z","level":"INFO","msg":"monitoring check executed and recorded","monitor_id":"e1bb1e87-0efd-41ef-bbb3-23d0be5584e1","target_url":"https://api.hamtapesh.com/api/blog","result":"success","status":"up","latency_ms":263}
{"time":"2026-11-02T19:08:43.945647152Z","level":"INFO","msg":"monitoring check executed and recorded","monitor_id":"e1bb1e87-0efd-41ef-bbb3-23d0be5584e1","target_url":"https://api.hamtapesh.com/api/blog","result":"success","status":"up","latency_ms":532}
```

### 2. Database Persisted Automatic Check Results
```sql
SELECT id, monitor_id, execution_type, result, latency_ms, http_status_code, scheduled_at, completed_at, created_at
FROM check_results
WHERE monitor_id = 'e1bb1e87-0efd-41ef-bbb3-23d0be5584e1' AND execution_type = 'scheduled'
ORDER BY created_at DESC;
```
| Execution | Result ID | Type | HTTP Code | Latency | Scheduled At (UTC) | Completed At (UTC) |
|---|---|---|---|---|---|---|
| #1 | `77ef9799-6091-4e14-8109-1c2eb579d514` | scheduled | 200 | 583 ms | `19:03:11.451` | `19:06:42.976` |
| #2 | `290dc840-5810-426f-b44e-f5fc0631affa` | scheduled | 200 | 263 ms | `19:07:42.991` | `19:07:43.653` |
| #3 | `fbb6e21a-35e5-4394-870b-bb130a3c6f5e` | scheduled | 200 | 532 ms | `19:08:42.991` | `19:08:43.922` |
| #4 | `d384e100-fd6a-4475-97bb-8f8d76042581` | scheduled | 200 | 516 ms | `19:09:42.991` | `19:09:43.906` |

### 3. Cadence and Status Verification
- Measured check intervals: Exactly **60.67s**, **60.27s**, and **59.98s** (normal scheduler tick jitter).
- Target monitor status transitioned from `pending` to `up`.
- Web UI receives real-time `up` status via the API.

---

## G. Protected Services Compliance

All eight protected containers were continuously monitored throughout the incident response. Zero restarts occurred, and no modifications were made to their networks, ports, or volumes:

| Protected Container | State | Baseline Restarts | Final Restarts |
|---|---|---|---|
| `rocketchat-rocketchat-1` | running | 0 | 0 |
| `rocketchat-mongodb-1` | running | 0 | 0 |
| `rustdesk-hbbs` | running | 0 | 0 |
| `rustdesk-hbbr` | running | 0 | 0 |
| `fchat-mattermost-1` | running | 0 | 0 |
| `fchat-postgres-1` | running | 0 | 0 |
| `docker-main-mattermost-1` | running | 0 | 0 |
| `docker-main-postgres-1` | running | 0 | 0 |

---

## H. Observability Findings & Recommendations

### Findings
1. **Heartbeat vs Scheduler Health:** The API health check `/health/worker` evaluates the worker's persistent heartbeat timestamp in `worker_heartbeats`. A worker whose scheduler claims 0 rows due to an RLS policy or query mismatch still writes heartbeats every 2 seconds, displaying a misleading "healthy" state while monitoring is completely halted.
2. **Silent Zero-Claim Loop:** The worker scheduler poll loop does not log empty claim cycles at `INFO` level (only at `DEBUG`), making it impossible from standard production logs to determine whether 0 monitors were due or whether claiming failed silently.

### Recommendations for Future Roadmap
1. **Migration Update:** Add `CREATE POLICY assets_definer_policy ON assets FOR ALL TO clientpulse_definer USING (true) WITH CHECK (true);` to the repository migrations (e.g. migration `000005` or a subsequent migration) so that fresh environments automatically include this policy.
2. **Scheduler Telemetry:** Add a metric or periodic log reporting total active monitors vs monitors claimed per minute.
3. **Stale Pending Monitor Alert:** Add an alert or dashboard indicator when an active monitor remains in `pending` state for more than 3x its configured interval.

---

## I. Security & Compliance Statement

This investigation and remediation was conducted in strict adherence to security guidelines:
- No database secrets, passwords, tokens, API keys, or private keys were exposed or written to this report.
- The remediation was applied through standard PostgreSQL role-level security configuration without bypassing authorization boundaries.

