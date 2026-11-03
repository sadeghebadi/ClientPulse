"use client";

import React, { useState, useEffect, useCallback, useRef } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch, humanizeApiError } from "@/lib/api";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Badge } from "@/components/ui/Badge";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import {
  FileText,
  Plus,
  Download,
  Calendar,
  Eye,
  X,
  Sparkles,
  Search,
  Filter,
  CheckCircle2,
  FileCheck,
  Building2,
  ShieldCheck,
  ChevronRight,
  ChevronLeft,
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

  const modalRef = useRef<HTMLDivElement>(null);

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
    fetchReports();
  }, [fetchReports]);

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

  // Modal accessibility (Escape key handler)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && modalOpen && !generating) {
        setModalOpen(false);
      }
    };
    if (modalOpen) {
      window.addEventListener("keydown", handleKeyDown);
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "unset";
    }
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      document.body.style.overflow = "unset";
    };
  }, [modalOpen, generating]);

  const handleOpenModal = () => {
    const { start, end } = getPrevMonthBoundaries();
    setCustomStart(start);
    setCustomEnd(end);
    setPeriodPreset("prev_month");
    setPreviewData(null);
    setPreviewError(null);
    setReportTitle("");
    setReportLocale(locale === "fa" ? "fa" : "en");
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
      <div className="space-y-6 max-w-7xl mx-auto">
        {/* Page Header */}
        <PageHeader
          title={t.reports}
          description={t.reportsDesc}
          icon={<FileText className="w-6 h-6 text-indigo-600" />}
          badge={
            <Badge variant="outline" className="text-xs font-mono">
              {reports.length} {t.reports}
            </Badge>
          }
          actions={
            canGenerate ? (
              <Button
                variant="primary"
                onClick={handleOpenModal}
                icon={<Plus className="w-4 h-4" />}
                className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm"
              >
                {t.generateReport}
              </Button>
            ) : undefined
          }
        />

        {/* Filter & Search Bar */}
        <div className="bg-white p-4 rounded-2xl border border-slate-200 shadow-xs flex flex-col sm:flex-row items-stretch sm:items-center gap-3">
          <div className="w-full sm:w-64">
            <label className="block text-xs font-semibold text-slate-600 mb-1">
              {t.filterByClient}
            </label>
            <select
              value={selectedClient}
              onChange={(e) => setSelectedClient(e.target.value)}
              className="w-full px-3 py-2 text-xs sm:text-sm bg-slate-50 border border-slate-300 rounded-xl focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none transition-all"
            >
              <option value="">{t.allClients}</option>
              {clients.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>

          <div className="flex-1">
            <label className="block text-xs font-semibold text-slate-600 mb-1">
              {t.search}
            </label>
            <div className="relative">
              <Input
                type="text"
                placeholder={isRTL ? "جستجو بر اساس عنوان یا نام مشتری..." : "Search by title or client name..."}
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full text-xs sm:text-sm pe-8"
              />
              {searchQuery && (
                <button
                  type="button"
                  onClick={() => setSearchQuery("")}
                  className="absolute end-2.5 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 text-xs"
                  aria-label={t.clearFilters}
                >
                  <X className="w-4 h-4" />
                </button>
              )}
            </div>
          </div>

          {(selectedClient || searchQuery) && (
            <div className="self-end sm:self-auto sm:mt-5">
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  setSelectedClient("");
                  setSearchQuery("");
                }}
                className="text-xs text-slate-600"
              >
                {t.clearFilters}
              </Button>
            </div>
          )}
        </div>

        {/* Content States */}
        {loading ? (
          <TableSkeleton rows={5} cols={5} />
        ) : error ? (
          <ErrorState error={error} onRetry={fetchReports} />
        ) : reports.length === 0 ? (
          <EmptyState
            icon={<FileText className="w-8 h-8 text-indigo-500" />}
            title={t.noReportsFound}
            description={
              isRTL
                ? "گزارش‌های دوره‌ای ماهانه شامل وضعیت قطعی، هارت‌بیت‌ها، پایش آپتایم و کارهای نگهداری را با برند وایت‌لیبل برای مشتریان خود صادر نمایید."
                : "Generate monthly care reports with verifiable SLA uptime proof, heartbeat health, and deterministic maintenance logs for your retainer clients."
            }
            actionLabel={canGenerate ? t.generateReport : undefined}
            onAction={canGenerate ? handleOpenModal : undefined}
          />
        ) : filteredReports.length === 0 ? (
          <EmptyState
            icon={<Search className="w-8 h-8 text-slate-400" />}
            title={t.noReportsFiltered}
            description={
              isRTL
                ? "هیچ گزارشی با فیلترهای انتخابی تطابق ندارد. لطفاً عبارت جستجو یا مشتری انتخابی را تغییر دهید."
                : "No care reports match your active search and filter criteria."
            }
            actionLabel={t.clearFilters}
            onAction={() => {
              setSelectedClient("");
              setSearchQuery("");
            }}
          />
        ) : (
          <div className="space-y-4">
            {/* Desktop Table View */}
            <div className="hidden md:block bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
              <table className="w-full text-sm text-left rtl:text-right">
                <thead className="bg-slate-50 text-slate-700 border-b border-slate-200 text-xs uppercase tracking-wider font-semibold">
                  <tr>
                    <th className="px-6 py-3.5">{t.clientName}</th>
                    <th className="px-6 py-3.5">{t.reportTitle}</th>
                    <th className="px-6 py-3.5">{t.reportPeriod}</th>
                    <th className="px-6 py-3.5">{t.version}</th>
                    <th className="px-6 py-3.5">{t.generatedAt}</th>
                    <th className="px-6 py-3.5 text-center">{isRTL ? "عملیات" : "Actions"}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {filteredReports.map((r) => (
                    <tr
                      key={r.id}
                      className="hover:bg-slate-50/60 transition-colors"
                    >
                      <td className="px-6 py-4 font-semibold text-slate-900">
                        <div className="flex items-center gap-2">
                          <Building2 className="w-4 h-4 text-slate-400 shrink-0" />
                          <span className="truncate max-w-[180px]">
                            {r.client_name || r.client_id}
                          </span>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <Link
                          href={`/reports/${r.id}`}
                          className="text-indigo-600 hover:text-indigo-700 font-medium hover:underline flex items-center gap-1.5"
                        >
                          <FileText className="w-4 h-4 shrink-0 text-indigo-500" />
                          <span className="truncate max-w-[240px]">{r.title}</span>
                        </Link>
                      </td>
                      <td className="px-6 py-4 text-slate-600">
                        <div className="flex items-center gap-1.5 text-xs text-slate-500 font-mono" dir="ltr">
                          <Calendar className="w-3.5 h-3.5 text-slate-400 shrink-0" />
                          <span>
                            {formatDate(r.period_start)} – {formatDate(r.period_end)}
                          </span>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex items-center gap-1.5">
                          <Badge variant="outline" className="font-mono text-xs">
                            v{r.version_number}
                          </Badge>
                          <Badge
                            variant={r.pdf_storage_key ? "success" : "secondary"}
                            className="text-[10px]"
                          >
                            {r.pdf_storage_key ? t.statusPdfReady : t.statusSnapshotReady}
                          </Badge>
                        </div>
                      </td>
                      <td className="px-6 py-4 text-xs text-slate-500 font-mono" dir="ltr">
                        {formatDate(r.generated_at)}
                      </td>
                      <td className="px-6 py-4 text-center">
                        <div className="flex items-center justify-center gap-2">
                          <Link href={`/reports/${r.id}`}>
                            <Button variant="outline" size="sm" className="h-8 px-2.5 text-xs">
                              <Eye className="w-3.5 h-3.5 mr-1" />
                              {t.reportDetails}
                            </Button>
                          </Link>
                          <a
                            href={`/api/v1/reports/${r.id}/pdf`}
                            download
                            className="inline-flex items-center justify-center h-8 px-2.5 rounded-lg border border-slate-200 text-slate-700 hover:bg-slate-100 hover:text-indigo-600 transition-colors"
                            title={t.downloadPDF}
                          >
                            <Download className="w-3.5 h-3.5" />
                          </a>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Mobile Card View (< 768px) */}
            <div className="md:hidden space-y-3">
              {filteredReports.map((r) => (
                <div
                  key={r.id}
                  className="bg-white p-4 rounded-2xl border border-slate-200 shadow-xs space-y-3 hover:border-slate-300 transition-colors"
                >
                  <div className="flex items-center justify-between gap-2">
                    <div className="flex items-center gap-1.5 min-w-0">
                      <Building2 className="w-4 h-4 text-slate-400 shrink-0" />
                      <span className="text-xs font-bold text-slate-700 truncate">
                        {r.client_name || r.client_id}
                      </span>
                    </div>
                    <div className="flex items-center gap-1 shrink-0">
                      <Badge variant="outline" className="font-mono text-xs">
                        v{r.version_number}
                      </Badge>
                      <Badge
                        variant={r.pdf_storage_key ? "success" : "secondary"}
                        className="text-[10px]"
                      >
                        {r.pdf_storage_key ? t.statusPdfReady : t.statusSnapshotReady}
                      </Badge>
                    </div>
                  </div>

                  <div>
                    <Link
                      href={`/reports/${r.id}`}
                      className="text-sm font-bold text-slate-900 hover:text-indigo-600 transition-colors block line-clamp-2"
                    >
                      {r.title}
                    </Link>
                  </div>

                  <div className="pt-2 border-t border-slate-100 flex flex-col gap-1 text-xs text-slate-500">
                    <div className="flex items-center gap-1.5 font-mono" dir="ltr">
                      <Calendar className="w-3.5 h-3.5 text-slate-400 shrink-0" />
                      <span>
                        {formatDate(r.period_start)} – {formatDate(r.period_end)}
                      </span>
                    </div>
                    <div className="text-[11px] text-slate-400">
                      {t.generatedAt}: <span dir="ltr">{formatDate(r.generated_at)}</span>
                    </div>
                  </div>

                  <div className="pt-2 border-t border-slate-100 flex items-center justify-end gap-2">
                    <a
                      href={`/api/v1/reports/${r.id}/pdf`}
                      download
                      className="px-3 py-1.5 text-xs font-semibold rounded-lg border border-slate-200 text-slate-700 hover:bg-slate-100 hover:text-indigo-600 flex items-center gap-1 transition-colors"
                    >
                      <Download className="w-3.5 h-3.5" />
                      <span>PDF</span>
                    </a>
                    <Link href={`/reports/${r.id}`}>
                      <Button variant="primary" size="sm" className="h-8 px-3 text-xs bg-indigo-600 text-white">
                        <Eye className="w-3.5 h-3.5 mr-1" />
                        {t.reportDetails}
                      </Button>
                    </Link>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Generate Report Modal */}
        {modalOpen && (
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby="generate-modal-title"
            className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm animate-in fade-in duration-150"
          >
            <div
              ref={modalRef}
              className="bg-white rounded-2xl shadow-xl w-full max-w-2xl overflow-hidden border border-slate-200 animate-in zoom-in-95 duration-150"
              dir={isRTL ? "rtl" : "ltr"}
            >
              {/* Modal Header */}
              <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 bg-slate-50/70">
                <h3
                  id="generate-modal-title"
                  className="text-base font-bold text-slate-900 flex items-center gap-2"
                >
                  <Sparkles className="w-5 h-5 text-indigo-600 shrink-0" />
                  {t.generateReport}
                </h3>
                <button
                  onClick={() => !generating && setModalOpen(false)}
                  disabled={generating}
                  className="text-slate-400 hover:text-slate-600 rounded-lg p-1 transition-colors"
                  aria-label={t.close}
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              {/* Modal Body */}
              <div className="p-6 space-y-5 max-h-[75vh] overflow-y-auto">
                {/* Client Select */}
                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 mb-1.5">
                    {t.clientName} *
                  </label>
                  <select
                    value={genClientId}
                    onChange={(e) => {
                      setGenClientId(e.target.value);
                      setPreviewData(null);
                    }}
                    className="w-full px-3 py-2 text-sm bg-slate-50 border border-slate-300 rounded-xl focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none transition-all"
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
                  <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 mb-1.5">
                    {t.reportPeriod}
                  </label>
                  <div className="grid grid-cols-2 gap-3 mb-3">
                    <button
                      type="button"
                      onClick={() => {
                        setPeriodPreset("prev_month");
                        setPreviewData(null);
                      }}
                      className={`px-3 py-2 text-xs sm:text-sm font-medium rounded-xl border transition-all text-center ${
                        periodPreset === "prev_month"
                          ? "bg-indigo-50 border-indigo-500 text-indigo-700 font-bold"
                          : "border-slate-300 text-slate-700 hover:bg-slate-50"
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
                      className={`px-3 py-2 text-xs sm:text-sm font-medium rounded-xl border transition-all text-center ${
                        periodPreset === "custom"
                          ? "bg-indigo-50 border-indigo-500 text-indigo-700 font-bold"
                          : "border-slate-300 text-slate-700 hover:bg-slate-50"
                      }`}
                    >
                      {t.customDateRange}
                    </button>
                  </div>

                  {periodPreset === "custom" && (
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 p-3.5 bg-slate-50 rounded-xl border border-slate-200">
                      <div>
                        <label className="block text-xs text-slate-500 mb-1">
                          {t.periodStart}
                        </label>
                        <Input
                          type="date"
                          value={customStart}
                          onChange={(e) => {
                            setCustomStart(e.target.value);
                            setPreviewData(null);
                          }}
                          className="text-xs"
                        />
                      </div>
                      <div>
                        <label className="block text-xs text-slate-500 mb-1">
                          {t.periodEnd}
                        </label>
                        <Input
                          type="date"
                          value={customEnd}
                          onChange={(e) => {
                            setCustomEnd(e.target.value);
                            setPreviewData(null);
                          }}
                          className="text-xs"
                        />
                      </div>
                    </div>
                  )}
                </div>

                {/* Locale & Title */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 mb-1.5">
                      {t.reportLanguage}
                    </label>
                    <select
                      value={reportLocale}
                      onChange={(e) => {
                        setReportLocale(e.target.value as "en" | "fa");
                        setPreviewData(null);
                      }}
                      className="w-full px-3 py-2 text-sm bg-slate-50 border border-slate-300 rounded-xl focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none transition-all"
                    >
                      <option value="en">{t.englishLtr}</option>
                      <option value="fa">{t.persianRtl}</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-slate-700 mb-1.5">
                      {t.customTitleOptional}
                    </label>
                    <Input
                      type="text"
                      placeholder={isRTL ? "گزارش ماهانه نگهداری وب‌سایت" : "Monthly Website Care Report"}
                      value={reportTitle}
                      onChange={(e) => setReportTitle(e.target.value)}
                      className="text-sm"
                    />
                  </div>
                </div>

                {/* Preview Trigger Button */}
                <div className="pt-1">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handlePreview}
                    disabled={previewLoading || !genClientId}
                    className="w-full flex items-center justify-center gap-2 text-xs sm:text-sm font-semibold"
                  >
                    <Eye className="w-4 h-4 text-indigo-600" />
                    {previewLoading ? t.loading : t.reportPreview}
                  </Button>
                </div>

                {/* Preview Error */}
                {previewError && (
                  <div className="p-3 bg-rose-50 text-rose-700 text-xs rounded-xl border border-rose-200">
                    {previewError}
                  </div>
                )}

                {/* Preview Calculated Card */}
                {previewData && (
                  <div className="p-4 bg-indigo-50/50 rounded-2xl border border-indigo-100 space-y-3">
                    <div className="flex items-center justify-between border-b border-indigo-100 pb-2">
                      <span className="font-bold text-sm text-indigo-950">
                        {previewData.title}
                      </span>
                      <Badge variant="outline" className="text-xs bg-white">
                        {previewData.locale === "fa" ? "فارسی" : "English"}
                      </Badge>
                    </div>

                    <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5 text-center">
                      <div className="p-2.5 bg-white rounded-xl shadow-2xs border border-indigo-50">
                        <div className="text-[11px] text-slate-500">
                          {t.previewAvgUptime}
                        </div>
                        <div className="text-base font-extrabold text-slate-900 mt-0.5">
                          {previewData.summary?.average_uptime != null
                            ? `${previewData.summary.average_uptime.toFixed(2)}%`
                            : t.insufficientData}
                        </div>
                      </div>

                      <div className="p-2.5 bg-white rounded-xl shadow-2xs border border-indigo-50">
                        <div className="text-[11px] text-slate-500">
                          {t.coveragePercentage}
                        </div>
                        <div className="text-base font-extrabold text-indigo-600 mt-0.5">
                          {previewData.monitors && previewData.monitors.length > 0 && previewData.monitors[0].coverage_percentage != null
                            ? `${previewData.monitors[0].coverage_percentage.toFixed(1)}%`
                            : "100.0%"}
                        </div>
                      </div>

                      <div className="p-2.5 bg-white rounded-xl shadow-2xs border border-indigo-50">
                        <div className="text-[11px] text-slate-500">
                          {t.previewIncidents}
                        </div>
                        <div className="text-base font-extrabold text-slate-900 mt-0.5">
                          {previewData.summary?.total_incidents ?? 0}
                        </div>
                      </div>

                      <div className="p-2.5 bg-white rounded-xl shadow-2xs border border-indigo-50">
                        <div className="text-[11px] text-slate-500">
                          {t.previewMaintenance}
                        </div>
                        <div className="text-base font-extrabold text-slate-900 mt-0.5">
                          {previewData.summary?.total_activities ?? (previewData.maintenance ? previewData.maintenance.length : 0)}
                        </div>
                      </div>
                    </div>

                    <p className="text-[11px] text-slate-500 text-center leading-relaxed">
                      {t.previewDisclaimer}
                    </p>
                  </div>
                )}
              </div>

              {/* Modal Footer */}
              <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-slate-200 bg-slate-50/70">
                <Button
                  variant="outline"
                  onClick={() => setModalOpen(false)}
                  disabled={generating}
                >
                  {t.close}
                </Button>
                <Button
                  onClick={handleGenerate}
                  disabled={generating || !genClientId}
                  className="bg-indigo-600 hover:bg-indigo-700 text-white flex items-center gap-2 shadow-sm font-semibold"
                >
                  <Sparkles className="w-4 h-4" />
                  {generating ? t.generatingAction : t.generateReport}
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </AppLayout>
  );
}
