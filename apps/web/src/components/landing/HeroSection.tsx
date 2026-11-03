"use client";

import React from "react";
import Link from "next/link";
import {
  ShieldCheck,
  ArrowRight,
  ArrowLeft,
  UserPlus,
  Compass,
  Activity,
  CheckCircle2,
  Clock,
  Radio,
  Server,
  Lock,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";

export function HeroSection() {
  const { t, isRTL } = useI18n();

  return (
    <section className="relative overflow-hidden pt-10 pb-16 sm:pt-16 sm:pb-24 lg:pt-20 lg:pb-32 bg-gradient-to-b from-white via-slate-50/70 to-slate-100/50">
      {/* Background Subtle Glow Grid */}
      <div className="absolute inset-0 bg-[radial-gradient(#3b82f6_1px,transparent_1px)] [background-size:24px_24px] opacity-[0.06] pointer-events-none" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Hero Top Copy */}
        <div className="max-w-3xl mx-auto text-center space-y-5 sm:space-y-6">
          {/* Target Audience Badge */}
          <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full text-xs font-bold text-blue-700 bg-blue-50 border border-blue-200/80 shadow-2xs">
            <ShieldCheck className="w-4 h-4 text-blue-600 shrink-0" />
            <span>{t.landingHeroBadge}</span>
          </div>

          {/* Primary Headline */}
          <h1 className="text-3xl sm:text-5xl lg:text-6xl font-black text-slate-900 tracking-tight leading-[1.15] sm:leading-[1.15]">
            {t.landingHeroHeadline}
          </h1>

          {/* Supporting Subtitle */}
          <p className="text-base sm:text-xl text-slate-600 leading-relaxed max-w-2xl mx-auto font-normal">
            {t.landingHeroSubtitle}
          </p>

          {/* Dual Call To Actions */}
          <div className="pt-2 sm:pt-4 flex flex-col sm:flex-row items-center justify-center gap-3 sm:gap-4">
            <Link
              href="/signup"
              className="w-full sm:w-auto inline-flex items-center justify-center gap-2 px-7 py-3.5 text-sm font-bold text-white bg-blue-600 hover:bg-blue-700 rounded-xl shadow-md hover:shadow-lg transition-all group"
            >
              <UserPlus className="w-4 h-4" />
              <span>{t.landingHeroCtaPrimary}</span>
              {isRTL ? (
                <ArrowLeft className="w-4 h-4 group-hover:-translate-x-1 transition-transform" />
              ) : (
                <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
              )}
            </Link>

            <a
              href="#features"
              className="w-full sm:w-auto inline-flex items-center justify-center gap-2 px-6 py-3.5 text-sm font-semibold text-slate-700 bg-white hover:bg-slate-50 border border-slate-200 rounded-xl shadow-2xs hover:border-slate-300 transition-colors"
            >
              <Compass className="w-4 h-4 text-slate-500" />
              <span>{t.landingHeroCtaSecondary}</span>
            </a>
          </div>

          {/* Operational Engine Trust Notice */}
          <div className="pt-2 flex flex-wrap items-center justify-center gap-2 text-xs text-slate-500">
            <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg bg-white/80 border border-slate-200/60 shadow-2xs">
              <Server className="w-3.5 h-3.5 text-blue-600" />
              <span>{t.landingHeroEngineBadge}</span>
            </span>
          </div>
        </div>

        {/* ======================================================== */}
        {/* Authentic High-Fidelity Executive Dashboard Preview */}
        {/* ======================================================== */}
        <div className="mt-12 sm:mt-16 lg:mt-20 max-w-5xl mx-auto">
          <div className="rounded-2xl sm:rounded-3xl border border-slate-300/80 bg-white shadow-2xl overflow-hidden ring-1 ring-slate-900/5">
            {/* Realistic Browser Chrome Header */}
            <div className="bg-slate-900 px-4 py-3 sm:px-6 flex items-center justify-between gap-4 border-b border-slate-800">
              <div className="flex items-center gap-2">
                <span className="w-3 h-3 rounded-full bg-rose-500/80 inline-block" />
                <span className="w-3 h-3 rounded-full bg-amber-500/80 inline-block" />
                <span className="w-3 h-3 rounded-full bg-emerald-500/80 inline-block" />
              </div>

              {/* Anonymized Safe URL Pill */}
              <div className="flex items-center gap-2 px-3 py-1 rounded-lg bg-slate-800/80 text-slate-300 text-[11px] font-mono border border-slate-700/60 max-w-xs sm:max-w-md w-full justify-center truncate">
                <Lock className="w-3 h-3 text-emerald-400 shrink-0" />
                <span className="truncate">https://mindu.ir/dashboard</span>
              </div>

              <div className="flex items-center gap-1.5 text-xs text-slate-400 font-semibold shrink-0">
                <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
                <span className="hidden sm:inline">LIVE</span>
              </div>
            </div>

            {/* Dashboard Inner Canvas Preview */}
            <div className="p-4 sm:p-6 lg:p-8 bg-slate-50 space-y-5">
              {/* System Health Bar Preview */}
              <div className="bg-white rounded-xl border border-emerald-200/90 shadow-2xs p-3.5 sm:p-4 flex flex-col md:flex-row md:items-center justify-between gap-3">
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-lg bg-emerald-500/15 text-emerald-600 flex items-center justify-center shrink-0">
                    <CheckCircle2 className="w-5 h-5" />
                  </div>
                  <div>
                    <div className="text-xs sm:text-sm font-bold text-slate-900">
                      {isRTL
                        ? "تمام سیستم‌ها پایدارند · ۱۰۰٪ اهداف پایش در وضعیت سالم"
                        : "All Systems Operational · 100% of monitored targets healthy"}
                    </div>
                    <div className="text-[11px] text-slate-500">
                      {isRTL
                        ? "۲۴ مانیتور و هارت‌بیت فعال بدون هیچ رخداد باز"
                        : "24 active monitors & background heartbeats with zero open incidents"}
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-2 text-xs font-semibold shrink-0">
                  <span className="px-2 py-0.5 rounded-md bg-emerald-50 text-emerald-700 border border-emerald-200">
                    24 / 24 Up
                  </span>
                  <span className="px-2 py-0.5 rounded-md bg-slate-100 text-slate-600">
                    0 Outages
                  </span>
                </div>
              </div>

              {/* 4 Primary KPI Cards Preview */}
              <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
                <div className="bg-white p-3.5 sm:p-4 rounded-xl border border-slate-200 shadow-2xs space-y-1">
                  <div className="text-[11px] font-semibold text-slate-500 truncate">
                    {t.monitoredTargetsKpi}
                  </div>
                  <div className="text-xl sm:text-2xl font-extrabold text-slate-900 tabular-nums">
                    24
                  </div>
                  <div className="text-[10px] text-emerald-600 font-medium flex items-center gap-1">
                    <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
                    <span>{isRTL ? "تمام دارایی‌ها فعال" : "All targets verified"}</span>
                  </div>
                </div>

                <div className="bg-white p-3.5 sm:p-4 rounded-xl border border-slate-200 shadow-2xs space-y-1">
                  <div className="text-[11px] font-semibold text-slate-500 truncate">
                    {t.targetsDownKpi}
                  </div>
                  <div className="text-xl sm:text-2xl font-extrabold text-emerald-600 tabular-nums">
                    0
                  </div>
                  <div className="text-[10px] text-slate-500 font-medium">
                    {isRTL ? "بدون قطعی وب‌سایت" : "Zero confirmed downtime"}
                  </div>
                </div>

                <div className="bg-white p-3.5 sm:p-4 rounded-xl border border-slate-200 shadow-2xs space-y-1">
                  <div className="text-[11px] font-semibold text-slate-500 truncate">
                    {t.openIncidentsKpi}
                  </div>
                  <div className="text-xl sm:text-2xl font-extrabold text-slate-900 tabular-nums">
                    0
                  </div>
                  <div className="text-[10px] text-emerald-600 font-medium">
                    {isRTL ? "تمام رخدادها حل‌شده" : "Clean resolution history"}
                  </div>
                </div>

                <div className="bg-white p-3.5 sm:p-4 rounded-xl border border-slate-200 shadow-2xs space-y-1">
                  <div className="text-[11px] font-semibold text-slate-500 truncate">
                    {t.avgLatencyKpi}
                  </div>
                  <div className="text-xl sm:text-2xl font-extrabold text-blue-600 tabular-nums">
                    42 <span className="text-xs font-semibold text-slate-500">ms</span>
                  </div>
                  <div className="text-[10px] text-slate-500 font-medium truncate">
                    {isRTL ? "میانگین تاخیر پاسخ" : "Fast HTTP response"}
                  </div>
                </div>
              </div>

              {/* Sample Operational Activity Feed Rows */}
              <div className="bg-white rounded-xl border border-slate-200 shadow-2xs overflow-hidden">
                <div className="px-4 py-2.5 bg-slate-50/80 border-b border-slate-200 flex items-center justify-between text-[11px] font-bold text-slate-500 uppercase tracking-wider">
                  <span>{isRTL ? "نمونه پایش‌های فعال آژانس" : "Sample Monitored Client Assets"}</span>
                  <span>{isRTL ? "وضعیت و تاخیر" : "Status & Latency"}</span>
                </div>
                <div className="divide-y divide-slate-100 text-xs">
                  <div className="p-3 sm:px-4 flex items-center justify-between gap-3 hover:bg-slate-50/50 transition-colors">
                    <div className="flex items-center gap-2.5 min-w-0">
                      <div className="w-2 h-2 rounded-full bg-emerald-500 shrink-0" />
                      <div className="min-w-0">
                        <div className="font-bold text-slate-800 truncate">Acme Global Portal</div>
                        <div className="text-[10px] text-slate-400 font-mono truncate">https://portal.acmeglobal.com</div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <span className="text-[11px] font-semibold text-slate-600 tabular-nums">34 ms</span>
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-emerald-50 text-emerald-700 border border-emerald-200">
                        HTTP 200
                      </span>
                    </div>
                  </div>

                  <div className="p-3 sm:px-4 flex items-center justify-between gap-3 hover:bg-slate-50/50 transition-colors">
                    <div className="flex items-center gap-2.5 min-w-0">
                      <div className="w-2 h-2 rounded-full bg-emerald-500 shrink-0" />
                      <div className="min-w-0">
                        <div className="font-bold text-slate-800 truncate">Core Payments API</div>
                        <div className="text-[10px] text-slate-400 font-mono truncate">https://api.acmeglobal.com/v1/health</div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <span className="text-[11px] font-semibold text-slate-600 tabular-nums">58 ms</span>
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-emerald-50 text-emerald-700 border border-emerald-200">
                        HTTP 200
                      </span>
                    </div>
                  </div>

                  <div className="p-3 sm:px-4 flex items-center justify-between gap-3 hover:bg-slate-50/50 transition-colors">
                    <div className="flex items-center gap-2.5 min-w-0">
                      <div className="w-2 h-2 rounded-full bg-blue-500 shrink-0" />
                      <div className="min-w-0">
                        <div className="font-bold text-slate-800 truncate">Nightly DB Backup Heartbeat</div>
                        <div className="text-[10px] text-slate-400 truncate">cron (0 2 * * *) · 15m grace period</div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <span className="text-[11px] text-slate-500 tabular-nums">{isRTL ? "۲ دقیقه پیش" : "2m ago"}</span>
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-blue-50 text-blue-700 border border-blue-200">
                        Healthy
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
