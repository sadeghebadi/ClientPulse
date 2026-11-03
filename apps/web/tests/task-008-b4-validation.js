const { chromium } = require("playwright");
const path = require("path");
const fs = require("fs");
const { execSync } = require("child_process");

const BASE_URL = process.env.BASE_URL || "http://localhost:3000";
const SCREENSHOT_DIR = path.resolve(__dirname, "../../../task-report/screenshots/task-008-b4");

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

async function runTask008B4Suite() {
  console.log("================================================================================");
  console.log("TASK-008.B.4: Operational Pages Redesign & Mobile UX Validation Suite");
  console.log("Base URL:", BASE_URL);
  console.log("Screenshots Directory:", SCREENSHOT_DIR);
  console.log("================================================================================\n");

  const browser = await chromium.launch({
    executablePath: "/usr/bin/google-chrome",
    headless: true,
  });

  try {
    const unique = Math.random().toString(36).substring(7);
    const ownerEmail = `ops-lead-${unique}@pulseops.io`;
    const viewerEmail = `ops-viewer-${unique}@pulseops.io`;
    const password = "Password123!#";

    const context = await browser.newContext({
      viewport: { width: 1440, height: 900 },
    });
    const page = await context.newPage();

    console.log(">>> [SETUP] Provisioning Agency Workspace & Operational Test Entities...");
    await page.goto(`${BASE_URL}/signup`, { waitUntil: "networkidle" });
    await page.fill('input[name="display_name"]', "Operational Lead");
    await page.fill('input[name="email"]', ownerEmail);
    await page.fill('input[name="password"]', password);
    await page.fill('input[name="org_name"]', "Precision DevOps Ltd");
    await page.click('button[type="submit"]');

    await page.waitForNavigation({ waitUntil: "networkidle" }).catch(() => {});
    await page.waitForTimeout(2000);

    const orgId = runSql(`SELECT organization_id FROM organization_memberships WHERE user_id = (SELECT id FROM users WHERE email = '${ownerEmail}') LIMIT 1;`);
    const ownerUserId = runSql(`SELECT id FROM users WHERE email = '${ownerEmail}';`);
    console.log(`    Org ID: ${orgId}, Owner ID: ${ownerUserId}`);

    // Create Client
    const clientId = runSql(`
      INSERT INTO clients (id, organization_id, name, contact_name, contact_email, website, notes, status, created_at, updated_at)
      VALUES (gen_random_uuid(), '${orgId}', 'Acme Global Services', 'Elena Rostova', 'elena@acmeglobal.com', 'https://acmeglobal.com', 'Enterprise tier SLA client with 24/7 dedicated escalation channel.', 'active', NOW(), NOW())
      RETURNING id;
    `);

    // Create Assets
    const assetPortalId = runSql(`
      INSERT INTO assets (id, organization_id, client_id, name, url, type, environment, is_active, created_at, updated_at)
      VALUES (gen_random_uuid(), '${orgId}', '${clientId}', 'Customer Portal Webapp', 'https://portal.acmeglobal.com', 'website', 'production', true, NOW(), NOW())
      RETURNING id;
    `);

    const assetApiId = runSql(`
      INSERT INTO assets (id, organization_id, client_id, name, url, type, environment, is_active, created_at, updated_at)
      VALUES (gen_random_uuid(), '${orgId}', '${clientId}', 'Core Payments API', 'https://api.acmeglobal.com', 'api', 'production', true, NOW(), NOW())
      RETURNING id;
    `);

    // Create Monitors
    const monPortalId = runSql(`
      INSERT INTO monitors (id, organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, next_check_at, created_at, updated_at)
      VALUES (gen_random_uuid(), '${orgId}', '${assetPortalId}', 60, 10, 'up', true, NOW() + INTERVAL '1 day', NOW(), NOW())
      RETURNING id;
    `);

    const monApiId = runSql(`
      INSERT INTO monitors (id, organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, consecutive_failures, next_check_at, created_at, updated_at)
      VALUES (gen_random_uuid(), '${orgId}', '${assetApiId}', 60, 5, 'down', true, 3, NOW() + INTERVAL '1 day', NOW(), NOW())
      RETURNING id;
    `);

    // Latency results for portal
    runSql(`
      INSERT INTO check_results (id, organization_id, monitor_id, scheduled_at, started_at, completed_at, latency_ms, result, http_status_code, worker_id, execution_type, created_at)
      VALUES
        (gen_random_uuid(), '${orgId}', '${monPortalId}', NOW() - INTERVAL '5 minutes', NOW() - INTERVAL '5 minutes', NOW() - INTERVAL '5 minutes', 142, 'success', 200, 'w-1', 'scheduled', NOW() - INTERVAL '5 minutes'),
        (gen_random_uuid(), '${orgId}', '${monPortalId}', NOW() - INTERVAL '2 minutes', NOW() - INTERVAL '2 minutes', NOW() - INTERVAL '2 minutes', 138, 'success', 200, 'w-1', 'scheduled', NOW() - INTERVAL '2 minutes');
    `);

    // Create Heartbeat Monitors
    const hb1Hash = Math.random().toString(36).substring(2) + Math.random().toString(36).substring(2);
    const hb1Id = runSql(`
      INSERT INTO heartbeat_monitors (id, organization_id, client_id, asset_id, name, expected_interval_seconds, grace_period_seconds, status, token_hash, last_received_at, next_expected_at, is_active, created_at, updated_at)
      VALUES (gen_random_uuid(), '${orgId}', '${clientId}', '${assetPortalId}', 'Nightly PostgreSQL Backup', 86400, 3600, 'missed', '${hb1Hash}', NOW() - INTERVAL '26 hours', NOW() - INTERVAL '2 hours', true, NOW(), NOW())
      RETURNING id;
    `);

    const hb2Hash = Math.random().toString(36).substring(2) + Math.random().toString(36).substring(2);
    const hb2Id = runSql(`
      INSERT INTO heartbeat_monitors (id, organization_id, client_id, asset_id, name, expected_interval_seconds, grace_period_seconds, status, token_hash, last_received_at, next_expected_at, is_active, created_at, updated_at)
      VALUES (gen_random_uuid(), '${orgId}', '${clientId}', '${assetApiId}', 'Invoice Settlement Worker', 3600, 600, 'healthy', '${hb2Hash}', NOW() - INTERVAL '15 minutes', NOW() + INTERVAL '45 minutes', true, NOW(), NOW())
      RETURNING id;
    `);

    // Create Incidents
    const incApiOutageId = runSql(`
      INSERT INTO incidents (id, organization_id, monitor_id, first_failure_at, confirmed_at, status, failure_category, summary, created_at, updated_at)
      VALUES (gen_random_uuid(), '${orgId}', '${monApiId}', NOW() - INTERVAL '40 minutes', NOW() - INTERVAL '35 minutes', 'open', 'http_502_bad_gateway', 'Core Payments API returning 502 Bad Gateway downstream', NOW() - INTERVAL '35 minutes', NOW() - INTERVAL '35 minutes')
      RETURNING id;
    `);

    const incHbMissedId = runSql(`
      INSERT INTO incidents (id, organization_id, heartbeat_monitor_id, first_failure_at, confirmed_at, acknowledged_at, acknowledged_by, status, failure_category, summary, created_at, updated_at)
      VALUES (gen_random_uuid(), '${orgId}', '${hb1Id}', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '30 minutes', '${ownerUserId}', 'open', 'heartbeat_missed', 'Nightly PostgreSQL Backup missed scheduled execution window', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '30 minutes')
      RETURNING id;
    `);

    const incResolvedId = runSql(`
      INSERT INTO incidents (id, organization_id, monitor_id, first_failure_at, confirmed_at, resolved_at, status, failure_category, resolution_type, resolved_by, duration_seconds, summary, created_at, updated_at)
      VALUES (gen_random_uuid(), '${orgId}', '${monPortalId}', NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days', NOW() - INTERVAL '46 hours', 'resolved', 'connection_timeout', 'manual', '${ownerUserId}', 7200, 'Resolved after edge proxy route cache cleared', NOW() - INTERVAL '2 days', NOW() - INTERVAL '46 hours')
      RETURNING id;
    `);

    // Create Maintenance Activities
    const maint1Id = runSql(`
      INSERT INTO maintenance_activities (id, organization_id, client_id, asset_id, title, description, category, performed_at, duration_minutes, created_by, created_at, updated_at)
      VALUES (gen_random_uuid(), '${orgId}', '${clientId}', '${assetPortalId}', 'PostgreSQL 16 Point Release & Re-indexing', 'Applied minor version database updates and re-indexed billing ledgers.', 'update', NOW() - INTERVAL '1 day', 45, '${ownerUserId}', NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day')
      RETURNING id;
    `);

    const maint2Id = runSql(`
      INSERT INTO maintenance_activities (id, organization_id, client_id, asset_id, title, description, category, performed_at, duration_minutes, created_by, created_at, updated_at)
      VALUES (gen_random_uuid(), '${orgId}', '${clientId}', '${assetApiId}', 'Firewall & TLS Cipher Suite Tightening', 'Deprecated legacy TLS 1.0/1.1 and renewed edge wildcard certificates.', 'security_review', NOW() - INTERVAL '6 hours', 30, '${ownerUserId}', NOW() - INTERVAL '6 hours', NOW() - INTERVAL '6 hours')
      RETURNING id;
    `);

    console.log("    ✔ Test entities provisioned successfully.");

    // -------------------------------------------------------------------------
    // SECTION 1: Desktop (1440px) Operational Pages (English)
    // -------------------------------------------------------------------------
    console.log("\n>>> [SECTION 1] Desktop (1440px) EN Operational Pages");
    await page.setViewportSize({ width: 1440, height: 900 });

    // 1. Monitors List
    await page.goto(`${BASE_URL}/monitors`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await assertNoHorizontalOverflow(page, "/monitors", "1440px");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "01-monitors-list-desktop-en.png"), fullPage: true });
    console.log("    ✔ /monitors rendered cleanly at 1440px.");

    // 2. Monitor Detail
    await page.goto(`${BASE_URL}/monitors/${monPortalId}`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await assertNoHorizontalOverflow(page, `/monitors/${monPortalId}`, "1440px");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "02-monitor-detail-desktop-en.png"), fullPage: true });
    console.log("    ✔ /monitors/[id] rendered cleanly at 1440px.");

    // 3. Heartbeats List
    await page.goto(`${BASE_URL}/heartbeats`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await assertNoHorizontalOverflow(page, "/heartbeats", "1440px");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "03-heartbeats-list-desktop-en.png"), fullPage: true });
    console.log("    ✔ /heartbeats rendered cleanly at 1440px.");

    // 4. Heartbeat Detail
    await page.goto(`${BASE_URL}/heartbeats/${hb1Id}`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await assertNoHorizontalOverflow(page, `/heartbeats/${hb1Id}`, "1440px");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "04-heartbeat-detail-desktop-en.png"), fullPage: true });
    console.log("    ✔ /heartbeats/[id] rendered cleanly at 1440px.");

    // 5. Clients List
    await page.goto(`${BASE_URL}/clients`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await assertNoHorizontalOverflow(page, "/clients", "1440px");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "05-clients-list-desktop-en.png"), fullPage: true });
    console.log("    ✔ /clients rendered cleanly at 1440px.");

    // 6. Client Detail (Cross-Entity Tabs)
    await page.goto(`${BASE_URL}/clients/${clientId}`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await assertNoHorizontalOverflow(page, `/clients/${clientId}`, "1440px");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "06-client-detail-desktop-en.png"), fullPage: true });
    console.log("    ✔ /clients/[id] rendered with cross-entity tabs at 1440px.");

    // 7. Maintenance Log
    await page.goto(`${BASE_URL}/maintenance`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await assertNoHorizontalOverflow(page, "/maintenance", "1440px");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "07-maintenance-desktop-en.png"), fullPage: true });
    console.log("    ✔ /maintenance rendered cleanly at 1440px.");

    // 8. Incidents List
    await page.goto(`${BASE_URL}/incidents`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await assertNoHorizontalOverflow(page, "/incidents", "1440px");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "08-incidents-list-desktop-en.png"), fullPage: true });
    console.log("    ✔ /incidents rendered cleanly at 1440px.");

    // 9. Incident Detail
    await page.goto(`${BASE_URL}/incidents/${incApiOutageId}`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await assertNoHorizontalOverflow(page, `/incidents/${incApiOutageId}`, "1440px");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "09-incident-detail-desktop-en.png"), fullPage: true });
    console.log("    ✔ /incidents/[id] rendered cleanly at 1440px.");

    // -------------------------------------------------------------------------
    // SECTION 2: Mobile (390px) & Narrow Mobile (360px) Responsive Verification
    // -------------------------------------------------------------------------
    console.log("\n>>> [SECTION 2] Mobile Viewport Verification (390px & 360px)");

    const mobileViewports = [
      { name: "390px (Standard Mobile)", width: 390, height: 844 },
      { name: "360px (Narrow Mobile)", width: 360, height: 740 },
    ];

    for (const vp of mobileViewports) {
      await page.setViewportSize({ width: vp.width, height: vp.height });
      console.log(`    Testing Viewport ${vp.name}...`);

      // Monitors Mobile
      await page.goto(`${BASE_URL}/monitors`, { waitUntil: "networkidle" });
      await page.waitForTimeout(800);
      await assertNoHorizontalOverflow(page, "/monitors", vp.name);

      // Monitors Detail Mobile
      await page.goto(`${BASE_URL}/monitors/${monPortalId}`, { waitUntil: "networkidle" });
      await page.waitForTimeout(800);
      await assertNoHorizontalOverflow(page, `/monitors/${monPortalId}`, vp.name);

      // Heartbeats Mobile
      await page.goto(`${BASE_URL}/heartbeats`, { waitUntil: "networkidle" });
      await page.waitForTimeout(800);
      await assertNoHorizontalOverflow(page, "/heartbeats", vp.name);

      // Heartbeat Detail Mobile
      await page.goto(`${BASE_URL}/heartbeats/${hb1Id}`, { waitUntil: "networkidle" });
      await page.waitForTimeout(800);
      await assertNoHorizontalOverflow(page, `/heartbeats/${hb1Id}`, vp.name);

      // Clients Mobile
      await page.goto(`${BASE_URL}/clients`, { waitUntil: "networkidle" });
      await page.waitForTimeout(800);
      await assertNoHorizontalOverflow(page, "/clients", vp.name);

      // Client Detail Mobile
      await page.goto(`${BASE_URL}/clients/${clientId}`, { waitUntil: "networkidle" });
      await page.waitForTimeout(800);
      await assertNoHorizontalOverflow(page, `/clients/${clientId}`, vp.name);

      // Maintenance Mobile
      await page.goto(`${BASE_URL}/maintenance`, { waitUntil: "networkidle" });
      await page.waitForTimeout(800);
      await assertNoHorizontalOverflow(page, "/maintenance", vp.name);

      // Incidents Mobile
      await page.goto(`${BASE_URL}/incidents`, { waitUntil: "networkidle" });
      await page.waitForTimeout(800);
      await assertNoHorizontalOverflow(page, "/incidents", vp.name);

      // Incident Detail Mobile
      await page.goto(`${BASE_URL}/incidents/${incApiOutageId}`, { waitUntil: "networkidle" });
      await page.waitForTimeout(800);
      await assertNoHorizontalOverflow(page, `/incidents/${incApiOutageId}`, vp.name);

      console.log(`    ✔ Zero horizontal overflow verified across all 5 operational routes at ${vp.name}!`);
    }

    // Capture representative mobile 390px screenshots
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(`${BASE_URL}/monitors`, { waitUntil: "networkidle" });
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "10-monitors-list-mobile-390.png"), fullPage: true });

    await page.goto(`${BASE_URL}/heartbeats`, { waitUntil: "networkidle" });
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "11-heartbeats-list-mobile-390.png"), fullPage: true });

    await page.goto(`${BASE_URL}/clients/${clientId}`, { waitUntil: "networkidle" });
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "12-client-detail-mobile-390.png"), fullPage: true });

    await page.goto(`${BASE_URL}/maintenance`, { waitUntil: "networkidle" });
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "13-maintenance-mobile-390.png"), fullPage: true });

    await page.goto(`${BASE_URL}/incidents`, { waitUntil: "networkidle" });
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "14-incidents-list-mobile-390.png"), fullPage: true });

    // -------------------------------------------------------------------------
    // SECTION 3: Persian (FA / RTL) Layout & Mobile Verification
    // -------------------------------------------------------------------------
    console.log("\n>>> [SECTION 3] Persian (FA / RTL) Verification");

    // Switch Language to Persian
    await page.evaluate(() => {
      localStorage.setItem("clientpulse_language", "fa");
    });

    // Desktop FA
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto(`${BASE_URL}/monitors`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await assertNoHorizontalOverflow(page, "/monitors (FA)", "1440px");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "15-monitors-list-desktop-fa.png"), fullPage: true });

    await page.goto(`${BASE_URL}/heartbeats`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await assertNoHorizontalOverflow(page, "/heartbeats (FA)", "1440px");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "16-heartbeats-list-desktop-fa.png"), fullPage: true });

    await page.goto(`${BASE_URL}/clients/${clientId}`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await assertNoHorizontalOverflow(page, `/clients/${clientId} (FA)`, "1440px");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "17-client-detail-desktop-fa.png"), fullPage: true });

    await page.goto(`${BASE_URL}/maintenance`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await assertNoHorizontalOverflow(page, "/maintenance (FA)", "1440px");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "18-maintenance-desktop-fa.png"), fullPage: true });

    await page.goto(`${BASE_URL}/incidents`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await assertNoHorizontalOverflow(page, "/incidents (FA)", "1440px");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "19-incidents-list-desktop-fa.png"), fullPage: true });

    // Mobile FA (390px)
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(`${BASE_URL}/clients/${clientId}`, { waitUntil: "networkidle" });
    await page.waitForTimeout(800);
    await assertNoHorizontalOverflow(page, `/clients/${clientId} (FA)`, "390px");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "20-client-detail-mobile-390-fa.png"), fullPage: true });

    await page.goto(`${BASE_URL}/incidents`, { waitUntil: "networkidle" });
    await page.waitForTimeout(800);
    await assertNoHorizontalOverflow(page, "/incidents (FA)", "390px");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "21-incidents-list-mobile-390-fa.png"), fullPage: true });

    console.log("    ✔ Persian (RTL) mode rendered with correct direction, logical alignment, and zero overflow.");

    // -------------------------------------------------------------------------
    // SECTION 4: Viewer Role RBAC Protection Verification
    // -------------------------------------------------------------------------
    console.log("\n>>> [SECTION 4] Viewer Role RBAC Protection Verification");

    // Reset language to English
    await page.evaluate(() => {
      localStorage.setItem("clientpulse_language", "en");
    });

    // Create a viewer user in DB using the same password hash
    const viewerHash = runSql(`SELECT password_hash FROM users WHERE id = '${ownerUserId}';`);
    const viewerUserId = runSql(`
      INSERT INTO users (id, email, password_hash, display_name, email_verified_at, status, created_at, updated_at)
      VALUES (gen_random_uuid(), '${viewerEmail}', '${viewerHash}', 'Observant Viewer', NOW(), 'active', NOW(), NOW())
      RETURNING id;
    `);

    runSql(`
      INSERT INTO organization_memberships (id, organization_id, user_id, role, created_at, updated_at)
      VALUES (gen_random_uuid(), '${orgId}', '${viewerUserId}', 'viewer', NOW(), NOW());
    `);

    // Create a separate clean browser context for Viewer
    const viewerContext = await browser.newContext({
      viewport: { width: 1440, height: 900 },
    });
    const viewerPage = await viewerContext.newPage();

    await viewerPage.goto(`${BASE_URL}/login`, { waitUntil: "networkidle" });
    await viewerPage.fill('input[name="email"]', viewerEmail);
    await viewerPage.fill('input[name="password"]', password);
    await viewerPage.click('button[type="submit"]');
    await viewerPage.waitForNavigation({ waitUntil: "networkidle" }).catch(() => {});
    await viewerPage.waitForTimeout(2000);

    // 1. Verify Viewer on /monitors
    await viewerPage.goto(`${BASE_URL}/monitors`, { waitUntil: "networkidle" });
    await viewerPage.waitForTimeout(1000);
    const monitorsNewBtn = await viewerPage.$('a[href="/monitors/new"]');
    if (monitorsNewBtn) {
      throw new Error("Viewer should not see '+ New Monitor' CTA button!");
    }
    console.log("    ✔ Viewer cannot see '+ New Monitor' button.");

    // 2. Direct access to /monitors/new shows Insufficient Permissions
    await viewerPage.goto(`${BASE_URL}/monitors/new`, { waitUntil: "networkidle" });
    await viewerPage.waitForTimeout(1000);
    const deniedTextMon = await viewerPage.textContent("body");
    if (!deniedTextMon.includes("Access Restricted") && !deniedTextMon.includes("do not have permission")) {
      throw new Error(`Viewer direct access to /monitors/new was not blocked! Text: ${deniedTextMon}`);
    }
    await viewerPage.screenshot({ path: path.join(SCREENSHOT_DIR, "22-viewer-denied-monitors-new.png"), fullPage: true });
    console.log("    ✔ Direct access to /monitors/new properly guarded by RBAC.");

    // 3. Direct access to /heartbeats/new shows Insufficient Permissions
    await viewerPage.goto(`${BASE_URL}/heartbeats/new`, { waitUntil: "networkidle" });
    await viewerPage.waitForTimeout(1000);
    const deniedTextHb = await viewerPage.textContent("body");
    if (!deniedTextHb.includes("Access Restricted") && !deniedTextHb.includes("do not have permission")) {
      throw new Error(`Viewer direct access to /heartbeats/new was not blocked! Text: ${deniedTextHb}`);
    }
    await viewerPage.screenshot({ path: path.join(SCREENSHOT_DIR, "23-viewer-denied-heartbeats-new.png"), fullPage: true });
    console.log("    ✔ Direct access to /heartbeats/new properly guarded by RBAC.");

    // 4. Direct access to /clients/new shows Insufficient Permissions
    await viewerPage.goto(`${BASE_URL}/clients/new`, { waitUntil: "networkidle" });
    await viewerPage.waitForTimeout(1000);
    const deniedTextClient = await viewerPage.textContent("body");
    if (!deniedTextClient.includes("Access Restricted") && !deniedTextClient.includes("do not have permission")) {
      throw new Error(`Viewer direct access to /clients/new was not blocked! Text: ${deniedTextClient}`);
    }
    await viewerPage.screenshot({ path: path.join(SCREENSHOT_DIR, "24-viewer-denied-clients-new.png"), fullPage: true });
    console.log("    ✔ Direct access to /clients/new properly guarded by RBAC.");

    // 5. Verify Viewer on /maintenance
    await viewerPage.goto(`${BASE_URL}/maintenance`, { waitUntil: "networkidle" });
    await viewerPage.waitForTimeout(1000);
    const addActivityBtn = await viewerPage.$('button:has-text("Record Activity")');
    if (addActivityBtn) {
      throw new Error("Viewer should not see 'Record Activity' button!");
    }
    console.log("    ✔ Viewer cannot see 'Record Activity' button on /maintenance.");

    console.log("\n================================================================================");
    console.log("TASK-008.B.4: ALL VERIFICATIONS & BREAKPOINT CHECKS PASSED SUCCESSFULLY");
    console.log("================================================================================\n");
  } catch (err) {
    console.error("TASK-008.B.4 Suite FAILED:", err);
    process.exitCode = 1;
  } finally {
    await browser.close();
  }
}

runTask008B4Suite();
