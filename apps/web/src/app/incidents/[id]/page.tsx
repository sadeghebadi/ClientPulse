"use client";

import React, { useEffect, useState, useCallback, use } from "react";
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
  Globe,
  Radio,
  UserCheck,
  Calendar,
  Activity,
  ShieldAlert,
  Server,
  ExternalLink,
  X,
} from "lucide-react";
import { PageHeader } from "@/components/ui/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { ErrorState } from "@/components/ui/ErrorState";
import { useToast } from "@/components/ui/Toast";

interface CheckResultDetail {
  id: string;
  result: string;
  http_status_code?: number;
  error_category?: string;
  error_message?: string;
  latency_ms: number;
  resolved_ip?: string;
  tls_version?: string;
  tls_days_remaining?: number;
  completed_at: string;
}

interface IncidentDetail {
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
  acknowledged_by_email?: string;
  resolved_by?: string;
  resolved_by_name?: string;
  resolved_by_email?: string;
  resolution_type?: "automatic_recovery" | "manual" | "source_monitor_deleted" | string;
  resolution_note?: string;
  heartbeat_archived_at?: string;
  duration_seconds?: number;
  summary?: string;
  created_at: string;
  first_failure_result?: CheckResultDetail;
  recovery_result?: CheckResultDetail;
}

interface EventItem {
  id: string;
  event_type: string;
  idempotency_key: string;
  payload: Record<string, any>;
  status: string;
  created_at: string;
}

export default function IncidentDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const resolvedParams = use(params);
  const id = resolvedParams.id;

  const { user, activeOrg, role } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const [incident, setIncident] = useState<IncidentDetail | null>(null);
  const [events, setEvents] = useState<EventItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<any>(null);
  const [acking, setAcking] = useState(false);

  // Manual resolution modal state
  const [showResolveModal, setShowResolveModal] = useState(false);
  const [resolutionNote, setResolutionNote] = useState("");
  const [resolvingLoading, setResolvingLoading] = useState(false);

  const canAck = role === "owner" || role === "admin" || role === "member";

  const fetchIncident = useCallback(() => {
    if (!user || !activeOrg) return;
    setLoading(true);
    setError(null);

    apiFetch<{ incident: IncidentDetail; events: EventItem[] }>(`/api/v1/incidents/${id}`)
      .then((data) => {
        setIncident(data.incident);
        setEvents(data.events || []);
      })
      .catch((err) => {
        setError(err);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [user, activeOrg, id]);

  useEffect(() => {
    if (!user || !activeOrg) return;
    let isMounted = true;

    apiFetch<{ incident: IncidentDetail; events: EventItem[] }>(`/api/v1/incidents/${id}`)
      .then((data) => {
        if (!isMounted) return;
        setIncident(data.incident);
        setEvents(data.events || []);
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
  }, [user, activeOrg, id]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && showResolveModal && !resolvingLoading) {
        setShowResolveModal(false);
        setResolutionNote("");
      }
    };
    if (showResolveModal) {
      window.addEventListener("keydown", handleKeyDown);
    }
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [showResolveModal, resolvingLoading]);

  const handleAcknowledge = async () => {
    setAcking(true);
    try {
      await apiFetch(`/api/v1/incidents/${id}/acknowledge`, { method: "POST" });
      toast.success(isRTL ? "حادثه با موفقیت تأیید شد" : "Incident acknowledged");
      fetchIncident();
    } catch (err: any) {
      toast.error(err.message || "Failed to acknowledge incident");
    } finally {
      setAcking(false);
    }
  };

  const handleResolve = async () => {
    setResolvingLoading(true);
    try {
      await apiFetch(`/api/v1/incidents/${id}/resolve`, {
        method: "POST",
        body: JSON.stringify({ note: resolutionNote.trim() || undefined }),
      });
      toast.success(isRTL ? "حادثه با موفقیت رفع شد" : "Incident manually resolved");
      setShowResolveModal(false);
      setResolutionNote("");
      fetchIncident();
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

  const isSourceDeleted =
    incident?.resolution_type === "source_monitor_deleted" || !!incident?.heartbeat_archived_at;

  return (
    <AppLayout>
      <div className="space-y-6 max-w-5xl mx-auto">
        <PageHeader
          title={isRTL ? `حادثه: ${incident?.target_name || id}` : `Incident: ${incident?.target_name || id}`}
          description={incident?.summary}
          breadcrumbs={[
            { label: t.dashboard, href: "/" },
            { label: t.incidents, href: "/incidents" },
            { label: incident?.target_name || id },
          ]}
          actions={
            incident && incident.status === "open" && canAck ? (
              <div className="flex items-center gap-2">
                {!incident.acknowledged_at && (
                  <Button
                    icon={<UserCheck className="w-4 h-4" />}
                    loading={acking}
                    onClick={handleAcknowledge}
                  >
                    {t.acknowledge}
                  </Button>
                )}
                <Button
                  variant="outline"
                  icon={<CheckCircle2 className="w-4 h-4 text-emerald-600" />}
                  onClick={() => {
                    setShowResolveModal(true);
                    setResolutionNote("");
                  }}
                >
                  {t.resolveIncident}
                </Button>
              </div>
            ) : null
          }
        />

        {loading ? (
          <div className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm">
            <TableSkeleton rows={4} cols={4} />
          </div>
        ) : error || !incident ? (
          <ErrorState
            error={error}
            onRetry={fetchIncident}
          />
        ) : (
          <div className="space-y-6">
            {/* Deleted Monitor Informative Banner */}
            {isSourceDeleted && (
              <div className="bg-amber-50/90 border border-amber-200/90 rounded-xl p-4 flex items-start gap-3 text-xs">
                <AlertTriangle className="w-4 h-4 text-amber-600 shrink-0 mt-0.5" />
                <div className="space-y-1">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-bold text-amber-900">{t.sourceDeletedNotice}</span>
                    <Badge variant="monitor_deleted" className="text-[10px] py-0 px-1.5" />
                  </div>
                  <p className="text-amber-700 leading-relaxed">
                    {t.deletedMonitorNotice}
                  </p>
                </div>
              </div>
            )}

            {/* Status and Summary Banner */}
            <div className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm">
              <div className="flex items-center justify-between pb-4 border-b border-slate-100 mb-6 flex-wrap gap-3">
                <div className="flex items-center gap-3">
                  <div
                    className={`w-10 h-10 rounded-xl flex items-center justify-center shrink-0 ${
                      incident.status === "open"
                        ? "bg-rose-100 text-rose-600"
                        : "bg-emerald-100 text-emerald-600"
                    }`}
                  >
                    {incident.status === "open" ? (
                      <AlertTriangle className="w-5 h-5" />
                    ) : (
                      <CheckCircle2 className="w-5 h-5" />
                    )}
                  </div>
                  <div>
                    <h2 className="text-sm font-bold text-slate-900">{incident.target_name}</h2>
                    <p className="text-xs text-slate-400">
                      ID: <span className="font-mono text-slate-600">{incident.id}</span>
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2 flex-wrap">
                  <Badge variant={incident.status} />
                  {incident.status === "open" && incident.acknowledged_at && (
                    <Badge variant="acknowledged" label={t.statusAcknowledged} />
                  )}
                  {incident.status === "resolved" && (
                    <Badge variant={incident.resolution_type || "legacy_resolution"} />
                  )}
                </div>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4 sm:gap-6 text-xs">
                <div>
                  <span className="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.client}
                  </span>
                  <Link
                    href={`/clients/${incident.client_id}`}
                    className="font-medium text-slate-900 hover:text-blue-600 hover:underline flex items-center gap-1"
                  >
                    <span>{incident.client_name}</span>
                    <ExternalLink className="w-3 h-3 text-slate-400" />
                  </Link>
                </div>

                <div>
                  <span className="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.targetType}
                  </span>
                  <div className="flex items-center gap-1 font-medium text-slate-700 flex-wrap">
                    {incident.target_type === "monitor" ? (
                      <>
                        <Globe className="w-3.5 h-3.5 text-blue-500" />
                        <span>{t.websiteMonitor}</span>
                      </>
                    ) : (
                      <>
                        <Radio className="w-3.5 h-3.5 text-purple-500" />
                        {isSourceDeleted ? (
                          <span className="text-slate-600">{t.cronHeartbeat}</span>
                        ) : (
                          <Link
                            href={`/heartbeats/${incident.heartbeat_monitor_id}`}
                            className="hover:text-purple-600 hover:underline"
                          >
                            {t.cronHeartbeat}
                          </Link>
                        )}
                        {isSourceDeleted && (
                          <Badge variant="monitor_deleted" className="text-[10px] py-0 px-1.5" />
                        )}
                      </>
                    )}
                  </div>
                </div>

                <div>
                  <span className="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.failureCategory}
                  </span>
                  <span className="font-mono text-slate-800 font-semibold bg-slate-100 px-2 py-0.5 rounded">
                    {incident.failure_category}
                  </span>
                </div>

                <div>
                  <span className="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.duration}
                  </span>
                  <span className="font-mono text-slate-800 font-semibold">
                    {formatDuration(incident.duration_seconds)}
                  </span>
                </div>

                <div>
                  <span className="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.firstFailure}
                  </span>
                  <span className="text-slate-700 font-medium">
                    {new Date(incident.first_failure_at).toLocaleString()}
                  </span>
                </div>

                <div>
                  <span className="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.confirmedAt}
                  </span>
                  <span className="text-slate-700 font-medium">
                    {new Date(incident.confirmed_at).toLocaleString()}
                  </span>
                </div>

                <div>
                  <span className="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.acknowledgedBy}
                  </span>
                  {incident.acknowledged_at ? (
                    <div className="text-slate-700 font-medium">
                      <span>{incident.acknowledged_by_name || incident.acknowledged_by_email}</span>
                      <div className="text-[10px] text-slate-400">
                        {new Date(incident.acknowledged_at).toLocaleString()}
                      </div>
                    </div>
                  ) : (
                    <span className="text-rose-500 font-medium">{isRTL ? "هنوز تأیید نشده است" : "Unacknowledged"}</span>
                  )}
                </div>

                <div>
                  <span className="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.resolvedAt}
                  </span>
                  {incident.resolved_at ? (
                    <div className="text-slate-700 font-medium">
                      <span>
                        {incident.resolved_by_name || incident.resolved_by_email
                          ? `${t.resolvedBy} ${incident.resolved_by_name || incident.resolved_by_email}`
                          : incident.resolution_type === "automatic_recovery"
                          ? t.automaticRecovery
                          : incident.resolution_type === "source_monitor_deleted"
                          ? t.sourceMonitorDeleted
                          : incident.resolution_type === "manual"
                          ? t.manualResolution
                          : t.legacyResolutionReasonNotRecorded}
                      </span>
                      <div className="text-[10px] text-slate-400">
                        {new Date(incident.resolved_at).toLocaleString()}
                      </div>
                    </div>
                  ) : (
                    <span className="text-slate-400 italic">—</span>
                  )}
                </div>

                {/* Optional Resolution Note Display */}
                {incident.resolution_note && (
                  <div className="col-span-2 md:col-span-4 p-3.5 bg-slate-50 border border-slate-200/80 rounded-xl space-y-1">
                    <span className="block text-[11px] font-semibold text-slate-500 uppercase tracking-wider">
                      {t.resolutionNote}
                    </span>
                    <p className="text-slate-800 text-xs leading-relaxed whitespace-pre-wrap">
                      {incident.resolution_note}
                    </p>
                  </div>
                )}

                {/* Legacy Resolution Display */}
                {incident.status === "resolved" && !incident.resolution_type && (
                  <div className="col-span-2 md:col-span-4 p-3.5 bg-slate-50 border border-slate-200/80 rounded-xl space-y-1">
                    <span className="block text-[11px] font-semibold text-slate-500 uppercase tracking-wider">
                      {t.resolutionType}
                    </span>
                    <p className="text-slate-700 text-xs italic">
                      {t.legacyResolutionReasonNotRecorded}
                    </p>
                  </div>
                )}
              </div>
            </div>

            {/* Check Results Diagnostic Panels */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {/* First Failure Check Result */}
              <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm space-y-3">
                <h3 className="text-xs font-bold text-slate-900 flex items-center gap-1.5 border-b border-slate-100 pb-2">
                  <AlertTriangle className="w-3.5 h-3.5 text-rose-600" />
                  {t.firstFailureResult}
                </h3>
                {incident.first_failure_result ? (
                  <div className="space-y-2 text-xs font-mono">
                    <div className="flex justify-between py-1 border-b border-slate-50">
                      <span className="text-slate-500 font-sans">{t.statusCode}:</span>
                      <span className="font-bold text-rose-600">
                        {incident.first_failure_result.http_status_code || "N/A"}
                      </span>
                    </div>
                    <div className="flex justify-between py-1 border-b border-slate-50">
                      <span className="text-slate-500 font-sans">{t.latency}:</span>
                      <span className="text-slate-800">{incident.first_failure_result.latency_ms}ms</span>
                    </div>
                    {incident.first_failure_result.resolved_ip && (
                      <div className="flex justify-between py-1 border-b border-slate-50">
                        <span className="text-slate-500 font-sans">Resolved IP:</span>
                        <span className="text-slate-800">{incident.first_failure_result.resolved_ip}</span>
                      </div>
                    )}
                    {incident.first_failure_result.error_message && (
                      <div className="pt-1">
                        <span className="text-slate-500 font-sans block mb-1">Diagnostic Message:</span>
                        <div className="p-2 bg-rose-50 border border-rose-200 rounded text-[11px] text-rose-800 break-all font-mono">
                          {incident.first_failure_result.error_message}
                        </div>
                      </div>
                    )}
                  </div>
                ) : (
                  <p className="text-xs text-slate-400 italic">
                    {isRTL ? "اطلاعات تشخیصی مستقیمی ثبت نشده است" : "Diagnostic check details not attached"}
                  </p>
                )}
              </div>

              {/* Recovery Check Result */}
              <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm space-y-3">
                <h3 className="text-xs font-bold text-slate-900 flex items-center gap-1.5 border-b border-slate-100 pb-2">
                  <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600" />
                  {t.recoveryResult}
                </h3>
                {incident.recovery_result ? (
                  <div className="space-y-2 text-xs font-mono">
                    <div className="flex justify-between py-1 border-b border-slate-50">
                      <span className="text-slate-500 font-sans">{t.statusCode}:</span>
                      <span className="font-bold text-emerald-600">
                        {incident.recovery_result.http_status_code || "200 OK"}
                      </span>
                    </div>
                    <div className="flex justify-between py-1 border-b border-slate-50">
                      <span className="text-slate-500 font-sans">{t.latency}:</span>
                      <span className="text-slate-800">{incident.recovery_result.latency_ms}ms</span>
                    </div>
                    {incident.recovery_result.resolved_ip && (
                      <div className="flex justify-between py-1 border-b border-slate-50">
                        <span className="text-slate-500 font-sans">Resolved IP:</span>
                        <span className="text-slate-800">{incident.recovery_result.resolved_ip}</span>
                      </div>
                    )}
                    {incident.recovery_result.tls_version && (
                      <div className="flex justify-between py-1 border-b border-slate-50">
                        <span className="text-slate-500 font-sans">TLS Version:</span>
                        <span className="text-slate-800">{incident.recovery_result.tls_version}</span>
                      </div>
                    )}
                  </div>
                ) : (
                  <p className="text-xs text-slate-500 italic">
                    {incident.status === "open"
                      ? (isRTL ? "حادثه هنوز باز است و ریکاور نشده است" : "Incident is currently open; awaiting recovery check")
                      : incident.resolution_type === "source_monitor_deleted"
                      ? (isRTL ? "بسته شده به دلیل حذف مانیتور منبع (بدون تست ریکاور)" : "Resolved via source monitor deletion (no recovery check required)")
                      : incident.resolution_type === "manual"
                      ? (isRTL ? "بسته شده با رفع دستی اپراتور (بدون تست ریکاور)" : "Resolved manually by operator (no recovery check required)")
                      : incident.resolution_type === "automatic_recovery"
                      ? (isRTL ? "اطلاعات ریکاور مستقلی ضمیمه نشده است" : "Recovery diagnostics not recorded")
                      : t.legacyResolutionReasonNotRecorded}
                  </p>
                )}
              </div>
            </div>

            {/* Event Timeline */}
            <div className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm space-y-4">
              <h3 className="text-xs font-bold text-slate-900 flex items-center gap-1.5 border-b border-slate-100 pb-3">
                <Clock className="w-4 h-4 text-slate-600" />
                {t.incidentTimeline}
              </h3>

              {events.length === 0 ? (
                <p className="text-xs text-slate-500 italic">
                  {incident.status === "resolved" && !incident.resolution_type
                    ? t.legacyResolutionReasonNotRecorded
                    : (isRTL ? "هیچ رویدادی برای این حادثه ثبت نشده است." : "No notification events logged for this incident.")}
                </p>
              ) : (
                <div className="space-y-4 relative before:absolute before:inset-0 before:left-3.5 rtl:before:right-3.5 rtl:before:left-auto before:w-0.5 before:bg-slate-100">
                  {events.map((ev) => (
                    <div key={ev.id} className="relative flex items-start gap-4">
                      <div className="w-7 h-7 rounded-full bg-white border-2 border-slate-300 flex items-center justify-center shrink-0 z-10">
                        {ev.event_type.includes("opened") || ev.event_type.includes("missed") ? (
                          <AlertTriangle className="w-3.5 h-3.5 text-rose-600" />
                        ) : ev.event_type.includes("acknowledged") ? (
                          <UserCheck className="w-3.5 h-3.5 text-blue-600" />
                        ) : (
                          <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600" />
                        )}
                      </div>
                      <div className="bg-slate-50 rounded-xl p-3 border border-slate-100 w-full text-xs space-y-1">
                        <div className="flex justify-between items-center">
                          <span className="font-bold text-slate-800 font-mono">
                            {ev.event_type}
                          </span>
                          <span className="text-[11px] text-slate-400">
                            {new Date(ev.created_at).toLocaleString()}
                          </span>
                        </div>
                        <div className="text-[11px] text-slate-500 font-mono bg-white p-2 rounded border border-slate-200/60 overflow-x-auto text-left" dir="ltr">
                          {JSON.stringify(ev.payload, null, 2)}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Manual Resolution Modal */}
      {showResolveModal && incident && (
        <div
          role="dialog"
          aria-modal="true"
          className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-xs animate-in fade-in duration-150"
          onClick={(e) => {
            if (e.target === e.currentTarget && !resolvingLoading) {
              setShowResolveModal(false);
              setResolutionNote("");
            }
          }}
        >
          <div
            className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-xl overflow-hidden p-6 relative transition-all"
            dir={isRTL ? "rtl" : "ltr"}
          >
            <button
              onClick={() => {
                setShowResolveModal(false);
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
                <p className="text-xs text-slate-500 font-mono mt-0.5">{incident.target_name}</p>
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
                className="w-full text-xs p-2.5 rounded-xl border border-slate-200 bg-slate-50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all text-slate-800 resize-none"
              />
            </div>

            <div className="flex flex-col-reverse sm:flex-row items-stretch sm:items-center justify-end gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  setShowResolveModal(false);
                  setResolutionNote("");
                }}
                disabled={resolvingLoading}
                className="w-full sm:w-auto"
              >
                {t.cancel}
              </Button>
              <Button
                variant="primary"
                size="sm"
                loading={resolvingLoading}
                onClick={handleResolve}
                className="w-full sm:w-auto"
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
