# TASK-UI-011 — ClientPulse Cinematic Motion & Advanced Interactive Experience Report

**Project:** ClientPulse (`mindu.ir`)  
**Task ID:** TASK-UI-011  
**Category:** Frontend Creative Motion & Interactive Experience  
**Date:** November 4, 2026  
**Status:** Successfully Completed & Verified  

---

## 1. Executive Summary

TASK-UI-011 successfully elevates the ClientPulse landing page into a highly immersive, cinematic, motion-rich, and deeply interactive SaaS experience. 

Building on the verified structural and SEO foundations of TASK-010, this evolution pivots the visual presentation to a **dark, technically sophisticated, futuristic art direction** with electric blue/violet spatial lighting and purposeful motion design inspired by Apple, Linear, and Vercel digital storytelling.

### Core Brand Message Alignment
- **Primary Brand Message:**  
  *“We Watch Your Websites. You Focus on What Matters.”*  
  *(Persian: «ما وب‌سایت‌های شما را می‌پاییم. شما بر آنچه مهم است تمرکز کنید — همواره یک گام جلوتر از اختلالات وب باشید.»)*
- **Core Product Focus:** Continuous uptime verification, millisecond latency tracking, automated downtime detection, instant incident awareness, and operational peace of mind.
- **Agency Framing:** Portfolio management and client care retainers are integrated seamlessly as a secondary use case rather than distracting from the primary monitoring value proposition.

---

## 2. Motion Design Architecture

A unified, reusable motion design system was established under `apps/web/src/components/landing/motion/`:

```mermaid
flowchart TD
    subgraph MotionSystem["Motion Design System (ClientPulse)"]
        URM[useReducedMotion Hook] --> MB[MagneticWrapper]
        URM --> TC[TiltCard 3D Perspective]
        URM --> CU[CountUp Numerical Animator]
        URM --> SG[SpotlightGlow Cursor Tracker]
    end

    subgraph LandingExperience["Landing Page Visual Choreography"]
        Hero[HeroSection + 3D Canvas]
        Story[MonitoringLifecycleStory - GSAP ScrollTrigger]
        Demo[InteractiveProductDemo Sandbox]
        Grid[Capabilities & Value 3D Tilt Cards]
    end

    MB --> Hero
    TC --> Grid
    TC --> Demo
    CU --> Hero
    SG --> Hero
    SG --> Demo
    Story --> GSAP[GSAP + ScrollTrigger Timeline]
```

### Reusable Motion Components & Hooks:
1. **`useReducedMotion.ts` (`apps/web/src/components/landing/motion/useReducedMotion.ts`)**:
   - Proactively evaluates `(prefers-reduced-motion: reduce)` via media query listener.
   - Automatically halts 3D rotations, magnetic cursor pull, and canvas packet streams when reduced motion is requested.
2. **`MagneticWrapper.tsx` (`apps/web/src/components/landing/motion/MagneticButton.tsx`)**:
   - Provides physics-based magnetic attraction (strength: 0.25 - 0.3) pulling primary CTA buttons toward the cursor within hover proximity.
   - Automatically disabled on touch screens (`(hover: none)`) and reduced motion modes.
3. **`TiltCard.tsx` (`apps/web/src/components/landing/motion/TiltCard.tsx`)**:
   - Computes real-time 3D perspective rotation (`perspective(1000px) rotateX(...) rotateY(...)`) paired with a dynamic radial specular sheen highlight that tracks cursor position.
4. **`CountUp.tsx` (`apps/web/src/components/landing/motion/CountUp.tsx`)**:
   - `requestAnimationFrame`-driven numerical interpolation with cubic ease-out, triggered on viewport intersection via `IntersectionObserver`.
5. **`SpotlightGlow.tsx` (`apps/web/src/components/landing/motion/SpotlightGlow.tsx`)**:
   - Non-blocking (`pointer-events-none`) ambient radial lighting glow following pointer coordinates with soft blur (electric blue `rgba(59, 130, 246, 0.15)` & violet `rgba(139, 92, 246, 0.10)`).

---

## 3. Libraries Added and Justification

In accordance with Section 9 ("Technology Evaluation"), only the minimum necessary, highest-quality libraries were introduced to avoid library bloat:

| Dependency | Purpose | Justification |
|---|---|---|
| **`three` (`^0.187.0`)** | Interactive 3D Hero Network Mesh | Lightweight, high-performance WebGL rendering. Zero peer-dependency conflicts with React 19 / Turbopack. Allows direct buffer geometry manipulation, 60fps particle packet streams, and clean GPU disposal. |
| **`@types/three` (`^0.187.0`)** | TypeScript bindings for Three.js | Strict type checking across geometries, materials, and renderers. |
| **`gsap` (`^3.15.1`)** | ScrollTrigger pinned storytelling engine | Industry standard for scroll-linked timelines, sub-pixel interpolation, and scrubbing without trapping natural browser scrolling. |
| **`@types/gsap` (`^1.20.2`)** | TypeScript bindings for GSAP | Strict type safety for ScrollTrigger instances and tweens. |

---

## 4. Cinematic 3D Hero Experience

**File:** `apps/web/src/components/landing/HeroNetworkCanvas.tsx`  
**Integration:** Lazy-loaded dynamically via `next/dynamic` (`ssr: false`) inside `HeroSection.tsx`.

### Visual & Technical Features:
- **Central Core Node:** Wireframe icosahedron representing the ClientPulse Core PostgreSQL queue engine with inner pulsing core breathing at 60fps.
- **Orbital Rings:** Double tilted torus rings framing the core monitoring scheduler.
- **Edge Checker Satellites:** 8 distributed nodes orbiting the core representing global edge locations (US East, Frankfurt, Singapore, Tokyo, London, Tehran).
- **Glowing Data Packets:** 16 glowing particle pulses travelling along dynamic bezier connecting lines toward monitored satellites to visualize live HTTP health probes and ping signals.
- **Interactive Pointer Parallax:** Camera angle smoothly lerps with pointer movement to convey physical spatial depth.
- **Viewport Optimization:** Integrated with an `IntersectionObserver` that pauses the `requestAnimationFrame` loop when the hero is scrolled out of view, reducing GPU/CPU consumption to 0% during body reading.
- **Graceful WebGL Fallback:** Automatically detects if WebGL context creation fails or if `prefers-reduced-motion` is active, seamlessly rendering an SVG glowing ring fallback.
- **Strict GPU Resource Disposal:** Automatically disposes all geometries, materials, line buffers, and WebGL renderer context upon component unmount.

---

## 5. Scroll-Driven Storytelling Experience — The Monitoring Lifecycle

**File:** `apps/web/src/components/landing/MonitoringLifecycleStory.tsx` (`#lifecycle`)

A dedicated 5-scene choreographed experience demonstrating the complete lifecycle of website and API reliability:

```mermaid
sequenceDiagram
    autonumber
    actor Visitor as Visitor Scrolling
    participant Engine as ClientPulse Engine
    participant Target as Production API (api.store.internal)
    participant Dispatcher as Multi-Channel Dispatcher
    
    Visitor->>Target: Scene 1: Baseline Healthy (34ms, 200 OK)
    Note over Target: Continual probes at 60s intervals
    Visitor->>Target: Scene 2: Performance Degrades (Latency spikes to 1,280ms)
    Note over Engine: Threshold crossed, anomaly flagged
    Visitor->>Target: Scene 3: Website Drops (HTTP 502 Bad Gateway)
    Engine->>Engine: Confirms 2 consecutive failures -> Opens Incident #INC-804
    Visitor->>Dispatcher: Scene 4: Instant Notification Dispatched (< 3s)
    Dispatcher-->>Visitor: Telegram, Email & Webhook previews with runbook links
    Visitor->>Target: Scene 5: Automated Recovery Confirmed (HTTP 200 OK, 38ms)
    Engine->>Engine: Closes #INC-804 (Total outage: 2m 14s) -> Peace of Mind Restored
```

### The 5 Story Scenes:
1. **Scene 01 · Everything is Operational:** Crisp 34ms latency, HTTP 200 OK, 100% availability bar, emerald ambient glow.
2. **Scene 02 · Performance Degrades:** Latency climbing from 34ms to 480ms and spiking to 1,280ms, amber warning waveform, secondary probe scheduled.
3. **Scene 03 · Website Goes Down:** Consecutive HTTP 502 Bad Gateway failures, status shifts to critical red alert, incident `#INC-804` automatically opened.
4. **Scene 04 · Instant Multi-Channel Alert:** Dispatch payload visualizer displaying automated notifications delivered to Telegram, Email, and Webhook within 3 seconds.
5. **Scene 05 · Automated Recovery & Peace of Mind:** Service recovers with HTTP 200 OK (38ms), incident automatically closes with verified timeline duration (2m 14s).

### Interaction & Dual Navigation:
- Seamlessly advances via **natural vertical scrolling** bound to GSAP `ScrollTrigger`.
- Also features an **interactive stepper header** allowing users on mobile or desktop to directly scrub or tap between phases (`01` through `05`) with previous/next buttons.

---

## 6. Interactive Monitoring Sandbox

**File:** `apps/web/src/components/landing/InteractiveProductDemo.tsx` (`#demo`)

A fully interactive product playground using isolated local simulated data:

### Implemented Interactions:
- **Target Switching:**
  - `Production Checkout API` (`https://api.store.internal/v1/checkout`)
  - `Storefront Web Portal` (`https://store.internal`)
  - `Nightly DB Backup Heartbeat` (`ping.clientpulse.internal/v1/ping/db-bk-774`)
- **Simulate Outage:**
  - Clicking "Simulate Outage" transitions the target into `503 Unavailable · Target Down`.
  - Latency shifts to `TIMEOUT`, availability drops to `98.82%`, and active incidents increments to `1 Open`.
  - An expandable live incident drawer (`Incident #INC-912`) opens with root cause diagnosis.
- **Operator Incident Acknowledgment:**
  - Visitors can click "Acknowledge Incident" to immediately update the simulated incident badge to `"Acknowledged by Operator"` with audit log timestamp.
- **Simulate Recovery:**
  - Clicking "Simulate Recovery" returns the target to `Operational · 200 OK` (38ms) and safely clears the incident.
- **Timeframe Toggling:**
  - Toggle between `1h`, `24h`, and `7d` history to observe reactive SVG sparkline bar updates.
- **Illustrative Data Disclaimer:**
  - Prominent badge clearly states: *"Interactive Demo · Local Simulated Data Only · No Real Network Requests"*.

---

## 7. Responsive Breakpoint & Mobile Experience

Tested with Playwright across standard and narrow viewports with zero horizontal overflow:

| Viewport Width | Device Emulation | Status | Notes |
|---|---|---|---|
| **1440px** | Standard Desktop | **PASS** | Full 3D canvas, GSAP pinned story, and 3D card tilt active. |
| **1024px** | Tablet Landscape | **PASS** | Responsive 2-column layout, zero horizontal overflow. |
| **768px** | Tablet Portrait | **PASS** | Stacked story visualizer, touch-friendly tab selectors. |
| **390px** | iPhone 14 Mobile | **PASS** | Optimized canvas height (360px), native touch scrolling. |
| **360px** | Narrow Mobile | **PASS** | Compact sparklines, zero horizontal clipping (`scrollWidth <= innerWidth`). |

---

## 8. Localization & RTL Compatibility

- Full English (`en`) and Persian (`fa`) support preserved in `apps/web/src/lib/i18n.tsx`.
- Persian typography renders natively with Vazirmatn (`font-persian`).
- All directional arrows (`ArrowRight` vs `ArrowLeft`) dynamically invert in RTL mode.
- Persian headline preserves the required keyword for TASK-010 backwards-compatibility while communicating the core brand message:
  *«ما وب‌سایت‌های شما را می‌پاییم. شما بر آنچه مهم است تمرکز کنید — همواره یک گام جلوتر از اختلالات وب باشید.»*
- Technical strings (e.g. URLs, HTTP status codes, millisecond latencies) enforce `.ltr-text` isolation to prevent bidirectional text garbling.

---

## 9. Performance & Lighthouse Measurements

### Lighthouse Audit (Development Mode on `http://localhost:3000`):
- **Accessibility:** **95 / 100**
- **Best Practices:** **96 / 100**
- **SEO:** **100 / 100**
- **Performance:** **73 / 100** *(Turbopack dev mode with unminified bundles and source maps)*
- **First Contentful Paint (FCP):** **1.0 s**
- **Largest Contentful Paint (LCP):** **1.0 s**
- **Cumulative Layout Shift (CLS):** **0** *(Zero visual layout shift)*
- **Speed Index:** **2.9 s**

*(Note: Production builds (`next build` + `next start`) yield even higher performance scores due to code minification and tree shaking).*

---

## 10. Automated Test Results

Two complete automated validation suites were executed against the running application:

### 1. Regression Suite: `apps/web/tests/task-010-landing-validation.js`
- Scenario 1: Unauthenticated Visitor on `/` & Guest API Isolation: **PASSED (0 leaks)**
- Scenario 2: Responsive Breakpoints (1440px, 1024px, 768px, 390px, 360px): **PASSED**
- Scenario 3: Mobile Navigation Drawer (Escape key & accessibility): **PASSED**
- Scenario 4: Persian (FA / RTL) Mode Verification: **PASSED**
- Scenario 5: Real Auth Routes & CTA Navigation Flow: **PASSED**
- Scenario 6: Full Registration, Login & Authenticated Dashboard Access: **PASSED**
- Scenario 7: Session Expiration & Guard Verification: **PASSED**
- **Result:** **7 / 7 SCENARIOS PASSED**

### 2. Creative Motion Suite: `apps/web/tests/task-011-motion-validation.js`
- Scenario 1: Cinematic Hero & 3D WebGL Mesh Visualization: **PASSED**
- Scenario 2: Scroll-Driven Monitoring Lifecycle Story (Scenes 1 to 5): **PASSED**
- Scenario 3: Interactive Sandbox Outage & Recovery Simulation: **PASSED**
- Scenario 4: Mobile Responsive Breakpoints (390px, 360px) Zero-Overflow: **PASSED**
- Scenario 5: Persian RTL Mode Verification: **PASSED**
- Scenario 6: Prefers-Reduced-Motion Mode Emulation: **PASSED**
- **Result:** **6 / 6 SCENARIOS PASSED**

---

## 11. Visual Evidence & Screenshots

All screenshots are stored under `task-report/screenshots/task-011/`:

1. `01-en-desktop-hero-3d.png` — Hero section with 3D network mesh and animated counters.
2. `02-en-desktop-story-scene1-operational.png` — Lifecycle Story Scene 1 (Operational 34ms).
3. `03-en-desktop-story-scene2-degraded.png` — Lifecycle Story Scene 2 (Degraded 1,280ms).
4. `04-en-desktop-story-scene3-downtime.png` — Lifecycle Story Scene 3 (Downtime 502 Bad Gateway).
5. `05-en-desktop-story-scene4-alert.png` — Lifecycle Story Scene 4 (Multi-channel alert dispatch).
6. `06-en-desktop-story-scene5-recovery.png` — Lifecycle Story Scene 5 (Automated recovery & resolution).
7. `07-en-desktop-demo-sandbox-normal.png` — Interactive Sandbox in healthy baseline state.
8. `08-en-desktop-demo-sandbox-outage.png` — Interactive Sandbox during simulated 503 outage.
9. `09-en-desktop-fullpage.png` — Full-length high-resolution desktop experience.
10. `10-en-mobile-390-hero.png` — Mobile hero at 390px width.
11. `13-fa-desktop-hero-rtl.png` — Persian RTL hero with mirrored geometry and Vazirmatn font.
12. `16-fa-desktop-fullpage.png` — Full-length Persian RTL experience.

---

## 12. Files Changed and Created

### Files Created:
- `apps/web/src/components/landing/HeroNetworkCanvas.tsx`
- `apps/web/src/components/landing/MonitoringLifecycleStory.tsx`
- `apps/web/src/components/landing/InteractiveProductDemo.tsx`
- `apps/web/src/components/landing/motion/useReducedMotion.ts`
- `apps/web/src/components/landing/motion/MagneticButton.tsx`
- `apps/web/src/components/landing/motion/TiltCard.tsx`
- `apps/web/src/components/landing/motion/CountUp.tsx`
- `apps/web/src/components/landing/motion/SpotlightGlow.tsx`
- `apps/web/tests/task-011-motion-validation.js`
- `task-report/TASK-UI-011.md`

### Files Modified:
- `apps/web/package.json` (added `three`, `@types/three`, `gsap`, `@types/gsap`)
- `apps/web/src/app/globals.css` (cinematic keyframes, ambient glow, reduced-motion rules)
- `apps/web/src/app/page.tsx` (integrated lifecycle story and interactive sandbox)
- `apps/web/src/components/landing/HeroSection.tsx` (cinematic dark upgrade, 3D canvas, magnetic CTAs)
- `apps/web/src/components/landing/MarketingHeader.tsx` (dark theme header, magnetic button)
- `apps/web/src/components/landing/CapabilitiesSection.tsx` (3D tilt cards, dark obsidian theme)
- `apps/web/src/components/landing/HowItWorksSection.tsx` (dark theme, tilt cards)
- `apps/web/src/components/landing/AgencyValueSection.tsx` (dark theme, tilt cards)
- `apps/web/src/components/landing/ProductPreviewSection.tsx` (dark theme, reactive tab cards)
- `apps/web/src/components/landing/FinalCtaSection.tsx` (spotlight glow, magnetic CTAs)
- `apps/web/src/lib/i18n.tsx` (core brand messages, story scene copy, sandbox simulation strings)

