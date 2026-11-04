# TASK-009.3 Implementation Report: Platform Admin Redesign & Locale-Aware Subscription Catalog

**Project:** ClientPulse  
**Environment:** Local Development Only  
**Date:** 2026-11-03  
**Status:** In Progress (Phase 0 Audit Complete)  

---

## 1. Initial Architecture Audit (Phase 0)

### 1.1 Existing Platform Admin Route Structure
The existing Platform Admin routes in `apps/web/src/app/platform-admin/` include:
- `/platform-admin`: Platform Dashboard / Overview
- `/platform-admin/users`: Users list
- `/platform-admin/users/[id]`: User details and suspension controls
- `/platform-admin/organizations`: Organizations list
- `/platform-admin/organizations/[id]`: Organization detail, resource usage, subscription assignment
- `/platform-admin/plans`: Subscription plans list
- `/platform-admin/plans/new`: Create plan wizard
- `/platform-admin/plans/[id]`: Plan detail, version history, publishing, and archiving
- `/platform-admin/subscriptions`: Subscription lifecycle view
- `/platform-admin/audit-logs`: Platform admin audit log records

### 1.2 Layout & Application Shell Audit
- `AppLayout.tsx` provides the primary ClientPulse application shell with responsive sidebar, header, active workspace switcher, user profile menu, and RTL/LTR direction switching.
- Platform Admin currently sits inside `apps/web/src/app/platform-admin/layout.tsx`. It uses an ad-hoc dark strip at the top with simple horizontal tabs, which feels disconnected from the polished look of the main application.
- The sidebar in `AppLayout.tsx` shows `navPlatformAdmin` section only when `isPlatformAdmin` is true.

### 1.3 Localization & i18n Audit
- The web app uses `apps/web/src/lib/i18n.tsx`.
- Locales currently defined: `"en"` (English) and `"fa"` (Persian).
- Locale storage: `localStorage.getItem("clientpulse_locale")`.
- Established Persian identifier in ClientPulse is `"fa"`. To preserve compatibility with existing saved preferences, `"fa"` is retained as the locale identifier.
- Currently, language toggling is purely client-side without platform admin governance or server-side feature flags.

### 1.4 Subscription Plans & Pricing Model Audit
- Migration `000018` introduced `subscription_plans`, `subscription_plan_versions`, `organization_subscriptions`, and `platform_admin_audit_logs`.
- Migration `000019` hardened plan versions to make published versions immutable, allow drafts to have null/unconfigured prices, and restrict foreign key cascades.
- Currently, each `subscription_plan_version` conflates technical limits (`max_clients`, `max_monitors`, etc.) with pricing (`price_amount`, `currency`, `billing_interval`). There is no localization of plan descriptions, titles, or market-specific currencies.
- In `internal/repository/currency.go`, `IRR` is registered as ISO 4217 with minor unit decimals = 2. Toman is not an ISO currency code.

### 1.5 Database Migrations
- Migrations `000001` through `000019` are already applied to the local database.
- Migration `000020_locale_settings_and_localized_catalogs` will be created to introduce:
  1. `platform_localization_settings`: Singleton configuration table for enabled locales and default locale.
  2. `subscription_offers`: Localized offers bound to a base plan, locale, and market.
  3. `subscription_offer_versions`: Immutable published pricing and presentation version with canonical stored price, currency, display denomination (`toman`, `usd`), and exact integer denomination conversion ratio.
  4. An optional `offer_version_id` column on `organization_subscriptions` referencing `subscription_offer_versions(id)`.
  5. Immutability triggers on published offer versions.
  6. Permissions granted to `clientpulse_app` and `clientpulse_definer`.

---

