"use client";

import React, { useState, useEffect, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { Header } from "@/components/Header";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import { Globe, ArrowLeft, ArrowRight, AlertCircle, AlertTriangle } from "lucide-react";

interface ClientOption {
  id: string;
  name: string;
}

function NewAssetContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const initialClientID = searchParams.get("client_id") || "";

  const { user, activeOrg } = useAuth();
  const { t, isRTL } = useI18n();

  const [clients, setClients] = useState<ClientOption[]>([]);
  const [clientID, setClientID] = useState(initialClientID);
  const [name, setName] = useState("");
  const [url, setURL] = useState("");
  const [type, setType] = useState("website");
  const [environment, setEnvironment] = useState("production");
  const [description, setDescription] = useState("");

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [duplicateWarning, setDuplicateWarning] = useState<string | null>(null);

  useEffect(() => {
    if (!user || !activeOrg) return;
    let isMounted = true;

    apiFetch<{ clients: ClientOption[] }>("/api/v1/clients?status=active")
      .then((data) => {
        if (isMounted) {
          const list = data.clients || [];
          setClients(list);
          setClientID((prev) => (prev ? prev : list.length > 0 ? list[0].id : ""));
        }
      })
      .catch((err) => {
        if (isMounted) {
          console.error("Failed to load clients", err);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user, activeOrg]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!clientID) {
      setError("Please select a client");
      return;
    }
    if (!name.trim()) {
      setError("Asset name is required");
      return;
    }
    if (!url.trim()) {
      setError("URL is required");
      return;
    }

    try {
      setLoading(true);
      setError(null);
      setDuplicateWarning(null);

      const res = await apiFetch<{ asset: { id: string }; warning?: string; warning_message?: string }>("/api/v1/assets", {
        method: "POST",
        body: JSON.stringify({
          client_id: clientID,
          name: name.trim(),
          url: url.trim(),
          type,
          environment,
          description: description.trim() || undefined,
        }),
      });

      if (res.warning === "DUPLICATE_URL") {
        setDuplicateWarning(res.warning_message || t.duplicateUrlWarning);
      }

      router.push(`/assets/${res.asset.id}`);
    } catch (err: any) {
      setError(err?.message || "Failed to create asset");
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">
      <Header />

      <main className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 py-8 flex-1 w-full">
        <div className="mb-6">
          <Link
            href={clientID ? `/clients/${clientID}` : "/clients"}
            className="inline-flex items-center gap-1.5 text-xs font-semibold text-slate-500 hover:text-slate-800 transition-colors"
          >
            {isRTL ? <ArrowRight className="w-3.5 h-3.5" /> : <ArrowLeft className="w-3.5 h-3.5" />}
            <span>{t.clientDetails}</span>
          </Link>
        </div>

        <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
          <div className="p-6 sm:p-8 border-b border-slate-100">
            <h1 className="text-xl font-bold text-slate-900 tracking-tight flex items-center gap-2">
              <Globe className="w-6 h-6 text-blue-600" />
              <span>{t.createAsset}</span>
            </h1>
            <p className="text-xs text-slate-500 mt-1">
              Register a public website or API endpoint to configure automated health checks.
            </p>
          </div>

          <form onSubmit={handleSubmit} className="p-6 sm:p-8 space-y-6">
            {error && (
              <div className="p-4 rounded-lg bg-red-50 border border-red-200 text-red-700 text-xs flex items-center gap-2">
                <AlertCircle className="w-4 h-4 flex-shrink-0" />
                <span>{error}</span>
              </div>
            )}

            {duplicateWarning && (
              <div className="p-4 rounded-lg bg-amber-50 border border-amber-200 text-amber-700 text-xs flex items-center gap-2">
                <AlertTriangle className="w-4 h-4 flex-shrink-0" />
                <span>{duplicateWarning}</span>
              </div>
            )}

            <div>
              <label className="block text-xs font-semibold text-slate-700 uppercase mb-2">
                {t.clients} <span className="text-red-500">*</span>
              </label>
              <select
                required
                value={clientID}
                onChange={(e) => setClientID(e.target.value)}
                className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all"
              >
                {clients.length === 0 ? (
                  <option value="">No clients available. Please create a client first.</option>
                ) : (
                  clients.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))
                )}
              </select>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 uppercase mb-2">
                {t.assetName} <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                required
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. Acme Production Store"
                className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 uppercase mb-2">
                {t.url} <span className="text-red-500">*</span>
              </label>
              <input
                type="url"
                required
                value={url}
                onChange={(e) => setURL(e.target.value)}
                placeholder="https://acme-store.com"
                className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all"
              />
              <p className="text-[11px] text-slate-500 mt-1.5">
                Must be an accessible HTTP or HTTPS URL (ports 80 or 443).
              </p>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-2">
                  {t.type}
                </label>
                <select
                  value={type}
                  onChange={(e) => setType(e.target.value)}
                  className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="website">Website (HTML / Webpage)</option>
                  <option value="api">API Endpoint (JSON)</option>
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-2">
                  {t.environment}
                </label>
                <select
                  value={environment}
                  onChange={(e) => setEnvironment(e.target.value)}
                  className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="production">{t.production}</option>
                  <option value="staging">{t.staging}</option>
                  <option value="development">{t.development}</option>
                </select>
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 uppercase mb-2">
                {t.description}
              </label>
              <textarea
                rows={3}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Details regarding infrastructure or hosting provider..."
                className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all"
              />
            </div>

            <div className="pt-4 flex items-center justify-end gap-3 border-t border-slate-100">
              <Link
                href="/clients"
                className="px-4 py-2 text-xs font-semibold text-slate-600 hover:text-slate-800 transition-colors"
              >
                {t.cancel}
              </Link>
              <button
                type="submit"
                disabled={loading || clients.length === 0}
                className="px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-xs font-semibold shadow-sm transition-colors"
              >
                {loading ? t.loading : t.createAsset}
              </button>
            </div>
          </form>
        </div>
      </main>
    </div>
  );
}

export default function NewAssetPage() {
  return (
    <Suspense fallback={<div className="p-12 text-center text-sm text-slate-500">Loading...</div>}>
      <NewAssetContent />
    </Suspense>
  );
}
