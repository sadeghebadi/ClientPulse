"use client";

import React from "react";
import Link from "next/link";
import { Header } from "@/components/Header";
import { HealthChecker } from "@/components/HealthChecker";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import {
  ShieldCheck,
  Building,
  Users,
  Settings,
  ArrowRight,
  ArrowLeft,
  Cpu,
  Database,
  CheckCircle,
  LogIn,
  UserPlus,
} from "lucide-react";

export default function Home() {
  const { user, activeOrg, role } = useAuth();
  const { t, isRTL, locale } = useI18n();

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">
      <Header />

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 flex-1 w-full">
        {user ? (
          /* Authenticated Dashboard View */
          <div className="space-y-8">
            <div className="bg-gradient-to-r from-blue-900 to-indigo-900 rounded-2xl p-8 text-white shadow-lg flex flex-col md:flex-row items-start md:items-center justify-between gap-6">
              <div>
                <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-semibold bg-emerald-500/20 text-emerald-300 border border-emerald-400/30 mb-3">
                  <ShieldCheck className="w-3.5 h-3.5" />
                  <span>Phase 1 — Auth & Multi-Tenancy Active</span>
                </div>
                <h2 className="text-2xl sm:text-3xl font-extrabold tracking-tight">
                  {t.welcomeBack}, {user.display_name}!
                </h2>
                <div className="flex items-center gap-2 mt-2 text-blue-200 text-sm">
                  <span>{t.activeWorkspace}:</span>
                  <span className="font-bold text-white bg-blue-800/60 px-2.5 py-0.5 rounded-md border border-blue-700/50">
                    {activeOrg?.name || "Agency Workspace"}
                  </span>
                  {role && (
                    <span className="text-xs uppercase font-bold px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-200 border border-emerald-400/30">
                      {role}
                    </span>
                  )}
                </div>
              </div>

              <div className="flex items-center gap-3">
                <Link
                  href="/organizations"
                  className="px-4 py-2 bg-white/10 hover:bg-white/20 text-white rounded-lg text-xs font-semibold backdrop-blur-sm transition-colors border border-white/20 flex items-center gap-1.5"
                >
                  <Building className="w-4 h-4" />
                  <span>{t.organizations}</span>
                </Link>
                <Link
                  href="/settings/members"
                  className="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-lg text-xs font-semibold transition-colors shadow-sm flex items-center gap-1.5"
                >
                  <Users className="w-4 h-4" />
                  <span>{t.members}</span>
                </Link>
              </div>
            </div>

            {/* Quick Management Cards */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              <Link
                href="/organizations"
                className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm hover:border-blue-400 transition-all group"
              >
                <div className="w-10 h-10 rounded-lg bg-blue-50 text-blue-600 flex items-center justify-center mb-4 group-hover:scale-105 transition-transform">
                  <Building className="w-5 h-5" />
                </div>
                <h3 className="text-base font-semibold text-slate-900 mb-1">{t.myOrganizations}</h3>
                <p className="text-xs text-slate-600 mb-4 leading-relaxed">
                  Switch between active client workspaces or provision a new agency organization.
                </p>
                <div className="flex items-center gap-1 text-xs font-semibold text-blue-600">
                  <span>Manage</span>
                  {isRTL ? <ArrowLeft className="w-3.5 h-3.5" /> : <ArrowRight className="w-3.5 h-3.5" />}
                </div>
              </Link>

              <Link
                href="/settings/members"
                className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm hover:border-blue-400 transition-all group"
              >
                <div className="w-10 h-10 rounded-lg bg-indigo-50 text-indigo-600 flex items-center justify-center mb-4 group-hover:scale-105 transition-transform">
                  <Users className="w-5 h-5" />
                </div>
                <h3 className="text-base font-semibold text-slate-900 mb-1">{t.members}</h3>
                <p className="text-xs text-slate-600 mb-4 leading-relaxed">
                  Invite teammates, assign role permissions (Owner, Admin, Member, Viewer) and collaborate.
                </p>
                <div className="flex items-center gap-1 text-xs font-semibold text-blue-600">
                  <span>Manage</span>
                  {isRTL ? <ArrowLeft className="w-3.5 h-3.5" /> : <ArrowRight className="w-3.5 h-3.5" />}
                </div>
              </Link>

              <Link
                href="/settings/account"
                className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm hover:border-blue-400 transition-all group"
              >
                <div className="w-10 h-10 rounded-lg bg-emerald-50 text-emerald-600 flex items-center justify-center mb-4 group-hover:scale-105 transition-transform">
                  <Settings className="w-5 h-5" />
                </div>
                <h3 className="text-base font-semibold text-slate-900 mb-1">{t.accountSettings}</h3>
                <p className="text-xs text-slate-600 mb-4 leading-relaxed">
                  Update profile information, verify your email address, or update your password with Argon2id.
                </p>
                <div className="flex items-center gap-1 text-xs font-semibold text-blue-600">
                  <span>Manage</span>
                  {isRTL ? <ArrowLeft className="w-3.5 h-3.5" /> : <ArrowRight className="w-3.5 h-3.5" />}
                </div>
              </Link>
            </div>

            {/* Live Service Health Checker */}
            <HealthChecker locale={locale} />
          </div>
        ) : (
          /* Public Guest View */
          <div>
            <div className="bg-gradient-to-r from-blue-900 to-indigo-900 rounded-2xl p-8 mb-8 text-white shadow-lg">
              <div className="max-w-3xl">
                <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-semibold bg-blue-500/30 text-blue-200 border border-blue-400/30 mb-4">
                  <ShieldCheck className="w-3.5 h-3.5" />
                  <span>Phase 1 — Authentication & Multi-Tenancy</span>
                </div>
                <h2 className="text-3xl font-extrabold tracking-tight sm:text-4xl mb-4">
                  {isRTL
                    ? "پلتفرم پایش وب‌سایت و مراقبت از کلاینت‌های آژانس"
                    : "Agency Website Monitoring & Reliability Care"}
                </h2>
                <p className="text-base sm:text-lg text-blue-100/90 leading-relaxed mb-6">
                  {isRTL
                    ? "سیستم جامع احراز هویت با Argon2id، سشن‌های امن کوکی، کنترل دسترسی چندمستأجری در PostgreSQL با RLS، و پشتیبانی کامل دوزبانه (فارسی و انگلیسی)."
                    : "Comprehensive authentication with Argon2id, secure HttpOnly cookie sessions, multi-tenant data isolation with PostgreSQL Row-Level Security, and native bilingual RTL support."}
                </p>

                <div className="flex items-center gap-3">
                  <Link
                    href="/signup"
                    className="px-5 py-2.5 bg-blue-600 hover:bg-blue-500 text-white font-semibold text-sm rounded-lg shadow-sm transition-colors flex items-center gap-2"
                  >
                    <UserPlus className="w-4 h-4" />
                    <span>{t.createAccount}</span>
                  </Link>
                  <Link
                    href="/login"
                    className="px-5 py-2.5 bg-white/10 hover:bg-white/20 text-white font-semibold text-sm rounded-lg backdrop-blur-sm transition-colors border border-white/20 flex items-center gap-2"
                  >
                    <LogIn className="w-4 h-4" />
                    <span>{t.login}</span>
                  </Link>
                </div>
              </div>
            </div>

            {/* Live Service Health Checker */}
            <HealthChecker locale={locale} />

            {/* Foundation Architecture Grid */}
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
                  {isRTL ? "PostgreSQL و Row-Level Security" : "PostgreSQL & RLS"}
                </h3>
                <p className="text-sm text-slate-600 mb-4">
                  {isRTL
                    ? "مایگریشن‌های نسخه دار با قفل مشاوره‌ای، ایزولاسیون تراکنشی کانتکست مستأجر و سیاست‌های سطح سطر (RLS)."
                    : "Advisory-locked migrations, transaction-scoped tenant context execution, and PostgreSQL Row-Level Security policies."}
                </p>
                <div className="flex items-center gap-1.5 text-xs font-medium text-emerald-600">
                  <CheckCircle className="w-4 h-4" />
                  <span>{isRTL ? "ایزولاسیون تست‌شده" : "Tenant Isolation Verified"}</span>
                </div>
              </div>

              <div className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm">
                <div className="w-10 h-10 rounded-lg bg-emerald-50 text-emerald-600 flex items-center justify-center mb-4">
                  <ShieldCheck className="w-5 h-5" />
                </div>
                <h3 className="text-base font-semibold text-slate-900 mb-2">
                  {isRTL ? "امنیت احراز هویت و سشن" : "Auth & Session Security"}
                </h3>
                <p className="text-sm text-slate-600 mb-4">
                  {isRTL
                    ? "هشینگ رمز عبور با Argon2id، توکن‌های تصادفی با CSPRNG، ذخیره سشن‌های هش‌شده با Pepper و محافظت در برابر CSRF."
                    : "Argon2id password hashing, CSPRNG tokens, SHA-256 peppered session hashes, and double-submit CSRF protection."}
                </p>
                <div className="flex items-center gap-1.5 text-xs font-medium text-emerald-600">
                  <CheckCircle className="w-4 h-4" />
                  <span>{isRTL ? "امنیت تایید‌شده" : "Security Enforced"}</span>
                </div>
              </div>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
