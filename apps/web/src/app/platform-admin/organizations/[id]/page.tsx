"use client";

import React, { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import {
  Building,
  Users,
  ShieldCheck,
  CheckCircle2,
  XCircle,
  ArrowLeft,
  ArrowRight,
  Layers,
  Palette,
  AlertTriangle,
  Activity,
  PlusCircle,
  FileText,
  Clock,
  Sparkles,
  Info,
} from "lucide-react";
import { apiFetch } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/Button";
import { LoadingSkeleton } from "@/components/ui/LoadingSkeleton";
import { ErrorState } from "@/components/ui/ErrorState";
import { Breadcrumbs } from "@/components/ui/PageHeader";

interface PlanLimits {
  max_clients: number;
  max_assets: number;
  max_monitors: number;
  max_heartbeats: number;
  max_members: number;
  min_check_interval_seconds: number;
  retention_days: number;
  custom_branding_enabled: boolean;
  public_reports_enabled: boolean;
}

interface OrgDetail {
  organization: {
    id: string;
    name: string;
    slug: string;
    status: string;
    locale: string;
    timezone: string;
    created_at: string;
    updated_at: string;
  };
  owner: {
    id: string;
    email: string;
    display_name: string;
  } | null;
  subscription: {
    id: string;
    organization_id: string;
    plan_id: string;
    plan_name: string;
    plan_slug: string;
    plan_version: number;
    plan_version_id: string;
    status: string;
    started_at: string;
    ends_at?: string;
    price_amount: number;
    currency: string;
    billing_interval: string;
    administrative_note?: string;
    limits: PlanLimits;
  } | null;
  usage: {
    client_count: number;
    asset_count: number;
    monitor_count: number;
    heartbeat_count: number;
    member_count: number;
  };
  members: {
    user_id: string;
    email: string;
    display_name: string;
    role: string;
    joined_at: string;
  }[];
}

interface PlanOption {
  id: string;
  name: string;
  slug: string;
  status: string;
  current_version: number;
  published_version_id?: string;
}

export default function PlatformAdminOrganizationDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { t, isRTL } = useI18n();

  const [detail, setDetail] = useState<OrgDetail | null>(null);
  const [plans, setPlans] = useState<PlanOption[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Status toggle modal
  const [statusModalOpen, setStatusModalOpen] = useState(false);
  const [statusReason, setStatusReason] = useState("");
  const [statusLoading, setStatusLoading] = useState(false);

  // Assign plan modal
  const [planModalOpen, setPlanModalOpen] = useState(false);
  const [selectedPlanId, setSelectedPlanId] = useState("");
  const [selectedVersionId, setSelectedVersionId] = useState("");
  const [assignNote, setAssignNote] = useState("");
  const [assignWarnings, setAssignWarnings] = useState<string[]>([]);
  const [checkingWarnings, setCheckingWarnings] = useState(false);
  const [assignLoading, setAssignLoading] = useState(false);

  const [toastMessage, setToastMessage] = useState<string | null>(null);

  const fetchDetail = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const [orgData, plansData] = await Promise.all([
        apiFetch<OrgDetail>(`/api/v1/platform-admin/organizations/${id}`),
        apiFetch<{ plans: PlanOption[] }>("/api/v1/platform-admin/plans").catch(() => ({ plans: [] })),
      ]);
      setDetail(orgData);
      setPlans(plansData.plans || []);
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Failed to load organization details");
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    if (id) fetchDetail();
  }, [id, fetchDetail]);

  // When plan selection changes in modal, check warnings
  const handlePlanSelectionChange = async (planId: string) => {
    setSelectedPlanId(planId);
    setAssignWarnings([]);
    if (!planId || !detail) return;

    try {
      setCheckingWarnings(true);
      const res = await apiFetch<{ warnings: string[]; has_warnings: boolean }>(
        "/api/v1/platform-admin/subscriptions/check-limits",
        {
          method: "POST",
          body: JSON.stringify({
            organization_id: detail.organization.id,
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

  const handleConfirmPlanAssignment = async () => {
    if (!detail || !selectedPlanId) return;
    try {
      setAssignLoading(true);
      await apiFetch("/api/v1/platform-admin/subscriptions/assign", {
        method: "POST",
        body: JSON.stringify({
          organization_id: detail.organization.id,
          plan_id: selectedPlanId,
          administrative_note: assignNote || "Manual subscription plan assignment",
        }),
      });

      setToastMessage("Subscription plan assigned successfully.");
      setPlanModalOpen(false);
      setSelectedPlanId("");
      setAssignNote("");
      setAssignWarnings([]);
      fetchDetail();
    } catch (err: unknown) {
      const e = err as Error;
      alert(`Plan assignment failed: ${e.message}`);
    } finally {
      setAssignLoading(false);
    }
  };

  const handleStatusToggle = async () => {
    if (!detail) return;
    const isSuspended = detail.organization.status === "suspended";
    const action = isSuspended ? "reactivate" : "suspend";

    try {
      setStatusLoading(true);
      await apiFetch(`/api/v1/platform-admin/organizations/${detail.organization.id}/${action}`, {
        method: "POST",
        body: JSON.stringify({ reason: statusReason || `Administrative org ${action}` }),
      });

      setToastMessage(
        isSuspended ? "Organization reactivated successfully." : "Organization suspended successfully."
      );
      setStatusModalOpen(false);
      setStatusReason("");
      fetchDetail();
    } catch (err: unknown) {
      const e = err as Error;
      alert(`Action failed: ${e.message}`);
    } finally {
      setStatusLoading(false);
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
        title="Failed to Load Organization"
        error={error || "Organization could not be found."}
        onRetry={fetchDetail}
      />
    );
  }

  const { organization, owner, subscription, usage, members } = detail;
  const isSuspended = organization.status === "suspended";
  const limits = subscription?.limits;

  // Quota percentage calculator
  const renderQuotaBar = (used: number, max: number | undefined, label: string) => {
    const isUnmetered = max === undefined || max < 0;
    const percent = isUnmetered ? 0 : Math.min(100, Math.round((used / max) * 100));
    const isExceeded = !isUnmetered && used > max;

    return (
      <div className="bg-white p-4 rounded-xl border border-slate-200">
        <div className="flex items-center justify-between text-xs mb-1.5">
          <span className="font-bold text-slate-700">{label}</span>
          <span className="font-mono text-slate-500">
            {used} / {isUnmetered ? "Unlimited" : max}
          </span>
        </div>
        {!isUnmetered && (
          <div className="w-full bg-slate-100 h-2 rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all ${
                isExceeded ? "bg-rose-500" : percent > 80 ? "bg-amber-500" : "bg-blue-600"
              }`}
              style={{ width: `${percent}%` }}
            />
          </div>
        )}
        {isExceeded && (
          <p className="text-[10px] text-rose-600 font-semibold mt-1">
            Over limit by {used - max!} resources
          </p>
        )}
      </div>
    );
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

      <Breadcrumbs
        items={[
          { label: t.platformOverview, href: "/platform-admin" },
          { label: t.platformOrgs, href: "/platform-admin/organizations" },
          { label: organization.name },
        ]}
      />

      {/* Org Header Card */}
      <div className="bg-white rounded-2xl border border-slate-200 p-6 shadow-sm flex flex-col sm:flex-row sm:items-center justify-between gap-6">
        <div className="flex items-center gap-4">
          <div className="w-14 h-14 rounded-2xl bg-indigo-600 text-white flex items-center justify-center font-black text-xl shadow-md shrink-0">
            <Building className="w-7 h-7" />
          </div>
          <div>
            <div className="flex items-center gap-2 flex-wrap">
              <h1 className="text-xl font-black text-slate-900">{organization.name}</h1>
              {isSuspended ? (
                <span className="inline-flex items-center gap-1 bg-rose-50 text-rose-700 border border-rose-200 px-2 py-0.5 rounded-full font-bold text-[10px]">
                  <XCircle className="w-3 h-3" />
                  <span>Suspended</span>
                </span>
              ) : (
                <span className="inline-flex items-center gap-1 bg-emerald-50 text-emerald-700 border border-emerald-200 px-2 py-0.5 rounded-full font-bold text-[10px]">
                  <CheckCircle2 className="w-3 h-3" />
                  <span>Active</span>
                </span>
              )}
            </div>
            <div className="flex items-center gap-3 text-xs text-slate-500 mt-1 font-mono">
              <span>Slug: /{organization.slug}</span>
              <span>•</span>
              <span>Owner: {owner ? owner.email : "No owner"}</span>
              <span>•</span>
              <span className="text-slate-400">ID: {organization.id}</span>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2 self-start sm:self-auto">
          <Button
            variant="secondary"
            onClick={() => setPlanModalOpen(true)}
            className="gap-1.5"
          >
            <Activity className="w-4 h-4 text-blue-600" />
            <span>{subscription ? "Change Plan" : t.assignPlan}</span>
          </Button>

          {isSuspended ? (
            <Button
              variant="secondary"
              className="text-emerald-700 hover:bg-emerald-50 border-emerald-200"
              onClick={() => {
                setStatusReason("Compliance or payment issue resolved.");
                setStatusModalOpen(true);
              }}
            >
              {t.reactivateOrg}
            </Button>
          ) : (
            <Button
              variant="danger"
              onClick={() => {
                setStatusReason("Account suspension requested.");
                setStatusModalOpen(true);
              }}
            >
              {t.suspendOrg}
            </Button>
          )}
        </div>
      </div>

      {/* Subscription Tier & Quotas Section */}
      <div className="bg-white rounded-2xl border border-slate-200 p-6 shadow-sm space-y-6">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between pb-4 border-b border-slate-100 gap-3">
          <div>
            <h2 className="text-base font-bold text-slate-900 flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-amber-500" />
              <span>Subscription & Resource Entitlements</span>
            </h2>
            <p className="text-xs text-slate-500 mt-0.5">
              Active plan parameters, quota caps, and feature entitlements.
            </p>
          </div>
          {subscription ? (
            <div className="inline-flex items-center gap-2 bg-blue-50 border border-blue-200 px-3 py-1.5 rounded-xl">
              <span className="text-xs font-bold text-blue-900">{subscription.plan_name}</span>
              <span className="text-[10px] font-mono text-blue-600 bg-blue-100 px-1.5 py-0.5 rounded">
                v{subscription.plan_version}
              </span>
              <span className="text-[10px] text-blue-500 uppercase font-bold tracking-wider">
                ({subscription.billing_interval})
              </span>
            </div>
          ) : (
            <div className="inline-flex items-center gap-1.5 bg-amber-50 border border-amber-200 text-amber-800 px-3 py-1.5 rounded-xl text-xs font-bold">
              <Info className="w-3.5 h-3.5" />
              <span>Unassigned (Legacy Unmetered Access)</span>
            </div>
          )}
        </div>

        {/* Quota Progress Grid */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3">
          {renderQuotaBar(usage.client_count, limits?.max_clients, t.maxClients)}
          {renderQuotaBar(usage.asset_count, limits?.max_assets, t.maxAssets)}
          {renderQuotaBar(usage.monitor_count, limits?.max_monitors, t.maxMonitors)}
          {renderQuotaBar(usage.heartbeat_count, limits?.max_heartbeats, t.maxHeartbeats)}
          {renderQuotaBar(usage.member_count, limits?.max_members, t.maxTeamMembers)}
        </div>

        {/* Additional Limit & Feature Tags */}
        {limits && (
          <div className="pt-2 border-t border-slate-100 flex items-center gap-4 text-xs text-slate-600 flex-wrap">
            <span className="flex items-center gap-1">
              <Clock className="w-3.5 h-3.5 text-slate-400" />
              <span>Min Check Interval: <strong>{limits.min_check_interval_seconds}s</strong></span>
            </span>
            <span className="flex items-center gap-1">
              <FileText className="w-3.5 h-3.5 text-slate-400" />
              <span>History Retention: <strong>{limits.retention_days} days</strong></span>
            </span>
            <span className="flex items-center gap-1">
              <Palette className="w-3.5 h-3.5 text-slate-400" />
              <span>Custom Branding: <strong>{limits.custom_branding_enabled ? "Enabled" : "Disabled"}</strong></span>
            </span>
            <span className="flex items-center gap-1">
              <Sparkles className="w-3.5 h-3.5 text-slate-400" />
              <span>Care Reports: <strong>{limits.public_reports_enabled ? "Enabled" : "Disabled"}</strong></span>
            </span>
          </div>
        )}
      </div>

      {/* Organization Members Table */}
      <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
        <div className="p-5 border-b border-slate-100 flex items-center justify-between">
          <h3 className="text-sm font-bold text-slate-900 flex items-center gap-2">
            <Users className="w-4 h-4 text-slate-500" />
            <span>Team Members ({members.length})</span>
          </h3>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-slate-50/80 border-b border-slate-200/80 text-[11px] font-bold uppercase tracking-wider text-slate-500">
                <th className="py-3 px-6">Member Identity</th>
                <th className="py-3 px-4">Role</th>
                <th className="py-3 px-4">Joined Date</th>
                <th className="py-3 px-6 text-right">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 text-xs">
              {members.map((m) => (
                <tr key={m.user_id} className="hover:bg-slate-50/60 transition-colors">
                  <td className="py-3 px-6 font-bold text-slate-900">
                    <div>{m.display_name || "—"}</div>
                    <div className="text-[11px] font-mono text-slate-400">{m.email}</div>
                  </td>
                  <td className="py-3 px-4">
                    <span className="capitalize font-semibold text-slate-700 bg-slate-100 px-2 py-0.5 rounded text-[11px]">
                      {m.role}
                    </span>
                  </td>
                  <td className="py-3 px-4 text-slate-500 font-mono text-[11px]">
                    {new Date(m.joined_at).toLocaleDateString()}
                  </td>
                  <td className="py-3 px-6 text-right">
                    <Link href={`/platform-admin/users/${m.user_id}`}>
                      <Button variant="ghost" size="sm" className="text-blue-600 hover:text-blue-700">
                        View User
                      </Button>
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Assign / Change Subscription Modal */}
      {planModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm animate-in fade-in duration-200">
          <div className="bg-white rounded-2xl max-w-lg w-full p-6 shadow-xl border border-slate-200 space-y-4">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-blue-100 text-blue-600 flex items-center justify-center shrink-0">
                <Activity className="w-5 h-5" />
              </div>
              <div>
                <h3 className="text-base font-bold text-slate-900">
                  {t.assignPlan}
                </h3>
                <p className="text-xs text-slate-500">
                  {organization.name}
                </p>
              </div>
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 mb-1">
                Select Subscription Plan Tier
              </label>
              <select
                value={selectedPlanId}
                onChange={(e) => handlePlanSelectionChange(e.target.value)}
                className="w-full text-xs p-3 rounded-xl border border-slate-200 bg-slate-50 font-medium focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">-- Choose a published plan --</option>
                {plans.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name} (Slug: {p.slug}) · Current Version: v{p.current_version}
                  </option>
                ))}
              </select>
            </div>

            {/* Over-limit warning check indicator */}
            {checkingWarnings && (
              <p className="text-xs text-slate-400 italic">Evaluating resource footprints against plan limits...</p>
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
                placeholder="Optional audit justification for subscription assignment..."
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
                  setPlanModalOpen(false);
                  setSelectedPlanId("");
                  setAssignWarnings([]);
                }}
              >
                Cancel
              </Button>
              <Button
                variant="primary"
                size="sm"
                disabled={!selectedPlanId}
                loading={assignLoading}
                onClick={handleConfirmPlanAssignment}
              >
                Confirm Assignment
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Suspend / Reactivate Confirmation Modal */}
      {statusModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm animate-in fade-in duration-200">
          <div className="bg-white rounded-2xl max-w-md w-full p-6 shadow-xl border border-slate-200 space-y-4">
            <div className="flex items-center gap-3">
              <div className={`w-10 h-10 rounded-xl flex items-center justify-center shrink-0 ${isSuspended ? "bg-emerald-100 text-emerald-600" : "bg-rose-100 text-rose-600"}`}>
                <AlertTriangle className="w-5 h-5" />
              </div>
              <div>
                <h3 className="text-base font-bold text-slate-900">
                  {isSuspended ? t.reactivateOrg : t.suspendOrg}
                </h3>
                <p className="text-xs text-slate-500">{organization.name}</p>
              </div>
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 mb-1">
                {t.administrativeNote} (Audit Justification)
              </label>
              <textarea
                value={statusReason}
                onChange={(e) => setStatusReason(e.target.value)}
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
                onClick={() => setStatusModalOpen(false)}
              >
                Cancel
              </Button>
              <Button
                variant={isSuspended ? "primary" : "danger"}
                size="sm"
                loading={statusLoading}
                onClick={handleStatusToggle}
              >
                {isSuspended ? "Confirm Reactivation" : "Confirm Suspension"}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
