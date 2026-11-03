import React from "react";
import Link from "next/link";
import { ArrowRight, ArrowLeft, Activity, Plus } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/Button";

interface MonitorDistributionProps {
  up: number;
  suspectedDown: number;
  down: number;
  paused: number;
  pending: number;
  unknown: number;
}

export function MonitorDistribution({
  up,
  suspectedDown,
  down,
  paused,
  pending,
  unknown,
}: MonitorDistributionProps) {
  const { t, isRTL } = useI18n();

  const total = up + suspectedDown + down + paused + pending + unknown;

  // Categories config with explicit semantic colors and backend status values
  const categories = [
    {
      key: "up",
      label: t.statusOperational,
      count: up,
      barColor: "bg-emerald-500",
      textColor: "text-emerald-700",
      bgColor: "bg-emerald-50/60",
      borderColor: "border-emerald-200",
      href: "/monitors?status=up",
    },
    {
      key: "suspected_down",
      label: t.statusSuspectedDownLabel,
      count: suspectedDown,
      barColor: "bg-amber-500",
      textColor: "text-amber-700",
      bgColor: "bg-amber-50/60",
      borderColor: "border-amber-200",
      href: "/monitors?status=suspected_down",
    },
    {
      key: "down",
      label: t.statusDownLabel,
      count: down,
      barColor: "bg-rose-500",
      textColor: "text-rose-700",
      bgColor: "bg-rose-50/60",
      borderColor: "border-rose-200",
      href: "/monitors?status=down",
    },
    {
      key: "paused",
      label: t.statusPausedLabel,
      count: paused,
      barColor: "bg-slate-400",
      textColor: "text-slate-700",
      bgColor: "bg-slate-50",
      borderColor: "border-slate-200",
      href: "/monitors?status=paused",
    },
    {
      key: "pending",
      label: t.statusPendingLabel,
      count: pending,
      barColor: "bg-blue-400",
      textColor: "text-blue-700",
      bgColor: "bg-blue-50/60",
      borderColor: "border-blue-200",
      href: "/monitors?status=pending",
    },
    {
      key: "unknown",
      label: t.statusUnknownLabel,
      count: unknown,
      barColor: "bg-zinc-400",
      textColor: "text-zinc-700",
      bgColor: "bg-zinc-50",
      borderColor: "border-zinc-200",
      href: "/monitors?status=unknown",
    },
  ];

  return (
    <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-5 sm:p-6">
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2 mb-4">
        <div>
          <h2 className="text-sm sm:text-base font-bold text-slate-900 tracking-tight flex items-center gap-2">
            <Activity className="w-4 h-4 text-blue-600" />
            <span>{t.monitorDistribution}</span>
          </h2>
          <p className="text-xs text-slate-500 mt-0.5">
            {t.monitorDistributionDesc}
          </p>
        </div>
        <Link
          href="/monitors"
          className="text-xs font-semibold text-blue-600 hover:text-blue-800 flex items-center gap-1 transition-colors"
        >
          <span>{t.viewAllMonitors}</span>
          {isRTL ? (
            <ArrowLeft className="w-3.5 h-3.5" />
          ) : (
            <ArrowRight className="w-3.5 h-3.5" />
          )}
        </Link>
      </div>

      {total === 0 ? (
        <div className="p-8 text-center bg-slate-50 rounded-xl border border-dashed border-slate-200">
          <p className="text-xs font-semibold text-slate-700 mb-1">
            {t.noMonitorsYet}
          </p>
          <p className="text-xs text-slate-500 mb-4">
            {t.noMonitorsConfiguredDesc}
          </p>
          <Link href="/monitors/new">
            <Button size="sm" variant="primary" icon={<Plus className="w-4 h-4" />}>
              {t.addFirstMonitor}
            </Button>
          </Link>
        </div>
      ) : (
        <div className="space-y-4">
          {/* Segmented Horizontal Distribution Bar */}
          <div
            role="progressbar"
            aria-valuenow={up}
            aria-valuemin={0}
            aria-valuemax={total}
            aria-label={t.monitorDistribution}
            className="h-3 w-full bg-slate-100 rounded-full overflow-hidden flex"
          >
            {categories.map(
              (cat) =>
                cat.count > 0 && (
                  <div
                    key={cat.key}
                    style={{ width: `${(cat.count / total) * 100}%` }}
                    className={`${cat.barColor} transition-all duration-300 first:rounded-l-full last:rounded-r-full`}
                    title={`${cat.label}: ${cat.count} (${Math.round(
                      (cat.count / total) * 100
                    )}%)`}
                  />
                )
            )}
          </div>

          {/* Status Breakdown Legend & Counts */}
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-2.5 pt-1">
            {categories.map((cat) => (
              <Link
                key={cat.key}
                href={cat.href}
                className={`p-3 rounded-xl border ${cat.borderColor} ${cat.bgColor} hover:brightness-95 transition-all block focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500`}
              >
                <div className="flex items-center gap-1.5 mb-1">
                  <span
                    className={`w-2 h-2 rounded-full ${cat.barColor} shrink-0`}
                  />
                  <span className={`text-[11px] font-semibold truncate ${cat.textColor}`}>
                    {cat.label}
                  </span>
                </div>
                <div
                  className={`text-lg sm:text-xl font-mono font-bold tabular-nums ${cat.textColor}`}
                >
                  {cat.count}
                </div>
              </Link>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

