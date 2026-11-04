"use client";

import React, { useEffect, useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
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
  Globe,
  Coins,
  Plus,
  Trash2,
  ExternalLink,
} from "lucide-react";
import { apiFetch } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/Button";
import { LoadingSkeleton } from "@/components/ui/LoadingSkeleton";
import { ErrorState } from "@/components/ui/ErrorState";

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

interface OfferVersion {
  id: string;
  version: number;
  status: string;
  price_amount: number;
  currency: string;
  display_denomination: string;
  denomination_ratio: number;
  billing_interval: string;
  display_price: number;
  formatted_price: string;
  created_at: string;
  published_at?: string;
}

interface OfferItem {
  id: string;
  plan_id: string;
  locale: string;
  market: string;
  name: string;
  description: string;
  features_summary: string[];
  status: string;
  versions_count: number;
  latest_version?: OfferVersion;
}

export default function PlatformAdminPlanDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { t, isRTL, formatPersianNumber } = useI18n();

  const [activeTab, setActiveTab] = useState<"entitlements" | "offer_en" | "offer_fa" | "history">("entitlements");
  const [detail, setDetail] = useState<PlanDetail | null>(null);
  const [offers, setOffers] = useState<OfferItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Base plan form values
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [maxClients, setMaxClients] = useState<number>(5);
  const [maxAssets, setMaxAssets] = useState<number>(10);
  const [maxMonitors, setMaxMonitors] = useState<number>(10);
  const [maxHeartbeats, setMaxHeartbeats] = useState<number>(5);
  const [maxMembers, setMaxMembers] = useState<number>(5);
  const [minCheckInterval, setMinCheckInterval] = useState<number>(60);
  const [retentionDays, setRetentionDays] = useState<number>(30);
  const [customBranding, setCustomBranding] = useState(false);
  const [careReports, setCareReports] = useState(true);

  // English Offer state
  const [enName, setEnName] = useState("");
  const [enDesc, setEnDesc] = useState("");
  const [enPriceDisplay, setEnPriceDisplay] = useState<number>(29);
  const [enFeatures, setEnFeatures] = useState<string[]>([]);
  const [newEnFeature, setNewEnFeature] = useState("");

  // Persian Offer state
  const [faName, setFaName] = useState("");
  const [faDesc, setFaDesc] = useState("");
  const [faPriceToman, setFaPriceToman] = useState<number>(290000);
  const [faDenomRatio, setFaDenomRatio] = useState<number>(10);
  const [faFeatures, setFaFeatures] = useState<string[]>([]);
  const [newFaFeature, setNewFaFeature] = useState("");

  const [saving, setSaving] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [toastMessage, setToastMessage] = useState<string | null>(null);

  const fetchDetail = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const [data, offersRes] = await Promise.all([
        apiFetch<PlanDetail>(`/api/v1/platform-admin/plans/${id}`),
        apiFetch<{ offers: OfferItem[] }>(`/api/v1/platform-admin/plans/${id}/offers`).catch(() => ({ offers: [] })),
      ]);

      setDetail(data);
      setOffers(offersRes.offers || []);

      setName(data.plan.name);
      setDescription(data.plan.description || "");

      if (data.versions && data.versions.length > 0) {
        const latest = data.versions[0];
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

      // Populate English Offer if present
      const en = offersRes.offers?.find((o) => o.locale === "en");
      if (en) {
        setEnName(en.name);
        setEnDesc(en.description || "");
        setEnFeatures(en.features_summary || []);
        if (en.latest_version) {
          // If price_amount is in cents (e.g. 2900), convert to display units ($29)
          setEnPriceDisplay(en.latest_version.display_price > 0 ? en.latest_version.display_price / 100 : 29);
        }
      } else {
        setEnName(data.plan.name);
        setEnDesc(data.plan.description || "");
      }

      // Populate Persian Offer if present
      const fa = offersRes.offers?.find((o) => o.locale === "fa");
      if (fa) {
        setFaName(fa.name);
        setFaDesc(fa.description || "");
        setFaFeatures(fa.features_summary || []);
        if (fa.latest_version) {
          setFaPriceToman(fa.latest_version.display_price);
          setFaDenomRatio(fa.latest_version.denomination_ratio || 10);
        }
      } else {
        setFaName(data.plan.name);
        setFaDesc(data.plan.description || "");
      }
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Failed to load plan details");
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    if (id) fetchDetail();
  }, [id, fetchDetail]);

  const existingEnOffer = offers.find((o) => o.locale === "en");
  const existingFaOffer = offers.find((o) => o.locale === "fa");

  // Save Base Entitlements
  const handleSaveEntitlements = async (e: React.FormEvent) => {
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

      setToastMessage("Shared plan entitlements updated successfully.");
      fetchDetail();
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Failed to update plan");
    } finally {
      setSaving(false);
    }
  };

  // Publish Base Plan
  const handlePublishPlan = async () => {
    try {
      setPublishing(true);
      await apiFetch(`/api/v1/platform-admin/plans/${id}/publish`, { method: "POST" });
      setToastMessage("Base plan version published successfully.");
      fetchDetail();
    } catch (err: unknown) {
      const e = err as Error;
      alert(`Publish failed: ${e.message}`);
    } finally {
      setPublishing(false);
    }
  };

  // Save / Update English Offer
  const handleSaveEnOffer = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!enName.trim()) {
      setError("English offer title is required.");
      return;
    }

    try {
      setSaving(true);
      setError(null);
      const displayPriceCents = Math.round(Number(enPriceDisplay) * 100);

      if (existingEnOffer) {
        await apiFetch(`/api/v1/platform-admin/offers/${existingEnOffer.id}`, {
          method: "PUT",
          body: JSON.stringify({
            name: enName.trim(),
            description: enDesc.trim(),
            features_summary: enFeatures,
            display_price: displayPriceCents,
            currency: "USD",
            display_denomination: "usd",
            denomination_ratio: 1,
            billing_interval: "monthly",
          }),
        });
        setToastMessage("English localized offer draft saved.");
      } else {
        await apiFetch(`/api/v1/platform-admin/plans/${id}/offers`, {
          method: "POST",
          body: JSON.stringify({
            locale: "en",
            market: "global",
            name: enName.trim(),
            description: enDesc.trim(),
            features_summary: enFeatures,
            display_price: displayPriceCents,
            currency: "USD",
            display_denomination: "usd",
            denomination_ratio: 1,
            billing_interval: "monthly",
          }),
        });
        setToastMessage("English localized offer created.");
      }
      fetchDetail();
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Failed to save English offer");
    } finally {
      setSaving(false);
    }
  };

  // Publish English Offer
  const handlePublishEnOffer = async () => {
    if (!existingEnOffer) return;
    try {
      setPublishing(true);
      await apiFetch(`/api/v1/platform-admin/offers/${existingEnOffer.id}/publish`, { method: "POST" });
      setToastMessage("English offer published as immutable version.");
      fetchDetail();
    } catch (err: unknown) {
      const e = err as Error;
      alert(`Publish failed: ${e.message}`);
    } finally {
      setPublishing(false);
    }
  };

  // Save / Update Persian Offer
  const handleSaveFaOffer = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!faName.trim()) {
      setError("عنوان آفر فارسی الزامی است.");
      return;
    }

    try {
      setSaving(true);
      setError(null);
      const displayPriceToman = Math.round(Number(faPriceToman));

      if (existingFaOffer) {
        await apiFetch(`/api/v1/platform-admin/offers/${existingFaOffer.id}`, {
          method: "PUT",
          body: JSON.stringify({
            name: faName.trim(),
            description: faDesc.trim(),
            features_summary: faFeatures,
            display_price: displayPriceToman,
            currency: "IRR",
            display_denomination: "toman",
            denomination_ratio: Number(faDenomRatio) || 10,
            billing_interval: "monthly",
          }),
        });
        setToastMessage("آفر بومی فارسی ذخیره شد.");
      } else {
        await apiFetch(`/api/v1/platform-admin/plans/${id}/offers`, {
          method: "POST",
          body: JSON.stringify({
            locale: "fa",
            market: "ir",
            name: faName.trim(),
            description: faDesc.trim(),
            features_summary: faFeatures,
            display_price: displayPriceToman,
            currency: "IRR",
            display_denomination: "toman",
            denomination_ratio: Number(faDenomRatio) || 10,
            billing_interval: "monthly",
          }),
        });
        setToastMessage("آفر بومی فارسی ایجاد شد.");
      }
      fetchDetail();
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "خطا در ذخیره آفر فارسی");
    } finally {
      setSaving(false);
    }
  };

  // Publish Persian Offer
  const handlePublishFaOffer = async () => {
    if (!existingFaOffer) return;
    try {
      setPublishing(true);
      await apiFetch(`/api/v1/platform-admin/offers/${existingFaOffer.id}/publish`, { method: "POST" });
      setToastMessage("نسخه آفر فارسی منتشر شد و غیرقابل تغییر گردید.");
      fetchDetail();
    } catch (err: unknown) {
      const e = err as Error;
      alert(`خطا در انتشار: ${e.message}`);
    } finally {
      setPublishing(false);
    }
  };

  if (loading) {
    return (
      <div className="space-y-6">
        <LoadingSkeleton className="h-32 w-full rounded-2xl" />
        <LoadingSkeleton className="h-96 w-full rounded-2xl" />
      </div>
    );
  }

  if (error && !detail) {
    return <ErrorState title="Failed to Load Plan" error={error} onRetry={fetchDetail} />;
  }

  if (!detail) return null;

  const currentVer = detail.versions[0];
  const isPlanDraft = detail.plan.status === "draft";

  return (
    <div className="space-y-6">
      {/* Toast */}
      {toastMessage && (
        <div className="p-4 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-900 text-xs flex items-center justify-between shadow-sm animate-in fade-in">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
            <span className="font-medium">{toastMessage}</span>
          </div>
          <button
            onClick={() => setToastMessage(null)}
            className="text-emerald-700 hover:text-emerald-900 font-bold text-sm ml-4"
          >
            &times;
          </button>
        </div>
      )}

      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-xs font-medium text-slate-500 mb-1">
            <Link href="/platform-admin" className="hover:text-slate-800">Platform Admin</Link>
            <span>/</span>
            <Link href="/platform-admin/plans" className="hover:text-slate-800">Plans & Catalogs</Link>
            <span>/</span>
            <span className="text-slate-800 font-semibold">{detail.plan.name}</span>
          </div>
          <h1 className="text-xl sm:text-2xl font-black text-slate-900 tracking-tight flex items-center gap-2.5">
            <Palette className="w-6 h-6 text-emerald-600" />
            <span>{detail.plan.name}</span>
            <span className="text-xs font-mono font-bold bg-slate-100 text-slate-600 px-2.5 py-1 rounded-lg">
              /{detail.plan.slug}
            </span>
          </h1>
        </div>

        <div className="flex items-center gap-2">
          {isPlanDraft ? (
            <Button
              variant="primary"
              size="sm"
              loading={publishing}
              onClick={handlePublishPlan}
              className="gap-1.5 shadow-sm"
            >
              <CheckCircle2 className="w-4 h-4" />
              <span>Publish Base Plan (v{detail.plan.current_version})</span>
            </Button>
          ) : (
            <span className="inline-flex items-center gap-1.5 bg-emerald-50 text-emerald-700 border border-emerald-200 px-3 py-1.5 rounded-xl font-bold text-xs">
              <CheckCircle2 className="w-4 h-4" />
              <span>Base Plan Published (v{detail.plan.current_version})</span>
            </span>
          )}
        </div>
      </div>

      {/* Navigation Tabs */}
      <div className="flex items-center gap-2 border-b border-slate-200 pb-2 overflow-x-auto">
        <button
          onClick={() => setActiveTab("entitlements")}
          className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold transition-all whitespace-nowrap ${
            activeTab === "entitlements"
              ? "bg-slate-900 text-white shadow-sm"
              : "text-slate-600 hover:text-slate-900 hover:bg-slate-100"
          }`}
        >
          <Layers className="w-4 h-4" />
          <span>Shared Entitlements</span>
        </button>

        <button
          onClick={() => setActiveTab("offer_en")}
          className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold transition-all whitespace-nowrap ${
            activeTab === "offer_en"
              ? "bg-emerald-600 text-white shadow-sm"
              : "text-slate-600 hover:text-slate-900 hover:bg-slate-100"
          }`}
        >
          <Globe className="w-4 h-4" />
          <span>English Offer (USD)</span>
          {existingEnOffer ? (
            <span className={`px-2 py-0.5 rounded-full text-[10px] ${existingEnOffer.status === "published" ? "bg-emerald-700 text-emerald-100" : "bg-amber-100 text-amber-800"}`}>
              {existingEnOffer.status}
            </span>
          ) : (
            <span className="px-2 py-0.5 rounded-full text-[10px] bg-slate-200 text-slate-600">Not Created</span>
          )}
        </button>

        <button
          onClick={() => setActiveTab("offer_fa")}
          className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold transition-all whitespace-nowrap ${
            activeTab === "offer_fa"
              ? "bg-emerald-600 text-white shadow-sm"
              : "text-slate-600 hover:text-slate-900 hover:bg-slate-100"
          }`}
        >
          <Coins className="w-4 h-4" />
          <span>آفر فارسی (تومان)</span>
          {existingFaOffer ? (
            <span className={`px-2 py-0.5 rounded-full text-[10px] ${existingFaOffer.status === "published" ? "bg-emerald-700 text-emerald-100" : "bg-amber-100 text-amber-800"}`}>
              {existingFaOffer.status}
            </span>
          ) : (
            <span className="px-2 py-0.5 rounded-full text-[10px] bg-slate-200 text-slate-600">ایجاد نشده</span>
          )}
        </button>

        <button
          onClick={() => setActiveTab("history")}
          className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold transition-all whitespace-nowrap ${
            activeTab === "history"
              ? "bg-slate-900 text-white shadow-sm"
              : "text-slate-600 hover:text-slate-900 hover:bg-slate-100"
          }`}
        >
          <History className="w-4 h-4" />
          <span>Version History</span>
        </button>
      </div>

      {/* TAB 1: SHARED ENTITLEMENTS */}
      {activeTab === "entitlements" && (
        <form onSubmit={handleSaveEntitlements} className="space-y-6">
          <div className="bg-white rounded-2xl border border-slate-200 p-6 shadow-sm space-y-5">
            <h3 className="text-sm font-bold text-slate-900 uppercase tracking-wider text-slate-400">
              Base Plan Identity
            </h3>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">Plan Display Name</label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full text-xs px-3.5 py-2.5 rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">Internal Slug (Read-only)</label>
                <input
                  type="text"
                  value={detail.plan.slug}
                  disabled
                  className="w-full text-xs px-3.5 py-2.5 rounded-xl border border-slate-200 bg-slate-100 text-slate-500 font-mono"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 mb-1">Internal Description</label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={2}
                className="w-full text-xs px-3.5 py-2.5 rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500"
              />
            </div>
          </div>

          <div className="bg-white rounded-2xl border border-slate-200 p-6 shadow-sm space-y-5">
            <h3 className="text-sm font-bold text-slate-900 uppercase tracking-wider text-slate-400">
              Technical Quotas & Limits
            </h3>

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">Max Clients</label>
                <input
                  type="number"
                  min="1"
                  value={maxClients}
                  onChange={(e) => setMaxClients(Number(e.target.value))}
                  className="w-full text-xs px-3.5 py-2.5 rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500 font-mono"
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">Max Assets</label>
                <input
                  type="number"
                  min="1"
                  value={maxAssets}
                  onChange={(e) => setMaxAssets(Number(e.target.value))}
                  className="w-full text-xs px-3.5 py-2.5 rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500 font-mono"
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">Max Monitors</label>
                <input
                  type="number"
                  min="1"
                  value={maxMonitors}
                  onChange={(e) => setMaxMonitors(Number(e.target.value))}
                  className="w-full text-xs px-3.5 py-2.5 rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500 font-mono"
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">Max Heartbeats</label>
                <input
                  type="number"
                  min="1"
                  value={maxHeartbeats}
                  onChange={(e) => setMaxHeartbeats(Number(e.target.value))}
                  className="w-full text-xs px-3.5 py-2.5 rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500 font-mono"
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">Max Team Members</label>
                <input
                  type="number"
                  min="1"
                  value={maxMembers}
                  onChange={(e) => setMaxMembers(Number(e.target.value))}
                  className="w-full text-xs px-3.5 py-2.5 rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500 font-mono"
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">Min Check Interval (sec)</label>
                <input
                  type="number"
                  min="60"
                  step="10"
                  value={minCheckInterval}
                  onChange={(e) => setMinCheckInterval(Number(e.target.value))}
                  className="w-full text-xs px-3.5 py-2.5 rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500 font-mono"
                  required
                />
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-3 border-t border-slate-100">
              <label className="flex items-center gap-3 p-3 rounded-xl border border-slate-200 cursor-pointer hover:bg-slate-50 transition-colors">
                <input
                  type="checkbox"
                  checked={customBranding}
                  onChange={(e) => setCustomBranding(e.target.checked)}
                  className="w-4 h-4 rounded text-emerald-600 focus:ring-emerald-500"
                />
                <div>
                  <span className="text-xs font-bold text-slate-900 block">Custom Client Branding</span>
                  <span className="text-[11px] text-slate-500">Allow agency custom logos, domain and theme</span>
                </div>
              </label>

              <label className="flex items-center gap-3 p-3 rounded-xl border border-slate-200 cursor-pointer hover:bg-slate-50 transition-colors">
                <input
                  type="checkbox"
                  checked={careReports}
                  onChange={(e) => setCareReports(e.target.checked)}
                  className="w-4 h-4 rounded text-emerald-600 focus:ring-emerald-500"
                />
                <div>
                  <span className="text-xs font-bold text-slate-900 block">Public Care Reports</span>
                  <span className="text-[11px] text-slate-500">Enable shareable public monthly care reports</span>
                </div>
              </label>
            </div>
          </div>

          <div className="flex items-center justify-end gap-3">
            <Button type="submit" variant="primary" size="sm" loading={saving} className="gap-1.5">
              <CheckCircle2 className="w-4 h-4" />
              <span>Save Entitlements Draft</span>
            </Button>
          </div>
        </form>
      )}

      {/* TAB 2: ENGLISH LOCALIZED OFFER */}
      {activeTab === "offer_en" && (
        <form onSubmit={handleSaveEnOffer} className="space-y-6">
          <div className="bg-white rounded-2xl border border-slate-200 p-6 shadow-sm space-y-5">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-base font-bold text-slate-900 flex items-center gap-2">
                  <Globe className="w-5 h-5 text-emerald-600" />
                  <span>English / Global Localized Offer</span>
                </h3>
                <p className="text-xs text-slate-500 mt-0.5">
                  Target market: Global. Currency: USD ($).
                </p>
              </div>

              {existingEnOffer && (
                <div className="flex items-center gap-2">
                  {existingEnOffer.status === "draft" && (
                    <Button
                      type="button"
                      variant="primary"
                      size="sm"
                      loading={publishing}
                      onClick={handlePublishEnOffer}
                    >
                      Publish English Offer
                    </Button>
                  )}
                  <span className="px-3 py-1 rounded-xl text-xs font-bold bg-slate-100 text-slate-700">
                    Status: {existingEnOffer.status.toUpperCase()}
                  </span>
                </div>
              )}
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">Localized Plan Title</label>
                <input
                  type="text"
                  value={enName}
                  onChange={(e) => setEnName(e.target.value)}
                  placeholder="e.g. Professional Plan"
                  className="w-full text-xs px-3.5 py-2.5 rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">Monthly Price (USD $)</label>
                <div className="relative">
                  <span className="absolute left-3 top-2.5 text-slate-400 font-bold">$</span>
                  <input
                    type="number"
                    min="0"
                    step="1"
                    value={enPriceDisplay}
                    onChange={(e) => setEnPriceDisplay(Number(e.target.value))}
                    className="w-full text-xs pl-7 pr-3.5 py-2.5 rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500 font-mono font-bold"
                    required
                  />
                </div>
                <p className="text-[11px] text-slate-400 mt-1">
                  Preview: ${Number(enPriceDisplay).toFixed(2)} / month
                </p>
              </div>
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 mb-1">Localized Description</label>
              <textarea
                value={enDesc}
                onChange={(e) => setEnDesc(e.target.value)}
                placeholder="Localized pitch displayed on the pricing table..."
                rows={2}
                className="w-full text-xs px-3.5 py-2.5 rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500"
              />
            </div>

            {/* Feature Bullets */}
            <div>
              <label className="block text-xs font-bold text-slate-700 mb-1">Localized Highlights</label>
              <div className="flex gap-2 mb-2">
                <input
                  type="text"
                  value={newEnFeature}
                  onChange={(e) => setNewEnFeature(e.target.value)}
                  placeholder="e.g. 24/7 Monitoring with 60s intervals"
                  className="flex-1 text-xs px-3.5 py-2 rounded-xl border border-slate-300"
                />
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={() => {
                    if (newEnFeature.trim()) {
                      setEnFeatures([...enFeatures, newEnFeature.trim()]);
                      setNewEnFeature("");
                    }
                  }}
                >
                  <Plus className="w-3.5 h-3.5 mr-1" /> Add
                </Button>
              </div>

              <div className="space-y-1.5">
                {enFeatures.map((f, i) => (
                  <div key={i} className="flex items-center justify-between p-2 rounded-lg bg-slate-50 border border-slate-200 text-xs text-slate-700">
                    <span>{f}</span>
                    <button
                      type="button"
                      onClick={() => setEnFeatures(enFeatures.filter((_, idx) => idx !== i))}
                      className="text-rose-500 hover:text-rose-700"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <div className="flex items-center justify-end gap-3">
            <Button type="submit" variant="primary" size="sm" loading={saving} className="gap-1.5">
              <CheckCircle2 className="w-4 h-4" />
              <span>{existingEnOffer ? "Save English Draft" : "Create English Offer"}</span>
            </Button>
          </div>
        </form>
      )}

      {/* TAB 3: PERSIAN LOCALIZED OFFER */}
      {activeTab === "offer_fa" && (
        <form onSubmit={handleSaveFaOffer} className="space-y-6">
          <div className="bg-white rounded-2xl border border-slate-200 p-6 shadow-sm space-y-5">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="text-base font-bold text-slate-900 flex items-center gap-2">
                  <Coins className="w-5 h-5 text-emerald-600" />
                  <span>آفر بومی فارسی (ایران / تومان)</span>
                </h3>
                <p className="text-xs text-slate-500 mt-0.5">
                  ارز پایه در دیتابیس: ریال (IRR). واحد نمایش در سیستم: تومان.
                </p>
              </div>

              {existingFaOffer && (
                <div className="flex items-center gap-2">
                  {existingFaOffer.status === "draft" && (
                    <Button
                      type="button"
                      variant="primary"
                      size="sm"
                      loading={publishing}
                      onClick={handlePublishFaOffer}
                    >
                      انتشار آفر فارسی
                    </Button>
                  )}
                  <span className="px-3 py-1 rounded-xl text-xs font-bold bg-slate-100 text-slate-700">
                    وضعیت: {existingFaOffer.status.toUpperCase()}
                  </span>
                </div>
              )}
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">عنوان پلن به فارسی</label>
                <input
                  type="text"
                  value={faName}
                  onChange={(e) => setFaName(e.target.value)}
                  placeholder="مثلاً: پلن حرفه‌ای"
                  className="w-full text-xs px-3.5 py-2.5 rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500"
                  required
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">
                  مبلغ به تومان (ورودی کاربر)
                </label>
                <input
                  type="number"
                  min="0"
                  step="1000"
                  value={faPriceToman}
                  onChange={(e) => setFaPriceToman(Number(e.target.value))}
                  className="w-full text-xs px-3.5 py-2.5 rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500 font-mono font-bold"
                  required
                />
                {/* Transparency Preview */}
                <div className="mt-2 p-3 rounded-xl bg-slate-50 border border-slate-200 text-xs space-y-1">
                  <div className="flex justify-between">
                    <span className="text-slate-500">نمایش به مشتری:</span>
                    <span className="font-bold text-emerald-700">
                      {formatPersianNumber(faPriceToman)} تومان / ماه
                    </span>
                  </div>
                  <div className="flex justify-between font-mono">
                    <span className="text-slate-500">ذخیره در دیتابیس (IRR):</span>
                    <span className="text-slate-800">
                      {formatPersianNumber(faPriceToman * faDenomRatio)} ریال (ضریب {faDenomRatio})
                    </span>
                  </div>
                </div>
              </div>
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 mb-1">توضیح آفر به فارسی</label>
              <textarea
                value={faDesc}
                onChange={(e) => setFaDesc(e.target.value)}
                placeholder="توضیحات کوتاه برای نمایش در جدول پلن‌ها..."
                rows={2}
                className="w-full text-xs px-3.5 py-2.5 rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-emerald-500"
              />
            </div>

            {/* Feature Bullets */}
            <div>
              <label className="block text-xs font-bold text-slate-700 mb-1">ویژگی‌های برجسته فارسی</label>
              <div className="flex gap-2 mb-2">
                <input
                  type="text"
                  value={newFaFeature}
                  onChange={(e) => setNewFaFeature(e.target.value)}
                  placeholder="مثال: گزارش دوره‌ای ماهانه با برند اختصاصی"
                  className="flex-1 text-xs px-3.5 py-2 rounded-xl border border-slate-300"
                />
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={() => {
                    if (newFaFeature.trim()) {
                      setFaFeatures([...faFeatures, newFaFeature.trim()]);
                      setNewFaFeature("");
                    }
                  }}
                >
                  <Plus className="w-3.5 h-3.5 mr-1" /> افزودن
                </Button>
              </div>

              <div className="space-y-1.5">
                {faFeatures.map((f, i) => (
                  <div key={i} className="flex items-center justify-between p-2 rounded-lg bg-slate-50 border border-slate-200 text-xs text-slate-700">
                    <span>{f}</span>
                    <button
                      type="button"
                      onClick={() => setFaFeatures(faFeatures.filter((_, idx) => idx !== i))}
                      className="text-rose-500 hover:text-rose-700"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <div className="flex items-center justify-end gap-3">
            <Button type="submit" variant="primary" size="sm" loading={saving} className="gap-1.5">
              <CheckCircle2 className="w-4 h-4" />
              <span>{existingFaOffer ? "ذخیره پیش‌نویس آفر فارسی" : "ایجاد آفر فارسی"}</span>
            </Button>
          </div>
        </form>
      )}

      {/* TAB 4: VERSION HISTORY */}
      {activeTab === "history" && (
        <div className="space-y-6">
          <div className="bg-white rounded-2xl border border-slate-200 p-6 shadow-sm">
            <h3 className="text-sm font-bold text-slate-900 mb-4 flex items-center gap-2">
              <History className="w-4 h-4 text-slate-500" />
              <span>Immutable Base Plan Versions</span>
            </h3>

            <div className="space-y-3">
              {detail.versions.map((ver) => (
                <div
                  key={ver.id}
                  className="p-4 rounded-xl border border-slate-200 bg-slate-50/50 flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs"
                >
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-bold text-slate-900">Version {ver.version}</span>
                      <span className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${ver.status === "published" ? "bg-emerald-100 text-emerald-800" : "bg-amber-100 text-amber-800"}`}>
                        {ver.status}
                      </span>
                    </div>
                    <div className="text-slate-500 mt-1 flex flex-wrap gap-x-4 gap-y-1">
                      <span>Clients: {ver.max_clients}</span>
                      <span>Assets: {ver.max_assets}</span>
                      <span>Monitors: {ver.max_monitors}</span>
                      <span>Heartbeats: {ver.max_heartbeats}</span>
                      <span>Retention: {ver.retention_days}d</span>
                    </div>
                  </div>

                  <div className="text-slate-400 font-mono text-[11px] self-end sm:self-auto">
                    {new Date(ver.created_at).toLocaleString()}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
