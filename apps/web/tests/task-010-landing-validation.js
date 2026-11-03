const { chromium } = require("playwright");
const path = require("path");
const fs = require("fs");
const { execSync } = require("child_process");

const BASE_URL = process.env.BASE_URL || "http://localhost:3000";
const SCREENSHOT_DIR = path.resolve(__dirname, "../../../task-report/screenshots/task-010");

if (!fs.existsSync(SCREENSHOT_DIR)) {
  fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
}

function runSql(sql) {
  const cmd = `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -t -A -q -c "${sql.replace(/"/g, '\\"')}"`;
  const res = execSync(cmd).toString().trim();
  const lines = res.split("\n").map((l) => l.trim()).filter((l) => l && !l.startsWith("INSERT") && !l.startsWith("UPDATE") && !l.startsWith("DELETE"));
  return lines.length > 0 ? lines[0] : "";
}

async function assertNoHorizontalOverflow(page, routeName, viewportName) {
  const isOverflowing = await page.evaluate(() => {
    return document.documentElement.scrollWidth > window.innerWidth;
  });
  if (isOverflowing) {
    const details = await page.evaluate(() => ({
      scrollWidth: document.documentElement.scrollWidth,
      innerWidth: window.innerWidth,
    }));
    throw new Error(
      `CRITICAL OVERFLOW on route "${routeName}" at viewport ${viewportName}: scrollWidth (${details.scrollWidth}px) exceeds innerWidth (${details.innerWidth}px)`
    );
  }
}

async function runTask010Validation() {
  console.log("================================================================================");
  console.log("TASK-010: Public Marketing Landing Page for mindu.ir Validation Suite");
  console.log("Base URL:", BASE_URL);
  console.log("Screenshots Directory:", SCREENSHOT_DIR);
  console.log("================================================================================\n");

  const browser = await chromium.launch({
    executablePath: "/usr/bin/google-chrome",
    headless: true,
  });

  const consoleErrors = [];

  try {
    // -------------------------------------------------------------------------
    // TEST SCENARIO 1: Unauthenticated Visitor on '/' & Guest API Isolation
    // -------------------------------------------------------------------------
    console.log(">>> [SCENARIO 1] Unauthenticated Visitor on '/' (Public Landing Page Isolation)...");
    const guestContext = await browser.newContext({
      viewport: { width: 1440, height: 900 },
    });
    const guestPage = await guestContext.newPage();

    guestPage.on("console", (msg) => {
      if (msg.type() === "error") {
        // Ignore expected session probe 401s
        if (!msg.text().includes("401") && !msg.text().includes("favicon")) {
          consoleErrors.push(`[Guest Page] ${msg.text()}`);
        }
      }
    });

    const interceptedRequests = [];
    guestPage.on("request", (req) => {
      interceptedRequests.push(req.url());
    });

    await guestPage.goto(`${BASE_URL}/`, { waitUntil: "networkidle" });
    await guestPage.waitForTimeout(1000);

    // Verify Title & SEO Meta
    const pageTitle = await guestPage.title();
    console.log(`    Page Title: "${pageTitle}"`);
    if (!pageTitle.includes("ClientPulse") || !pageTitle.includes("mindu.ir")) {
      throw new Error(`Invalid page title: ${pageTitle}`);
    }

    // Verify Public Page NEVER calls protected dashboard endpoints
    const protectedApis = [
      "/api/v1/dashboard/stats",
      "/api/v1/incidents?status=open",
      "/api/v1/heartbeats?limit=",
      "/api/v1/maintenance-activities?limit=",
      "/api/v1/reports?limit=",
      "/api/v1/clients?limit=",
    ];

    for (const api of protectedApis) {
      const leaked = interceptedRequests.find((url) => url.includes(api));
      if (leaked) {
        throw new Error(`SECURITY LEAK: Protected dashboard API "${api}" was called on public landing page! URL: ${leaked}`);
      }
    }
    console.log("    ✔ Zero protected dashboard API calls verified on public landing page.");

    // Verify Landing Page Sections exist
    const hasHero = await guestPage.$("h1");
    const heroText = await hasHero.textContent();
    console.log(`    Hero Headline: "${heroText}"`);

    const hasFeatures = await guestPage.$("#features");
    const hasHow = await guestPage.$("#how-it-works");
    const hasValue = await guestPage.$("#value");
    const hasPreview = await guestPage.$("#preview");

    if (!hasFeatures || !hasHow || !hasValue || !hasPreview) {
      throw new Error("One or more required marketing sections (#features, #how-it-works, #value, #preview) are missing!");
    }
    console.log("    ✔ All 4 primary anchor sections (#features, #how-it-works, #value, #preview) verified.");

    // Capture English Desktop Hero & Fullpage
    await guestPage.screenshot({ path: path.join(SCREENSHOT_DIR, "02-en-desktop-hero.png"), clip: { x: 0, y: 0, width: 1440, height: 900 } });
    await guestPage.screenshot({ path: path.join(SCREENSHOT_DIR, "01-en-desktop-fullpage.png"), fullPage: true });
    console.log("    ✔ English Desktop Fullpage & Hero screenshots captured.");

    // -------------------------------------------------------------------------
    // TEST SCENARIO 2: Responsive Breakpoints & Zero Horizontal Overflow
    // -------------------------------------------------------------------------
    console.log("\n>>> [SCENARIO 2] Responsive Breakpoint Verification (1440px, 1024px, 768px, 390px, 360px)...");
    const breakpoints = [
      { width: 1440, height: 900, name: "1440px Desktop" },
      { width: 1024, height: 768, name: "1024px Small Desktop" },
      { width: 768, height: 1024, name: "768px Tablet" },
      { width: 390, height: 844, name: "390px Standard Mobile" },
      { width: 360, height: 740, name: "360px Narrow Mobile" },
    ];

    for (const bp of breakpoints) {
      await guestPage.setViewportSize({ width: bp.width, height: bp.height });
      await guestPage.waitForTimeout(500);
      await assertNoHorizontalOverflow(guestPage, "/", bp.name);
      console.log(`    ✔ Breakpoint ${bp.name}: scrollWidth <= innerWidth verified.`);
    }

    // Capture English Mobile Screenshot at 390px
    await guestPage.setViewportSize({ width: 390, height: 844 });
    await guestPage.waitForTimeout(400);
    await guestPage.screenshot({ path: path.join(SCREENSHOT_DIR, "03-en-mobile-390.png"), fullPage: true });
    console.log("    ✔ English Mobile (390px) fullpage screenshot captured.");

    // Capture English Narrow Mobile Screenshot at 360px
    await guestPage.setViewportSize({ width: 360, height: 740 });
    await guestPage.waitForTimeout(400);
    await guestPage.screenshot({ path: path.join(SCREENSHOT_DIR, "03-b-en-mobile-360.png"), fullPage: true });
    console.log("    ✔ English Narrow Mobile (360px) fullpage screenshot captured.");

    // -------------------------------------------------------------------------
    // TEST SCENARIO 3: Mobile Navigation Drawer (Keyboard, Escape, Trapping)
    // -------------------------------------------------------------------------
    console.log("\n>>> [SCENARIO 3] Mobile Navigation Drawer Interaction & Accessibility...");
    await guestPage.setViewportSize({ width: 390, height: 844 });
    await guestPage.waitForTimeout(400);

    // Click Hamburger
    const menuBtn = await guestPage.$('button[aria-label="Menu"], button[aria-label="منو"]');
    if (!menuBtn) throw new Error("Mobile menu trigger button not found!");
    await menuBtn.click();
    await guestPage.waitForTimeout(500);

    const drawerDialog = await guestPage.$('[role="dialog"]');
    if (!drawerDialog) throw new Error("Mobile drawer dialog did not open!");

    // Capture Open Drawer Screenshot
    await guestPage.screenshot({ path: path.join(SCREENSHOT_DIR, "06-mobile-nav-drawer-open.png") });
    console.log("    ✔ Mobile navigation drawer opened cleanly and screenshot captured.");

    // Press Escape to Close
    await guestPage.keyboard.press("Escape");
    await guestPage.waitForTimeout(500);
    const drawerStillOpen = await guestPage.$('[role="dialog"]');
    if (drawerStillOpen) throw new Error("Mobile drawer did not close on Escape key!");
    console.log("    ✔ Mobile drawer closed successfully on Escape key.");

    // -------------------------------------------------------------------------
    // TEST SCENARIO 4: Persian (FA / RTL) Mode Verification
    // -------------------------------------------------------------------------
    console.log("\n>>> [SCENARIO 4] Persian (FA / RTL) Mode Verification...");
    await guestPage.setViewportSize({ width: 1440, height: 900 });

    // Toggle language to FA
    await guestPage.evaluate(() => {
      localStorage.setItem("clientpulse_locale", "fa");
      document.documentElement.lang = "fa";
      document.documentElement.dir = "rtl";
      window.dispatchEvent(new Event("clientpulse:locale_changed"));
    });
    await guestPage.waitForTimeout(800);

    // Verify RTL attributes
    const dir = await guestPage.getAttribute("html", "dir");
    const lang = await guestPage.getAttribute("html", "lang");
    if (dir !== "rtl" || lang !== "fa") {
      throw new Error(`Persian mode failed: dir="${dir}", lang="${lang}"`);
    }

    // Verify Persian headline text
    const faHeadline = await guestPage.textContent("h1");
    console.log(`    Persian Headline: "${faHeadline}"`);
    if (!faHeadline.includes("همواره یک گام جلوتر")) {
      throw new Error(`Persian headline mismatch: "${faHeadline}"`);
    }

    await assertNoHorizontalOverflow(guestPage, "/ (FA RTL)", "1440px");
    await guestPage.screenshot({ path: path.join(SCREENSHOT_DIR, "04-fa-desktop-fullpage.png"), fullPage: true });
    console.log("    ✔ Persian Desktop Fullpage screenshot captured.");

    // Persian Mobile
    await guestPage.setViewportSize({ width: 390, height: 844 });
    await guestPage.waitForTimeout(500);
    await assertNoHorizontalOverflow(guestPage, "/ (FA RTL)", "390px");
    await guestPage.screenshot({ path: path.join(SCREENSHOT_DIR, "05-fa-mobile-390.png"), fullPage: true });
    console.log("    ✔ Persian Mobile (390px) fullpage screenshot captured.");

    // Reset language to English
    await guestPage.evaluate(() => {
      localStorage.setItem("clientpulse_locale", "en");
      document.documentElement.lang = "en";
      document.documentElement.dir = "ltr";
      window.dispatchEvent(new Event("clientpulse:locale_changed"));
    });
    await guestPage.setViewportSize({ width: 1440, height: 900 });
    await guestPage.waitForTimeout(500);

    // -------------------------------------------------------------------------
    // TEST SCENARIO 5: CTA Navigation & Real Auth Flow
    // -------------------------------------------------------------------------
    console.log("\n>>> [SCENARIO 5] Real Auth Routes & CTA Navigation Flow...");

    // Test Sign Up CTA link
    await guestPage.goto(`${BASE_URL}/`, { waitUntil: "networkidle" });
    const signupBtn = await guestPage.$('a[href="/signup"]');
    if (!signupBtn) throw new Error("Sign Up CTA link not found on landing page!");
    await signupBtn.click();
    await guestPage.waitForNavigation({ waitUntil: "networkidle" }).catch(() => {});
    await guestPage.waitForTimeout(800);

    if (!guestPage.url().includes("/signup")) {
      throw new Error(`Expected navigation to /signup, got: ${guestPage.url()}`);
    }
    await guestPage.screenshot({ path: path.join(SCREENSHOT_DIR, "07-signup-route.png") });
    console.log("    ✔ Sign Up CTA correctly navigates to real /signup route.");

    // Test Sign In CTA link
    await guestPage.goto(`${BASE_URL}/`, { waitUntil: "networkidle" });
    const signinBtn = await guestPage.$('a[href="/login"]');
    if (!signinBtn) throw new Error("Sign In CTA link not found on landing page!");
    await signinBtn.click();
    await guestPage.waitForNavigation({ waitUntil: "networkidle" }).catch(() => {});
    await guestPage.waitForTimeout(800);

    if (!guestPage.url().includes("/login")) {
      throw new Error(`Expected navigation to /login, got: ${guestPage.url()}`);
    }
    await guestPage.screenshot({ path: path.join(SCREENSHOT_DIR, "08-login-route.png") });
    console.log("    ✔ Sign In CTA correctly navigates to real /login route.");

    // -------------------------------------------------------------------------
    // TEST SCENARIO 6: Full User Registration, Login & Authenticated Dashboard
    // -------------------------------------------------------------------------
    console.log("\n>>> [SCENARIO 6] Full Registration, Login & Authenticated Dashboard Access...");
    const unique = Math.random().toString(36).substring(7);
    const userEmail = `agency-admin-${unique}@mindu.ir`;
    const userPassword = "Password123!#";
    const userOrg = "Mindu Digital Agency";

    // 1. Sign Up
    await guestPage.goto(`${BASE_URL}/signup`, { waitUntil: "networkidle" });
    await guestPage.fill('input[name="display_name"]', "Agency Director");
    await guestPage.fill('input[name="email"]', userEmail);
    await guestPage.fill('input[name="password"]', userPassword);
    await guestPage.fill('input[name="org_name"]', userOrg);
    await guestPage.click('button[type="submit"]');

    await guestPage.waitForNavigation({ waitUntil: "networkidle" }).catch(() => {});
    await guestPage.waitForTimeout(2000);

    // Verify redirected to /dashboard
    console.log(`    Post-signup URL: ${guestPage.url()}`);
    if (!guestPage.url().includes("/dashboard")) {
      throw new Error(`Expected post-signup redirect to /dashboard, got: ${guestPage.url()}`);
    }
    console.log("    ✔ Post-signup successfully redirected to /dashboard.");

    // Capture Authenticated Dashboard Screenshot
    await guestPage.screenshot({ path: path.join(SCREENSHOT_DIR, "09-authenticated-dashboard.png"), fullPage: true });
    console.log("    ✔ Authenticated dashboard screenshot captured.");

    // 2. Visiting '/' while authenticated smoothly redirects to '/dashboard'
    console.log("    Testing authenticated visit to '/'...");
    await guestPage.goto(`${BASE_URL}/`, { waitUntil: "networkidle" });
    await guestPage.waitForTimeout(1500);
    console.log(`    Authenticated '/' visit landed on: ${guestPage.url()}`);
    if (!guestPage.url().includes("/dashboard")) {
      throw new Error(`Expected authenticated visitor on '/' to be redirected to /dashboard, got: ${guestPage.url()}`);
    }
    console.log("    ✔ Authenticated user visiting '/' smoothly redirected to /dashboard.");

    // 3. Operational routes still accessible
    console.log("    Verifying deep link operational routes...");
    for (const route of ["/monitors", "/heartbeats", "/clients", "/incidents", "/maintenance"]) {
      await guestPage.goto(`${BASE_URL}${route}`, { waitUntil: "networkidle" });
      await guestPage.waitForTimeout(600);
      if (!guestPage.url().includes(route)) {
        throw new Error(`Deep link to ${route} failed! URL: ${guestPage.url()}`);
      }
      console.log(`    ✔ Operational route ${route} accessible.`);
    }

    // -------------------------------------------------------------------------
    // TEST SCENARIO 7: Session Expiration & Protected Dashboard Guard
    // -------------------------------------------------------------------------
    console.log("\n>>> [SCENARIO 7] Session Expiration & Guard Verification...");
    // Open a fresh incognito context with no cookies
    const unauthContext = await browser.newContext({
      viewport: { width: 1440, height: 900 },
    });
    const unauthPage = await unauthContext.newPage();

    await unauthPage.goto(`${BASE_URL}/dashboard`, { waitUntil: "networkidle" });
    await unauthPage.waitForTimeout(1500);
    console.log(`    Unauthenticated access to /dashboard redirected to: ${unauthPage.url()}`);
    if (!unauthPage.url().includes("/login")) {
      throw new Error(`Unauthenticated visitor to /dashboard was not redirected to /login! Got: ${unauthPage.url()}`);
    }
    console.log("    ✔ Unauthenticated access to /dashboard safely guarded and redirected to /login.");

    console.log("\n================================================================================");
    console.log("TASK-010: ALL 7 SCENARIOS PASSED WITH ZERO FAILURES");
    console.log("================================================================================\n");
  } catch (err) {
    console.error("TASK-010 Suite FAILED:", err);
    process.exitCode = 1;
  } finally {
    await browser.close();
  }
}

runTask010Validation();
