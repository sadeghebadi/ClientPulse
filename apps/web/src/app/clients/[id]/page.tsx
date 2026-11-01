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

export default function ClientDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = use(params);
  const router = useRouter();
  const { user, activeOrg } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const [client, setClient] = useState<ClientDetail | null>(null);
  const [assets, setAssets] = useState<Asset[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<any>(null);

  // Archive / Restore Modal
  const [modalOpen, setModalOpen] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);

  const reloadClientData = useCallback(() => {
    if (!user || !activeOrg) return;
    setLoading(true);
    setError(null);

    Promise.all([
      apiFetch<{ client: ClientDetail }>(`/api/v1/clients/${resolvedParams.id}`),
      apiFetch<{ assets: Asset[] }>(`/api/v1/assets?client_id=${resolvedParams.id}`),
    ])
      .then(([clientRes, assetsRes]) => {
        setClient(clientRes.client);
        setAssets(assetsRes.assets || []);
      })
      .catch((err) => {
        setError(err);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [user, activeOrg, resolvedParams.id]);

  useEffect(() => {
    if (!user || !activeOrg) return;
    let isMounted = true;

    Promise.all([
      apiFetch<{ client: ClientDetail }>(`/api/v1/clients/${resolvedParams.id}`),
      apiFetch<{ assets: Asset[] }>(`/api/v1/assets?client_id=${resolvedParams.id}`),
    ])
      .then(([clientRes, assetsRes]) => {
        if (!isMounted) return;
        setClient(clientRes.client);
        setAssets(assetsRes.assets || []);
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
  }, [user, activeOrg, resolvedParams.id]);

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
        isArchived ? "Client restored to active status" : "Client archived successfully"
      );
      setModalOpen(false);
      reloadClientData();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Action failed");
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <AppLayout>
      <PageHeader
        title={client?.name || t.clientDetails}
        description={`Manage client profile, monitored websites and SLAs for ${client?.name || ""}.`}
        icon={<Users className="w-5 h-5" />}
        breadcrumbs={[
          { label: t.clients, href: "/clients" },
          { label: client?.name || t.clientDetails },
        ]}
        badge={client && <Badge variant={client.status} />}
        actions={
          client && (
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
          )
        }
      />

      {error && <ErrorState error={error} onRetry={reloadClientData} className="mb-6" />}

      {loading ? (
        <TableSkeleton rows={4} cols={5} />
      ) : client ? (
        <div className="space-y-8">
          {/* Client Overview Card */}
          <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 sm:p-8">
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-6 text-xs text-slate-600 pb-6 border-b border-slate-100">
              <div>
                <div className="text-[10px] uppercase font-bold text-slate-400 mb-1">{t.contactName}</div>
                <div className="text-sm font-bold text-slate-900">{client.contact_name || "—"}</div>
              </div>

              <div>
                <div className="text-[10px] uppercase font-bold text-slate-400 mb-1">{t.contactEmail}</div>
                <div className="text-sm font-bold text-slate-900 flex items-center gap-1.5">
                  {client.contact_email ? (
                    <>
                      <Mail className="w-3.5 h-3.5 text-slate-400" />
                      <a href={`mailto:${client.contact_email}`} className="text-blue-600 hover:underline">
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
                <div className="text-sm font-bold text-slate-900">
                  {client.website ? (
                    <a
                      href={client.website}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1 text-blue-600 hover:underline"
                    >
                      <span>{client.website}</span>
                      <ExternalLink className="w-3 h-3" />
                    </a>
                  ) : (
                    "—"
                  )}
                </div>
              </div>
            </div>

            {client.notes && (
              <div className="pt-6">
                <div className="text-[10px] uppercase font-bold text-slate-400 mb-1">{t.notes}</div>
                <p className="text-xs text-slate-600 bg-slate-50 p-4 rounded-xl border border-slate-100 leading-relaxed">
                  {client.notes}
                </p>
              </div>
            )}
          </div>

          {/* Registered Assets Section */}
          <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
            <div className="p-6 border-b border-slate-100 flex items-center justify-between">
              <div>
                <h2 className="text-base font-bold text-slate-900 tracking-tight flex items-center gap-2">
                  <Globe className="w-5 h-5 text-blue-600" />
                  <span>{t.assets}</span>
                </h2>
                <p className="text-xs text-slate-500 mt-0.5">
                  Websites and API endpoints monitored for {client.name}.
                </p>
              </div>

              <Link href={`/assets/new?client_id=${client.id}`}>
                <Button size="sm" variant="primary" icon={<Plus className="w-3.5 h-3.5" />}>
                  {t.createAsset}
                </Button>
              </Link>
            </div>

            {assets.length === 0 ? (
              <EmptyState
                icon={<Globe className="w-6 h-6 text-slate-400" />}
                title={t.noAssetsFound}
                description={t.noAssetsDesc}
                actionHref={`/assets/new?client_id=${client.id}`}
                actionLabel={t.createAsset}
              />
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm" dir={isRTL ? "rtl" : "ltr"}>
                  <thead className="bg-slate-50 text-slate-600 text-xs uppercase font-semibold border-b border-slate-200">
                    <tr>
                      <th className="px-6 py-4">{t.assetName}</th>
                      <th className="px-6 py-4">{t.url}</th>
                      <th className="px-6 py-4">{t.environment}</th>
                      <th className="px-6 py-4">{t.status}</th>
                      <th className="px-6 py-4 text-right">{t.actions}</th>
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
                        <td className="px-6 py-4 text-xs font-mono text-slate-600">
                          <a
                            href={asset.url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex items-center gap-1 text-blue-600 hover:underline"
                          >
                            <span>{asset.url}</span>
                            <ExternalLink className="w-3 h-3" />
                          </a>
                        </td>
                        <td className="px-6 py-4">
                          <Badge variant={asset.environment} />
                        </td>
                        <td className="px-6 py-4">
                          <Badge variant={asset.monitor_status || "pending"} />
                        </td>
                        <td className="px-6 py-4 text-right">
                          <div className="flex items-center justify-end gap-2">
                            {asset.monitor_id ? (
                              <Link
                                href={`/monitors/${asset.monitor_id}`}
                                className="text-xs font-semibold text-blue-600 hover:underline"
                              >
                                {t.monitorDetails}
                              </Link>
                            ) : (
                              <Link
                                href={`/monitors/new?asset_id=${asset.id}`}
                                className="text-xs font-semibold text-emerald-600 hover:underline"
                              >
                                + {t.createMonitor}
                              </Link>
                            )}
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
