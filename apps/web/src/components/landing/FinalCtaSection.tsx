"use client";

import React from "react";
import Link from "next/link";
import { UserPlus, LogIn, ArrowRight, ArrowLeft, ShieldCheck } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { MagneticWrapper } from "./motion/MagneticButton";
import { SpotlightGlow } from "./motion/SpotlightGlow";

export function FinalCtaSection() {
  const { t, isRTL } = useI18n();

  return (
    <section className="py-20 sm:py-32 bg-gradient-to-b from-[#090e1a] via-[#0d1527] to-[#070b14] text-white relative overflow-hidden border-t border-slate-800">
      {/* Subtle Spotlight Glow */}
      <SpotlightGlow color="rgba(99, 102, 241, 0.15)" size={450} />

      {/* Atmospheric Radial Aura */}
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[700px] h-[350px] bg-gradient-to-r from-blue-600/15 via-violet-600/15 to-transparent blur-[120px] pointer-events-none rounded-full" />
      <div className="absolute inset-0 bg-[radial-gradient(#3b82f6_1px,transparent_1px)] [background-size:28px_28px] opacity-[0.08] pointer-events-none" />

      <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10 text-center space-y-6 sm:space-y-8">
        <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full text-xs font-semibold bg-blue-500/15 text-blue-300 border border-blue-400/30 backdrop-blur-md">
          <ShieldCheck className="w-4 h-4 text-blue-400" />
          <span>{isRTL ? "دیده‌بانی هوشمند و آرامش خاطر کامل وب‌سایت‌ها" : "Enterprise Observability & Peace of Mind"}</span>
        </div>

        <h2 className="text-3xl sm:text-5xl font-black tracking-tight max-w-3xl mx-auto leading-tight text-white drop-shadow-sm">
          {t.landingFinalCtaTitle}
        </h2>

        <p className="text-base sm:text-lg text-slate-300 max-w-2xl mx-auto leading-relaxed">
          {t.landingFinalCtaSubtitle}
        </p>

        <div className="pt-4 flex flex-col sm:flex-row items-center justify-center gap-3 sm:gap-4">
          <MagneticWrapper strength={0.3}>
            <Link
              href="/signup"
              className="w-full sm:w-auto inline-flex items-center justify-center gap-2.5 px-8 py-4 text-sm font-bold text-white bg-blue-600 hover:bg-blue-500 rounded-xl shadow-xl shadow-blue-600/30 hover:shadow-blue-500/40 transition-all group"
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
      </div>
    </section>
  );
}
