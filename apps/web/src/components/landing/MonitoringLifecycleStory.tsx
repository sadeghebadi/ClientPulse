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
  Server,
  Layers,
  Sparkles,
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
      statusText: isRTL ? "اتصال برقرار · ۱۰۰٪ سالم" : "Connected · 100% Healthy",
      statusColor: "emerald",
      accentBg: "from-emerald-500/20 via-teal-500/10 to-transparent",
      borderColor: "border-emerald-500/40",
      glowColor: "rgba(16, 185, 129, 0.18)",
      latency: "34 ms",
      httpCode: "200 OK",
      uptime: "100.0%",
      graphData: [34, 33, 35, 34, 36, 34, 33, 35, 34, 34],
    },
    {
      id: 1,
      number: "02",
      title: t.storyScene2Title,
      tag: t.storyScene2Tag,
      desc: t.storyScene2Desc,
      metric: t.storyScene2Metric,
      status: "probing",
      statusText: isRTL ? "تله‌متری مداوم · بررسی پروب‌ها" : "Active Telemetry · Edge Probes",
      statusColor: "cyan",
      accentBg: "from-cyan-500/20 via-blue-500/10 to-transparent",
      borderColor: "border-cyan-500/40",
      glowColor: "rgba(6, 182, 212, 0.18)",
      latency: "38 ms",
      httpCode: "200 OK",
      uptime: "99.99%",
      graphData: [34, 36, 38, 37, 39, 38, 36, 38, 37, 38],
    },
    {
      id: 2,
      number: "03",
      title: t.storyScene3Title,
      tag: t.storyScene3Tag,
      desc: t.storyScene3Desc,
      metric: t.storyScene3Metric,
      status: "degraded",
      statusText: isRTL ? "هشدار · افت سرعت پاسخ" : "Warning · Latency Spike",
      statusColor: "amber",
      accentBg: "from-amber-500/20 via-orange-500/10 to-transparent",
      borderColor: "border-amber-500/40",
      glowColor: "rgba(245, 158, 11, 0.20)",
      latency: "1,280 ms",
      httpCode: "200 OK (Slow)",
      uptime: "99.95%",
      graphData: [38, 48, 140, 320, 680, 950, 1180, 1280, 1240, 1280],
    },
    {
      id: 3,
      number: "04",
      title: t.storyScene4Title,
      tag: t.storyScene4Tag,
      desc: t.storyScene4Desc,
      metric: t.storyScene4Metric,
      status: "down",
      statusText: isRTL ? "قطعی بحرانی تایید شد" : "Critical Outage Confirmed",
      statusColor: "rose",
      accentBg: "from-rose-600/25 via-red-600/15 to-transparent",
      borderColor: "border-rose-500/50",
      glowColor: "rgba(239, 68, 68, 0.25)",
      latency: "TIMEOUT",
      httpCode: "502 Bad Gateway",
      uptime: "98.40%",
      graphData: [1280, 1340, 0, 0, 0, 0, 0, 0, 0, 0],
    },
    {
      id: 4,
      number: "05",
      title: t.storyScene5Title,
      tag: t.storyScene5Tag,
      desc: t.storyScene5Desc,
      metric: t.storyScene5Metric,
      status: "incident",
      statusText: isRTL ? "رخداد باز · ثبت علائم خطا" : "Incident #INC-902 Opened",
      statusColor: "violet",
      accentBg: "from-violet-600/20 via-purple-600/10 to-transparent",
      borderColor: "border-violet-500/40",
      glowColor: "rgba(139, 92, 246, 0.20)",
      latency: "TIMEOUT",
      httpCode: "502 Bad Gateway",
      uptime: "98.40%",
      graphData: [0, 0, 0, 0, 0, 0, 0, 0, 0, 0],
    },
    {
      id: 5,
      number: "06",
      title: t.storyScene6Title || "06 · Automated Recovery & Resolution",
      tag: t.storyScene6Tag || "100% Operational",
      desc: t.storyScene6Desc || "Service restores normal execution. Probes verify HTTP 200 OK (38ms). Incident #INC-902 automatically closes with complete audit history and total downtime of 2m 14s.",
      metric: t.storyScene6Metric || "Resolved · Outage 2m 14s · Peace of Mind Restored",
      status: "recovered",
      statusText: isRTL ? "بازیابی کامل و آرامش خاطر" : "Recovered · Fully Operational",
      statusColor: "emerald",
      accentBg: "from-emerald-500/20 via-blue-500/10 to-transparent",
      borderColor: "border-emerald-500/40",
      glowColor: "rgba(16, 185, 129, 0.20)",
      latency: "38 ms",
      httpCode: "200 OK",
      uptime: "99.98%",
      graphData: [0, 0, 180, 72, 44, 38, 36, 38, 37, 38],
    },
  ];

  // GSAP ScrollTrigger Integration
  useEffect(() => {
    if (typeof window === "undefined" || reducedMotion) return;

    gsap.registerPlugin(ScrollTrigger);

    const triggerEl = triggerRef.current;
    if (!triggerEl) return;

    // Smooth scrubbed trigger traversing all 6 scenes naturally
    const st = ScrollTrigger.create({
      trigger: triggerEl,
      start: "top 25%",
      end: "+=2600",
      pin: false,
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

  return (
    <section
      id="story"
      ref={triggerRef}
      className="py-20 sm:py-32 bg-[#060913] text-slate-100 relative overflow-hidden border-b border-slate-800 scroll-mt-20"
    >
      {/* Target anchors for backward compatibility */}
      <span id="lifecycle" className="sr-only" />
      <span id="how-it-works" className="sr-only" />

      {/* Dynamic Background Ambient Glow reacting to current scene color */}
      <div
        className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[850px] h-[550px] blur-[160px] pointer-events-none rounded-full transition-colors duration-700"
        style={{ backgroundColor: current.glowColor }}
      />
      <div className="absolute inset-0 bg-[radial-gradient(#3b82f6_1px,transparent_1px)] [background-size:32px_32px] opacity-[0.05] pointer-events-none" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10" ref={containerRef}>
        {/* Section Header */}
        <div className="max-w-3xl mx-auto text-center space-y-4 mb-12 sm:mb-16">
          <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full text-xs font-semibold text-blue-400 bg-blue-950/60 border border-blue-800/60 shadow-lg shadow-blue-950/30 backdrop-blur-md">
            <Sparkles className="w-3.5 h-3.5 text-blue-400" />
            <span>{t.landingStoryBadge}</span>
          </div>

          <h2 className="text-3xl sm:text-5xl lg:text-6xl font-black text-white tracking-tight leading-tight">
            {t.landingStoryTitle}
          </h2>

          <p className="text-base sm:text-lg text-slate-300 leading-relaxed font-normal">
            {t.landingStorySubtitle}
          </p>
        </div>

        {/* ======================================================== */}
        {/* 6-Scene Interactive Stepper Navigation Bar               */}
        {/* ======================================================== */}
        <div className="mb-10 max-w-4xl mx-auto">
          <div className="grid grid-cols-3 sm:grid-cols-6 gap-2 p-1.5 rounded-2xl bg-slate-900/80 border border-slate-800/80 backdrop-blur-xl shadow-xl">
            {scenes.map((s, idx) => {
              const isActive = activeScene === idx;
              return (
                <button
                  key={s.id}
                  type="button"
                  onClick={() => setActiveScene(idx)}
                  className={`flex flex-col items-center justify-center py-2.5 px-2 rounded-xl text-center transition-all ${
                    isActive
                      ? "bg-slate-800 text-white shadow-md border border-slate-700 ring-1 ring-blue-500/50"
                      : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/40"
                  }`}
                >
                  <span className="text-[11px] font-mono font-extrabold tracking-wider text-blue-400">
                    {s.number}
                  </span>
                  <span className="text-[11px] font-medium truncate max-w-full mt-0.5">
                    {s.tag}
                  </span>
                </button>
              );
            })}
          </div>
        </div>

        {/* ======================================================== */}
        {/* Cinematic Main Scene Stage Container                     */}
        {/* ======================================================== */}
        <div
          className={`max-w-5xl mx-auto rounded-3xl bg-[#090e1a] border ${current.borderColor} shadow-2xl overflow-hidden transition-all duration-500 ring-1 ring-white/10`}
        >
          {/* Top Scene HUD Header */}
          <div className="bg-slate-950/90 px-6 py-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-800/80 backdrop-blur-md">
            <div className="flex items-center gap-3">
              <span className="px-3 py-1 rounded-lg bg-blue-950/80 border border-blue-800/80 text-blue-400 font-mono text-xs font-black">
                PHASE {current.number}
              </span>
              <span className="text-sm sm:text-base font-bold text-white tracking-tight">
                {current.title}
              </span>
            </div>

            <div className="flex items-center gap-3">
              <div
                className={`inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-semibold ${
                  current.statusColor === "emerald"
                    ? "bg-emerald-950/70 border border-emerald-800/80 text-emerald-400"
                    : current.statusColor === "amber"
                    ? "bg-amber-950/70 border border-amber-800/80 text-amber-400"
                    : current.statusColor === "rose"
                    ? "bg-rose-950/70 border border-rose-800/80 text-rose-400"
                    : current.statusColor === "violet"
                    ? "bg-violet-950/70 border border-violet-800/80 text-violet-400"
                    : "bg-cyan-950/70 border border-cyan-800/80 text-cyan-400"
                }`}
              >
                <span
                  className={`w-2 h-2 rounded-full ${
                    current.statusColor === "emerald"
                      ? "bg-emerald-400 animate-pulse"
                      : current.statusColor === "amber"
                      ? "bg-amber-400 animate-ping"
                      : current.statusColor === "rose"
                      ? "bg-rose-500 animate-bounce"
                      : current.statusColor === "violet"
                      ? "bg-violet-400 animate-pulse"
                      : "bg-cyan-400 animate-pulse"
                  }`}
                />
                <span>{current.statusText}</span>
              </div>
            </div>
          </div>

          {/* Inner Stage Narrative & Visualizer */}
          <div className="p-6 sm:p-8 lg:p-10 space-y-8">
            {/* Story Description & Primary Telemetry */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-center">
              <div className="lg:col-span-2 space-y-3">
                <h3 className="text-xl sm:text-2xl font-bold text-white tracking-tight">
                  {current.tag}
                </h3>
                <p className="text-sm sm:text-base text-slate-300 leading-relaxed font-normal">
                  {current.desc}
                </p>
              </div>

              {/* Real-time Metric Pill Card */}
              <div className="p-4 rounded-2xl bg-slate-900/90 border border-slate-800 shadow-inner space-y-2">
                <div className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
                  {isRTL ? "شاخص لحظه‌ای" : "Live State Telemetry"}
                </div>
                <div className="text-2xl font-extrabold text-white font-mono flex items-center justify-between">
                  <span className="ltr-text">{current.latency}</span>
                  <span
                    className={`text-xs px-2.5 py-1 rounded-md ${
                      current.statusColor === "emerald"
                        ? "bg-emerald-500/20 text-emerald-400"
                        : current.statusColor === "amber"
                        ? "bg-amber-500/20 text-amber-400"
                        : current.statusColor === "rose"
                        ? "bg-rose-500/20 text-rose-400"
                        : current.statusColor === "violet"
                        ? "bg-violet-500/20 text-violet-400"
                        : "bg-cyan-500/20 text-cyan-400"
                    }`}
                  >
                    {current.httpCode}
                  </span>
                </div>
                <div className="text-[11px] text-slate-400 font-medium">
                  {current.metric}
                </div>
              </div>
            </div>

            {/* ======================================================== */}
            {/* Morphing Visualizer Canvas: Dynamic Per-Scene Geometry  */}
            {/* ======================================================== */}
            <div className="w-full h-44 sm:h-52 rounded-2xl bg-slate-950/80 border border-slate-800/80 p-4 sm:p-6 relative overflow-hidden flex flex-col justify-between">
              {/* Target Metadata Bar */}
              <div className="flex items-center justify-between text-xs text-slate-400 font-mono">
                <div className="flex items-center gap-2">
                  <Server className="w-3.5 h-3.5 text-blue-400" />
                  <span className="ltr-text font-bold text-slate-300">https://storefront.production/api/v1</span>
                </div>
                <span className="text-[11px] text-slate-500">
                  {isRTL ? "پایش هر ۶۰ ثانیه" : "Probed every 60s"}
                </span>
              </div>

              {/* Dynamic Waveform Graph */}
              <div className="h-24 sm:h-28 flex items-end justify-between gap-1.5 sm:gap-2 px-2 relative">
                {current.graphData.map((val, idx) => {
                  const maxVal = 1400;
                  const heightPercent = val === 0 ? 6 : Math.max(12, Math.min(100, (val / maxVal) * 100));
                  return (
                    <div key={idx} className="flex-1 flex flex-col items-center justify-end h-full">
                      <div
                        style={{ height: `${heightPercent}%` }}
                        className={`w-full rounded-t transition-all duration-500 ${
                          val === 0
                            ? "bg-rose-900/30 border-t border-rose-500/50"
                            : current.statusColor === "emerald"
                            ? "bg-gradient-to-t from-emerald-600/40 to-emerald-400"
                            : current.statusColor === "amber"
                            ? "bg-gradient-to-t from-amber-600/40 to-amber-400"
                            : current.statusColor === "rose"
                            ? "bg-gradient-to-t from-rose-600/40 to-rose-400"
                            : current.statusColor === "violet"
                            ? "bg-gradient-to-t from-violet-600/40 to-violet-400"
                            : "bg-gradient-to-t from-cyan-600/40 to-cyan-400"
                        }`}
                      />
                    </div>
                  );
                })}
              </div>

              {/* Bottom State Bar */}
              <div className="flex items-center justify-between text-[11px] text-slate-400 pt-2 border-t border-slate-800/80">
                <span>{isRTL ? "پنجره ۱۰ بررسی اخیر" : "Recent 10 Execution Samples"}</span>
                <span className="font-mono font-bold text-slate-300">
                  Availability: {current.uptime}
                </span>
              </div>
            </div>

            {/* Stage Footer Controls */}
            <div className="flex flex-col sm:flex-row items-center justify-between gap-4 pt-4 border-t border-slate-800/80">
              <div className="text-xs text-slate-400">
                <span>{isRTL ? `مرحله ${activeScene + 1} از ۶` : `Scene ${activeScene + 1} of 6`}</span>
                <span className="mx-2">·</span>
                <span className="text-slate-300">{current.statusText}</span>
              </div>

              <div className="flex items-center gap-2">
                <button
                  type="button"
                  disabled={activeScene === 0}
                  onClick={() => setActiveScene((prev) => Math.max(0, prev - 1))}
                  className="px-4 py-2 rounded-xl text-xs font-bold text-slate-300 bg-slate-900 hover:bg-slate-800 disabled:opacity-40 disabled:pointer-events-none border border-slate-800 transition-colors"
                >
                  {isRTL ? "مرحله قبل" : "Previous Phase"}
                </button>
                <button
                  type="button"
                  disabled={activeScene === scenes.length - 1}
                  onClick={() => setActiveScene((prev) => Math.min(scenes.length - 1, prev + 1))}
                  className="px-5 py-2 rounded-xl text-xs font-bold text-white bg-blue-600 hover:bg-blue-500 disabled:opacity-40 disabled:pointer-events-none shadow-lg shadow-blue-600/20 transition-all flex items-center gap-1.5"
                >
                  <span>{isRTL ? "مرحله بعد" : "Next Phase"}</span>
                  {isRTL ? <ArrowLeft className="w-3.5 h-3.5" /> : <ArrowRight className="w-3.5 h-3.5" />}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
