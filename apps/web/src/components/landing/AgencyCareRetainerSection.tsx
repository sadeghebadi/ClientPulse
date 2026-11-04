"use client";

import React from "react";
import {
  Users,
  FolderKanban,
  Wrench,
  ShieldCheck,
  CheckCircle2,
  FileText,
  Lock,
  Layers,
  ArrowRight,
  ArrowLeft,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { TiltCard } from "./motion/TiltCard";

export function AgencyCareRetainerSection() {
  const { t, isRTL } = useI18n();

  const features = [
    {
      id: "portfolio",
      icon: FolderKanban,
      title: t.landingVal1Title,
      description: t.landingVal1Desc,
      tag: isRTL ? "مدیریت پرتفوی" : "Portfolio Care",
      badge: "Multi-Client",
    },
    {
      id: "proactive",
      icon: ShieldCheck,
      title: t.landingVal2Title,
      description: t.landingVal2Desc,
      tag: isRTL ? "کشف پیشگیرانه" : "Proactive Discovery",
      badge: "Early Warning",
    },
    {
      id: "maintenance",
      icon: Wrench,
      title: t.landingCap5Title,
      description: t.landingCap5Desc,
      tag: isRTL ? "سوابق نگهداری" : "Maintenance Logs",
      badge: "Retainer Proof",
    },
    {
      id: "workspace",
      icon: Lock,
      title: t.landingCap4Title,
      description: t.landingCap4Desc,
      tag: isRTL ? "امنیت و تفکیک سازمانی" : "Enterprise RLS",
      badge: "PostgreSQL Isolation",
    },
  ];

  return (
    <section id="value" className="py-20 sm:py-32 bg-[#060913] text-slate-100 border-b border-slate-800 scroll-mt-20 relative overflow-hidden">
      {/* Background ambient lighting */}
      <div className="absolute top-1/3 left-1/4 w-96 h-96 bg-indigo-600/10 rounded-full blur-[140px] pointer-events-none" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Section Header */}
        <div className="max-w-3xl mx-auto text-center space-y-4 mb-16 sm:mb-20">
          <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full text-xs font-semibold text-indigo-400 bg-indigo-950/60 border border-indigo-800/60 shadow-lg shadow-indigo-950/30 backdrop-blur-md">
            <Users className="w-3.5 h-3.5 text-indigo-400" />
            <span>{isRTL ? "امکانات ویژه برای تیم‌ها و آژانس‌های دیجیتال" : "Dedicated Team & Agency Workspaces"}</span>
          </div>

          <h2 className="text-3xl sm:text-5xl font-black text-white tracking-tight leading-tight">
            {t.landingValueTitle}
          </h2>

          <p className="text-base sm:text-lg text-slate-300 leading-relaxed font-normal">
            {t.landingValueSubtitle}
          </p>
        </div>

        {/* 2x2 Clean Value Grid with 3D Tilt */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 sm:gap-8 max-w-5xl mx-auto">
          {features.map((item) => {
            const Icon = item.icon;
            return (
              <TiltCard key={item.id} maxTilt={3} sheen={true} className="h-full">
                <div className="h-full bg-[#0a0f1d] rounded-2xl sm:rounded-3xl border border-slate-800 p-6 sm:p-8 hover:border-slate-700 transition-all space-y-4 shadow-xl flex flex-col justify-between">
                  <div>
                    <div className="flex items-center justify-between mb-4">
                      <div className="w-11 h-11 rounded-xl bg-indigo-950/80 border border-indigo-800/60 text-indigo-400 flex items-center justify-center shadow-md">
                        <Icon className="w-5 h-5" />
                      </div>
                      <span className="text-[11px] font-mono font-bold text-slate-400 bg-slate-900 px-3 py-1 rounded-lg border border-slate-800">
                        {item.badge}
                      </span>
                    </div>

                    <h3 className="text-lg font-bold text-white tracking-tight mb-2">
                      {item.title}
                    </h3>

                    <p className="text-xs sm:text-sm text-slate-400 leading-relaxed font-normal">
                      {item.description}
                    </p>
                  </div>

                  <div className="pt-4 border-t border-slate-800/80 flex items-center justify-between text-xs text-slate-500 font-mono">
                    <span>{item.tag}</span>
                    <span className="text-indigo-400 font-bold">Enabled</span>
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

