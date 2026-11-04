"use client";

import React from "react";
import { Users, Sliders, BellRing, Award } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { TiltCard } from "./motion/TiltCard";

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
    <section id="how-it-works" className="py-20 sm:py-28 bg-[#070b14] text-slate-100 border-b border-slate-800 scroll-mt-20 relative overflow-hidden">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Section Header */}
        <div className="max-w-3xl mx-auto text-center space-y-4 mb-12 sm:mb-16">
          <div className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-full text-xs font-semibold text-blue-400 bg-blue-950/60 border border-blue-800/60 shadow-lg shadow-blue-950/30">
            <span>{isRTL ? "گردش کار عملیاتی" : "Operational Workflow"}</span>
          </div>

          <h2 className="text-3xl sm:text-5xl font-black text-white tracking-tight">
            {t.landingHowTitle}
          </h2>

          <p className="text-base sm:text-lg text-slate-400 leading-relaxed font-normal">
            {t.landingHowSubtitle}
          </p>
        </div>

        {/* 4 Steps Horizontal / Responsive Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          {steps.map((step, idx) => {
            const Icon = step.icon;
            return (
              <TiltCard key={step.num} maxTilt={4} sheen={true} className="h-full">
                <div className="h-full relative bg-[#0c1222] rounded-2xl border border-slate-800 p-6 shadow-xl hover:border-slate-700 transition-all flex flex-col justify-between">
                  <div>
                    {/* Top Bar with Number and Tag */}
                    <div className="flex items-center justify-between mb-5">
                      <span className="text-3xl font-black text-blue-500 tabular-nums">
                        {step.num}
                      </span>
                      <span className="text-[10px] font-bold uppercase tracking-wider text-slate-400 bg-slate-800/80 px-2 py-0.5 rounded border border-slate-700">
                        {step.tag}
                      </span>
                    </div>

                    <div className="w-10 h-10 rounded-xl bg-blue-950/80 text-blue-400 flex items-center justify-center mb-4 border border-blue-800/60">
                      <Icon className="w-5 h-5" />
                    </div>

                    <h3 className="text-base font-bold text-white tracking-tight mb-2">
                      {step.title}
                    </h3>

                    <p className="text-xs text-slate-400 leading-relaxed font-normal">
                      {step.description}
                    </p>
                  </div>

                  {/* Progress Indicator Dots */}
                  <div className="pt-4 mt-6 border-t border-slate-800 flex items-center justify-between text-[11px] text-slate-500">
                    <span>{isRTL ? `گام ${idx + 1} از ۴` : `Phase ${idx + 1} of 4`}</span>
                    <div className="flex items-center gap-1">
                      {[0, 1, 2, 3].map((dotIdx) => (
                        <span
                          key={dotIdx}
                          className={`w-1.5 h-1.5 rounded-full transition-all ${
                            dotIdx === idx ? "bg-blue-500 w-3" : "bg-slate-700"
                          }`}
                        />
                      ))}
                    </div>
                  </div>
                </div>
              </TiltCard>
            );
          })}
        </div>
      </div>
    </section>
  );
}
