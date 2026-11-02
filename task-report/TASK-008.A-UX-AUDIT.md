# TASK-008.A: ClientPulse Full Product UI/UX Audit & Redesign Blueprint

**Document Version:** 1.0.0  
**Audit Date:** November 2, 2026  
**Auditor:** Lead SaaS UI/UX Architect (Agentic Coding Team)  
**Status:** COMPLETE (READ-ONLY AUDIT & SPECIFICATION)  
**Target Output Document:** `task-report/TASK-008.A-UX-AUDIT.md`  
**Dedicated Screenshot Directory:** `task-report/screenshots/task-008-audit/` (45 captured screenshots)

---

## Executive Summary

A comprehensive visual, structural, and user experience audit was conducted across the entire ClientPulse B2B SaaS application. Every frontend route, reusable component, layout wrapper, state machine, and bilingual presentation (English LTR and Persian RTL) was inspected in detail.

The primary objective of this audit is to provide a concrete, step-by-step redesign blueprint to transform ClientPulse into an internationally competitive, premium SaaS product tailored for digital web agencies managing 5 to 500+ client websites.

### Key Audit Metrics
- **Total Frontend Routes Audited:** 32 routes (100% cataloged and tested).
- **Audit Screenshots Captured:** 45 high-resolution images covering Desktop (1440px), Laptop (1280px), Tablet Landscape (1024px), Tablet Portrait (768px), Mobile (390px), and Small Mobile (360px).
- **Critical Architectural UX Defects Uncovered:**
  - **[P0] Guest Session Expiration Loop:** Visiting the root route unauthenticated automatically triggers an API 401 on dashboard stats, instantly redirecting the user to `/login?session_expired=true` and completely concealing the guest landing page.
  - **[P0] Broken Mobile Table Overflows:** 5 core operational pages (Incidents, Heartbeats, Maintenance, Reports, Delivery History) lack responsive card layouts and overflow off-screen on devices under 768px.
  - **[P1] Hardcoded Text Alignment (`text-left` / `text-right`) in Persian RTL:** Flips column headers and action buttons into contradictory alignments across all table views.
  - **[P1] System Font Fallback for Persian:** Missing dedicated Persian typography (e.g., Vazirmatn), causing clipped glyphs and poor vertical rhythm.
  - **[P1] Information Overload & Misplaced Infra Checker:** Agency dashboard renders internal backend container health checks (Go worker, Postgres, Redis, Mailpit) instead of focusing strictly on client website uptime.
  - **[P1] Missing Monitoring Data Confused with 100% Uptime:** Unmonitored or pending assets lack distinctive visual cues, risking false confidence.

---

## Section 1: Complete Page Inventory

| Route | Page Type | Auth / Role | Key APIs Used | Visual Structure & Summary |
|---|---|---|---|---|
| `/` | Dashboard / Guest | Public / Auth | `GET /api/v1/dashboard/stats`, `GET /api/v1/clients`, `GET /api/v1/maintenance-activities` | Authenticated: Dark gradient hero with 6 crammed buttons, 6 metric cards, monitor status bar, failed checks table, activity feed, clients panel, and server health checker. Guest: Obscured due to redirect bug. |
| `/login` | Auth | Public | `POST /api/v1/auth/login` | Centered 440px card with blue icon box, raw `<input>` elements, remember/forgot link, submit button, error banner. |
| `/signup` | Auth | Public | `POST /api/v1/auth/signup` | Centered 440px card with 4 fields, live 12-char password validator checkmark, organization name field. |
| `/forgot-password` | Auth | Public | `POST /api/v1/auth/forgot-password` | Single-field form with anti-enumeration success state and back-to-login button. |
| `/reset-password` | Auth | Public | `POST /api/v1/auth/reset-password` | Token verification form with new password, confirmation, and error alert. |
| `/verify-email` | Auth | Public / Auth | `POST /api/v1/auth/verify-email`, `POST /api/v1/auth/resend-verification` | Status card processing token from URL query params. Dispatches auto-verification on mount. |
| `/onboarding` | Main / Setup | Auth (No Org) | `POST /api/v1/organizations` | Agency onboarding wizard collecting Org Name, Slug, Default Locale, Timezone. |
| `/organizations` | Settings | Auth | `GET /api/v1/auth/me`, `POST /api/v1/organizations` | Split layout: Left 2/3 shows list of member organizations with active badges and switch buttons; Right 1/3 provides new organization provisioning form. |
| `/clients` | Main | Auth | `GET /api/v1/clients` | Filter bar (search, active/archived tabs), Add Client CTA, desktop table and mobile card view. |
| `/clients/new` | Main | Auth (Admin) | `POST /api/v1/clients` | Two-column form for client name, contact person, email, website URL, notes. |
| `/clients/[id]` | Main | Auth | `GET /api/v1/clients/[id]` | Client summary card, associated assets list, associated monitors list, archive/restore controls. |
| `/clients/[id]/edit`| Main | Auth (Admin) | `PATCH /api/v1/clients/[id]` | Client editing form with status toggle. |
| `/assets` | Main | Auth | `GET /api/v1/assets` | Directory of monitored web assets with client badges, environments (prod/staging/dev), URL preview. |
| `/assets/new` | Main | Auth (Admin) | `POST /api/v1/assets` | Asset registration form with client dropdown, environment selection, URL validation. |
| `/assets/[id]` | Main | Auth | `GET /api/v1/assets/[id]` | Asset details, active monitors attached, latest SSL certificate details, delete modal. |
| `/assets/[id]/edit`| Main | Auth (Admin) | `PATCH /api/v1/assets/[id]` | Asset editing form. |
| `/monitors` | Main | Auth | `GET /api/v1/monitors` | Status filter pills (All, Up, Down, Suspected Down, Paused, Pending), test execution trigger, quick pause/resume. |
| `/monitors/new` | Main | Auth (Admin) | `POST /api/v1/monitors` | Comprehensive monitor configuration: asset selector, interval, timeout, method, assertions. |
| `/monitors/[id]` | Main | Auth | `GET /api/v1/monitors/[id]`, `GET /api/v1/monitors/[id]/history`, `POST /test`, `POST /pause` | Executive monitor dashboard: test now button, pause/resume, SSL handshake details card, chronological execution history table. |
| `/monitors/[id]/edit`| Main | Auth (Admin) | `PATCH /api/v1/monitors/[id]`| Monitor configuration editor. |
| `/incidents` | Main | Auth | `GET /api/v1/incidents`, `POST /acknowledge` | Multi-filter incident directory (status, target type, ack state), failure category badges, duration timers. |
| `/incidents/[id]` | Main | Auth | `GET /api/v1/incidents/[id]`, `POST /acknowledge` | Incident post-mortem view: failure diagnostics, root cause summary, ack operator details, timeline. |
| `/heartbeats` | Main | Auth | `GET /api/v1/heartbeat-monitors`| Cron job & worker heartbeat list: status (healthy, missed, pending), interval, grace period, last ping. |
| `/heartbeats/new` | Main | Auth (Admin) | `POST /api/v1/heartbeat-monitors`| Heartbeat generator form with cron/interval selector and curl ping instructions. |
| `/heartbeats/[id]` | Main | Auth | `GET /api/v1/heartbeat-monitors/[id]`, `POST /rotate-secret` | Heartbeat operational view: secret reveal toggle, curl ping snippet generator, ping history. |
| `/heartbeats/[id]/edit`| Main | Auth (Admin)| `PATCH /api/v1/heartbeat-monitors/[id]`| Heartbeat editor. |
| `/maintenance` | Main | Auth | `GET /api/v1/maintenance-activities`, `POST /api/v1/maintenance-activities` | Agency work log: client, asset, activity title, category, duration, date performed. |
| `/reports` | Main | Auth | `GET /api/v1/reports`, `POST /preview`, `POST /generate` | Care report directory, client filter, modal wizard with previous month preset and live metrics preview. |
| `/reports/[id]` | Main | Auth | `GET /api/v1/reports/[id]`, `POST /share` | High-fidelity agency care report view: uptime %, incidents breakdown, maintenance activity, PDF download, public share modal. |
| `/public/reports/[token]`| Public View | Public | `GET /public/reports/[token]` | Standalone branded customer report viewer without app chrome; includes print/PDF trigger. |
| `/settings/account`| Settings | Auth | `GET /api/v1/auth/me`, `POST /change-password`, `POST /resend-verification` | Operator profile, verified email state, password update form. |
| `/settings/branding`| Settings | Auth (Admin) | `GET /api/v1/settings/branding`, `PATCH /api/v1/settings/branding`, `POST /logo` | White-label settings: agency logo upload, primary/accent color picker, contrast ratio validator, footer text. |
| `/settings/members`| Settings | Auth (Admin) | `GET /api/v1/organizations/current/members`, `POST /members`, `DELETE /members` | Team directory with RBAC roles (Owner, Admin, Member, Viewer) and invite modal. |
| `/settings/notifications`| Settings | Auth (Admin) | `GET /api/v1/settings/notifications/destinations`, `GET /deliveries`, `POST /telegram/token`, `POST /email` | Tabbed settings: Telegram Private/Group connection with approval code workflow, email verification, event preference matrix, delivery history audit table. |

---

## Section 2: Screenshot Catalog Reference

All screenshots are stored in `task-report/screenshots/task-008-audit/`:

1. `01_guest_homepage.png` — Guest landing state (illustrates the redirect issue).
2. `02_login_page.png` — Login card layout.
3. `03_signup_page.png` — Registration flow with live validation.
4. `04_forgot_password_page.png` — Forgot password interface.
5. `05_reset_password_page.png` — Password reset form.
6. `06_verify_email_page.png` — Email verification handler.
7. `07_onboarding_page.png` — Initial agency workspace onboarding.
8. `08_dashboard_authenticated.png` — Full authenticated dashboard (Desktop 1440px).
9. `09_monitors_list.png` — Monitors table view.
10. `10_monitor_detail.png` — Monitor details with SSL handshake card and execution history.
11. `11_monitor_new.png` — Create monitor configuration form.
12. `12_incidents_list.png` — Incident directory with filter pills.
13. `13_incident_detail.png` — Incident diagnostic timeline and acknowledge controls.
14. `14_clients_list.png` — Client directory with search and tabs.
15. `15_client_detail.png` — Client profile and attached assets/monitors.
16. `16_client_new.png` — Client creation form.
17. `17_assets_list.png` — Monitored assets directory.
18. `18_asset_detail.png` — Asset details view.
19. `19_asset_new.png` — New asset form.
20. `20_heartbeats_list.png` — Cron and background worker heartbeats directory.
21. `21_heartbeat_detail.png` — Heartbeat curl snippets and rotation modal.
22. `22_heartbeat_new.png` — New heartbeat configuration form.
23. `23_maintenance_list.png` — Logged agency maintenance activities.
24. `24_reports_list.png` — Monthly care reports directory.
25. `25_report_detail_en.png` — High-fidelity agency report (English).
26. `26_report_detail_fa.png` — High-fidelity agency report (Persian).
27. `27_settings_branding.png` — Agency white-label branding and logo upload.
28. `28_settings_account.png` — Account settings and security.
29. `29_settings_members.png` — Team members and RBAC roles.
30. `30_organizations_list.png` — Organization switcher and provisioning.
31. `31_settings_notifications_destinations.png` — Telegram & Email notification channels.
32. `32_settings_notifications_history.png` — Notification delivery logs.
33. `33_persian_rtl_dashboard.png` — Dashboard in Persian RTL mode.
34. `34_persian_rtl_monitors.png` — Monitors directory in Persian RTL mode.
35. `35_persian_rtl_reports.png` — Reports directory in Persian RTL mode.
36. `36_persian_rtl_notifications.png` — Notifications interface in Persian RTL mode.
37. `37_responsive_1440_dashboard.png` — Desktop layout (1440px).
38. `38_responsive_1280_dashboard.png` — Laptop layout (1280px).
39. `39_responsive_1024_dashboard.png` — Tablet landscape layout (1024px).
40. `40_responsive_768_dashboard.png` — Tablet portrait layout (768px).
41. `41_responsive_390_dashboard.png` — Mobile layout (390px).
42. `42_responsive_360_dashboard.png` — Small mobile layout (360px).
43. `43_responsive_390_mobile_nav.png` — Mobile navigation drawer.
44. `44_responsive_390_monitors_table.png` — Mobile monitors card view.
45. `45_responsive_390_report_detail.png` — Mobile care report view.

---

## Section 3: Design System Audit & Token Deficiencies

### 3.1 Color System Inconsistencies
- **Dual Primary Brands:** The application alternates between Tailwind `blue-600` (`#2563eb`) on Auth, Dashboard, and Navigation, and `indigo-600` (`#4f46e5`) on Care Reports and Public Views. `Input.tsx` uses `focus:ring-indigo-500`, clashing with `blue-600` buttons.
- **Surface Palette Bleed:** Pages indiscriminately mix `slate` (`bg-slate-50`, `border-slate-200`) and `gray` (`bg-gray-50`, `border-gray-200`).
- **Dark Mode Drift:** `darkMode: "class"` is defined in `tailwind.config.ts`, but no theme provider exists. Orphaned dark mode classes (e.g., `dark:bg-slate-900` in `Input.tsx`) cause styling inconsistencies.

### 3.2 Typography & Font Deficiencies
- **Missing Persian Web Font:** Uses `-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto`. In Persian mode, browsers fall back to OS Arabic fonts, causing uneven baselines and clipped descenders.
- **Tabular Numerics Absent:** Numeric displays (latencies, status codes, dates) lack `tabular-nums`, leading to visual jitter during metric updates.
- **Scale Irregularities:** Headers range from `text-3xl font-extrabold` to `text-base font-bold` without a standard typographic scale.

### 3.3 Layout & Structural Limitations
- **Sidebar-in-Container Anti-Pattern:** The sidebar is rendered inside a `max-w-7xl` container instead of functioning as a true full-height application viewport sidebar. On large screens (1440px+), the main operational area is constrained to ~1000px.
- **Dark Sidebar Evaluation:** Adopting a deep neutral slate sidebar (`#0f172a` or `#1e293b`) with light content (`#f8fafc`) creates clear visual hierarchy, distinguishing navigation from tenant data.

### 3.4 Component Duplication & Gaps
- **Input Bypassing:** Forms on `login`, `signup`, `forgot-password`, `onboarding`, and `clients/new` bypass `Input.tsx` in favor of raw `<input>` elements.
- **Monolithic Badge Component:** `Badge.tsx` spans 320 lines of hardcoded switch cases, lacking a unified structure for border styling, padding, and iconography.
- **Missing UI Primitives:** Dropdowns, Tooltips, Tabs, Data Tables, and Date Pickers are implemented ad-hoc across pages.

---

## Section 4: Critical UX Problems (Severity Classification)

### [P0] Critical Usability & Functional Blockers
1. **Root Route Guest Redirect Loop:** Unauthenticated visitors to `http://localhost:3001/` trigger an unauthorized API call that redirects them to `/login?session_expired=true`, hiding the guest marketing banner.
2. **Mobile Table Cutoff:** Pages for Incidents, Heartbeats, Maintenance, Reports, and Notification History lack mobile card views and overflow horizontally on viewports under 768px.

### [P1] High Priority UX Defects
1. **Persian RTL Table Inversion:** Hardcoded `text-left` and `text-right` classes cause column headers and action buttons to misalign in RTL mode.
2. **Missing Persian Font Family:** Absence of Vazirmatn leads to inconsistent text rendering and line-height issues in Persian mode.
3. **Internal Server Health Checker on Tenant Dashboard:** Infrastructure status (worker, Postgres, Mailpit) is exposed on the client dashboard rather than an administrative view.
4. **Hero Banner CTA Clutter:** The dashboard banner packs six buttons into a single row, creating cognitive overload.
5. **False Uptime Perception for Pending Data:** Monitors in `unknown` or `pending` status lack clear warnings, risking confusion with 100% uptime.

### [P2] Improvements & Usability Enhancements
1. **Dual Primary Color Scheme:** Inconsistent use of `blue-600` and `indigo-600` across views.
2. **Manual Directional Icon Swapping:** Inconsistent ternary checks for `ArrowLeft`/`ArrowRight` result in inverted navigation arrows.
3. **Latin Digits in Persian UI:** Numbers and dates in Persian mode render in ASCII Latin digits rather than Persian formats.
4. **Ad-Hoc Tab Switching:** Navigation tabs across pages use differing styles and active states.

### [P3] Cosmetic Refinements
1. **Metric Skeleton Layout Shift:** `MetricSkeleton` renders 4 cards while the dashboard renders 6, causing layout shift on load.
2. **Native Tooltip Latency:** Browser native `title` attributes cause delays and are inaccessible on touch devices.

---

## Section 5: Proposed Design System & Token Architecture

```
Token Hierarchy:
Global Tokens (Primitive Colors, Spacing, Radii)
   └── Semantic Tokens (Brand, Surface, Text, State)
         └── Component Tokens (Button, Table, Card, Input)
```

### Proposed Color Palette
```css
/* Core Brand */
--brand-50:  #eff6ff;
--brand-100: #dbeafe;
--brand-500: #3b82f6;
--brand-600: #2563eb; /* Primary Accent */
--brand-700: #1d4ed8;

/* Neutral Surfaces */
--surface-bg:      #f8fafc;
--surface-card:    #ffffff;
--surface-sidebar: #0f172a; /* Dark Sidebar */
--surface-border:  #e2e8f0;

/* Status Semantics */
--status-operational: #10b981; /* Emerald */
--status-degraded:    #f59e0b; /* Amber */
--status-down:        #ef4444; /* Rose */
--status-paused:      #64748b; /* Slate */
--status-missing:     #8b5cf6; /* Violet */
```

### Proposed Typography
- **Latin Font:** `Inter`, system fallback
- **Persian Font:** `Vazirmatn`, system fallback
- **Monospace Font:** `JetBrains Mono`, `Fira Code`
- **Tabular Figures:** Enabled globally for numerical metrics (`font-variant-numeric: tabular-nums`).

---

## Section 6: Dashboard Redesign Blueprint

### Operational Hierarchy
1. **Executive Health Bar:** Instant status indication ("All 42 Monitored Sites Operational" or "1 Site Down • 2 Open Incidents").
2. **Primary Metric Cards (4 KPIs):** Total Sites, Active Outages, Open Incidents, Average 24h Response Latency.
3. **Operational Split View:**
   - **Main Area (65%):** Monitor status distribution, active incident alerts, recent failed checks feed.
   - **Sidebar Area (35%):** Heartbeat summary, recent maintenance activities, care reports overview.
4. **Relocation:** Remove the internal infrastructure health checker from the tenant dashboard.

### Proposed ASCII Wireframe

```
+---------------------------------------------------------------------------------------+
| [ClientPulse]   [Active Agency: Nexus Labs v]          [EN | FA]  [🔔 2]  [Operator v] |
+---------------------------------------------------------------------------------------+
| SIDEBAR     |  BREADCRUMB: Agency Dashboard                                            |
|             |-------------------------------------------------------------------------|
| [⊞] Overview|  [🟢 ALL SYSTEMS OPERATIONAL — 48/48 Targets Monitored | Latency: 142ms] |
| [👥] Clients |-------------------------------------------------------------------------|
| [🌐] Assets  |  +---------------+  +---------------+  +---------------+  +---------------+  |
| [⚡] Monitors|  | TOTAL SITES   |  | SITES DOWN    |  | OPEN INCIDENTS|  | 24H AVG LATENCY|  |
| [💓] Crons   |  | 48            |  | 0             |  | 0             |  | 142 ms        |  |
| [⚠️] Incidents|  +---------------+  +---------------+  +---------------+  +---------------+  |
| [🛠️] Mainten.|-------------------------------------------------------------------------|
| [📄] Reports |  LIVE MONITOR HEALTH OVERVIEW                         | HEARTBEAT STATUS|
| [⚙️] Settings|  [46 Up] [1 Degraded] [0 Down] [1 Paused]              | 12 Healthy      |
|             |-------------------------------------------------------| 0 Missed        |
|             |  ACTIVE INCIDENTS & RECENT FAILURES                   |-----------------|
|             |  +---------------------------------------------------+| RECENT MAINT.   |
|             |  | No active outages recorded in the last 24 hours.  || Client A: WP Upd|
|             |  +---------------------------------------------------+| Client B: Plugin|
|             |-------------------------------------------------------|-----------------|
|             |  RECENT MONITORING VERIFICATIONS                      | CARE REPORTS    |
|             |  Target URL         Status  Latency  Checked At       | Oct 2026: Ready |
|             |  api.client.com     200 OK   84ms    14:22:01         | [Generate Report|
+---------------------------------------------------------------------------------------+
```

---

## Section 7: Monitoring & Incident Experience Proposal

### Status Vocabulary
- **Operational:** Target responds with expected HTTP status and assertions; latency within threshold.
- **Degraded:** First failure awaiting confirmation, high latency (>3000ms), or SSL expiring in <14 days.
- **Down:** Multi-check confirmed outage, connection refused, or 5xx response.
- **Paused:** Manually paused by operator; checks suspended.
- **Data Missing:** New monitor with no recorded checks or missed worker execution.
- **Heartbeat Missed:** Scheduled ping failed to arrive within expected window and grace period.
- **Heartbeat Recovered:** First successful ping received following a missed state.

---

## Section 8: Care Reports UX Proposal

1. **Step-by-Step Generator Modal:** Client selection -> Period selector (Previous Month / Custom) -> Metrics preview.
2. **Instant Metrics Preview:** Calculates and validates uptime %, incidents count, and maintenance logs prior to PDF generation.
3. **White-Label Customization:** Agency logo, brand primary color, custom disclaimer, and operator contact info.
4. **Persian PDF & RTL Support:** RTL layout rendering with embedded Vazirmatn font, Jalali dates, and Persian numerals.
5. **Interactive Public Viewer:** Dedicated shareable view (`/public/reports/[token]`) with client-facing presentation and print styles.

---

## Section 9: Notifications Experience Proposal

1. **Telegram Channels:**
   - **Private Chat:** 1-click deep-link authentication.
   - **Group Chat:** Bot connection workflow with visible 6-digit confirmation code and approval status badge.
2. **Email Verification:** Prominent pending verification banner with inline token submission modal.
3. **Event Notification Matrix:** Checkbox grid for configuring alerts (Down, Degraded, SSL Expiry, Heartbeat Missed, Report Ready).
4. **Delivery History Drawer:** Searchable delivery log with failure details (e.g., Telegram rate limits, SMTP rejections).

---

## Section 10: Responsive & Mobile Findings

- **1440px Desktop:** Fluid sidebar with fixed 240px width; content area scales cleanly up to 1600px.
- **1024px Tablet Landscape:** Collapsible sidebar into compact icon rail (64px) to preserve data table density.
- **768px Tablet Portrait:** Sidebar converts to off-canvas drawer; tables maintain card views where needed.
- **390px & 360px Mobile:**
  - Replace desktop tables with structured mobile cards across all operational lists.
  - Full-width touch-friendly form buttons and input fields.
  - Active workspace indicator visible in the mobile navigation header.

---

## Section 11: Persian RTL & English LTR Strategy

1. **CSS Logical Properties:** Replace `text-left`, `text-right`, `pl-*`, `pr-*` with `text-start`, `text-end`, `ps-*`, `pe-*`.
2. **Direction-Aware Layout:** Set `dir="rtl"` or `dir="ltr"` on the root `<html>` element based on active locale.
3. **Bilingual Typography:** Apply `font-vazirmatn` for Persian and `font-inter` for English.
4. **BiDi Text Handling:** Wrap monospaced technical strings (URLs, error logs) in `<bdi>` or `dir="ltr"` containers within Persian layouts.

---

## Section 12: Accessibility & Usability (WCAG 2.1 AA)

1. **Color Contrast:** Elevate badge and text contrast ratios to meet the 4.5:1 minimum threshold.
2. **Keyboard Navigation & Focus Trapping:** Ensure modals trap focus and restore it on close; add visible focus rings (`focus-visible:ring-2`).
3. **Dual Status Indicators:** Pair color indicators with descriptive text and distinct iconography.
4. **Screen Reader Landmarks:** Implement appropriate `<main>`, `<nav>`, `<aside>`, and `aria-live` attributes.

---

## Section 13: Landing Page & Public Website Planning

### Proposed Site Map
- **Homepage:** Agency value proposition, features overview, live demo preview, client testimonials, primary CTA.
- **Features:** Website uptime, SSL monitoring, heartbeat tracking, white-label PDF care reports, Telegram alerts.
- **How It Works:** 3-step onboarding guide (Connect Asset -> Automated Verification -> One-Click Reports).
- **Pricing:** Agency-focused tiers based on monitor volume, check frequency, and white-labeling features.
- **Login / Register:** Unified access point with existing authentication flows.
- **Contact & Sales:** Lead capture form for agency inquiries.
- **FAQ:** Common agency questions regarding SSRF security, notification delivery, and report customization.

---

## Section 14: Platform Admin Panel Design Preparation

### Design Direction
Distinct from the tenant dashboard, using a deep slate/navy palette (`#090d16`) with an amber administrative badge.

### Core Modules
1. **User Management:** Search operators, view verification status, manage account states.
2. **Organization Management:** Multi-tenant directory, resource consumption metrics, tenant isolation auditing.
3. **Plan & Subscription Controls:** Monitor quotas, check intervals, trial management.
4. **Payments & Revenue Overview:** Billing metrics, invoice tracking, subscription statuses.
5. **System Health & Outbound Queues:** Relocated infrastructure health dashboard, Mailpit/SMTP queue metrics, delivery logs.

---

## Section 15: Prioritized Implementation Roadmap

| Phase | Milestone | Scope | Complexity |
|---|---|---|---|
| **Phase 1** | Foundation & Tokens | Centralized Tailwind config, color tokens, Vazirmatn font, primitive components (`Button`, `Input`, `Badge`). | Medium |
| **Phase 2** | Root Fix & Layout | Resolve P0 guest redirect issue; implement dark sidebar application shell and mobile drawer. | Medium |
| **Phase 3** | Operational Dashboard | Redesign main dashboard with operational focus; relocate backend health checker. | High |
| **Phase 4** | Monitoring & Incidents | Standardize status vocabulary; add mobile card layouts for Incidents and Heartbeats. | Medium |
| **Phase 5** | Care Reports Overhaul | Improve generator modal, metric preview, Persian PDF styling, and public viewer. | High |
| **Phase 6** | Notifications Polish | Telegram group approval flow, email verification banner, delivery history drawer. | Medium |
| **Phase 7** | Public Marketing Site | Implement public landing, features, pricing, and FAQ pages. | Medium |
| **Phase 8** | Platform Admin Console | Build administrative dashboard for tenant, user, and subscription management. | High |

---

## Section 16: Final Stop Condition Verification

- Application source code: **Unmodified**
- Database schemas and data: **Unmodified**
- Docker configuration and containers: **Unmodified**
- Concurrent tasks (TASK-007.A and TASK-006.3.1): **Protected and uninterrupted**
- Screenshots directory: `task-report/screenshots/task-008-audit/` (45 images)
- Deliverable: Complete audit report written to `task-report/TASK-008.A-UX-AUDIT.md`.

*Awaiting explicit user review and approval prior to beginning implementation of TASK-008.*

