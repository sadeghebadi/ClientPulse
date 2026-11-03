"use client";

import React, { useEffect, useState, useCallback, useMemo } from "react";
import Link from "next/link";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import {
  Activity,
  Plus,
  Play,
  Pause,
  ExternalLink,
  ArrowRight,
  ArrowLeft,
  Building,
  Info,
  Search,
  Clock,
  RotateCw,
} from "lucide-react";
import { PageHeader } from "@/components/ui/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { useToast } from "@/components/ui/Toast";

interface MonitorItem {
  id: string;
  asset_name: string;
  asset_url: string;
  client_name: string;
  client_id: string;
  effective_url: string;
  method: string;
  status: "up" | "down" | "suspected_down" | "pending" | "paused" | "unknown";
  interval_seconds: number;
  last_checked_at?: string;
  next_check_at: string;
}

export default function MonitorsPage() {
  const { user, activeOrg, role } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const [monitors, setMonitors] = useState<MonitorItem[]>([]);
  const [total, setTotal] = useState(0);
  const [statusFilter, setStatusFilter] = useState("all");
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<any>(null);

  const canManage = role === "owner" || role === "admin" || role === "member";

  const reloadMonitors = useCallback(() => {
    if (!user || !activeOrg) return;
    setLoading(true);
    setError(null);

    const params = new URLSearchParams();
    if (statusFilter && statusFilter !== "all") params.set("status", statusFilter);

    apiFetch<{ monitors: MonitorItem[]; total: number }>(`/api/v1/monitors?${params.toString()}`)
      .then((data) => {
        setMonitors(data.monitors || []);
        setTotal(data.total || 0);
      })
      .catch((err) => {
        setError(err);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [user, activeOrg, statusFilter]);

  useEffect(() => {
    if (!user || !activeOrg) return;
    let isMounted = true;

    const params = new URLSearchParams();
    if (statusFilter && statusFilter !== "all") params.set("status", statusFilter);

    apiFetch<{ monitors: MonitorItem[]; total: number }>(`/api/v1/monitors?${params.toString()}`)
      .then((data) => {
        if (!isMounted) return;
        setMonitors(data.monitors || []);
        setTotal(data.total || 0);
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
  }, [user, activeOrg, statusFilter]);

  const handleTogglePause = async (id: string, currentStatus: string) => {
    try {
      const endpoint =
        currentStatus === "paused" ? `/api/v1/monitors/${id}/resume` : `/api/v1/monitors/${id}/pause`;
      await apiFetch(endpoint, { method: "POST" });
      toast.success(
        currentStatus === "paused"
          ? (isRTL ? "پایش با موفقیت فعال شد." : "Monitor resumed. Scheduled checks re-activated.")
          : (isRTL ? "پایش موقتاً متوقف شد." : "Monitor paused. Scheduled checks halted.")
      );
      reloadMonitors();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to toggle monitor state");
    }
  };

  const filteredMonitors = useMemo(() => {
    if (!search.trim()) return monitors;
    const q = search.trim().toLowerCase();
    return monitors.filter(
      (m) =>
        m.asset_name.toLowerCase().includes(q) ||
        m.effective_url.toLowerCase().includes(q) ||
        (m.client_name && m.client_name.toLowerCase().includes(q))
    );
  }, [monitors, search]);

  return (
    <AppLayout>
      <PageHeader
        title={t.monitors}
        description={isRTL ? "پایش خودکار و بی‌درنگ سلامت، زمان پاسخ و گواهینامه‌های TLS وب‌سایت‌ها." : "Live automated uptime verification, latency tracking, and TLS security assertions."}
        icon={<Activity className="w-5 h-5" />}
        breadcrumbs={[
          { label: t.dashboard, href: "/dashboard" },
          { label: t.monitors },
        ]}
        actions={
          canManage ? (
            <Link href="/monitors/new">
              <Button size="sm" variant="primary" icon={<Plus className="w-4 h-4" />}>
                {t.createMonitor}
              </Button>
            </Link>
          ) : null
        }
      />

      {/* Threshold Explanation Banner */}
      <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-4 sm:p-5 mb-6 text-xs text-slate-600">
        <div className="flex items-center gap-2 mb-2 font-bold text-slate-900">
          <Info className="w-4 h-4 text-blue-600 shrink-0" />
          <span>{t.monitorThresholds}</span>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 text-[11px] leading-relaxed">
          <div className="p-2.5 rounded-xl bg-emerald-50/60 border border-emerald-100">
            <strong className="text-emerald-800 block mb-0.5">{t.statusUp}:</strong>
            <span>{t.thresholdUpDesc}</span>
          </div>
          <div className="p-2.5 rounded-xl bg-amber-50/60 border border-amber-100">
            <strong className="text-amber-800 block mb-0.5">{t.statusSuspectedDown}:</strong>
            <span>{t.thresholdSuspectedDesc}</span>
          </div>
          <div className="p-2.5 rounded-xl bg-rose-50/60 border border-rose-100">
            <strong className="text-rose-800 block mb-0.5">{t.statusDown}:</strong>
            <span>{t.thresholdDownDesc}</span>
          </div>
        </div>
      </div>

      {/* Filter & Search Bar */}
      <div className="bg-white p-3.5 sm:p-4 rounded-2xl border border-slate-200 shadow-sm mb-6 flex flex-col sm:flex-row gap-3 items-stretch sm:items-center justify-between">
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2.5 flex-1">
          {/* Status select */}
          <div className="flex items-center gap-2 shrink-0">
            <span className="text-xs font-semibold text-slate-600">{t.status}:</span>
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="px-3 py-1.5 bg-slate-50 border border-slate-200 rounded-xl text-xs font-semibold text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500 w-full sm:w-auto"
            >
              <option value="all">{t.allStatuses}</option>
              <option value="up">{t.statusUp}</option>
              <option value="suspected_down">{t.statusSuspectedDown}</option>
              <option value="down">{t.statusDown}</option>
              <option value="pending">{t.statusPending}</option>
              <option value="paused">{t.statusPaused}</option>
              <option value="unknown">{t.statusUnknown}</option>
            </select>
          </div>

          {/* Search Input */}
          <div className="relative flex-1">
            <Search
              className={`w-3.5 h-3.5 text-slate-400 absolute top-1/2 -translate-y-1/2 ${
                isRTL ? "right-3" : "left-3"
              }`}
            />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t.searchMonitors}
              className={`w-full py-1.5 bg-slate-50 border border-slate-200 rounded-xl text-xs focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all ${
                isRTL ? "pr-8 pl-3" : "pl-8 pr-3"
              }`}
            />
          </div>
        </div>

        <div className="flex items-center justify-between sm:justify-end gap-3 text-xs font-medium text-slate-500 shrink-0 border-t sm:border-t-0 pt-2 sm:pt-0 border-slate-100">
          <div>
            {t.totalCount}: <span className="font-bold text-slate-900">{filteredMonitors.length}</span>
            {filteredMonitors.length !== total && (
              <span className="text-slate-400 ms-1">({total})</span>
            )}
          </div>
          <Button
            size="sm"
            variant="ghost"
            onClick={reloadMonitors}
            icon={<RotateCw className="w-3.5 h-3.5 text-slate-400" />}
          >
            {t.refresh}
          </Button>
        </div>
      </div>

      {error && <ErrorState error={error} onRetry={reloadMonitors} className="mb-6" />}

      {/* Main Content Area */}
      {loading ? (
        <TableSkeleton rows={5} cols={6} />
      ) : filteredMonitors.length === 0 ? (
        <EmptyState
          icon={<Activity className="w-6 h-6 text-slate-400" />}
          title={search ? t.noHistory : t.noMonitorsConfigured}
          description={
            search
              ? (isRTL ? "هیچ مانیتوری با مشخصات جستجوشده یافت نشد." : "No monitors match your current search criteria.")
              : t.noMonitorsConfiguredDesc
          }
          actionHref={canManage ? "/monitors/new" : undefined}
          actionLabel={canManage ? t.createMonitor : undefined}
        />
      ) : (
        <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
          {/* Desktop Table View (>= 768px) */}
          <div className="hidden md:block overflow-x-auto">
            <table className="w-full text-start text-sm" dir={isRTL ? "rtl" : "ltr"}>
              <thead className="bg-slate-50 text-slate-600 text-xs uppercase font-semibold border-b border-slate-200">
                <tr>
                  <th className="px-6 py-4">{t.status}</th>
                  <th className="px-6 py-4">{t.assetName}</th>
                  <th className="px-6 py-4">{t.clients}</th>
                  <th className="px-6 py-4">{t.method}</th>
                  <th className="px-6 py-4">{t.latestCheck}</th>
                  <th className="px-6 py-4 text-end">{t.actions}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {filteredMonitors.map((m) => (
                  <tr key={m.id} className="hover:bg-slate-50/70 transition-colors">
                    <td className="px-6 py-4">
                      <Badge variant={m.status} />
                    </td>

                    <td className="px-6 py-4">
                      <Link
                        href={`/monitors/${m.id}`}
                        className="font-bold text-slate-900 hover:text-blue-600 transition-colors block"
                      >
                        {m.asset_name}
                      </Link>
                      <div
                        dir="ltr"
                        className="text-xs text-slate-400 font-mono truncate max-w-xs mt-0.5 text-start"
                      >
                        {m.effective_url}
                      </div>
                    </td>

                    <td className="px-6 py-4 text-xs">
                      <Link
                        href={`/clients/${m.client_id}`}
                        className="font-semibold text-slate-700 hover:text-blue-600 flex items-center gap-1.5"
                      >
                        <Building className="w-3.5 h-3.5 text-slate-400 shrink-0" />
                        <span className="truncate max-w-[160px]">{m.client_name}</span>
                      </Link>
                    </td>

                    <td className="px-6 py-4">
                      <span className="text-[11px] font-mono font-bold px-2 py-0.5 rounded bg-slate-100 text-slate-700 uppercase" dir="ltr">
                        {m.method}
                      </span>
                    </td>

                    <td className="px-6 py-4 text-xs text-slate-600">
                      {m.last_checked_at
                        ? new Date(m.last_checked_at).toLocaleTimeString(undefined, {
                            timeZone: activeOrg?.timezone || "UTC",
                            hour: "2-digit",
                            minute: "2-digit",
                            second: "2-digit",
                          })
                        : <span className="text-slate-400 italic">{t.statusPending}</span>}
                    </td>

                    <td className="px-6 py-4 text-end">
                      <div className="flex items-center justify-end gap-2">
                        {canManage && (
                          <button
                            onClick={() => handleTogglePause(m.id, m.status)}
                            className="p-1.5 text-slate-500 hover:text-slate-800 hover:bg-slate-100 rounded-lg transition-colors border border-slate-200"
                            title={m.status === "paused" ? t.resumeMonitor : t.pauseMonitor}
                            aria-label={m.status === "paused" ? t.resumeMonitor : t.pauseMonitor}
                          >
                            {m.status === "paused" ? (
                              <Play className="w-3.5 h-3.5 text-emerald-600" />
                            ) : (
                              <Pause className="w-3.5 h-3.5 text-slate-600" />
                            )}
                          </button>
                        )}
                        <Link
                          href={`/monitors/${m.id}`}
                          className="inline-flex items-center gap-1 text-xs font-semibold text-blue-600 hover:text-blue-800 transition-colors ms-1"
                        >
                          <span>{t.viewDetails}</span>
                          {isRTL ? <ArrowLeft className="w-3.5 h-3.5" /> : <ArrowRight className="w-3.5 h-3.5" />}
                        </Link>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Mobile Card View (< 768px, 390px, 360px) */}
          <div className="md:hidden divide-y divide-slate-100">
            {filteredMonitors.map((m) => (
              <div key={m.id} className="p-4 space-y-3">
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0 flex-1">
                    <Link
                      href={`/monitors/${m.id}`}
                      className="font-bold text-slate-900 text-sm hover:text-blue-600 truncate block"
                    >
                      {m.asset_name}
                    </Link>
                    <div
                      dir="ltr"
                      className="text-xs text-slate-500 font-mono truncate max-w-full mt-0.5 text-start"
                    >
                      {m.effective_url}
                    </div>
                  </div>
                  <Badge variant={m.status} />
                </div>

                <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-slate-600 pt-1">
                  <Link
                    href={`/clients/${m.client_id}`}
                    className="flex items-center gap-1 text-slate-700 hover:text-blue-600 font-medium"
                  >
                    <Building className="w-3.5 h-3.5 text-slate-400 shrink-0" />
                    <span className="truncate max-w-[180px]">{m.client_name}</span>
                  </Link>

                  <div className="flex items-center gap-2">
                    <span className="font-mono text-[11px] bg-slate-100 px-1.5 py-0.5 rounded font-bold uppercase" dir="ltr">
                      {m.method}
                    </span>
                    <span className="text-[11px] text-slate-500 font-mono" dir="ltr">
                      {m.interval_seconds}{t.secondsUnit}
                    </span>
                  </div>
                </div>

                <div className="flex items-center gap-1.5 text-[11px] text-slate-500">
                  <Clock className="w-3.5 h-3.5 text-slate-400 shrink-0" />
                  <span>{t.latestCheck}: </span>
                  <span className="font-medium text-slate-700">
                    {m.last_checked_at
                      ? new Date(m.last_checked_at).toLocaleTimeString(undefined, {
                          timeZone: activeOrg?.timezone || "UTC",
                          hour: "2-digit",
                          minute: "2-digit",
                          second: "2-digit",
                        })
                      : t.statusPending}
                  </span>
                </div>

                <div className="flex items-center justify-between pt-2.5 border-t border-slate-50 gap-2">
                  {canManage ? (
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => handleTogglePause(m.id, m.status)}
                      icon={
                        m.status === "paused" ? (
                          <Play className="w-3.5 h-3.5 text-emerald-600" />
                        ) : (
                          <Pause className="w-3.5 h-3.5 text-slate-600" />
                        )
                      }
                    >
                      {m.status === "paused" ? t.resumeMonitor : t.pauseMonitor}
                    </Button>
                  ) : <div />}
                  <Link href={`/monitors/${m.id}`}>
                    <Button size="sm" variant="secondary">
                      {t.viewDetails}
                    </Button>
                  </Link>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </AppLayout>
  );
}
