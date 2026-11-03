"use client";

import React, { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import {
  AlertTriangle,
  ArrowRight,
  ArrowLeft,
  CheckCircle2,
  Clock,
  RotateCw,
  Globe,
  Radio,
  UserCheck,
  X,
} from "lucide-react";
import { PageHeader } from "@/components/ui/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { useToast } from "@/components/ui/Toast";

interface IncidentItem {
  id: string;
  target_type: "monitor" | "heartbeat";
  target_name: string;
  client_id: string;
  client_name: string;
  asset_id?: string;
  asset_name?: string;
  monitor_id?: string;
  heartbeat_monitor_id?: string;
  first_failure_at: string;
  confirmed_at: string;
  resolved_at?: string;
  status: "open" | "resolved";
  failure_category: string;
  acknowledged_at?: string;
  acknowledged_by?: string;
  acknowledged_by_name?: string;
  resolved_by?: string;
  resolved_by_name?: string;
  resolution_type?: "automatic_recovery" | "manual" | "source_monitor_deleted" | string;
  resolution_note?: string;
  heartbeat_archived_at?: string;
  duration_seconds?: number;
  summary?: string;
}

export default function IncidentsPage() {
  const { user, activeOrg, role } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const [incidents, setIncidents] = useState<IncidentItem[]>([]);
  const [total, setTotal] = useState(0);
  const [statusFilter, setStatusFilter] = useState("all");
  const [targetTypeFilter, setTargetTypeFilter] = useState("all");
  const [ackFilter, setAckFilter] = useState("all");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<any>(null);
  const [ackingID, setAckingID] = useState<string | null>(null);

  // Manual resolution modal state
  const [resolvingIncident, setResolvingIncident] = useState<IncidentItem | null>(null);
  const [resolutionNote, setResolutionNote] = useState("");
  const [resolvingLoading, setResolvingLoading] = useState(false);

  const canAck = role === "owner" || role === "admin" || role === "member";

  const fetchIncidents = useCallback(() => {
    if (!user || !activeOrg) return;
    setLoading(true);
    setError(null);

    const params = new URLSearchParams();
    if (statusFilter && statusFilter !== "all") params.set("status", statusFilter);
    if (targetTypeFilter && targetTypeFilter !== "all") params.set("target_type", targetTypeFilter);
    if (ackFilter && ackFilter !== "all") params.set("ack_state", ackFilter);

    apiFetch<{ incidents: IncidentItem[]; total: number }>(`/api/v1/incidents?${params.toString()}`)
      .then((data) => {
        setIncidents(data.incidents || []);
        setTotal(data.total || 0);
      })
      .catch((err) => {
        setError(err);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [user, activeOrg, statusFilter, targetTypeFilter, ackFilter]);

  useEffect(() => {
    if (!user || !activeOrg) return;
    let isMounted = true;

    const params = new URLSearchParams();
    if (statusFilter && statusFilter !== "all") params.set("status", statusFilter);
    if (targetTypeFilter && targetTypeFilter !== "all") params.set("target_type", targetTypeFilter);
    if (ackFilter && ackFilter !== "all") params.set("ack_state", ackFilter);

    apiFetch<{ incidents: IncidentItem[]; total: number }>(`/api/v1/incidents?${params.toString()}`)
      .then((data) => {
        if (!isMounted) return;
        setIncidents(data.incidents || []);
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
  }, [user, activeOrg, statusFilter, targetTypeFilter, ackFilter]);

  const handleAcknowledge = async (id: string) => {
    setAckingID(id);
    try {
      await apiFetch(`/api/v1/incidents/${id}/acknowledge`, { method: "POST" });
      toast.success(isRTL ? "حادثه با موفقیت تأیید شد" : "Incident acknowledged");
      fetchIncidents();
    } catch (err: any) {
      toast.error(err.message || "Failed to acknowledge incident");
    } finally {
      setAckingID(null);
    }
  };

  const handleResolve = async () => {
    if (!resolvingIncident) return;
    setResolvingLoading(true);
    try {
      await apiFetch(`/api/v1/incidents/${resolvingIncident.id}/resolve`, {
        method: "POST",
        body: JSON.stringify({ note: resolutionNote.trim() || undefined }),
      });
      toast.success(isRTL ? "حادثه با موفقیت رفع شد" : "Incident manually resolved");
      setResolvingIncident(null);
      setResolutionNote("");
      fetchIncidents();
    } catch (err: any) {
      toast.error(err.message || "Failed to resolve incident");
    } finally {
      setResolvingLoading(false);
    }
  };

  const formatDuration = (sec?: number) => {
    if (sec === undefined || sec === null) return "—";
    if (sec >= 3600) return `${Math.floor(sec / 3600)}h ${Math.floor((sec % 3600) / 60)}m`;
    if (sec >= 60) return `${Math.floor(sec / 60)}m ${sec % 60}s`;
    return `${sec}s`;
  };

  return (
    <AppLayout>
      <div className="space-y-6">
        <PageHeader
          title={t.incidents}
          description={isRTL ? "مدیریت رخدادهای قطعی وب‌سایت‌ها و عدم دریافت هارت‌بیت‌ها با شفافیت کامل" : "Operational outages and missed heartbeats detected across client properties"}
          breadcrumbs={[
            { label: t.dashboard, href: "/" },
            { label: t.incidents },
          ]}
        />

        {/* Filters */}
        <div className="bg-white border border-slate-200 rounded-xl p-4 shadow-sm flex flex-col md:flex-row gap-3 justify-between items-stretch md:items-center">
          <div className="flex flex-wrap items-center gap-2">
            {[
              { id: "all", label: t.all },
              { id: "open", label: t.statusOpen },
              { id: "resolved", label: t.statusResolved },
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

            <span className="text-slate-300">|</span>

            {/* Target Filter */}
            <select
              value={targetTypeFilter}
              onChange={(e) => setTargetTypeFilter(e.target.value)}
              className="text-xs px-2.5 py-1.5 rounded-lg border border-slate-200 bg-white focus:outline-none"
            >
              <option value="all">{t.allTargets}</option>
              <option value="monitor">{t.websiteMonitor}</option>
              <option value="heartbeat">{t.cronHeartbeat}</option>
            </select>

            {/* Ack Filter */}
            <select
              value={ackFilter}
              onChange={(e) => setAckFilter(e.target.value)}
              className="text-xs px-2.5 py-1.5 rounded-lg border border-slate-200 bg-white focus:outline-none"
            >
              <option value="all">{isRTL ? "همه وضعیت‌های تأیید" : "All Ack States"}</option>
              <option value="unack">{t.unacknowledged}</option>
              <option value="ack">{t.statusAcknowledged}</option>
            </select>
          </div>

          <Button
            variant="secondary"
            size="sm"
            icon={<RotateCw className="w-3.5 h-3.5" />}
            onClick={fetchIncidents}
          >
            {t.refresh}
          </Button>
        </div>

        {/* Incidents Table or States */}
        {loading ? (
          <div className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm">
            <TableSkeleton rows={5} cols={6} />
          </div>
        ) : error ? (
          <ErrorState
            error={error}
            onRetry={fetchIncidents}
          />
        ) : incidents.length === 0 ? (
          <EmptyState
            icon={<CheckCircle2 className="w-10 h-10 text-emerald-500" />}
            title={isRTL ? "همه سیستم‌ها پایدار هستند" : "All systems operational"}
            description={t.noIncidentsFound}
          />
        ) : (
          <div className="bg-white border border-slate-200 rounded-xl shadow-sm overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-left rtl:text-right border-collapse text-xs">
                <thead>
                  <tr className="bg-slate-50/75 border-b border-slate-200 text-slate-500 font-semibold uppercase tracking-wider text-[11px]">
                    <th className="py-3 px-4">{t.status}</th>
                    <th className="py-3 px-4">{t.targetType}</th>
                    <th className="py-3 px-4">{t.client}</th>
                    <th className="py-3 px-4">{t.failureCategory}</th>
                    <th className="py-3 px-4">{t.firstFailure}</th>
                    <th className="py-3 px-4">{t.duration}</th>
                    <th className="py-3 px-4">{t.acknowledgedBy}</th>
                    <th className="py-3 px-4 text-right rtl:text-left">{t.actions}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {incidents.map((inc) => (
                    <tr
                      key={inc.id}
                      className="hover:bg-slate-50/60 transition-colors group"
                    >
                      <td className="py-3.5 px-4">
                        <div className="flex flex-col gap-1 items-start">
                          <Badge variant={inc.status} />
                          {inc.status === "open" && inc.acknowledged_at && (
                            <span className="inline-flex items-center gap-1 text-[10px] text-blue-700 bg-blue-50 border border-blue-200 px-1.5 py-0.5 rounded font-medium">
                              <UserCheck className="w-2.5 h-2.5" />
                              <span>{t.statusAcknowledged}</span>
                            </span>
                          )}
                          {inc.status === "resolved" && inc.resolution_type && (
                            <Badge
                              variant={inc.resolution_type}
                              className="text-[10px]"
                            />
                          )}
                        </div>
                      </td>
                      <td className="py-3.5 px-4 font-medium text-slate-900">
                        <div className="flex items-center gap-1.5 flex-wrap">
                          <Link
                            href={`/incidents/${inc.id}`}
                            className="hover:text-blue-600 transition-colors flex items-center gap-1.5 font-semibold"
                          >
                            {inc.target_type === "monitor" ? (
                              <Globe className="w-3.5 h-3.5 text-blue-500 shrink-0" />
                            ) : (
                              <Radio className="w-3.5 h-3.5 text-purple-500 shrink-0" />
                            )}
                            <span>{inc.target_name}</span>
                          </Link>
                          {(inc.heartbeat_archived_at || inc.resolution_type === "source_monitor_deleted") && (
                            <Badge variant="monitor_deleted" className="text-[10px] py-0 px-1.5" />
                          )}
                        </div>
                        {inc.summary && (
                          <p className="text-[11px] text-slate-400 mt-0.5 truncate max-w-xs font-normal">
                            {inc.summary}
                          </p>
                        )}
                      </td>
                      <td className="py-3.5 px-4">
                        <Link
                          href={`/clients/${inc.client_id}`}
                          className="text-slate-600 hover:text-slate-900 font-medium hover:underline"
                        >
                          {inc.client_name}
                        </Link>
                      </td>
                      <td className="py-3.5 px-4 font-mono text-slate-700 font-medium">
                        {inc.failure_category}
                      </td>
                      <td className="py-3.5 px-4 text-slate-500">
                        <span title={new Date(inc.first_failure_at).toLocaleString()}>
                          {new Date(inc.first_failure_at).toLocaleTimeString([], {
                            hour: "2-digit",
                            minute: "2-digit",
                            month: "short",
                            day: "numeric",
                          })}
                        </span>
                      </td>
                      <td className="py-3.5 px-4 text-slate-600 font-mono">
                        {formatDuration(inc.duration_seconds)}
                      </td>
                      <td className="py-3.5 px-4">
                        {inc.status === "open" ? (
                          inc.acknowledged_at ? (
                            <div className="flex flex-col gap-0.5">
                              <div className="flex items-center gap-1 text-[11px] text-emerald-700 bg-emerald-50 border border-emerald-200 px-2 py-0.5 rounded w-fit">
                                <UserCheck className="w-3 h-3 text-emerald-600 shrink-0" />
                                <span className="truncate max-w-[120px]">{inc.acknowledged_by_name || (isRTL ? "تأییدشده" : "Acked")}</span>
                              </div>
                              <span className="text-[10px] text-slate-400">
                                {new Date(inc.acknowledged_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
                              </span>
                            </div>
                          ) : (
                            <span className="text-[11px] text-rose-500 font-medium">
                              {isRTL ? "در انتظار بررسی" : "Unacknowledged"}
                            </span>
                          )
                        ) : (
                          <div className="flex flex-col gap-0.5 text-[11px]">
                            <span className="text-slate-700 font-medium truncate max-w-[140px]">
                              {inc.resolved_by_name ? `${t.resolvedBy} ${inc.resolved_by_name}` : (isRTL ? "رفع خودکار" : "Auto Recovery")}
                            </span>
                            {inc.resolved_at && (
                              <span className="text-[10px] text-slate-400">
                                {new Date(inc.resolved_at).toLocaleDateString([], { month: "short", day: "numeric" })}
                              </span>
                            )}
                          </div>
                        )}
                      </td>
                      <td className="py-3.5 px-4 text-right rtl:text-left">
                        <div className="flex items-center justify-end rtl:justify-start gap-1">
                          {inc.status === "open" && !inc.acknowledged_at && canAck && (
                            <Button
                              size="sm"
                              variant="secondary"
                              loading={ackingID === inc.id}
                              onClick={() => handleAcknowledge(inc.id)}
                            >
                              {t.acknowledge}
                            </Button>
                          )}
                          {inc.status === "open" && canAck && (
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => {
                                setResolvingIncident(inc);
                                setResolutionNote("");
                              }}
                            >
                              {t.resolveIncident}
                            </Button>
                          )}
                          <Link href={`/incidents/${inc.id}`}>
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
              <span>{isRTL ? `نمایش ${incidents.length} از ${total}` : `Showing ${incidents.length} of ${total}`}</span>
            </div>
          </div>
        )}
      </div>

      {/* Manual Resolution Modal */}
      {resolvingIncident && (
        <div
          role="dialog"
          aria-modal="true"
          className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm animate-in fade-in duration-150"
        >
          <div
            className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-xl overflow-hidden p-6 relative transition-all"
            dir={isRTL ? "rtl" : "ltr"}
          >
            <button
              onClick={() => {
                setResolvingIncident(null);
                setResolutionNote("");
              }}
              disabled={resolvingLoading}
              className="absolute top-4 end-4 p-1.5 text-slate-400 hover:text-slate-600 rounded-lg hover:bg-slate-100 transition-colors"
            >
              <X className="w-4 h-4" />
            </button>

            <div className="flex items-center gap-3 mb-4">
              <div className="w-10 h-10 rounded-xl bg-blue-50 border border-blue-200 flex items-center justify-center text-blue-600 shrink-0">
                <CheckCircle2 className="w-5 h-5" />
              </div>
              <div>
                <h3 className="text-sm font-bold text-slate-900">{t.confirmResolveTitle}</h3>
                <p className="text-xs text-slate-500 font-mono mt-0.5">{resolvingIncident.target_name}</p>
              </div>
            </div>

            <p className="text-xs text-slate-600 mb-4 leading-relaxed">
              {t.confirmResolveDesc}
            </p>

            <div className="space-y-1.5 mb-5">
              <label className="block text-xs font-semibold text-slate-700">
                {t.resolutionNote} <span className="text-slate-400 font-normal">({isRTL ? "اختیاری" : "Optional"})</span>
              </label>
              <textarea
                value={resolutionNote}
                onChange={(e) => setResolutionNote(e.target.value)}
                placeholder={t.resolutionNotePlaceholder}
                rows={3}
                className="w-full text-xs p-2.5 rounded-lg border border-slate-200 bg-slate-50/50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all text-slate-800 resize-none"
              />
            </div>

            <div className="flex items-center justify-end gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  setResolvingIncident(null);
                  setResolutionNote("");
                }}
                disabled={resolvingLoading}
              >
                {t.cancel}
              </Button>
              <Button
                variant="primary"
                size="sm"
                loading={resolvingLoading}
                onClick={handleResolve}
              >
                {t.resolveIncident}
              </Button>
            </div>
          </div>
        </div>
      )}
    </AppLayout>
  );
}
