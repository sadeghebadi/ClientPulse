import React from "react";
import Link from "next/link";
import {
  CheckCircle2,
  AlertTriangle,
  XCircle,
  HelpCircle,
  Clock,
  ArrowRight,
  ArrowLeft,
  RefreshCw,
  Plus,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/Button";

export type HealthOverallState =
  | "operational"
  | "degraded"
  | "outage"
  | "heartbeat_attention"
  | "incomplete"
  | "no_active"
  | "empty"
  | "stale"
  | "unavailable";

interface SystemHealthBarProps {
  state: HealthOverallState;
  monitorsUp: number;
  monitorsDown: number;
  monitorsSuspectedDown: number;
  monitorsPending?: number;
  monitorsPaused?: number;
  monitorsUnknown?: number;
  openIncidents?: number;
  websiteOutageIncidents?: number;
  heartbeatMissedIncidents?: number;
  totalMonitors: number;
  isStale?: boolean;
  onRefresh?: () => void;
  refreshing?: boolean;
}

export function SystemHealthBar({
  state,
  monitorsUp,
  monitorsDown,
  monitorsSuspectedDown,
  monitorsPending = 0,
  monitorsPaused = 0,
  monitorsUnknown = 0,
  openIncidents = 0,
  websiteOutageIncidents = 0,
  heartbeatMissedIncidents = 0,
  totalMonitors,
  isStale = false,
  onRefresh,
  refreshing = false,
}: SystemHealthBarProps) {
  const { t, isRTL } = useI18n();

  const activeMonitors =
    monitorsUp + monitorsDown + monitorsSuspectedDown + monitorsPending + monitorsUnknown;
  const pausedText =
    monitorsPaused > 0 ? ` · ${monitorsPaused} ${t.statusPausedLabel.toLowerCase()}` : "";

  // Pick semantic styling and content based on strict operational state
  const config = {
    operational: {
      container: "bg-emerald-50/80 border-emerald-200 text-emerald-950",
      badge: "bg-emerald-100 text-emerald-800 border-emerald-300",
      icon: <CheckCircle2 className="w-5 h-5 text-emerald-600 shrink-0" />,
      title: t.allMonitoredTargetsOperational,
      description: `${monitorsUp} ${t.allMonitorsOperationalDesc.replace(
        "{total}",
        String(activeMonitors || totalMonitors)
      )}${pausedText}`,
      accentLink: "/monitors?status=up",
      linkText: t.viewAllMonitors,
    },
    degraded: {
      container: "bg-amber-50/90 border-amber-300 text-amber-950",
      badge: "bg-amber-100 text-amber-800 border-amber-300",
      icon: <AlertTriangle className="w-5 h-5 text-amber-600 shrink-0" />,
      title: `${t.attentionRequired} — ${t.statusSuspectedDownLabel}`,
      description: `${monitorsSuspectedDown} ${t.degradedHealthDesc} · ${monitorsUp}/${
        activeMonitors || totalMonitors
      } ${t.statusOperational.toLowerCase()}`,
      accentLink: "/monitors?status=suspected_down",
      linkText: t.viewAllMonitors,
    },
    outage: {
      container: "bg-rose-50 border-rose-300 text-rose-950",
      badge: "bg-rose-100 text-rose-800 border-rose-300",
      icon: <XCircle className="w-5 h-5 text-rose-600 shrink-0" />,
      title: `${t.attentionRequired} — ${t.websiteOutageDetected}`,
      description: `${
        monitorsDown > 0 ? `${monitorsDown} ${t.outageHealthDesc}` : ""
      }${
        monitorsDown > 0 && websiteOutageIncidents > 0 ? " · " : ""
      }${
        websiteOutageIncidents > 0
          ? `${websiteOutageIncidents} ${t.websiteOutageIncidentsDesc}`
          : ""
      }${
        monitorsSuspectedDown > 0 ? ` · ${monitorsSuspectedDown} ${t.suspectedDownSub}` : ""
      }`.trim() || `${monitorsDown} ${t.outageHealthDesc}`,
      accentLink: "/incidents?status=open&target_type=monitor",
      linkText: t.viewAllIncidents,
    },
    heartbeat_attention: {
      container: "bg-amber-50 border-amber-300 text-amber-950",
      badge: "bg-amber-100 text-amber-800 border-amber-300",
      icon: <AlertTriangle className="w-5 h-5 text-amber-600 shrink-0" />,
      title: t.heartbeatAttentionTitle,
      description: `${heartbeatMissedIncidents} ${t.heartbeatIncidentsCountDesc} · ${t.heartbeatAttentionDesc}`,
      accentLink: "/incidents?status=open&target_type=heartbeat",
      linkText: t.viewBackgroundJobs,
    },
    incomplete: {
      container: "bg-blue-50/80 border-blue-200 text-blue-950",
      badge: "bg-blue-100 text-blue-800 border-blue-300",
      icon: <HelpCircle className="w-5 h-5 text-blue-600 shrink-0" />,
      title: t.monitoringIncompleteTitle,
      description: `${monitorsPending + monitorsUnknown} ${t.incompleteHealthDesc} · ${monitorsUp}/${
        totalMonitors
      } ${t.statusOperational.toLowerCase()}`,
      accentLink: "/monitors",
      linkText: t.viewAllMonitors,
    },
    no_active: {
      container: "bg-slate-100 border-slate-300 text-slate-800",
      badge: "bg-slate-200 text-slate-700 border-slate-300",
      icon: <HelpCircle className="w-5 h-5 text-slate-500 shrink-0" />,
      title: t.noActiveMonitoringTitle,
      description: t.noActiveMonitoringDesc.replace("{total}", String(totalMonitors)),
      accentLink: "/monitors?status=paused",
      linkText: t.viewAllMonitors,
    },
    empty: {
      container: "bg-slate-50 border-slate-200 text-slate-800",
      badge: "bg-slate-100 text-slate-700 border-slate-300",
      icon: <HelpCircle className="w-5 h-5 text-slate-500 shrink-0" />,
      title: t.noMonitorsConfigured,
      description: t.noMonitorsConfiguredDesc,
      accentLink: "/monitors/new",
      linkText: t.addFirstMonitor,
    },
    stale: {
      container: "bg-amber-50/80 border-amber-200 text-amber-900",
      badge: "bg-amber-100 text-amber-800 border-amber-300",
      icon: <Clock className="w-5 h-5 text-amber-600 shrink-0" />,
      title: t.monitoringDataStaleWarning,
      description: `${monitorsUp}/${totalMonitors} ${t.statusOperational.toLowerCase()} · ${t.lastUpdated}`,
      accentLink: "/monitors",
      linkText: t.viewAllMonitors,
    },
    unavailable: {
      container: "bg-slate-100 border-slate-300 text-slate-700",
      badge: "bg-slate-200 text-slate-800 border-slate-300",
      icon: <AlertTriangle className="w-5 h-5 text-slate-500 shrink-0" />,
      title: t.dataTemporarilyUnavailable,
      description: t.widgetError,
      accentLink: "/monitors",
      linkText: t.viewAllMonitors,
    },
  }[state];

  return (
    <div
      role="region"
      aria-label={t.executiveHealthTitle}
      className={`rounded-2xl border p-4 sm:p-5 shadow-sm transition-colors ${config.container}`}
    >
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div className="flex items-start gap-3.5">
          <div className="p-2 rounded-xl bg-white/70 shadow-2xs mt-0.5">
            {config.icon}
          </div>
          <div>
            <div className="flex items-center gap-2 flex-wrap">
              <h2 className="text-sm sm:text-base font-bold tracking-tight">
                {config.title}
              </h2>
              {isStale && state !== "stale" && (
                <span className="text-[11px] font-semibold px-2 py-0.5 rounded-full bg-amber-100 text-amber-800 border border-amber-300">
                  {t.monitoringDataStaleWarning}
                </span>
              )}
            </div>
            <p className="text-xs mt-1 opacity-90 font-medium">
              {config.description}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2.5 w-full sm:w-auto justify-end shrink-0">
          {state === "empty" ? (
            <Link href="/monitors/new">
              <Button size="sm" variant="primary" icon={<Plus className="w-4 h-4" />}>
                {t.addFirstMonitor}
              </Button>
            </Link>
          ) : (
            <>
              {onRefresh && (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={onRefresh}
                  disabled={refreshing}
                  className="bg-white/80"
                  icon={
                    <RefreshCw
                      className={`w-3.5 h-3.5 ${refreshing ? "animate-spin" : ""}`}
                    />
                  }
                >
                  {t.refresh}
                </Button>
              )}
              <Link href={config.accentLink}>
                <Button size="sm" variant="secondary" className="bg-white/90">
                  <span>{config.linkText}</span>
                  {isRTL ? (
                    <ArrowLeft className="w-3.5 h-3.5" />
                  ) : (
                    <ArrowRight className="w-3.5 h-3.5" />
                  )}
                </Button>
              </Link>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

