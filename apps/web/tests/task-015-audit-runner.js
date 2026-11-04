const { chromium } = require("playwright");
const path = require("path");
const fs = require("fs");
const { execSync } = require("child_process");

const TARGET_URL = process.env.TARGET_URL || "https://mindu.ir";
const LOCAL_URL = "http://localhost:3000";
const REPORT_DIR = path.resolve(__dirname, "../../../task-report/TASK-UI-015");
const SCREENSHOT_DIR = path.join(REPORT_DIR, "screenshots");
const FRAMES_DIR = path.join(REPORT_DIR, "frames");

if (!fs.existsSync(REPORT_DIR)) fs.mkdirSync(REPORT_DIR, { recursive: true });
if (!fs.existsSync(SCREENSHOT_DIR)) fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
if (!fs.existsSync(FRAMES_DIR)) fs.mkdirSync(FRAMES_DIR, { recursive: true });

async function runAudit() {
  console.log("================================================================================");
  console.log("TASK-UI-015: Visual Evidence Capture & Cinematic Experience Audit");
  console.log("Target Deployed URL:", TARGET_URL);
  console.log("Local Fallback URL: ", LOCAL_URL);
  console.log("Output Directory:   ", REPORT_DIR);
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

  const auditData = {
    urlTested: TARGET_URL,
    timestamp: new Date().toISOString(),
    viewports: {},
    consoleErrors: [],
    performanceMetrics: {},
    observations: {},
  };

  try {
    let activeUrl = TARGET_URL;
    let page = await browser.newPage({ viewport: { width: 1440, height: 900 } });

    page.on("console", (msg) => {
      if (msg.type() === "error" && !msg.text().includes("401") && !msg.text().includes("favicon")) {
        auditData.consoleErrors.push(`[Console Error] ${msg.text()}`);
      }
    });

    console.log(`>>> [1/7] Connecting to ${activeUrl}...`);
    try {
      const resp = await page.goto(activeUrl, { waitUntil: "networkidle", timeout: 25000 });
      if (!resp || resp.status() >= 400) throw new Error(`HTTP ${resp ? resp.status() : 'null'}`);
    } catch (netErr) {
      console.warn(`    Primary ${activeUrl} connection failed: ${netErr.message}. Falling back to ${LOCAL_URL}`);
      activeUrl = LOCAL_URL;
      auditData.urlTested = activeUrl;
      await page.goto(activeUrl, { waitUntil: "networkidle", timeout: 20000 });
    }

    await page.waitForSelector("h1", { timeout: 10000 });
    await page.waitForTimeout(2000); // Allow WebGL to stabilize

    // Record DOM architecture
    const domSummary = await page.evaluate(() => {
      const canvases = Array.from(document.querySelectorAll("canvas")).map((c) => ({
        width: c.width,
        height: c.height,
        parentClass: c.parentElement?.className,
        style: c.getAttribute("style"),
      }));
      const sections = Array.from(document.querySelectorAll("section")).map((s) => ({
        id: s.id,
        className: s.className.slice(0, 80),
        height: s.scrollHeight,
      }));
      return { canvases, sections, bodyHeight: document.body.scrollHeight };
    });
    auditData.domSummary = domSummary;

    // -------------------------------------------------------------------------
    // 1. DESKTOP VIEWPORT EVIDENCE
    // -------------------------------------------------------------------------
    console.log(">>> [2/7] Capturing Desktop Viewport Progression (1440x900)...");
    
    // Hero Initial
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.waitForTimeout(600);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "01-desktop-hero.png"), clip: { x: 0, y: 0, width: 1440, height: 900 } });
    console.log("    ✔ Saved: 01-desktop-hero.png");

    // Hero to Story Transition
    const storyMetrics = await page.evaluate(() => {
      const el = document.querySelector("#story");
      if (!el) return null;
      const rect = el.getBoundingClientRect();
      return {
        offsetTop: rect.top + window.scrollY,
        height: el.scrollHeight,
        windowHeight: window.innerHeight,
      };
    });

    if (storyMetrics) {
      await page.evaluate((top) => window.scrollTo(0, top - 350), storyMetrics.offsetTop);
      await page.waitForTimeout(600);
      await page.screenshot({ path: path.join(SCREENSHOT_DIR, "02-desktop-hero-to-story.png") });
      console.log("    ✔ Saved: 02-desktop-hero-to-story.png");

      const totalDist = storyMetrics.height - storyMetrics.windowHeight;

      // Scene 01 Connected
      await page.evaluate((y) => window.scrollTo(0, y), storyMetrics.offsetTop + 0.05 * totalDist);
      await page.waitForTimeout(500);
      await page.screenshot({ path: path.join(SCREENSHOT_DIR, "03-desktop-story-scene1.png") });
      console.log("    ✔ Saved: 03-desktop-story-scene1.png");

      // Scene 02 Monitoring
      await page.evaluate((y) => window.scrollTo(0, y), storyMetrics.offsetTop + 0.25 * totalDist);
      await page.waitForTimeout(500);
      await page.screenshot({ path: path.join(SCREENSHOT_DIR, "04-desktop-story-scene2.png") });
      console.log("    ✔ Saved: 04-desktop-story-scene2.png");

      // Scene 03 Degradation
      await page.evaluate((y) => window.scrollTo(0, y), storyMetrics.offsetTop + 0.44 * totalDist);
      await page.waitForTimeout(500);
      await page.screenshot({ path: path.join(SCREENSHOT_DIR, "05-desktop-story-scene3.png") });
      console.log("    ✔ Saved: 05-desktop-story-scene3.png");

      // Scene 04 Outage
      await page.evaluate((y) => window.scrollTo(0, y), storyMetrics.offsetTop + 0.60 * totalDist);
      await page.waitForTimeout(500);
      await page.screenshot({ path: path.join(SCREENSHOT_DIR, "06-desktop-story-scene4.png") });
      console.log("    ✔ Saved: 06-desktop-story-scene4.png");

      // Scene 05 Incident
      await page.evaluate((y) => window.scrollTo(0, y), storyMetrics.offsetTop + 0.76 * totalDist);
      await page.waitForTimeout(500);
      await page.screenshot({ path: path.join(SCREENSHOT_DIR, "07-desktop-story-scene5.png") });
      console.log("    ✔ Saved: 07-desktop-story-scene5.png");

      // Scene 06 Recovery
      await page.evaluate((y) => window.scrollTo(0, y), storyMetrics.offsetTop + 0.93 * totalDist);
      await page.waitForTimeout(500);
      await page.screenshot({ path: path.join(SCREENSHOT_DIR, "08-desktop-story-scene6.png") });
      console.log("    ✔ Saved: 08-desktop-story-scene6.png");

      // Recovery to Next section
      await page.evaluate((y) => window.scrollTo(0, y), storyMetrics.offsetTop + storyMetrics.height + 100);
      await page.waitForTimeout(600);
      await page.screenshot({ path: path.join(SCREENSHOT_DIR, "09-desktop-playground.png") });
      console.log("    ✔ Saved: 09-desktop-playground.png");
    }

    // Capture Remaining Sections
    await page.evaluate(() => {
      const el = document.querySelector("#features") || document.querySelectorAll("section")[2];
      if (el) el.scrollIntoView();
    });
    await page.waitForTimeout(500);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "10-desktop-features.png") });
    console.log("    ✔ Saved: 10-desktop-features.png");

    await page.evaluate(() => {
      const el = document.querySelector("#comparison") || document.querySelectorAll("section")[4];
      if (el) el.scrollIntoView();
    });
    await page.waitForTimeout(500);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "11-desktop-comparison.png") });
    console.log("    ✔ Saved: 11-desktop-comparison.png");

    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    await page.waitForTimeout(500);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "12-desktop-footer.png") });
    console.log("    ✔ Saved: 12-desktop-footer.png");

    // -------------------------------------------------------------------------
    // 2. PERSIAN RTL VIEWPORT EVIDENCE
    // -------------------------------------------------------------------------
    console.log(">>> [3/7] Capturing Persian RTL Progression...");
    await page.evaluate(() => {
      localStorage.setItem("clientpulse_locale", "fa");
      window.location.reload();
    });
    await page.waitForSelector("h1", { timeout: 10000 });
    await page.waitForTimeout(2000);

    await page.evaluate(() => window.scrollTo(0, 0));
    await page.waitForTimeout(500);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "13-persian-hero.png"), clip: { x: 0, y: 0, width: 1440, height: 900 } });
    console.log("    ✔ Saved: 13-persian-hero.png");

    if (storyMetrics) {
      const faStory = await page.evaluate(() => {
        const el = document.querySelector("#story");
        const rect = el.getBoundingClientRect();
        return { offsetTop: rect.top + window.scrollY, height: el.scrollHeight, windowHeight: window.innerHeight };
      });
      const faTotal = faStory.height - faStory.windowHeight;

      await page.evaluate((y) => window.scrollTo(0, y), faStory.offsetTop + 0.60 * faTotal);
      await page.waitForTimeout(500);
      await page.screenshot({ path: path.join(SCREENSHOT_DIR, "14-persian-story-outage.png") });
      console.log("    ✔ Saved: 14-persian-story-outage.png");

      await page.evaluate((y) => window.scrollTo(0, y), faStory.offsetTop + 0.93 * faTotal);
      await page.waitForTimeout(500);
      await page.screenshot({ path: path.join(SCREENSHOT_DIR, "15-persian-story-recovery.png") });
      console.log("    ✔ Saved: 15-persian-story-recovery.png");
    }

    // -------------------------------------------------------------------------
    // 3. MOBILE VIEWPORT EVIDENCE (390x844)
    // -------------------------------------------------------------------------
    console.log(">>> [4/7] Capturing Mobile Viewport Progression (390x844)...");
    const mobilePage = await browser.newPage({
      viewport: { width: 390, height: 844 },
      isMobile: true,
      hasTouch: true,
    });
    await mobilePage.goto(activeUrl, { waitUntil: "networkidle" });
    await mobilePage.evaluate(() => {
      localStorage.setItem("clientpulse_locale", "en");
      window.location.reload();
    });
    await mobilePage.waitForSelector("h1", { timeout: 10000 });
    await mobilePage.waitForTimeout(1500);

    await mobilePage.screenshot({ path: path.join(SCREENSHOT_DIR, "16-mobile-hero.png"), clip: { x: 0, y: 0, width: 390, height: 844 } });
    console.log("    ✔ Saved: 16-mobile-hero.png");

    const mobStory = await mobilePage.evaluate(() => {
      const el = document.querySelector("#story");
      if (!el) return null;
      const rect = el.getBoundingClientRect();
      return { offsetTop: rect.top + window.scrollY, height: el.scrollHeight, windowHeight: window.innerHeight };
    });

    if (mobStory) {
      const mobTotal = mobStory.height - mobStory.windowHeight;
      await mobilePage.evaluate((y) => window.scrollTo(0, y), mobStory.offsetTop + 0.60 * mobTotal);
      await mobilePage.waitForTimeout(500);
      await mobilePage.screenshot({ path: path.join(SCREENSHOT_DIR, "17-mobile-story-outage.png") });
      console.log("    ✔ Saved: 17-mobile-story-outage.png");

      await mobilePage.evaluate((y) => window.scrollTo(0, y), mobStory.offsetTop + 0.93 * mobTotal);
      await mobilePage.waitForTimeout(500);
      await mobilePage.screenshot({ path: path.join(SCREENSHOT_DIR, "18-mobile-story-recovery.png") });
      console.log("    ✔ Saved: 18-mobile-story-recovery.png");
    }
    await mobilePage.close();

    // -------------------------------------------------------------------------
    // 4. REDUCED-MOTION EVIDENCE
    // -------------------------------------------------------------------------
    console.log(">>> [5/7] Testing prefers-reduced-motion...");
    const rmPage = await browser.newPage({
      viewport: { width: 1440, height: 900 },
      colorScheme: "dark",
      reducedMotion: "reduce",
    });
    await rmPage.goto(activeUrl, { waitUntil: "networkidle" });
    await rmPage.waitForSelector("h1", { timeout: 10000 });
    await rmPage.waitForTimeout(1500);

    if (storyMetrics) {
      await rmPage.evaluate((top) => window.scrollTo(0, top + 1500), storyMetrics.offsetTop);
      await rmPage.waitForTimeout(600);
      await rmPage.screenshot({ path: path.join(SCREENSHOT_DIR, "19-reduced-motion-story.png") });
      console.log("    ✔ Saved: 19-reduced-motion-story.png");
    }
    await rmPage.close();

    // -------------------------------------------------------------------------
    // 5. CAPTURE CONTINUOUS FRAMES FOR VIDEO & TRANSITION STUDY
    // -------------------------------------------------------------------------
    console.log(">>> [6/7] Capturing 24 Intermediate Frames for Continuous Scroll...");
    await page.evaluate(() => {
      localStorage.setItem("clientpulse_locale", "en");
      window.location.reload();
    });
    await page.waitForSelector("h1", { timeout: 10000 });
    await page.waitForTimeout(1500);

    const frameStory = await page.evaluate(() => {
      const el = document.querySelector("#story");
      const rect = el.getBoundingClientRect();
      return { offsetTop: rect.top + window.scrollY, height: el.scrollHeight, windowHeight: window.innerHeight };
    });
    const frameTotal = frameStory.height - frameStory.windowHeight;

    const frameCount = 24;
    for (let f = 0; f < frameCount; f++) {
      const p = f / (frameCount - 1);
      const y = frameStory.offsetTop + p * frameTotal;
      await page.evaluate((scr) => window.scrollTo(0, scr), y);
      await page.waitForTimeout(90);
      const num = String(f + 1).padStart(3, "0");
      await page.screenshot({ path: path.join(FRAMES_DIR, `frame-${num}.png`) });
    }
    console.log(`    ✔ Captured ${frameCount} frames in ${FRAMES_DIR}`);

    // Encode MP4 & GIF
    try {
      const mp4Path = path.join(REPORT_DIR, "scroll-recording.mp4");
      const gifPath = path.join(REPORT_DIR, "scroll-recording.gif");
      execSync(`ffmpeg -y -framerate 6 -i "${FRAMES_DIR}/frame-%03d.png" -c:v libx264 -pix_fmt yuv420p -vf "scale=1280:-2" "${mp4Path}"`, { stdio: "ignore" });
      execSync(`ffmpeg -y -framerate 6 -i "${FRAMES_DIR}/frame-%03d.png" -vf "scale=720:-2:flags=lanczos,split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse" "${gifPath}"`, { stdio: "ignore" });
      console.log("    ✔ Generated scroll-recording.mp4 & scroll-recording.gif");
    } catch (e) {
      console.warn("    Warning: ffmpeg encoding failed:", e.message);
    }

    // -------------------------------------------------------------------------
    // 6. CONTACT SHEETS VIA MONTAGE / CONVERT
    // -------------------------------------------------------------------------
    console.log(">>> [7/7] Generating Contact Sheets...");
    try {
      // Story Progression Contact Sheet (Scenes 1-6)
      const storySheetPath = path.join(REPORT_DIR, "contact-sheet-story.png");
      execSync(
        `montage "${SCREENSHOT_DIR}/03-desktop-story-scene1.png" "${SCREENSHOT_DIR}/04-desktop-story-scene2.png" "${SCREENSHOT_DIR}/05-desktop-story-scene3.png" "${SCREENSHOT_DIR}/06-desktop-story-scene4.png" "${SCREENSHOT_DIR}/07-desktop-story-scene5.png" "${SCREENSHOT_DIR}/08-desktop-story-scene6.png" -geometry 480x270+6+6 -tile 3x2 "${storySheetPath}"`,
        { stdio: "ignore" }
      );
      console.log("    ✔ Generated: contact-sheet-story.png");

      // Full Landing Overview Contact Sheet
      const landingSheetPath = path.join(REPORT_DIR, "contact-sheet-landing.png");
      execSync(
        `montage "${SCREENSHOT_DIR}/01-desktop-hero.png" "${SCREENSHOT_DIR}/02-desktop-hero-to-story.png" "${SCREENSHOT_DIR}/06-desktop-story-scene4.png" "${SCREENSHOT_DIR}/08-desktop-story-scene6.png" "${SCREENSHOT_DIR}/09-desktop-playground.png" "${SCREENSHOT_DIR}/10-desktop-features.png" "${SCREENSHOT_DIR}/11-desktop-comparison.png" "${SCREENSHOT_DIR}/12-desktop-footer.png" -geometry 400x225+4+4 -tile 4x2 "${landingSheetPath}"`,
        { stdio: "ignore" }
      );
      console.log("    ✔ Generated: contact-sheet-landing.png");

      // Transition Comparison Sheet
      const transSheetPath = path.join(REPORT_DIR, "transition-comparison.png");
      execSync(
        `montage "${FRAMES_DIR}/frame-001.png" "${FRAMES_DIR}/frame-006.png" "${FRAMES_DIR}/frame-012.png" "${FRAMES_DIR}/frame-016.png" "${FRAMES_DIR}/frame-020.png" "${FRAMES_DIR}/frame-024.png" -geometry 480x270+6+6 -tile 3x2 "${transSheetPath}"`,
        { stdio: "ignore" }
      );
      console.log("    ✔ Generated: transition-comparison.png");
    } catch (montageErr) {
      console.warn("    Warning: montage contact sheet generation warning:", montageErr.message);
    }

    console.log("\n================================================================================");
    console.log("AUDIT DATA COLLECTION COMPLETE");
    console.log("Writing report to:", path.join(REPORT_DIR, "report.md"));
    console.log("================================================================================");

    return auditData;
  } finally {
    await browser.close();
  }
}

runAudit().then((data) => {
  fs.writeFileSync(path.join(REPORT_DIR, "audit-metrics.json"), JSON.stringify(data, null, 2));
  console.log("Metrics saved to audit-metrics.json");
}).catch((err) => {
  console.error("FATAL in task-015 audit:", err);
  process.exit(1);
});

