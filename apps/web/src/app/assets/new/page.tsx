"use client";

import React, { useState, useEffect, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch, humanizeApiError } from "@/lib/api";
import { Globe, AlertTriangle } from "lucide-react";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { ErrorState } from "@/components/ui/ErrorState";
import { useToast } from "@/components/ui/Toast";

interface ClientOption {
  id: string;
  name: string;
}

function NewAssetContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const initialClientID = searchParams.get("client_id") || "";

  const { user, activeOrg, role } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const canManage = role === "owner" || role === "admin" || role === "member";

  const [clients, setClients] = useState<ClientOption[]>([]);
  const [clientID, setClientID] = useState(initialClientID);
  const [name, setName] = useState("");
  const [url, setURL] = useState("");
  const [type, setType] = useState("website");
  const [environment, setEnvironment] = useState("production");
  const [description, setDescription] = useState("");

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<any>(null);
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
    if (loading) return; // Prevent duplicate submission

    if (!clientID) {
      setError(new Error("Please select a client"));
      return;
    }
    if (!name.trim()) {
      setError(new Error(t.assetName + " is required"));
      return;
    }
    if (!url.trim()) {
      setError(new Error(t.url + " is required"));
      return;
    }

    try {
      setLoading(true);
      setError(null);
      setDuplicateWarning(null);

      const res = await apiFetch<{
        asset: { id: string; name: string };
        warning?: string;
        warning_message?: string;
      }>("/api/v1/assets", {
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
        toast.warning(res.warning_message || t.duplicateUrlWarning);
      } else {
        toast.success(`Asset "${res.asset.name}" registered successfully`);
      }

      router.push(`/assets/${res.asset.id}`);
    } catch (err) {
      setError(err);
      setLoading(false);
    }
  };

  if (!canManage && !loading) {
    return (
      <AppLayout>
        <div className="max-w-md mx-auto p-8 text-center bg-white rounded-2xl border border-slate-200 mt-12">
          <AlertTriangle className="w-12 h-12 text-rose-500 mx-auto mb-3" />
          <h2 className="text-base font-bold text-slate-900">{isRTL ? "دسترسی محدود است" : "Access Restricted"}</h2>
          <p className="text-xs text-slate-500 mt-1 mb-4">
            {isRTL
              ? "حساب‌های فقط-خواندنی (Viewer) اجازه ایجاد دارایی را ندارند."
              : "Viewer accounts do not have permission to add websites or API assets."}
          </p>
          <Link href="/assets">
            <Button size="sm" variant="outline">{t.assets}</Button>
          </Link>
        </div>
      </AppLayout>
    );
  }

  return (
    <AppLayout>
      <div className="max-w-3xl mx-auto">
        <PageHeader
          title={t.createAsset}
          description="Register a public website or API endpoint to configure automated health checks."
          icon={<Globe className="w-5 h-5" />}
          breadcrumbs={[
            { label: t.assets, href: "/assets" },
            { label: t.createAsset },
          ]}
        />

        <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden p-6 sm:p-8">
          <form onSubmit={handleSubmit} className="space-y-6">
            {error && <ErrorState error={error} className="mb-4" />}

            {duplicateWarning && (
              <div className="p-4 rounded-xl bg-amber-50 border border-amber-200 text-amber-800 text-xs flex items-center gap-2.5">
                <AlertTriangle className="w-4 h-4 shrink-0 text-amber-600" />
                <span>{duplicateWarning}</span>
              </div>
            )}

            <div>
              <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                {t.clients} <span className="text-red-500">*</span>
              </label>
              <select
                name="client_id"
                required
                disabled={loading}
                value={clientID}
                onChange={(e) => setClientID(e.target.value)}
                className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
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
              <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                {t.assetName} <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                name="name"
                required
                disabled={loading}
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. Acme Production Store"
                className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
              />
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                {t.url} <span className="text-red-500">*</span>
              </label>
              <input
                type="url"
                name="url"
                required
                disabled={loading}
                value={url}
                onChange={(e) => setURL(e.target.value)}
                placeholder="https://acme-store.com"
                className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all font-mono disabled:opacity-50 text-start"
                dir="ltr"
              />
              <p className="text-[11px] text-slate-500 mt-1.5">
                Must be an accessible HTTP or HTTPS URL (ports 80 or 443). Private IP addresses, localhost, and internal services are restricted for security.
              </p>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                  {t.type}
                </label>
                <select
                  name="type"
                  disabled={loading}
                  value={type}
                  onChange={(e) => setType(e.target.value)}
                  className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                >
                  <option value="website">Website (HTML / Webpage)</option>
                  <option value="api">API Endpoint (JSON)</option>
                </select>
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                  {t.environment}
                </label>
                <select
                  name="environment"
                  disabled={loading}
                  value={environment}
                  onChange={(e) => setEnvironment(e.target.value)}
                  className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                >
                  <option value="production">{t.production}</option>
                  <option value="staging">{t.staging}</option>
                  <option value="development">{t.development}</option>
                </select>
              </div>
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                {t.description}
              </label>
              <textarea
                rows={3}
                disabled={loading}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Details regarding infrastructure or hosting provider..."
                className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
              />
            </div>

            <div className="pt-4 flex items-center justify-end gap-3 border-t border-slate-100">
              <Link href="/assets">
                <Button variant="ghost" size="sm" disabled={loading}>
                  {t.cancel}
                </Button>
              </Link>
              <Button
                type="submit"
                variant="primary"
                size="md"
                loading={loading}
                disabled={clients.length === 0}
              >
                {t.createAsset}
              </Button>
            </div>
          </form>
        </div>
      </div>
    </AppLayout>
  );
}

export default function NewAssetPage() {
  return (
    <Suspense fallback={<div className="p-12 text-center text-sm text-slate-500">Loading...</div>}>
      <NewAssetContent />
    </Suspense>
  );
}
