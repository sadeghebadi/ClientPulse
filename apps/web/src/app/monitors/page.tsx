"use client";

import React, { useEffect, useState, useCallback } from "react";
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
  const { user, activeOrg } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const [monitors, setMonitors] = useState<MonitorItem[]>([]);
  const [total, setTotal] = useState(0);
  const [statusFilter, setStatusFilter] = useState("all");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<any>(null);

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
          ? "Monitor resumed. Scheduled checks re-activated."
          : "Monitor paused. Scheduled checks halted."
      );
      reloadMonitors();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to toggle monitor state");
    }
  };

  return (
    <AppLayout>
      <PageHeader
        title={t.monitors}
        description="Live automated uptime verification, latency tracking, and TLS security assertions."
        icon={<Activity className="w-5 h-5" />}
        actions={
          <Link href="/monitors/new">
            <Button size="sm" variant="primary" icon={<Plus className="w-4 h-4" />}>
              {t.createMonitor}
            </Button>
          </Link>
        }
      />

      {/* Threshold Explanation Banner */}
      <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-4 sm:p-5 mb-6 text-xs text-slate-600">
        <div className="flex items-center gap-2 mb-2 font-bold text-slate-900">
          <Info className="w-4 h-4 text-blue-600 shrink-0" />
          <span>Monitor Operational State Thresholds:</span>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 text-[11px] leading-relaxed">
          <div className="p-2.5 rounded-xl bg-emerald-50/50 border border-emerald-100">
            <strong className="text-emerald-800">Operational (Up):</strong> Last check succeeded with expected HTTP status and assertions.
          </div>
          <div className="p-2.5 rounded-xl bg-amber-50/50 border border-amber-100">
            <strong className="text-amber-800">Suspected Down:</strong> 1 check failed. Verifying in the next cycle before alerting to avoid false alarms.
          </div>
          <div className="p-2.5 rounded-xl bg-rose-50/50 border border-rose-100">
            <strong className="text-rose-800">Downtime (Down):</strong> 2+ consecutive checks failed. Site is confirmed unreachable or in error.
          </div>
        </div>
      </div>

      {/* Filter Bar */}
      <div className="bg-white p-4 rounded-2xl border border-slate-200 shadow-sm mb-6 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-xs font-semibold text-slate-600">{t.status}:</span>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="px-3 py-1.5 bg-slate-50 border border-slate-200 rounded-xl text-xs font-semibold text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
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

        <div className="text-xs font-medium text-slate-500">
          Total: <span className="font-bold text-slate-900">{total}</span>
        </div>
      </div>

      {error && <ErrorState error={error} onRetry={reloadMonitors} className="mb-6" />}

      {/* Main Content Area */}
      {loading ? (
        <TableSkeleton rows={5} cols={6} />
      ) : monitors.length === 0 ? (
        <EmptyState
          icon={<Activity className="w-6 h-6 text-slate-400" />}
          title={t.noHistory}
          description="Configure automated monitoring for your registered client websites."
          actionHref="/monitors/new"
          actionLabel={t.createMonitor}
        />
      ) : (
        <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
          {/* Desktop Table View */}
          <div className="hidden md:block overflow-x-auto">
            <table className="w-full text-left text-sm" dir={isRTL ? "rtl" : "ltr"}>
              <thead className="bg-slate-50 text-slate-600 text-xs uppercase font-semibold border-b border-slate-200">
                <tr>
                  <th className="px-6 py-4">{t.status}</th>
                  <th className="px-6 py-4">{t.assetName}</th>
                  <th className="px-6 py-4">{t.clients}</th>
                  <th className="px-6 py-4">{t.method}</th>
                  <th className="px-6 py-4">{t.latestCheck}</th>
                  <th className="px-6 py-4 text-right">{t.actions}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {monitors.map((m) => (
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
                      <div className="text-xs text-slate-400 font-mono truncate max-w-xs mt-0.5">
                        {m.effective_url}
                      </div>
                    </td>

                    <td className="px-6 py-4 text-xs">
                      <Link
                        href={`/clients/${m.client_id}`}
                        className="font-semibold text-slate-700 hover:text-blue-600 flex items-center gap-1.5"
                      >
                        <Building className="w-3.5 h-3.5 text-slate-400" />
                        <span>{m.client_name}</span>
                      </Link>
                    </td>

                    <td className="px-6 py-4">
                      <span className="text-[11px] font-mono font-bold px-2 py-0.5 rounded bg-slate-100 text-slate-700 uppercase">
                        {m.method}
                      </span>
                    </td>

                    <td className="px-6 py-4 text-xs text-slate-600">
                      {m.last_checked_at
                        ? new Date(m.last_checked_at).toLocaleTimeString(undefined, {
                            timeZone: activeOrg?.timezone || "UTC",
                          })
                        : "Pending"}
                    </td>

                    <td className="px-6 py-4 text-right">
                      <div className="flex items-center justify-end gap-2">
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

          {/* Mobile Card View (< 768px) */}
          <div className="md:hidden divide-y divide-slate-100">
            {monitors.map((m) => (
              <div key={m.id} className="p-4 space-y-3">
                <div className="flex items-start justify-between gap-2">
                  <div>
                    <Link
                      href={`/monitors/${m.id}`}
                      className="font-bold text-slate-900 text-sm hover:text-blue-600"
                    >
                      {m.asset_name}
                    </Link>
                    <div className="text-xs text-slate-500 font-mono truncate max-w-xs mt-0.5">
                      {m.effective_url}
                    </div>
                  </div>
                  <Badge variant={m.status} />
                </div>

                <div className="flex items-center justify-between text-xs text-slate-600 pt-1">
                  <div className="flex items-center gap-1">
                    <Building className="w-3.5 h-3.5 text-slate-400" />
                    <span>{m.client_name}</span>
                  </div>
                  <div className="font-mono text-[11px] bg-slate-100 px-1.5 py-0.5 rounded">
                    {m.method} • {m.interval_seconds}s
                  </div>
                </div>

                <div className="flex items-center justify-between pt-2 border-t border-slate-50">
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
