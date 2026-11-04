"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import {
  Palette,
  PlusCircle,
  Eye,
  CheckCircle2,
  Archive,
  Clock,
  Layers,
  Sparkles,
  ArrowRight,
  ArrowLeft,
  AlertTriangle,
  Building,
  Globe,
  Tag,
  Coins,
  Shield,
  Filter,
} from "lucide-react";
import { apiFetch } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/Button";
import { LoadingSkeleton } from "@/components/ui/LoadingSkeleton";
import { ErrorState } from "@/components/ui/ErrorState";

interface PlanItem {
  id: string;
  name: string;
  slug: string;
  description: string;
  status: string;
  current_version: number;
  assigned_orgs_count: number;
  created_at: string;
  updated_at: string;
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
  published_at?: string;
}

interface CatalogOffer {
  id: string;
  plan_id: string;
  plan_name: string;
  plan_slug: string;
  locale: string;
  market: string;
  name: string;
  description: string;
  features_summary: string[];
  status: string;
  versions_count: number;
  latest_version?: OfferVersion;
}

export default function PlatformAdminPlansPage() {
  const { t, isRTL, language, formatPersianNumber } = useI18n();
  const [activeTab, setActiveTab] = useState<"all" | "en" | "fa">("all");
  const [plans, setPlans] = useState<PlanItem[]>([]);
  const [offers, setOffers] = useState<CatalogOffer[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Publish / Archive modal states
  const [selectedPlan, setSelectedPlan] = useState<PlanItem | null>(null);
  const [selectedOffer, setSelectedOffer] = useState<CatalogOffer | null>(null);
  const [modalAction, setModalAction] = useState<"publish" | "archive" | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [toastMessage, setToastMessage] = useState<string | null>(null);

  const fetchData = async () => {
    try {
      setLoading(true);
      setError(null);

      const [plansRes, catalogsRes] = await Promise.all([
        apiFetch<{ plans: PlanItem[] }>("/api/v1/platform-admin/plans"),
        apiFetch<{ offers: CatalogOffer[] }>("/api/v1/platform-admin/catalogs").catch(() => ({ offers: [] })),
      ]);

      setPlans(plansRes.plans || []);
      setOffers(catalogsRes.offers || []);
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Failed to load subscription plans and catalogs");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  const handleConfirmPlanAction = async () => {
    if (!selectedPlan || !modalAction) return;
    try {
      setActionLoading(true);
      const endpoint = `/api/v1/platform-admin/plans/${selectedPlan.id}/${modalAction}`;
      await apiFetch(endpoint, { method: "POST" });

      setToastMessage(
        modalAction === "publish"
          ? `Plan ${selectedPlan.name} published successfully.`
          : `Plan ${selectedPlan.name} has been archived.`
      );
      setSelectedPlan(null);
      setModalAction(null);
      fetchData();
    } catch (err: unknown) {
      const e = err as Error;
      alert(`Action failed: ${e.message}`);
    } finally {
      setActionLoading(false);
    }
  };

  const handleConfirmOfferAction = async () => {
    if (!selectedOffer || !modalAction) return;
    try {
      setActionLoading(true);
      const endpoint = `/api/v1/platform-admin/offers/${selectedOffer.id}/${modalAction}`;
      await apiFetch(endpoint, { method: "POST" });

      setToastMessage(
        modalAction === "publish"
          ? `Offer ${selectedOffer.name} (${selectedOffer.locale.toUpperCase()}) published successfully.`
          : `Offer ${selectedOffer.name} (${selectedOffer.locale.toUpperCase()}) has been archived.`
      );
      setSelectedOffer(null);
      setModalAction(null);
      fetchData();
    } catch (err: unknown) {
      const e = err as Error;
      alert(`Offer action failed: ${e.message}`);
    } finally {
      setActionLoading(false);
    }
  };

  const englishOffers = offers.filter((o) => o.locale === "en");
  const persianOffers = offers.filter((o) => o.locale === "fa");

  return (
    <div className="space-y-6">
      {/* Toast Notification */}
      {toastMessage && (
        <div className="p-4 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-900 text-xs flex items-center justify-between shadow-sm animate-in fade-in">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
            <span className="font-medium">{toastMessage}</span>
          </div>
          <button
            onClick={() => setToastMessage(null)}
            className="font-bold text-emerald-700 hover:text-emerald-900 text-sm ml-4"
          >
            &times;
          </button>
        </div>
      )}

      {/* Header and Create Button */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-xl sm:text-2xl font-black text-slate-900 tracking-tight flex items-center gap-2">
            <Palette className="w-6 h-6 text-emerald-600" />
            <span>Subscription Plans & Regional Catalogs</span>
          </h1>
          <p className="text-xs sm:text-sm text-slate-500 mt-1">
            Design shared technical quotas and manage independent localized offers for USD (Global) and IRR/Toman (Iran).
          </p>
        </div>

        <Link href="/platform-admin/plans/new" className="self-start sm:self-auto">
          <Button variant="primary" className="gap-1.5 shadow-sm">
            <PlusCircle className="w-4 h-4" />
            <span>{t.createPlan}</span>
          </Button>
        </Link>
      </div>

      {/* Catalog Selector Tabs */}
      <div className="flex items-center gap-2 border-b border-slate-200 pb-2">
        <button
          onClick={() => setActiveTab("all")}
          className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold transition-all ${
            activeTab === "all"
              ? "bg-slate-900 text-white shadow-sm"
              : "text-slate-600 hover:text-slate-900 hover:bg-slate-100"
          }`}
        >
          <Layers className="w-4 h-4" />
          <span>All Plans & Shared Quotas</span>
          <span className={`px-2 py-0.5 rounded-full text-[10px] ${activeTab === "all" ? "bg-slate-800 text-slate-200" : "bg-slate-200 text-slate-700"}`}>
            {plans.length}
          </span>
        </button>

        <button
          onClick={() => setActiveTab("en")}
          className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold transition-all ${
            activeTab === "en"
              ? "bg-emerald-600 text-white shadow-sm"
              : "text-slate-600 hover:text-slate-900 hover:bg-slate-100"
          }`}
        >
          <Globe className="w-4 h-4" />
          <span>English Catalog (USD / Global)</span>
          <span className={`px-2 py-0.5 rounded-full text-[10px] ${activeTab === "en" ? "bg-emerald-700 text-emerald-100" : "bg-slate-200 text-slate-700"}`}>
            {englishOffers.length}
          </span>
        </button>

        <button
          onClick={() => setActiveTab("fa")}
          className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold transition-all ${
            activeTab === "fa"
              ? "bg-emerald-600 text-white shadow-sm"
              : "text-slate-600 hover:text-slate-900 hover:bg-slate-100"
          }`}
        >
          <Coins className="w-4 h-4" />
          <span>کاتالوگ فارسی (تومان / ایران)</span>
          <span className={`px-2 py-0.5 rounded-full text-[10px] ${activeTab === "fa" ? "bg-emerald-700 text-emerald-100" : "bg-slate-200 text-slate-700"}`}>
            {persianOffers.length}
          </span>
        </button>
      </div>

      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
          <LoadingSkeleton className="h-56 w-full rounded-2xl" />
          <LoadingSkeleton className="h-56 w-full rounded-2xl" />
          <LoadingSkeleton className="h-56 w-full rounded-2xl" />
        </div>
      ) : error ? (
        <ErrorState title="Failed to Load Plans" error={error} onRetry={fetchData} />
      ) : (
        <>
          {/* TAB 1: ALL BASE PLANS */}
          {activeTab === "all" && (
            <div>
              {plans.length === 0 ? (
                <div className="bg-white rounded-2xl border border-slate-200 p-12 text-center shadow-sm">
                  <Palette className="w-12 h-12 text-slate-300 mx-auto mb-3" />
                  <h3 className="text-base font-bold text-slate-900">No subscription plans created yet</h3>
                  <p className="text-xs text-slate-500 max-w-sm mx-auto mt-1 mb-6">
                    Create your base SaaS tiers to define shared technical capabilities and limits.
                  </p>
                  <Link href="/platform-admin/plans/new">
                    <Button variant="primary" className="gap-1.5">
                      <PlusCircle className="w-4 h-4" />
                      <span>{t.createPlan}</span>
                    </Button>
                  </Link>
                </div>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
                  {plans.map((p) => {
                    const isDraft = p.status === "draft";
                    const isPublished = p.status === "published";
                    const isArchived = p.status === "archived";

                    const planOffers = offers.filter((o) => o.plan_id === p.id);
                    const enOffer = planOffers.find((o) => o.locale === "en");
                    const faOffer = planOffers.find((o) => o.locale === "fa");

                    return (
                      <div
                        key={p.id}
                        className="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 flex flex-col justify-between hover:border-slate-300 transition-all group"
                      >
                        <div>
                          <div className="flex items-center justify-between gap-2 mb-3">
                            <span className="text-xs font-mono font-bold text-slate-500 bg-slate-100 px-2 py-0.5 rounded">
                              /{p.slug}
                            </span>
                            {isPublished ? (
                              <span className="inline-flex items-center gap-1 bg-emerald-50 text-emerald-700 border border-emerald-200 px-2 py-0.5 rounded-full font-bold text-[10px]">
                                <CheckCircle2 className="w-3 h-3" />
                                <span>Published (v{p.current_version})</span>
                              </span>
                            ) : isDraft ? (
                              <span className="inline-flex items-center gap-1 bg-amber-50 text-amber-700 border border-amber-200 px-2 py-0.5 rounded-full font-bold text-[10px]">
                                <Clock className="w-3 h-3" />
                                <span>Draft (v{p.current_version})</span>
                              </span>
                            ) : (
                              <span className="inline-flex items-center gap-1 bg-slate-100 text-slate-600 px-2 py-0.5 rounded-full font-bold text-[10px]">
                                <Archive className="w-3 h-3" />
                                <span>Archived</span>
                              </span>
                            )}
                          </div>

                          <h3 className="text-base font-bold text-slate-900 group-hover:text-emerald-600 transition-colors">
                            {p.name}
                          </h3>
                          <p className="text-xs text-slate-500 mt-1 line-clamp-2 min-h-[32px]">
                            {p.description || "No description provided."}
                          </p>

                          {/* Regional Catalog Availability Badges */}
                          <div className="mt-4 pt-3 border-t border-slate-100 space-y-2">
                            <div className="text-[10px] font-bold uppercase tracking-wider text-slate-400">
                              Regional Catalogs
                            </div>
                            <div className="flex flex-wrap items-center gap-2">
                              {enOffer ? (
                                <span
                                  className={`inline-flex items-center gap-1 text-[11px] font-medium px-2 py-0.5 rounded-lg border ${
                                    enOffer.status === "published"
                                      ? "bg-emerald-50 text-emerald-700 border-emerald-200"
                                      : "bg-amber-50 text-amber-700 border-amber-200"
                                  }`}
                                >
                                  <span>EN:</span>
                                  <span className="font-bold">
                                    {enOffer.latest_version?.formatted_price || "Draft"}
                                  </span>
                                </span>
                              ) : (
                                <span className="inline-flex items-center text-[10px] font-medium px-2 py-0.5 rounded-lg bg-slate-50 text-slate-400 border border-slate-200">
                                  EN: None
                                </span>
                              )}

                              {faOffer ? (
                                <span
                                  className={`inline-flex items-center gap-1 text-[11px] font-medium px-2 py-0.5 rounded-lg border ${
                                    faOffer.status === "published"
                                      ? "bg-emerald-50 text-emerald-700 border-emerald-200"
                                      : "bg-amber-50 text-amber-700 border-amber-200"
                                  }`}
                                >
                                  <span>FA:</span>
                                  <span className="font-bold">
                                    {faOffer.latest_version?.formatted_price || "پیش‌نویس"}
                                  </span>
                                </span>
                              ) : (
                                <span className="inline-flex items-center text-[10px] font-medium px-2 py-0.5 rounded-lg bg-slate-50 text-slate-400 border border-slate-200">
                                  FA: None
                                </span>
                              )}
                            </div>
                          </div>

                          <div className="mt-4 pt-3 border-t border-slate-100 flex items-center justify-between text-xs text-slate-500 font-mono">
                            <span className="flex items-center gap-1.5">
                              <Building className="w-3.5 h-3.5 text-slate-400" />
                              <span>{p.assigned_orgs_count} assigned {p.assigned_orgs_count === 1 ? "org" : "orgs"}</span>
                            </span>
                            <span>v{p.current_version}</span>
                          </div>
                        </div>

                        <div className="mt-6 pt-4 border-t border-slate-100 flex items-center justify-between gap-2">
                          <Link href={`/platform-admin/plans/${p.id}`} className="flex-1">
                            <Button variant="secondary" size="sm" className="w-full justify-center gap-1">
                              <Eye className="w-3.5 h-3.5" />
                              <span>Manage Plan & Offers</span>
                            </Button>
                          </Link>

                          {isDraft && (
                            <Button
                              variant="primary"
                              size="sm"
                              onClick={() => {
                                setSelectedPlan(p);
                                setModalAction("publish");
                              }}
                            >
                              Publish Base
                            </Button>
                          )}

                          {!isArchived && (
                            <Button
                              variant="ghost"
                              size="sm"
                              className="text-slate-500 hover:text-rose-600"
                              onClick={() => {
                                setSelectedPlan(p);
                                setModalAction("archive");
                              }}
                              title="Archive Plan"
                            >
                              <Archive className="w-4 h-4" />
                            </Button>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          )}

          {/* TAB 2: ENGLISH CATALOG */}
          {activeTab === "en" && (
            <div>
              {englishOffers.length === 0 ? (
                <div className="bg-white rounded-2xl border border-slate-200 p-12 text-center shadow-sm">
                  <Globe className="w-12 h-12 text-slate-300 mx-auto mb-3" />
                  <h3 className="text-base font-bold text-slate-900">No English localized offers created</h3>
                  <p className="text-xs text-slate-500 max-w-sm mx-auto mt-1 mb-6">
                    Open any plan to author an English/Global localized offer with USD pricing.
                  </p>
                </div>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
                  {englishOffers.map((o) => {
                    const isPublished = o.status === "published";
                    const isDraft = o.status === "draft";
                    const isArchived = o.status === "archived";

                    return (
                      <div
                        key={o.id}
                        className="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 flex flex-col justify-between hover:border-slate-300 transition-all"
                      >
                        <div>
                          <div className="flex items-center justify-between gap-2 mb-3">
                            <span className="text-xs font-mono font-bold text-slate-500 bg-slate-100 px-2 py-0.5 rounded">
                              Base: /{o.plan_slug}
                            </span>
                            {isPublished ? (
                              <span className="inline-flex items-center gap-1 bg-emerald-50 text-emerald-700 border border-emerald-200 px-2 py-0.5 rounded-full font-bold text-[10px]">
                                <CheckCircle2 className="w-3 h-3" />
                                <span>Published (v{o.latest_version?.version || 1})</span>
                              </span>
                            ) : isDraft ? (
                              <span className="inline-flex items-center gap-1 bg-amber-50 text-amber-700 border border-amber-200 px-2 py-0.5 rounded-full font-bold text-[10px]">
                                <Clock className="w-3 h-3" />
                                <span>Draft</span>
                              </span>
                            ) : (
                              <span className="inline-flex items-center gap-1 bg-slate-100 text-slate-600 px-2 py-0.5 rounded-full font-bold text-[10px]">
                                <Archive className="w-3 h-3" />
                                <span>Archived</span>
                              </span>
                            )}
                          </div>

                          <h3 className="text-lg font-bold text-slate-900">{o.name}</h3>
                          <div className="text-2xl font-black text-emerald-600 mt-2 font-mono">
                            {o.latest_version?.formatted_price || "Price unconfigured"}
                          </div>
                          <p className="text-xs text-slate-500 mt-2 line-clamp-2 min-h-[32px]">
                            {o.description || "No localized description."}
                          </p>

                          {o.features_summary && o.features_summary.length > 0 && (
                            <ul className="mt-4 pt-3 border-t border-slate-100 space-y-1.5 text-xs text-slate-600">
                              {o.features_summary.slice(0, 3).map((f, i) => (
                                <li key={i} className="flex items-center gap-1.5">
                                  <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500 shrink-0" />
                                  <span>{f}</span>
                                </li>
                              ))}
                            </ul>
                          )}
                        </div>

                        <div className="mt-6 pt-4 border-t border-slate-100 flex items-center justify-between gap-2">
                          <Link href={`/platform-admin/plans/${o.plan_id}`} className="flex-1">
                            <Button variant="secondary" size="sm" className="w-full justify-center gap-1">
                              <Eye className="w-3.5 h-3.5" />
                              <span>Edit English Offer</span>
                            </Button>
                          </Link>

                          {isDraft && (
                            <Button
                              variant="primary"
                              size="sm"
                              onClick={() => {
                                setSelectedOffer(o);
                                setModalAction("publish");
                              }}
                            >
                              Publish Offer
                            </Button>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          )}

          {/* TAB 3: PERSIAN CATALOG */}
          {activeTab === "fa" && (
            <div>
              {persianOffers.length === 0 ? (
                <div className="bg-white rounded-2xl border border-slate-200 p-12 text-center shadow-sm">
                  <Coins className="w-12 h-12 text-slate-300 mx-auto mb-3" />
                  <h3 className="text-base font-bold text-slate-900">هیچ آفر فارسی برای این پلن‌ها ثبت نشده است</h3>
                  <p className="text-xs text-slate-500 max-w-sm mx-auto mt-1 mb-6">
                    وارد هر پلن شوید و آفر فارسی با نرخ تومان و متن‌های بومی‌سازی‌شده ایجاد کنید.
                  </p>
                </div>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
                  {persianOffers.map((o) => {
                    const isPublished = o.status === "published";
                    const isDraft = o.status === "draft";
                    const isArchived = o.status === "archived";

                    return (
                      <div
                        key={o.id}
                        className="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 flex flex-col justify-between hover:border-slate-300 transition-all font-sans"
                      >
                        <div>
                          <div className="flex items-center justify-between gap-2 mb-3">
                            <span className="text-xs font-mono font-bold text-slate-500 bg-slate-100 px-2 py-0.5 rounded">
                              /{o.plan_slug}
                            </span>
                            {isPublished ? (
                              <span className="inline-flex items-center gap-1 bg-emerald-50 text-emerald-700 border border-emerald-200 px-2 py-0.5 rounded-full font-bold text-[10px]">
                                <CheckCircle2 className="w-3 h-3" />
                                <span>منتشر شده (نسخه {formatPersianNumber(o.latest_version?.version || 1)})</span>
                              </span>
                            ) : isDraft ? (
                              <span className="inline-flex items-center gap-1 bg-amber-50 text-amber-700 border border-amber-200 px-2 py-0.5 rounded-full font-bold text-[10px]">
                                <Clock className="w-3 h-3" />
                                <span>پیش‌نویس</span>
                              </span>
                            ) : (
                              <span className="inline-flex items-center gap-1 bg-slate-100 text-slate-600 px-2 py-0.5 rounded-full font-bold text-[10px]">
                                <Archive className="w-3 h-3" />
                                <span>بایگانی شده</span>
                              </span>
                            )}
                          </div>

                          <h3 className="text-lg font-bold text-slate-900">{o.name}</h3>
                          <div className="text-xl font-black text-emerald-600 mt-2">
                            {o.latest_version?.formatted_price || "قیمت‌گذاری نشده"}
                          </div>
                          <p className="text-xs text-slate-500 mt-2 line-clamp-2 min-h-[32px]">
                            {o.description || "بدون توضیح بومی."}
                          </p>

                          {/* Canonical Rial breakdown */}
                          {o.latest_version && (
                            <div className="mt-3 p-2.5 rounded-lg bg-slate-50 text-[11px] text-slate-500 border border-slate-100 space-y-1">
                              <div className="flex justify-between">
                                <span>واحد نمایشی:</span>
                                <span className="font-bold text-slate-800">تومان (ضریب ۱۰ ریال)</span>
                              </div>
                              <div className="flex justify-between font-mono">
                                <span>مبلغ پایه ریال:</span>
                                <span>{formatPersianNumber(o.latest_version.price_amount)} IRR</span>
                              </div>
                            </div>
                          )}

                          {o.features_summary && o.features_summary.length > 0 && (
                            <ul className="mt-4 pt-3 border-t border-slate-100 space-y-1.5 text-xs text-slate-600">
                              {o.features_summary.slice(0, 3).map((f, i) => (
                                <li key={i} className="flex items-center gap-1.5">
                                  <CheckCircle2 className="w-3.5 h-3.5 text-emerald-500 shrink-0" />
                                  <span>{f}</span>
                                </li>
                              ))}
                            </ul>
                          )}
                        </div>

                        <div className="mt-6 pt-4 border-t border-slate-100 flex items-center justify-between gap-2">
                          <Link href={`/platform-admin/plans/${o.plan_id}`} className="flex-1">
                            <Button variant="secondary" size="sm" className="w-full justify-center gap-1">
                              <Eye className="w-3.5 h-3.5" />
                              <span>ویرایش آفر فارسی</span>
                            </Button>
                          </Link>

                          {isDraft && (
                            <Button
                              variant="primary"
                              size="sm"
                              onClick={() => {
                                setSelectedOffer(o);
                                setModalAction("publish");
                              }}
                            >
                              انتشار
                            </Button>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          )}
        </>
      )}

      {/* Confirmation Modal for Plans */}
      {selectedPlan && modalAction && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm animate-in fade-in duration-200">
          <div className="bg-white rounded-2xl max-w-md w-full p-6 shadow-xl border border-slate-200 space-y-4">
            <div className="flex items-center gap-3">
              <div
                className={`w-10 h-10 rounded-xl flex items-center justify-center shrink-0 ${
                  modalAction === "publish" ? "bg-emerald-100 text-emerald-600" : "bg-slate-100 text-slate-600"
                }`}
              >
                <AlertTriangle className="w-5 h-5" />
              </div>
              <div>
                <h3 className="text-base font-bold text-slate-900">
                  {modalAction === "publish" ? t.publishPlan : t.archivePlan}
                </h3>
                <p className="text-xs text-slate-500">{selectedPlan.name}</p>
              </div>
            </div>

            <p className="text-xs text-slate-600 leading-relaxed">
              {modalAction === "publish"
                ? "Publishing this plan base freezes its shared entitlement quotas. Existing subscriptions will retain their versioned limits."
                : "Archiving this plan prevents future assignments while protecting all active organization subscriptions."}
            </p>

            <div className="flex items-center justify-end gap-2 pt-2">
              <Button
                variant="secondary"
                size="sm"
                disabled={actionLoading}
                onClick={() => {
                  setSelectedPlan(null);
                  setModalAction(null);
                }}
              >
                Cancel
              </Button>
              <Button
                variant={modalAction === "publish" ? "primary" : "danger"}
                size="sm"
                loading={actionLoading}
                onClick={handleConfirmPlanAction}
              >
                {modalAction === "publish" ? "Confirm Publication" : "Confirm Archive"}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Confirmation Modal for Localized Offers */}
      {selectedOffer && modalAction && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm animate-in fade-in duration-200">
          <div className="bg-white rounded-2xl max-w-md w-full p-6 shadow-xl border border-slate-200 space-y-4">
            <div className="flex items-center gap-3">
              <div
                className={`w-10 h-10 rounded-xl flex items-center justify-center shrink-0 ${
                  modalAction === "publish" ? "bg-emerald-100 text-emerald-600" : "bg-slate-100 text-slate-600"
                }`}
              >
                <AlertTriangle className="w-5 h-5" />
              </div>
              <div>
                <h3 className="text-base font-bold text-slate-900">
                  {modalAction === "publish" ? "Publish Regional Offer" : "Archive Regional Offer"}
                </h3>
                <p className="text-xs text-slate-500">
                  {selectedOffer.name} ({selectedOffer.locale.toUpperCase()})
                </p>
              </div>
            </div>

            <p className="text-xs text-slate-600 leading-relaxed">
              {modalAction === "publish"
                ? "Publishing this offer version creates an immutable snapshot of localized pricing and titles for customer self-serve and administrative assignment."
                : "Archiving this offer removes it from customer-facing pricing without affecting existing subscriptions that reference this version."}
            </p>

            <div className="flex items-center justify-end gap-2 pt-2">
              <Button
                variant="secondary"
                size="sm"
                disabled={actionLoading}
                onClick={() => {
                  setSelectedOffer(null);
                  setModalAction(null);
                }}
              >
                Cancel
              </Button>
              <Button
                variant={modalAction === "publish" ? "primary" : "danger"}
                size="sm"
                loading={actionLoading}
                onClick={handleConfirmOfferAction}
              >
                {modalAction === "publish" ? "Confirm Publication" : "Confirm Archive"}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
