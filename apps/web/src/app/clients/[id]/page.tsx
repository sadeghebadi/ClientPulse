"use client";

import React, { useEffect, useState, use, useCallback } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import {
  Users,
  Globe,
  Activity,
  Plus,
  ExternalLink,
  Edit,
  Archive,
  RotateCcw,
  ArrowLeft,
  ArrowRight,
  Mail,
  Wrench,
  FileText,
  Radio,
  AlertTriangle,
  Clock,
  Calendar,
} from "lucide-react";
import { PageHeader } from "@/components/ui/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { ConfirmModal } from "@/components/ui/ConfirmModal";
import { useToast } from "@/components/ui/Toast";

interface Asset {
  id: string;
  name: string;
  url: string;
  type: string;
  environment: string;
  is_active: boolean;
  monitor_id?: string;
  monitor_status?: string;
  last_checked_at?: string;
}

interface ClientDetail {
  id: string;
  name: string;
  contact_name?: string;
  contact_email?: string;
  website?: string;
  notes?: string;
  status: "active" | "archived";
  asset_count: number;
  monitor_count: number;
  created_at: string;
}

interface HeartbeatItem {
  id: string;
  name: string;
  status: string;
  expected_interval_seconds: number;
  last_received_at?: string;
}

interface IncidentItem {
  id: string;
  target_name: string;
  target_type: "monitor" | "heartbeat";
  status: "open" | "resolved";
  failure_category: string;
  first_failure_at: string;
  acknowledged_at?: string;
}

interface MaintenanceItem {
  id: string;
  title: string;
  category: string;
  performed_at: string;
  duration_minutes?: number;
}

export default function ClientDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = use(params);
  const clientId = resolvedParams.id;
  const router = useRouter();
  const { user, activeOrg, role } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const [client, setClient] = useState<ClientDetail | null>(null);
  const [assets, setAssets] = useState<Asset[]>([]);
  const [heartbeats, setHeartbeats] = useState<HeartbeatItem[]>([]);
  const [incidents, setIncidents] = useState<IncidentItem[]>([]);
  const [maintenance, setMaintenance] = useState<MaintenanceItem[]>([]);

  const [activeTab, setActiveTab] = useState<"assets" | "heartbeats" | "incidents" | "maintenance">("assets");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<any>(null);

  // Archive / Restore Modal
  const [modalOpen, setModalOpen] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);

  const canManage = role === "owner" || role === "admin" || role === "member";

  const reloadClientData = useCallback(() => {
    if (!user || !activeOrg) return;
    setLoading(true);
    setError(null);

    Promise.all([
      apiFetch<{ client: ClientDetail }>(`/api/v1/clients/${clientId}`),
      apiFetch<{ assets: Asset[] }>(`/api/v1/assets?client_id=${clientId}`),
      apiFetch<{ heartbeats: HeartbeatItem[] }>(`/api/v1/heartbeats?client_id=${clientId}&limit=10`).catch(() => ({ heartbeats: [] })),
      apiFetch<{ incidents: IncidentItem[] }>(`/api/v1/incidents?client_id=${clientId}&limit=10`).catch(() => ({ incidents: [] })),
      apiFetch<{ activities: MaintenanceItem[] }>(`/api/v1/maintenance-activities?client_id=${clientId}&limit=10`).catch(() => ({ activities: [] })),
    ])
      .then(([clientRes, assetsRes, hbRes, incRes, maintRes]) => {
        setClient(clientRes.client);
        setAssets(assetsRes.assets || []);
        setHeartbeats(hbRes.heartbeats || []);
        setIncidents(incRes.incidents || []);
        setMaintenance(maintRes.activities || []);
      })
      .catch((err) => {
        setError(err);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [user, activeOrg, clientId]);

  useEffect(() => {
    if (!user || !activeOrg) return;
    let isMounted = true;

    Promise.all([
      apiFetch<{ client: ClientDetail }>(`/api/v1/clients/${clientId}`),
      apiFetch<{ assets: Asset[] }>(`/api/v1/assets?client_id=${clientId}`),
      apiFetch<{ heartbeats: HeartbeatItem[] }>(`/api/v1/heartbeats?client_id=${clientId}&limit=10`).catch(() => ({ heartbeats: [] })),
      apiFetch<{ incidents: IncidentItem[] }>(`/api/v1/incidents?client_id=${clientId}&limit=10`).catch(() => ({ incidents: [] })),
      apiFetch<{ activities: MaintenanceItem[] }>(`/api/v1/maintenance-activities?client_id=${clientId}&limit=10`).catch(() => ({ activities: [] })),
    ])
      .then(([clientRes, assetsRes, hbRes, incRes, maintRes]) => {
        if (!isMounted) return;
        setClient(clientRes.client);
        setAssets(assetsRes.assets || []);
        setHeartbeats(hbRes.heartbeats || []);
        setIncidents(incRes.incidents || []);
        setMaintenance(maintRes.activities || []);
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
  }, [user, activeOrg, clientId]);

  const handleConfirmArchiveToggle = async () => {
    if (!client) return;
    try {
      setActionLoading(true);
      const isArchived = client.status === "archived";
      const endpoint = isArchived
        ? `/api/v1/clients/${client.id}/restore`
        : `/api/v1/clients/${client.id}/archive`;

      await apiFetch(endpoint, { method: "POST" });
      toast.success(
        isArchived
          ? (isRTL ? "مشتری به حالت فعال بازگردانده شد." : "Client restored to active status")
          : (isRTL ? "مشتری با موفقیت آرشیو شد." : "Client archived successfully")
      );
      setModalOpen(false);
      reloadClientData();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Action failed");
    } finally {
      setActionLoading(false);
    }
  };

  const openIncidentsCount = incidents.filter((i) => i.status === "open").length;

  return (
    <AppLayout>
      <PageHeader
        title={client?.name || t.clientDetails}
        description={isRTL ? `مدیریت مشخصات، وب‌سایت‌های پایش‌شده، هارت‌بیت‌ها و سوابق ${client?.name || ""}.` : `Manage client profile, monitored websites, heartbeats, and SLAs for ${client?.name || ""}.`}
        icon={<Users className="w-5 h-5" />}
        breadcrumbs={[
          { label: t.clients, href: "/clients" },
          { label: client?.name || t.clientDetails },
        ]}
        badge={client && <Badge variant={client.status} />}
        actions={
          client && (
            <div className="flex flex-wrap items-center gap-2">
              <Link href={`/maintenance?client_id=${client.id}`}>
                <Button size="sm" variant="outline" icon={<Wrench className="w-4 h-4" />}>
                  {t.maintenance}
                </Button>
              </Link>
              <Link href={`/reports?client_id=${client.id}`}>
                <Button size="sm" variant="outline" icon={<FileText className="w-4 h-4" />}>
                  {t.reports}
                </Button>
              </Link>
              {canManage && (
                <>
                  <Link href={`/clients/${client.id}/edit`}>
                    <Button size="sm" variant="outline" icon={<Edit className="w-4 h-4" />}>
                      {t.editClient}
                    </Button>
                  </Link>
                  <Button
                    size="sm"
                    variant={client.status === "active" ? "outline" : "primary"}
                    onClick={() => setModalOpen(true)}
                    icon={client.status === "active" ? <Archive className="w-4 h-4" /> : <RotateCcw className="w-4 h-4" />}
                  >
                    {client.status === "active" ? t.archiveClient : t.restoreClient}
                  </Button>
                  <Link href={`/assets/new?client_id=${client.id}`}>
                    <Button size="sm" variant="primary" icon={<Plus className="w-4 h-4" />}>
                      {t.createAsset}
                    </Button>
                  </Link>
                </>
              )}
            </div>
          )
        }
      />

      {error && <ErrorState error={error} onRetry={reloadClientData} className="mb-6" />}

      {loading ? (
        <TableSkeleton rows={4} cols={5} />
      ) : client ? (
        <div className="space-y-6 sm:space-y-8">
          {/* Client Overview Card & Contact Info */}
          <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-5 sm:p-7">
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-5 text-xs text-slate-600 pb-6 border-b border-slate-100">
              <div>
                <div className="text-[10px] uppercase font-bold text-slate-400 mb-1">{t.contactName}</div>
                <div className="text-sm font-bold text-slate-900">{client.contact_name || "—"}</div>
              </div>

              <div>
                <div className="text-[10px] uppercase font-bold text-slate-400 mb-1">{t.contactEmail}</div>
                <div className="text-sm font-bold text-slate-900 flex items-center gap-1.5" dir="ltr">
                  {client.contact_email ? (
                    <>
                      <Mail className="w-3.5 h-3.5 text-slate-400 shrink-0" />
                      <a href={`mailto:${client.contact_email}`} className="text-blue-600 hover:underline truncate">
                        {client.contact_email}
                      </a>
                    </>
                  ) : (
                    "—"
                  )}
                </div>
              </div>

              <div>
                <div className="text-[10px] uppercase font-bold text-slate-400 mb-1">{t.website}</div>
                <div className="text-sm font-bold text-slate-900" dir="ltr">
                  {client.website ? (
                    <a
                      href={client.website}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1 text-blue-600 hover:underline truncate"
                    >
                      <span className="truncate">{client.website}</span>
                      <ExternalLink className="w-3 h-3 shrink-0" />
                    </a>
                  ) : (
                    "—"
                  )}
                </div>
              </div>
            </div>

            {/* Operational Summary Badges */}
            <div className="pt-5 flex flex-wrap items-center gap-3">
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-slate-50 border border-slate-200 text-xs font-semibold text-slate-700">
                <Globe className="w-3.5 h-3.5 text-blue-500 shrink-0" />
                <span>{t.clientWebsitesCount.replace("{count}", String(client.asset_count))}</span>
              </div>
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-slate-50 border border-slate-200 text-xs font-semibold text-slate-700">
                <Activity className="w-3.5 h-3.5 text-emerald-500 shrink-0" />
                <span>{t.clientMonitorsCount.replace("{count}", String(client.monitor_count))}</span>
              </div>
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-slate-50 border border-slate-200 text-xs font-semibold text-slate-700">
                <Radio className="w-3.5 h-3.5 text-purple-500 shrink-0" />
                <span>{t.clientHeartbeatsCount.replace("{count}", String(heartbeats.length))}</span>
              </div>
              {openIncidentsCount > 0 ? (
                <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-rose-50 border border-rose-200 text-xs font-bold text-rose-700">
                  <AlertTriangle className="w-3.5 h-3.5 text-rose-600 shrink-0" />
                  <span>{t.clientOpenIncidentsCount.replace("{count}", String(openIncidentsCount))}</span>
                </div>
              ) : (
                <div className="flex items-center gap-2 px-3 py-1.5 rounded-xl bg-emerald-50 border border-emerald-200 text-xs font-medium text-emerald-700">
                  <span className="w-2 h-2 rounded-full bg-emerald-500"></span>
                  <span>{t.noClientIncidents}</span>
                </div>
              )}
            </div>

            {client.notes && (
              <div className="pt-5 border-t border-slate-100 mt-5">
                <div className="text-[10px] uppercase font-bold text-slate-400 mb-1">{t.notes}</div>
                <p className="text-xs text-slate-600 bg-slate-50 p-3 sm:p-4 rounded-xl border border-slate-100 leading-relaxed">
                  {client.notes}
                </p>
              </div>
            )}
          </div>

          {/* Operational Relationship Tabs */}
          <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
            {/* Tabs Header */}
            <div className="border-b border-slate-100 bg-slate-50/50 p-2 sm:p-3 flex flex-wrap items-center gap-1 sm:gap-2">
              <button
                onClick={() => setActiveTab("assets")}
                className={`px-3 py-2 rounded-xl text-xs font-bold transition-all flex items-center gap-1.5 ${
                  activeTab === "assets"
                    ? "bg-white text-blue-600 shadow-sm border border-slate-200"
                    : "text-slate-600 hover:text-slate-900 hover:bg-slate-100"
                }`}
              >
                <Globe className="w-3.5 h-3.5" />
                <span>{t.clientOverviewTab}</span>
                <span className="ms-1 px-1.5 py-0.2 bg-slate-100 text-slate-600 rounded text-[10px] font-mono">
                  {assets.length}
                </span>
              </button>

              <button
                onClick={() => setActiveTab("heartbeats")}
                className={`px-3 py-2 rounded-xl text-xs font-bold transition-all flex items-center gap-1.5 ${
                  activeTab === "heartbeats"
                    ? "bg-white text-purple-600 shadow-sm border border-slate-200"
                    : "text-slate-600 hover:text-slate-900 hover:bg-slate-100"
                }`}
              >
                <Radio className="w-3.5 h-3.5" />
                <span>{t.clientHeartbeatsTab}</span>
                <span className="ms-1 px-1.5 py-0.2 bg-slate-100 text-slate-600 rounded text-[10px] font-mono">
                  {heartbeats.length}
                </span>
              </button>

              <button
                onClick={() => setActiveTab("incidents")}
                className={`px-3 py-2 rounded-xl text-xs font-bold transition-all flex items-center gap-1.5 ${
                  activeTab === "incidents"
                    ? "bg-white text-rose-600 shadow-sm border border-slate-200"
                    : "text-slate-600 hover:text-slate-900 hover:bg-slate-100"
                }`}
              >
                <AlertTriangle className="w-3.5 h-3.5" />
                <span>{t.clientIncidentsTab}</span>
                {openIncidentsCount > 0 ? (
                  <span className="ms-1 px-1.5 py-0.2 bg-rose-100 text-rose-700 rounded text-[10px] font-bold font-mono">
                    {openIncidentsCount}
                  </span>
                ) : (
                  <span className="ms-1 px-1.5 py-0.2 bg-slate-100 text-slate-600 rounded text-[10px] font-mono">
                    {incidents.length}
                  </span>
                )}
              </button>

              <button
                onClick={() => setActiveTab("maintenance")}
                className={`px-3 py-2 rounded-xl text-xs font-bold transition-all flex items-center gap-1.5 ${
                  activeTab === "maintenance"
                    ? "bg-white text-sky-600 shadow-sm border border-slate-200"
                    : "text-slate-600 hover:text-slate-900 hover:bg-slate-100"
                }`}
              >
                <Wrench className="w-3.5 h-3.5" />
                <span>{t.clientMaintenanceTab}</span>
                <span className="ms-1 px-1.5 py-0.2 bg-slate-100 text-slate-600 rounded text-[10px] font-mono">
                  {maintenance.length}
                </span>
              </button>
            </div>

            {/* Tab 1: Websites & Monitors */}
            {activeTab === "assets" && (
              <div>
                <div className="p-4 sm:p-5 border-b border-slate-100 flex items-center justify-between">
                  <div>
                    <h3 className="text-sm font-bold text-slate-900 tracking-tight">
                      {t.assets}
                    </h3>
                    <p className="text-xs text-slate-500 mt-0.5">
                      {isRTL ? `وب‌سایت‌ها و اندپوینت‌های ثبت‌شده برای ${client.name}.` : `Websites and API endpoints monitored for ${client.name}.`}
                    </p>
                  </div>

                  {canManage && (
                    <Link href={`/assets/new?client_id=${client.id}`}>
                      <Button size="sm" variant="primary" icon={<Plus className="w-3.5 h-3.5" />}>
                        {t.createAsset}
                      </Button>
                    </Link>
                  )}
                </div>

                {assets.length === 0 ? (
                  <EmptyState
                    icon={<Globe className="w-6 h-6 text-slate-400" />}
                    title={t.noAssetsFound}
                    description={t.noAssetsDesc}
                    actionHref={canManage ? `/assets/new?client_id=${client.id}` : undefined}
                    actionLabel={canManage ? t.createAsset : undefined}
                  />
                ) : (
                  <>
                    {/* Desktop Table View (>= 768px) */}
                    <div className="hidden md:block overflow-x-auto">
                      <table className="w-full text-start text-sm" dir={isRTL ? "rtl" : "ltr"}>
                        <thead className="bg-slate-50 text-slate-600 text-xs uppercase font-semibold border-b border-slate-200">
                          <tr>
                            <th className="px-6 py-4">{t.assetName}</th>
                            <th className="px-6 py-4">{t.url}</th>
                            <th className="px-6 py-4">{t.environment}</th>
                            <th className="px-6 py-4">{t.status}</th>
                            <th className="px-6 py-4 text-end">{t.actions}</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                          {assets.map((asset) => (
                            <tr key={asset.id} className="hover:bg-slate-50/70 transition-colors">
                              <td className="px-6 py-4 font-bold text-slate-900">
                                <Link href={`/assets/${asset.id}`} className="hover:text-blue-600 transition-colors">
                                  {asset.name}
                                </Link>
                                <Badge variant={asset.type} className="ms-2" />
                              </td>
                              <td className="px-6 py-4 text-xs font-mono text-slate-600" dir="ltr">
                                <a
                                  href={asset.url}
                                  target="_blank"
                                  rel="noopener noreferrer"
                                  className="inline-flex items-center gap-1 text-blue-600 hover:underline"
                                >
                                  <span className="truncate max-w-xs">{asset.url}</span>
                                  <ExternalLink className="w-3 h-3" />
                                </a>
                              </td>
                              <td className="px-6 py-4">
                                <Badge variant={asset.environment} />
                              </td>
                              <td className="px-6 py-4">
                                <Badge variant={asset.monitor_status || "pending"} />
                              </td>
                              <td className="px-6 py-4 text-end">
                                <div className="flex items-center justify-end gap-2">
                                  {asset.monitor_id ? (
                                    <Link
                                      href={`/monitors/${asset.monitor_id}`}
                                      className="text-xs font-semibold text-blue-600 hover:underline"
                                    >
                                      {t.monitorDetails}
                                    </Link>
                                  ) : canManage ? (
                                    <Link
                                      href={`/monitors/new?asset_id=${asset.id}`}
                                      className="text-xs font-semibold text-emerald-600 hover:underline"
                                    >
                                      + {t.createMonitor}
                                    </Link>
                                  ) : null}
                                  <Link
                                    href={`/assets/${asset.id}`}
                                    className="inline-flex items-center gap-1 text-xs font-semibold text-slate-600 hover:text-slate-900 ms-3"
                                  >
                                    <span>{t.viewDetails}</span>
                                    {isRTL ? <ArrowLeft className="w-3 h-3" /> : <ArrowRight className="w-3 h-3" />}
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
                      {assets.map((asset) => (
                        <div key={asset.id} className="p-4 space-y-3">
                          <div className="flex items-start justify-between gap-2">
                            <div>
                              <Link
                                href={`/assets/${asset.id}`}
                                className="font-bold text-slate-900 text-sm hover:text-blue-600"
                              >
                                {asset.name}
                              </Link>
                              <div className="flex items-center gap-1.5 mt-1">
                                <Badge variant={asset.environment} />
                                <Badge variant={asset.type} />
                              </div>
                            </div>
                            <Badge variant={asset.monitor_status || "pending"} />
                          </div>

                          <div className="font-mono text-xs text-blue-600 truncate text-start" dir="ltr">
                            <a href={asset.url} target="_blank" rel="noopener noreferrer" className="hover:underline flex items-center gap-1">
                              <span className="truncate">{asset.url}</span>
                              <ExternalLink className="w-3 h-3 shrink-0" />
                            </a>
                          </div>

                          <div className="flex items-center justify-between pt-2 border-t border-slate-50">
                            {asset.monitor_id ? (
                              <Link href={`/monitors/${asset.monitor_id}`}>
                                <Button size="sm" variant="outline">
                                  {t.monitorDetails}
                                </Button>
                              </Link>
                            ) : canManage ? (
                              <Link href={`/monitors/new?asset_id=${asset.id}`}>
                                <Button size="sm" variant="outline">
                                  + {t.createMonitor}
                                </Button>
                              </Link>
                            ) : <div />}

                            <Link href={`/assets/${asset.id}`}>
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
            )}

            {/* Tab 2: Heartbeats */}
            {activeTab === "heartbeats" && (
              <div>
                <div className="p-4 sm:p-5 border-b border-slate-100 flex items-center justify-between">
                  <div>
                    <h3 className="text-sm font-bold text-slate-900 tracking-tight">
                      {t.heartbeats}
                    </h3>
                    <p className="text-xs text-slate-500 mt-0.5">
                      {isRTL ? `مانیتورهای دوره‌ای کرون و بک‌آپ برای ${client.name}.` : `Scheduled cron jobs and backup signals for ${client.name}.`}
                    </p>
                  </div>

                  {canManage && (
                    <Link href={`/heartbeats/new?client_id=${client.id}`}>
                      <Button size="sm" variant="primary" icon={<Plus className="w-3.5 h-3.5" />}>
                        {t.addHeartbeat}
                      </Button>
                    </Link>
                  )}
                </div>

                {heartbeats.length === 0 ? (
                  <div className="p-8 text-center text-xs text-slate-400">
                    <Radio className="w-8 h-8 text-slate-300 mx-auto mb-2" />
                    <p className="font-semibold text-slate-600">{t.noClientHeartbeats}</p>
                    {canManage && (
                      <Link href={`/heartbeats/new?client_id=${client.id}`} className="mt-3 inline-block">
                        <Button size="sm" variant="outline">+ {t.addHeartbeat}</Button>
                      </Link>
                    )}
                  </div>
                ) : (
                  <div className="divide-y divide-slate-100">
                    {heartbeats.map((hb) => (
                      <div key={hb.id} className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 hover:bg-slate-50/70 transition-colors">
                        <div className="flex items-center gap-3">
                          <Radio className="w-4 h-4 text-purple-500 shrink-0" />
                          <div>
                            <Link href={`/heartbeats/${hb.id}`} className="font-bold text-sm text-slate-900 hover:text-blue-600">
                              {hb.name}
                            </Link>
                            <div className="text-xs text-slate-400 font-mono mt-0.5" dir="ltr">
                              ID: {hb.id}
                            </div>
                          </div>
                        </div>

                        <div className="flex items-center justify-between sm:justify-end gap-3">
                          <Badge variant={hb.status} />
                          <Link href={`/heartbeats/${hb.id}`}>
                            <Button size="sm" variant="secondary">
                              {t.viewDetails}
                            </Button>
                          </Link>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Tab 3: Incidents */}
            {activeTab === "incidents" && (
              <div>
                <div className="p-4 sm:p-5 border-b border-slate-100 flex items-center justify-between">
                  <div>
                    <h3 className="text-sm font-bold text-slate-900 tracking-tight">
                      {t.incidents}
                    </h3>
                    <p className="text-xs text-slate-500 mt-0.5">
                      {isRTL ? `رخدادهای قطعی و حوادث ثبت‌شده برای ${client.name}.` : `Outages and missed signals recorded for ${client.name}.`}
                    </p>
                  </div>

                  <Link href={`/incidents?client_id=${client.id}`}>
                    <Button size="sm" variant="outline">
                      {t.viewAllIncidents}
                    </Button>
                  </Link>
                </div>

                {incidents.length === 0 ? (
                  <div className="p-8 text-center text-xs text-slate-400">
                    <AlertTriangle className="w-8 h-8 text-slate-300 mx-auto mb-2" />
                    <p className="font-semibold text-slate-600">{t.noClientIncidents}</p>
                  </div>
                ) : (
                  <div className="divide-y divide-slate-100">
                    {incidents.map((inc) => (
                      <div key={inc.id} className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 hover:bg-slate-50/70 transition-colors">
                        <div>
                          <div className="flex items-center gap-2">
                            <Link href={`/incidents/${inc.id}`} className="font-bold text-sm text-slate-900 hover:text-blue-600">
                              {inc.target_name}
                            </Link>
                            <Badge variant={inc.status} />
                          </div>
                          <div className="flex items-center gap-3 text-xs text-slate-500 mt-1">
                            <span className="font-mono">{inc.failure_category}</span>
                            <span>•</span>
                            <span>{new Date(inc.first_failure_at).toLocaleString([], { timeZone: activeOrg?.timezone || "UTC" })}</span>
                          </div>
                        </div>

                        <div className="flex items-center justify-end">
                          <Link href={`/incidents/${inc.id}`}>
                            <Button size="sm" variant="secondary">
                              {t.viewDetails}
                            </Button>
                          </Link>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Tab 4: Maintenance Work */}
            {activeTab === "maintenance" && (
              <div>
                <div className="p-4 sm:p-5 border-b border-slate-100 flex items-center justify-between">
                  <div>
                    <h3 className="text-sm font-bold text-slate-900 tracking-tight">
                      {t.maintenance}
                    </h3>
                    <p className="text-xs text-slate-500 mt-0.5">
                      {isRTL ? `فعالیت‌های نگهداری ثبت‌شده برای گزارش ماهانه ${client.name}.` : `Maintenance activities recorded for monthly reports for ${client.name}.`}
                    </p>
                  </div>

                  <Link href={`/maintenance?client_id=${client.id}`}>
                    <Button size="sm" variant="outline" icon={<Wrench className="w-3.5 h-3.5" />}>
                      {t.viewAllMaintenance}
                    </Button>
                  </Link>
                </div>

                {maintenance.length === 0 ? (
                  <div className="p-8 text-center text-xs text-slate-400">
                    <Wrench className="w-8 h-8 text-slate-300 mx-auto mb-2" />
                    <p className="font-semibold text-slate-600">{t.noClientMaintenance}</p>
                    <Link href={`/maintenance?client_id=${client.id}`} className="mt-3 inline-block">
                      <Button size="sm" variant="outline">+ {t.addActivity}</Button>
                    </Link>
                  </div>
                ) : (
                  <div className="divide-y divide-slate-100">
                    {maintenance.map((m) => (
                      <div key={m.id} className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 hover:bg-slate-50/70 transition-colors">
                        <div>
                          <div className="font-bold text-sm text-slate-900">{m.title}</div>
                          <div className="flex items-center gap-3 text-xs text-slate-500 mt-1">
                            <Badge variant="neutral" className="capitalize text-[10px]">
                              {m.category.replace("_", " ")}
                            </Badge>
                            <span>•</span>
                            <span className="flex items-center gap-1">
                              <Calendar className="w-3 h-3 text-slate-400" />
                              {new Date(m.performed_at).toLocaleDateString([], { timeZone: activeOrg?.timezone || "UTC" })}
                            </span>
                            {m.duration_minutes && (
                              <>
                                <span>•</span>
                                <span className="flex items-center gap-1 font-mono">
                                  <Clock className="w-3 h-3 text-slate-400" />
                                  {m.duration_minutes}m
                                </span>
                              </>
                            )}
                          </div>
                        </div>

                        <div className="flex items-center justify-end">
                          <Link href={`/maintenance?client_id=${client.id}`}>
                            <Button size="sm" variant="secondary">
                              {t.viewDetails}
                            </Button>
                          </Link>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      ) : null}

      {/* Confirmation Modal */}
      <ConfirmModal
        isOpen={modalOpen}
        onClose={() => setModalOpen(false)}
        onConfirm={handleConfirmArchiveToggle}
        title={client?.status === "active" ? t.archiveClient : t.restoreClient}
        message={
          client?.status === "active"
            ? t.confirmArchiveClient
            : t.confirmRestoreClient
        }
        confirmLabel={client?.status === "active" ? t.archiveClient : t.restoreClient}
        variant={client?.status === "active" ? "danger" : "primary"}
        loading={actionLoading}
      />
    </AppLayout>
  );
}
