import React from "react";
import { Globe, XCircle, AlertTriangle, Activity } from "lucide-react";
import { MetricCard } from "@/components/ui/MetricCard";
import { useI18n } from "@/lib/i18n";

interface PrimaryKpiCardsProps {
  totalMonitoredTargets: number;
  confirmedDown: number;
  suspectedDown: number;
  openIncidents: number;
  unacknowledgedIncidents?: number;
  acknowledgedIncidents?: number;
  avgLatencyMs: number | null;
  successfulMeasurementsCount: number;
}

export function PrimaryKpiCards({
  totalMonitoredTargets,
  confirmedDown,
  suspectedDown,
  openIncidents,
  unacknowledgedIncidents,
  acknowledgedIncidents,
  avgLatencyMs,
  successfulMeasurementsCount,
}: PrimaryKpiCardsProps) {
  const { t } = useI18n();

  return (
    <div
      role="region"
      aria-label="Key Performance Indicators"
      className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 sm:gap-6"
    >
      {/* 5.1 Monitored Targets */}
      <MetricCard
        title={t.monitoredTargetsKpi}
        value={<span className="tabular-nums font-mono">{totalMonitoredTargets}</span>}
        icon={<Globe className="w-5 h-5 text-blue-600" />}
        subtitle={t.monitoredTargetsKpiSub}
        href="/monitors"
        trendColor="blue"
        actionText={t.viewAllMonitors}
      />

      {/* 5.2 Targets Down */}
      <MetricCard
        title={t.targetsDownKpi}
        value={<span className="tabular-nums font-mono">{confirmedDown}</span>}
        icon={<XCircle className="w-5 h-5 text-rose-600" />}
        subtitle={
          suspectedDown > 0
            ? `${confirmedDown} ${t.targetsDownKpiSub} · ${suspectedDown} ${t.suspectedDownSub}`
            : `${confirmedDown} ${t.targetsDownKpiSub}`
        }
        href={confirmedDown > 0 ? "/monitors?status=down" : "/monitors"}
        trendColor={confirmedDown > 0 ? "rose" : "slate"}
        actionText={confirmedDown > 0 ? t.statusDownLabel : undefined}
      />

      {/* 5.3 Open Incidents */}
      <MetricCard
        title={t.openIncidentsKpi}
        value={<span className="tabular-nums font-mono">{openIncidents}</span>}
        icon={<AlertTriangle className="w-5 h-5 text-amber-600" />}
        subtitle={
          openIncidents > 0
            ? `${unacknowledgedIncidents ?? 0} ${t.unacknowledgedSub} · ${acknowledgedIncidents ?? 0} ${t.acknowledgedSub}`
            : t.openIncidentsKpiSub
        }
        href={openIncidents > 0 ? "/incidents?status=open" : "/incidents"}
        trendColor={openIncidents > 0 ? "rose" : "emerald"}
        actionText={openIncidents > 0 ? t.viewAllIncidents : undefined}
      />

      {/* 5.4 Average Response Latency */}
      <MetricCard
        title={t.avgLatencyKpi}
        value={
          avgLatencyMs !== null ? (
            <span className="tabular-nums font-mono">
              {avgLatencyMs}{" "}
              <span className="text-sm font-semibold text-slate-500">ms</span>
            </span>
          ) : (
            <span className="tabular-nums font-mono text-slate-400">—</span>
          )
        }
        icon={<Activity className="w-5 h-5 text-indigo-600" />}
        subtitle={
          avgLatencyMs !== null
            ? `${successfulMeasurementsCount} ${t.avgLatencyKpiSub}`
            : t.latencyUnavailable
        }
        href="/monitors"
        trendColor="indigo"
      />
    </div>
  );
}

