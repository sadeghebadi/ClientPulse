"use client";

import React, { useEffect, useState, useCallback, Suspense } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import {
  Globe,
  Plus,
  Search,
  ExternalLink,
  Archive,
  RotateCcw,
  Activity,
  ArrowRight,
  ArrowLeft,
  Building,
  Filter,
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
  client_id: string;
  client_name: string;
  name: string;
  url: string;
  type: string;
  environment: string;
  is_active: boolean;
  monitor_id?: string;
  monitor_status?: string;
  last_checked_at?: string;
  created_at: string;
}

interface ClientOption {
  id: string;
  name: string;
}

function AssetsContent() {
  const searchParams = useSearchParams();
  const initialClientID = searchParams.get("client_id") || "all";

  const { user, activeOrg, role } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const canManage = role === "owner" || role === "admin" || role === "member";

  const [assets, setAssets] = useState<Asset[]>([]);
  const [clients, setClients] = useState<ClientOption[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [clientFilter, setClientFilter] = useState(initialClientID);
  const [environmentFilter, setEnvironmentFilter] = useState("all");
  const [typeFilter, setTypeFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState("active");
  const [error, setError] = useState<any>(null);

  // Modal State for Archive / Restore
  const [targetAsset, setTargetAsset] = useState<Asset | null>(null);
  const [modalOpen, setModalOpen] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);

  // Load clients for filter dropdown
  useEffect(() => {
    if (!user || !activeOrg) return;
    apiFetch<{ clients: ClientOption[] }>("/api/v1/clients?status=active")
      .then((data) => setClients(data.clients || []))
      .catch(() => {});
  }, [user, activeOrg]);

  const reloadAssets = useCallback(() => {
    if (!user || !activeOrg) return;
    setLoading(true);
    setError(null);

    const params = new URLSearchParams();
    if (clientFilter && clientFilter !== "all") params.set("client_id", clientFilter);
    if (environmentFilter && environmentFilter !== "all") params.set("environment", environmentFilter);
    if (typeFilter && typeFilter !== "all") params.set("type", typeFilter);
    if (statusFilter === "active") params.set("is_active", "true");
    if (statusFilter === "archived") params.set("is_active", "false");

    apiFetch<{ assets: Asset[]; total: number }>(`/api/v1/assets?${params.toString()}`)
      .then((data) => {
        let list = data.assets || [];
        if (search.trim()) {
          const query = search.toLowerCase();
          list = list.filter(
            (a) =>
              a.name.toLowerCase().includes(query) ||
              a.url.toLowerCase().includes(query) ||
              (a.client_name && a.client_name.toLowerCase().includes(query))
          );
        }
        setAssets(list);
        setTotal(data.total || list.length);
      })
      .catch((err) => {
        setError(err);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [user, activeOrg, clientFilter, environmentFilter, typeFilter, statusFilter, search]);

  useEffect(() => {
    if (!user || !activeOrg) return;
    let isMounted = true;

    const params = new URLSearchParams();
    if (clientFilter && clientFilter !== "all") params.set("client_id", clientFilter);
    if (environmentFilter && environmentFilter !== "all") params.set("environment", environmentFilter);
    if (typeFilter && typeFilter !== "all") params.set("type", typeFilter);
    if (statusFilter === "active") params.set("is_active", "true");
    if (statusFilter === "archived") params.set("is_active", "false");

    apiFetch<{ assets: Asset[]; total: number }>(`/api/v1/assets?${params.toString()}`)
      .then((data) => {
        if (!isMounted) return;
        let list = data.assets || [];
        if (search.trim()) {
          const query = search.toLowerCase();
          list = list.filter(
            (a) =>
              a.name.toLowerCase().includes(query) ||
              a.url.toLowerCase().includes(query) ||
              (a.client_name && a.client_name.toLowerCase().includes(query))
          );
        }
        setAssets(list);
        setTotal(data.total || list.length);
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
  }, [user, activeOrg, clientFilter, environmentFilter, typeFilter, statusFilter, search]);

  const handleConfirmArchiveToggle = async () => {
    if (!targetAsset) return;
    try {
      setActionLoading(true);
      const isArchived = !targetAsset.is_active;
      const endpoint = isArchived
        ? `/api/v1/assets/${targetAsset.id}/restore`
        : `/api/v1/assets/${targetAsset.id}/archive`;

      await apiFetch(endpoint, { method: "POST" });
      toast.success(
        isArchived ? "Asset restored to active status" : "Asset archived successfully"
      );
      setModalOpen(false);
      setTargetAsset(null);
      reloadAssets();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update asset status");
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <AppLayout>
      <PageHeader
        title={t.assetsDirectory}
        description={t.assetsDirectoryDesc}
        icon={<Globe className="w-5 h-5" />}
        actions={
          canManage ? (
            <Link href="/assets/new">
              <Button size="sm" variant="primary" icon={<Plus className="w-4 h-4" />}>
                {t.createAsset}
              </Button>
            </Link>
          ) : null
        }
      />

      {/* Filter and Search Bar */}
      <div className="bg-white p-4 rounded-2xl border border-slate-200 shadow-sm mb-6 flex flex-col md:flex-row gap-3 items-center justify-between">
        <div className="relative flex-1 w-full">
          <Search
            className={`w-4 h-4 text-slate-400 absolute top-1/2 -translate-y-1/2 ${
              isRTL ? "right-3" : "left-3"
            }`}
          />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={isRTL ? "جستجوی وب‌سایت، URL یا مشتری..." : "Search assets by name, URL, or client..."}
            className={`w-full py-2 bg-slate-50 border border-slate-200 rounded-xl text-xs sm:text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all ${
              isRTL ? "pr-9 pl-4" : "pl-9 pr-4"
            }`}
          />
        </div>

        <div className="flex flex-wrap items-center gap-2 w-full md:w-auto">
          {/* Client Filter */}
          <select
            value={clientFilter}
            onChange={(e) => setClientFilter(e.target.value)}
            className="px-3 py-2 bg-slate-50 border border-slate-200 rounded-xl text-xs font-semibold text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">{t.allClients}</option>
            {clients.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>

          {/* Environment Filter */}
          <select
            value={environmentFilter}
            onChange={(e) => setEnvironmentFilter(e.target.value)}
            className="px-3 py-2 bg-slate-50 border border-slate-200 rounded-xl text-xs font-semibold text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">{t.allEnvironments}</option>
            <option value="production">{t.production}</option>
            <option value="staging">{t.staging}</option>
            <option value="development">{t.development}</option>
          </select>

          {/* Type Filter */}
          <select
            value={typeFilter}
            onChange={(e) => setTypeFilter(e.target.value)}
            className="px-3 py-2 bg-slate-50 border border-slate-200 rounded-xl text-xs font-semibold text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">{t.allTypes}</option>
            <option value="website">Website</option>
            <option value="api">API</option>
          </select>

          {/* Status Filter */}
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="px-3 py-2 bg-slate-50 border border-slate-200 rounded-xl text-xs font-semibold text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="active">{t.active}</option>
            <option value="archived">{t.archived}</option>
            <option value="all">{t.allStatuses}</option>
          </select>
        </div>
      </div>

      {error && <ErrorState error={error} onRetry={reloadAssets} className="mb-6" />}

      {/* Main Content Area */}
      {loading ? (
        <TableSkeleton rows={6} cols={6} />
      ) : assets.length === 0 ? (
        <EmptyState
          icon={<Globe className="w-6 h-6 text-slate-400" />}
          title={t.noAssetsFound}
          description={t.noAssetsDesc}
          actionHref="/assets/new"
          actionLabel={t.createAsset}
        />
      ) : (
        <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
          {/* Desktop Table View */}
          <div className="hidden md:block overflow-x-auto">
            <table className="w-full text-left text-sm" dir={isRTL ? "rtl" : "ltr"}>
              <thead className="bg-slate-50 text-slate-600 text-xs uppercase font-semibold border-b border-slate-200">
                <tr>
                  <th className="px-6 py-4">{t.assetName}</th>
                  <th className="px-6 py-4">{t.clients}</th>
                  <th className="px-6 py-4">{t.environment}</th>
                  <th className="px-6 py-4">{t.monitors}</th>
                  <th className="px-6 py-4">{t.status}</th>
                  <th className="px-6 py-4 text-right">{t.actions}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {assets.map((asset) => (
                  <tr key={asset.id} className="hover:bg-slate-50/70 transition-colors">
                    <td className="px-6 py-4">
                      <div className="flex items-center gap-2">
                        <Link
                          href={`/assets/${asset.id}`}
                          className="font-bold text-slate-900 hover:text-blue-600 transition-colors"
                        >
                          {asset.name}
                        </Link>
                        <Badge variant={asset.type} />
                      </div>
                      <a
                        href={asset.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 text-xs text-blue-600 hover:underline mt-1 font-mono"
                      >
                        <span className="truncate max-w-xs">{asset.url}</span>
                        <ExternalLink className="w-3 h-3 shrink-0" />
                      </a>
                    </td>

                    <td className="px-6 py-4">
                      <Link
                        href={`/clients/${asset.client_id}`}
                        className="text-xs font-semibold text-slate-700 hover:text-blue-600 flex items-center gap-1.5"
                      >
                        <Building className="w-3.5 h-3.5 text-slate-400" />
                        <span>{asset.client_name || "—"}</span>
                      </Link>
                    </td>

                    <td className="px-6 py-4">
                      <Badge variant={asset.environment} />
                    </td>

                    <td className="px-6 py-4">
                      {asset.monitor_id ? (
                        <div className="flex items-center gap-2">
                          <Badge variant={asset.monitor_status || "pending"} />
                          <Link
                            href={`/monitors/${asset.monitor_id}`}
                            className="text-xs font-semibold text-blue-600 hover:underline"
                          >
                            {t.viewDetails}
                          </Link>
                        </div>
                      ) : (
                        <Link
                          href={`/monitors/new?asset_id=${asset.id}`}
                          className="text-xs font-semibold text-emerald-600 hover:text-emerald-700"
                        >
                          + {t.createMonitor}
                        </Link>
                      )}
                    </td>

                    <td className="px-6 py-4">
                      <Badge variant={asset.is_active ? "active" : "archived"} />
                    </td>

                    <td className="px-6 py-4 text-end">
                      <div className="flex items-center justify-end gap-2">
                        {canManage && (
                          <button
                            onClick={() => {
                              setTargetAsset(asset);
                              setModalOpen(true);
                            }}
                            className={`p-1.5 rounded-lg border text-xs transition-colors ${
                              asset.is_active
                                ? "border-slate-200 text-slate-500 hover:text-rose-600 hover:bg-rose-50 hover:border-rose-200"
                                : "border-emerald-200 text-emerald-700 hover:bg-emerald-50"
                            }`}
                            title={asset.is_active ? t.archiveAsset : t.restoreAsset}
                            aria-label={asset.is_active ? t.archiveAsset : t.restoreAsset}
                          >
                            {asset.is_active ? (
                              <Archive className="w-3.5 h-3.5" />
                            ) : (
                              <RotateCcw className="w-3.5 h-3.5" />
                            )}
                          </button>
                        )}
                        <Link
                          href={`/assets/${asset.id}`}
                          className="inline-flex items-center gap-1 text-xs font-semibold text-blue-600 hover:text-blue-800 transition-colors ms-1"
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

          {/* Mobile Card View (< 768px) */}
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
                    <div className="flex items-center gap-2 mt-1">
                      <Badge variant={asset.environment} />
                      <Badge variant={asset.type} />
                    </div>
                  </div>
                  <Badge variant={asset.is_active ? "active" : "archived"} />
                </div>

                <a
                  href={asset.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1 text-xs text-blue-600 font-mono truncate"
                >
                  <span className="truncate">{asset.url}</span>
                  <ExternalLink className="w-3 h-3 shrink-0" />
                </a>

                <div className="flex items-center justify-between text-xs text-slate-600 pt-2 border-t border-slate-50">
                  <div className="flex items-center gap-1">
                    <Building className="w-3.5 h-3.5 text-slate-400" />
                    <span>{asset.client_name || "—"}</span>
                  </div>
                  {asset.monitor_status && <Badge variant={asset.monitor_status} />}
                </div>

                <div className="flex items-center justify-between pt-2">
                  {canManage ? (
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => {
                        setTargetAsset(asset);
                        setModalOpen(true);
                      }}
                      icon={asset.is_active ? <Archive className="w-3.5 h-3.5" /> : <RotateCcw className="w-3.5 h-3.5" />}
                    >
                      {asset.is_active ? t.archiveAsset : t.restoreAsset}
                    </Button>
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
        </div>
      )}

      {/* Confirmation Modal for Archive / Restore */}
      <ConfirmModal
        isOpen={modalOpen}
        onClose={() => {
          setModalOpen(false);
          setTargetAsset(null);
        }}
        onConfirm={handleConfirmArchiveToggle}
        title={targetAsset?.is_active ? t.archiveAsset : t.restoreAsset}
        message={
          targetAsset?.is_active
            ? t.confirmArchiveAsset
            : (isRTL ? "آیا این دارایی به حالت فعال بازگردانده شود؟ پایش‌های مرتبط مجدداً فعال خواهند شد." : "Restore this asset to active status? Associated monitoring jobs can be resumed.")
        }
        confirmLabel={targetAsset?.is_active ? t.archiveAsset : t.restoreAsset}
        variant={targetAsset?.is_active ? "danger" : "primary"}
        loading={actionLoading}
      />
    </AppLayout>
  );
}

export default function AssetsPage() {
  return (
    <Suspense fallback={<TableSkeleton rows={6} cols={5} />}>
      <AssetsContent />
    </Suspense>
  );
}
