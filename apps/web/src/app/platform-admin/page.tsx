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
  PlusCircle,
  ArrowRight,
  ArrowLeft,
  Languages,
  ShieldCheck,
  Tag,
  Globe2,
  Clock,
  UserCheck,
} from "lucide-react";
import { apiFetch } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { MetricCard } from "@/components/ui/MetricCard";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { LoadingSkeleton } from "@/components/ui/LoadingSkeleton";
import { ErrorState } from "@/components/ui/ErrorState";

interface OverviewRecentSignup {
  id: string;
  email: string;
  display_name: string;
  status: string;
  created_at: string;
}

interface OverviewSubscriptionDistribution {
  category: string;
  count: number;
}

interface AuditLogItem {
  id: string;
  actor_user_id: string;
  actor_email?: string;
  actor_name?: string;
  action: string;
  target_type: string;
  target_id: string;
  reason?: string;
  metadata?: Record<string, unknown>;
  ip_address?: string;
  user_agent?: string;
  created_at: string;
}

interface PlatformOverviewData {
  total_users: number;
  active_users: number;
  suspended_users: number;
  total_organizations: number;
  active_organizations: number;
  suspended_organizations: number;
  published_plans: number;
  active_subscriptions: number;
  unassigned_organizations: number;
  published_offers: number;
  recent_signups?: OverviewRecentSignup[];
  recent_activity?: AuditLogItem[];
  subscription_distribution_locale?: OverviewSubscriptionDistribution[];
  subscription_distribution_plan?: OverviewSubscriptionDistribution[];
  recent_security_events?: AuditLogItem[];
}

export default function PlatformAdminOverviewPage() {
  const { t, isRTL, formatNumber } = useI18n();
  const [data, setData] = useState<PlatformOverviewData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchOverview = async () => {
    try {
      setLoading(true);
      setError(null);
      const res = await apiFetch<PlatformOverviewData>("/api/v1/platform-admin/overview");
      setData(res);
    } catch (err: unknown) {
      const e = err as Error;
      setError(e.message || "Failed to load platform administration telemetry");
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
        <LoadingSkeleton className="h-32 w-full rounded-2xl" />
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
          {[...Array(5)].map((_, i) => (
            <LoadingSkeleton key={i} className="h-28 w-full rounded-2xl" />
          ))}
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <LoadingSkeleton className="h-72 w-full rounded-2xl" />
          <LoadingSkeleton className="h-72 w-full rounded-2xl" />
        </div>
      </div>
    );
  }

  if (error || !data) {
    return (
      <ErrorState
        title="Failed to Load Platform Overview"
        error={error || "An error occurred while loading administrative statistics."}
        onRetry={fetchOverview}
      />
    );
  }

  const signups = data.recent_signups || [];
  const activity = data.recent_activity || [];
  const localeDist = data.subscription_distribution_locale || [];
  const planDist = data.subscription_distribution_plan || [];
  const securityEvents = data.recent_security_events || [];

  return (
    <div className="space-y-8">
      {/* KPI Section */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-base font-bold text-slate-900 tracking-tight flex items-center gap-2">
            <span>{t.platformOverview}</span>
          </h2>
          <span className="text-xs text-slate-500 font-medium">
            Live Global Counters
          </span>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-4">
          <MetricCard
            title={t.totalUsers}
            value={formatNumber(data.total_users)}
            icon={<Users className="w-4 h-4 text-blue-600" />}
            subtitle={`${formatNumber(data.active_users)} active · ${formatNumber(data.suspended_users)} suspended`}
            href="/platform-admin/users"
            trendColor="blue"
          />
          <MetricCard
            title={t.totalOrgs}
            value={formatNumber(data.total_organizations)}
            icon={<Building className="w-4 h-4 text-indigo-600" />}
            subtitle={`${formatNumber(data.active_organizations)} active · ${formatNumber(data.suspended_organizations)} suspended`}
            href="/platform-admin/organizations"
            trendColor="indigo"
          />
          <MetricCard
            title={t.activePlans}
            value={formatNumber(data.published_plans)}
            icon={<Palette className="w-4 h-4 text-emerald-600" />}
            subtitle={`${formatNumber(data.published_offers)} localized offers`}
            href="/platform-admin/plans"
            trendColor="emerald"
          />
          <MetricCard
            title={t.totalSubscriptions}
            value={formatNumber(data.active_subscriptions)}
            icon={<Activity className="w-4 h-4 text-amber-600" />}
            subtitle="Active tenant assignments"
            href="/platform-admin/subscriptions"
            trendColor="amber"
          />
          <MetricCard
            title={t.unassignedOrgs}
            value={formatNumber(data.unassigned_organizations)}
            icon={<AlertCircle className="w-4 h-4 text-rose-600" />}
            subtitle="Organizations without plan"
            href="/platform-admin/organizations"
            trendColor="rose"
          />
        </div>
      </div>

      {/* Quick Action Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <Link
          href="/platform-admin/plans/new"
          className="group bg-white p-5 rounded-2xl border border-slate-200 shadow-sm hover:border-blue-400 hover:shadow-md transition-all flex flex-col justify-between"
        >
          <div>
            <div className="w-10 h-10 rounded-xl bg-blue-50 border border-blue-200 flex items-center justify-center text-blue-600 mb-3 group-hover:scale-105 transition-transform">
              <PlusCircle className="w-5 h-5" />
            </div>
            <h3 className="text-sm font-bold text-slate-900 mb-1">{t.createPlan}</h3>
            <p className="text-xs text-slate-500 leading-relaxed">
              Define shared entitlement limits and configure market-specific pricing in USD or تومان.
            </p>
          </div>
          <div className="mt-4 pt-3 border-t border-slate-100 flex items-center justify-between text-xs font-semibold text-blue-600">
            <span>{t.createPlan}</span>
            {isRTL ? <ArrowLeft className="w-4 h-4" /> : <ArrowRight className="w-4 h-4" />}
          </div>
        </Link>

        <Link
          href="/platform-admin/settings/localization"
          className="group bg-white p-5 rounded-2xl border border-slate-200 shadow-sm hover:border-indigo-400 hover:shadow-md transition-all flex flex-col justify-between"
        >
          <div>
            <div className="w-10 h-10 rounded-xl bg-indigo-50 border border-indigo-200 flex items-center justify-center text-indigo-600 mb-3 group-hover:scale-105 transition-transform">
              <Languages className="w-5 h-5" />
            </div>
            <h3 className="text-sm font-bold text-slate-900 mb-1">{t.localizationSettings}</h3>
            <p className="text-xs text-slate-500 leading-relaxed">
              Enable or disable English and Persian globally and manage system default fallback language.
            </p>
          </div>
          <div className="mt-4 pt-3 border-t border-slate-100 flex items-center justify-between text-xs font-semibold text-indigo-600">
            <span>{t.localizationSettings}</span>
            {isRTL ? <ArrowLeft className="w-4 h-4" /> : <ArrowRight className="w-4 h-4" />}
          </div>
        </Link>

        <Link
          href="/platform-admin/subscriptions"
          className="group bg-white p-5 rounded-2xl border border-slate-200 shadow-sm hover:border-emerald-400 hover:shadow-md transition-all flex flex-col justify-between"
        >
          <div>
            <div className="w-10 h-10 rounded-xl bg-emerald-50 border border-emerald-200 flex items-center justify-center text-emerald-600 mb-3 group-hover:scale-105 transition-transform">
              <Tag className="w-5 h-5" />
            </div>
            <h3 className="text-sm font-bold text-slate-900 mb-1">{t.platformSubscriptions}</h3>
            <p className="text-xs text-slate-500 leading-relaxed">
              Assign localized offers to organizations, review historical prices, and monitor quotas.
            </p>
          </div>
          <div className="mt-4 pt-3 border-t border-slate-100 flex items-center justify-between text-xs font-semibold text-emerald-600">
            <span>{t.platformSubscriptions}</span>
            {isRTL ? <ArrowLeft className="w-4 h-4" /> : <ArrowRight className="w-4 h-4" />}
          </div>
        </Link>
      </div>

      {/* Operational Breakdown Grid: Subscription Distribution */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Subscription Distribution by Locale */}
        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm">
          <div className="flex items-center justify-between mb-4 pb-3 border-b border-slate-100">
            <div className="flex items-center gap-2">
              <Globe2 className="w-4 h-4 text-blue-600" />
              <h3 className="text-sm font-bold text-slate-900">{t.localeDistribution}</h3>
            </div>
            <Badge variant="neutral" size="sm">Real Distribution</Badge>
          </div>
          {localeDist.length === 0 ? (
            <div className="py-8 text-center text-xs text-slate-400 font-medium">
              No localized subscriptions active yet.
            </div>
          ) : (
            <div className="space-y-3">
              {localeDist.map((item) => (
                <div key={item.category} className="flex items-center justify-between p-3 rounded-xl bg-slate-50 border border-slate-100">
                  <div className="flex items-center gap-2.5">
                    <span className="w-2.5 h-2.5 rounded-full bg-blue-500" />
                    <span className="text-xs font-bold text-slate-800">
                      {item.category === "fa" ? "Persian (Iran · Toman)" : item.category === "en" ? "English (Global · USD)" : "Legacy Unspecified"}
                    </span>
                  </div>
                  <span className="text-xs font-mono font-bold text-slate-900 bg-white px-2 py-0.5 rounded border border-slate-200">
                    {formatNumber(item.count)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Subscription Distribution by Plan */}
        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm">
          <div className="flex items-center justify-between mb-4 pb-3 border-b border-slate-100">
            <div className="flex items-center gap-2">
              <Palette className="w-4 h-4 text-indigo-600" />
              <h3 className="text-sm font-bold text-slate-900">{t.planDistribution}</h3>
            </div>
            <Badge variant="neutral" size="sm">Top Plans</Badge>
          </div>
          {planDist.length === 0 ? (
            <div className="py-8 text-center text-xs text-slate-400 font-medium">
              No active subscription plans assigned yet.
            </div>
          ) : (
            <div className="space-y-3">
              {planDist.map((item) => (
                <div key={item.category} className="flex items-center justify-between p-3 rounded-xl bg-slate-50 border border-slate-100">
                  <div className="flex items-center gap-2.5">
                    <span className="w-2.5 h-2.5 rounded-full bg-indigo-500" />
                    <span className="text-xs font-bold text-slate-800 truncate max-w-[200px]">
                      {item.category}
                    </span>
                  </div>
                  <span className="text-xs font-mono font-bold text-slate-900 bg-white px-2 py-0.5 rounded border border-slate-200">
                    {formatNumber(item.count)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Operational Sections: Recent Signups & Latest Activity */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Recent Signups */}
        <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
          <div className="px-5 py-4 border-b border-slate-100 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <UserCheck className="w-4 h-4 text-blue-600" />
              <h3 className="text-sm font-bold text-slate-900">{t.recentSignups}</h3>
            </div>
            <Link href="/platform-admin/users" className="text-xs font-semibold text-blue-600 hover:text-blue-700 flex items-center gap-1">
              <span>View all</span>
              {isRTL ? <ArrowLeft className="w-3.5 h-3.5" /> : <ArrowRight className="w-3.5 h-3.5" />}
            </Link>
          </div>
          {signups.length === 0 ? (
            <div className="p-8 text-center text-xs text-slate-400">No recent users.</div>
          ) : (
            <div className="divide-y divide-slate-100">
              {signups.map((u) => (
                <div key={u.id} className="p-3.5 sm:px-5 flex items-center justify-between gap-3 hover:bg-slate-50/50 transition-colors">
                  <div className="min-w-0">
                    <div className="text-xs font-bold text-slate-900 truncate">{u.display_name || "Unnamed"}</div>
                    <div className="text-[11px] text-slate-500 font-mono truncate">{u.email}</div>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <Badge variant={u.status === "active" ? "success" : "danger"} size="sm">
                      {u.status}
                    </Badge>
                    <span className="text-[10px] text-slate-400 font-mono hidden sm:inline">
                      {new Date(u.created_at).toLocaleDateString()}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Recent Security Events */}
        <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
          <div className="px-5 py-4 border-b border-slate-100 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <ShieldCheck className="w-4 h-4 text-rose-600" />
              <h3 className="text-sm font-bold text-slate-900">{t.recentSecurityEvents}</h3>
            </div>
            <Link href="/platform-admin/audit-logs" className="text-xs font-semibold text-blue-600 hover:text-blue-700 flex items-center gap-1">
              <span>View audit</span>
              {isRTL ? <ArrowLeft className="w-3.5 h-3.5" /> : <ArrowRight className="w-3.5 h-3.5" />}
            </Link>
          </div>
          {securityEvents.length === 0 ? (
            <div className="p-8 text-center text-xs text-slate-400">No security incidents logged.</div>
          ) : (
            <div className="divide-y divide-slate-100">
              {securityEvents.map((evt) => (
                <div key={evt.id} className="p-3.5 sm:px-5 flex items-center justify-between gap-3 hover:bg-slate-50/50 transition-colors">
                  <div className="min-w-0">
                    <div className="text-xs font-mono font-bold text-slate-900 truncate">{evt.action}</div>
                    <div className="text-[11px] text-slate-500 truncate">{evt.reason || `Target: ${evt.target_type}`}</div>
                  </div>
                  <div className="text-[10px] text-slate-400 font-mono shrink-0">
                    {new Date(evt.created_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Latest Administrative Activity Feed */}
      <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
        <div className="px-5 py-4 border-b border-slate-100 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Clock className="w-4 h-4 text-slate-600" />
            <h3 className="text-sm font-bold text-slate-900">{t.recentActivity}</h3>
          </div>
          <Link
            href="/platform-admin/audit-logs"
            className="text-xs font-semibold text-blue-600 hover:text-blue-700 flex items-center gap-1"
          >
            <span>View all audit logs</span>
            {isRTL ? <ArrowLeft className="w-3.5 h-3.5" /> : <ArrowRight className="w-3.5 h-3.5" />}
          </Link>
        </div>

        {activity.length === 0 ? (
          <div className="p-8 text-center text-xs text-slate-400">
            No administrative actions recorded yet.
          </div>
        ) : (
          <div className="divide-y divide-slate-100">
            {activity.map((log) => (
              <div key={log.id} className="p-4 sm:px-5 flex flex-col sm:flex-row sm:items-center justify-between gap-2 hover:bg-slate-50/50 transition-colors">
                <div className="flex items-start sm:items-center gap-3">
                  <div className="w-2 h-2 rounded-full bg-blue-500 mt-1.5 sm:mt-0 shrink-0" />
                  <div>
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-xs font-bold text-slate-900 font-mono">
                        {log.action}
                      </span>
                      <span className="text-[11px] text-slate-400 font-medium">
                        on {log.target_type} {log.target_id ? `(${log.target_id.slice(0, 8)}...)` : ""}
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
