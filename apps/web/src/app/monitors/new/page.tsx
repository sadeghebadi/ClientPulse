"use client";

import React, { useState, useEffect, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { Header } from "@/components/Header";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import { Activity, ArrowLeft, ArrowRight, AlertCircle } from "lucide-react";

interface AssetOption {
  id: string;
  name: string;
  url: string;
  client_name: string;
}

function NewMonitorContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const initialAssetID = searchParams.get("asset_id") || "";

  const { user, activeOrg } = useAuth();
  const { t, isRTL } = useI18n();

  const [assets, setAssets] = useState<AssetOption[]>([]);
  const [assetID, setAssetID] = useState(initialAssetID);
  const [method, setMethod] = useState("GET");
  const [urlOverride, setUrlOverride] = useState("");
  const [interval, setInterval] = useState(300);
  const [timeout, setTimeout] = useState(10);
  const [expectedStatusMin, setExpectedStatusMin] = useState(200);
  const [expectedStatusMax, setExpectedStatusMax] = useState(399);
  const [followRedirects, setFollowRedirects] = useState(true);
  const [maxRedirects, setMaxRedirects] = useState(5);
  const [expectedText, setExpectedText] = useState("");
  const [jsonPointer, setJsonPointer] = useState("");
  const [jsonExpectedValue, setJsonExpectedValue] = useState("");

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!user || !activeOrg) return;
    let isMounted = true;

    apiFetch<{ assets: AssetOption[] }>("/api/v1/assets?is_active=true")
      .then((data) => {
        if (isMounted) {
          const list = data.assets || [];
          setAssets(list);
          setAssetID((prev) => (prev ? prev : list.length > 0 ? list[0].id : ""));
        }
      })
      .catch((err) => {
        if (isMounted) {
          console.error("Failed to load assets", err);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user, activeOrg]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!assetID) {
      setError("Please select an asset");
      return;
    }

    try {
      setLoading(true);
      setError(null);

      const res = await apiFetch<{ monitor: { id: string } }>("/api/v1/monitors", {
        method: "POST",
        body: JSON.stringify({
          asset_id: assetID,
          method,
          url_override: urlOverride.trim() || undefined,
          interval_seconds: Number(interval),
          timeout_seconds: Number(timeout),
          expected_status_min: Number(expectedStatusMin),
          expected_status_max: Number(expectedStatusMax),
          follow_redirects: followRedirects,
          max_redirects: Number(maxRedirects),
          expected_text: expectedText.trim() || undefined,
          json_pointer: jsonPointer.trim() || undefined,
          json_expected_value: jsonExpectedValue.trim() || undefined,
        }),
      });

      router.push(`/monitors/${res.monitor.id}`);
    } catch (err: any) {
      setError(err?.message || "Failed to create monitor");
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col">
      <Header />

      <main className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 py-8 flex-1 w-full">
        <div className="mb-6">
          <Link
            href="/monitors"
            className="inline-flex items-center gap-1.5 text-xs font-semibold text-slate-500 hover:text-slate-800 transition-colors"
          >
            {isRTL ? <ArrowRight className="w-3.5 h-3.5" /> : <ArrowLeft className="w-3.5 h-3.5" />}
            <span>{t.monitors}</span>
          </Link>
        </div>

        <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
          <div className="p-6 sm:p-8 border-b border-slate-100">
            <h1 className="text-xl font-bold text-slate-900 tracking-tight flex items-center gap-2">
              <Activity className="w-6 h-6 text-blue-600" />
              <span>{t.createMonitor}</span>
            </h1>
            <p className="text-xs text-slate-500 mt-1">
              Configure automated check parameters, timeouts, SSL validation, and assertions.
            </p>
          </div>

          <form onSubmit={handleSubmit} className="p-6 sm:p-8 space-y-6">
            {error && (
              <div className="p-4 rounded-lg bg-red-50 border border-red-200 text-red-700 text-xs flex items-center gap-2">
                <AlertCircle className="w-4 h-4 flex-shrink-0" />
                <span>{error}</span>
              </div>
            )}

            <div>
              <label className="block text-xs font-semibold text-slate-700 uppercase mb-2">
                Target Website / Asset <span className="text-red-500">*</span>
              </label>
              <select
                required
                value={assetID}
                onChange={(e) => setAssetID(e.target.value)}
                className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all"
              >
                {assets.length === 0 ? (
                  <option value="">No assets registered. Please register an asset first.</option>
                ) : (
                  assets.map((a) => (
                    <option key={a.id} value={a.id}>
                      {a.name} ({a.url}) — {a.client_name}
                    </option>
                  ))
                )}
              </select>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-2">
                  {t.method}
                </label>
                <select
                  value={method}
                  onChange={(e) => setMethod(e.target.value)}
                  className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="GET">GET (Full response & body)</option>
                  <option value="HEAD">HEAD (Headers only)</option>
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-2">
                  {t.interval}
                </label>
                <select
                  value={interval}
                  onChange={(e) => setInterval(Number(e.target.value))}
                  className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value={60}>60 seconds (1 minute)</option>
                  <option value={180}>180 seconds (3 minutes)</option>
                  <option value={300}>300 seconds (5 minutes — Default)</option>
                  <option value={600}>600 seconds (10 minutes)</option>
                </select>
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 uppercase mb-2">
                URL Override (Optional)
              </label>
              <input
                type="url"
                value={urlOverride}
                onChange={(e) => setUrlOverride(e.target.value)}
                placeholder="Leave blank to use Asset URL"
                className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all"
              />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-6">
              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-2">
                  {t.timeout} (3–30s)
                </label>
                <input
                  type="number"
                  min={3}
                  max={30}
                  value={timeout}
                  onChange={(e) => setTimeout(Number(e.target.value))}
                  className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-2">
                  Min Expected Status
                </label>
                <input
                  type="number"
                  min={100}
                  max={599}
                  value={expectedStatusMin}
                  onChange={(e) => setExpectedStatusMin(Number(e.target.value))}
                  className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-2">
                  Max Expected Status
                </label>
                <input
                  type="number"
                  min={100}
                  max={599}
                  value={expectedStatusMax}
                  onChange={(e) => setExpectedStatusMax(Number(e.target.value))}
                  className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            </div>

            <div className="flex items-center gap-3 p-4 bg-slate-50 rounded-xl border border-slate-200">
              <input
                type="checkbox"
                id="followRedirects"
                checked={followRedirects}
                onChange={(e) => setFollowRedirects(e.target.checked)}
                className="w-4 h-4 text-blue-600 rounded border-slate-300 focus:ring-blue-500"
              />
              <label htmlFor="followRedirects" className="text-xs font-medium text-slate-700 cursor-pointer">
                {t.followRedirects} (Maximum 5 hops with SSRF revalidation)
              </label>
            </div>

            {/* Response Assertions */}
            <div className="border-t border-slate-100 pt-6 space-y-4">
              <h3 className="text-xs font-bold text-slate-900 uppercase tracking-wider">
                Response Body Assertions (Optional)
              </h3>

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase mb-2">
                  {t.expectedText}
                </label>
                <input
                  type="text"
                  value={expectedText}
                  onChange={(e) => setExpectedText(e.target.value)}
                  placeholder="e.g. Welcome to Acme or OK"
                  className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all"
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase mb-2">
                    {t.jsonPointer}
                  </label>
                  <input
                    type="text"
                    value={jsonPointer}
                    onChange={(e) => setJsonPointer(e.target.value)}
                    placeholder="/status or /data/ready"
                    className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase mb-2">
                    {t.jsonExpectedValue}
                  </label>
                  <input
                    type="text"
                    value={jsonExpectedValue}
                    onChange={(e) => setJsonExpectedValue(e.target.value)}
                    placeholder="true or healthy"
                    className="w-full px-3.5 py-2.5 bg-slate-50 border border-slate-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all"
                  />
                </div>
              </div>
            </div>

            <div className="pt-4 flex items-center justify-end gap-3 border-t border-slate-100">
              <Link
                href="/monitors"
                className="px-4 py-2 text-xs font-semibold text-slate-600 hover:text-slate-800 transition-colors"
              >
                {t.cancel}
              </Link>
              <button
                type="submit"
                disabled={loading || assets.length === 0}
                className="px-6 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-xs font-semibold shadow-sm transition-colors"
              >
                {loading ? t.loading : t.createMonitor}
              </button>
            </div>
          </form>
        </div>
      </main>
    </div>
  );
}

export default function NewMonitorPage() {
  return (
    <Suspense fallback={<div className="p-12 text-center text-sm text-slate-500">Loading...</div>}>
      <NewMonitorContent />
    </Suspense>
  );
}
