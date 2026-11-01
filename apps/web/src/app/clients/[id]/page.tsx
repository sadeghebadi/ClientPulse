"use client";

import React, { useEffect, useState, use } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Header } from "@/components/Header";
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
  AlertCircle,
  Clock,
  CheckCircle,
  XCircle,
  PauseCircle,
} from "lucide-react";

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

  const [client, setClient] = useState<ClientDetail | null>(null);
  const [assets, setAssets] = useState<Asset[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState(false);

  const reloadClientData = () => {
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
        setError(err?.message || "Failed to load client details");
      })
      .finally(() => {
        setLoading(false);
      });
  };

  useEffect(() => {
    if (!user || !activeOrg) return;
    let isMounted = true;

    Promise.all([
      apiFetch<{ client: ClientDetail }>(`/api/v1/clients/${resolvedParams.id}`),
      apiFetch<{ assets: Asset[] }>(`/api/v1/assets?client_id=${resolvedParams.id}`),
    ])
      .then(([clientRes, assetsRes]) => {
        if (isMounted) {
          setClient(clientRes.client);
          setAssets(assetsRes.assets || []);
          setLoading(false);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setError(err?.message || "Failed to load client details");
          setLoading(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user, activeOrg, resolvedParams.id]);

  const handleArchiveToggle = async () => {
    if (!client) return;
    const isArchived = client.status === "archived";
    const confirmMsg = isArchived ? t.confirmRestoreClient : t.confirmArchiveClient;
    if (!confirm(confirmMsg)) return;

    try {
      setActionLoading(true);
      const endpoint = isArchived
        ? `/api/v1/clients/${client.id}/restore`
        : `/api/v1/clients/${client.id}/archive`;
      await apiFetch(endpoint, { method: "POST" });
      reloadClientData();
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
        {/* Breadcrumb */}
        <div className="mb-6">
          <Link
            href="/clients"
            className="inline-flex items-center gap-1.5 text-xs font-semibold text-slate-500 hover:text-slate-800 transition-colors"
          >
            {isRTL ? <ArrowRight className="w-3.5 h-3.5" /> : <ArrowLeft className="w-3.5 h-3.5" />}
            <span>{t.clients}</span>
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
        ) : client ? (
          <div className="space-y-8">
            {/* Client Header Card */}
            <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 sm:p-8 flex flex-col md:flex-row md:items-center justify-between gap-6">
              <div>
                <div className="flex items-center gap-3">
                  <h1 className="text-2xl font-bold text-slate-900 tracking-tight">{client.name}</h1>
                  <span
                    className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold ${
                      client.status === "active"
                        ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                        : "bg-slate-100 text-slate-600 border border-slate-200"
                    }`}
                  >
                    {client.status === "active" ? t.active : t.archived}
                  </span>
                </div>

                <div className="flex flex-wrap items-center gap-4 mt-3 text-xs text-slate-600">
                  {client.contact_name && (
                    <span className="flex items-center gap-1.5">
                      <Users className="w-3.5 h-3.5 text-slate-400" />
                      <span>{client.contact_name}</span>
                    </span>
                  )}
                  {client.contact_email && (
                    <span className="text-slate-400">{client.contact_email}</span>
                  )}
                  {client.website && (
                    <a
                      href={client.website}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1 text-blue-600 hover:underline"
                    >
                      <Globe className="w-3.5 h-3.5" />
                      <span>{client.website}</span>
                      <ExternalLink className="w-3 h-3" />
                    </a>
                  )}
                </div>

                {client.notes && (
                  <p className="mt-4 text-xs text-slate-500 bg-slate-50 p-3 rounded-lg border border-slate-100 max-w-2xl">
                    {client.notes}
                  </p>
                )}
              </div>

              {/* Action Buttons */}
              <div className="flex items-center gap-3">
                <Link
                  href={`/clients/${client.id}/edit`}
                  className="px-3.5 py-2 bg-white border border-slate-200 hover:bg-slate-50 text-slate-700 rounded-lg text-xs font-semibold transition-colors flex items-center gap-1.5"
                >
                  <Edit className="w-3.5 h-3.5" />
                  <span>{t.editClient}</span>
                </Link>

                <button
                  onClick={handleArchiveToggle}
                  disabled={actionLoading}
                  className={`px-3.5 py-2 rounded-lg text-xs font-semibold transition-colors flex items-center gap-1.5 border ${
                    client.status === "active"
                      ? "bg-amber-50 border-amber-200 text-amber-700 hover:bg-amber-100"
                      : "bg-emerald-50 border-emerald-200 text-emerald-700 hover:bg-emerald-100"
                  }`}
                >
                  {client.status === "active" ? (
                    <>
                      <Archive className="w-3.5 h-3.5" />
                      <span>{t.archiveClient}</span>
                    </>
                  ) : (
                    <>
                      <RotateCcw className="w-3.5 h-3.5" />
                      <span>{t.restoreClient}</span>
                    </>
                  )}
                </button>
              </div>
            </div>

            {/* Registered Assets & Websites Section */}
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

                <Link
                  href={`/assets/new?client_id=${client.id}`}
                  className="inline-flex items-center gap-1.5 px-3.5 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-semibold shadow-sm transition-colors"
                >
                  <Plus className="w-3.5 h-3.5" />
                  <span>{t.createAsset}</span>
                </Link>
              </div>

              {assets.length === 0 ? (
                <div className="p-8 text-center">
                  <Globe className="w-10 h-10 text-slate-300 mx-auto mb-2" />
                  <p className="text-xs font-semibold text-slate-700">{t.noAssetsFound}</p>
                  <p className="text-xs text-slate-400 mt-1 max-w-sm mx-auto mb-4">{t.noAssetsDesc}</p>
                  <Link
                    href={`/assets/new?client_id=${client.id}`}
                    className="inline-flex items-center gap-1.5 px-3.5 py-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-semibold transition-colors"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    <span>{t.createAsset}</span>
                  </Link>
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <thead className="bg-slate-50 text-slate-600 text-xs uppercase font-semibold border-b border-slate-200">
                      <tr>
                        <th className="px-6 py-3.5">{t.assetName}</th>
                        <th className="px-6 py-3.5">{t.url}</th>
                        <th className="px-6 py-3.5">{t.environment}</th>
                        <th className="px-6 py-3.5">{t.status}</th>
                        <th className="px-6 py-3.5 text-right">{t.actions}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                      {assets.map((asset) => (
                        <tr key={asset.id} className="hover:bg-slate-50/70 transition-colors">
                          <td className="px-6 py-4 font-semibold text-slate-900">
                            <Link href={`/assets/${asset.id}`} className="hover:text-blue-600 transition-colors">
                              {asset.name}
                            </Link>
                            <span className="ml-2 text-[10px] uppercase font-bold px-1.5 py-0.5 rounded bg-slate-100 text-slate-600">
                              {asset.type}
                            </span>
                          </td>
                          <td className="px-6 py-4 text-slate-600 text-xs">
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
                            <span className="text-xs uppercase font-semibold text-slate-600">
                              {asset.environment}
                            </span>
                          </td>
                          <td className="px-6 py-4">
                            {asset.monitor_status === "up" && (
                              <span className="inline-flex items-center gap-1 text-xs font-semibold text-emerald-600 bg-emerald-50 px-2 py-0.5 rounded-full border border-emerald-200">
                                <CheckCircle className="w-3 h-3" />
                                <span>{t.statusUp}</span>
                              </span>
                            )}
                            {asset.monitor_status === "down" && (
                              <span className="inline-flex items-center gap-1 text-xs font-semibold text-rose-600 bg-rose-50 px-2 py-0.5 rounded-full border border-rose-200">
                                <XCircle className="w-3 h-3" />
                                <span>{t.statusDown}</span>
                              </span>
                            )}
                            {asset.monitor_status === "paused" && (
                              <span className="inline-flex items-center gap-1 text-xs font-semibold text-slate-600 bg-slate-100 px-2 py-0.5 rounded-full border border-slate-200">
                                <PauseCircle className="w-3 h-3" />
                                <span>{t.statusPaused}</span>
                              </span>
                            )}
                            {(!asset.monitor_status || asset.monitor_status === "pending") && (
                              <span className="inline-flex items-center gap-1 text-xs font-semibold text-amber-600 bg-amber-50 px-2 py-0.5 rounded-full border border-amber-200">
                                <Clock className="w-3 h-3" />
                                <span>{t.statusPending}</span>
                              </span>
                            )}
                          </td>
                          <td className="px-6 py-4 text-right">
                            <div className="flex items-center justify-end gap-2">
                              {asset.monitor_id ? (
                                <Link
                                  href={`/monitors/${asset.monitor_id}`}
                                  className="text-xs font-semibold text-blue-600 hover:text-blue-800"
                                >
                                  {t.monitorDetails}
                                </Link>
                              ) : (
                                <Link
                                  href={`/monitors/new?asset_id=${asset.id}`}
                                  className="text-xs font-semibold text-emerald-600 hover:text-emerald-800"
                                >
                                  {t.createMonitor}
                                </Link>
                              )}
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
      </main>
    </div>
  );
}
