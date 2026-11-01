"use client";

import React, { useState, useEffect, use } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { Header } from "@/components/Header";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import { Globe, ArrowLeft, ArrowRight, AlertCircle, AlertTriangle } from "lucide-react";

export default function EditAssetPage({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = use(params);
  const router = useRouter();
  const { user, activeOrg } = useAuth();
  const { t, isRTL } = useI18n();

  const [name, setName] = useState("");
  const [url, setURL] = useState("");
  const [type, setType] = useState("website");
  const [environment, setEnvironment] = useState("production");
  const [description, setDescription] = useState("");

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [warning, setWarning] = useState<string | null>(null);

  useEffect(() => {
    async function loadAsset() {
      try {
        setLoading(true);
        const data = await apiFetch<{ asset: any }>(`/api/v1/assets/${resolvedParams.id}`);
        setName(data.asset.name || "");
        setURL(data.asset.url || "");
        setType(data.asset.type || "website");
        setEnvironment(data.asset.environment || "production");
        setDescription(data.asset.description || "");
      } catch (err: any) {
        setError(err?.message || "Failed to load asset");
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
    if (!name.trim()) {
      setError("Asset name is required");
      return;
    }
    if (!url.trim()) {
      setError("Asset URL is required");
      return;
    }

    try {
      setSaving(true);
      setError(null);
      setWarning(null);

      const res = await apiFetch<{ asset: any; warning?: string; warning_message?: string }>(`/api/v1/assets/${resolvedParams.id}`, {
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
        setWarning(res.warning_message || t.duplicateUrlWarning);
      }

      router.push(`/assets/${resolvedParams.id}`);
    } catch (err: any) {
      setError(err?.message || "Failed to update asset");
      setSaving(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">
      <Header />

      <main className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 py-8 flex-1 w-full">
        <div className="mb-6">
          <Link
            href={`/assets/${resolvedParams.id}`}
            className="inline-flex items-center gap-1.5 text-xs font-semibold text-slate-500 hover:text-slate-800 transition-colors"
          >
            {isRTL ? <ArrowRight className="w-3.5 h-3.5" /> : <ArrowLeft className="w-3.5 h-3.5" />}
            <span>{t.assetDetails}</span>
          </Link>
        </div>

        <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
          <div className="p-6 sm:p-8 border-b border-slate-100">
            <h1 className="text-xl font-bold text-slate-900 tracking-tight flex items-center gap-2">
              <Globe className="w-6 h-6 text-blue-600" />
              <span>{t.editAsset}</span>
            </h1>
          </div>

          {loading ? (
            <div className="p-12 text-center text-slate-500 text-sm">{t.loading}</div>
          ) : (
            <form onSubmit={handleSubmit} className="p-6 sm:p-8 space-y-6">
              {error && (
                <div className="p-4 rounded-lg bg-red-50 border border-red-200 text-red-700 text-xs flex items-center gap-2">
                  <AlertCircle className="w-4 h-4 flex-shrink-0" />
                  <span>{error}</span>
                </div>
              )}

              {warning && (
                <div className="p-4 rounded-lg bg-amber-50 border border-amber-200 text-amber-700 text-xs flex items-center gap-2">
                  <AlertTriangle className="w-4 h-4 flex-shrink-0" />
                  <span>{warning}</span>
                </div>
              )}

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-2">
                  {t.assetName} <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
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
                  className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all"
                />
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
                    <option value="website">Website</option>
                    <option value="api">API</option>
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
                  className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all"
                />
              </div>

              <div className="pt-4 flex items-center justify-end gap-3 border-t border-slate-100">
                <Link
                  href={`/assets/${resolvedParams.id}`}
                  className="px-4 py-2 text-xs font-semibold text-slate-600 hover:text-slate-800 transition-colors"
                >
                  {t.cancel}
                </Link>
                <button
                  type="submit"
                  disabled={saving}
                  className="px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-xs font-semibold shadow-sm transition-colors"
                >
                  {saving ? t.loading : t.saveChanges}
                </button>
              </div>
            </form>
          )}
        </div>
      </main>
    </div>
  );
}

