"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import {
  Globe,
  CheckCircle2,
  AlertTriangle,
  ShieldAlert,
  ArrowRight,
  ArrowLeft,
  Sparkles,
  Info,
  RefreshCw,
  Sliders,
  Radio,
  Clock,
  Languages,
} from "lucide-react";
import { apiFetch } from "@/lib/api";
import { useI18n, Language } from "@/lib/i18n";
import { Button } from "@/components/ui/Button";
import { LoadingSkeleton } from "@/components/ui/LoadingSkeleton";
import { ErrorState } from "@/components/ui/ErrorState";

interface LocalizationSettings {
  id: number;
  enabled_locales: string[];
  default_locale: string;
  updated_at: string;
}

interface LocaleDefinition {
  code: string;
  name: string;
  nativeName: string;
  direction: "ltr" | "rtl";
  currencyDefault: string;
  samplePrice: number;
  sampleDate: string;
  description: string;
}

const SUPPORTED_LOCALES: LocaleDefinition[] = [
  {
    code: "en",
    name: "English",
    nativeName: "English (US / Global)",
    direction: "ltr",
    currencyDefault: "USD ($)",
    samplePrice: 2900, // $29.00
    sampleDate: "November 4, 2026",
    description: "Default international locale with Left-to-Right layout, USD currency, and Western Arabic numerals.",
  },
  {
    code: "fa",
    nativeName: "فارسی (ایران)",
    name: "Persian",
    direction: "rtl",
    currencyDefault: "IRR (Toman / تومان)",
    samplePrice: 290000, // 290,000 Toman
    sampleDate: "۱۴ آبان ۱۴۰۵",
    description: "Iranian market locale with Right-to-Left layout, Vazirmatn typography, Toman denomination, and Persian numerals.",
  },
];

export default function PlatformAdminLocalizationSettingsPage() {
  const { t, isRTL, setLanguage, language } = useI18n();
  const [settings, setSettings] = useState<LocalizationSettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Staged form state
  const [enabledLocales, setEnabledLocales] = useState<string[]>(["en", "fa"]);
  const [defaultLocale, setDefaultLocale] = useState<string>("en");
  const [auditReason, setAuditReason] = useState<string>("");

  const fetchSettings = async () => {
    try {
      setLoading(true);
      setError(null);
      const res = await apiFetch<{ settings: LocalizationSettings }>(
        "/api/v1/platform-admin/settings/localization"
      );
      if (res && res.settings) {
        setSettings(res.settings);
        setEnabledLocales(res.settings.enabled_locales || ["en"]);
        setDefaultLocale(res.settings.default_locale || "en");
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

  const handleToggleLocale = (code: string) => {
    const isCurrentlyEnabled = enabledLocales.includes(code);

    if (isCurrentlyEnabled) {
      // Trying to disable
      if (enabledLocales.length <= 1) {
        setError("At least one language must remain enabled on the platform.");
        return;
      }
      if (defaultLocale === code) {
        setError("You cannot disable the current default platform language. Set another language as default first.");
        return;
      }
      setError(null);
      setEnabledLocales((prev) => prev.filter((c) => c !== code));
    } else {
      // Enabling
      setError(null);
      setEnabledLocales((prev) => [...prev, code]);
    }
  };

  const handleSetDefault = (code: string) => {
    if (!enabledLocales.includes(code)) {
      // Auto-enable if setting as default
      setEnabledLocales((prev) => [...prev, code]);
    }
    setDefaultLocale(code);
    setError(null);
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();

    if (enabledLocales.length === 0) {
      setError("At least one language must be enabled.");
      return;
    }
    if (!enabledLocales.includes(defaultLocale)) {
      setError("The default language must be one of the enabled languages.");
      return;
    }

    try {
      setSaving(true);
      setError(null);
      setSuccessMessage(null);

      const res = await apiFetch<{ settings: LocalizationSettings }>(
        "/api/v1/platform-admin/settings/localization",
        {
          method: "PUT",
          body: JSON.stringify({
            enabled_locales: enabledLocales,
            default_locale: defaultLocale,
            reason: auditReason.trim() || undefined,
          }),
        }
      );

      if (res && res.settings) {
        setSettings(res.settings);
        setEnabledLocales(res.settings.enabled_locales);
        setDefaultLocale(res.settings.default_locale);
        setAuditReason("");
        setSuccessMessage("Global localization settings have been saved and public caches invalidated.");

        // If current language was disabled, fallback to default
        if (!res.settings.enabled_locales.includes(language)) {
          setLanguage(res.settings.default_locale as Language);
        }
      }
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Failed to update localization settings.");
    } finally {
      setSaving(false);
    }
  };

  const hasUnsavedChanges =
    settings &&
    (JSON.stringify(settings.enabled_locales.slice().sort()) !==
      JSON.stringify(enabledLocales.slice().sort()) ||
      settings.default_locale !== defaultLocale ||
      auditReason.length > 0);

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
            <Globe className="w-6 h-6 text-emerald-600" />
            <span>Centralized Language & Market Activation</span>
          </h1>
          <p className="text-xs sm:text-sm text-slate-500 mt-1">
            Activate or deactivate platform-wide languages, define the fallback default, and preview formatting.
          </p>
        </div>

        <div className="flex items-center gap-2 self-start sm:self-auto">
          <Button
            variant="secondary"
            size="sm"
            onClick={fetchSettings}
            disabled={loading || saving}
            className="gap-1.5"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? "animate-spin" : ""}`} />
            <span>Reload</span>
          </Button>
        </div>
      </div>

      {/* Notifications */}
      {successMessage && (
        <div className="p-4 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-900 text-xs flex items-center justify-between shadow-sm animate-in fade-in">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
            <span className="font-medium">{successMessage}</span>
          </div>
          <button
            onClick={() => setSuccessMessage(null)}
            className="text-emerald-700 hover:text-emerald-900 font-bold text-sm ml-4"
          >
            &times;
          </button>
        </div>
      )}

      {error && (
        <div className="p-4 rounded-xl bg-rose-50 border border-rose-200 text-rose-900 text-xs flex items-center justify-between shadow-sm animate-in fade-in">
          <div className="flex items-center gap-2">
            <AlertTriangle className="w-4 h-4 text-rose-600 shrink-0" />
            <span className="font-medium">{error}</span>
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
          <LoadingSkeleton className="h-44 w-full rounded-2xl" />
          <LoadingSkeleton className="h-64 w-full rounded-2xl" />
        </div>
      ) : (
        <form onSubmit={handleSave} className="space-y-6">
          {/* Quick Metrics Bar */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div className="bg-white rounded-2xl p-4 border border-slate-200 shadow-sm flex items-center justify-between">
              <div>
                <p className="text-[11px] font-bold uppercase tracking-wider text-slate-400">
                  Enabled Languages
                </p>
                <p className="text-xl font-black text-slate-900 mt-0.5">
                  {enabledLocales.length} of {SUPPORTED_LOCALES.length}
                </p>
              </div>
              <div className="w-10 h-10 rounded-xl bg-emerald-50 text-emerald-600 flex items-center justify-center font-bold">
                <Languages className="w-5 h-5" />
              </div>
            </div>

            <div className="bg-white rounded-2xl p-4 border border-slate-200 shadow-sm flex items-center justify-between">
              <div>
                <p className="text-[11px] font-bold uppercase tracking-wider text-slate-400">
                  Default Fallback
                </p>
                <p className="text-xl font-black text-slate-900 mt-0.5 uppercase">
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
                  Source of Truth: Postgres
                </p>
              </div>
              <div className="w-10 h-10 rounded-xl bg-slate-50 text-slate-500 flex items-center justify-center font-bold">
                <Sliders className="w-5 h-5" />
              </div>
            </div>
          </div>

          {/* Language Cards */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {SUPPORTED_LOCALES.map((locale) => {
              const isEnabled = enabledLocales.includes(locale.code);
              const isDefault = defaultLocale === locale.code;
              const isOnlyEnabled = isEnabled && enabledLocales.length === 1;

              return (
                <div
                  key={locale.code}
                  className={`bg-white rounded-2xl border transition-all duration-200 p-6 flex flex-col justify-between shadow-sm relative overflow-hidden ${
                    isEnabled
                      ? isDefault
                        ? "border-emerald-500 ring-2 ring-emerald-500/20"
                        : "border-slate-300"
                      : "border-slate-200 bg-slate-50/50 opacity-80"
                  }`}
                >
                  {isDefault && (
                    <div className="absolute top-0 right-0 bg-emerald-600 text-white text-[10px] font-black uppercase tracking-wider px-3 py-1 rounded-bl-xl shadow-xs">
                      Platform Default
                    </div>
                  )}

                  <div>
                    <div className="flex items-center justify-between gap-3 mb-3">
                      <div className="flex items-center gap-2.5">
                        <span className="text-xs font-mono font-bold bg-slate-100 text-slate-700 px-2.5 py-1 rounded-lg border border-slate-200 uppercase">
                          {locale.code}
                        </span>
                        <div>
                          <h3 className="text-base font-bold text-slate-900 flex items-center gap-2">
                            <span>{locale.name}</span>
                            <span className="text-xs font-normal text-slate-500">
                              ({locale.nativeName})
                            </span>
                          </h3>
                        </div>
                      </div>

                      <div>
                        {isEnabled ? (
                          <span className="inline-flex items-center gap-1 bg-emerald-50 text-emerald-700 border border-emerald-200 px-2.5 py-1 rounded-full text-xs font-semibold">
                            <CheckCircle2 className="w-3.5 h-3.5" />
                            <span>Active</span>
                          </span>
                        ) : (
                          <span className="inline-flex items-center gap-1 bg-slate-100 text-slate-500 border border-slate-200 px-2.5 py-1 rounded-full text-xs font-semibold">
                            <span>Disabled</span>
                          </span>
                        )}
                      </div>
                    </div>

                    <p className="text-xs text-slate-600 leading-relaxed mb-4">
                      {locale.description}
                    </p>

                    {/* Live Preview Box */}
                    <div className="bg-slate-50 rounded-xl p-4 border border-slate-200/80 mb-5 space-y-2.5">
                      <div className="text-[10px] font-bold uppercase tracking-wider text-slate-400 flex items-center gap-1">
                        <Sparkles className="w-3 h-3 text-slate-400" />
                        <span>Live Regional Preview</span>
                      </div>

                      <div className="grid grid-cols-2 gap-3 text-xs">
                        <div>
                          <span className="text-slate-400 text-[11px] block">Direction:</span>
                          <span className="font-semibold text-slate-800 uppercase font-mono">
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
                            {locale.code === "fa"
                              ? "۲۹۰٬۰۰۰ تومان / ماه"
                              : "$29.00 / month"}
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

                  {/* Operational Controls for this Locale */}
                  <div className="pt-4 border-t border-slate-100 space-y-3">
                    <div className="flex items-center justify-between gap-3">
                      <label className="text-xs font-semibold text-slate-700 flex items-center gap-2 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={isEnabled}
                          disabled={isOnlyEnabled || (isDefault && isEnabled)}
                          onChange={() => handleToggleLocale(locale.code)}
                          className="w-4 h-4 rounded text-emerald-600 focus:ring-emerald-500 border-slate-300 disabled:opacity-50"
                        />
                        <span>Enable {locale.name} across platform</span>
                      </label>

                      {isOnlyEnabled && (
                        <span className="text-[10px] font-medium text-amber-600 bg-amber-50 px-2 py-0.5 rounded border border-amber-200">
                          Last active language
                        </span>
                      )}
                    </div>

                    <div className="flex items-center justify-between gap-3">
                      <label className="text-xs font-semibold text-slate-700 flex items-center gap-2 cursor-pointer">
                        <input
                          type="radio"
                          name="default_locale_selector"
                          value={locale.code}
                          checked={isDefault}
                          onChange={() => handleSetDefault(locale.code)}
                          className="w-4 h-4 text-emerald-600 focus:ring-emerald-500 border-slate-300"
                        />
                        <span>Set as Default Platform Fallback</span>
                      </label>

                      {isDefault && (
                        <span className="text-[10px] font-bold text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded border border-emerald-200">
                          Current Default
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>

          {/* Fallback & Behavior Notice */}
          <div className="bg-slate-50 border border-slate-200 rounded-2xl p-5 text-xs text-slate-600 space-y-2">
            <div className="flex items-center gap-2 text-slate-900 font-bold text-sm">
              <Info className="w-4 h-4 text-blue-600" />
              <span>Locale Behavior Invariants</span>
            </div>
            <ul className="list-disc pl-5 space-y-1 leading-relaxed text-slate-600">
              <li>
                <strong>Disabled Languages:</strong> Immediately hidden from public language switchers, user profiles, and signup views.
              </li>
              <li>
                <strong>Graceful Fallback:</strong> Any authenticated or guest user with a disabled saved language automatically falls back to <code className="text-slate-800 bg-slate-200 px-1 py-0.5 rounded">{defaultLocale}</code> without session termination.
              </li>
              <li>
                <strong>Public Localization API:</strong> The endpoint <code className="text-slate-800 bg-slate-200 px-1 py-0.5 rounded">GET /api/v1/public/localization</code> delivers cached public status and invalidates immediately upon saving.
              </li>
            </ul>
          </div>

          {/* Audit Reason & Submit Action */}
          <div className="bg-white rounded-2xl border border-slate-200 p-6 shadow-sm space-y-4">
            <div>
              <label className="block text-xs font-bold uppercase tracking-wider text-slate-700 mb-1">
                Audit Log Reason (Optional)
              </label>
              <input
                type="text"
                value={auditReason}
                onChange={(e) => setAuditReason(e.target.value)}
                placeholder="e.g. Enabling Persian language and Iranian market pricing for Q4 rollout"
                className="w-full text-xs px-3.5 py-2.5 rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 bg-white"
              />
              <p className="text-[11px] text-slate-400 mt-1">
                This rationale will be recorded in the immutable platform administrator audit trail.
              </p>
            </div>

            <div className="flex items-center justify-between pt-2 border-t border-slate-100">
              <div className="text-xs text-slate-500">
                {hasUnsavedChanges ? (
                  <span className="text-amber-600 font-medium flex items-center gap-1.5">
                    <span className="w-2 h-2 rounded-full bg-amber-500 animate-pulse" />
                    Unsaved localization changes detected.
                  </span>
                ) : (
                  <span>Settings are synchronized with the database.</span>
                )}
              </div>

              <div className="flex items-center gap-3">
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={() => {
                    if (settings) {
                      setEnabledLocales(settings.enabled_locales);
                      setDefaultLocale(settings.default_locale);
                      setAuditReason("");
                      setError(null);
                    }
                  }}
                  disabled={!hasUnsavedChanges || saving}
                >
                  Discard
                </Button>
                <Button
                  type="submit"
                  variant="primary"
                  size="sm"
                  loading={saving}
                  disabled={saving || !hasUnsavedChanges}
                  className="gap-1.5"
                >
                  <CheckCircle2 className="w-4 h-4" />
                  <span>Save Configuration</span>
                </Button>
              </div>
            </div>
          </div>
        </form>
      )}
    </div>
  );
}

