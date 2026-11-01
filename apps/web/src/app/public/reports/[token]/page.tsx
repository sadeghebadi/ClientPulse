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
        const res = await fetch(`${API_BASE_URL}/public/reports/${token}`, {
          cache: "no-store",
        });

        if (!res.ok) {
          if (res.status === 404) {
            throw new Error("Report link is invalid, expired, or has been revoked.");
          } else if (res.status === 429) {
            throw new Error("Rate limit exceeded. Please try again shortly.");
          }
          throw new Error("Unable to load report at this time.");
        }

        const data = await res.json();
        setReport(data.report);
        setSnapshot(data.snapshot);
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : "Report unavailable");
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
      <div className="min-h-screen bg-gray-50 flex items-center justify-center p-6">
        <div className="text-center">
          <div className="animate-spin rounded-full h-10 w-10 border-b-2 border-indigo-600 mx-auto" />
          <p className="mt-4 text-sm text-gray-500">Loading website care report...</p>
        </div>
      </div>
    );
  }

  if (error || !report || !snapshot) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center p-6">
        <div className="max-w-md w-full bg-white p-8 rounded-2xl shadow-sm border border-gray-200 text-center">
          <AlertCircle className="w-12 h-12 text-gray-400 mx-auto mb-3" />
          <h2 className="text-lg font-bold text-gray-900 mb-2">Report Not Found</h2>
          <p className="text-sm text-gray-600 mb-6">
            {error || "This report link is no longer active, has expired, or was revoked."}
          </p>
          <div className="text-xs text-gray-400">ClientPulse Secure Report Sharing</div>
        </div>
      </div>
    );
  }

  const summary = snapshot.summary || {};
  const branding = snapshot.branding || {};
  const primaryColor = branding.primary_color || "#3b82f6";
  const accentColor = branding.accent_color || "#10b981";

  return (
    <div
      dir={isPersian ? "rtl" : "ltr"}
      className={`min-h-screen bg-gray-100 py-8 px-4 sm:px-6 lg:px-8 text-gray-800 ${
        isPersian ? "font-sans" : ""
      }`}
    >
      <div className="max-w-4xl mx-auto space-y-6">
        {/* Top Control Bar (Hidden when printing) */}
        <div className="flex items-center justify-between bg-white px-6 py-3 rounded-xl shadow-xs border border-gray-200 print:hidden">
          <div className="flex items-center gap-2">
            <FileText className="w-5 h-5 text-indigo-600" />
            <span className="text-xs sm:text-sm font-semibold text-gray-700">
              {isPersian ? "گزارش اختصاصی نگهداری وب‌سایت" : "Monthly Website Care Report"}
            </span>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => window.print()}
              className="px-3 py-1.5 text-xs font-medium border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50 flex items-center gap-1.5 transition-colors"
            >
              <Printer className="w-3.5 h-3.5" />
              <span>{isPersian ? "چاپ" : "Print"}</span>
            </button>

            <a
              href={`${API_BASE_URL}/public/reports/${token}/pdf`}
              download
              className="px-3.5 py-1.5 text-xs font-semibold bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg flex items-center gap-1.5 shadow-xs transition-colors"
            >
              <Download className="w-3.5 h-3.5" />
              <span>{isPersian ? "دانلود PDF" : "Download PDF"}</span>
            </a>
          </div>
        </div>

        {/* Printable Report Document */}
        <div className="bg-white rounded-2xl shadow-sm border border-gray-200 overflow-hidden print:shadow-none print:border-none">
          {/* Branded Header */}
          <div
            className="p-8 sm:p-10 border-b border-gray-200"
            style={{ borderTop: `6px solid ${primaryColor}` }}
          >
            <div className="flex flex-col sm:flex-row sm:items-start justify-between gap-6">
              <div>
                <div className="inline-block px-2.5 py-0.5 rounded text-xs font-mono font-semibold bg-gray-100 text-gray-700 mb-3">
                  v{report.version_number} &bull; {snapshot.template_version || "1.0"}
                </div>
                <h1 className="text-2xl sm:text-3xl font-extrabold text-gray-900 tracking-tight">
                  {snapshot.title}
                </h1>
                <div className="mt-2 text-sm text-gray-600">
                  {isPersian ? "تهیه شده برای:" : "Prepared for:"}{" "}
                  <span className="font-bold text-gray-900">{snapshot.client?.name}</span>
                </div>
                <div className="mt-1 text-xs text-gray-500 flex items-center gap-1.5">
                  <Calendar className="w-3.5 h-3.5 text-gray-400" />
                  <span>
                    {formatDate(report.period_start)} — {formatDate(report.period_end)}
                  </span>
                </div>
              </div>

              {/* Agency Branding Details */}
              <div className="text-left rtl:text-right sm:text-right rtl:sm:text-left border-t sm:border-t-0 pt-4 sm:pt-0 border-gray-100">
                <div className="text-xl font-bold text-gray-900">
                  {branding.display_name || "ClientPulse Agency"}
                </div>
                {branding.website_url && (
                  <div className="text-xs font-medium text-indigo-600 mt-1">
                    {branding.website_url}
                  </div>
                )}
                {branding.contact_email && (
                  <div className="text-xs text-gray-500 mt-0.5">
                    {branding.contact_email}
                  </div>
                )}
                <div className="text-xs text-gray-400 mt-2">
                  {isPersian ? "تاریخ صدور:" : "Generated:"} {formatDateTime(report.generated_at)}
                </div>
              </div>
            </div>
          </div>

          <div className="p-8 sm:p-10 space-y-8">
            {/* 1. Executive Summary */}
            <div>
              <h2 className="text-base sm:text-lg font-bold text-gray-900 mb-4 flex items-center gap-2">
                <Activity className="w-5 h-5 text-indigo-600" />
                <span>{isPersian ? "خلاصه وضعیت اجرایی" : "Executive Summary"}</span>
              </h2>
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
                <div className="p-4 bg-gray-50 rounded-xl border border-gray-200">
                  <div className="text-xs font-medium text-gray-500">
                    {isPersian ? "درصد آپتایم" : "Avg Uptime %"}
                  </div>
                  <div className="text-2xl font-black text-gray-900 mt-1">
                    {summary.average_uptime != null
                      ? `${summary.average_uptime.toFixed(2)}%`
                      : isPersian
                      ? "داده ناکافی"
                      : "Insufficient data"}
                  </div>
                  <div className="text-[10px] text-gray-400 mt-0.5">
                    {isPersian ? "در بازه فعال مانیتورها" : "During active intervals"}
                  </div>
                </div>

                <div className="p-4 bg-gray-50 rounded-xl border border-gray-200">
                  <div className="text-xs font-medium text-gray-500">
                    {isPersian ? "پوشش پایش" : "Coverage %"}
                  </div>
                  <div className="text-2xl font-black text-indigo-600 mt-1">
                    {snapshot.monitors && snapshot.monitors.length > 0 && snapshot.monitors[0].coverage_percentage != null
                      ? `${snapshot.monitors[0].coverage_percentage.toFixed(1)}%`
                      : "100.0%"}
                  </div>
                  <div className="text-[10px] text-gray-400 mt-0.5">
                    {isPersian ? "پوشش کل دوره" : "Coverage ratio"}
                  </div>
                </div>

                <div className="p-4 bg-gray-50 rounded-xl border border-gray-200">
                  <div className="text-xs font-medium text-gray-500">
                    {isPersian ? "رخدادهای قطعی" : "Incidents"}
                  </div>
                  <div className="text-2xl font-black text-gray-900 mt-1">
                    {summary.total_incidents ?? 0}
                  </div>
                  <div className="text-[10px] text-gray-400 mt-0.5">
                    {summary.total_downtime_sec ? `${Math.round(summary.total_downtime_sec / 60)} min total` : "Zero downtime"}
                  </div>
                </div>

                <div className="p-4 bg-gray-50 rounded-xl border border-gray-200">
                  <div className="text-xs font-medium text-gray-500">
                    {isPersian ? "کارهای نگهداری" : "Maintenance"}
                  </div>
                  <div className="text-2xl font-black text-gray-900 mt-1">
                    {summary.total_activities ?? (snapshot.maintenance ? snapshot.maintenance.length : 0)}
                  </div>
                  <div className="text-[10px] text-gray-400 mt-0.5">
                    {isPersian ? "اقدام ثبت‌شده" : "Recorded activities"}
                  </div>
                </div>
              </div>
            </div>

            {/* 2. Monitors & Uptime */}
            <div>
              <h2 className="text-base sm:text-lg font-bold text-gray-900 mb-4 flex items-center gap-2">
                <Globe className="w-5 h-5 text-indigo-600" />
                <span>{isPersian ? "پایش عملکرد و بررسی‌ها" : "Monitoring Performance & Checks"}</span>
              </h2>
              {snapshot.monitors && snapshot.monitors.length > 0 ? (
                <div className="overflow-x-auto border border-gray-200 rounded-xl">
                  <table className="w-full text-sm text-left rtl:text-right">
                    <thead className="bg-gray-50 text-gray-700">
                      <tr>
                        <th className="px-4 py-3">{isPersian ? "سایت / سرویس" : "Target / Asset"}</th>
                        <th className="px-4 py-3">{isPersian ? "آپتایم" : "Uptime %"}</th>
                        <th className="px-4 py-3">{isPersian ? "پوشش" : "Coverage %"}</th>
                        <th className="px-4 py-3">{isPersian ? "زمان پاسخ (میانگین/میانه)" : "Latency (Avg/Med)"}</th>
                        <th className="px-4 py-3">{isPersian ? "بررسی‌ها" : "Total Checks"}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200">
                      {snapshot.monitors.map((m: any, idx: number) => (
                        <tr key={idx}>
                          <td className="px-4 py-3">
                            <div className="font-semibold text-gray-900">{m.name || m.asset_name || m.url}</div>
                            <div className="text-xs text-gray-500 font-mono truncate max-w-xs">{m.url}</div>
                          </td>
                          <td className="px-4 py-3">
                            {m.uptime_percentage != null ? (
                              <span className="font-semibold text-emerald-600">
                                {m.uptime_percentage.toFixed(2)}%
                              </span>
                            ) : (
                              <span className="text-amber-600 text-xs font-semibold">
                                {isPersian ? "داده ناکافی" : "Insufficient data"}
                              </span>
                            )}
                          </td>
                          <td className="px-4 py-3 text-xs text-gray-600">
                            {m.coverage_percentage != null ? `${m.coverage_percentage.toFixed(1)}%` : "—"}
                          </td>
                          <td className="px-4 py-3 text-xs font-mono">
                            {m.avg_response_time_ms != null ? `${m.avg_response_time_ms}ms / ${m.median_response_time_ms ?? "—"}ms` : "—"}
                          </td>
                          <td className="px-4 py-3 text-xs font-medium text-gray-700">
                            {m.total_checks} {isPersian ? "بررسی" : "checks"}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="p-4 bg-gray-50 rounded-xl text-sm text-gray-500">
                  {isPersian ? "مانیتورینگی ثبت نشده است." : "No monitors configured."}
                </div>
              )}
            </div>

            {/* 3. Incidents History */}
            <div>
              <h2 className="text-base sm:text-lg font-bold text-gray-900 mb-4 flex items-center gap-2">
                <AlertTriangle className="w-5 h-5 text-amber-500" />
                <span>{isPersian ? "رخدادهای قطعی و بازیابی" : "Incidents & Outages"}</span>
              </h2>
              {snapshot.incidents && snapshot.incidents.length > 0 ? (
                <div className="overflow-x-auto border border-gray-200 rounded-xl">
                  <table className="w-full text-sm text-left rtl:text-right">
                    <thead className="bg-gray-50 text-gray-700">
                      <tr>
                        <th className="px-4 py-3">{isPersian ? "سرویس" : "Target"}</th>
                        <th className="px-4 py-3">{isPersian ? "زمان وقوع" : "Confirmed At"}</th>
                        <th className="px-4 py-3">{isPersian ? "مدت زمان" : "Duration"}</th>
                        <th className="px-4 py-3">{isPersian ? "شرح خطا" : "Cause / Error"}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200">
                      {snapshot.incidents.map((inc: any, idx: number) => (
                        <tr key={idx}>
                          <td className="px-4 py-3 font-semibold">{inc.target_name || "Monitor"}</td>
                          <td className="px-4 py-3 text-xs text-gray-500">{formatDateTime(inc.confirmed_at)}</td>
                          <td className="px-4 py-3 text-xs">{inc.duration_minutes ? `${inc.duration_minutes} min` : "—"}</td>
                          <td className="px-4 py-3 text-xs text-gray-600">{inc.summary || inc.failure_category || "Service Interruption"}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="p-4 bg-emerald-50 text-emerald-800 rounded-xl text-sm flex items-center gap-2 border border-emerald-200">
                  <CheckCircle2 className="w-4 h-4 text-emerald-600 flex-shrink-0" />
                  <span>
                    {isPersian
                      ? "در این بازه زمانی هیچ‌گونه رخداد قطعی ثبت نشده است (آپتایم کامل)."
                      : "Zero outages or service interruptions recorded during this period."}
                  </span>
                </div>
              )}
            </div>

            {/* 4. Heartbeats & Scheduled Cron Jobs */}
            <div>
              <h2 className="text-base sm:text-lg font-bold text-gray-900 mb-4 flex items-center gap-2">
                <Radio className="w-5 h-5 text-indigo-600" />
                <span>{isPersian ? "وضعیت کران‌جاب‌ها و بک‌آپ‌ها" : "Heartbeats & Scheduled Tasks"}</span>
              </h2>
              {snapshot.heartbeats && snapshot.heartbeats.length > 0 ? (
                <div className="overflow-x-auto border border-gray-200 rounded-xl">
                  <table className="w-full text-sm text-left rtl:text-right">
                    <thead className="bg-gray-50 text-gray-700">
                      <tr>
                        <th className="px-4 py-3">{isPersian ? "نام وظیفه" : "Task"}</th>
                        <th className="px-4 py-3">{isPersian ? "وضعیت" : "Status"}</th>
                        <th className="px-4 py-3">{isPersian ? "آخرین وصول" : "Last Received"}</th>
                        <th className="px-4 py-3">{isPersian ? "دوره انتظار" : "Interval"}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200">
                      {snapshot.heartbeats.map((hb: any, idx: number) => (
                        <tr key={idx}>
                          <td className="px-4 py-3 font-semibold">{hb.name}</td>
                          <td className="px-4 py-3">
                            <span
                              className={`px-2 py-0.5 text-xs font-semibold rounded-full ${
                                hb.status === "healthy"
                                  ? "bg-emerald-100 text-emerald-800"
                                  : hb.status === "missed"
                                  ? "bg-red-100 text-red-800"
                                  : "bg-gray-100 text-gray-800"
                              }`}
                            >
                              {hb.status}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-xs text-gray-500">{formatDateTime(hb.last_received_at)}</td>
                          <td className="px-4 py-3 text-xs">{hb.expected_interval || "—"}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="p-4 bg-gray-50 rounded-xl text-sm text-gray-500">
                  {isPersian ? "مانیتورینگ هارت‌بیت فعال نشده است." : "No heartbeat monitors configured."}
                </div>
              )}
            </div>

            {/* 5. Maintenance Activities */}
            <div>
              <h2 className="text-base sm:text-lg font-bold text-gray-900 mb-4 flex items-center gap-2">
                <Wrench className="w-5 h-5 text-indigo-600" />
                <span>{isPersian ? "اقدامات نگهداری انجام‌شده" : "Maintenance Activities"}</span>
              </h2>
              {snapshot.maintenance && snapshot.maintenance.length > 0 ? (
                <div className="space-y-3">
                  {snapshot.maintenance.map((act: any, idx: number) => (
                    <div key={idx} className="p-4 rounded-xl border border-gray-200 bg-gray-50/50">
                      <div className="flex flex-wrap items-center justify-between gap-2 mb-1">
                        <div className="flex items-center gap-2">
                          <span className="font-semibold text-gray-900">{act.title}</span>
                          <span className="px-2 py-0.5 text-xs font-medium bg-gray-200 rounded text-gray-700">
                            {act.category}
                          </span>
                        </div>
                        <span className="text-xs text-gray-500">
                          {formatDateTime(act.performed_at)}
                          {act.duration_minutes ? ` • ${act.duration_minutes} min` : ""}
                        </span>
                      </div>
                      <p className="text-sm text-gray-600">{act.description}</p>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="p-4 bg-gray-50 rounded-xl text-sm text-gray-500">
                  {isPersian ? "هیچ فعالیت نگهداری در این دوره ثبت نشده است." : "No activities were recorded in this period."}
                </div>
              )}
            </div>

            {/* 6. Deterministic Recommendations */}
            {snapshot.recommendations && snapshot.recommendations.length > 0 && (
              <div>
                <h2 className="text-base sm:text-lg font-bold text-gray-900 mb-4 flex items-center gap-2">
                  <FileCheck className="w-5 h-5 text-indigo-600" />
                  <span>{isPersian ? "توصیه‌ها و اقدامات بعدی" : "Recommendations"}</span>
                </h2>
                <div className="space-y-2">
                  {snapshot.recommendations.map((rec: any, idx: number) => (
                    <div
                      key={idx}
                      className="p-3.5 bg-blue-50/70 border border-blue-100 rounded-lg text-sm text-blue-900"
                    >
                      <div className="font-semibold text-blue-800 mb-0.5">{rec.title}</div>
                      <div className="text-xs text-blue-700">{rec.description}</div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* 7. Methodology & Limitations Notice */}
            <div className="p-4 bg-gray-50 rounded-xl border border-gray-200 text-xs text-gray-500 space-y-1">
              <div className="font-semibold text-gray-700">
                {isPersian ? "روش محاسبه و محدودیت‌ها" : "Calculation Methods & Limitations"}
              </div>
              <p>
                {snapshot.limitation_notice ||
                  (isPersian
                    ? "درصد آپتایم تنها بر اساس پایش‌های زمان‌بندی‌شده دوره‌های فعال محاسبه شده است و بازه‌های توقف دستی محاسبه نمی‌شوند. هارت‌بیت‌ها صرفاً تایید وصول هستند و به تنهایی بازگردانی داده‌های بک‌آپ را اثبات نمی‌کنند."
                    : "Uptime percentages exclude paused monitoring intervals and manual checks. Heartbeats confirm ping receipt only and do not independently prove backup recoverability. Missing data is not interpreted as uptime.")}
              </p>
            </div>
          </div>

          {/* Footer Branding */}
          {(branding.footer_text || branding.report_footer_text) && (
            <div className="p-6 bg-gray-50 border-t border-gray-200 text-center text-xs text-gray-500">
              {branding.footer_text || branding.report_footer_text}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
