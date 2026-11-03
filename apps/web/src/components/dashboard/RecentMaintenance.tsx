import React from "react";
import Link from "next/link";
import { Wrench, ArrowRight, ArrowLeft, Plus, Calendar } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/Button";

export interface MaintenanceActivityItem {
  id: string;
  client_id: string;
  client_name?: string;
  asset_id?: string;
  asset_name?: string;
  title: string;
  category: string;
  performed_at: string;
  duration_minutes?: number;
}

interface RecentMaintenanceProps {
  activities: MaintenanceActivityItem[];
}

export function RecentMaintenance({ activities }: RecentMaintenanceProps) {
  const { t, isRTL, locale } = useI18n();

  return (
    <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-5 sm:p-6">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-2 rounded-xl bg-amber-50 text-amber-600">
            <Wrench className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-bold text-slate-900 tracking-tight">
              {t.recentMaintenanceHeading}
            </h3>
            <p className="text-[11px] text-slate-500">
              {t.recentMaintenanceSub}
            </p>
          </div>
        </div>
        <Link
          href="/maintenance"
          className="text-xs font-semibold text-blue-600 hover:text-blue-800 flex items-center gap-1 transition-colors"
        >
          <span>{t.manageMaintenance}</span>
          {isRTL ? (
            <ArrowLeft className="w-3.5 h-3.5" />
          ) : (
            <ArrowRight className="w-3.5 h-3.5" />
          )}
        </Link>
      </div>

      {activities.length === 0 ? (
        <div className="p-6 text-center bg-slate-50 rounded-xl border border-dashed border-slate-200">
          <p className="text-xs font-semibold text-slate-700 mb-1">
            {t.noRecentMaintenance}
          </p>
          <p className="text-[11px] text-slate-500 mb-3">
            {t.noRecentMaintenanceDesc}
          </p>
          <Link href="/maintenance">
            <Button size="sm" variant="outline" icon={<Plus className="w-3.5 h-3.5" />}>
              {t.logMaintenance}
            </Button>
          </Link>
        </div>
      ) : (
        <div className="space-y-2.5">
          {activities.slice(0, 5).map((act) => (
            <div
              key={act.id}
              className="p-3 rounded-xl border border-slate-100 hover:border-slate-200 hover:bg-slate-50/60 transition-colors"
            >
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <div className="text-xs font-bold text-slate-900 truncate">
                    {act.title}
                  </div>
                  <div className="text-[11px] text-slate-500 mt-0.5 flex items-center gap-1.5 flex-wrap">
                    <span className="font-medium text-slate-700">
                      {act.client_name || act.client_id}
                    </span>
                    {act.asset_name && (
                      <>
                        <span className="text-slate-300">·</span>
                        <span>{act.asset_name}</span>
                      </>
                    )}
                  </div>
                </div>
                <span className="text-[10px] font-semibold px-2 py-0.5 rounded-md bg-slate-100 text-slate-700 shrink-0">
                  {act.category}
                </span>
              </div>
              <div className="mt-2 pt-2 border-t border-slate-100 flex items-center justify-between text-[11px] text-slate-400">
                <span className="flex items-center gap-1">
                  <Calendar className="w-3 h-3 text-slate-400" />
                  <span>
                    {new Date(act.performed_at).toLocaleDateString(
                      locale === "fa" ? "fa-IR" : "en-US",
                      { month: "short", day: "numeric" }
                    )}
                  </span>
                </span>
                {act.duration_minutes !== undefined && (
                  <span className="font-mono text-slate-600 font-medium">
                    {act.duration_minutes} min
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

