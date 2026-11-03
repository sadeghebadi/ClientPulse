"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import {
  Users,
  Building,
  Palette,
  Activity,
  FileText,
  AlertCircle,
  CheckCircle2,
  XCircle,
  PlusCircle,
  ArrowRight,
  ArrowLeft,
  Shield,
  Layers,
  History,
} from "lucide-react";
import { apiFetch } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { MetricCard } from "@/components/ui/MetricCard";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { LoadingSkeleton } from "@/components/ui/LoadingSkeleton";
import { ErrorState } from "@/components/ui/ErrorState";

interface OverviewStats {
  total_users: number;
  active_users: number;
  suspended_users: number;
  total_organizations: number;
  active_organizations: number;
  suspended_organizations: number;
  active_plans: number;
  total_subscriptions: number;
  unassigned_organizations: number;
}

interface AuditLogItem {
  id: string;
  actor_user_id: string;
  actor_email?: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  reason?: string;
  created_at: string;
  ip_address?: string;
}

export default function PlatformAdminOverviewPage() {
  const { t, isRTL } = useI18n();
  const [stats, setStats] = useState<OverviewStats | null>(null);
  const [recentLogs, setRecentLogs] = useState<AuditLogItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchOverview = async () => {
    try {
      setLoading(true);
      setError(null);
      const [overviewData, auditData] = await Promise.all([
        apiFetch<OverviewStats>("/api/v1/platform-admin/overview"),
        apiFetch<{ audit_logs: AuditLogItem[] }>("/api/v1/platform-admin/audit-logs?limit=5").catch(() => ({
          audit_logs: [],
        })),
      ]);
      setStats(overviewData);
      setRecentLogs(auditData.audit_logs || []);
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Failed to load platform administration metrics");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchOverview();
  }, []);

  if (loading) {
    return (
      <div className="space-y-6">
        <LoadingSkeleton className="h-28 w-full" />
        <LoadingSkeleton className="h-64 w-full" />
      </div>
    );
  }

  if (error || !stats) {
    return (
      <ErrorState
        title="Failed to Load Platform Overview"
        error={error || "An error occurred while loading administrative statistics."}
        onRetry={fetchOverview}
      />
    );
  }

  return (
    <div className="space-y-8">
      {/* Platform Overview Metrics Grid */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-slate-900 tracking-tight flex items-center gap-2">
            <span>{t.platformOverview}</span>
          </h2>
          <span className="text-xs text-slate-500 font-medium">
            Live Global Telemetry
          </span>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-4">
          <MetricCard
            title={t.totalUsers}
            value={stats.total_users}
            icon={<Users className="w-4 h-4 text-blue-600" />}
            subtitle={`${stats.active_users} active · ${stats.suspended_users} suspended`}
            href="/platform-admin/users"
            trendColor="blue"
          />
          <MetricCard
            title={t.totalOrgs}
            value={stats.total_organizations}
            icon={<Building className="w-4 h-4 text-indigo-600" />}
            subtitle={`${stats.active_organizations} active · ${stats.suspended_organizations} suspended`}
            href="/platform-admin/organizations"
            trendColor="indigo"
          />
          <MetricCard
            title={t.activePlans}
            value={stats.active_plans}
            icon={<Palette className="w-4 h-4 text-emerald-600" />}
            subtitle="Published tiers available"
            href="/platform-admin/plans"
            trendColor="emerald"
          />
          <MetricCard
            title={t.totalSubscriptions}
            value={stats.total_subscriptions}
            icon={<Activity className="w-4 h-4 text-amber-600" />}
            subtitle="Active customer subscriptions"
            href="/platform-admin/subscriptions"
            trendColor="amber"
          />
          <MetricCard
            title={t.unassignedOrgs}
            value={stats.unassigned_organizations}
            icon={<AlertCircle className="w-4 h-4 text-rose-600" />}
            subtitle="Legacy unmetered access"
            href="/platform-admin/organizations"
            trendColor="rose"
          />
        </div>
      </div>

      {/* Quick Action Operations */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm flex flex-col justify-between">
          <div>
            <div className="w-10 h-10 rounded-xl bg-blue-50 border border-blue-200 flex items-center justify-center text-blue-600 mb-3">
              <PlusCircle className="w-5 h-5" />
            </div>
            <h3 className="text-sm font-bold text-slate-900 mb-1">{t.createPlan}</h3>
            <p className="text-xs text-slate-500 leading-relaxed">
              Design a new subscription tier with structured limits for clients, assets, monitors, and intervals.
            </p>
          </div>
          <Link href="/platform-admin/plans/new" className="mt-4">
            <Button variant="primary" size="sm" className="w-full justify-center gap-1.5">
              <span>{t.createPlan}</span>
              {isRTL ? <ArrowLeft className="w-3.5 h-3.5" /> : <ArrowRight className="w-3.5 h-3.5" />}
            </Button>
          </Link>
        </div>

        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm flex flex-col justify-between">
          <div>
            <div className="w-10 h-10 rounded-xl bg-emerald-50 border border-emerald-200 flex items-center justify-center text-emerald-600 mb-3">
              <Activity className="w-5 h-5" />
            </div>
            <h3 className="text-sm font-bold text-slate-900 mb-1">{t.assignPlan}</h3>
            <p className="text-xs text-slate-500 leading-relaxed">
              Manually assign or upgrade an organization’s plan with pre-flight resource limit validation.
            </p>
          </div>
          <Link href="/platform-admin/subscriptions" className="mt-4">
            <Button variant="secondary" size="sm" className="w-full justify-center gap-1.5">
              <span>{t.platformSubscriptions}</span>
              {isRTL ? <ArrowLeft className="w-3.5 h-3.5" /> : <ArrowRight className="w-3.5 h-3.5" />}
            </Button>
          </Link>
        </div>

        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm flex flex-col justify-between">
          <div>
            <div className="w-10 h-10 rounded-xl bg-purple-50 border border-purple-200 flex items-center justify-center text-purple-600 mb-3">
              <History className="w-5 h-5" />
            </div>
            <h3 className="text-sm font-bold text-slate-900 mb-1">{t.platformAuditLogs}</h3>
            <p className="text-xs text-slate-500 leading-relaxed">
              Inspect immutable audit records for all user suspensions, org changes, and plan modifications.
            </p>
          </div>
          <Link href="/platform-admin/audit-logs" className="mt-4">
            <Button variant="secondary" size="sm" className="w-full justify-center gap-1.5">
              <span>{t.platformAuditLogs}</span>
              {isRTL ? <ArrowLeft className="w-3.5 h-3.5" /> : <ArrowRight className="w-3.5 h-3.5" />}
            </Button>
          </Link>
        </div>
      </div>

      {/* Recent Administrative Activity Feed */}
      <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
        <div className="px-5 py-4 border-b border-slate-100 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <History className="w-4 h-4 text-slate-500" />
            <h3 className="text-sm font-bold text-slate-900">{t.platformAuditLogs} (Recent)</h3>
          </div>
          <Link
            href="/platform-admin/audit-logs"
            className="text-xs font-semibold text-blue-600 hover:text-blue-700 flex items-center gap-1"
          >
            <span>View all logs</span>
            {isRTL ? <ArrowLeft className="w-3.5 h-3.5" /> : <ArrowRight className="w-3.5 h-3.5" />}
          </Link>
        </div>

        {recentLogs.length === 0 ? (
          <div className="p-8 text-center text-xs text-slate-500">
            No administrative actions recorded yet.
          </div>
        ) : (
          <div className="divide-y divide-slate-100">
            {recentLogs.map((log) => (
              <div key={log.id} className="p-4 sm:px-5 flex flex-col sm:flex-row sm:items-center justify-between gap-2 hover:bg-slate-50/50 transition-colors">
                <div className="flex items-start sm:items-center gap-3">
                  <div className="w-2 h-2 rounded-full bg-blue-500 mt-1.5 sm:mt-0 shrink-0" />
                  <div>
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-xs font-bold text-slate-900 font-mono">
                        {log.action}
                      </span>
                      <span className="text-[10px] text-slate-400 font-medium">
                        on {log.resource_type} {log.resource_id ? `(${log.resource_id.slice(0, 8)}...)` : ""}
                      </span>
                    </div>
                    {log.reason && (
                      <p className="text-xs text-slate-600 mt-0.5 line-clamp-1">
                        Reason: {log.reason}
                      </p>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-3 text-[11px] text-slate-400 shrink-0 self-end sm:self-auto font-mono">
                  <span>{new Date(log.created_at).toLocaleString()}</span>
                  {log.ip_address && (
                    <span className="bg-slate-100 text-slate-600 px-1.5 py-0.5 rounded text-[10px]">
                      {log.ip_address}
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
