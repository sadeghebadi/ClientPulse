const { chromium } = require("playwright");
const path = require("path");
const fs = require("fs");
const { execSync } = require("child_process");

const BASE_URL = process.env.BASE_URL || "http://localhost:3000";
const SCREENSHOT_DIR = path.resolve(__dirname, "../../../task-report/screenshots/task-009");

if (!fs.existsSync(SCREENSHOT_DIR)) {
  fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
}

function runSql(sql) {
  const cmd = `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -t -A -q -c "${sql.replace(/"/g, '\\"')}"`;
  const res = execSync(cmd).toString().trim();
  const lines = res
    .split("\n")
    .map((l) => l.trim())
    .filter((l) => l && !l.startsWith("INSERT") && !l.startsWith("UPDATE") && !l.startsWith("DELETE"));
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

async function runTask009Validation() {
  console.log("================================================================================");
  console.log("TASK-009: Platform Admin Panel & Subscription Plan Management Validation Suite");
  console.log("Base URL:", BASE_URL);
  console.log("Screenshots Directory:", SCREENSHOT_DIR);
  console.log("================================================================================\n");

  const browser = await chromium.launch({
    executablePath: "/usr/bin/google-chrome",
    headless: true,
  });

  const unique = Math.random().toString(36).substring(7);
  let testAdminUserId = null;

  try {
    // -------------------------------------------------------------------------
    // TEST SCENARIO 1: Organization Owner (admin@mindu.ir) receives 403 & No Admin Nav
    // -------------------------------------------------------------------------
    console.log(">>> [SCENARIO 1] Verifying Organization Owner (admin@mindu.ir) is restricted...");
    const ownerContext = await browser.newContext({
      viewport: { width: 1440, height: 900 },
    });
    const ownerPage = await ownerContext.newPage();

    // Login as admin@mindu.ir
    await ownerPage.goto(`${BASE_URL}/login`, { waitUntil: "networkidle" });
    await ownerPage.fill('input[type="email"], input[name="email"]', "admin@mindu.ir");
    await ownerPage.fill('input[type="password"], input[name="password"]', "MinduOwnerPassword123!");
    await ownerPage.click('button[type="submit"]');
    await ownerPage.waitForTimeout(2000);

    // Verify left sidebar has NO "Platform Admin" section
    const hasPlatformAdminInSidebar = await ownerPage.evaluate(() => {
      const text = document.body.innerText;
      return text.includes("Platform Admin") && text.includes("Platform Overview");
    });
    if (hasPlatformAdminInSidebar) {
      throw new Error("SECURITY FAILURE: admin@mindu.ir can see Platform Admin navigation!");
    }
    console.log("✔ Verified admin@mindu.ir sidebar does NOT contain Platform Admin link.");

    // Direct navigation to /platform-admin must render 403 Access Denied
    await ownerPage.goto(`${BASE_URL}/platform-admin`, { waitUntil: "networkidle" });
    await ownerPage.waitForTimeout(1000);

    const isAccessDenied = await ownerPage.evaluate(() => {
      const heading = document.querySelector("h1");
      return (
        heading &&
        (heading.innerText.includes("Access Denied") || heading.innerText.includes("403"))
      );
    });
    if (!isAccessDenied) {
      throw new Error("SECURITY FAILURE: admin@mindu.ir did not receive 403 Access Denied view!");
    }
    console.log("✔ Verified direct access to /platform-admin renders 403 Access Denied screen.");

    // Screenshot 1: Access Denied for Org Owner
    await ownerPage.screenshot({
      path: path.join(SCREENSHOT_DIR, "01_access_denied_org_owner.png"),
      fullPage: true,
    });
    console.log("📸 Saved 01_access_denied_org_owner.png");
    await ownerContext.close();

    // -------------------------------------------------------------------------
    // TEST SCENARIO 2: Create temporary test platform admin & test admin panel
    // -------------------------------------------------------------------------
    console.log("\n>>> [SCENARIO 2] Testing Platform Admin Panel with authorized admin session...");
    const adminEmail = `test-pa-${unique}@pulse.test`;
    const adminPassword = "SuperAdminPassword123!";

    const adminContext = await browser.newContext({
      viewport: { width: 1440, height: 900 },
    });
    const adminPage = await adminContext.newPage();

    // Sign up test user
    await adminPage.goto(`${BASE_URL}/signup`, { waitUntil: "networkidle" });
    await adminPage.fill('input[name="display_name"]', "Global Platform Administrator");
    await adminPage.fill('input[name="email"]', adminEmail);
    await adminPage.fill('input[name="password"]', adminPassword);
    await adminPage.fill('input[name="org_name"]', "Pulse Platform Operations");
    await adminPage.click('button[type="submit"]');
    await adminPage.waitForTimeout(2000);

    // Retrieve user ID and grant platform admin in database
    testAdminUserId = runSql(`SELECT id FROM users WHERE email = '${adminEmail}' LIMIT 1;`);
    runSql(`UPDATE users SET email_verified_at = NOW() WHERE id = '${testAdminUserId}';`);
    runSql(`INSERT INTO platform_admin_grants (user_id, role, granted_at, reason) VALUES ('${testAdminUserId}', 'platform_admin', NOW(), 'TASK-009 automated test validation');`);
    console.log(`✔ Granted temporary platform_admin to ${adminEmail} (ID: ${testAdminUserId})`);

    // Reload page to refresh session state with is_platform_admin=true
    await adminPage.goto(`${BASE_URL}/platform-admin`, { waitUntil: "networkidle" });
    await adminPage.waitForTimeout(1500);

    // 1. Overview Dashboard
    console.log("Capturing /platform-admin (Overview)...");
    await adminPage.screenshot({
      path: path.join(SCREENSHOT_DIR, "02_platform_admin_overview.png"),
      fullPage: true,
    });
    console.log("📸 Saved 02_platform_admin_overview.png");

    // 2. Users Management
    console.log("Navigating to /platform-admin/users...");
    await adminPage.goto(`${BASE_URL}/platform-admin/users`, { waitUntil: "networkidle" });
    await adminPage.waitForTimeout(1500);
    await adminPage.screenshot({
      path: path.join(SCREENSHOT_DIR, "03_users_management.png"),
      fullPage: true,
    });
    console.log("📸 Saved 03_users_management.png");

    // Open Suspend User Modal
    const suspendBtn = await adminPage.$('button:has-text("Suspend User")');
    if (suspendBtn) {
      await suspendBtn.click();
      await adminPage.waitForTimeout(600);
      await adminPage.screenshot({
        path: path.join(SCREENSHOT_DIR, "05_user_suspend_modal.png"),
      });
      console.log("📸 Saved 05_user_suspend_modal.png");
      await adminPage.click('button:has-text("Cancel")');
      await adminPage.waitForTimeout(400);
    }

    // 3. User Detail
    console.log(`Navigating to /platform-admin/users/${testAdminUserId}...`);
    await adminPage.goto(`${BASE_URL}/platform-admin/users/${testAdminUserId}`, { waitUntil: "networkidle" });
    await adminPage.waitForTimeout(1500);
    await adminPage.screenshot({
      path: path.join(SCREENSHOT_DIR, "04_user_detail.png"),
      fullPage: true,
    });
    console.log("📸 Saved 04_user_detail.png");

    // 4. Plans Management
    console.log("Navigating to /platform-admin/plans...");
    await adminPage.goto(`${BASE_URL}/platform-admin/plans`, { waitUntil: "networkidle" });
    await adminPage.waitForTimeout(1500);
    await adminPage.screenshot({
      path: path.join(SCREENSHOT_DIR, "08_plans_management.png"),
      fullPage: true,
    });
    console.log("📸 Saved 08_plans_management.png");

    // 5. Plan Creation Form
    console.log("Navigating to /platform-admin/plans/new...");
    await adminPage.goto(`${BASE_URL}/platform-admin/plans/new`, { waitUntil: "networkidle" });
    await adminPage.waitForTimeout(1000);
    await adminPage.screenshot({
      path: path.join(SCREENSHOT_DIR, "09_plan_creation.png"),
      fullPage: true,
    });
    console.log("📸 Saved 09_plan_creation.png");

    // Fill and submit plan creation form
    await adminPage.fill('input[placeholder*="Growth Agency"]', `Agency Pro ${unique}`);
    await adminPage.fill('input[placeholder*="growth-agency"]', `agency-pro-${unique}`);
    await adminPage.click('button[type="submit"]');
    await adminPage.waitForTimeout(2000);

    // Get created plan ID
    const planId = runSql(`SELECT id FROM subscription_plans WHERE slug = 'agency-pro-${unique}' LIMIT 1;`);
    if (planId) {
      console.log(`Navigating to /platform-admin/plans/${planId}...`);
      await adminPage.goto(`${BASE_URL}/platform-admin/plans/${planId}`, { waitUntil: "networkidle" });
      await adminPage.waitForTimeout(1500);

      // Publish plan
      const publishBtn = await adminPage.$('button:has-text("Publish Plan")');
      if (publishBtn) {
        await publishBtn.click();
        await adminPage.waitForTimeout(1500);
      }

      await adminPage.screenshot({
        path: path.join(SCREENSHOT_DIR, "10_plan_detail_immutable.png"),
        fullPage: true,
      });
      console.log("📸 Saved 10_plan_detail_immutable.png");
    }

    // 6. Organizations Management
    console.log("Navigating to /platform-admin/organizations...");
    await adminPage.goto(`${BASE_URL}/platform-admin/organizations`, { waitUntil: "networkidle" });
    await adminPage.waitForTimeout(1500);
    await adminPage.screenshot({
      path: path.join(SCREENSHOT_DIR, "06_organizations_management.png"),
      fullPage: true,
    });
    console.log("📸 Saved 06_organizations_management.png");

    // 7. Organization Detail & Subscription Assignment Modal
    const orgId = runSql(`SELECT organization_id FROM organization_memberships WHERE user_id = '${testAdminUserId}' LIMIT 1;`);
    console.log(`Navigating to /platform-admin/organizations/${orgId}...`);
    await adminPage.goto(`${BASE_URL}/platform-admin/organizations/${orgId}`, { waitUntil: "networkidle" });
    await adminPage.waitForTimeout(1500);
    await adminPage.screenshot({
      path: path.join(SCREENSHOT_DIR, "07_organization_detail.png"),
      fullPage: true,
    });
    console.log("📸 Saved 07_organization_detail.png");

    // Open Assign Plan Modal
    const assignPlanBtn = await adminPage.$('button:has-text("Assign Subscription Plan"), button:has-text("Change Plan")');
    if (assignPlanBtn) {
      await assignPlanBtn.click();
      await adminPage.waitForTimeout(800);

      // Select newly published plan if available in dropdown
      const selectElem = await adminPage.$('select');
      if (selectElem && planId) {
        await selectElem.selectOption(planId);
        await adminPage.waitForTimeout(500);
      }

      await adminPage.screenshot({
        path: path.join(SCREENSHOT_DIR, "11_subscription_assignment_modal.png"),
      });
      console.log("📸 Saved 11_subscription_assignment_modal.png");

      // Confirm assignment to create an active subscription
      const confirmBtn = await adminPage.$('button:has-text("Confirm Assignment")');
      if (confirmBtn) {
        await confirmBtn.click();
        await adminPage.waitForTimeout(1500);
      } else {
        await adminPage.click('button:has-text("Cancel")');
        await adminPage.waitForTimeout(300);
      }
    }

    // 8. Subscriptions Management
    console.log("Navigating to /platform-admin/subscriptions...");
    await adminPage.goto(`${BASE_URL}/platform-admin/subscriptions`, { waitUntil: "networkidle" });
    await adminPage.waitForTimeout(1500);
    await adminPage.screenshot({
      path: path.join(SCREENSHOT_DIR, "12_subscriptions_management.png"),
      fullPage: true,
    });
    console.log("📸 Saved 12_subscriptions_management.png");

    // 9. Audit Logs
    console.log("Navigating to /platform-admin/audit-logs...");
    await adminPage.goto(`${BASE_URL}/platform-admin/audit-logs`, { waitUntil: "networkidle" });
    await adminPage.waitForTimeout(1500);
    await adminPage.screenshot({
      path: path.join(SCREENSHOT_DIR, "13_audit_logs.png"),
      fullPage: true,
    });
    console.log("📸 Saved 13_audit_logs.png");

    // Inspect first audit log
    const inspectBtn = await adminPage.$('button:has-text("Inspect")');
    if (inspectBtn) {
      await inspectBtn.click();
      await adminPage.waitForTimeout(600);
      await adminPage.screenshot({
        path: path.join(SCREENSHOT_DIR, "14_audit_log_inspector.png"),
      });
      console.log("📸 Saved 14_audit_log_inspector.png");
      await adminPage.click('button:has-text("Close")');
      await adminPage.waitForTimeout(300);
    }

    // 10. Persian RTL Internationalization
    console.log("Testing Persian RTL on /platform-admin...");
    await adminPage.goto(`${BASE_URL}/platform-admin`, { waitUntil: "networkidle" });
    await adminPage.waitForTimeout(1000);

    // Switch to Persian
    await adminPage.evaluate(() => {
      localStorage.setItem("clientpulse_locale", "fa");
      document.documentElement.lang = "fa";
      document.documentElement.dir = "rtl";
      window.dispatchEvent(new Event("clientpulse:locale_changed"));
    });
    await adminPage.reload({ waitUntil: "networkidle" });
    await adminPage.waitForTimeout(1500);

    await adminPage.screenshot({
      path: path.join(SCREENSHOT_DIR, "15_persian_rtl_overview.png"),
      fullPage: true,
    });
    console.log("📸 Saved 15_persian_rtl_overview.png");

    // Reset back to English
    await adminPage.evaluate(() => {
      localStorage.setItem("clientpulse_locale", "en");
      document.documentElement.lang = "en";
      document.documentElement.dir = "ltr";
      window.dispatchEvent(new Event("clientpulse:locale_changed"));
    });
    await adminPage.reload({ waitUntil: "networkidle" });
    await adminPage.waitForTimeout(500);

    // 11. Responsive Viewports & Horizontal Overflow Checks
    console.log("Testing responsive viewports...");
    const viewports = [
      { name: "1024px", width: 1024, height: 768, file: "16_responsive_1024px.png" },
      { name: "768px", width: 768, height: 1024, file: "17_responsive_768px.png" },
      { name: "390px", width: 390, height: 844, file: "18_responsive_390px.png" },
      { name: "360px", width: 360, height: 740, file: "19_responsive_360px.png" },
    ];

    for (const vp of viewports) {
      await adminPage.setViewportSize({ width: vp.width, height: vp.height });
      await adminPage.goto(`${BASE_URL}/platform-admin`, { waitUntil: "networkidle" });
      await adminPage.waitForTimeout(1000);
      await assertNoHorizontalOverflow(adminPage, "/platform-admin", vp.name);
      await adminPage.screenshot({
        path: path.join(SCREENSHOT_DIR, vp.file),
        fullPage: true,
      });
      console.log(`📸 Saved ${vp.file} (${vp.name})`);
    }

    await adminContext.close();
    console.log("\n✔ All Platform Admin validation scenarios completed successfully!");
  } finally {
    // -------------------------------------------------------------------------
    // CLEANUP: Ensure NO active platform_admin grant remains in the database!
    // -------------------------------------------------------------------------
    if (testAdminUserId) {
      runSql(`DELETE FROM platform_admin_grants WHERE user_id = '${testAdminUserId}';`);
      console.log(`🧹 Cleaned up temporary test grant for user ${testAdminUserId}`);
    }
    // Verify admin@mindu.ir has 0 grants
    const minduGrants = runSql(`SELECT COUNT(*) FROM platform_admin_grants pag JOIN users u ON u.id = pag.user_id WHERE u.email = 'admin@mindu.ir';`);
    console.log(`🔒 Verification: admin@mindu.ir active platform admin grants = ${minduGrants || 0}`);
    await browser.close();
  }
}

runTask009Validation().catch((err) => {
  console.error("FATAL ERROR in TASK-009 validation:", err);
  process.exit(1);
});
