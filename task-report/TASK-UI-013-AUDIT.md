# TASK-UI-013 — Scroll Experience & Cinematic Narrative Audit

**Project:** ClientPulse (`mindu.ir`)  
**Audit Date:** November 4, 2026  
**Auditor:** Advanced Agentic UI/Motion Engineer  
**Scope:** Landing page scroll storytelling, Three.js spatial mesh, GSAP choreography, and section transitions  

---

## 1. Executive Summary

A comprehensive forensic audit was conducted on the current landing page implementation (`apps/web/src/components/landing/`), evaluating the visual storytelling quality against world-class creative benchmarks (Lusion, Apple AirPods Pro, Locomotive, Active Theory).

**Verdict:** While individual static components (3D Hero mesh card, interactive playground, bespoke section cards) possess high visual polish in isolation, the **scrolling experience is fundamentally non-cinematic and disjointed**. The visitor does not experience a continuous digital world; instead, they scroll past a self-contained 3D box in the Hero, followed by an unpinned 2D card that abruptly switches text and color themes via React state.

---

## 2. Forensic Findings & Root Cause Analysis

### 2.1 Hero Section (`HeroNetworkCanvas.tsx` & `HeroSection.tsx`)
- **Isolation:** The Three.js canvas is trapped inside a static card (`h-[520px] sm:h-[620px] lg:h-[720px]`).
- **Disconnection from Scroll:** The 3D animation loop is driven exclusively by continuous `requestAnimationFrame` clock ticks and subtle pointer coordinates (`mouseRef`). There is **zero connection between scroll position and 3D camera, geometry, or lighting**.
- **Abrupt Exit:** When the user scrolls down, the entire 3D canvas physically scrolls out of view and is suspended by an `IntersectionObserver`.
- **Missed Opportunity:** The 12-node observability cosmos simply disappears rather than morphing or flying the camera into the monitoring narrative below.

### 2.2 Story Section (`MonitoringLifecycleStory.tsx`)
- **Unpinned and Ephemeral:** Although GSAP `ScrollTrigger` is registered, it was set with `pin: false` and a simple `onUpdate` callback:
  ```typescript
  const newScene = Math.min(Math.floor(progress * scenes.length), scenes.length - 1);
  setActiveScene(newScene);
  ```
- **React State Thrashing:** Every scroll increment triggers React re-renders to swap strings and background gradient strings (`glowColor`, `borderColor`).
- **No Spatial 3D Environment:** The story section contains **zero 3D WebGL context**. It is merely a 2D card containing an array of 10 static `<div>` bars whose CSS heights and background colors change with a 500ms transition.
- **Physical Drift:** Because `pin: false`, as the visitor scrolls down, the card scrolls upward and out of viewport while text flickers between phases. By Phase 4 or 5, the card is scrolled partially or completely offscreen.
- **Absence of Real Visual Transformations:**
  - *Outage (Scene 04):* Instead of a dramatic network severance, shockwave propagation, or camera zoom, the bars turn red and the text says "502 Bad Gateway".
  - *Recovery (Scene 06):* Instead of network reconnection and light beam rekindling, bars turn green.
  - *Transitions:* There are no spatial transitions, camera trajectories, or geometry morphs—only CSS property interpolation.

### 2.3 Section-to-Section Rhythm & Scroll Continuity
- Section boundaries are standard vertical stacks with static padding (`py-20 sm:py-32`).
- No continuous visual anchors (e.g. data threads, probe paths, ambient atmospheric shifts) connect the Hero to the Story or the Story to the Interactive Playground.

---

## 3. Classification: Decorative vs. Scroll-Driven Elements

| Component / Element | Current Mechanism | Classification | Required Target Mechanism |
|---|---|---|---|
| **Hero 3D Mesh & Rings** | `Date.now()` clock in `rAF` | **Purely Decorative** | Scroll-driven entry & camera zoom transitioning into Story |
| **Hero Node Packets** | Linear offset increment | **Purely Decorative** | Dynamic packet speed and path divergence reacting to scroll |
| **Story Phase Stepper** | ScrollTrigger progress -> `setActiveScene` | **Weakly Driven** (Discrete React state toggle) | Continuous scrubbed timeline with micro-phase scrub |
| **Story Scene Visualizer** | 2D CSS `<div>` bars & colors | **Pseudo-Interactive** (2D CSS) | **Full 3D WebGL Scene** with spatial nodes, dynamic spline beams, shockwaves & HUD |
| **Outage Shockwave** | Non-existent (CSS ping dot) | **Absent** | Physical Three.js shockwave ring expanding across fractured nodes |
| **Incident HUD Assembly** | CSS opacity transition | **Absent** | Spatially pinned 3D vector brackets & floating telemetry cards |
| **Network Recovery** | Static CSS color reset | **Absent** | Physical spline re-connection, particle surge, and camera pan |

---

## 4. Architectural Plan for TASK-UI-013

1. **Unified Pinned Spatial Story Engine:**
   - Create a dedicated full-viewport 3D WebGL canvas (`CinematicStoryCanvas.tsx`) pinned via GSAP `ScrollTrigger` (`pin: true`, `scrub: 1`).
   - Anchor the pinned scroll container to provide ~4000px of smooth, scrubbing scroll real estate.
   - Maintain continuous camera translation and rotation along a 3D CatmullRom spline through all 6 scenes.

2. **Choreographed 6-Scene Narrative Transformations:**
   - **Scene 01 (Connected):** Deep wide perspective into a 16-node interconnected constellation; glowing CatmullRom splines; central verification core pulses cyan/emerald.
   - **Scene 02 (Active Probing):** Camera descends closer to edge paths; active probe packets orbit across splines; real-time telemetry pills emerge at node anchors.
   - **Scene 03 (Degradation):** Camera pans toward the target node (`storefront.production`); spline vibrations and jitter shaders; amber lighting storm begins; packet flows become turbulent.
   - **Scene 04 (Total Outage):** Catastrophic visual moment; target spline physically fractures into dissipated spark particles; expanding red shockwave ripples across surrounding nodes; camera performs a dramatic orbit around the severed node; spatial `502 Bad Gateway` alert HUD locks into 3D coordinates.
   - **Scene 05 (Incident Awareness):** Camera pulls back into tactical command isometric angle; incident diagnostics envelope (`#INC-902`) forms spatial brackets; dispatch telemetry vectors illuminate toward notification channels.
   - **Scene 06 (Recovery):** Sapphire-emerald restorative wave propagates; severed spline reconstructs; light packets surge in synchrony; all nodes return to serene emerald harmony; camera swoops through the restored network into the next section.

3. **Hero-to-Story Bridge:**
   - Seamlessly connect the visual aesthetic of the Hero mesh into Scene 01. The transition provides visual continuity so the visitor feels they are descending into the very network they observed in the Hero.

4. **Performance & Robustness:**
   - Single shared or cleanly managed WebGL contexts with zero memory leaks.
   - Render loop active only when in viewport; throttled on low power.
   - Responsive camera FOV and node layout adjustments for mobile (360px - 768px).
   - Full RTL and Persian typography compatibility preserved.
   - Accessible fallback when `prefers-reduced-motion` is active.

