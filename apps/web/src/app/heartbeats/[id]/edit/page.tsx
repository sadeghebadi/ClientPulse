"use client";

import React, { useState, useEffect, use } from "react";
import { useRouter } from "next/navigation";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch, humanizeApiError } from "@/lib/api";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { useToast } from "@/components/ui/Toast";
import { Radio, AlertTriangle } from "lucide-react";

interface AssetOption {
  id: string;
  name: string;
  url: string;
}

export default function EditHeartbeatPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const resolvedParams = use(params);
  const id = resolvedParams.id;

  const router = useRouter();
  const { user, activeOrg } = useAuth();
  const { t, isRTL } = useI18n();
  const toast = useToast();

  const [assets, setAssets] = useState<AssetOption[]>([]);
  const [selectedAsset, setSelectedAsset] = useState("");
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [interval, setInterval] = useState("3600");
  const [grace, setGrace] = useState("720");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!user || !activeOrg) return;

    apiFetch<{ heartbeat: any }>(`/api/v1/heartbeats/${id}`)
      .then((data) => {
        const hb = data.heartbeat;
        setName(hb.name);
        setDescription(hb.description || "");
        setInterval(hb.expected_interval_seconds.toString());
        setGrace(hb.grace_period_seconds.toString());
        setSelectedAsset(hb.asset_id || "");

        // Fetch client's assets
        if (hb.client_id) {
          apiFetch<{ assets: AssetOption[] }>(`/api/v1/assets?client_id=${hb.client_id}`)
            .then((aData) => setAssets(aData.assets || []))
            .catch(() => {});
        }
      })
      .catch((err) => {
        setError(humanizeApiError(err, isRTL));
      })
      .finally(() => {
        setLoading(false);
      });
  }, [user, activeOrg, id, isRTL]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      setError(isRTL ? "نام هارت‌بیت الزامی است" : "Name is required");
      return;
    }

    setSubmitting(true);
    setError(null);

    try {
      await apiFetch(`/api/v1/heartbeats/${id}`, {
        method: "PATCH",
        body: JSON.stringify({
          name: name.trim(),
          description: description.trim() || undefined,
          asset_id: selectedAsset || undefined,
          expected_interval_seconds: parseInt(interval, 10),
          grace_period_seconds: parseInt(grace, 10),
        }),
      });

      toast.success(t.saveChanges);
      router.push(`/heartbeats/${id}`);
    } catch (err: any) {
      setError(humanizeApiError(err, isRTL));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <AppLayout>
      <div className="max-w-3xl mx-auto space-y-6">
        <PageHeader
          title={isRTL ? "ویرایش هارت‌بیت" : "Edit Heartbeat"}
          breadcrumbs={[
            { label: t.dashboard, href: "/" },
            { label: t.heartbeats, href: "/heartbeats" },
            { label: name || id, href: `/heartbeats/${id}` },
            { label: t.edit },
          ]}
        />

        {error && (
          <div className="p-4 bg-rose-50 border border-rose-200 rounded-xl text-xs text-rose-700 flex items-start gap-2">
            <AlertTriangle className="w-4 h-4 text-rose-600 shrink-0 mt-0.5" />
            <span>{error}</span>
          </div>
        )}

        {loading ? (
          <div className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm">
            <div className="animate-pulse space-y-4">
              <div className="h-4 bg-slate-100 rounded w-1/4"></div>
              <div className="h-8 bg-slate-100 rounded w-full"></div>
            </div>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm space-y-5">
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1.5">
                {t.heartbeatName} <span className="text-rose-500">*</span>
              </label>
              <input
                type="text"
                required
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full text-xs px-3 py-2 rounded-lg border border-slate-200 focus:outline-none focus:ring-2 focus:ring-slate-900/10"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1.5">
                {t.asset} ({isRTL ? "اختیاری" : "Optional"})
              </label>
              <select
                value={selectedAsset}
                onChange={(e) => setSelectedAsset(e.target.value)}
                className="w-full text-xs px-3 py-2 rounded-lg border border-slate-200 bg-white focus:outline-none focus:ring-2 focus:ring-slate-900/10"
              >
                <option value="">{isRTL ? "بدون انتساب به وب‌سایت خاص" : "Not tied to a specific website"}</option>
                {assets.map((a) => (
                  <option key={a.id} value={a.id}>
                    {a.name} ({a.url})
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1.5">
                {t.heartbeatDesc}
              </label>
              <textarea
                rows={2}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="w-full text-xs px-3 py-2 rounded-lg border border-slate-200 focus:outline-none focus:ring-2 focus:ring-slate-900/10"
              />
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1.5">
                  {t.expectedInterval} ({isRTL ? "ثانیه" : "seconds"})
                </label>
                <input
                  type="number"
                  min="60"
                  max="2592000"
                  required
                  value={interval}
                  onChange={(e) => setInterval(e.target.value)}
                  className="w-full text-xs px-3 py-2 rounded-lg border border-slate-200 focus:outline-none focus:ring-2 focus:ring-slate-900/10"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1.5">
                  {t.gracePeriod} ({isRTL ? "ثانیه" : "seconds"})
                </label>
                <input
                  type="number"
                  min="60"
                  max="86400"
                  required
                  value={grace}
                  onChange={(e) => setGrace(e.target.value)}
                  className="w-full text-xs px-3 py-2 rounded-lg border border-slate-200 focus:outline-none focus:ring-2 focus:ring-slate-900/10"
                />
              </div>
            </div>

            <div className="pt-4 border-t border-slate-200 flex justify-end gap-2">
              <Button
                type="button"
                variant="secondary"
                onClick={() => router.push(`/heartbeats/${id}`)}
              >
                {t.cancel}
              </Button>
              <Button
                type="submit"
                loading={submitting}
                icon={<Radio className="w-4 h-4" />}
              >
                {t.saveChanges}
              </Button>
            </div>
          </form>
        )}
      </div>
    </AppLayout>
  );
}

