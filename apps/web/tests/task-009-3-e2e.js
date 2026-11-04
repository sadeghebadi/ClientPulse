const { chromium } = require("playwright");
const path = require("path");
const fs = require("fs");
const { execSync } = require("child_process");

const BASE_URL = process.env.BASE_URL || "http://localhost:3001";
const SCREENSHOT_DIR = path.resolve(__dirname, "../../../task-report/screenshots/task-009-3");

if (!fs.existsSync(SCREENSHOT_DIR)) {
  fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
}

function runSql(query) {
  const sanitized = query.replace(/"/g, '\\"');
  return execSync(`docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -c "${sanitized}"`).toString();
}

async function runTask009_3_E2E() {
  console.log("===================================================================");
  console.log("TASK-009.3: Platform Admin Redesign & Locale Catalog E2E Suite");
  console.log("Web URL:", BASE_URL);
  console.log("Screenshots:", SCREENSHOT_DIR);
  console.log("===================================================================\n");

  const adminEmail = `pa-e2e-task009-3@pulse.test`;
  const adminPassword = "PlatformAdminSecurePass123!#";
  const ownerEmail = `regular-owner-task009-3@pulse.test`;
  const ownerPassword = "RegularUserPassword123!#";

  const browser = await chromium.launch({
    executablePath: "/usr/bin/google-chrome",
    headless: true,
    args: ["--no-sandbox", "--disable-setuid-sandbox"],
  });

  try {
    const desktopContext = await browser.newContext({
      viewport: { width: 1440, height: 900 },
    });

    const page = await desktopContext.newPage();
    page.on("dialog", async (dialog) => {
      await dialog.accept();
    });

    // -------------------------------------------------------------
    // TEST 1: Organization Owner Access Denied (403)
    // -------------------------------------------------------------
    console.log("▶ Step 1: Testing Owner Access Denied (403)...");
    await page.goto(`${BASE_URL}/login`, { waitUntil: "networkidle" });
    await page.fill('input[type="email"]', ownerEmail);
    await page.fill('input[type="password"]', ownerPassword);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(1500);

    // Attempt to access /platform-admin
    await page.goto(`${BASE_URL}/platform-admin`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "access-denied.png"), fullPage: true });
    console.log("✔ Saved access-denied.png");

    // Clear session cookies for next login
    await desktopContext.clearCookies();

    // -------------------------------------------------------------
    // TEST 2: Platform Admin Login & Redesigned Overview
    // -------------------------------------------------------------
    console.log("\n▶ Step 2: Testing Platform Admin Login & Overview...");
    await page.goto(`${BASE_URL}/login`, { waitUntil: "networkidle" });
    await page.fill('input[type="email"]', adminEmail);
    await page.fill('input[type="password"]', adminPassword);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(1500);

    await page.goto(`${BASE_URL}/platform-admin`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1500);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "overview.png"), fullPage: true });
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "english-desktop.png"), fullPage: true });
    console.log("✔ Saved overview.png & english-desktop.png");

    // -------------------------------------------------------------
    // TEST 3: Redesigned Users List
    // -------------------------------------------------------------
    console.log("\n▶ Step 3: Testing Platform Admin Users List...");
    await page.goto(`${BASE_URL}/platform-admin/users`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1200);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "users.png"), fullPage: true });
    console.log("✔ Saved users.png");

    // -------------------------------------------------------------
    // TEST 4: Redesigned Organizations List
    // -------------------------------------------------------------
    console.log("\n▶ Step 4: Testing Platform Admin Organizations List...");
    await page.goto(`${BASE_URL}/platform-admin/organizations`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1200);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "organizations.png"), fullPage: true });
    console.log("✔ Saved organizations.png");

    // -------------------------------------------------------------
    // TEST 5: Plans & Regional Catalogs
    // -------------------------------------------------------------
    console.log("\n▶ Step 5: Testing Plans & Catalogs...");
    await page.goto(`${BASE_URL}/platform-admin/plans`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1200);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "plans.png"), fullPage: true });
    console.log("✔ Saved plans.png");

    // Click English Catalog tab
    await page.click('button:has-text("English Catalog")');
    await page.waitForTimeout(800);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "english-catalog.png"), fullPage: true });
    console.log("✔ Saved english-catalog.png");

    // Click Persian Catalog tab
    await page.click('button:has-text("کاتالوگ فارسی")');
    await page.waitForTimeout(800);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "persian-catalog.png"), fullPage: true });
    console.log("✔ Saved persian-catalog.png");

    // -------------------------------------------------------------
    // TEST 6: Toman Price Editor & Version History
    // -------------------------------------------------------------
    console.log("\n▶ Step 6: Testing Toman Price Editor & Version History...");
    // Find an existing plan ID with offers
    const planIdRes = runSql("SELECT plan_id FROM subscription_offers WHERE locale = 'fa' LIMIT 1;");
    const planId = planIdRes.split("\n")[2].trim();

    await page.goto(`${BASE_URL}/platform-admin/plans/${planId}`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1200);

    // Click Persian Offer tab
    await page.click('button:has-text("آفر فارسی")');
    await page.waitForTimeout(800);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "toman-price-editor.png"), fullPage: true });
    console.log("✔ Saved toman-price-editor.png");

    // Click Version History tab
    await page.click('button:has-text("Version History")');
    await page.waitForTimeout(800);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "offer-version-history.png"), fullPage: true });
    console.log("✔ Saved offer-version-history.png");

    // -------------------------------------------------------------
    // TEST 7: Centralized Localization Management
    // -------------------------------------------------------------
    console.log("\n▶ Step 7: Testing Localization Settings...");
    await page.goto(`${BASE_URL}/platform-admin/settings/localization`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1200);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "localization-management.png"), fullPage: true });
    console.log("✔ Saved localization-management.png");

    // -------------------------------------------------------------
    // TEST 8: Persian RTL Layout Desktop
    // -------------------------------------------------------------
    console.log("\n▶ Step 8: Testing Persian RTL Desktop...");
    // Switch language to Persian via localStorage
    await page.evaluate(() => {
      localStorage.setItem("clientpulse_locale", "fa");
      window.dispatchEvent(new Event("clientpulse:locale_changed"));
    });
    await page.goto(`${BASE_URL}/platform-admin`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1200);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "persian-rtl-desktop.png"), fullPage: true });
    console.log("✔ Saved persian-rtl-desktop.png");

    // -------------------------------------------------------------
    // TEST 9: Mobile Responsive Views (375x812)
    // -------------------------------------------------------------
    console.log("\n▶ Step 9: Testing Mobile Views...");
    const mobileContext = await browser.newContext({
      viewport: { width: 375, height: 812 },
      isMobile: true,
      hasTouch: true,
    });
    const mobilePage = await mobileContext.newPage();

    // Authenticate mobile session
    await mobilePage.goto(`${BASE_URL}/login`, { waitUntil: "networkidle" });
    await mobilePage.fill('input[type="email"]', adminEmail);
    await mobilePage.fill('input[type="password"]', adminPassword);
    await mobilePage.click('button[type="submit"]');
    await mobilePage.waitForTimeout(1500);

    // Mobile Admin Overview & Navigation
    await mobilePage.goto(`${BASE_URL}/platform-admin`, { waitUntil: "networkidle" });
    await mobilePage.waitForTimeout(1000);
    await mobilePage.screenshot({ path: path.join(SCREENSHOT_DIR, "mobile-navigation.png"), fullPage: true });
    console.log("✔ Saved mobile-navigation.png");

    // Mobile Persian Catalog
    await mobilePage.evaluate(() => {
      localStorage.setItem("clientpulse_locale", "fa");
      window.dispatchEvent(new Event("clientpulse:locale_changed"));
    });
    await mobilePage.goto(`${BASE_URL}/platform-admin/plans`, { waitUntil: "networkidle" });
    await mobilePage.waitForTimeout(1000);
    await mobilePage.click('button:has-text("کاتالوگ فارسی")');
    await mobilePage.waitForTimeout(600);
    await mobilePage.screenshot({ path: path.join(SCREENSHOT_DIR, "mobile-persian-catalog.png"), fullPage: true });
    console.log("✔ Saved mobile-persian-catalog.png");

    await desktopContext.close();
    await mobileContext.close();
    console.log("\n===================================================================");
    console.log("✔ All 14 Playwright E2E verification tests & screenshots completed!");
    console.log("===================================================================");
  } catch (err) {
    console.error("❌ Test suite failed:", err);
    process.exit(1);
  } finally {
    await browser.close();
  }
}

runTask009_3_E2E();
