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
  ChevronRight,
  ArrowRight,
  ArrowLeft,
  Wrench,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";

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
    <section id="preview" className="py-16 sm:py-24 bg-slate-50 border-b border-slate-200/80 scroll-mt-20">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Section Header */}
        <div className="max-w-3xl mx-auto text-center space-y-3 sm:space-y-4 mb-10 sm:mb-12">
          <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-bold text-slate-700 bg-white border border-slate-200 shadow-2xs">
            <span className="w-2 h-2 rounded-full bg-blue-500" />
            <span>{isRTL ? "پیش‌نمایش محصول · داده‌های نمونه" : "Product Preview · Sample Data"}</span>
          </div>

          <h2 className="text-2xl sm:text-4xl font-extrabold text-slate-900 tracking-tight">
            {t.landingPreviewTitle}
          </h2>

          <p className="text-sm sm:text-base text-slate-600 leading-relaxed font-normal">
            {t.landingPreviewSubtitle}
          </p>
        </div>

        {/* Tab Selection Bar */}
        <div className="flex items-center justify-start sm:justify-center overflow-x-auto pb-2 mb-8 gap-2 no-scrollbar">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            const isActive = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                type="button"
                onClick={() => setActiveTab(tab.id as any)}
                className={`inline-flex items-center gap-2 px-3.5 sm:px-4 py-2 rounded-xl text-xs sm:text-sm font-bold whitespace-nowrap transition-all shrink-0 ${
                  isActive
                    ? "bg-blue-600 text-white shadow-sm"
                    : "bg-white text-slate-600 hover:text-slate-900 hover:bg-slate-100/80 border border-slate-200"
                }`}
              >
                <Icon className="w-4 h-4 shrink-0" />
                <span>{tab.label}</span>
              </button>
            );
          })}
        </div>

        {/* Tab Content Display Container */}
        <div className="max-w-5xl mx-auto rounded-2xl bg-white border border-slate-200 shadow-xl overflow-hidden">
          {/* Subheader Context */}
          <div className="px-5 py-3 bg-slate-900 text-slate-300 text-xs flex items-center justify-between border-b border-slate-800">
            <div className="flex items-center gap-2 font-mono text-[11px]">
              <span className="w-2 h-2 rounded-full bg-blue-500" />
              <span>mindu.ir / {activeTab}</span>
            </div>
            <span className="text-[10px] text-slate-400 font-semibold">
              {isRTL ? "پیش‌نمایش محصول · داده‌های نمونه" : "Product Preview · Sample Data"}
            </span>
          </div>

          {/* Tab 1: Executive Dashboard */}
          {activeTab === "dashboard" && (
            <div className="p-4 sm:p-6 lg:p-8 space-y-6 bg-slate-50">
              {/* Executive System Health Bar */}
              <div className="p-4 rounded-xl bg-white border border-emerald-200 flex flex-col sm:flex-row sm:items-center justify-between gap-3 shadow-2xs">
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-lg bg-emerald-100 text-emerald-600 flex items-center justify-center font-bold">
                    <CheckCircle2 className="w-5 h-5" />
                  </div>
                  <div>
                    <h4 className="text-xs sm:text-sm font-bold text-slate-900">
                      {isRTL ? "تمام سیستم‌ها پایدارند" : "All Systems Operational"}
                    </h4>
                    <p className="text-[11px] text-slate-500">
                      {isRTL ? "پایش فعال در ۱۸ وب‌سایت و ۶ هارت‌بیت کرون‌جاب" : "Active checks across 18 websites and 6 cron heartbeats"}
                    </p>
                  </div>
                </div>
                <span className="px-2.5 py-1 text-xs font-bold text-emerald-700 bg-emerald-50 rounded-lg border border-emerald-200 self-start sm:self-auto">
                  100% Available
                </span>
              </div>

              {/* 4 KPIs */}
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                <div className="p-3 bg-white rounded-xl border border-slate-200 text-center space-y-0.5">
                  <div className="text-[10px] uppercase font-bold text-slate-400">{t.monitoredTargetsKpi}</div>
                  <div className="text-xl font-black text-slate-900">24</div>
                </div>
                <div className="p-3 bg-white rounded-xl border border-slate-200 text-center space-y-0.5">
                  <div className="text-[10px] uppercase font-bold text-slate-400">{t.targetsDownKpi}</div>
                  <div className="text-xl font-black text-emerald-600">0</div>
                </div>
                <div className="p-3 bg-white rounded-xl border border-slate-200 text-center space-y-0.5">
                  <div className="text-[10px] uppercase font-bold text-slate-400">{t.openIncidentsKpi}</div>
                  <div className="text-xl font-black text-slate-900">0</div>
                </div>
                <div className="p-3 bg-white rounded-xl border border-slate-200 text-center space-y-0.5">
                  <div className="text-[10px] uppercase font-bold text-slate-400">{t.avgLatencyKpi}</div>
                  <div className="text-xl font-black text-blue-600">38 ms</div>
                </div>
              </div>
            </div>
          )}

          {/* Tab 2: Monitors */}
          {activeTab === "monitors" && (
            <div className="p-4 sm:p-6 lg:p-8 space-y-4 bg-slate-50">
              <div className="bg-white rounded-xl border border-slate-200 divide-y divide-slate-100 overflow-hidden shadow-2xs">
                {[
                  { name: "Main Marketing Portal", url: "https://example-client.com", code: "200 OK", latency: "29 ms", status: "up" },
                  { name: "Customer Checkout API", url: "https://api.example-client.com/pay", code: "200 OK", latency: "44 ms", status: "up" },
                  { name: "Staging Preview Webapp", url: "https://staging.example-client.com", code: "200 OK", latency: "62 ms", status: "up" },
                ].map((mon, i) => (
                  <div key={i} className="p-3.5 sm:px-4 flex items-center justify-between gap-4">
                    <div className="min-w-0">
                      <div className="font-bold text-xs sm:text-sm text-slate-900 truncate">{mon.name}</div>
                      <div className="text-[11px] font-mono text-slate-400 truncate">{mon.url}</div>
                    </div>
                    <div className="flex items-center gap-2 sm:gap-4 shrink-0 text-xs">
                      <span className="text-slate-600 font-semibold tabular-nums">{mon.latency}</span>
                      <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-emerald-50 text-emerald-700 border border-emerald-200">
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
            <div className="p-4 sm:p-6 lg:p-8 space-y-4 bg-slate-50">
              <div className="bg-white rounded-xl border border-slate-200 divide-y divide-slate-100 overflow-hidden shadow-2xs">
                {[
                  { name: "Nightly Invoice Generation", interval: "Every 24 hours", grace: "30m grace", lastPing: "1 hour ago", status: "healthy" },
                  { name: "Hourly Redis Cache Warmer", interval: "Every 60 minutes", grace: "10m grace", lastPing: "14 minutes ago", status: "healthy" },
                  { name: "Elasticsearch Index Sync", interval: "Every 15 minutes", grace: "5m grace", lastPing: "2 minutes ago", status: "healthy" },
                ].map((hb, i) => (
                  <div key={i} className="p-3.5 sm:px-4 flex items-center justify-between gap-4">
                    <div className="min-w-0">
                      <div className="font-bold text-xs sm:text-sm text-slate-900 truncate">{hb.name}</div>
                      <div className="text-[11px] text-slate-400">{hb.interval} · {hb.grace}</div>
                    </div>
                    <div className="flex items-center gap-2 sm:gap-4 shrink-0 text-xs">
                      <span className="text-slate-500 text-[11px] hidden sm:inline">{hb.lastPing}</span>
                      <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-blue-50 text-blue-700 border border-blue-200">
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
            <div className="p-4 sm:p-6 lg:p-8 space-y-4 bg-slate-50">
              <div className="bg-white rounded-xl border border-slate-200 p-4 space-y-3 shadow-2xs">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-slate-100 text-slate-700">
                      Resolved Incident #104
                    </span>
                    <span className="text-xs text-slate-400">· 2 days ago</span>
                  </div>
                  <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-emerald-50 text-emerald-700 border border-emerald-200">
                    Resolved in 8m
                  </span>
                </div>
                <div className="text-xs sm:text-sm font-bold text-slate-900">
                  HTTP 502 Bad Gateway detected on Checkout Service API
                </div>
                <div className="p-2.5 rounded-lg bg-slate-50 text-[11px] text-slate-600 border border-slate-100">
                  <span className="font-bold">{isRTL ? "یادداشت رفع مشکل: " : "Resolution Note: "}</span>
                  {isRTL
                    ? "ورکر بالادستی پس از به‌روزرسانی مجدداً راه‌اندازی شد و بررسی سلامت با موفقیت تایید گردید."
                    : "Upstream PHP-FPM pool restarted following worker pool crash. Latency restored to 34ms."}
                </div>
              </div>
            </div>
          )}

          {/* Tab 5: Clients */}
          {activeTab === "clients" && (
            <div className="p-4 sm:p-6 lg:p-8 space-y-4 bg-slate-50">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                {[
                  { name: "Nordic Commerce Ltd", contact: "Elena Rostova", assets: 4, monitors: 4, status: "Active Retainer" },
                  { name: "Atlas Media Agency", contact: "Marcus Vance", assets: 6, monitors: 6, status: "Active Retainer" },
                ].map((c, i) => (
                  <div key={i} className="p-4 bg-white rounded-xl border border-slate-200 shadow-2xs space-y-2">
                    <div className="flex items-center justify-between">
                      <div className="font-bold text-sm text-slate-900">{c.name}</div>
                      <span className="text-[10px] font-bold text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded border border-emerald-200">
                        {c.status}
                      </span>
                    </div>
                    <div className="text-xs text-slate-500">Contact: {c.contact}</div>
                    <div className="pt-2 flex items-center gap-4 text-[11px] font-semibold text-slate-600 border-t border-slate-100">
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
