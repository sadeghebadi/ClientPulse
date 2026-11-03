import React from "react";
import Link from "next/link";
import { Radio, ArrowRight, ArrowLeft, CheckCircle2, AlertTriangle, Plus } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/Button";

export interface HeartbeatSummaryData {
  total: number;
  healthy: number;
  missed: number;
  pending: number;
  latestMissedName?: string;
  latestMissedAt?: string;
}

interface HeartbeatSummaryProps {
  summary: HeartbeatSummaryData;
}

export function HeartbeatSummary({ summary }: HeartbeatSummaryProps) {
  const { t, isRTL } = useI18n();

  return (
    <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-5 sm:p-6">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-2 rounded-xl bg-indigo-50 text-indigo-600">
            <Radio className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-bold text-slate-900 tracking-tight">
              {t.heartbeatSummaryHeading}
            </h3>
            <p className="text-[11px] text-slate-500">
              {t.heartbeatSummarySub}
            </p>
          </div>
        </div>
        <Link
          href="/heartbeats"
          className="text-xs font-semibold text-blue-600 hover:text-blue-800 flex items-center gap-1 transition-colors"
        >
          <span>{t.manageHeartbeats}</span>
          {isRTL ? (
            <ArrowLeft className="w-3.5 h-3.5" />
          ) : (
            <ArrowRight className="w-3.5 h-3.5" />
          )}
        </Link>
      </div>

      {summary.total === 0 ? (
        <div className="p-6 text-center bg-slate-50 rounded-xl border border-dashed border-slate-200">
          <p className="text-xs font-semibold text-slate-700 mb-1">
            {t.noHeartbeatsConfigured}
          </p>
          <p className="text-[11px] text-slate-500 mb-3">
            {t.noHeartbeatsDesc}
          </p>
          <Link href="/heartbeats/new">
            <Button size="sm" variant="outline" icon={<Plus className="w-3.5 h-3.5" />}>
              {t.configureHeartbeat}
            </Button>
          </Link>
        </div>
      ) : (
        <div className="space-y-3">
          <div className="grid grid-cols-3 gap-2">
            <Link
              href="/heartbeats?status=healthy"
              className="p-3 rounded-xl bg-emerald-50/60 border border-emerald-200 hover:bg-emerald-50 transition-colors text-center block"
            >
              <div className="text-[11px] font-semibold text-emerald-800">
                {t.healthyHeartbeats}
              </div>
              <div className="text-lg font-mono font-bold text-emerald-700 mt-0.5 tabular-nums">
                {summary.healthy}
              </div>
            </Link>

            <Link
              href="/heartbeats?status=missed"
              className="p-3 rounded-xl bg-rose-50/60 border border-rose-200 hover:bg-rose-50 transition-colors text-center block"
            >
              <div className="text-[11px] font-semibold text-rose-800">
                {t.missedHeartbeats}
              </div>
              <div className="text-lg font-mono font-bold text-rose-700 mt-0.5 tabular-nums">
                {summary.missed}
              </div>
            </Link>

            <Link
              href="/heartbeats?status=pending"
              className="p-3 rounded-xl bg-slate-50 border border-slate-200 hover:bg-slate-100 transition-colors text-center block"
            >
              <div className="text-[11px] font-semibold text-slate-700">
                {t.pendingHeartbeats}
              </div>
              <div className="text-lg font-mono font-bold text-slate-800 mt-0.5 tabular-nums">
                {summary.pending}
              </div>
            </Link>
          </div>

          {summary.latestMissedName && (
            <div className="p-2.5 rounded-xl bg-rose-50 border border-rose-200 text-rose-900 text-xs flex items-center justify-between">
              <span className="flex items-center gap-1.5 font-medium">
                <AlertTriangle className="w-3.5 h-3.5 text-rose-600 shrink-0" />
                <span className="truncate max-w-[180px]">
                  {summary.latestMissedName}
                </span>
              </span>
              <span className="text-[11px] text-rose-700 font-semibold">
                {t.statusMissed}
              </span>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

