"use client";

import React, { useState, useEffect, useRef, useCallback } from "react";
import dynamic from "next/dynamic";
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
  ChevronDown,
  Send,
  MessageSquare,
  Mail,
  Webhook,
  Server,
  Layers,
  Sparkles,
  Radio,
  Flame,
  Check,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { useReducedMotion } from "./motion/useReducedMotion";
import gsap from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";

// Dynamically import the 3D WebGL Canvas
const CinematicStoryCanvas = dynamic(
  () => import("./CinematicStoryCanvas").then((mod) => mod.CinematicStoryCanvas),
  {
    ssr: false,
    loading: () => (
      <div className="absolute inset-0 w-full h-full flex items-center justify-center bg-[#050811]">
        <div className="w-48 h-48 rounded-full border border-blue-500/20 bg-blue-950/20 animate-pulse-slow" />
      </div>
    ),
  }
);

export function MonitoringLifecycleStory() {
  const { t, isRTL } = useI18n();
  const [activeScene, setActiveScene] = useState<number>(0);
  const [scrollProgress, setScrollProgress] = useState<number>(0);
  const outerWrapperRef = useRef<HTMLDivElement>(null);
  const stickyContainerRef = useRef<HTMLDivElement>(null);
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
      borderColor: "border-emerald-500/50",
      glowColor: "rgba(16, 185, 129, 0.22)",
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
      borderColor: "border-cyan-500/50",
      glowColor: "rgba(6, 182, 212, 0.22)",
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
      borderColor: "border-amber-500/60",
      glowColor: "rgba(245, 158, 11, 0.25)",
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
      accentBg: "from-rose-600/30 via-red-600/20 to-transparent",
      borderColor: "border-rose-500/70",
      glowColor: "rgba(239, 68, 68, 0.32)",
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
      accentBg: "from-violet-600/25 via-purple-600/15 to-transparent",
      borderColor: "border-violet-500/50",
      glowColor: "rgba(139, 92, 246, 0.25)",
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
      desc:
        t.storyScene6Desc ||
        "Service restores normal execution. Probes verify HTTP 200 OK (38ms). Incident #INC-902 automatically closes with complete audit history and total downtime of 2m 14s.",
      metric: t.storyScene6Metric || "Resolved · Outage 2m 14s · Peace of Mind Restored",
      status: "recovered",
      statusText: isRTL ? "بازیابی کامل و آرامش خاطر" : "Recovered · Fully Operational",
      statusColor: "emerald",
      accentBg: "from-emerald-500/25 via-teal-500/15 to-transparent",
      borderColor: "border-emerald-500/50",
      glowColor: "rgba(16, 185, 129, 0.25)",
      latency: "38 ms",
      httpCode: "200 OK",
      uptime: "99.98%",
      graphData: [0, 0, 180, 72, 44, 38, 36, 38, 37, 38],
    },
  ];

  // GSAP ScrollTrigger Scrubbing Architecture
  useEffect(() => {
    if (typeof window === "undefined" || reducedMotion) return;

    gsap.registerPlugin(ScrollTrigger);

    const outerEl = outerWrapperRef.current;
    if (!outerEl) return;

    const st = ScrollTrigger.create({
      trigger: outerEl,
      start: "top top",
      end: "bottom bottom",
      scrub: 1.2,
      onUpdate: (self) => {
        const p = self.progress;
        setScrollProgress(p);
        const newScene = Math.min(Math.floor(p * scenes.length), scenes.length - 1);
        setActiveScene(newScene);
      },
    });

    return () => {
      st.kill();
    };
  }, [reducedMotion, scenes.length]);

  const current = scenes[activeScene];

  // Jump to specific scene smoothly
  const handleJumpToScene = useCallback(
    (sceneIndex: number) => {
      if (typeof window === "undefined") return;
      const outerEl = outerWrapperRef.current;
      if (!outerEl) return;

      const rect = outerEl.getBoundingClientRect();
      const scrollTop = window.scrollY || document.documentElement.scrollTop;
      const startY = rect.top + scrollTop;
      const totalDist = outerEl.scrollHeight - window.innerHeight;
      const targetScroll = startY + (sceneIndex / (scenes.length - 1)) * totalDist;

      window.scrollTo({
        top: targetScroll,
        behavior: "smooth",
      });
    },
    [scenes.length]
  );

  // If user prefers reduced motion, render an accessible non-pinned interactive view
  if (reducedMotion) {
    return (
      <section
        id="story"
        className="py-20 sm:py-32 bg-[#060913] text-slate-100 relative overflow-hidden border-b border-slate-800 scroll-mt-20"
      >
        <span id="lifecycle" className="sr-only" />
        <span id="how-it-works" className="sr-only" />

        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
          <div className="max-w-3xl mx-auto text-center space-y-4 mb-12 sm:mb-16">
            <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full text-xs font-semibold text-blue-400 bg-blue-950/60 border border-blue-800/60">
              <Sparkles className="w-3.5 h-3.5 text-blue-400" />
              <span>{t.landingStoryBadge}</span>
            </div>
            <h2 className="text-3xl sm:text-5xl font-black text-white tracking-tight">
              {t.landingStoryTitle}
            </h2>
            <p className="text-base sm:text-lg text-slate-300 font-normal">
              {t.landingStorySubtitle}
            </p>
          </div>

          <div className="mb-10 max-w-4xl mx-auto grid grid-cols-3 sm:grid-cols-6 gap-2 p-1.5 rounded-2xl bg-slate-900/80 border border-slate-800">
            {scenes.map((s, idx) => (
              <button
                key={s.id}
                type="button"
                onClick={() => setActiveScene(idx)}
                className={`flex flex-col items-center justify-center py-2.5 px-2 rounded-xl text-center transition-all ${
                  activeScene === idx
                    ? "bg-slate-800 text-white shadow-md border border-slate-700 ring-1 ring-blue-500/50"
                    : "text-slate-400 hover:text-slate-200"
                }`}
              >
                <span className="text-[11px] font-mono font-extrabold text-blue-400">{s.number}</span>
                <span className="text-[11px] font-medium truncate max-w-full mt-0.5">{s.tag}</span>
              </button>
            ))}
          </div>

          <div className={`max-w-4xl mx-auto rounded-3xl bg-[#090e1a] border ${current.borderColor} p-6 sm:p-10 shadow-2xl space-y-6`}>
            <div className="flex items-center justify-between border-b border-slate-800 pb-4">
              <span className="font-mono text-xs font-bold text-blue-400">PHASE {current.number}</span>
              <span className="text-xs px-3 py-1 rounded-full bg-slate-800 text-slate-300">{current.statusText}</span>
            </div>
            <h3 className="text-2xl font-bold text-white">{current.title}</h3>
            <p className="text-slate-300 leading-relaxed">{current.desc}</p>
            <div className="p-4 rounded-xl bg-slate-900 border border-slate-800 flex items-center justify-between font-mono">
              <span className="text-slate-400 text-xs">Live Telemetry</span>
              <span className="text-white font-bold">{current.metric}</span>
            </div>
          </div>
        </div>
      </section>
    );
  }

  return (
    <section
      id="story"
      ref={outerWrapperRef}
      className="relative min-h-[500vh] sm:min-h-[550vh] bg-[#050811] text-slate-100 scroll-mt-0"
    >
      {/* Target anchors for backward compatibility */}
      <span id="lifecycle" className="sr-only" />
      <span id="how-it-works" className="sr-only" />

      {/* ================================================================= */}
      {/* Sticky Full-Viewport Cinematic Stage                             */}
      {/* ================================================================= */}
      <div
        ref={stickyContainerRef}
        className="sticky top-0 h-screen w-full overflow-hidden flex flex-col justify-between select-none"
      >
        {/* Full-viewport Three.js Spatial 3D Canvas */}
        <CinematicStoryCanvas
          progress={scrollProgress}
          activeScene={activeScene}
          isRTL={isRTL}
        />

        {/* Dynamic Background Atmosphere Glow */}
        <div
          className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[900px] h-[650px] blur-[180px] pointer-events-none rounded-full transition-colors duration-700 opacity-60"
          style={{ backgroundColor: current.glowColor, zIndex: 2 }}
        />

        {/* Spatial Grid Texture */}
        <div
          className="absolute inset-0 bg-[radial-gradient(#38bdf8_1px,transparent_1px)] [background-size:40px_40px] opacity-[0.04] pointer-events-none"
          style={{ zIndex: 2 }}
        />

        {/* =============================================================== */}
        {/* TOP HUD: Narrative Header & Interactive Phase Navigation        */}
        {/* =============================================================== */}
        <header className="relative z-30 pt-6 px-4 sm:px-8 max-w-7xl mx-auto w-full">
          <div className="flex flex-col sm:flex-row items-center justify-between gap-3 bg-slate-950/70 backdrop-blur-xl border border-slate-800/80 rounded-2xl px-5 py-3 shadow-2xl">
            {/* Story Identity Brand Pill */}
            <div className="flex items-center gap-3">
              <div className="w-2.5 h-2.5 rounded-full bg-cyan-400 animate-pulse shadow-[0_0_10px_#06b6d4]" />
              <div className="flex items-baseline gap-2">
                <span className="font-extrabold text-sm tracking-tight text-white">
                  {t.landingStoryTitle || "The Reliability Chronicle"}
                </span>
                <span className="text-[11px] font-mono text-slate-400 hidden md:inline">
                  {isRTL ? "سفر سینمایی پایش بلادرنگ" : "Cinematic Observability Engine"}
                </span>
              </div>
            </div>

            {/* 6-Phase Interactive Stepper Pills */}
            <div className="flex items-center gap-1 sm:gap-1.5 p-1 rounded-xl bg-slate-900/90 border border-slate-800/80">
              {scenes.map((s, idx) => {
                const isActive = activeScene === idx;
                return (
                  <button
                    key={s.id}
                    type="button"
                    onClick={() => handleJumpToScene(idx)}
                    className={`px-2.5 py-1.5 rounded-lg text-xs font-mono transition-all flex items-center gap-1.5 cursor-pointer ${
                      isActive
                        ? "bg-blue-600 text-white font-bold shadow-lg shadow-blue-600/40 ring-1 ring-blue-400/50 scale-105"
                        : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/60"
                    }`}
                  >
                    <span className="font-black text-[10px]">{s.number}</span>
                    <span className="hidden lg:inline text-[11px] font-sans font-medium">{s.tag}</span>
                  </button>
                );
              })}
            </div>

            {/* Live Progress Scrubber Metric */}
            <div className="hidden sm:flex items-center gap-2 text-xs font-mono text-slate-400">
              <span>{Math.round(scrollProgress * 100)}%</span>
              <div className="w-16 h-1.5 rounded-full bg-slate-800 overflow-hidden">
                <div
                  className="h-full bg-gradient-to-r from-blue-500 to-cyan-400 transition-all duration-75"
                  style={{ width: `${Math.round(scrollProgress * 100)}%` }}
                />
              </div>
            </div>
          </div>
        </header>

        {/* =============================================================== */}
        {/* CENTER SPATIAL HUD CALLOUTS (Scenes 04, 05, 06 Special Moments)   */}
        {/* =============================================================== */}

        {/* Scene 04: TOTAL OUTAGE Catastrophic Alert Centerpiece */}
        {activeScene === 3 && (
          <div
            className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 z-30 pointer-events-none max-w-lg w-full px-4 animate-in fade-in zoom-in-95 duration-300"
          >
            <div className="relative rounded-2xl bg-rose-950/85 border-2 border-rose-500/80 p-6 sm:p-7 shadow-[0_0_50px_rgba(239,68,68,0.45)] backdrop-blur-2xl ring-1 ring-rose-400/40">
              {/* Tactical Crosshairs */}
              <div className="absolute -top-2 -left-2 w-4 h-4 border-t-2 border-l-2 border-rose-400" />
              <div className="absolute -top-2 -right-2 w-4 h-4 border-t-2 border-r-2 border-rose-400" />
              <div className="absolute -bottom-2 -left-2 w-4 h-4 border-b-2 border-l-2 border-rose-400" />
              <div className="absolute -bottom-2 -right-2 w-4 h-4 border-b-2 border-r-2 border-rose-400" />

              <div className="flex items-center gap-3 mb-3">
                <span className="w-3.5 h-3.5 rounded-full bg-rose-500 animate-ping inline-block" />
                <span className="font-mono text-xs font-black tracking-widest text-rose-300 uppercase">
                  {isRTL ? "هشدار قطعی سراسری · تایید شکست متوالی" : "CRITICAL INCIDENT · 2/2 CHECKS FAILED"}
                </span>
              </div>

              <h4 className="text-2xl sm:text-3xl font-black text-white tracking-tight leading-tight">
                HTTP 502 Bad Gateway
              </h4>

              <p className="mt-2 text-xs sm:text-sm text-rose-200/90 font-mono leading-relaxed">
                Target: <span className="ltr-text text-white font-bold">https://storefront.production</span>
                <br />
                Route severed · Shockwave propagated across 12 edge nodes.
              </p>

              <div className="mt-4 pt-3 border-t border-rose-800/80 flex items-center justify-between text-[11px] font-mono text-rose-300">
                <span>False-Positive Guard: Passed</span>
                <span className="font-bold text-white bg-rose-900/80 px-2 py-0.5 rounded border border-rose-700">
                  CONFIRMED DOWN
                </span>
              </div>
            </div>
          </div>
        )}

        {/* Scene 05: INCIDENT AWARENESS Tactical Drawer Centerpiece */}
        {activeScene === 4 && (
          <div
            className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 z-30 pointer-events-none max-w-xl w-full px-4 animate-in fade-in zoom-in-95 duration-300"
          >
            <div className="relative rounded-2xl bg-slate-950/90 border border-violet-500/60 p-6 sm:p-7 shadow-[0_0_40px_rgba(139,92,246,0.35)] backdrop-blur-2xl ring-1 ring-violet-400/30">
              <div className="flex items-center justify-between border-b border-slate-800 pb-3 mb-4">
                <div className="flex items-center gap-2 text-xs font-mono text-violet-400 font-bold">
                  <Activity className="w-4 h-4 text-violet-400" />
                  <span>INCIDENT #INC-902 ACTIVE</span>
                </div>
                <span className="text-[10px] font-mono px-2.5 py-0.5 rounded-full bg-violet-950 text-violet-300 border border-violet-800">
                  SEVERITY: P1-CRITICAL
                </span>
              </div>

              <div className="grid grid-cols-2 gap-3 text-xs font-mono text-slate-300">
                <div className="p-2.5 rounded-lg bg-slate-900/80 border border-slate-800">
                  <span className="text-[10px] text-slate-500 block uppercase">Root Cause</span>
                  <span className="text-white font-bold text-[11px]">Upstream Timeout (504)</span>
                </div>
                <div className="p-2.5 rounded-lg bg-slate-900/80 border border-slate-800">
                  <span className="text-[10px] text-slate-500 block uppercase">Affected Service</span>
                  <span className="text-white font-bold text-[11px] ltr-text">storefront.prod</span>
                </div>
              </div>

              <div className="mt-4 flex items-center justify-between text-xs text-slate-400 font-mono pt-3 border-t border-slate-800/80">
                <span className="flex items-center gap-1.5 text-emerald-400">
                  <Check className="w-3.5 h-3.5" />
                  <span>Dispatch Vectors Dispatched (Telegram, SMS, Webhook)</span>
                </span>
                <span className="text-slate-500">T+ 00:00:24</span>
              </div>
            </div>
          </div>
        )}

        {/* Scene 06: AUTOMATED RECOVERY Resolution Seal */}
        {activeScene === 5 && (
          <div
            className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 z-30 pointer-events-none max-w-lg w-full px-4 animate-in fade-in zoom-in-95 duration-400"
          >
            <div className="relative rounded-2xl bg-emerald-950/85 border border-emerald-500/70 p-6 sm:p-7 shadow-[0_0_45px_rgba(16,185,129,0.35)] backdrop-blur-2xl ring-1 ring-emerald-400/40 text-center space-y-3">
              <div className="w-12 h-12 mx-auto rounded-full bg-emerald-500/20 border border-emerald-400/60 flex items-center justify-center text-emerald-400 shadow-lg shadow-emerald-500/30">
                <ShieldCheck className="w-6 h-6" />
              </div>

              <h4 className="text-2xl sm:text-3xl font-black text-white tracking-tight">
                {isRTL ? "سامانه به پایداری کامل رسید" : "100% Operational Restored"}
              </h4>

              <p className="text-xs sm:text-sm text-emerald-200/90 font-mono">
                HTTP 200 OK (38ms) · Incident #INC-902 automatically closed.
                <br />
                Total Outage Duration: <span className="text-white font-bold">2m 14s</span>
              </p>

              <div className="pt-2 text-[11px] font-mono text-emerald-300">
                <span>{isRTL ? "تمام خطوط ارتباطی با موفقیت بازسازی شدند" : "Network Splines Reconnected & Verified"}</span>
              </div>
            </div>
          </div>
        )}

        {/* =============================================================== */}
        {/* BOTTOM HUD: Cinematic Story Card & Navigation Controller       */}
        {/* =============================================================== */}
        <footer className="relative z-30 pb-6 px-4 sm:px-8 max-w-7xl mx-auto w-full">
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-4 items-end">
            {/* Primary Cinematic Narrative Card */}
            <div
              className={`lg:col-span-8 rounded-2xl bg-slate-950/85 border ${current.borderColor} p-5 sm:p-6 shadow-2xl backdrop-blur-2xl transition-all duration-300 ring-1 ring-white/10`}
            >
              {/* Header Badge & Phase Marker */}
              <div className="flex items-center justify-between gap-3 mb-3">
                <div className="flex items-center gap-2">
                  <span className="px-2.5 py-0.5 rounded-md bg-blue-950/90 border border-blue-800 text-blue-400 font-mono text-xs font-black">
                    PHASE {current.number}
                  </span>
                  <span className="text-sm font-bold text-white tracking-tight">
                    {current.title}
                  </span>
                </div>

                <div
                  className={`inline-flex items-center gap-2 px-3 py-0.5 rounded-full text-xs font-mono font-semibold ${
                    current.statusColor === "emerald"
                      ? "bg-emerald-950/80 border border-emerald-800 text-emerald-400"
                      : current.statusColor === "amber"
                      ? "bg-amber-950/80 border border-amber-800 text-amber-400"
                      : current.statusColor === "rose"
                      ? "bg-rose-950/80 border border-rose-800 text-rose-400"
                      : current.statusColor === "violet"
                      ? "bg-violet-950/80 border border-violet-800 text-violet-400"
                      : "bg-cyan-950/80 border border-cyan-800 text-cyan-400"
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

              {/* Narrative Prose */}
              <p className="text-xs sm:text-sm text-slate-300 leading-relaxed font-normal">
                {current.desc}
              </p>

              {/* Live Metric Footprint */}
              <div className="mt-4 pt-3 border-t border-slate-800/80 flex flex-wrap items-center justify-between gap-2 text-xs font-mono">
                <div className="flex items-center gap-3">
                  <span className="text-slate-400">{isRTL ? "شاخص عملکرد:" : "Live Telemetry:"}</span>
                  <span className="text-white font-bold bg-slate-900/90 px-2.5 py-1 rounded-md border border-slate-800">
                    {current.metric}
                  </span>
                </div>

                {/* Micro sparkline */}
                <div className="flex items-center gap-1 h-5 px-2 bg-slate-900/80 rounded-md border border-slate-800">
                  {current.graphData.slice(-6).map((val, idx) => (
                    <div
                      key={idx}
                      style={{
                        height: val === 0 ? "3px" : `${Math.max(4, Math.min(18, (val / 1300) * 18))}px`,
                      }}
                      className={`w-1 rounded-t ${
                        val === 0
                          ? "bg-rose-500"
                          : current.statusColor === "emerald"
                          ? "bg-emerald-400"
                          : current.statusColor === "amber"
                          ? "bg-amber-400"
                          : current.statusColor === "rose"
                          ? "bg-rose-400"
                          : current.statusColor === "violet"
                          ? "bg-violet-400"
                          : "bg-cyan-400"
                      }`}
                    />
                  ))}
                </div>
              </div>
            </div>

            {/* Scroll Navigation Helper Pill */}
            <div className="lg:col-span-4 flex flex-col justify-end">
              <div className="rounded-2xl bg-slate-950/80 border border-slate-800/80 p-4 backdrop-blur-xl shadow-xl flex items-center justify-between gap-3 text-xs text-slate-400 font-mono">
                <div className="flex items-center gap-2">
                  <span className="text-cyan-400 font-black">SCROLL</span>
                  <span>{isRTL ? "برای هدایت زمان‌بندی سینمایی ↑↓" : "TO SCRUB TIMELINE ↑↓"}</span>
                </div>
                <div className="flex items-center gap-1.5">
                  <button
                    type="button"
                    onClick={() => {
                      const prevIdx = (activeScene - 1 + scenes.length) % scenes.length;
                      setActiveScene(prevIdx);
                      handleJumpToScene(prevIdx);
                    }}
                    className="px-2.5 py-1.5 rounded-lg bg-slate-900 hover:bg-slate-800 border border-slate-800 transition-colors cursor-pointer text-slate-300 text-xs font-semibold flex items-center gap-1"
                    title={isRTL ? "مرحله قبل" : "Previous Phase"}
                  >
                    {isRTL ? <ArrowRight className="w-3.5 h-3.5" /> : <ArrowLeft className="w-3.5 h-3.5" />}
                    <span className="hidden sm:inline">{isRTL ? "مرحله قبل" : "Previous Phase"}</span>
                    <span className="sr-only">Previous Phase</span>
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      const nextIdx = (activeScene + 1) % scenes.length;
                      setActiveScene(nextIdx);
                      handleJumpToScene(nextIdx);
                    }}
                    className="px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 transition-all shadow-md shadow-blue-600/30 cursor-pointer text-white text-xs font-semibold flex items-center gap-1"
                    title={isRTL ? "مرحله بعد" : "Next Phase"}
                  >
                    <span className="hidden sm:inline">{isRTL ? "مرحله بعد" : "Next Phase"}</span>
                    <span className="sr-only">Next Phase</span>
                    {isRTL ? <ArrowLeft className="w-3.5 h-3.5" /> : <ArrowRight className="w-3.5 h-3.5" />}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </footer>
      </div>
    </section>
  );
}
