const { chromium } = require("playwright");
const path = require("path");
const fs = require("fs");

const BASE_URL = process.env.BASE_URL || "http://localhost:3000";
const SCREENSHOT_DIR = path.resolve(__dirname, "../../task-report/screenshots/task-008-b1");

if (!fs.existsSync(SCREENSHOT_DIR)) {
  fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
}

async function capture() {
  console.log("Starting visual regression & screenshot capture on:", BASE_URL);
  const browser = await chromium.launch({
    executablePath: "/usr/bin/google-chrome",
    headless: true,
  });

  // Pages to test
  const pagesToTest = [
    { route: "/login", name: "login" },
    { route: "/signup", name: "signup" },
    { route: "/forgot-password", name: "forgot_password" },
  ];

  const viewports = [
    { width: 1440, height: 900, label: "1440" },
    { width: 1280, height: 800, label: "1280" },
    { width: 1024, height: 768, label: "1024" },
    { width: 768, height: 1024, label: "768" },
    { width: 390, height: 844, label: "390" },
    { width: 360, height: 740, label: "360" },
  ];

  for (const p of pagesToTest) {
    for (const locale of ["en", "fa"]) {
      const context = await browser.newContext({
        viewport: { width: 1440, height: 900 },
      });
      const page = await context.newPage();
      
      // Seed locale
      await page.addInitScript((loc) => {
        localStorage.setItem("clientpulse_locale", loc);
      }, locale);

      await page.goto(`${BASE_URL}${p.route}`, { waitUntil: "networkidle" });
      await page.waitForTimeout(500);

      // Verify root html attributes
      const lang = await page.getAttribute("html", "lang");
      const dir = await page.getAttribute("html", "dir");
      console.log(`[${locale}] ${p.route} -> html lang="${lang}" dir="${dir}"`);

      // Capture desktop screenshot
      const shotPath = path.join(SCREENSHOT_DIR, `${p.name}_${locale}_1440.png`);
      await page.screenshot({ path: shotPath, fullPage: true });
      console.log(`Saved screenshot: ${shotPath}`);

      await context.close();
    }
  }

  // Responsive checks on Login page
  for (const vp of viewports) {
    const context = await browser.newContext({
      viewport: { width: vp.width, height: vp.height },
    });
    const page = await context.newPage();
    await page.goto(`${BASE_URL}/login`, { waitUntil: "networkidle" });
    await page.waitForTimeout(500);
    const shotPath = path.join(SCREENSHOT_DIR, `login_responsive_${vp.label}.png`);
    await page.screenshot({ path: shotPath, fullPage: true });
    console.log(`Saved responsive screenshot (${vp.label}px): ${shotPath}`);
    await context.close();
  }

  // Authenticated flow verification and dashboard / monitors / reports capture
  console.log("Registering and logging in test user for authenticated viewports...");
  const authContext = await browser.newContext({
    viewport: { width: 1440, height: 900 },
  });
  const authPage = await authContext.newPage();

  const testId = Date.now().toString().slice(-5);
  const email = `design-test-${testId}@clientpulse.test`;
  const password = "TestPassword123!#";

  await authPage.goto(`${BASE_URL}/signup`, { waitUntil: "networkidle" });
  await authPage.fill('input[name="display_name"]', "Design System Test");
  await authPage.fill('input[name="email"]', email);
  await authPage.fill('input[name="org_name"]', `Design Org ${testId}`);
  await authPage.fill('input[name="password"]', password);
  await authPage.click('button[type="submit"]');

  await authPage.waitForURL((url) => !url.pathname.includes("/signup"), { timeout: 10000 });
  console.log("User authenticated, current URL:", authPage.url());

  const authRoutes = [
    { route: "/", name: "dashboard" },
    { route: "/monitors", name: "monitors" },
    { route: "/reports", name: "reports" },
    { route: "/settings/notifications", name: "notifications" },
  ];

  for (const r of authRoutes) {
    for (const locale of ["en", "fa"]) {
      await authPage.evaluate((loc) => {
        localStorage.setItem("clientpulse_locale", loc);
      }, locale);

      await authPage.goto(`${BASE_URL}${r.route}`, { waitUntil: "networkidle" });
      await authPage.waitForTimeout(600);

      const shotPath = path.join(SCREENSHOT_DIR, `${r.name}_${locale}_1440.png`);
      await authPage.screenshot({ path: shotPath, fullPage: true });
      console.log(`Saved auth screenshot [${locale}]: ${shotPath}`);
    }
  }

  await authContext.close();
  await browser.close();
  console.log("Visual validation complete!");
}

capture().catch((err) => {
  console.error("Capture script error:", err);
  process.exit(1);
});

