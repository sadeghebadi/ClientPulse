"use client";

import React, { useState } from "react";
import {
  Globe,
  ShieldCheck,
  Zap,
  Lock,
  ArrowRight,
  ArrowLeft,
  CheckCircle2,
  Terminal,
  Activity,
  Layers,
  Copy,
  Check,
  Send,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";

export function WebAndApiObservabilitySection() {
  const { t, isRTL } = useI18n();

  const [httpMethod, setHttpMethod] = useState<"GET" | "POST" | "HEAD">("GET");
  const [isExecuting, setIsExecuting] = useState<boolean>(false);
  const [copied, setCopied] = useState<boolean>(false);
  const [activeStep, setActiveStep] = useState<number>(3); // All complete by default

  const handleDispatch = () => {
    setIsExecuting(true);
    setActiveStep(0);

    setTimeout(() => setActiveStep(1), 300);
    setTimeout(() => setActiveStep(2), 600);
    setTimeout(() => {
      setActiveStep(3);
      setIsExecuting(false);
    }, 900);
  };

  const handleCopy = () => {
    navigator.clipboard?.writeText(
      JSON.stringify(
        {
          status: "200 OK",
          latency_ms: 34,
          ttfb_ms: 16,
          dns_ms: 4,
          ssl_valid: true,
          cert_days_remaining: 76,
          ssrf_guard: "passed",
        },
        null,
        2
      )
    );
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <section id="features" className="py-20 sm:py-32 bg-[#060913] text-slate-100 border-b border-slate-800 scroll-mt-20 relative overflow-hidden">
      {/* Background ambient lighting */}
      <div className="absolute top-1/3 left-10 w-96 h-96 bg-blue-600/10 rounded-full blur-[120px] pointer-events-none" />
      <div className="absolute bottom-10 right-10 w-96 h-96 bg-cyan-600/10 rounded-full blur-[120px] pointer-events-none" />

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 relative z-10">
        {/* Section Header */}
        <div className="max-w-3xl mx-auto text-center space-y-4 mb-16 sm:mb-20">
          <div className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full text-xs font-semibold text-blue-400 bg-blue-950/60 border border-blue-800/60 shadow-lg shadow-blue-950/30 backdrop-blur-md">
            <Globe className="w-3.5 h-3.5 text-blue-400" />
            <span>{isRTL ? "موتور پایش وب‌سایت و API" : "Website & API Observability"}</span>
          </div>

          <h2 className="text-3xl sm:text-5xl font-black text-white tracking-tight leading-tight">
            {t.landingCap1Title}
          </h2>

          <p className="text-base sm:text-lg text-slate-300 leading-relaxed font-normal">
            {t.landingCap1Desc}
          </p>
        </div>

        {/* Asymmetrical Split Visual Layout (No Card Grid!) */}
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 sm:gap-12 items-center">
          {/* Left Column (5 Cols): Technical Pillars */}
          <div className="lg:col-span-5 space-y-6">
            <div className="p-6 rounded-2xl bg-[#090e1a] border border-slate-800/80 shadow-xl space-y-3 hover:border-slate-700 transition-all">
              <div className="w-10 h-10 rounded-xl bg-blue-950 text-blue-400 flex items-center justify-center border border-blue-800/60">
                <ShieldCheck className="w-5 h-5" />
              </div>
              <h3 className="text-lg font-bold text-white tracking-tight">
                {isRTL ? "محافظت چندلایه SSRF و مسدودسازی شبکه داخلی" : "Multi-Layer SSRF & RFC-1918 Block"}
              </h3>
              <p className="text-xs sm:text-sm text-slate-400 leading-relaxed font-normal">
                {isRTL
                  ? "مسدودسازی هوشمند آدرس‌های Loopback، شبکه‌های خصوصی، اندپوینت‌های متادیتای کلود (169.254.169.254) و حفاظت از زیرساخت داخلی سازمان."
                  : "Automatic DNS rebinding defenses and blocking of loopback addresses, local private subnets, and cloud instance metadata (169.254.169.254)."}
              </p>
            </div>

            <div className="p-6 rounded-2xl bg-[#090e1a] border border-slate-800/80 shadow-xl space-y-3 hover:border-slate-700 transition-all">
              <div className="w-10 h-10 rounded-xl bg-cyan-950 text-cyan-400 flex items-center justify-center border border-cyan-800/60">
                <Lock className="w-5 h-5" />
              </div>
              <h3 className="text-lg font-bold text-white tracking-tight">
                {isRTL ? "پایش هوشمند انقضای گواهی امنیتی SSL/TLS" : "Automated SSL/TLS Expiration Telemetry"}
              </h3>
              <p className="text-xs sm:text-sm text-slate-400 leading-relaxed font-normal">
                {isRTL
                  ? "ارزیابی زنجیره اعتبار گواهی SSL، رمزنگاری TLS و هشدار پیشگیرانه ۳۰ روز پیش از موعد انقضا برای جلوگیری از ازکارافتادگی دامنه."
                  : "Continuous TLS certificate chain validation, cipher handshake timing, and proactive 30-day early warnings before certificate lapse."}
              </p>
            </div>

            <div className="p-6 rounded-2xl bg-[#090e1a] border border-slate-800/80 shadow-xl space-y-3 hover:border-slate-700 transition-all">
              <div className="w-10 h-10 rounded-xl bg-indigo-950 text-indigo-400 flex items-center justify-center border border-indigo-800/60">
                <Zap className="w-5 h-5" />
              </div>
              <h3 className="text-lg font-bold text-white tracking-tight">
                {isRTL ? "اعتبارسنجی کدهای وضعیت و متن بدنه پاسخ" : "Status Assertion & JSON Path Verification"}
              </h3>
              <p className="text-xs sm:text-sm text-slate-400 leading-relaxed font-normal">
                {isRTL
                  ? "تطبیق کدهای مجاز ۲۰۰ تا ۵۹۹، بررسی وجود کلمات کلیدی در متن پاسخ و استخراج فیلدهای JSON برای اطمینان از کارکرد صحیح لاجیک بک‌اند."
                  : "Configurable expected HTTP codes, body string assertions, and JSON pointer validation to verify not just connectivity, but backend logic."}
              </p>
            </div>
          </div>

          {/* Right Column (7 Cols): Interactive API Request & Response Studio */}
          <div className="lg:col-span-7">
            <div className="rounded-3xl bg-[#0a0f1d] border border-slate-800 shadow-2xl overflow-hidden ring-1 ring-white/10">
              {/* Studio Browser Header */}
              <div className="px-5 py-4 bg-slate-950 border-b border-slate-800/80 flex items-center justify-between gap-3">
                <div className="flex items-center gap-2">
                  <span className="w-3 h-3 rounded-full bg-rose-500/80 inline-block" />
                  <span className="w-3 h-3 rounded-full bg-amber-500/80 inline-block" />
                  <span className="w-3 h-3 rounded-full bg-emerald-500/80 inline-block" />
                  <span className="ml-2 text-xs font-mono font-semibold text-slate-400">
                    HTTP Probe Dispatcher
                  </span>
                </div>

                <div className="flex items-center gap-1.5 text-[11px] font-mono text-emerald-400">
                  <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
                  <span>Probing Engine Ready</span>
                </div>
              </div>

              {/* URL & Method Selector Bar */}
              <div className="p-4 sm:p-6 bg-[#080c18] border-b border-slate-800/80 space-y-4">
                <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
                  {/* Method Picker */}
                  <div className="flex items-center gap-1 p-1 rounded-xl bg-slate-900 border border-slate-800 text-xs font-mono font-bold">
                    {(["GET", "POST", "HEAD"] as const).map((m) => (
                      <button
                        key={m}
                        type="button"
                        onClick={() => setHttpMethod(m)}
                        className={`px-3 py-1.5 rounded-lg transition-colors ${
                          httpMethod === m
                            ? "bg-blue-600 text-white shadow-sm"
                            : "text-slate-400 hover:text-white"
                        }`}
                      >
                        {m}
                      </button>
                    ))}
                  </div>

                  {/* Target URL */}
                  <div className="flex-1 px-4 py-2.5 rounded-xl bg-slate-900 border border-slate-800 text-xs font-mono text-slate-200 flex items-center justify-between truncate">
                    <span className="ltr-text truncate">https://api.store.internal/v1/health</span>
                    <Lock className="w-3.5 h-3.5 text-emerald-400 shrink-0 ml-2" />
                  </div>

                  {/* Dispatch Button */}
                  <button
                    type="button"
                    disabled={isExecuting}
                    onClick={handleDispatch}
                    className="px-5 py-2.5 rounded-xl text-xs font-bold text-white bg-blue-600 hover:bg-blue-500 disabled:opacity-50 transition-all flex items-center justify-center gap-2 shadow-lg shadow-blue-600/30 shrink-0"
                  >
                    <Send className={`w-3.5 h-3.5 ${isExecuting ? "animate-spin" : ""}`} />
                    <span>{isExecuting ? "Dispatching..." : "Dispatch Check"}</span>
                  </button>
                </div>

                {/* Animated Probe Stage Stepper */}
                <div className="grid grid-cols-4 gap-2 pt-2">
                  {[
                    { label: "DNS Lookup", timing: "4 ms", step: 0 },
                    { label: "TLS Handshake", timing: "12 ms", step: 1 },
                    { label: "TTFB Latency", timing: "16 ms", step: 2 },
                    { label: "Content Transfer", timing: "2 ms", step: 3 },
                  ].map((phase, idx) => {
                    const isDone = activeStep >= phase.step;
                    return (
                      <div
                        key={phase.label}
                        className={`p-2.5 rounded-xl border text-center transition-all ${
                          isDone
                            ? "bg-emerald-950/40 border-emerald-500/40 text-emerald-300"
                            : "bg-slate-900/50 border-slate-800 text-slate-500"
                        }`}
                      >
                        <div className="text-[10px] font-medium truncate">{phase.label}</div>
                        <div className="text-xs font-mono font-bold mt-0.5">{phase.timing}</div>
                      </div>
                    );
                  })}
                </div>
              </div>

              {/* JSON Response Payload Preview */}
              <div className="p-4 sm:p-6 bg-[#060913] space-y-3 font-mono text-xs">
                <div className="flex items-center justify-between text-slate-400">
                  <div className="flex items-center gap-2">
                    <span className="px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-400 font-bold">
                      200 OK
                    </span>
                    <span className="text-[11px] text-slate-400">Total Latency: 34 ms</span>
                  </div>

                  <button
                    type="button"
                    onClick={handleCopy}
                    className="flex items-center gap-1.5 text-[11px] text-slate-400 hover:text-white transition-colors"
                  >
                    {copied ? (
                      <>
                        <Check className="w-3.5 h-3.5 text-emerald-400" />
                        <span className="text-emerald-400">Copied</span>
                      </>
                    ) : (
                      <>
                        <Copy className="w-3.5 h-3.5" />
                        <span>Copy JSON</span>
                      </>
                    )}
                  </button>
                </div>

                <div className="p-4 rounded-xl bg-slate-950 border border-slate-800/80 overflow-x-auto text-slate-300 text-[11px] leading-relaxed">
                  <pre className="ltr-text">
{`{
  "status": "operational",
  "endpoint": "https://api.store.internal/v1/health",
  "http_code": 200,
  "latency_ms": 34,
  "ttfb_ms": 16,
  "tls_version": "TLSv1.3",
  "certificate": {
    "issuer": "Let's Encrypt Authority X3",
    "days_remaining": 76,
    "valid": true
  },
  "ssrf_protection": "RFC-1918 Block Active"
}`}
                  </pre>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

