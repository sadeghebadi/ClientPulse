"use client";

import React from "react";
import { Activity, Globe } from "lucide-react";

interface HeaderProps {
  locale: "en" | "fa";
  onToggleLocale: () => void;
}

export function Header({ locale, onToggleLocale }: HeaderProps) {
  const isRTL = locale === "fa";

  return (
    <header className="border-b border-slate-200 bg-white sticky top-0 z-50 shadow-sm">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-blue-600 flex items-center justify-center text-white shadow-md">
            <Activity className="w-6 h-6" />
          </div>
          <div>
            <h1 className="text-xl font-bold text-slate-900 tracking-tight">ClientPulse</h1>
            <p className="text-xs text-slate-500 font-medium">
              {isRTL ? "سامانه پایش وب‌سایت و گزارش‌های دوره‌ای آژانس" : "Website Monitoring & Agency Care Reports"}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-4">
          <button
            onClick={onToggleLocale}
            className="flex items-center gap-2 px-3 py-1.5 text-xs font-semibold text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-md transition-colors"
            title="Toggle Language / تغییر زبان"
          >
            <Globe className="w-3.5 h-3.5" />
            <span>{locale === "en" ? "فارسی (FA)" : "English (EN)"}</span>
          </button>
          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-100 text-emerald-800">
            Phase 0 Foundation
          </span>
        </div>
      </div>
    </header>
  );
}

