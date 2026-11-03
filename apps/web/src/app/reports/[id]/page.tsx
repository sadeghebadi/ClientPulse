"use client";

import React, { useState, useEffect, useCallback, useRef } from "react";
import Link from "next/link";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch, humanizeApiError, API_BASE_URL } from "@/lib/api";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { Input } from "@/components/ui/Input";
import { ConfirmModal } from "@/components/ui/ConfirmModal";
import { ErrorState } from "@/components/ui/ErrorState";
import { LoadingSkeleton, MetricSkeleton, TableSkeleton } from "@/components/ui/LoadingSkeleton";
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
  Building2,
  Info,
  Check,
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

  // Share Modal State
  const [shareModalOpen, setShareModalOpen] = useState(false);
  const [shares, setShares] = useState<ShareItem[]>([]);
  const [sharesLoading, setSharesLoading] = useState(false);
  const [expiresInDays, setExpiresInDays] = useState<number>(30);
  const [creatingShare, setCreatingShare] = useState(false);
  const [newlyCreatedUrl, setNewlyCreatedUrl] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  // Revoke Share Confirmation Modal State
  const [revokeTargetId, setRevokeTargetId] = useState<string | null>(null);
  const [revokingShare, setRevokingShare] = useState(false);

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
    fetchReport();
  }, [fetchReport]);

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
    fetchShares();
  }, [shareModalOpen, fetchShares]);

  // Modal Escape handler
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && shareModalOpen && !creatingShare && !revokingShare) {
        setShareModalOpen(false);
        setNewlyCreatedUrl(null);
      }
    };
    if (shareModalOpen) {
      window.addEventListener("keydown", handleKeyDown);
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "unset";
    }
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      document.body.style.overflow = "unset";
    };
  }, [shareModalOpen, creatingShare, revokingShare]);

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

  const handleConfirmRevoke = async () => {
    if (!revokeTargetId) return;
    try {
      setRevokingShare(true);
      await apiFetch(`/api/v1/reports/${reportId}/revoke-share`, {
        method: "POST",
        body: JSON.stringify({ share_id: revokeTargetId }),
      });
      fetchShares();
      if (newlyCreatedUrl) setNewlyCreatedUrl(null);
      setRevokeTargetId(null);
    } catch (err: unknown) {
      alert(humanizeApiError(err, isRTL));
    } finally {
      setRevokingShare(false);
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
        <div className="space-y-6 max-w-5xl mx-auto">
          <LoadingSkeleton className="h-10 w-48" />
          <MetricSkeleton />
          <TableSkeleton rows={4} cols={4} />
        </div>
      </AppLayout>
    );
  }

  if (error || !report || !snapshot) {
    return (
      <AppLayout>
        <div className="max-w-2xl mx-auto pt-8">
          <ErrorState error={error || t.noReportsFound} onRetry={fetchReport} />
          <div className="mt-4">
            <Link
              href="/reports"
              className="inline-flex items-center text-xs font-semibold text-indigo-600 hover:text-indigo-800"
            >
              {isRTL ? <ArrowRight className="w-4 h-4 ml-1" /> : <ArrowLeft className="w-4 h-4 mr-1" />}
              {t.backToReports}
            </Link>
          </div>
        </div>
      </AppLayout>
    );
  }

  const summary = snapshot.summary || {};
  const branding = snapshot.branding || {};
  const isPersianReport = snapshot.locale === "fa";
  const primaryColor = branding.primary_color || "#4f46e5";

  return (
    <AppLayout>
      <div className="space-y-6 max-w-5xl mx-auto">
        {/* Navigation & Action Bar */}
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 pb-4 border-b border-slate-200">
          <div>
            <Link
              href="/reports"
              className="inline-flex items-center text-xs font-semibold text-slate-500 hover:text-slate-800 transition-colors"
            >
              {isRTL ? <ArrowRight className="w-4 h-4 ml-1" /> : <ArrowLeft className="w-4 h-4 mr-1" />}
              {t.backToReports}
            </Link>
            <h1 className="text-xl sm:text-2xl font-black text-slate-900 tracking-tight mt-1">
              {report.title}
            </h1>
          </div>

          <div className="flex items-center gap-2.5 flex-wrap">
            {canShare && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShareModalOpen(true)}
                icon={<Share2 className="w-4 h-4 text-indigo-600" />}
                className="text-xs font-semibold"
              >
                {t.shareReport}
              </Button>
            )}

            <a
              href={`${API_BASE_URL}/api/v1/reports/${report.id}/pdf`}
              download
              className="inline-flex items-center justify-center gap-1.5 px-4 py-2 text-xs font-semibold bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl shadow-xs transition-colors"
            >
              <Download className="w-4 h-4" />
              <span>{t.downloadPDF}</span>
            </a>
          </div>
        </div>

        {/* Notice: Historical Deterministic Snapshot vs Live Monitoring Telemetry */}
        <div className="p-4 bg-slate-50 rounded-2xl border border-slate-200 text-xs text-slate-700 flex items-start gap-3">
          <Info className="w-5 h-5 text-indigo-600 shrink-0 mt-0.5" />
          <div className="space-y-1">
            <div className="flex items-center gap-2 flex-wrap">
              <span className="font-bold text-slate-900">{t.snapshotNotice}</span>
              <Badge variant="outline" className="text-[10px] font-mono">
                {t.snapshotBadge}
              </Badge>
              <span className="text-slate-400 font-mono" dir="ltr">
                {formatDate(report.period_start)} – {formatDate(report.period_end)}
              </span>
            </div>
            <p className="text-slate-600 leading-relaxed text-[11px] sm:text-xs">
              {t.snapshotDescription}
            </p>
          </div>
        </div>

        {/* Report Document Shell */}
        <div
          dir={isPersianReport ? "rtl" : "ltr"}
          className={`bg-white rounded-2xl shadow-sm border border-slate-200 overflow-hidden ${
            isPersianReport ? "font-sans" : ""
          }`}
        >
          {/* Branded Header */}
          <div
            className="p-6 sm:p-8 border-b border-slate-200 relative"
            style={{
              borderTop: `6px solid ${primaryColor}`,
            }}
          >
            <div className="flex flex-col md:flex-row md:items-start justify-between gap-6">
              <div>
                <div className="flex items-center gap-2.5 mb-2 flex-wrap">
                  <Badge variant="outline" className="font-mono text-xs uppercase font-bold">
                    v{report.version_number} &bull; {snapshot.template_version || "1.0"}
                  </Badge>
                  <span className="text-xs text-slate-500 font-mono" dir="ltr">
                    {formatDate(report.period_start)} — {formatDate(report.period_end)}
                  </span>
                </div>
                <h2 className="text-2xl sm:text-3xl font-black text-slate-900 tracking-tight">
                  {snapshot.title}
                </h2>
                <div className="mt-2 text-sm text-slate-600 flex items-center gap-1.5">
                  <span className="font-medium">{t.preparedFor}:</span>
                  <span className="font-bold text-slate-900 flex items-center gap-1">
                    <Building2 className="w-4 h-4 text-slate-400 shrink-0" />
                    {snapshot.client?.name}
                  </span>
                </div>
              </div>

              {/* Agency Branding Display */}
              <div className="text-left rtl:text-right md:text-right rtl:md:text-left border-t md:border-t-0 pt-4 md:pt-0 border-slate-100">
                <div className="text-base sm:text-lg font-black text-slate-900">
                  {branding.display_name || "ClientPulse Agency"}
                </div>
                {branding.website_url && (
                  <div className="text-xs text-indigo-600 font-medium mt-0.5" dir="ltr">
                    {branding.website_url}
                  </div>
                )}
                {branding.contact_email && (
                  <div className="text-xs text-slate-500 mt-0.5" dir="ltr">
                    {branding.contact_email}
                  </div>
                )}
                <div className="text-[11px] text-slate-400 mt-1.5" dir="ltr">
                  {t.generatedAt}: {formatDateTime(report.generated_at)}
                </div>
              </div>
            </div>
          </div>

          <div className="p-6 sm:p-8 space-y-8">
            {/* 1. Executive Summary Cards */}
            <div>
              <h3 className="text-base sm:text-lg font-bold text-slate-900 mb-4 flex items-center gap-2">
                <Activity className="w-5 h-5 text-indigo-600 shrink-0" />
                <span>{t.executiveSummary}</span>
              </h3>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-3 sm:gap-4">
                <div className="p-4 rounded-xl bg-slate-50 border border-slate-200">
                  <div className="text-xs font-semibold text-slate-500">
                    {t.previewAvgUptime}
                  </div>
                  <div className="text-2xl font-black text-slate-900 mt-1" dir="ltr">
                    {summary.average_uptime != null
                      ? `${summary.average_uptime.toFixed(2)}%`
                      : t.insufficientData}
                  </div>
                  <div className="text-[11px] text-slate-400 mt-0.5">
                    {isPersianReport ? "در بازه فعال مانیتورها" : "During active intervals"}
                  </div>
                </div>

                <div className="p-4 rounded-xl bg-slate-50 border border-slate-200">
                  <div className="text-xs font-semibold text-slate-500">
                    {t.coveragePercentage}
                  </div>
                  <div className="text-2xl font-black text-indigo-600 mt-1" dir="ltr">
                    {snapshot.monitors && snapshot.monitors.length > 0 && snapshot.monitors[0].coverage_percentage != null
                      ? `${snapshot.monitors[0].coverage_percentage.toFixed(1)}%`
                      : "100.0%"}
                  </div>
                  <div className="text-[11px] text-slate-400 mt-0.5">
                    {isPersianReport ? "پوشش کل بازه تقویمی" : "Monitoring coverage ratio"}
                  </div>
                </div>

                <div className="p-4 rounded-xl bg-slate-50 border border-slate-200">
                  <div className="text-xs font-semibold text-slate-500">
                    {t.previewIncidents}
                  </div>
                  <div className="text-2xl font-black text-slate-900 mt-1">
                    {summary.total_incidents ?? 0}
                  </div>
                  <div className="text-[11px] text-slate-400 mt-0.5">
                    {summary.total_downtime_sec
                      ? `${Math.round(summary.total_downtime_sec / 60)} min total`
                      : "Zero downtime"}
                  </div>
                </div>

                <div className="p-4 rounded-xl bg-slate-50 border border-slate-200">
                  <div className="text-xs font-semibold text-slate-500">
                    {t.previewMaintenance}
                  </div>
                  <div className="text-2xl font-black text-slate-900 mt-1">
                    {summary.total_activities ?? (snapshot.maintenance ? snapshot.maintenance.length : 0)}
                  </div>
                  <div className="text-[11px] text-slate-400 mt-0.5">
                    {isPersianReport ? "اقدام نگهداری ثبت‌شده" : "Recorded activities"}
                  </div>
                </div>
              </div>
            </div>

            {/* 2. Assets & Monitors Performance */}
            <div>
              <h3 className="text-base sm:text-lg font-bold text-slate-900 mb-4 flex items-center gap-2">
                <Globe className="w-5 h-5 text-indigo-600 shrink-0" />
                <span>{isPersianReport ? "وضعیت وب‌سایت‌ها و مانیتورها" : "Websites & Monitoring Performance"}</span>
              </h3>
              {snapshot.monitors && snapshot.monitors.length > 0 ? (
                <>
                  {/* Desktop Table */}
                  <div className="hidden sm:block overflow-x-auto border border-slate-200 rounded-xl">
                    <table className="w-full text-sm text-left rtl:text-right">
                      <thead className="bg-slate-50 text-slate-700 text-xs uppercase tracking-wider font-semibold">
                        <tr>
                          <th className="px-4 py-3">{isPersianReport ? "سرویس / مانیتور" : "Target / Monitor"}</th>
                          <th className="px-4 py-3">{isPersianReport ? "آپتایم" : "Uptime %"}</th>
                          <th className="px-4 py-3">{t.coveragePercentage}</th>
                          <th className="px-4 py-3">{isPersianReport ? "پاسخگویی (میانگین/میانه)" : "Latency (Avg/Med)"}</th>
                          <th className="px-4 py-3">{isPersianReport ? "بررسی‌ها" : "Total Checks"}</th>
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
                                  {t.insufficientData}
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
                                {m.total_checks} {isPersianReport ? "بررسی" : "checks"}
                              </span>
                              {m.failed_checks > 0 && (
                                <span className="text-rose-600 ml-1">({m.failed_checks} {isPersianReport ? "خطا" : "failed"})</span>
                              )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>

                  {/* Mobile Monitor Cards (<640px) */}
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
                              <span className="text-amber-600 text-[10px]">{t.insufficientData}</span>
                            )}
                          </div>
                        </div>
                        <div className="flex items-center justify-between text-[11px] text-slate-500 pt-1.5 border-t border-slate-200 font-mono" dir="ltr">
                          <span>{m.avg_response_time_ms != null ? `${m.avg_response_time_ms}ms avg` : "—"}</span>
                          <span>{m.total_checks} checks {m.failed_checks > 0 ? `(${m.failed_checks} failed)` : ""}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                </>
              ) : (
                <div className="p-4 bg-slate-50 rounded-xl text-xs text-slate-500">
                  {t.notMonitored}
                </div>
              )}
            </div>

            {/* 3. Incidents History */}
            <div>
              <h3 className="text-base sm:text-lg font-bold text-slate-900 mb-4 flex items-center gap-2">
                <AlertTriangle className="w-5 h-5 text-amber-500 shrink-0" />
                <span>{isPersianReport ? "تاریخچه رخدادها و قطعی‌ها" : "Incidents History"}</span>
              </h3>
              {snapshot.incidents && snapshot.incidents.length > 0 ? (
                <div className="overflow-x-auto border border-slate-200 rounded-xl">
                  <table className="w-full text-sm text-left rtl:text-right">
                    <thead className="bg-slate-50 text-slate-700 text-xs uppercase tracking-wider font-semibold">
                      <tr>
                        <th className="px-4 py-3">{isPersianReport ? "سرویس" : "Target"}</th>
                        <th className="px-4 py-3">{isPersianReport ? "زمان وقوع" : "Confirmed At"}</th>
                        <th className="px-4 py-3">{isPersianReport ? "مدت زمان" : "Duration"}</th>
                        <th className="px-4 py-3">{isPersianReport ? "علت رخداد" : "Cause / Error"}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                      {snapshot.incidents.map((inc: any, idx: number) => (
                        <tr key={idx}>
                          <td className="px-4 py-3 font-medium text-slate-900">{inc.target_name || "Monitor"}</td>
                          <td className="px-4 py-3 text-xs text-slate-500 font-mono" dir="ltr">
                            {formatDateTime(inc.confirmed_at)}
                          </td>
                          <td className="px-4 py-3 text-xs" dir="ltr">
                            {inc.duration_minutes ? `${inc.duration_minutes} min` : "—"}
                          </td>
                          <td className="px-4 py-3 text-xs text-slate-600">
                            {inc.summary || inc.failure_category || "Service Interruption"}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="p-4 bg-emerald-50 text-emerald-800 rounded-xl text-xs sm:text-sm flex items-center gap-2 border border-emerald-200">
                  <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
                  <span>
                    {isPersianReport
                      ? "هیچ رخداد قطعی در این بازه زمانی ثبت نشده است (آپتایم کامل)."
                      : "Zero outages or service interruptions recorded during this reporting period (100% Uptime)."}
                  </span>
                </div>
              )}
            </div>

            {/* 4. Heartbeats & Scheduled Cron Jobs */}
            <div>
              <h3 className="text-base sm:text-lg font-bold text-slate-900 mb-4 flex items-center gap-2">
                <Radio className="w-5 h-5 text-indigo-600 shrink-0" />
                <span>{isPersianReport ? "وضعیت کران‌جاب‌ها و بک‌آپ‌ها" : "Heartbeats & Scheduled Tasks"}</span>
              </h3>
              {snapshot.heartbeats && snapshot.heartbeats.length > 0 ? (
                <div className="overflow-x-auto border border-slate-200 rounded-xl">
                  <table className="w-full text-sm text-left rtl:text-right">
                    <thead className="bg-slate-50 text-slate-700 text-xs uppercase tracking-wider font-semibold">
                      <tr>
                        <th className="px-4 py-3">{isPersianReport ? "نام وظیفه" : "Task Name"}</th>
                        <th className="px-4 py-3">{t.status}</th>
                        <th className="px-4 py-3">{isPersianReport ? "آخرین وصول" : "Last Received"}</th>
                        <th className="px-4 py-3">{isPersianReport ? "دوره انتظار" : "Expected Interval"}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                      {snapshot.heartbeats.map((hb: any, idx: number) => (
                        <tr key={idx}>
                          <td className="px-4 py-3 font-semibold text-slate-900">{hb.name}</td>
                          <td className="px-4 py-3">
                            <Badge
                              variant={hb.status === "healthy" ? "success" : hb.status === "missed" ? "danger" : "default"}
                              className="text-xs"
                            >
                              {hb.status}
                            </Badge>
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
                  {t.notMonitored}
                </div>
              )}
            </div>

            {/* 5. Maintenance Activities */}
            <div>
              <h3 className="text-base sm:text-lg font-bold text-slate-900 mb-4 flex items-center gap-2">
                <Wrench className="w-5 h-5 text-indigo-600 shrink-0" />
                <span>{t.maintenance}</span>
              </h3>
              {snapshot.maintenance && snapshot.maintenance.length > 0 ? (
                <div className="space-y-3">
                  {snapshot.maintenance.map((act: any, idx: number) => (
                    <div
                      key={idx}
                      className="p-4 rounded-xl border border-slate-200 bg-slate-50/50 space-y-1.5"
                    >
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <div className="flex items-center gap-2">
                          <span className="font-bold text-sm text-slate-900">{act.title}</span>
                          <Badge variant="outline" className="text-xs">
                            {act.category}
                          </Badge>
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
                  {t.noActivitiesRecorded}
                </div>
              )}
            </div>

            {/* 6. Deterministic Recommendations */}
            {snapshot.recommendations && snapshot.recommendations.length > 0 && (
              <div>
                <h3 className="text-base sm:text-lg font-bold text-slate-900 mb-4 flex items-center gap-2">
                  <FileCheck className="w-5 h-5 text-indigo-600 shrink-0" />
                  <span>{t.recommendations}</span>
                </h3>
                <div className="space-y-2">
                  {snapshot.recommendations.map((rec: any, idx: number) => (
                    <div
                      key={idx}
                      className="p-3.5 rounded-xl bg-blue-50/70 border border-blue-100 text-xs sm:text-sm text-blue-900"
                    >
                      <div className="font-bold text-blue-900 mb-0.5">
                        {rec.title}
                      </div>
                      <div className="text-xs text-blue-800 leading-relaxed">
                        {rec.description}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* 7. Methodology & Limitations Notice */}
            <div className="p-4 rounded-xl bg-slate-50 border border-slate-200 text-xs text-slate-500 space-y-1">
              <div className="font-bold text-slate-700">{t.limitationMethods}</div>
              <p className="leading-relaxed">
                {snapshot.limitation_notice ||
                  (isPersianReport
                    ? "محاسبه درصد آپتایم بر مبنای دوره‌های پایش فعال انجام شده و بازه‌های متوقف‌شده یا تست‌های دستی حذف شده‌اند. هارت‌بیت‌ها تنها اعلام وصول موفق هستند و به تنهایی بازگردانی داده‌های بک‌آپ را اثبات نمی‌کنند."
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

        {/* Secure Share Modal */}
        {shareModalOpen && (
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby="share-modal-title"
            className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm animate-in fade-in duration-150"
          >
            <div
              className="bg-white rounded-2xl shadow-xl w-full max-w-lg overflow-hidden border border-slate-200 animate-in zoom-in-95 duration-150"
              dir={isRTL ? "rtl" : "ltr"}
            >
              <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 bg-slate-50/70">
                <h3
                  id="share-modal-title"
                  className="text-base font-bold text-slate-900 flex items-center gap-2"
                >
                  <Share2 className="w-5 h-5 text-indigo-600 shrink-0" />
                  {t.shareReport}
                </h3>
                <button
                  onClick={() => {
                    setShareModalOpen(false);
                    setNewlyCreatedUrl(null);
                  }}
                  className="text-slate-400 hover:text-slate-600 rounded-lg p-1 transition-colors"
                  aria-label={t.close}
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              <div className="p-6 space-y-4 max-h-[75vh] overflow-y-auto">
                <p className="text-xs text-slate-600 leading-relaxed">
                  {t.singleUseTokenNotice}
                </p>

                {/* Expiry Input */}
                <div>
                  <label className="block text-xs font-semibold uppercase text-slate-700 mb-1">
                    {t.shareExpiryDays}
                  </label>
                  <Input
                    type="number"
                    min="1"
                    max="365"
                    value={expiresInDays}
                    onChange={(e) => setExpiresInDays(parseInt(e.target.value) || 0)}
                    placeholder="30"
                    className="text-sm"
                  />
                </div>

                <Button
                  onClick={handleCreateShare}
                  disabled={creatingShare}
                  className="w-full bg-indigo-600 hover:bg-indigo-700 text-white flex items-center justify-center gap-2 font-semibold text-xs sm:text-sm"
                >
                  <Lock className="w-4 h-4" />
                  {creatingShare ? t.loading : t.createShareLink}
                </Button>

                {/* Newly Created Link Box */}
                {newlyCreatedUrl && (
                  <div className="p-3.5 bg-emerald-50 border border-emerald-200 rounded-xl space-y-2">
                    <span className="text-xs font-bold text-emerald-900 block">
                      {isRTL ? "لینک جدید عمومی با موفقیت ایجاد شد:" : "New Public Share Link Created:"}
                    </span>
                    <div className="flex items-center gap-2">
                      <input
                        type="text"
                        readOnly
                        value={newlyCreatedUrl}
                        className="w-full px-2.5 py-1.5 text-xs font-mono bg-white border border-emerald-300 rounded-lg select-all"
                        dir="ltr"
                      />
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => copyToClipboard(newlyCreatedUrl)}
                        className="shrink-0 text-xs"
                      >
                        {copied ? <Check className="w-3.5 h-3.5 mr-1 text-emerald-600" /> : <Copy className="w-3.5 h-3.5 mr-1" />}
                        {copied ? t.copied : t.copyCode}
                      </Button>
                    </div>
                  </div>
                )}

                {/* Active Shares List */}
                <div className="pt-2 border-t border-slate-200">
                  <h4 className="text-xs font-bold uppercase tracking-wider text-slate-700 mb-2">
                    {t.activeShares}
                  </h4>
                  {sharesLoading ? (
                    <div className="text-xs text-slate-400 py-2">{t.loading}</div>
                  ) : shares.filter((s) => !s.revoked_at).length === 0 ? (
                    <div className="text-xs text-slate-400 py-2">
                      {isRTL ? "هیچ لینک فعالی وجود ندارد." : "No active share links."}
                    </div>
                  ) : (
                    <div className="space-y-2 max-h-48 overflow-y-auto">
                      {shares
                        .filter((s) => !s.revoked_at)
                        .map((s) => (
                          <div
                            key={s.id}
                            className="p-3 bg-slate-50 rounded-xl border border-slate-200 flex items-center justify-between text-xs"
                          >
                            <div className="space-y-0.5">
                              <div className="font-semibold text-slate-800">
                                {isRTL ? "انقضا:" : "Expires:"}{" "}
                                <span className="font-mono" dir="ltr">
                                  {s.expires_at ? formatDate(s.expires_at) : (isRTL ? "دائمی" : "Never")}
                                </span>
                              </div>
                              <div className="text-slate-400 text-[11px] font-mono" dir="ltr">
                                {formatDateTime(s.created_at)}
                              </div>
                            </div>
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => setRevokeTargetId(s.id)}
                              className="text-rose-600 hover:text-rose-700 border-rose-200 h-7 text-xs px-2.5"
                            >
                              <Trash2 className="w-3.5 h-3.5 mr-1" />
                              {t.revokeShareLink}
                            </Button>
                          </div>
                        ))}
                    </div>
                  )}
                </div>
              </div>

              <div className="px-6 py-3 bg-slate-50/70 border-t border-slate-200 flex justify-end">
                <Button
                  variant="outline"
                  onClick={() => {
                    setShareModalOpen(false);
                    setNewlyCreatedUrl(null);
                  }}
                >
                  {t.close}
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Revoke Confirmation Modal */}
        <ConfirmModal
          isOpen={!!revokeTargetId}
          onClose={() => setRevokeTargetId(null)}
          onConfirm={handleConfirmRevoke}
          title={t.revokeConfirmTitle}
          message={t.confirmRevokeShare}
          confirmLabel={t.revokeShareLink}
          variant="danger"
          loading={revokingShare}
        />
      </div>
    </AppLayout>
  );
}
