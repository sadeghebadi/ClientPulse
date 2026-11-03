# TASK-008.B.2.1: Application Shell Authentication & Accessibility Regression Verification Report

**Project:** ClientPulse  
**Task ID:** TASK-008.B.2.1  
**Target:** Frontend (`apps/web/`) & Application Shell Verification  
**Execution Context:** Local Development Only (`http://localhost:3000`)  
**Production Deployment:** Strictly Prohibited  
**Related Task:** TASK-008.B.2  

---

## 1. Executive Summary

TASK-008.B.2.1 conducted an end-to-end acceptance and regression audit of the newly implemented Application Shell (`AppLayout.tsx`), public guest routing, session expiration interceptors, RBAC-aware navigation, and mobile drawer accessibility in ClientPulse.

All acceptance criteria have been verified and passed across real browser sessions (Chromium headless) against the live development backend and database containers. No regressions were introduced, no production systems were accessed or altered, and concurrent task artifacts were preserved intact.

---

## 2. Test Scope & Verification Results

### 2.1 Guest Session & Public Routes
- **Fresh Browser Context:** Navigating directly to `http://localhost:3000/` in an unauthenticated, fresh context loaded the guest landing overview cleanly.
- **Zero Protected API Calls:** Verified network logs confirmed 0 requests to `/api/v1/dashboard/stats`, `/api/v1/clients`, `/api/v1/assets`, `/api/v1/monitors`, or `/api/v1/incidents`.
- **No Redirect Loop:** URL remained `/` with HTTP 200 without redirecting to `/login?session_expired=true`.
- **Public Header Controls:** Verified Login (`/login`) and Signup (`/signup`) links are present and operational in the guest header.
- **Chrome-Free Public Pages:** Verified that public routes (`/login`, `/signup`, `/forgot-password`, `/reset-password`, `/verify-email`, `/public/reports/[token]`) do not render the authenticated application sidebar (`<aside>`), workspace selector, or member controls.

### 2.2 Expired Session Handling
- **Protected Page Invalidation:** Navigated authenticated user to `/clients`, deleted session in backend database (`sessions` table), and simulated API access.
- **Clean Redirect:** Verified immediate redirect to `/login?session_expired=true`.
- **Multi-Route Verification:** Confirmed identical redirect behavior on `/monitors`, `/reports`, and `/settings/notifications`.
- **Guest Distinction:** Confirmed that missing sessions on `/` remain a normal guest state, whereas missing/invalid sessions on protected routes trigger login redirection.

### 2.3 Organization & RBAC Navigation
- **Owner Account:** Sees all navigation groups including agency-restricted items: Branding (`/settings/branding`) and Members (`/settings/members`).
- **Admin Account:** Sees Branding (`/settings/branding`) and Members (`/settings/members`) alongside operational menus.
- **Member Account:** Management and operational routes visible; Branding and Members management items are strictly hidden.
- **Viewer Account:** Restricted management actions hidden; Branding and Members items are strictly hidden.
- **User Without Organization:** Successfully navigates to `/onboarding` ("Setup Your Agency Workspace" / "راه‌اندازی فضای کاری آژانس") with workspace creation form.
- **Workspace Switching:** Switching active workspace via `/organizations` and header dropdown cleanly updates active tenant context across the shell.

### 2.4 Mobile Drawer Accessibility (<1024px)
- **Trigger & Opening:** Mobile hamburger menu triggers open off-canvas drawer with `role="dialog"` and `aria-modal="true"`.
- **Body Scroll Lock:** Verified `document.body.style.overflow === "hidden"` when drawer is open.
- **Keyboard Navigation & Escape:** Pressing `Escape` closes the drawer and restores `overflow = ""`.
- **Backdrop Click:** Clicking the background overlay dismisses the drawer.
- **Focus Management:** Focus is restored properly after drawer dismiss.
- **Reduced Motion:** Verified transitions follow CSS standard styling without disorienting loops.

### 2.5 Visual Regression Across Breakpoints & Locales
Validated layouts across all six required breakpoints in both English (LTR) and Persian (RTL):
1. **1440px (Desktop):** Full sidebar expanded, multi-metric dashboard grids.
2. **1280px (Laptop):** High density navigation and content hierarchy preserved.
3. **1024px (Tablet Landscape):** Responsive transition breakpoint between sidebar and mobile drawer.
4. **768px (Tablet Portrait):** Sticky top header with hamburger trigger, centered content.
5. **390px (Standard Mobile):** Mobile drawer slides from left in English LTR, and from right in Persian RTL.
6. **360px (Small Mobile):** Compact mobile layout, zero horizontal overflow, clipped menus, or header collisions.

### 2.6 API Interceptor Inspection (`apps/web/src/lib/api.ts`)
- **Public Path Whitelist:** `/`, `/login`, `/signup`, `/forgot-password`, `/reset-password`, `/verify-email`, and `/public` are explicitly whitelisted.
- **Session Probe Exemption:** Requests matching `/api/v1/auth/me` are exempted from triggering session expiration events, allowing initial page loads to resolve guest state without redirect loops.
- **Protected 401 Interception:** Genuine 401 responses on protected paths dispatch `clientpulse:session_expired` to inform the `AuthProvider`.
- **CSRF Protection:** Mutating requests (`POST`, `PUT`, `PATCH`, `DELETE`) automatically attach `X-CSRF-Token` from cookies.

---

## 3. Test Commands & Evidence Inventory

### 3.1 Verification Execution Commands
```bash
# Static checks and compilation
cd apps/web
npx tsc --noEmit
npm run lint
npm run build

# Playwright regression test suite
node tests/task-008-b2-1-regression.js
```

### 3.2 Output Summary
- **TypeScript:** Passed with 0 errors.
- **ESLint:** Passed with 0 warnings/errors.
- **Turbopack Build:** 25/25 static pages compiled successfully.
- **Automated Regression Suite:** 100% passed across all sections.

### 3.3 Screenshot Deliverables
All verification screenshots are saved in `task-report/screenshots/task-008-b2-verification/`:

| Screenshot | Description |
|---|---|
| `01-guest-root-1440px-en.png` | Clean guest entry view at `/` in English (no redirect loop). |
| `02-public-login.png` | Login route `/login` rendered outside authenticated shell chrome. |
| `03-public-signup.png` | Signup route `/signup` rendered outside authenticated shell chrome. |
| `04-public-forgot-password.png` | Forgot Password route rendered outside authenticated shell chrome. |
| `05-public-reset-password.png` | Reset Password route rendered outside authenticated shell chrome. |
| `06-public-verify-email.png` | Email verification route rendered outside authenticated shell chrome. |
| `07-public-reports-token.png` | Public care report route rendered outside authenticated shell chrome. |
| `08-session-expired-redirect-login.png` | User redirected to `/login?session_expired=true` after session expiration. |
| `09-rbac-nav-owner.png` | Owner navigation showing full management, branding, and members options. |
| `10-rbac-nav-admin.png` | Admin navigation showing management, branding, and members options. |
| `10-rbac-nav-member.png` | Member navigation with restricted settings (Branding/Members) hidden. |
| `10-rbac-nav-viewer.png` | Viewer navigation with restricted settings (Branding/Members) hidden. |
| `11-user-without-org-onboarding.png` | Authenticated user without organization reaching onboarding page. |
| `12-mobile-drawer-open-accessibility.png` | Mobile drawer open at 390px with active backdrop and body scroll lock. |
| `13-viewport-1440px-desktop-en.png` | English layout at 1440px desktop breakpoint. |
| `13-viewport-1280px-laptop-en.png` | English layout at 1280px laptop breakpoint. |
| `13-viewport-1024px-tablet-landscape-en.png` | English layout at 1024px tablet landscape breakpoint. |
| `13-viewport-768px-tablet-portrait-en.png` | English layout at 768px tablet portrait breakpoint. |
| `13-viewport-390px-mobile-en.png` | English layout at 390px mobile breakpoint. |
| `13-viewport-360px-small-mobile-en.png` | English layout at 360px small mobile breakpoint. |
| `14-viewport-1440px-desktop-fa.png` | Persian RTL layout at 1440px desktop breakpoint. |
| `14-viewport-1280px-laptop-fa.png` | Persian RTL layout at 1280px laptop breakpoint. |
| `14-viewport-1024px-tablet-landscape-fa.png` | Persian RTL layout at 1024px tablet landscape breakpoint. |
| `14-viewport-768px-tablet-portrait-fa.png` | Persian RTL layout at 768px tablet portrait breakpoint. |
| `14-viewport-390px-mobile-fa.png` | Persian RTL layout at 390px mobile breakpoint. |
| `14-viewport-360px-small-mobile-fa.png` | Persian RTL layout at 360px small mobile breakpoint. |

---

## 4. Acceptance Criteria Checklist

- [x] **Guest root remains public:** Fresh sessions open `/` without 401 loop or protected API calls.
- [x] **Expired sessions on protected pages redirect correctly:** Redirects cleanly to `/login?session_expired=true`.
- [x] **Users without an organization reach onboarding:** Verified at `/onboarding`.
- [x] **RBAC navigation matches existing permissions:** Owner/Admin see branding/members; Member/Viewer do not.
- [x] **Public routes remain outside authenticated shell:** Verified across `/login`, `/signup`, `/forgot-password`, `/reset-password`, `/verify-email`, `/public/reports/[token]`.
- [x] **Mobile drawer accessibility works:** Body scroll locking, Escape closing, backdrop dismissal, and `role="dialog"` confirmed.
- [x] **RTL/LTR and all required breakpoints verified:** 1440px, 1280px, 1024px, 768px, 390px, 360px validated in English and Persian.
- [x] **TypeScript, lint, and build pass:** `tsc --noEmit`, `npm run lint`, and `npm run build` all pass without errors.
- [x] **Production remains untouched:** Execution strictly localized to dev environment.

---

## 5. Remaining Limitations

- Full marketing landing page content on `/` will be implemented under `TASK-010`.
- Data tables continue in tabular mode on mobile viewports; card refactoring is scheduled for future tasks.

