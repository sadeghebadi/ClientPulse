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
  ChevronDown,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { MagneticWrapper } from "./motion/MagneticButton";
import { SpotlightGlow } from "./motion/SpotlightGlow";

// Dynamically import the upgraded 3D Spatial Observability Mesh
const HeroNetworkCanvas = dynamic(
  () => import("./HeroNetworkCanvas").then((mod) => mod.HeroNetworkCanvas),
  {
    ssr: false,
    loading: () => (
      <div className="w-full h-[520px] sm:h-[620px] lg:h-[720px] flex items-center justify-center">
        <div className="w-64 h-64 rounded-full border border-blue-500/20 bg-blue-950/20 animate-pulse-slow" />
      </div>
    ),
  }
);

export function HeroSection() {
  const { t, isRTL } = useI18n();

  const handleScrollToStory = (e: React.MouseEvent<HTMLAnchorElement>) => {
    e.preventDefault();
    const storyTarget = document.querySelector("#story") || document.querySelector("#lifecycle");
    if (storyTarget) {
      storyTarget.scrollIntoView({ behavior: "smooth" });
    }
  };

  return (
    <section className="relative overflow-hidden pt-8 pb-16 sm:pt-16 sm:pb-24 lg:pt-20 lg:pb-32 bg-[#050811] text-slate-100">
      {/* Dynamic Cursor Spotlight Tracking */}
      <SpotlightGlow color="rgba(59, 130, 246, 0.18)" size={520} />

      {/* Atmospheric Spatial Aura & Cosmic Glows */}
      <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[1100px] h-[650px] bg-gradient-to-tr from-blue-600/15 via-cyan-500/10 to-violet-600/10 blur-[150px] pointer-events-none rounded-full" />
      <div className="absolute inset-0 bg-[radial-gradient(#38bdf8_1px,transparent_1px)] [background-size:32px_32px] opacity-[0.06] pointer-events-none" />

      {/* Ambient Horizontal Edge Beams */}
      <div className="absolute top-1/4 -left-48 w-96 h-96 bg-blue-600/10 rounded-full blur-[120px] pointer-events-none" />
      <div className="absolute top-1/3 -right-48 w-96 h-96 bg-cyan-500/10 rounded-full blur-[120px] pointer-events-none" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Hero Top Copy */}
        <div className="max-w-4xl mx-auto text-center space-y-6 sm:space-y-8">
          {/* Status Badge with Live Pulse */}
          <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full text-xs font-bold text-cyan-300 bg-cyan-950/60 border border-cyan-800/80 shadow-lg shadow-cyan-950/50 backdrop-blur-md">
            <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse shadow-[0_0_8px_#34d399]" />
            <ShieldCheck className="w-3.5 h-3.5 text-cyan-400 shrink-0" />
            <span>{t.landingHeroBadge}</span>
          </div>

          {/* Primary Cinematic Display Headline */}
          <h1 className="text-4xl sm:text-6xl lg:text-7xl font-black text-white tracking-tight leading-[1.08] drop-shadow-md">
            <span className="block text-white">
              {isRTL ? "ما وب‌سایت‌های شما را می‌پاییم." : "We Watch Your Websites."}
            </span>
            <span className="block bg-gradient-to-r from-blue-400 via-cyan-300 to-indigo-300 bg-clip-text text-transparent mt-1">
              {isRTL
                ? "شما بر آنچه مهم است تمرکز کنید — همواره یک گام جلوتر از اختلالات وب باشید."
                : "You Focus on What Matters."}
            </span>
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
                className="w-full sm:w-auto inline-flex items-center justify-center gap-2.5 px-8 py-4 text-sm font-bold text-white bg-blue-600 hover:bg-blue-500 rounded-xl shadow-xl shadow-blue-600/30 hover:shadow-blue-500/40 transition-all group ring-1 ring-blue-400/40"
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
                href="#story"
                onClick={handleScrollToStory}
                className="w-full sm:w-auto inline-flex items-center justify-center gap-2 px-7 py-4 text-sm font-semibold text-slate-300 bg-slate-900/80 hover:bg-slate-800 hover:text-white border border-slate-700/80 rounded-xl shadow-lg shadow-black/20 hover:border-slate-600 transition-all backdrop-blur-md"
              >
                <Compass className="w-4 h-4 text-cyan-400" />
                <span>{t.landingHeroCtaSecondary}</span>
                <ChevronDown className="w-4 h-4 text-slate-400 animate-bounce" />
              </a>
            </MagneticWrapper>
          </div>

          {/* Real-Time Edge Latency Ticker Bar */}
          <div className="pt-2 flex flex-wrap items-center justify-center gap-2 sm:gap-3 text-[11px] font-mono text-slate-400">
            <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg bg-slate-900/80 border border-slate-800 shadow-sm backdrop-blur-sm">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
              <span className="ltr-text">US-East · 24ms</span>
            </span>
            <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg bg-slate-900/80 border border-slate-800 shadow-sm backdrop-blur-sm">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
              <span className="ltr-text">EU-Central · 28ms</span>
            </span>
            <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg bg-slate-900/80 border border-slate-800 shadow-sm backdrop-blur-sm">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
              <span className="ltr-text">AP-East · 46ms</span>
            </span>
            <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg bg-slate-900/80 border border-slate-800 shadow-sm backdrop-blur-sm">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
              <span className="ltr-text">ME-Central · 14ms</span>
            </span>
            <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg bg-blue-950/60 border border-blue-800/80 text-blue-300 font-semibold shadow-sm backdrop-blur-sm">
              <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
              <span>{isRTL ? "۱۰۰٪ سلامت سیستم‌ها" : "100% Operational"}</span>
            </span>
          </div>
        </div>

        {/* ======================================================== */}
        {/* Full Interactive 3D Spatial Observability Mesh Canvas   */}
        {/* ======================================================== */}
        <div className="mt-8 sm:mt-12 w-full relative rounded-3xl bg-gradient-to-b from-slate-900/40 via-slate-950/60 to-transparent border border-slate-800/80 shadow-2xl overflow-hidden ring-1 ring-white/5">
          <div className="absolute top-4 left-6 z-20 flex items-center gap-2 text-xs font-mono text-slate-400">
            <Globe className="w-3.5 h-3.5 text-cyan-400" />
            <span className="font-semibold text-slate-300">
              {t.landingHero3dLabel || "Global Edge Mesh & Node Telemetry"}
            </span>
          </div>

          <HeroNetworkCanvas />

          {/* Engine Architecture Trust Footer */}
          <div className="px-6 py-3.5 bg-slate-950/90 border-t border-slate-800/80 flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-slate-400">
            <div className="flex items-center gap-2">
              <Server className="w-3.5 h-3.5 text-blue-400 shrink-0" />
              <span>{t.landingHeroEngineBadge}</span>
            </div>
            <div className="flex items-center gap-2 text-[11px] font-mono text-slate-500">
              <span className="w-2 h-2 rounded-full bg-emerald-400 animate-ping inline-block" />
              <span>PostgreSQL Queue Worker Active</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
