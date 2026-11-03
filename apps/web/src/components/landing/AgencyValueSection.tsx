"use client";

import React from "react";
import { FolderKanban, Eye, Cpu, CheckSquare, ArrowUpRight } from "lucide-react";
import { useI18n } from "@/lib/i18n";

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
    <section id="value" className="py-16 sm:py-24 bg-white border-b border-slate-200/80 scroll-mt-20">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Section Header */}
        <div className="max-w-3xl mx-auto text-center space-y-3 sm:space-y-4 mb-12 sm:mb-16">
          <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-bold text-indigo-700 bg-indigo-50 border border-indigo-200/70">
            <span>{isRTL ? "مزایای اختصاصی برای وب‌آژانس‌ها" : "Agency Value Proposition"}</span>
          </div>

          <h2 className="text-2xl sm:text-4xl font-extrabold text-slate-900 tracking-tight">
            {t.landingValueTitle}
          </h2>

          <p className="text-sm sm:text-base text-slate-600 leading-relaxed font-normal">
            {t.landingValueSubtitle}
          </p>
        </div>

        {/* 2x2 Clean Value Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 sm:gap-8">
          {values.map((v) => {
            const Icon = v.icon;
            return (
              <div
                key={v.id}
                className="bg-slate-50 rounded-2xl border border-slate-200/80 p-6 sm:p-8 hover:bg-white hover:border-blue-300 hover:shadow-md transition-all space-y-4"
              >
                <div className="flex items-center justify-between">
                  <div className="w-11 h-11 rounded-xl bg-white border border-slate-200 text-blue-600 flex items-center justify-center shadow-2xs">
                    <Icon className="w-5 h-5" />
                  </div>
                  <span className="text-[11px] font-bold text-slate-500 bg-white px-2.5 py-1 rounded-md border border-slate-200">
                    {v.tag}
                  </span>
                </div>

                <h3 className="text-lg font-bold text-slate-900 tracking-tight">
                  {v.title}
                </h3>

                <p className="text-xs sm:text-sm text-slate-600 leading-relaxed font-normal">
                  {v.description}
                </p>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}

