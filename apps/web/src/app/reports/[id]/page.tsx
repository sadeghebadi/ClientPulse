"use client";

import React, { useState, useEffect, useCallback } from "react";
import Link from "next/link";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch, humanizeApiError, API_BASE_URL } from "@/lib/api";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { Input } from "@/components/ui/Input";
import {
  FileText,
  Download,
  Share2,
  ArrowLeft,
  ArrowRight,
  Calendar,
  Clock,
  ShieldCheck,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  Copy,
  Trash2,
  X,
  ExternalLink,
  Lock,
  Globe,
  Activity,
  Layers,
  Wrench,
  Radio,
  FileCheck,
} from "lucide-react";

interface ShareItem {
  id: string;
  report_id: string;
  expires_at?: string | null;
  created_at: string;
  created_by_name?: string;
  revoked_at?: string | null;
}

export default function ReportDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const unwrappedParams = React.use(params);
  const reportId = unwrappedParams.id;

  const { role } = useAuth();
  const { t, isRTL, locale } = useI18n();

  const [report, setReport] = useState<any | null>(null);
  const [snapshot, setSnapshot] = useState<any | null>(null);
  const [hasPdf, setHasPdf] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // PDF Generation State
  const [generatingPdf, setGeneratingPdf] = useState(false);

  // Share Modal State
  const [shareModalOpen, setShareModalOpen] = useState(false);
  const [shares, setShares] = useState<ShareItem[]>([]);
  const [sharesLoading, setSharesLoading] = useState(false);
  const [expiresInDays, setExpiresInDays] = useState<number>(30);
  const [creatingShare, setCreatingShare] = useState(false);
  const [newlyCreatedUrl, setNewlyCreatedUrl] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const canShare = role === "owner" || role === "admin" || role === "member";

  const fetchReport = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const res = await apiFetch<{ report: any; snapshot: any; has_pdf: boolean }>(
        `/api/v1/reports/${reportId}`
      );
      setReport(res.report);
      setSnapshot(res.snapshot);
      setHasPdf(res.has_pdf);
    } catch (err: unknown) {
      setError(humanizeApiError(err, isRTL));
    } finally {
      setLoading(false);
    }
  }, [reportId, isRTL]);

  useEffect(() => {
    let isMounted = true;
    apiFetch<{ report: any; snapshot: any; has_pdf: boolean }>(`/api/v1/reports/${reportId}`)
      .then((res) => {
        if (isMounted) {
          setReport(res.report);
          setSnapshot(res.snapshot);
          setHasPdf(res.has_pdf);
          setLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (isMounted) {
          setError(humanizeApiError(err, isRTL));
          setLoading(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [reportId, isRTL]);

  const fetchShares = useCallback(async () => {
    try {
      setSharesLoading(true);
      const res = await apiFetch<{ shares: ShareItem[] }>(`/api/v1/reports/${reportId}/shares`);
      setShares(res.shares || []);
    } catch {
      // ignore
    } finally {
      setSharesLoading(false);
    }
  }, [reportId]);

  useEffect(() => {
    if (!shareModalOpen) return;
    let isMounted = true;
    apiFetch<{ shares: ShareItem[] }>(`/api/v1/reports/${reportId}/shares`)
      .then((res) => {
        if (isMounted) {
          setShares(res.shares || []);
          setSharesLoading(false);
        }
      })
      .catch(() => {
        if (isMounted) {
          setSharesLoading(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [shareModalOpen, reportId]);

  const handleCreateShare = async () => {
    try {
      setCreatingShare(true);
      const res = await apiFetch<{
        share_token: string;
        share_url: string;
        expires_at?: string;
        share_id: string;
      }>(`/api/v1/reports/${reportId}/share`, {
        method: "POST",
        body: JSON.stringify({
          expires_in_days: expiresInDays > 0 ? expiresInDays : undefined,
        }),
      });

      const fullUrl = `${window.location.origin}${res.share_url}`;
      setNewlyCreatedUrl(fullUrl);
      fetchShares();
    } catch (err: unknown) {
      alert(humanizeApiError(err, isRTL));
    } finally {
      setCreatingShare(false);
    }
  };

  const handleRevokeShare = async (shareId: string) => {
    if (!confirm(isRTL ? "آیا از ابطال این لینک عمومی مطمئن هستید؟" : "Revoke this share link immediately?")) {
      return;
    }
    try {
      await apiFetch(`/api/v1/reports/${reportId}/revoke-share`, {
        method: "POST",
        body: JSON.stringify({ share_id: shareId }),
      });
      fetchShares();
      if (newlyCreatedUrl) setNewlyCreatedUrl(null);
    } catch (err: unknown) {
      alert(humanizeApiError(err, isRTL));
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2500);
  };

  const formatDate = (isoString?: string) => {
    if (!isoString) return "—";
    try {
      const d = new Date(isoString);
      return d.toLocaleDateString(locale === "fa" ? "fa-IR" : "en-US", {
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
      return d.toLocaleString(locale === "fa" ? "fa-IR" : "en-US", {
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
      <AppLayout>
        <div className="bg-white dark:bg-gray-800 p-12 rounded-xl text-center border border-gray-200 dark:border-gray-700">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600 mx-auto" />
          <p className="mt-3 text-sm text-gray-500">{t.loading}</p>
        </div>
      </AppLayout>
    );
  }

  if (error || !report || !snapshot) {
    return (
      <AppLayout>
        <div className="p-6 bg-red-50 dark:bg-red-900/20 rounded-xl border border-red-200 dark:border-red-800 text-red-700 dark:text-red-400">
          <p className="font-semibold">{error || t.noReportsFound}</p>
          <Link href="/reports" className="mt-4 inline-flex items-center text-sm font-medium underline">
            <ArrowLeft className="w-4 h-4 mr-1 rtl:rotate-180" />
            {isRTL ? "بازگشت به فهرست گزارش‌ها" : "Back to Reports"}
          </Link>
        </div>
      </AppLayout>
    );
  }

  const summary = snapshot.summary || {};
  const branding = snapshot.branding || {};
  const isPersianReport = snapshot.locale === "fa";

  return (
    <AppLayout>
      <div className="space-y-6 max-w-5xl mx-auto">
        {/* Navigation & Action Bar */}
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 pb-4 border-b border-gray-200 dark:border-gray-700">
          <Link
            href="/reports"
            className="inline-flex items-center text-sm text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
          >
            {isRTL ? <ArrowRight className="w-4 h-4 ml-1" /> : <ArrowLeft className="w-4 h-4 mr-1" />}
            {isRTL ? "بازگشت به فهرست گزارش‌ها" : "Back to Reports"}
          </Link>

          <div className="flex items-center gap-3">
            {canShare && (
              <Button
                variant="outline"
                onClick={() => setShareModalOpen(true)}
                className="flex items-center gap-1.5"
              >
                <Share2 className="w-4 h-4 text-indigo-600" />
                {t.shareReport}
              </Button>
            )}

            <a href={`${API_BASE_URL}/api/v1/reports/${report.id}/pdf`} download>
              <Button className="bg-indigo-600 hover:bg-indigo-700 text-white flex items-center gap-1.5">
                <Download className="w-4 h-4" />
                {t.downloadPDF}
              </Button>
            </a>
          </div>
        </div>

        {/* Report Document Shell */}
        <div
          dir={isPersianReport ? "rtl" : "ltr"}
          className={`bg-white dark:bg-gray-800 rounded-2xl shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden ${
            isPersianReport ? "font-sans" : ""
          }`}
        >
          {/* Branded Header */}
          <div
            className="p-8 border-b border-gray-200 dark:border-gray-700 relative"
            style={{
              borderTop: `6px solid ${branding.primary_color || "#3b82f6"}`,
            }}
          >
            <div className="flex flex-col md:flex-row md:items-center justify-between gap-6">
              <div>
                <div className="flex items-center gap-3 mb-2">
                  <Badge variant="outline" className="font-mono text-xs uppercase">
                    v{report.version_number} &bull; {snapshot.template_version || "1.0"}
                  </Badge>
                  <span className="text-xs text-gray-400">
                    {formatDate(report.period_start)} — {formatDate(report.period_end)}
                  </span>
                </div>
                <h1 className="text-2xl sm:text-3xl font-extrabold text-gray-900 dark:text-gray-100">
                  {snapshot.title}
                </h1>
                <p className="mt-1 text-base text-gray-600 dark:text-gray-300">
                  {isPersianReport ? "مشتری:" : "Client:"}{" "}
                  <span className="font-semibold text-gray-900 dark:text-gray-100">
                    {snapshot.client?.name}
                  </span>
                </p>
              </div>

              {/* Agency Branding Display */}
              <div className="text-left rtl:text-right md:text-right rtl:md:text-left border-t md:border-t-0 pt-4 md:pt-0 border-gray-100 dark:border-gray-700">
                <div className="text-lg font-bold text-gray-900 dark:text-gray-100">
                  {branding.display_name || "ClientPulse Agency"}
                </div>
                {branding.website_url && (
                  <div className="text-xs text-indigo-600 dark:text-indigo-400 mt-0.5">
                    {branding.website_url}
                  </div>
                )}
                {branding.contact_email && (
                  <div className="text-xs text-gray-500 dark:text-gray-400">
                    {branding.contact_email}
                  </div>
                )}
                <div className="text-xs text-gray-400 mt-1">
                  {isPersianReport ? "تاریخ صدور:" : "Generated:"} {formatDateTime(report.generated_at)}
                </div>
              </div>
            </div>
          </div>

          <div className="p-8 space-y-8">
            {/* 1. Executive Summary Cards */}
            <div>
              <h2 className="text-lg font-bold text-gray-900 dark:text-gray-100 mb-4 flex items-center gap-2">
                <Activity className="w-5 h-5 text-indigo-600" />
                {t.executiveSummary}
              </h2>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div className="p-4 rounded-xl bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-700">
                  <div className="text-xs text-gray-500 dark:text-gray-400">
                    {isPersianReport ? "آپتایم میانگین" : "Avg Uptime"}
                  </div>
                  <div className="text-2xl font-black text-gray-900 dark:text-gray-100 mt-1">
                    {summary.average_uptime != null
                      ? `${summary.average_uptime.toFixed(2)}%`
                      : t.insufficientData}
                  </div>
                  <div className="text-xs text-gray-400 mt-1">
                    {isPersianReport ? "در بازه فعال مانیتورها" : "During active intervals"}
                  </div>
                </div>

                <div className="p-4 rounded-xl bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-700">
                  <div className="text-xs text-gray-500 dark:text-gray-400">
                    {t.coveragePercentage}
                  </div>
                  <div className="text-2xl font-black text-indigo-600 dark:text-indigo-400 mt-1">
                    {snapshot.monitors && snapshot.monitors.length > 0 && snapshot.monitors[0].coverage_percentage != null
                      ? `${snapshot.monitors[0].coverage_percentage.toFixed(1)}%`
                      : "100.0%"}
                  </div>
                  <div className="text-xs text-gray-400 mt-1">
                    {isPersianReport ? "پوشش کل بازه تقویمی" : "Monitoring coverage ratio"}
                  </div>
                </div>

                <div className="p-4 rounded-xl bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-700">
                  <div className="text-xs text-gray-500 dark:text-gray-400">
                    {isPersianReport ? "رخدادهای قطعی" : "Incidents"}
                  </div>
                  <div className="text-2xl font-black text-gray-900 dark:text-gray-100 mt-1">
                    {summary.total_incidents ?? 0}
                  </div>
                  <div className="text-xs text-gray-400 mt-1">
                    {summary.total_downtime_sec ? `${Math.round(summary.total_downtime_sec / 60)} min total` : "No downtime"}
                  </div>
                </div>

                <div className="p-4 rounded-xl bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-700">
                  <div className="text-xs text-gray-500 dark:text-gray-400">
                    {t.maintenance}
                  </div>
                  <div className="text-2xl font-black text-gray-900 dark:text-gray-100 mt-1">
                    {summary.total_activities ?? (snapshot.maintenance ? snapshot.maintenance.length : 0)}
                  </div>
                  <div className="text-xs text-gray-400 mt-1">
                    {isPersianReport ? "اقدام نگهداری ثبت‌شده" : "Recorded activities"}
                  </div>
                </div>
              </div>
            </div>

            {/* 2. Assets & Monitors Table */}
            <div>
              <h2 className="text-lg font-bold text-gray-900 dark:text-gray-100 mb-4 flex items-center gap-2">
                <Globe className="w-5 h-5 text-indigo-600" />
                {isPersianReport ? "وضعیت وب‌سایت‌ها و مانیتورها" : "Websites & Monitoring Performance"}
              </h2>
              {snapshot.monitors && snapshot.monitors.length > 0 ? (
                <div className="overflow-x-auto border border-gray-200 dark:border-gray-700 rounded-xl">
                  <table className="w-full text-sm text-left rtl:text-right">
                    <thead className="bg-gray-50 dark:bg-gray-700/50 text-gray-700 dark:text-gray-300">
                      <tr>
                        <th className="px-4 py-3">{isPersianReport ? "سرویس / مانیتور" : "Target / Monitor"}</th>
                        <th className="px-4 py-3">{isPersianReport ? "آپتایم" : "Uptime %"}</th>
                        <th className="px-4 py-3">{t.coveragePercentage}</th>
                        <th className="px-4 py-3">{isPersianReport ? "پاسخگویی (میانگین / میانه)" : "Latency (Avg/Med)"}</th>
                        <th className="px-4 py-3">{isPersianReport ? "وضعیت بررسی‌ها" : "Total Checks"}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                      {snapshot.monitors.map((m: any, idx: number) => (
                        <tr key={idx} className="hover:bg-gray-50/50 dark:hover:bg-gray-700/20">
                          <td className="px-4 py-3">
                            <div className="font-semibold text-gray-900 dark:text-gray-100">{m.name || m.asset_name || m.url}</div>
                            <div className="text-xs text-gray-500 font-mono truncate max-w-xs">{m.url}</div>
                          </td>
                          <td className="px-4 py-3">
                            {m.uptime_percentage != null ? (
                              <span className="font-semibold text-emerald-600 dark:text-emerald-400">
                                {m.uptime_percentage.toFixed(2)}%
                              </span>
                            ) : (
                              <span className="text-amber-600 dark:text-amber-400 text-xs font-semibold">
                                {t.insufficientData}
                              </span>
                            )}
                          </td>
                          <td className="px-4 py-3 text-xs text-gray-600 dark:text-gray-300">
                            {m.coverage_percentage != null ? `${m.coverage_percentage.toFixed(1)}%` : "—"}
                          </td>
                          <td className="px-4 py-3 text-xs font-mono">
                            {m.avg_response_time_ms != null ? `${m.avg_response_time_ms}ms / ${m.median_response_time_ms ?? "—"}ms` : "—"}
                          </td>
                          <td className="px-4 py-3 text-xs">
                            <span className="font-medium text-gray-700 dark:text-gray-300">
                              {m.total_checks} {isPersianReport ? "بررسی" : "checks"}
                            </span>
                            {m.failed_checks > 0 && (
                              <span className="text-red-500 ml-1">({m.failed_checks} {isPersianReport ? "خطا" : "failed"})</span>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="p-4 bg-gray-50 dark:bg-gray-900/30 rounded-xl text-sm text-gray-500">
                  {t.notMonitored}
                </div>
              )}
            </div>

            {/* 3. Incidents History */}
            <div>
              <h2 className="text-lg font-bold text-gray-900 dark:text-gray-100 mb-4 flex items-center gap-2">
                <AlertTriangle className="w-5 h-5 text-amber-500" />
                {isPersianReport ? "تاریخچه رخدادها و قطعی‌ها" : "Incidents History"}
              </h2>
              {snapshot.incidents && snapshot.incidents.length > 0 ? (
                <div className="overflow-x-auto border border-gray-200 dark:border-gray-700 rounded-xl">
                  <table className="w-full text-sm text-left rtl:text-right">
                    <thead className="bg-gray-50 dark:bg-gray-700/50 text-gray-700 dark:text-gray-300">
                      <tr>
                        <th className="px-4 py-3">{isPersianReport ? "سرویس" : "Target"}</th>
                        <th className="px-4 py-3">{isPersianReport ? "زمان وقوع" : "Confirmed At"}</th>
                        <th className="px-4 py-3">{isPersianReport ? "مدت زمان" : "Duration"}</th>
                        <th className="px-4 py-3">{isPersianReport ? "علت رخداد" : "Cause / Error"}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                      {snapshot.incidents.map((inc: any, idx: number) => (
                        <tr key={idx}>
                          <td className="px-4 py-3 font-medium">{inc.target_name || "Monitor"}</td>
                          <td className="px-4 py-3 text-xs text-gray-500">{formatDateTime(inc.confirmed_at)}</td>
                          <td className="px-4 py-3 text-xs">{inc.duration_minutes ? `${inc.duration_minutes} min` : "—"}</td>
                          <td className="px-4 py-3 text-xs text-gray-600 dark:text-gray-300">{inc.summary || inc.failure_category || "Service Interruption"}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="p-4 bg-emerald-50 dark:bg-emerald-950/20 text-emerald-800 dark:text-emerald-300 rounded-xl text-sm flex items-center gap-2">
                  <CheckCircle2 className="w-4 h-4 text-emerald-600 flex-shrink-0" />
                  {isPersianReport ? "هیچ رخداد قطعی در این بازه زمانی ثبت نشده است." : "Zero downtime incidents recorded during this reporting period."}
                </div>
              )}
            </div>

            {/* 4. Heartbeats & Cron Backups */}
            <div>
              <h2 className="text-lg font-bold text-gray-900 dark:text-gray-100 mb-4 flex items-center gap-2">
                <Radio className="w-5 h-5 text-indigo-600" />
                {isPersianReport ? "وضعیت هارت‌بیت و بک‌آپ‌ها" : "Heartbeats & Scheduled Tasks"}
              </h2>
              {snapshot.heartbeats && snapshot.heartbeats.length > 0 ? (
                <div className="overflow-x-auto border border-gray-200 dark:border-gray-700 rounded-xl">
                  <table className="w-full text-sm text-left rtl:text-right">
                    <thead className="bg-gray-50 dark:bg-gray-700/50 text-gray-700 dark:text-gray-300">
                      <tr>
                        <th className="px-4 py-3">{isPersianReport ? "نام وظیفه" : "Task Name"}</th>
                        <th className="px-4 py-3">{t.status}</th>
                        <th className="px-4 py-3">{isPersianReport ? "آخرین وصول" : "Last Received"}</th>
                        <th className="px-4 py-3">{isPersianReport ? "دوره انتظار" : "Expected Interval"}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                      {snapshot.heartbeats.map((hb: any, idx: number) => (
                        <tr key={idx}>
                          <td className="px-4 py-3 font-semibold">{hb.name}</td>
                          <td className="px-4 py-3">
                            <Badge
                              variant={hb.status === "healthy" ? "success" : hb.status === "missed" ? "danger" : "default"}
                              className="text-xs"
                            >
                              {hb.status}
                            </Badge>
                          </td>
                          <td className="px-4 py-3 text-xs text-gray-500">{formatDateTime(hb.last_received_at)}</td>
                          <td className="px-4 py-3 text-xs">{hb.expected_interval || "—"}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="p-4 bg-gray-50 dark:bg-gray-900/30 rounded-xl text-sm text-gray-500">
                  {t.notMonitored}
                </div>
              )}
            </div>

            {/* 5. Maintenance Activities */}
            <div>
              <h2 className="text-lg font-bold text-gray-900 dark:text-gray-100 mb-4 flex items-center gap-2">
                <Wrench className="w-5 h-5 text-indigo-600" />
                {t.maintenance}
              </h2>
              {snapshot.maintenance && snapshot.maintenance.length > 0 ? (
                <div className="space-y-3">
                  {snapshot.maintenance.map((act: any, idx: number) => (
                    <div
                      key={idx}
                      className="p-4 rounded-xl border border-gray-200 dark:border-gray-700 bg-gray-50/50 dark:bg-gray-900/30"
                    >
                      <div className="flex flex-wrap items-center justify-between gap-2 mb-1.5">
                        <div className="flex items-center gap-2">
                          <span className="font-semibold text-gray-900 dark:text-gray-100">{act.title}</span>
                          <Badge variant="outline" className="text-xs">
                            {act.category}
                          </Badge>
                        </div>
                        <span className="text-xs text-gray-500">
                          {formatDateTime(act.performed_at)}
                          {act.duration_minutes ? ` • ${act.duration_minutes} min` : ""}
                        </span>
                      </div>
                      <p className="text-sm text-gray-600 dark:text-gray-300">{act.description}</p>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="p-4 bg-gray-50 dark:bg-gray-900/30 rounded-xl text-sm text-gray-500">
                  {t.noActivitiesRecorded}
                </div>
              )}
            </div>

            {/* 6. Deterministic Recommendations */}
            {snapshot.recommendations && snapshot.recommendations.length > 0 && (
              <div>
                <h2 className="text-lg font-bold text-gray-900 dark:text-gray-100 mb-4 flex items-center gap-2">
                  <FileCheck className="w-5 h-5 text-indigo-600" />
                  {t.recommendations}
                </h2>
                <div className="space-y-2">
                  {snapshot.recommendations.map((rec: any, idx: number) => (
                    <div
                      key={idx}
                      className="p-3.5 rounded-lg bg-blue-50/70 dark:bg-blue-950/20 border border-blue-100 dark:border-blue-900/30 text-sm text-blue-900 dark:text-blue-200"
                    >
                      <div className="font-semibold text-blue-800 dark:text-blue-300 mb-0.5">
                        {rec.title}
                      </div>
                      <div className="text-xs text-blue-700 dark:text-blue-400">
                        {rec.description}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* 7. Methodology & Limitations Notice */}
            <div className="p-4 rounded-xl bg-gray-50 dark:bg-gray-900/40 border border-gray-200 dark:border-gray-700 text-xs text-gray-500 dark:text-gray-400 space-y-1">
              <div className="font-semibold text-gray-700 dark:text-gray-300">{t.limitationMethods}</div>
              <p>
                {snapshot.limitation_notice ||
                  (isPersianReport
                    ? "محاسبه درصد آپتایم بر مبنای دوره‌های پایش فعال انجام شده و بازه‌های متوقف‌شده یا تست‌های دستی حذف شده‌اند. هارت‌بیت‌ها تنها اعلام وصول موفق هستند و به تنهایی بازگردانی داده‌های بک‌آپ را اثبات نمی‌کنند."
                    : "Uptime percentages exclude paused monitoring intervals and manual checks. Heartbeats confirm ping receipt only and do not independently prove backup recoverability. Missing data is not interpreted as uptime.")}
              </p>
            </div>
          </div>

          {/* Footer Branding */}
          {(branding.footer_text || branding.report_footer_text) && (
            <div className="p-6 bg-gray-50 dark:bg-gray-900/60 border-t border-gray-200 dark:border-gray-700 text-center text-xs text-gray-500">
              {branding.footer_text || branding.report_footer_text}
            </div>
          )}
          {branding.report_footer_text && (
            <div className="p-6 bg-gray-50 dark:bg-gray-900/60 border-t border-gray-200 dark:border-gray-700 text-center text-xs text-gray-500">
              {branding.report_footer_text}
            </div>
          )}
        </div>

        {/* Secure Share Modal */}
        {shareModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm animate-fade-in">
            <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-xl w-full max-w-lg overflow-hidden border border-gray-200 dark:border-gray-700">
              <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700">
                <h3 className="text-lg font-bold text-gray-900 dark:text-gray-100 flex items-center gap-2">
                  <Share2 className="w-5 h-5 text-indigo-600" />
                  {t.shareReport}
                </h3>
                <button
                  onClick={() => {
                    setShareModalOpen(false);
                    setNewlyCreatedUrl(null);
                  }}
                  className="text-gray-400 hover:text-gray-500 rounded-lg p-1"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              <div className="p-6 space-y-4">
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  {isRTL
                    ? "لینک عمومی گزارش به صورت فقط‌خواندنی و با توکن رمزنگاری‌شده ۳۲ بایتی تولید می‌شود. شناسه پایگاه‌داده فاش نشده و در صورت نیاز بلافاصله قابل ابطال است."
                    : "Cryptographically secure 32-byte public token. Links are read-only and can be revoked immediately."}
                </p>

                {/* Expiry Input */}
                <div>
                  <label className="block text-xs font-semibold uppercase text-gray-700 dark:text-gray-300 mb-1">
                    {t.shareExpiryDays}
                  </label>
                  <Input
                    type="number"
                    min="1"
                    max="365"
                    value={expiresInDays}
                    onChange={(e) => setExpiresInDays(parseInt(e.target.value) || 0)}
                    placeholder="30"
                  />
                </div>

                <Button
                  onClick={handleCreateShare}
                  disabled={creatingShare}
                  className="w-full bg-indigo-600 hover:bg-indigo-700 text-white flex items-center justify-center gap-2"
                >
                  <Lock className="w-4 h-4" />
                  {creatingShare ? t.loading : t.createShareLink}
                </Button>

                {/* Newly Created Link Box */}
                {newlyCreatedUrl && (
                  <div className="p-3 bg-emerald-50 dark:bg-emerald-950/30 border border-emerald-200 dark:border-emerald-800 rounded-xl space-y-2">
                    <span className="text-xs font-semibold text-emerald-800 dark:text-emerald-300 block">
                      {isRTL ? "لینک جدید عمومی با موفقیت ایجاد شد:" : "New Public Share Link Created:"}
                    </span>
                    <div className="flex items-center gap-2">
                      <input
                        type="text"
                        readOnly
                        value={newlyCreatedUrl}
                        className="w-full px-2.5 py-1.5 text-xs font-mono bg-white dark:bg-gray-900 border border-emerald-300 dark:border-emerald-700 rounded-lg"
                      />
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => copyToClipboard(newlyCreatedUrl)}
                        className="flex-shrink-0"
                      >
                        <Copy className="w-3.5 h-3.5 mr-1" />
                        {copied ? t.copiedToClipboard : t.copyShareLink}
                      </Button>
                    </div>
                  </div>
                )}

                {/* Active Shares List */}
                <div className="pt-2 border-t border-gray-200 dark:border-gray-700">
                  <h4 className="text-xs font-semibold uppercase text-gray-700 dark:text-gray-300 mb-2">
                    {t.activeShares}
                  </h4>
                  {sharesLoading ? (
                    <div className="text-xs text-gray-400 py-2">{t.loading}</div>
                  ) : shares.filter((s) => !s.revoked_at).length === 0 ? (
                    <div className="text-xs text-gray-400 py-2">
                      {isRTL ? "هیچ لینک فعالی وجود ندارد." : "No active share links."}
                    </div>
                  ) : (
                    <div className="space-y-2 max-h-40 overflow-y-auto">
                      {shares
                        .filter((s) => !s.revoked_at)
                        .map((s) => (
                          <div
                            key={s.id}
                            className="p-2.5 bg-gray-50 dark:bg-gray-900/50 rounded-lg border border-gray-200 dark:border-gray-700 flex items-center justify-between text-xs"
                          >
                            <div>
                              <div className="font-medium text-gray-800 dark:text-gray-200">
                                {isRTL ? "انقضا:" : "Expires:"}{" "}
                                {s.expires_at ? formatDate(s.expires_at) : (isRTL ? "دائمی" : "Never")}
                              </div>
                              <div className="text-gray-400 text-[10px]">
                                {formatDateTime(s.created_at)}
                              </div>
                            </div>
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => handleRevokeShare(s.id)}
                              className="text-red-600 hover:text-red-700 border-red-200 h-7 text-xs px-2"
                            >
                              <Trash2 className="w-3 h-3 mr-1" />
                              {t.revokeShareLink}
                            </Button>
                          </div>
                        ))}
                    </div>
                  )}
                </div>
              </div>

              <div className="px-6 py-3 bg-gray-50 dark:bg-gray-800/50 border-t border-gray-200 dark:border-gray-700 flex justify-end">
                <Button variant="outline" onClick={() => setShareModalOpen(false)}>
                  {t.close}
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </AppLayout>
  );
}
