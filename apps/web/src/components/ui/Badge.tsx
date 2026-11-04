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
  UserCheck,
} from "lucide-react";
import { useI18n } from "@/lib/i18n";

export type BadgeVariant =
  | "up"
  | "down"
  | "suspected_down"
  | "pending"
  | "paused"
  | "unknown"
  | "healthy"
  | "missed"
  | "late"
  | "revoked"
  | "open"
  | "resolved"
  | "legacy_resolution"
  | "source_monitor_deleted"
  | "monitor_deleted"
  | "manual"
  | "automatic_recovery"
  | "acknowledged"
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
  | "warning"
  | "danger"
  | "default"
  | "secondary"
  | "outline"
  | "error";

export interface BadgeProps {
  variant: BadgeVariant | string;
  label?: string;
  children?: React.ReactNode;
  showIcon?: boolean;
  className?: string;
  icon?: React.ReactNode;
  size?: "sm" | "md" | "lg";
}

type BadgeConfig = {
  bg: string;
  text: string;
  border?: string;
  icon?: React.ComponentType<{ className?: string }>;
  iconClass?: string;
  defaultLabelKey?: string;
  defaultLabelFallback?: string;
  titleKey?: string;
  isPill?: boolean;
  isUppercase?: boolean;
  isMinimalText?: boolean;
};

const BADGE_CONFIG_MAP: Record<string, BadgeConfig> = {
  // Operational statuses
  up: {
    bg: "bg-emerald-50",
    text: "text-emerald-700",
    border: "border-emerald-200",
    icon: CheckCircle2,
    iconClass: "text-emerald-600",
    defaultLabelKey: "statusUp",
    titleKey: "statusUpDesc",
    isPill: true,
  },
  healthy: {
    bg: "bg-emerald-50",
    text: "text-emerald-700",
    border: "border-emerald-200",
    icon: CheckCircle2,
    iconClass: "text-emerald-600",
    defaultLabelKey: "statusHealthy",
    isPill: true,
  },
  resolved: {
    bg: "bg-emerald-50",
    text: "text-emerald-700",
    border: "border-emerald-200",
    icon: CheckCircle2,
    iconClass: "text-emerald-600",
    defaultLabelKey: "statusResolved",
    isPill: true,
  },
  legacy_resolution: {
    bg: "bg-slate-100",
    text: "text-slate-700",
    border: "border-slate-300",
    icon: HelpCircle,
    iconClass: "text-slate-500",
    defaultLabelKey: "legacyResolutionReasonNotRecorded",
    defaultLabelFallback: "Legacy incident — resolution reason not recorded",
    isPill: true,
  },
  source_monitor_deleted: {
    bg: "bg-slate-100",
    text: "text-slate-700",
    border: "border-slate-300",
    icon: XCircle,
    iconClass: "text-slate-500",
    defaultLabelKey: "sourceMonitorDeleted",
    defaultLabelFallback: "Closed (Source Deleted)",
    isPill: true,
  },
  monitor_deleted: {
    bg: "bg-slate-100",
    text: "text-slate-700",
    border: "border-slate-300",
    icon: XCircle,
    iconClass: "text-slate-500",
    defaultLabelKey: "monitorDeletedBadge",
    defaultLabelFallback: "Monitor Deleted",
    isPill: true,
  },
  manual: {
    bg: "bg-blue-50",
    text: "text-blue-700",
    border: "border-blue-200",
    icon: UserCheck,
    iconClass: "text-blue-600",
    defaultLabelKey: "manualResolution",
    defaultLabelFallback: "Manual Resolution",
    isPill: true,
  },
  automatic_recovery: {
    bg: "bg-emerald-50",
    text: "text-emerald-700",
    border: "border-emerald-200",
    icon: CheckCircle2,
    iconClass: "text-emerald-600",
    defaultLabelKey: "automaticRecovery",
    defaultLabelFallback: "Automatic Recovery",
    isPill: true,
  },
  active: {
    bg: "bg-emerald-50",
    text: "text-emerald-700",
    border: "border-emerald-200",
    defaultLabelKey: "active",
    isPill: true,
  },
  success: {
    bg: "transparent",
    text: "text-emerald-700",
    icon: CheckCircle2,
    iconClass: "text-emerald-600",
    defaultLabelKey: "success",
    isMinimalText: true,
  },

  // Degraded / warning statuses
  suspected_down: {
    bg: "bg-amber-50",
    text: "text-amber-800",
    border: "border-amber-300",
    icon: AlertCircle,
    iconClass: "text-amber-600",
    defaultLabelKey: "statusSuspectedDown",
    titleKey: "statusSuspectedDownDesc",
    isPill: true,
  },
  late: {
    bg: "bg-amber-50",
    text: "text-amber-800",
    border: "border-amber-300",
    icon: AlertCircle,
    iconClass: "text-amber-600",
    defaultLabelKey: "statusLate",
    isPill: true,
  },
  warning: {
    bg: "bg-amber-50",
    text: "text-amber-800",
    border: "border-amber-300",
    icon: AlertCircle,
    iconClass: "text-amber-600",
    defaultLabelFallback: "Warning",
    isPill: true,
  },
  timeout: {
    bg: "transparent",
    text: "text-amber-700",
    icon: Clock,
    iconClass: "text-amber-600",
    defaultLabelKey: "timeoutResult",
    isMinimalText: true,
  },
  api: {
    bg: "bg-amber-50",
    text: "text-amber-800",
    border: "border-amber-200",
    defaultLabelFallback: "API",
    isUppercase: true,
  },

  // Down / critical statuses
  down: {
    bg: "bg-rose-50",
    text: "text-rose-700",
    border: "border-rose-200",
    icon: XCircle,
    iconClass: "text-rose-600",
    defaultLabelKey: "statusDown",
    titleKey: "statusDownDesc",
    isPill: true,
  },
  missed: {
    bg: "bg-rose-50",
    text: "text-rose-700",
    border: "border-rose-200",
    icon: XCircle,
    iconClass: "text-rose-600",
    defaultLabelKey: "statusMissed",
    isPill: true,
  },
  open: {
    bg: "bg-rose-100",
    text: "text-rose-800",
    border: "border-rose-300",
    icon: XCircle,
    iconClass: "text-rose-600",
    defaultLabelKey: "statusOpen",
    isPill: true,
    isUppercase: true,
  },
  danger: {
    bg: "bg-rose-50",
    text: "text-rose-700",
    border: "border-rose-200",
    icon: XCircle,
    iconClass: "text-rose-600",
    defaultLabelFallback: "Danger",
    isPill: true,
  },
  failure: {
    bg: "transparent",
    text: "text-rose-700",
    icon: XCircle,
    iconClass: "text-rose-600",
    defaultLabelKey: "failure",
    isMinimalText: true,
  },
  error: {
    bg: "bg-rose-50",
    text: "text-rose-700",
    border: "border-rose-200",
    icon: XCircle,
    iconClass: "text-rose-600",
    defaultLabelFallback: "Error",
    isPill: true,
  },

  // Informational / Pending / Blue statuses
  pending: {
    bg: "bg-blue-50",
    text: "text-blue-700",
    border: "border-blue-200",
    icon: Clock,
    iconClass: "text-blue-600",
    defaultLabelKey: "statusPending",
    titleKey: "statusPendingDesc",
    isPill: true,
  },
  acknowledged: {
    bg: "bg-blue-50",
    text: "text-blue-700",
    border: "border-blue-200",
    icon: Clock,
    iconClass: "text-blue-600",
    defaultLabelKey: "statusAcknowledged",
    isPill: true,
  },
  website: {
    bg: "bg-blue-50",
    text: "text-blue-700",
    border: "border-blue-200",
    defaultLabelFallback: "Website",
    isUppercase: true,
  },
  owner: {
    bg: "bg-blue-100",
    text: "text-blue-800",
    defaultLabelFallback: "Owner",
    isUppercase: true,
  },
  admin: {
    bg: "bg-blue-100",
    text: "text-blue-800",
    defaultLabelFallback: "Admin",
    isUppercase: true,
  },

  // Paused / Muted / Neutral statuses
  paused: {
    bg: "bg-slate-100",
    text: "text-slate-700",
    border: "border-slate-200",
    icon: PauseCircle,
    iconClass: "text-slate-500",
    defaultLabelKey: "statusPaused",
    titleKey: "statusPausedDesc",
    isPill: true,
  },
  unknown: {
    bg: "bg-slate-50",
    text: "text-slate-600",
    border: "border-slate-200",
    icon: HelpCircle,
    iconClass: "text-slate-400",
    defaultLabelKey: "statusUnknown",
    titleKey: "statusUnknownDesc",
    isPill: true,
  },
  archived: {
    bg: "bg-slate-100",
    text: "text-slate-600",
    border: "border-slate-200",
    defaultLabelKey: "archived",
    isPill: true,
  },
  revoked: {
    bg: "bg-slate-100",
    text: "text-slate-600",
    border: "border-slate-300",
    icon: ShieldAlert,
    iconClass: "text-slate-500",
    defaultLabelKey: "statusRevoked",
    isPill: true,
  },
  development: {
    bg: "bg-slate-100",
    text: "text-slate-700",
    border: "border-slate-200",
    defaultLabelKey: "development",
    isUppercase: true,
  },
  member: {
    bg: "bg-slate-100",
    text: "text-slate-700",
    defaultLabelFallback: "Member",
    isUppercase: true,
  },
  viewer: {
    bg: "bg-slate-100",
    text: "text-slate-700",
    defaultLabelFallback: "Viewer",
    isUppercase: true,
  },
  secondary: {
    bg: "bg-slate-100",
    text: "text-slate-700",
    border: "border-slate-200",
    isPill: true,
  },
  outline: {
    bg: "bg-transparent",
    text: "text-slate-700",
    border: "border-slate-300",
    isPill: true,
  },
  default: {
    bg: "bg-slate-100",
    text: "text-slate-700",
    isPill: true,
  },

  // Environments & Violet
  production: {
    bg: "bg-blue-50",
    text: "text-blue-700",
    border: "border-blue-200",
    defaultLabelKey: "production",
    isUppercase: true,
  },
  staging: {
    bg: "bg-purple-50",
    text: "text-purple-700",
    border: "border-purple-200",
    defaultLabelKey: "staging",
    isUppercase: true,
  },
  blocked: {
    bg: "transparent",
    text: "text-purple-700",
    icon: ShieldAlert,
    iconClass: "text-purple-600",
    defaultLabelKey: "blockedResult",
    isMinimalText: true,
  },
};

export function Badge({
  variant,
  label,
  children,
  showIcon = true,
  className = "",
  icon: customIcon,
}: BadgeProps) {
  const { t } = useI18n();

  const config = BADGE_CONFIG_MAP[variant] || {
    bg: "bg-slate-100",
    text: "text-slate-700",
    border: "border-slate-200",
    isPill: true,
  };

  const IconComponent = config.icon;
  const resolvedTitle = config.titleKey ? (t as Record<string, string>)[config.titleKey] : undefined;
  
  let resolvedLabel = label || children;
  if (!resolvedLabel && config.defaultLabelKey) {
    resolvedLabel = (t as Record<string, string>)[config.defaultLabelKey];
  }
  if (!resolvedLabel) {
    resolvedLabel = config.defaultLabelFallback || variant;
  }

  if (config.isMinimalText) {
    return (
      <span
        title={resolvedTitle}
        className={`inline-flex items-center gap-1 font-semibold text-xs ${config.text} ${className}`}
      >
        {customIcon ? (
          <span className="shrink-0">{customIcon}</span>
        ) : (
          showIcon && IconComponent && <IconComponent className={`w-3.5 h-3.5 ${config.iconClass || ""}`} />
        )}
        <span>{resolvedLabel}</span>
      </span>
    );
  }

  const shapeClasses = config.isPill
    ? "rounded-full px-2.5 py-0.5 text-xs font-semibold"
    : config.isUppercase
    ? "rounded px-2 py-0.5 text-[11px] font-bold uppercase tracking-wider"
    : "rounded px-2 py-0.5 text-xs font-medium";

  const borderClass = config.border ? `border ${config.border}` : "";

  return (
    <span
      title={resolvedTitle}
      className={`inline-flex items-center gap-1.5 select-none ${shapeClasses} ${config.bg} ${config.text} ${borderClass} ${className}`}
    >
      {customIcon ? (
        <span className="shrink-0">{customIcon}</span>
      ) : (
        showIcon && IconComponent && <IconComponent className={`w-3.5 h-3.5 shrink-0 ${config.iconClass || ""}`} />
      )}
      <span>{resolvedLabel}</span>
    </span>
  );
}
