"use client";

import React, { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Activity,
  Users,
  Globe,
  Settings,
  Building,
  LogOut,
  LogIn,
  UserPlus,
  Menu,
  X,
  ChevronDown,
  LayoutDashboard,
  Shield,
  Languages,
} from "lucide-react";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { Badge } from "./ui/Badge";

interface AppLayoutProps {
  children: React.ReactNode;
}

export function AppLayout({ children }: AppLayoutProps) {
  const pathname = usePathname();
  const { user, activeOrg, role, logout } = useAuth();
  const { t, locale, toggleLocale, isRTL } = useI18n();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const navItems = [
    {
      label: t.dashboard,
      href: "/",
      icon: <LayoutDashboard className="w-4 h-4" />,
      active: pathname === "/",
    },
    {
      label: t.clients,
      href: "/clients",
      icon: <Users className="w-4 h-4" />,
      active: pathname.startsWith("/clients"),
    },
    {
      label: t.assets,
      href: "/assets",
      icon: <Globe className="w-4 h-4" />,
      active: pathname.startsWith("/assets"),
    },
    {
      label: t.monitors,
      href: "/monitors",
      icon: <Activity className="w-4 h-4" />,
      active: pathname.startsWith("/monitors"),
    },
    {
      label: t.members,
      href: "/settings/members",
      icon: <Shield className="w-4 h-4" />,
      active: pathname === "/settings/members",
    },
    {
      label: t.accountSettings,
      href: "/settings/account",
      icon: <Settings className="w-4 h-4" />,
      active: pathname === "/settings/account",
    },
  ];

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col antialiased">
      {/* Top Header */}
      <header className="sticky top-0 z-40 bg-white border-b border-slate-200 shadow-sm">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            {/* Mobile Menu Button */}
            {user && (
              <button
                onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
                className="lg:hidden p-2 text-slate-600 hover:text-slate-900 hover:bg-slate-100 rounded-lg transition-colors"
                aria-label={t.menu}
              >
                {mobileMenuOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
              </button>
            )}

            {/* Brand Logo */}
            <Link href="/" className="flex items-center gap-2.5 group">
              <div className="w-9 h-9 rounded-xl bg-blue-600 group-hover:bg-blue-700 text-white flex items-center justify-center shadow-md transition-colors">
                <Activity className="w-5 h-5" />
              </div>
              <div>
                <span className="font-extrabold text-slate-900 tracking-tight text-lg">
                  {t.brandName}
                </span>
                <span className="hidden sm:inline-block ms-2 text-[11px] font-semibold text-slate-500">
                  {t.tagline}
                </span>
              </div>
            </Link>
          </div>

          {/* Desktop Top Links & Tenant Picker */}
          <div className="flex items-center gap-2.5 sm:gap-3">
            {user ? (
              <>
                {/* Active Workspace Pill */}
                <Link
                  href="/organizations"
                  className="hidden sm:flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-slate-200 bg-slate-50 hover:bg-slate-100 text-xs font-semibold text-slate-700 transition-colors"
                  title={t.activeWorkspace}
                >
                  <Building className="w-3.5 h-3.5 text-blue-600" />
                  <span className="max-w-[120px] sm:max-w-[160px] truncate">
                    {activeOrg?.name || t.activeOrg}
                  </span>
                  {role && <Badge variant={role} />}
                </Link>

                {/* Language Toggle */}
                <button
                  onClick={toggleLocale}
                  className="inline-flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-semibold text-slate-700 bg-slate-50 hover:bg-slate-100 border border-slate-200 rounded-lg transition-colors"
                  title="Toggle Language / تغییر زبان"
                >
                  <Languages className="w-3.5 h-3.5 text-slate-500" />
                  <span className="hidden sm:inline">{locale === "en" ? "فارسی" : "English"}</span>
                </button>

                {/* Logout Button */}
                <button
                  onClick={logout}
                  className="inline-flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-semibold text-rose-600 hover:text-rose-700 hover:bg-rose-50 border border-transparent rounded-lg transition-colors"
                  title={t.logout}
                >
                  <LogOut className="w-3.5 h-3.5" />
                  <span className="hidden sm:inline">{t.logout}</span>
                </button>
              </>
            ) : (
              <>
                <button
                  onClick={toggleLocale}
                  className="inline-flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-semibold text-slate-700 bg-slate-50 hover:bg-slate-100 border border-slate-200 rounded-lg transition-colors me-1"
                >
                  <Languages className="w-3.5 h-3.5 text-slate-500" />
                  <span>{locale === "en" ? "فارسی" : "English"}</span>
                </button>
                <Link
                  href="/login"
                  className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-semibold text-slate-700 hover:text-slate-900 hover:bg-slate-100 rounded-lg transition-colors"
                >
                  <LogIn className="w-3.5 h-3.5" />
                  <span>{t.login}</span>
                </Link>
                <Link
                  href="/signup"
                  className="inline-flex items-center gap-1 px-3.5 py-1.5 text-xs font-semibold text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm transition-colors"
                >
                  <UserPlus className="w-3.5 h-3.5" />
                  <span>{t.signup}</span>
                </Link>
              </>
            )}
          </div>
        </div>
      </header>

      {/* Main Container with Sidebar + Content */}
      <div className="flex-1 flex max-w-7xl mx-auto w-full px-4 sm:px-6 lg:px-8 py-6 sm:py-8 gap-8">
        {/* Desktop Sidebar (Rendered only when logged in) */}
        {user && (
          <aside className="hidden lg:block w-56 shrink-0 space-y-6">
            <div className="sticky top-24 space-y-1">
              <div className="text-[10px] uppercase font-bold text-slate-400 px-3 mb-2 tracking-wider">
                {t.navigation}
              </div>
              {navItems.map((item) => (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`flex items-center gap-2.5 px-3 py-2 rounded-xl text-xs font-semibold transition-all ${
                    item.active
                      ? "bg-blue-600 text-white shadow-sm"
                      : "text-slate-600 hover:text-slate-900 hover:bg-slate-100"
                  }`}
                >
                  <span className={item.active ? "text-white" : "text-slate-400"}>
                    {item.icon}
                  </span>
                  <span>{item.label}</span>
                </Link>
              ))}

              <div className="pt-6">
                <div className="text-[10px] uppercase font-bold text-slate-400 px-3 mb-2 tracking-wider">
                  {t.organizations}
                </div>
                <Link
                  href="/organizations"
                  className={`flex items-center gap-2.5 px-3 py-2 rounded-xl text-xs font-semibold transition-all ${
                    pathname === "/organizations"
                      ? "bg-blue-600 text-white shadow-sm"
                      : "text-slate-600 hover:text-slate-900 hover:bg-slate-100"
                  }`}
                >
                  <Building className="w-4 h-4 text-slate-400" />
                  <span>{t.myOrganizations}</span>
                </Link>
              </div>
            </div>
          </aside>
        )}

        {/* Mobile Navigation Drawer */}
        {user && mobileMenuOpen && (
          <div
            className="fixed inset-0 z-50 lg:hidden bg-slate-900/40 backdrop-blur-sm"
            onClick={() => setMobileMenuOpen(false)}
          >
            <div
              className={`fixed top-0 bottom-0 ${
                isRTL ? "right-0" : "left-0"
              } w-64 bg-white shadow-2xl p-6 flex flex-col justify-between`}
              onClick={(e) => e.stopPropagation()}
            >
              <div className="space-y-6">
                <div className="flex items-center justify-between pb-4 border-b border-slate-100">
                  <div className="flex items-center gap-2">
                    <Activity className="w-5 h-5 text-blue-600" />
                    <span className="font-bold text-slate-900 text-base">{t.brandName}</span>
                  </div>
                  <button
                    onClick={() => setMobileMenuOpen(false)}
                    className="p-1 text-slate-400 hover:text-slate-600 rounded-lg"
                  >
                    <X className="w-5 h-5" />
                  </button>
                </div>

                <div className="space-y-1">
                  {navItems.map((item) => (
                    <Link
                      key={item.href}
                      href={item.href}
                      onClick={() => setMobileMenuOpen(false)}
                      className={`flex items-center gap-2.5 px-3 py-2.5 rounded-xl text-xs font-semibold transition-all ${
                        item.active
                          ? "bg-blue-600 text-white shadow-sm"
                          : "text-slate-600 hover:bg-slate-100 hover:text-slate-900"
                      }`}
                    >
                      <span className={item.active ? "text-white" : "text-slate-400"}>
                        {item.icon}
                      </span>
                      <span>{item.label}</span>
                    </Link>
                  ))}

                  <Link
                    href="/organizations"
                    onClick={() => setMobileMenuOpen(false)}
                    className={`flex items-center gap-2.5 px-3 py-2.5 rounded-xl text-xs font-semibold transition-all ${
                      pathname === "/organizations"
                        ? "bg-blue-600 text-white shadow-sm"
                        : "text-slate-600 hover:bg-slate-100 hover:text-slate-900"
                    }`}
                  >
                    <Building className="w-4 h-4 text-slate-400" />
                    <span>{t.myOrganizations}</span>
                  </Link>
                </div>
              </div>

              <div className="pt-4 border-t border-slate-100 space-y-3">
                <div className="text-xs text-slate-500 font-medium">
                  {user.display_name} ({user.email})
                </div>
                <button
                  onClick={() => {
                    setMobileMenuOpen(false);
                    logout();
                  }}
                  className="w-full flex items-center justify-center gap-2 px-3 py-2 text-xs font-semibold text-rose-600 bg-rose-50 hover:bg-rose-100 rounded-lg transition-colors"
                >
                  <LogOut className="w-4 h-4" />
                  <span>{t.logout}</span>
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Page Main Content Area */}
        <main className="flex-1 w-full min-w-0">{children}</main>
      </div>
    </div>
  );
}
