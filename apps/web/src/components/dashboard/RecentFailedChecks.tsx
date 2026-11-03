import React from "react";
import Link from "next/link";
import { AlertTriangle, CheckCircle2, ArrowRight, ArrowLeft } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { Badge } from "@/components/ui/Badge";

export interface FailedCheckItem {
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

interface RecentFailedChecksProps {
  failedChecks: FailedCheckItem[];
  timezone?: string;
}

export function RecentFailedChecks({
  failedChecks,
  timezone = "UTC",
}: RecentFailedChecksProps) {
  const { t, isRTL, locale } = useI18n();

  return (
    <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
      <div className="p-5 sm:p-6 border-b border-slate-100 flex items-center justify-between">
        <div className="flex items-center gap-2.5">
          <div className="p-2 rounded-xl bg-rose-50 text-rose-600">
            <AlertTriangle className="w-5 h-5" />
          </div>
          <div>
            <h2 className="text-sm sm:text-base font-bold text-slate-900 tracking-tight flex items-center gap-2">
              <span>{t.recentFailedChecksHeading}</span>
              {failedChecks.length > 0 && (
                <span className="px-2 py-0.5 text-xs font-mono font-bold bg-rose-100 text-rose-800 rounded-full">
                  {failedChecks.length}
                </span>
              )}
            </h2>
            <p className="text-xs text-slate-500 mt-0.5">
              {t.recentFailedChecksSub}
            </p>
          </div>
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

      {failedChecks.length === 0 ? (
        <div className="p-8 text-center bg-white">
          <div className="w-10 h-10 rounded-full bg-emerald-50 text-emerald-600 flex items-center justify-center mx-auto mb-2">
            <CheckCircle2 className="w-6 h-6" />
          </div>
          <p className="text-xs font-semibold text-slate-800">
            {t.noFailedChecksTitle}
          </p>
          <p className="text-xs text-slate-500 mt-0.5">
            {t.noFailedChecksDesc}
          </p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table
            className="w-full text-left text-sm"
            dir={isRTL ? "rtl" : "ltr"}
          >
            <thead className="bg-slate-50 text-slate-600 text-xs uppercase font-semibold border-b border-slate-200">
              <tr>
                <th className="px-5 py-3.5">{t.status}</th>
                <th className="px-5 py-3.5">{t.latestCheck}</th>
                <th className="px-5 py-3.5">{t.latency}</th>
                <th className="px-5 py-3.5">{t.statusCode}</th>
                <th className="px-5 py-3.5">{t.activityCategory}</th>
                <th className="px-5 py-3.5 text-right">{t.actions}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 text-xs">
              {failedChecks.map((chk) => (
                <tr
                  key={chk.id}
                  className="hover:bg-rose-50/30 transition-colors"
                >
                  <td className="px-5 py-3.5">
                    <Badge variant={chk.result} />
                  </td>
                  <td className="px-5 py-3.5 text-slate-600 tabular-nums">
                    {new Date(chk.completed_at).toLocaleTimeString(
                      locale === "fa" ? "fa-IR" : "en-US",
                      {
                        timeZone: timezone,
                        hour: "2-digit",
                        minute: "2-digit",
                        second: "2-digit",
                      }
                    )}
                  </td>
                  <td className="px-5 py-3.5 font-mono font-medium text-slate-800 tabular-nums">
                    {chk.latency_ms} ms
                  </td>
                  <td className="px-5 py-3.5 font-mono font-bold text-rose-600 tabular-nums">
                    {chk.http_status_code || "—"}
                  </td>
                  <td className="px-5 py-3.5 text-slate-700 max-w-xs truncate">
                    <span className="font-mono text-[11px] text-rose-700">
                      {chk.error_category || chk.error_message || "Target unresponsive"}
                    </span>
                  </td>
                  <td className="px-5 py-3.5 text-right">
                    <Link
                      href={`/monitors/${chk.monitor_id}`}
                      className="font-semibold text-blue-600 hover:text-blue-800 transition-colors"
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
  );
}

