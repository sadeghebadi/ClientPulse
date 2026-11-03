"use client";

import React, { useEffect, useState, useCallback, use } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import {
  Radio,
  Edit,
  Play,
  Pause,
  KeyRound,
  Trash2,
  Clock,
  ExternalLink,
  ShieldAlert,
  AlertTriangle,
  Info,
  Check,
  Copy,
  Terminal,
  ArrowRight,
  ArrowLeft,
  X,
} from "lucide-react";
import { PageHeader } from "@/components/ui/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { ConfirmModal } from "@/components/ui/ConfirmModal";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { ErrorState } from "@/components/ui/ErrorState";
import { useToast } from "@/components/ui/Toast";

interface HeartbeatDetail {
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

interface IncidentItem {
  id: string;
  status: "open" | "resolved";
  failure_category: string;
  first_failure_at: string;
  confirmed_at: string;
  resolved_at?: string;
  duration_seconds?: number;
}

export default function HeartbeatDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const resolvedParams = use(params);
  const id = resolvedParams.id;

  const router = useRouter();
  const { user, activeOrg, role } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const [heartbeat, setHeartbeat] = useState<HeartbeatDetail | null>(null);
  const [incidents, setIncidents] = useState<IncidentItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<any>(null);

  // Action states
  const [showRevokeModal, setShowRevokeModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showRotateModal, setShowRotateModal] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);

  // Rotated token modal state
  const [rotatedData, setRotatedData] = useState<{
    token: string;
    pingURL: string;
    bashSnippet: string;
    powerShellSnippet: string;
  } | null>(null);

  const [copiedToken, setCopiedToken] = useState(false);
  const [copiedBash, setCopiedBash] = useState(false);
  const [copiedPS, setCopiedPS] = useState(false);

  const canManage = role === "owner" || role === "admin" || role === "member";
  const canAdmin = role === "owner" || role === "admin";

  const fetchDetail = useCallback(() => {
    if (!user || !activeOrg) return;
    setLoading(true);
    setError(null);

    Promise.all([
      apiFetch<{ heartbeat: HeartbeatDetail }>(`/api/v1/heartbeats/${id}`),
      apiFetch<{ incidents: IncidentItem[] }>(`/api/v1/incidents?heartbeat_id=${id}&limit=10`),
    ])
      .then(([hbRes, incRes]) => {
        setHeartbeat(hbRes.heartbeat);
        setIncidents(incRes.incidents || []);
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

    Promise.all([
      apiFetch<{ heartbeat: HeartbeatDetail }>(`/api/v1/heartbeats/${id}`),
      apiFetch<{ incidents: IncidentItem[] }>(`/api/v1/incidents?heartbeat_id=${id}&limit=10`),
    ])
      .then(([hbRes, incRes]) => {
        if (!isMounted) return;
        setHeartbeat(hbRes.heartbeat);
        setIncidents(incRes.incidents || []);
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

  // Rotated modal Escape key listener
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && rotatedData) {
        setRotatedData(null);
      }
    };
    if (rotatedData) {
      document.body.style.overflow = "hidden";
      window.addEventListener("keydown", handleKeyDown);
    } else {
      document.body.style.overflow = "unset";
    }
    return () => {
      document.body.style.overflow = "unset";
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [rotatedData]);

  const handleTogglePause = async () => {
    if (!heartbeat) return;
    const isPaused = heartbeat.status === "paused";
    const endpoint = isPaused
      ? `/api/v1/heartbeats/${id}/resume`
      : `/api/v1/heartbeats/${id}/pause`;

    try {
      await apiFetch(endpoint, { method: "POST" });
      toast.success(isPaused ? t.heartbeatResumed : t.heartbeatPaused);
      fetchDetail();
    } catch (err: any) {
      toast.error(err.message || "Failed to toggle status");
    }
  };

  const handleRotate = async () => {
    setActionLoading(true);
    try {
      const resp = await apiFetch<{
        token: string;
        ping_url: string;
        bash_snippet: string;
        powershell_snippet: string;
      }>(`/api/v1/heartbeats/${id}/rotate-token`, { method: "POST" });

      setShowRotateModal(false);
      setRotatedData({
        token: resp.token,
        pingURL: resp.ping_url,
        bashSnippet: resp.bash_snippet,
        powerShellSnippet: resp.powershell_snippet,
      });
      toast.success(t.tokenRotatedSuccess);
      fetchDetail();
    } catch (err: any) {
      toast.error(err.message || "Failed to rotate token");
    } finally {
      setActionLoading(false);
    }
  };

  const handleRevoke = async () => {
    setActionLoading(true);
    try {
      await apiFetch(`/api/v1/heartbeats/${id}/revoke-token`, { method: "POST" });
      setShowRevokeModal(false);
      toast.success(t.tokenRevokedSuccess);
      fetchDetail();
    } catch (err: any) {
      toast.error(err.message || "Failed to revoke token");
    } finally {
      setActionLoading(false);
    }
  };

  const handleDelete = async () => {
    setActionLoading(true);
    try {
      await apiFetch(`/api/v1/heartbeats/${id}`, { method: "DELETE" });
      toast.success(t.heartbeatArchivedSuccess);
      router.push("/heartbeats");
    } catch (err: any) {
      toast.error(err.message || "Failed to archive heartbeat");
      setActionLoading(false);
    }
  };

  const copyToClipboard = (text: string, type: "token" | "bash" | "ps") => {
    navigator.clipboard.writeText(text);
    if (type === "token") {
      setCopiedToken(true);
      setTimeout(() => setCopiedToken(false), 2000);
    } else if (type === "bash") {
      setCopiedBash(true);
      setTimeout(() => setCopiedBash(false), 2000);
    } else {
      setCopiedPS(true);
      setTimeout(() => setCopiedPS(false), 2000);
    }
    toast.success(t.copiedToClipboard);
  };

  const formatInterval = (sec: number) => {
    if (sec >= 86400) return `${Math.round(sec / 86400)} ${t.daysUnit}`;
    if (sec >= 3600) return `${Math.round(sec / 3600)} ${t.hoursUnit}`;
    if (sec >= 60) return `${Math.round(sec / 60)} ${t.minutesUnit}`;
    return `${sec}${t.secondsUnit}`;
  };

  const formatDuration = (sec?: number) => {
    if (sec === undefined || sec === null) return "—";
    if (sec >= 3600) return `${Math.floor(sec / 3600)}h ${Math.floor((sec % 3600) / 60)}m`;
    if (sec >= 60) return `${Math.floor(sec / 60)}m ${sec % 60}s`;
    return `${sec}s`;
  };

  return (
    <AppLayout>
      <div className="space-y-6 max-w-5xl mx-auto">
        <PageHeader
          title={heartbeat?.name || t.heartbeat}
          description={heartbeat?.description}
          breadcrumbs={[
            { label: t.dashboard, href: "/dashboard" },
            { label: t.heartbeats, href: "/heartbeats" },
            { label: heartbeat?.name || id },
          ]}
          actions={
            <div className="flex flex-wrap items-center gap-2">
              {heartbeat && heartbeat.status !== "revoked" && canManage && (
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={handleTogglePause}
                  icon={heartbeat.status === "paused" ? <Play className="w-3.5 h-3.5 text-emerald-600" /> : <Pause className="w-3.5 h-3.5 text-slate-500" />}
                >
                  {heartbeat.status === "paused" ? t.resumeHeartbeat : t.pauseHeartbeat}
                </Button>
              )}

              {canAdmin && heartbeat?.status !== "revoked" && (
                <Button
                  variant="secondary"
                  size="sm"
                  icon={<KeyRound className="w-3.5 h-3.5 text-amber-500" />}
                  onClick={() => setShowRotateModal(true)}
                >
                  {t.rotateToken}
                </Button>
              )}

              {canManage && (
                <Link href={`/heartbeats/${id}/edit`}>
                  <Button variant="secondary" size="sm" icon={<Edit className="w-3.5 h-3.5" />}>
                    {t.edit}
                  </Button>
                </Link>
              )}

              {canAdmin && heartbeat?.status !== "revoked" && (
                <Button
                  variant="danger"
                  size="sm"
                  icon={<ShieldAlert className="w-3.5 h-3.5" />}
                  onClick={() => setShowRevokeModal(true)}
                >
                  {t.revokeToken}
                </Button>
              )}

              {canAdmin && (
                <Button
                  variant="danger"
                  size="sm"
                  icon={<Trash2 className="w-3.5 h-3.5" />}
                  onClick={() => setShowDeleteModal(true)}
                >
                  {t.delete}
                </Button>
              )}
            </div>
          }
        />

        {/* Semantic limitation alert */}
        <div className="flex items-start gap-3 p-4 bg-sky-50 border border-sky-200 rounded-xl text-xs text-sky-900">
          <Info className="w-4 h-4 text-sky-600 mt-0.5 shrink-0" />
          <p className="leading-relaxed">{t.semanticNotice}</p>
        </div>

        {/* Missed heartbeat distinct notice if currently missed */}
        {heartbeat && heartbeat.status === "missed" && (
          <div className="flex items-start gap-3 p-4 bg-purple-50 border border-purple-200 rounded-xl text-xs text-purple-900">
            <AlertTriangle className="w-5 h-5 text-purple-600 mt-0.5 shrink-0" />
            <div>
              <h4 className="font-bold">{t.heartbeatMissedNotice}</h4>
              <p className="mt-0.5 text-purple-700 leading-relaxed">
                {isRTL
                  ? "این وضعیت نشان‌دهنده عدم ارسال پینگ از سوی اسکریپت پس‌زمینه یا کرون‌جاب است و به معنای قطعی وب‌سایت یا سرور نیست."
                  : "This state indicates a background job or cron script failed to ping within the expected window. Website and public API availability remain unaffected."}
              </p>
            </div>
          </div>
        )}

        {loading ? (
          <div className="bg-white border border-slate-200 rounded-2xl p-6 shadow-sm">
            <TableSkeleton rows={4} cols={4} />
          </div>
        ) : error || !heartbeat ? (
          <ErrorState error={error} onRetry={fetchDetail} />
        ) : (
          <div className="space-y-6">
            {/* Overview Card */}
            <div className="bg-white border border-slate-200 rounded-2xl p-5 sm:p-7 shadow-sm">
              <div className="flex items-center justify-between pb-4 border-b border-slate-100 mb-6 flex-wrap gap-3">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-xl bg-purple-50 border border-purple-100 flex items-center justify-center text-purple-600 shrink-0">
                    <Radio className="w-5 h-5" />
                  </div>
                  <div className="min-w-0">
                    <h2 className="text-base font-bold text-slate-900 truncate">{heartbeat.name}</h2>
                    <p className="text-xs text-slate-400 mt-0.5" dir="ltr">
                      ID: <span className="font-mono text-slate-600">{heartbeat.id}</span>
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-1.5 flex-wrap">
                  <Badge variant={heartbeat.status} />
                  {heartbeat.status === "missed" && (
                    <span className="text-[10px] bg-purple-50 text-purple-700 border border-purple-200 px-2 py-0.5 rounded font-medium">
                      {isRTL ? "جاب پس‌زمینه" : "Cron Job Missed"}
                    </span>
                  )}
                </div>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5 text-xs">
                <div>
                  <span className="block text-[10px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.client}
                  </span>
                  <Link
                    href={`/clients/${heartbeat.client_id}`}
                    className="font-medium text-slate-900 hover:text-blue-600 hover:underline flex items-center gap-1"
                  >
                    <span className="truncate">{heartbeat.client_name}</span>
                    <ExternalLink className="w-3 h-3 text-slate-400 shrink-0" />
                  </Link>
                </div>

                <div>
                  <span className="block text-[10px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.asset}
                  </span>
                  {heartbeat.asset_name ? (
                    <span className="font-medium text-slate-700 truncate block">{heartbeat.asset_name}</span>
                  ) : (
                    <span className="text-slate-400 italic">—</span>
                  )}
                </div>

                <div>
                  <span className="block text-[10px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.expectedInterval}
                  </span>
                  <span className="font-mono text-slate-800 font-medium" dir="ltr">
                    {formatInterval(heartbeat.expected_interval_seconds)}
                  </span>
                </div>

                <div>
                  <span className="block text-[10px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.gracePeriod}
                  </span>
                  <span className="font-mono text-slate-800 font-medium" dir="ltr">
                    {formatInterval(heartbeat.grace_period_seconds)}
                  </span>
                </div>

                <div>
                  <span className="block text-[10px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.lastReceived}
                  </span>
                  <span className="text-slate-700 font-medium">
                    {heartbeat.last_received_at
                      ? new Date(heartbeat.last_received_at).toLocaleString([], {
                          timeZone: activeOrg?.timezone || "UTC",
                        })
                      : t.neverReceived}
                  </span>
                </div>

                <div>
                  <span className="block text-[10px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.nextExpected}
                  </span>
                  <span className="text-slate-700 font-medium">
                    {heartbeat.next_expected_at
                      ? new Date(heartbeat.next_expected_at).toLocaleString([], {
                          timeZone: activeOrg?.timezone || "UTC",
                        })
                      : "—"}
                  </span>
                </div>

                <div>
                  <span className="block text-[10px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {isRTL ? "تاریخ ساخت" : "Created At"}
                  </span>
                  <span className="text-slate-500">
                    {new Date(heartbeat.created_at).toLocaleDateString([], {
                      timeZone: activeOrg?.timezone || "UTC",
                    })}
                  </span>
                </div>
              </div>
            </div>

            {/* Incidents Section */}
            <div className="bg-white border border-slate-200 rounded-2xl shadow-sm overflow-hidden">
              <div className="p-4 sm:p-5 border-b border-slate-100 flex justify-between items-center">
                <h3 className="text-xs font-bold text-slate-900 flex items-center gap-1.5 uppercase tracking-wider">
                  <AlertTriangle className="w-4 h-4 text-purple-600" />
                  <span>{t.incidents}</span>
                </h3>
                {incidents.length > 0 && (
                  <span className="text-xs text-slate-400">
                    {incidents.length} {t.incidents}
                  </span>
                )}
              </div>

              {incidents.length === 0 ? (
                <div className="p-8 text-center text-xs text-slate-400">
                  {t.noIncidentsRecordedForHeartbeat}
                </div>
              ) : (
                <>
                  {/* Desktop Table View (>= 768px) */}
                  <div className="hidden md:block overflow-x-auto">
                    <table className="w-full text-start border-collapse text-xs" dir={isRTL ? "rtl" : "ltr"}>
                      <thead>
                        <tr className="bg-slate-50 border-b border-slate-200 text-slate-500 font-semibold uppercase tracking-wider text-[11px]">
                          <th className="py-2.5 px-4">{t.status}</th>
                          <th className="py-2.5 px-4">{t.failureCategory}</th>
                          <th className="py-2.5 px-4">{t.firstFailure}</th>
                          <th className="py-2.5 px-4">{t.resolvedAt}</th>
                          <th className="py-2.5 px-4">{t.duration}</th>
                          <th className="py-2.5 px-4 text-end">{t.actions}</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-100">
                        {incidents.map((inc) => (
                          <tr key={inc.id} className="hover:bg-slate-50 transition-colors">
                            <td className="py-3 px-4">
                              <Badge variant={inc.status} />
                            </td>
                            <td className="py-3 px-4 font-mono text-slate-700 font-semibold" dir="ltr">
                              {inc.failure_category}
                            </td>
                            <td className="py-3 px-4 text-slate-600">
                              {new Date(inc.first_failure_at).toLocaleString([], {
                                timeZone: activeOrg?.timezone || "UTC",
                              })}
                            </td>
                            <td className="py-3 px-4 text-slate-600">
                              {inc.resolved_at
                                ? new Date(inc.resolved_at).toLocaleString([], {
                                    timeZone: activeOrg?.timezone || "UTC",
                                  })
                                : "—"}
                            </td>
                            <td className="py-3 px-4 text-slate-600 font-mono" dir="ltr">
                              {formatDuration(inc.duration_seconds)}
                            </td>
                            <td className="py-3 px-4 text-end">
                              <Link
                                href={`/incidents/${inc.id}`}
                                className="inline-flex items-center gap-1 text-xs font-semibold text-blue-600 hover:text-blue-800"
                              >
                                <span>{t.viewDetails}</span>
                                {isRTL ? <ArrowLeft className="w-3.5 h-3.5" /> : <ArrowRight className="w-3.5 h-3.5" />}
                              </Link>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>

                  {/* Mobile Card View (< 768px, 390px, 360px) */}
                  <div className="md:hidden divide-y divide-slate-100">
                    {incidents.map((inc) => (
                      <div key={inc.id} className="p-3.5 space-y-2">
                        <div className="flex items-center justify-between gap-2">
                          <Badge variant={inc.status} />
                          <span className="font-mono text-[11px] font-bold text-slate-700" dir="ltr">
                            {inc.failure_category}
                          </span>
                        </div>

                        <div className="text-xs text-slate-600 space-y-1 pt-1">
                          <div className="flex justify-between">
                            <span className="text-slate-400">{t.firstFailure}:</span>
                            <span className="font-medium">
                              {new Date(inc.first_failure_at).toLocaleTimeString([], {
                                hour: "2-digit",
                                minute: "2-digit",
                                month: "short",
                                day: "numeric",
                              })}
                            </span>
                          </div>
                          <div className="flex justify-between">
                            <span className="text-slate-400">{t.duration}:</span>
                            <span className="font-mono font-medium" dir="ltr">{formatDuration(inc.duration_seconds)}</span>
                          </div>
                        </div>

                        <div className="pt-2 border-t border-slate-50 flex justify-end">
                          <Link href={`/incidents/${inc.id}`}>
                            <Button size="sm" variant="secondary">
                              {t.viewDetails}
                            </Button>
                          </Link>
                        </div>
                      </div>
                    ))}
                  </div>
                </>
              )}
            </div>
          </div>
        )}

        {/* Rotate Confirmation Modal */}
        <ConfirmModal
          isOpen={showRotateModal}
          onClose={() => setShowRotateModal(false)}
          onConfirm={handleRotate}
          title={t.rotateToken}
          message={t.rotateTokenConfirm}
          confirmLabel={t.rotateToken}
          variant="danger"
          loading={actionLoading}
        />

        {/* Revoke Confirmation Modal */}
        <ConfirmModal
          isOpen={showRevokeModal}
          onClose={() => setShowRevokeModal(false)}
          onConfirm={handleRevoke}
          title={t.revokeToken}
          message={t.revokeTokenConfirm}
          confirmLabel={t.revokeToken}
          variant="danger"
          loading={actionLoading}
        />

        {/* Delete Confirmation Modal */}
        <ConfirmModal
          isOpen={showDeleteModal}
          onClose={() => setShowDeleteModal(false)}
          onConfirm={handleDelete}
          title={t.deleteHeartbeatTitle}
          message={t.deleteHeartbeatConfirm}
          confirmLabel={t.delete}
          variant="danger"
          loading={actionLoading}
        />

        {/* Accessible Rotated Secret Token Reveal Modal */}
        {rotatedData && (
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby="rotated-token-modal-title"
            className="fixed inset-0 z-50 bg-slate-900/50 backdrop-blur-sm flex items-center justify-center p-4 animate-in fade-in duration-150"
            onClick={() => setRotatedData(null)}
          >
            <div
              className="bg-white rounded-2xl max-w-xl w-full p-5 sm:p-7 shadow-2xl border border-slate-200 space-y-5 relative max-h-[90vh] overflow-y-auto"
              dir={isRTL ? "rtl" : "ltr"}
              onClick={(e) => e.stopPropagation()}
            >
              <button
                onClick={() => setRotatedData(null)}
                className="absolute top-4 end-4 p-1.5 text-slate-400 hover:text-slate-600 rounded-lg hover:bg-slate-100 transition-colors"
                aria-label={t.close}
              >
                <X className="w-4 h-4" />
              </button>

              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl bg-amber-100 flex items-center justify-center text-amber-600 shrink-0">
                  <KeyRound className="w-6 h-6" />
                </div>
                <div>
                  <h3 id="rotated-token-modal-title" className="text-base font-bold text-slate-900">
                    {t.newTokenGenerated}
                  </h3>
                  <p className="text-xs text-slate-500 mt-0.5">
                    {t.tokenWarning}
                  </p>
                </div>
              </div>

              {/* Token Copy Box */}
              <div>
                <label className="block text-[11px] font-bold text-slate-500 uppercase tracking-wider mb-1">
                  {isRTL ? "توکن جدید" : "New Secret Token"}
                </label>
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    readOnly
                    value={rotatedData.token}
                    className="w-full text-xs font-mono bg-slate-50 border border-slate-200 px-3 py-2 rounded-xl text-slate-900 select-all text-start"
                    dir="ltr"
                  />
                  <Button
                    size="sm"
                    variant="secondary"
                    icon={copiedToken ? <Check className="w-3.5 h-3.5 text-emerald-600" /> : <Copy className="w-3.5 h-3.5" />}
                    onClick={() => copyToClipboard(rotatedData.token, "token")}
                  >
                    {copiedToken ? t.tokenCopied : t.copyToken}
                  </Button>
                </div>
              </div>

              {/* Ready-to-copy Command Snippets */}
              <div className="space-y-3 pt-2 border-t border-slate-100">
                <h4 className="text-xs font-bold text-slate-900 flex items-center gap-1.5">
                  <Terminal className="w-3.5 h-3.5 text-slate-600" />
                  <span>{t.commandSnippets}</span>
                </h4>

                <div className="space-y-1">
                  <div className="flex justify-between items-center text-[11px] font-semibold text-slate-600">
                    <span>{t.bashCronSnippet}</span>
                    <button
                      onClick={() => copyToClipboard(rotatedData.bashSnippet, "bash")}
                      className="text-blue-600 hover:text-blue-700 flex items-center gap-1 text-xs"
                    >
                      {copiedBash ? <Check className="w-3 h-3 text-emerald-600" /> : <Copy className="w-3 h-3" />}
                      <span>{copiedBash ? t.tokenCopied : t.copyCommand}</span>
                    </button>
                  </div>
                  <pre className="text-[11px] font-mono bg-slate-900 text-slate-100 p-3 rounded-xl overflow-x-auto text-start" dir="ltr">
                    <code>{rotatedData.bashSnippet}</code>
                  </pre>
                </div>

                <div className="space-y-1">
                  <div className="flex justify-between items-center text-[11px] font-semibold text-slate-600">
                    <span>{t.powerShellSnippet}</span>
                    <button
                      onClick={() => copyToClipboard(rotatedData.powerShellSnippet, "ps")}
                      className="text-blue-600 hover:text-blue-700 flex items-center gap-1 text-xs"
                    >
                      {copiedPS ? <Check className="w-3 h-3 text-emerald-600" /> : <Copy className="w-3 h-3" />}
                      <span>{copiedPS ? t.tokenCopied : t.copyCommand}</span>
                    </button>
                  </div>
                  <pre className="text-[11px] font-mono bg-slate-900 text-slate-100 p-3 rounded-xl overflow-x-auto text-start" dir="ltr">
                    <code>{rotatedData.powerShellSnippet}</code>
                  </pre>
                </div>
              </div>

              <div className="pt-3 border-t border-slate-100 flex justify-end">
                <Button onClick={() => setRotatedData(null)}>
                  {t.close}
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </AppLayout>
  );
}
