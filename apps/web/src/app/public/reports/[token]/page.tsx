"use client";

import React, { useState, useEffect } from "react";
import { API_BASE_URL } from "@/lib/api";
import {
  FileText,
  Download,
  Printer,
  Calendar,
  Clock,
  Activity,
  Globe,
  AlertTriangle,
  Radio,
  Wrench,
  FileCheck,
  CheckCircle2,
  ShieldCheck,
  AlertCircle,
  Building2,
  Info,
} from "lucide-react";

export default function PublicReportPage({ params }: { params: Promise<{ token: string }> }) {
  const unwrappedParams = React.use(params);
  const token = unwrappedParams.token;

  const [report, setReport] = useState<any | null>(null);
  const [snapshot, setSnapshot] = useState<any | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function loadPublicReport() {
      try {
        setLoading(true);
        setError(null);
        const res = await fetch(`${API_BASE_URL}/public/reports/${encodeURIComponent(token)}`, {
          cache: "no-store",
          headers: {
            "Accept": "application/json",
          },
        });

        if (!res.ok) {
          if (res.status === 404) {
            throw new Error("REPORT_NOT_FOUND");
          } else if (res.status === 429) {
            throw new Error("RATE_LIMIT_EXCEEDED");
          }
          throw new Error("UNAVAILABLE");
        }

        const data = await res.json();
        setReport(data.report);
        setSnapshot(data.snapshot);
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : "UNAVAILABLE";
        setError(msg);
      } finally {
        setLoading(false);
      }
    }
    loadPublicReport();
  }, [token]);

  const isPersian = snapshot?.locale === "fa";

  const formatDate = (isoString?: string) => {
    if (!isoString) return "—";
    try {
      const d = new Date(isoString);
      return d.toLocaleDateString(isPersian ? "fa-IR" : "en-US", {
        year: "numeric",
        month: "short",
        day: "numeric",
      });
    } catch {
      return isoString;
    }
  };

  const formatDateTime = (isoString?: string) => {
    if (!isoString) return "—";
    try {
      const d = new Date(isoString);
      return d.toLocaleString(isPersian ? "fa-IR" : "en-US", {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      });
    } catch {
      return isoString;
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-slate-50 flex items-center justify-center p-6">
        <div className="text-center space-y-4">
          <div className="animate-spin rounded-full h-10 w-10 border-b-2 border-indigo-600 mx-auto" />
          <p className="text-sm font-medium text-slate-500">
            Loading website care report...
          </p>
        </div>
      </div>
    );
  }

  if (error || !report || !snapshot) {
    const isNotFound = error === "REPORT_NOT_FOUND";
    const isRateLimit = error === "RATE_LIMIT_EXCEEDED";

    return (
      <div className="min-h-screen bg-slate-50 flex items-center justify-center p-6">
        <div className="max-w-md w-full bg-white p-8 rounded-2xl shadow-sm border border-slate-200 text-center space-y-4">
          <div className="w-12 h-12 rounded-2xl bg-slate-100 flex items-center justify-center mx-auto text-slate-500">
            <AlertCircle className="w-6 h-6" />
          </div>
          <h2 className="text-lg font-bold text-slate-900">
            {isNotFound
              ? "Report Not Found or Expired"
              : isRateLimit
              ? "Too Many Requests"
              : "Report Unavailable"}
          </h2>
          <p className="text-xs text-slate-600 leading-relaxed">
            {isNotFound
              ? "This public report link is invalid, has expired, or was revoked by the workspace administrator."
              : isRateLimit
              ? "Rate limit exceeded. Please wait a moment before refreshing the page."
              : "Unable to retrieve the care report at this time. Please try again shortly."}
          </p>
          <div className="pt-4 border-t border-slate-100 text-[11px] text-slate-400 font-medium">
            ClientPulse · Secure Care Reports
          </div>
        </div>
      </div>
    );
  }

  const summary = snapshot.summary || {};
  const branding = snapshot.branding || {};
  const primaryColor = branding.primary_color || "#4f46e5";

  return (
    <div
      dir={isPersian ? "rtl" : "ltr"}
      className={`min-h-screen bg-slate-100 py-6 sm:py-10 px-3 sm:px-6 lg:px-8 text-slate-800 ${
        isPersian ? "font-sans" : ""
      }`}
    >
      <div className="max-w-4xl mx-auto space-y-5">
        {/* Top Control Bar (Hidden when printing) */}
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 bg-white px-5 py-3 rounded-2xl shadow-xs border border-slate-200 print:hidden">
          <div className="flex items-center gap-2 min-w-0">
            <FileText className="w-5 h-5 text-indigo-600 shrink-0" />
            <span className="text-xs sm:text-sm font-bold text-slate-800 truncate">
              {isPersian ? "گزارش اختصاصی نگهداری وب‌سایت" : "Website Care Report"}
            </span>
            <span className="hidden sm:inline-block px-2 py-0.5 text-[10px] font-semibold bg-indigo-50 text-indigo-700 rounded-md">
              {isPersian ? "سند محرمانه" : "Confidential"}
            </span>
          </div>

          <div className="flex items-center gap-2 shrink-0">
            <button
              type="button"
              onClick={() => window.print()}
              className="flex-1 sm:flex-initial px-3.5 py-1.5 text-xs font-semibold border border-slate-300 rounded-xl text-slate-700 hover:bg-slate-50 flex items-center justify-center gap-1.5 transition-colors"
            >
              <Printer className="w-3.5 h-3.5" />
              <span>{isPersian ? "چاپ / ذخیره" : "Print"}</span>
            </button>

            <a
              href={`${API_BASE_URL}/public/reports/${encodeURIComponent(token)}/pdf`}
              download
              className="flex-1 sm:flex-initial px-4 py-1.5 text-xs font-semibold bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl flex items-center justify-center gap-1.5 shadow-xs transition-colors"
            >
              <Download className="w-3.5 h-3.5" />
              <span>{isPersian ? "دانلود PDF" : "Download PDF"}</span>
            </a>
          </div>
        </div>

        {/* Printable Report Document Card */}
        <div className="bg-white rounded-2xl shadow-sm border border-slate-200 overflow-hidden print:shadow-none print:border-none">
          {/* Branded Accent Top Border & Header */}
          <div
            className="p-6 sm:p-10 border-b border-slate-200"
            style={{ borderTop: `6px solid ${primaryColor}` }}
          >
            <div className="flex flex-col sm:flex-row sm:items-start justify-between gap-6">
              <div>
                <div className="inline-block px-2.5 py-0.5 rounded-md text-xs font-mono font-bold bg-slate-100 text-slate-700 mb-3" dir="ltr">
                  v{report.version_number} &bull; {snapshot.template_version || "1.0"}
                </div>
                <h1 className="text-2xl sm:text-3xl font-black text-slate-900 tracking-tight">
                  {snapshot.title}
                </h1>
                <div className="mt-2 text-sm text-slate-600 flex items-center gap-1.5">
                  <span className="font-medium">{isPersian ? "تهیه شده برای:" : "Prepared for:"}</span>
                  <span className="font-bold text-slate-900 flex items-center gap-1">
                    <Building2 className="w-4 h-4 text-slate-400 shrink-0" />
                    {snapshot.client?.name}
                  </span>
                </div>
                <div className="mt-1 text-xs text-slate-500 flex items-center gap-1.5 font-mono" dir="ltr">
                  <Calendar className="w-3.5 h-3.5 text-slate-400 shrink-0" />
                  <span>
                    {formatDate(report.period_start)} — {formatDate(report.period_end)}
                  </span>
                </div>
              </div>

              {/* Agency Branding Details */}
              <div className="text-left rtl:text-right sm:text-right rtl:sm:text-left border-t sm:border-t-0 pt-4 sm:pt-0 border-slate-100">
                <div className="text-base sm:text-lg font-black text-slate-900">
                  {branding.display_name || "ClientPulse Agency"}
                </div>
                {branding.website_url && (
                  <div className="text-xs font-semibold text-indigo-600 mt-0.5" dir="ltr">
                    {branding.website_url}
                  </div>
                )}
                {branding.contact_email && (
                  <div className="text-xs text-slate-500 mt-0.5" dir="ltr">
                    {branding.contact_email}
                  </div>
                )}
                <div className="text-[11px] text-slate-400 mt-1.5 font-mono" dir="ltr">
                  {isPersian ? "تاریخ صدور:" : "Generated:"} {formatDateTime(report.generated_at)}
                </div>
              </div>
            </div>
          </div>

          <div className="p-6 sm:p-10 space-y-8">
            {/* 1. Executive Summary */}
            <div>
              <h2 className="text-base sm:text-lg font-bold text-slate-900 mb-4 flex items-center gap-2">
                <Activity className="w-5 h-5 text-indigo-600 shrink-0" />
                <span>{isPersian ? "خلاصه وضعیت اجرایی" : "Executive Summary"}</span>
              </h2>
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 sm:gap-4">
                <div className="p-4 bg-slate-50 rounded-xl border border-slate-200">
                  <div className="text-xs font-semibold text-slate-500">
                    {isPersian ? "درصد آپتایم میانگین" : "Avg Uptime %"}
                  </div>
                  <div className="text-2xl font-black text-slate-900 mt-1" dir="ltr">
                    {summary.average_uptime != null
                      ? `${summary.average_uptime.toFixed(2)}%`
                      : isPersian
                      ? "داده ناکافی"
                      : "Insufficient data"}
                  </div>
                  <div className="text-[10px] text-slate-400 mt-0.5">
                    {isPersian ? "در بازه فعال مانیتورها" : "During active intervals"}
                  </div>
                </div>

                <div className="p-4 bg-slate-50 rounded-xl border border-slate-200">
                  <div className="text-xs font-semibold text-slate-500">
                    {isPersian ? "پوشش پایش" : "Coverage %"}
                  </div>
                  <div className="text-2xl font-black text-indigo-600 mt-1" dir="ltr">
                    {snapshot.monitors && snapshot.monitors.length > 0 && snapshot.monitors[0].coverage_percentage != null
                      ? `${snapshot.monitors[0].coverage_percentage.toFixed(1)}%`
                      : "100.0%"}
                  </div>
                  <div className="text-[10px] text-slate-400 mt-0.5">
                    {isPersian ? "پوشش کل دوره" : "Coverage ratio"}
                  </div>
                </div>

                <div className="p-4 bg-slate-50 rounded-xl border border-slate-200">
                  <div className="text-xs font-semibold text-slate-500">
                    {isPersian ? "رخدادهای قطعی" : "Incidents"}
                  </div>
                  <div className="text-2xl font-black text-slate-900 mt-1">
                    {summary.total_incidents ?? 0}
                  </div>
                  <div className="text-[10px] text-slate-400 mt-0.5">
                    {summary.total_downtime_sec
                      ? `${Math.round(summary.total_downtime_sec / 60)} min total`
                      : isPersian
                      ? "بدون قطعی"
                      : "Zero downtime"}
                  </div>
                </div>

                <div className="p-4 bg-slate-50 rounded-xl border border-slate-200">
                  <div className="text-xs font-semibold text-slate-500">
                    {isPersian ? "کارهای نگهداری" : "Maintenance"}
                  </div>
                  <div className="text-2xl font-black text-slate-900 mt-1">
                    {summary.total_activities ?? (snapshot.maintenance ? snapshot.maintenance.length : 0)}
                  </div>
                  <div className="text-[10px] text-slate-400 mt-0.5">
                    {isPersian ? "اقدام ثبت‌شده" : "Recorded activities"}
                  </div>
                </div>
              </div>
            </div>

            {/* 2. Monitors & Performance */}
            <div>
              <h2 className="text-base sm:text-lg font-bold text-slate-900 mb-4 flex items-center gap-2">
                <Globe className="w-5 h-5 text-indigo-600 shrink-0" />
                <span>{isPersian ? "پایش عملکرد و بررسی‌ها" : "Monitoring Performance & Checks"}</span>
              </h2>
              {snapshot.monitors && snapshot.monitors.length > 0 ? (
                <>
                  {/* Desktop Table View */}
                  <div className="hidden sm:block overflow-x-auto border border-slate-200 rounded-xl">
                    <table className="w-full text-sm text-left rtl:text-right">
                      <thead className="bg-slate-50 text-slate-700 text-xs uppercase tracking-wider font-semibold">
                        <tr>
                          <th className="px-4 py-3">{isPersian ? "سایت / سرویس" : "Target / Asset"}</th>
                          <th className="px-4 py-3">{isPersian ? "آپتایم" : "Uptime %"}</th>
                          <th className="px-4 py-3">{isPersian ? "پوشش" : "Coverage %"}</th>
                          <th className="px-4 py-3">{isPersian ? "زمان پاسخ (میانگین/میانه)" : "Latency (Avg/Med)"}</th>
                          <th className="px-4 py-3">{isPersian ? "بررسی‌ها" : "Total Checks"}</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-100">
                        {snapshot.monitors.map((m: any, idx: number) => (
                          <tr key={idx} className="hover:bg-slate-50/50">
                            <td className="px-4 py-3">
                              <div className="font-semibold text-slate-900">{m.name || m.asset_name || m.url}</div>
                              <div className="text-xs text-slate-500 font-mono truncate max-w-xs" dir="ltr">{m.url}</div>
                            </td>
                            <td className="px-4 py-3" dir="ltr">
                              {m.uptime_percentage != null ? (
                                <span className="font-bold text-emerald-600">
                                  {m.uptime_percentage.toFixed(2)}%
                                </span>
                              ) : (
                                <span className="text-amber-600 text-xs font-semibold">
                                  {isPersian ? "داده ناکافی" : "Insufficient data"}
                                </span>
                              )}
                            </td>
                            <td className="px-4 py-3 text-xs text-slate-600" dir="ltr">
                              {m.coverage_percentage != null ? `${m.coverage_percentage.toFixed(1)}%` : "—"}
                            </td>
                            <td className="px-4 py-3 text-xs font-mono" dir="ltr">
                              {m.avg_response_time_ms != null ? `${m.avg_response_time_ms}ms / ${m.median_response_time_ms ?? "—"}ms` : "—"}
                            </td>
                            <td className="px-4 py-3 text-xs">
                              <span className="font-medium text-slate-700">
                                {m.total_checks} {isPersian ? "بررسی" : "checks"}
                              </span>
                              {m.failed_checks > 0 && (
                                <span className="text-rose-600 ml-1">({m.failed_checks} {isPersian ? "خطا" : "failed"})</span>
                              )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>

                  {/* Mobile Cards (<640px) */}
                  <div className="sm:hidden space-y-2.5">
                    {snapshot.monitors.map((m: any, idx: number) => (
                      <div key={idx} className="p-3.5 bg-slate-50 rounded-xl border border-slate-200 space-y-2">
                        <div className="flex items-start justify-between gap-2">
                          <div className="min-w-0">
                            <div className="font-bold text-xs text-slate-900 truncate">
                              {m.name || m.asset_name || m.url}
                            </div>
                            <div className="text-[11px] text-slate-500 font-mono truncate" dir="ltr">
                              {m.url}
                            </div>
                          </div>
                          <div className="shrink-0 font-bold text-xs" dir="ltr">
                            {m.uptime_percentage != null ? (
                              <span className="text-emerald-600">{m.uptime_percentage.toFixed(2)}%</span>
                            ) : (
                              <span className="text-amber-600 text-[10px]">{isPersian ? "داده ناکافی" : "Insufficient"}</span>
                            )}
                          </div>
                        </div>
                        <div className="flex items-center justify-between text-[11px] text-slate-500 pt-1.5 border-t border-slate-200 font-mono" dir="ltr">
                          <span>{m.avg_response_time_ms != null ? `${m.avg_response_time_ms}ms avg` : "—"}</span>
                          <span>{m.total_checks} checks {m.failed_checks > 0 ? `(${m.failed_checks} err)` : ""}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                </>
              ) : (
                <div className="p-4 bg-slate-50 rounded-xl text-xs text-slate-500">
                  {isPersian ? "هیچ مانیتورینگی ثبت نشده است." : "No monitors configured."}
                </div>
              )}
            </div>

            {/* 3. Incidents History */}
            <div>
              <h2 className="text-base sm:text-lg font-bold text-slate-900 mb-4 flex items-center gap-2">
                <AlertTriangle className="w-5 h-5 text-amber-500 shrink-0" />
                <span>{isPersian ? "رخدادهای قطعی و بازیابی" : "Incidents & Outages"}</span>
              </h2>
              {snapshot.incidents && snapshot.incidents.length > 0 ? (
                <div className="overflow-x-auto border border-slate-200 rounded-xl">
                  <table className="w-full text-sm text-left rtl:text-right">
                    <thead className="bg-slate-50 text-slate-700 text-xs uppercase tracking-wider font-semibold">
                      <tr>
                        <th className="px-4 py-3">{isPersian ? "سرویس" : "Target"}</th>
                        <th className="px-4 py-3">{isPersian ? "زمان وقوع" : "Confirmed At"}</th>
                        <th className="px-4 py-3">{isPersian ? "مدت زمان" : "Duration"}</th>
                        <th className="px-4 py-3">{isPersian ? "شرح خطا" : "Cause / Error"}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                      {snapshot.incidents.map((inc: any, idx: number) => (
                        <tr key={idx}>
                          <td className="px-4 py-3 font-semibold text-slate-900">{inc.target_name || "Monitor"}</td>
                          <td className="px-4 py-3 text-xs text-slate-500 font-mono" dir="ltr">{formatDateTime(inc.confirmed_at)}</td>
                          <td className="px-4 py-3 text-xs" dir="ltr">{inc.duration_minutes ? `${inc.duration_minutes} min` : "—"}</td>
                          <td className="px-4 py-3 text-xs text-slate-600">{inc.summary || inc.failure_category || "Service Interruption"}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="p-4 bg-emerald-50 text-emerald-800 rounded-xl text-xs sm:text-sm flex items-center gap-2 border border-emerald-200">
                  <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
                  <span>
                    {isPersian
                      ? "در این بازه زمانی هیچ‌گونه رخداد قطعی ثبت نشده است (آپتایم کامل)."
                      : "Zero outages or service interruptions recorded during this period (100% Uptime)."}
                  </span>
                </div>
              )}
            </div>

            {/* 4. Heartbeats & Scheduled Cron Jobs */}
            <div>
              <h2 className="text-base sm:text-lg font-bold text-slate-900 mb-4 flex items-center gap-2">
                <Radio className="w-5 h-5 text-indigo-600 shrink-0" />
                <span>{isPersian ? "وضعیت کران‌جاب‌ها و بک‌آپ‌ها" : "Heartbeats & Scheduled Tasks"}</span>
              </h2>
              {snapshot.heartbeats && snapshot.heartbeats.length > 0 ? (
                <div className="overflow-x-auto border border-slate-200 rounded-xl">
                  <table className="w-full text-sm text-left rtl:text-right">
                    <thead className="bg-slate-50 text-slate-700 text-xs uppercase tracking-wider font-semibold">
                      <tr>
                        <th className="px-4 py-3">{isPersian ? "نام وظیفه" : "Task"}</th>
                        <th className="px-4 py-3">{isPersian ? "وضعیت" : "Status"}</th>
                        <th className="px-4 py-3">{isPersian ? "آخرین وصول" : "Last Received"}</th>
                        <th className="px-4 py-3">{isPersian ? "دوره انتظار" : "Interval"}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                      {snapshot.heartbeats.map((hb: any, idx: number) => (
                        <tr key={idx}>
                          <td className="px-4 py-3 font-semibold text-slate-900">{hb.name}</td>
                          <td className="px-4 py-3">
                            <span
                              className={`px-2 py-0.5 text-xs font-semibold rounded-md ${
                                hb.status === "healthy"
                                  ? "bg-emerald-100 text-emerald-800"
                                  : hb.status === "missed"
                                  ? "bg-rose-100 text-rose-800"
                                  : "bg-slate-100 text-slate-800"
                              }`}
                            >
                              {hb.status}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-xs text-slate-500 font-mono" dir="ltr">
                            {formatDateTime(hb.last_received_at)}
                          </td>
                          <td className="px-4 py-3 text-xs font-mono" dir="ltr">
                            {hb.expected_interval || "—"}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="p-4 bg-slate-50 rounded-xl text-xs text-slate-500">
                  {isPersian ? "مانیتورینگ هارت‌بیت فعال نشده است." : "No heartbeat monitors configured."}
                </div>
              )}
            </div>

            {/* 5. Maintenance Activities */}
            <div>
              <h2 className="text-base sm:text-lg font-bold text-slate-900 mb-4 flex items-center gap-2">
                <Wrench className="w-5 h-5 text-indigo-600 shrink-0" />
                <span>{isPersian ? "اقدامات نگهداری انجام‌شده" : "Maintenance Activities"}</span>
              </h2>
              {snapshot.maintenance && snapshot.maintenance.length > 0 ? (
                <div className="space-y-3">
                  {snapshot.maintenance.map((act: any, idx: number) => (
                    <div key={idx} className="p-4 rounded-xl border border-slate-200 bg-slate-50/50 space-y-1.5">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <div className="flex items-center gap-2">
                          <span className="font-bold text-sm text-slate-900">{act.title}</span>
                          <span className="px-2 py-0.5 text-[11px] font-medium bg-slate-200 rounded-md text-slate-700">
                            {act.category}
                          </span>
                        </div>
                        <span className="text-xs text-slate-500 font-mono" dir="ltr">
                          {formatDateTime(act.performed_at)}
                          {act.duration_minutes ? ` • ${act.duration_minutes} min` : ""}
                        </span>
                      </div>
                      <p className="text-xs sm:text-sm text-slate-600 leading-relaxed">{act.description}</p>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="p-4 bg-slate-50 rounded-xl text-xs text-slate-500">
                  {isPersian ? "هیچ فعالیت نگهداری در این دوره ثبت نشده است." : "No activities were recorded in this period."}
                </div>
              )}
            </div>

            {/* 6. Deterministic Recommendations */}
            {snapshot.recommendations && snapshot.recommendations.length > 0 && (
              <div>
                <h2 className="text-base sm:text-lg font-bold text-slate-900 mb-4 flex items-center gap-2">
                  <FileCheck className="w-5 h-5 text-indigo-600 shrink-0" />
                  <span>{isPersian ? "توصیه‌ها و اقدامات بعدی" : "Recommendations"}</span>
                </h2>
                <div className="space-y-2">
                  {snapshot.recommendations.map((rec: any, idx: number) => (
                    <div
                      key={idx}
                      className="p-3.5 bg-blue-50/70 border border-blue-100 rounded-xl text-xs sm:text-sm text-blue-900"
                    >
                      <div className="font-bold text-blue-900 mb-0.5">{rec.title}</div>
                      <div className="text-xs text-blue-800 leading-relaxed">{rec.description}</div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* 7. Methodology & Limitations Notice */}
            <div className="p-4 bg-slate-50 rounded-xl border border-slate-200 text-xs text-slate-500 space-y-1">
              <div className="font-bold text-slate-700">
                {isPersian ? "روش محاسبه و محدودیت‌ها" : "Calculation Methods & Limitations"}
              </div>
              <p className="leading-relaxed">
                {snapshot.limitation_notice ||
                  (isPersian
                    ? "درصد آپتایم تنها بر اساس پایش‌های زمان‌بندی‌شده دوره‌های فعال محاسبه شده است و بازه‌های توقف دستی محاسبه نمی‌شوند. هارت‌بیت‌ها صرفاً تایید وصول هستند و به تنهایی بازگردانی داده‌های بک‌آپ را اثبات نمی‌کنند."
                    : "Uptime percentages exclude paused monitoring intervals and manual checks. Heartbeats confirm ping receipt only and do not independently prove backup recoverability. Missing data is not interpreted as uptime.")}
              </p>
            </div>
          </div>

          {/* Footer Branding (Rendered Exactly Once) */}
          {(branding.report_footer_text || branding.footer_text) && (
            <div className="p-6 bg-slate-50 border-t border-slate-200 text-center text-xs text-slate-500">
              {branding.report_footer_text || branding.footer_text}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
