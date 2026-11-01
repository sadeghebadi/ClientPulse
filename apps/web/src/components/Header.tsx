"use client";

import React from "react";
import Link from "next/link";
import { Activity, Globe, Users, Building, Settings, LogOut, LogIn, UserPlus } from "lucide-react";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";

export function Header() {
  const { user, activeOrg, role, logout } = useAuth();
  const { t, locale, toggleLocale, isRTL } = useI18n();

  return (
    <header className="border-b border-slate-200 bg-white sticky top-0 z-50 shadow-sm">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
        {/* Brand */}
        <Link href="/" className="flex items-center gap-3 group">
          <div className="w-10 h-10 rounded-lg bg-blue-600 group-hover:bg-blue-700 flex items-center justify-center text-white shadow-md transition-colors">
            <Activity className="w-6 h-6" />
          </div>
          <div>
            <h1 className="text-xl font-bold text-slate-900 tracking-tight">{t.brandName}</h1>
            <p className="text-xs text-slate-500 font-medium">{t.tagline}</p>
          </div>
        </Link>

        {/* Right Section */}
        <div className="flex items-center gap-3">
          {user ? (
            <>
              {/* Active Workspace / Org Switcher Link */}
              <Link
                href="/organizations"
                className="hidden sm:flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-slate-200 bg-slate-50 hover:bg-slate-100 text-xs font-semibold text-slate-700 transition-colors"
                title={t.activeWorkspace}
              >
                <Building className="w-3.5 h-3.5 text-blue-600" />
                <span className="max-w-[140px] truncate">{activeOrg?.name || t.activeOrg}</span>
                {role && (
                  <span className="text-[10px] uppercase font-bold px-1.5 py-0.5 rounded bg-blue-100 text-blue-800">
                    {role}
                  </span>
                )}
              </Link>

              {/* Navigation links for active tenant */}
              <Link
                href="/clients"
                className="hidden md:flex items-center gap-1 px-2.5 py-1.5 text-xs font-semibold text-slate-700 hover:text-blue-600 hover:bg-slate-50 rounded-md transition-colors"
              >
                <span>{t.clients}</span>
              </Link>

              <Link
                href="/monitors"
                className="hidden md:flex items-center gap-1 px-2.5 py-1.5 text-xs font-semibold text-slate-700 hover:text-blue-600 hover:bg-slate-50 rounded-md transition-colors"
              >
                <span>{t.monitors}</span>
              </Link>

              <Link
                href="/settings/members"
                className="hidden md:flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium text-slate-600 hover:text-slate-900 hover:bg-slate-50 rounded-md transition-colors"
              >
                <Users className="w-3.5 h-3.5" />
                <span>{t.members}</span>
              </Link>

              <Link
                href="/settings/account"
                className="hidden md:flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium text-slate-600 hover:text-slate-900 hover:bg-slate-50 rounded-md transition-colors"
              >
                <Settings className="w-3.5 h-3.5" />
                <span>{t.accountSettings}</span>
              </Link>

              <button
                onClick={logout}
                className="flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium text-red-600 hover:text-red-700 hover:bg-red-50 rounded-md transition-colors"
                title={t.logout}
              >
                <LogOut className="w-3.5 h-3.5" />
                <span className="hidden sm:inline">{t.logout}</span>
              </button>
            </>
          ) : (
            <>
              <Link
                href="/login"
                className="flex items-center gap-1 px-3 py-1.5 text-xs font-semibold text-slate-700 hover:text-slate-900 hover:bg-slate-100 rounded-md transition-colors"
              >
                <LogIn className="w-3.5 h-3.5" />
                <span>{t.login}</span>
              </Link>

              <Link
                href="/signup"
                className="flex items-center gap-1 px-3 py-1.5 text-xs font-semibold text-white bg-blue-600 hover:bg-blue-700 rounded-md shadow-sm transition-colors"
              >
                <UserPlus className="w-3.5 h-3.5" />
                <span>{t.signup}</span>
              </Link>
            </>
          )}

          {/* Language Toggle Button */}
          <button
            onClick={toggleLocale}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-md transition-colors border border-slate-200"
            title="Toggle Language / تغییر زبان"
          >
            <Globe className="w-3.5 h-3.5 text-slate-500" />
            <span>{locale === "en" ? "فارسی" : "English"}</span>
          </button>
        </div>
      </div>
    </header>
  );
}
