import React from "react";
import Link from "next/link";
import {
  AlertOctagon,
  Clock,
  Building,
  Globe,
  ArrowRight,
  ArrowLeft,
  CheckCircle2,
  Check,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";

export interface ActiveIncidentItem {
  id: string;
  target_type: "monitor" | "heartbeat";
  target_name: string;
  client_id: string;
  client_name: string;
  asset_id?: string;
  asset_name?: string;
  monitor_id?: string;
  heartbeat_monitor_id?: string;
  first_failure_at: string;
  confirmed_at: string;
  resolved_at?: string;
  status: "open" | "resolved";
  failure_category: string;
  acknowledged_at?: string;
  acknowledged_by?: string;
  acknowledged_by_name?: string;
  duration_seconds?: number;
  summary?: string;
}

interface ActiveIncidentsListProps {
  incidents: ActiveIncidentItem[];
  loading?: boolean;
  onAcknowledge?: (id: string) => Promise<void>;
  acknowledgingId?: string | null;
  canAcknowledge?: boolean;
}

function formatDuration(seconds: number | undefined, isRTL: boolean): string {
  if (!seconds || seconds <= 0) return isRTL ? "چند لحظه پیش" : "Just now";
  const mins = Math.floor(seconds / 60);
  const hours = Math.floor(mins / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) return `${days}d ${hours % 24}h`;
  if (hours > 0) return `${hours}h ${mins % 60}m`;
  return `${mins}m`;
}

export function ActiveIncidentsList({
  incidents,
  loading = false,
  onAcknowledge,
  acknowledgingId = null,
  canAcknowledge = false,
}: ActiveIncidentsListProps) {
  const { t, isRTL, locale } = useI18n();

  // Sort: Unacknowledged first, then confirmed outages, then suspected/other
  const sortedIncidents = [...incidents].sort((a, b) => {
    // 1. Unacknowledged first
    const aUnack = !a.acknowledged_at;
    const bUnack = !b.acknowledged_at;
    if (aUnack !== bUnack) return aUnack ? -1 : 1;

    // 2. Confirmed outages before other failures
    const aConfirmed = a.confirmed_at ? 1 : 0;
    const bConfirmed = b.confirmed_at ? 1 : 0;
    if (aConfirmed !== bConfirmed) return bConfirmed - aConfirmed;

    // 3. Most recent first
    return (
      new Date(b.first_failure_at).getTime() -
      new Date(a.first_failure_at).getTime()
    );
  });

  return (
    <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
      <div className="p-5 sm:p-6 border-b border-slate-100 flex items-center justify-between">
        <div className="flex items-center gap-2.5">
          <div className="p-2 rounded-xl bg-rose-50 text-rose-600">
            <AlertOctagon className="w-5 h-5" />
          </div>
          <div>
            <h2 className="text-sm sm:text-base font-bold text-slate-900 tracking-tight flex items-center gap-2">
              <span>{t.activeIncidentsHeading}</span>
              {sortedIncidents.length > 0 && (
                <span className="px-2 py-0.5 text-xs font-mono font-bold bg-rose-100 text-rose-800 rounded-full">
                  {sortedIncidents.length}
                </span>
              )}
            </h2>
            <p className="text-xs text-slate-500 mt-0.5">
              {t.activeIncidentsSub}
            </p>
          </div>
        </div>
        <Link
          href="/incidents?status=open"
          className="text-xs font-semibold text-blue-600 hover:text-blue-800 flex items-center gap-1 transition-colors"
        >
          <span>{t.viewAllIncidents}</span>
          {isRTL ? (
            <ArrowLeft className="w-3.5 h-3.5" />
          ) : (
            <ArrowRight className="w-3.5 h-3.5" />
          )}
        </Link>
      </div>

      {sortedIncidents.length === 0 ? (
        <div className="p-8 text-center bg-white">
          <div className="w-10 h-10 rounded-full bg-emerald-50 text-emerald-600 flex items-center justify-center mx-auto mb-2">
            <CheckCircle2 className="w-6 h-6" />
          </div>
          <p className="text-xs font-semibold text-slate-800">
            {t.noActiveIncidents}
          </p>
          <p className="text-xs text-slate-500 mt-0.5 max-w-sm mx-auto">
            {t.noActiveIncidentsDesc}
          </p>
        </div>
      ) : (
        <div className="divide-y divide-slate-100">
          {sortedIncidents.map((inc) => {
            const isUnack = !inc.acknowledged_at;
            const targetHref =
              inc.target_type === "heartbeat"
                ? `/heartbeats/${inc.heartbeat_monitor_id || inc.target_name}`
                : `/monitors/${inc.monitor_id || ""}`;

            return (
              <div
                key={inc.id}
                className={`p-4 sm:p-5 flex flex-col md:flex-row items-start md:items-center justify-between gap-4 transition-colors ${
                  isUnack ? "bg-rose-50/40 hover:bg-rose-50/60" : "hover:bg-slate-50/80"
                }`}
              >
                <div className="space-y-1.5 flex-1 min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <span
                      className={`text-xs font-bold ${
                        isUnack ? "text-rose-950 font-extrabold" : "text-slate-900"
                      }`}
                    >
                      {inc.summary || inc.target_name}
                    </span>
                    <Badge variant={inc.failure_category || "error"} />
                    {isUnack ? (
                      <span className="text-[11px] font-bold px-2 py-0.5 rounded-full bg-rose-100 text-rose-800 border border-rose-300">
                        {t.unacknowledged}
                      </span>
                    ) : (
                      <span className="text-[11px] font-semibold px-2 py-0.5 rounded-full bg-slate-100 text-slate-700">
                        {t.statusAcknowledged}
                      </span>
                    )}
                  </div>

                  <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-slate-600">
                    <span className="flex items-center gap-1 font-medium">
                      <Building className="w-3.5 h-3.5 text-slate-400" />
                      <span>{inc.client_name}</span>
                    </span>

                    {inc.asset_name && (
                      <span className="flex items-center gap-1 text-slate-500">
                        <Globe className="w-3.5 h-3.5 text-slate-400" />
                        <span>{inc.asset_name}</span>
                      </span>
                    )}

                    <span className="flex items-center gap-1 text-slate-500">
                      <Clock className="w-3.5 h-3.5 text-slate-400" />
                      <span>
                        {t.startedAt}:{" "}
                        {new Date(inc.first_failure_at).toLocaleTimeString(
                          locale === "fa" ? "fa-IR" : "en-US",
                          { hour: "2-digit", minute: "2-digit" }
                        )}
                      </span>
                      {inc.duration_seconds !== undefined && (
                        <span className="font-mono text-slate-700 font-semibold">
                          ({formatDuration(inc.duration_seconds, isRTL)})
                        </span>
                      )}
                    </span>
                  </div>
                </div>

                <div className="flex items-center gap-2 shrink-0 self-end md:self-center">
                  {isUnack && canAcknowledge && onAcknowledge && (
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => onAcknowledge(inc.id)}
                      disabled={acknowledgingId === inc.id}
                      icon={<Check className="w-3.5 h-3.5" />}
                    >
                      {acknowledgingId === inc.id ? "..." : t.acknowledge}
                    </Button>
                  )}
                  <Link href={`/incidents/${inc.id}`}>
                    <Button size="sm" variant="secondary">
                      <span>{t.viewDetails}</span>
                      {isRTL ? (
                        <ArrowLeft className="w-3 h-3" />
                      ) : (
                        <ArrowRight className="w-3 h-3" />
                      )}
                    </Button>
                  </Link>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

