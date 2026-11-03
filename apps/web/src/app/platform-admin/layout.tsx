"use client";

import React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  ShieldAlert,
  ShieldCheck,
  LayoutDashboard,
  Users,
  Building,
  Palette,
  Activity,
  FileText,
  ArrowLeft,
  ArrowRight,
} from "lucide-react";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/Button";

export default function PlatformAdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const { user, isPlatformAdmin, loading } = useAuth();
  const { t, isRTL } = useI18n();

  if (loading) {
    return (
      <div className="min-h-[60vh] flex flex-col items-center justify-center p-8">
        <div className="w-10 h-10 border-4 border-blue-600 border-t-transparent rounded-full animate-spin mb-4" />
        <p className="text-sm font-medium text-slate-500">Checking administrator privileges...</p>
      </div>
    );
  }

  // 403 Forbidden Access Denied guard
  if (!user || !isPlatformAdmin) {
    return (
      <div className="min-h-[70vh] flex flex-col items-center justify-center p-6 text-center max-w-xl mx-auto">
        <div className="w-20 h-20 rounded-3xl bg-rose-50 border border-rose-200 flex items-center justify-center mb-6 text-rose-600 shadow-sm animate-pulse">
          <ShieldAlert className="w-10 h-10" />
        </div>
        <h1 className="text-2xl sm:text-3xl font-black text-slate-900 mb-3 tracking-tight">
          {t.accessDeniedTitle}
        </h1>
        <p className="text-slate-600 text-sm sm:text-base leading-relaxed mb-8">
          {t.accessDeniedDesc}
        </p>
        <div className="flex flex-col sm:flex-row gap-3 items-center justify-center w-full">
          <Link href="/dashboard" className="w-full sm:w-auto">
            <Button variant="primary" className="w-full justify-center gap-2">
              {isRTL ? <ArrowRight className="w-4 h-4" /> : <ArrowLeft className="w-4 h-4" />}
              {t.returnToDashboard}
            </Button>
          </Link>
        </div>
      </div>
    );
  }

  const navTabs = [
    { label: t.platformOverview, href: "/platform-admin", icon: LayoutDashboard, exact: true },
    { label: t.platformUsers, href: "/platform-admin/users", icon: Users, exact: false },
    { label: t.platformOrgs, href: "/platform-admin/organizations", icon: Building, exact: false },
    { label: t.platformPlans, href: "/platform-admin/plans", icon: Palette, exact: false },
    { label: t.platformSubscriptions, href: "/platform-admin/subscriptions", icon: Activity, exact: false },
    { label: t.platformAuditLogs, href: "/platform-admin/audit-logs", icon: FileText, exact: false },
  ];

  return (
    <div className="space-y-6">
      {/* Platform Admin Top Header Strip */}
      <div className="bg-slate-900 text-white rounded-2xl p-4 sm:p-5 shadow-sm border border-slate-800">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-blue-600/30 border border-blue-400/40 flex items-center justify-center text-blue-400 shrink-0">
              <ShieldCheck className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="text-base sm:text-lg font-bold tracking-tight text-white">
                  {t.navPlatformAdmin}
                </span>
                <span className="text-[10px] font-black uppercase tracking-wider bg-blue-500/20 text-blue-300 border border-blue-400/30 px-2 py-0.5 rounded-full">
                  {t.platformAdminBadge}
                </span>
              </div>
              <p className="text-xs text-slate-400 mt-0.5">
                Global governance · Independent platform authorization & subscription management
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2 self-start md:self-auto text-xs text-slate-400">
            <span className="w-2 h-2 rounded-full bg-emerald-500 animate-ping" />
            <span>Admin Session Active ({user.email})</span>
          </div>
        </div>

        {/* Admin Navigation Tabs */}
        <div className="flex items-center gap-1.5 overflow-x-auto pt-4 mt-4 border-t border-slate-800/80 no-scrollbar">
          {navTabs.map((tab) => {
            const isActive = tab.exact ? pathname === tab.href : pathname.startsWith(tab.href);
            const Icon = tab.icon;
            return (
              <Link
                key={tab.href}
                href={tab.href}
                className={`flex items-center gap-2 px-3.5 py-1.5 rounded-xl text-xs font-semibold whitespace-nowrap transition-all ${
                  isActive
                    ? "bg-blue-600 text-white shadow-sm"
                    : "text-slate-300 hover:text-white hover:bg-slate-800"
                }`}
              >
                <Icon className="w-3.5 h-3.5" />
                <span>{tab.label}</span>
              </Link>
            );
          })}
        </div>
      </div>

      {/* Main Content View */}
      {children}
    </div>
  );
}

