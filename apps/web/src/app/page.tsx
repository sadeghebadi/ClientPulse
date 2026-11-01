"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Header } from "@/components/Header";
import { HealthChecker } from "@/components/HealthChecker";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
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
  XCircle,
  Clock,
  LogIn,
  UserPlus,
  Globe,
  Activity,
  Plus,
  Zap,
  PauseCircle,
  RotateCcw,
} from "lucide-react";

interface RecentCheck {
  id: string;
  monitor_id: string;
  scheduled_at: string;
  completed_at: string;
  latency_ms: number;
  result: "success" | "failure" | "timeout" | "blocked" | "error";
  http_status_code?: number;
  error_message?: string;
  execution_type: "scheduled" | "manual";
}

interface DashboardStats {
  total_clients: number;
  total_assets: number;
  monitors_up: number;
  monitors_down: number;
  monitors_pending: number;
  monitors_paused: number;
  recent_checks: RecentCheck[];
}

export default function Home() {
  const { user, activeOrg, role } = useAuth();
  const { t, isRTL, locale } = useI18n();

  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchStats = () => {
    setLoading(true);
    apiFetch<DashboardStats>("/api/v1/dashboard/stats")
      .then((data) => {
        setStats(data);
      })
      .catch((err) => {
        console.error("Failed to load dashboard stats", err);
      })
      .finally(() => {
        setLoading(false);
      });
  };

  useEffect(() => {
    if (!user || !activeOrg) return;
    let isMounted = true;
    apiFetch<DashboardStats>("/api/v1/dashboard/stats")
      .then((data) => {
        if (isMounted) {
          setStats(data);
          setLoading(false);
        }
      })
      .catch((err) => {
        if (isMounted) {
          console.error("Failed to load dashboard stats", err);
          setLoading(false);
        }
      });
    return () => {
      isMounted = false;
    };
  }, [user, activeOrg]);

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">
      <Header />

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 flex-1 w-full">
        {user ? (
          /* Authenticated Dashboard View */
          <div className="space-y-8">
            {/* Header Banner */}
            <div className="bg-gradient-to-r from-blue-900 to-indigo-900 rounded-2xl p-8 text-white shadow-lg flex flex-col md:flex-row items-start md:items-center justify-between gap-6">
              <div>
                <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-semibold bg-emerald-500/20 text-emerald-300 border border-emerald-400/30 mb-3">
                  <ShieldCheck className="w-3.5 h-3.5" />
                  <span>Phase 2 — Clients, Assets & Monitoring Core Active</span>
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

              <div className="flex flex-wrap items-center gap-3">
                <Link
                  href="/clients/new"
                  className="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-lg text-xs font-semibold transition-colors shadow-sm flex items-center gap-1.5"
                >
                  <Plus className="w-4 h-4" />
                  <span>{t.createClient}</span>
                </Link>
                <Link
                  href="/assets/new"
                  className="px-4 py-2 bg-white/10 hover:bg-white/20 text-white rounded-lg text-xs font-semibold backdrop-blur-sm transition-colors border border-white/20 flex items-center gap-1.5"
                >
                  <Globe className="w-4 h-4" />
                  <span>{t.createAsset}</span>
                </Link>
                <Link
                  href="/monitors/new"
                  className="px-4 py-2 bg-white/10 hover:bg-white/20 text-white rounded-lg text-xs font-semibold backdrop-blur-sm transition-colors border border-white/20 flex items-center gap-1.5"
                >
                  <Activity className="w-4 h-4" />
                  <span>{t.createMonitor}</span>
                </Link>
              </div>
            </div>

            {/* Real Metrics Grid */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 sm:gap-6">
              {/* Total Clients */}
              <Link
                href="/clients"
                className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm hover:border-blue-400 transition-all group"
              >
                <div className="flex items-center justify-between text-slate-500 mb-2">
                  <span className="text-xs font-semibold uppercase">{t.totalClients}</span>
                  <Users className="w-4 h-4 text-blue-600 group-hover:scale-110 transition-transform" />
                </div>
                <div className="text-2xl sm:text-3xl font-extrabold text-slate-900">
                  {stats ? stats.total_clients : "—"}
                </div>
                <div className="text-[11px] text-blue-600 font-semibold mt-2 flex items-center gap-1">
                  <span>Manage</span>
                  {isRTL ? <ArrowLeft className="w-3 h-3" /> : <ArrowRight className="w-3 h-3" />}
                </div>
              </Link>

              {/* Total Assets */}
              <Link
                href="/clients"
                className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm hover:border-blue-400 transition-all group"
              >
                <div className="flex items-center justify-between text-slate-500 mb-2">
                  <span className="text-xs font-semibold uppercase">{t.totalAssets}</span>
                  <Globe className="w-4 h-4 text-indigo-600 group-hover:scale-110 transition-transform" />
                </div>
                <div className="text-2xl sm:text-3xl font-extrabold text-slate-900">
                  {stats ? stats.total_assets : "—"}
                </div>
                <div className="text-[11px] text-indigo-600 font-semibold mt-2 flex items-center gap-1">
                  <span>View All</span>
                  {isRTL ? <ArrowLeft className="w-3 h-3" /> : <ArrowRight className="w-3 h-3" />}
                </div>
              </Link>

              {/* Monitors Operational (Up) */}
              <Link
                href="/monitors?status=up"
                className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm hover:border-emerald-400 transition-all group"
              >
                <div className="flex items-center justify-between text-slate-500 mb-2">
                  <span className="text-xs font-semibold uppercase">{t.monitorsUp}</span>
                  <CheckCircle className="w-4 h-4 text-emerald-600 group-hover:scale-110 transition-transform" />
                </div>
                <div className="text-2xl sm:text-3xl font-extrabold text-emerald-600">
                  {stats ? stats.monitors_up : "—"}
                </div>
                <div className="text-[11px] text-emerald-700 font-semibold mt-2 flex items-center gap-1">
                  <span>Healthy</span>
                  {isRTL ? <ArrowLeft className="w-3 h-3" /> : <ArrowRight className="w-3 h-3" />}
                </div>
              </Link>

              {/* Monitors Downtime (Down) */}
              <Link
                href="/monitors?status=down"
                className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm hover:border-rose-400 transition-all group"
              >
                <div className="flex items-center justify-between text-slate-500 mb-2">
                  <span className="text-xs font-semibold uppercase">{t.monitorsDown}</span>
                  <XCircle className="w-4 h-4 text-rose-600 group-hover:scale-110 transition-transform" />
                </div>
                <div className="text-2xl sm:text-3xl font-extrabold text-rose-600">
                  {stats ? stats.monitors_down : "—"}
                </div>
                <div className="text-[11px] text-rose-700 font-semibold mt-2 flex items-center gap-1">
                  <span>Issues</span>
                  {isRTL ? <ArrowLeft className="w-3 h-3" /> : <ArrowRight className="w-3 h-3" />}
                </div>
              </Link>
            </div>

            {/* Recent Real Checks Table */}
            <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
              <div className="p-6 border-b border-slate-100 flex items-center justify-between">
                <div>
                  <h3 className="text-base font-bold text-slate-900 tracking-tight flex items-center gap-2">
                    <Activity className="w-5 h-5 text-blue-600" />
                    <span>{t.recentChecks}</span>
                  </h3>
                  <p className="text-xs text-slate-500 mt-0.5">
                    Live verification checks recorded by the Go worker and scheduler service.
                  </p>
                </div>

                <button
                  onClick={fetchStats}
                  className="px-3 py-1.5 bg-slate-100 hover:bg-slate-200 text-slate-700 rounded-lg text-xs font-semibold transition-colors flex items-center gap-1.5"
                >
                  <RotateCcw className="w-3.5 h-3.5" />
                  <span>Refresh</span>
                </button>
              </div>

              {!stats || stats.recent_checks.length === 0 ? (
                <div className="p-12 text-center text-slate-500 text-xs">
                  {t.noHistory}
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <thead className="bg-slate-50 text-slate-600 text-xs uppercase font-semibold border-b border-slate-200">
                      <tr>
                        <th className="px-6 py-3.5">{t.status}</th>
                        <th className="px-6 py-3.5">{t.latestCheck}</th>
                        <th className="px-6 py-3.5">{t.latency}</th>
                        <th className="px-6 py-3.5">Status Code</th>
                        <th className="px-6 py-3.5">{t.executionType}</th>
                        <th className="px-6 py-3.5 text-right">{t.actions}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 text-xs">
                      {stats.recent_checks.map((chk) => (
                        <tr key={chk.id} className="hover:bg-slate-50/70 transition-colors">
                          <td className="px-6 py-3.5">
                            {chk.result === "success" && (
                              <span className="inline-flex items-center gap-1 text-emerald-600 font-semibold">
                                <CheckCircle className="w-3.5 h-3.5" />
                                <span>Success</span>
                              </span>
                            )}
                            {chk.result === "failure" && (
                              <span className="inline-flex items-center gap-1 text-rose-600 font-semibold">
                                <XCircle className="w-3.5 h-3.5" />
                                <span>Failure</span>
                              </span>
                            )}
                            {chk.result === "timeout" && (
                              <span className="inline-flex items-center gap-1 text-amber-600 font-semibold">
                                <Clock className="w-3.5 h-3.5" />
                                <span>Timeout</span>
                              </span>
                            )}
                            {chk.result === "blocked" && (
                              <span className="inline-flex items-center gap-1 text-purple-600 font-semibold">
                                <ShieldCheck className="w-3.5 h-3.5" />
                                <span>Blocked</span>
                              </span>
                            )}
                          </td>
                          <td className="px-6 py-3.5 text-slate-600">
                            {new Date(chk.completed_at).toLocaleTimeString()}
                          </td>
                          <td className="px-6 py-3.5 font-mono font-medium text-slate-800">
                            {chk.latency_ms} ms
                          </td>
                          <td className="px-6 py-3.5 font-mono font-bold text-slate-800">
                            {chk.http_status_code || "—"}
                          </td>
                          <td className="px-6 py-3.5">
                            <span
                              className={`text-[10px] uppercase font-bold px-2 py-0.5 rounded ${
                                chk.execution_type === "manual"
                                  ? "bg-purple-100 text-purple-800"
                                  : "bg-slate-100 text-slate-700"
                              }`}
                            >
                              {chk.execution_type}
                            </span>
                          </td>
                          <td className="px-6 py-3.5 text-right">
                            <Link
                              href={`/monitors/${chk.monitor_id}`}
                              className="font-semibold text-blue-600 hover:underline"
                            >
                              View Monitor
                            </Link>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>

            {/* Live Infrastructure Health Checker */}
            <HealthChecker locale={locale} />
          </div>
        ) : (
          /* Public Guest View */
          <div>
            <div className="bg-gradient-to-r from-blue-900 to-indigo-900 rounded-2xl p-8 mb-8 text-white shadow-lg">
              <div className="max-w-3xl">
                <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-semibold bg-blue-500/30 text-blue-200 border border-blue-400/30 mb-4">
                  <ShieldCheck className="w-3.5 h-3.5" />
                  <span>Phase 2 — Clients, Assets & Monitoring Core</span>
                </div>
                <h2 className="text-3xl font-extrabold tracking-tight sm:text-4xl mb-4">
                  {isRTL
                    ? "پلتفرم پایش وب‌سایت و مراقبت از کلاینت‌های آژانس"
                    : "Agency Website Monitoring & Reliability Care"}
                </h2>
                <p className="text-base sm:text-lg text-blue-100/90 leading-relaxed mb-6">
                  {isRTL
                    ? "سیستم جامع مدیریت مشتریان، ثبت وب‌سایت‌ها، موتور پایش سلامت با زمان‌بندی دقیق در PostgreSQL، محافظت پیشرفته در برابر SSRF و بررسی‌های واقعی HTTP/SSL/DNS."
                    : "Complete agency client management, website assets registry, automated PostgreSQL-backed monitoring scheduler, advanced SSRF guard, and live HTTP, SSL and DNS checks."}
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
                  {isRTL ? "موتور پایش و زمان‌بند Go" : "Go Monitoring Engine"}
                </h3>
                <p className="text-sm text-slate-600 mb-4">
                  {isRTL
                    ? "ورکر مستقل با تخصیص اتمیک جاب‌ها (SKIP LOCKED)، مقاومت در برابر خطای drift و اعتبارسنجی واقعی سلامت."
                    : "Independent background worker daemon with atomic job claiming (SKIP LOCKED), drift-free scheduling, and real checks."}
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
                  {isRTL ? "امنیت RLS و ایزولاسیون کامل" : "PostgreSQL RLS Isolation"}
                </h3>
                <p className="text-sm text-slate-600 mb-4">
                  {isRTL
                    ? "جداسازی داده‌های مشتریان، دارایی‌ها، مانیتورها و نتایج در سطح پایگاه‌داده با سیاست‌های اجباری FORCE RLS."
                    : "Complete isolation for Clients, Assets, Monitors, and Check Results enforced by PostgreSQL FORCE ROW LEVEL SECURITY."}
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
                  {isRTL ? "محافظت چندلایه SSRF" : "Multi-Layer SSRF Guard"}
                </h3>
                <p className="text-sm text-slate-600 mb-4">
                  {isRTL
                    ? "مسدودسازی IPهای خصوصی، متادیتا، DNS Rebinding با پین کردن IP در لایه TCP، و کنترل کامل ریدایرکت‌ها."
                    : "Blocks private IPs, loopback, cloud metadata, and DNS rebinding via TCP IP-pinning with redirect revalidation."}
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
