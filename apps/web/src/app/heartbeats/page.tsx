"use client";

import React, { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import {
  Radio,
  Plus,
  Play,
  Pause,
  ExternalLink,
  ArrowRight,
  ArrowLeft,
  Clock,
  Info,
  RotateCw,
  Search,
  Building,
  AlertTriangle,
} from "lucide-react";
import { PageHeader } from "@/components/ui/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { useToast } from "@/components/ui/Toast";

interface HeartbeatItem {
  id: string;
  client_id: string;
  client_name: string;
  asset_id?: string;
  asset_name?: string;
  name: string;
  description?: string;
  expected_interval_seconds: number;
  grace_period_seconds: number;
  status: "pending" | "healthy" | "late" | "missed" | "paused" | "revoked";
  last_received_at?: string;
  next_expected_at?: string;
  created_at: string;
}

export default function HeartbeatsPage() {
  const router = useRouter();
  const { user, activeOrg, role } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const [heartbeats, setHeartbeats] = useState<HeartbeatItem[]>([]);
  const [total, setTotal] = useState(0);
  const [statusFilter, setStatusFilter] = useState("all");
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<any>(null);

  const canManage = role === "owner" || role === "admin" || role === "member";

  const fetchHeartbeats = useCallback(() => {
    if (!user || !activeOrg) return;
    setLoading(true);
    setError(null);

    const params = new URLSearchParams();
    if (statusFilter && statusFilter !== "all") params.set("status", statusFilter);
    if (search.trim()) params.set("search", search.trim());

    apiFetch<{ heartbeats: HeartbeatItem[]; total: number }>(`/api/v1/heartbeats?${params.toString()}`)
      .then((data) => {
        setHeartbeats(data.heartbeats || []);
        setTotal(data.total || 0);
      })
      .catch((err) => {
        setError(err);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [user, activeOrg, statusFilter, search]);

  useEffect(() => {
    if (!user || !activeOrg) return;
    let isMounted = true;

    const params = new URLSearchParams();
    if (statusFilter && statusFilter !== "all") params.set("status", statusFilter);
    if (search.trim()) params.set("search", search.trim());

    apiFetch<{ heartbeats: HeartbeatItem[]; total: number }>(`/api/v1/heartbeats?${params.toString()}`)
      .then((data) => {
        if (!isMounted) return;
        setHeartbeats(data.heartbeats || []);
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
  }, [user, activeOrg, statusFilter, search]);

  const handleTogglePause = async (id: string, currentStatus: string) => {
    const isPaused = currentStatus === "paused";
    const endpoint = isPaused
      ? `/api/v1/heartbeats/${id}/resume`
      : `/api/v1/heartbeats/${id}/pause`;

    try {
      await apiFetch(endpoint, { method: "POST" });
      toast.success(isPaused ? t.heartbeatResumed : t.heartbeatPaused);
      fetchHeartbeats();
    } catch (err: any) {
      toast.error(err.message || "Failed to toggle status");
    }
  };

  const formatInterval = (sec: number) => {
    if (sec >= 86400) return `${Math.round(sec / 86400)} ${t.daysUnit}`;
    if (sec >= 3600) return `${Math.round(sec / 3600)} ${t.hoursUnit}`;
    if (sec >= 60) return `${Math.round(sec / 60)} ${t.minutesUnit}`;
    return `${sec}${t.secondsUnit}`;
  };

  return (
    <AppLayout>
      <div className="space-y-6">
        <PageHeader
          title={t.heartbeats}
          description={t.createHeartbeatDesc}
          breadcrumbs={[
            { label: t.dashboard, href: "/" },
            { label: t.heartbeats },
          ]}
          actions={
            canManage ? (
              <Link href="/heartbeats/new">
                <Button size="sm" variant="primary" icon={<Plus className="w-4 h-4" />}>
                  {t.addHeartbeat}
                </Button>
              </Link>
            ) : null
          }
        />

        {/* Semantic Limitation Alert */}
        <div className="flex items-start gap-3 p-4 bg-sky-50 border border-sky-200 rounded-xl text-xs text-sky-900">
          <Info className="w-4 h-4 text-sky-600 mt-0.5 shrink-0" />
          <p className="leading-relaxed">{t.semanticNotice}</p>
        </div>

        {/* Filters and Search Bar */}
        <div className="bg-white border border-slate-200 rounded-2xl p-3.5 sm:p-4 shadow-sm flex flex-col md:flex-row gap-3 justify-between items-stretch md:items-center">
          <div className="flex flex-wrap items-center gap-1.5 sm:gap-2">
            {[
              { id: "all", label: t.all },
              { id: "healthy", label: t.statusHealthy },
              { id: "missed", label: t.statusMissed },
              { id: "late", label: t.statusLate },
              { id: "pending", label: t.statusPending },
              { id: "paused", label: t.statusPaused },
            ].map((f) => (
              <button
                key={f.id}
                onClick={() => setStatusFilter(f.id)}
                className={`px-3 py-1.5 rounded-xl text-xs font-medium transition-colors ${
                  statusFilter === f.id
                    ? "bg-slate-900 text-white shadow-sm"
                    : "bg-slate-50 text-slate-600 hover:bg-slate-100 border border-slate-200"
                }`}
              >
                {f.label}
              </button>
            ))}
          </div>

          <div className="flex items-center gap-2">
            <div className="relative flex-1 md:w-56">
              <Search
                className={`w-3.5 h-3.5 text-slate-400 absolute top-1/2 -translate-y-1/2 ${
                  isRTL ? "right-3" : "left-3"
                }`}
              />
              <input
                type="text"
                placeholder={t.searchHeartbeats}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className={`text-xs py-1.5 rounded-xl border border-slate-200 bg-slate-50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 w-full ${
                  isRTL ? "pr-8 pl-3" : "pl-8 pr-3"
                }`}
              />
            </div>
            <Button
              variant="secondary"
              size="sm"
              icon={<RotateCw className="w-3.5 h-3.5" />}
              onClick={fetchHeartbeats}
            >
              {t.refresh}
            </Button>
          </div>
        </div>

        {/* Heartbeats Content */}
        {loading ? (
          <div className="bg-white border border-slate-200 rounded-2xl p-6 shadow-sm">
            <TableSkeleton rows={5} cols={6} />
          </div>
        ) : error ? (
          <ErrorState error={error} onRetry={fetchHeartbeats} />
        ) : heartbeats.length === 0 ? (
          <EmptyState
            icon={<Radio className="w-10 h-10 text-slate-400" />}
            title={t.noHeartbeatsFound}
            description={t.createHeartbeatDesc}
            actionLabel={canManage ? t.addHeartbeat : undefined}
            onAction={canManage ? () => router.push("/heartbeats/new") : undefined}
          />
        ) : (
          <div className="bg-white border border-slate-200 rounded-2xl shadow-sm overflow-hidden">
            {/* Desktop Table View (>= 768px) */}
            <div className="hidden md:block overflow-x-auto">
              <table className="w-full text-start border-collapse text-xs" dir={isRTL ? "rtl" : "ltr"}>
                <thead>
                  <tr className="bg-slate-50/75 border-b border-slate-200 text-slate-500 font-semibold uppercase tracking-wider text-[11px]">
                    <th className="py-3 px-4">{t.heartbeatName}</th>
                    <th className="py-3 px-4">{t.client}</th>
                    <th className="py-3 px-4">{t.expectedInterval}</th>
                    <th className="py-3 px-4">{t.gracePeriod}</th>
                    <th className="py-3 px-4">{t.status}</th>
                    <th className="py-3 px-4">{t.lastReceived}</th>
                    <th className="py-3 px-4">{t.nextExpected}</th>
                    <th className="py-3 px-4 text-end">{t.actions}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {heartbeats.map((hb) => (
                    <tr
                      key={hb.id}
                      className="hover:bg-slate-50/60 transition-colors group"
                    >
                      <td className="py-3.5 px-4 font-medium text-slate-900">
                        <Link
                          href={`/heartbeats/${hb.id}`}
                          className="hover:text-blue-600 transition-colors flex items-center gap-1.5 font-bold"
                        >
                          <Radio className="w-3.5 h-3.5 text-purple-500 group-hover:text-blue-600 shrink-0" />
                          <span>{hb.name}</span>
                        </Link>
                        {hb.description && (
                          <p className="text-[11px] text-slate-400 mt-0.5 truncate max-w-xs font-normal">
                            {hb.description}
                          </p>
                        )}
                      </td>
                      <td className="py-3.5 px-4">
                        <Link
                          href={`/clients/${hb.client_id}`}
                          className="text-slate-600 hover:text-slate-900 font-medium hover:underline flex items-center gap-1"
                        >
                          <Building className="w-3 h-3 text-slate-400 shrink-0" />
                          <span className="truncate max-w-[140px]">{hb.client_name}</span>
                        </Link>
                        {hb.asset_name && (
                          <div className="text-[11px] text-slate-400 mt-0.5 truncate max-w-[140px]">
                            {hb.asset_name}
                          </div>
                        )}
                      </td>
                      <td className="py-3.5 px-4 font-mono text-slate-600" dir="ltr">
                        {formatInterval(hb.expected_interval_seconds)}
                      </td>
                      <td className="py-3.5 px-4 font-mono text-slate-500" dir="ltr">
                        {formatInterval(hb.grace_period_seconds)}
                      </td>
                      <td className="py-3.5 px-4">
                        <div className="flex items-center gap-1.5 flex-wrap">
                          <Badge variant={hb.status} />
                          {hb.status === "missed" && (
                            <span className="text-[10px] bg-purple-50 text-purple-700 border border-purple-200 px-1.5 py-0.2 rounded font-medium">
                              {isRTL ? "جاب پس‌زمینه" : "Cron"}
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="py-3.5 px-4 text-slate-500">
                        {hb.last_received_at ? (
                          <span title={new Date(hb.last_received_at).toLocaleString()}>
                            {new Date(hb.last_received_at).toLocaleTimeString([], {
                              hour: "2-digit",
                              minute: "2-digit",
                              month: "short",
                              day: "numeric",
                            })}
                          </span>
                        ) : (
                          <span className="text-slate-400 italic">{t.neverReceived}</span>
                        )}
                      </td>
                      <td className="py-3.5 px-4 text-slate-500">
                        {hb.next_expected_at ? (
                          <span title={new Date(hb.next_expected_at).toLocaleString()}>
                            {new Date(hb.next_expected_at).toLocaleTimeString([], {
                              hour: "2-digit",
                              minute: "2-digit",
                              month: "short",
                              day: "numeric",
                            })}
                          </span>
                        ) : (
                          <span className="text-slate-400 italic">—</span>
                        )}
                      </td>
                      <td className="py-3.5 px-4 text-end">
                        <div className="flex items-center justify-end gap-1">
                          {canManage && (
                            <button
                              onClick={() => handleTogglePause(hb.id, hb.status)}
                              title={hb.status === "paused" ? t.resumeHeartbeat : t.pauseHeartbeat}
                              className={`p-1.5 rounded-lg border transition-colors ${
                                hb.status === "paused"
                                  ? "bg-emerald-50 text-emerald-600 border-emerald-200 hover:bg-emerald-100"
                                  : "bg-slate-50 text-slate-500 border-slate-200 hover:bg-slate-100"
                              }`}
                            >
                              {hb.status === "paused" ? (
                                <Play className="w-3.5 h-3.5" />
                              ) : (
                                <Pause className="w-3.5 h-3.5" />
                              )}
                            </button>
                          )}
                          <Link href={`/heartbeats/${hb.id}`}>
                            <button className="p-1.5 rounded-lg bg-slate-50 text-slate-600 border border-slate-200 hover:bg-slate-100 transition-colors">
                              {isRTL ? <ArrowLeft className="w-3.5 h-3.5" /> : <ArrowRight className="w-3.5 h-3.5" />}
                            </button>
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
              {heartbeats.map((hb) => (
                <div key={hb.id} className="p-4 space-y-3">
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0 flex-1">
                      <Link
                        href={`/heartbeats/${hb.id}`}
                        className="font-bold text-slate-900 text-sm hover:text-blue-600 flex items-center gap-1.5"
                      >
                        <Radio className="w-3.5 h-3.5 text-purple-500 shrink-0" />
                        <span className="truncate">{hb.name}</span>
                      </Link>
                      {hb.description && (
                        <p className="text-xs text-slate-500 mt-0.5 line-clamp-1">{hb.description}</p>
                      )}
                    </div>
                    <div className="flex items-center gap-1 shrink-0">
                      <Badge variant={hb.status} />
                    </div>
                  </div>

                  {/* Distinct banner for missed heartbeat on mobile */}
                  {hb.status === "missed" && (
                    <div className="flex items-center gap-1.5 p-2 bg-purple-50 border border-purple-200 rounded-lg text-[11px] text-purple-900">
                      <AlertTriangle className="w-3.5 h-3.5 text-purple-600 shrink-0" />
                      <span>{t.heartbeatMissedNotice}</span>
                    </div>
                  )}

                  <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-slate-600 pt-1">
                    <Link
                      href={`/clients/${hb.client_id}`}
                      className="flex items-center gap-1 text-slate-700 hover:text-blue-600 font-medium"
                    >
                      <Building className="w-3.5 h-3.5 text-slate-400 shrink-0" />
                      <span className="truncate max-w-[160px]">{hb.client_name}</span>
                    </Link>

                    <div className="flex items-center gap-2 font-mono text-[11px] text-slate-600" dir="ltr">
                      <span className="bg-slate-100 px-1.5 py-0.5 rounded">
                        {formatInterval(hb.expected_interval_seconds)}
                      </span>
                      <span className="text-slate-400">
                        +{formatInterval(hb.grace_period_seconds)}
                      </span>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-2 text-[11px] text-slate-500 bg-slate-50/70 p-2.5 rounded-xl border border-slate-100">
                    <div>
                      <span className="text-slate-400 block text-[10px] uppercase font-semibold">{t.lastReceived}</span>
                      <span className="font-medium text-slate-700 mt-0.5 block truncate">
                        {hb.last_received_at
                          ? new Date(hb.last_received_at).toLocaleTimeString([], {
                              hour: "2-digit",
                              minute: "2-digit",
                              month: "short",
                              day: "numeric",
                            })
                          : t.neverReceived}
                      </span>
                    </div>

                    <div>
                      <span className="text-slate-400 block text-[10px] uppercase font-semibold">{t.nextExpected}</span>
                      <span className="font-medium text-slate-700 mt-0.5 block truncate">
                        {hb.next_expected_at
                          ? new Date(hb.next_expected_at).toLocaleTimeString([], {
                              hour: "2-digit",
                              minute: "2-digit",
                              month: "short",
                              day: "numeric",
                            })
                          : "—"}
                      </span>
                    </div>
                  </div>

                  <div className="flex items-center justify-between pt-2 border-t border-slate-50 gap-2">
                    {canManage ? (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleTogglePause(hb.id, hb.status)}
                        icon={
                          hb.status === "paused" ? (
                            <Play className="w-3.5 h-3.5 text-emerald-600" />
                          ) : (
                            <Pause className="w-3.5 h-3.5 text-slate-600" />
                          )
                        }
                      >
                        {hb.status === "paused" ? t.resumeHeartbeat : t.pauseHeartbeat}
                      </Button>
                    ) : <div />}
                    <Link href={`/heartbeats/${hb.id}`}>
                      <Button size="sm" variant="secondary">
                        {t.viewDetails}
                      </Button>
                    </Link>
                  </div>
                </div>
              ))}
            </div>

            <div className="p-3 bg-slate-50 border-t border-slate-200 flex justify-between items-center text-xs text-slate-500">
              <span>
                {t.showingCount
                  .replace("{count}", String(heartbeats.length))
                  .replace("{total}", String(total))}
              </span>
            </div>
          </div>
        )}
      </div>
    </AppLayout>
  );
}
