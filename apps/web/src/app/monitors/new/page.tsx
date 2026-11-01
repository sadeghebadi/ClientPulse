"use client";

import React, { useState, useEffect, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import { Activity, Info } from "lucide-react";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { ErrorState } from "@/components/ui/ErrorState";
import { useToast } from "@/components/ui/Toast";

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
  const toast = useToast();

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
  const [error, setError] = useState<any>(null);

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
    if (loading) return; // Prevent duplicate submission

    if (!assetID) {
      setError(new Error("Please select an asset to monitor"));
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

      toast.success("Monitor configured successfully");
      router.push(`/monitors/${res.monitor.id}`);
    } catch (err) {
      setError(err);
      setLoading(false);
    }
  };

  return (
    <AppLayout>
      <div className="max-w-3xl mx-auto">
        <PageHeader
          title={t.createMonitor}
          description="Configure automated check parameters, timeouts, SSL validation, and assertions."
          icon={<Activity className="w-5 h-5" />}
          breadcrumbs={[
            { label: t.monitors, href: "/monitors" },
            { label: t.createMonitor },
          ]}
        />

        <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden p-6 sm:p-8">
          <form onSubmit={handleSubmit} className="space-y-6">
            {error && <ErrorState error={error} className="mb-4" />}

            <div>
              <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                Target Website / Asset <span className="text-red-500">*</span>
              </label>
              <select
                name="asset_id"
                required
                disabled={loading}
                value={assetID}
                onChange={(e) => setAssetID(e.target.value)}
                className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
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
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                  {t.method}
                </label>
                <select
                  name="method"
                  disabled={loading}
                  value={method}
                  onChange={(e) => setMethod(e.target.value)}
                  className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                >
                  <option value="GET">GET (Full response body & TLS handshake)</option>
                  <option value="HEAD">HEAD (Fast headers check)</option>
                </select>
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                  {t.interval}
                </label>
                <select
                  name="interval"
                  disabled={loading}
                  value={interval}
                  onChange={(e) => setInterval(Number(e.target.value))}
                  className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                >
                  <option value={60}>60 seconds (1 minute)</option>
                  <option value={180}>180 seconds (3 minutes)</option>
                  <option value={300}>300 seconds (5 minutes — Default)</option>
                  <option value={600}>600 seconds (10 minutes)</option>
                </select>
              </div>
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                URL Override (Optional)
              </label>
              <input
                type="url"
                disabled={loading}
                value={urlOverride}
                onChange={(e) => setUrlOverride(e.target.value)}
                placeholder="Leave blank to use base asset URL"
                className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all font-mono disabled:opacity-50"
              />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-6">
              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                  {t.timeout} (3–30s)
                </label>
                <input
                  type="number"
                  min={3}
                  max={30}
                  disabled={loading}
                  value={timeout}
                  onChange={(e) => setTimeout(Number(e.target.value))}
                  className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                  Min Expected Status
                </label>
                <input
                  type="number"
                  min={100}
                  max={599}
                  disabled={loading}
                  value={expectedStatusMin}
                  onChange={(e) => setExpectedStatusMin(Number(e.target.value))}
                  className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                  Max Expected Status
                </label>
                <input
                  type="number"
                  min={100}
                  max={599}
                  disabled={loading}
                  value={expectedStatusMax}
                  onChange={(e) => setExpectedStatusMax(Number(e.target.value))}
                  className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                />
              </div>
            </div>

            <div className="flex items-center gap-3 p-4 bg-slate-50 rounded-xl border border-slate-200">
              <input
                type="checkbox"
                id="followRedirects"
                disabled={loading}
                checked={followRedirects}
                onChange={(e) => setFollowRedirects(e.target.checked)}
                className="w-4 h-4 text-blue-600 rounded border-slate-300 focus:ring-blue-500 disabled:opacity-50"
              />
              <label htmlFor="followRedirects" className="text-xs font-medium text-slate-700 cursor-pointer">
                {t.followRedirects} (Maximum 5 hops with SSRF revalidation per redirect)
              </label>
            </div>

            {/* Response Assertions */}
            <div className="border-t border-slate-100 pt-6 space-y-4">
              <div className="flex items-center gap-2">
                <Info className="w-4 h-4 text-blue-600" />
                <h3 className="text-xs font-bold text-slate-900 uppercase tracking-wider">
                  Response Body Assertions (Optional)
                </h3>
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                  {t.expectedText}
                </label>
                <input
                  type="text"
                  disabled={loading}
                  value={expectedText}
                  onChange={(e) => setExpectedText(e.target.value)}
                  placeholder="e.g. Welcome to Acme or Operational"
                  className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all disabled:opacity-50"
                />
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                    {t.jsonPointer}
                  </label>
                  <input
                    type="text"
                    disabled={loading}
                    value={jsonPointer}
                    onChange={(e) => setJsonPointer(e.target.value)}
                    placeholder="/status or /data/ready"
                    className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all font-mono disabled:opacity-50"
                  />
                </div>

                <div>
                  <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                    {t.jsonExpectedValue}
                  </label>
                  <input
                    type="text"
                    disabled={loading}
                    value={jsonExpectedValue}
                    onChange={(e) => setJsonExpectedValue(e.target.value)}
                    placeholder="true or ok"
                    className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all font-mono disabled:opacity-50"
                  />
                </div>
              </div>
            </div>

            <div className="pt-4 flex items-center justify-end gap-3 border-t border-slate-100">
              <Link href="/monitors">
                <Button variant="ghost" size="sm" disabled={loading}>
                  {t.cancel}
                </Button>
              </Link>
              <Button
                type="submit"
                variant="primary"
                size="md"
                loading={loading}
                disabled={assets.length === 0}
              >
                {t.createMonitor}
              </Button>
            </div>
          </form>
        </div>
      </div>
    </AppLayout>
  );
}

export default function NewMonitorPage() {
  return (
    <Suspense fallback={<div className="p-12 text-center text-sm text-slate-500">Loading...</div>}>
      <NewMonitorContent />
    </Suspense>
  );
}
