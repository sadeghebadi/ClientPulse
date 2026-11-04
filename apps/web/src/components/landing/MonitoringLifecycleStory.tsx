"use client";

import React, { useState, useEffect, useRef } from "react";
import {
  Activity,
  AlertTriangle,
  XCircle,
  Bell,
  CheckCircle2,
  Clock,
  ShieldCheck,
  Zap,
  ArrowRight,
  ArrowLeft,
  ChevronRight,
  Send,
  MessageSquare,
  Mail,
  Webhook,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { useReducedMotion } from "./motion/useReducedMotion";
import gsap from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";

export function MonitoringLifecycleStory() {
  const { t, isRTL } = useI18n();
  const [activeScene, setActiveScene] = useState<number>(0);
  const containerRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLDivElement>(null);
  const reducedMotion = useReducedMotion();

  const scenes = [
    {
      id: 0,
      number: "01",
      title: t.storyScene1Title,
      tag: t.storyScene1Tag,
      desc: t.storyScene1Desc,
      metric: t.storyScene1Metric,
      status: "operational",
      statusText: isRTL ? "عملیاتی · ۱۰۰٪ پایدار" : "Operational · 100% Healthy",
      statusColor: "emerald",
      latency: "34 ms",
      httpCode: "200 OK",
      uptime: "99.99%",
      graphData: [32, 35, 34, 38, 33, 36, 34, 35, 33, 34],
    },
    {
      id: 1,
      number: "02",
      title: t.storyScene2Title,
      tag: t.storyScene2Tag,
      desc: t.storyScene2Desc,
      metric: t.storyScene2Metric,
      status: "degraded",
      statusText: isRTL ? "هشدار · افزایش تاخیر" : "Warning · Latency Spike",
      statusColor: "amber",
      latency: "1,280 ms",
      httpCode: "200 OK (Slow)",
      uptime: "99.95%",
      graphData: [34, 45, 120, 280, 540, 890, 1150, 1280, 1220, 1280],
    },
    {
      id: 2,
      number: "03",
      title: t.storyScene3Title,
      tag: t.storyScene3Tag,
      desc: t.storyScene3Desc,
      metric: t.storyScene3Metric,
      status: "down",
      statusText: isRTL ? "قطعی بحرانی · رخداد باز" : "Critical Outage · Incident Open",
      statusColor: "rose",
      latency: "TIMEOUT",
      httpCode: "502 Bad Gateway",
      uptime: "98.40%",
      graphData: [1280, 1340, 0, 0, 0, 0, 0, 0, 0, 0],
    },
    {
      id: 3,
      number: "04",
      title: t.storyScene4Title,
      tag: t.storyScene4Tag,
      desc: t.storyScene4Desc,
      metric: t.storyScene4Metric,
      status: "alerting",
      statusText: isRTL ? "ارسال فوری هشدارها" : "Alert Dispatched (< 3s)",
      statusColor: "violet",
      latency: "TIMEOUT",
      httpCode: "502 Bad Gateway",
      uptime: "98.40%",
      graphData: [0, 0, 0, 0, 0, 0, 0, 0, 0, 0],
    },
    {
      id: 4,
      number: "05",
      title: t.storyScene5Title,
      tag: t.storyScene5Tag,
      desc: t.storyScene5Desc,
      metric: t.storyScene5Metric,
      status: "recovered",
      statusText: isRTL ? "بازیابی کامل و پایدار" : "Recovered · Fully Operational",
      statusColor: "emerald",
      latency: "38 ms",
      httpCode: "200 OK",
      uptime: "99.98%",
      graphData: [0, 0, 240, 80, 42, 38, 36, 38, 37, 38],
    },
  ];

  // GSAP ScrollTrigger Integration
  useEffect(() => {
    if (typeof window === "undefined" || reducedMotion) return;

    gsap.registerPlugin(ScrollTrigger);

    const triggerEl = triggerRef.current;
    if (!triggerEl) return;

    // Create a pinned scroll trigger that drives the 5 scenes smoothly as user scrolls
    const st = ScrollTrigger.create({
      trigger: triggerEl,
      start: "top 20%",
      end: "+=2200",
      pin: false, // gentle scroll scrubber without locking viewport
      onUpdate: (self) => {
        const progress = self.progress;
        const newScene = Math.min(
          Math.floor(progress * scenes.length),
          scenes.length - 1
        );
        setActiveScene(newScene);
      },
    });

    return () => {
      st.kill();
    };
  }, [reducedMotion, scenes.length]);

  const current = scenes[activeScene];

  // Generate SVG path for the latency graph
  const renderGraphPath = (data: number[]) => {
    const width = 500;
    const height = 140;
    const maxVal = 1400;

    const points = data.map((val, idx) => {
      const x = (idx / (data.length - 1)) * width;
      // Invert y (0 is at top)
      const y = height - (val / maxVal) * (height - 20) - 10;
      return `${x},${y}`;
    });

    return `M ${points.join(" L ")}`;
  };

  return (
    <section
      id="lifecycle"
      ref={triggerRef}
      className="relative py-20 sm:py-28 lg:py-36 bg-[#080d1a] text-slate-100 overflow-hidden border-t border-b border-slate-800/80"
    >
      {/* Background Cinematic Electric Glow */}
      <div className="absolute top-1/4 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[700px] h-[400px] bg-gradient-to-tr from-blue-600/15 via-violet-600/10 to-transparent blur-[120px] pointer-events-none rounded-full" />
      <div className="absolute inset-0 bg-[linear-gradient(to_right,#1e293b12_1px,transparent_1px),linear-gradient(to_bottom,#1e293b12_1px,transparent_1px)] bg-[size:32px_32px] pointer-events-none" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10" ref={containerRef}>
        {/* Section Header */}
        <div className="max-w-3xl mx-auto text-center space-y-4 mb-12 sm:mb-16">
          <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full text-xs font-semibold text-blue-400 bg-blue-950/60 border border-blue-800/60 shadow-lg shadow-blue-950/40">
            <Activity className="w-3.5 h-3.5 text-blue-400" />
            <span>{t.landingStoryBadge}</span>
          </div>

          <h2 className="text-3xl sm:text-5xl font-black text-white tracking-tight leading-tight">
            {t.landingStoryTitle}
          </h2>

          <p className="text-base sm:text-lg text-slate-400 leading-relaxed max-w-2xl mx-auto font-normal">
            {t.landingStorySubtitle}
          </p>
        </div>

        {/* Interactive Scene Navigation Stepper */}
        <div className="flex items-center justify-start sm:justify-center overflow-x-auto pb-4 mb-10 gap-2 sm:gap-3 no-scrollbar" role="tablist">
          {scenes.map((s, idx) => {
            const isActive = activeScene === idx;
            return (
              <button
                key={s.id}
                role="tab"
                aria-selected={isActive}
                type="button"
                onClick={() => setActiveScene(idx)}
                className={`group flex items-center gap-2.5 px-4 py-2.5 rounded-xl text-xs sm:text-sm font-semibold transition-all duration-300 shrink-0 border ${
                  isActive
                    ? "bg-blue-600/20 text-white border-blue-500/80 shadow-lg shadow-blue-600/20 ring-1 ring-blue-500/50"
                    : "bg-slate-900/60 text-slate-400 hover:text-slate-200 hover:bg-slate-800/80 border-slate-800/90"
                }`}
              >
                <span
                  className={`w-5 h-5 rounded-md flex items-center justify-center text-[10px] font-mono font-bold ${
                    isActive ? "bg-blue-500 text-white" : "bg-slate-800 text-slate-400"
                  }`}
                >
                  {s.number}
                </span>
                <span className="whitespace-nowrap">{s.tag}</span>
              </button>
            );
          })}
        </div>

        {/* Choreographed Story Stage Grid */}
        <div className="max-w-5xl mx-auto grid grid-cols-1 lg:grid-cols-12 gap-8 items-center">
          {/* Left Column: Visual Story Canvas */}
          <div className="lg:col-span-7">
            <div className="rounded-2xl sm:rounded-3xl border border-slate-800 bg-[#0b1120]/90 backdrop-blur-xl p-5 sm:p-7 shadow-2xl relative overflow-hidden ring-1 ring-slate-700/40">
              {/* Dynamic Status Bar Accent */}
              <div
                className={`absolute top-0 left-0 right-0 h-1 transition-colors duration-500 ${
                  current.statusColor === "emerald"
                    ? "bg-emerald-500 shadow-[0_0_12px_rgba(16,185,129,0.8)]"
                    : current.statusColor === "amber"
                    ? "bg-amber-500 shadow-[0_0_12px_rgba(245,158,11,0.8)]"
                    : current.statusColor === "rose"
                    ? "bg-rose-500 shadow-[0_0_12px_rgba(239,68,68,0.8)]"
                    : "bg-violet-500 shadow-[0_0_12px_rgba(139,92,246,0.8)]"
                }`}
              />

              {/* Target Header Header */}
              <div className="flex items-center justify-between pb-4 border-b border-slate-800/80 mb-5">
                <div className="flex items-center gap-3">
                  <div
                    className={`w-3 h-3 rounded-full animate-pulse ${
                      current.statusColor === "emerald"
                        ? "bg-emerald-400 shadow-[0_0_8px_#34d399]"
                        : current.statusColor === "amber"
                        ? "bg-amber-400 shadow-[0_0_8px_#fbbf24]"
                        : current.statusColor === "rose"
                        ? "bg-rose-500 shadow-[0_0_8px_#f43f5e]"
                        : "bg-violet-400 shadow-[0_0_8px_#a78bfa]"
                    }`}
                  />
                  <div>
                    <div className="text-sm font-bold text-white flex items-center gap-2">
                      <span>api.store.internal</span>
                      <span className="text-[10px] px-2 py-0.5 rounded bg-slate-800 text-slate-300 font-mono">
                        PROD
                      </span>
                    </div>
                    <div className="text-xs text-slate-400 font-mono">HTTPS GET · 60s interval</div>
                  </div>
                </div>

                {/* Status Pill */}
                <div
                  className={`px-3 py-1 rounded-full text-xs font-bold border transition-colors duration-300 ${
                    current.statusColor === "emerald"
                      ? "bg-emerald-500/10 text-emerald-400 border-emerald-500/30"
                      : current.statusColor === "amber"
                      ? "bg-amber-500/10 text-amber-400 border-amber-500/30"
                      : current.statusColor === "rose"
                      ? "bg-rose-500/15 text-rose-400 border-rose-500/30"
                      : "bg-violet-500/15 text-violet-300 border-violet-500/30"
                  }`}
                >
                  {current.statusText}
                </div>
              </div>

              {/* 3 Real-Time Telemetry Metrics */}
              <div className="grid grid-cols-3 gap-3 mb-6">
                <div className="p-3 rounded-xl bg-slate-900/80 border border-slate-800 text-center">
                  <div className="text-[11px] text-slate-400 font-medium">{isRTL ? "تاخیر پاسخ" : "Latency"}</div>
                  <div
                    className={`text-lg sm:text-xl font-bold font-mono transition-colors duration-300 ${
                      current.statusColor === "emerald"
                        ? "text-emerald-400"
                        : current.statusColor === "amber"
                        ? "text-amber-400"
                        : "text-rose-400"
                    }`}
                  >
                    {current.latency}
                  </div>
                </div>

                <div className="p-3 rounded-xl bg-slate-900/80 border border-slate-800 text-center">
                  <div className="text-[11px] text-slate-400 font-medium">{isRTL ? "کد وضعیت HTTP" : "HTTP Status"}</div>
                  <div
                    className={`text-lg sm:text-xl font-bold font-mono transition-colors duration-300 ${
                      current.statusColor === "emerald"
                        ? "text-emerald-400"
                        : current.statusColor === "amber"
                        ? "text-amber-300"
                        : "text-rose-400"
                    }`}
                  >
                    {current.httpCode}
                  </div>
                </div>

                <div className="p-3 rounded-xl bg-slate-900/80 border border-slate-800 text-center">
                  <div className="text-[11px] text-slate-400 font-medium">{isRTL ? "پایداری (آپ‌تایم)" : "Availability"}</div>
                  <div className="text-lg sm:text-xl font-bold font-mono text-white">
                    {current.uptime}
                  </div>
                </div>
              </div>

              {/* Dynamic SVG Waveform & Anomaly Visualizer */}
              <div className="rounded-xl bg-[#070b14] border border-slate-800 p-4 relative overflow-hidden">
                <div className="flex items-center justify-between text-[11px] text-slate-400 font-mono mb-2">
                  <span className="flex items-center gap-1.5">
                    <Activity className="w-3.5 h-3.5 text-blue-400" />
                    <span>{isRTL ? "نمودار تاخیر بلادرنگ (ms)" : "Real-time Latency Waveform (ms)"}</span>
                  </span>
                  <span>{current.metric}</span>
                </div>

                {/* SVG Graph Canvas */}
                <div className="h-32 w-full relative flex items-center justify-center">
                  <svg
                    viewBox="0 0 500 140"
                    className="w-full h-full overflow-visible"
                    preserveAspectRatio="none"
                  >
                    {/* Background Grid Lines */}
                    <line x1="0" y1="20" x2="500" y2="20" stroke="#1e293b" strokeDasharray="4 4" />
                    <line x1="0" y1="70" x2="500" y2="70" stroke="#1e293b" strokeDasharray="4 4" />
                    <line x1="0" y1="120" x2="500" y2="120" stroke="#1e293b" strokeDasharray="4 4" />

                    {/* Latency Curve Path */}
                    <path
                      d={renderGraphPath(current.graphData)}
                      fill="none"
                      stroke={
                        current.statusColor === "emerald"
                          ? "#10b981"
                          : current.statusColor === "amber"
                          ? "#f59e0b"
                          : "#ef4444"
                      }
                      strokeWidth="3"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      className="transition-all duration-700 ease-out"
                    />
                  </svg>

                  {/* Scene-Specific Overlay Banners */}
                  {current.status === "down" && (
                    <div className="absolute inset-0 bg-rose-950/40 backdrop-blur-xs flex items-center justify-center rounded-lg animate-fadeIn">
                      <div className="px-4 py-2 rounded-xl bg-rose-600/90 text-white font-bold text-xs flex items-center gap-2 shadow-xl shadow-rose-900/50">
                        <AlertTriangle className="w-4 h-4" />
                        <span>{isRTL ? "هشدار: قطعی قطعی شد · باز شدن رخداد #INC-804" : "Outage Confirmed · Incident #INC-804 Opened"}</span>
                      </div>
                    </div>
                  )}

                  {current.status === "alerting" && (
                    <div className="absolute inset-0 bg-violet-950/50 backdrop-blur-xs flex items-center justify-center p-3 rounded-lg animate-fadeIn">
                      <div className="w-full max-w-sm rounded-xl bg-slate-900 border border-violet-500/50 p-3 shadow-2xl text-left space-y-1.5">
                        <div className="flex items-center justify-between text-[11px] text-violet-400 font-bold">
                          <span className="flex items-center gap-1.5">
                            <Bell className="w-3.5 h-3.5 animate-bounce" />
                            <span>ClientPulse Dispatcher</span>
                          </span>
                          <span className="font-mono text-slate-400">14:22:05 UTC</span>
                        </div>
                        <div className="text-xs text-white font-medium">
                          [ALERT] api.store.internal is DOWN (502 Bad Gateway)
                        </div>
                        <div className="flex items-center gap-2 text-[10px] text-slate-400 pt-1">
                          <span className="px-1.5 py-0.5 rounded bg-slate-800 text-slate-300">Telegram</span>
                          <span className="px-1.5 py-0.5 rounded bg-slate-800 text-slate-300">Email</span>
                          <span className="px-1.5 py-0.5 rounded bg-slate-800 text-slate-300">Webhook</span>
                        </div>
                      </div>
                    </div>
                  )}

                  {current.status === "recovered" && (
                    <div className="absolute top-2 right-2 px-3 py-1 rounded-lg bg-emerald-500/20 border border-emerald-500/40 text-emerald-300 text-xs font-semibold flex items-center gap-1.5">
                      <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
                      <span>{isRTL ? "بازیابی خودکار تایید شد" : "Auto Recovery Verified"}</span>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* Right Column: Scene Story Narrative */}
          <div className="lg:col-span-5 space-y-6">
            <div className="space-y-3">
              <div className="inline-flex items-center gap-2 px-3 py-1 rounded-lg bg-slate-800/80 border border-slate-700/80 text-xs font-mono font-semibold text-blue-400">
                <span>Phase {current.number} of 05</span>
              </div>

              <h3 className="text-2xl sm:text-3xl font-extrabold text-white tracking-tight">
                {current.title}
              </h3>

              <p className="text-slate-300 text-sm sm:text-base leading-relaxed">
                {current.desc}
              </p>
            </div>

            {/* Scene Highlights */}
            <div className="space-y-2.5 pt-2">
              <div className="flex items-center gap-3 p-3 rounded-xl bg-slate-900/60 border border-slate-800/80">
                <Clock className="w-4 h-4 text-blue-400 shrink-0" />
                <span className="text-xs sm:text-sm text-slate-300 font-medium">
                  {current.status === "operational" && (isRTL ? "پایش پیوسته در بازه زمانی ۶۰ ثانیه" : "Continuous scheduled HTTP checks at 60s intervals")}
                  {current.status === "degraded" && (isRTL ? "ثبت ناهنجاری تاخیر و فعال‌سازی پروب دوم" : "Latency anomaly threshold crossed; secondary check queued")}
                  {current.status === "down" && (isRTL ? "تایید ۲ شکست متوالی پیش از ایجاد رخداد رسمی" : "2 consecutive failures confirmed before declaring incident")}
                  {current.status === "alerting" && (isRTL ? "ارسال کمتر از ۳ ثانیه به تلگرام، ایمیل و وب‌هوک" : "Dispatched in < 3 seconds across team channels")}
                  {current.status === "recovered" && (isRTL ? "۲ بررسی موفق متوالی؛ بستن خودکار رخداد" : "2 consecutive 200 OK checks close incident cleanly")}
                </span>
              </div>
            </div>

            {/* Step Controls */}
            <div className="flex items-center gap-3 pt-4">
              <button
                type="button"
                onClick={() => setActiveScene((prev) => (prev > 0 ? prev - 1 : scenes.length - 1))}
                className="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-bold transition-colors"
              >
                {isRTL ? "مرحله قبل" : "Previous Phase"}
              </button>

              <button
                type="button"
                onClick={() => setActiveScene((prev) => (prev < scenes.length - 1 ? prev + 1 : 0))}
                className="px-5 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-xs font-bold flex items-center gap-1.5 shadow-md shadow-blue-600/30 transition-all group"
              >
                <span>{isRTL ? "مرحله بعد" : "Next Phase"}</span>
                {isRTL ? (
                  <ArrowLeft className="w-3.5 h-3.5 group-hover:-translate-x-0.5 transition-transform" />
                ) : (
                  <ArrowRight className="w-3.5 h-3.5 group-hover:translate-x-0.5 transition-transform" />
                )}
              </button>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
