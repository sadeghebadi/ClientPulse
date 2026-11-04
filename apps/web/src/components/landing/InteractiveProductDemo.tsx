"use client";

import React, { useState } from "react";
import {
  Activity,
  Radio,
  AlertTriangle,
  CheckCircle2,
  Clock,
  ShieldCheck,
  Server,
  Zap,
  Bell,
  RefreshCw,
  Terminal,
  ExternalLink,
  Info,
  ChevronDown,
  ChevronUp,
  UserCheck,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { TiltCard } from "./motion/TiltCard";

interface DemoTarget {
  id: string;
  name: string;
  url: string;
  type: "http" | "heartbeat";
  normalLatency: number;
  environment: "Production" | "Staging";
  checkInterval: string;
}

export function InteractiveProductDemo() {
  const { t, isRTL } = useI18n();

  const targets: DemoTarget[] = [
    {
      id: "api",
      name: t.landingDemoTarget1,
      url: "https://api.store.internal/v1/checkout",
      type: "http",
      normalLatency: 38,
      environment: "Production",
      checkInterval: "60s",
    },
    {
      id: "web",
      name: t.landingDemoTarget2,
      url: "https://store.internal",
      type: "http",
      normalLatency: 24,
      environment: "Production",
      checkInterval: "60s",
    },
    {
      id: "backup",
      name: t.landingDemoTarget3,
      url: "ping.clientpulse.internal/v1/ping/db-bk-774",
      type: "heartbeat",
      normalLatency: 12,
      environment: "Staging",
      checkInterval: "24h",
    },
  ];

  const [selectedTargetId, setSelectedTargetId] = useState<string>("api");
  const [isSimulatedDown, setIsSimulatedDown] = useState<boolean>(false);
  const [incidentAcknowledged, setIncidentAcknowledged] = useState<boolean>(false);
  const [timeframe, setTimeframe] = useState<"1h" | "24h" | "7d">("1h");
  const [showNotificationToast, setShowNotificationToast] = useState<boolean>(false);
  const [incidentDrawerOpen, setIncidentDrawerOpen] = useState<boolean>(true);

  const selectedTarget = targets.find((tg) => tg.id === selectedTargetId) || targets[0];

  const handleSimulateOutage = () => {
    setIsSimulatedDown(true);
    setIncidentAcknowledged(false);
    setShowNotificationToast(true);
    setIncidentDrawerOpen(true);
    setTimeout(() => {
      setShowNotificationToast(false);
    }, 7000);
  };

  const handleSimulateRecovery = () => {
    setIsSimulatedDown(false);
    setIncidentAcknowledged(false);
    setShowNotificationToast(false);
  };

  const handleAcknowledge = () => {
    setIncidentAcknowledged(true);
  };

  // Sparkline data based on active simulation state
  const getSparklineData = () => {
    if (isSimulatedDown) {
      return [38, 42, 39, 45, 120, 480, 1250, 0, 0, 0, 0];
    }
    if (timeframe === "1h") {
      return [36, 38, 37, 39, 36, 42, 38, 37, 39, 38, 37];
    }
    if (timeframe === "24h") {
      return [34, 35, 38, 45, 38, 36, 35, 37, 40, 38, 37];
    }
    return [38, 38, 37, 39, 36, 36, 38, 38, 37, 37, 38];
  };

  const sparkline = getSparklineData();

  return (
    <section
      id="demo"
      className="py-20 sm:py-32 bg-[#050811] text-slate-100 relative overflow-hidden border-b border-slate-800 scroll-mt-20"
    >
      {/* Anchor for playground navigation */}
      <span id="playground" className="sr-only" />

      {/* Background Accent Gradients */}
      <div className="absolute top-1/2 right-1/4 w-[500px] h-[500px] bg-blue-600/10 rounded-full blur-[140px] pointer-events-none" />
      <div className="absolute bottom-10 left-1/4 w-[500px] h-[500px] bg-violet-600/10 rounded-full blur-[140px] pointer-events-none" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Section Header */}
        <div className="max-w-3xl mx-auto text-center space-y-4 mb-12 sm:mb-16">
          <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full text-xs font-semibold text-emerald-400 bg-emerald-950/60 border border-emerald-800/60 shadow-lg shadow-emerald-950/30 backdrop-blur-md">
            <Zap className="w-3.5 h-3.5 text-emerald-400" />
            <span>{t.landingDemoBadge}</span>
          </div>

          <h2 className="text-3xl sm:text-5xl font-black text-white tracking-tight leading-tight">
            {t.landingDemoTitle}
          </h2>

          <p className="text-base sm:text-lg text-slate-300 leading-relaxed font-normal">
            {t.landingDemoSubtitle}
          </p>

          {/* Prominent Simulated Data Disclaimer */}
          <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-lg bg-slate-900/80 border border-slate-800 text-[11px] text-slate-400 shadow-sm">
            <Info className="w-3.5 h-3.5 text-blue-400 shrink-0" />
            <span>{t.landingDemoDisclaimer}</span>
          </div>
        </div>

        {/* ======================================================== */}
        {/* Interactive Sandbox Cockpit                              */}
        {/* ======================================================== */}
        <div className="max-w-5xl mx-auto bg-[#090e1a] rounded-3xl border border-slate-800 shadow-2xl overflow-hidden ring-1 ring-white/10">
          {/* Top Target Switcher Bar */}
          <div className="p-4 sm:p-6 bg-slate-950/80 border-b border-slate-800 flex flex-col md:flex-row md:items-center justify-between gap-4">
            <div className="space-y-1">
              <span className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider block">
                {t.landingDemoTargets}
              </span>
              <div className="flex flex-wrap items-center gap-2">
                {targets.map((tgt) => (
                  <button
                    key={tgt.id}
                    type="button"
                    onClick={() => {
                      setSelectedTargetId(tgt.id);
                      setIsSimulatedDown(false);
                      setIncidentAcknowledged(false);
                    }}
                    className={`px-3.5 py-2 rounded-xl text-xs font-bold transition-all flex items-center gap-2 ${
                      selectedTargetId === tgt.id
                        ? "bg-blue-600 text-white shadow-lg shadow-blue-600/30 ring-1 ring-blue-400"
                        : "bg-slate-900 text-slate-400 hover:text-white hover:bg-slate-800 border border-slate-800"
                    }`}
                  >
                    {tgt.type === "http" ? (
                      <Activity className="w-3.5 h-3.5 text-blue-300" />
                    ) : (
                      <Radio className="w-3.5 h-3.5 text-purple-300" />
                    )}
                    <span>{tgt.name}</span>
                  </button>
                ))}
              </div>
            </div>

            {/* Simulation Action Buttons */}
            <div className="flex items-center gap-2.5 shrink-0">
              {!isSimulatedDown ? (
                <button
                  type="button"
                  onClick={handleSimulateOutage}
                  className="px-4 py-2.5 rounded-xl text-xs font-bold bg-rose-600/90 hover:bg-rose-500 text-white border border-rose-500/50 shadow-lg shadow-rose-600/20 transition-all flex items-center gap-2"
                >
                  <AlertTriangle className="w-3.5 h-3.5" />
                  <span>{t.landingDemoSimulateDown}</span>
                </button>
              ) : (
                <button
                  type="button"
                  onClick={handleSimulateRecovery}
                  className="px-4 py-2.5 rounded-xl text-xs font-bold bg-emerald-600/90 hover:bg-emerald-500 text-white border border-emerald-500/50 shadow-lg shadow-emerald-600/20 transition-all flex items-center gap-2"
                >
                  <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                  <span>{t.landingDemoSimulateRecover}</span>
                </button>
              )}
            </div>
          </div>

          {/* Main Sandbox Telemetry Display */}
          <div className="p-6 sm:p-8 space-y-6">
            {/* Target Live Header */}
            <div className="p-4 sm:p-5 rounded-2xl bg-slate-900/90 border border-slate-800/80 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
              <div className="space-y-1">
                <div className="flex items-center gap-2 text-xs font-mono text-slate-400">
                  <span className="px-2 py-0.5 rounded bg-slate-800 text-[10px] text-slate-300 font-semibold uppercase">
                    {selectedTarget.environment}
                  </span>
                  <span className="ltr-text font-bold text-slate-200 truncate max-w-xs sm:max-w-md">
                    {selectedTarget.url}
                  </span>
                </div>
                <div className="text-base sm:text-lg font-bold text-white">
                  {selectedTarget.name}
                </div>
              </div>

              {/* Status Badge */}
              <div className="flex items-center gap-3 shrink-0">
                <div
                  className={`px-3.5 py-1.5 rounded-xl text-xs font-bold border flex items-center gap-2 ${
                    isSimulatedDown
                      ? "bg-rose-950/80 border-rose-800 text-rose-400"
                      : "bg-emerald-950/80 border-emerald-800 text-emerald-400"
                  }`}
                >
                  <span
                    className={`w-2.5 h-2.5 rounded-full ${
                      isSimulatedDown ? "bg-rose-500 animate-ping" : "bg-emerald-400 animate-pulse"
                    }`}
                  />
                  <span>
                    {isSimulatedDown
                      ? isRTL ? "قطعی ۵۰۳ · خارج از دسترس" : "503 Unavailable · Target Down"
                      : isRTL ? "عملیاتی · پاسخ ۲۰۰" : "Operational · 200 OK"}
                  </span>
                </div>
              </div>
            </div>

            {/* KPI Telemetry Gauges */}
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
              <div className="p-4 rounded-xl bg-slate-900/80 border border-slate-800 space-y-1">
                <div className="text-[11px] font-semibold text-slate-400">
                  {isRTL ? "سرعت پاسخ لحظه‌ای" : "Current Latency"}
                </div>
                <div
                  className={`text-2xl font-black font-mono ${
                    isSimulatedDown ? "text-rose-400" : "text-blue-400"
                  }`}
                >
                  {isSimulatedDown ? "TIMEOUT" : `${selectedTarget.normalLatency} ms`}
                </div>
                <div className="text-[10px] text-slate-500 font-medium">
                  {isSimulatedDown
                    ? isRTL ? "پاسخ دریافت نشد" : "Exceeded 5000ms threshold"
                    : isRTL ? "سریع و نرمال" : "Fast HTTP response"}
                </div>
              </div>

              <div className="p-4 rounded-xl bg-slate-900/80 border border-slate-800 space-y-1">
                <div className="text-[11px] font-semibold text-slate-400">
                  {isRTL ? "نرخ دسترسی (Uptime)" : "Availability"}
                </div>
                <div
                  className={`text-2xl font-black font-mono ${
                    isSimulatedDown ? "text-amber-400" : "text-emerald-400"
                  }`}
                >
                  {isSimulatedDown ? "98.82%" : "99.99%"}
                </div>
                <div className="text-[10px] text-slate-500 font-medium">
                  {isSimulatedDown
                    ? isRTL ? "کاهش به دلیل قطعی" : "Impacted by downtime"
                    : isRTL ? "عالی بدون وقفه" : "No interruptions recorded"}
                </div>
              </div>

              <div className="p-4 rounded-xl bg-slate-900/80 border border-slate-800 space-y-1">
                <div className="text-[11px] font-semibold text-slate-400">
                  {isRTL ? "رخدادهای باز" : "Active Incidents"}
                </div>
                <div
                  className={`text-2xl font-black font-mono ${
                    isSimulatedDown ? "text-rose-400" : "text-slate-300"
                  }`}
                >
                  {isSimulatedDown ? "1 Open" : "0 Active"}
                </div>
                <div className="text-[10px] text-slate-500 font-medium">
                  {isSimulatedDown
                    ? isRTL ? "رخداد در دست بررسی" : "Incident #INC-912 open"
                    : isRTL ? "بدون رخداد فعال" : "All services operational"}
                </div>
              </div>

              <div className="p-4 rounded-xl bg-slate-900/80 border border-slate-800 space-y-1">
                <div className="text-[11px] font-semibold text-slate-400">
                  {isRTL ? "دوره بررسی" : "Check Interval"}
                </div>
                <div className="text-2xl font-black font-mono text-slate-200">
                  {selectedTarget.checkInterval}
                </div>
                <div className="text-[10px] text-slate-500 font-medium">
                  {isRTL ? "زمان‌بند صفی اتوماتیک" : "Automated Queue Scheduler"}
                </div>
              </div>
            </div>

            {/* Interactive Latency Waveform Sparkline with Timeframe Toggles */}
            <div className="p-5 rounded-2xl bg-slate-950/80 border border-slate-800/80 space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Activity className="w-4 h-4 text-blue-400" />
                  <span className="text-xs font-bold text-white uppercase tracking-wider">
                    {t.landingDemoLiveMetrics}
                  </span>
                </div>

                {/* Timeframe selector */}
                <div className="flex items-center gap-1 p-1 rounded-lg bg-slate-900 border border-slate-800 text-[11px]">
                  {(["1h", "24h", "7d"] as const).map((tf) => (
                    <button
                      key={tf}
                      type="button"
                      onClick={() => setTimeframe(tf)}
                      className={`px-2.5 py-1 rounded-md font-mono font-bold transition-colors ${
                        timeframe === tf
                          ? "bg-blue-600 text-white"
                          : "text-slate-400 hover:text-white"
                      }`}
                    >
                      {tf}
                    </button>
                  ))}
                </div>
              </div>

              {/* Sparkline Visual */}
              <div className="h-28 flex items-end justify-between gap-1.5 px-2">
                {sparkline.map((val, idx) => {
                  const maxH = 1300;
                  const pct = val === 0 ? 6 : Math.max(12, Math.min(100, (val / maxH) * 100));
                  return (
                    <div key={idx} className="flex-1 flex flex-col items-center justify-end h-full">
                      <div
                        style={{ height: `${pct}%` }}
                        className={`w-full rounded-t transition-all duration-300 ${
                          val === 0
                            ? "bg-rose-600/40 border-t border-rose-500"
                            : isSimulatedDown
                            ? "bg-rose-500/80"
                            : "bg-blue-500/70 hover:bg-blue-400"
                        }`}
                      />
                    </div>
                  );
                })}
              </div>

              <div className="flex items-center justify-between text-[11px] text-slate-500 pt-2 border-t border-slate-800/80 font-mono">
                <span>Timeline Start</span>
                <span>Current Probe Window</span>
              </div>
            </div>

            {/* ======================================================== */}
            {/* Simulated Live Incident Drawer (Appears on Outage)       */}
            {/* ======================================================== */}
            {isSimulatedDown && (
              <div className="p-5 rounded-2xl bg-rose-950/30 border border-rose-500/40 shadow-xl space-y-4 animate-in fade-in slide-in-from-top-4 duration-300">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2 text-rose-400 font-bold text-sm">
                    <AlertTriangle className="w-4 h-4 shrink-0" />
                    <span>Incident #INC-912 · Target Down</span>
                  </div>

                  <div className="flex items-center gap-2">
                    {incidentAcknowledged ? (
                      <span className="px-3 py-1 rounded-full text-xs font-bold bg-amber-500/20 text-amber-300 border border-amber-500/40 flex items-center gap-1.5">
                        <UserCheck className="w-3.5 h-3.5" />
                        <span>Acknowledged by Operator</span>
                      </span>
                    ) : (
                      <button
                        type="button"
                        onClick={handleAcknowledge}
                        className="px-3.5 py-1.5 rounded-lg text-xs font-bold text-white bg-amber-600 hover:bg-amber-500 transition-colors shadow-md"
                      >
                        Acknowledge Incident
                      </button>
                    )}
                  </div>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs">
                  <div className="p-3 rounded-xl bg-slate-950/60 border border-slate-800 space-y-1">
                    <span className="text-[10px] text-slate-400 uppercase font-semibold">Diagnosis</span>
                    <p className="font-mono text-slate-200">HTTP 503 · Service Unavailable</p>
                  </div>
                  <div className="p-3 rounded-xl bg-slate-950/60 border border-slate-800 space-y-1">
                    <span className="text-[10px] text-slate-400 uppercase font-semibold">Consecutive Failures</span>
                    <p className="font-mono text-slate-200">3 probes failed consecutively</p>
                  </div>
                  <div className="p-3 rounded-xl bg-slate-950/60 border border-slate-800 space-y-1">
                    <span className="text-[10px] text-slate-400 uppercase font-semibold">Triggered At</span>
                    <p className="font-mono text-slate-200">14:02:18 UTC (Probed)</p>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </section>
  );
}
