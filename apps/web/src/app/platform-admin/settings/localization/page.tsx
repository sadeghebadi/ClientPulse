"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import {
  Globe,
  CheckCircle2,
  AlertTriangle,
  Sparkles,
  Info,
  RefreshCw,
  Sliders,
  Languages,
  Check,
  Star,
  Loader2,
} from "lucide-react";
import { apiFetch } from "@/lib/api";
import { useI18n, Language } from "@/lib/i18n";
import { Button } from "@/components/ui/Button";
import { LoadingSkeleton } from "@/components/ui/LoadingSkeleton";

interface LocalizationSettings {
  enabled_locales: string[];
  default_locale: string;
  supported_locales?: Array<{
    code: string;
    label: string;
    native_label: string;
    dir: string;
    enabled: boolean;
    is_default: boolean;
  }>;
  updated_at?: string;
}

interface LocaleDefinition {
  code: "en" | "fa";
  name: string;
  nativeName: string;
  direction: "ltr" | "rtl";
  currencyDefault: string;
  samplePrice: string;
  sampleDate: string;
  description: string;
}

const SUPPORTED_LOCALES: LocaleDefinition[] = [
  {
    code: "en",
    name: "English",
    nativeName: "English",
    direction: "ltr",
    currencyDefault: "USD ($)",
    samplePrice: "$29.00 / month",
    sampleDate: "November 4, 2026",
    description: "Default international locale with Left-to-Right layout, Inter typography, and USD pricing.",
  },
  {
    code: "fa",
    name: "Persian",
    nativeName: "فارسی",
    direction: "rtl",
    currencyDefault: "IRR (Toman / تومان)",
    samplePrice: "۲۹۰٬۰۰۰ تومان / ماه",
    sampleDate: "۱۴ آبان ۱۴۰۵",
    description: "Iranian market locale with Right-to-Left layout, Vazirmatn typography, and Toman pricing.",
  },
];

export default function PlatformAdminLocalizationSettingsPage() {
  const { isRTL, setLanguage, language } = useI18n();
  const [settings, setSettings] = useState<LocalizationSettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [loadingLocale, setLoadingLocale] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Staged / live configuration state
  const [enabledLocales, setEnabledLocales] = useState<string[]>(["en", "fa"]);
  const [defaultLocale, setDefaultLocale] = useState<string>("en");
  const [auditReason, setAuditReason] = useState<string>("");

  const fetchSettings = async () => {
    try {
      setLoading(true);
      setError(null);
      const res = await apiFetch<LocalizationSettings | { settings: LocalizationSettings }>(
        "/api/v1/platform-admin/settings/localization"
      );
      const data = ("settings" in res && res.settings) ? res.settings : (res as LocalizationSettings);
      if (data && data.enabled_locales) {
        setSettings(data);
        setEnabledLocales(data.enabled_locales);
        setDefaultLocale(data.default_locale || "en");
      }
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Failed to load localization settings.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSettings();
  }, []);

  // Atomic configuration updater
  const executeUpdate = async (
    newEnabled: string[],
    newDefault: string,
    actionDesc: string,
    triggerLocale: string
  ) => {
    // Invariants validation
    if (newEnabled.length === 0) {
      setError("At least one language must remain enabled on the platform.");
      return;
    }
    if (!newEnabled.includes(newDefault)) {
      setError("The default platform language must be one of the enabled languages.");
      return;
    }

    try {
      setSaving(true);
      setLoadingLocale(triggerLocale);
      setError(null);
      setSuccessMessage(null);

      const res = await apiFetch<LocalizationSettings | { settings: LocalizationSettings }>(
        "/api/v1/platform-admin/settings/localization",
        {
          method: "PUT",
          body: JSON.stringify({
            enabled_locales: newEnabled,
            default_locale: newDefault,
            reason: auditReason.trim() || actionDesc,
          }),
        }
      );

      const updated = ("settings" in res && res.settings) ? res.settings : (res as LocalizationSettings);
      setSettings(updated);
      setEnabledLocales(updated.enabled_locales);
      setDefaultLocale(updated.default_locale);
      setAuditReason("");
      setSuccessMessage(actionDesc + " successfully.");

      // Invalidate frontend caches and notify all application language selectors
      if (typeof window !== "undefined") {
        window.dispatchEvent(new Event("clientpulse:localization_settings_updated"));
      }

      // If current administrator language was disabled, gracefully fallback without logging out
      if (!updated.enabled_locales.includes(language)) {
        setLanguage(updated.default_locale as Language);
      }
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Failed to update localization settings.");
      // Rollback UI state to current known server settings
      if (settings) {
        setEnabledLocales(settings.enabled_locales);
        setDefaultLocale(settings.default_locale);
      }
    } finally {
      setSaving(false);
      setLoadingLocale(null);
    }
  };

  // Toggle enable/disable for a specific locale
  const handleToggleLocale = (code: string) => {
    const isCurrentlyEnabled = enabledLocales.includes(code);

    if (isCurrentlyEnabled) {
      // Attempting to disable
      if (enabledLocales.length <= 1) {
        setError("At least one language must remain enabled on the platform.");
        return;
      }

      // If disabling the current default language:
      // Require selection of the other language as the new default and update atomically
      if (defaultLocale === code) {
        const otherCode = code === "en" ? "fa" : "en";
        executeUpdate(
          [otherCode],
          otherCode,
          `Disabled ${code === "en" ? "English" : "Persian"} and switched default to ${otherCode === "en" ? "English" : "Persian"}`,
          code
        );
        return;
      }

      // Disabling non-default language
      const newEnabled = enabledLocales.filter((c) => c !== code);
      executeUpdate(
        newEnabled,
        defaultLocale,
        `Disabled ${code === "en" ? "English" : "Persian"}`,
        code
      );
    } else {
      // Enabling language
      const newEnabled = [...enabledLocales, code];
      executeUpdate(
        newEnabled,
        defaultLocale,
        `Enabled ${code === "en" ? "English" : "Persian"}`,
        code
      );
    }
  };

  // Change default platform fallback
  const handleSetDefault = (code: string) => {
    if (defaultLocale === code) return;

    // If language is currently disabled, enable it atomically when setting as default
    const newEnabled = enabledLocales.includes(code) ? enabledLocales : [...enabledLocales, code];
    executeUpdate(
      newEnabled,
      code,
      `Set ${code === "en" ? "English" : "Persian"} as default platform language`,
      code
    );
  };

  return (
    <div className="space-y-6">
      {/* Top Banner / Breadcrumb */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-xs font-medium text-slate-500 mb-1">
            <Link href="/platform-admin" className="hover:text-slate-800 transition-colors">
              Platform Admin
            </Link>
            <span>/</span>
            <span className="text-slate-800 font-semibold">Settings</span>
            <span>/</span>
            <span className="text-slate-800 font-semibold">Localization</span>
          </div>
          <h1 className="text-xl sm:text-2xl font-black text-slate-900 tracking-tight flex items-center gap-2">
            <Globe className="w-6 h-6 text-emerald-600 shrink-0" />
            <span>Persian & English Language Activation</span>
          </h1>
          <p className="text-xs sm:text-sm text-slate-500 mt-1">
            Activate or deactivate English and Persian system-wide, manage fallback defaults, and control language visibility.
          </p>
        </div>

        <div className="flex items-center gap-2 self-start sm:self-auto">
          <Button
            variant="secondary"
            size="sm"
            onClick={fetchSettings}
            disabled={loading || saving}
            className="gap-1.5"
            data-testid="reload-localization-btn"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? "animate-spin" : ""}`} />
            <span>Reload</span>
          </Button>
        </div>
      </div>

      {/* Success Notification Banner */}
      {successMessage && (
        <div
          data-testid="localization-success-alert"
          className="p-4 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-900 text-xs flex items-center justify-between shadow-sm animate-in fade-in"
        >
          <div className="flex items-center gap-2.5">
            <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
            <span className="font-semibold">{successMessage}</span>
          </div>
          <button
            onClick={() => setSuccessMessage(null)}
            className="text-emerald-700 hover:text-emerald-900 font-bold text-sm ml-4"
          >
            &times;
          </button>
        </div>
      )}

      {/* Error Notification Banner */}
      {error && (
        <div
          data-testid="localization-error-alert"
          className="p-4 rounded-xl bg-rose-50 border border-rose-200 text-rose-900 text-xs flex items-center justify-between shadow-sm animate-in fade-in"
        >
          <div className="flex items-center gap-2.5">
            <AlertTriangle className="w-4 h-4 text-rose-600 shrink-0" />
            <span className="font-semibold">{error}</span>
          </div>
          <button
            onClick={() => setError(null)}
            className="text-rose-700 hover:text-rose-900 font-bold text-sm ml-4"
          >
            &times;
          </button>
        </div>
      )}

      {loading ? (
        <div className="space-y-4">
          <LoadingSkeleton className="h-28 w-full rounded-2xl" />
          <LoadingSkeleton className="h-80 w-full rounded-2xl" />
        </div>
      ) : (
        <div className="space-y-6">
          {/* Quick Metrics Bar */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div className="bg-white rounded-2xl p-4 border border-slate-200 shadow-sm flex items-center justify-between">
              <div>
                <p className="text-[11px] font-bold uppercase tracking-wider text-slate-400">
                  Enabled Languages
                </p>
                <p className="text-xl font-black text-slate-900 mt-0.5">
                  {enabledLocales.length} of 2
                </p>
              </div>
              <div className="w-10 h-10 rounded-xl bg-emerald-50 text-emerald-600 flex items-center justify-center font-bold">
                <Languages className="w-5 h-5" />
              </div>
            </div>

            <div className="bg-white rounded-2xl p-4 border border-slate-200 shadow-sm flex items-center justify-between">
              <div>
                <p className="text-[11px] font-bold uppercase tracking-wider text-slate-400">
                  Default Platform Fallback
                </p>
                <p className="text-xl font-black text-slate-900 mt-0.5">
                  {defaultLocale === "en" ? "English (en)" : "فارسی (fa)"}
                </p>
              </div>
              <div className="w-10 h-10 rounded-xl bg-blue-50 text-blue-600 flex items-center justify-center font-bold">
                <Globe className="w-5 h-5" />
              </div>
            </div>

            <div className="bg-white rounded-2xl p-4 border border-slate-200 shadow-sm flex items-center justify-between">
              <div>
                <p className="text-[11px] font-bold uppercase tracking-wider text-slate-400">
                  Configuration Mode
                </p>
                <p className="text-sm font-bold text-emerald-600 mt-1 flex items-center gap-1.5">
                  <span className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />
                  PostgreSQL Singleton (Live)
                </p>
              </div>
              <div className="w-10 h-10 rounded-xl bg-slate-50 text-slate-500 flex items-center justify-center font-bold">
                <Sliders className="w-5 h-5" />
              </div>
            </div>
          </div>

          {/* Exactly Two Built-in Language Cards */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {SUPPORTED_LOCALES.map((locale) => {
              const isEnabled = enabledLocales.includes(locale.code);
              const isDefault = defaultLocale === locale.code;
              const isOnlyEnabled = isEnabled && enabledLocales.length === 1;
              const isOperating = saving && loadingLocale === locale.code;

              return (
                <div
                  key={locale.code}
                  data-testid={`language-card-${locale.code}`}
                  className={`bg-white rounded-2xl border transition-all duration-200 p-6 flex flex-col justify-between shadow-sm relative overflow-hidden ${
                    isEnabled
                      ? isDefault
                        ? "border-emerald-500 ring-2 ring-emerald-500/20 shadow-md"
                        : "border-slate-300"
                      : "border-slate-200 bg-slate-50/70 opacity-85"
                  }`}
                >
                  {/* Default Language Badge */}
                  {isDefault && (
                    <div className="absolute top-0 right-0 bg-emerald-600 text-white text-[10px] font-black uppercase tracking-wider px-3.5 py-1 rounded-bl-xl shadow-xs flex items-center gap-1">
                      <Star className="w-3 h-3 fill-white" />
                      <span>Platform Default</span>
                    </div>
                  )}

                  <div>
                    {/* Header: Code & Native Name */}
                    <div className="flex items-center justify-between gap-3 mb-4">
                      <div className="flex items-center gap-3">
                        <span className="text-xs font-mono font-black bg-slate-100 text-slate-800 px-3 py-1.5 rounded-lg border border-slate-200 uppercase">
                          {locale.code}
                        </span>
                        <div>
                          <h2 className="text-lg font-black text-slate-900 flex items-center gap-2">
                            <span>{locale.name}</span>
                            <span className="text-sm font-semibold text-slate-500">
                              ({locale.nativeName})
                            </span>
                          </h2>
                          <span className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
                            Direction: {locale.direction.toUpperCase()}
                          </span>
                        </div>
                      </div>

                      {/* Status Pill */}
                      <div>
                        {isEnabled ? (
                          <span
                            data-testid={`status-badge-${locale.code}-active`}
                            className="inline-flex items-center gap-1 bg-emerald-50 text-emerald-700 border border-emerald-200 px-3 py-1 rounded-full text-xs font-bold"
                          >
                            <CheckCircle2 className="w-3.5 h-3.5" />
                            <span>Active</span>
                          </span>
                        ) : (
                          <span
                            data-testid={`status-badge-${locale.code}-disabled`}
                            className="inline-flex items-center gap-1 bg-slate-100 text-slate-500 border border-slate-200 px-3 py-1 rounded-full text-xs font-bold"
                          >
                            <span>Disabled</span>
                          </span>
                        )}
                      </div>
                    </div>

                    <p className="text-xs text-slate-600 leading-relaxed mb-4">
                      {locale.description}
                    </p>

                    {/* Live Regional Formatting Preview */}
                    <div className="bg-slate-50 rounded-xl p-4 border border-slate-200/80 mb-5 space-y-2.5">
                      <div className="text-[10px] font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1">
                        <Sparkles className="w-3 h-3 text-slate-400" />
                        <span>Live Regional Preview</span>
                      </div>

                      <div className="grid grid-cols-2 gap-3 text-xs">
                        <div>
                          <span className="text-slate-400 text-[11px] block">Direction:</span>
                          <span className="font-bold text-slate-800 uppercase font-mono">
                            {locale.direction}
                          </span>
                        </div>
                        <div>
                          <span className="text-slate-400 text-[11px] block">Primary Currency:</span>
                          <span className="font-semibold text-slate-800">
                            {locale.currencyDefault}
                          </span>
                        </div>
                        <div>
                          <span className="text-slate-400 text-[11px] block">Sample Price:</span>
                          <span className="font-mono font-bold text-slate-900">
                            {locale.samplePrice}
                          </span>
                        </div>
                        <div>
                          <span className="text-slate-400 text-[11px] block">Date Presentation:</span>
                          <span className="font-medium text-slate-800">
                            {locale.sampleDate}
                          </span>
                        </div>
                      </div>
                    </div>
                  </div>

                  {/* Operational Controls */}
                  <div className="pt-4 border-t border-slate-100 space-y-3.5">
                    {/* Toggle Switch */}
                    <div className="flex items-center justify-between gap-3">
                      <div>
                        <span className="text-xs font-bold text-slate-800 block">
                          Language Activation Switch
                        </span>
                        <span className="text-[11px] text-slate-500">
                          {isEnabled
                            ? `Visible across all website language selectors.`
                            : `Hidden from all public and authenticated selectors.`}
                        </span>
                      </div>

                      <div className="flex items-center gap-2">
                        {isOnlyEnabled && (
                          <span className="text-[10px] font-bold text-amber-700 bg-amber-50 px-2.5 py-1 rounded-md border border-amber-200">
                            Final Active Language
                          </span>
                        )}

                        <button
                          type="button"
                          role="switch"
                          aria-checked={isEnabled}
                          disabled={saving || isOnlyEnabled}
                          onClick={() => handleToggleLocale(locale.code)}
                          data-testid={`switch-${locale.code}`}
                          className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 ${
                            isEnabled ? "bg-emerald-600" : "bg-slate-300"
                          }`}
                        >
                          <span className="sr-only">Toggle {locale.name}</span>
                          <span
                            aria-hidden="true"
                            className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow-md ring-0 transition duration-200 ease-in-out flex items-center justify-center ${
                              isEnabled ? "translate-x-5" : "translate-x-0"
                            }`}
                          >
                            {isOperating && (
                              <Loader2 className="w-3 h-3 text-emerald-600 animate-spin" />
                            )}
                          </span>
                        </button>
                      </div>
                    </div>

                    {/* Default Language Selector Button / Indicator */}
                    <div className="flex items-center justify-between gap-3 pt-2 border-t border-slate-100/70">
                      <div>
                        <span className="text-xs font-bold text-slate-800 block">
                          Platform Fallback Default
                        </span>
                        <span className="text-[11px] text-slate-500">
                          Applied when a user&apos;s language is unassigned or disabled.
                        </span>
                      </div>

                      <div>
                        {isDefault ? (
                          <div
                            data-testid={`default-indicator-${locale.code}`}
                            className="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg bg-emerald-50 text-emerald-800 border border-emerald-200 text-xs font-bold"
                          >
                            <Check className="w-3.5 h-3.5 text-emerald-600" />
                            <span>Current Default</span>
                          </div>
                        ) : (
                          <Button
                            type="button"
                            variant="secondary"
                            size="sm"
                            disabled={saving}
                            loading={isOperating}
                            onClick={() => handleSetDefault(locale.code)}
                            data-testid={`set-default-btn-${locale.code}`}
                            className="text-xs font-semibold text-slate-700 hover:text-slate-900 border-slate-200"
                          >
                            Set as Default
                          </Button>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>

          {/* Operational Invariants & Behavior Guide */}
          <div className="bg-slate-50 border border-slate-200 rounded-2xl p-5 text-xs text-slate-600 space-y-2.5">
            <div className="flex items-center gap-2 text-slate-900 font-bold text-sm">
              <Info className="w-4 h-4 text-blue-600 shrink-0" />
              <span>Language Activation Behavioral Invariants</span>
            </div>
            <ul className="list-disc pl-5 space-y-1.5 leading-relaxed text-slate-600">
              <li>
                <strong>Single Active Language (Scenario D):</strong> When only one language is enabled, the language switcher button or dropdown disappears completely from all site navigation, login, and application views to prevent redundant single-option controls.
              </li>
              <li>
                <strong>Automatic Locale Fallback (Scenario B &amp; C):</strong> Any visitor or authenticated user whose saved language is disabled automatically falls back to <code className="text-slate-800 bg-slate-200 font-bold px-1 py-0.5 rounded">{defaultLocale}</code>, updating <code className="text-slate-800 bg-slate-200 font-bold px-1 py-0.5 rounded">localStorage</code> and document direction (<code className="text-slate-800 bg-slate-200 font-bold px-1 py-0.5 rounded">dir</code>) without terminating active sessions.
              </li>
              <li>
                <strong>Platform Admin Accessibility:</strong> Disabling the language currently used by an administrator seamlessly transitions the admin interface to English while maintaining access to <code className="text-slate-800 bg-slate-200 font-bold px-1 py-0.5 rounded">/platform-admin</code> and enabling future re-activation.
              </li>
              <li>
                <strong>Atomic Default Management:</strong> Disabling the platform default language atomically designates the remaining active language as the new default fallback.
              </li>
            </ul>
          </div>

          {/* Audit Log Rationale Section */}
          <div className="bg-white rounded-2xl border border-slate-200 p-5 shadow-sm space-y-3">
            <div>
              <label className="block text-xs font-bold uppercase tracking-wider text-slate-700 mb-1">
                Custom Audit Log Reason (Optional)
              </label>
              <input
                type="text"
                value={auditReason}
                onChange={(e) => setAuditReason(e.target.value)}
                placeholder="e.g. Disabling Persian temporarily for system maintenance or rollout"
                className="w-full text-xs px-3.5 py-2.5 rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 bg-white"
                data-testid="localization-audit-reason-input"
              />
              <p className="text-[11px] text-slate-400 mt-1">
                Recorded in the immutable platform administrator audit trail with actor identity and timestamp.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
