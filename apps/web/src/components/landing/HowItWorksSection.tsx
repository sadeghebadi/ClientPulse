"use client";

import React from "react";
import { Users, Sliders, BellRing, Award, ArrowRight, ArrowLeft } from "lucide-react";
import { useI18n } from "@/lib/i18n";

export function HowItWorksSection() {
  const { t, isRTL } = useI18n();

  const steps = [
    {
      num: "01",
      icon: Users,
      title: t.landingStep1Title,
      description: t.landingStep1Desc,
      tag: isRTL ? "مرحله ۱" : "Step 1",
    },
    {
      num: "02",
      icon: Sliders,
      title: t.landingStep2Title,
      description: t.landingStep2Desc,
      tag: isRTL ? "مرحله ۲" : "Step 2",
    },
    {
      num: "03",
      icon: BellRing,
      title: t.landingStep3Title,
      description: t.landingStep3Desc,
      tag: isRTL ? "مرحله ۳" : "Step 3",
    },
    {
      num: "04",
      icon: Award,
      title: t.landingStep4Title,
      description: t.landingStep4Desc,
      tag: isRTL ? "مرحله ۴" : "Step 4",
    },
  ];

  return (
    <section id="how-it-works" className="py-16 sm:py-24 bg-slate-50 border-b border-slate-200/80 scroll-mt-20">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Section Header */}
        <div className="max-w-3xl mx-auto text-center space-y-3 sm:space-y-4 mb-12 sm:mb-16">
          <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-bold text-slate-700 bg-white border border-slate-200 shadow-2xs">
            <span>{isRTL ? "گردش کار عملیاتی" : "Operational Workflow"}</span>
          </div>

          <h2 className="text-2xl sm:text-4xl font-extrabold text-slate-900 tracking-tight">
            {t.landingHowTitle}
          </h2>

          <p className="text-sm sm:text-base text-slate-600 leading-relaxed font-normal">
            {t.landingHowSubtitle}
          </p>
        </div>

        {/* 4 Steps Horizontal / Responsive Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          {steps.map((step, idx) => {
            const Icon = step.icon;
            return (
              <div
                key={step.num}
                className="relative bg-white rounded-2xl border border-slate-200 p-6 shadow-2xs hover:shadow-md hover:border-blue-200 transition-all flex flex-col justify-between"
              >
                <div>
                  {/* Top Bar with Number and Tag */}
                  <div className="flex items-center justify-between mb-5">
                    <span className="text-2xl font-black text-blue-600 tabular-nums">
                      {step.num}
                    </span>
                    <span className="text-[10px] font-bold uppercase tracking-wider text-slate-400 bg-slate-100 px-2 py-0.5 rounded">
                      {step.tag}
                    </span>
                  </div>

                  <div className="w-10 h-10 rounded-xl bg-blue-50 text-blue-600 flex items-center justify-center mb-4">
                    <Icon className="w-5 h-5" />
                  </div>

                  <h3 className="text-base font-bold text-slate-900 tracking-tight mb-2">
                    {step.title}
                  </h3>

                  <p className="text-xs text-slate-600 leading-relaxed font-normal">
                    {step.description}
                  </p>
                </div>

                {/* Progress Indicator Dots */}
                <div className="pt-4 mt-6 border-t border-slate-100 flex items-center justify-between text-[11px] text-slate-400">
                  <span>{isRTL ? `گام ${idx + 1} از ۴` : `Phase ${idx + 1} of 4`}</span>
                  <div className="flex items-center gap-1">
                    {[0, 1, 2, 3].map((dotIdx) => (
                      <span
                        key={dotIdx}
                        className={`w-1.5 h-1.5 rounded-full ${
                          dotIdx === idx ? "bg-blue-600 w-3" : "bg-slate-200"
                        } transition-all`}
                      />
                    ))}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}

