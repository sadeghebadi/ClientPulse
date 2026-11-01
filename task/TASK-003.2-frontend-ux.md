# TASK-003.2: Frontend UX & Functional Stabilization Specification

- **Task ID:** TASK-003.2
- **Scope:** Frontend Quality, UX, Visual Consistency, and End-to-End Functional Verification
- **Status:** Complete (Ready for Review)
- **Target Audience:** Engineering, Product, and QA

---

## 1. Executive Summary

TASK-003.2 stabilized and modernized the entire ClientPulse agency frontend into a production-grade, B2B SaaS dashboard. The previous frontend suffered from functional inconsistencies, non-standard styling, broken asset routing (404), unhandled SSRF errors, native blocking dialogs (`window.alert`, `window.confirm`), missing loading/error states, incomplete Persian/RTL mirroring, and missing tenant context resolution in database writes.

This phase established a unified visual design system, resolved 24 distinct audit findings (P0–P3), built complete asset and monitor management workflows, ensured bidirectional English/Persian internationalization, and verified seven complete end-to-end user journeys via automated headless browser automation.

---

## 2. Visual Design System

The visual design system adheres strictly to professional B2B SaaS standards (clean lines, high contrast, generous whitespace, semantic tokens, and no decorative animations or artificial metrics):

### 2.1 Color Tokens & Semantics
- **Background:** Neutral slate gray (`bg-slate-50`), soft container borders (`border-slate-200`).
- **Surfaces:** Pure white (`bg-white`), rounded cards (`rounded-2xl`), subtle elevations (`shadow-sm`).
- **Primary Action:** Solid Indigo/Blue (`bg-blue-600` hover `bg-blue-700`, text `text-white`).
- **Secondary Action:** Slate outline (`border-slate-300` text `text-slate-700` hover `bg-slate-50`).
- **Success Token:** Emerald (`bg-emerald-50 text-emerald-800 border-emerald-200`).
- **Warning Token:** Amber (`bg-amber-50 text-amber-800 border-amber-200`).
- **Danger / Error Token:** Rose/Red (`bg-rose-50 text-rose-800 border-rose-200`).
- **Typography:** Standard system sans-serif hierarchy: Headings (`font-extrabold tracking-tight`), Section titles (`font-bold text-slate-900`), Secondary metadata (`text-slate-500 text-xs`), Code/URLs (`font-mono text-xs`).

### 2.2 Reusable Shared Components (`apps/web/src/components/ui/`)
1. **`AppLayout.tsx`:** Standard desktop sidebar + sticky top navigation + mobile collapsible drawer.
2. **`Badge.tsx`:** Semantic badges with status-specific colors and tooltips for all monitor statuses (`up`, `down`, `suspected_down`, `pending`, `paused`, `unknown`), asset environments (`production`, `staging`, `development`), and RBAC roles (`owner`, `admin`, `member`, `viewer`).
3. **`Button.tsx`:** Accessible button with variants (`primary`, `secondary`, `outline`, `danger`, `ghost`), loading spinners, and duplicate submission prevention (`disabled={loading}`).
4. **`ConfirmModal.tsx`:** Accessible modal dialog replacing native `confirm()` for destructive operations (archiving clients, deleting monitors, removing members).
5. **`Toast.tsx`:** Context-based non-blocking notification toast provider (`useToast().success / error / info`).
6. **`MetricCard.tsx`:** High-contrast SaaS metric card with icons, real data, and directional drill-down links.
7. **`LoadingSkeleton.tsx`:** `MetricSkeleton` and `TableSkeleton` for seamless shimmer loading states without layout shift.
8. **`EmptyState.tsx`:** Helpful empty states with contextual icons, clear explanations, and direct primary action buttons.
9. **`PageHeader.tsx` & `Breadcrumbs.tsx`:** Consistent top page headers with icons, breadcrumb paths, and action buttons.
10. **`ErrorState.tsx`:** Accessible alert banner with humanized SSRF explanations and retry actions.

---

## 3. Route Inventory & Implementation Map

All 21 application routes are fully functional, typed, and compiled:

| Route Path | Category | Component File | Description & Stabilization Applied |
|---|---|---|---|
| `/` | Dashboard | `apps/web/src/app/page.tsx` | Answers 6 core questions, live monitor health overview bar, failed checks feed, quick actions. |
| `/clients` | Clients | `apps/web/src/app/clients/page.tsx` | Search, status filter, table skeleton, responsive cards, archive modal, toasts. |
| `/clients/new` | Clients | `apps/web/src/app/clients/new/page.tsx` | Form validation, locking submit button, accessible input names. |
| `/clients/[id]` | Clients | `apps/web/src/app/clients/[id]/page.tsx` | Client details, asset relations list, archive modal, real-time status badges. |
| `/clients/[id]/edit` | Clients | `apps/web/src/app/clients/[id]/edit/page.tsx` | Edit client fields, duplicate submit prevention, breadcrumbs. |
| `/assets` | Assets | `apps/web/src/app/assets/page.tsx` | **Created missing directory**: filters by client, type, environment, status; responsive cards. |
| `/assets/new` | Assets | `apps/web/src/app/assets/new/page.tsx` | URL validation, humanized SSRF error feedback, duplicate URL warning banner. |
| `/assets/[id]` | Assets | `apps/web/src/app/assets/[id]/page.tsx` | Asset details, associated monitors table, archive modal, create monitor CTA. |
| `/assets/[id]/edit` | Assets | `apps/web/src/app/assets/[id]/edit/page.tsx` | Edit asset configuration with URL format protection. |
| `/monitors` | Monitors | `apps/web/src/app/monitors/page.tsx` | Monitor directory, status filter, pause/resume quick actions, org timezone formatting. |
| `/monitors/new` | Monitors | `apps/web/src/app/monitors/new/page.tsx` | Monitor configuration with prefilled asset relation, interval selector, threshold guidance. |
| `/monitors/[id]` | Monitors | `apps/web/src/app/monitors/[id]/page.tsx` | Instant "Test Now", live check result banner, persisted check history, SSL cert info. |
| `/monitors/[id]/edit` | Monitors | `apps/web/src/app/monitors/[id]/edit/page.tsx` | Edit check intervals, assertions, timeout, and thresholds. |
| `/organizations` | Auth / Tenant | `apps/web/src/app/organizations/page.tsx` | Workspace management, tenant switcher, create new organization. |
| `/settings/account` | Settings | `apps/web/src/app/settings/account/page.tsx` | Argon2id password change, user profile details, email verification status. |
| `/settings/members` | Settings | `apps/web/src/app/settings/members/page.tsx` | Organization member management, role assignment, remove member modal. |
| `/login` | Authentication | `apps/web/src/app/login/page.tsx` | Standard email/password authentication, session expiration alert, redirect handling. |
| `/signup` | Authentication | `apps/web/src/app/signup/page.tsx` | User registration, auto-provisioned organization, password strength indicators. |
| `/forgot-password` | Authentication | `apps/web/src/app/forgot-password/page.tsx` | Rate-limited password reset link dispatching via Mailpit/SMTP. |
| `/reset-password` | Authentication | `apps/web/src/app/reset-password/page.tsx` | Token-validated password reset form with strength validation. |
| `/verify-email` | Authentication | `apps/web/src/app/verify-email/page.tsx` | Email verification token verification banner with resend CTA. |

---

## 4. Internationalization & RTL Specifications

- **Languages:** English (`en`) and Persian (`fa`).
- **Translation File:** `apps/web/src/lib/i18n.tsx`.
- **Direction Switching:** `document.documentElement.dir = isRTL ? "rtl" : "ltr"` and `document.documentElement.lang = locale`.
- **Hydration Safe:** Initial locale state is read lazily on the client without causing SSR hydration mismatches.
- **Mirroring:** Sidebar mirrors to the right side on desktop; table columns align to the right; forms and badges align cleanly to the right in Persian mode.

---

## 5. Security & SSRF Policy Preservation

- **SSRF Restrictions Uncompromised:** `ssrf.Guard` remains strictly enforced on the backend. Private IPs, loopback, internal ports, and invalid protocols are rejected.
- **Humanized Guidance:** Technical errors like `"restricted internal or private ip"` are mapped in `humanizeApiError()` to:
  > *"This URL points to a private or internal network address which is restricted for security. Please provide a publicly accessible website or API URL."*
- **Tenant Context Preservation:** All tenant mutations strictly require `app.current_organization_id` via PostgreSQL connection-scoped `WithTenantTx`, guaranteeing row-level security boundary enforcement.

