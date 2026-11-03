"use client";

import React from "react";
import Link from "next/link";
import { UserPlus, LogIn, ArrowRight, ArrowLeft, ShieldCheck } from "lucide-react";
import { useI18n } from "@/lib/i18n";

export function FinalCtaSection() {
  const { t, isRTL } = useI18n();

  return (
    <section className="py-16 sm:py-24 bg-gradient-to-br from-slate-900 via-indigo-950 to-slate-900 text-white relative overflow-hidden">
      {/* Subtle Pattern */}
      <div className="absolute inset-0 bg-[radial-gradient(#3b82f6_1px,transparent_1px)] [background-size:24px_24px] opacity-10 pointer-events-none" />

      <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10 text-center space-y-6 sm:space-y-8">
        <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full text-xs font-semibold bg-blue-500/20 text-blue-300 border border-blue-400/30">
          <ShieldCheck className="w-4 h-4 text-blue-400" />
          <span>{isRTL ? "زیرساخت مطمئن پایش و مراقبت از کلاینت‌ها" : "Enterprise-Grade Reliability for Web Agencies"}</span>
        </div>

        <h2 className="text-3xl sm:text-5xl font-black tracking-tight max-w-3xl mx-auto leading-tight">
          {t.landingFinalCtaTitle}
        </h2>

        <p className="text-base sm:text-lg text-slate-300 max-w-2xl mx-auto leading-relaxed">
          {t.landingFinalCtaSubtitle}
        </p>

        <div className="pt-2 flex flex-col sm:flex-row items-center justify-center gap-3 sm:gap-4">
          <Link
            href="/signup"
            className="w-full sm:w-auto inline-flex items-center justify-center gap-2 px-8 py-3.5 text-sm font-bold text-white bg-blue-600 hover:bg-blue-500 rounded-xl shadow-lg hover:shadow-xl transition-all group"
          >
            <UserPlus className="w-4 h-4" />
            <span>{t.landingFinalCtaBtn}</span>
            {isRTL ? (
              <ArrowLeft className="w-4 h-4 group-hover:-translate-x-1 transition-transform" />
            ) : (
              <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
            )}
          </Link>

          <Link
            href="/login"
            className="w-full sm:w-auto inline-flex items-center justify-center gap-2 px-6 py-3.5 text-sm font-semibold text-slate-300 hover:text-white bg-slate-800/80 hover:bg-slate-800 border border-slate-700 rounded-xl transition-colors"
          >
            <LogIn className="w-4 h-4" />
            <span>{t.landingNavSignIn}</span>
          </Link>
        </div>
      </div>
    </section>
  );
}

