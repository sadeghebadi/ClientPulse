const { chromium } = require("playwright");
const path = require("path");
const fs = require("fs");
const { execSync } = require("child_process");

const BASE_URL = process.env.BASE_URL || "http://localhost:3000";
const SCREENSHOT_DIR = path.resolve(__dirname, "../../../task-report/screenshots/task-incident-004");

if (!fs.existsSync(SCREENSHOT_DIR)) {
  fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
}

function runSql(sql) {
  const cmd = `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -t -A -q -c "${sql.replace(/"/g, '\\"')}"`;
  const res = execSync(cmd).toString().trim();
  const lines = res.split("\n").map(l => l.trim()).filter(l => l && !l.startsWith("INSERT") && !l.startsWith("UPDATE") && !l.startsWith("DELETE"));
  return lines.length > 0 ? lines[0] : "";
}

async function runScreenshots() {
  console.log("Launching browser for TASK-INCIDENT-004 screenshots...");
  const browser = await chromium.launch({
    executablePath: "/usr/bin/google-chrome",
    headless: true,
  });

  try {
    const unique = Math.random().toString(36).substring(7);
    const ownerEmail = `incident-lead-${unique}@example.com`;
    const password = "TestPassword123!#";

    const context = await browser.newContext({
      viewport: { width: 1440, height: 900 },
    });
    const page = await context.newPage();

    // 1. Sign up user
    console.log("1. Signing up user...");
    await page.goto(`${BASE_URL}/signup`, { waitUntil: "networkidle" });
    await page.fill('input[name="display_name"]', "Incident Response Lead");
    await page.fill('input[name="email"]', ownerEmail);
    await page.fill('input[name="password"]', password);
    await page.fill('input[name="org_name"]', "Apex SRE Ops");
    await page.click('button[type="submit"]');
    await page.waitForNavigation({ waitUntil: "networkidle" }).catch(() => {});
    await page.waitForTimeout(1500);

    // Retrieve IDs
    const userId = runSql(`SELECT id FROM users WHERE email = '${ownerEmail}' LIMIT 1;`);
    const orgId = runSql(`SELECT organization_id FROM organization_memberships WHERE user_id = '${userId}' LIMIT 1;`);

    // Verify email
    runSql(`UPDATE users SET email_verified_at = NOW() WHERE id = '${userId}';`);

    console.log(`User: ${userId}, Org: ${orgId}`);

    // Create Client
    const clientId = runSql(`INSERT INTO clients (id, organization_id, name, status, created_at, updated_at) VALUES (gen_random_uuid(), '${orgId}', 'Core Banking Client', 'active', NOW(), NOW()) RETURNING id;`);

    // Create Asset & Monitor (Website)
    const assetId = runSql(`INSERT INTO assets (id, organization_id, client_id, name, url, is_active, created_at, updated_at) VALUES (gen_random_uuid(), '${orgId}', '${clientId}', 'Payment Gateway API', 'https://api.gateway.clientpulse.local/health', true, NOW(), NOW()) RETURNING id;`);
    const monitorId = runSql(`INSERT INTO monitors (id, organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, consecutive_failures, created_at, updated_at) VALUES (gen_random_uuid(), '${orgId}', '${assetId}', 60, 10, 'down', true, 3, NOW(), NOW()) RETURNING id;`);

    // Create Heartbeat Monitor 1 (Active)
    const hb1Id = runSql(`INSERT INTO heartbeat_monitors (id, organization_id, client_id, name, expected_interval_seconds, grace_period_seconds, status, token_hash, created_at, updated_at) VALUES (gen_random_uuid(), '${orgId}', '${clientId}', 'Database Daily Backup Cron', 86400, 3600, 'missed', 'dummytokenhash1', NOW(), NOW()) RETURNING id;`);

    // Create Heartbeat Monitor 2 (Deleted/Archived)
    const hb2Id = runSql(`INSERT INTO heartbeat_monitors (id, organization_id, client_id, name, expected_interval_seconds, grace_period_seconds, status, token_hash, archived_at, created_at, updated_at) VALUES (gen_random_uuid(), '${orgId}', '${clientId}', 'Legacy S3 Archive Exporter', 3600, 300, 'missed', 'dummytokenhash2', NOW() - INTERVAL '1 hour', NOW() - INTERVAL '1 day', NOW()) RETURNING id;`);

    // Incident 1: Open Website Outage (Acknowledged)
    const inc1Id = runSql(`INSERT INTO incidents (id, organization_id, monitor_id, first_failure_at, confirmed_at, status, failure_category, acknowledged_at, acknowledged_by, duration_seconds, summary, created_at, updated_at) VALUES (gen_random_uuid(), '${orgId}', '${monitorId}', NOW() - INTERVAL '45 minutes', NOW() - INTERVAL '42 minutes', 'open', 'http_502_bad_gateway', NOW() - INTERVAL '30 minutes', '${userId}', 2700, '502 Bad Gateway observed on Payment Gateway API', NOW(), NOW()) RETURNING id;`);

    // Incident 2: Open Heartbeat Incident (Unacknowledged)
    const inc2Id = runSql(`INSERT INTO incidents (id, organization_id, heartbeat_monitor_id, first_failure_at, confirmed_at, status, failure_category, duration_seconds, summary, created_at, updated_at) VALUES (gen_random_uuid(), '${orgId}', '${hb1Id}', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '1 hour', 'open', 'heartbeat_missed', 7200, 'Database backup cron did not send heartbeat ping', NOW(), NOW()) RETURNING id;`);

    // Incident 3: Manually Resolved Incident (with note and actor)
    const inc3Id = runSql(`INSERT INTO incidents (id, organization_id, monitor_id, first_failure_at, confirmed_at, resolved_at, resolved_by, resolution_type, resolution_note, status, failure_category, acknowledged_at, acknowledged_by, duration_seconds, summary, created_at, updated_at) VALUES (gen_random_uuid(), '${orgId}', '${monitorId}', NOW() - INTERVAL '10 hours', NOW() - INTERVAL '9 hours', NOW() - INTERVAL '8 hours', '${userId}', 'manual', 'Operator verified upstream BGP rerouting restored gateway access. Manual resolution approved by SRE Lead.', 'resolved', 'connection_timeout', NOW() - INTERVAL '9 hours', '${userId}', 3600, 'Connection timeout on gateway', NOW(), NOW()) RETURNING id;`);

    // Incident 4: Closed Due to Source Monitor Deleted
    const inc4Id = runSql(`INSERT INTO incidents (id, organization_id, heartbeat_monitor_id, first_failure_at, confirmed_at, resolved_at, resolved_by, resolution_type, resolution_note, status, failure_category, duration_seconds, summary, created_at, updated_at) VALUES (gen_random_uuid(), '${orgId}', '${hb2Id}', NOW() - INTERVAL '5 hours', NOW() - INTERVAL '4 hours', NOW() - INTERVAL '1 hour', '${userId}', 'source_monitor_deleted', 'Automatically resolved upon source monitor deletion', 'resolved', 'heartbeat_missed', 14400, 'Legacy S3 Exporter missed scheduled ping', NOW(), NOW()) RETURNING id;`);

    console.log("Seeded incidents:", { inc1Id, inc2Id, inc3Id, inc4Id });

    // SCREENSHOT 1: Incidents List (English)
    console.log("Capturing 01-incident-list-lifecycle-en.png...");
    await page.goto(`${BASE_URL}/incidents`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "01-incident-list-lifecycle-en.png"),
      fullPage: true,
    });

    // SCREENSHOT 2: Incident Detail - Manual Resolution
    console.log("Capturing 02-incident-detail-manual-resolution-en.png...");
    await page.goto(`${BASE_URL}/incidents/${inc3Id}`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "02-incident-detail-manual-resolution-en.png"),
      fullPage: true,
    });

    // SCREENSHOT 3: Incident Detail - Source Monitor Deleted
    console.log("Capturing 03-incident-detail-source-deleted-en.png...");
    await page.goto(`${BASE_URL}/incidents/${inc4Id}`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "03-incident-detail-source-deleted-en.png"),
      fullPage: true,
    });

    // SCREENSHOT 4: Heartbeat Delete Confirmation Modal
    console.log("Capturing 04-heartbeat-delete-modal-en.png...");
    await page.goto(`${BASE_URL}/heartbeats/${hb1Id}`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    const deleteBtn = await page.$('button:has-text("Delete")');
    if (deleteBtn) {
      await deleteBtn.click();
      await page.waitForTimeout(500);
      await page.screenshot({
        path: path.join(SCREENSHOT_DIR, "04-heartbeat-delete-modal-en.png"),
        fullPage: true,
      });
      // Close modal
      const cancelBtn = await page.$('button:has-text("Cancel")');
      if (cancelBtn) await cancelBtn.click();
    }

    // SCREENSHOT 5: Incidents List (Persian RTL)
    console.log("Capturing 05-incident-list-lifecycle-fa-rtl.png...");
    // Toggle language
    const langBtn = await page.$('button:has-text("فارسی")');
    if (langBtn) {
      await langBtn.click();
      await page.waitForTimeout(500);
    }
    await page.goto(`${BASE_URL}/incidents`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "05-incident-list-lifecycle-fa-rtl.png"),
      fullPage: true,
    });

    console.log("All screenshots captured successfully in:", SCREENSHOT_DIR);
  } catch (err) {
    console.error("Screenshot error:", err);
  } finally {
    await browser.close();
  }
}

runScreenshots();
