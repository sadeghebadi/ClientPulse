"use client";

import React, { useState } from "react";
import {
  Activity,
  Radio,
  AlertTriangle,
  Users,
  LayoutDashboard,
  CheckCircle2,
  Clock,
  ShieldCheck,
  Globe,
  ExternalLink,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { TiltCard } from "./motion/TiltCard";

export function ProductPreviewSection() {
  const { t, isRTL } = useI18n();
  const [activeTab, setActiveTab] = useState<"dashboard" | "monitors" | "heartbeats" | "incidents" | "clients">("dashboard");

  const tabs = [
    { id: "dashboard", label: t.landingTabDashboard, icon: LayoutDashboard },
    { id: "monitors", label: t.landingTabMonitors, icon: Activity },
    { id: "heartbeats", label: t.landingTabHeartbeats, icon: Radio },
    { id: "incidents", label: t.landingTabIncidents, icon: AlertTriangle },
    { id: "clients", label: t.landingTabClients, icon: Users },
  ];

  return (
    <section id="preview" className="py-20 sm:py-28 bg-[#070b14] text-slate-100 border-b border-slate-800 scroll-mt-20 relative overflow-hidden">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Section Header */}
        <div className="max-w-3xl mx-auto text-center space-y-4 mb-10 sm:mb-12">
          <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full text-xs font-semibold text-blue-400 bg-blue-950/60 border border-blue-800/60 shadow-lg shadow-blue-950/30">
            <span className="w-2 h-2 rounded-full bg-blue-400 animate-pulse" />
            <span>{isRTL ? "پیش‌نمایش محصول · داده‌های نمونه" : "Product Preview · Sample Data"}</span>
          </div>

          <h2 className="text-3xl sm:text-5xl font-black text-white tracking-tight">
            {t.landingPreviewTitle}
          </h2>

          <p className="text-base sm:text-lg text-slate-400 leading-relaxed font-normal">
            {t.landingPreviewSubtitle}
          </p>
        </div>

        {/* Tab Selection Bar */}
        <div className="flex items-center justify-start sm:justify-center overflow-x-auto pb-3 mb-8 gap-2 no-scrollbar" role="tablist">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            const isActive = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                role="tab"
                aria-selected={isActive}
                type="button"
                onClick={() => setActiveTab(tab.id as any)}
                className={`inline-flex items-center gap-2 px-4 py-2.5 rounded-xl text-xs sm:text-sm font-bold whitespace-nowrap transition-all shrink-0 border ${
                  isActive
                    ? "bg-blue-600 text-white border-blue-500 shadow-lg shadow-blue-600/30 ring-1 ring-blue-400"
                    : "bg-slate-900/80 text-slate-400 hover:text-white hover:bg-slate-800 border-slate-800"
                }`}
              >
                <Icon className="w-4 h-4 shrink-0" />
                <span>{tab.label}</span>
              </button>
            );
          })}
        </div>

        {/* Tab Content Display Container */}
        <div className="max-w-5xl mx-auto rounded-2xl sm:rounded-3xl bg-[#0c1222] border border-slate-800 shadow-2xl overflow-hidden ring-1 ring-slate-700/50">
          {/* Subheader Context */}
          <div className="px-5 py-3 bg-slate-950 text-slate-300 text-xs flex items-center justify-between border-b border-slate-800/80">
            <div className="flex items-center gap-2 font-mono text-[11px]">
              <span className="w-2 h-2 rounded-full bg-blue-400" />
              <span>mindu.ir / {activeTab}</span>
            </div>
            <span className="text-[10px] text-slate-400 font-semibold">
              {isRTL ? "پیش‌نمایش محصول · داده‌های نمونه" : "Product Preview · Sample Data"}
            </span>
          </div>

          {/* Tab 1: Executive Dashboard */}
          {activeTab === "dashboard" && (
            <div className="p-4 sm:p-6 lg:p-8 space-y-6 bg-[#090e1a]">
              {/* Executive System Health Bar */}
              <div className="p-4 rounded-xl bg-slate-900/90 border border-emerald-500/30 flex flex-col sm:flex-row sm:items-center justify-between gap-3 shadow-lg">
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-lg bg-emerald-500/20 text-emerald-400 flex items-center justify-center font-bold">
                    <CheckCircle2 className="w-5 h-5" />
                  </div>
                  <div>
                    <h4 className="text-xs sm:text-sm font-bold text-white">
                      {isRTL ? "تمام سیستم‌ها پایدارند" : "All Systems Operational"}
                    </h4>
                    <p className="text-[11px] text-slate-400">
                      {isRTL ? "پایش فعال در ۱۸ وب‌سایت و ۶ هارت‌بیت کرون‌جاب" : "Active checks across 18 websites and 6 cron heartbeats"}
                    </p>
                  </div>
                </div>
                <span className="px-2.5 py-1 text-xs font-bold text-emerald-400 bg-emerald-500/15 rounded-lg border border-emerald-500/30 self-start sm:self-auto">
                  100% Available
                </span>
              </div>

              {/* 4 KPIs */}
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                <div className="p-3.5 bg-slate-900/80 rounded-xl border border-slate-800 text-center space-y-0.5">
                  <div className="text-[10px] uppercase font-bold text-slate-400">{t.monitoredTargetsKpi}</div>
                  <div className="text-xl font-black text-white">24</div>
                </div>
                <div className="p-3.5 bg-slate-900/80 rounded-xl border border-slate-800 text-center space-y-0.5">
                  <div className="text-[10px] uppercase font-bold text-slate-400">{t.targetsDownKpi}</div>
                  <div className="text-xl font-black text-emerald-400">0</div>
                </div>
                <div className="p-3.5 bg-slate-900/80 rounded-xl border border-slate-800 text-center space-y-0.5">
                  <div className="text-[10px] uppercase font-bold text-slate-400">{t.openIncidentsKpi}</div>
                  <div className="text-xl font-black text-white">0</div>
                </div>
                <div className="p-3.5 bg-slate-900/80 rounded-xl border border-slate-800 text-center space-y-0.5">
                  <div className="text-[10px] uppercase font-bold text-slate-400">{t.avgLatencyKpi}</div>
                  <div className="text-xl font-black text-blue-400">38 ms</div>
                </div>
              </div>
            </div>
          )}

          {/* Tab 2: Monitors */}
          {activeTab === "monitors" && (
            <div className="p-4 sm:p-6 lg:p-8 space-y-4 bg-[#090e1a]">
              <div className="bg-slate-900/80 rounded-xl border border-slate-800 divide-y divide-slate-800/80 overflow-hidden shadow-lg">
                {[
                  { name: "Main Marketing Portal", url: "https://example-client.com", code: "200 OK", latency: "29 ms", status: "up" },
                  { name: "Customer Checkout API", url: "https://api.example-client.com/pay", code: "200 OK", latency: "44 ms", status: "up" },
                  { name: "Staging Preview Webapp", url: "https://staging.example-client.com", code: "200 OK", latency: "62 ms", status: "up" },
                ].map((mon, i) => (
                  <div key={i} className="p-3.5 sm:px-4 flex items-center justify-between gap-4 hover:bg-slate-850 transition-colors">
                    <div className="min-w-0">
                      <div className="font-bold text-xs sm:text-sm text-white truncate">{mon.name}</div>
                      <div className="text-[11px] font-mono text-slate-400 truncate">{mon.url}</div>
                    </div>
                    <div className="flex items-center gap-2 sm:gap-4 shrink-0 text-xs">
                      <span className="text-slate-300 font-semibold tabular-nums">{mon.latency}</span>
                      <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-emerald-500/15 text-emerald-400 border border-emerald-500/30">
                        {mon.code}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Tab 3: Heartbeats */}
          {activeTab === "heartbeats" && (
            <div className="p-4 sm:p-6 lg:p-8 space-y-4 bg-[#090e1a]">
              <div className="bg-slate-900/80 rounded-xl border border-slate-800 divide-y divide-slate-800/80 overflow-hidden shadow-lg">
                {[
                  { name: "Nightly Invoice Generation", interval: "Every 24 hours", grace: "30m grace", lastPing: "1 hour ago", status: "healthy" },
                  { name: "Hourly Redis Cache Warmer", interval: "Every 60 minutes", grace: "10m grace", lastPing: "14 minutes ago", status: "healthy" },
                  { name: "Elasticsearch Index Sync", interval: "Every 15 minutes", grace: "5m grace", lastPing: "2 minutes ago", status: "healthy" },
                ].map((hb, i) => (
                  <div key={i} className="p-3.5 sm:px-4 flex items-center justify-between gap-4 hover:bg-slate-850 transition-colors">
                    <div className="min-w-0">
                      <div className="font-bold text-xs sm:text-sm text-white truncate">{hb.name}</div>
                      <div className="text-[11px] text-slate-400">{hb.interval} · {hb.grace}</div>
                    </div>
                    <div className="flex items-center gap-2 sm:gap-4 shrink-0 text-xs">
                      <span className="text-slate-400 text-[11px] hidden sm:inline">{hb.lastPing}</span>
                      <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-blue-500/15 text-blue-300 border border-blue-500/30">
                        Healthy
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Tab 4: Incidents */}
          {activeTab === "incidents" && (
            <div className="p-4 sm:p-6 lg:p-8 space-y-4 bg-[#090e1a]">
              <div className="bg-slate-900/80 rounded-xl border border-slate-800 p-4 space-y-3 shadow-lg">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-slate-800 text-slate-300 border border-slate-700">
                      Resolved Incident #104
                    </span>
                    <span className="text-xs text-slate-400">· 2 days ago</span>
                  </div>
                  <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-emerald-500/15 text-emerald-400 border border-emerald-500/30">
                    Resolved in 8m
                  </span>
                </div>
                <div className="text-xs sm:text-sm font-bold text-white">
                  HTTP 502 Bad Gateway detected on Checkout Service API
                </div>
                <div className="p-2.5 rounded-lg bg-slate-950 text-[11px] text-slate-400 border border-slate-800">
                  <span className="font-bold text-slate-300">{isRTL ? "یادداشت رفع مشکل: " : "Resolution Note: "}</span>
                  {isRTL
                    ? "ورکر بالادستی پس از به‌روزرسانی مجدداً راه‌اندازی شد و بررسی سلامت با موفقیت تایید گردید."
                    : "Upstream PHP-FPM pool restarted following worker pool crash. Latency restored to 34ms."}
                </div>
              </div>
            </div>
          )}

          {/* Tab 5: Clients */}
          {activeTab === "clients" && (
            <div className="p-4 sm:p-6 lg:p-8 space-y-4 bg-[#090e1a]">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                {[
                  { name: "Nordic Commerce Ltd", contact: "Elena Rostova", assets: 4, monitors: 4, status: "Active Retainer" },
                  { name: "Atlas Media Agency", contact: "Marcus Vance", assets: 6, monitors: 6, status: "Active Retainer" },
                ].map((c, i) => (
                  <div key={i} className="p-4 bg-slate-900/80 rounded-xl border border-slate-800 shadow-lg space-y-2">
                    <div className="flex items-center justify-between">
                      <div className="font-bold text-sm text-white">{c.name}</div>
                      <span className="text-[10px] font-bold text-emerald-400 bg-emerald-500/15 px-2 py-0.5 rounded border border-emerald-500/30">
                        {c.status}
                      </span>
                    </div>
                    <div className="text-xs text-slate-400">Contact: {c.contact}</div>
                    <div className="pt-2 flex items-center gap-4 text-[11px] font-semibold text-slate-300 border-t border-slate-800">
                      <span>{c.assets} Digital Assets</span>
                      <span>{c.monitors} Live Monitors</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
