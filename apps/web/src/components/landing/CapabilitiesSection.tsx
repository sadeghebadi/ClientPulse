"use client";

import React from "react";
import {
  Activity,
  Radio,
  AlertTriangle,
  Users,
  Wrench,
  FileText,
  ShieldCheck,
  CheckCircle2,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { TiltCard } from "./motion/TiltCard";

export function CapabilitiesSection() {
  const { t, isRTL } = useI18n();

  const capabilities = [
    {
      id: "monitoring",
      icon: Activity,
      color: "text-blue-400 bg-blue-950/60 border-blue-800/80",
      title: t.landingCap1Title,
      description: t.landingCap1Desc,
      highlights: [
        isRTL ? "پایش کدهای پاسخ ۲۰۰ تا ۵۹۹" : "Full HTTP status validation (200-599)",
        isRTL ? "اندازه‌گیری دقیق تاخیر پاسخ بر حسب ms" : "Millisecond response latency tracking",
        isRTL ? "محافظت چندلایه SSRF و مسدودسازی IP محلی" : "Multi-layer SSRF & loopback address block",
      ],
    },
    {
      id: "heartbeats",
      icon: Radio,
      color: "text-purple-400 bg-purple-950/60 border-purple-800/80",
      title: t.landingCap2Title,
      description: t.landingCap2Desc,
      highlights: [
        isRTL ? "آدرس‌های پینگ منحصر‌به‌فرد برای هر وظیفه" : "Unique cryptographically secure ping URLs",
        isRTL ? "بازه انتظار و مهلت انعطاف‌پذیر (Grace Period)" : "Configurable expected interval & grace window",
        isRTL ? "تمایز دقیق قطعی وب‌سرور از توقف ورکر" : "Distinct missed heartbeat alerting",
      ],
    },
    {
      id: "incidents",
      icon: AlertTriangle,
      color: "text-rose-400 bg-rose-950/60 border-rose-800/80",
      title: t.landingCap3Title,
      description: t.landingCap3Desc,
      highlights: [
        isRTL ? "ایجاد خودکار رخداد بر مبنای سیاست شکست" : "Automated creation on confirmed failures",
        isRTL ? "گردش کار رسمی: باز -> تایید -> حل‌شده" : "Standard lifecycle: Open -> Ack -> Resolved",
        isRTL ? "ثبت سوابق رفع عیب و مسئول پیگیری" : "Audit timeline with resolution notes",
      ],
    },
    {
      id: "clients",
      icon: Users,
      color: "text-indigo-400 bg-indigo-950/60 border-indigo-800/80",
      title: t.landingCap4Title,
      description: t.landingCap4Desc,
      highlights: [
        isRTL ? "دسته‌بندی دارایی‌ها بر پایه قرارداد کارفرما" : "Client-centric asset association",
        isRTL ? "جداسازی محیط‌های Production و Staging" : "Environment classification (prod / staging)",
        isRTL ? "ثبت مخاطبین فنی و یادداشت‌های عملیاتی" : "Dedicated contact & operational metadata",
      ],
    },
    {
      id: "maintenance",
      icon: Wrench,
      color: "text-amber-400 bg-amber-950/60 border-amber-800/80",
      title: t.landingCap5Title,
      description: t.landingCap5Desc,
      highlights: [
        isRTL ? "ثبت زمان‌بندی پیش‌بینی‌شده در برابر زمان واقعی" : "Scheduled vs actual maintenance duration",
        isRTL ? "مستندسازی اقدامات برای تمدید قرارداد" : "Documented work history for SLA proof",
        isRTL ? "اتصال مستقیم هر عملیات به پروفایل مشتری" : "Linked directly to client account history",
      ],
    },
    {
      id: "workspace",
      icon: FileText,
      color: "text-emerald-400 bg-emerald-950/60 border-emerald-800/80",
      title: t.landingCap6Title,
      description: t.landingCap6Desc,
      highlights: [
        isRTL ? "جداسازی سازمانی با PostgreSQL RLS" : "Tenant isolation with Row-Level Security",
        isRTL ? "دسترسی‌های نقشی: مالک، مدیر، عضو، ناظر" : "Granular RBAC (Owner, Admin, Member, Viewer)",
        isRTL ? "لینک‌های امن و اختصاصی گزارش به کارفرما" : "Tokenized shareable public client reports",
      ],
    },
  ];

  return (
    <section id="features" className="py-20 sm:py-28 bg-[#090d18] text-slate-100 border-b border-slate-800 scroll-mt-20 relative overflow-hidden">
      {/* Background ambient lighting */}
      <div className="absolute top-1/3 left-10 w-72 h-72 bg-blue-600/5 rounded-full blur-[90px] pointer-events-none" />
      <div className="absolute bottom-10 right-10 w-72 h-72 bg-violet-600/5 rounded-full blur-[90px] pointer-events-none" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Section Header */}
        <div className="max-w-3xl mx-auto text-center space-y-4 mb-12 sm:mb-16">
          <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full text-xs font-semibold text-blue-400 bg-blue-950/60 border border-blue-800/60 shadow-lg shadow-blue-950/30">
            <ShieldCheck className="w-3.5 h-3.5 text-blue-400" />
            <span>{isRTL ? "قابلیت‌های پیاده‌سازی‌شده و فعال" : "Core Implemented Capabilities"}</span>
          </div>

          <h2 className="text-3xl sm:text-5xl font-black text-white tracking-tight">
            {t.landingCapabilitiesTitle}
          </h2>

          <p className="text-base sm:text-lg text-slate-400 leading-relaxed font-normal">
            {t.landingCapabilitiesSubtitle}
          </p>
        </div>

        {/* 6 Capabilities Cards Grid with 3D Tilt */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 sm:gap-8">
          {capabilities.map((cap) => {
            const Icon = cap.icon;
            return (
              <TiltCard key={cap.id} maxTilt={5} sheen={true} className="h-full">
                <div className="h-full group relative bg-[#0c1222]/90 rounded-2xl border border-slate-800/90 p-6 sm:p-7 shadow-xl hover:border-slate-700 hover:shadow-2xl transition-all flex flex-col justify-between">
                  <div className="space-y-4">
                    <div className={`w-12 h-12 rounded-xl flex items-center justify-center border shadow-md ${cap.color}`}>
                      <Icon className="w-6 h-6" />
                    </div>

                    <h3 className="text-lg font-bold text-white tracking-tight group-hover:text-blue-400 transition-colors">
                      {cap.title}
                    </h3>

                    <p className="text-xs sm:text-sm text-slate-400 leading-relaxed font-normal">
                      {cap.description}
                    </p>
                  </div>

                  <div className="pt-6 mt-6 border-t border-slate-800 space-y-2.5">
                    {cap.highlights.map((hl, idx) => (
                      <div key={idx} className="flex items-start gap-2.5 text-xs text-slate-300">
                        <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400 shrink-0 mt-0.5" />
                        <span className="leading-tight">{hl}</span>
                      </div>
                    ))}
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
