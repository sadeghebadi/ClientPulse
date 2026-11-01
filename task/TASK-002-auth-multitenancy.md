# TASK-002: Authentication & Multi-Tenancy

**Phase:** Phase 1  
**Status:** Completed  
**Started:** 2026-10-31  
**Completed:** 2026-11-01  
**Single Source of Truth:** `PRD.md` (Sections 3.5, 6.1, 6.2, 8.1, 9.7, 12.1, 12.2, 12.4, 16.3, 16.4)

---

## 1. Objective

Implement the complete, production-grade **Authentication & Multi-Tenancy** foundation for ClientPulse:
- User registration, login, logout, session management, and current authenticated user endpoint.
- Argon2id password hashing and secure token generation (CSPRNG, HMAC-SHA256 with pepper).
- Session security using `HttpOnly`, `SameSite=Lax`, and `Secure` cookies with server-side revocation.
- CSRF protection for cookie-authenticated mutations.
- Password change, forgot password, and reset password flows with anti-enumeration protection.
- Email verification flow with Mailpit integration.
- Rate limiting on authentication mutations.
- Multi-tenant organization foundation: independent memberships, active organization resolution, and role-based access control (Owner, Admin, Member, Viewer).
- Multi-tenant data isolation defense in depth:
  - Repository-level explicit tenant scoping (`WHERE organization_id = $context_org_id`).
  - PostgreSQL Row-Level Security (RLS) policies.
  - Dedicated non-superuser role `clientpulse_app` with `NOBYPASSRLS`.
  - Transaction-scoped tenant context (`SET LOCAL ROLE clientpulse_app; SELECT set_config('app.current_organization_id', $orgID, true)`) to prevent connection pool leakage.
  - Mandatory server-side membership validation on every protected request.
- Frontend authentication, organization switcher, member management, and account settings screens in Next.js 16 (English & Persian RTL).
- Comprehensive automated integration tests against real PostgreSQL.

---

## 2. Work Breakdown

- [x] Step 1: Pre-implementation audit and architectural documentation (`ADR 0002`).
- [x] Step 2: Database migration `000003_auth_and_multitenancy.up.sql` & `.down.sql`.
- [x] Step 3: Backend core packages:
  - Password hashing (Argon2id) & token utilities (`crypto/rand`, SHA-256 peppered hash).
  - Rate limiting & CSRF protection.
  - Email sender abstraction (SMTP / Mailpit).
  - Database repository layer with transaction-scoped tenant context and RLS integration.
- [x] Step 4: Backend service layer:
  - Authentication service (signup, login, logout, me, verify-email, forgot/reset password, change password).
  - Organization & membership service (create, list, switch, update, member roles, remove).
- [x] Step 5: Backend HTTP server & middleware:
  - Authentication middleware (`session_token` cookie / header validation).
  - Tenant context middleware (membership validation, active organization resolution).
  - RBAC role check middleware (Owner, Admin, Member, Viewer).
  - CSRF middleware & Rate limit middleware.
  - REST API route handlers.
- [x] Step 6: Security and integration tests:
  - Authentication & session tests (registration, duplicate email rejection, login, logout, revocation, password reset single-use).
  - Role authorization tests (Owner vs Admin vs Member vs Viewer).
  - Multi-tenant isolation tests (Agency A vs Agency B, cross-tenant access rejection, connection pool context isolation, RLS).
- [x] Step 7: Frontend implementation (Next.js 16):
  - Auth context & API client (`fetch` with credentials and CSRF tokens).
  - Authentication pages: `/login`, `/signup`, `/forgot-password`, `/reset-password`, `/verify-email`.
  - Organization onboarding & management: `/onboarding`, `/organizations`, `/settings/members`, `/settings/account`.
  - Bilingual support (English & Persian RTL) and UI integration.
- [x] Step 8: Verification & final acceptance:
  - `go test -count=1 -race ./...` (PASS)
  - `go vet ./...` (PASS)
  - `npm run lint` & `npm run build` (PASS)
  - `docker compose up -d --build` (PASS)
  - Live API validation with Mailpit integration (PASS)
  - Completion report `task-report/TASK-002-report.md`.
