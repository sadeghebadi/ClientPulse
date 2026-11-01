"use client";

import React from "react";
import Link from "next/link";
import { ArrowRight, ArrowLeft } from "lucide-react";
import { useI18n } from "@/lib/i18n";

interface MetricCardProps {
  title: string;
  value: number | string;
  icon: React.ReactNode;
  subtitle?: string;
  href?: string;
  trendColor?: "blue" | "emerald" | "rose" | "indigo" | "amber" | "slate";
  actionText?: string;
}

export function MetricCard({
  title,
  value,
  icon,
  subtitle,
  href,
  trendColor = "blue",
  actionText,
}: MetricCardProps) {
  const { isRTL } = useI18n();

  const colorClasses = {
    blue: "text-blue-600 hover:border-blue-300",
    emerald: "text-emerald-600 hover:border-emerald-300",
    rose: "text-rose-600 hover:border-rose-300",
    indigo: "text-indigo-600 hover:border-indigo-300",
    amber: "text-amber-600 hover:border-amber-300",
    slate: "text-slate-600 hover:border-slate-300",
  }[trendColor];

  const valueColorClasses = {
    blue: "text-slate-900",
    emerald: "text-emerald-600",
    rose: "text-rose-600",
    indigo: "text-slate-900",
    amber: "text-amber-600",
    slate: "text-slate-700",
  }[trendColor];

  const content = (
    <div className={`bg-white p-5 sm:p-6 rounded-2xl border border-slate-200 shadow-sm transition-all group ${href ? colorClasses : ""}`}>
      <div className="flex items-center justify-between text-slate-500 mb-3">
        <span className="text-[11px] font-bold uppercase tracking-wider text-slate-500">{title}</span>
        <div className={`p-2 rounded-xl bg-slate-50 group-hover:scale-105 transition-transform ${colorClasses.split(" ")[0]}`}>
          {icon}
        </div>
      </div>
      <div className={`text-2xl sm:text-3xl font-extrabold tracking-tight ${valueColorClasses}`}>
        {value}
      </div>
      {(subtitle || actionText) && (
        <div className="mt-3 flex items-center justify-between text-xs text-slate-500">
          <span>{subtitle}</span>
          {actionText && href && (
            <span className={`font-semibold flex items-center gap-1 ${colorClasses.split(" ")[0]}`}>
              <span>{actionText}</span>
              {isRTL ? <ArrowLeft className="w-3 h-3" /> : <ArrowRight className="w-3 h-3" />}
            </span>
          )}
        </div>
      )}
    </div>
  );

  if (href) {
    return (
      <Link href={href} className="block focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 rounded-2xl">
        {content}
      </Link>
    );
  }

  return content;
}

