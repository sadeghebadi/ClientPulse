"use client";

import React, { useState, useEffect, useRef } from "react";
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
  ChevronRight,
  ChevronLeft,
  LayoutDashboard,
  Shield,
  Languages,
  Radio,
  AlertTriangle,
  Wrench,
  FileText,
  Palette,
  Bell,
  PanelLeftClose,
  PanelLeftOpen,
  User as UserIcon,
} from "lucide-react";
import { useAuth } from "@/lib/auth-context";
import { useI18n } from "@/lib/i18n";
import { Badge } from "./ui/Badge";

interface AppLayoutProps {
  children: React.ReactNode;
}

interface NavGroup {
  groupKey: string;
  label: string;
  items: {
    label: string;
    href: string;
    icon: React.ComponentType<{ className?: string }>;
    isActive: (pathname: string) => boolean;
    requiredRole?: string[];
  }[];
}

export function AppLayout({ children }: AppLayoutProps) {
  const pathname = usePathname();
  const { user, activeOrg, role, memberships, switchOrganization, logout, loading } = useAuth();
  const { t, locale, toggleLocale, isRTL } = useI18n();

  // Mobile drawer state
  const [mobileDrawerOpen, setMobileDrawerOpen] = useState(false);
  const mobileTriggerRef = useRef<HTMLButtonElement>(null);
  const mobileDrawerRef = useRef<HTMLDivElement>(null);

  // Desktop sidebar collapse state (for intermediate screens 1024-1279px or manual user toggle)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  // Header Dropdown States
  const [workspaceMenuOpen, setWorkspaceMenuOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);

  const workspaceDropdownRef = useRef<HTMLDivElement>(null);
  const userDropdownRef = useRef<HTMLDivElement>(null);

  // Close menus on outside click
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        workspaceDropdownRef.current &&
        !workspaceDropdownRef.current.contains(event.target as Node)
      ) {
        setWorkspaceMenuOpen(false);
      }
      if (
        userDropdownRef.current &&
        !userDropdownRef.current.contains(event.target as Node)
      ) {
        setUserMenuOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, []);

  // Close mobile drawer on Escape and handle body scroll lock
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") {
        if (mobileDrawerOpen) {
          setMobileDrawerOpen(false);
          mobileTriggerRef.current?.focus();
        }
        if (workspaceMenuOpen) setWorkspaceMenuOpen(false);
        if (userMenuOpen) setUserMenuOpen(false);
      }
    }

    if (mobileDrawerOpen) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      document.body.style.overflow = "";
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [mobileDrawerOpen, workspaceMenuOpen, userMenuOpen]);

  // Track pathname changes to reset open popovers/drawers during render
  const [prevPathname, setPrevPathname] = useState(pathname);
  if (prevPathname !== pathname) {
    setPrevPathname(pathname);
    setMobileDrawerOpen(false);
    setWorkspaceMenuOpen(false);
    setUserMenuOpen(false);
  }

  // Structured Navigation Groups
  const navGroups: NavGroup[] = [
    {
      groupKey: "overview",
      label: t.navOverview,
      items: [
        {
          label: t.dashboard,
          href: "/",
          icon: LayoutDashboard,
          isActive: (p) => p === "/",
        },
      ],
    },
    {
      groupKey: "management",
      label: t.navManagement,
      items: [
        {
          label: t.clients,
          href: "/clients",
          icon: Users,
          isActive: (p) => p === "/clients" || p.startsWith("/clients/"),
        },
        {
          label: t.assets,
          href: "/assets",
          icon: Globe,
          isActive: (p) => p === "/assets" || p.startsWith("/assets/"),
        },
        {
          label: t.monitors,
          href: "/monitors",
          icon: Activity,
          isActive: (p) => p === "/monitors" || p.startsWith("/monitors/"),
        },
        {
          label: t.heartbeats,
          href: "/heartbeats",
          icon: Radio,
          isActive: (p) => p === "/heartbeats" || p.startsWith("/heartbeats/"),
        },
      ],
    },
    {
      groupKey: "operations",
      label: t.navOperations,
      items: [
        {
          label: t.incidents,
          href: "/incidents",
          icon: AlertTriangle,
          isActive: (p) => p === "/incidents" || p.startsWith("/incidents/"),
        },
        {
          label: t.maintenance,
          href: "/maintenance",
          icon: Wrench,
          isActive: (p) => p === "/maintenance" || p.startsWith("/maintenance/"),
        },
        {
          label: t.reports,
          href: "/reports",
          icon: FileText,
          isActive: (p) => p === "/reports" || p.startsWith("/reports/"),
        },
      ],
    },
    {
      groupKey: "settings",
      label: t.navSettings,
      items: [
        {
          label: t.accountSettings,
          href: "/settings/account",
          icon: Settings,
          isActive: (p) => p === "/settings/account",
        },
        {
          label: t.agencyBranding,
          href: "/settings/branding",
          icon: Palette,
          isActive: (p) => p === "/settings/branding",
          requiredRole: ["owner", "admin"],
        },
        {
          label: t.members,
          href: "/settings/members",
          icon: Shield,
          isActive: (p) => p === "/settings/members",
          requiredRole: ["owner", "admin"],
        },
        {
          label: t.notifications,
          href: "/settings/notifications",
          icon: Bell,
          isActive: (p) => p === "/settings/notifications",
        },
      ],
    },
  ];

  // Helper to determine active breadcrumb label for the top header
  const getPageContextLabel = (): string => {
    if (pathname === "/") return t.dashboard;
    if (pathname.startsWith("/clients")) return t.clients;
    if (pathname.startsWith("/assets")) return t.assets;
    if (pathname.startsWith("/monitors")) return t.monitors;
    if (pathname.startsWith("/heartbeats")) return t.heartbeats;
    if (pathname.startsWith("/incidents")) return t.incidents;
    if (pathname.startsWith("/maintenance")) return t.maintenance;
    if (pathname.startsWith("/reports")) return t.reports;
    if (pathname.startsWith("/settings/notifications")) return t.notifications;
    if (pathname.startsWith("/settings/branding")) return t.agencyBranding;
    if (pathname.startsWith("/settings/members")) return t.members;
    if (pathname.startsWith("/settings/account")) return t.accountSettings;
    if (pathname.startsWith("/organizations")) return t.organizations;
    if (pathname.startsWith("/onboarding")) return t.onboardingTitle;
    return "";
  };

  // If user is not authenticated, render simplified guest layout with top navigation
  if (!user && !loading) {
    return (
      <div className="min-h-screen bg-slate-50 flex flex-col antialiased">
        {/* Guest Public Header */}
        <header className="sticky top-0 z-40 bg-white/95 backdrop-blur-md border-b border-slate-200 shadow-xs">
          <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between gap-4">
            <Link href="/" className="flex items-center gap-2.5 group">
              <div className="w-9 h-9 rounded-xl bg-blue-600 group-hover:bg-blue-700 text-white flex items-center justify-center shadow-md transition-colors">
                <Activity className="w-5 h-5" />
              </div>
              <div className="flex items-baseline">
                <span className="font-extrabold text-slate-900 tracking-tight text-lg">
                  {t.brandName}
                </span>
                <span className="hidden sm:inline-block ms-2.5 text-[11px] font-semibold text-slate-500">
                  {t.tagline}
                </span>
              </div>
            </Link>

            <div className="flex items-center gap-2 sm:gap-3">
              {/* Language Switcher */}
              <button
                type="button"
                onClick={toggleLocale}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold text-slate-700 bg-white hover:bg-slate-50 border border-slate-200 rounded-lg shadow-2xs transition-colors"
                title="Toggle Language / تغییر زبان"
              >
                <Languages className="w-3.5 h-3.5 text-slate-500" />
                <span>{locale === "en" ? "فارسی" : "English"}</span>
              </button>

              <Link
                href="/login"
                className="inline-flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-semibold text-slate-700 hover:text-slate-900 hover:bg-slate-100 rounded-lg transition-colors"
              >
                <LogIn className="w-3.5 h-3.5" />
                <span>{t.login}</span>
              </Link>
              <Link
                href="/signup"
                className="inline-flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-semibold text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm transition-colors"
              >
                <UserPlus className="w-3.5 h-3.5" />
                <span>{t.signup}</span>
              </Link>
            </div>
          </div>
        </header>

        {/* Public Content Container */}
        <main className="flex-1 w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          {children}
        </main>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col antialiased">
      <div className="flex-1 flex w-full min-h-screen overflow-x-hidden">
        {/* ======================================================== */}
        {/* DESKTOP SIDEBAR (Deep Slate #0f172a, full-height viewport) */}
        {/* ======================================================== */}
        {user && (
          <aside
            className={`hidden lg:flex flex-col shrink-0 bg-[#0f172a] text-slate-200 border-e border-slate-800 transition-all duration-300 z-30 select-none ${
              sidebarCollapsed ? "w-18" : "w-60 xl:w-64"
            }`}
          >
            {/* Sidebar Brand Header */}
            <div className="h-16 flex items-center justify-between px-4 border-b border-slate-800/80">
              <Link
                href="/"
                className={`flex items-center gap-3 overflow-hidden ${
                  sidebarCollapsed ? "justify-center w-full" : ""
                }`}
                title={t.brandName}
              >
                <div className="w-9 h-9 rounded-xl bg-blue-600 hover:bg-blue-500 text-white flex items-center justify-center shadow-md shrink-0 transition-colors">
                  <Activity className="w-5 h-5" />
                </div>
                {!sidebarCollapsed && (
                  <div className="flex flex-col min-w-0">
                    <span className="font-extrabold text-white text-base tracking-tight truncate">
                      {t.brandName}
                    </span>
                    <span className="text-[10px] text-slate-400 font-medium truncate">
                      {t.tagline}
                    </span>
                  </div>
                )}
              </Link>
            </div>

            {/* Sidebar Active Workspace Card */}
            {!sidebarCollapsed && (
              <div className="p-3 border-b border-slate-800/60">
                <Link
                  href="/organizations"
                  className="flex items-center justify-between p-2 rounded-xl bg-slate-800/50 hover:bg-slate-800 border border-slate-700/60 transition-colors group"
                  title={t.switchWorkspace}
                >
                  <div className="flex items-center gap-2.5 min-w-0">
                    <div className="w-7 h-7 rounded-lg bg-blue-500/10 border border-blue-500/20 text-blue-400 flex items-center justify-center shrink-0">
                      <Building className="w-4 h-4" />
                    </div>
                    <div className="min-w-0 text-start">
                      <div className="text-xs font-semibold text-slate-200 truncate">
                        {activeOrg?.name || t.activeOrg}
                      </div>
                      <div className="text-[10px] text-slate-400 capitalize">
                        {role || "member"}
                      </div>
                    </div>
                  </div>
                  <ChevronRight className="w-4 h-4 text-slate-500 group-hover:text-slate-300 shrink-0 rtl:rotate-180 transition-transform" />
                </Link>
              </div>
            )}

            {/* Sidebar Navigation Items */}
            <nav className="flex-1 overflow-y-auto px-3 py-4 space-y-6">
              {navGroups.map((group) => {
                const visibleItems = group.items.filter((item) => {
                  if (!item.requiredRole) return true;
                  return role && item.requiredRole.includes(role);
                });

                if (visibleItems.length === 0) return null;

                return (
                  <div key={group.groupKey} className="space-y-1">
                    {!sidebarCollapsed && (
                      <div className="px-3 mb-1.5 text-[10px] font-bold tracking-wider text-slate-400 uppercase">
                        {group.label}
                      </div>
                    )}
                    {visibleItems.map((item) => {
                      const active = item.isActive(pathname);
                      const Icon = item.icon;

                      return (
                        <Link
                          key={item.href}
                          href={item.href}
                          title={sidebarCollapsed ? item.label : undefined}
                          aria-current={active ? "page" : undefined}
                          className={`flex items-center gap-3 px-3 py-2 rounded-xl text-xs font-semibold transition-all relative group ${
                            active
                              ? "bg-blue-600 text-white shadow-sm"
                              : "text-slate-300 hover:text-white hover:bg-slate-800/60"
                          } ${sidebarCollapsed ? "justify-center px-2" : ""}`}
                        >
                          <Icon
                            className={`w-4 h-4 shrink-0 transition-colors ${
                              active ? "text-white" : "text-slate-400 group-hover:text-slate-200"
                            }`}
                          />
                          {!sidebarCollapsed && <span className="truncate">{item.label}</span>}
                          {sidebarCollapsed && (
                            <span className="sr-only">{item.label}</span>
                          )}
                        </Link>
                      );
                    })}
                  </div>
                );
              })}
            </nav>

            {/* Sidebar Footer Controls */}
            <div className="p-3 border-t border-slate-800/80 flex items-center justify-between gap-2">
              <button
                type="button"
                onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
                className="w-full flex items-center justify-center gap-2 p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800/60 text-xs font-medium transition-colors"
                title={sidebarCollapsed ? t.expandSidebar : t.collapseSidebar}
                aria-label={sidebarCollapsed ? t.expandSidebar : t.collapseSidebar}
              >
                {sidebarCollapsed ? (
                  isRTL ? <PanelLeftClose className="w-4 h-4" /> : <PanelLeftOpen className="w-4 h-4" />
                ) : (
                  <>
                    {isRTL ? <PanelLeftOpen className="w-4 h-4" /> : <PanelLeftClose className="w-4 h-4" />}
                    <span className="truncate">{t.collapseSidebar}</span>
                  </>
                )}
              </button>
            </div>
          </aside>
        )}

        {/* ======================================================== */}
        {/* MAIN APPLICATION AREA (Header + Content) */}
        {/* ======================================================== */}
        <div className="flex-1 flex flex-col min-w-0 bg-slate-50 min-h-screen">
          {/* Top Application Header */}
          <header className="sticky top-0 z-20 h-16 bg-white border-b border-slate-200 shadow-2xs flex items-center justify-between px-4 sm:px-6 lg:px-8 gap-4">
            {/* Start Side: Mobile Trigger + Breadcrumb/Context */}
            <div className="flex items-center gap-3 min-w-0">
              {/* Mobile Drawer Hamburger (Visible below 1024px) */}
              {user && (
                <button
                  ref={mobileTriggerRef}
                  type="button"
                  onClick={() => setMobileDrawerOpen(true)}
                  className="lg:hidden p-2 text-slate-600 hover:text-slate-900 hover:bg-slate-100 rounded-lg transition-colors"
                  aria-label={t.menu}
                  aria-expanded={mobileDrawerOpen}
                >
                  <Menu className="w-5 h-5" />
                </button>
              )}

              {/* Mobile Logo Fallback (below 1024px) */}
              <Link href="/" className="lg:hidden flex items-center gap-2">
                <div className="w-8 h-8 rounded-lg bg-blue-600 text-white flex items-center justify-center shadow-xs">
                  <Activity className="w-4 h-4" />
                </div>
                <span className="font-extrabold text-slate-900 tracking-tight text-sm hidden sm:inline">
                  {t.brandName}
                </span>
              </Link>

              {/* Header Context / Breadcrumb Indicator */}
              <div className="hidden md:flex items-center gap-2 text-xs text-slate-500 ps-2 border-s border-slate-200">
                <span className="font-medium">{t.brandName}</span>
                <span className="text-slate-300">/</span>
                <span className="font-bold text-slate-900">{getPageContextLabel()}</span>
              </div>
            </div>

            {/* End Side: Workspace Switcher + Language + User Menu */}
            <div className="flex items-center gap-2 sm:gap-3">
              {/* Active Workspace Selector Dropdown */}
              {user && (
                <div className="relative" ref={workspaceDropdownRef}>
                  <button
                    type="button"
                    onClick={() => setWorkspaceMenuOpen(!workspaceMenuOpen)}
                    className="flex items-center gap-2 px-2.5 sm:px-3 py-1.5 rounded-lg border border-slate-200 bg-slate-50 hover:bg-slate-100 text-xs font-semibold text-slate-700 transition-colors shadow-2xs"
                    aria-expanded={workspaceMenuOpen}
                    aria-label={t.activeWorkspace}
                  >
                    <Building className="w-3.5 h-3.5 text-blue-600 shrink-0" />
                    <span className="max-w-[100px] sm:max-w-[140px] truncate">
                      {activeOrg?.name || t.activeOrg}
                    </span>
                    {role && (
                      <span className="hidden sm:inline-block">
                        <Badge variant={role} />
                      </span>
                    )}
                    <ChevronDown className="w-3.5 h-3.5 text-slate-400 shrink-0" />
                  </button>

                  {/* Dropdown Menu */}
                  {workspaceMenuOpen && (
                    <div
                      className={`absolute top-full mt-1.5 w-60 bg-white rounded-xl shadow-xl border border-slate-200 py-1.5 z-50 text-xs ${
                        isRTL ? "left-0" : "right-0"
                      }`}
                    >
                      <div className="px-3 py-2 border-b border-slate-100 font-semibold text-slate-400 uppercase text-[10px] tracking-wider">
                        {t.myOrganizations}
                      </div>
                      <div className="max-h-56 overflow-y-auto py-1">
                        {memberships.map((m) => {
                          const isActive = m.organization_id === activeOrg?.id;
                          return (
                            <button
                              key={m.id}
                              type="button"
                              onClick={async () => {
                                setWorkspaceMenuOpen(false);
                                if (!isActive) {
                                  await switchOrganization(m.organization_id);
                                }
                              }}
                              className={`w-full flex items-center justify-between px-3 py-2 text-start transition-colors ${
                                isActive
                                  ? "bg-blue-50/70 text-blue-700 font-bold"
                                  : "text-slate-700 hover:bg-slate-50"
                              }`}
                            >
                              <div className="min-w-0">
                                <div className="truncate">{m.org_name || m.organization_id}</div>
                                <div className="text-[10px] text-slate-400 capitalize">{m.role}</div>
                              </div>
                              {isActive && (
                                <span className="w-2 h-2 rounded-full bg-blue-600 shrink-0" />
                              )}
                            </button>
                          );
                        })}
                      </div>
                      <div className="border-t border-slate-100 pt-1 mt-1">
                        <Link
                          href="/organizations"
                          onClick={() => setWorkspaceMenuOpen(false)}
                          className="flex items-center gap-2 px-3 py-2 text-slate-700 hover:bg-slate-50 font-medium"
                        >
                          <Building className="w-3.5 h-3.5 text-slate-400" />
                          <span>{t.organizations}</span>
                        </Link>
                      </div>
                    </div>
                  )}
                </div>
              )}

              {/* Language Switcher */}
              <button
                type="button"
                onClick={toggleLocale}
                className="inline-flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-semibold text-slate-700 bg-slate-50 hover:bg-slate-100 border border-slate-200 rounded-lg transition-colors shadow-2xs"
                title="Toggle Language / تغییر زبان"
              >
                <Languages className="w-3.5 h-3.5 text-slate-500" />
                <span className="hidden sm:inline">{locale === "en" ? "فارسی" : "English"}</span>
              </button>

              {/* User Account Menu Dropdown */}
              {user && (
                <div className="relative" ref={userDropdownRef}>
                  <button
                    type="button"
                    onClick={() => setUserMenuOpen(!userMenuOpen)}
                    className="flex items-center gap-2 p-1.5 sm:px-2.5 sm:py-1.5 rounded-lg border border-slate-200 bg-slate-50 hover:bg-slate-100 text-xs font-semibold text-slate-700 transition-colors shadow-2xs"
                    aria-expanded={userMenuOpen}
                    aria-label={t.userAccount}
                  >
                    <div className="w-6 h-6 rounded-full bg-blue-600 text-white flex items-center justify-center text-[11px] font-bold shrink-0">
                      {user.display_name?.charAt(0).toUpperCase() || "U"}
                    </div>
                    <span className="hidden sm:inline-block max-w-[120px] truncate">
                      {user.display_name}
                    </span>
                    <ChevronDown className="w-3.5 h-3.5 text-slate-400 shrink-0" />
                  </button>

                  {/* Dropdown Menu */}
                  {userMenuOpen && (
                    <div
                      className={`absolute top-full mt-1.5 w-56 bg-white rounded-xl shadow-xl border border-slate-200 py-1.5 z-50 text-xs ${
                        isRTL ? "left-0" : "right-0"
                      }`}
                    >
                      <div className="px-3 py-2 border-b border-slate-100">
                        <div className="font-bold text-slate-900 truncate">{user.display_name}</div>
                        <div className="text-[11px] text-slate-500 truncate dir-ltr text-start">
                          {user.email}
                        </div>
                      </div>

                      <div className="py-1">
                        <Link
                          href="/settings/account"
                          onClick={() => setUserMenuOpen(false)}
                          className="flex items-center gap-2 px-3 py-2 text-slate-700 hover:bg-slate-50 font-medium"
                        >
                          <Settings className="w-3.5 h-3.5 text-slate-400" />
                          <span>{t.accountSettings}</span>
                        </Link>
                        <Link
                          href="/organizations"
                          onClick={() => setUserMenuOpen(false)}
                          className="flex items-center gap-2 px-3 py-2 text-slate-700 hover:bg-slate-50 font-medium"
                        >
                          <Building className="w-3.5 h-3.5 text-slate-400" />
                          <span>{t.myOrganizations}</span>
                        </Link>
                      </div>

                      <div className="border-t border-slate-100 pt-1 mt-1">
                        <button
                          type="button"
                          onClick={() => {
                            setUserMenuOpen(false);
                            logout();
                          }}
                          className="w-full flex items-center gap-2 px-3 py-2 text-rose-600 hover:bg-rose-50 font-semibold transition-colors"
                        >
                          <LogOut className="w-3.5 h-3.5" />
                          <span>{t.logout}</span>
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          </header>

          {/* Page Main Content Area */}
          <main className="flex-1 w-full min-w-0 p-4 sm:p-6 lg:p-8">
            {children}
          </main>
        </div>
      </div>

      {/* ======================================================== */}
      {/* MOBILE NAVIGATION DRAWER (Off-canvas below 1024px) */}
      {/* ======================================================== */}
      {user && mobileDrawerOpen && (
        <div
          className="fixed inset-0 z-50 lg:hidden bg-slate-900/60 backdrop-blur-xs transition-opacity"
          onClick={() => setMobileDrawerOpen(false)}
          aria-modal="true"
          role="dialog"
        >
          <div
            ref={mobileDrawerRef}
            className={`fixed top-0 bottom-0 ${
              isRTL ? "right-0" : "left-0"
            } w-72 max-w-[85vw] bg-[#0f172a] text-slate-200 shadow-2xl flex flex-col justify-between z-50`}
            onClick={(e) => e.stopPropagation()}
          >
            {/* Drawer Header */}
            <div>
              <div className="h-16 flex items-center justify-between px-5 border-b border-slate-800">
                <div className="flex items-center gap-2.5">
                  <div className="w-8 h-8 rounded-lg bg-blue-600 text-white flex items-center justify-center shadow-xs">
                    <Activity className="w-4 h-4" />
                  </div>
                  <span className="font-extrabold text-white text-base tracking-tight">
                    {t.brandName}
                  </span>
                </div>
                <button
                  type="button"
                  onClick={() => setMobileDrawerOpen(false)}
                  className="p-1.5 text-slate-400 hover:text-white rounded-lg hover:bg-slate-800 transition-colors"
                  aria-label={t.close}
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              {/* Active Workspace Banner */}
              <div className="p-3 border-b border-slate-800/80">
                <Link
                  href="/organizations"
                  onClick={() => setMobileDrawerOpen(false)}
                  className="flex items-center justify-between p-2 rounded-xl bg-slate-800/60 border border-slate-700/60 text-xs font-semibold text-slate-200"
                >
                  <div className="flex items-center gap-2 min-w-0">
                    <Building className="w-3.5 h-3.5 text-blue-400 shrink-0" />
                    <span className="truncate">{activeOrg?.name || t.activeOrg}</span>
                  </div>
                  {role && <Badge variant={role} />}
                </Link>
              </div>

              {/* Navigation Items */}
              <nav className="p-3 space-y-4 max-h-[calc(100vh-220px)] overflow-y-auto">
                {navGroups.map((group) => {
                  const visibleItems = group.items.filter((item) => {
                    if (!item.requiredRole) return true;
                    return role && item.requiredRole.includes(role);
                  });

                  if (visibleItems.length === 0) return null;

                  return (
                    <div key={group.groupKey} className="space-y-1">
                      <div className="px-3 mb-1 text-[10px] font-bold tracking-wider text-slate-400 uppercase">
                        {group.label}
                      </div>
                      {visibleItems.map((item) => {
                        const active = item.isActive(pathname);
                        const Icon = item.icon;

                        return (
                          <Link
                            key={item.href}
                            href={item.href}
                            onClick={() => setMobileDrawerOpen(false)}
                            aria-current={active ? "page" : undefined}
                            className={`flex items-center gap-3 px-3 py-2.5 rounded-xl text-xs font-semibold transition-all ${
                              active
                                ? "bg-blue-600 text-white shadow-sm"
                                : "text-slate-300 hover:text-white hover:bg-slate-800/60"
                            }`}
                          >
                            <Icon
                              className={`w-4 h-4 shrink-0 ${
                                active ? "text-white" : "text-slate-400"
                              }`}
                            />
                            <span>{item.label}</span>
                          </Link>
                        );
                      })}
                    </div>
                  );
                })}
              </nav>
            </div>

            {/* Drawer User Footer */}
            <div className="p-4 border-t border-slate-800 bg-slate-900/60 space-y-3">
              <div className="flex items-center gap-2.5 text-xs">
                <div className="w-7 h-7 rounded-full bg-blue-600 text-white flex items-center justify-center font-bold text-xs shrink-0">
                  {user.display_name?.charAt(0).toUpperCase() || "U"}
                </div>
                <div className="min-w-0">
                  <div className="font-semibold text-slate-200 truncate">{user.display_name}</div>
                  <div className="text-[10px] text-slate-400 truncate dir-ltr text-start">
                    {user.email}
                  </div>
                </div>
              </div>

              <button
                type="button"
                onClick={() => {
                  setMobileDrawerOpen(false);
                  logout();
                }}
                className="w-full flex items-center justify-center gap-2 px-3 py-2 text-xs font-semibold text-rose-400 bg-rose-500/10 hover:bg-rose-500/20 border border-rose-500/20 rounded-lg transition-colors"
              >
                <LogOut className="w-3.5 h-3.5" />
                <span>{t.logout}</span>
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
