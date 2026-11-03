import React from "react";
import Link from "next/link";
import { FileText, ArrowRight, ArrowLeft, Plus, Download, Sparkles } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/Button";

export interface CareReportSummaryItem {
  id: string;
  client_id: string;
  client_name?: string;
  title: string;
  report_type: string;
  period_start: string;
  period_end: string;
  generated_at: string;
  pdf_storage_key?: string | null;
}

interface CareReportsOverviewProps {
  reports: CareReportSummaryItem[];
  canGenerate?: boolean;
}

export function CareReportsOverview({
  reports,
  canGenerate = false,
}: CareReportsOverviewProps) {
  const { t, isRTL, locale } = useI18n();

  return (
    <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-5 sm:p-6">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-2 rounded-xl bg-blue-50 text-blue-600">
            <FileText className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-bold text-slate-900 tracking-tight">
              {t.careReportsHeading}
            </h3>
            <p className="text-[11px] text-slate-500">
              {t.careReportsSub}
            </p>
          </div>
        </div>
        <Link
          href="/reports"
          className="text-xs font-semibold text-blue-600 hover:text-blue-800 flex items-center gap-1 transition-colors"
        >
          <span>{t.manageReports}</span>
          {isRTL ? (
            <ArrowLeft className="w-3.5 h-3.5" />
          ) : (
            <ArrowRight className="w-3.5 h-3.5" />
          )}
        </Link>
      </div>

      {reports.length === 0 ? (
        <div className="p-6 text-center bg-slate-50 rounded-xl border border-dashed border-slate-200">
          <p className="text-xs font-semibold text-slate-700 mb-1">
            {t.noReportsYet}
          </p>
          <p className="text-[11px] text-slate-500 mb-3">
            {t.noReportsDesc}
          </p>
          <Link href="/reports">
            <Button size="sm" variant="outline" icon={<Plus className="w-3.5 h-3.5" />}>
              {t.createFirstReport}
            </Button>
          </Link>
        </div>
      ) : (
        <div className="space-y-2.5">
          {reports.slice(0, 5).map((rep) => (
            <Link
              key={rep.id}
              href={`/reports/${rep.id}`}
              className="p-3 rounded-xl border border-slate-100 hover:border-blue-200 hover:bg-slate-50/80 transition-all flex items-center justify-between group block"
            >
              <div className="min-w-0 pr-2">
                <div className="text-xs font-bold text-slate-900 group-hover:text-blue-600 transition-colors truncate">
                  {rep.title || "Client Care Report"}
                </div>
                <div className="text-[11px] text-slate-500 mt-0.5 flex items-center gap-1.5 flex-wrap">
                  <span className="font-medium text-slate-700">
                    {rep.client_name || rep.client_id}
                  </span>
                  <span className="text-slate-300">·</span>
                  <span>
                    {new Date(rep.generated_at).toLocaleDateString(
                      locale === "fa" ? "fa-IR" : "en-US",
                      { month: "short", day: "numeric" }
                    )}
                  </span>
                </div>
              </div>
              <div className="shrink-0 flex items-center gap-1.5">
                {rep.pdf_storage_key ? (
                  <span className="text-[11px] font-semibold text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded-full border border-emerald-200">
                    PDF
                  </span>
                ) : (
                  <span className="text-[11px] font-medium text-slate-500 bg-slate-100 px-2 py-0.5 rounded-full">
                    {rep.report_type || "monthly"}
                  </span>
                )}
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

