"use client";

import React, { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { MarketingHeader } from "@/components/landing/MarketingHeader";
import { HeroSection } from "@/components/landing/HeroSection";
import { MonitoringLifecycleStory } from "@/components/landing/MonitoringLifecycleStory";
import { CapabilitiesSection } from "@/components/landing/CapabilitiesSection";
import { HowItWorksSection } from "@/components/landing/HowItWorksSection";
import { InteractiveProductDemo } from "@/components/landing/InteractiveProductDemo";
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
    <div className="min-h-screen flex flex-col bg-[#070b14] text-slate-100 antialiased selection:bg-blue-600 selection:text-white overflow-x-hidden">
      {/* Public Marketing Header */}
      <MarketingHeader />

      {/* Main Cinematic Marketing Narrative Sections */}
      <main className="flex-1 w-full">
        {/* 1. Cinematic Hero with 3D Node Mesh & Dynamic Counters */}
        <HeroSection />

        {/* 2. Scroll-Driven Monitoring Lifecycle Story (5 Choreographed Scenes) */}
        <MonitoringLifecycleStory />

        {/* 3. Core Implemented Capabilities with 3D Tilt Micro-Interactions */}
        <CapabilitiesSection />

        {/* 4. Interactive Live Monitoring Sandbox (Outage & Recovery Simulator) */}
        <InteractiveProductDemo />

        {/* 5. How It Works - Agency Operational Workflow */}
        <HowItWorksSection />

        {/* 6. Agency Value Proposition */}
        <AgencyValueSection />

        {/* 7. Product Preview Showcase */}
        <ProductPreviewSection />

        {/* 8. Final Call to Action */}
        <FinalCtaSection />
      </main>

      {/* Public Marketing Footer */}
      <MarketingFooter />
    </div>
  );
}
