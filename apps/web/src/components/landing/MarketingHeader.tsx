"use client";

import React, { useState, useEffect, useRef } from "react";
import Link from "next/link";
import { Activity, Languages, LogIn, UserPlus, Menu, X, ArrowRight, ArrowLeft, LayoutDashboard } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { useAuth } from "@/lib/auth-context";

export function MarketingHeader() {
  const { t, locale, toggleLocale, isRTL } = useI18n();
  const { user, loading } = useAuth();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const mobileTriggerRef = useRef<HTMLButtonElement>(null);
  const mobileDrawerRef = useRef<HTMLDivElement>(null);

  // Close mobile drawer on Escape and lock body scroll
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape" && mobileMenuOpen) {
        setMobileMenuOpen(false);
        mobileTriggerRef.current?.focus();
      }
    }

    if (mobileMenuOpen) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      document.body.style.overflow = "";
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [mobileMenuOpen]);

  const navLinks = [
    { href: "#features", label: t.landingNavFeatures },
    { href: "#how-it-works", label: t.landingNavHowItWorks },
    { href: "#value", label: t.landingNavValue },
    { href: "#preview", label: t.landingNavPreview },
  ];

  const handleSmoothScroll = (e: React.MouseEvent<HTMLAnchorElement>, href: string) => {
    if (href.startsWith("#")) {
      e.preventDefault();
      const target = document.querySelector(href);
      if (target) {
        target.scrollIntoView({ behavior: "smooth" });
        setMobileMenuOpen(false);
      }
    }
  };

  return (
    <header className="sticky top-0 z-40 w-full bg-white/90 backdrop-blur-md border-b border-slate-200/80 transition-colors">
      <div className="max-w-7xl mx-auto px-3 sm:px-6 lg:px-8 h-16 sm:h-20 flex items-center justify-between gap-2 sm:gap-4">
        {/* Brand Logo & Wordmark */}
        <Link href="/" className="flex items-center gap-2 sm:gap-3 group shrink-0 min-w-0">
          <div className="w-8 h-8 sm:w-10 sm:h-10 rounded-xl bg-blue-600 text-white flex items-center justify-center shadow-md group-hover:bg-blue-700 transition-colors shrink-0">
            <Activity className="w-4 h-4 sm:w-5 sm:h-5" />
          </div>
          <div className="flex flex-col min-w-0">
            <div className="flex items-center gap-1.5 sm:gap-2">
              <span className="font-extrabold text-slate-900 tracking-tight text-base sm:text-xl truncate">
                {t.brandName}
              </span>
              <span className="hidden sm:inline-block px-1.5 py-0.5 text-[10px] font-bold text-blue-700 bg-blue-50 border border-blue-200 rounded-md">
                mindu.ir
              </span>
            </div>
            <span className="hidden md:inline-block text-[11px] font-medium text-slate-500 -mt-0.5">
              {t.tagline}
            </span>
          </div>
        </Link>

        {/* Desktop Anchor Navigation */}
        <nav className="hidden lg:flex items-center gap-1 xl:gap-2 text-xs font-semibold text-slate-600">
          {navLinks.map((link) => (
            <a
              key={link.href}
              href={link.href}
              onClick={(e) => handleSmoothScroll(e, link.href)}
              className="px-3 py-2 rounded-lg hover:text-blue-600 hover:bg-slate-50 transition-colors"
            >
              {link.label}
            </a>
          ))}
        </nav>

        {/* Action Controls & Language Switcher */}
        <div className="flex items-center gap-1.5 sm:gap-3 shrink-0">
          {/* Language Switcher */}
          <button
            type="button"
            onClick={toggleLocale}
            className="inline-flex items-center gap-1 px-2 sm:px-3 py-1.5 text-xs font-semibold text-slate-700 bg-white hover:bg-slate-50 border border-slate-200 rounded-lg shadow-2xs transition-colors shrink-0"
            title="Toggle Language / تغییر زبان"
          >
            <Languages className="w-3.5 h-3.5 text-slate-500 shrink-0" />
            <span className="hidden sm:inline">{locale === "en" ? "فارسی" : "English"}</span>
            <span className="sm:hidden text-[11px] font-bold">{locale === "en" ? "FA" : "EN"}</span>
          </button>

          {/* Conditional Auth CTAs */}
          {!loading && user ? (
            <Link
              href="/dashboard"
              className="inline-flex items-center gap-1.5 px-3 py-1.5 sm:px-3.5 sm:py-2 text-xs font-bold text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm transition-colors shrink-0"
            >
              <LayoutDashboard className="w-3.5 h-3.5" />
              <span>{t.landingNavDashboard}</span>
            </Link>
          ) : (
            <>
              <Link
                href="/login"
                className="hidden sm:inline-flex items-center gap-1.5 px-3.5 py-2 text-xs font-semibold text-slate-700 hover:text-slate-900 hover:bg-slate-100 rounded-lg transition-colors"
              >
                <LogIn className="w-3.5 h-3.5 text-slate-500" />
                <span>{t.landingNavSignIn}</span>
              </Link>
              <Link
                href="/signup"
                className="inline-flex items-center gap-1 px-2.5 sm:px-3.5 py-1.5 sm:py-2 text-xs font-bold text-white bg-blue-600 hover:bg-blue-700 rounded-lg shadow-sm hover:shadow transition-all shrink-0"
              >
                <UserPlus className="w-3.5 h-3.5 shrink-0" />
                <span>{t.landingNavSignUp}</span>
              </Link>
            </>
          )}

          {/* Mobile Drawer Hamburger Button */}
          <button
            ref={mobileTriggerRef}
            type="button"
            onClick={() => setMobileMenuOpen(true)}
            className="lg:hidden p-1.5 sm:p-2 text-slate-600 hover:text-slate-900 hover:bg-slate-100 rounded-lg transition-colors shrink-0"
            aria-label={t.menu}
            aria-expanded={mobileMenuOpen}
          >
            <Menu className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* Mobile Navigation Drawer */}
      {mobileMenuOpen && (
        <div
          className="fixed inset-0 z-50 lg:hidden bg-slate-900/60 backdrop-blur-xs transition-opacity"
          onClick={() => setMobileMenuOpen(false)}
          aria-modal="true"
          role="dialog"
        >
          <div
            ref={mobileDrawerRef}
            className={`fixed top-0 bottom-0 ${
              isRTL ? "right-0" : "left-0"
            } w-72 max-w-[85vw] bg-white text-slate-900 shadow-2xl flex flex-col justify-between z-50`}
            onClick={(e) => e.stopPropagation()}
          >
            <div>
              {/* Drawer Header */}
              <div className="h-16 flex items-center justify-between px-5 border-b border-slate-200">
                <div className="flex items-center gap-2.5">
                  <div className="w-8 h-8 rounded-lg bg-blue-600 text-white flex items-center justify-center shadow-xs">
                    <Activity className="w-4 h-4" />
                  </div>
                  <span className="font-extrabold text-slate-900 text-base tracking-tight">
                    {t.brandName}
                  </span>
                </div>
                <button
                  type="button"
                  onClick={() => setMobileMenuOpen(false)}
                  className="p-1.5 text-slate-500 hover:text-slate-900 rounded-lg hover:bg-slate-100 transition-colors"
                  aria-label={t.close}
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              {/* Navigation Links */}
              <nav className="p-4 space-y-1">
                {navLinks.map((link) => (
                  <a
                    key={link.href}
                    href={link.href}
                    onClick={(e) => handleSmoothScroll(e, link.href)}
                    className="flex items-center justify-between px-3 py-2.5 rounded-xl text-xs font-semibold text-slate-700 hover:text-blue-600 hover:bg-slate-50 transition-colors"
                  >
                    <span>{link.label}</span>
                    {isRTL ? (
                      <ArrowLeft className="w-3.5 h-3.5 text-slate-400" />
                    ) : (
                      <ArrowRight className="w-3.5 h-3.5 text-slate-400" />
                    )}
                  </a>
                ))}
              </nav>
            </div>

            {/* Mobile Footer CTAs */}
            <div className="p-4 border-t border-slate-200 bg-slate-50/80 space-y-2.5">
              {!loading && user ? (
                <Link
                  href="/dashboard"
                  onClick={() => setMobileMenuOpen(false)}
                  className="w-full flex items-center justify-center gap-2 px-3 py-2.5 text-xs font-bold text-white bg-blue-600 hover:bg-blue-700 rounded-xl shadow-sm transition-colors"
                >
                  <LayoutDashboard className="w-4 h-4" />
                  <span>{t.landingNavDashboard}</span>
                </Link>
              ) : (
                <>
                  <Link
                    href="/signup"
                    onClick={() => setMobileMenuOpen(false)}
                    className="w-full flex items-center justify-center gap-2 px-3 py-2.5 text-xs font-bold text-white bg-blue-600 hover:bg-blue-700 rounded-xl shadow-sm transition-colors"
                  >
                    <UserPlus className="w-4 h-4" />
                    <span>{t.landingNavSignUp}</span>
                  </Link>
                  <Link
                    href="/login"
                    onClick={() => setMobileMenuOpen(false)}
                    className="w-full flex items-center justify-center gap-2 px-3 py-2.5 text-xs font-semibold text-slate-700 hover:text-slate-900 bg-white border border-slate-200 rounded-xl shadow-2xs transition-colors"
                  >
                    <LogIn className="w-4 h-4 text-slate-500" />
                    <span>{t.landingNavSignIn}</span>
                  </Link>
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </header>
  );
}
