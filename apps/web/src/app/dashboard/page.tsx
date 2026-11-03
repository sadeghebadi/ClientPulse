"use client";

import React, { useEffect, useState, useCallback, useMemo } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { AppLayout } from "@/components/AppLayout";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { apiFetch } from "@/lib/api";
import {
  ShieldCheck,
  Building,
  Users,
  ArrowRight,
  ArrowLeft,
  Clock,
  Globe,
  Activity,
  Plus,
  RotateCcw,
} from "lucide-react";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { ErrorState } from "@/components/ui/ErrorState";
import { useToast } from "@/components/ui/Toast";

// Dashboard Components
import {
  SystemHealthBar,
  HealthOverallState,
} from "@/components/dashboard/SystemHealthBar";
import { PrimaryKpiCards } from "@/components/dashboard/PrimaryKpiCards";
import { MonitorDistribution } from "@/components/dashboard/MonitorDistribution";
import {
  ActiveIncidentsList,
  ActiveIncidentItem,
} from "@/components/dashboard/ActiveIncidentsList";
import {
  RecentFailedChecks,
  FailedCheckItem,
} from "@/components/dashboard/RecentFailedChecks";
import {
  HeartbeatSummary,
  HeartbeatSummaryData,
} from "@/components/dashboard/HeartbeatSummary";
import {
  RecentMaintenance,
  MaintenanceActivityItem,
} from "@/components/dashboard/RecentMaintenance";
import {
  CareReportsOverview,
  CareReportSummaryItem,
} from "@/components/dashboard/CareReportsOverview";
import { DashboardSkeleton } from "@/components/dashboard/DashboardSkeleton";

interface RecentCheck {
  id: string;
  monitor_id: string;
  scheduled_at: string;
  completed_at: string;
  latency_ms: number;
  result: "success" | "failure" | "timeout" | "blocked" | "error";
  http_status_code?: number;
  error_category?: string;
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
  unacknowledged_incidents?: number;
  acknowledged_incidents?: number;
  resolved_incidents?: number;
  website_outage_incidents?: number;
  heartbeat_missed_incidents?: number;
  total_heartbeats?: number;
  heartbeats_healthy?: number;
  heartbeats_missed?: number;
  heartbeats_pending?: number;
  recent_checks: RecentCheck[];
}

export default function DashboardPage() {
  const router = useRouter();
  const { user, activeOrg, role, loading: authLoading } = useAuth();
  const { t, isRTL, locale } = useI18n();
  const toast = useToast();

  // Core State
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [incidents, setIncidents] = useState<ActiveIncidentItem[]>([]);
  const [heartbeats, setHeartbeats] = useState<any[]>([]);
  const [activities, setActivities] = useState<MaintenanceActivityItem[]>([]);
  const [reports, setReports] = useState<CareReportSummaryItem[]>([]);
  const [clients, setClients] = useState<ClientSummary[]>([]);

  // Metadata & Async State
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [error, setError] = useState<any>(null);
  const [acknowledgingId, setAcknowledgingId] = useState<string | null>(null);

  const canModify = role === "owner" || role === "admin" || role === "member";
  const [currentTime, setCurrentTime] = useState<number>(0);

  // Protected route guard: redirect to login if unauthenticated
  useEffect(() => {
    if (!authLoading && !user) {
      router.replace("/login");
    }
  }, [user, authLoading, router]);

  // Data Reload with Isolated Widget Resilience
  const loadDashboardData = useCallback(
    async (isManualRefresh = false) => {
      if (!user || !activeOrg) {
        return;
      }

      if (isManualRefresh) {
        setRefreshing(true);
      } else {
        setLoading(true);
      }
      setError(null);

      try {
        const [
          statsRes,
          incidentsRes,
          heartbeatsRes,
          activitiesRes,
          reportsRes,
          clientsRes,
        ] = await Promise.all([
          apiFetch<DashboardStats>("/api/v1/dashboard/stats"),
          apiFetch<{ incidents: ActiveIncidentItem[]; total: number }>(
            "/api/v1/incidents?status=open&limit=10"
          ).catch((e) => {
            console.warn("Failed to fetch open incidents:", e);
            return { incidents: [], total: 0 };
          }),
          apiFetch<{ heartbeats: any[]; total: number }>(
            "/api/v1/heartbeats?limit=10"
          ).catch((e) => {
            console.warn("Failed to fetch heartbeats list:", e);
            return { heartbeats: [], total: 0 };
          }),
          apiFetch<{ activities: MaintenanceActivityItem[] }>(
            "/api/v1/maintenance-activities?limit=5"
          ).catch((e) => {
            console.warn("Failed to fetch maintenance activities:", e);
            return { activities: [] };
          }),
          apiFetch<{ reports: CareReportSummaryItem[] }>(
            "/api/v1/reports?limit=5"
          ).catch((e) => {
            console.warn("Failed to fetch care reports:", e);
            return { reports: [] };
          }),
          apiFetch<{ clients: ClientSummary[] }>(
            "/api/v1/clients?limit=5&status=active"
          ).catch((e) => {
            console.warn("Failed to fetch clients summary:", e);
            return { clients: [] };
          }),
        ]);

        setStats(statsRes);
        setIncidents(incidentsRes.incidents || []);
        setHeartbeats(heartbeatsRes.heartbeats || []);
        setActivities(activitiesRes.activities || []);
        setReports(reportsRes.reports || []);
        setClients(clientsRes.clients || []);
        const now = new Date();
        setLastUpdated(now);
        setCurrentTime(now.getTime());
      } catch (err: any) {
        console.error("Dashboard primary stats failed to load:", err);
        setError(err);
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [user, activeOrg]
  );

  useEffect(() => {
    if (!user || !activeOrg) return;
    let isMounted = true;

    Promise.all([
      apiFetch<DashboardStats>("/api/v1/dashboard/stats"),
      apiFetch<{ incidents: ActiveIncidentItem[]; total: number }>(
        "/api/v1/incidents?status=open&limit=10"
      ).catch(() => ({ incidents: [], total: 0 })),
      apiFetch<{ heartbeats: any[]; total: number }>(
        "/api/v1/heartbeats?limit=10"
      ).catch(() => ({ heartbeats: [], total: 0 })),
      apiFetch<{ activities: MaintenanceActivityItem[] }>(
        "/api/v1/maintenance-activities?limit=5"
      ).catch(() => ({ activities: [] })),
      apiFetch<{ reports: CareReportSummaryItem[] }>(
        "/api/v1/reports?limit=5"
      ).catch(() => ({ reports: [] })),
      apiFetch<{ clients: ClientSummary[] }>(
        "/api/v1/clients?limit=5&status=active"
      ).catch(() => ({ clients: [] })),
    ])
      .then(([statsRes, incidentsRes, heartbeatsRes, activitiesRes, reportsRes, clientsRes]) => {
        if (!isMounted) return;
        setStats(statsRes);
        setIncidents(incidentsRes.incidents || []);
        setHeartbeats(heartbeatsRes.heartbeats || []);
        setActivities(activitiesRes.activities || []);
        setReports(reportsRes.reports || []);
        setClients(clientsRes.clients || []);
        const now = new Date();
        setLastUpdated(now);
        setCurrentTime(now.getTime());
      })
      .catch((err) => {
        if (!isMounted) return;
        setError(err);
      })
      .finally(() => {
        if (!isMounted) return;
        setLoading(false);
      });

    return () => {
      isMounted = false;
    };
  }, [user, activeOrg]);

  // Acknowledge incident handler
  const handleAcknowledge = async (id: string) => {
    setAcknowledgingId(id);
    try {
      await apiFetch(`/api/v1/incidents/${id}/acknowledge`, { method: "POST" });
      toast.success(
        isRTL ? "حادثه با موفقیت تأیید شد" : "Incident acknowledged successfully"
      );
      const res = await apiFetch<{
        incidents: ActiveIncidentItem[];
        total: number;
      }>("/api/v1/incidents?status=open&limit=10");
      setIncidents(res.incidents || []);
    } catch (err: any) {
      toast.error(
        err.message ||
          (isRTL
            ? "خطا در تأیید رخداد"
            : "Failed to acknowledge incident")
      );
    } finally {
      setAcknowledgingId(null);
    }
  };

  // Derive Monitor Counts strictly according to TASK-008.B.3.1 rules
  const counts = useMemo(() => {
    const up = stats?.monitors_up ?? 0;
    const down = stats?.monitors_down ?? 0;
    const suspectedDown = stats?.monitors_suspected_down ?? 0;
    const pending = stats?.monitors_pending ?? 0;
    const paused = stats?.monitors_paused ?? 0;
    const unknown = stats?.monitors_unknown ?? 0;
    const totalMonitors = up + down + suspectedDown + pending + paused + unknown;

    const openIncidents =
      stats?.open_incidents ??
      incidents.filter((i) => i.status === "open").length;
    const unacknowledgedIncidents =
      stats?.unacknowledged_incidents ??
      incidents.filter((i) => i.status === "open" && !i.acknowledged_at).length;
    const acknowledgedIncidents =
      stats?.acknowledged_incidents ??
      incidents.filter((i) => i.status === "open" && !i.acknowledged_at).length;
    const websiteOutageIncidents =
      stats?.website_outage_incidents ??
      incidents.filter((i) => i.status === "open" && i.target_type === "monitor").length;
    const heartbeatMissedIncidents =
      stats?.heartbeat_missed_incidents ??
      incidents.filter((i) => i.status === "open" && i.target_type === "heartbeat").length;

    return {
      up,
      down,
      suspectedDown,
      pending,
      paused,
      unknown,
      totalMonitors,
      openIncidents,
      unacknowledgedIncidents,
      acknowledgedIncidents,
      websiteOutageIncidents,
      heartbeatMissedIncidents,
    };
  }, [stats, incidents]);

  // Derive Average Response Latency ms from valid scheduled check results
  const latencyData = useMemo(() => {
    if (!stats || !stats.recent_checks || stats.recent_checks.length === 0) {
      return { avgMs: null, count: 0 };
    }
    const validSuccessChecks = stats.recent_checks.filter(
      (c) => c.result === "success" && typeof c.latency_ms === "number" && c.latency_ms >= 0
    );
    if (validSuccessChecks.length === 0) {
      return { avgMs: null, count: 0 };
    }
    const totalLatency = validSuccessChecks.reduce(
      (acc, c) => acc + c.latency_ms,
      0
    );
    return {
      avgMs: Math.round(totalLatency / validSuccessChecks.length),
      count: validSuccessChecks.length,
    };
  }, [stats]);

  // Derive Recent Failed Checks (result !== "success")
  const failedChecks: FailedCheckItem[] = useMemo(() => {
    if (!stats || !stats.recent_checks) return [];
    return stats.recent_checks.filter((c) => c.result !== "success");
  }, [stats]);

  // Derive Heartbeat Summary
  const heartbeatSummary: HeartbeatSummaryData = useMemo(() => {
    const total = stats?.total_heartbeats ?? heartbeats.length;
    const healthy = stats?.heartbeats_healthy ?? heartbeats.filter((h) => h.status === "healthy").length;
    const missed = stats?.heartbeats_missed ?? heartbeats.filter((h) => h.status === "missed").length;
    const pending = stats?.heartbeats_pending ?? heartbeats.filter((h) => h.status === "pending").length;

    const missedItem = heartbeats.find((h) => h.status === "missed");

    return {
      total,
      healthy,
      missed,
      pending,
      latestMissedName: missedItem?.name,
      latestMissedAt: missedItem?.last_received_at,
    };
  }, [stats, heartbeats]);

  // Check if monitoring data is stale (over 10 minutes without refresh)
  const isStale = useMemo(() => {
    if (!lastUpdated || !currentTime) return false;
    const diffMs = currentTime - lastUpdated.getTime();
    return diffMs > 10 * 60 * 1000;
  }, [lastUpdated, currentTime]);

  // Derive Overall System Health State according to Precedence
  const overallState: HealthOverallState = useMemo(() => {
    if (error && !stats) return "unavailable";
    if (counts.websiteOutageIncidents > 0 || counts.down > 0) return "outage";
    if (counts.heartbeatMissedIncidents > 0) return "heartbeat_attention";
    if (counts.suspectedDown > 0) return "degraded";
    if (counts.pending > 0 || counts.unknown > 0) return "incomplete";
    const activeMonitors =
      counts.up + counts.down + counts.suspectedDown + counts.pending + counts.unknown;
    if (counts.totalMonitors > 0 && activeMonitors === 0) return "no_active";
    if (counts.totalMonitors === 0) return "empty";
    if (isStale) return "stale";
    return "operational";
  }, [counts, error, stats, isStale]);

  if (authLoading || !user) {
    return (
      <AppLayout>
        <DashboardSkeleton />
      </AppLayout>
    );
  }

  return (
    <AppLayout>
      <div className="space-y-6 sm:space-y-8">
        {/* Section A — Professional Page Heading */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 pb-2 border-b border-slate-200">
          <div>
            <div className="flex items-center gap-2 mb-1">
              <span className="text-xs font-bold uppercase tracking-wider text-slate-500">
                {t.overview}
              </span>
              <span className="text-slate-300">·</span>
              <span className="text-xs font-semibold text-slate-700 bg-slate-100 px-2 py-0.5 rounded-md">
                {activeOrg?.name || "Agency Workspace"}
              </span>
              {role && <Badge variant={role} />}
            </div>
            <h1 className="text-xl sm:text-2xl font-extrabold text-slate-900 tracking-tight">
              {t.welcomeBack}, {user.display_name}!
            </h1>
            <p className="text-xs text-slate-500 mt-0.5">
              {isRTL
                ? "مرکز پایش عملیاتی وضعیت وب‌سایت‌ها، APIها و جاب‌های پس‌زمینه مشتریان."
                : "Operational monitoring control center for client websites, APIs, and background heartbeat jobs."}
            </p>
          </div>

          {/* Header Right: Last Updated Timestamp & Quick Action Buttons */}
          <div className="flex flex-wrap items-center gap-2 shrink-0">
            {lastUpdated && (
              <div
                className="text-xs text-slate-500 flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-slate-50 border border-slate-200 tabular-nums"
                title={lastUpdated.toISOString()}
              >
                <Clock className="w-3.5 h-3.5 text-slate-400" />
                <span>
                  {t.lastUpdated}:{" "}
                  {lastUpdated.toLocaleTimeString(locale === "fa" ? "fa-IR" : "en-US", {
                    timeZone: activeOrg?.timezone || "UTC",
                    hour: "2-digit",
                    minute: "2-digit",
                    second: "2-digit",
                  })}
                </span>
              </div>
            )}

            <Button
              size="sm"
              variant="outline"
              onClick={() => loadDashboardData(true)}
              disabled={refreshing}
              icon={
                <RotateCcw
                  className={`w-3.5 h-3.5 ${refreshing ? "animate-spin" : ""}`}
                />
              }
            >
              {t.refresh}
            </Button>

            <Link href="/monitors/new">
              <Button size="sm" variant="primary" icon={<Plus className="w-4 h-4" />}>
                {t.createMonitor}
              </Button>
            </Link>
          </div>
        </div>

        {/* Primary Error Notification */}
        {error && <ErrorState error={error} onRetry={() => loadDashboardData(true)} />}

        {loading ? (
          <DashboardSkeleton />
        ) : (
          <>
            {/* Section B — Executive System Health Bar */}
            <SystemHealthBar
              state={overallState}
              monitorsUp={counts.up}
              monitorsDown={counts.down}
              monitorsSuspectedDown={counts.suspectedDown}
              monitorsPending={counts.pending}
              monitorsPaused={counts.paused}
              monitorsUnknown={counts.unknown}
              openIncidents={counts.openIncidents}
              websiteOutageIncidents={counts.websiteOutageIncidents}
              heartbeatMissedIncidents={counts.heartbeatMissedIncidents}
              totalMonitors={counts.totalMonitors}
              isStale={isStale}
              onRefresh={() => loadDashboardData(true)}
              refreshing={refreshing}
            />

            {/* Section C — Four Primary KPI Cards */}
            <PrimaryKpiCards
              totalMonitoredTargets={counts.totalMonitors}
              confirmedDown={counts.down}
              suspectedDown={counts.suspectedDown}
              openIncidents={counts.openIncidents}
              unacknowledgedIncidents={counts.unacknowledgedIncidents}
              acknowledgedIncidents={counts.acknowledgedIncidents}
              avgLatencyMs={latencyData.avgMs}
              successfulMeasurementsCount={latencyData.count}
            />

            {/* Section F — Balanced Operational Split View (65% / 35%) */}
            <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
              {/* Main Operational Column (col-span-8 ~67%) */}
              <div className="lg:col-span-8 space-y-6">
                {/* Section D — Monitor Health Distribution */}
                <MonitorDistribution
                  up={counts.up}
                  suspectedDown={counts.suspectedDown}
                  down={counts.down}
                  paused={counts.paused}
                  pending={counts.pending}
                  unknown={counts.unknown}
                />

                {/* Section E.1 — Active Incidents */}
                <ActiveIncidentsList
                  incidents={incidents}
                  onAcknowledge={handleAcknowledge}
                  acknowledgingId={acknowledgingId}
                  canAcknowledge={canModify}
                />

                {/* Section E.2 — Recent Failed Checks */}
                <RecentFailedChecks
                  failedChecks={failedChecks}
                  timezone={activeOrg?.timezone || "UTC"}
                />
              </div>

              {/* Supporting Information Column (col-span-4 ~33%) */}
              <div className="lg:col-span-4 space-y-6">
                {/* Section G — Heartbeat Summary */}
                <HeartbeatSummary summary={heartbeatSummary} />

                {/* Section H — Recent Maintenance Activities */}
                <RecentMaintenance activities={activities} />

                {/* Section I — Care Reports Overview */}
                <CareReportsOverview reports={reports} canGenerate={canModify} />

                {/* Clients Quick Link Summary */}
                <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-5 sm:p-6">
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex items-center gap-2">
                      <Users className="w-4 h-4 text-blue-600" />
                      <h3 className="text-sm font-bold text-slate-900 tracking-tight">
                        {t.clientsSummary}
                      </h3>
                    </div>
                    <Link
                      href="/clients"
                      className="text-xs font-semibold text-blue-600 hover:text-blue-800 flex items-center gap-1 transition-colors"
                    >
                      <span>{t.manageClients}</span>
                      {isRTL ? (
                        <ArrowLeft className="w-3.5 h-3.5" />
                      ) : (
                        <ArrowRight className="w-3.5 h-3.5" />
                      )}
                    </Link>
                  </div>

                  {clients.length === 0 ? (
                    <div className="p-4 text-center text-xs text-slate-400 bg-slate-50 rounded-xl">
                      {t.noClientsFound}
                    </div>
                  ) : (
                    <div className="space-y-2">
                      {clients.slice(0, 4).map((c) => (
                        <Link
                          key={c.id}
                          href={`/clients/${c.id}`}
                          className="p-2.5 rounded-xl border border-slate-100 hover:border-blue-200 hover:bg-slate-50 transition-all flex items-center justify-between group block"
                        >
                          <span className="text-xs font-bold text-slate-800 group-hover:text-blue-600 transition-colors truncate">
                            {c.name}
                          </span>
                          <div className="flex items-center gap-2 text-[11px] font-semibold text-slate-500 tabular-nums">
                            <span
                              title="Assets"
                              className="flex items-center gap-0.5"
                            >
                              <Globe className="w-3 h-3 text-slate-400" />
                              <span>{c.asset_count}</span>
                            </span>
                            <span
                              title="Monitors"
                              className="flex items-center gap-0.5"
                            >
                              <Activity className="w-3 h-3 text-slate-400" />
                              <span>{c.monitor_count}</span>
                            </span>
                          </div>
                        </Link>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </div>
          </>
        )}
      </div>
    </AppLayout>
  );
}

