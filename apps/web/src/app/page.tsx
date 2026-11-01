"use client";

import React, { useEffect, useState, useCallback } from "react";
import Link from "next/link";
import { AppLayout } from "@/components/AppLayout";
import { HealthChecker } from "@/components/HealthChecker";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import {
  ShieldCheck,
  Building,
  Users,
  Settings,
  ArrowRight,
  ArrowLeft,
  Cpu,
  Database,
  CheckCircle2,
  XCircle,
  Clock,
  LogIn,
  UserPlus,
  Globe,
  Activity,
  Plus,
  Zap,
  PauseCircle,
  RotateCcw,
  AlertTriangle,
  ExternalLink,
  Radio,
  AlertOctagon,
  Wrench,
  FileText,
} from "lucide-react";
import { MetricCard } from "@/components/ui/MetricCard";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { MetricSkeleton, TableSkeleton } from "@/components/ui/LoadingSkeleton";
import { ErrorState } from "@/components/ui/ErrorState";
import { EmptyState } from "@/components/ui/EmptyState";

interface RecentCheck {
  id: string;
  monitor_id: string;
  scheduled_at: string;
  completed_at: string;
  latency_ms: number;
  result: "success" | "failure" | "timeout" | "blocked" | "error";
  http_status_code?: number;
  error_message?: string;
  execution_type: "scheduled" | "manual";
}

interface ClientSummary {
  id: string;
  name: string;
  contact_name?: string;
  website?: string;
  asset_count: number;
  monitor_count: number;
  status: string;
}

interface DashboardStats {
  total_clients: number;
  total_assets: number;
  monitors_up: number;
  monitors_down: number;
  monitors_suspected_down?: number;
  monitors_pending: number;
  monitors_paused: number;
  monitors_unknown?: number;
  open_incidents?: number;
  total_heartbeats?: number;
  heartbeats_healthy?: number;
  heartbeats_missed?: number;
  heartbeats_pending?: number;
  recent_checks: RecentCheck[];
}

export default function Home() {
  const { user, activeOrg, role } = useAuth();
  const { t, isRTL, locale } = useI18n();

  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [clients, setClients] = useState<ClientSummary[]>([]);
  const [recentActivities, setRecentActivities] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<any>(null);

  const reloadDashboardData = useCallback(() => {
    if (!user || !activeOrg) {
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);

    Promise.all([
      apiFetch<DashboardStats>("/api/v1/dashboard/stats"),
      apiFetch<{ clients: ClientSummary[] }>("/api/v1/clients?limit=5&status=active"),
      apiFetch<{ activities: any[] }>("/api/v1/maintenance-activities?limit=5").catch(() => ({ activities: [] })),
    ])
      .then(([statsData, clientsData, actData]) => {
        setStats(statsData);
        setClients(clientsData.clients || []);
        setRecentActivities(actData.activities || []);
      })
      .catch((err) => {
        setError(err);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [user, activeOrg]);

  useEffect(() => {
    if (!user || !activeOrg) return;
    let isMounted = true;

    Promise.all([
      apiFetch<DashboardStats>("/api/v1/dashboard/stats"),
      apiFetch<{ clients: ClientSummary[] }>("/api/v1/clients?limit=5&status=active"),
      apiFetch<{ activities: any[] }>("/api/v1/maintenance-activities?limit=5").catch(() => ({ activities: [] })),
    ])
      .then(([statsData, clientsData, actData]) => {
        if (isMounted) {
          setStats(statsData);
          setClients(clientsData.clients || []);
          setRecentActivities(actData.activities || []);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setError(err);
        }
      })
      .finally(() => {
        if (isMounted) {
          setLoading(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user, activeOrg]);

  const failedChecks = stats?.recent_checks.filter((c) => c.result !== "success") || [];

  return (
    <AppLayout>
      {user ? (
        /* Authenticated Dashboard View */
        <div className="space-y-8">
          {/* Header Banner */}
          <div className="bg-gradient-to-r from-slate-900 via-indigo-950 to-blue-950 rounded-2xl p-6 sm:p-8 text-white shadow-lg flex flex-col md:flex-row items-start md:items-center justify-between gap-6 border border-slate-800">
            <div>
              <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-semibold bg-emerald-500/20 text-emerald-300 border border-emerald-400/30 mb-3">
                <ShieldCheck className="w-3.5 h-3.5" />
                <span>Multi-Tenant Monitoring Active</span>
              </div>
              <h1 className="text-2xl sm:text-3xl font-extrabold tracking-tight">
                {t.welcomeBack}, {user.display_name}!
              </h1>
              <div className="flex flex-wrap items-center gap-2 mt-2 text-blue-200 text-xs sm:text-sm">
                <span>{t.activeWorkspace}:</span>
                <span className="font-bold text-white bg-white/10 px-2.5 py-0.5 rounded-lg border border-white/10">
                  {activeOrg?.name || "Agency Workspace"}
                </span>
                {role && <Badge variant={role} />}
              </div>
            </div>

            {/* Quick Actions in Header */}
            <div className="flex flex-wrap items-center gap-2.5">
              <Link href="/clients/new">
                <Button size="sm" variant="primary" icon={<Plus className="w-4 h-4" />}>
                  {t.createClient}
                </Button>
              </Link>
              <Link href="/assets/new">
                <Button size="sm" variant="secondary" icon={<Globe className="w-4 h-4" />}>
                  {t.createAsset}
                </Button>
              </Link>
              <Link href="/monitors/new">
                <Button size="sm" variant="secondary" icon={<Activity className="w-4 h-4" />}>
                  {t.createMonitor}
                </Button>
              </Link>
              <Link href="/heartbeats/new">
                <Button size="sm" variant="secondary" icon={<Radio className="w-4 h-4" />}>
                  {t.createHeartbeat}
                </Button>
              </Link>
              <Link href="/maintenance">
                <Button size="sm" variant="secondary" icon={<Wrench className="w-4 h-4" />}>
                  {t.maintenance}
                </Button>
              </Link>
              <Link href="/reports">
                <Button size="sm" variant="secondary" icon={<FileText className="w-4 h-4" />}>
                  {t.reports}
                </Button>
              </Link>
            </div>
          </div>

          {error && <ErrorState error={error} onRetry={reloadDashboardData} />}

          {/* Active Incidents Alert Banner */}
          {stats && (stats.open_incidents || 0) > 0 && (
            <div className="bg-rose-50 border-2 border-rose-200 rounded-2xl p-4 sm:p-5 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 shadow-sm">
              <div className="flex items-center gap-3">
                <div className="p-2.5 bg-rose-600 text-white rounded-xl shadow-sm">
                  <AlertOctagon className="w-6 h-6" />
                </div>
                <div>
                  <h3 className="text-sm font-bold text-rose-950">
                    {stats.open_incidents} {t.openIncidents}
                  </h3>
                  <p className="text-xs text-rose-700 mt-0.5">
                    {isRTL
                      ? "یک یا چند هدف مانیتورینگ یا هارت‌بیت دچار قطعی تأییدشده هستند و نیاز به رسیدگی دارند."
                      : "One or more monitors or heartbeats are currently down or missed and require attention."}
                  </p>
                </div>
              </div>
              <Link href="/incidents?status=open">
                <Button size="sm" variant="danger">
                  {t.incidents}
                </Button>
              </Link>
            </div>
          )}

          {/* Section 1: Real Summary Metrics Grid */}
          {loading ? (
            <MetricSkeleton />
          ) : (
            <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4 sm:gap-6">
              <MetricCard
                title={t.totalClients}
                value={stats ? stats.total_clients : 0}
                icon={<Users className="w-5 h-5" />}
                href="/clients"
                trendColor="blue"
                actionText={t.manageClients}
              />
              <MetricCard
                title={t.totalAssets}
                value={stats ? stats.total_assets : 0}
                icon={<Globe className="w-5 h-5" />}
                href="/assets"
                trendColor="indigo"
                actionText={t.viewAllAssets}
              />
              <MetricCard
                title={t.monitorsUp}
                value={stats ? stats.monitors_up : 0}
                icon={<CheckCircle2 className="w-5 h-5" />}
                href="/monitors?status=up"
                trendColor="emerald"
                actionText={t.statusUp}
              />
              <MetricCard
                title={t.monitorsDown}
                value={stats ? stats.monitors_down : 0}
                icon={<XCircle className="w-5 h-5" />}
                href="/monitors?status=down"
                trendColor="rose"
                actionText={t.statusDown}
              />
              <MetricCard
                title={t.openIncidents}
                value={stats ? stats.open_incidents || 0 : 0}
                icon={<AlertTriangle className="w-5 h-5" />}
                href="/incidents?status=open"
                trendColor={(stats?.open_incidents || 0) > 0 ? "rose" : "emerald"}
                actionText={t.incidents}
              />
              <MetricCard
                title={t.heartbeats}
                value={stats ? stats.total_heartbeats || 0 : 0}
                icon={<Radio className="w-5 h-5" />}
                href="/heartbeats"
                trendColor="indigo"
                actionText={t.heartbeats}
              />
            </div>
          )}

          {/* Section 2: Monitor Health Overview Distribution Bar */}
          {stats && (
            <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-6">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <h2 className="text-base font-bold text-slate-900 tracking-tight">
                    {t.monitorHealthOverview}
                  </h2>
                  <p className="text-xs text-slate-500 mt-0.5">
                    Live operational distribution across all active agency monitors.
                  </p>
                </div>
                <Link
                  href="/monitors"
                  className="text-xs font-semibold text-blue-600 hover:text-blue-800 flex items-center gap-1"
                >
                  <span>{t.viewAllMonitors}</span>
                  {isRTL ? <ArrowLeft className="w-3.5 h-3.5" /> : <ArrowRight className="w-3.5 h-3.5" />}
                </Link>
              </div>

              <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3 pt-2">
                <Link
                  href="/monitors?status=up"
                  className="p-3 rounded-xl border border-emerald-100 bg-emerald-50/40 hover:bg-emerald-50 transition-colors block"
                >
                  <div className="text-[11px] font-semibold text-emerald-800">{t.statusUp}</div>
                  <div className="text-xl font-bold text-emerald-700 mt-1">{stats.monitors_up}</div>
                </Link>

                <Link
                  href="/monitors?status=suspected_down"
                  className="p-3 rounded-xl border border-amber-200 bg-amber-50/40 hover:bg-amber-50 transition-colors block"
                >
                  <div className="text-[11px] font-semibold text-amber-800">{t.statusSuspectedDown}</div>
                  <div className="text-xl font-bold text-amber-700 mt-1">
                    {stats.monitors_suspected_down || 0}
                  </div>
                </Link>

                <Link
                  href="/monitors?status=down"
                  className="p-3 rounded-xl border border-rose-200 bg-rose-50/40 hover:bg-rose-50 transition-colors block"
                >
                  <div className="text-[11px] font-semibold text-rose-800">{t.statusDown}</div>
                  <div className="text-xl font-bold text-rose-700 mt-1">{stats.monitors_down}</div>
                </Link>

                <Link
                  href="/monitors?status=pending"
                  className="p-3 rounded-xl border border-blue-100 bg-blue-50/40 hover:bg-blue-50 transition-colors block"
                >
                  <div className="text-[11px] font-semibold text-blue-800">{t.statusPending}</div>
                  <div className="text-xl font-bold text-blue-700 mt-1">{stats.monitors_pending}</div>
                </Link>

                <Link
                  href="/monitors?status=paused"
                  className="p-3 rounded-xl border border-slate-200 bg-slate-50 hover:bg-slate-100 transition-colors block"
                >
                  <div className="text-[11px] font-semibold text-slate-700">{t.statusPaused}</div>
                  <div className="text-xl font-bold text-slate-800 mt-1">{stats.monitors_paused}</div>
                </Link>

                <Link
                  href="/monitors?status=unknown"
                  className="p-3 rounded-xl border border-slate-200 bg-slate-50 hover:bg-slate-100 transition-colors block"
                >
                  <div className="text-[11px] font-semibold text-slate-700">{t.statusUnknown}</div>
                  <div className="text-xl font-bold text-slate-800 mt-1">{stats.monitors_unknown || 0}</div>
                </Link>
              </div>
            </div>
          )}

          {/* Section 3: Recently Failed Checks Panel */}
          <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
            <div className="p-6 border-b border-slate-100 flex items-center justify-between">
              <div className="flex items-center gap-2.5">
                <div className="p-2 rounded-xl bg-rose-50 text-rose-600">
                  <AlertTriangle className="w-5 h-5" />
                </div>
                <div>
                  <h2 className="text-base font-bold text-slate-900 tracking-tight">
                    {t.recentlyFailedChecks}
                  </h2>
                  <p className="text-xs text-slate-500 mt-0.5">
                    Critical checks reporting HTTP error codes, timeout, or policy blocks.
                  </p>
                </div>
              </div>
              <Badge variant={failedChecks.length > 0 ? "failure" : "success"} />
            </div>

            {failedChecks.length === 0 ? (
              <div className="p-8 text-center text-xs text-slate-500">
                <CheckCircle2 className="w-8 h-8 text-emerald-500 mx-auto mb-2" />
                <p className="font-semibold text-slate-700">{t.noFailedChecks}</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm" dir={isRTL ? "rtl" : "ltr"}>
                  <thead className="bg-slate-50 text-slate-600 text-xs uppercase font-semibold border-b border-slate-200">
                    <tr>
                      <th className="px-6 py-3.5">{t.status}</th>
                      <th className="px-6 py-3.5">{t.latestCheck}</th>
                      <th className="px-6 py-3.5">{t.latency}</th>
                      <th className="px-6 py-3.5">{t.statusCode}</th>
                      <th className="px-6 py-3.5">Error Message</th>
                      <th className="px-6 py-3.5 text-right">{t.actions}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 text-xs">
                    {failedChecks.map((chk) => (
                      <tr key={chk.id} className="hover:bg-rose-50/30 transition-colors">
                        <td className="px-6 py-3.5">
                          <Badge variant={chk.result} />
                        </td>
                        <td className="px-6 py-3.5 text-slate-600">
                          {new Date(chk.completed_at).toLocaleTimeString(undefined, {
                            timeZone: activeOrg?.timezone || "UTC",
                          })}
                        </td>
                        <td className="px-6 py-3.5 font-mono font-medium text-slate-800">
                          {chk.latency_ms} ms
                        </td>
                        <td className="px-6 py-3.5 font-mono font-bold text-rose-600">
                          {chk.http_status_code || "—"}
                        </td>
                        <td className="px-6 py-3.5 text-rose-700 font-mono text-[11px] max-w-xs truncate">
                          {chk.error_message || "Target unresponsive"}
                        </td>
                        <td className="px-6 py-3.5 text-right">
                          <Link
                            href={`/monitors/${chk.monitor_id}`}
                            className="font-semibold text-blue-600 hover:underline"
                          >
                            {t.viewMonitor}
                          </Link>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Section 4 & 5: Latest Monitoring Activity & Clients Summary */}
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
            {/* Latest Checks (2 Cols) */}
            <div className="lg:col-span-2 bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
              <div className="p-6 border-b border-slate-100 flex items-center justify-between">
                <div>
                  <h2 className="text-base font-bold text-slate-900 tracking-tight flex items-center gap-2">
                    <Activity className="w-5 h-5 text-blue-600" />
                    <span>{t.latestMonitoringActivity}</span>
                  </h2>
                  <p className="text-xs text-slate-500 mt-0.5">
                    Continuous verification cycles recorded by the background Go worker.
                  </p>
                </div>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={reloadDashboardData}
                  icon={<RotateCcw className="w-3.5 h-3.5" />}
                >
                  {t.refresh}
                </Button>
              </div>

              {!stats || stats.recent_checks.length === 0 ? (
                <div className="p-12 text-center text-slate-500 text-xs">{t.noHistory}</div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm" dir={isRTL ? "rtl" : "ltr"}>
                    <thead className="bg-slate-50 text-slate-600 text-xs uppercase font-semibold border-b border-slate-200">
                      <tr>
                        <th className="px-5 py-3.5">{t.status}</th>
                        <th className="px-5 py-3.5">{t.latestCheck}</th>
                        <th className="px-5 py-3.5">{t.latency}</th>
                        <th className="px-5 py-3.5">{t.statusCode}</th>
                        <th className="px-5 py-3.5 text-right">{t.actions}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 text-xs">
                      {stats.recent_checks.slice(0, 10).map((chk) => (
                        <tr key={chk.id} className="hover:bg-slate-50/70 transition-colors">
                          <td className="px-5 py-3.5">
                            <Badge variant={chk.result} />
                          </td>
                          <td className="px-5 py-3.5 text-slate-600">
                            {new Date(chk.completed_at).toLocaleTimeString(undefined, {
                              timeZone: activeOrg?.timezone || "UTC",
                            })}
                          </td>
                          <td className="px-5 py-3.5 font-mono font-medium text-slate-800">
                            {chk.latency_ms} ms
                          </td>
                          <td className="px-5 py-3.5 font-mono font-bold text-slate-800">
                            {chk.http_status_code || "—"}
                          </td>
                          <td className="px-5 py-3.5 text-right">
                            <Link
                              href={`/monitors/${chk.monitor_id}`}
                              className="font-semibold text-blue-600 hover:underline"
                            >
                              {t.viewMonitor}
                            </Link>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>

            {/* Clients Summary Panel (1 Col) */}
            <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden flex flex-col justify-between">
              <div>
                <div className="p-6 border-b border-slate-100 flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Users className="w-5 h-5 text-blue-600" />
                    <h2 className="text-base font-bold text-slate-900 tracking-tight">
                      {t.clientsSummary}
                    </h2>
                  </div>
                  <Link
                    href="/clients"
                    className="text-xs font-semibold text-blue-600 hover:underline"
                  >
                    {t.viewDetails}
                  </Link>
                </div>

                <div className="p-4 space-y-3">
                  {clients.length === 0 ? (
                    <div className="py-8 text-center text-xs text-slate-400">
                      {t.noClientsFound}
                    </div>
                  ) : (
                    clients.map((c) => (
                      <Link
                        key={c.id}
                        href={`/clients/${c.id}`}
                        className="p-3 rounded-xl border border-slate-100 hover:border-blue-200 hover:bg-slate-50/80 transition-all flex items-center justify-between group block"
                      >
                        <div>
                          <div className="font-bold text-xs text-slate-900 group-hover:text-blue-600 transition-colors">
                            {c.name}
                          </div>
                          <div className="text-[11px] text-slate-500 mt-0.5">
                            {c.contact_name || c.website || "No contact info"}
                          </div>
                        </div>
                        <div className="flex items-center gap-2 text-[11px] font-semibold text-slate-600">
                          <span title="Assets" className="flex items-center gap-1">
                            <Globe className="w-3 h-3 text-slate-400" />
                            <span>{c.asset_count}</span>
                          </span>
                          <span title="Monitors" className="flex items-center gap-1">
                            <Activity className="w-3 h-3 text-slate-400" />
                            <span>{c.monitor_count}</span>
                          </span>
                        </div>
                      </Link>
                    ))
                  )}
                </div>
              </div>

              <div className="p-4 bg-slate-50 border-t border-slate-100">
                <Link href="/clients/new">
                  <Button size="sm" variant="outline" className="w-full" icon={<Plus className="w-3.5 h-3.5" />}>
                    {t.createClient}
                  </Button>
                </Link>
              </div>
            </div>
          </div>

          {/* Section 5.5: Recent Maintenance Activities */}
          <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
            <div className="p-6 border-b border-slate-100 flex items-center justify-between">
              <div>
                <h2 className="text-base font-bold text-slate-900 tracking-tight flex items-center gap-2">
                  <Wrench className="w-5 h-5 text-indigo-600" />
                  <span>{t.maintenance}</span>
                </h2>
                <p className="text-xs text-slate-500 mt-0.5">
                  {isRTL
                    ? "اقدامات و کارهای اخیر انجام‌شده برای سایت‌های مشتریان جهت درج در گزارش‌های ماهانه."
                    : "Recent maintenance work performed for client assets, ready for monthly care reporting."}
                </p>
              </div>
              <Link href="/maintenance">
                <Button size="sm" variant="outline">
                  {isRTL ? "مشاهده همه کارها" : "View All"}
                </Button>
              </Link>
            </div>

            {recentActivities.length === 0 ? (
              <div className="p-8 text-center text-slate-400 text-xs">
                {t.noActivitiesFound}
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm" dir={isRTL ? "rtl" : "ltr"}>
                  <thead className="bg-slate-50 text-slate-600 text-xs uppercase font-semibold border-b border-slate-200">
                    <tr>
                      <th className="px-5 py-3.5">{t.clientName}</th>
                      <th className="px-5 py-3.5">{t.activityTitle}</th>
                      <th className="px-5 py-3.5">{t.activityCategory}</th>
                      <th className="px-5 py-3.5">{t.activityPerformedAt}</th>
                      <th className="px-5 py-3.5">{t.activityDuration}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 text-xs">
                    {recentActivities.map((act) => (
                      <tr key={act.id} className="hover:bg-slate-50/70 transition-colors">
                        <td className="px-5 py-3.5 font-medium text-slate-900">
                          {act.client_name || act.client_id}
                        </td>
                        <td className="px-5 py-3.5">
                          <span className="font-semibold text-slate-800">{act.title}</span>
                          {act.asset_name && (
                            <span className="text-slate-400 text-[11px] block">{act.asset_name}</span>
                          )}
                        </td>
                        <td className="px-5 py-3.5">
                          <span className="px-2 py-0.5 text-[11px] font-medium rounded-full bg-slate-100 text-slate-700">
                            {act.category}
                          </span>
                        </td>
                        <td className="px-5 py-3.5 text-slate-500">
                          {new Date(act.performed_at).toLocaleDateString(locale === "fa" ? "fa-IR" : "en-US", {
                            month: "short",
                            day: "numeric",
                          })}
                        </td>
                        <td className="px-5 py-3.5 text-slate-600">
                          {act.duration_minutes ? `${act.duration_minutes} min` : "—"}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* Section 6: Live Infrastructure Health Checker */}
          <HealthChecker locale={locale} />
        </div>
      ) : (
        /* Public Guest View */
        <div className="max-w-4xl mx-auto py-8">
          <div className="bg-gradient-to-r from-slate-900 via-indigo-950 to-blue-950 rounded-2xl p-8 sm:p-12 mb-8 text-white shadow-xl border border-slate-800">
            <div className="max-w-2xl">
              <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full text-xs font-semibold bg-blue-500/30 text-blue-200 border border-blue-400/30 mb-4">
                <ShieldCheck className="w-3.5 h-3.5" />
                <span>ClientPulse B2B SaaS Platform</span>
              </div>
              <h2 className="text-3xl sm:text-4xl font-extrabold tracking-tight mb-4">
                {isRTL
                  ? "پلتفرم پایش وب‌سایت و مراقبت از کلاینت‌های آژانس"
                  : "Agency Website Monitoring & Reliability Care"}
              </h2>
              <p className="text-base text-blue-100/90 leading-relaxed mb-8">
                {isRTL
                  ? "سیستم جامع مدیریت مشتریان، ثبت وب‌سایت‌ها، موتور پایش سلامت با زمان‌بندی دقیق در PostgreSQL، محافظت چندلایه در برابر SSRF و بررسی‌های زنده HTTP/SSL/DNS."
                  : "Complete agency client management, website assets registry, automated PostgreSQL-backed monitoring scheduler, multi-layer SSRF guard, and live HTTP, SSL and DNS checks."}
              </p>

              <div className="flex flex-wrap items-center gap-3">
                <Link href="/signup">
                  <Button size="lg" variant="primary" icon={<UserPlus className="w-4 h-4" />}>
                    {t.createAccount}
                  </Button>
                </Link>
                <Link href="/login">
                  <Button size="lg" variant="secondary" icon={<LogIn className="w-4 h-4" />}>
                    {t.login}
                  </Button>
                </Link>
              </div>
            </div>
          </div>

          {/* Live Service Health Checker */}
          <HealthChecker locale={locale} />
        </div>
      )}
    </AppLayout>
  );
}
