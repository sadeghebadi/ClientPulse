# TASK-008.B.2 Implementation Report: Application Shell, Navigation & Guest Access Fix

**Project:** ClientPulse  
**Task ID:** TASK-008.B.2  
**Target:** Existing Next.js Frontend (`apps/web`)  
**Priority:** High  
**Execution Context:** Local Development Only  
**Production Deployment:** Strictly Prohibited (Preserved for separate release pipeline)  
**Reference Audit:** `task-report/TASK-008.A-UX-AUDIT.md`  
**Previous Task:** `TASK-008.B.1 — Design System Foundation`  
**Parallel Task:** `TASK-INCIDENT-003.P — Production Scheduler Observability Deployment`

---

## 1. Executive Summary

TASK-008.B.2 successfully addressed two major architectural challenges identified during the UX audit:
1. **The Guest Root-Route 401 Loop:** Resolved the issue where unauthenticated visitors visiting `mindu.ir/` or `localhost:3000/` encountered a session expiration redirect (`/login?session_expired=true`), preventing the public entry view from rendering.
2. **The Application Shell & Navigation Modernization:** Eliminated the legacy sidebar-in-container layout anti-pattern (where the navigation was constrained inside an arbitrary centered `max-w-7xl` wrapper). Implemented a full-viewport, deep slate SaaS application shell with responsive 3-mode navigation (desktop expanded, compact rail, and off-canvas mobile drawer), integrated workspace switcher, account menu, and seamless English (LTR) / Persian (RTL) support.

All work was strictly confined to frontend files under `apps/web/`. Zero modifications were made to backend APIs, workers, database migrations, Docker deployment files, or production environments. Uncommitted changes and concurrent scheduler tasks remained completely untouched.

---

## 2. Pre-Implementation Findings

Inspection of `apps/web/src/lib/api.ts`, `apps/web/src/lib/auth-context.tsx`, and `apps/web/src/app/page.tsx` revealed the root causes:
- **Global 401 Interception:** `apps/web/src/lib/api.ts` contained an un-scoped 401 response handler that dispatched `clientpulse:session_expired` whenever an API call returned 401, without checking if the user was on the root route `/` or if the request was an initial `/api/v1/auth/me` session probe.
- **Premature Protected Calls on Root:** While `apps/web/src/app/page.tsx` checked `if (!user || !activeOrg) return;` before loading stats and clients, the initial session probe (`/api/v1/auth/me`) returned 401 for guests. The 401 trigger dispatched `clientpulse:session_expired`, causing `apps/web/src/lib/auth-context.tsx` to immediately redirect the browser to `/login?session_expired=true`.
- **Layout Constraint:** `AppLayout.tsx` wrapped both the sidebar and main content in `max-w-7xl mx-auto px-4 ...`, resulting in a cramped sidebar floating inside a white page container, with inadequate horizontal space for data tables and metrics.

---

## 3. Architecture & Key Changes

### 3.1 Guest Root-Route Resolution (`lib/api.ts`)
- Modified the 401 interceptor in `apps/web/src/lib/api.ts`:
  - Added public path recognition: `/` is explicitly treated as a public entry path alongside `/login`, `/signup`, `/forgot-password`, `/reset-password`, `/verify-email`, and `/public`.
  - Added session probe check: `path.includes("/api/v1/auth/me")` is explicitly excluded from triggering session expired events. When `/api/v1/auth/me` returns 401, it cleanly indicates an unauthenticated visitor, setting `user = null` and allowing the guest view to render seamlessly.
  - Protected routes (e.g. `/clients`, `/monitors`, `/reports`) continue to intercept 401 and redirect to `/login?session_expired=true` as expected.

### 3.2 Responsive Application Shell (`components/AppLayout.tsx`)
- **Full Viewport Shell:** Replaced the centered container with a viewport-filling flex layout (`min-h-screen w-full flex overflow-x-hidden`).
- **Deep Slate Sidebar (`#0f172a`):**
  - High-contrast, restrained dark sidebar with slate typography and brand blue active accents.
  - Features ClientPulse brand logo and subtitle.
  - Active workspace identity card with direct link to organization switcher.
  - Four structured navigation groups matching the product capabilities:
    - **Overview:** Dashboard (`/`)
    - **Management:** Clients (`/clients`), Assets (`/assets`), Monitors (`/monitors`), Heartbeats (`/heartbeats`)
    - **Operations:** Incidents (`/incidents`), Maintenance (`/maintenance`), Reports (`/reports`)
    - **Settings:** Account (`/settings/account`), Branding (`/settings/branding` - Owner/Admin), Members (`/settings/members` - Owner/Admin), Notifications (`/settings/notifications`)
  - Collapse / Expand toggle button for desktop user control.
- **Application Header:**
  - Sticky top header with clean border and subtle elevation.
  - Mobile hamburger trigger (visible `<1024px`).
  - Breadcrumb / current context display (`ClientPulse / <Page>`).
  - Active workspace switcher dropdown with checkmark for current active organization and link to `/organizations`.
  - Locale toggle button (`English` / `فارسی`).
  - User account menu dropdown with avatar, display name, email, account settings link, and logout action.
- **Off-Canvas Mobile Drawer (`<1024px`):**
  - Slide-over drawer with backdrop blur overlay.
  - Automatically respects text direction: slides from left in LTR, and from right in Persian RTL.
  - Closes on outside click, route transition, or pressing `Escape`.
  - Accessible focus management with restoration to hamburger trigger upon closing.
  - Prevents background body scrolling while open.

### 3.3 RTL / LTR Bilingual Perfection
- Uses CSS logical properties (`ps-*`, `pe-*`, `ms-*`, `me-*`, `text-start`, `text-end`, `border-s`, `border-e`).
- Directional icons (chevrons, collapse/expand arrows) flip correctly in RTL mode using Tailwind `rtl:rotate-180`.
- Technical strings (URLs, emails, IDs) maintain proper LTR isolation with `.dir-ltr`.

---

## 4. Route & File Inventory

### 4.1 Permitted Files Modified
- [`apps/web/src/components/AppLayout.tsx`](file:///home/sadpc/Documents/ClientPulse/apps/web/src/components/AppLayout.tsx): Complete redesign into a full-height SaaS application shell with dark slate sidebar, responsive modes, header dropdowns, and mobile off-canvas drawer.
- [`apps/web/src/lib/api.ts`](file:///home/sadpc/Documents/ClientPulse/apps/web/src/lib/api.ts): Fixed 401 handling so that `/` and `/api/v1/auth/me` do not trigger session expiration redirects.
- [`apps/web/src/lib/i18n.tsx`](file:///home/sadpc/Documents/ClientPulse/apps/web/src/lib/i18n.tsx): Added localized navigation group labels (`navOverview`, `navManagement`, `navOperations`, `navSettings`, `switchWorkspace`, `userAccount`, `collapseSidebar`, `expandSidebar`) for English and Persian.

### 4.2 Test & Report Deliverables Created
- [`apps/web/tests/task-008-b2-validation.js`](file:///home/sadpc/Documents/ClientPulse/apps/web/tests/task-008-b2-validation.js): Automated Playwright validation suite verifying guest root route, authentication, responsive viewports, mobile drawer, active route highlighting, and dropdown menus.
- `task-report/TASK-008.B.2-IMPLEMENTATION.md`: This implementation report.
- `task-report/screenshots/task-008-b2/*.png`: 17 comprehensive validation screenshots.

---

## 5. Verification & Test Results

### 5.1 Static Checks & Production Build
```bash
cd apps/web
npx tsc --noEmit
npm run lint
npm run build
```
- **TypeScript:** Pass (`Finished TypeScript in 1287ms`, 0 errors).
- **ESLint:** Pass (0 warnings, 0 errors).
- **Turbopack Build:** Pass (`Generating static pages using 11 workers (25/25) in 463ms`).

### 5.2 Automated End-to-End Suite
Executed `node tests/task-008-b2-validation.js`:
- [x] **Guest Root Test:** Unauthenticated browser visits `http://localhost:3000/`. URL stays at `/`. No redirect to `/login?session_expired=true`.
- [x] **Zero Protected API Calls:** Verified that no calls to `/api/v1/dashboard/stats`, `/api/v1/clients`, or `/api/v1/maintenance-activities` were fired by unauthenticated guests.
- [x] **Authenticated Session:** Authenticated user registers/logs in and visits `/`. Session resolves, application shell renders, and dashboard data loads cleanly.
- [x] **Navigation Highlighting:** Verified across `/clients`, `/monitors`, `/incidents`, `/reports`, and `/settings/notifications`.
- [x] **Header Menus:** Workspace dropdown and user dropdown open and close properly with outside click and `Escape`.
- [x] **Mobile Drawer:** Drawer opens on mobile breakpoint (`390px`), displays navigation groups, and closes via `Escape` in both English and Persian RTL.

---

## 6. Screenshots Inventory

All screenshots are stored under `task-report/screenshots/task-008-b2/`:

| Screenshot | Description |
|---|---|
| `guest-root-en-1440px.png` | Public guest entry page at `/` in English (no redirect loop). |
| `guest-root-fa-1440px.png` | Public guest entry page at `/` in Persian RTL. |
| `shell-dashboard-en-1440px.png` | Authenticated dashboard inside full-viewport application shell (English LTR). |
| `shell-dashboard-fa-1440px.png` | Authenticated dashboard inside full-viewport application shell (Persian RTL). |
| `shell-1280px-laptop.png` | Application shell at 1280px laptop breakpoint. |
| `shell-1024px-tablet-landscape.png` | Application shell at 1024px tablet landscape breakpoint. |
| `shell-768px-tablet-portrait.png` | Application shell at 768px tablet portrait breakpoint. |
| `shell-390px-mobile.png` | Mobile view at 390px with top header, logo, and active workspace indicator. |
| `shell-360px-small-mobile.png` | Small mobile view at 360px. |
| `drawer-mobile-en-390px.png` | Mobile off-canvas navigation drawer open (English LTR, slides from left). |
| `drawer-mobile-fa-390px.png` | Mobile off-canvas navigation drawer open (Persian RTL, slides from right). |
| `shell-header-workspace-dropdown.png` | Active organization switcher dropdown in top application header. |
| `shell-header-user-dropdown.png` | User account profile and logout dropdown in top header. |
| `shell-route-clients.png` | Clients list route rendering inside application shell with active route pill. |
| `shell-route-monitors.png` | Monitors route rendering inside application shell. |
| `shell-route-incidents.png` | Incidents route rendering inside application shell. |
| `shell-route-reports.png` | Reports route rendering inside application shell. |
| `shell-route-notifications.png` | Notification settings route rendering inside application shell. |

---

## 7. Parallel Task Isolation & Boundary Confirmation

In strict compliance with parallel task isolation instructions:
- **Zero Backend Changes:** No files in `services/api/`, `services/worker/`, `migrations/`, or Docker configurations were modified.
- **Zero Production Operations:** No build containers, deployment scripts, or VPS configs were altered.
- **Zero Destructive Git Commands:** No hard resets, clean commands, or branch switching occurred. Working tree remains clean and fully isolated.
- **Preserved Concurrent Work:** `TASK-INCIDENT-003.P` files and state remain completely unimpacted.

---

## 8. Known Limitations & Next Scope
- The public root page currently presents interim guest-facing overview and authentication CTA cards; the full marketing landing page experience will be implemented under `TASK-010`.
- Data table mobile card views (for incidents, crons, and maintenance logs) remain in their standard tabular view; responsive card refactoring is scheduled for future feature-specific tasks.
