"use client";

import React from "react";
import Link from "next/link";
import { UserPlus, LogIn, ArrowRight, ArrowLeft, ShieldCheck, Sparkles, CheckCircle2 } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { MagneticWrapper } from "./motion/MagneticButton";
import { SpotlightGlow } from "./motion/SpotlightGlow";

export function FinalCtaSection() {
  const { t, isRTL } = useI18n();

  return (
    <section className="py-24 sm:py-36 bg-gradient-to-b from-[#060913] via-[#0b1224] to-[#04060c] text-white relative overflow-hidden border-t border-slate-800">
      {/* Subtle Spotlight Glow */}
      <SpotlightGlow color="rgba(99, 102, 241, 0.18)" size={480} />

      {/* Atmospheric Radial Aura */}
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[850px] h-[450px] bg-gradient-to-r from-blue-600/20 via-cyan-500/15 to-violet-600/20 blur-[150px] pointer-events-none rounded-full" />
      <div className="absolute inset-0 bg-[radial-gradient(#38bdf8_1px,transparent_1px)] [background-size:32px_32px] opacity-[0.06] pointer-events-none" />

      <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10 text-center space-y-6 sm:space-y-8">
        <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full text-xs font-semibold bg-cyan-500/15 text-cyan-300 border border-cyan-400/30 backdrop-blur-md shadow-lg shadow-cyan-950/40">
          <Sparkles className="w-4 h-4 text-cyan-400" />
          <span>{isRTL ? "دیده‌بانی هوشمند و آرامش خاطر کامل وب‌سایت‌ها" : "Intelligent Observability & Peace of Mind"}</span>
        </div>

        <h2 className="text-3xl sm:text-5xl lg:text-6xl font-black tracking-tight max-w-3xl mx-auto leading-tight text-white drop-shadow-md">
          {t.landingFinalCtaTitle}
        </h2>

        <p className="text-base sm:text-xl text-slate-300 max-w-2xl mx-auto leading-relaxed font-normal">
          {t.landingFinalCtaSubtitle}
        </p>

        {/* Dual CTAs with Magnetic Physics */}
        <div className="pt-4 flex flex-col sm:flex-row items-center justify-center gap-3 sm:gap-4">
          <MagneticWrapper strength={0.3}>
            <Link
              href="/signup"
              className="w-full sm:w-auto inline-flex items-center justify-center gap-2.5 px-8 py-4 text-sm font-bold text-white bg-blue-600 hover:bg-blue-500 rounded-xl shadow-xl shadow-blue-600/30 hover:shadow-blue-500/40 transition-all group ring-1 ring-blue-400/40"
            >
              <UserPlus className="w-4 h-4" />
              <span>{t.landingFinalCtaBtn}</span>
              {isRTL ? (
                <ArrowLeft className="w-4 h-4 group-hover:-translate-x-1 transition-transform" />
              ) : (
                <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
              )}
            </Link>
          </MagneticWrapper>

          <MagneticWrapper strength={0.2}>
            <Link
              href="/login"
              className="w-full sm:w-auto inline-flex items-center justify-center gap-2 px-7 py-4 text-sm font-semibold text-slate-300 hover:text-white bg-slate-900/80 hover:bg-slate-800 border border-slate-700/80 rounded-xl transition-all shadow-md backdrop-blur-md"
            >
              <LogIn className="w-4 h-4 text-slate-400" />
              <span>{t.landingNavSignIn}</span>
            </Link>
          </MagneticWrapper>
        </div>

        {/* Trust Badges */}
        <div className="pt-6 flex flex-wrap items-center justify-center gap-6 text-xs text-slate-400 font-mono">
          <span className="flex items-center gap-1.5">
            <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
            <span>Zero Setup Overhead</span>
          </span>
          <span className="flex items-center gap-1.5">
            <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
            <span>SSRF Protection Included</span>
          </span>
          <span className="flex items-center gap-1.5">
            <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
            <span>No Credit Card Required</span>
          </span>
        </div>
      </div>
    </section>
  );
}
