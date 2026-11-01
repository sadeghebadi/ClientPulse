# TASK-003.2 — Comprehensive Frontend UX & Functional Audit

**Project:** ClientPulse  
**Task:** TASK-003.2  
**Scope:** Frontend Quality, UX, Visual Consistency, End-to-End Functional Verification & Architecture  
**Audit Date:** 2026-11-01  
**Status:** Complete Audit — Prior to Redesign & Implementation  

---

## 1. Executive Summary

This audit evaluates all 21 frontend routes, components, state management contexts, API integrations, and localization/RTL behavior in the ClientPulse Next.js application (`apps/web`). The application implements an agency multi-tenant website monitoring dashboard. While the underlying Go backend API and PostgreSQL database are robust, secure, and feature-complete (hardened in TASK-003.1), the frontend interface presents critical architectural, functional, UX, accessibility, and visual shortcomings that hinder production readiness.

### Key Audit Metrics
- **Total Application Routes Audited:** 21 routes (including missing `/assets` listing route)
- **Total Identified Findings:** 24 distinct findings across 4 severity tiers
- **Severity Breakdown:**
  - **Critical (P0):** 4 findings (Broken mobile navigation, missing `/assets` route, API base URL environment variable mismatch, SSR/localStorage RTL hydration mismatch)
  - **High (P1):** 7 findings (Tenant switching desynchronization, native blocking `alert`/`confirm` dialogs, unhandled session expiration, unlocalized error messages / missing translation keys, duplicate form submissions, unhelpful SSRF error messages, missing dashboard failure highlights)
  - **Medium (P2):** 8 findings (Lack of responsive table cards on mobile, missing loading skeletons, missing empty states with guided CTAs, hardcoded text in dashboard and monitor views, lack of organization timezone formatting, unstyled focus indicators, unhandled touch targets < 44px, unsafe TypeScript `any` in client/asset/monitor edit pages)
  - **Low (P3):** 5 findings (Missing breadcrumbs on detail sub-pages, search debounce absence, lack of optimistic UI feedback, unstyled scrollbars in RTL, inconsistent border radius tokens)

---

## 2. Route Inventory & Status Matrix

| Route | View Name | Auth Required | Tenant Scoped | Current Status | Primary Issues Identified |
|---|---|---|---|---|---|
| `/` | Dashboard | Optional (Dual) | Yes (when authed) | Incomplete UX | Lacks monitor health overview breakdown, lacks dedicated recently failed checks section, lacks client summary panel, hardcoded English copy in stats table. |
| `/clients` | Clients Directory | Yes | Yes | Functional / Weak UX | Table cramped on mobile, filter button redundant/confusing, missing loading skeletons, uses generic text instead of localized badges. |
| `/clients/new` | Client Creation | Yes | Yes | Functional | Missing cancel confirmation if dirty, form can be double-submitted, error banner lacks structured recovery. |
| `/clients/[id]` | Client Details | Yes | Yes | Functional / Flawed UX | Uses native browser `confirm()` and `alert()` for archive/restore; no breadcrumb hierarchy; table cramped on narrow mobile. |
| `/clients/[id]/edit` | Client Editing | Yes | Yes | Functional / Typing Flaws | Uses `any` type for client response, does not lock inputs during save, lacks dirty-state warning. |
| `/assets` | Assets Directory | Yes | Yes | **MISSING (404)** | Route does not exist. Dashboard "View All" redirects to `/clients` instead of listing assets. No central place to filter assets by environment or type. |
| `/assets/new` | Asset Creation | Yes | Yes | Functional | SSRF errors displayed as raw technical backend messages; duplicate URL warning shown only post-submission; client dropdown select can default incorrectly. |
| `/assets/[id]` | Asset Details | Yes | Yes | Functional / Flawed UX | Uses native `confirm()` / `alert()`; lacks quick action to trigger immediate test; missing breadcrumbs; layout collapses on mobile. |
| `/assets/[id]/edit` | Asset Editing | Yes | Yes | Functional / Typing Flaws | Uses `any` type; SSRF validation error not humanized; inputs not disabled during save. |
| `/monitors` | Health Monitors | Yes | Yes | Functional / Weak UX | Pause/Resume uses native `alert()` on error; filter resets without indicator; no explanation of monitor state thresholds; table overflows on mobile. |
| `/monitors/new` | Monitor Creation | Yes | Yes | Functional | Complex technical fields (JSON Pointer, min/max status) lack inline helper tooltips; interval choices hardcoded without localized labels. |
| `/monitors/[id]` | Monitor Details | Yes | Yes | Functional / Weak UX | Test Now button lacks granular stage feedback; pause/resume uses native `alert()`; execution times not formatted in org timezone; lacks clear threshold explanations. |
| `/monitors/[id]/edit` | Monitor Editing | Yes | Yes | Functional / Typing Flaws | Uses `any` type; cannot change asset association; lacks validation error mapping. |
| `/login` | Log In | No | No | Functional | No auto-redirect if already logged in; generic error messages; layout lacks brand polish. |
| `/signup` | Sign Up | No | No | Functional | Password requirement (12 chars) lacks dynamic progress/checklist indicator; agency name field optional but unclear. |
| `/forgot-password` | Forgot Password | No | No | Functional | Form success state lacks resend cooldown timer; icon alignment in RTL inverted. |
| `/reset-password` | Reset Password | No | No | Functional | Token validation happens client-side without initial server token check; error display lacks retry CTA. |
| `/verify-email` | Email Verification | Optional | No | Functional | Initial token check triggers immediate state transition; resend verification lacks rate-limit cooldown indicator in UI. |
| `/organizations` | Organization Management | Yes | No | Functional / Flawed UX | Switching org does not refresh current view or redirect to dashboard; slug generation in creation form is manual; lacks member count in list. |
| `/settings/account` | Account Settings | Yes | No | Functional | Password change lacks strength meter; email unverified state lacks persistent banner on other pages. |
| `/settings/members` | Member Management | Yes | Yes | Functional / Security UX | Role descriptions not rendered in select; remove member uses native `confirm()`; lacks pending invite status. |

---

## 3. Detailed Audit Findings by Category

### Category A: Broken Pages, Missing Routes & Navigation (Critical / High)

#### Finding AUDIT-001 (Severity: Critical — P0)
- **Affected Route:** `/assets`
- **Description:** Missing Assets directory route.
- **Reproduction Steps:**
  1. Navigate to `/assets` in browser.
  2. Next.js returns 404 Not Found (`_not-found`).
  3. On Dashboard (`/`), click "View All" under "Total Websites & APIs" card.
  4. User is redirected to `/clients` instead of an assets directory.
- **Expected Behavior:** A dedicated `/assets` page displaying all registered assets across clients in the active organization, with filters for Client, Environment (`production`, `staging`, `development`), Type (`website`, `api`), and Status (`active`, `archived`), along with direct links to details and associated monitors.

#### Finding AUDIT-002 (Severity: Critical — P0)
- **Affected Route:** All Routes (`Header.tsx`)
- **Description:** Complete collapse and inaccessibility of navigation on mobile viewports (< 768px).
- **Reproduction Steps:**
  1. Resize browser viewport to 390px (iPhone 14) or 360px (Android).
  2. Inspect the top header.
  3. Notice that `Clients`, `Monitors`, `Team Members`, `Account Settings`, and `Active Workspace` are all styled with `hidden md:flex` or `hidden sm:flex`.
  4. There is no hamburger menu button, sliding drawer, or mobile navigation sheet.
  5. Logged-in users on mobile are completely trapped on whichever page they land on and cannot navigate to any dashboard section.
- **Expected Behavior:** Responsive navigation with a collapsible App Sidebar or accessible mobile drawer that provides instant navigation to all workspace sections, language switching, and user profile management on any screen size.

#### Finding AUDIT-003 (Severity: Critical — P0)
- **Affected Route:** Global (`apps/web/src/lib/api.ts`)
- **Description:** API Base URL environment variable mismatch causing runtime connectivity failures.
- **Reproduction Steps:**
  1. In `docker-compose.yml`, the environment variable passed to the Next.js container is `NEXT_PUBLIC_API_BASE_URL`.
  2. In `apps/web/src/lib/api.ts`:
     ```ts
     const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
     ```
  3. `api.ts` checks `NEXT_PUBLIC_API_URL` instead of `NEXT_PUBLIC_API_BASE_URL`.
  4. In environments where `NEXT_PUBLIC_API_BASE_URL` is configured to a remote or port-forwarded URL, `apiFetch` falls back to `http://localhost:8080`, causing requests from client browsers to fail.
- **Expected Behavior:** `api.ts` must consistently check `process.env.NEXT_PUBLIC_API_BASE_URL || process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"`.

---

### Category B: State Management, Session Handling & Tenant Isolation (High)

#### Finding AUDIT-004 (Severity: High — P1)
- **Affected Route:** `/organizations`, `/clients/[id]`, `/assets/[id]`, `/monitors/[id]`
- **Description:** Organization switching does not reset or reactively re-evaluate nested detail views.
- **Reproduction Steps:**
  1. Open a client detail page in Org A: `/clients/11111111-1111-1111-1111-111111111111`.
  2. Open another tab or switch organization to Org B via `/organizations`.
  3. Navigate back to the client detail page.
  4. The page attempts to fetch client `11111111-1111-1111-1111-111111111111` with Org B's active tenant session.
  5. The API enforces strict tenant isolation and returns 404 `CLIENT_NOT_FOUND`.
  6. The UI displays an unformatted raw error and leaves the user in an orphaned broken view without redirecting to `/clients`.
- **Expected Behavior:** Switching organizations should redirect to `/` (Dashboard) or reset the router view, and detail views must gracefully detect tenant mismatches with a clear "Resource does not belong to active organization" banner and a one-click CTA to return to the list.

#### Finding AUDIT-005 (Severity: High — P1)
- **Affected Route:** Global (`api.ts`, `auth-context.tsx`)
- **Description:** Unhandled 401 Unauthorized / Session Expiration causing silent UI freezing.
- **Reproduction Steps:**
  1. Log in to the application.
  2. Clear session cookie or let session expire (or revoke session on backend).
  3. Click "Create Client" or "Test Now".
  4. `apiFetch` throws `Error: HTTP_401` or `Unauthorized`.
  5. The error is either caught and shown in an alert/error box, or unhandled.
  6. The user remains in an apparent logged-in state in `AuthContext` while all backend mutations fail.
- **Expected Behavior:** `apiFetch` or `AuthContext` should intercept `401 Unauthorized` responses on authenticated endpoints, automatically clear local user state, dispatch a session expired toast notification, and redirect cleanly to `/login?session_expired=true`.

#### Finding AUDIT-006 (Severity: High — P1)
- **Affected Route:** `/clients/new`, `/clients/[id]/edit`, `/assets/new`, `/monitors/new`
- **Description:** Vulnerability to duplicate form submissions on rapid button click or double Enter key.
- **Reproduction Steps:**
  1. Fill in client creation form.
  2. Double-click "Add Client" rapidly or hit Enter multiple times.
  3. Because `setLoading(true)` is asynchronous and not guarded by an immediate ref/abort lock, two concurrent `POST /api/v1/clients` requests are dispatched with the same payload.
  4. Two identical client records are created in PostgreSQL.
- **Expected Behavior:** Forms must immediately disable the submit button, ignore subsequent enter presses during pending mutations, and implement submission idempotency guards.

---

### Category C: User Experience & Interaction Design (High / Medium)

#### Finding AUDIT-007 (Severity: High — P1)
- **Affected Route:** `/clients/[id]`, `/assets/[id]`, `/monitors`, `/monitors/[id]`, `/settings/members`
- **Description:** Use of synchronous, blocking browser native `window.alert()` and `window.confirm()`.
- **Reproduction Steps:**
  1. On `/clients/[id]`, click "Archive Client".
  2. Browser native modal `confirm(...)` blocks JavaScript execution.
  3. If an error occurs, `alert(err.message)` displays an unstyled system dialogue.
  4. In RTL mode (Persian), native dialogs are rendered in system LTR with no RTL layout mirroring or custom styling.
  5. Automated testing frameworks and screen readers have inconsistent interactions with synchronous window prompts.
- **Expected Behavior:** All destructive or irreversible actions must use accessible, styled Confirmation Modals (Dialog) with clear impact warnings, confirm/cancel buttons, and non-blocking asynchronous Toast Notifications for success/error feedback.

#### Finding AUDIT-008 (Severity: High — P1)
- **Affected Route:** `/assets/new`, `/assets/[id]/edit`
- **Description:** Technical SSRF error messages confuse users and do not provide actionable guidance.
- **Reproduction Steps:**
  1. On `/assets/new`, enter URL `http://192.168.1.1` or `http://169.254.169.254` or `http://localhost:3000`.
  2. Submit form.
  3. Backend correctly blocks request via `ssrf.Guard` and returns:
     `"url targets a restricted internal or private IP address"`.
  4. UI renders the raw technical string inside a red alert box without context.
- **Expected Behavior:** The UI must recognize SSRF validation error codes (`SSRF_DISALLOWED_IP`, `SSRF_INVALID_SCHEME`, `SSRF_DNS_LOOKUP_FAILED`) and present clear, friendly explanations:
  *"For security and privacy, monitoring private networks, localhost, or cloud metadata addresses is prohibited. Please provide a publicly resolvable domain (e.g. https://example.com) accessible over the public internet."*

#### Finding AUDIT-009 (Severity: High — P1)
- **Affected Route:** `/` (Dashboard)
- **Description:** Dashboard does not highlight recently failed checks or provide monitor health breakdown.
- **Reproduction Steps:**
  1. Create monitors, some passing, some failing (e.g. bad status code).
  2. Navigate to Dashboard.
  3. The "Recent Health Checks" table lists all recent checks mixed together.
  4. If 15 checks are passing and 1 failed, the operator has to visually scan each row to find the failure.
  5. There is no dedicated "Recently Failed Checks" section, no visual monitor health status distribution bar (Up vs Suspected Down vs Down vs Pending vs Paused), and no client summary panel.
- **Expected Behavior:** The dashboard must satisfy all 6 core questions defined in TASK-003.2 Section 3:
  1. Total clients.
  2. Total assets.
  3. Monitor health overview (Up / Suspected Down / Down / Pending / Paused / Unknown).
  4. Dedicated "Recently Failed Checks" panel showing incident context and error reasons.
  5. Latest monitoring activity table.
  6. Clients summary panel.
  7. Direct quick actions.

#### Finding AUDIT-010 (Severity: Medium — P2)
- **Affected Route:** `/monitors/[id]`, `/monitors`
- **Description:** Lack of clear status explanation for monitoring states (`suspected_down`, `down`, `pending`, `paused`).
- **Reproduction Steps:**
  1. Observe a monitor in `suspected_down` state.
  2. The badge shows "Suspected Down (1 Failure)".
  3. An agency user does not know what this means (is the site down? why hasn't an alert triggered? what is the threshold?).
- **Expected Behavior:** Provide clear contextual explanations and helper tooltips:
  - **Pending:** Waiting for first scheduled verification cycle.
  - **Operational (Up):** Last health check succeeded within expected status and assertion limits.
  - **Suspected Down:** 1 health check failed. Re-verifying in next cycle to prevent false positives before flagging full downtime.
  - **Downtime (Down):** 2 or more consecutive checks failed. Service is confirmed unreachable or returning error status.
  - **Monitoring Paused:** Scheduled checks are temporarily suspended.
  - **Unknown:** Check lease expired or health status could not be determined.

#### Finding AUDIT-011 (Severity: Medium — P2)
- **Affected Route:** `/monitors/[id]`, `/` (Dashboard)
- **Description:** Execution timestamps are formatted using browser locale time instead of organization timezone.
- **Reproduction Steps:**
  1. Create an organization with timezone `Asia/Tehran` or `America/New_York`.
  2. View check results on Monitor Details or Dashboard.
  3. Check timestamps use `new Date(chk.completed_at).toLocaleTimeString()`, which ignores the active organization's configured timezone and displays local machine time.
- **Expected Behavior:** Display execution times in the organization timezone (e.g., using `Intl.DateTimeFormat` with `timeZone: activeOrg.timezone || 'UTC'`), while displaying the relative time ("2 minutes ago") for immediate readability.

---

### Category D: Localization, Persian Translation & RTL (High / Medium)

#### Finding AUDIT-012 (Severity: Critical — P0)
- **Affected Route:** Global (`I18nProvider.tsx`, `layout.tsx`)
- **Description:** Server-Client hydration mismatch and missing mount synchronization for RTL direction.
- **Reproduction Steps:**
  1. Set language to Persian (فارسی). `localStorage` sets `clientpulse_locale = "fa"`.
  2. Refresh the browser page.
  3. `RootLayout` in `layout.tsx` hardcodes `<html lang="en" dir="ltr">`.
  4. On server render, Next.js generates LTR markup.
  5. On client mount, `useState` reads `"fa"`, setting the inner `div` to `dir="rtl"`, but `document.documentElement` remains `dir="ltr"` because `changeLocale` is never triggered on initial mount!
  6. The browser console logs hydration warnings regarding mismatched attributes on SSR vs CSR.
- **Expected Behavior:** Synchronize `document.documentElement.lang` and `document.documentElement.dir` on mount via `useEffect`, prevent hydration flash, and ensure all dialogs/portals inherit the proper `dir` attribute.

#### Finding AUDIT-013 (Severity: High — P1)
- **Affected Route:** Multiple Routes (`page.tsx`, `clients/page.tsx`, `monitors/new/page.tsx`, `Header.tsx`, `HealthChecker.tsx`)
- **Description:** Hardcoded English text not registered in `translations` dictionary.
- **Reproduction Steps:**
  1. Switch language to فارسی (FA).
  2. View Dashboard: "Phase 2 — Clients, Assets & Monitoring Core Active", "Manage", "View All", "Healthy", "Issues", "Live verification checks recorded by the Go worker and scheduler service.", "Refresh", "Success", "Failure", "Timeout", "Blocked", "View Monitor", "Status Code".
  3. View Clients page: "Filter", "Apply".
  4. View Monitor creation: "Target Website / Asset", "URL Override (Optional)", "Min Expected Status", "Max Expected Status", "Response Body Assertions (Optional)", "60 seconds (1 minute)".
  5. View Monitor details: "Interval", "Timeout", "Method", "Client: ", "Protocol & Cipher", "Dismiss".
  6. All above phrases remain in English despite Persian locale being active.
- **Expected Behavior:** Every user-facing string must be translated in both English and Persian dictionaries in `i18n.tsx`. Zero hardcoded user-facing strings.

#### Finding AUDIT-014 (Severity: Medium — P2)
- **Affected Route:** Global (Tables, Forms, Headers, Breadcrumbs)
- **Description:** Directional icons and arrow mirroring issues in RTL mode.
- **Reproduction Steps:**
  1. Switch language to Persian.
  2. Observe arrow icons in breadcrumbs and table action buttons (`ArrowRight` vs `ArrowLeft`).
  3. In several locations, `ArrowRight` is used statically without checking `isRTL`, pointing backward relative to reading order.
  4. Table headers and cell alignments in some subcomponents have explicit `text-left` or `text-right` instead of logical alignment (`text-start` or responsive RTL alignment).
- **Expected Behavior:** Use logical Tailwind classes or bidirectional components where arrow icons flip appropriately (`rotate-180` or conditional icon selection) and table columns align according to RTL text conventions.

---

### Category E: Responsive Design & Viewport Adaptation (High / Medium)

#### Finding AUDIT-015 (Severity: High — P1)
- **Affected Route:** `/clients`, `/monitors`, `/`, `/clients/[id]`
- **Description:** Unresponsive data tables overflowing viewports on mobile devices (390px / 360px).
- **Reproduction Steps:**
  1. Open `/clients` or `/monitors` on a mobile viewport (360px x 640px).
  2. Inspect the table.
  3. While `overflow-x-auto` prevents page blowout, table columns (Name, URL, Contact, Status, Actions) are extremely squished and require tedious horizontal swiping.
  4. Touch targets for "View Details" and "Pause" buttons are small (< 32px), violating touch accessibility guidelines.
- **Expected Behavior:** Provide a responsive hybrid table/card presentation:
  - On desktop and tablet (≥ 768px): Full-featured clean data table with aligned headers.
  - On mobile (< 768px): Responsive card-based layout displaying the item name, status badge, primary metadata, and full-width touch-friendly actions.

#### Finding AUDIT-016 (Severity: Medium — P2)
- **Affected Route:** `/clients/new`, `/assets/new`, `/monitors/new`, `/settings/account`, `/settings/members`
- **Description:** Form and card padding causes excessive horizontal cramping on narrow 360px viewports.
- **Reproduction Steps:**
  1. Open form pages on 360px viewport.
  2. Cards have `p-6 sm:p-8` along with container padding `px-4`.
  3. Available input width drops below 280px, making multi-column grid layouts awkwardly broken.
- **Expected Behavior:** Use responsive padding tokens (`p-4 sm:p-6 lg:p-8`) and single-column stack on screens < 640px.

---

### Category F: Visual System, Typography & Component Consistency (Medium)

#### Finding AUDIT-017 (Severity: Medium — P2)
- **Affected Route:** All Routes
- **Description:** Absence of unified shared component library resulting in scattered hardcoded styling.
- **Reproduction Steps:**
  1. Compare table headers between `/clients/page.tsx` and `/monitors/page.tsx`.
  2. Compare buttons across forms: some use `px-3 py-1.5`, some `px-4 py-2`, some `px-6 py-2.5`, with different hover colors, shadows, and focus rings.
  3. Loading states use `text-slate-500 text-sm` plain text instead of consistent skeleton loaders.
  4. Status badges duplicate color classes across 5 different files.
- **Expected Behavior:** Refactor all common UI elements into reusable components:
  - `Button`, `Input`, `Select`, `Textarea`
  - `Badge` (with variants: `up`, `down`, `suspected_down`, `pending`, `paused`, `unknown`, `active`, `archived`)
  - `DataTable` & `DataCard`
  - `MetricCard`
  - `PageHeader` & `Breadcrumbs`
  - `EmptyState`
  - `LoadingSkeleton`
  - `ConfirmModal`
  - `Toast` notification system

#### Finding AUDIT-018 (Severity: Medium — P2)
- **Affected Route:** Global (`globals.css`, `tailwind.config.ts`)
- **Description:** Lack of semantic design tokens and inconsistent neutral gray palettes.
- **Reproduction Steps:**
  1. Inspect color utility classes across components: mixes of `slate-50`, `slate-100`, `slate-200`, `slate-600`, `slate-700`, `slate-900` alongside occasional `gray-*` and `blue-*`.
  2. Primary action colors use hardcoded `bg-blue-600 hover:bg-blue-700` instead of semantic `primary` token.
- **Expected Behavior:** Define clean semantic tokens in Tailwind (`primary`, `surface`, `background`, `success`, `warning`, `error`, `text-muted`) to ensure visual coherence matching modern B2B SaaS design standards.

---

### Category G: Accessibility (A11y) & TypeScript Typing (Medium)

#### Finding AUDIT-019 (Severity: Medium — P2)
- **Affected Route:** All Form Inputs, Buttons, and Interactive Links
- **Description:** Inadequate keyboard focus visibility and missing accessible labels.
- **Reproduction Steps:**
  1. Navigate the dashboard using the `Tab` key.
  2. Action buttons in tables (e.g. Pause, Play, Refresh) show faint or invisible focus rings.
  3. Form inputs use `focus:outline-none focus:ring-2 focus:ring-blue-500` which works for mouse focus, but icon-only buttons lack `aria-label` attributes.
  4. Search inputs lack descriptive labels for screen readers.
- **Expected Behavior:** All interactive elements must have prominent, accessible focus rings (`focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-primary-500`), and icon-only buttons must carry explicit `aria-label` attributes in both languages.

#### Finding AUDIT-020 (Severity: Medium — P2)
- **Affected Route:** `clients/[id]/edit/page.tsx`, `assets/[id]/edit/page.tsx`, `monitors/[id]/edit/page.tsx`
- **Description:** Unsafe TypeScript `any` types in page components.
- **Reproduction Steps:**
  1. Inspect `EditClientPage`:
     ```ts
     const data = await apiFetch<{ client: any }>(`/api/v1/clients/${resolvedParams.id}`);
     ```
  2. Inspect `EditAssetPage`:
     ```ts
     const data = await apiFetch<{ asset: any }>(`/api/v1/assets/${resolvedParams.id}`);
     ```
  3. Inspect `EditMonitorPage`:
     ```ts
     const data = await apiFetch<{ monitor: any }>(`/api/v1/monitors/${resolvedParams.id}`);
     ```
  4. Any property rename or backend response mismatch bypasses TypeScript compiler checks.
- **Expected Behavior:** Replace all `any` usages with strictly typed interfaces matching the API schema.

---

## 4. Remediation Roadmap & Implementation Architecture

To ensure a structured, non-disruptive transformation of the ClientPulse frontend, the remediation will be executed in four coordinated phases:

```
┌────────────────────────────────────────────────────────┐
│ Phase A: Architecture, Design Tokens & Reusable UI     │
│ - Design Tokens, Shared Tailwind Palette               │
│ - Shared Components (Button, Modal, Toast, Badge, etc.)│
│ - Accessible Navigation (Sidebar + TopNav + Mobile)    │
│ - Complete English / Persian Dictionary & RTL sync     │
└──────────────────────────┬─────────────────────────────┘
                           │
┌──────────────────────────▼─────────────────────────────┐
│ Phase B: Route Implementation & UX Stabilization       │
│ - Build Missing /assets Route (Inventory, Filter, CTA) │
│ - Redesign Dashboard (6 core questions, failures panel)│
│ - Modernize Client & Asset Management (humanized SSRF) │
│ - Stabilize Monitor Views (threshold explanations)     │
│ - Replace all alert/confirm with ConfirmModal & Toasts │
└──────────────────────────┬─────────────────────────────┘
                           │
┌──────────────────────────▼─────────────────────────────┐
│ Phase C: Auth, Session & Multi-Tenancy Hardening       │
│ - Reactive Tenant Switching with Safe Routing          │
│ - Clean 401 Session Interception & Redirection         │
│ - Idempotent Form Submission Locks & Error Boundary    │
└──────────────────────────┬─────────────────────────────┘
                           │
┌──────────────────────────▼─────────────────────────────┐
│ Phase D: Verification, Lint, Build & E2E Testing       │
│ - ESLint Flat Config (0 errors/warnings)               │
│ - Next.js Production Build Verification                │
│ - Playwright End-to-End Test Suite (Journeys A through G)│
│ - Multi-viewport responsiveness (1440, 1024, 768, 390) │
└────────────────────────────────────────────────────────┘
```

---

## 5. Acceptance Criteria Checklist for Redesign
- [ ] All 21 inventory routes functional and accessible.
- [ ] Dedicated `/assets` listing route active with client and environment filters.
- [ ] No browser `alert()` or `confirm()` remains anywhere in the codebase.
- [ ] Zero unhandled console errors or hydration warnings.
- [ ] Mobile navigation fully functional with drawer on screens down to 360px.
- [ ] All data tables feature responsive mobile view (card fallback).
- [ ] Full English and Persian coverage with zero hardcoded user-facing strings.
- [ ] RTL layout mirrors correctly including sidebar, cards, tables, and icons.
- [ ] SSRF errors explained clearly to non-technical users without weakening security.
- [ ] Monitor status transitions (`up`, `down`, `suspected_down`, `pending`, `paused`, `unknown`) explained with clear operational threshold context.
- [ ] All numbers and metrics derived strictly from live backend endpoints.
- [ ] All 7 Playwright E2E User Journeys (A through G) pass reliably.

Audit completed and approved for implementation.

