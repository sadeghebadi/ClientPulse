"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import {
  Users,
  Search,
  Filter,
  ShieldCheck,
  ShieldAlert,
  CheckCircle2,
  XCircle,
  Eye,
  AlertTriangle,
  ChevronLeft,
  ChevronRight,
  MoreVertical,
} from "lucide-react";
import { apiFetch } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { ErrorState } from "@/components/ui/ErrorState";
import { ConfirmModal } from "@/components/ui/ConfirmModal";

interface PlatformUser {
  id: string;
  email: string;
  display_name: string;
  status: string;
  is_platform_admin: boolean;
  email_verified_at?: string;
  created_at: string;
}

export default function PlatformAdminUsersPage() {
  const { t, isRTL } = useI18n();
  const [users, setUsers] = useState<PlatformUser[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [limit] = useState(15);
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Modal dialog states
  const [selectedUser, setSelectedUser] = useState<PlatformUser | null>(null);
  const [modalAction, setModalAction] = useState<"suspend" | "reactivate" | null>(null);
  const [actionReason, setActionReason] = useState("");
  const [actionLoading, setActionLoading] = useState(false);
  const [toastMessage, setToastMessage] = useState<string | null>(null);

  const fetchUsers = async () => {
    try {
      setLoading(true);
      setError(null);
      const params = new URLSearchParams({
        page: page.toString(),
        limit: limit.toString(),
      });
      if (search.trim()) params.append("search", search.trim());
      if (statusFilter !== "all") params.append("status", statusFilter);

      const data = await apiFetch<{
        users: PlatformUser[];
        total: number;
        page: number;
        limit: number;
      }>(`/api/v1/platform-admin/users?${params.toString()}`);

      setUsers(data.users || []);
      setTotal(data.total || 0);
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Failed to load platform users");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUsers();
  }, [page, statusFilter]);

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(1);
    fetchUsers();
  };

  const handleConfirmAction = async () => {
    if (!selectedUser || !modalAction) return;
    try {
      setActionLoading(true);
      const endpoint = `/api/v1/platform-admin/users/${selectedUser.id}/${modalAction}`;
      await apiFetch(endpoint, {
        method: "POST",
        body: JSON.stringify({ reason: actionReason || `Administrative user ${modalAction}` }),
      });

      setToastMessage(
        modalAction === "suspend"
          ? `User ${selectedUser.email} has been suspended.`
          : `User ${selectedUser.email} has been reactivated.`
      );
      setSelectedUser(null);
      setModalAction(null);
      setActionReason("");
      fetchUsers();
    } catch (err: unknown) {
      const e = err as Error;
      alert(`Action failed: ${e.message}`);
    } finally {
      setActionLoading(false);
    }
  };

  const totalPages = Math.ceil(total / limit) || 1;

  return (
    <div className="space-y-6">
      {/* Notification Banner */}
      {toastMessage && (
        <div className="p-4 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-800 text-xs flex items-center justify-between">
          <span>{toastMessage}</span>
          <button onClick={() => setToastMessage(null)} className="font-bold text-slate-500 hover:text-slate-800">
            &times;
          </button>
        </div>
      )}

      {/* Page Header and Controls */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-xl sm:text-2xl font-black text-slate-900 tracking-tight flex items-center gap-2">
            <Users className="w-6 h-6 text-blue-600" />
            <span>{t.platformUsers}</span>
          </h1>
          <p className="text-xs sm:text-sm text-slate-500 mt-1">
            Manage global accounts, administrative status, and user account lifecycle.
          </p>
        </div>
      </div>

      {/* Filter and Search Bar */}
      <div className="bg-white p-4 rounded-2xl border border-slate-200 shadow-sm flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3">
        <form onSubmit={handleSearchSubmit} className="flex-1 flex items-center gap-2">
          <div className="relative flex-1">
            <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              placeholder="Search users by name or email address..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full pl-9 pr-3 py-2 text-xs rounded-xl border border-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
          <Button type="submit" variant="secondary" size="sm">
            Search
          </Button>
        </form>

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
            <option value="all">All Statuses</option>
            <option value="active">Active Accounts</option>
            <option value="suspended">Suspended Accounts</option>
          </select>
        </div>
      </div>

      {/* Users Table / List */}
      <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
        {loading ? (
          <div className="p-6">
            <TableSkeleton rows={5} cols={5} />
          </div>
        ) : error ? (
          <div className="p-6">
            <ErrorState title="Failed to Load Users" error={error} onRetry={fetchUsers} />
          </div>
        ) : users.length === 0 ? (
          <div className="p-12 text-center text-slate-500">
            <Users className="w-10 h-10 text-slate-300 mx-auto mb-3" />
            <p className="text-sm font-semibold">No platform users found</p>
            <p className="text-xs text-slate-400 mt-1">Try refining your search query or status filter.</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-slate-50/80 border-b border-slate-200/80 text-[11px] font-bold uppercase tracking-wider text-slate-500">
                  <th className="py-3 px-4 sm:px-6">User / Identity</th>
                  <th className="py-3 px-4">Role & Privileges</th>
                  <th className="py-3 px-4">Account Status</th>
                  <th className="py-3 px-4">Registered Date</th>
                  <th className="py-3 px-4 sm:px-6 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 text-xs">
                {users.map((u) => {
                  const isSuspended = u.status === "suspended";
                  return (
                    <tr key={u.id} className="hover:bg-slate-50/60 transition-colors">
                      <td className="py-3 px-4 sm:px-6">
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 rounded-full bg-slate-100 border border-slate-200 flex items-center justify-center font-bold text-slate-600 shrink-0 text-xs">
                            {u.display_name ? u.display_name.charAt(0).toUpperCase() : u.email.charAt(0).toUpperCase()}
                          </div>
                          <div>
                            <div className="font-bold text-slate-900">{u.display_name || "—"}</div>
                            <div className="text-slate-500 font-mono text-[11px]">{u.email}</div>
                          </div>
                        </div>
                      </td>
                      <td className="py-3 px-4">
                        {u.is_platform_admin ? (
                          <span className="inline-flex items-center gap-1 bg-purple-50 text-purple-700 border border-purple-200 px-2 py-0.5 rounded-full font-bold text-[10px]">
                            <ShieldCheck className="w-3 h-3" />
                            <span>{t.platformAdminBadge}</span>
                          </span>
                        ) : (
                          <span className="text-slate-400 font-medium text-[11px]">Standard User</span>
                        )}
                      </td>
                      <td className="py-3 px-4">
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
                      </td>
                      <td className="py-3 px-4 text-slate-500 font-mono text-[11px]">
                        {new Date(u.created_at).toLocaleDateString()}
                      </td>
                      <td className="py-3 px-4 sm:px-6 text-right">
                        <div className="flex items-center justify-end gap-1.5">
                          <Link href={`/platform-admin/users/${u.id}`}>
                            <Button variant="ghost" size="sm" className="gap-1 text-slate-600 hover:text-slate-900">
                              <Eye className="w-3.5 h-3.5" />
                              <span className="hidden sm:inline">Details</span>
                            </Button>
                          </Link>
                          {isSuspended ? (
                            <Button
                              variant="secondary"
                              size="sm"
                              className="text-emerald-700 hover:bg-emerald-50 border-emerald-200 text-xs py-1 px-2.5"
                              onClick={() => {
                                setSelectedUser(u);
                                setModalAction("reactivate");
                                setActionReason("Account review complete; privileges restored.");
                              }}
                            >
                              {t.reactivateUser}
                            </Button>
                          ) : (
                            <Button
                              variant="secondary"
                              size="sm"
                              className="text-rose-700 hover:bg-rose-50 border-rose-200 text-xs py-1 px-2.5"
                              onClick={() => {
                                setSelectedUser(u);
                                setModalAction("suspend");
                                setActionReason("Terms of service investigation.");
                              }}
                            >
                              {t.suspendUser}
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
              Showing {users.length} of {total} registered users
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

      {/* Suspend / Reactivate Confirmation Modal */}
      {selectedUser && modalAction && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm animate-in fade-in duration-200">
          <div className="bg-white rounded-2xl max-w-md w-full p-6 shadow-xl border border-slate-200 space-y-4">
            <div className="flex items-center gap-3">
              <div className={`w-10 h-10 rounded-xl flex items-center justify-center shrink-0 ${modalAction === "suspend" ? "bg-rose-100 text-rose-600" : "bg-emerald-100 text-emerald-600"}`}>
                <AlertTriangle className="w-5 h-5" />
              </div>
              <div>
                <h3 className="text-base font-bold text-slate-900">
                  {modalAction === "suspend" ? t.suspendUser : t.reactivateUser}
                </h3>
                <p className="text-xs text-slate-500">
                  {selectedUser.email}
                </p>
              </div>
            </div>

            <p className="text-xs text-slate-600 leading-relaxed">
              {modalAction === "suspend"
                ? "Suspending this user will immediately reject any requests made with their active sessions and prevent further logins until an administrator reactivates the account."
                : "Reactivating this user will restore their ability to log in and access their organizations."}
            </p>

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
                onClick={() => {
                  setSelectedUser(null);
                  setModalAction(null);
                }}
              >
                Cancel
              </Button>
              <Button
                variant={modalAction === "suspend" ? "danger" : "primary"}
                size="sm"
                loading={actionLoading}
                onClick={handleConfirmAction}
              >
                {modalAction === "suspend" ? "Confirm Suspension" : "Confirm Reactivation"}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
