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
  Lock,
  ArrowRight,
  ArrowLeft,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";

export function CapabilitiesSection() {
  const { t, isRTL } = useI18n();

  const capabilities = [
    {
      id: "monitoring",
      icon: Activity,
      color: "text-blue-600 bg-blue-50 border-blue-200",
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
      color: "text-purple-600 bg-purple-50 border-purple-200",
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
      color: "text-rose-600 bg-rose-50 border-rose-200",
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
      color: "text-indigo-600 bg-indigo-50 border-indigo-200",
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
      color: "text-amber-600 bg-amber-50 border-amber-200",
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
      color: "text-emerald-600 bg-emerald-50 border-emerald-200",
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
    <section id="features" className="py-16 sm:py-24 bg-white border-b border-slate-200/80 scroll-mt-20">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Section Header */}
        <div className="max-w-3xl mx-auto text-center space-y-3 sm:space-y-4 mb-12 sm:mb-16">
          <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-bold text-blue-700 bg-blue-50 border border-blue-200/60">
            <ShieldCheck className="w-3.5 h-3.5 text-blue-600" />
            <span>{isRTL ? "قابلیت‌های پیاده‌سازی‌شده و فعال" : "Core Implemented Capabilities"}</span>
          </div>

          <h2 className="text-2xl sm:text-4xl font-extrabold text-slate-900 tracking-tight">
            {t.landingCapabilitiesTitle}
          </h2>

          <p className="text-sm sm:text-base text-slate-600 leading-relaxed font-normal">
            {t.landingCapabilitiesSubtitle}
          </p>
        </div>

        {/* 6 Capabilities Cards Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 sm:gap-8">
          {capabilities.map((cap) => {
            const Icon = cap.icon;
            return (
              <div
                key={cap.id}
                className="group relative bg-white rounded-2xl border border-slate-200/80 p-6 sm:p-7 shadow-2xs hover:shadow-lg hover:border-blue-300 transition-all flex flex-col justify-between"
              >
                <div className="space-y-4">
                  <div className={`w-12 h-12 rounded-xl flex items-center justify-center border shadow-xs ${cap.color}`}>
                    <Icon className="w-6 h-6" />
                  </div>

                  <h3 className="text-lg font-bold text-slate-900 tracking-tight group-hover:text-blue-600 transition-colors">
                    {cap.title}
                  </h3>

                  <p className="text-xs sm:text-sm text-slate-600 leading-relaxed font-normal">
                    {cap.description}
                  </p>
                </div>

                <div className="pt-6 mt-6 border-t border-slate-100 space-y-2">
                  {cap.highlights.map((hl, idx) => (
                    <div key={idx} className="flex items-start gap-2 text-xs text-slate-700">
                      <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600 shrink-0 mt-0.5" />
                      <span className="leading-tight">{hl}</span>
                    </div>
                  ))}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}

