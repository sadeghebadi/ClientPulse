"use client";

import React, { useState } from "react";
import { useRouter } from "next/navigation";
import {
  Palette,
  Sparkles,
  Layers,
  Clock,
  ArrowLeft,
  ArrowRight,
  Shield,
  CheckCircle2,
} from "lucide-react";
import { apiFetch } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/Button";
import { Breadcrumbs } from "@/components/ui/PageHeader";

export default function PlatformAdminNewPlanPage() {
  const router = useRouter();
  const { t, isRTL } = useI18n();

  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [priceType, setPriceType] = useState<"unconfigured" | "free" | "paid">("unconfigured");
  const [priceAmount, setPriceAmount] = useState<string>("");
  const [currency, setCurrency] = useState<string>("");
  const [billingInterval, setBillingInterval] = useState<string>("");

  // Limits
  const [maxClients, setMaxClients] = useState<number>(5);
  const [maxAssets, setMaxAssets] = useState<number>(10);
  const [maxMonitors, setMaxMonitors] = useState<number>(10);
  const [maxHeartbeats, setMaxHeartbeats] = useState<number>(5);
  const [maxMembers, setMaxMembers] = useState<number>(5);
  const [minCheckInterval, setMinCheckInterval] = useState<number>(60);
  const [retentionDays, setRetentionDays] = useState<number>(30);

  // Features
  const [customBranding, setCustomBranding] = useState(false);
  const [careReports, setCareReports] = useState(true);

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleNameChange = (val: string) => {
    setName(val);
    if (!slug || slug === name.toLowerCase().replace(/[^a-z0-9]+/g, "-")) {
      setSlug(val.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, ""));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !slug.trim()) {
      setError("Plan name and slug are required.");
      return;
    }
    if (minCheckInterval < 60) {
      setError("Minimum check interval must be at least 60 seconds.");
      return;
    }

    try {
      setLoading(true);
      setError(null);

      let finalPrice: number | null = null;
      let finalCurrency: string | null = null;
      let finalInterval: string | null = null;

      if (priceType === "free") {
        finalPrice = 0;
        finalCurrency = currency.trim().toUpperCase() || "USD";
        finalInterval = billingInterval || "monthly";
      } else if (priceType === "paid") {
        if (!priceAmount || Number(priceAmount) <= 0) {
          setError("A paid plan requires a positive price amount.");
          setLoading(false);
          return;
        }
        if (!currency.trim()) {
          setError("A paid plan requires a valid 3-letter currency code.");
          setLoading(false);
          return;
        }
        if (!billingInterval) {
          setError("A paid plan requires a billing interval.");
          setLoading(false);
          return;
        }
        finalPrice = Number(priceAmount);
        finalCurrency = currency.trim().toUpperCase();
        finalInterval = billingInterval;
      }

      await apiFetch("/api/v1/platform-admin/plans", {
        method: "POST",
        body: JSON.stringify({
          name: name.trim(),
          slug: slug.trim(),
          description: description.trim(),
          price_amount: finalPrice,
          currency: finalCurrency,
          billing_interval: finalInterval,
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

      router.push("/platform-admin/plans");
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Failed to create subscription plan");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6 max-w-4xl mx-auto">
      <Breadcrumbs
        items={[
          { label: t.platformOverview, href: "/platform-admin" },
          { label: t.platformPlans, href: "/platform-admin/plans" },
          { label: t.createPlan },
        ]}
      />

      <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 sm:p-8">
        <div className="flex items-center gap-3 pb-6 border-b border-slate-100 mb-6">
          <div className="w-12 h-12 rounded-2xl bg-blue-50 border border-blue-200 flex items-center justify-center text-blue-600 shrink-0">
            <Palette className="w-6 h-6" />
          </div>
          <div>
            <h1 className="text-xl font-black text-slate-900">{t.createPlan}</h1>
            <p className="text-xs text-slate-500 mt-0.5">
              Draft a new commercial subscription tier with structured limits. You can review and publish it when ready.
            </p>
          </div>
        </div>

        {error && (
          <div className="p-4 rounded-xl bg-rose-50 border border-rose-200 text-rose-800 text-xs mb-6">
            <strong>Error:</strong> {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-8">
          {/* Basic Plan Metadata */}
          <div className="space-y-4">
            <h2 className="text-sm font-bold text-slate-900 uppercase tracking-wider text-[11px] text-slate-400">
              Basic Tier Information
            </h2>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">
                  {t.planName} *
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. Growth Agency"
                  value={name}
                  onChange={(e) => handleNameChange(e.target.value)}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">
                  {t.planSlug} (Unique Identifier) *
                </label>
                <input
                  type="text"
                  required
                  placeholder="growth-agency"
                  value={slug}
                  onChange={(e) => setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]+/g, ""))}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 mb-1">
                Description
              </label>
              <textarea
                placeholder="Brief summary of the plan audience and capabilities..."
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={2}
                className="w-full text-xs p-3 rounded-xl border border-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>

          {/* Pricing & Billing Period */}
          <div className="space-y-4 pt-4 border-t border-slate-100">
            <h2 className="text-sm font-bold text-slate-900 uppercase tracking-wider text-[11px] text-slate-400">
              Pricing Configuration
            </h2>

            <div className="flex gap-4 mb-2">
              <label className="flex items-center gap-2 text-xs font-medium cursor-pointer">
                <input
                  type="radio"
                  name="priceType"
                  value="unconfigured"
                  checked={priceType === "unconfigured"}
                  onChange={() => setPriceType("unconfigured")}
                  className="rounded-full border-slate-300 text-blue-600 focus:ring-blue-500"
                />
                Unconfigured (Draft only)
              </label>
              <label className="flex items-center gap-2 text-xs font-medium cursor-pointer">
                <input
                  type="radio"
                  name="priceType"
                  value="free"
                  checked={priceType === "free"}
                  onChange={() => setPriceType("free")}
                  className="rounded-full border-slate-300 text-blue-600 focus:ring-blue-500"
                />
                Free Tier ($0.00)
              </label>
              <label className="flex items-center gap-2 text-xs font-medium cursor-pointer">
                <input
                  type="radio"
                  name="priceType"
                  value="paid"
                  checked={priceType === "paid"}
                  onChange={() => setPriceType("paid")}
                  className="rounded-full border-slate-300 text-blue-600 focus:ring-blue-500"
                />
                Paid Commercial Plan
              </label>
            </div>

            {priceType === "paid" && (
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                <div>
                  <label className="block text-xs font-bold text-slate-700 mb-1">
                    Price Amount (Cents / Smallest Unit)
                  </label>
                  <input
                    type="number"
                    min="1"
                    placeholder="e.g. 2900"
                    value={priceAmount}
                    onChange={(e) => setPriceAmount(e.target.value)}
                    className="w-full text-xs p-3 rounded-xl border border-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                  {priceAmount && (
                    <span className="text-[10px] text-slate-400 font-mono mt-1 block">
                      Displayed as {(Number(priceAmount) / 100).toFixed(2)} {currency || "USD"}
                    </span>
                  )}
                </div>

                <div>
                  <label className="block text-xs font-bold text-slate-700 mb-1">
                    Currency (ISO 4217)
                  </label>
                  <input
                    type="text"
                    maxLength={3}
                    placeholder="USD"
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
                    <option value="">Select billing interval...</option>
                    <option value="monthly">Monthly</option>
                    <option value="yearly">Yearly</option>
                    <option value="quarterly">Quarterly</option>
                    <option value="one_time">One-time</option>
                  </select>
                </div>
              </div>
            )}

            {priceType === "unconfigured" && (
              <p className="text-xs text-slate-500 italic bg-slate-50 p-3 rounded-xl border border-slate-100">
                Plan will be created with no finalized price or currency. It cannot be published until explicit pricing is configured.
              </p>
            )}

            {priceType === "free" && (
              <p className="text-xs text-emerald-700 bg-emerald-50 p-3 rounded-xl border border-emerald-100 font-medium">
                Explicit Free plan with 0.00 price amount.
              </p>
            )}
          </div>

          {/* Structured Quotas & Limits */}
          <div className="space-y-4 pt-4 border-t border-slate-100">
            <h2 className="text-sm font-bold text-slate-900 uppercase tracking-wider text-[11px] text-slate-400">
              Structured Resource Limits (-1 for unlimited)
            </h2>

            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">
                  {t.maxClients}
                </label>
                <input
                  type="number"
                  value={maxClients}
                  onChange={(e) => setMaxClients(Number(e.target.value))}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">
                  {t.maxAssets}
                </label>
                <input
                  type="number"
                  value={maxAssets}
                  onChange={(e) => setMaxAssets(Number(e.target.value))}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">
                  {t.maxMonitors}
                </label>
                <input
                  type="number"
                  value={maxMonitors}
                  onChange={(e) => setMaxMonitors(Number(e.target.value))}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">
                  {t.maxHeartbeats}
                </label>
                <input
                  type="number"
                  value={maxHeartbeats}
                  onChange={(e) => setMaxHeartbeats(Number(e.target.value))}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 font-mono focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">
                  {t.maxTeamMembers}
                </label>
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

          {/* Feature Toggles */}
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
                  <span className="text-xs font-bold text-slate-900 block">
                    {t.customBrandingFeature}
                  </span>
                  <span className="text-[11px] text-slate-500">
                    Allows tenant organizations to customize agency logo, portal title, and care report header.
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
                  <span className="text-xs font-bold text-slate-900 block">
                    {t.careReportsFeature}
                  </span>
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
              disabled={loading}
            >
              Cancel
            </Button>
            <Button type="submit" variant="primary" loading={loading} className="gap-1.5">
              <span>Create Draft Plan</span>
              {isRTL ? <ArrowLeft className="w-4 h-4" /> : <ArrowRight className="w-4 h-4" />}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}

