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
  const { user, activeOrg } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const [heartbeats, setHeartbeats] = useState<HeartbeatItem[]>([]);
  const [total, setTotal] = useState(0);
  const [statusFilter, setStatusFilter] = useState("all");
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<any>(null);

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
      toast.success(
        isPaused
          ? (isRTL ? "هارت‌بیت فعال شد" : "Heartbeat resumed")
          : (isRTL ? "هارت‌بیت متوقف شد" : "Heartbeat paused")
      );
      fetchHeartbeats();
    } catch (err: any) {
      toast.error(err.message || (isRTL ? "خطا در تغییر وضعیت" : "Failed to toggle status"));
    }
  };

  const formatInterval = (sec: number) => {
    if (sec >= 86400) return `${Math.round(sec / 86400)} ${isRTL ? "روز" : "days"}`;
    if (sec >= 3600) return `${Math.round(sec / 3600)} ${isRTL ? "ساعت" : "hours"}`;
    if (sec >= 60) return `${Math.round(sec / 60)} ${isRTL ? "دقیقه" : "min"}`;
    return `${sec}s`;
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
            <Link href="/heartbeats/new">
              <Button icon={<Plus className="w-4 h-4" />}>
                {t.addHeartbeat}
              </Button>
            </Link>
          }
        />

        {/* Semantic Limitation Alert */}
        <div className="flex items-start gap-3 p-4 bg-sky-50 border border-sky-200 rounded-xl text-xs text-sky-900">
          <Info className="w-4 h-4 text-sky-600 mt-0.5 shrink-0" />
          <p className="leading-relaxed">{t.semanticNotice}</p>
        </div>

        {/* Filters bar */}
        <div className="bg-white border border-slate-200 rounded-xl p-4 shadow-sm flex flex-col md:flex-row gap-3 justify-between items-stretch md:items-center">
          <div className="flex flex-wrap items-center gap-2">
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
                className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
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
            <input
              type="text"
              placeholder={t.search}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="text-xs px-3 py-1.5 rounded-lg border border-slate-200 focus:outline-none focus:ring-2 focus:ring-slate-900/10 w-full md:w-56"
            />
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

        {/* Heartbeats Table or States */}
        {loading ? (
          <div className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm">
            <TableSkeleton rows={5} cols={6} />
          </div>
        ) : error ? (
          <ErrorState
            error={error}
            onRetry={fetchHeartbeats}
          />
        ) : heartbeats.length === 0 ? (
          <EmptyState
            icon={<Radio className="w-10 h-10 text-slate-400" />}
            title={t.noHeartbeatsFound}
            description={t.createHeartbeatDesc}
            actionLabel={t.addHeartbeat}
            onAction={() => router.push("/heartbeats/new")}
          />
        ) : (
          <div className="bg-white border border-slate-200 rounded-xl shadow-sm overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-left rtl:text-right border-collapse text-xs">
                <thead>
                  <tr className="bg-slate-50/75 border-b border-slate-200 text-slate-500 font-semibold uppercase tracking-wider text-[11px]">
                    <th className="py-3 px-4">{t.heartbeatName}</th>
                    <th className="py-3 px-4">{t.client}</th>
                    <th className="py-3 px-4">{t.expectedInterval}</th>
                    <th className="py-3 px-4">{t.gracePeriod}</th>
                    <th className="py-3 px-4">{t.status}</th>
                    <th className="py-3 px-4">{t.lastReceived}</th>
                    <th className="py-3 px-4">{t.nextExpected}</th>
                    <th className="py-3 px-4 text-right rtl:text-left">{t.actions}</th>
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
                          className="hover:text-blue-600 transition-colors flex items-center gap-1.5"
                        >
                          <Radio className="w-3.5 h-3.5 text-slate-400 group-hover:text-blue-600" />
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
                          className="text-slate-600 hover:text-slate-900 font-medium hover:underline"
                        >
                          {hb.client_name}
                        </Link>
                        {hb.asset_name && (
                          <div className="text-[11px] text-slate-400 mt-0.5">
                            {hb.asset_name}
                          </div>
                        )}
                      </td>
                      <td className="py-3.5 px-4 font-mono text-slate-600">
                        {formatInterval(hb.expected_interval_seconds)}
                      </td>
                      <td className="py-3.5 px-4 font-mono text-slate-500">
                        {formatInterval(hb.grace_period_seconds)}
                      </td>
                      <td className="py-3.5 px-4">
                        <Badge variant={hb.status} />
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
                          <span className="text-slate-400 italic">—</span>
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
                      <td className="py-3.5 px-4 text-right rtl:text-left">
                        <div className="flex items-center justify-end rtl:justify-start gap-1">
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
            <div className="p-3 bg-slate-50 border-t border-slate-200 flex justify-between items-center text-xs text-slate-500">
              <span>{isRTL ? `نمایش ${heartbeats.length} از ${total}` : `Showing ${heartbeats.length} of ${total}`}</span>
            </div>
          </div>
        )}
      </div>
    </AppLayout>
  );
}
