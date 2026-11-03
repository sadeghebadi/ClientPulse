"use client";

import React from "react";
import Link from "next/link";
import { Activity, Shield, ExternalLink, Globe } from "lucide-react";
import { useI18n } from "@/lib/i18n";

export function MarketingFooter() {
  const { t, isRTL } = useI18n();
  const currentYear = new Date().getFullYear();

  const handleSmoothScroll = (e: React.MouseEvent<HTMLAnchorElement>, href: string) => {
    if (href.startsWith("#")) {
      e.preventDefault();
      const target = document.querySelector(href);
      if (target) {
        target.scrollIntoView({ behavior: "smooth" });
      }
    }
  };

  return (
    <footer className="bg-slate-950 text-slate-400 border-t border-slate-800 text-xs antialiased">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12 sm:py-16">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8 lg:gap-12 pb-12 border-b border-slate-800/80">
          {/* Column 1: Brand Info */}
          <div className="space-y-4">
            <Link href="/" className="flex items-center gap-2.5">
              <div className="w-8 h-8 rounded-lg bg-blue-600 text-white flex items-center justify-center shadow-md">
                <Activity className="w-4 h-4" />
              </div>
              <span className="font-extrabold text-white text-base tracking-tight">
                {t.brandName}
              </span>
            </Link>

            <p className="text-slate-400 leading-relaxed text-xs">
              {t.landingFooterDesc}
            </p>

            <div className="pt-1">
              <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-slate-900 border border-slate-800 text-[11px] font-mono text-slate-300">
                <Globe className="w-3.5 h-3.5 text-blue-400" />
                <span>mindu.ir</span>
              </span>
            </div>
          </div>

          {/* Column 2: Product Sections */}
          <div className="space-y-3">
            <div className="text-xs font-bold text-slate-200 uppercase tracking-wider">
              {t.landingFooterProduct}
            </div>
            <ul className="space-y-2">
              <li>
                <a
                  href="#features"
                  onClick={(e) => handleSmoothScroll(e, "#features")}
                  className="hover:text-white transition-colors"
                >
                  {t.landingNavFeatures}
                </a>
              </li>
              <li>
                <a
                  href="#how-it-works"
                  onClick={(e) => handleSmoothScroll(e, "#how-it-works")}
                  className="hover:text-white transition-colors"
                >
                  {t.landingNavHowItWorks}
                </a>
              </li>
              <li>
                <a
                  href="#value"
                  onClick={(e) => handleSmoothScroll(e, "#value")}
                  className="hover:text-white transition-colors"
                >
                  {t.landingNavValue}
                </a>
              </li>
              <li>
                <a
                  href="#preview"
                  onClick={(e) => handleSmoothScroll(e, "#preview")}
                  className="hover:text-white transition-colors"
                >
                  {t.landingNavPreview}
                </a>
              </li>
            </ul>
          </div>

          {/* Column 3: Platform Access */}
          <div className="space-y-3">
            <div className="text-xs font-bold text-slate-200 uppercase tracking-wider">
              {t.landingFooterAuth}
            </div>
            <ul className="space-y-2">
              <li>
                <Link href="/login" className="hover:text-white transition-colors">
                  {t.login}
                </Link>
              </li>
              <li>
                <Link href="/signup" className="hover:text-white transition-colors">
                  {t.createAccount}
                </Link>
              </li>
              <li>
                <Link href="/dashboard" className="hover:text-white transition-colors">
                  {t.dashboard}
                </Link>
              </li>
            </ul>
          </div>

          {/* Column 4: Architecture & Security */}
          <div className="space-y-3">
            <div className="text-xs font-bold text-slate-200 uppercase tracking-wider">
              {t.landingFooterTrust}
            </div>
            <div className="space-y-2 text-slate-400 text-xs">
              <div className="flex items-center gap-1.5">
                <Shield className="w-3.5 h-3.5 text-emerald-400 shrink-0" />
                <span>Multi-Layer SSRF Guard</span>
              </div>
              <div className="flex items-center gap-1.5">
                <Shield className="w-3.5 h-3.5 text-blue-400 shrink-0" />
                <span>PostgreSQL Row-Level Security</span>
              </div>
              <div className="flex items-center gap-1.5">
                <Shield className="w-3.5 h-3.5 text-purple-400 shrink-0" />
                <span>Deterministic Queue Leases</span>
              </div>
            </div>
          </div>
        </div>

        {/* Bottom Bar */}
        <div className="pt-8 flex flex-col sm:flex-row items-center justify-between gap-4 text-[11px] text-slate-500">
          <div>
            © {currentYear} {t.brandName} · mindu.ir. {t.landingFooterRights}
          </div>
          <div className="flex items-center gap-4">
            <span className="text-slate-400 font-mono">B2B SaaS</span>
          </div>
        </div>
      </div>
    </footer>
  );
}

