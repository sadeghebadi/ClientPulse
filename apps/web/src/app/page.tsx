"use client";

import React, { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { MarketingHeader } from "@/components/landing/MarketingHeader";
import { HeroSection } from "@/components/landing/HeroSection";
import { CapabilitiesSection } from "@/components/landing/CapabilitiesSection";
import { HowItWorksSection } from "@/components/landing/HowItWorksSection";
import { AgencyValueSection } from "@/components/landing/AgencyValueSection";
import { ProductPreviewSection } from "@/components/landing/ProductPreviewSection";
import { FinalCtaSection } from "@/components/landing/FinalCtaSection";
import { MarketingFooter } from "@/components/landing/MarketingFooter";

export default function MarketingLandingPage() {
  const router = useRouter();
  const { user, loading } = useAuth();

  // If user is already authenticated, smoothly transition to their protected dashboard
  useEffect(() => {
    if (!loading && user) {
      router.replace("/dashboard");
    }
  }, [user, loading, router]);

  return (
    <div className="min-h-screen flex flex-col bg-white text-slate-900 antialiased selection:bg-blue-600 selection:text-white">
      {/* Public Marketing Header */}
      <MarketingHeader />

      {/* Main Marketing Narrative Sections */}
      <main className="flex-1 w-full">
        <HeroSection />
        <CapabilitiesSection />
        <HowItWorksSection />
        <AgencyValueSection />
        <ProductPreviewSection />
        <FinalCtaSection />
      </main>

      {/* Public Marketing Footer */}
      <MarketingFooter />
    </div>
  );
}
