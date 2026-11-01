"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Header } from "@/components/Header";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import {
  Activity,
  Plus,
  Play,
  Pause,
  ExternalLink,
  CheckCircle,
  XCircle,
  Clock,
  PauseCircle,
  ArrowRight,
  ArrowLeft,
  AlertCircle,
  Globe,
  Building,
} from "lucide-react";

interface MonitorItem {
  id: string;
  asset_name: string;
  asset_url: string;
  client_name: string;
  effective_url: string;
  method: string;
  status: "up" | "down" | "pending" | "paused";
  interval_seconds: number;
  last_checked_at?: string;
  next_check_at: string;
}

export default function MonitorsPage() {
  const { user, activeOrg } = useAuth();
  const { t, isRTL } = useI18n();

  const [monitors, setMonitors] = useState<MonitorItem[]>([]);
  const [total, setTotal] = useState(0);
  const [statusFilter, setStatusFilter] = useState("all");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const reloadMonitors = () => {
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
        setError(err?.message || "Failed to load monitors");
      })
      .finally(() => {
        setLoading(false);
      });
  };

  useEffect(() => {
    if (!user || !activeOrg) return;
    let isMounted = true;
    const params = new URLSearchParams();
    if (statusFilter && statusFilter !== "all") params.set("status", statusFilter);

    apiFetch<{ monitors: MonitorItem[]; total: number }>(`/api/v1/monitors?${params.toString()}`)
      .then((data) => {
        if (isMounted) {
          setMonitors(data.monitors || []);
          setTotal(data.total || 0);
          setLoading(false);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setError(err?.message || "Failed to load monitors");
          setLoading(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user, activeOrg, statusFilter]);

  const handleTogglePause = async (id: string, currentStatus: string) => {
    try {
      const endpoint = currentStatus === "paused" ? `/api/v1/monitors/${id}/resume` : `/api/v1/monitors/${id}/pause`;
      await apiFetch(endpoint, { method: "POST" });
      reloadMonitors();
    } catch (err: any) {
      alert(err?.message || "Action failed");
    }
  };

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">
      <Header />

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 flex-1 w-full">
        {/* Page Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-8">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 tracking-tight flex items-center gap-2.5">
              <Activity className="w-7 h-7 text-blue-600" />
              <span>{t.monitors}</span>
            </h1>
            <p className="text-sm text-slate-600 mt-1">
              Live automated health checks, uptime monitoring, and latency tracking.
            </p>
          </div>

          <Link
            href="/monitors/new"
            className="inline-flex items-center gap-2 px-4 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-semibold shadow-sm transition-colors"
          >
            <Plus className="w-4 h-4" />
            <span>{t.createMonitor}</span>
          </Link>
        </div>

        {/* Filter Bar */}
        <div className="bg-white p-4 rounded-xl border border-slate-200 shadow-sm mb-6 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-xs font-semibold text-slate-600">{t.status}:</span>
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="px-3 py-1.5 bg-slate-50 border border-slate-200 rounded-lg text-xs font-semibold text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="all">{t.allStatuses}</option>
              <option value="up">{t.statusUp}</option>
              <option value="down">{t.statusDown}</option>
              <option value="pending">{t.statusPending}</option>
              <option value="paused">{t.statusPaused}</option>
            </select>
          </div>

          <div className="text-xs font-medium text-slate-500">
            Total: <span className="font-bold text-slate-800">{total}</span>
          </div>
        </div>

        {error && (
          <div className="p-4 mb-6 rounded-lg bg-red-50 border border-red-200 text-red-700 text-xs flex items-center gap-2">
            <AlertCircle className="w-4 h-4 flex-shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {loading ? (
          <div className="p-12 text-center text-slate-500 text-sm bg-white rounded-xl border border-slate-200">
            {t.loading}
          </div>
        ) : monitors.length === 0 ? (
          <div className="p-12 text-center bg-white rounded-xl border border-dashed border-slate-300">
            <Activity className="w-12 h-12 text-slate-300 mx-auto mb-3" />
            <h3 className="text-base font-semibold text-slate-800">{t.noHistory}</h3>
            <p className="text-xs text-slate-500 mt-1 max-w-sm mx-auto mb-6">
              Configure automated monitoring for your registered client websites.
            </p>
            <Link
              href="/monitors/new"
              className="inline-flex items-center gap-1.5 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-semibold shadow-sm transition-colors"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>{t.createMonitor}</span>
            </Link>
          </div>
        ) : (
          <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
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
                        {m.status === "up" && (
                          <span className="inline-flex items-center gap-1.5 text-xs font-semibold text-emerald-700 bg-emerald-50 px-2.5 py-1 rounded-full border border-emerald-200">
                            <CheckCircle className="w-3.5 h-3.5 text-emerald-600" />
                            <span>{t.statusUp}</span>
                          </span>
                        )}
                        {m.status === "down" && (
                          <span className="inline-flex items-center gap-1.5 text-xs font-semibold text-rose-700 bg-rose-50 px-2.5 py-1 rounded-full border border-rose-200">
                            <XCircle className="w-3.5 h-3.5 text-rose-600" />
                            <span>{t.statusDown}</span>
                          </span>
                        )}
                        {m.status === "paused" && (
                          <span className="inline-flex items-center gap-1.5 text-xs font-semibold text-slate-700 bg-slate-100 px-2.5 py-1 rounded-full border border-slate-200">
                            <PauseCircle className="w-3.5 h-3.5 text-slate-500" />
                            <span>{t.statusPaused}</span>
                          </span>
                        )}
                        {m.status === "pending" && (
                          <span className="inline-flex items-center gap-1.5 text-xs font-semibold text-amber-700 bg-amber-50 px-2.5 py-1 rounded-full border border-amber-200">
                            <Clock className="w-3.5 h-3.5 text-amber-600" />
                            <span>{t.statusPending}</span>
                          </span>
                        )}
                      </td>
                      <td className="px-6 py-4 font-semibold text-slate-900">
                        <Link href={`/monitors/${m.id}`} className="hover:text-blue-600 transition-colors block">
                          {m.asset_name}
                        </Link>
                        <div className="text-xs text-slate-400 font-normal truncate max-w-xs mt-0.5">
                          {m.effective_url}
                        </div>
                      </td>
                      <td className="px-6 py-4 text-xs text-slate-600">
                        <span className="font-medium text-slate-800">{m.client_name}</span>
                      </td>
                      <td className="px-6 py-4">
                        <span className="text-[11px] font-mono font-bold px-2 py-0.5 rounded bg-slate-100 text-slate-700">
                          {m.method}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-xs text-slate-500">
                        {m.last_checked_at ? new Date(m.last_checked_at).toLocaleTimeString() : "Pending"}
                      </td>
                      <td className="px-6 py-4 text-right">
                        <div className="flex items-center justify-end gap-2">
                          <button
                            onClick={() => handleTogglePause(m.id, m.status)}
                            className="p-1.5 text-slate-500 hover:text-slate-800 hover:bg-slate-100 rounded-md transition-colors"
                            title={m.status === "paused" ? t.resumeMonitor : t.pauseMonitor}
                          >
                            {m.status === "paused" ? (
                              <Play className="w-3.5 h-3.5 text-emerald-600" />
                            ) : (
                              <Pause className="w-3.5 h-3.5 text-slate-600" />
                            )}
                          </button>
                          <Link
                            href={`/monitors/${m.id}`}
                            className="inline-flex items-center gap-1 text-xs font-semibold text-blue-600 hover:text-blue-800 transition-colors ml-2"
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
          </div>
        )}
      </main>
    </div>
  );
}
