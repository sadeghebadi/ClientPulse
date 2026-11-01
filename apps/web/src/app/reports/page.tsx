"use client";

import React, { useState, useEffect, useCallback } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch, humanizeApiError } from "@/lib/api";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Badge } from "@/components/ui/Badge";
import {
  FileText,
  Plus,
  Download,
  Share2,
  Calendar,
  ExternalLink,
  ShieldCheck,
  AlertCircle,
  Eye,
  X,
  Clock,
  Sparkles,
  CheckCircle2,
} from "lucide-react";

interface ReportItem {
  id: string;
  organization_id: string;
  client_id: string;
  client_name?: string;
  report_type: string;
  period_start: string;
  period_end: string;
  organization_timezone: string;
  locale: string;
  version_number: number;
  template_version: string;
  title: string;
  pdf_storage_key?: string | null;
  generated_by: string;
  generated_by_name?: string;
  generated_at: string;
}

interface ClientOption {
  id: string;
  name: string;
}

export default function ReportsPage() {
  const router = useRouter();
  const { activeOrg, role } = useAuth();
  const { t, isRTL, locale } = useI18n();

  const [reports, setReports] = useState<ReportItem[]>([]);
  const [clients, setClients] = useState<ClientOption[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Filters
  const [selectedClient, setSelectedClient] = useState<string>("");
  const [searchQuery, setSearchQuery] = useState<string>("");

  // Generate Modal State
  const [modalOpen, setModalOpen] = useState(false);
  const [genClientId, setGenClientId] = useState("");
  const [periodPreset, setPeriodPreset] = useState<"prev_month" | "custom">("prev_month");
  const [customStart, setCustomStart] = useState("");
  const [customEnd, setCustomEnd] = useState("");
  const [reportLocale, setReportLocale] = useState<"en" | "fa">(locale === "fa" ? "fa" : "en");
  const [reportTitle, setReportTitle] = useState("");

  // Preview & Generating State
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewData, setPreviewData] = useState<any | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [generating, setGenerating] = useState(false);

  const canGenerate = role === "owner" || role === "admin" || role === "member";

  // Calculate default previous month date strings for form helpers
  const getPrevMonthBoundaries = () => {
    const now = new Date();
    const firstOfCurrent = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), 1));
    const endOfPrev = new Date(firstOfCurrent.getTime() - 1);
    const firstOfPrev = new Date(Date.UTC(endOfPrev.getUTCFullYear(), endOfPrev.getUTCMonth(), 1));
    return {
      start: firstOfPrev.toISOString().slice(0, 10),
      end: endOfPrev.toISOString().slice(0, 10),
    };
  };

  const fetchReports = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      let url = "/api/v1/reports?limit=100";
      if (selectedClient) {
        url += `&client_id=${selectedClient}`;
      }
      const data = await apiFetch<{ reports: ReportItem[]; total: number }>(url);
      setReports(data.reports || []);
    } catch (err: unknown) {
      setError(humanizeApiError(err, isRTL));
    } finally {
      setLoading(false);
    }
  }, [selectedClient, isRTL]);

  useEffect(() => {
    let isMounted = true;
    let url = "/api/v1/reports?limit=100";
    if (selectedClient) {
      url += `&client_id=${selectedClient}`;
    }
    apiFetch<{ reports: ReportItem[]; total: number }>(url)
      .then((data) => {
        if (isMounted) {
          setReports(data.reports || []);
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
  }, [selectedClient, isRTL]);

  useEffect(() => {
    let isMounted = true;
    apiFetch<{ clients: ClientOption[] }>("/api/v1/clients?limit=100")
      .then((data) => {
        if (isMounted) {
          const list = data.clients || [];
          setClients(list);
          if (list.length > 0) {
            setGenClientId((prev) => prev || list[0].id);
          }
        }
      })
      .catch(() => {
        // silent fallback
      });

    return () => {
      isMounted = false;
    };
  }, []);

  const handleOpenModal = () => {
    const { start, end } = getPrevMonthBoundaries();
    setCustomStart(start);
    setCustomEnd(end);
    setPeriodPreset("prev_month");
    setPreviewData(null);
    setPreviewError(null);
    setReportTitle("");
    setModalOpen(true);
  };

  const handlePreview = async () => {
    if (!genClientId) {
      setPreviewError(isRTL ? "لطفاً ابتدا مشتری را انتخاب نمایید" : "Please select a client first");
      return;
    }

    try {
      setPreviewLoading(true);
      setPreviewError(null);

      const payload: any = {
        client_id: genClientId,
        locale: reportLocale,
      };

      if (periodPreset === "custom") {
        if (!customStart || !customEnd) {
          setPreviewError(isRTL ? "لطفاً تاریخ شروع و پایان را وارد کنید" : "Start and end dates are required");
          setPreviewLoading(false);
          return;
        }
        payload.period_start = new Date(customStart + "T00:00:00Z").toISOString();
        payload.period_end = new Date(customEnd + "T23:59:59Z").toISOString();
      }

      const res = await apiFetch<{ preview: any }>("/api/v1/reports/preview", {
        method: "POST",
        body: JSON.stringify(payload),
      });

      setPreviewData(res.preview);
    } catch (err: unknown) {
      setPreviewError(humanizeApiError(err, isRTL));
    } finally {
      setPreviewLoading(false);
    }
  };

  const handleGenerate = async () => {
    if (!genClientId) return;

    try {
      setGenerating(true);
      setPreviewError(null);

      const payload: any = {
        client_id: genClientId,
        locale: reportLocale,
        title: reportTitle.trim() || undefined,
      };

      if (periodPreset === "custom") {
        payload.period_start = new Date(customStart + "T00:00:00Z").toISOString();
        payload.period_end = new Date(customEnd + "T23:59:59Z").toISOString();
      }

      const res = await apiFetch<{ report: ReportItem }>("/api/v1/reports", {
        method: "POST",
        body: JSON.stringify(payload),
      });

      setModalOpen(false);
      fetchReports();
      // Redirect to report details
      if (res?.report?.id) {
        router.push(`/reports/${res.report.id}`);
      }
    } catch (err: unknown) {
      setPreviewError(humanizeApiError(err, isRTL));
    } finally {
      setGenerating(false);
    }
  };

  const filteredReports = reports.filter((r) => {
    if (searchQuery.trim() === "") return true;
    const q = searchQuery.toLowerCase();
    return (
      r.title?.toLowerCase().includes(q) ||
      r.client_name?.toLowerCase().includes(q) ||
      `v${r.version_number}`.includes(q)
    );
  });

  const formatDate = (isoString: string) => {
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

  return (
    <AppLayout>
      <div className="space-y-6">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100 flex items-center gap-2">
              <FileText className="w-7 h-7 text-indigo-600 dark:text-indigo-400" />
              {t.reports}
            </h1>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {t.reportsDesc}
            </p>
          </div>

          {canGenerate && (
            <Button
              onClick={handleOpenModal}
              className="bg-indigo-600 hover:bg-indigo-700 text-white flex items-center gap-2"
            >
              <Plus className="w-4 h-4" />
              {t.generateReport}
            </Button>
          )}
        </div>

        {/* Filters */}
        <div className="bg-white dark:bg-gray-800 p-4 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 flex flex-col sm:flex-row items-center gap-4">
          <div className="w-full sm:w-64">
            <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
              {t.filterByClient}
            </label>
            <select
              value={selectedClient}
              onChange={(e) => setSelectedClient(e.target.value)}
              className="w-full px-3 py-2 text-sm bg-gray-50 dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500"
            >
              <option value="">{t.allClients}</option>
              {clients.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>

          <div className="w-full sm:flex-1">
            <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
              {t.search}
            </label>
            <Input
              type="text"
              placeholder={isRTL ? "جستجو در گزارش‌ها..." : "Search reports by title or client..."}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full text-sm"
            />
          </div>
        </div>

        {/* Content Table / Empty States */}
        {loading ? (
          <div className="bg-white dark:bg-gray-800 p-12 rounded-xl text-center border border-gray-200 dark:border-gray-700">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600 mx-auto" />
            <p className="mt-3 text-sm text-gray-500">{t.loading}</p>
          </div>
        ) : error ? (
          <div className="bg-red-50 dark:bg-red-900/20 p-6 rounded-xl border border-red-200 dark:border-red-800 text-red-700 dark:text-red-400 flex items-center gap-3">
            <AlertCircle className="w-6 h-6 flex-shrink-0" />
            <span>{error}</span>
          </div>
        ) : filteredReports.length === 0 ? (
          <div className="bg-white dark:bg-gray-800 p-12 rounded-xl text-center border border-gray-200 dark:border-gray-700">
            <FileText className="w-12 h-12 text-gray-400 mx-auto mb-3" />
            <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
              {t.noReportsFound}
            </h3>
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1 max-w-md mx-auto">
              {isRTL
                ? "گزارش‌های دوره‌ای ماهانه وب‌سایت شامل بررسی آپتایم، فعالیت‌های نگهداری و وضعیت امنیتی را تولید و با مشتریان خود به اشتراک بگذارید."
                : "Generate monthly care reports with uptime analytics, maintenance activity logs, and deterministic recommendations."}
            </p>
            {canGenerate && (
              <Button onClick={handleOpenModal} className="mt-4 bg-indigo-600 hover:bg-indigo-700 text-white">
                <Plus className="w-4 h-4 mr-2" />
                {t.generateReport}
              </Button>
            )}
          </div>
        ) : (
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm text-left rtl:text-right">
                <thead className="bg-gray-50 dark:bg-gray-700/50 text-gray-700 dark:text-gray-300 border-b border-gray-200 dark:border-gray-700">
                  <tr>
                    <th className="px-6 py-3 font-semibold">{t.clientName}</th>
                    <th className="px-6 py-3 font-semibold">{isRTL ? "عنوان گزارش" : "Report Title"}</th>
                    <th className="px-6 py-3 font-semibold">{t.reportPeriod}</th>
                    <th className="px-6 py-3 font-semibold">{t.version}</th>
                    <th className="px-6 py-3 font-semibold">{isRTL ? "تاریخ صدور" : "Generated At"}</th>
                    <th className="px-6 py-3 font-semibold text-center">{isRTL ? "عملیات" : "Actions"}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                  {filteredReports.map((r) => (
                    <tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors">
                      <td className="px-6 py-4 font-medium text-gray-900 dark:text-gray-100">
                        {r.client_name || r.client_id}
                      </td>
                      <td className="px-6 py-4">
                        <Link
                          href={`/reports/${r.id}`}
                          className="text-indigo-600 dark:text-indigo-400 font-medium hover:underline flex items-center gap-1.5"
                        >
                          {r.title}
                        </Link>
                      </td>
                      <td className="px-6 py-4 text-gray-500 dark:text-gray-400">
                        <div className="flex items-center gap-1.5 text-xs">
                          <Calendar className="w-3.5 h-3.5 text-gray-400" />
                          <span>
                            {formatDate(r.period_start)} - {formatDate(r.period_end)}
                          </span>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <Badge variant="outline" className="font-mono text-xs">
                          v{r.version_number}
                        </Badge>
                      </td>
                      <td className="px-6 py-4 text-xs text-gray-500 dark:text-gray-400">
                        {formatDate(r.generated_at)}
                      </td>
                      <td className="px-6 py-4 text-center">
                        <div className="flex items-center justify-center gap-2">
                          <Link href={`/reports/${r.id}`}>
                            <Button variant="outline" size="sm" className="h-8 px-2.5">
                              <Eye className="w-3.5 h-3.5 mr-1" />
                              {t.reportDetails}
                            </Button>
                          </Link>
                          <a href={`/api/v1/reports/${r.id}/pdf`} download>
                            <Button variant="outline" size="sm" className="h-8 px-2 text-indigo-600 hover:text-indigo-700">
                              <Download className="w-3.5 h-3.5" />
                            </Button>
                          </a>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Generate Report Modal */}
        {modalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm animate-fade-in">
            <div className="bg-white dark:bg-gray-800 rounded-2xl shadow-xl w-full max-w-2xl overflow-hidden border border-gray-200 dark:border-gray-700">
              <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700 bg-gray-50/50 dark:bg-gray-800/50">
                <h3 className="text-lg font-bold text-gray-900 dark:text-gray-100 flex items-center gap-2">
                  <Sparkles className="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
                  {t.generateReport}
                </h3>
                <button
                  onClick={() => setModalOpen(false)}
                  className="text-gray-400 hover:text-gray-500 rounded-lg p-1"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              <div className="p-6 space-y-4 max-h-[75vh] overflow-y-auto">
                {/* Client Select */}
                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-gray-700 dark:text-gray-300 mb-1">
                    {t.clientName} *
                  </label>
                  <select
                    value={genClientId}
                    onChange={(e) => {
                      setGenClientId(e.target.value);
                      setPreviewData(null);
                    }}
                    className="w-full px-3 py-2 text-sm bg-gray-50 dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500"
                  >
                    {clients.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name}
                      </option>
                    ))}
                  </select>
                </div>

                {/* Period Select */}
                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-gray-700 dark:text-gray-300 mb-1">
                    {t.reportPeriod}
                  </label>
                  <div className="grid grid-cols-2 gap-3 mb-3">
                    <button
                      type="button"
                      onClick={() => {
                        setPeriodPreset("prev_month");
                        setPreviewData(null);
                      }}
                      className={`px-3 py-2 text-sm font-medium rounded-lg border transition-all text-center ${
                        periodPreset === "prev_month"
                          ? "bg-indigo-50 dark:bg-indigo-950/40 border-indigo-500 text-indigo-700 dark:text-indigo-300"
                          : "border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700"
                      }`}
                    >
                      {t.defaultPreviousMonth}
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        setPeriodPreset("custom");
                        setPreviewData(null);
                      }}
                      className={`px-3 py-2 text-sm font-medium rounded-lg border transition-all text-center ${
                        periodPreset === "custom"
                          ? "bg-indigo-50 dark:bg-indigo-950/40 border-indigo-500 text-indigo-700 dark:text-indigo-300"
                          : "border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700"
                      }`}
                    >
                      {isRTL ? "بازه زمانی سفارشی" : "Custom Date Range"}
                    </button>
                  </div>

                  {periodPreset === "custom" && (
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 p-3 bg-gray-50 dark:bg-gray-900/50 rounded-xl border border-gray-200 dark:border-gray-700">
                      <div>
                        <label className="block text-xs text-gray-500 dark:text-gray-400 mb-1">
                          {t.periodStart}
                        </label>
                        <Input
                          type="date"
                          value={customStart}
                          onChange={(e) => {
                            setCustomStart(e.target.value);
                            setPreviewData(null);
                          }}
                        />
                      </div>
                      <div>
                        <label className="block text-xs text-gray-500 dark:text-gray-400 mb-1">
                          {t.periodEnd}
                        </label>
                        <Input
                          type="date"
                          value={customEnd}
                          onChange={(e) => {
                            setCustomEnd(e.target.value);
                            setPreviewData(null);
                          }}
                        />
                      </div>
                    </div>
                  )}
                </div>

                {/* Locale & Title */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-gray-700 dark:text-gray-300 mb-1">
                      {isRTL ? "زبان گزارش" : "Report Language"}
                    </label>
                    <select
                      value={reportLocale}
                      onChange={(e) => {
                        setReportLocale(e.target.value as "en" | "fa");
                        setPreviewData(null);
                      }}
                      className="w-full px-3 py-2 text-sm bg-gray-50 dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-indigo-500"
                    >
                      <option value="en">English (LTR)</option>
                      <option value="fa">فارسی (Persian - RTL)</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-gray-700 dark:text-gray-300 mb-1">
                      {isRTL ? "عنوان اختصاصی (اختیاری)" : "Custom Title (Optional)"}
                    </label>
                    <Input
                      type="text"
                      placeholder={isRTL ? "گزارش ماهانه نگهداری وب‌سایت" : "Monthly Website Care Report"}
                      value={reportTitle}
                      onChange={(e) => setReportTitle(e.target.value)}
                    />
                  </div>
                </div>

                {/* Preview Button */}
                <div className="pt-2">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handlePreview}
                    disabled={previewLoading || !genClientId}
                    className="w-full flex items-center justify-center gap-2"
                  >
                    <Eye className="w-4 h-4 text-indigo-600" />
                    {previewLoading ? t.loading : t.reportPreview}
                  </Button>
                </div>

                {/* Preview Error */}
                {previewError && (
                  <div className="p-3 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400 text-xs rounded-lg border border-red-200 dark:border-red-800">
                    {previewError}
                  </div>
                )}

                {/* Preview Card */}
                {previewData && (
                  <div className="p-4 bg-indigo-50/50 dark:bg-indigo-950/20 rounded-xl border border-indigo-100 dark:border-indigo-900/40 space-y-3">
                    <div className="flex items-center justify-between border-b border-indigo-100 dark:border-indigo-900/30 pb-2">
                      <span className="font-semibold text-sm text-indigo-900 dark:text-indigo-200">
                        {previewData.title}
                      </span>
                      <Badge variant="outline" className="text-xs bg-white dark:bg-gray-800">
                        {previewData.locale === "fa" ? "فارسی" : "English"}
                      </Badge>
                    </div>

                    <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-center">
                      <div className="p-2 bg-white dark:bg-gray-800 rounded-lg shadow-xs">
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          {isRTL ? "آپتایم میانگین" : "Avg Uptime"}
                        </div>
                        <div className="text-base font-bold text-gray-900 dark:text-gray-100">
                          {previewData.summary?.average_uptime != null
                            ? `${previewData.summary.average_uptime.toFixed(2)}%`
                            : t.insufficientData}
                        </div>
                      </div>

                      <div className="p-2 bg-white dark:bg-gray-800 rounded-lg shadow-xs">
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          {t.coveragePercentage}
                        </div>
                        <div className="text-base font-bold text-gray-900 dark:text-gray-100">
                          {previewData.monitors && previewData.monitors.length > 0 && previewData.monitors[0].coverage_percentage != null
                            ? `${previewData.monitors[0].coverage_percentage.toFixed(1)}%`
                            : "100.0%"}
                        </div>
                      </div>

                      <div className="p-2 bg-white dark:bg-gray-800 rounded-lg shadow-xs">
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          {isRTL ? "رخدادهای قطعی" : "Incidents"}
                        </div>
                        <div className="text-base font-bold text-gray-900 dark:text-gray-100">
                          {previewData.summary?.total_incidents ?? 0}
                        </div>
                      </div>

                      <div className="p-2 bg-white dark:bg-gray-800 rounded-lg shadow-xs">
                        <div className="text-xs text-gray-500 dark:text-gray-400">
                          {t.maintenance}
                        </div>
                        <div className="text-base font-bold text-gray-900 dark:text-gray-100">
                          {previewData.summary?.total_activities ?? (previewData.maintenance ? previewData.maintenance.length : 0)}
                        </div>
                      </div>
                    </div>

                    <p className="text-xs text-gray-500 dark:text-gray-400 text-center">
                      {isRTL
                        ? "این پیش‌نمایش به صورت قطعی محاسبه شده است. تأیید شما یک نسخه دائمی و غیرقابل تغییر (Immutable Snapshot) ایجاد خواهد کرد."
                        : "Deterministic preview calculated. Generating will seal an immutable report version."}
                    </p>
                  </div>
                )}
              </div>

              <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50/50 dark:bg-gray-800/50">
                <Button variant="outline" onClick={() => setModalOpen(false)}>
                  {t.close}
                </Button>
                <Button
                  onClick={handleGenerate}
                  disabled={generating || !genClientId}
                  className="bg-indigo-600 hover:bg-indigo-700 text-white flex items-center gap-2"
                >
                  <Sparkles className="w-4 h-4" />
                  {generating ? (isRTL ? "در حال صدور..." : "Generating...") : t.generateReport}
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </AppLayout>
  );
}
