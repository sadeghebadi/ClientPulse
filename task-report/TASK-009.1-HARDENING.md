# TASK-009.1 Implementation & Security Hardening Report

**Project:** ClientPulse  
**Priority:** Critical — Pre-Acceptance Hardening  
**Task Reference:** TASK-009.1 (Hardening and Verification for TASK-009)  
**Execution Environment:** Local Development Only (`postgres:16.15-alpine3.24`, `go 1.24`, Next.js 16.4.3 Turbopack)  
**Production Status:** Strictly Untouched (Zero remote connections, zero production migrations, zero production commands executed)

---

## 1. Actual Defects Found and Confirmation Status

A thorough code and database inspection was conducted against the potential defects noted in the task objective:

| Defect Candidate | Status | Root Cause & Evidence | Correction Implemented |
|---|---|---|---|
| **1. Draft plan version immediately published & immutable** | **Confirmed** | Migration 000018 defined status default `'draft'`, but the trigger `prevent_plan_version_mutation()` was documented in report but omitted in SQL. Without the trigger, immutability was purely application-level. When draft plans were edited, if they were published, the system created new drafts, but draft deletion was missing and draft vs. published transitions were not database-enforced. | Implemented database trigger `prevent_plan_version_mutation()` in migration `000019`. Drafts (`published_at IS NULL`) are editable in place and deletable when unreferenced. Transitions to `published` atomically set `published_at` and validate completeness. Once published (`published_at IS NOT NULL`), UPDATE and DELETE are rejected at the database level. Added `DELETE /api/v1/platform-admin/plans/{id}` for unreferenced drafts. |
| **2. Draft plans cannot clearly represent an unconfigured price** | **Confirmed** | Column `price_amount` was `NOT NULL DEFAULT 0`. A draft with missing price defaulted to 0, conflating unconfigured price with an explicit free plan. | Altered `price_amount` to nullable without default. In Go types, `PriceAmount *int64` explicitly differentiates `nil` (unconfigured), `0` (free plan), and `>0` (paid plan). Added frontend radio selectors to distinguish "Unconfigured (Draft)", "Free Tier ($0.00)", and "Paid Plan". |
| **3. USD, zero price, and monthly billing applied as database defaults** | **Confirmed** | `price_amount DEFAULT 0`, `currency DEFAULT 'USD'`, and `billing_interval DEFAULT 'monthly'` were hardcoded in `000018` DDL and repo params. | Migration `000019` removed all implicit column defaults. `ValidateParams()` and API handlers no longer force `USD` or `monthly` defaults onto drafts. Billing interval and currency are nullable in drafts and required only upon publishing. |
| **4. Multiple active subscriptions assignable to same organization** | **Partially Addressed / Hardened** | While a partial unique index `idx_org_subscriptions_active_unique` existed on `(organization_id) WHERE status = 'active'`, the repository plan assignment operation lacked row-level locking, exposing concurrent assignment requests to potential serialization conflicts or races. | `AssignPlan()` now performs `SELECT name FROM organizations WHERE id = $1 AND deleted_at IS NULL FOR UPDATE` to serialize subscription changes per organization. Concurrency tests verify that concurrent assignments cleanly serialize and only 1 active subscription is maintained. |
| **5. Platform Admin grant history may be lost through user deletion** | **Confirmed** | Migration `000018` defined `user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE`. If a user holding platform admin grant records was deleted, all grant and revocation audit history was cascade-deleted. | Migration `000019` altered the foreign key constraint to `ON DELETE RESTRICT`. A user with platform admin history cannot be cascade-deleted. Added unique index `uq_platform_admin_grants_active_user_role` preventing duplicate active grants. |
| **6. Concurrency and full regression verification not sufficiently documented** | **Confirmed** | Quota check-and-create routines in `entitlement.go` checked counts without tenant serialization locks, allowing two concurrent requests to simultaneously pass quota checks. Last-admin revoke check in CLI and API was subject to a simultaneous revocation race. | Introduced PostgreSQL transactional advisory locks (`pg_advisory_xact_lock`) for tenant quota evaluation and platform admin grant mutations. Automated concurrency test suites added in `task_009_1_hardening_test.go`. |

---

## 2. Database State Before Corrections

Before applying migration `000019`:
- Migration version in `schema_migrations`: 18 (`000018_platform_admin_and_subscriptions`).
- Subscription Plans: 25 records (12 active, 6 archived, 7 draft).
- Subscription Plan Versions: 31 records (18 published with timestamps, 13 draft without timestamps).
- Organization Subscriptions: 13 active records, 0 suspended/canceled.
- Platform Admin Grants: 0 duplicate active grants. Existing non-test platform admin was `admin-f6okyq@example.com` (from initial bootstrap test). `admin@mindu.ir` had exactly 0 grants and was an Organization Owner.
- Audit Logs: 93 records in `platform_admin_audit_logs`.
- All existing data and tables were preserved without deletion or truncation.

---

## 3. Forward Migration Details: `000019_subscription_hardening_and_platform_security`

File path: [000019_subscription_hardening_and_platform_security.up.sql](file:///home/sadpc/Documents/ClientPulse/migrations/000019_subscription_hardening_and_platform_security.up.sql)  
Rollback path: [000019_subscription_hardening_and_platform_security.down.sql](file:///home/sadpc/Documents/ClientPulse/migrations/000019_subscription_hardening_and_platform_security.down.sql)

Key DDL and procedural enhancements:
1. **Foreign Key Protection**:
   ```sql
   ALTER TABLE platform_admin_grants DROP CONSTRAINT IF EXISTS platform_admin_grants_user_id_fkey;
   ALTER TABLE platform_admin_grants ADD CONSTRAINT platform_admin_grants_user_id_fkey 
       FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT;
   ```
2. **Duplicate Active Grant Prevention**:
   ```sql
   CREATE UNIQUE INDEX IF NOT EXISTS uq_platform_admin_grants_active_user_role 
       ON platform_admin_grants (user_id, role) 
       WHERE revoked_at IS NULL;
   ```
3. **Draft Pricing Flexibility & Removal of Defaults**:
   ```sql
   ALTER TABLE subscription_plan_versions ALTER COLUMN price_amount DROP NOT NULL, ALTER COLUMN price_amount DROP DEFAULT;
   ALTER TABLE subscription_plan_versions ALTER COLUMN currency DROP NOT NULL, ALTER COLUMN currency DROP DEFAULT;
   ALTER TABLE subscription_plan_versions ALTER COLUMN billing_interval DROP NOT NULL, ALTER COLUMN billing_interval DROP DEFAULT;
   ```
4. **Publish-Time Completeness Constraint**:
   ```sql
   ALTER TABLE subscription_plan_versions ADD CONSTRAINT chk_plan_version_published_config
       CHECK (
           status != 'published' OR (
               published_at IS NOT NULL AND
               price_amount IS NOT NULL AND
               currency IS NOT NULL AND
               billing_interval IS NOT NULL
           )
       );
   ```
5. **PostgreSQL Trigger `prevent_plan_version_mutation()`**:
   - Rejects `DELETE` if version is published (`status = 'published'` or `published_at IS NOT NULL`) or referenced in `organization_subscriptions`.
   - Rejects `UPDATE` on already published versions.
   - Allows draft editing in place.
   - Allows transition from draft to published, automatically validating completeness and assigning `published_at = NOW()`.
6. **Audit Log Immutability Trigger `prevent_platform_audit_log_mutation()`**:
   - Blocks `UPDATE`, `DELETE`, and `TRUNCATE` operations on `platform_admin_audit_logs`.
   - Re-affirms role privileges restricting `clientpulse_app` to `SELECT` and `INSERT` only.

Migration verified via bidirectional execution (`up` -> `down 1` -> `up`) using the Go migration CLI engine.

---

## 4. Draft and Published Lifecycle Behavior

1. **Draft Creation**:
   - Platform admin can create a draft plan without price, currency, or billing interval.
   - Draft version is inserted with `status = 'draft'` and `published_at = NULL`.
2. **Draft Editing**:
   - Updating a draft modifies the draft version row in place.
   - Unconfigured fields can be updated or left unconfigured.
3. **Draft Deletion**:
   - If a plan is draft, has never had a published version, and is unreferenced by subscriptions, it can be deleted via `DELETE /api/v1/platform-admin/plans/{id}`.
4. **Publishing**:
   - Administrative action validates `price_amount != nil`, valid 3-letter currency, and approved billing interval (`monthly`, `yearly`, `quarterly`, `one_time`).
   - Atomically updates status to `published` and sets `published_at = NOW()`.
   - The database trigger strictly protects this row against any subsequent updates or deletion.
5. **Iterating on Published Plans**:
   - Modifying a published plan automatically creates a new draft version `(v + 1)` while leaving the original published version untouched and available to existing subscriptions.

---

## 5. Price and Currency Representation

1. **Integer Minor Units**:
   - Price amounts are stored as integer minor units (`int64` / PostgreSQL `BIGINT`) representing cents (or minor currency units).
   - Validated server-side to ensure `price_amount >= 0`.
2. **Nullable State Distinction**:
   - `price_amount == nil`: Unconfigured price (allowed only in draft versions; cannot be published).
   - `price_amount == 0`: Explicitly configured Free plan.
   - `price_amount > 0`: Explicitly configured Paid plan.
   - Missing price is never inferred as a free plan.
3. **Currency Validation**:
   - Server validates currency against ISO 4217 uppercase 3-letter code (`^[A-Z]{3}$`).
   - Currency is optional in draft, required before publishing.
4. **Billing Interval**:
   - Optional in draft; validated against `monthly`, `yearly`, `quarterly`, or `one_time` before publication.

---

## 6. Subscription Uniqueness and Inactive Entitlement Lifecycle

1. **Active Subscription Invariant**:
   - Enforced by partial unique index `idx_org_subscriptions_active_unique` on `(organization_id) WHERE status = 'active'`.
   - `AssignPlan()` locks the organization row with `FOR UPDATE` to serialize replacements.
   - Atomically cancels the current active subscription (`status = 'canceled', ends_at = NOW()`) before inserting the new active subscription in the same transaction.
2. **Inactive Subscription Entitlement Behavior**:
   - **Legacy Unassigned Organization**: If an organization has never had any subscription assigned, it continues to enjoy the documented legacy unmetered behavior.
   - **Inactive Subscription (Suspended / Canceled / Expired)**: `entitlement.Service` evaluates `getEffectiveSubscription()`. If no active subscription exists, it inspects whether a historical subscription ever existed via `GetLatestSubscriptionForOrg()`. If the most recent subscription is inactive, creation of new resources (clients, assets, monitors, heartbeats, members) is blocked with `ErrSubscriptionInactive ("organization subscription is inactive (suspended, canceled, or expired)")`.
   - **Preservation of Existing Data**: Inactive subscriptions do not delete or destroy existing tenant assets, monitors, or client records.

---

## 7. Platform Admin Grant History and Last-Admin Concurrency

1. **History Preservation**:
   - `ON DELETE RESTRICT` foreign key prevents deleting a user who holds platform admin history.
   - Revocation sets `revoked_at = NOW()` and logs an audit entry, preserving the grant record.
2. **Duplicate Active Grants**:
   - Enforced at database level via `uq_platform_admin_grants_active_user_role`.
   - Repository `GrantAdmin()` method performs an idempotent check under advisory lock.
3. **Last-Admin Concurrency Protection**:
   - Both API and CLI `executeRevoke()` and repository `RevokeAdmin()` routines acquire a transaction-scoped PostgreSQL advisory lock (`pg_advisory_xact_lock(982347103)`).
   - Tested under concurrent goroutines: When two administrators simultaneously attempt to revoke their own grants, exactly one succeeds and the other receives `"CANNOT REVOKE: last-administrator protection active (only 1 platform administrator remains)"`. The system can never end up with zero active admins.
4. **Bootstrap CLI Audit Logging**:
   - The CLI executes grant and revocation actions inside a single atomic transaction that simultaneously inserts into `platform_admin_audit_logs`.
   - The CLI was verified in tests and was **not executed** for real users.

---

## 8. Audit Log Protection Verification

1. **Table Role Privileges**:
   - `clientpulse_app` is granted only `SELECT` and `INSERT` on `platform_admin_audit_logs`.
   - `UPDATE`, `DELETE`, and `TRUNCATE` are explicitly revoked.
   - Direct execution tests confirmed that `clientpulse_app` receives `ERROR: permission denied for table platform_admin_audit_logs` when attempting delete or truncate.
2. **Defense-in-Depth Trigger**:
   - Trigger `prevent_platform_audit_log_mutation()` raises an exception on any attempted `UPDATE` or `DELETE`.
3. **Availability**:
   - User or organization suspension does not impact the visibility of platform admin audit logs.

---

## 9. Entitlement Concurrency Verification

1. **Check-and-Create Quota Race Prevention**:
   - `entitlement.Service` methods (`CheckClientCreation`, `CheckAssetCreation`, `CheckMonitorCreation`, `CheckHeartbeatCreation`, `CheckMemberAddition`) acquire a transaction-level advisory lock hashed by organization UUID (`pg_advisory_xact_lock(hashtext(orgID))`).
   - This serializes concurrent creation attempts for the same tenant.
2. **Concurrency Test Results**:
   - Automated test `TestTask009_1_EntitlementConcurrency` configured a plan with a limit of 2 clients. 1 client was created initially.
   - 5 concurrent requests simultaneously attempted to create a client.
   - **Result**: Exactly 1 request succeeded (`201 Created`), while the other 4 were rejected with `LIMIT_EXCEEDED` (`403 Forbidden`). The final client count in PostgreSQL remained exactly 2.
3. **Over-Limit Handling**:
   - If an organization's usage exceeds newly assigned plan limits, existing resources are retained intact; over-limit warnings are returned in API responses and logged in the audit record.

---

## 10. Suspension Behavior Verification

1. **User Suspension**:
   - `RequireAuth` and `RequirePlatformAdmin` middlewares inspect `u.Status == "suspended"` on every incoming request.
   - Suspended users receive `HTTP 403 Forbidden` (`USER_SUSPENDED`) immediately, invalidating active sessions across all endpoints.
   - Reactivation by platform admin restores access according to session policy.
2. **Organization Suspension**:
   - `RequireTenant` middleware inspects `t.Status == "suspended"` on every operational route.
   - Members receive `HTTP 403 Forbidden` (`ORGANIZATION_SUSPENDED`).
   - Platform admins operate via `/api/v1/platform-admin/` routes which bypass tenant-scoped middleware, allowing platform admins to inspect, monitor, and reactivate suspended organizations at any time.

---

## 11. Full Regression Test Verification

### Backend Go Suite (`services/api`)
```
go test ./...
PASS: ok  clientpulse/services/api/internal/appurl
PASS: ok  clientpulse/services/api/internal/auth
PASS: ok  clientpulse/services/api/internal/checker
PASS: ok  clientpulse/services/api/internal/config
PASS: ok  clientpulse/services/api/internal/database
PASS: ok  clientpulse/services/api/internal/health
PASS: ok  clientpulse/services/api/internal/logger
PASS: ok  clientpulse/services/api/internal/migrate
PASS: ok  clientpulse/services/api/internal/notification
PASS: ok  clientpulse/services/api/internal/notification/email
PASS: ok  clientpulse/services/api/internal/notification/telegram
PASS: ok  clientpulse/services/api/internal/pdf
PASS: ok  clientpulse/services/api/internal/rate
PASS: ok  clientpulse/services/api/internal/repository
PASS: ok  clientpulse/services/api/internal/server (all platform admin, hardening, and concurrency tests passed)
PASS: ok  clientpulse/services/api/internal/service
PASS: ok  clientpulse/services/api/internal/ssrf
PASS: ok  clientpulse/services/api/internal/storage
PASS: ok  clientpulse/services/api/internal/worker
All packages passed with ZERO failures.
```

### Frontend Verification (`apps/web`)
1. **TypeScript Compilation**:
   ```
   npx tsc --noEmit -> Exit 0 (Clean, 0 errors)
   ```
2. **ESLint Verification**:
   ```
   npm run lint -> Exit 0 (0 errors, 7 warnings)
   ```
3. **Next.js Production Build**:
   ```
   npm run build -> Exit 0 (All 35 static/dynamic routes compiled successfully in Turbopack)
   ```

### Playwright End-to-End & Regression Suites
- `task-009-admin-validation.js`: 100% PASSED (All 19 screenshots captured, owner restrictions confirmed, admin privileges cleaned).
- `task-010-landing-validation.js`: 100% PASSED (All 7 marketing and CTA flow scenarios verified).
- `task-008-b3-1-validation.js`: 100% PASSED (All 16 dashboard incident-aware regression scenarios verified).
- `task-008-b4-validation.js`: 100% PASSED (All operational pages, 390px/360px viewports, RTL, and Viewer RBAC verified).

---

## 12. Known Limitations

1. **Single Currency per Plan Version**: Each subscription plan version specifies one base currency (e.g. USD, EUR). Multi-currency pricing matrices within the same plan version are not supported; separate plans or versions can be created for different currencies.
2. **Manual Reconciliation for Downgrades**: When an organization is assigned a plan with lower quotas than its current usage, existing resources remain active and over-limit warnings are issued; administrative remediation or manual deletion is required by the customer to regain ability to create new resources.

---

## 13. Production Safety Affirmation

- **Production SSH Access**: Never attempted or accessed.
- **Production Database**: Never connected to or modified.
- **Production Deployments**: Zero deployments or CI/CD pipelines triggered.
- **Real Accounts**: `admin@mindu.ir` was never granted platform admin privileges; database inspection confirmed 0 active or revoked grants for `admin@mindu.ir`.
- **Development Database**: All operations were strictly isolated within local Docker container `clientpulse_postgres` on port 5433.

