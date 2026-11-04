"use client";

import React, { useState } from "react";
import {
  Radio,
  Clock,
  AlertTriangle,
  CheckCircle2,
  Terminal,
  Copy,
  Check,
  RefreshCw,
  Cpu,
  Database,
  ArrowRight,
  ArrowLeft,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";

export function HeartbeatPulseSection() {
  const { t, isRTL } = useI18n();

  const [isHungState, setIsHungState] = useState<boolean>(false);
  const [copied, setCopied] = useState<boolean>(false);

  const curlCommand = "curl -m 10 --retry 3 https://ping.mindu.ir/v1/ping/sec-token-7749";

  const handleCopy = () => {
    navigator.clipboard?.writeText(curlCommand);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <section className="py-20 sm:py-32 bg-[#050811] text-slate-100 border-b border-slate-800 relative overflow-hidden">
      {/* Background radial aura */}
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[700px] h-[400px] bg-purple-600/10 rounded-full blur-[140px] pointer-events-none" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Section Header */}
        <div className="max-w-3xl mx-auto text-center space-y-4 mb-16 sm:mb-20">
          <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full text-xs font-semibold text-purple-400 bg-purple-950/60 border border-purple-800/60 shadow-lg shadow-purple-950/30 backdrop-blur-md">
            <Radio className="w-3.5 h-3.5 text-purple-400" />
            <span>{isRTL ? "پایش ضربان ورکرها و کرون‌جاب‌ها" : "Cron & Worker Heartbeats"}</span>
          </div>

          <h2 className="text-3xl sm:text-5xl font-black text-white tracking-tight leading-tight">
            {t.landingCap2Title}
          </h2>

          <p className="text-base sm:text-lg text-slate-300 leading-relaxed font-normal">
            {t.landingCap2Desc}
          </p>
        </div>

        {/* Panoramic Oscillogram Visual Console */}
        <div className="max-w-5xl mx-auto rounded-3xl bg-[#090d18] border border-slate-800 shadow-2xl overflow-hidden ring-1 ring-white/10">
          {/* Top Console Bar */}
          <div className="p-4 sm:p-6 bg-slate-950/80 border-b border-slate-800 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <div className="w-9 h-9 rounded-xl bg-purple-950/80 text-purple-400 flex items-center justify-center border border-purple-800/60 shrink-0">
                <Database className="w-4 h-4" />
              </div>
              <div>
                <div className="text-sm font-bold text-white">
                  Nightly DB Backup & S3 Archival
                </div>
                <div className="text-xs font-mono text-slate-400">
                  Expected Interval: every 24h · Grace Window: 15m
                </div>
              </div>
            </div>

            {/* Interactive State Toggle */}
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setIsHungState(false)}
                className={`px-3 py-1.5 rounded-xl text-xs font-bold transition-all ${
                  !isHungState
                    ? "bg-purple-600 text-white shadow-md ring-1 ring-purple-400"
                    : "bg-slate-900 text-slate-400 hover:text-white"
                }`}
              >
                {isRTL ? "ضربان منظم (عادی)" : "Normal Cadence"}
              </button>
              <button
                type="button"
                onClick={() => setIsHungState(true)}
                className={`px-3 py-1.5 rounded-xl text-xs font-bold transition-all ${
                  isHungState
                    ? "bg-rose-600 text-white shadow-md ring-1 ring-rose-400"
                    : "bg-slate-900 text-slate-400 hover:text-white"
                }`}
              >
                {isRTL ? "توقف ورکر (هشدار)" : "Simulate Hung Worker"}
              </button>
            </div>
          </div>

          {/* EKG Oscillogram Waveform Area */}
          <div className="p-6 sm:p-8 bg-[#070b14] space-y-6">
            <div className="h-32 sm:h-40 rounded-2xl bg-slate-950/90 border border-slate-800/80 p-4 relative overflow-hidden flex items-center justify-center">
              {/* Animated SVG Waveform Line */}
              <svg className="w-full h-full" viewBox="0 0 600 120" preserveAspectRatio="none">
                <defs>
                  <linearGradient id="purpleGrad" x1="0%" y1="0%" x2="100%" y2="0%">
                    <stop offset="0%" stopColor="#a855f7" stopOpacity="0.4" />
                    <stop offset="50%" stopColor="#c084fc" stopOpacity="1" />
                    <stop offset="100%" stopColor="#a855f7" stopOpacity="0.4" />
                  </linearGradient>
                  <linearGradient id="roseGrad" x1="0%" y1="0%" x2="100%" y2="0%">
                    <stop offset="0%" stopColor="#f43f5e" stopOpacity="0.6" />
                    <stop offset="100%" stopColor="#e11d48" stopOpacity="1" />
                  </linearGradient>
                </defs>

                {!isHungState ? (
                  /* Rhythmic Healthy EKG Pulse Wave */
                  <path
                    d="M0,60 L120,60 L140,20 L160,100 L180,60 L320,60 L340,20 L360,100 L380,60 L520,60 L540,20 L560,100 L600,60"
                    fill="none"
                    stroke="url(#purpleGrad)"
                    strokeWidth="3"
                    className="animate-pulse"
                  />
                ) : (
                  /* Flatlined Red Missed-Beat Wave */
                  <path
                    d="M0,60 L160,60 L180,60 L300,60 L600,60"
                    fill="none"
                    stroke="url(#roseGrad)"
                    strokeWidth="3.5"
                  />
                )}
              </svg>

              {/* Status Overlay Pill */}
              <div className="absolute top-4 right-4">
                <span
                  className={`px-3 py-1 rounded-full text-xs font-bold border flex items-center gap-1.5 ${
                    !isHungState
                      ? "bg-purple-950/80 border-purple-800 text-purple-300"
                      : "bg-rose-950/80 border-rose-800 text-rose-300"
                  }`}
                >
                  <span
                    className={`w-2 h-2 rounded-full ${
                      !isHungState ? "bg-purple-400 animate-pulse" : "bg-rose-400 animate-ping"
                    }`}
                  />
                  <span>
                    {!isHungState
                      ? isRTL ? "ضربان منظم · دریافت آخرین پینگ ۵ دقیقه پیش" : "Cadence Verified · Last Ping 5m ago"
                      : isRTL ? "هشدار · عدم دریافت ضربان در بازه مهلت" : "Missed Heartbeat · Grace Window Expired"}
                  </span>
                </span>
              </div>
            </div>

            {/* Diagnostic Details Grid */}
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 text-xs">
              <div className="p-4 rounded-xl bg-slate-900/80 border border-slate-800 space-y-1">
                <span className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
                  {isRTL ? "تمایز قطعی وب از ورکر" : "Isolated Failure Mode"}
                </span>
                <p className="text-slate-200">
                  {isRTL
                    ? "وب‌سرور آنلاین است؛ تنها ورکر پس‌زمینه متوقف شده و سیستم هشداری دقیق ارسال می‌کند."
                    : "Web server remains UP; only the background worker halted. Diagnostics stay unambiguous."}
                </p>
              </div>

              <div className="p-4 rounded-xl bg-slate-900/80 border border-slate-800 space-y-1">
                <span className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
                  {isRTL ? "بازه مهلت منعطف" : "Grace Window Allowance"}
                </span>
                <p className="text-slate-200">
                  {isRTL
                    ? "پیشگیری از هشدارهای کاذب در زمان پردازش سنگین با تعیین بازه مهلت اضافه."
                    : "Configurable grace window prevents false alarms during transient spikes or backup delays."}
                </p>
              </div>

              <div className="p-4 rounded-xl bg-slate-900/80 border border-slate-800 space-y-1">
                <span className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
                  {isRTL ? "امنیت آدرس پینگ" : "Cryptographic Tokens"}
                </span>
                <p className="text-slate-200">
                  {isRTL
                    ? "تولید توکن‌های غیرقابل حدس برای هر جاب بدون نیاز به افشای اطلاعات احراز هویت."
                    : "Cryptographically secure ping tokens requiring zero credential exposure in scripts."}
                </p>
              </div>
            </div>

            {/* Copyable Ping Snippet */}
            <div className="p-4 rounded-xl bg-slate-950 border border-slate-800/80 flex flex-col sm:flex-row sm:items-center justify-between gap-3 font-mono text-xs">
              <div className="flex items-center gap-2 truncate text-slate-300">
                <Terminal className="w-4 h-4 text-purple-400 shrink-0" />
                <span className="ltr-text truncate">{curlCommand}</span>
              </div>

              <button
                type="button"
                onClick={handleCopy}
                className="px-3.5 py-1.5 rounded-lg text-xs font-bold bg-slate-900 hover:bg-slate-800 text-slate-300 border border-slate-700 transition-colors flex items-center gap-1.5 shrink-0"
              >
                {copied ? (
                  <>
                    <Check className="w-3.5 h-3.5 text-emerald-400" />
                    <span className="text-emerald-400">Copied</span>
                  </>
                ) : (
                  <>
                    <Copy className="w-3.5 h-3.5" />
                    <span>Copy Ping Curl</span>
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

