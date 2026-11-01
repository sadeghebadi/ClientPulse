"use client";

import React from "react";
import {
  CheckCircle2,
  XCircle,
  AlertCircle,
  Clock,
  PauseCircle,
  HelpCircle,
  ShieldAlert,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";

export type BadgeVariant =
  | "up"
  | "down"
  | "suspected_down"
  | "pending"
  | "paused"
  | "unknown"
  | "active"
  | "archived"
  | "production"
  | "staging"
  | "development"
  | "website"
  | "api"
  | "owner"
  | "admin"
  | "member"
  | "viewer"
  | "success"
  | "failure"
  | "timeout"
  | "blocked"
  | "error";

interface BadgeProps {
  variant: BadgeVariant | string;
  label?: string;
  showIcon?: boolean;
  className?: string;
}

export function Badge({ variant, label, showIcon = true, className = "" }: BadgeProps) {
  const { t } = useI18n();

  switch (variant) {
    case "up":
      return (
        <span
          title={t.statusUpDesc}
          className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200 ${className}`}
        >
          {showIcon && <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600" />}
          <span>{label || t.statusUp}</span>
        </span>
      );

    case "suspected_down":
      return (
        <span
          title={t.statusSuspectedDownDesc}
          className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-amber-50 text-amber-800 border border-amber-300 ${className}`}
        >
          {showIcon && <AlertCircle className="w-3.5 h-3.5 text-amber-600" />}
          <span>{label || t.statusSuspectedDown}</span>
        </span>
      );

    case "down":
      return (
        <span
          title={t.statusDownDesc}
          className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-rose-50 text-rose-700 border border-rose-200 ${className}`}
        >
          {showIcon && <XCircle className="w-3.5 h-3.5 text-rose-600" />}
          <span>{label || t.statusDown}</span>
        </span>
      );

    case "paused":
      return (
        <span
          title={t.statusPausedDesc}
          className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-slate-100 text-slate-700 border border-slate-200 ${className}`}
        >
          {showIcon && <PauseCircle className="w-3.5 h-3.5 text-slate-500" />}
          <span>{label || t.statusPaused}</span>
        </span>
      );

    case "pending":
      return (
        <span
          title={t.statusPendingDesc}
          className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-blue-50 text-blue-700 border border-blue-200 ${className}`}
        >
          {showIcon && <Clock className="w-3.5 h-3.5 text-blue-600" />}
          <span>{label || t.statusPending}</span>
        </span>
      );

    case "unknown":
      return (
        <span
          title={t.statusUnknownDesc}
          className={`inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-slate-50 text-slate-600 border border-slate-200 ${className}`}
        >
          {showIcon && <HelpCircle className="w-3.5 h-3.5 text-slate-400" />}
          <span>{label || t.statusUnknown}</span>
        </span>
      );

    case "active":
      return (
        <span
          className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200 ${className}`}
        >
          {label || t.active}
        </span>
      );

    case "archived":
      return (
        <span
          className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-slate-100 text-slate-600 border border-slate-200 ${className}`}
        >
          {label || t.archived}
        </span>
      );

    case "production":
      return (
        <span
          className={`inline-flex items-center px-2 py-0.5 rounded text-[11px] font-bold uppercase tracking-wider bg-indigo-50 text-indigo-700 border border-indigo-200 ${className}`}
        >
          {label || t.production}
        </span>
      );

    case "staging":
      return (
        <span
          className={`inline-flex items-center px-2 py-0.5 rounded text-[11px] font-bold uppercase tracking-wider bg-purple-50 text-purple-700 border border-purple-200 ${className}`}
        >
          {label || t.staging}
        </span>
      );

    case "development":
      return (
        <span
          className={`inline-flex items-center px-2 py-0.5 rounded text-[11px] font-bold uppercase tracking-wider bg-slate-100 text-slate-700 border border-slate-200 ${className}`}
        >
          {label || t.development}
        </span>
      );

    case "website":
      return (
        <span
          className={`inline-flex items-center px-1.5 py-0.5 rounded text-[10px] uppercase font-bold bg-blue-50 text-blue-700 border border-blue-200 ${className}`}
        >
          Website
        </span>
      );

    case "api":
      return (
        <span
          className={`inline-flex items-center px-1.5 py-0.5 rounded text-[10px] uppercase font-bold bg-amber-50 text-amber-700 border border-amber-200 ${className}`}
        >
          API
        </span>
      );

    case "success":
      return (
        <span className={`inline-flex items-center gap-1 text-emerald-600 font-semibold text-xs ${className}`}>
          {showIcon && <CheckCircle2 className="w-3.5 h-3.5" />}
          <span>{label || t.success}</span>
        </span>
      );

    case "failure":
      return (
        <span className={`inline-flex items-center gap-1 text-rose-600 font-semibold text-xs ${className}`}>
          {showIcon && <XCircle className="w-3.5 h-3.5" />}
          <span>{label || t.failure}</span>
        </span>
      );

    case "timeout":
      return (
        <span className={`inline-flex items-center gap-1 text-amber-600 font-semibold text-xs ${className}`}>
          {showIcon && <Clock className="w-3.5 h-3.5" />}
          <span>{label || t.timeoutResult}</span>
        </span>
      );

    case "blocked":
      return (
        <span className={`inline-flex items-center gap-1 text-purple-600 font-semibold text-xs ${className}`}>
          {showIcon && <ShieldAlert className="w-3.5 h-3.5" />}
          <span>{label || t.blockedResult}</span>
        </span>
      );

    case "owner":
    case "admin":
      return (
        <span
          className={`text-[10px] uppercase font-bold px-2 py-0.5 rounded bg-blue-100 text-blue-800 ${className}`}
        >
          {label || variant}
        </span>
      );

    case "member":
    case "viewer":
      return (
        <span
          className={`text-[10px] uppercase font-bold px-2 py-0.5 rounded bg-slate-100 text-slate-700 ${className}`}
        >
          {label || variant}
        </span>
      );

    default:
      return (
        <span
          className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-700 ${className}`}
        >
          {label || variant}
        </span>
      );
  }
}

