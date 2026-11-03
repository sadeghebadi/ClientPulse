"use client";

import React, { useState, useEffect, use } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import { Globe, AlertTriangle } from "lucide-react";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { ErrorState } from "@/components/ui/ErrorState";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { useToast } from "@/components/ui/Toast";

interface AssetResponse {
  asset: {
    id: string;
    name: string;
    url: string;
    type: string;
    environment: string;
    description?: string;
  };
}

export default function EditAssetPage({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = use(params);
  const router = useRouter();
  const { user, activeOrg, role } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const canManage = role === "owner" || role === "admin" || role === "member";

  const [name, setName] = useState("");
  const [url, setURL] = useState("");
  const [type, setType] = useState("website");
  const [environment, setEnvironment] = useState("production");
  const [description, setDescription] = useState("");

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<any>(null);
  const [warning, setWarning] = useState<string | null>(null);

  useEffect(() => {
    async function loadAsset() {
      try {
        setLoading(true);
        const data = await apiFetch<AssetResponse>(`/api/v1/assets/${resolvedParams.id}`);
        setName(data.asset.name || "");
        setURL(data.asset.url || "");
        setType(data.asset.type || "website");
        setEnvironment(data.asset.environment || "production");
        setDescription(data.asset.description || "");
      } catch (err) {
        setError(err);
      } finally {
        setLoading(false);
      }
    }
    if (user && activeOrg) {
      loadAsset();
    }
  }, [user, activeOrg, resolvedParams.id]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (saving) return; // Prevent duplicate submission

    if (!name.trim()) {
      setError(new Error(t.assetName + " is required"));
      return;
    }
    if (!url.trim()) {
      setError(new Error(t.url + " is required"));
      return;
    }

    try {
      setSaving(true);
      setError(null);
      setWarning(null);

      const res = await apiFetch<{
        asset: { id: string; name: string };
        warning?: string;
        warning_message?: string;
      }>(`/api/v1/assets/${resolvedParams.id}`, {
        method: "PATCH",
        body: JSON.stringify({
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
        toast.success("Asset details updated successfully");
      }

      router.push(`/assets/${resolvedParams.id}`);
    } catch (err) {
      setError(err);
      setSaving(false);
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
              ? "حساب‌های فقط-خواندنی (Viewer) اجازه ویرایش دارایی را ندارند."
              : "Viewer accounts do not have permission to edit websites or API assets."}
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
          title={t.editAsset}
          description="Update asset name, target URL, type, and environment specifications."
          icon={<Globe className="w-5 h-5" />}
          breadcrumbs={[
            { label: t.assets, href: "/assets" },
            { label: name || t.assetDetails, href: `/assets/${resolvedParams.id}` },
            { label: t.editAsset },
          ]}
        />

        {loading ? (
          <TableSkeleton rows={4} cols={2} />
        ) : (
          <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden p-6 sm:p-8">
            <form onSubmit={handleSubmit} className="space-y-6">
              {error && <ErrorState error={error} className="mb-4" />}

              {warning && (
                <div className="p-4 rounded-xl bg-amber-50 border border-amber-200 text-amber-800 text-xs flex items-center gap-2.5">
                  <AlertTriangle className="w-4 h-4 shrink-0 text-amber-600" />
                  <span>{warning}</span>
                </div>
              )}

              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                  {t.assetName} <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  required
                  disabled={saving}
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                  {t.url} <span className="text-red-500">*</span>
                </label>
                <input
                  type="url"
                  required
                  disabled={saving}
                  value={url}
                  onChange={(e) => setURL(e.target.value)}
                  className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all font-mono disabled:opacity-50 text-start"
                  dir="ltr"
                />
                <p className="text-[11px] text-slate-500 mt-1.5">
                  Must be an accessible HTTP or HTTPS URL (ports 80 or 443). SSRF restrictions apply.
                </p>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
                <div>
                  <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                    {t.type}
                  </label>
                  <select
                    disabled={saving}
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
                    disabled={saving}
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
                  disabled={saving}
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
                />
              </div>

              <div className="pt-4 flex items-center justify-end gap-3 border-t border-slate-100">
                <Link href={`/assets/${resolvedParams.id}`}>
                  <Button variant="ghost" size="sm" disabled={saving}>
                    {t.cancel}
                  </Button>
                </Link>
                <Button type="submit" variant="primary" size="md" loading={saving}>
                  {t.saveChanges}
                </Button>
              </div>
            </form>
          </div>
        )}
      </div>
    </AppLayout>
  );
}
