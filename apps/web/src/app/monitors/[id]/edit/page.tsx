"use client";

import React, { useState, useEffect, use } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import { Activity } from "lucide-react";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { ErrorState } from "@/components/ui/ErrorState";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { useToast } from "@/components/ui/Toast";

interface MonitorResponse {
  monitor: {
    id: string;
    asset_name: string;
    method: string;
    url_override?: string;
    interval_seconds: number;
    timeout_seconds: number;
    expected_status_min: number;
    expected_status_max: number;
    follow_redirects: boolean;
    max_redirects: number;
    expected_text?: string;
    json_pointer?: string;
    json_expected_value?: string;
  };
}

export default function EditMonitorPage({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = use(params);
  const router = useRouter();
  const { user, activeOrg } = useAuth();
  const { t } = useI18n();
  const toast = useToast();

  const [assetName, setAssetName] = useState("");
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

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<any>(null);

  useEffect(() => {
    async function loadMonitor() {
      try {
        setLoading(true);
        const data = await apiFetch<MonitorResponse>(`/api/v1/monitors/${resolvedParams.id}`);
        setAssetName(data.monitor.asset_name || "");
        setMethod(data.monitor.method || "GET");
        setUrlOverride(data.monitor.url_override || "");
        setInterval(data.monitor.interval_seconds || 300);
        setTimeout(data.monitor.timeout_seconds || 10);
        setExpectedStatusMin(data.monitor.expected_status_min || 200);
        setExpectedStatusMax(data.monitor.expected_status_max || 399);
        setFollowRedirects(data.monitor.follow_redirects ?? true);
        setMaxRedirects(data.monitor.max_redirects || 5);
        setExpectedText(data.monitor.expected_text || "");
        setJsonPointer(data.monitor.json_pointer || "");
        setJsonExpectedValue(data.monitor.json_expected_value || "");
      } catch (err) {
        setError(err);
      } finally {
        setLoading(false);
      }
    }
    if (user && activeOrg) {
      loadMonitor();
    }
  }, [user, activeOrg, resolvedParams.id]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (saving) return; // Prevent duplicate submission

    try {
      setSaving(true);
      setError(null);

      await apiFetch(`/api/v1/monitors/${resolvedParams.id}`, {
        method: "PATCH",
        body: JSON.stringify({
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

      toast.success("Monitor settings saved successfully");
      router.push(`/monitors/${resolvedParams.id}`);
    } catch (err) {
      setError(err);
      setSaving(false);
    }
  };

  return (
    <AppLayout>
      <div className="max-w-3xl mx-auto">
        <PageHeader
          title={t.editMonitor}
          description="Adjust check interval, response status expectations, timeouts, and body assertions."
          icon={<Activity className="w-5 h-5" />}
          breadcrumbs={[
            { label: t.monitors, href: "/monitors" },
            { label: assetName || t.monitorDetails, href: `/monitors/${resolvedParams.id}` },
            { label: t.editMonitor },
          ]}
        />

        {loading ? (
          <TableSkeleton rows={5} cols={2} />
        ) : (
          <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden p-6 sm:p-8">
            <form onSubmit={handleSubmit} className="space-y-6">
              {error && <ErrorState error={error} className="mb-4" />}

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-6">
                <div>
                  <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                    {t.method}
                  </label>
                  <select
                    disabled={saving}
                    value={method}
                    onChange={(e) => setMethod(e.target.value)}
                    className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                  >
                    <option value="GET">GET</option>
                    <option value="HEAD">HEAD</option>
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                    {t.interval}
                  </label>
                  <select
                    disabled={saving}
                    value={interval}
                    onChange={(e) => setInterval(Number(e.target.value))}
                    className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                  >
                    <option value={60}>60 seconds (1 minute)</option>
                    <option value={180}>180 seconds (3 minutes)</option>
                    <option value={300}>300 seconds (5 minutes)</option>
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
                  disabled={saving}
                  value={urlOverride}
                  onChange={(e) => setUrlOverride(e.target.value)}
                  placeholder="Leave blank to use asset URL"
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
                    disabled={saving}
                    value={timeout}
                    onChange={(e) => setTimeout(Number(e.target.value))}
                    className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                  />
                </div>

                <div>
                  <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                    Min Status
                  </label>
                  <input
                    type="number"
                    min={100}
                    max={599}
                    disabled={saving}
                    value={expectedStatusMin}
                    onChange={(e) => setExpectedStatusMin(Number(e.target.value))}
                    className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
                  />
                </div>

                <div>
                  <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                    Max Status
                  </label>
                  <input
                    type="number"
                    min={100}
                    max={599}
                    disabled={saving}
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
                  disabled={saving}
                  checked={followRedirects}
                  onChange={(e) => setFollowRedirects(e.target.checked)}
                  className="w-4 h-4 text-blue-600 rounded border-slate-300 focus:ring-blue-500 disabled:opacity-50"
                />
                <label htmlFor="followRedirects" className="text-xs font-medium text-slate-700 cursor-pointer">
                  {t.followRedirects}
                </label>
              </div>

              <div className="border-t border-slate-100 pt-6 space-y-4">
                <h3 className="text-xs font-bold text-slate-900 uppercase tracking-wider">
                  Response Body Assertions (Optional)
                </h3>

                <div>
                  <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                    {t.expectedText}
                  </label>
                  <input
                    type="text"
                    disabled={saving}
                    value={expectedText}
                    onChange={(e) => setExpectedText(e.target.value)}
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
                      disabled={saving}
                      value={jsonPointer}
                      onChange={(e) => setJsonPointer(e.target.value)}
                      className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all font-mono disabled:opacity-50"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                      {t.jsonExpectedValue}
                    </label>
                    <input
                      type="text"
                      disabled={saving}
                      value={jsonExpectedValue}
                      onChange={(e) => setJsonExpectedValue(e.target.value)}
                      className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:bg-white transition-all font-mono disabled:opacity-50"
                    />
                  </div>
                </div>
              </div>

              <div className="pt-4 flex items-center justify-end gap-3 border-t border-slate-100">
                <Link href={`/monitors/${resolvedParams.id}`}>
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
