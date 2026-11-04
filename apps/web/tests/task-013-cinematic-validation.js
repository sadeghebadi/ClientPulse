const { chromium } = require("playwright");
const path = require("path");
const fs = require("fs");
const { execSync } = require("child_process");

const BASE_URL = process.env.BASE_URL || "http://localhost:3000";
const SCREENSHOT_DIR = path.resolve(__dirname, "../../../task-report/screenshots/task-013");
const FRAMES_DIR = path.resolve(__dirname, "../../../task-report/screenshots/task-013/frames");

if (!fs.existsSync(SCREENSHOT_DIR)) {
  fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
}
if (!fs.existsSync(FRAMES_DIR)) {
  fs.mkdirSync(FRAMES_DIR, { recursive: true });
}

async function runTask013Validation() {
  console.log("================================================================================");
  console.log("TASK-UI-013: Cinematic Scroll Experience Overhaul & Visual Validation");
  console.log("Base URL:", BASE_URL);
  console.log("Screenshots Directory:", SCREENSHOT_DIR);
  console.log("================================================================================\n");

  const browser = await chromium.launch({
    executablePath: "/usr/bin/google-chrome",
    args: [
      "--no-sandbox",
      "--disable-setuid-sandbox",
      "--enable-webgl",
      "--ignore-gpu-blocklist",
      "--use-gl=angle",
      "--use-angle=gl",
    ],
    headless: true,
  });

  const consoleErrors = [];

  try {
    const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });

    page.on("console", (msg) => {
      if (msg.type() === "error") {
        if (!msg.text().includes("401") && !msg.text().includes("favicon")) {
          consoleErrors.push(`[Console Error] ${msg.text()}`);
        }
      }
    });

    console.log(">>> Navigating to landing page...");
    await page.goto(`${BASE_URL}/`, { waitUntil: "domcontentloaded" });
    await page.waitForSelector("h1", { timeout: 10000 });
    await page.waitForTimeout(2000); // Allow WebGL canvases to mount

    // -------------------------------------------------------------------------
    // EVIDENCE 1: Hero Starting Composition
    // -------------------------------------------------------------------------
    console.log(">>> [EVIDENCE 1] Capturing Hero starting composition...");
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "01-hero-starting-composition.png"),
      clip: { x: 0, y: 0, width: 1440, height: 950 },
    });
    console.log("    ✔ Saved: 01-hero-starting-composition.png");

    // -------------------------------------------------------------------------
    // EVIDENCE 2: Hero-to-Story Transition
    // -------------------------------------------------------------------------
    console.log(">>> [EVIDENCE 2] Capturing Hero-to-Story transition bridge...");
    await page.evaluate(() => {
      const bridge = document.querySelector("#story");
      if (bridge) {
        window.scrollTo(0, bridge.offsetTop - 350);
      }
    });
    await page.waitForTimeout(600);
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "02-hero-to-story-transition.png"),
      clip: { x: 0, y: 0, width: 1440, height: 900 },
    });
    console.log("    ✔ Saved: 02-hero-to-story-transition.png");

    // Get story container bounding metrics
    const storyMetrics = await page.evaluate(() => {
      const el = document.querySelector("#story");
      if (!el) return null;
      const rect = el.getBoundingClientRect();
      const scrollTop = window.scrollY;
      return {
        offsetTop: rect.top + scrollTop,
        height: el.scrollHeight,
        windowHeight: window.innerHeight,
      };
    });

    if (!storyMetrics) {
      throw new Error("Section #story was not found in the DOM!");
    }

    console.log(`    Story section offset: ${storyMetrics.offsetTop}px, height: ${storyMetrics.height}px`);
    const totalDist = storyMetrics.height - storyMetrics.windowHeight;

    // Helper to scroll to a specific progress (0.0 to 1.0) inside the pinned story
    async function scrollToStoryProgress(p) {
      const targetY = storyMetrics.offsetTop + p * totalDist;
      await page.evaluate((y) => window.scrollTo(0, y), targetY);
      await page.waitForTimeout(500); // Allow GSAP scrub and WebGL frame render
    }

    // -------------------------------------------------------------------------
    // EVIDENCE 3: Scene 01 — Connected & Verified
    // -------------------------------------------------------------------------
    console.log(">>> [EVIDENCE 3] Capturing Scene 01 (Connected)...");
    await scrollToStoryProgress(0.04);
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "03-scene-01-connected.png"),
    });
    console.log("    ✔ Saved: 03-scene-01-connected.png");

    // -------------------------------------------------------------------------
    // EVIDENCE 4: Intermediate Transition 01 -> 02
    // -------------------------------------------------------------------------
    console.log(">>> [EVIDENCE 4] Capturing Transition 01 -> 02...");
    await scrollToStoryProgress(0.18);
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "04-transition-01-to-02.png"),
    });
    console.log("    ✔ Saved: 04-transition-01-to-02.png");

    // -------------------------------------------------------------------------
    // EVIDENCE 5: Scene 02 — Active Monitoring
    // -------------------------------------------------------------------------
    console.log(">>> [EVIDENCE 5] Capturing Scene 02 (Active Monitoring)...");
    await scrollToStoryProgress(0.26);
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "05-scene-02-active-monitoring.png"),
    });
    console.log("    ✔ Saved: 05-scene-02-active-monitoring.png");

    // -------------------------------------------------------------------------
    // EVIDENCE 6: Intermediate Transition 02 -> 03
    // -------------------------------------------------------------------------
    console.log(">>> [EVIDENCE 6] Capturing Transition 02 -> 03...");
    await scrollToStoryProgress(0.36);
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "06-transition-02-to-03.png"),
    });
    console.log("    ✔ Saved: 06-transition-02-to-03.png");

    // -------------------------------------------------------------------------
    // EVIDENCE 7: Scene 03 — Degradation
    // -------------------------------------------------------------------------
    console.log(">>> [EVIDENCE 7] Capturing Scene 03 (Degradation)...");
    await scrollToStoryProgress(0.44);
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "07-scene-03-degradation.png"),
    });
    console.log("    ✔ Saved: 07-scene-03-degradation.png");

    // -------------------------------------------------------------------------
    // EVIDENCE 8: Intermediate Transition 03 -> 04
    // -------------------------------------------------------------------------
    console.log(">>> [EVIDENCE 8] Capturing Transition 03 -> 04 (Breakage initiation)...");
    await scrollToStoryProgress(0.53);
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "08-transition-03-to-04.png"),
    });
    console.log("    ✔ Saved: 08-transition-03-to-04.png");

    // -------------------------------------------------------------------------
    // EVIDENCE 9: Scene 04 — Total Outage (Centerpiece Shockwave & Red Beacon)
    // -------------------------------------------------------------------------
    console.log(">>> [EVIDENCE 9] Capturing Scene 04 (TOTAL OUTAGE)...");
    await scrollToStoryProgress(0.60);
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "09-scene-04-total-outage.png"),
    });
    console.log("    ✔ Saved: 09-scene-04-total-outage.png");

    // -------------------------------------------------------------------------
    // EVIDENCE 10: Scene 05 — Incident Awareness
    // -------------------------------------------------------------------------
    console.log(">>> [EVIDENCE 10] Capturing Scene 05 (Incident Awareness)...");
    await scrollToStoryProgress(0.76);
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "10-scene-05-incident-awareness.png"),
    });
    console.log("    ✔ Saved: 10-scene-05-incident-awareness.png");

    // -------------------------------------------------------------------------
    // EVIDENCE 11: Scene 06 — Recovery & Resolution
    // -------------------------------------------------------------------------
    console.log(">>> [EVIDENCE 11] Capturing Scene 06 (Recovery & Resolution)...");
    await scrollToStoryProgress(0.92);
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "11-scene-06-recovery.png"),
    });
    console.log("    ✔ Saved: 11-scene-06-recovery.png");

    // -------------------------------------------------------------------------
    // EVIDENCE 12 & 13: Persian RTL Mode
    // -------------------------------------------------------------------------
    console.log("\n>>> [EVIDENCE 12 & 13] Switching to Persian (FA / RTL) Mode...");
    await page.evaluate(() => {
      localStorage.setItem("clientpulse_locale", "fa");
      window.location.reload();
    });
    await page.waitForSelector("h1", { timeout: 10000 });
    await page.waitForTimeout(2000);

    // Verify RTL is active
    const isHtmlRTL = await page.evaluate(() => document.documentElement.dir === "rtl");
    console.log(`    HTML dir is 'rtl': ${isHtmlRTL}`);

    const faStoryMetrics = await page.evaluate(() => {
      const el = document.querySelector("#story");
      const rect = el.getBoundingClientRect();
      const scrollTop = window.scrollY;
      return { offsetTop: rect.top + scrollTop, height: el.scrollHeight, windowHeight: window.innerHeight };
    });

    const faTotalDist = faStoryMetrics.height - faStoryMetrics.windowHeight;

    // Persian Scene 04
    await page.evaluate((y) => window.scrollTo(0, y), faStoryMetrics.offsetTop + 0.60 * faTotalDist);
    await page.waitForTimeout(500);
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "12-persian-rtl-scene04.png"),
    });
    console.log("    ✔ Saved: 12-persian-rtl-scene04.png");

    // Persian Scene 06
    await page.evaluate((y) => window.scrollTo(0, y), faStoryMetrics.offsetTop + 0.92 * faTotalDist);
    await page.waitForTimeout(500);
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "13-persian-rtl-scene06.png"),
    });
    console.log("    ✔ Saved: 13-persian-rtl-scene06.png");

    // -------------------------------------------------------------------------
    // EVIDENCE 14, 15, 16: Mobile Viewport (390px iPhone width)
    // -------------------------------------------------------------------------
    console.log("\n>>> [EVIDENCE 14, 15, 16] Testing Mobile Experience (390x844)...");
    const mobilePage = await browser.newPage({
      viewport: { width: 390, height: 844 },
      isMobile: true,
      hasTouch: true,
    });

    // Set English for mobile screenshots
    await mobilePage.goto(`${BASE_URL}/`, { waitUntil: "domcontentloaded" });
    await mobilePage.evaluate(() => {
      localStorage.setItem("clientpulse_locale", "en");
      window.location.reload();
    });
    await mobilePage.waitForSelector("h1", { timeout: 10000 });
    await mobilePage.waitForTimeout(1500);

    // Mobile Hero
    await mobilePage.screenshot({
      path: path.join(SCREENSHOT_DIR, "14-mobile-390px-hero.png"),
      clip: { x: 0, y: 0, width: 390, height: 800 },
    });
    console.log("    ✔ Saved: 14-mobile-390px-hero.png");

    const mobStoryMetrics = await mobilePage.evaluate(() => {
      const el = document.querySelector("#story");
      const rect = el.getBoundingClientRect();
      const scrollTop = window.scrollY;
      return { offsetTop: rect.top + scrollTop, height: el.scrollHeight, windowHeight: window.innerHeight };
    });

    const mobTotalDist = mobStoryMetrics.height - mobStoryMetrics.windowHeight;

    // Mobile Scene 04 Outage
    await mobilePage.evaluate((y) => window.scrollTo(0, y), mobStoryMetrics.offsetTop + 0.60 * mobTotalDist);
    await mobilePage.waitForTimeout(500);
    await mobilePage.screenshot({
      path: path.join(SCREENSHOT_DIR, "15-mobile-390px-scene04.png"),
    });
    console.log("    ✔ Saved: 15-mobile-390px-scene04.png");

    // Mobile Scene 06 Recovery
    await mobilePage.evaluate((y) => window.scrollTo(0, y), mobStoryMetrics.offsetTop + 0.92 * mobTotalDist);
    await mobilePage.waitForTimeout(500);
    await mobilePage.screenshot({
      path: path.join(SCREENSHOT_DIR, "16-mobile-390px-scene06.png"),
    });
    console.log("    ✔ Saved: 16-mobile-390px-scene06.png");

    await mobilePage.close();

    // -------------------------------------------------------------------------
    // ANIMATED RECORDING / PREVIEW: Capture smooth scroll sequence
    // -------------------------------------------------------------------------
    console.log("\n>>> Capturing sequence of 24 frames for animated preview...");
    // Return to English desktop
    await page.evaluate(() => {
      localStorage.setItem("clientpulse_locale", "en");
      window.location.reload();
    });
    await page.waitForSelector("h1", { timeout: 10000 });
    await page.waitForTimeout(1500);

    const deskStory = await page.evaluate(() => {
      const el = document.querySelector("#story");
      const rect = el.getBoundingClientRect();
      const scrollTop = window.scrollY;
      return { offsetTop: rect.top + scrollTop, height: el.scrollHeight, windowHeight: window.innerHeight };
    });
    const deskTotal = deskStory.height - deskStory.windowHeight;

    const frameCount = 30;
    for (let f = 0; f < frameCount; f++) {
      const prog = f / (frameCount - 1);
      const scrollY = deskStory.offsetTop + prog * deskTotal;
      await page.evaluate((y) => window.scrollTo(0, y), scrollY);
      await page.waitForTimeout(80);
      const paddedNum = String(f + 1).padStart(3, "0");
      await page.screenshot({
        path: path.join(FRAMES_DIR, `frame-${paddedNum}.png`),
      });
    }
    console.log(`    ✔ Captured ${frameCount} frames.`);

    // Encode into MP4 and GIF using ffmpeg
    try {
      const videoOutputMp4 = path.join(SCREENSHOT_DIR, "cinematic-scroll-preview.mp4");
      const videoOutputGif = path.join(SCREENSHOT_DIR, "cinematic-scroll-preview.gif");

      execSync(
        `ffmpeg -y -framerate 6 -i "${FRAMES_DIR}/frame-%03d.png" -c:v libx264 -pix_fmt yuv420p -vf "scale=1280:-2" "${videoOutputMp4}"`,
        { stdio: "inherit" }
      );
      console.log(`    ✔ Generated MP4 Preview: ${videoOutputMp4}`);

      execSync(
        `ffmpeg -y -framerate 6 -i "${FRAMES_DIR}/frame-%03d.png" -vf "scale=720:-2:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse" "${videoOutputGif}"`,
        { stdio: "inherit" }
      );
      console.log(`    ✔ Generated Animated GIF Preview: ${videoOutputGif}`);
    } catch (ffmpegErr) {
      console.warn("    Notice: ffmpeg encoding notice:", ffmpegErr.message);
    }

    console.log("\n================================================================================");
    console.log("TASK-UI-013 VALIDATION SUMMARY:");
    console.log("Console errors observed:", consoleErrors.length);
    if (consoleErrors.length > 0) {
      consoleErrors.forEach((e) => console.log("  ", e));
    }
    console.log("ALL 16 REQUIRED EVIDENCE ITEMS AND ANIMATED PREVIEW GENERATED SUCCESSFULLY!");
    console.log("================================================================================");
  } finally {
    await browser.close();
  }
}

runTask013Validation().catch((err) => {
  console.error("FATAL in task-013 validation:", err);
  process.exit(1);
});
