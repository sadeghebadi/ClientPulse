"use client";

import React, { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { useParams, useRouter } from "next/navigation";
import {
  Users,
  Building,
  ShieldCheck,
  ShieldAlert,
  CheckCircle2,
  XCircle,
  ArrowLeft,
  ArrowRight,
  Mail,
  Calendar,
  AlertTriangle,
  BadgeCheck,
} from "lucide-react";
import { apiFetch } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { LoadingSkeleton } from "@/components/ui/LoadingSkeleton";
import { ErrorState } from "@/components/ui/ErrorState";
import { Breadcrumbs } from "@/components/ui/PageHeader";

interface UserMembership {
  organization_id: string;
  organization_name: string;
  organization_slug: string;
  organization_status: string;
  role: string;
  joined_at: string;
}

interface UserDetail {
  user: {
    id: string;
    email: string;
    display_name: string;
    status: string;
    email_verified_at?: string;
    created_at: string;
    updated_at: string;
  };
  is_platform_admin: boolean;
  memberships: UserMembership[];
}

export default function PlatformAdminUserDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { t, isRTL } = useI18n();

  const [detail, setDetail] = useState<UserDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Suspension modal
  const [modalOpen, setModalOpen] = useState(false);
  const [actionReason, setActionReason] = useState("");
  const [actionLoading, setActionLoading] = useState(false);
  const [toastMessage, setToastMessage] = useState<string | null>(null);

  const fetchDetail = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await apiFetch<UserDetail>(`/api/v1/platform-admin/users/${id}`);
      setDetail(data);
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Failed to load user details");
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    if (id) fetchDetail();
  }, [id, fetchDetail]);

  const handleStatusToggle = async () => {
    if (!detail) return;
    const isSuspended = detail.user.status === "suspended";
    const action = isSuspended ? "reactivate" : "suspend";

    try {
      setActionLoading(true);
      await apiFetch(`/api/v1/platform-admin/users/${detail.user.id}/${action}`, {
        method: "POST",
        body: JSON.stringify({ reason: actionReason || `Administrative user ${action}` }),
      });

      setToastMessage(
        isSuspended ? "User account reactivated successfully." : "User account suspended successfully."
      );
      setModalOpen(false);
      setActionReason("");
      fetchDetail();
    } catch (err: unknown) {
      const e = err as Error;
      alert(`Action failed: ${e.message}`);
    } finally {
      setActionLoading(false);
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
        title="Failed to Load User"
        error={error || "User could not be found."}
        onRetry={fetchDetail}
      />
    );
  }

  const { user, is_platform_admin, memberships } = detail;
  const isSuspended = user.status === "suspended";

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
          { label: t.platformUsers, href: "/platform-admin/users" },
          { label: user.display_name || user.email },
        ]}
      />

      {/* User Header Profile Card */}
      <div className="bg-white rounded-2xl border border-slate-200 p-6 shadow-sm flex flex-col sm:flex-row sm:items-center justify-between gap-6">
        <div className="flex items-center gap-4">
          <div className="w-14 h-14 rounded-2xl bg-gradient-to-tr from-blue-600 to-indigo-600 text-white flex items-center justify-center font-black text-xl shadow-md shrink-0">
            {user.display_name ? user.display_name.charAt(0).toUpperCase() : user.email.charAt(0).toUpperCase()}
          </div>
          <div>
            <div className="flex items-center gap-2 flex-wrap">
              <h1 className="text-xl font-black text-slate-900">{user.display_name || "—"}</h1>
              {is_platform_admin && (
                <span className="inline-flex items-center gap-1 bg-purple-50 text-purple-700 border border-purple-200 px-2 py-0.5 rounded-full font-bold text-[10px]">
                  <ShieldCheck className="w-3 h-3" />
                  <span>{t.platformAdminBadge}</span>
                </span>
              )}
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
              <span className="flex items-center gap-1">
                <Mail className="w-3.5 h-3.5 text-slate-400" />
                {user.email}
              </span>
              <span>•</span>
              <span className="text-slate-400">ID: {user.id}</span>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2 self-start sm:self-auto">
          {isSuspended ? (
            <Button
              variant="secondary"
              className="text-emerald-700 hover:bg-emerald-50 border-emerald-200"
              onClick={() => {
                setActionReason("Administrative review completed; access restored.");
                setModalOpen(true);
              }}
            >
              {t.reactivateUser}
            </Button>
          ) : (
            <Button
              variant="danger"
              onClick={() => {
                setActionReason("Administrative suspension due to compliance or terms review.");
                setModalOpen(true);
              }}
            >
              {t.suspendUser}
            </Button>
          )}
        </div>
      </div>

      {/* Account Telemetry & Verification Info */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm">
          <span className="text-[11px] font-bold text-slate-400 uppercase tracking-wider block mb-1">
            Email Verification
          </span>
          <div className="flex items-center gap-2 mt-1">
            {user.email_verified_at ? (
              <>
                <BadgeCheck className="w-5 h-5 text-emerald-600" />
                <span className="text-sm font-bold text-slate-800">Verified</span>
                <span className="text-[11px] text-slate-400 font-mono">
                  ({new Date(user.email_verified_at).toLocaleDateString()})
                </span>
              </>
            ) : (
              <>
                <AlertTriangle className="w-5 h-5 text-amber-500" />
                <span className="text-sm font-bold text-amber-700">Unverified</span>
              </>
            )}
          </div>
        </div>

        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm">
          <span className="text-[11px] font-bold text-slate-400 uppercase tracking-wider block mb-1">
            Registration Date
          </span>
          <div className="flex items-center gap-2 mt-1 text-sm font-bold text-slate-800 font-mono">
            <Calendar className="w-4 h-4 text-slate-400" />
            <span>{new Date(user.created_at).toLocaleString()}</span>
          </div>
        </div>

        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm">
          <span className="text-[11px] font-bold text-slate-400 uppercase tracking-wider block mb-1">
            Organization Memberships
          </span>
          <div className="text-sm font-bold text-slate-800 mt-1">
            {memberships.length} {memberships.length === 1 ? "organization" : "organizations"}
          </div>
        </div>
      </div>

      {/* Organization Memberships Table */}
      <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
        <div className="p-5 border-b border-slate-100 flex items-center justify-between">
          <h3 className="text-sm font-bold text-slate-900 flex items-center gap-2">
            <Building className="w-4 h-4 text-slate-500" />
            <span>Organization Memberships</span>
          </h3>
          <span className="text-xs text-slate-400">{memberships.length} total</span>
        </div>

        {memberships.length === 0 ? (
          <div className="p-8 text-center text-xs text-slate-500">
            This user has not joined any organizations yet.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-slate-50/80 border-b border-slate-200/80 text-[11px] font-bold uppercase tracking-wider text-slate-500">
                  <th className="py-3 px-6">Organization</th>
                  <th className="py-3 px-4">Org Status</th>
                  <th className="py-3 px-4">Role</th>
                  <th className="py-3 px-4">Joined Date</th>
                  <th className="py-3 px-6 text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 text-xs">
                {memberships.map((m) => (
                  <tr key={m.organization_id} className="hover:bg-slate-50/60 transition-colors">
                    <td className="py-3 px-6 font-bold text-slate-900">
                      <div>{m.organization_name}</div>
                      <div className="text-[11px] font-mono text-slate-400">/{m.organization_slug}</div>
                    </td>
                    <td className="py-3 px-4">
                      {m.organization_status === "suspended" ? (
                        <span className="bg-rose-50 text-rose-700 border border-rose-200 px-2 py-0.5 rounded-full text-[10px] font-bold">
                          Suspended
                        </span>
                      ) : (
                        <span className="bg-emerald-50 text-emerald-700 border border-emerald-200 px-2 py-0.5 rounded-full text-[10px] font-bold">
                          Active
                        </span>
                      )}
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
                      <Link href={`/platform-admin/organizations/${m.organization_id}`}>
                        <Button variant="ghost" size="sm" className="text-blue-600 hover:text-blue-700">
                          View Org
                        </Button>
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Confirmation Modal */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm animate-in fade-in duration-200">
          <div className="bg-white rounded-2xl max-w-md w-full p-6 shadow-xl border border-slate-200 space-y-4">
            <div className="flex items-center gap-3">
              <div className={`w-10 h-10 rounded-xl flex items-center justify-center shrink-0 ${isSuspended ? "bg-emerald-100 text-emerald-600" : "bg-rose-100 text-rose-600"}`}>
                <AlertTriangle className="w-5 h-5" />
              </div>
              <div>
                <h3 className="text-base font-bold text-slate-900">
                  {isSuspended ? t.reactivateUser : t.suspendUser}
                </h3>
                <p className="text-xs text-slate-500">{user.email}</p>
              </div>
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 mb-1">
                {t.administrativeNote} (Audit Justification)
              </label>
              <textarea
                value={actionReason}
                onChange={(e) => setActionReason(e.target.value)}
                placeholder="Enter justification for administrative audit history..."
                rows={3}
                className="w-full text-xs p-3 rounded-xl border border-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            <div className="flex items-center justify-end gap-2 pt-2">
              <Button
                variant="secondary"
                size="sm"
                disabled={actionLoading}
                onClick={() => setModalOpen(false)}
              >
                Cancel
              </Button>
              <Button
                variant={isSuspended ? "primary" : "danger"}
                size="sm"
                loading={actionLoading}
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
