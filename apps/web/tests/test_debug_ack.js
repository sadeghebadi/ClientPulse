const { chromium } = require("playwright");
const { execSync } = require("child_process");

async function testAck() {
  const browser = await chromium.launch({ executablePath: "/usr/bin/google-chrome", headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

  page.on("console", msg => console.log("PAGE LOG:", msg.text()));

  const unique = Math.random().toString(36).substring(7);
  const email = `ack-test-${unique}@example.com`;
  await page.goto("http://localhost:3000/signup");
  await page.fill('input[name="display_name"]', "Ack Tester");
  await page.fill('input[name="email"]', email);
  await page.fill('input[name="password"]', "TestPassword123!#");
  await page.fill('input[name="org_name"]', "Ack Agency");
  await page.click('button[type="submit"]');
  await page.waitForNavigation({ waitUntil: "networkidle" }).catch(() => {});
  await page.waitForTimeout(2000);

  const orgId = execSync(`docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -t -A -c "SELECT organization_id FROM organization_memberships WHERE user_id = (SELECT id FROM users WHERE email = '${email}') LIMIT 1;"`).toString().trim();
  const cId = execSync(`docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -t -A -c "INSERT INTO clients (id, organization_id, name, status, created_at, updated_at) VALUES (gen_random_uuid(), '${orgId}', 'Client A', 'active', NOW(), NOW()) RETURNING id;"`).toString().trim().split("\n").pop();
  const aId = execSync(`docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -t -A -c "INSERT INTO assets (id, organization_id, client_id, name, url, is_active, created_at, updated_at) VALUES (gen_random_uuid(), '${orgId}', '${cId}', 'Web A', 'https://example.com', true, NOW(), NOW()) RETURNING id;"`).toString().trim().split("\n").pop();
  const mId = execSync(`docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -t -A -c "INSERT INTO monitors (id, organization_id, asset_id, interval_seconds, timeout_seconds, status, is_active, consecutive_failures, created_at, updated_at) VALUES (gen_random_uuid(), '${orgId}', '${aId}', 60, 10, 'down', true, 2, NOW(), NOW()) RETURNING id;"`).toString().trim().split("\n").pop();
  const incId = execSync(`docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -t -A -c "INSERT INTO incidents (id, organization_id, monitor_id, first_failure_at, confirmed_at, status, failure_category, created_at, updated_at) VALUES (gen_random_uuid(), '${orgId}', '${mId}', NOW() - INTERVAL '5 minutes', NOW() - INTERVAL '3 minutes', 'open', 'http_5xx', NOW(), NOW()) RETURNING id;"`).toString().trim().split("\n").pop();

  console.log("Created test incident ID:", incId);
  await page.reload({ waitUntil: "networkidle" });
  await page.waitForTimeout(2000);

  const buttons = await page.$$eval("button", btns => btns.map(b => b.textContent.trim()));
  console.log("Buttons on page:", buttons);

  const incRows = await page.$$eval('div[class*="divide-y"] > div', divs => divs.map(d => d.textContent.trim()));
  console.log("Incident rows:", incRows);

  const ackBtn = await page.$('button:has-text("Acknowledge")');
  console.log("Found ack button:", !!ackBtn);
  if (ackBtn) {
    await ackBtn.click();
    await page.waitForTimeout(2000);
    const dbAck = execSync(`docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -t -A -c "SELECT acknowledged_at FROM incidents WHERE id = '${incId}';"`).toString().trim();
    console.log("DB Ack result:", dbAck);
  }

  await browser.close();
}
testAck().catch(console.error);
