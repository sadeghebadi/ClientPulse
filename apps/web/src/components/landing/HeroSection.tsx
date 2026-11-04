"use client";

import React from "react";
import Link from "next/link";
import dynamic from "next/dynamic";
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
  Zap,
  Globe,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { MagneticWrapper } from "./motion/MagneticButton";
import { TiltCard } from "./motion/TiltCard";
import { CountUp } from "./motion/CountUp";
import { SpotlightGlow } from "./motion/SpotlightGlow";

// Dynamically import 3D Three.js canvas with zero SSR overhead
const HeroNetworkCanvas = dynamic(
  () => import("./HeroNetworkCanvas").then((mod) => mod.HeroNetworkCanvas),
  {
    ssr: false,
    loading: () => (
      <div className="w-full h-[360px] sm:h-[460px] lg:h-[540px] flex items-center justify-center">
        <div className="w-48 h-48 rounded-full border border-blue-500/20 bg-blue-950/20 animate-pulse-slow" />
      </div>
    ),
  }
);

export function HeroSection() {
  const { t, isRTL } = useI18n();

  return (
    <section className="relative overflow-hidden pt-12 pb-20 sm:pt-20 sm:pb-28 lg:pt-24 lg:pb-36 bg-[#070b14] text-slate-100">
      {/* Dynamic Cursor Spotlight Tracking */}
      <SpotlightGlow color="rgba(59, 130, 246, 0.15)" size={480} />

      {/* Atmospheric Ambient Glow Orbs */}
      <div className="absolute top-10 left-1/2 -translate-x-1/2 w-[850px] h-[500px] bg-gradient-to-tr from-blue-600/15 via-violet-600/10 to-transparent blur-[140px] pointer-events-none rounded-full" />
      <div className="absolute inset-0 bg-[radial-gradient(#3b82f6_1px,transparent_1px)] [background-size:28px_28px] opacity-[0.08] pointer-events-none" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Hero Top Copy */}
        <div className="max-w-3xl mx-auto text-center space-y-6">
          {/* Target Audience Badge with Ambient Ring */}
          <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full text-xs font-bold text-blue-400 bg-blue-950/70 border border-blue-800/80 shadow-lg shadow-blue-950/50 backdrop-blur-md">
            <span className="w-2 h-2 rounded-full bg-blue-400 animate-pulse" />
            <ShieldCheck className="w-3.5 h-3.5 text-blue-400 shrink-0" />
            <span>{t.landingHeroBadge}</span>
          </div>

          {/* Primary Cinematic Headline */}
          <h1 className="text-3xl sm:text-5xl lg:text-6xl font-black text-white tracking-tight leading-[1.12] sm:leading-[1.12] drop-shadow-sm">
            {t.landingHeroHeadline}
          </h1>

          {/* Supporting Subtitle */}
          <p className="text-base sm:text-xl text-slate-300 leading-relaxed max-w-2xl mx-auto font-normal">
            {t.landingHeroSubtitle}
          </p>

          {/* Dual Call To Actions with Magnetic Micro-Interactions */}
          <div className="pt-2 sm:pt-4 flex flex-col sm:flex-row items-center justify-center gap-3 sm:gap-4">
            <MagneticWrapper strength={0.3}>
              <Link
                href="/signup"
                className="w-full sm:w-auto inline-flex items-center justify-center gap-2.5 px-8 py-4 text-sm font-bold text-white bg-blue-600 hover:bg-blue-500 rounded-xl shadow-xl shadow-blue-600/30 hover:shadow-blue-500/40 transition-all group"
              >
                <UserPlus className="w-4 h-4" />
                <span>{t.landingHeroCtaPrimary}</span>
                {isRTL ? (
                  <ArrowLeft className="w-4 h-4 group-hover:-translate-x-1 transition-transform" />
                ) : (
                  <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
                )}
              </Link>
            </MagneticWrapper>

            <MagneticWrapper strength={0.2}>
              <a
                href="#features"
                className="w-full sm:w-auto inline-flex items-center justify-center gap-2 px-7 py-4 text-sm font-semibold text-slate-300 bg-slate-900/80 hover:bg-slate-800 hover:text-white border border-slate-700/80 rounded-xl shadow-lg shadow-black/20 hover:border-slate-600 transition-all backdrop-blur-md"
              >
                <Compass className="w-4 h-4 text-slate-400" />
                <span>{t.landingHeroCtaSecondary}</span>
              </a>
            </MagneticWrapper>
          </div>

          {/* Operational Engine Trust Notice */}
          <div className="pt-2 flex flex-wrap items-center justify-center gap-2 text-xs text-slate-400">
            <span className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg bg-slate-900/80 border border-slate-800 shadow-sm backdrop-blur-sm">
              <Server className="w-3.5 h-3.5 text-blue-400" />
              <span>{t.landingHeroEngineBadge}</span>
            </span>
          </div>
        </div>

        {/* ======================================================== */}
        {/* Interactive 3D Monitoring Network Mesh Canvas */}
        {/* ======================================================== */}
        <div className="mt-8 sm:mt-12 max-w-4xl mx-auto relative">
          <div className="flex items-center justify-center gap-2 mb-2 text-[11px] font-mono text-slate-400">
            <Globe className="w-3.5 h-3.5 text-blue-400" />
            <span>{t.landingHero3dLabel || "Live 3D Network Mesh & Global Edge Checkers"}</span>
          </div>
          <HeroNetworkCanvas />
        </div>

        {/* ======================================================== */}
        {/* Authentic High-Fidelity Executive Dashboard Preview */}
        {/* ======================================================== */}
        <div className="mt-8 sm:mt-12 max-w-5xl mx-auto">
          <TiltCard maxTilt={3} sheen={true}>
            <div className="rounded-2xl sm:rounded-3xl border border-slate-800 bg-[#0b1120] shadow-2xl overflow-hidden ring-1 ring-slate-700/50">
              {/* Realistic Browser Chrome Header */}
              <div className="bg-slate-950 px-4 py-3 sm:px-6 flex items-center justify-between gap-4 border-b border-slate-800/80">
                <div className="flex items-center gap-2">
                  <span className="w-3 h-3 rounded-full bg-rose-500/80 inline-block" />
                  <span className="w-3 h-3 rounded-full bg-amber-500/80 inline-block" />
                  <span className="w-3 h-3 rounded-full bg-emerald-500/80 inline-block" />
                </div>

                {/* Anonymized Safe URL Pill */}
                <div className="flex items-center gap-2 px-3.5 py-1 rounded-lg bg-slate-900 text-slate-300 text-[11px] font-mono border border-slate-800 max-w-xs sm:max-w-md w-full justify-center truncate">
                  <Lock className="w-3 h-3 text-emerald-400 shrink-0" />
                  <span className="truncate">https://mindu.ir/dashboard</span>
                </div>

                <div className="flex items-center gap-1.5 text-xs text-slate-400 font-semibold shrink-0">
                  <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
                  <span className="text-[11px] font-medium text-slate-300">
                    {isRTL ? "پیش‌نمایش محصول · داده‌های نمونه" : "Product Preview · Sample Data"}
                  </span>
                </div>
              </div>

              {/* Dashboard Inner Canvas Preview */}
              <div className="p-4 sm:p-6 lg:p-8 bg-[#090e1a] space-y-5">
                {/* System Health Bar Preview */}
                <div className="bg-slate-900/90 rounded-xl border border-emerald-500/30 shadow-lg p-3.5 sm:p-4 flex flex-col md:flex-row md:items-center justify-between gap-3">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-lg bg-emerald-500/20 text-emerald-400 flex items-center justify-center shrink-0">
                      <CheckCircle2 className="w-5 h-5" />
                    </div>
                    <div>
                      <div className="text-xs sm:text-sm font-bold text-white">
                        {isRTL
                          ? "تمام سیستم‌ها پایدارند · ۱۰۰٪ اهداف پایش در وضعیت سالم"
                          : "All Systems Operational · 100% of monitored targets healthy"}
                      </div>
                      <div className="text-[11px] text-slate-400">
                        {isRTL
                          ? "۲۴ مانیتور و هارت‌بیت فعال بدون هیچ رخداد باز"
                          : "24 active monitors & background heartbeats with zero open incidents"}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center gap-2 text-xs font-semibold shrink-0">
                    <span className="px-2.5 py-1 rounded-md bg-emerald-500/15 text-emerald-400 border border-emerald-500/30">
                      24 / 24 Up
                    </span>
                    <span className="px-2.5 py-1 rounded-md bg-slate-800 text-slate-400">
                      0 Outages
                    </span>
                  </div>
                </div>

                {/* 4 Primary KPI Cards Preview with Dynamic Animated CountUp */}
                <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
                  <div className="bg-slate-900/80 p-3.5 sm:p-4 rounded-xl border border-slate-800 shadow-sm space-y-1">
                    <div className="text-[11px] font-semibold text-slate-400 truncate">
                      {t.monitoredTargetsKpi}
                    </div>
                    <div className="text-xl sm:text-2xl font-extrabold text-white tabular-nums">
                      <CountUp end={24} duration={1400} />
                    </div>
                    <div className="text-[10px] text-emerald-400 font-medium flex items-center gap-1">
                      <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
                      <span>{isRTL ? "تمام دارایی‌ها فعال" : "All targets verified"}</span>
                    </div>
                  </div>

                  <div className="bg-slate-900/80 p-3.5 sm:p-4 rounded-xl border border-slate-800 shadow-sm space-y-1">
                    <div className="text-[11px] font-semibold text-slate-400 truncate">
                      {t.targetsDownKpi}
                    </div>
                    <div className="text-xl sm:text-2xl font-extrabold text-emerald-400 tabular-nums">
                      0
                    </div>
                    <div className="text-[10px] text-slate-400 font-medium">
                      {isRTL ? "بدون قطعی وب‌سایت" : "Zero confirmed downtime"}
                    </div>
                  </div>

                  <div className="bg-slate-900/80 p-3.5 sm:p-4 rounded-xl border border-slate-800 shadow-sm space-y-1">
                    <div className="text-[11px] font-semibold text-slate-400 truncate">
                      {t.openIncidentsKpi}
                    </div>
                    <div className="text-xl sm:text-2xl font-extrabold text-white tabular-nums">
                      0
                    </div>
                    <div className="text-[10px] text-emerald-400 font-medium">
                      {isRTL ? "تمام رخدادها حل‌شده" : "Clean resolution history"}
                    </div>
                  </div>

                  <div className="bg-slate-900/80 p-3.5 sm:p-4 rounded-xl border border-slate-800 shadow-sm space-y-1">
                    <div className="text-[11px] font-semibold text-slate-400 truncate">
                      {t.avgLatencyKpi}
                    </div>
                    <div className="text-xl sm:text-2xl font-extrabold text-blue-400 tabular-nums flex items-baseline gap-1">
                      <CountUp end={42} duration={1200} />
                      <span className="text-xs font-semibold text-slate-400">ms</span>
                    </div>
                    <div className="text-[10px] text-slate-400 font-medium truncate">
                      {isRTL ? "میانگین تاخیر پاسخ" : "Fast HTTP response"}
                    </div>
                  </div>
                </div>

                {/* Sample Operational Activity Feed Rows */}
                <div className="bg-slate-900/80 rounded-xl border border-slate-800 shadow-sm overflow-hidden">
                  <div className="px-4 py-2.5 bg-slate-950 border-b border-slate-800 flex items-center justify-between text-[11px] font-bold text-slate-400 uppercase tracking-wider">
                    <span>{isRTL ? "نمونه پایش‌های فعال" : "Sample Monitored Targets"}</span>
                    <span>{isRTL ? "وضعیت و تاخیر" : "Status & Latency"}</span>
                  </div>
                  <div className="divide-y divide-slate-800/80 text-xs">
                    <div className="p-3 sm:px-4 flex items-center justify-between gap-3 hover:bg-slate-800/40 transition-colors">
                      <div className="flex items-center gap-2.5 min-w-0">
                        <div className="w-2 h-2 rounded-full bg-emerald-400 shrink-0 shadow-[0_0_6px_#34d399]" />
                        <div className="min-w-0">
                          <div className="font-bold text-white truncate">Acme Global Portal</div>
                          <div className="text-[10px] text-slate-400 font-mono truncate">https://portal.acmeglobal.com</div>
                        </div>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <span className="text-[11px] font-semibold text-slate-300 tabular-nums">34 ms</span>
                        <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-emerald-500/15 text-emerald-400 border border-emerald-500/30">
                          HTTP 200
                        </span>
                      </div>
                    </div>

                    <div className="p-3 sm:px-4 flex items-center justify-between gap-3 hover:bg-slate-800/40 transition-colors">
                      <div className="flex items-center gap-2.5 min-w-0">
                        <div className="w-2 h-2 rounded-full bg-emerald-400 shrink-0 shadow-[0_0_6px_#34d399]" />
                        <div className="min-w-0">
                          <div className="font-bold text-white truncate">Core Payments API</div>
                          <div className="text-[10px] text-slate-400 font-mono truncate">https://api.acmeglobal.com/v1/health</div>
                        </div>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <span className="text-[11px] font-semibold text-slate-300 tabular-nums">58 ms</span>
                        <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-emerald-500/15 text-emerald-400 border border-emerald-500/30">
                          HTTP 200
                        </span>
                      </div>
                    </div>

                    <div className="p-3 sm:px-4 flex items-center justify-between gap-3 hover:bg-slate-800/40 transition-colors">
                      <div className="flex items-center gap-2.5 min-w-0">
                        <div className="w-2 h-2 rounded-full bg-blue-400 shrink-0 shadow-[0_0_6px_#60a5fa]" />
                        <div className="min-w-0">
                          <div className="font-bold text-white truncate">Nightly DB Backup Heartbeat</div>
                          <div className="text-[10px] text-slate-400 truncate">cron (0 2 * * *) · 15m grace period</div>
                        </div>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <span className="text-[11px] text-slate-400 tabular-nums">{isRTL ? "۲ دقیقه پیش" : "2m ago"}</span>
                        <span className="px-2 py-0.5 rounded-full text-[10px] font-bold bg-blue-500/15 text-blue-300 border border-blue-500/30">
                          Healthy
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </TiltCard>
        </div>
      </div>
    </section>
  );
}
