# TASK-008.B.3.1: Incident-Aware Dashboard Health & Legacy Resolution Display

**Project:** ClientPulse  
**Priority:** High — Operational Data Accuracy  
**Execution Environment:** Local Development Only  
**Production Deployment:** Strictly Prohibited  
**Base Release:** v1.0.2  
**Database Migration:** `000017_incident_lifecycle_resolution`  
**Related Tasks:** TASK-008.B.3, TASK-INCIDENT-004  
**Date:** November 3, 2026  
**Status:** Completed & Fully Verified  

---

## 1. Executive Summary

In accordance with specification `TASK-008.B.3.1`, the Next.js agency frontend has been updated to integrate the incident lifecycle semantics from `TASK-INCIDENT-004`. 

### Key Objectives Achieved:
1. **Executive System Health Bar Correction:** Corrected health state calculation so that scheduled background heartbeat incidents (`heartbeat_missed_incidents`) are strictly isolated and **never** misrepresented as website or API outages.
2. **Executive System Health State Precedence:** Implemented the strict 8-state semantic hierarchy: Data Unavailable $\rightarrow$ Website/API Outage $\rightarrow$ Heartbeat Attention $\rightarrow$ Degraded Monitoring $\rightarrow$ Incomplete Monitoring $\rightarrow$ No Active Monitoring $\rightarrow$ Empty Workspace $\rightarrow$ Operational.
3. **KPI Card Incident Distribution:** Preserved the 4 primary KPI cards while enhancing Card 3 (*Open Incidents*) with small supporting text distinguishing unacknowledged vs. acknowledged counts (`X unacknowledged · Y acknowledged` / `X در انتظار بررسی · Y تأییدشده`).
4. **Historical Incident Presentation (Legacy Resolution):** Corrected legacy incident presentation for historical records where `status = 'resolved'` and `resolution_type IS NULL`. Completely eradicated false inference of "Auto Recovery / Automatic", replacing it with the accurate non-inferred label:
   - English: `Legacy incident — resolution reason not recorded`
   - Persian: `رخداد قدیمی — علت بستهشدن ثبت نشده است`
5. **Zero Backend & Migration Footprint:** Strictly frontend integration in `apps/web/`. Zero backend Go services, worker code, database migrations, or production configurations were touched.

---

## 2. Previous Status-Calculation Defect

In `TASK-008.B.3`, the Executive System Health Bar derived overall state using:
```typescript
// Previous flawed derivation in page.tsx:
if (counts.down > 0 || counts.openIncidents > 0) return "outage";
```
Because `openIncidents` aggregated **all** open incident records, an open incident caused by a missed background cron ping (`heartbeat_missed`) caused `counts.openIncidents > 0` to evaluate to true. As a result:
- The dashboard presented a red outage banner: `Attention required — 0 target(s) experiencing confirmed outage`.
- Agency operators were misled into believing a customer's website or public API was down, when in reality only a scheduled background job had missed its ping interval.
- Paused monitors and pending monitors were not recognized in health bar precedence, potentially showing an unqualified operational state when all monitors were paused.

---

## 3. Final Status Precedence

The revised state derivation strictly enforces the following semantic hierarchy in [apps/web/src/app/page.tsx](file:///home/sadpc/Documents/ClientPulse/apps/web/src/app/page.tsx) and [apps/web/src/components/dashboard/SystemHealthBar.tsx](file:///home/sadpc/Documents/ClientPulse/apps/web/src/components/dashboard/SystemHealthBar.tsx):

| Precedence | State Identifier | Trigger Condition | English Display | Persian Display |
|---|---|---|---|---|
| **State 1** | `unavailable` | Required API stats fail to load (`error && !stats`) | **Data temporarily unavailable**<br>Widget temporarily unable to load monitoring data | **داده‌ها موقتاً در دسترس نیستند**<br>خطا در بارگذاری داده‌ها |
| **State 2** | `outage` | `website_outage_incidents > 0` OR `monitors_down > 0` | **Attention required — Website/API outage detected**<br>`{down} target(s) confirmed down · {open} open outage(s)` | **نیاز به بررسی و اقدام فوری — قطعی در وبسایت یا API شناسایی شده است**<br>`{down} هدف با قطعی تأییدشده · {open} رخداد قطعی باز` |
| **State 3** | `heartbeat_attention` | No website outages (`website_outage_incidents == 0 && monitors_down == 0`) AND `heartbeat_missed_incidents > 0` | **Background jobs require attention**<br>`{missed} background job incident(s) require attention · Website and API monitors remain unaffected` | **وظیفههای پسزمینه نیازمند بررسی هستند**<br>`{missed} رخداد وظایف پس‌زمینه نیازمند بررسی است · مانیتورهای وب‌سایت و API بدون اختلال فعال هستند` |
| **State 4** | `degraded` | No confirmed outages or heartbeat incidents, but `monitors_suspected_down > 0` | **Attention required — Suspected Down**<br>`{suspected} target(s) degraded / suspected down · {up}/{active} operational` | **نیاز به بررسی و اقدام فوری — مشکوک به قطعی**<br>`{suspected} هدف با اختلال / احتمال قطعی` |
| **State 5** | `incomplete` | Active monitors exist, but `monitors_pending > 0` or `monitors_unknown > 0` | **Monitoring setup in progress**<br>`{count} target(s) pending initial check or status unknown · {up} operational` | **پایش ناقص است**<br>`{count} هدف در انتظار اولین بررسی یا با وضعیت نامشخص · {up} عملیاتی` |
| **State 6** | `no_active` | All configured monitors are paused (`totalMonitors > 0 && activeMonitors == 0`) | **No active monitoring**<br>`All {total} configured monitors are currently paused` | **پایش فعالی وجود ندارد**<br>`تمامی {total} مانیتور تنظیم‌شده در حالت متوقف قرار دارند` |
| **State 7** | `empty` | Zero monitors configured (`totalMonitors == 0`) | **No monitoring configured**<br>`Configure your first website or API monitor...` | **هنوز هدفی برای پایش تنظیم نشده است**<br>`اولین وب‌سایت یا API خود را جهت آغاز پایش مداوم تعریف نمایید` |
| **State 8** | `operational` | All eligible active monitors have verified operational state (`monitors_up > 0 && down==0 && suspected==0 && pending==0 && unknown==0 && open_incidents==0`) and data is fresh | **All monitored targets operational**<br>`{up} of {active} monitored targets operational · {paused} paused` | **تمامی اهداف پایش فعال و در دسترس هستند**<br>`از {total} هدف پایش‌شده در دسترس هستند` |

*(Note: If dashboard data is stale, the `stale` warning badge or state is explicitly displayed; stale data is never presented as freshly operational.)*

---

## 4. Exact API Fields Consumed

The frontend interface [DashboardStats](file:///home/sadpc/Documents/ClientPulse/apps/web/src/app/page.tsx#L87-L107) directly consumes the aggregate fields returned by `GET /api/v1/dashboard/stats`:

```typescript
interface DashboardStats {
  total_clients: number;
  total_assets: number;
  monitors_up: number;
  monitors_down: number;
  monitors_suspected_down?: number;
  monitors_pending: number;
  monitors_paused: number;
  monitors_unknown?: number;
  // Specific Incident Lifecycle Aggregates from TASK-INCIDENT-004:
  open_incidents?: number;
  unacknowledged_incidents?: number;
  acknowledged_incidents?: number;
  resolved_incidents?: number;
  website_outage_incidents?: number;
  heartbeat_missed_incidents?: number;
  // Heartbeats:
  total_heartbeats?: number;
  heartbeats_healthy?: number;
  heartbeats_missed?: number;
  heartbeats_pending?: number;
  recent_checks: RecentCheck[];
}
```

---

## 5. Aggregate Consistency Findings

An inspection of the database schema and backend repository queries was performed:

1. **Unacknowledged vs. Acknowledged:**
   $$\text{open\_incidents} = \text{unacknowledged\_incidents} + \text{acknowledged\_incidents}$$
   In `check_result_repo.go`:
   - `open_incidents`: `COUNT(*) FILTER (WHERE status = 'open')`
   - `unacknowledged_incidents`: `COUNT(*) FILTER (WHERE status = 'open' AND acknowledged_at IS NULL)`
   - `acknowledged_incidents`: `COUNT(*) FILTER (WHERE status = 'open' AND acknowledged_at IS NOT NULL)`
   Because `acknowledged_at` is either NULL or NOT NULL, this equality is unconditionally guaranteed.

2. **Website Outages vs. Missed Heartbeats:**
   $$\text{open\_incidents} = \text{website\_outage\_incidents} + \text{heartbeat\_missed\_incidents}$$
   Inspection of `migrations/000006_incident_heartbeat_monitoring.up.sql` revealed check constraint:
   ```sql
   CONSTRAINT chk_incident_single_target CHECK (
       (monitor_id IS NOT NULL AND heartbeat_monitor_id IS NULL) OR
       (monitor_id IS NULL AND heartbeat_monitor_id IS NOT NULL)
   );
   ```
   **Finding:** Every incident row in the PostgreSQL schema is constrained to reference *either* a monitor (`monitor_id`) *or* a heartbeat monitor (`heartbeat_monitor_id`), and never both or neither. Thus, in the current data model, every open incident is guaranteed to belong to either `website_outage_incidents` or `heartbeat_missed_incidents`.
   The frontend gracefully handles any future hypothetical incident categories without forcing a broken equality.

---

## 6. Legacy Resolution Fallback Behavior

Prior to migration `000017`, the `incidents` table only stored `status = 'open' | 'resolved'` and `resolved_at`, without `resolution_type` or `resolved_by`. 

### The Problem
The earlier frontend code contained:
```tsx
// Flawed fallback:
{inc.resolved_by_name ? `${t.resolvedBy} ${inc.resolved_by_name}` : (isRTL ? "رفع خودکار" : "Auto Recovery")}
```
When historical records with `status = 'resolved'` and `resolution_type IS NULL` were loaded, they were falsely displayed as "Auto Recovery", misleading operators into believing the target recovered on its own.

### The Correction
1. **New Badge Variant:** Added `legacy_resolution` in [Badge.tsx](file:///home/sadpc/Documents/ClientPulse/apps/web/src/components/ui/Badge.tsx).
2. **Explicit Fallback Text:**
   - English: `Legacy incident — resolution reason not recorded`
   - Persian: `رخداد قدیمی — علت بستهشدن ثبت نشده است`
3. **Consistent Presentation across all views:**
   - **Incident List Table ([incidents/page.tsx](file:///home/sadpc/Documents/ClientPulse/apps/web/src/app/incidents/page.tsx)):** Displays the `legacy_resolution` badge in the status column and the fallback string in the *Resolved By* column.
   - **Incident Detail Header ([incidents/[id]/page.tsx](file:///home/sadpc/Documents/ClientPulse/apps/web/src/app/incidents/[id]/page.tsx)):** Renders `legacy_resolution` badge next to the status badge.
   - **Metadata Details Panel:** Displays `t.legacyResolutionReasonNotRecorded` under *Resolved At / By*.
   - **Diagnostic Panels:** Dedicated resolution info block explicitly communicates the historical origin.
   - **Diagnostic Recovery Check:** Informs the operator that diagnostics were not captured due to pre-v1.0.2 resolution tracking.
   - **Timeline:** Avoids fabricating artificial events or actors.
4. **Valid Resolution Types Preserved:**
   - `automatic_recovery` $\rightarrow$ `Automatic Recovery` / `رفع خودکار پس از بازیابی`
   - `manual` $\rightarrow$ `Manual Resolution` / `رفع دستی توسط اپراتور`
   - `source_monitor_deleted` $\rightarrow$ `Closed (Source Monitor Deleted)` / `بسته شده به دلیل حذف مانیتور`

---

## 7. Files Modified

| File Path | Description of Changes |
|---|---|
| [apps/web/src/lib/i18n.tsx](file:///home/sadpc/Documents/ClientPulse/apps/web/src/lib/i18n.tsx) | Added bilingual (EN/FA) translation entries for State 2 website outage, State 3 heartbeat attention, State 5 incomplete monitoring, State 6 no active monitoring, KPI unacknowledged/acknowledged subtitles, and legacy resolution fallback. |
| [apps/web/src/components/ui/Badge.tsx](file:///home/sadpc/Documents/ClientPulse/apps/web/src/components/ui/Badge.tsx) | Added `legacy_resolution` badge variant with `HelpCircle` icon, slate border, and mapped default localization key. |
| [apps/web/src/components/dashboard/SystemHealthBar.tsx](file:///home/sadpc/Documents/ClientPulse/apps/web/src/components/dashboard/SystemHealthBar.tsx) | Expanded `HealthOverallState` and config to handle `heartbeat_attention`, `incomplete`, and `no_active`. Replaced outage titles with `websiteOutageDetected` and separated heartbeat counts. |
| [apps/web/src/components/dashboard/PrimaryKpiCards.tsx](file:///home/sadpc/Documents/ClientPulse/apps/web/src/components/dashboard/PrimaryKpiCards.tsx) | Added supporting text to Card 3 (Open Incidents) displaying `X unacknowledged · Y acknowledged` (`unacknowledgedSub` / `acknowledgedSub`). |
| [apps/web/src/app/page.tsx](file:///home/sadpc/Documents/ClientPulse/apps/web/src/app/page.tsx) | Updated `DashboardStats` interface, enhanced `counts` useMemo to consume real API fields, re-ordered `overallState` strictly following the 8-state precedence, and moved `isStale` before use. |
| [apps/web/src/app/incidents/page.tsx](file:///home/sadpc/Documents/ClientPulse/apps/web/src/app/incidents/page.tsx) | Replaced automatic fallback with `legacy_resolution` badge and `legacyResolutionReasonNotRecorded` in the incidents list. |
| [apps/web/src/app/incidents/[id]/page.tsx](file:///home/sadpc/Documents/ClientPulse/apps/web/src/app/incidents/[id]/page.tsx) | Replaced automatic fallback in detail header badge, resolved-by info, recovery diagnostic box, and timeline. |
| [apps/web/tests/task-008-b3-validation.js](file:///home/sadpc/Documents/ClientPulse/apps/web/tests/task-008-b3-validation.js) | Added explicit assertions for heartbeat-only incident scenario in the existing acceptance suite. |
| [apps/web/tests/task-008-b3-1-validation.js](file:///home/sadpc/Documents/ClientPulse/apps/web/tests/task-008-b3-1-validation.js) | Comprehensive end-to-end regression suite covering all 16 scenarios and capturing visual artifacts. |

---

## 8. Verification & Test Results

### 8.1 TypeScript & ESLint Verification
```bash
cd apps/web
npx tsc --noEmit
# Exit Code: 0 (Zero type errors)

npm run lint
# Exit Code: 0 (ESLint passed cleanly with zero warnings)

npm run build
# Exit Code: 0 (Next.js Turbopack production build succeeded, all 25 routes compiled)
```

### 8.2 Execution of the 16 Regression Scenarios (`task-008-b3-1-validation.js`)
All 16 required regression scenarios were tested against the local development environment using realistic PostgreSQL database mutations:

| # | Scenario Description | Tested State | Result |
|---|---|---|:---:|
| **1** | Zero website outages and zero heartbeat incidents | State 8 (`operational`) — `All monitored targets operational` with paused monitor excluded | **PASS** |
| **2** | Zero website outages and one open heartbeat incident | State 3 (`heartbeat_attention`) — `Background jobs require attention`, Targets Down = 0 | **PASS** |
| **3** | One confirmed website outage and zero heartbeat incidents | State 2 (`outage`) — `Website/API outage detected` based on `website_outage_incidents` | **PASS** |
| **4** | Website outage plus heartbeat incident | State 2 (`outage`) — Confirmed website outage takes priority over heartbeat attention | **PASS** |
| **5** | One acknowledged heartbeat incident | State 3 remains active, Open Incidents KPI reflects `0 unacknowledged · 1 acknowledged` | **PASS** |
| **6** | One unacknowledged heartbeat incident | Open Incidents KPI reflects `1 unacknowledged · 0 acknowledged` | **PASS** |
| **7** | Suspected Down without confirmed incident | State 4 (`degraded`) — `Attention required — Suspected Down` | **PASS** |
| **8** | Pending and Unknown monitors | State 5 (`incomplete`) — `Monitoring setup in progress`, prevents false operational status | **PASS** |
| **9** | All monitors paused | State 6 (`no_active`) — `No active monitoring`, prevents false 100% operational status | **PASS** |
| **10** | Zero configured monitors | State 7 (`empty`) — `No monitoring configured` onboarding state with Create Monitor action | **PASS** |
| **11** | Stale dashboard data | Subtle staleness warning displayed; prevents unqualified fresh healthy state | **PASS** |
| **12** | Required API request failure | State 1 (`unavailable`) — `Data temporarily unavailable`, never renders operational | **PASS** |
| **13** | Legacy resolved incident (NULL `resolution_type`) | List and detail render `Legacy incident — resolution reason not recorded` without fake auto recovery | **PASS** |
| **14** | Manual resolution | `manual` resolution renders `Manual Resolution` badge and operator note | **PASS** |
| **15** | Automatic recovery | `automatic_recovery` resolution renders `Automatic Recovery` badge | **PASS** |
| **16** | Source monitor deletion resolution | `source_monitor_deleted` resolution renders `Closed (Source Monitor Deleted)` badge | **PASS** |

### 8.3 Acceptance & Regression Suites
1. **Extended Acceptance Suite (`task-008-b3-validation.js`):** Passed with 100% success (including the new explicit assertion that an open heartbeat incident never triggers a website outage).
2. **Guest & Authentication Regression Suite (`task-008-b2-1-regression.js`):** Re-run and passed with 100% success across all 9 audit dimensions (guest landing page isolation, session expiration, RBAC navigation, mobile drawer accessibility, etc.).

---

## 9. Screenshot Inventory

Saved in `task-report/screenshots/task-008-b3-1/`:

1. `01-healthy-dashboard.png` — State 8: All monitored targets operational with active counts and paused exclusion.
2. `02-website-outage.png` — State 2: Website/API outage detected based on confirmed monitor failures.
3. `03-heartbeat-only-incident.png` — State 3: Background jobs require attention; Targets Down remains 0.
4. `04-mixed-outage-and-heartbeat.png` — State 2 precedence: Outage priority over background heartbeat failures.
5. `05-legacy-resolved-incident-list.png` — Incident list table demonstrating the legacy resolution fallback badge and resolution column alongside manual, automatic, and source-deleted badges.
6. `06-legacy-resolved-incident-detail.png` — Incident detail page for a historical legacy incident showing non-inferred fallback in badges, metadata, resolution block, and diagnostics.
7. `07-persian-rtl-heartbeat-only.png` — Persian RTL layout showing `وظیفههای پسزمینه نیازمند بررسی هستند` with right-to-left layout and localized digits.

---

## 10. Known Limitations

- **Backend Aggregation Coupling:** The frontend relies on the `open_incidents`, `website_outage_incidents`, and `heartbeat_missed_incidents` aggregates returned by `GetDashboardStats` from `services/api`. In the rare event of transient network drops during individual widget polling, the frontend safely falls back to local incident arrays without throwing fatal unhandled exceptions.
- **Client-Side Refresh Interval:** The executive dashboard tracks last updated timestamps and subtle staleness after 10 minutes without refresh. Background jobs running on 5-minute intervals (e.g. Hamtapesh) will reflect upon automated or manual refresh.

---

## 11. Parallel Isolation & Safety Compliance

- **Backend & Worker Code:** Unchanged. Zero Go files modified.
- **Database Schema & Migrations:** Unchanged. No SQL files created or modified.
- **Production Files & Configs:** Untouched. No deployment or connection to production occurred.
- **Hamtapesh Interval:** Preserved at 5 minutes.
- **Version Integrity:** Base release remains v1.0.2 / migration 000017 in local development.

