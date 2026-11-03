"use client";

import React, { useState, useEffect, use } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch, humanizeApiError } from "@/lib/api";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { useToast } from "@/components/ui/Toast";
import { Radio, AlertTriangle, ShieldAlert } from "lucide-react";

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
  const { user, activeOrg, role } = useAuth();
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

  const canManage = role === "owner" || role === "admin" || role === "member";

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
    if (!canManage) {
      setError(isRTL ? "شما اجازه ویرایش هارت‌بیت را ندارید." : "You do not have permission to edit heartbeats.");
      return;
    }
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

  if (!canManage && !loading) {
    return (
      <AppLayout>
        <div className="max-w-md mx-auto p-8 text-center bg-white rounded-2xl border border-slate-200 mt-12">
          <ShieldAlert className="w-12 h-12 text-rose-500 mx-auto mb-3" />
          <h2 className="text-base font-bold text-slate-900">{isRTL ? "دسترسی محدود است" : "Access Restricted"}</h2>
          <p className="text-xs text-slate-500 mt-1 mb-4">
            {isRTL
              ? "حساب‌های فقط-خواندنی (Viewer) اجازه ویرایش هارت‌بیت را ندارند."
              : "Viewer accounts do not have permission to edit heartbeats."}
          </p>
          <Link href={`/heartbeats/${id}`}>
            <Button size="sm" variant="outline">{t.viewDetails}</Button>
          </Link>
        </div>
      </AppLayout>
    );
  }

  return (
    <AppLayout>
      <div className="max-w-3xl mx-auto space-y-6">
        <PageHeader
          title={isRTL ? "ویرایش هارت‌بیت" : "Edit Heartbeat"}
          breadcrumbs={[
            { label: t.dashboard, href: "/dashboard" },
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
          <div className="bg-white border border-slate-200 rounded-2xl p-6 shadow-sm">
            <div className="animate-pulse space-y-4">
              <div className="h-4 bg-slate-100 rounded w-1/4"></div>
              <div className="h-10 bg-slate-100 rounded w-full"></div>
              <div className="h-4 bg-slate-100 rounded w-1/4"></div>
              <div className="h-10 bg-slate-100 rounded w-full"></div>
            </div>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="bg-white border border-slate-200 rounded-2xl p-5 sm:p-7 shadow-sm space-y-5">
            <div>
              <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                {t.heartbeatName} <span className="text-rose-500">*</span>
              </label>
              <input
                type="text"
                required
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full text-xs sm:text-sm px-3.5 py-2.5 rounded-xl border border-slate-200 bg-slate-50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 transition-all"
              />
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                {t.asset} ({isRTL ? "اختیاری" : "Optional"})
              </label>
              <select
                value={selectedAsset}
                onChange={(e) => setSelectedAsset(e.target.value)}
                className="w-full text-xs sm:text-sm px-3.5 py-2.5 rounded-xl border border-slate-200 bg-slate-50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 transition-all"
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
              <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                {t.heartbeatDesc}
              </label>
              <textarea
                rows={2}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="w-full text-xs sm:text-sm px-3.5 py-2.5 rounded-xl border border-slate-200 bg-slate-50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 transition-all"
              />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 sm:gap-6 pt-2">
              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                  {t.expectedInterval} ({t.secondsUnit})
                </label>
                <input
                  type="number"
                  min="60"
                  max="2592000"
                  required
                  value={interval}
                  onChange={(e) => setInterval(e.target.value)}
                  className="w-full text-xs sm:text-sm px-3.5 py-2.5 rounded-xl border border-slate-200 bg-slate-50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 transition-all text-start"
                  dir="ltr"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                  {t.gracePeriod} ({t.secondsUnit})
                </label>
                <input
                  type="number"
                  min="60"
                  max="86400"
                  required
                  value={grace}
                  onChange={(e) => setGrace(e.target.value)}
                  className="w-full text-xs sm:text-sm px-3.5 py-2.5 rounded-xl border border-slate-200 bg-slate-50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-blue-500 transition-all text-start"
                  dir="ltr"
                />
              </div>
            </div>

            <div className="pt-4 border-t border-slate-100 flex items-center justify-end gap-3">
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
