# ClientPulse Frontend Route Inventory & Audit Report

**Task Reference:** TASK-003.2  
**Date:** 2026-11-01  
**Scope:** Complete Route Inventory, Verification Status, API Interactions, and UX Assessment.

---

## 1. Route Inventory Overview

ClientPulse frontend is built with Next.js App Router (TypeScript, Tailwind CSS).
Below is the exhaustive inventory of all 21 user-facing routes, their authentication requirements, API dependencies, and current audit evaluation.

| # | Route | Method/Type | Auth | Tenant | API Endpoints Called | Audit Status | Key Remediation |
|---|---|---|---|---|---|---|---|
| 1 | `/` | Page (SSR/CSR) | Dual | Yes | `GET /api/v1/dashboard/stats`, `GET /health/*` | Incomplete UX | Redesign with 6 core questions, failures panel, client overview, and live health metrics. |
| 2 | `/clients` | Page (CSR) | Yes | Yes | `GET /api/v1/clients?search=&status=` | Needs Polish | Add mobile card view, loading skeleton, badge tokens. |
| 3 | `/clients/new` | Page (CSR) | Yes | Yes | `POST /api/v1/clients` | Functional | Add double-submit lock, accessible form feedback. |
| 4 | `/clients/[id]` | Page (CSR) | Yes | Yes | `GET /api/v1/clients/:id`, `GET /api/v1/assets?client_id=:id`, `POST /archive`, `POST /restore` | Flawed UX | Replace native `confirm()` with ConfirmModal, add breadcrumbs, add mobile cards. |
| 5 | `/clients/[id]/edit` | Page (CSR) | Yes | Yes | `GET /api/v1/clients/:id`, `PATCH /api/v1/clients/:id` | Functional | Remove `any` typing, add double-submit protection. |
| 6 | `/assets` | Page (CSR) | Yes | Yes | `GET /api/v1/assets?client_id=&type=&environment=&is_active=` | **MISSING (404)** | Create full page with filters, client selection, environment pills, and mobile view. |
| 7 | `/assets/new` | Page (CSR) | Yes | Yes | `GET /api/v1/clients?status=active`, `POST /api/v1/assets` | Needs UX Fix | Translate & humanize SSRF errors, lock submit. |
| 8 | `/assets/[id]` | Page (CSR) | Yes | Yes | `GET /api/v1/assets/:id`, `POST /archive`, `POST /restore` | Flawed UX | Replace `confirm()` with ConfirmModal, add quick Test Now action, add breadcrumbs. |
| 9 | `/assets/[id]/edit` | Page (CSR) | Yes | Yes | `GET /api/v1/assets/:id`, `PATCH /api/v1/assets/:id` | Functional | Remove `any`, humanize SSRF errors, lock inputs. |
| 10 | `/monitors` | Page (CSR) | Yes | Yes | `GET /api/v1/monitors?status=`, `POST /pause`, `POST /resume` | Needs Polish | Replace `alert()` with Toasts, add threshold guidance, mobile card view. |
| 11 | `/monitors/new` | Page (CSR) | Yes | Yes | `GET /api/v1/assets?is_active=true`, `POST /api/v1/monitors` | Functional | Add inline guidance for assertions, localize intervals, lock submit. |
| 12 | `/monitors/[id]` | Page (CSR) | Yes | Yes | `GET /api/v1/monitors/:id`, `GET /history`, `POST /test`, `POST /pause`, `POST /resume` | Flawed UX | Format times in org timezone, explain status thresholds, replace `alert()` with Toasts/Modals. |
| 13 | `/monitors/[id]/edit` | Page (CSR) | Yes | Yes | `GET /api/v1/monitors/:id`, `PATCH /api/v1/monitors/:id` | Functional | Remove `any`, lock form, validate input boundaries. |
| 14 | `/login` | Page (CSR) | No | No | `POST /api/v1/auth/login`, `GET /api/v1/auth/me` | Functional | Auto-redirect if logged in, professional B2B layout. |
| 15 | `/signup` | Page (CSR) | No | No | `POST /api/v1/auth/signup`, `GET /api/v1/auth/me` | Functional | Clear password requirements checklist (12 chars), professional layout. |
| 16 | `/forgot-password` | Page (CSR) | No | No | `POST /api/v1/auth/forgot-password` | Functional | Fix RTL arrow direction, add cooldown timer. |
| 17 | `/reset-password` | Page (CSR) | No | No | `POST /api/v1/auth/reset-password` | Functional | Polish password validation and success guidance. |
| 18 | `/verify-email` | Page (CSR) | Dual | No | `POST /api/v1/auth/verify-email`, `POST /resend-verification` | Functional | Add clear feedback and rate-limit feedback. |
| 19 | `/organizations` | Page (CSR) | Yes | No | `GET /api/v1/organizations`, `POST /organizations`, `POST /organizations/select` | Flawed UX | Safe redirect after switch, reactive tenant update. |
| 20 | `/settings/account` | Page (CSR) | Yes | No | `GET /api/v1/auth/me`, `POST /change-password`, `POST /resend-verification` | Functional | Password strength feedback, email verified status. |
| 21 | `/settings/members` | Page (CSR) | Yes | Yes | `GET /members`, `POST /members`, `PATCH /members/:id`, `DELETE /members/:id` | Flawed UX | Replace native `confirm()` with ConfirmModal, add role descriptions. |

---

## 2. API Contract & Payload Verification

All frontend requests map directly to the documented Go REST API:

- **CSRF Token Handling:** Automatically extracted from `clientpulse_csrf` cookie and transmitted in the `X-CSRF-Token` header for `POST`, `PUT`, `PATCH`, `DELETE` operations.
- **Tenant Context:** Selected organization ID is persisted on the backend session and optionally supplied via `X-Organization-ID` header.
- **Session Credentials:** Cookies are transmitted via `credentials: "include"` on every request.
- **Error Payloads:** Handled gracefully by parsing `{ error: { code: string, message: string } }` structures.

---

## 3. Localization & RTL Audit Summary

| Component / Page | English Status | Persian (فارسی) Status | RTL Layout Status |
|---|---|---|---|
| Navigation & Header | Complete | Complete | Mirroring requires sidebar & drawer support. |
| Dashboard | Partial (hardcoded stats) | Missing several keys | Table alignment needs RTL logical properties. |
| Clients Directory | Complete | Complete | Needs RTL table pagination & icon flip. |
| Client Details | Complete | Complete | Confirmation prompt was in browser native LTR. |
| Assets Directory | Missing Route | Missing Route | Must be built with full bidirectional layout. |
| Asset Details | Complete | Complete | SSL and environment badges need RTL consistency. |
| Monitors Directory | Complete | Complete | Pause/Resume icons and table cells need RTL alignment. |
| Monitor Details | Partial (hardcoded labels) | Missing several keys | History table needs RTL text alignment. |
| Auth Flows | Complete | Complete | Arrow icons need RTL bidirectional flipping. |
| Org & Member Settings | Complete | Complete | Form controls align well with text-start. |

---

## 4. Next Steps
With the audit complete and recorded in `task/TASK-003.2-frontend-audit.md` and `task-report/frontend-route-audit.md`, execution moves to:
1. Shared Visual Design System & Component Library
2. App Sidebar, Top Navigation & Mobile Drawer
3. Missing `/assets` route implementation
4. Dashboard redesign with 6 core real metrics & failure focus
5. Client, Asset & Monitor UX stabilization (humanized SSRF, threshold explanations, modals, toasts)
6. Playwright E2E functional verification
