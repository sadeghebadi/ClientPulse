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

export default function PlatformAdminPlansPage() {
  const { t, isRTL } = useI18n();
  const [plans, setPlans] = useState<PlanItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Publish / Archive modal states
  const [selectedPlan, setSelectedPlan] = useState<PlanItem | null>(null);
  const [modalAction, setModalAction] = useState<"publish" | "archive" | null>(null);
  const [actionLoading, setActionLoading] = useState(false);
  const [toastMessage, setToastMessage] = useState<string | null>(null);

  const fetchPlans = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await apiFetch<{ plans: PlanItem[] }>("/api/v1/platform-admin/plans");
      setPlans(data.plans || []);
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Failed to load subscription plans");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchPlans();
  }, []);

  const handleConfirmAction = async () => {
    if (!selectedPlan || !modalAction) return;
    try {
      setActionLoading(true);
      const endpoint = `/api/v1/platform-admin/plans/${selectedPlan.id}/${modalAction}`;
      await apiFetch(endpoint, {
        method: "POST",
      });

      setToastMessage(
        modalAction === "publish"
          ? `Plan ${selectedPlan.name} published successfully.`
          : `Plan ${selectedPlan.name} has been archived.`
      );
      setSelectedPlan(null);
      setModalAction(null);
      fetchPlans();
    } catch (err: unknown) {
      const e = err as Error;
      alert(`Action failed: ${e.message}`);
    } finally {
      setActionLoading(false);
    }
  };

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

      {/* Header and Create Button */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-xl sm:text-2xl font-black text-slate-900 tracking-tight flex items-center gap-2">
            <Palette className="w-6 h-6 text-emerald-600" />
            <span>{t.platformPlans}</span>
          </h1>
          <p className="text-xs sm:text-sm text-slate-500 mt-1">
            Design, version, publish, and manage immutable SaaS subscription tiers and limits.
          </p>
        </div>

        <Link href="/platform-admin/plans/new" className="self-start sm:self-auto">
          <Button variant="primary" className="gap-1.5 shadow-sm">
            <PlusCircle className="w-4 h-4" />
            <span>{t.createPlan}</span>
          </Button>
        </Link>
      </div>

      {/* Plans List Cards */}
      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
          <LoadingSkeleton className="h-56 w-full" />
          <LoadingSkeleton className="h-56 w-full" />
          <LoadingSkeleton className="h-56 w-full" />
        </div>
      ) : error ? (
        <ErrorState title="Failed to Load Plans" error={error} onRetry={fetchPlans} />
      ) : plans.length === 0 ? (
        <div className="bg-white rounded-2xl border border-slate-200 p-12 text-center shadow-sm">
          <Palette className="w-12 h-12 text-slate-300 mx-auto mb-3" />
          <h3 className="text-base font-bold text-slate-900">No subscription plans created yet</h3>
          <p className="text-xs text-slate-500 max-w-sm mx-auto mt-1 mb-6">
            Create your agency SaaS subscription tiers to enforce client, asset, and monitor quotas.
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

            return (
              <div
                key={p.id}
                className="bg-white rounded-2xl border border-slate-200 shadow-sm p-6 flex flex-col justify-between hover:border-slate-300 transition-all group"
              >
                <div>
                  <div className="flex items-center justify-between gap-2 mb-3">
                    <span className="text-xs font-mono text-slate-400 bg-slate-100 px-2 py-0.5 rounded">
                      /{p.slug}
                    </span>
                    {isPublished ? (
                      <span className="inline-flex items-center gap-1 bg-emerald-50 text-emerald-700 border border-emerald-200 px-2 py-0.5 rounded-full font-bold text-[10px]">
                        <CheckCircle2 className="w-3 h-3" />
                        <span>Published</span>
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

                  <h3 className="text-base font-bold text-slate-900 group-hover:text-blue-600 transition-colors">
                    {p.name}
                  </h3>
                  <p className="text-xs text-slate-500 mt-1 line-clamp-2 min-h-[32px]">
                    {p.description || "No description provided."}
                  </p>

                  <div className="mt-4 pt-4 border-t border-slate-100 flex items-center justify-between text-xs text-slate-500 font-mono">
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
                      <span>{isDraft ? "Edit Draft" : "View / Revise"}</span>
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
                      {t.publishPlan}
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

      {/* Confirmation Modal */}
      {selectedPlan && modalAction && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm animate-in fade-in duration-200">
          <div className="bg-white rounded-2xl max-w-md w-full p-6 shadow-xl border border-slate-200 space-y-4">
            <div className="flex items-center gap-3">
              <div className={`w-10 h-10 rounded-xl flex items-center justify-center shrink-0 ${modalAction === "publish" ? "bg-emerald-100 text-emerald-600" : "bg-slate-100 text-slate-600"}`}>
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
                ? "Publishing this plan makes its current limits active for customer assignment. Once published, this version cannot be modified; future edits will create a new draft version."
                : "Archiving this plan prevents new customer organizations from being assigned to it. Existing subscriptions will continue running without interruption."}
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
                onClick={handleConfirmAction}
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
