"use client";

import React, { useState } from "react";
import { Header } from "@/components/Header";
import { HealthChecker } from "@/components/HealthChecker";
import { ShieldCheck, Layers, Cpu, Database, CheckCircle, Terminal } from "lucide-react";

export default function Home() {
  const [locale, setLocale] = useState<"en" | "fa">("en");
  const isRTL = locale === "fa";

  const toggleLocale = () => {
    setLocale((prev) => (prev === "en" ? "fa" : "en"));
  };

  return (
    <div className={`min-h-screen bg-slate-50 ${isRTL ? "font-sans" : ""}`} dir={isRTL ? "rtl" : "ltr"}>
      <Header locale={locale} onToggleLocale={toggleLocale} />

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Hero Section */}
        <div className="bg-gradient-to-r from-blue-900 to-indigo-900 rounded-2xl p-8 mb-8 text-white shadow-lg">
          <div className="max-w-3xl">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-semibold bg-blue-500/30 text-blue-200 border border-blue-400/30 mb-4">
              <span>PRD v1.0 Foundation</span>
            </div>
            <h2 className="text-3xl font-extrabold tracking-tight sm:text-4xl mb-4">
              {isRTL
                ? "پایه‌گذاری معماری پایدار ClientPulse"
                : "ClientPulse Production Architecture Foundation"}
            </h2>
            <p className="text-base sm:text-lg text-blue-100/90 leading-relaxed">
              {isRTL
                ? "معماری یکپارچه ماژولار (Modular Monolith) با Go برای API و Worker، فرانت‌اند Next.js با تایپ‌اسکریپت، و ذخیره‌سازی داده در PostgreSQL همراه با ایزولاسیون کامل چندمستأجری."
                : "Modular Monolith architecture in Go for high-throughput API & Worker services, Next.js with TypeScript for the agency dashboard, and PostgreSQL for persistent multi-tenant data storage."}
            </p>
          </div>
        </div>

        {/* Live Service Health Checker */}
        <HealthChecker locale={locale} />

        {/* Foundation Overview Grid */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
          <div className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm">
            <div className="w-10 h-10 rounded-lg bg-blue-50 text-blue-600 flex items-center justify-center mb-4">
              <Cpu className="w-5 h-5" />
            </div>
            <h3 className="text-base font-semibold text-slate-900 mb-2">
              {isRTL ? "سرویس‌های Go (API و Worker)" : "Go API & Worker"}
            </h3>
            <p className="text-sm text-slate-600 mb-4">
              {isRTL
                ? "اجرای مستقل فرآیند API و Worker با استفاده از یک کدبیس مشترک، ثبت وقایع ساخت‌یافته (JSON) و Graceful Shutdown."
                : "Single codebase with independent entrypoints for API and Worker processes. Fast startup and minimal footprint."}
            </p>
            <div className="flex items-center gap-1.5 text-xs font-medium text-emerald-600">
              <CheckCircle className="w-4 h-4" />
              <span>{isRTL ? "کامپایل و تست موفق" : "Build & Tests Verified"}</span>
            </div>
          </div>

          <div className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm">
            <div className="w-10 h-10 rounded-lg bg-indigo-50 text-indigo-600 flex items-center justify-center mb-4">
              <Database className="w-5 h-5" />
            </div>
            <h3 className="text-base font-semibold text-slate-900 mb-2">
              {isRTL ? "دیتابیس و مدیریت نسخه‌های اسکیما" : "PostgreSQL & Migrations"}
            </h3>
            <p className="text-sm text-slate-600 mb-4">
              {isRTL
                ? "موتور مهاجرت اختصاصی با اسکریپت‌های SQL نسخه‌دار، پشتیبانی از تراکنش، و ردیابی وضعیت در schema_migrations."
                : "Versioned SQL migrations engine with transactional up/down execution and schema_migrations tracking table."}
            </p>
            <div className="flex items-center gap-1.5 text-xs font-medium text-emerald-600">
              <CheckCircle className="w-4 h-4" />
              <span>{isRTL ? "چارچوب مهاجرت آماده" : "Migration Engine Ready"}</span>
            </div>
          </div>

          <div className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm">
            <div className="w-10 h-10 rounded-lg bg-emerald-50 text-emerald-600 flex items-center justify-center mb-4">
              <ShieldCheck className="w-5 h-5" />
            </div>
            <h3 className="text-base font-semibold text-slate-900 mb-2">
              {isRTL ? "امنیت و ایزولاسیون چندمستأجری" : "Security & Multi-Tenancy"}
            </h3>
            <p className="text-sm text-slate-600 mb-4">
              {isRTL
                ? "میدل‌ویرهای امنیت HTTP، شناسه درخواست یکتا، ردگیری وقایع، و پایه‌های حفاظت SSRF و تفکیک سازمان‌ها."
                : "HTTP security headers, unique correlation IDs, structured logging, and foundation for tenant context isolation."}
            </p>
            <div className="flex items-center gap-1.5 text-xs font-medium text-emerald-600">
              <CheckCircle className="w-4 h-4" />
              <span>{isRTL ? "میدل‌ویرها فعال" : "Middlewares Enforced"}</span>
            </div>
          </div>
        </div>

        {/* Verification Command Snippet */}
        <div className="bg-slate-900 text-slate-200 rounded-xl p-6 font-mono text-sm border border-slate-800 shadow-inner">
          <div className="flex items-center justify-between pb-3 mb-3 border-b border-slate-800 text-xs text-slate-400">
            <span className="flex items-center gap-2">
              <Terminal className="w-4 h-4 text-emerald-400" />
              {isRTL ? "دستورات راه‌اندازی و بررسی سلامت" : "Setup & Verification Commands"}
            </span>
            <span>Docker Compose</span>
          </div>
          <pre className="overflow-x-auto text-xs leading-relaxed">
{`# 1. Start all services in background
docker compose up -d --build

# 2. Check API live probe
curl http://localhost:8080/health/live

# 3. Check DB readiness probe
curl http://localhost:8080/health/ready`}
          </pre>
        </div>
      </main>

      {/* Footer */}
      <footer className="mt-auto border-t border-slate-200 bg-white py-6">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex flex-col sm:flex-row items-center justify-between text-xs text-slate-500 gap-4">
          <p>© 2026 ClientPulse. {isRTL ? "تمامی حقوق محفوظ است." : "All rights reserved."}</p>
          <p className="font-mono">PRD v1.0 | Phase 0 — Foundation Complete</p>
        </div>
      </footer>
    </div>
  );
}

