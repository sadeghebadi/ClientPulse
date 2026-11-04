"use client";

import React from "react";
import { Languages } from "lucide-react";
import { useI18n } from "@/lib/i18n";

interface LanguageSelectorProps {
  variant?: "header" | "compact" | "dropdown";
  className?: string;
}

export function LanguageSelector({ className = "" }: LanguageSelectorProps) {
  const { locale, toggleLocale, enabledLocales, canToggleLanguage } = useI18n();

  // Scenario D: When only one language is enabled, hide the selector entirely.
  // Do not display a redundant selector containing only one option.
  if (!canToggleLanguage || enabledLocales.length <= 1) {
    return null;
  }

  // When both languages are enabled:
  const label = locale === "en" ? "فارسی" : "English";
  const shortLabel = locale === "en" ? "FA" : "EN";

  return (
    <button
      type="button"
      onClick={toggleLocale}
      className={`inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold text-slate-700 bg-white hover:bg-slate-50 border border-slate-200 rounded-lg shadow-2xs transition-colors shrink-0 ${className}`}
      title={locale === "en" ? "Switch to Persian (فارسی)" : "تغییر زبان به انگلیسی (English)"}
      aria-label={locale === "en" ? "Switch to Persian" : "تغییر زبان به انگلیسی"}
      data-testid="language-selector-button"
    >
      <Languages className="w-3.5 h-3.5 text-slate-500 shrink-0" />
      <span className="hidden sm:inline font-medium">{label}</span>
      <span className="sm:hidden text-[11px] font-bold">{shortLabel}</span>
    </button>
  );
}

export default LanguageSelector;

