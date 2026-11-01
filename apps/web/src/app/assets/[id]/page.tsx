"use client";

import React, { useEffect, useState, use, useCallback } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import {
  Globe,
  ExternalLink,
  Edit,
  Archive,
  RotateCcw,
  Activity,
  Building,
  Plus,
} from "lucide-react";
import { PageHeader } from "@/components/ui/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { ErrorState } from "@/components/ui/ErrorState";
import { ConfirmModal } from "@/components/ui/ConfirmModal";
import { useToast } from "@/components/ui/Toast";

interface AssetDetail {
  id: string;
  name: string;
  url: string;
  type: string;
  environment: string;
  description?: string;
  is_active: boolean;
  client_id: string;
  client_name: string;
  monitor_id?: string;
  monitor_status?: string;
  last_checked_at?: string;
  created_at: string;
}

export default function AssetDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = use(params);
  const router = useRouter();
  const { user, activeOrg } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const [asset, setAsset] = useState<AssetDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<any>(null);

  // Modal State
  const [modalOpen, setModalOpen] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);

  const reloadAsset = useCallback(() => {
    if (!user || !activeOrg) return;
    setLoading(true);
    setError(null);

    apiFetch<{ asset: AssetDetail }>(`/api/v1/assets/${resolvedParams.id}`)
      .then((data) => {
        setAsset(data.asset);
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

    apiFetch<{ asset: AssetDetail }>(`/api/v1/assets/${resolvedParams.id}`)
      .then((data) => {
        if (isMounted) {
          setAsset(data.asset);
          setLoading(false);
        }
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
    if (!asset) return;
    try {
      setActionLoading(true);
      const isArchived = !asset.is_active;
      const endpoint = isArchived
        ? `/api/v1/assets/${asset.id}/restore`
        : `/api/v1/assets/${asset.id}/archive`;

      await apiFetch(endpoint, { method: "POST" });
      toast.success(
        isArchived ? "Asset restored to active status" : "Asset archived successfully"
      );
      setModalOpen(false);
      reloadAsset();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Action failed");
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <AppLayout>
      <PageHeader
        title={asset?.name || t.assetDetails}
        description="Target asset details, environment specifications, and associated health monitor."
        icon={<Globe className="w-5 h-5" />}
        breadcrumbs={[
          { label: t.assets, href: "/assets" },
          { label: asset?.name || t.assetDetails },
        ]}
        badge={
          asset && (
            <div className="flex items-center gap-2">
              <Badge variant={asset.environment} />
              <Badge variant={asset.is_active ? "active" : "archived"} />
            </div>
          )
        }
        actions={
          asset && (
            <>
              <Link href={`/assets/${asset.id}/edit`}>
                <Button size="sm" variant="outline" icon={<Edit className="w-4 h-4" />}>
                  {t.editAsset}
                </Button>
              </Link>
              <Button
                size="sm"
                variant={asset.is_active ? "outline" : "primary"}
                onClick={() => setModalOpen(true)}
                icon={asset.is_active ? <Archive className="w-4 h-4" /> : <RotateCcw className="w-4 h-4" />}
              >
                {asset.is_active ? t.archiveAsset : t.restoreAsset}
              </Button>
              {asset.monitor_id ? (
                <Link href={`/monitors/${asset.monitor_id}`}>
                  <Button size="sm" variant="primary" icon={<Activity className="w-4 h-4" />}>
                    {t.monitorDetails}
                  </Button>
                </Link>
              ) : (
                <Link href={`/monitors/new?asset_id=${asset.id}`}>
                  <Button size="sm" variant="primary" icon={<Activity className="w-4 h-4" />}>
                    {t.createMonitor}
                  </Button>
                </Link>
              )}
            </>
          )
        }
      />

      {error && <ErrorState error={error} onRetry={reloadAsset} className="mb-6" />}

      {loading ? (
        <TableSkeleton rows={4} cols={4} />
      ) : asset ? (
        <div className="space-y-8">
          {/* Asset Details Card */}
          <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 sm:p-8">
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 text-xs text-slate-600 pb-6 border-b border-slate-100">
              <div>
                <div className="text-[10px] uppercase font-bold text-slate-400 mb-1">{t.clients}</div>
                <Link
                  href={`/clients/${asset.client_id}`}
                  className="text-sm font-bold text-blue-600 hover:underline flex items-center gap-1.5"
                >
                  <Building className="w-3.5 h-3.5 text-slate-400" />
                  <span>{asset.client_name}</span>
                </Link>
              </div>

              <div>
                <div className="text-[10px] uppercase font-bold text-slate-400 mb-1">{t.type}</div>
                <div className="text-sm font-bold text-slate-900 uppercase">{asset.type}</div>
              </div>

              <div>
                <div className="text-[10px] uppercase font-bold text-slate-400 mb-1">{t.environment}</div>
                <div className="text-sm font-bold text-slate-900 uppercase">{asset.environment}</div>
              </div>

              <div>
                <div className="text-[10px] uppercase font-bold text-slate-400 mb-1">{t.status}</div>
                <div>
                  <Badge variant={asset.is_active ? "active" : "archived"} />
                </div>
              </div>
            </div>

            <div className="pt-6">
              <div className="text-[10px] uppercase font-bold text-slate-400 mb-2">{t.url}</div>
              <div className="flex items-center gap-2 p-3 bg-slate-50 border border-slate-200 rounded-xl font-mono text-sm text-slate-900">
                <Globe className="w-4 h-4 text-blue-600 shrink-0" />
                <span className="truncate">{asset.url}</span>
                <a
                  href={asset.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="p-1 text-slate-400 hover:text-blue-600 ms-auto"
                >
                  <ExternalLink className="w-4 h-4" />
                </a>
              </div>
            </div>

            {asset.description && (
              <div className="pt-6">
                <div className="text-[10px] uppercase font-bold text-slate-400 mb-2">{t.description}</div>
                <p className="text-xs text-slate-600 bg-slate-50 p-4 rounded-xl border border-slate-100 leading-relaxed">
                  {asset.description}
                </p>
              </div>
            )}
          </div>

          {/* Associated Health Monitor Section */}
          <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 sm:p-8">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-6 border-b border-slate-100">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-xl bg-blue-50 text-blue-600 flex items-center justify-center shrink-0">
                  <Activity className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-base font-bold text-slate-900 tracking-tight">
                    {t.monitors}
                  </h2>
                  <p className="text-xs text-slate-500 mt-0.5">
                    Automated uptime verification and latency monitoring.
                  </p>
                </div>
              </div>

              {asset.monitor_id ? (
                <div className="flex items-center gap-3">
                  <Badge variant={asset.monitor_status || "pending"} />
                  <Link href={`/monitors/${asset.monitor_id}`}>
                    <Button size="sm" variant="primary">
                      {t.viewDetails}
                    </Button>
                  </Link>
                </div>
              ) : (
                <Link href={`/monitors/new?asset_id=${asset.id}`}>
                  <Button size="sm" variant="primary" icon={<Plus className="w-4 h-4" />}>
                    {t.createMonitor}
                  </Button>
                </Link>
              )}
            </div>

            {asset.monitor_id ? (
              <div className="pt-6 text-xs text-slate-600 flex items-center justify-between">
                <div>
                  <span className="font-semibold text-slate-800">Operational Health: </span>
                  <span>Currently monitored with automated HTTP/TLS checks.</span>
                </div>
                <Link
                  href={`/monitors/${asset.monitor_id}`}
                  className="font-semibold text-blue-600 hover:underline"
                >
                  Configure Check Intervals & Assertions &rarr;
                </Link>
              </div>
            ) : (
              <div className="pt-6 text-center py-6 text-xs text-slate-500">
                <p>No health monitor configured for this website yet.</p>
                <div className="mt-3">
                  <Link href={`/monitors/new?asset_id=${asset.id}`}>
                    <Button size="sm" variant="outline">
                      {t.createMonitor}
                    </Button>
                  </Link>
                </div>
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
        title={asset?.is_active ? t.archiveAsset : t.restoreAsset}
        message={
          asset?.is_active
            ? t.confirmArchiveAsset
            : "Restore this asset to active status? Associated monitoring jobs can be resumed."
        }
        confirmLabel={asset?.is_active ? t.archiveAsset : t.restoreAsset}
        variant={asset?.is_active ? "danger" : "primary"}
        loading={actionLoading}
      />
    </AppLayout>
  );
}
