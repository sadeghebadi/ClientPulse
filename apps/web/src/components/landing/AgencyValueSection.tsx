"use client";

import React from "react";
import { FolderKanban, Eye, Cpu, CheckSquare } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { TiltCard } from "./motion/TiltCard";

export function AgencyValueSection() {
  const { t, isRTL } = useI18n();

  const values = [
    {
      id: "portfolio",
      icon: FolderKanban,
      title: t.landingVal1Title,
      description: t.landingVal1Desc,
      tag: isRTL ? "مدیریت پرتفوی" : "Portfolio Care",
    },
    {
      id: "proactive",
      icon: Eye,
      title: t.landingVal2Title,
      description: t.landingVal2Desc,
      tag: isRTL ? "کاهش تنش کاری" : "Client Trust",
    },
    {
      id: "clarity",
      icon: Cpu,
      title: t.landingVal3Title,
      description: t.landingVal3Desc,
      tag: isRTL ? "دقت عیب‌یابی" : "Root Cause",
    },
    {
      id: "retainer",
      icon: CheckSquare,
      title: t.landingVal4Title,
      description: t.landingVal4Desc,
      tag: isRTL ? "ارزش قرارداد" : "Retainer ROI",
    },
  ];

  return (
    <section id="value" className="py-20 sm:py-28 bg-[#090d18] text-slate-100 border-b border-slate-800 scroll-mt-20 relative overflow-hidden">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Section Header */}
        <div className="max-w-3xl mx-auto text-center space-y-4 mb-12 sm:mb-16">
          <div className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-full text-xs font-semibold text-indigo-400 bg-indigo-950/60 border border-indigo-800/60 shadow-lg shadow-indigo-950/30">
            <span>{isRTL ? "مزایای اختصاصی برای وب‌آژانس‌ها" : "Agency Value Proposition"}</span>
          </div>

          <h2 className="text-3xl sm:text-5xl font-black text-white tracking-tight">
            {t.landingValueTitle}
          </h2>

          <p className="text-base sm:text-lg text-slate-400 leading-relaxed font-normal">
            {t.landingValueSubtitle}
          </p>
        </div>

        {/* 2x2 Clean Value Grid with 3D Tilt */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 sm:gap-8">
          {values.map((v) => {
            const Icon = v.icon;
            return (
              <TiltCard key={v.id} maxTilt={4} sheen={true} className="h-full">
                <div className="h-full bg-[#0c1222] rounded-2xl border border-slate-800 p-6 sm:p-8 hover:border-slate-700 transition-all space-y-4 shadow-xl">
                  <div className="flex items-center justify-between">
                    <div className="w-11 h-11 rounded-xl bg-blue-950/80 border border-blue-800/60 text-blue-400 flex items-center justify-center shadow-md">
                      <Icon className="w-5 h-5" />
                    </div>
                    <span className="text-[11px] font-bold text-slate-400 bg-slate-900 px-3 py-1 rounded-lg border border-slate-800">
                      {v.tag}
                    </span>
                  </div>

                  <h3 className="text-lg font-bold text-white tracking-tight">
                    {v.title}
                  </h3>

                  <p className="text-xs sm:text-sm text-slate-400 leading-relaxed font-normal">
                    {v.description}
                  </p>
                </div>
              </TiltCard>
            );
          })}
        </div>
      </div>
    </section>
  );
}
