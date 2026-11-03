"use client";

import React, { useEffect, useState } from "react";
import {
  FileText,
  Search,
  Filter,
  History,
  Shield,
  Clock,
  Eye,
  ChevronLeft,
  ChevronRight,
  Code,
  X,
} from "lucide-react";
import { apiFetch } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/Button";
import { TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { ErrorState } from "@/components/ui/ErrorState";

interface AuditLogItem {
  id: string;
  actor_user_id: string;
  actor_email?: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  reason?: string;
  metadata?: Record<string, unknown>;
  ip_address?: string;
  user_agent?: string;
  created_at: string;
}

export default function PlatformAdminAuditLogsPage() {
  const { t, isRTL } = useI18n();
  const [logs, setLogs] = useState<AuditLogItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [limit] = useState(25);
  const [actionFilter, setActionFilter] = useState("all");
  const [targetTypeFilter, setTargetTypeFilter] = useState("all");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // JSON metadata viewer modal
  const [selectedLog, setSelectedLog] = useState<AuditLogItem | null>(null);

  const fetchLogs = async () => {
    try {
      setLoading(true);
      setError(null);
      const params = new URLSearchParams({
        page: page.toString(),
        limit: limit.toString(),
      });
      if (actionFilter !== "all") params.append("action", actionFilter);
      if (targetTypeFilter !== "all") params.append("target_type", targetTypeFilter);

      const data = await apiFetch<{
        audit_logs: AuditLogItem[];
        total: number;
        page: number;
        limit: number;
      }>(`/api/v1/platform-admin/audit-logs?${params.toString()}`);

      setLogs(data.audit_logs || []);
      setTotal(data.total || 0);
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Failed to load platform audit logs");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLogs();
  }, [page, actionFilter, targetTypeFilter]);

  const totalPages = Math.ceil(total / limit) || 1;

  const renderActionBadge = (action: string) => {
    let colorClass = "bg-slate-100 text-slate-700 border-slate-200";
    if (action.includes("suspend")) colorClass = "bg-rose-50 text-rose-700 border-rose-200";
    else if (action.includes("reactivate") || action.includes("publish")) colorClass = "bg-emerald-50 text-emerald-700 border-emerald-200";
    else if (action.includes("created")) colorClass = "bg-blue-50 text-blue-700 border-blue-200";
    else if (action.includes("assigned")) colorClass = "bg-amber-50 text-amber-700 border-amber-200";

    return (
      <span className={`inline-flex items-center px-2 py-0.5 rounded-full font-mono font-bold text-[10px] border ${colorClass}`}>
        {action}
      </span>
    );
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-xl sm:text-2xl font-black text-slate-900 tracking-tight flex items-center gap-2">
            <History className="w-6 h-6 text-purple-600" />
            <span>{t.platformAuditLogs}</span>
          </h1>
          <p className="text-xs sm:text-sm text-slate-500 mt-1">
            Immutable platform governance ledger recording all administrative actions, suspensions, and plan events.
          </p>
        </div>
      </div>

      {/* Filter Bar */}
      <div className="bg-white p-4 rounded-2xl border border-slate-200 shadow-sm flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3">
        <div className="flex items-center gap-2 flex-wrap">
          <Filter className="w-4 h-4 text-slate-400" />
          <select
            value={actionFilter}
            onChange={(e) => {
              setActionFilter(e.target.value);
              setPage(1);
            }}
            className="text-xs bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-slate-700 font-medium focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">All Administrative Actions</option>
            <option value="user.suspended">user.suspended</option>
            <option value="user.reactivated">user.reactivated</option>
            <option value="organization.suspended">organization.suspended</option>
            <option value="organization.reactivated">organization.reactivated</option>
            <option value="plan.created">plan.created</option>
            <option value="plan.updated">plan.updated</option>
            <option value="plan.published">plan.published</option>
            <option value="plan.archived">plan.archived</option>
            <option value="subscription.assigned">subscription.assigned</option>
            <option value="subscription.suspended">subscription.suspended</option>
            <option value="subscription.reactivated">subscription.reactivated</option>
          </select>

          <select
            value={targetTypeFilter}
            onChange={(e) => {
              setTargetTypeFilter(e.target.value);
              setPage(1);
            }}
            className="text-xs bg-slate-50 border border-slate-200 rounded-xl px-3 py-2 text-slate-700 font-medium focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">All Target Resources</option>
            <option value="user">User Accounts</option>
            <option value="organization">Organizations</option>
            <option value="plan">Subscription Plans</option>
            <option value="subscription">Subscriptions</option>
          </select>
        </div>
      </div>

      {/* Audit Logs Table */}
      <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
        {loading ? (
          <div className="p-6">
            <TableSkeleton rows={6} cols={6} />
          </div>
        ) : error ? (
          <div className="p-6">
            <ErrorState title="Failed to Load Audit Logs" error={error} onRetry={fetchLogs} />
          </div>
        ) : logs.length === 0 ? (
          <div className="p-12 text-center text-slate-500">
            <History className="w-10 h-10 text-slate-300 mx-auto mb-3" />
            <p className="text-sm font-semibold">No platform audit logs found</p>
            <p className="text-xs text-slate-400 mt-1">Try adjusting the action or resource filter.</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-slate-50/80 border-b border-slate-200/80 text-[11px] font-bold uppercase tracking-wider text-slate-500">
                  <th className="py-3 px-4 sm:px-6">Timestamp</th>
                  <th className="py-3 px-4">Action</th>
                  <th className="py-3 px-4">Target Entity</th>
                  <th className="py-3 px-4">Actor</th>
                  <th className="py-3 px-4">Justification / Reason</th>
                  <th className="py-3 px-4 sm:px-6 text-right">Raw Metadata</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 text-xs">
                {logs.map((log) => (
                  <tr key={log.id} className="hover:bg-slate-50/60 transition-colors">
                    <td className="py-3 px-4 sm:px-6 font-mono text-[11px] text-slate-500 whitespace-nowrap">
                      {new Date(log.created_at).toLocaleString()}
                    </td>
                    <td className="py-3 px-4">
                      {renderActionBadge(log.action)}
                    </td>
                    <td className="py-3 px-4">
                      <span className="font-semibold text-slate-800 capitalize">
                        {log.resource_type}
                      </span>
                      {log.resource_id && (
                        <span className="block text-[10px] font-mono text-slate-400">
                          {log.resource_id.slice(0, 8)}...
                        </span>
                      )}
                    </td>
                    <td className="py-3 px-4 font-mono text-[11px] text-slate-600">
                      <div>{log.actor_email || log.actor_user_id.slice(0, 8)}</div>
                      {log.ip_address && (
                        <div className="text-[10px] text-slate-400">{log.ip_address}</div>
                      )}
                    </td>
                    <td className="py-3 px-4 text-slate-600 max-w-xs truncate text-[11px]">
                      {log.reason || "—"}
                    </td>
                    <td className="py-3 px-4 sm:px-6 text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setSelectedLog(log)}
                        className="text-slate-500 hover:text-slate-900 gap-1 text-[11px]"
                      >
                        <Code className="w-3.5 h-3.5" />
                        <span>Inspect</span>
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination */}
        {!loading && totalPages > 1 && (
          <div className="px-5 py-3 border-t border-slate-100 flex items-center justify-between text-xs text-slate-500 bg-slate-50/50">
            <span>
              Showing {logs.length} of {total} audit records
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

      {/* JSON Metadata Inspector Modal */}
      {selectedLog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm animate-in fade-in duration-200">
          <div className="bg-white rounded-2xl max-w-xl w-full p-6 shadow-xl border border-slate-200 space-y-4">
            <div className="flex items-center justify-between pb-3 border-b border-slate-100">
              <div className="flex items-center gap-2">
                <Code className="w-4 h-4 text-purple-600" />
                <h3 className="text-sm font-bold text-slate-900 font-mono">
                  Audit Event #{selectedLog.id.slice(0, 8)}
                </h3>
              </div>
              <button
                onClick={() => setSelectedLog(null)}
                className="text-slate-400 hover:text-slate-600 p-1 rounded-lg"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="space-y-2 text-xs">
              <div className="grid grid-cols-2 gap-2 text-slate-600">
                <div>Action: <strong className="font-mono text-slate-900">{selectedLog.action}</strong></div>
                <div>Resource: <strong className="font-mono text-slate-900">{selectedLog.resource_type}</strong></div>
                <div>Actor: <strong className="font-mono text-slate-900">{selectedLog.actor_email || selectedLog.actor_user_id}</strong></div>
                <div>Timestamp: <span className="font-mono">{new Date(selectedLog.created_at).toISOString()}</span></div>
                {selectedLog.ip_address && (
                  <div>IP Address: <span className="font-mono">{selectedLog.ip_address}</span></div>
                )}
                {selectedLog.resource_id && (
                  <div>Target ID: <span className="font-mono">{selectedLog.resource_id}</span></div>
                )}
              </div>

              {selectedLog.reason && (
                <div className="p-2.5 rounded-xl bg-slate-50 border border-slate-100 text-slate-700">
                  <span className="font-bold block text-[11px] text-slate-500 mb-0.5">Audit Justification:</span>
                  {selectedLog.reason}
                </div>
              )}

              <div>
                <span className="font-bold block text-[11px] text-slate-500 mb-1">Raw Event Payload (JSON):</span>
                <pre className="p-3 rounded-xl bg-slate-900 text-slate-100 font-mono text-[11px] overflow-x-auto max-h-60 border border-slate-800">
                  {JSON.stringify(
                    {
                      id: selectedLog.id,
                      actor_user_id: selectedLog.actor_user_id,
                      actor_email: selectedLog.actor_email,
                      action: selectedLog.action,
                      resource_type: selectedLog.resource_type,
                      resource_id: selectedLog.resource_id,
                      reason: selectedLog.reason,
                      metadata: selectedLog.metadata,
                      ip_address: selectedLog.ip_address,
                      user_agent: selectedLog.user_agent,
                      created_at: selectedLog.created_at,
                    },
                    null,
                    2
                  )}
                </pre>
              </div>
            </div>

            <div className="flex justify-end pt-2">
              <Button variant="secondary" size="sm" onClick={() => setSelectedLog(null)}>
                Close
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
