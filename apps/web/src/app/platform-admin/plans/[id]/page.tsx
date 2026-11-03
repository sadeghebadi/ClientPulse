"use client";

import React, { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import {
  Palette,
  Sparkles,
  Layers,
  Clock,
  ArrowLeft,
  ArrowRight,
  Shield,
  CheckCircle2,
  AlertTriangle,
  Archive,
  History,
  Building,
} from "lucide-react";
import { apiFetch } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/Button";
import { LoadingSkeleton } from "@/components/ui/LoadingSkeleton";
import { ErrorState } from "@/components/ui/ErrorState";
import { Breadcrumbs } from "@/components/ui/PageHeader";

interface PlanVersionItem {
  id: string;
  version: number;
  status: string;
  price_amount: number;
  currency: string;
  billing_interval: string;
  max_clients: number;
  max_assets: number;
  max_monitors: number;
  max_heartbeats: number;
  max_members: number;
  min_check_interval_seconds: number;
  retention_days: number;
  custom_branding_enabled: boolean;
  public_reports_enabled: boolean;
  created_at: string;
}

interface PlanDetail {
  plan: {
    id: string;
    name: string;
    slug: string;
    description: string;
    status: string;
    current_version: number;
    created_at: string;
    updated_at: string;
  };
  versions: PlanVersionItem[];
}

export default function PlatformAdminPlanDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { t, isRTL } = useI18n();

  const [detail, setDetail] = useState<PlanDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Form values
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [priceAmount, setPriceAmount] = useState<number>(2900);
  const [currency, setCurrency] = useState("USD");
  const [billingInterval, setBillingInterval] = useState("monthly");

  const [maxClients, setMaxClients] = useState<number>(5);
  const [maxAssets, setMaxAssets] = useState<number>(10);
  const [maxMonitors, setMaxMonitors] = useState<number>(10);
  const [maxHeartbeats, setMaxHeartbeats] = useState<number>(5);
  const [maxMembers, setMaxMembers] = useState<number>(5);
  const [minCheckInterval, setMinCheckInterval] = useState<number>(60);
  const [retentionDays, setRetentionDays] = useState<number>(30);

  const [customBranding, setCustomBranding] = useState(false);
  const [careReports, setCareReports] = useState(true);

  const [saving, setSaving] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [toastMessage, setToastMessage] = useState<string | null>(null);

  const fetchDetail = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await apiFetch<PlanDetail>(`/api/v1/platform-admin/plans/${id}`);
      setDetail(data);

      setName(data.plan.name);
      setDescription(data.plan.description || "");

      // Populate from latest version
      if (data.versions && data.versions.length > 0) {
        const latest = data.versions[0];
        setPriceAmount(latest.price_amount);
        setCurrency(latest.currency);
        setBillingInterval(latest.billing_interval);
        setMaxClients(latest.max_clients);
        setMaxAssets(latest.max_assets);
        setMaxMonitors(latest.max_monitors);
        setMaxHeartbeats(latest.max_heartbeats);
        setMaxMembers(latest.max_members);
        setMinCheckInterval(latest.min_check_interval_seconds);
        setRetentionDays(latest.retention_days);
        setCustomBranding(latest.custom_branding_enabled);
        setCareReports(latest.public_reports_enabled);
      }
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Failed to load plan details");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (id) fetchDetail();
  }, [id]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      setError("Plan name is required.");
      return;
    }
    if (minCheckInterval < 60) {
      setError("Minimum check interval must be at least 60 seconds.");
      return;
    }

    try {
      setSaving(true);
      setError(null);
      await apiFetch(`/api/v1/platform-admin/plans/${id}`, {
        method: "PUT",
        body: JSON.stringify({
          name: name.trim(),
          description: description.trim(),
          price_amount: Number(priceAmount),
          currency: currency.trim().toUpperCase(),
          billing_interval: billingInterval,
          max_clients: Number(maxClients),
          max_assets: Number(maxAssets),
          max_monitors: Number(maxMonitors),
          max_heartbeats: Number(maxHeartbeats),
          max_members: Number(maxMembers),
          min_check_interval_seconds: Number(minCheckInterval),
          retention_days: Number(retentionDays),
          custom_branding_enabled: customBranding,
          public_reports_enabled: careReports,
        }),
      });

      setToastMessage("Plan saved successfully.");
      fetchDetail();
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Failed to update plan");
    } finally {
      setSaving(false);
    }
  };

  const handlePublish = async () => {
    try {
      setPublishing(true);
      await apiFetch(`/api/v1/platform-admin/plans/${id}/publish`, {
        method: "POST",
      });
      setToastMessage("Plan version published and made available for assignment.");
      fetchDetail();
    } catch (err: unknown) {
      const e = err as Error;
      alert(`Publish failed: ${e.message}`);
    } finally {
      setPublishing(false);
    }
  };

  if (loading) {
    return (
      <div className="space-y-6">
        <LoadingSkeleton className="h-32 w-full" />
        <LoadingSkeleton className="h-64 w-full" />
      </div>
    );
  }

  if (error || !detail) {
    return (
      <ErrorState
        title="Failed to Load Plan"
        error={error || "Subscription plan could not be found."}
        onRetry={fetchDetail}
      />
    );
  }

  const { plan, versions } = detail;
  const isDraft = plan.status === "draft";
  const isPublished = plan.status === "published";

  return (
    <div className="space-y-6 max-w-4xl mx-auto">
      {toastMessage && (
        <div className="p-4 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-800 text-xs flex items-center justify-between">
          <span>{toastMessage}</span>
          <button onClick={() => setToastMessage(null)} className="font-bold text-slate-500 hover:text-slate-800">
            &times;
          </button>
        </div>
      )}

      <Breadcrumbs
        items={[
          { label: t.platformOverview, href: "/platform-admin" },
          { label: t.platformPlans, href: "/platform-admin/plans" },
          { label: plan.name },
        ]}
      />

      {/* Plan Header Card */}
      <div className="bg-white rounded-2xl border border-slate-200 p-6 shadow-sm flex flex-col sm:flex-row sm:items-center justify-between gap-6">
        <div className="flex items-center gap-4">
          <div className="w-14 h-14 rounded-2xl bg-emerald-600 text-white flex items-center justify-center font-black text-xl shadow-md shrink-0">
            <Palette className="w-7 h-7" />
          </div>
          <div>
            <div className="flex items-center gap-2 flex-wrap">
              <h1 className="text-xl font-black text-slate-900">{plan.name}</h1>
              {isPublished ? (
                <span className="inline-flex items-center gap-1 bg-emerald-50 text-emerald-700 border border-emerald-200 px-2.5 py-0.5 rounded-full font-bold text-[10px]">
                  <CheckCircle2 className="w-3 h-3" />
                  <span>Published (v{plan.current_version})</span>
                </span>
              ) : isDraft ? (
                <span className="inline-flex items-center gap-1 bg-amber-50 text-amber-700 border border-amber-200 px-2.5 py-0.5 rounded-full font-bold text-[10px]">
                  <Clock className="w-3 h-3" />
                  <span>Draft (v{plan.current_version})</span>
                </span>
              ) : (
                <span className="inline-flex items-center gap-1 bg-slate-100 text-slate-600 px-2.5 py-0.5 rounded-full font-bold text-[10px]">
                  <Archive className="w-3 h-3" />
                  <span>Archived</span>
                </span>
              )}
            </div>
            <div className="flex items-center gap-3 text-xs text-slate-500 mt-1 font-mono">
              <span>Slug: /{plan.slug}</span>
              <span>•</span>
              <span className="text-slate-400">ID: {plan.id}</span>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2 self-start sm:self-auto">
          {isDraft && (
            <Button
              variant="primary"
              loading={publishing}
              onClick={handlePublish}
              className="gap-1.5"
            >
              <CheckCircle2 className="w-4 h-4" />
              <span>{t.publishPlan}</span>
            </Button>
          )}
        </div>
      </div>

      {/* Immutability Notice for Published Plans */}
      {isPublished && (
        <div className="p-4 rounded-2xl bg-blue-50 border border-blue-200 flex items-start gap-3 text-blue-900">
          <AlertTriangle className="w-5 h-5 text-blue-600 shrink-0 mt-0.5" />
          <div className="text-xs space-y-1">
            <strong className="block font-bold">Immutability Architecture Notice</strong>
            <p className="text-blue-800 leading-relaxed">
              {t.immutablePlanNotice}
            </p>
          </div>
        </div>
      )}

      {/* Plan Form */}
      <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 sm:p-8">
        <form onSubmit={handleSubmit} className="space-y-8">
          <div className="space-y-4">
            <h2 className="text-sm font-bold text-slate-900 uppercase tracking-wider text-[11px] text-slate-400">
              Basic Tier Parameters
            </h2>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">
                  {t.planName} *
                </label>
                <input
                  type="text"
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">
                  {t.planSlug} (Immutable)
                </label>
                <input
                  type="text"
                  disabled
                  value={plan.slug}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 bg-slate-50 font-mono text-slate-500 cursor-not-allowed"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 mb-1">
                Description
              </label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={2}
                className="w-full text-xs p-3 rounded-xl border border-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>

          <div className="space-y-4 pt-4 border-t border-slate-100">
            <h2 className="text-sm font-bold text-slate-900 uppercase tracking-wider text-[11px] text-slate-400">
              Pricing Configuration
            </h2>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">
                  Price Amount (Cents)
                </label>
                <input
                  type="number"
                  min="0"
                  value={priceAmount}
                  onChange={(e) => setPriceAmount(Number(e.target.value))}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <span className="text-[10px] text-slate-400 font-mono mt-1 block">
                  Displayed as {(priceAmount / 100).toFixed(2)} {currency}
                </span>
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">
                  Currency
                </label>
                <input
                  type="text"
                  maxLength={3}
                  value={currency}
                  onChange={(e) => setCurrency(e.target.value.toUpperCase())}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">
                  {t.billingPeriod}
                </label>
                <select
                  value={billingInterval}
                  onChange={(e) => setBillingInterval(e.target.value)}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 bg-slate-50 font-medium focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  <option value="monthly">Monthly</option>
                  <option value="annual">Annual</option>
                  <option value="quarterly">Quarterly</option>
                </select>
              </div>
            </div>
          </div>

          <div className="space-y-4 pt-4 border-t border-slate-100">
            <h2 className="text-sm font-bold text-slate-900 uppercase tracking-wider text-[11px] text-slate-400">
              Structured Resource Limits (-1 for unlimited)
            </h2>

            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">{t.maxClients}</label>
                <input
                  type="number"
                  value={maxClients}
                  onChange={(e) => setMaxClients(Number(e.target.value))}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">{t.maxAssets}</label>
                <input
                  type="number"
                  value={maxAssets}
                  onChange={(e) => setMaxAssets(Number(e.target.value))}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">{t.maxMonitors}</label>
                <input
                  type="number"
                  value={maxMonitors}
                  onChange={(e) => setMaxMonitors(Number(e.target.value))}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">{t.maxHeartbeats}</label>
                <input
                  type="number"
                  value={maxHeartbeats}
                  onChange={(e) => setMaxHeartbeats(Number(e.target.value))}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">{t.maxTeamMembers}</label>
                <input
                  type="number"
                  value={maxMembers}
                  onChange={(e) => setMaxMembers(Number(e.target.value))}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">
                  {t.minCheckInterval} (Seconds &ge; 60)
                </label>
                <input
                  type="number"
                  min="60"
                  value={minCheckInterval}
                  onChange={(e) => setMinCheckInterval(Number(e.target.value))}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">
                  {t.retentionDays} (Days)
                </label>
                <input
                  type="number"
                  min="1"
                  value={retentionDays}
                  onChange={(e) => setRetentionDays(Number(e.target.value))}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            </div>
          </div>

          <div className="space-y-4 pt-4 border-t border-slate-100">
            <h2 className="text-sm font-bold text-slate-900 uppercase tracking-wider text-[11px] text-slate-400">
              Feature Entitlements
            </h2>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <label className="flex items-start gap-3 p-4 rounded-xl border border-slate-200 cursor-pointer hover:bg-slate-50/50 transition-colors">
                <input
                  type="checkbox"
                  checked={customBranding}
                  onChange={(e) => setCustomBranding(e.target.checked)}
                  className="mt-1 w-4 h-4 rounded text-blue-600 focus:ring-blue-500"
                />
                <div>
                  <span className="text-xs font-bold text-slate-900 block">{t.customBrandingFeature}</span>
                  <span className="text-[11px] text-slate-500">
                    Allows tenant organizations to customize agency logo and care report branding.
                  </span>
                </div>
              </label>

              <label className="flex items-start gap-3 p-4 rounded-xl border border-slate-200 cursor-pointer hover:bg-slate-50/50 transition-colors">
                <input
                  type="checkbox"
                  checked={careReports}
                  onChange={(e) => setCareReports(e.target.checked)}
                  className="mt-1 w-4 h-4 rounded text-blue-600 focus:ring-blue-500"
                />
                <div>
                  <span className="text-xs font-bold text-slate-900 block">{t.careReportsFeature}</span>
                  <span className="text-[11px] text-slate-500">
                    Enables monthly executive maintenance reports generation and public token sharing.
                  </span>
                </div>
              </label>
            </div>
          </div>

          <div className="pt-6 border-t border-slate-100 flex items-center justify-end gap-3">
            <Button
              type="button"
              variant="secondary"
              onClick={() => router.push("/platform-admin/plans")}
              disabled={saving}
            >
              Cancel
            </Button>
            <Button type="submit" variant="primary" loading={saving} className="gap-1.5">
              <span>{isPublished ? "Save As New Revision" : "Save Changes"}</span>
              {isRTL ? <ArrowLeft className="w-4 h-4" /> : <ArrowRight className="w-4 h-4" />}
            </Button>
          </div>
        </form>
      </div>

      {/* Version History Table */}
      <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
        <div className="p-5 border-b border-slate-100 flex items-center justify-between">
          <h3 className="text-sm font-bold text-slate-900 flex items-center gap-2">
            <History className="w-4 h-4 text-slate-500" />
            <span>Version Audit History ({versions.length})</span>
          </h3>
          <span className="text-xs text-slate-400 font-mono">Plan ID: {plan.id}</span>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-slate-50/80 border-b border-slate-200/80 text-[11px] font-bold uppercase tracking-wider text-slate-500">
                <th className="py-3 px-6">Version</th>
                <th className="py-3 px-4">Status</th>
                <th className="py-3 px-4">Price</th>
                <th className="py-3 px-4">Limits (c/a/m/h)</th>
                <th className="py-3 px-4">Created Date</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 text-xs">
              {versions.map((v) => (
                <tr key={v.id} className="hover:bg-slate-50/60 transition-colors">
                  <td className="py-3 px-6 font-bold text-slate-900 font-mono">
                    v{v.version}
                  </td>
                  <td className="py-3 px-4">
                    {v.status === "published" ? (
                      <span className="bg-emerald-50 text-emerald-700 border border-emerald-200 px-2 py-0.5 rounded-full text-[10px] font-bold">
                        Published
                      </span>
                    ) : v.status === "draft" ? (
                      <span className="bg-amber-50 text-amber-700 border border-amber-200 px-2 py-0.5 rounded-full text-[10px] font-bold">
                        Draft
                      </span>
                    ) : (
                      <span className="bg-slate-100 text-slate-600 px-2 py-0.5 rounded-full text-[10px] font-bold">
                        Archived
                      </span>
                    )}
                  </td>
                  <td className="py-3 px-4 font-mono text-slate-700">
                    {(v.price_amount / 100).toFixed(2)} {v.currency} / {v.billing_interval}
                  </td>
                  <td className="py-3 px-4 font-mono text-slate-500 text-[11px]">
                    {v.max_clients}/{v.max_assets}/{v.max_monitors}/{v.max_heartbeats}
                  </td>
                  <td className="py-3 px-4 text-slate-500 font-mono text-[11px]">
                    {new Date(v.created_at).toLocaleString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
