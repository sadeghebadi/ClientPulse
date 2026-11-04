"use client";

import React, { useState } from "react";
import {
  AlertTriangle,
  CheckCircle2,
  Clock,
  ShieldCheck,
  UserCheck,
  FileText,
  ChevronRight,
  ChevronDown,
  Terminal,
  Activity,
  Layers,
  ArrowRight,
  ArrowLeft,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";

export function IncidentCommandSection() {
  const { t, isRTL } = useI18n();

  const [activeStep, setActiveStep] = useState<number>(2);

  const steps = [
    {
      id: 0,
      title: isRTL ? "تایید شکست متوالی" : "Consecutive Failure Verified",
      time: "14:02:11 UTC",
      status: "critical",
      summary: isRTL
        ? "۲ بررسی پیاپی با خطای ۵۰۲ روبرو شد؛ بروز قطعی رسمی تایید و فرآیند بازگشایی رخداد آغاز شد."
        : "2 consecutive HTTP 502 probes confirmed target outage. Flapping prevention verified.",
      details: {
        code: "HTTP 502 Bad Gateway",
        threshold: "2 consecutive checks",
        probe: "us-east-1",
      },
    },
    {
      id: 1,
      title: isRTL ? "بازگشایی خودکار رخداد #INC-904" : "Incident #INC-904 Opened",
      time: "14:02:14 UTC",
      status: "open",
      summary: isRTL
        ? "رخداد بحرانی در سیستم ثبت شد و خطای اولیه همراه با متادیتای سرور در گزارش رخداد لاگ گردید."
        : "Triage engine automatically assigned Critical priority and captured runtime diagnostic headers.",
      details: {
        incidentId: "INC-904",
        priority: "P1 - Critical",
        affectedAsset: "api.store.internal",
      },
    },
    {
      id: 2,
      title: isRTL ? "تایید دریافت توسط اپراتور کشیک" : "Operator Acknowledgment",
      time: "14:03:02 UTC",
      status: "ack",
      summary: isRTL
        ? "اپراتور کشیک دریافت هشدار را تایید نمود؛ مسئولیت پیگیری ثبت و وضعیت به «در حال بررسی» تغییر یافت."
        : "On-call engineer acknowledged incident via operational cockpit. Time-to-acknowledge: 48s.",
      details: {
        assignedTo: "Lead SRE Engineer",
        state: "Acknowledged & In Progress",
      },
    },
    {
      id: 3,
      title: isRTL ? "رفع ریشه عیب و بازگشت سرویس" : "Mitigation & Recovery Verified",
      time: "14:04:25 UTC",
      status: "resolved",
      summary: isRTL
        ? "سرور با موفقیت پاسخ ۲۰۰ بازگرداند؛ رخداد بسته شد و یادداشت ریشه‌یابی به همراه مدت زمان کل قطعی ذخیره گردید."
        : "Probes confirm HTTP 200 OK across 3 consecutive checks. Incident closed with total duration: 2m 14s.",
      details: {
        rootCause: "Memory pressure on checkout container",
        totalDowntime: "2m 14s",
        resolutionStatus: "Resolved",
      },
    },
  ];

  return (
    <section className="py-20 sm:py-32 bg-[#060913] text-slate-100 border-b border-slate-800 relative overflow-hidden">
      {/* Background radial accent */}
      <div className="absolute top-1/3 right-10 w-96 h-96 bg-rose-600/10 rounded-full blur-[140px] pointer-events-none" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Section Header */}
        <div className="max-w-3xl mx-auto text-center space-y-4 mb-16 sm:mb-20">
          <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full text-xs font-semibold text-rose-400 bg-rose-950/60 border border-rose-800/60 shadow-lg shadow-rose-950/30 backdrop-blur-md">
            <AlertTriangle className="w-3.5 h-3.5 text-rose-400" />
            <span>{isRTL ? "میز فرماندهی و چرخه رخدادها" : "Incident Command & Resolution"}</span>
          </div>

          <h2 className="text-3xl sm:text-5xl font-black text-white tracking-tight leading-tight">
            {t.landingCap3Title}
          </h2>

          <p className="text-base sm:text-lg text-slate-300 leading-relaxed font-normal">
            {t.landingCap3Desc}
          </p>
        </div>

        {/* Tactical Command Desk Timeline (No Card Grid!) */}
        <div className="max-w-5xl mx-auto rounded-3xl bg-[#090e1a] border border-slate-800 shadow-2xl overflow-hidden ring-1 ring-white/10">
          {/* Header Bar */}
          <div className="p-4 sm:p-6 bg-slate-950/80 border-b border-slate-800 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <span className="w-2.5 h-2.5 rounded-full bg-rose-500 animate-pulse" />
              <span className="font-mono text-sm font-bold text-white">
                Incident Lifecycle Audit: #INC-904
              </span>
            </div>

            <div className="flex items-center gap-2 text-xs font-mono text-slate-400">
              <Clock className="w-3.5 h-3.5 text-blue-400" />
              <span>Full Audit Trail Preserved</span>
            </div>
          </div>

          {/* Interactive Timeline Stepper Area */}
          <div className="p-6 sm:p-8 space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
              {steps.map((st, idx) => {
                const isSelected = activeStep === idx;
                return (
                  <button
                    key={st.id}
                    type="button"
                    onClick={() => setActiveStep(idx)}
                    className={`p-4 rounded-2xl border text-left transition-all ${
                      isSelected
                        ? "bg-slate-800/90 border-blue-500 shadow-lg ring-1 ring-blue-500/40"
                        : "bg-slate-900/60 border-slate-800 hover:bg-slate-800/40 text-slate-400"
                    }`}
                  >
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-[10px] font-mono font-bold uppercase tracking-wider text-blue-400">
                        Phase 0{idx + 1}
                      </span>
                      <span className="text-[10px] font-mono text-slate-500">{st.time}</span>
                    </div>
                    <div className="text-xs font-bold text-white truncate">{st.title}</div>
                  </button>
                );
              })}
            </div>

            {/* Selected Step Expanded Details Display */}
            <div className="p-6 rounded-2xl bg-[#070b14] border border-slate-800/80 space-y-4">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-800/80 pb-4">
                <div>
                  <h4 className="text-base font-bold text-white">
                    {steps[activeStep].title}
                  </h4>
                  <p className="text-xs text-slate-400 mt-1">
                    {steps[activeStep].summary}
                  </p>
                </div>

                <div className="text-xs font-mono text-slate-300 px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-800 shrink-0">
                  Timestamp: {steps[activeStep].time}
                </div>
              </div>

              {/* Step Parameter Readouts */}
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs font-mono">
                {Object.entries(steps[activeStep].details).map(([key, val]) => (
                  <div key={key} className="p-3 rounded-xl bg-slate-950 border border-slate-800">
                    <span className="text-[10px] text-slate-500 uppercase font-semibold block mb-0.5">
                      {key}
                    </span>
                    <span className="text-slate-200 font-bold">{val}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

