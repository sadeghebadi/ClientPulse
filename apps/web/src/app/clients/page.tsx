"use client";

import React, { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import {
  Users,
  Plus,
  Search,
  ExternalLink,
  Archive,
  RotateCcw,
  Globe,
  Activity,
  ArrowRight,
  ArrowLeft,
} from "lucide-react";
import { PageHeader } from "@/components/ui/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { ConfirmModal } from "@/components/ui/ConfirmModal";
import { useToast } from "@/components/ui/Toast";

interface Client {
  id: string;
  name: string;
  contact_name?: string;
  contact_email?: string;
  website?: string;
  status: "active" | "archived";
  asset_count: number;
  monitor_count: number;
  created_at: string;
}

export default function ClientsPage() {
  const { user, activeOrg } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const [clients, setClients] = useState<Client[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("active");
  const [error, setError] = useState<any>(null);

  // Modal State for Archive / Restore
  const [targetClient, setTargetClient] = useState<Client | null>(null);
  const [modalOpen, setModalOpen] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);

  const reloadClients = useCallback(() => {
    if (!user || !activeOrg) return;
    setLoading(true);
    setError(null);

    const params = new URLSearchParams();
    if (search.trim()) params.set("search", search.trim());
    if (statusFilter && statusFilter !== "all") params.set("status", statusFilter);

    apiFetch<{ clients: Client[]; total: number }>(`/api/v1/clients?${params.toString()}`)
      .then((data) => {
        setClients(data.clients || []);
        setTotal(data.total || 0);
      })
      .catch((err) => {
        setError(err);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [user, activeOrg, search, statusFilter]);

  useEffect(() => {
    if (!user || !activeOrg) return;
    let isMounted = true;

    const params = new URLSearchParams();
    if (search.trim()) params.set("search", search.trim());
    if (statusFilter && statusFilter !== "all") params.set("status", statusFilter);

    apiFetch<{ clients: Client[]; total: number }>(`/api/v1/clients?${params.toString()}`)
      .then((data) => {
        if (!isMounted) return;
        setClients(data.clients || []);
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
  }, [user, activeOrg, search, statusFilter]);

  const handleConfirmArchiveToggle = async () => {
    if (!targetClient) return;
    try {
      setActionLoading(true);
      const isArchived = targetClient.status === "archived";
      const endpoint = isArchived
        ? `/api/v1/clients/${targetClient.id}/restore`
        : `/api/v1/clients/${targetClient.id}/archive`;

      await apiFetch(endpoint, { method: "POST" });
      toast.success(
        isArchived ? "Client restored to active status" : "Client archived successfully"
      );
      setModalOpen(false);
      setTargetClient(null);
      reloadClients();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to update client status");
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <AppLayout>
      <PageHeader
        title={t.clients}
        description={`Manage client accounts, registered websites, and monitoring configurations for ${activeOrg?.name || "your workspace"}.`}
        icon={<Users className="w-5 h-5" />}
        actions={
          <Link href="/clients/new">
            <Button size="sm" variant="primary" icon={<Plus className="w-4 h-4" />}>
              {t.createClient}
            </Button>
          </Link>
        }
      />

      {/* Filter & Search Bar */}
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
            placeholder={t.searchClients}
            className={`w-full py-2 bg-slate-50 border border-slate-200 rounded-xl text-xs sm:text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all ${
              isRTL ? "pr-9 pl-4" : "pl-9 pr-4"
            }`}
          />
        </div>

        <div className="flex items-center gap-2 w-full md:w-auto">
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="px-3 py-2 bg-slate-50 border border-slate-200 rounded-xl text-xs font-semibold text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="active">{t.active}</option>
            <option value="archived">{t.archived}</option>
            <option value="all">{t.allStatuses}</option>
          </select>
          <Button size="sm" variant="outline" onClick={reloadClients}>
            {t.filter}
          </Button>
        </div>
      </div>

      {error && <ErrorState error={error} onRetry={reloadClients} className="mb-6" />}

      {/* Main Table / Card Content */}
      {loading ? (
        <TableSkeleton rows={5} cols={6} />
      ) : clients.length === 0 ? (
        <EmptyState
          icon={<Users className="w-6 h-6 text-slate-400" />}
          title={t.noClientsFound}
          description={t.noClientsDesc}
          actionHref="/clients/new"
          actionLabel={t.createClient}
        />
      ) : (
        <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
          {/* Desktop Table View */}
          <div className="hidden md:block overflow-x-auto">
            <table className="w-full text-left text-sm" dir={isRTL ? "rtl" : "ltr"}>
              <thead className="bg-slate-50 text-slate-600 text-xs uppercase font-semibold border-b border-slate-200">
                <tr>
                  <th className="px-6 py-4">{t.clientName}</th>
                  <th className="px-6 py-4">{t.contactName}</th>
                  <th className="px-6 py-4">{t.website}</th>
                  <th className="px-6 py-4">{t.assets}</th>
                  <th className="px-6 py-4">{t.status}</th>
                  <th className="px-6 py-4 text-right">{t.actions}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {clients.map((client) => (
                  <tr key={client.id} className="hover:bg-slate-50/70 transition-colors">
                    <td className="px-6 py-4 font-bold text-slate-900">
                      <Link
                        href={`/clients/${client.id}`}
                        className="hover:text-blue-600 transition-colors"
                      >
                        {client.name}
                      </Link>
                    </td>

                    <td className="px-6 py-4 text-slate-600">
                      <div>{client.contact_name || "—"}</div>
                      {client.contact_email && (
                        <div className="text-xs text-slate-400">{client.contact_email}</div>
                      )}
                    </td>

                    <td className="px-6 py-4 text-slate-600">
                      {client.website ? (
                        <a
                          href={client.website}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="inline-flex items-center gap-1 text-blue-600 hover:underline text-xs"
                        >
                          <span>{client.website.replace(/^https?:\/\//, "")}</span>
                          <ExternalLink className="w-3 h-3" />
                        </a>
                      ) : (
                        "—"
                      )}
                    </td>

                    <td className="px-6 py-4">
                      <div className="flex items-center gap-3 text-xs text-slate-600">
                        <Link
                          href={`/assets?client_id=${client.id}`}
                          className="flex items-center gap-1 hover:text-blue-600"
                          title="View Assets"
                        >
                          <Globe className="w-3.5 h-3.5 text-slate-400" />
                          <span>{client.asset_count}</span>
                        </Link>
                        <span className="flex items-center gap-1" title="Monitors">
                          <Activity className="w-3.5 h-3.5 text-slate-400" />
                          <span>{client.monitor_count}</span>
                        </span>
                      </div>
                    </td>

                    <td className="px-6 py-4">
                      <Badge variant={client.status} />
                    </td>

                    <td className="px-6 py-4 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <button
                          onClick={() => {
                            setTargetClient(client);
                            setModalOpen(true);
                          }}
                          className={`p-1.5 rounded-lg border text-xs transition-colors ${
                            client.status === "active"
                              ? "border-slate-200 text-slate-500 hover:text-rose-600 hover:bg-rose-50 hover:border-rose-200"
                              : "border-emerald-200 text-emerald-700 hover:bg-emerald-50"
                          }`}
                          title={client.status === "active" ? t.archiveClient : t.restoreClient}
                          aria-label={client.status === "active" ? t.archiveClient : t.restoreClient}
                        >
                          {client.status === "active" ? (
                            <Archive className="w-3.5 h-3.5" />
                          ) : (
                            <RotateCcw className="w-3.5 h-3.5" />
                          )}
                        </button>
                        <Link
                          href={`/clients/${client.id}`}
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
            {clients.map((client) => (
              <div key={client.id} className="p-4 space-y-3">
                <div className="flex items-start justify-between gap-2">
                  <div>
                    <Link
                      href={`/clients/${client.id}`}
                      className="font-bold text-slate-900 text-sm hover:text-blue-600"
                    >
                      {client.name}
                    </Link>
                    {client.contact_name && (
                      <div className="text-xs text-slate-500 mt-0.5">{client.contact_name}</div>
                    )}
                  </div>
                  <Badge variant={client.status} />
                </div>

                <div className="flex items-center gap-4 text-xs text-slate-600 pt-1">
                  <span className="flex items-center gap-1">
                    <Globe className="w-3.5 h-3.5 text-slate-400" />
                    <span>{client.asset_count} Websites</span>
                  </span>
                  <span className="flex items-center gap-1">
                    <Activity className="w-3.5 h-3.5 text-slate-400" />
                    <span>{client.monitor_count} Monitors</span>
                  </span>
                </div>

                <div className="flex items-center justify-between pt-2 border-t border-slate-50">
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => {
                      setTargetClient(client);
                      setModalOpen(true);
                    }}
                    icon={client.status === "active" ? <Archive className="w-3.5 h-3.5" /> : <RotateCcw className="w-3.5 h-3.5" />}
                  >
                    {client.status === "active" ? t.archiveClient : t.restoreClient}
                  </Button>
                  <Link href={`/clients/${client.id}`}>
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

      {/* Confirmation Modal */}
      <ConfirmModal
        isOpen={modalOpen}
        onClose={() => {
          setModalOpen(false);
          setTargetClient(null);
        }}
        onConfirm={handleConfirmArchiveToggle}
        title={targetClient?.status === "active" ? t.archiveClient : t.restoreClient}
        message={
          targetClient?.status === "active"
            ? t.confirmArchiveClient
            : t.confirmRestoreClient
        }
        confirmLabel={targetClient?.status === "active" ? t.archiveClient : t.restoreClient}
        variant={targetClient?.status === "active" ? "danger" : "primary"}
        loading={actionLoading}
      />
    </AppLayout>
  );
}
