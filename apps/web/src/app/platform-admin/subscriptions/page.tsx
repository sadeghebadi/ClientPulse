"use client";

import React, { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import {
  Activity,
  PlusCircle,
  Building,
  Palette,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  Clock,
  Sparkles,
  ChevronLeft,
  ChevronRight,
  Filter,
} from "lucide-react";
import { apiFetch } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/Button";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { ErrorState } from "@/components/ui/ErrorState";

interface SubscriptionItem {
  id: string;
  organization_id: string;
  organization_name: string;
  plan_id: string;
  plan_name: string;
  plan_slug: string;
  plan_version: number;
  status: string;
  started_at: string;
  ends_at?: string;
  price_amount: number;
  currency: string;
  billing_interval: string;
  assigned_by: string;
  administrative_note?: string;
  created_at: string;
  offer_version_id?: string;
  offer_name?: string;
  offer_locale?: string;
  offer_market?: string;
  offer_version?: number;
  display_denomination?: string;
  denomination_ratio?: number;
  display_price?: number;
  formatted_price?: string;
}

interface OfferOption {
  id: string;
  locale: string;
  market: string;
  name: string;
  status: string;
  latest_version?: {
    id: string;
    version: number;
    status: string;
    formatted_price: string;
  };
}

interface PlanOption {
  id: string;
  name: string;
  slug: string;
  status: string;
  current_version: number;
}

interface OrgOption {
  id: string;
  name: string;
  slug: string;
  status: string;
}

export default function PlatformAdminSubscriptionsPage() {
  const { t, isRTL } = useI18n();
  const [subscriptions, setSubscriptions] = useState<SubscriptionItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [limit] = useState(15);
  const [statusFilter, setStatusFilter] = useState("all");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Assign modal states
  const [assignModalOpen, setAssignModalOpen] = useState(false);
  const [orgOptions, setOrgOptions] = useState<OrgOption[]>([]);
  const [planOptions, setPlanOptions] = useState<PlanOption[]>([]);
  const [targetOrgId, setTargetOrgId] = useState("");
  const [targetPlanId, setTargetPlanId] = useState("");
  const [planOffers, setPlanOffers] = useState<OfferOption[]>([]);
  const [targetOfferVersionId, setTargetOfferVersionId] = useState("");
  const [assignNote, setAssignNote] = useState("");
  const [assignWarnings, setAssignWarnings] = useState<string[]>([]);
  const [checkingWarnings, setCheckingWarnings] = useState(false);
  const [assignLoading, setAssignLoading] = useState(false);

  // Status modal (suspend/reactivate/cancel)
  const [selectedSub, setSelectedSub] = useState<SubscriptionItem | null>(null);
  const [statusAction, setStatusAction] = useState<"suspend" | "reactivate" | "cancel" | null>(null);
  const [statusNote, setStatusNote] = useState("");
  const [statusLoading, setStatusLoading] = useState(false);

  const [toastMessage, setToastMessage] = useState<string | null>(null);

  const fetchSubscriptions = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const params = new URLSearchParams({
        page: page.toString(),
        limit: limit.toString(),
      });
      if (statusFilter !== "all") params.append("status", statusFilter);

      const data = await apiFetch<{
        subscriptions: SubscriptionItem[];
        total: number;
        page: number;
        limit: number;
      }>(`/api/v1/platform-admin/subscriptions?${params.toString()}`);

      setSubscriptions(data.subscriptions || []);
      setTotal(data.total || 0);
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Failed to load subscriptions");
    } finally {
      setLoading(false);
    }
  }, [page, limit, statusFilter]);

  const loadAssignOptions = async () => {
    try {
      const [orgsRes, plansRes] = await Promise.all([
        apiFetch<{ organizations: OrgOption[] }>("/api/v1/platform-admin/organizations?limit=100"),
        apiFetch<{ plans: PlanOption[] }>("/api/v1/platform-admin/plans?limit=100"),
      ]);
      setOrgOptions(orgsRes.organizations || []);
      setPlanOptions(plansRes.plans || []);
    } catch (err: unknown) {
      const e = err as Error;
      console.error("Failed to load options for assignment modal:", e);
    }
  };

  useEffect(() => {
    fetchSubscriptions();
  }, [fetchSubscriptions]);

  const handlePlanOrOrgChange = async (orgId: string, planId: string) => {
    setAssignWarnings([]);
    if (planId) {
      try {
        const offersRes = await apiFetch<{ offers: OfferOption[] }>(
          `/api/v1/platform-admin/plans/${planId}/offers`
        );
        const publishedOffers = (offersRes.offers || []).filter(
          (o) => o.status === "published" && o.latest_version
        );
        setPlanOffers(publishedOffers);
        if (publishedOffers.length > 0) {
          setTargetOfferVersionId(publishedOffers[0].latest_version?.id || "");
        } else {
          setTargetOfferVersionId("");
        }
      } catch {
        setPlanOffers([]);
        setTargetOfferVersionId("");
      }
    } else {
      setPlanOffers([]);
      setTargetOfferVersionId("");
    }

    if (!orgId || !planId) return;

    try {
      setCheckingWarnings(true);
      const res = await apiFetch<{ warnings: string[]; has_warnings: boolean }>(
        "/api/v1/platform-admin/subscriptions/check-limits",
        {
          method: "POST",
          body: JSON.stringify({
            organization_id: orgId,
            plan_id: planId,
          }),
        }
      );
      setAssignWarnings(res.warnings || []);
    } catch {
      setAssignWarnings([]);
    } finally {
      setCheckingWarnings(false);
    }
  };

  const handleConfirmAssignment = async () => {
    if (!targetOrgId || !targetPlanId) return;
    try {
      setAssignLoading(true);
      await apiFetch("/api/v1/platform-admin/subscriptions/assign", {
        method: "POST",
        body: JSON.stringify({
          organization_id: targetOrgId,
          plan_id: targetPlanId,
          offer_version_id: targetOfferVersionId || undefined,
          administrative_note: assignNote || "Manual assignment from admin subscriptions panel",
        }),
      });

      setToastMessage("Subscription assigned successfully.");
      setAssignModalOpen(false);
      setTargetOrgId("");
      setTargetPlanId("");
      setTargetOfferVersionId("");
      setPlanOffers([]);
      setAssignNote("");
      setAssignWarnings([]);
      fetchSubscriptions();
    } catch (err: unknown) {
      const e = err as Error;
      alert(`Assignment failed: ${e.message}`);
    } finally {
      setAssignLoading(false);
    }
  };

  const handleConfirmStatusChange = async () => {
    if (!selectedSub || !statusAction) return;
    try {
      setStatusLoading(true);
      await apiFetch(`/api/v1/platform-admin/subscriptions/${selectedSub.id}/${statusAction}`, {
        method: "POST",
        body: JSON.stringify({ note: statusNote || `Administrative subscription ${statusAction}` }),
      });

      setToastMessage(`Subscription ${statusAction}ed successfully.`);
      setSelectedSub(null);
      setStatusAction(null);
      setStatusNote("");
      fetchSubscriptions();
    } catch (err: unknown) {
      const e = err as Error;
      alert(`Status update failed: ${e.message}`);
    } finally {
      setStatusLoading(false);
    }
  };

  const totalPages = Math.ceil(total / limit) || 1;

  return (
    <div className="space-y-6">
      {toastMessage && (
        <div className="p-4 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-800 text-xs flex items-center justify-between">
          <span>{toastMessage}</span>
          <button onClick={() => setToastMessage(null)} className="font-bold text-slate-500 hover:text-slate-800">
            &times;
          </button>
        </div>
      )}

      {/* Header and Action */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-xl sm:text-2xl font-black text-slate-900 tracking-tight flex items-center gap-2">
            <Activity className="w-6 h-6 text-amber-600" />
            <span>{t.platformSubscriptions}</span>
          </h1>
          <p className="text-xs sm:text-sm text-slate-500 mt-1">
            Manual customer subscription assignment, plan tier upgrades, and lifecycle controls.
          </p>
        </div>

        <Button
          variant="primary"
          onClick={() => {
            loadAssignOptions();
            setAssignModalOpen(true);
          }}
          className="gap-1.5 self-start sm:self-auto shadow-sm"
        >
          <PlusCircle className="w-4 h-4" />
          <span>{t.assignPlan}</span>
        </Button>
      </div>

      {/* Filter Bar */}
      <div className="bg-white p-4 rounded-2xl border border-slate-200 shadow-sm flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <Filter className="w-4 h-4 text-slate-400" />
          <select
            value={statusFilter}
            onChange={(e) => {
              setStatusFilter(e.target.value);
              setPage(1);
            }}
            className="text-xs bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-slate-700 font-medium focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">All Subscription Statuses</option>
            <option value="active">Active Subscriptions</option>
            <option value="suspended">Suspended Subscriptions</option>
            <option value="canceled">Canceled / Replaced Subscriptions</option>
          </select>
        </div>
      </div>

      {/* Subscriptions Table */}
      <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
        {loading ? (
          <div className="p-6">
            <TableSkeleton rows={5} cols={6} />
          </div>
        ) : error ? (
          <div className="p-6">
            <ErrorState title="Failed to Load Subscriptions" error={error} onRetry={fetchSubscriptions} />
          </div>
        ) : subscriptions.length === 0 ? (
          <div className="p-12 text-center text-slate-500">
            <Activity className="w-10 h-10 text-slate-300 mx-auto mb-3" />
            <p className="text-sm font-semibold">No assigned subscriptions found</p>
            <p className="text-xs text-slate-400 mt-1">Assign a published plan to an organization to get started.</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-slate-50/80 border-b border-slate-200/80 text-[11px] font-bold uppercase tracking-wider text-slate-500">
                  <th className="py-3 px-4 sm:px-6">Organization</th>
                  <th className="py-3 px-4">Subscription Plan</th>
                  <th className="py-3 px-4">Status</th>
                  <th className="py-3 px-4">Billing Rate</th>
                  <th className="py-3 px-4">Started Date</th>
                  <th className="py-3 px-4 sm:px-6 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 text-xs">
                {subscriptions.map((sub) => {
                  const isActive = sub.status === "active";
                  const isSuspended = sub.status === "suspended";
                  const isCanceled = sub.status === "canceled";

                  return (
                    <tr key={sub.id} className="hover:bg-slate-50/60 transition-colors">
                      <td className="py-3 px-4 sm:px-6">
                        <Link
                          href={`/platform-admin/organizations/${sub.organization_id}`}
                          className="font-bold text-slate-900 hover:text-blue-600 transition-colors"
                        >
                          {sub.organization_name}
                        </Link>
                        {sub.administrative_note && (
                          <div className="text-[11px] text-slate-400 line-clamp-1 mt-0.5">
                            Note: {sub.administrative_note}
                          </div>
                        )}
                      </td>
                      <td className="py-3 px-4">
                        <div className="font-bold text-slate-800">{sub.plan_name}</div>
                        <div className="text-[10px] text-slate-400 font-mono flex items-center gap-1.5 mt-0.5">
                          <span>v{sub.plan_version}</span>
                          {sub.offer_name && (
                            <>
                              <span>•</span>
                              <span className="text-emerald-700 font-semibold bg-emerald-50 px-1.5 py-0.5 rounded border border-emerald-200">
                                {sub.offer_name} ({sub.offer_locale?.toUpperCase()})
                              </span>
                            </>
                          )}
                        </div>
                      </td>
                      <td className="py-3 px-4">
                        {isActive ? (
                          <span className="inline-flex items-center gap-1 bg-emerald-50 text-emerald-700 border border-emerald-200 px-2 py-0.5 rounded-full font-bold text-[10px]">
                            <CheckCircle2 className="w-3 h-3" />
                            <span>Active</span>
                          </span>
                        ) : isSuspended ? (
                          <span className="inline-flex items-center gap-1 bg-rose-50 text-rose-700 border border-rose-200 px-2 py-0.5 rounded-full font-bold text-[10px]">
                            <XCircle className="w-3 h-3" />
                            <span>Suspended</span>
                          </span>
                        ) : (
                          <span className="inline-flex items-center gap-1 bg-slate-100 text-slate-600 px-2 py-0.5 rounded-full font-bold text-[10px]">
                            <span>Canceled</span>
                          </span>
                        )}
                      </td>
                      <td className="py-3 px-4 font-mono text-slate-700">
                        {sub.formatted_price ? (
                          <div className="font-bold text-slate-900">{sub.formatted_price}</div>
                        ) : (
                          <div>{(sub.price_amount / 100).toFixed(2)} {sub.currency} / {sub.billing_interval}</div>
                        )}
                      </td>
                      <td className="py-3 px-4 text-slate-500 font-mono text-[11px]">
                        {new Date(sub.started_at).toLocaleDateString()}
                      </td>
                      <td className="py-3 px-4 sm:px-6 text-right">
                        <div className="flex items-center justify-end gap-1.5">
                          {isActive && (
                            <>
                              <Button
                                variant="secondary"
                                size="sm"
                                className="text-amber-700 hover:bg-amber-50 border-amber-200 text-xs py-1 px-2.5"
                                onClick={() => {
                                  setSelectedSub(sub);
                                  setStatusAction("suspend");
                                  setStatusNote("Delinquent account payment or suspension.");
                                }}
                              >
                                Suspend
                              </Button>
                              <Button
                                variant="ghost"
                                size="sm"
                                className="text-slate-500 hover:text-rose-600 text-xs py-1 px-2.5"
                                onClick={() => {
                                  setSelectedSub(sub);
                                  setStatusAction("cancel");
                                  setStatusNote("Customer subscription canceled by platform admin.");
                                }}
                              >
                                Cancel
                              </Button>
                            </>
                          )}
                          {isSuspended && (
                            <Button
                              variant="secondary"
                              size="sm"
                              className="text-emerald-700 hover:bg-emerald-50 border-emerald-200 text-xs py-1 px-2.5"
                              onClick={() => {
                                setSelectedSub(sub);
                                setStatusAction("reactivate");
                                setStatusNote("Payment received; subscription reinstated.");
                              }}
                            >
                              Reactivate
                            </Button>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination Bar */}
        {!loading && totalPages > 1 && (
          <div className="px-5 py-3 border-t border-slate-100 flex items-center justify-between text-xs text-slate-500 bg-slate-50/50">
            <span>
              Showing {subscriptions.length} of {total} subscriptions
            </span>
            <div className="flex items-center gap-1">
              <Button
                variant="ghost"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
              >
                {isRTL ? <ChevronRight className="w-4 h-4" /> : <ChevronLeft className="w-4 h-4" />}
              </Button>
              <span className="px-2 font-semibold text-slate-700">
                Page {page} of {totalPages}
              </span>
              <Button
                variant="ghost"
                size="sm"
                disabled={page >= totalPages}
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              >
                {isRTL ? <ChevronLeft className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
              </Button>
            </div>
          </div>
        )}
      </div>

      {/* Manual Assignment Modal */}
      {assignModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm animate-in fade-in duration-200">
          <div className="bg-white rounded-2xl max-w-lg w-full p-6 shadow-xl border border-slate-200 space-y-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-amber-100 text-amber-700 flex items-center justify-center shrink-0">
                <Activity className="w-5 h-5" />
              </div>
              <div>
                <h3 className="text-base font-bold text-slate-900">{t.assignPlan}</h3>
                <p className="text-xs text-slate-500">
                  Select customer organization and active plan tier.
                </p>
              </div>
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 mb-1">
                Target Organization *
              </label>
              <select
                value={targetOrgId}
                onChange={(e) => {
                  setTargetOrgId(e.target.value);
                  handlePlanOrOrgChange(e.target.value, targetPlanId);
                }}
                className="w-full text-xs p-3 rounded-xl border border-slate-200 bg-slate-50 font-medium focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">-- Choose Organization --</option>
                {orgOptions.map((o) => (
                  <option key={o.id} value={o.id}>
                    {o.name} (/{o.slug})
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 mb-1">
                Published Subscription Plan Tier *
              </label>
              <select
                value={targetPlanId}
                onChange={(e) => {
                  setTargetPlanId(e.target.value);
                  handlePlanOrOrgChange(targetOrgId, e.target.value);
                }}
                className="w-full text-xs p-3 rounded-xl border border-slate-200 bg-slate-50 font-medium focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">-- Choose Subscription Tier --</option>
                {planOptions.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name} (v{p.current_version}) · /{p.slug}
                  </option>
                ))}
              </select>
            </div>

            {planOffers.length > 0 && (
              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">
                  Localized Regional Offer (Optional)
                </label>
                <select
                  value={targetOfferVersionId}
                  onChange={(e) => setTargetOfferVersionId(e.target.value)}
                  className="w-full text-xs p-3 rounded-xl border border-slate-200 bg-slate-50 font-medium focus:outline-none focus:ring-2 focus:ring-emerald-500"
                >
                  <option value="">Base Plan Default (Unlocalized)</option>
                  {planOffers.map((o) => (
                    <option key={o.latest_version?.id || o.id} value={o.latest_version?.id || ""}>
                      {o.name} ({o.locale.toUpperCase()} / {o.market.toUpperCase()}) — {o.latest_version?.formatted_price || "Draft"}
                    </option>
                  ))}
                </select>
                <p className="text-[11px] text-slate-400 mt-1">
                  Selecting an offer locks in the immutable regional pricing and denomination.
                </p>
              </div>
            )}

            {checkingWarnings && (
              <p className="text-xs text-slate-400 italic">Validating resource limits against existing usage...</p>
            )}

            {assignWarnings.length > 0 && (
              <div className="p-4 rounded-xl bg-amber-50 border border-amber-200 text-amber-900 space-y-2">
                <div className="flex items-center gap-2 font-bold text-xs text-amber-800">
                  <AlertTriangle className="w-4 h-4 shrink-0" />
                  <span>{t.overLimitWarningTitle}</span>
                </div>
                <p className="text-[11px] text-amber-700 leading-relaxed">
                  {t.overLimitWarningDesc}
                </p>
                <ul className="list-disc list-inside text-xs space-y-1 font-medium pl-1 text-amber-900">
                  {assignWarnings.map((w, idx) => (
                    <li key={idx}>{w}</li>
                  ))}
                </ul>
              </div>
            )}

            <div>
              <label className="block text-xs font-bold text-slate-700 mb-1">
                {t.administrativeNote}
              </label>
              <textarea
                value={assignNote}
                onChange={(e) => setAssignNote(e.target.value)}
                placeholder="Optional administrative note / audit trail..."
                rows={2}
                className="w-full text-xs p-3 rounded-xl border border-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            <div className="flex items-center justify-end gap-2 pt-2">
              <Button
                variant="secondary"
                size="sm"
                disabled={assignLoading}
                onClick={() => {
                  setAssignModalOpen(false);
                  setTargetOrgId("");
                  setTargetPlanId("");
                  setAssignWarnings([]);
                }}
              >
                Cancel
              </Button>
              <Button
                variant="primary"
                size="sm"
                disabled={!targetOrgId || !targetPlanId}
                loading={assignLoading}
                onClick={handleConfirmAssignment}
              >
                Confirm Assignment
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Status Modal (suspend/reactivate/cancel) */}
      {selectedSub && statusAction && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm animate-in fade-in duration-200">
          <div className="bg-white rounded-2xl max-w-md w-full p-6 shadow-xl border border-slate-200 space-y-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-slate-100 text-slate-700 flex items-center justify-center shrink-0">
                <AlertTriangle className="w-5 h-5" />
              </div>
              <div>
                <h3 className="text-base font-bold text-slate-900 capitalize">
                  {statusAction} Subscription
                </h3>
                <p className="text-xs text-slate-500">
                  {selectedSub.organization_name} · {selectedSub.plan_name}
                </p>
              </div>
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 mb-1">
                {t.administrativeNote} (Audit Justification)
              </label>
              <textarea
                value={statusNote}
                onChange={(e) => setStatusNote(e.target.value)}
                placeholder="Enter justification for administrative audit history..."
                rows={3}
                className="w-full text-xs p-3 rounded-xl border border-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            <div className="flex items-center justify-end gap-2 pt-2">
              <Button
                variant="secondary"
                size="sm"
                disabled={statusLoading}
                onClick={() => {
                  setSelectedSub(null);
                  setStatusAction(null);
                }}
              >
                Cancel
              </Button>
              <Button
                variant={statusAction === "reactivate" ? "primary" : "danger"}
                size="sm"
                loading={statusLoading}
                onClick={handleConfirmStatusChange}
              >
                Confirm {statusAction}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
