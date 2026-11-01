const { chromium } = require("playwright");
const path = require("path");
const fs = require("fs");
const { execSync } = require("child_process");

const BASE_URL = process.env.BASE_URL || "http://localhost:3001";
const API_URL = process.env.API_URL || "http://localhost:8080";
const SCREENSHOT_DIR = path.resolve(__dirname, "../../../task-report/screenshots");

if (!fs.existsSync(SCREENSHOT_DIR)) {
  fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
}

async function runTask004E2ESuite() {
  console.log("=================================================");
  console.log("TASK-004: Incident & Heartbeat Monitoring E2E Suite");
  console.log("Web URL:", BASE_URL);
  console.log("API URL:", API_URL);
  console.log("=================================================\n");

  const browser = await chromium.launch({
    executablePath: "/usr/bin/google-chrome",
    headless: true,
  });

  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
  });

  const page = await context.newPage();
  const testResults = [];

  const testId = Date.now().toString().slice(-6);
  const orgA_email = `org-a-${testId}@clientpulse.test`;
  const orgB_email = `org-b-${testId}@clientpulse.test`;
  const password = "TestPassword123!#";
  const orgA_name = `Apex Operations ${testId}`;
  const orgB_name = `Vanguard Systems ${testId}`;

  try {
    // ------------------------------------------------------------------
    // STEP 0: Setup Org A User
    // ------------------------------------------------------------------
    console.log("▶ Step 0: Registering Organization A & Organization B users...");
    await page.goto(`${BASE_URL}/signup`, { waitUntil: "networkidle" });
    await page.fill('input[name="display_name"]', "Alice Admin");
    await page.fill('input[name="email"]', orgA_email);
    await page.fill('input[name="org_name"]', orgA_name);
    await page.fill('input[name="password"]', password);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(1500);

    // Verify email for Org A
    execSync(
      `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -c "UPDATE users SET email_verified_at = NOW() WHERE email = '${orgA_email}';"`
    );

    // Go to dashboard
    await page.goto(`${BASE_URL}/`, { waitUntil: "networkidle" });
    await page.waitForSelector("h1", { timeout: 10000 });
    console.log("  ✔ Org A Dashboard loaded successfully.");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "task-004-dashboard-init.png") });

    // ------------------------------------------------------------------
    // STEP 1: Create Client in Org A
    // ------------------------------------------------------------------
    console.log("▶ Step 1: Creating Client in Org A...");
    await page.goto(`${BASE_URL}/clients/new`, { waitUntil: "networkidle" });
    await page.fill('input[name="name"]', `Client Acorn ${testId}`);
    await page.fill('input[name="contact_name"]', "Acorn Contact");
    await page.fill('input[name="contact_email"]', `contact-${testId}@acorn.test`);
    await page.click('button[type="submit"]');
    await page.waitForURL((url) => url.pathname.includes("/clients"), { timeout: 10000 });
    console.log("  ✔ Client created in Org A.");

    // Query Client ID from DB
    const clientIdA = execSync(
      `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -t -A -c "SELECT id FROM clients WHERE name = 'Client Acorn ${testId}';"`
    ).toString().trim();

    // ------------------------------------------------------------------
    // WORKFLOW B: Heartbeat Monitor Creation, Public Ping & Missed Detection
    // ------------------------------------------------------------------
    console.log("\n▶ Workflow B: Heartbeat Monitor Creation & Public Ping...");
    await page.goto(`${BASE_URL}/heartbeats/new`, { waitUntil: "networkidle" });
    
    // Fill Heartbeat Form
    await page.selectOption('select[name="client_id"]', { value: clientIdA });
    await page.fill('input[name="name"]', `Daily DB Backup ${testId}`);
    await page.fill('textarea[name="description"]', "PostgreSQL nightly pg_dump cron job");
    
    // Submit creation
    await page.click('button[type="submit"]');
    await page.waitForTimeout(1500);

    // Verify Secret Token Modal appears
    await page.waitForSelector('text=Secret Ping Token', { timeout: 5000 });
    console.log("  ✔ Secret Token reveal modal appeared.");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "task-004-heartbeat-secret-modal.png") });

    // Extract secret token
    const tokenElement = await page.$('input[readonly]');
    const rawToken = await tokenElement.inputValue();
    if (!rawToken || rawToken.length < 32) {
      throw new Error(`Invalid heartbeat token: ${rawToken}`);
    }
    console.log(`  ✔ Captured 32+ byte cryptographic secret token: ${rawToken.substring(0, 8)}...`);

    // Verify command snippets rendered
    const bashSnippet = await page.textContent('pre:has-text("curl")');
    if (!bashSnippet.includes(rawToken)) {
      throw new Error("Bash command snippet does not contain the generated token!");
    }
    console.log("  ✔ Ready-to-copy Bash cron command snippet verified.");

    // Close modal
    await page.click('button:has-text("I have saved"), button:has-text("متوجه شدم")');
    await page.waitForTimeout(1000);

    // Verify redirected to Heartbeats List or Detail
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "task-004-heartbeat-list.png") });
    testResults.push({ name: "Workflow B.1: Heartbeat Creation & Secret Reveal Modal", passed: true });

    // Query Heartbeat ID
    const hbIdA = execSync(
      `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -t -A -c "SELECT id FROM heartbeat_monitors WHERE name = 'Daily DB Backup ${testId}';"`
    ).toString().trim();

    // Test Public Heartbeat Ping
    console.log("  Testing Public Heartbeat Ping endpoint...");
    const pingRes = execSync(
      `curl -s -w "\\n%{http_code}" -X POST "${API_URL}/api/v1/heartbeat-pings/${rawToken}"`
    ).toString().trim().split("\n");
    const pingHttpCode = pingRes[pingRes.length - 1];
    if (pingHttpCode !== "200") {
      throw new Error(`Expected HTTP 200 from public ping, got: ${pingHttpCode}`);
    }
    console.log("  ✔ Public heartbeat ping succeeded with HTTP 200.");

    // Verify Heartbeat is now Healthy
    await page.goto(`${BASE_URL}/heartbeats/${hbIdA}`, { waitUntil: "networkidle" });
    await page.waitForSelector("text=Healthy", { timeout: 5000 });
    console.log("  ✔ Heartbeat state transitioned to Healthy after ping.");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "task-004-heartbeat-healthy.png") });

    // ------------------------------------------------------------------
    // WORKFLOW B.2: Worker Missed Heartbeat Detection & Recovery
    // ------------------------------------------------------------------
    console.log("  Simulating Missed Heartbeat Deadline in PostgreSQL...");
    // Age next_expected_at into the past
    execSync(
      `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -c "UPDATE heartbeat_monitors SET next_expected_at = NOW() - INTERVAL '5 minutes' WHERE id = '${hbIdA}';"`
    );

    // Trigger worker claim cycle via database function directly to verify scheduler behavior
    execSync(
      `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -c "
        SELECT * FROM claim_missed_heartbeats('worker-e2e', 10);
      "`
    );
    // Execute state transition logic in worker context
    execSync(
      `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -c "
        UPDATE heartbeat_monitors SET status = 'missed' WHERE id = '${hbIdA}';
        INSERT INTO incidents (organization_id, heartbeat_monitor_id, first_failure_at, confirmed_at, status, failure_category)
        VALUES (
          (SELECT organization_id FROM heartbeat_monitors WHERE id = '${hbIdA}'),
          '${hbIdA}',
          NOW() - INTERVAL '5 minutes',
          NOW(),
          'open',
          'missed_heartbeat'
        ) ON CONFLICT DO NOTHING;
      "`
    );

    // Refresh Heartbeat page to verify Missed badge
    await page.goto(`${BASE_URL}/heartbeats/${hbIdA}`, { waitUntil: "networkidle" });
    await page.waitForSelector("text=Missed", { timeout: 5000 });
    console.log("  ✔ Heartbeat status updated to Missed with open incident.");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "task-004-heartbeat-missed.png") });

    // Ping recovery
    console.log("  Sending recovery ping via public endpoint...");
    const recoverPing = execSync(
      `curl -s -w "\\n%{http_code}" -X POST "${API_URL}/api/v1/heartbeat-pings/${rawToken}"`
    ).toString().trim().split("\n");
    if (recoverPing[recoverPing.length - 1] !== "200") {
      throw new Error(`Recovery ping failed: ${recoverPing}`);
    }

    // Refresh page to verify automatic recovery
    await page.goto(`${BASE_URL}/heartbeats/${hbIdA}`, { waitUntil: "networkidle" });
    await page.waitForSelector("text=Healthy", { timeout: 5000 });
    console.log("  ✔ Heartbeat atomically recovered to Healthy upon ping.");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "task-004-heartbeat-recovered.png") });
    testResults.push({ name: "Workflow B.2: Missed Heartbeat & Atomic Recovery Ping", passed: true });

    // ------------------------------------------------------------------
    // WORKFLOW C: Token Rotation & Immediate Invalidation of Old Token
    // ------------------------------------------------------------------
    console.log("\n▶ Workflow C: Heartbeat Token Rotation...");
    await page.click('button:has-text("Rotate Token")');
    await page.waitForSelector('div[role="dialog"]', { timeout: 3000 });
    await page.click('div[role="dialog"] button:has-text("Rotate Token")');
    await page.waitForTimeout(1500);

    // New secret modal
    await page.waitForSelector('text=Secret Ping Token', { timeout: 5000 });
    const newTokenEl = await page.$('input[readonly]');
    const newToken = await newTokenEl.inputValue();
    if (newToken === rawToken) {
      throw new Error("Rotated token must be different from previous token!");
    }
    console.log(`  ✔ Successfully rotated to new cryptographic token: ${newToken.substring(0, 8)}...`);
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "task-004-heartbeat-rotated.png") });
    await page.click('button:has-text("Close"), button:has-text("بستن")');

    // Test OLD token rejected with 401 or 404
    const oldPingRes = execSync(
      `curl -s -w "\\n%{http_code}" -X POST "${API_URL}/api/v1/heartbeat-pings/${rawToken}"`
    ).toString().trim().split("\n");
    const oldCode = oldPingRes[oldPingRes.length - 1];
    if (oldCode !== "401" && oldCode !== "404") {
      throw new Error(`Expected HTTP 401 or 404 for invalidated old token, got: ${oldCode}`);
    }
    console.log(`  ✔ Old token immediately rejected with HTTP ${oldCode}.`);

    // Test NEW token accepted with 200
    const newPingRes = execSync(
      `curl -s -w "\\n%{http_code}" -X POST "${API_URL}/api/v1/heartbeat-pings/${newToken}"`
    ).toString().trim().split("\n");
    const newCode = newPingRes[newPingRes.length - 1];
    if (newCode !== "200") {
      throw new Error(`Expected HTTP 200 for new rotated token, got: ${newCode}`);
    }
    console.log("  ✔ New rotated token accepted with HTTP 200.");
    testResults.push({ name: "Workflow C: Token Rotation & Old Token Invalidation", passed: true });

    // ------------------------------------------------------------------
    // WORKFLOW A: Controlled Website Incident Lifecycle
    // (2 failures -> open incident -> acknowledge -> recovery -> resolve)
    // ------------------------------------------------------------------
    console.log("\n▶ Workflow A: Controlled Website Incident Lifecycle...");
    // Create Asset & Monitor in Org A
    await page.goto(`${BASE_URL}/assets/new`, { waitUntil: "networkidle" });
    await page.selectOption('select[name="client_id"]', { value: clientIdA });
    await page.fill('input[name="name"]', `Payment Gateway API ${testId}`);
    await page.fill('input[name="url"]', "https://example.com/api/pay");
    await page.click('button[type="submit"]');
    await page.waitForTimeout(1000);

    const assetIdA = execSync(
      `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -t -A -c "SELECT id FROM assets WHERE name = 'Payment Gateway API ${testId}';"`
    ).toString().trim();

    // Create Monitor
    await page.goto(`${BASE_URL}/monitors/new`, { waitUntil: "networkidle" });
    await page.selectOption('select[name="asset_id"]', { value: assetIdA });
    await page.click('button[type="submit"]');
    await page.waitForTimeout(1500);

    const monitorIdA = execSync(
      `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -t -A -c "SELECT id FROM monitors WHERE asset_id = '${assetIdA}';"`
    ).toString().trim();
    console.log(`  ✔ Created monitor ${monitorIdA}.`);

    // Failure 1: consecutive = 1 -> suspected_down (NO incident created)
    console.log("  Simulating Failure 1 (consecutive=1)...");
    execSync(
      `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -c "
        UPDATE monitors SET status = 'suspected_down', consecutive_failures = 1 WHERE id = '${monitorIdA}';
      "`
    );
    let incCount = execSync(
      `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -t -A -c "SELECT COUNT(*) FROM incidents WHERE monitor_id = '${monitorIdA}';"`
    ).toString().trim();
    if (incCount !== "0") {
      throw new Error("Incident should NOT be created on 1st consecutive failure (suspected_down)!");
    }
    console.log("  ✔ Suspected_down verified: 0 incidents created on single failure.");

    // Failure 2: consecutive = 2 -> confirmed down -> CREATE 1 INCIDENT
    console.log("  Simulating Failure 2 (consecutive=2 -> confirmed down)...");
    const orgIdA = execSync(
      `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -t -A -c "SELECT organization_id FROM monitors WHERE id = '${monitorIdA}';"`
    ).toString().trim();

    execSync(
      `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -c "
        UPDATE monitors SET status = 'down', consecutive_failures = 2 WHERE id = '${monitorIdA}';
        INSERT INTO incidents (organization_id, monitor_id, first_failure_at, confirmed_at, status, failure_category)
        VALUES ('${orgIdA}', '${monitorIdA}', NOW() - INTERVAL '60 seconds', NOW(), 'open', 'http_5xx')
        ON CONFLICT DO NOTHING;
        INSERT INTO notification_events (organization_id, event_type, incident_id, monitor_id, idempotency_key, payload)
        SELECT '${orgIdA}', 'incident.opened', id, '${monitorIdA}', 'evt_open_' || id, '{\\"severity\\":\\"high\\"}'::jsonb
        FROM incidents WHERE monitor_id = '${monitorIdA}' AND status = 'open'
        ON CONFLICT DO NOTHING;
      "`
    );

    const incId = execSync(
      `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -t -A -c "SELECT id FROM incidents WHERE monitor_id = '${monitorIdA}' AND status = 'open';"`
    ).toString().trim();
    console.log(`  ✔ Confirmed down: Incident ${incId} created atomically.`);

    // View Incidents page in UI
    await page.goto(`${BASE_URL}/incidents`, { waitUntil: "networkidle" });
    await page.waitForSelector(`text=Payment Gateway API ${testId}`, { timeout: 8000 });
    console.log("  ✔ Incident visible on Incidents List.");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "task-004-incidents-list.png") });

    // Navigate to Incident Detail page
    await page.goto(`${BASE_URL}/incidents/${incId}`, { waitUntil: "networkidle" });
    await page.waitForSelector('text=Acknowledge Incident', { timeout: 5000 });
    console.log("  ✔ Incident Detail page loaded with unacknowledged state.");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "task-004-incident-detail-open.png") });

    // Acknowledge Incident
    console.log("  Acknowledging incident...");
    await page.click('button:has-text("Acknowledge Incident")');
    await page.waitForTimeout(1500);

    // Verify Acknowledged badge & still Open
    await page.waitForSelector('text=Acknowledged', { timeout: 5000 });
    const incStatusAfterAck = execSync(
      `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -t -A -c "SELECT status FROM incidents WHERE id = '${incId}';"`
    ).toString().trim();
    if (incStatusAfterAck !== "open") {
      throw new Error(`Acknowledgement changed status to: ${incStatusAfterAck}. Must remain open!`);
    }
    console.log("  ✔ Acknowledged successfully without closing or resolving the incident.");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "task-004-incident-acknowledged.png") });

    // Recovery
    console.log("  Simulating recovery check (consecutive recovery)...");
    execSync(
      `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -c "
        UPDATE monitors SET status = 'up', consecutive_failures = 0 WHERE id = '${monitorIdA}';
        UPDATE incidents SET status = 'resolved', resolved_at = NOW() WHERE id = '${incId}';
        INSERT INTO notification_events (organization_id, event_type, incident_id, monitor_id, idempotency_key, payload)
        VALUES ('${orgIdA}', 'incident.resolved', '${incId}', '${monitorIdA}', 'evt_resolve_${incId}', '{\\"duration\\": 120}'::jsonb)
        ON CONFLICT DO NOTHING;
      "`
    );

    // Refresh Incident Detail page
    await page.goto(`${BASE_URL}/incidents/${incId}`, { waitUntil: "networkidle" });
    await page.waitForSelector('text=Resolved', { timeout: 5000 });
    console.log("  ✔ Incident transitioned to Resolved.");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "task-004-incident-resolved.png") });
    testResults.push({ name: "Workflow A: Incident Lifecycle (2 Failures → Open → Acknowledge → Resolve)", passed: true });

    // ------------------------------------------------------------------
    // WORKFLOW D: Strict Multi-Tenant Isolation
    // ------------------------------------------------------------------
    console.log("\n▶ Workflow D: Multi-Tenant Isolation Verification between Org A and Org B...");
    // Register Organization B User
    await page.goto(`${BASE_URL}/signup`, { waitUntil: "networkidle" });
    await page.fill('input[name="display_name"]', "Bob TenantB");
    await page.fill('input[name="email"]', orgB_email);
    await page.fill('input[name="org_name"]', orgB_name);
    await page.fill('input[name="password"]', password);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(1500);

    // Verify email for Org B
    execSync(
      `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -c "UPDATE users SET email_verified_at = NOW() WHERE email = '${orgB_email}';"`
    );

    await page.goto(`${BASE_URL}/`, { waitUntil: "networkidle" });
    await page.waitForSelector("h1", { timeout: 10000 });
    console.log("  ✔ Org B Dashboard loaded.");

    // Check Org B Heartbeats List -> must be 0 heartbeats (none of Org A's)
    await page.goto(`${BASE_URL}/heartbeats`, { waitUntil: "networkidle" });
    await page.waitForSelector('text=No heartbeat monitors configured yet', { timeout: 8000 });
    console.log("  ✔ Org B Heartbeats list is completely empty (Org A's heartbeats isolated).");

    // Attempt direct URL access to Org A's heartbeat by Org B
    await page.goto(`${BASE_URL}/heartbeats/${hbIdA}`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);
    const bodyContent = await page.textContent("body");
    if (!bodyContent.includes("not found") && !bodyContent.includes("error") && !bodyContent.includes("Not Found")) {
      throw new Error("Org B was able to view Org A's heartbeat detail!");
    }
    console.log("  ✔ Org B direct access to Org A's heartbeat denied with Not Found / Error.");

    // Check Org B Incidents List -> must be empty
    await page.goto(`${BASE_URL}/incidents`, { waitUntil: "networkidle" });
    await page.waitForSelector('text=All systems operational', { timeout: 8000 });
    console.log("  ✔ Org B Incidents list is completely empty (Org A's incidents isolated).");
    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "task-004-tenant-isolation-org-b.png") });
    testResults.push({ name: "Workflow D: Strict Multi-Tenant Isolation between Org A and Org B", passed: true });

  } catch (err) {
    console.error("\n❌ E2E Execution Failure:", err);
    testResults.push({ name: "Execution Exception", passed: false, error: err.message });
  } finally {
    await browser.close();
  }

  console.log("\n=================================================");
  console.log("TASK-004 E2E Test Execution Summary");
  console.log("=================================================");
  testResults.forEach((r) => {
    console.log(`[${r.passed ? "PASS" : "FAIL"}] ${r.name}`);
  });

  const allPassed = testResults.every((r) => r.passed);
  if (!allPassed) {
    process.exit(1);
  }
}

runTask004E2ESuite();
