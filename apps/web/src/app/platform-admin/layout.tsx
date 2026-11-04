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
  Languages,
  ArrowLeft,
  ArrowRight,
  CheckCircle2,
} from "lucide-react";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { Button } from "@/components/ui/Button";
import { AppLayout } from "@/components/AppLayout";

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
      <AppLayout>
        <div className="min-h-[60vh] flex flex-col items-center justify-center p-8">
          <div className="w-10 h-10 border-4 border-blue-600 border-t-transparent rounded-full animate-spin mb-4" />
          <p className="text-sm font-medium text-slate-500">Checking administrator privileges...</p>
        </div>
      </AppLayout>
    );
  }

  // 403 Forbidden Access Denied guard
  if (!user || !isPlatformAdmin) {
    return (
      <AppLayout>
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
      </AppLayout>
    );
  }

  const navTabs = [
    { label: t.platformOverview, href: "/platform-admin", icon: LayoutDashboard, exact: true },
    { label: t.platformUsers, href: "/platform-admin/users", icon: Users, exact: false },
    { label: t.platformOrgs, href: "/platform-admin/organizations", icon: Building, exact: false },
    { label: t.platformPlans, href: "/platform-admin/plans", icon: Palette, exact: false },
    { label: t.platformSubscriptions, href: "/platform-admin/subscriptions", icon: Activity, exact: false },
    { label: t.localizationSettings, href: "/platform-admin/settings/localization", icon: Languages, exact: false },
    { label: t.platformAuditLogs, href: "/platform-admin/audit-logs", icon: FileText, exact: false },
  ];

  return (
    <AppLayout>
      <div className="space-y-6">
        {/* Executive Platform Control Center Banner */}
        <div className="relative overflow-hidden bg-slate-900 text-white rounded-2xl p-5 sm:p-6 shadow-md border border-slate-800">
          <div className="absolute top-0 right-0 -mr-16 -mt-16 w-64 h-64 bg-blue-600/10 rounded-full blur-3xl pointer-events-none" />
          <div className="relative z-10 flex flex-col md:flex-row md:items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 rounded-2xl bg-blue-600/20 border border-blue-400/30 flex items-center justify-center text-blue-400 shrink-0 shadow-inner">
                <ShieldCheck className="w-6 h-6" />
              </div>
              <div>
                <div className="flex items-center gap-2.5 flex-wrap">
                  <h1 className="text-lg sm:text-xl font-bold tracking-tight text-white">
                    {t.navPlatformAdmin}
                  </h1>
                  <span className="text-[11px] font-bold uppercase tracking-wider bg-blue-500/20 text-blue-300 border border-blue-400/30 px-2.5 py-0.5 rounded-full">
                    {t.platformAdminBadge}
                  </span>
                </div>
                <p className="text-xs sm:text-sm text-slate-400 mt-1">
                  Global governance · Independent platform authorization & subscription catalog
                </p>
              </div>
            </div>

            <div className="flex items-center gap-3 self-start md:self-auto text-xs flex-wrap">
              <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-emerald-500/10 text-emerald-300 border border-emerald-500/30 font-medium">
                <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
                <span>{t.platformStatusOperational}</span>
              </div>
              <div className="text-slate-400 bg-slate-800/80 px-3 py-1 rounded-full border border-slate-700/60 font-mono text-[11px]">
                {user.email}
              </div>
            </div>
          </div>

          {/* Sub-Navigation Navigation Pills */}
          <div className="relative z-10 flex items-center gap-2 overflow-x-auto pt-5 mt-5 border-t border-slate-800 no-scrollbar">
            {navTabs.map((tab) => {
              const isActive = tab.exact ? pathname === tab.href : pathname.startsWith(tab.href);
              const Icon = tab.icon;
              return (
                <Link
                  key={tab.href}
                  href={tab.href}
                  className={`flex items-center gap-2 px-3.5 py-2 rounded-xl text-xs font-semibold whitespace-nowrap transition-all duration-150 ${
                    isActive
                      ? "bg-blue-600 text-white shadow-sm ring-1 ring-blue-500"
                      : "text-slate-300 hover:text-white hover:bg-slate-800/90"
                  }`}
                >
                  <Icon className="w-4 h-4 shrink-0" />
                  <span>{tab.label}</span>
                </Link>
              );
            })}
          </div>
        </div>

        {/* Child Page View */}
        <div className="min-h-[400px]">
          {children}
        </div>
      </div>
    </AppLayout>
  );
}
