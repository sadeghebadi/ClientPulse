# TASK-009.3.1: Persian & English Language Activation Management — Final Report

**Project:** ClientPulse  
**Priority:** High — Localization Management Completion  
**Environment:** Local Development Only (`clientpulse-postgres:5433`, `clientpulse-api:8080`, `clientpulse-web:3001`)  
**Production Status:** Strictly Untouched (Zero remote access, zero production connections, zero production migrations)  
**Related Tasks:** TASK-009, TASK-009.3  
**Scope Limitation:** Strictly restricted to exactly TWO supported platform languages: **English (`en`)** and **Persian (`fa`)**. All dynamic language registration, translation pack uploads, and arbitrary locale creation were explicitly cancelled and prevented.

---

## Executive Summary

TASK-009.3.1 successfully diagnosed and resolved the root-cause defects preventing operational language activation and deactivation in ClientPulse. The platform administration interface at `/platform-admin/settings/localization` now provides immediate, reactive activation switches for English (`en`) and Persian (`fa`), atomic platform fallback default switching, real-time cache invalidation, site-wide selector responsiveness, and seamless client fallback without session disruption or administrative lockout.

All verification steps, unit test suites, regression test suites, and Playwright end-to-end browser tests passed with 100% success across desktop and mobile form factors.

---

## 1. Exact Existing Defect & Root Cause Analysis

### 1.1 The Frontend Payload Shape Mismatch Defect
In the preceding implementation (`TASK-009.3`), `/platform-admin/settings/localization/page.tsx` attempted to unpack responses from the backend localization endpoints as:
```typescript
const res = await apiFetch<{ settings: LocalizationSettings }>(
  "/api/v1/platform-admin/settings/localization"
);
if (res && res.settings) {
  setSettings(res.settings);
  setEnabledLocales(res.settings.enabled_locales || ["en"]);
  setDefaultLocale(res.settings.default_locale || "en");
}
```
However, the Go API handler `GetLocalizationSettings` in `services/api/internal/server/platform_admin_handlers.go`:
```go
func (h *PlatformAdminHandler) GetLocalizationSettings(w http.ResponseWriter, r *http.Request) {
    cfg, err := h.localizationRepo.GetLocalizationConfig(r.Context())
    ...
    sendJSONResponse(w, http.StatusOK, cfg)
}
```
encoded the `LocalizationConfig` struct directly at the root JSON object:
```json
{
  "enabled_locales": ["en", "fa"],
  "default_locale": "en",
  "supported_locales": [...],
  "updated_at": "2026-11-04T06:04:52.100756Z"
}
```
Because the JSON payload had no top-level `.settings` wrapper key, `res.settings` was consistently `undefined`. Consequently:
1. `setSettings` was never executed on initial load; `settings` state remained `null`.
2. Form change detection `hasUnsavedChanges` evaluated `settings && ...`, which evaluated to `null` (falsy).
3. The "Save Configuration" button had `disabled={saving || !hasUnsavedChanges}`, permanently locking the submit button in a disabled state regardless of user checkbox toggling.
4. When `handleSave` was invoked programmatically, the response unpacking `res.settings` similarly failed, preventing state synchronization and suppressing success feedback.

### 1.2 Redundant Language Selector Display Defect
In `AppLayout.tsx`, when only one language was active, the markup previously rendered a disabled indicator badge:
```tsx
<div className="inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium text-slate-500 bg-slate-100 rounded-lg border border-slate-200">
  <Languages className="w-3.5 h-3.5 text-slate-400" />
  <span>{locale === "en" ? "EN" : "فا"}</span>
</div>
```
This violated Scenario D, which requires hiding the language switching control entirely when only one language is enabled rather than displaying a redundant badge.

### 1.3 Client Fallback Scope Defect
In `apps/web/src/lib/i18n.tsx`, graceful fallback only triggered if `localStorage.getItem("clientpulse_locale")` was non-null:
```typescript
const saved = localStorage.getItem("clientpulse_locale") as Locale;
if (saved && !data.enabled_locales.includes(saved)) {
  changeLocale(def as Locale);
}
```
If a new visitor arrived with an empty `localStorage` snapshot when a language was disabled globally, they remained locked to the disabled language instead of falling back to the configured platform default.

---

## 2. Current Localization API Architecture

ClientPulse maintains PostgreSQL as the singular source of truth for platform language settings, using the singleton table `platform_localization_settings` (ID = 1) established in migration `000020_locale_settings_and_localized_catalogs.up.sql`.

### 2.1 Database Schema Invariants
- `enabled_locales TEXT[] NOT NULL DEFAULT ARRAY['en', 'fa']::TEXT[]`
- `default_locale VARCHAR(16) NOT NULL DEFAULT 'en'`
- Check constraints enforce:
  - `chk_localization_singleton CHECK (id = 1)`
  - `chk_at_least_one_enabled CHECK (cardinality(enabled_locales) > 0)`
  - `chk_default_in_enabled CHECK (default_locale = ANY(enabled_locales))`

### 2.2 Endpoints Summary
1. `GET /api/v1/public/localization`:
   - Public, unauthenticated endpoint.
   - Utilizes in-memory cached read lock (`RWMutex`), serving cached `PublicLocalizationConfig`.
   - Returns:
     ```json
     {
       "enabled_locales": ["en", "fa"],
       "default_locale": "en",
       "supported_locales": [
         {"code": "en", "label": "English", "native_label": "English", "dir": "ltr", "enabled": true, "is_default": true},
         {"code": "fa", "label": "Persian", "native_label": "فارسی", "dir": "rtl", "enabled": true, "is_default": false}
       ]
     }
     ```
2. `GET /api/v1/platform-admin/settings/localization`:
   - Scoped to active Platform Administrators.
   - Bypasses public read cache, querying fresh row data from PostgreSQL.
3. `PUT /api/v1/platform-admin/settings/localization`:
   - Requires Platform Admin authentication, CSRF token validation, and rate limiting (`platformAdminMutation`).
   - Executes inside a transaction with `SELECT id FROM platform_localization_settings WHERE id = 1 FOR UPDATE` row lock.
   - Atomically updates `enabled_locales`, `default_locale`, `updated_at`, and `updated_by`.
   - Flushes cache atomically (`r.cachedConf = nil`).
   - Records audit log in `platform_admin_audit_logs` under `localization.updated`.
4. `GET /api/v1/public/catalog?locale={locale}`:
   - Evaluates whether `locale` is enabled. If disabled, automatically and gracefully falls back to `default_locale`.

---

## 3. Enable / Disable Workflow & Default Language Transitions

### 3.1 Interaction Architecture
On the `/platform-admin/settings/localization` route, exactly two cards are presented:
- **English (`en`)**: Native name "English", LTR direction, USD currency format, Western date formatting.
- **Persian (`fa`)**: Native name "فارسی", RTL direction, Toman currency format, Solar Hijri date formatting.

Both cards remain persistently rendered on screen, regardless of whether a language is enabled or disabled. This guarantees that administrators can re-enable a deactivated language at any time.

### 3.2 Reactive Toggle Interaction & Loading States
Each card features:
1. **Interactive Toggle Switch:**
   - Accessible button with `role="switch"` and `aria-checked`.
   - Clicking immediately submits the change to `PUT /api/v1/platform-admin/settings/localization`.
   - While the request is in flight, an inline `Loader2` spinner animates within the toggle knob, and controls are disabled to prevent duplicate concurrent submissions.
2. **Success & Error Feedback:**
   - On success: An emerald notification banner confirms the exact change (e.g., *"Disabled English and switched default to Persian successfully"*).
   - On error: A rose alert banner displays the server validation message, and the UI automatically rolls back to the persistent database state.
3. **Atomic Default Transition:**
   - The platform fallback default cannot be disabled.
   - If an administrator toggles off the language currently designated as default, the frontend automatically designates the remaining active language as the new default fallback and executes the update atomically in a single request.
   - A dedicated "Set as Default" button allows explicit switching of the default locale between English and Persian.
4. **Final Language Invariant Protection:**
   - If only one language remains enabled, its switch is disabled and an explicit badge *"Final Active Language"* is displayed.
   - If an administrator attempts to send an empty array or disable the last language, the backend rejects the request with HTTP `400 Bad Request` (`ErrAtLeastOneLocaleRequired`).

---

## 4. Site-Wide Language Selector Architecture

A single centralized component was created at `apps/web/src/components/LanguageSelector.tsx`:

```tsx
"use client";

import React from "react";
import { Languages } from "lucide-react";
import { useI18n } from "@/lib/i18n";

export function LanguageSelector({ className = "" }: { className?: string }) {
  const { locale, toggleLocale, enabledLocales, canToggleLanguage } = useI18n();

  // Scenario D: When only one language is enabled, hide the selector entirely.
  if (!canToggleLanguage || enabledLocales.length <= 1) {
    return null;
  }

  const label = locale === "en" ? "فارسی" : "English";
  const shortLabel = locale === "en" ? "FA" : "EN";

  return (
    <button
      type="button"
      onClick={toggleLocale}
      className={`inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold text-slate-700 bg-white hover:bg-slate-50 border border-slate-200 rounded-lg shadow-2xs transition-colors shrink-0 ${className}`}
      title={locale === "en" ? "Switch to Persian (فارسی)" : "تغییر زبان به انگلیسی (English)"}
      data-testid="language-selector-button"
    >
      <Languages className="w-3.5 h-3.5 text-slate-500 shrink-0" />
      <span className="hidden sm:inline font-medium">{label}</span>
      <span className="sm:hidden text-[11px] font-bold">{shortLabel}</span>
    </button>
  );
}
```

### 4.1 Audited & Updated Navigation Points
The centralized `<LanguageSelector />` was integrated across the application:
1. **Public Landing Page Navigation (`MarketingHeader.tsx`):**
   - Displays toggle button when both languages are active.
   - Completely hidden when only one language is active.
2. **Application Shell (`AppLayout.tsx`):**
   - Unauthenticated header: uses `<LanguageSelector />`; removed redundant single-option badge.
   - Authenticated header: uses `<LanguageSelector />`; removed redundant single-option badge.
3. **Authentication Pages (`/login`, `/signup`, `/forgot-password`, `/reset-password`):**
   - Wrap `<AppLayout>`, automatically inheriting the centralized responsive selector.
4. **Platform Admin Panel (`PlatformAdminLayout`):**
   - Wraps `<AppLayout>`, automatically inheriting the header language switcher while maintaining persistent access to both language cards in `/platform-admin/settings/localization`.
5. **Organizational Locale Selectors:**
   - `/onboarding/page.tsx`: Select dropdown dynamically filters out disabled locales via `isLocaleEnabled(code)`.
   - `/settings/branding/page.tsx`: Report locale dropdown filters out disabled locales via `isLocaleEnabled(code)`.
   - `/reports/page.tsx`: Report generation modal filters out disabled locales via `isLocaleEnabled(code)`.

---

## 5. Automatic Fallback & Platform Admin Access Preservation

### 5.1 Fallback Synchronization Lifecycle
1. On initial page load or when `clientpulse:localization_settings_updated` fires:
   `I18nProvider` fetches `GET /api/v1/public/localization`.
2. If `localStorage.getItem("clientpulse_locale")` is set to a locale not present in `data.enabled_locales`:
   - Calls `changeLocale(data.default_locale)`.
   - Updates `localStorage.setItem("clientpulse_locale", data.default_locale)`.
   - Updates `document.documentElement.lang = data.default_locale`.
   - Updates `document.documentElement.dir = data.default_locale === "fa" ? "rtl" : "ltr"`.
   - Fires `clientpulse:locale_changed` to notify all `useSyncExternalStore` hooks.
3. If no locale was saved in `localStorage`, but the active locale is disabled:
   - Automatically applies `data.default_locale` and persists it.
4. User sessions, cookies, CSRF tokens, and account records remain completely intact.

### 5.2 Platform Admin Access Preservation Under Locale Deactivation
Special care was engineered for the critical administrative scenario:
- **Scenario:** Administrator is viewing `/platform-admin/settings/localization` in Persian (`locale = 'fa'`).
- Administrator disables Persian.
- The backend updates `enabled_locales = ['en']` and `default_locale = 'en'`.
- The frontend receives the response, detects `!updated.enabled_locales.includes(language)`, and immediately calls `setLanguage('en')`.
- The administrator's interface seamlessly transforms from RTL to LTR typography without refreshing the browser, without redirecting away from `/platform-admin/settings/localization`, and without terminating the administrative session.
- The Persian card remains on screen, showing status "Disabled", with its toggle switch enabled and ready to be re-activated.

---

## 6. Verification & Test Results

### 6.1 Backend Go Test Suite
Test suite executed via:
```bash
docker pause clientpulse_worker && cd services/api && go test -v -run "TestLanguageManagement" ./internal/server ; docker unpause clientpulse_worker
```
Results:
```
=== RUN   TestLanguageManagement_Availability
--- PASS: TestLanguageManagement_Availability (0.15s)
=== RUN   TestLanguageManagement_DefaultTransitions
--- PASS: TestLanguageManagement_DefaultTransitions (0.16s)
=== RUN   TestLanguageManagement_ConcurrentUpdates
--- PASS: TestLanguageManagement_ConcurrentUpdates (0.16s)
=== RUN   TestLanguageManagement_PublicCatalogGracefulFallback
--- PASS: TestLanguageManagement_PublicCatalogGracefulFallback (0.11s)
PASS
ok  	clientpulse/services/api/internal/server	0.591s
```
Full backend suite `go test ./...`:
```
ok  	clientpulse/services/api/internal/appurl	(cached)
ok  	clientpulse/services/api/internal/auth	(cached)
ok  	clientpulse/services/api/internal/checker	(cached)
ok  	clientpulse/services/api/internal/config	(cached)
ok  	clientpulse/services/api/internal/database	(cached)
ok  	clientpulse/services/api/internal/health	(cached)
ok  	clientpulse/services/api/internal/logger	(cached)
ok  	clientpulse/services/api/internal/migrate	(cached)
ok  	clientpulse/services/api/internal/notification	(cached)
ok  	clientpulse/services/api/internal/notification/email	(cached)
ok  	clientpulse/services/api/internal/notification/telegram	(cached)
ok  	clientpulse/services/api/internal/pdf	(cached)
ok  	clientpulse/services/api/internal/rate	(cached)
ok  	clientpulse/services/api/internal/repository	(cached)
ok  	clientpulse/services/api/internal/server	8.891s
ok  	clientpulse/services/api/internal/service	(cached)
ok  	clientpulse/services/api/internal/ssrf	(cached)
ok  	clientpulse/services/api/internal/storage	(cached)
ok  	clientpulse/services/api/internal/worker	(cached)
```

### 6.2 Frontend Type Checking, Linting & Build
Executed in `apps/web`:
```bash
cd apps/web && npx tsc --noEmit && npm run lint && npm run build
```
Results:
- `npx tsc --noEmit`: 0 errors.
- `npm run lint`: 0 errors, 0 warnings.
- `next build`: Compiled successfully in 247ms; 36 static/dynamic routes generated without warnings or hydration errors.

### 6.3 Playwright End-to-End Suite
E2E automated suite executed via:
```bash
node apps/web/tests/task-009-3-1-e2e.js
```
Results:
```
===================================================================
TASK-009.3.1: Persian & English Language Activation Management E2E
Web URL: http://localhost:3001
Screenshots: /home/sadpc/Documents/ClientPulse/task-report/screenshots/task-009-3-1
===================================================================

▶ Step 1: Testing Landing Page with Both Languages Enabled...
  Language selector button visible on landing page: true
✔ Saved 01_both_languages_landing_desktop.png
✔ Saved 02_both_languages_landing_mobile.png
  Switching to Persian via language selector...
  HTML dir after toggle to Persian: rtl
  HTML dir after toggle back to English: ltr

▶ Step 2: Logging into Platform Admin Settings Localization...
✔ Saved 03_both_languages_enabled_admin.png

▶ Step 3: Transitioning Default Language to Persian...
  Success banner text: Set Persian as default platform language successfully.
✔ Saved 04_default_language_transition_to_persian.png

▶ Step 4: Disabling English (Persian Only)...
  Persian switch disabled (final language protection): true
  Language selector hidden in header: true
✔ Saved 05_persian_only_admin.png
  Landing page direction under Persian-only: rtl
✔ Saved 06_persian_only_landing_selector_hidden.png
✔ Saved 07_persian_only_landing_mobile.png

▶ Step 5: Testing Automatic Fallback for user with saved English locale...
  Stored locale after fallback: fa
  HTML lang: fa, dir: rtl
✔ Saved 08_automatic_fallback_from_disabled_english.png

▶ Step 6: Platform Admin Re-Enabling English...
✔ Saved 09_admin_re_enabling_english.png

▶ Step 7: Setting English Default & Disabling Persian (English Only)...
✔ Saved 10_english_only_admin.png
✔ Saved 11_english_only_landing_selector_hidden.png
✔ Saved 12_english_only_landing_mobile.png

▶ Step 8: Testing Automatic Fallback for user with saved Persian locale...
  Stored locale after fallback: en
  HTML lang: en, dir: ltr
✔ Saved 13_automatic_fallback_from_disabled_persian.png

▶ Step 9: Platform Admin Re-Enabling Persian...
✔ Saved 14_admin_re_enabling_persian.png

▶ Step 10: Testing Admin viewing in Persian disabling Persian...
  Admin panel direction in Persian: rtl
  Admin URL after disabling Persian: http://localhost:3001/platform-admin/settings/localization
  Admin direction after disabling Persian: ltr
  Successfully re-enabled Persian from English view!
✔ Saved 15_admin_graceful_language_transition.png

===================================================================
🎉 ALL E2E TEST SCENARIOS PASSED SUCCESSFULLY!
===================================================================
```

---

## 7. Screenshots Inventory

All 15 screenshots have been captured, verified, and placed in `task-report/screenshots/task-009-3-1/`:

| File | Scenario / State Description | Dimensions |
| :--- | :--- | :--- |
| `01_both_languages_landing_desktop.png` | Landing page desktop with both English and Persian enabled. Shows language switcher button in header navigation. | 1440 × 900 (full) |
| `02_both_languages_landing_mobile.png` | Landing page mobile view with language toggle button visible. | 390 × 844 |
| `03_both_languages_enabled_admin.png` | Platform Admin Localization Settings page showing both English and Persian cards active with English as default. | 1440 × 900 (full) |
| `04_default_language_transition_to_persian.png` | Default platform fallback transitioned to Persian with success banner and updated default indicator. | 1440 × 900 (full) |
| `05_persian_only_admin.png` | Persian-only configuration. English is marked Disabled; Persian switch is locked (final language protection); header language switcher is completely hidden. | 1440 × 900 (full) |
| `06_persian_only_landing_selector_hidden.png` | Public landing page in Persian-only mode with language switcher completely hidden. | 1440 × 900 (full) |
| `07_persian_only_landing_mobile.png` | Mobile landing page in Persian-only mode with language switcher completely hidden. | 390 × 844 |
| `08_automatic_fallback_from_disabled_english.png` | Login view showing automatic fallback from disabled English to Persian (`fa` / RTL). | 1440 × 900 (full) |
| `09_admin_re_enabling_english.png` | Platform Admin re-enabling English from the disabled state; header switcher immediately reappears. | 1440 × 900 (full) |
| `10_english_only_admin.png` | English-only configuration. Persian is marked Disabled; English switch is locked; header switcher is hidden. | 1440 × 900 (full) |
| `11_english_only_landing_selector_hidden.png` | Public landing page in English-only mode with language switcher completely hidden. | 1440 × 900 (full) |
| `12_english_only_landing_mobile.png` | Mobile landing page in English-only mode with language switcher completely hidden. | 390 × 844 |
| `13_automatic_fallback_from_disabled_persian.png` | Login view showing automatic fallback from disabled Persian to English (`en` / LTR). | 1440 × 900 (full) |
| `14_admin_re_enabling_persian.png` | Platform Admin re-enabling Persian; both languages active. | 1440 × 900 (full) |
| `15_admin_graceful_language_transition.png` | Administrator viewing panel in Persian disabling Persian; interface seamlessly switches to English without logout or redirect. | 1440 × 900 (full) |

---

## 8. Known Limitations & Architectural Notes

1. **Fixed Dual-Language Universe:**
   As mandated by the task specification, the system supports strictly English (`en`) and Persian (`fa`). The database table `platform_localization_settings` and server validation logic explicitly reject any arbitrary third locale.
2. **Immutable Snapshot Reports:**
   Historical reports generated under a specific locale snapshot preserve their original language dictionary and format for compliance reasons. Deactivating a language globally does not alter previously published historical reports.

---

## 9. Explicit Compliance & Safety Confirmation

- **Two Languages Only:** ClientPulse strictly supports English (`en`) and Persian (`fa`). Dynamic language registration, translation pack uploads, and arbitrary language creation were prevented.
- **Migration Integrity:** Migration `000020` was preserved without mutation.
- **Production Boundary:** All development, testing, database queries, and screenshot generation were executed exclusively within the local Docker environment (`127.0.0.1:5433`, `127.0.0.1:8080`, `127.0.0.1:3001`). No production servers, production databases, or external networks were accessed.

