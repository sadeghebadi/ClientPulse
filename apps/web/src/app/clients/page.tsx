"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Header } from "@/components/Header";
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
  AlertCircle,
} from "lucide-react";

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

  const [clients, setClients] = useState<Client[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("active");
  const [error, setError] = useState<string | null>(null);

  const reloadClients = () => {
    setLoading(true);
    setError(null);
    const params = new URLSearchParams();
    if (search) params.set("search", search);
    if (statusFilter && statusFilter !== "all") params.set("status", statusFilter);

    apiFetch<{ clients: Client[]; total: number }>(`/api/v1/clients?${params.toString()}`)
      .then((data) => {
        setClients(data.clients || []);
        setTotal(data.total || 0);
      })
      .catch((err) => {
        setError(err?.message || "Failed to load clients");
      })
      .finally(() => {
        setLoading(false);
      });
  };

  useEffect(() => {
    if (!user || !activeOrg) return;
    let isMounted = true;
    const params = new URLSearchParams();
    if (search) params.set("search", search);
    if (statusFilter && statusFilter !== "all") params.set("status", statusFilter);

    apiFetch<{ clients: Client[]; total: number }>(`/api/v1/clients?${params.toString()}`)
      .then((data) => {
        if (isMounted) {
          setClients(data.clients || []);
          setTotal(data.total || 0);
          setLoading(false);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setError(err?.message || "Failed to load clients");
          setLoading(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user, activeOrg, statusFilter, search]);

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    reloadClients();
  };

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">
      <Header />

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 flex-1 w-full">
        {/* Page Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-8">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 tracking-tight flex items-center gap-2.5">
              <Users className="w-7 h-7 text-blue-600" />
              <span>{t.clients}</span>
            </h1>
            <p className="text-sm text-slate-600 mt-1">
              Manage client accounts, registered websites, and monitoring configurations for {activeOrg?.name}.
            </p>
          </div>

          <Link
            href="/clients/new"
            className="inline-flex items-center gap-2 px-4 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-semibold shadow-sm transition-colors"
          >
            <Plus className="w-4 h-4" />
            <span>{t.createClient}</span>
          </Link>
        </div>

        {/* Filter & Search Bar */}
        <div className="bg-white p-4 rounded-xl border border-slate-200 shadow-sm mb-6 flex flex-col md:flex-row gap-4 items-center justify-between">
          <form onSubmit={handleSearchSubmit} className="relative flex-1 w-full">
            <Search className={`w-4 h-4 text-slate-400 absolute top-1/2 -translate-y-1/2 ${isRTL ? "right-3" : "left-3"}`} />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t.searchClients}
              className={`w-full py-2 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all ${
                isRTL ? "pr-9 pl-4" : "pl-9 pr-4"
              }`}
            />
          </form>

          <div className="flex items-center gap-2 w-full md:w-auto">
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="px-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-xs font-semibold text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="active">{t.active}</option>
              <option value="archived">{t.archived}</option>
              <option value="all">{t.allStatuses}</option>
            </select>
            <button
              onClick={reloadClients}
              className="px-4 py-2 bg-slate-100 hover:bg-slate-200 text-slate-700 rounded-lg text-xs font-semibold transition-colors"
            >
              {t.searchClients ? "Filter" : "Apply"}
            </button>
          </div>
        </div>

        {/* Error State */}
        {error && (
          <div className="p-4 mb-6 rounded-lg bg-red-50 border border-red-200 text-red-700 text-sm flex items-center gap-2">
            <AlertCircle className="w-4 h-4 flex-shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* Clients Table / List */}
        {loading ? (
          <div className="p-12 text-center text-slate-500 text-sm bg-white rounded-xl border border-slate-200">
            {t.loading}
          </div>
        ) : clients.length === 0 ? (
          <div className="p-12 text-center bg-white rounded-xl border border-dashed border-slate-300">
            <Users className="w-12 h-12 text-slate-300 mx-auto mb-3" />
            <h3 className="text-base font-semibold text-slate-800">{t.noClientsFound}</h3>
            <p className="text-xs text-slate-500 mt-1 max-w-sm mx-auto mb-6">{t.noClientsDesc}</p>
            <Link
              href="/clients/new"
              className="inline-flex items-center gap-1.5 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-semibold transition-colors shadow-sm"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>{t.createClient}</span>
            </Link>
          </div>
        ) : (
          <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
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
                      <td className="px-6 py-4 font-semibold text-slate-900">
                        <Link href={`/clients/${client.id}`} className="hover:text-blue-600 transition-colors">
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
                          <span className="flex items-center gap-1" title="Assets">
                            <Globe className="w-3.5 h-3.5 text-slate-400" />
                            <span>{client.asset_count}</span>
                          </span>
                          <span className="flex items-center gap-1" title="Monitors">
                            <Activity className="w-3.5 h-3.5 text-slate-400" />
                            <span>{client.monitor_count}</span>
                          </span>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <span
                          className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold ${
                            client.status === "active"
                              ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                              : "bg-slate-100 text-slate-600 border border-slate-200"
                          }`}
                        >
                          {client.status === "active" ? t.active : t.archived}
                        </span>
                      </td>
                      <td className="px-6 py-4 text-right">
                        <Link
                          href={`/clients/${client.id}`}
                          className="inline-flex items-center gap-1 text-xs font-semibold text-blue-600 hover:text-blue-800 transition-colors"
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
          </div>
        )}
      </main>
    </div>
  );
}
