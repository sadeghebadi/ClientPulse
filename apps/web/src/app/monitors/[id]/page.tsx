"use client";

import React, { useEffect, useState, use, useCallback } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import {
  Activity,
  Play,
  Pause,
  ExternalLink,
  Edit,
  ShieldCheck,
  ShieldAlert,
  Zap,
  RotateCcw,
  Globe,
  Building,
  Info,
  Clock,
  Trash2,
} from "lucide-react";
import { PageHeader } from "@/components/ui/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { ErrorState } from "@/components/ui/ErrorState";
import { ConfirmModal } from "@/components/ui/ConfirmModal";
import { useToast } from "@/components/ui/Toast";

interface MonitorDetail {
  id: string;
  asset_name: string;
  asset_url: string;
  client_name: string;
  client_id: string;
  effective_url: string;
  method: string;
  interval_seconds: number;
  timeout_seconds: number;
  expected_status_min: number;
  expected_status_max: number;
  follow_redirects: boolean;
  max_redirects: number;
  expected_text?: string;
  json_pointer?: string;
  json_expected_value?: string;
  status: "up" | "down" | "suspected_down" | "pending" | "paused" | "unknown";
  is_active: boolean;
  next_check_at: string;
  last_checked_at?: string;
}

interface CheckResultItem {
  id: string;
  scheduled_at: string;
  completed_at: string;
  latency_ms: number;
  result: "success" | "failure" | "timeout" | "blocked" | "error";
  http_status_code?: number;
  error_category?: string;
  error_message?: string;
  tls_version?: string;
  tls_cipher?: string;
  tls_cert_issuer?: string;
  tls_cert_subject?: string;
  tls_days_remaining?: number;
  tls_is_valid?: boolean;
  execution_type: "scheduled" | "manual";
}

export default function MonitorDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = use(params);
  const router = useRouter();
  const { user, activeOrg, role } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const [monitor, setMonitor] = useState<MonitorDetail | null>(null);
  const [history, setHistory] = useState<CheckResultItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<any>(null);

  const [testing, setTesting] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);

  const canManage = role === "owner" || role === "admin" || role === "member";
  const canDelete = role === "owner" || role === "admin";

  const reloadMonitorData = useCallback(() => {
    if (!user || !activeOrg) return;
    setLoading(true);
    setError(null);

    Promise.all([
      apiFetch<{ monitor: MonitorDetail }>(`/api/v1/monitors/${resolvedParams.id}`),
      apiFetch<{ results: CheckResultItem[] }>(
        `/api/v1/monitors/${resolvedParams.id}/history?limit=25`
      ),
    ])
      .then(([monRes, histRes]) => {
        setMonitor(monRes.monitor);
        setHistory(histRes.results || []);
      })
      .catch((err) => {
        setError(err);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [user, activeOrg, resolvedParams.id]);

  useEffect(() => {
    if (!user || !activeOrg) return;
    let isMounted = true;

    Promise.all([
      apiFetch<{ monitor: MonitorDetail }>(`/api/v1/monitors/${resolvedParams.id}`),
      apiFetch<{ results: CheckResultItem[] }>(
        `/api/v1/monitors/${resolvedParams.id}/history?limit=25`
      ),
    ])
      .then(([monRes, histRes]) => {
        if (!isMounted) return;
        setMonitor(monRes.monitor);
        setHistory(histRes.results || []);
        setLoading(false);
      })
      .catch((err) => {
        if (isMounted) {
          setError(err);
          setLoading(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user, activeOrg, resolvedParams.id]);

  const handleTestNow = async () => {
    try {
      setTesting(true);
      const res = await apiFetch<{ result: CheckResultItem }>(
        `/api/v1/monitors/${resolvedParams.id}/test`,
        { method: "POST" }
      );

      if (res.result.result === "success") {
        toast.success(
          `${t.testSuccess} (${res.result.http_status_code || 200} · ${res.result.latency_ms}ms)`,
          t.testSuccess
        );
      } else {
        toast.error(
          res.result.error_message || `${t.testFailed}: ${res.result.result}`,
          t.testFailed
        );
      }

      reloadMonitorData();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Test check execution failed");
    } finally {
      setTesting(false);
    }
  };

  const handleTogglePause = async () => {
    if (!monitor) return;
    try {
      setActionLoading(true);
      const endpoint =
        monitor.status === "paused"
          ? `/api/v1/monitors/${monitor.id}/resume`
          : `/api/v1/monitors/${monitor.id}/pause`;

      await apiFetch(endpoint, { method: "POST" });
      toast.success(
        monitor.status === "paused"
          ? t.heartbeatResumed
          : t.heartbeatPaused
      );
      reloadMonitorData();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to toggle monitor state");
    } finally {
      setActionLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!monitor) return;
    try {
      setActionLoading(true);
      await apiFetch(`/api/v1/monitors/${monitor.id}`, { method: "DELETE" });
      toast.success(isRTL ? "مانیتور با موفقیت حذف شد." : "Monitor deleted successfully");
      router.push("/monitors");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to delete monitor");
      setActionLoading(false);
    }
  };

  const latestSSL = history.find((h) => h.tls_cert_issuer || h.tls_cert_subject);

  return (
    <AppLayout>
      <PageHeader
        title={monitor?.asset_name || t.monitorDetails}
        description={isRTL ? "پارامترهای سلامت، گواهینامه امنیتی SSL/TLS و سابقه بررسی‌های دوره‌ای." : "Target health parameters, SSL/TLS handshake certificate, and chronological execution history."}
        icon={<Activity className="w-5 h-5" />}
        breadcrumbs={[
          { label: t.monitors, href: "/monitors" },
          { label: monitor?.asset_name || t.monitorDetails },
        ]}
        badge={monitor && <Badge variant={monitor.status} />}
        actions={
          monitor && canManage ? (
            <div className="flex flex-wrap items-center gap-2">
              <Button
                size="sm"
                variant="primary"
                onClick={handleTestNow}
                loading={testing}
                icon={<Zap className="w-4 h-4 text-amber-300" />}
              >
                {testing ? t.testing : t.testNow}
              </Button>

              <Button
                size="sm"
                variant="outline"
                onClick={handleTogglePause}
                loading={actionLoading}
                icon={
                  monitor.status === "paused" ? (
                    <Play className="w-4 h-4 text-emerald-600" />
                  ) : (
                    <Pause className="w-4 h-4 text-slate-500" />
                  )
                }
              >
                {monitor.status === "paused" ? t.resumeMonitor : t.pauseMonitor}
              </Button>

              <Link href={`/monitors/${monitor.id}/edit`}>
                <Button size="sm" variant="outline" icon={<Edit className="w-4 h-4" />}>
                  {t.editMonitor}
                </Button>
              </Link>

              {canDelete && (
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => setDeleteModalOpen(true)}
                  icon={<Trash2 className="w-4 h-4 text-rose-500" />}
                >
                  {t.delete}
                </Button>
              )}
            </div>
          ) : null
        }
      />

      {error && <ErrorState error={error} onRetry={reloadMonitorData} className="mb-6" />}

      {loading ? (
        <TableSkeleton rows={6} cols={5} />
      ) : monitor ? (
        <div className="space-y-6 sm:space-y-8">
          {/* Monitor Configuration Summary Card */}
          <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-5 sm:p-7">
            <div className="flex flex-col md:flex-row md:items-center justify-between gap-6 pb-6 border-b border-slate-100">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-3 flex-wrap">
                  <h2 className="text-xl font-bold text-slate-900 tracking-tight">
                    {monitor.asset_name}
                  </h2>
                  <Badge variant={monitor.status} />
                </div>

                <div className="flex flex-wrap items-center gap-4 mt-2.5 text-xs text-slate-600">
                  <span className="flex items-center gap-1.5">
                    <Building className="w-3.5 h-3.5 text-slate-400 shrink-0" />
                    <span>{t.client}: </span>
                    <Link
                      href={`/clients/${monitor.client_id}`}
                      className="font-semibold text-blue-600 hover:underline"
                    >
                      {monitor.client_name}
                    </Link>
                  </span>

                  <a
                    href={monitor.effective_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1 text-blue-600 hover:underline font-mono text-start"
                    dir="ltr"
                  >
                    <Globe className="w-3.5 h-3.5 text-slate-400 shrink-0" />
                    <span className="truncate max-w-xs">{monitor.effective_url}</span>
                    <ExternalLink className="w-3 h-3 shrink-0" />
                  </a>
                </div>
              </div>

              {/* Interval & Timeout Metrics */}
              <div className="grid grid-cols-3 gap-2.5 sm:gap-3 text-center shrink-0">
                <div className="bg-slate-50 p-2.5 sm:p-3 rounded-xl border border-slate-100 min-w-[76px]">
                  <div className="text-[10px] uppercase font-bold text-slate-400">{t.interval}</div>
                  <div className="text-sm font-bold text-slate-900 mt-0.5" dir="ltr">
                    {monitor.interval_seconds}{t.secondsUnit}
                  </div>
                </div>
                <div className="bg-slate-50 p-2.5 sm:p-3 rounded-xl border border-slate-100 min-w-[76px]">
                  <div className="text-[10px] uppercase font-bold text-slate-400">{t.timeout}</div>
                  <div className="text-sm font-bold text-slate-900 mt-0.5" dir="ltr">
                    {monitor.timeout_seconds}{t.secondsUnit}
                  </div>
                </div>
                <div className="bg-slate-50 p-2.5 sm:p-3 rounded-xl border border-slate-100 min-w-[76px]">
                  <div className="text-[10px] uppercase font-bold text-slate-400">{t.method}</div>
                  <div className="text-sm font-bold text-slate-900 mt-0.5 uppercase font-mono" dir="ltr">
                    {monitor.method}
                  </div>
                </div>
              </div>
            </div>

            {/* Check Configuration Details */}
            <div className="pt-6 grid grid-cols-1 sm:grid-cols-3 gap-4 sm:gap-6 text-xs text-slate-600">
              <div>
                <span className="font-semibold text-slate-800">{t.expectedStatusRange}: </span>
                <span className="font-mono font-bold text-slate-900 ms-1" dir="ltr">
                  {monitor.expected_status_min}–{monitor.expected_status_max}
                </span>
              </div>
              <div>
                <span className="font-semibold text-slate-800">{t.followRedirectsLabel}: </span>
                <span className="ms-1">
                  {monitor.follow_redirects
                    ? (isRTL ? `بله (حداکثر ${monitor.max_redirects})` : `Yes (max ${monitor.max_redirects})`)
                    : (isRTL ? "خیر" : "No")}
                </span>
              </div>
              <div>
                <span className="font-semibold text-slate-800">{t.nextScheduledCheck}: </span>
                <span className="ms-1">
                  {monitor.status === "paused"
                    ? t.statusPaused
                    : new Date(monitor.next_check_at).toLocaleTimeString(undefined, {
                        timeZone: activeOrg?.timezone || "UTC",
                        hour: "2-digit",
                        minute: "2-digit",
                        second: "2-digit",
                      })}
                </span>
              </div>

              {monitor.expected_text && (
                <div className="sm:col-span-2">
                  <span className="font-semibold text-slate-800">{t.expectedTextLabel}: </span>
                  <span className="font-mono bg-slate-100 px-1.5 py-0.5 rounded text-slate-800 ms-1 text-[11px]" dir="ltr">
                    {monitor.expected_text}
                  </span>
                </div>
              )}

              {monitor.json_pointer && (
                <div className="sm:col-span-2">
                  <span className="font-semibold text-slate-800">{t.jsonAssertionLabel}: </span>
                  <span className="font-mono bg-slate-100 px-1.5 py-0.5 rounded text-slate-800 ms-1 text-[11px]" dir="ltr">
                    {monitor.json_pointer} = {monitor.json_expected_value}
                  </span>
                </div>
              )}
            </div>
          </div>

          {/* SSL / TLS Certificate Card */}
          {latestSSL && (
            <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-5 sm:p-7">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between pb-6 border-b border-slate-100 gap-3">
                <div className="flex items-center gap-2.5">
                  <ShieldCheck className="w-5 h-5 text-emerald-600 shrink-0" />
                  <div>
                    <h3 className="text-base font-bold text-slate-900 tracking-tight">
                      {t.sslDetails}
                    </h3>
                    <p className="text-xs text-slate-500 mt-0.5">
                      {isRTL
                        ? "راستی‌آزمایی پروتکل امنیتی TLS و تاریخ انقضای گواهینامه دامنه."
                        : "Verified TLS security handshake and certificate expiration."}
                    </p>
                  </div>
                </div>

                {latestSSL.tls_days_remaining !== undefined && latestSSL.tls_days_remaining < 30 ? (
                  <span className="inline-flex items-center gap-1.5 px-3 py-1 bg-amber-50 border border-amber-200 text-amber-800 rounded-full text-xs font-semibold w-fit">
                    <ShieldAlert className="w-3.5 h-3.5 text-amber-600 shrink-0" />
                    <span>{t.sslExpiryWarning}</span>
                  </span>
                ) : (
                  <span className="inline-flex items-center gap-1.5 px-3 py-1 bg-emerald-50 border border-emerald-200 text-emerald-800 rounded-full text-xs font-semibold w-fit">
                    <ShieldCheck className="w-3.5 h-3.5 text-emerald-600 shrink-0" />
                    <span>{t.sslValid}</span>
                  </span>
                )}
              </div>

              <div className="pt-6 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5 text-xs">
                <div>
                  <div className="text-slate-400 font-semibold uppercase text-[10px]">{t.sslSubject}</div>
                  <div className="text-slate-800 font-bold mt-1 font-mono text-start break-all" dir="ltr">
                    {latestSSL.tls_cert_subject || "—"}
                  </div>
                </div>
                <div>
                  <div className="text-slate-400 font-semibold uppercase text-[10px]">{t.sslIssuer}</div>
                  <div className="text-slate-800 font-bold mt-1 break-words">
                    {latestSSL.tls_cert_issuer || "—"}
                  </div>
                </div>
                <div>
                  <div className="text-slate-400 font-semibold uppercase text-[10px]">{t.sslDaysRemaining}</div>
                  <div className="text-slate-800 font-bold mt-1 text-sm font-mono" dir="ltr">
                    {latestSSL.tls_days_remaining !== undefined
                      ? `${latestSSL.tls_days_remaining} ${t.daysUnit}`
                      : "—"}
                  </div>
                </div>
                <div>
                  <div className="text-slate-400 font-semibold uppercase text-[10px]">{t.protocolAndCipher}</div>
                  <div
                    className="text-slate-800 font-bold mt-1 truncate font-mono text-[11px] text-start"
                    dir="ltr"
                    title={latestSSL.tls_cipher}
                  >
                    {latestSSL.tls_version} • {latestSSL.tls_cipher}
                  </div>
                </div>
              </div>
            </div>
          )}

          {/* Check Execution History */}
          <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
            <div className="p-4 sm:p-6 border-b border-slate-100 flex items-center justify-between">
              <div>
                <h3 className="text-base font-bold text-slate-900 tracking-tight flex items-center gap-2">
                  <Activity className="w-5 h-5 text-blue-600 shrink-0" />
                  <span>{t.checkHistory}</span>
                </h3>
                <p className="text-xs text-slate-500 mt-0.5">
                  {isRTL
                    ? "سوابق زمانی بررسی‌ها به همراه مدت زمان پاسخ، کدهای وضعیت و جزئیات خطاها."
                    : "Chronological log of checks with execution latencies, status codes and error traces."}
                </p>
              </div>

              <Button
                size="sm"
                variant="outline"
                onClick={reloadMonitorData}
                icon={<RotateCcw className="w-3.5 h-3.5" />}
              >
                {t.refresh}
              </Button>
            </div>

            {history.length === 0 ? (
              <div className="p-12 text-center text-slate-500 text-xs">{t.noHistory}</div>
            ) : (
              <>
                {/* Desktop Table View (>= 768px) */}
                <div className="hidden md:block overflow-x-auto">
                  <table className="w-full text-start text-sm" dir={isRTL ? "rtl" : "ltr"}>
                    <thead className="bg-slate-50 text-slate-600 text-xs uppercase font-semibold border-b border-slate-200">
                      <tr>
                        <th className="px-6 py-4">{t.status}</th>
                        <th className="px-6 py-4">{t.latestCheck}</th>
                        <th className="px-6 py-4">{t.latency}</th>
                        <th className="px-6 py-4">{t.statusCode}</th>
                        <th className="px-6 py-4">{t.executionType}</th>
                        <th className="px-6 py-4">{isRTL ? "جزئیات نتیجه / خطا" : "Result / Error Details"}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 text-xs">
                      {history.map((chk) => (
                        <tr key={chk.id} className="hover:bg-slate-50/70 transition-colors">
                          <td className="px-6 py-4">
                            <Badge variant={chk.result} />
                          </td>
                          <td className="px-6 py-4 text-slate-600">
                            {new Date(chk.completed_at).toLocaleTimeString(undefined, {
                              timeZone: activeOrg?.timezone || "UTC",
                              hour: "2-digit",
                              minute: "2-digit",
                              second: "2-digit",
                            })}
                          </td>
                          <td className="px-6 py-4 font-mono font-medium text-slate-800" dir="ltr">
                            {chk.latency_ms} ms
                          </td>
                          <td className="px-6 py-4 font-mono font-bold text-slate-800" dir="ltr">
                            {chk.http_status_code || "—"}
                          </td>
                          <td className="px-6 py-4">
                            <span
                              className={`text-[10px] uppercase font-bold px-2 py-0.5 rounded ${
                                chk.execution_type === "manual"
                                  ? "bg-purple-100 text-purple-800"
                                  : "bg-slate-100 text-slate-700"
                              }`}
                            >
                              {chk.execution_type === "manual" ? t.manual : t.scheduled}
                            </span>
                          </td>
                          <td className="px-6 py-4 text-slate-600">
                            {chk.error_message ? (
                              <span className="text-rose-600 font-mono text-[11px] font-medium" dir="ltr">
                                {chk.error_message}
                              </span>
                            ) : (
                              <span className="text-emerald-700 font-medium">{t.resultHttpSuccess}</span>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                {/* Mobile Card View (< 768px, 390px, 360px) */}
                <div className="md:hidden divide-y divide-slate-100">
                  {history.map((chk) => (
                    <div key={chk.id} className="p-3.5 space-y-2">
                      <div className="flex items-center justify-between gap-2">
                        <Badge variant={chk.result} />
                        <span className="text-[11px] text-slate-500">
                          {new Date(chk.completed_at).toLocaleTimeString(undefined, {
                            timeZone: activeOrg?.timezone || "UTC",
                            hour: "2-digit",
                            minute: "2-digit",
                            second: "2-digit",
                          })}
                        </span>
                      </div>

                      <div className="flex items-center justify-between text-xs pt-1">
                        <div className="flex items-center gap-2">
                          <span className="text-slate-500">{t.latency}:</span>
                          <span className="font-mono font-bold text-slate-800" dir="ltr">
                            {chk.latency_ms} ms
                          </span>
                        </div>

                        <div className="flex items-center gap-2">
                          <span className="text-slate-500">{t.statusCode}:</span>
                          <span className="font-mono font-bold text-slate-800" dir="ltr">
                            {chk.http_status_code || "—"}
                          </span>
                          <span
                            className={`text-[9px] uppercase font-bold px-1.5 py-0.5 rounded ${
                              chk.execution_type === "manual"
                                ? "bg-purple-100 text-purple-800"
                                : "bg-slate-100 text-slate-700"
                            }`}
                          >
                            {chk.execution_type === "manual" ? t.manual : t.scheduled}
                          </span>
                        </div>
                      </div>

                      {chk.error_message ? (
                        <div className="text-[11px] font-mono text-rose-600 bg-rose-50/70 p-2 rounded-lg border border-rose-100 break-all text-start" dir="ltr">
                          {chk.error_message}
                        </div>
                      ) : (
                        <div className="text-[11px] text-emerald-700 font-medium">
                          {t.resultHttpSuccess}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </>
            )}
          </div>
        </div>
      ) : null}

      {/* Standardized Delete Confirmation Modal */}
      <ConfirmModal
        isOpen={deleteModalOpen}
        onClose={() => setDeleteModalOpen(false)}
        onConfirm={handleDelete}
        title={t.deleteMonitorTitle}
        message={t.deleteMonitorConfirm}
        confirmLabel={t.delete}
        cancelLabel={t.cancel}
        variant="danger"
        loading={actionLoading}
      />
    </AppLayout>
  );
}
