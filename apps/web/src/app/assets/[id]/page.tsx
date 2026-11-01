"use client";

import React, { useEffect, useState, use } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Header } from "@/components/Header";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import {
  Globe,
  ExternalLink,
  Edit,
  Archive,
  RotateCcw,
  ArrowLeft,
  ArrowRight,
  Activity,
  Plus,
  AlertCircle,
  CheckCircle,
  XCircle,
  Clock,
  PauseCircle,
  Building,
} from "lucide-react";

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

  const [asset, setAsset] = useState<AssetDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);

  const reloadAsset = () => {
    setLoading(true);
    setError(null);
    apiFetch<{ asset: AssetDetail }>(`/api/v1/assets/${resolvedParams.id}`)
      .then((data) => {
        setAsset(data.asset);
      })
      .catch((err) => {
        setError(err?.message || "Failed to load asset");
      })
      .finally(() => {
        setLoading(false);
      });
  };

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
          setError(err?.message || "Failed to load asset");
          setLoading(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user, activeOrg, resolvedParams.id]);

  const handleArchiveToggle = async () => {
    if (!asset) return;
    const isArchived = !asset.is_active;
    const confirmMsg = isArchived ? "Restore this asset to active status?" : t.confirmArchiveAsset;
    if (!confirm(confirmMsg)) return;

    try {
      setActionLoading(true);
      const endpoint = isArchived
        ? `/api/v1/assets/${asset.id}/restore`
        : `/api/v1/assets/${asset.id}/archive`;
      await apiFetch(endpoint, { method: "POST" });
      reloadAsset();
    } catch (err: any) {
      alert(err?.message || "Action failed");
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">
      <Header />

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 flex-1 w-full">
        <div className="mb-6">
          <Link
            href={asset?.client_id ? `/clients/${asset.client_id}` : "/clients"}
            className="inline-flex items-center gap-1.5 text-xs font-semibold text-slate-500 hover:text-slate-800 transition-colors"
          >
            {isRTL ? <ArrowRight className="w-3.5 h-3.5" /> : <ArrowLeft className="w-3.5 h-3.5" />}
            <span>{asset?.client_name || t.clientDetails}</span>
          </Link>
        </div>

        {error && (
          <div className="p-4 mb-6 rounded-lg bg-red-50 border border-red-200 text-red-700 text-xs flex items-center gap-2">
            <AlertCircle className="w-4 h-4 flex-shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {loading ? (
          <div className="p-12 text-center text-slate-500 text-sm bg-white rounded-xl border border-slate-200">
            {t.loading}
          </div>
        ) : asset ? (
          <div className="space-y-8">
            {/* Asset Header Card */}
            <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 sm:p-8 flex flex-col md:flex-row md:items-center justify-between gap-6">
              <div>
                <div className="flex items-center gap-3">
                  <h1 className="text-2xl font-bold text-slate-900 tracking-tight">{asset.name}</h1>
                  <span
                    className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold ${
                      asset.is_active
                        ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                        : "bg-slate-100 text-slate-600 border border-slate-200"
                    }`}
                  >
                    {asset.is_active ? t.active : t.archived}
                  </span>
                  <span className="text-xs uppercase font-bold px-2 py-0.5 rounded bg-blue-50 text-blue-700 border border-blue-200">
                    {asset.environment}
                  </span>
                </div>

                <div className="flex flex-wrap items-center gap-4 mt-3 text-xs text-slate-600">
                  <span className="flex items-center gap-1.5">
                    <Building className="w-3.5 h-3.5 text-slate-400" />
                    <span>Client: </span>
                    <Link href={`/clients/${asset.client_id}`} className="font-semibold text-blue-600 hover:underline">
                      {asset.client_name}
                    </Link>
                  </span>

                  <a
                    href={asset.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1 text-blue-600 hover:underline"
                  >
                    <Globe className="w-3.5 h-3.5" />
                    <span>{asset.url}</span>
                    <ExternalLink className="w-3 h-3" />
                  </a>
                </div>

                {asset.description && (
                  <p className="mt-4 text-xs text-slate-500 bg-slate-50 p-3 rounded-lg border border-slate-100 max-w-2xl">
                    {asset.description}
                  </p>
                )}
              </div>

              <div className="flex items-center gap-3">
                <Link
                  href={`/assets/${asset.id}/edit`}
                  className="px-3.5 py-2 bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 rounded-lg text-xs font-semibold transition-colors flex items-center gap-1.5"
                >
                  <Edit className="w-3.5 h-3.5" />
                  <span>{t.editAsset}</span>
                </Link>

                <button
                  onClick={handleArchiveToggle}
                  disabled={actionLoading}
                  className={`px-3.5 py-2 rounded-lg text-xs font-semibold transition-colors flex items-center gap-1.5 border ${
                    asset.is_active
                      ? "bg-amber-50 border-amber-200 text-amber-700 hover:bg-amber-100"
                      : "bg-emerald-50 border-emerald-200 text-emerald-700 hover:bg-emerald-100"
                  }`}
                >
                  {asset.is_active ? (
                    <>
                      <Archive className="w-3.5 h-3.5" />
                      <span>{t.archiveAsset}</span>
                    </>
                  ) : (
                    <>
                      <RotateCcw className="w-3.5 h-3.5" />
                      <span>{t.restoreAsset}</span>
                    </>
                  )}
                </button>
              </div>
            </div>

            {/* Health Monitor Section */}
            <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 sm:p-8">
              <div className="flex items-center justify-between pb-6 border-b border-slate-100">
                <div>
                  <h2 className="text-base font-bold text-slate-900 tracking-tight flex items-center gap-2">
                    <Activity className="w-5 h-5 text-blue-600" />
                    <span>{t.monitors}</span>
                  </h2>
                  <p className="text-xs text-slate-500 mt-0.5">
                    Automated scheduled checks and uptime status for this website.
                  </p>
                </div>

                {asset.monitor_id ? (
                  <Link
                    href={`/monitors/${asset.monitor_id}`}
                    className="inline-flex items-center gap-1.5 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-semibold shadow-sm transition-colors"
                  >
                    <span>{t.viewDetails}</span>
                    {isRTL ? <ArrowLeft className="w-3.5 h-3.5" /> : <ArrowRight className="w-3.5 h-3.5" />}
                  </Link>
                ) : (
                  <Link
                    href={`/monitors/new?asset_id=${asset.id}`}
                    className="inline-flex items-center gap-1.5 px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg text-xs font-semibold shadow-sm transition-colors"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    <span>{t.createMonitor}</span>
                  </Link>
                )}
              </div>

              <div className="pt-6">
                {asset.monitor_id ? (
                  <div className="flex items-center justify-between bg-slate-50 p-4 rounded-xl border border-slate-200">
                    <div className="flex items-center gap-3">
                      {asset.monitor_status === "up" && (
                        <CheckCircle className="w-5 h-5 text-emerald-500" />
                      )}
                      {asset.monitor_status === "down" && (
                        <XCircle className="w-5 h-5 text-rose-500" />
                      )}
                      {asset.monitor_status === "paused" && (
                        <PauseCircle className="w-5 h-5 text-slate-500" />
                      )}
                      {(!asset.monitor_status || asset.monitor_status === "pending") && (
                        <Clock className="w-5 h-5 text-amber-500" />
                      )}

                      <div>
                        <div className="text-sm font-bold text-slate-900 capitalize">
                          {asset.monitor_status === "up" && t.statusUp}
                          {asset.monitor_status === "down" && t.statusDown}
                          {asset.monitor_status === "paused" && t.statusPaused}
                          {(!asset.monitor_status || asset.monitor_status === "pending") && t.statusPending}
                        </div>
                        {asset.last_checked_at && (
                          <div className="text-xs text-slate-500 mt-0.5">
                            {t.latestCheck}: {new Date(asset.last_checked_at).toLocaleString()}
                          </div>
                        )}
                      </div>
                    </div>

                    <Link
                      href={`/monitors/${asset.monitor_id}`}
                      className="text-xs font-semibold text-blue-600 hover:underline"
                    >
                      {t.checkHistory} →
                    </Link>
                  </div>
                ) : (
                  <div className="text-center py-6">
                    <p className="text-xs text-slate-500 mb-3">
                      No automated health monitor has been configured for this website yet.
                    </p>
                    <Link
                      href={`/monitors/new?asset_id=${asset.id}`}
                      className="inline-flex items-center gap-1.5 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-semibold shadow-sm transition-colors"
                    >
                      <Plus className="w-3.5 h-3.5" />
                      <span>{t.createMonitor}</span>
                    </Link>
                  </div>
                )}
              </div>
            </div>
          </div>
        ) : null}
      </main>
    </div>
  );
}
