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

  const handleTogglePause = async () => {
    if (!heartbeat) return;
    const isPaused = heartbeat.status === "paused";
    const endpoint = isPaused
      ? `/api/v1/heartbeats/${id}/resume`
      : `/api/v1/heartbeats/${id}/pause`;

    try {
      await apiFetch(endpoint, { method: "POST" });
      toast.success(isPaused ? (isRTL ? "فعال شد" : "Resumed") : (isRTL ? "متوقف شد" : "Paused"));
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
      toast.success(isRTL ? "توکن با موفقیت بازتولید شد" : "Token rotated successfully");
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
      toast.success(isRTL ? "توکن با موفقیت باطل شد" : "Token revoked");
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
      toast.success(isRTL ? "هارت‌بیت آرشیو شد" : "Heartbeat archived");
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
    if (sec >= 86400) return `${Math.round(sec / 86400)} ${isRTL ? "روز" : "days"}`;
    if (sec >= 3600) return `${Math.round(sec / 3600)} ${isRTL ? "ساعت" : "hours"}`;
    if (sec >= 60) return `${Math.round(sec / 60)} ${isRTL ? "دقیقه" : "min"}`;
    return `${sec}s`;
  };

  return (
    <AppLayout>
      <div className="space-y-6 max-w-5xl mx-auto">
        <PageHeader
          title={heartbeat?.name || t.heartbeat}
          description={heartbeat?.description}
          breadcrumbs={[
            { label: t.dashboard, href: "/" },
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
                  icon={heartbeat.status === "paused" ? <Play className="w-3.5 h-3.5" /> : <Pause className="w-3.5 h-3.5" />}
                >
                  {heartbeat.status === "paused" ? t.resumeHeartbeat : t.pauseHeartbeat}
                </Button>
              )}

              {canAdmin && heartbeat?.status !== "revoked" && (
                <Button
                  variant="secondary"
                  size="sm"
                  icon={<KeyRound className="w-3.5 h-3.5" />}
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

        {loading ? (
          <div className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm">
            <TableSkeleton rows={4} cols={4} />
          </div>
        ) : error || !heartbeat ? (
          <ErrorState
            error={error}
            onRetry={fetchDetail}
          />
        ) : (
          <div className="space-y-6">
            {/* Overview Card */}
            <div className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm">
              <div className="flex items-center justify-between pb-4 border-b border-slate-100 mb-6">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 rounded-xl bg-slate-100 flex items-center justify-center text-slate-700">
                    <Radio className="w-5 h-5" />
                  </div>
                  <div>
                    <h2 className="text-sm font-bold text-slate-900">{heartbeat.name}</h2>
                    <p className="text-xs text-slate-400">
                      ID: <span className="font-mono text-slate-600">{heartbeat.id}</span>
                    </p>
                  </div>
                </div>
                <Badge variant={heartbeat.status} />
              </div>

              <div className="grid grid-cols-2 md:grid-cols-4 gap-6 text-xs">
                <div>
                  <span className="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.client}
                  </span>
                  <Link
                    href={`/clients/${heartbeat.client_id}`}
                    className="font-medium text-slate-900 hover:text-blue-600 hover:underline flex items-center gap-1"
                  >
                    <span>{heartbeat.client_name}</span>
                    <ExternalLink className="w-3 h-3 text-slate-400" />
                  </Link>
                </div>

                <div>
                  <span className="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.asset}
                  </span>
                  {heartbeat.asset_name ? (
                    <span className="font-medium text-slate-700">{heartbeat.asset_name}</span>
                  ) : (
                    <span className="text-slate-400 italic">—</span>
                  )}
                </div>

                <div>
                  <span className="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.expectedInterval}
                  </span>
                  <span className="font-mono text-slate-800 font-medium">
                    {formatInterval(heartbeat.expected_interval_seconds)}
                  </span>
                </div>

                <div>
                  <span className="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.gracePeriod}
                  </span>
                  <span className="font-mono text-slate-800 font-medium">
                    {formatInterval(heartbeat.grace_period_seconds)}
                  </span>
                </div>

                <div>
                  <span className="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.lastReceived}
                  </span>
                  <span className="text-slate-700 font-medium">
                    {heartbeat.last_received_at
                      ? new Date(heartbeat.last_received_at).toLocaleString()
                      : (isRTL ? "هنوز پینگی ثبت نشده است" : "Never received")}
                  </span>
                </div>

                <div>
                  <span className="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {t.nextExpected}
                  </span>
                  <span className="text-slate-700 font-medium">
                    {heartbeat.next_expected_at
                      ? new Date(heartbeat.next_expected_at).toLocaleString()
                      : "—"}
                  </span>
                </div>

                <div>
                  <span className="block text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                    {isRTL ? "تاریخ ساخت" : "Created At"}
                  </span>
                  <span className="text-slate-500">
                    {new Date(heartbeat.created_at).toLocaleDateString()}
                  </span>
                </div>
              </div>
            </div>

            {/* Incidents Section */}
            <div className="bg-white border border-slate-200 rounded-xl shadow-sm overflow-hidden">
              <div className="p-4 border-b border-slate-100 flex justify-between items-center">
                <h3 className="text-xs font-bold text-slate-900 flex items-center gap-1.5">
                  <AlertTriangle className="w-4 h-4 text-slate-600" />
                  {t.incidents}
                </h3>
              </div>

              {incidents.length === 0 ? (
                <div className="p-8 text-center text-xs text-slate-400">
                  {isRTL ? "هیچ حادثه یا قطعی برای این هارت‌بیت ثبت نشده است." : "No incidents recorded for this heartbeat."}
                </div>
              ) : (
                <table className="w-full text-left rtl:text-right border-collapse text-xs">
                  <thead>
                    <tr className="bg-slate-50 border-b border-slate-200 text-slate-500 font-semibold uppercase tracking-wider text-[11px]">
                      <th className="py-2.5 px-4">{t.status}</th>
                      <th className="py-2.5 px-4">{t.failureCategory}</th>
                      <th className="py-2.5 px-4">{t.firstFailure}</th>
                      <th className="py-2.5 px-4">{t.resolvedAt}</th>
                      <th className="py-2.5 px-4">{t.duration}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {incidents.map((inc) => (
                      <tr key={inc.id} className="hover:bg-slate-50 transition-colors">
                        <td className="py-3 px-4">
                          <Badge variant={inc.status} />
                        </td>
                        <td className="py-3 px-4 font-mono text-slate-700">
                          {inc.failure_category}
                        </td>
                        <td className="py-3 px-4 text-slate-600">
                          {new Date(inc.first_failure_at).toLocaleString()}
                        </td>
                        <td className="py-3 px-4 text-slate-600">
                          {inc.resolved_at ? new Date(inc.resolved_at).toLocaleString() : "—"}
                        </td>
                        <td className="py-3 px-4 text-slate-600 font-mono">
                          {inc.duration_seconds !== undefined ? `${inc.duration_seconds}s` : "—"}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
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
          message={
            isRTL
              ? "آیا از چرخش توکن اطمینان دارید؟ توکن قبلی بلافاصله باطل شده و تا زمان بروزرسانی اسکریپت‌ها، پینگ‌های قبلی مردود خواهند شد."
              : "Are you sure you want to rotate this token? The previous token will be invalidated immediately."
          }
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
          message={
            isRTL
              ? "آیا از ابطال دائمی توکن این هارت‌بیت اطمینان دارید؟ پس از ابطال، هیچ پینگی برای آن پذیرفته نخواهد شد."
              : "Are you sure you want to permanently revoke this token? No pings will be accepted."
          }
          confirmLabel={t.revokeToken}
          variant="danger"
          loading={actionLoading}
        />

        {/* Delete Confirmation Modal */}
        <ConfirmModal
          isOpen={showDeleteModal}
          onClose={() => setShowDeleteModal(false)}
          onConfirm={handleDelete}
          title={t.delete}
          message={t.areYouSure}
          confirmLabel={t.delete}
          variant="danger"
          loading={actionLoading}
        />

        {/* Rotated Secret Token Reveal Modal */}
        {rotatedData && (
          <div className="fixed inset-0 z-50 bg-slate-900/50 backdrop-blur-sm flex items-center justify-center p-4">
            <div className="bg-white rounded-2xl max-w-xl w-full p-6 shadow-2xl border border-slate-200 space-y-5 animate-in fade-in duration-200">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl bg-amber-100 flex items-center justify-center text-amber-600 shrink-0">
                  <KeyRound className="w-6 h-6" />
                </div>
                <div>
                  <h3 className="text-sm font-bold text-slate-900">
                    {isRTL ? "توکن جدید با موفقیت تولید شد" : "New Token Generated"}
                  </h3>
                  <p className="text-xs text-slate-500">
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
                    className="w-full text-xs font-mono bg-slate-50 border border-slate-200 px-3 py-2 rounded-lg text-slate-900 select-all"
                  />
                  <Button
                    size="sm"
                    variant="secondary"
                    icon={copiedToken ? <Check className="w-3.5 h-3.5 text-emerald-600" /> : <Copy className="w-3.5 h-3.5" />}
                    onClick={() => copyToClipboard(rotatedData.token, "token")}
                  >
                    {copiedToken ? (isRTL ? "کپی شد" : "Copied") : (isRTL ? "کپی" : "Copy")}
                  </Button>
                </div>
              </div>

              {/* Ready-to-copy Command Snippets */}
              <div className="space-y-3 pt-2 border-t border-slate-100">
                <h4 className="text-xs font-bold text-slate-900 flex items-center gap-1.5">
                  <Terminal className="w-3.5 h-3.5 text-slate-600" />
                  {t.commandSnippets}
                </h4>

                <div className="space-y-1">
                  <div className="flex justify-between items-center text-[11px] font-semibold text-slate-600">
                    <span>{t.bashCronSnippet}</span>
                    <button
                      onClick={() => copyToClipboard(rotatedData.bashSnippet, "bash")}
                      className="text-blue-600 hover:text-blue-700 flex items-center gap-1"
                    >
                      {copiedBash ? <Check className="w-3 h-3 text-emerald-600" /> : <Copy className="w-3 h-3" />}
                      <span>{copiedBash ? (isRTL ? "کپی شد" : "Copied") : (isRTL ? "کپی" : "Copy")}</span>
                    </button>
                  </div>
                  <pre className="text-[11px] font-mono bg-slate-900 text-slate-100 p-3 rounded-lg overflow-x-auto text-left" dir="ltr">
                    <code>{rotatedData.bashSnippet}</code>
                  </pre>
                </div>

                <div className="space-y-1">
                  <div className="flex justify-between items-center text-[11px] font-semibold text-slate-600">
                    <span>{t.powerShellSnippet}</span>
                    <button
                      onClick={() => copyToClipboard(rotatedData.powerShellSnippet, "ps")}
                      className="text-blue-600 hover:text-blue-700 flex items-center gap-1"
                    >
                      {copiedPS ? <Check className="w-3 h-3 text-emerald-600" /> : <Copy className="w-3 h-3" />}
                      <span>{copiedPS ? (isRTL ? "کپی شد" : "Copied") : (isRTL ? "کپی" : "Copy")}</span>
                    </button>
                  </div>
                  <pre className="text-[11px] font-mono bg-slate-900 text-slate-100 p-3 rounded-lg overflow-x-auto text-left" dir="ltr">
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
