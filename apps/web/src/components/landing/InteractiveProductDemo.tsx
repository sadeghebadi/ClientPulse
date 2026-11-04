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
    },
    {
      id: "web",
      name: t.landingDemoTarget2,
      url: "https://store.internal",
      type: "http",
      normalLatency: 24,
      environment: "Production",
    },
    {
      id: "backup",
      name: t.landingDemoTarget3,
      url: "ping.clientpulse.internal/v1/ping/db-bk-774",
      type: "heartbeat",
      normalLatency: 12,
      environment: "Staging",
    },
  ];

  const [selectedTargetId, setSelectedTargetId] = useState<string>("api");
  const [isSimulatedDown, setIsSimulatedDown] = useState<boolean>(false);
  const [incidentAcknowledged, setIncidentAcknowledged] = useState<boolean>(false);
  const [timeframe, setTimeframe] = useState<"1h" | "24h" | "7d">("1h");
  const [showNotificationToast, setShowNotificationToast] = useState<boolean>(false);

  const selectedTarget = targets.find((tg) => tg.id === selectedTargetId) || targets[0];

  const handleSimulateOutage = () => {
    setIsSimulatedDown(true);
    setIncidentAcknowledged(false);
    setShowNotificationToast(true);
    setTimeout(() => {
      setShowNotificationToast(false);
    }, 6000);
  };

  const handleSimulateRecovery = () => {
    setIsSimulatedDown(false);
    setIncidentAcknowledged(false);
    setShowNotificationToast(false);
  };

  // Sparkline data generation based on state and timeframe
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
    <section id="demo" className="py-20 sm:py-28 bg-[#070b14] text-slate-100 relative overflow-hidden border-b border-slate-800">
      {/* Background Accent Gradients */}
      <div className="absolute top-1/2 right-1/4 w-96 h-96 bg-blue-600/10 rounded-full blur-[100px] pointer-events-none" />
      <div className="absolute bottom-10 left-1/4 w-96 h-96 bg-violet-600/10 rounded-full blur-[100px] pointer-events-none" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Section Header */}
        <div className="max-w-3xl mx-auto text-center space-y-4 mb-12 sm:mb-16">
          <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full text-xs font-semibold text-emerald-400 bg-emerald-950/60 border border-emerald-800/60 shadow-lg shadow-emerald-950/30">
            <Zap className="w-3.5 h-3.5 text-emerald-400" />
            <span>{t.landingDemoBadge}</span>
          </div>

          <h2 className="text-3xl sm:text-5xl font-black text-white tracking-tight">
            {t.landingDemoTitle}
          </h2>

          <p className="text-base sm:text-lg text-slate-400 leading-relaxed font-normal">
            {t.landingDemoSubtitle}
          </p>

          {/* Illustrative Disclaimer Pill */}
          <div className="pt-1">
            <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-[11px] font-mono font-medium text-amber-300 bg-amber-950/40 border border-amber-800/60">
              <Info className="w-3.5 h-3.5 shrink-0" />
              <span>{t.landingDemoDisclaimer}</span>
            </span>
          </div>
        </div>

        {/* Interactive Sandbox Dashboard Mockup */}
        <div className="max-w-5xl mx-auto rounded-2xl sm:rounded-3xl border border-slate-800 bg-[#0c1222] shadow-2xl overflow-hidden ring-1 ring-slate-700/50">
          {/* Top Control Bar: Target Selector & Downtime Simulator Buttons */}
          <div className="p-4 sm:p-5 bg-slate-900/90 border-b border-slate-800 flex flex-col md:flex-row items-center justify-between gap-4">
            {/* Target Selectors */}
            <div className="flex flex-wrap items-center gap-2 w-full md:w-auto">
              <span className="text-xs font-bold text-slate-400 uppercase tracking-wider mr-1">
                {isRTL ? "انتخاب اندپوینت:" : "Target:"}
              </span>
              {targets.map((tg) => {
                const isSelected = selectedTargetId === tg.id;
                return (
                  <button
                    key={tg.id}
                    type="button"
                    onClick={() => {
                      setSelectedTargetId(tg.id);
                      setIsSimulatedDown(false);
                      setIncidentAcknowledged(false);
                    }}
                    className={`px-3 py-1.5 rounded-xl text-xs font-bold transition-all ${
                      isSelected
                        ? "bg-blue-600 text-white shadow-md shadow-blue-600/30 ring-1 ring-blue-400"
                        : "bg-slate-800 text-slate-300 hover:bg-slate-700 border border-slate-700"
                    }`}
                  >
                    {tg.name}
                  </button>
                );
              })}
            </div>

            {/* Simulation Action Buttons */}
            <div className="flex items-center gap-2.5 w-full md:w-auto justify-end">
              {!isSimulatedDown ? (
                <button
                  type="button"
                  onClick={handleSimulateOutage}
                  className="px-4 py-2 rounded-xl text-xs font-bold bg-rose-600 hover:bg-rose-500 text-white flex items-center gap-2 shadow-lg shadow-rose-900/40 transition-all hover:scale-105 active:scale-95"
                >
                  <AlertTriangle className="w-3.5 h-3.5" />
                  <span>{t.landingDemoSimulateDown}</span>
                </button>
              ) : (
                <button
                  type="button"
                  onClick={handleSimulateRecovery}
                  className="px-4 py-2 rounded-xl text-xs font-bold bg-emerald-600 hover:bg-emerald-500 text-white flex items-center gap-2 shadow-lg shadow-emerald-900/40 transition-all hover:scale-105 active:scale-95"
                >
                  <RefreshCw className="w-3.5 h-3.5" />
                  <span>{t.landingDemoSimulateRecover}</span>
                </button>
              )}
            </div>
          </div>

          {/* Interactive Target Canvas */}
          <div className="p-5 sm:p-7 space-y-6">
            {/* Real-time Status Card with Tilt Effect */}
            <TiltCard maxTilt={4} sheen={true}>
              <div
                className={`p-5 rounded-2xl border transition-all duration-500 ${
                  isSimulatedDown
                    ? "bg-rose-950/30 border-rose-600/60 shadow-[0_0_30px_rgba(225,29,72,0.15)]"
                    : "bg-slate-900/90 border-slate-700/80 shadow-xl"
                }`}
              >
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                  <div className="flex items-start sm:items-center gap-3.5">
                    <div
                      className={`w-10 h-10 rounded-xl flex items-center justify-center shrink-0 transition-colors duration-500 ${
                        isSimulatedDown
                          ? "bg-rose-500/20 text-rose-400"
                          : "bg-emerald-500/20 text-emerald-400"
                      }`}
                    >
                      {isSimulatedDown ? (
                        <AlertTriangle className="w-5 h-5 animate-bounce" />
                      ) : (
                        <CheckCircle2 className="w-5 h-5" />
                      )}
                    </div>

                    <div>
                      <div className="flex items-center gap-2">
                        <h4 className="text-base sm:text-lg font-bold text-white">
                          {selectedTarget.name}
                        </h4>
                        <span className="text-[10px] px-2 py-0.5 rounded bg-slate-800 text-slate-300 font-mono">
                          {selectedTarget.environment}
                        </span>
                      </div>
                      <div className="text-xs text-slate-400 font-mono truncate max-w-md">
                        {selectedTarget.url}
                      </div>
                    </div>
                  </div>

                  {/* Status Indicator Badge */}
                  <div className="flex items-center gap-3">
                    <div
                      className={`px-3.5 py-1.5 rounded-full text-xs font-bold border transition-colors duration-300 ${
                        isSimulatedDown
                          ? "bg-rose-500/20 text-rose-400 border-rose-500/40"
                          : "bg-emerald-500/15 text-emerald-400 border-emerald-500/30"
                      }`}
                    >
                      {isSimulatedDown
                        ? (isRTL ? "خطای ۵۰۳ · قطعی تاییدشده" : "503 Unavailable · Target Down")
                        : (isRTL ? "پایدار · ۲۰۰ OK" : "Operational · 200 OK")}
                    </div>
                  </div>
                </div>

                {/* KPI Metrics Strip */}
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mt-6 pt-5 border-t border-slate-800">
                  <div className="p-3 rounded-xl bg-slate-950/60 border border-slate-800/80">
                    <div className="text-[11px] text-slate-400 font-medium">
                      {isRTL ? "تاخیر آخرین بررسی" : "Latest Latency"}
                    </div>
                    <div
                      className={`text-lg font-bold font-mono ${
                        isSimulatedDown ? "text-rose-400" : "text-blue-400"
                      }`}
                    >
                      {isSimulatedDown ? "TIMEOUT" : `${selectedTarget.normalLatency} ms`}
                    </div>
                  </div>

                  <div className="p-3 rounded-xl bg-slate-950/60 border border-slate-800/80">
                    <div className="text-[11px] text-slate-400 font-medium">
                      {isRTL ? "آپ‌تایم ۲۴ ساعته" : "24h Uptime"}
                    </div>
                    <div
                      className={`text-lg font-bold font-mono ${
                        isSimulatedDown ? "text-amber-400" : "text-emerald-400"
                      }`}
                    >
                      {isSimulatedDown ? "98.82%" : "100.00%"}
                    </div>
                  </div>

                  <div className="p-3 rounded-xl bg-slate-950/60 border border-slate-800/80">
                    <div className="text-[11px] text-slate-400 font-medium">
                      {isRTL ? "بازه پایش" : "Check Interval"}
                    </div>
                    <div className="text-lg font-bold font-mono text-white">60s</div>
                  </div>

                  <div className="p-3 rounded-xl bg-slate-950/60 border border-slate-800/80">
                    <div className="text-[11px] text-slate-400 font-medium">
                      {isRTL ? "رخدادهای فعال" : "Active Incidents"}
                    </div>
                    <div
                      className={`text-lg font-bold font-mono ${
                        isSimulatedDown ? "text-rose-400 font-black" : "text-slate-400"
                      }`}
                    >
                      {isSimulatedDown ? "1 Open" : "0"}
                    </div>
                  </div>
                </div>
              </div>
            </TiltCard>

            {/* Latency History Waveform Chart */}
            <div className="rounded-2xl border border-slate-800 bg-slate-900/60 p-4 sm:p-5">
              <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 mb-4">
                <div className="flex items-center gap-2">
                  <Activity className="w-4 h-4 text-blue-400" />
                  <span className="text-xs sm:text-sm font-bold text-white">
                    {isRTL ? "روند پاسخ و تله‌متری تاخیر" : "Response Time & Latency Waveform"}
                  </span>
                </div>

                {/* Timeframe Controls */}
                <div className="flex items-center gap-1.5 p-1 rounded-xl bg-slate-950 border border-slate-800 text-xs">
                  {(["1h", "24h", "7d"] as const).map((tf) => (
                    <button
                      key={tf}
                      type="button"
                      onClick={() => setTimeframe(tf)}
                      className={`px-2.5 py-1 rounded-lg font-mono font-semibold transition-colors ${
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

              {/* Sparkline Visualization */}
              <div className="h-28 w-full flex items-end gap-1.5 sm:gap-2 px-2 pt-4">
                {sparkline.map((val, idx) => {
                  const heightPercent = val === 0 ? 8 : Math.min((val / 1300) * 100, 100);
                  const isZero = val === 0;
                  return (
                    <div
                      key={idx}
                      className="flex-1 flex flex-col items-center gap-1.5 group relative"
                    >
                      <div
                        className={`w-full rounded-t-sm transition-all duration-500 ${
                          isZero
                            ? "bg-rose-500/80 shadow-[0_0_8px_rgba(244,63,94,0.6)]"
                            : val > 200
                            ? "bg-amber-400/80"
                            : "bg-blue-500/70 hover:bg-blue-400"
                        }`}
                        style={{ height: `${heightPercent}%` }}
                      />
                      <span className="text-[9px] font-mono text-slate-500 hidden sm:block">
                        {isZero ? "FAIL" : `${val}ms`}
                      </span>
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Simulated Live Incident Drawer (Appears when outage is simulated) */}
            {isSimulatedDown && (
              <div className="rounded-2xl border border-rose-500/50 bg-rose-950/20 p-5 space-y-4 animate-fadeIn">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                  <div className="flex items-center gap-3">
                    <span className="w-3 h-3 rounded-full bg-rose-500 animate-ping inline-block" />
                    <div>
                      <div className="text-sm font-bold text-white flex items-center gap-2">
                        <span>Incident #INC-912</span>
                        <span className="px-2 py-0.5 rounded bg-rose-500/20 text-rose-300 text-[10px] font-mono border border-rose-500/30">
                          {incidentAcknowledged ? "Acknowledged" : "Open"}
                        </span>
                      </div>
                      <div className="text-xs text-rose-300/80">
                        {isRTL
                          ? "خطای ۵۰۳ عدم پاسخگویی سرور · شروع در ۱۴:۰۲:۴۱ UTC"
                          : "HTTP 503 Backend Gateway Timeout · Started at 14:02:41 UTC"}
                      </div>
                    </div>
                  </div>

                  {/* Incident Operator Actions */}
                  <div className="flex items-center gap-2">
                    {!incidentAcknowledged ? (
                      <button
                        type="button"
                        onClick={() => setIncidentAcknowledged(true)}
                        className="px-3.5 py-1.5 rounded-xl text-xs font-bold bg-amber-500 hover:bg-amber-400 text-slate-950 transition-colors shadow"
                      >
                        {isRTL ? "تایید دریافت رخداد (Ack)" : "Acknowledge Incident"}
                      </button>
                    ) : (
                      <span className="px-3 py-1.5 rounded-xl text-xs font-bold bg-slate-800 text-emerald-400 border border-emerald-500/30 flex items-center gap-1.5">
                        <CheckCircle2 className="w-3.5 h-3.5" />
                        <span>{isRTL ? "تایید شد توسط اپراتور" : "Acknowledged by Operator"}</span>
                      </span>
                    )}
                  </div>
                </div>

                {/* Audit Timeline Notes */}
                <div className="text-xs font-mono text-slate-400 space-y-1 bg-slate-950/70 p-3 rounded-xl border border-slate-800/80">
                  <div className="flex items-center gap-2 text-rose-400">
                    <span>[14:02:10]</span>
                    <span>1st failure: connection timeout (5000ms exceeded)</span>
                  </div>
                  <div className="flex items-center gap-2 text-rose-400">
                    <span>[14:02:40]</span>
                    <span>2nd failure: HTTP 503 verified by worker daemon</span>
                  </div>
                  <div className="flex items-center gap-2 text-blue-300">
                    <span>[14:02:41]</span>
                    <span>Incident created atomically & notifications dispatched</span>
                  </div>
                  {incidentAcknowledged && (
                    <div className="flex items-center gap-2 text-amber-300">
                      <span>[14:03:02]</span>
                      <span>Operator marked incident as acknowledged</span>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </section>
  );
}

