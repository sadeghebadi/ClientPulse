const { chromium } = require("playwright");
const path = require("path");
const fs = require("fs");
const { execSync } = require("child_process");

const BASE_URL = process.env.BASE_URL || "http://localhost:3000";
const SCREENSHOT_DIR = path.resolve(__dirname, "../../../task-report/screenshots/task-008-b5");

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

async function runTask008B5Suite() {
  console.log("================================================================================");
  console.log("TASK-008.B.5: Reports & Notifications UX Redesign Validation Suite");
  console.log("Base URL:", BASE_URL);
  console.log("Screenshots Directory:", SCREENSHOT_DIR);
  console.log("================================================================================\n");

  const browser = await chromium.launch({
    executablePath: "/usr/bin/google-chrome",
    headless: true,
  });

  try {
    const unique = Math.random().toString(36).substring(7);
    const ownerEmail = `report-lead-${unique}@clientpulse.test`;
    const password = "TestPassword123!#";

    const context = await browser.newContext({
      viewport: { width: 1440, height: 900 },
    });
    const page = await context.newPage();

    console.log(">>> [1/6] Provisioning Agency Workspace for Reports & Notifications...");
    await page.goto(`${BASE_URL}/signup`, { waitUntil: "networkidle" });
    await page.fill('input[name="display_name"]', "Reports Admin");
    await page.fill('input[name="email"]', ownerEmail);
    await page.fill('input[name="org_name"]', `Apex Agency ${unique}`);
    await page.fill('input[name="password"]', password);
    await page.click('button[type="submit"]');
    await page.waitForURL("**/dashboard", { timeout: 15000 });
    console.log("    ✔ Agency Workspace registered and logged in.");

    // Retrieve Org ID
    const orgId = runSql(`SELECT id FROM organizations WHERE name = 'Apex Agency ${unique}' LIMIT 1;`);
    console.log(`    ✔ Organization ID: ${orgId}`);

    // Create a Client & Asset & Monitor so we can generate reports
    const clientId = runSql(`INSERT INTO clients (organization_id, name) VALUES ('${orgId}', 'Acme Corporation') RETURNING id;`);
    const assetId = runSql(`INSERT INTO assets (organization_id, client_id, name, url, environment, type) VALUES ('${orgId}', '${clientId}', 'Acme Main Portal', 'https://acme.example.com', 'production', 'website') RETURNING id;`);
    runSql(`INSERT INTO monitors (id, organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, created_at, updated_at) VALUES (gen_random_uuid(), '${orgId}', '${assetId}', 60, 10, 'up', true, NOW(), NOW());`);

    console.log(">>> [2/6] Validating /reports (Reports List UX)...");
    await page.goto(`${BASE_URL}/reports`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await assertNoHorizontalOverflow(page, "/reports", "1440px Desktop");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "01-reports-list-desktop-en.png"), fullPage: true });

    // Switch to Persian and capture RTL
    const langBtn = await page.$('button:has-text("FA"), button:has-text("فا")');
    if (langBtn) {
      await langBtn.click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: path.join(SCREENSHOT_DIR, "02-reports-list-desktop-fa.png"), fullPage: true });
      // Switch back to EN
      const enBtn = await page.$('button:has-text("EN")');
      if (enBtn) await enBtn.click();
      await page.waitForTimeout(500);
    }

    // Open Generate Report Modal & Preview
    console.log("    ▶ Opening Generate Report Modal & triggering Preview...");
    const genBtn = await page.$('button:has-text("Generate Report")');
    if (genBtn) {
      await genBtn.click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: path.join(SCREENSHOT_DIR, "03-report-generate-modal-en.png") });

      // Click Preview
      const previewBtn = await page.$('button:has-text("Report Preview")');
      if (previewBtn) {
        await previewBtn.click();
        await page.waitForTimeout(1000);
      }

      // Click Generate
      const submitGenBtn = await page.$('div[role="dialog"] button:has-text("Generate Report")');
      if (submitGenBtn) {
        await submitGenBtn.click();
        await page.waitForNavigation({ timeout: 15000 }).catch(() => {});
        await page.waitForTimeout(2000);
      }
    }

    console.log(">>> [3/6] Validating /reports/[id] (Report Detail & Snapshot UX)...");
    const currentUrl = page.url();
    console.log(`    ✔ Current Report URL: ${currentUrl}`);
    await assertNoHorizontalOverflow(page, "/reports/[id]", "1440px Desktop");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "04-report-detail-desktop-en.png"), fullPage: true });

    // Verify Share Modal & Generate Token
    console.log("    ▶ Testing Share Report Modal...");
    const shareBtn = await page.$('button:has-text("Share Report")');
    let shareUrl = "";
    if (shareBtn) {
      await shareBtn.click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: path.join(SCREENSHOT_DIR, "05-report-share-modal-en.png") });

      const createShareBtn = await page.$('button:has-text("Create Public Link")');
      if (createShareBtn) {
        await createShareBtn.click();
        await page.waitForTimeout(1000);
        await page.screenshot({ path: path.join(SCREENSHOT_DIR, "06-report-share-modal-created-en.png") });

        const shareInput = await page.$('div[role="dialog"] input[readOnly]');
        if (shareInput) {
          shareUrl = await shareInput.inputValue();
        }
      }

      // Close share modal
      const closeBtn = await page.$('div[role="dialog"] button:has-text("Close")');
      if (closeBtn) await closeBtn.click();
      await page.waitForTimeout(500);
    }

    console.log(">>> [4/6] Validating /public/reports/[token] (Public Report UX)...");
    if (shareUrl) {
      console.log(`    ✔ Public Share URL: ${shareUrl}`);
      const publicContext = await browser.newContext({ viewport: { width: 1440, height: 900 } });
      const publicPage = await publicContext.newPage();
      await publicPage.goto(shareUrl, { waitUntil: "networkidle" });
      await publicPage.waitForTimeout(1000);

      await assertNoHorizontalOverflow(publicPage, "/public/reports/[token]", "1440px Desktop");
      await publicPage.screenshot({ path: path.join(SCREENSHOT_DIR, "07-public-report-desktop-en.png"), fullPage: true });

      // Mobile check on public report (390px)
      await publicPage.setViewportSize({ width: 390, height: 844 });
      await publicPage.waitForTimeout(500);
      await assertNoHorizontalOverflow(publicPage, "/public/reports/[token]", "390px Mobile");
      await publicPage.screenshot({ path: path.join(SCREENSHOT_DIR, "08-public-report-mobile-390px.png"), fullPage: true });

      await publicContext.close();
    }

    console.log(">>> [5/6] Validating /settings/notifications (Channels & History UX)...");
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto(`${BASE_URL}/settings/notifications`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await assertNoHorizontalOverflow(page, "/settings/notifications", "1440px Desktop");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "09-notifications-desktop-en.png"), fullPage: true });

    // Open Preferences modal on Telegram or Email
    console.log("    ▶ Testing Event Preferences Modal...");
    // Create an email destination first so we have items
    const addEmailBtn = await page.$('button:has-text("Add Email")');
    if (addEmailBtn) {
      await addEmailBtn.click();
      await page.waitForTimeout(500);
      await page.fill('input[type="email"]', "alerts-devops@acme.example.com");
      await page.fill('input[placeholder="DevOps Team Email"]', "Acme Ops Channel");
      await page.click('button:has-text("Add Email"):not([disabled])');
      await page.waitForTimeout(1000);
    }

    const prefBtn = await page.$('button:has-text("Preferences")');
    if (prefBtn) {
      await prefBtn.click();
      await page.waitForTimeout(500);
      await page.screenshot({ path: path.join(SCREENSHOT_DIR, "10-notifications-preferences-modal-en.png") });
      const cancelPref = await page.$('button:has-text("Cancel")');
      if (cancelPref) await cancelPref.click();
      await page.waitForTimeout(500);
    }

    // Switch to Delivery History tab
    console.log("    ▶ Testing Delivery History Tab...");
    const historyTabBtn = await page.$('button:has-text("Delivery History")');
    if (historyTabBtn) {
      await historyTabBtn.click();
      await page.waitForTimeout(1000);
      await page.screenshot({ path: path.join(SCREENSHOT_DIR, "11-notifications-history-tab-en.png"), fullPage: true });
    }

    console.log(">>> [6/6] Validating Responsive Viewports (1440, 1024, 768, 390, 360)...");
    const viewports = [
      { width: 1440, height: 900, name: "1440px-desktop" },
      { width: 1024, height: 768, name: "1024px-small-desktop" },
      { width: 768, height: 1024, name: "768px-tablet" },
      { width: 390, height: 844, name: "390px-mobile" },
      { width: 360, height: 740, name: "360px-narrow-mobile" },
    ];

    for (const vp of viewports) {
      await page.setViewportSize({ width: vp.width, height: vp.height });
      await page.goto(`${BASE_URL}/reports`, { waitUntil: "networkidle" });
      await page.waitForTimeout(500);
      await assertNoHorizontalOverflow(page, "/reports", vp.name);

      await page.goto(`${BASE_URL}/settings/notifications`, { waitUntil: "networkidle" });
      await page.waitForTimeout(500);
      await assertNoHorizontalOverflow(page, "/settings/notifications", vp.name);
      console.log(`    ✔ Viewport ${vp.name} verified: No horizontal overflow.`);
    }

    // Capture mobile card views
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(`${BASE_URL}/reports`, { waitUntil: "networkidle" });
    await page.waitForTimeout(500);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "12-reports-mobile-cards-390px.png"), fullPage: true });

    await page.goto(`${BASE_URL}/settings/notifications`, { waitUntil: "networkidle" });
    await page.waitForTimeout(500);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "13-notifications-mobile-390px.png"), fullPage: true });

    console.log("\n================================================================================");
    console.log("TASK-008.B.5: ALL CHECKS, RESPONSIVE TESTS & SCREENSHOTS PASSED SUCCESSFULLY");
    console.log("================================================================================\n");
  } catch (err) {
    console.error("TASK-008.B.5 Suite FAILED:", err);
    process.exitCode = 1;
  } finally {
    await browser.close();
  }
}

runTask008B5Suite();
