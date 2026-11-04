"use client";

import React, { useState } from "react";
import {
  FileText,
  BarChart3,
  Calendar,
  Share2,
  Download,
  ExternalLink,
  ShieldCheck,
  CheckCircle2,
  TrendingUp,
  Percent,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";

export function HistoricalReportsAnalyticsSection() {
  const { t, isRTL } = useI18n();

  const [hoveredDay, setHoveredDay] = useState<{ day: number; uptime: string; latency: number } | null>(null);
  const [selectedPercentile, setSelectedPercentile] = useState<"p50" | "p95" | "p99">("p95");

  // Generate 90 days of uptime blocks (99.9% - 100%)
  const days = Array.from({ length: 90 }, (_, i) => {
    const isSlightDip = i === 34 || i === 71;
    return {
      day: 90 - i,
      uptime: isSlightDip ? "99.82%" : "100.0%",
      latency: isSlightDip ? 58 : 32 + (i % 7),
      status: isSlightDip ? "amber" : "emerald",
    };
  });

  return (
    <section className="py-20 sm:py-32 bg-[#050811] text-slate-100 border-b border-slate-800 relative overflow-hidden">
      {/* Background ambient lighting */}
      <div className="absolute top-1/2 left-1/3 w-[600px] h-[350px] bg-emerald-600/10 rounded-full blur-[140px] pointer-events-none" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Section Header */}
        <div className="max-w-3xl mx-auto text-center space-y-4 mb-16 sm:mb-20">
          <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full text-xs font-semibold text-emerald-400 bg-emerald-950/60 border border-emerald-800/60 shadow-lg shadow-emerald-950/30 backdrop-blur-md">
            <BarChart3 className="w-3.5 h-3.5 text-emerald-400" />
            <span>{isRTL ? "تحلیل تاریخی و گزارش‌های شفاف" : "Historical Analytics & Reports"}</span>
          </div>

          <h2 className="text-3xl sm:text-5xl font-black text-white tracking-tight leading-tight">
            {t.landingCap6Title}
          </h2>

          <p className="text-base sm:text-lg text-slate-300 leading-relaxed font-normal">
            {t.landingCap6Desc}
          </p>
        </div>

        {/* 90-Day Analytics Heatmap Console */}
        <div className="max-w-5xl mx-auto rounded-3xl bg-[#090e1a] border border-slate-800 shadow-2xl overflow-hidden ring-1 ring-white/10">
          {/* Console Header */}
          <div className="p-4 sm:p-6 bg-slate-950/80 border-b border-slate-800 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div>
              <div className="text-base font-bold text-white flex items-center gap-2">
                <Calendar className="w-4 h-4 text-emerald-400" />
                <span>90-Day Operational Health Matrix</span>
              </div>
              <div className="text-xs text-slate-400 font-mono mt-0.5">
                Aggregate Uptime: 99.98% · 24 Monitored Targets
              </div>
            </div>

            {/* Percentile Selector */}
            <div className="flex items-center gap-1.5 p-1 rounded-xl bg-slate-900 border border-slate-800 text-xs font-mono font-bold">
              {(["p50", "p95", "p99"] as const).map((p) => (
                <button
                  key={p}
                  type="button"
                  onClick={() => setSelectedPercentile(p)}
                  className={`px-3 py-1 rounded-lg uppercase transition-colors ${
                    selectedPercentile === p
                      ? "bg-emerald-600 text-white shadow-sm"
                      : "text-slate-400 hover:text-white"
                  }`}
                >
                  {p}
                </button>
              ))}
            </div>
          </div>

          {/* Heatmap Area */}
          <div className="p-6 sm:p-8 space-y-6">
            <div className="space-y-3">
              <div className="flex items-center justify-between text-xs text-slate-400 font-mono">
                <span>90 Days Ago</span>
                <span className="text-emerald-400 font-bold">Today (100% Operational)</span>
              </div>

              {/* 90-Block Grid */}
              <div className="grid grid-cols-[repeat(15,minmax(0,1fr))] sm:grid-cols-[repeat(30,minmax(0,1fr))] gap-1 sm:gap-1.5">
                {days.map((d) => (
                  <div
                    key={d.day}
                    onMouseEnter={() => setHoveredDay(d)}
                    onMouseLeave={() => setHoveredDay(null)}
                    className={`h-5 sm:h-7 rounded transition-all cursor-pointer ${
                      d.status === "amber"
                        ? "bg-amber-500/80 hover:bg-amber-400"
                        : "bg-emerald-500/80 hover:bg-emerald-400 hover:scale-110"
                    }`}
                  />
                ))}
              </div>
            </div>

            {/* Hovered Day Feedback Pill */}
            <div className="p-3.5 rounded-xl bg-slate-950/80 border border-slate-800/80 flex items-center justify-between text-xs font-mono">
              <span className="text-slate-400">
                {hoveredDay
                  ? `Day -${hoveredDay.day}: ${hoveredDay.uptime} Availability · Avg Latency: ${hoveredDay.latency}ms`
                  : "Hover any block to inspect daily performance metrics"}
              </span>
              <span className="text-emerald-400 font-bold">
                {selectedPercentile.toUpperCase()}: {selectedPercentile === "p50" ? "28ms" : selectedPercentile === "p95" ? "44ms" : "78ms"}
              </span>
            </div>

            {/* Tokenized Public Client Report Preview Card */}
            <div className="p-4 sm:p-5 rounded-2xl bg-slate-900/90 border border-slate-800 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div className="space-y-1">
                <div className="text-xs font-bold text-white flex items-center gap-1.5">
                  <Share2 className="w-3.5 h-3.5 text-blue-400" />
                  <span>Tokenized Shareable Client Report</span>
                </div>
                <p className="text-[11px] text-slate-400">
                  {isRTL
                    ? "ارائه گزارش‌های شفاف عملکرد به کارفرما با لینک امن عمومی بدون نیاز به ایجاد حساب کاربری مجزا."
                    : "Generate secure, branded, read-only status links to share with clients or stakeholders."}
                </p>
              </div>

              <div className="px-3.5 py-1.5 rounded-xl bg-slate-800 border border-slate-700 text-xs font-mono text-slate-300 flex items-center gap-2 shrink-0">
                <span className="ltr-text">https://mindu.ir/r/pub-token-884</span>
                <ExternalLink className="w-3 h-3 text-slate-400" />
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
