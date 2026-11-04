"use client";

import React, { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import { MarketingHeader } from "@/components/landing/MarketingHeader";
import { HeroSection } from "@/components/landing/HeroSection";
import { MonitoringLifecycleStory } from "@/components/landing/MonitoringLifecycleStory";
import { InteractiveProductDemo } from "@/components/landing/InteractiveProductDemo";
import { WebAndApiObservabilitySection } from "@/components/landing/WebAndApiObservabilitySection";
import { HeartbeatPulseSection } from "@/components/landing/HeartbeatPulseSection";
import { IncidentCommandSection } from "@/components/landing/IncidentCommandSection";
import { HistoricalReportsAnalyticsSection } from "@/components/landing/HistoricalReportsAnalyticsSection";
import { AgencyCareRetainerSection } from "@/components/landing/AgencyCareRetainerSection";
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
    <div className="min-h-screen flex flex-col bg-[#050811] text-slate-100 antialiased selection:bg-blue-600 selection:text-white overflow-x-clip">
      {/* Public Marketing Header */}
      <MarketingHeader />

      {/* Main Award-Level Marketing Narrative & Interactive Experience */}
      <main className="flex-1 w-full">
        {/* 1. Immersive 3D Spatial Observability Hero */}
        <HeroSection />

        {/* 2. Scroll-Driven Cinematic Reliability Story (6 Choreographed Scenes) */}
        <MonitoringLifecycleStory />

        {/* 3. Live Interactive Monitoring Playground (Outage & Recovery Sandbox) */}
        <InteractiveProductDemo />

        {/* 4. Website & API Observability Engine (Interactive Request/Response Studio) */}
        <WebAndApiObservabilitySection />

        {/* 5. Cron & Worker Heartbeat Pulse Chamber (EKG-style Oscillogram) */}
        <HeartbeatPulseSection />

        {/* 6. Tactical Incident Command & Resolution Timeline Desk */}
        <IncidentCommandSection />

        {/* 7. Historical Analytics & 90-Day Reliability Heatmap */}
        <HistoricalReportsAnalyticsSection />

        {/* 8. Dedicated Team & Agency Retainer Capabilities (Secondary Use Case) */}
        <AgencyCareRetainerSection />

        {/* 9. Authentic High-Fidelity Product Preview Showcase */}
        <ProductPreviewSection />

        {/* 10. Cinematic Horizon Final Call to Action */}
        <FinalCtaSection />
      </main>

      {/* Public Marketing Footer */}
      <MarketingFooter />
    </div>
  );
}
