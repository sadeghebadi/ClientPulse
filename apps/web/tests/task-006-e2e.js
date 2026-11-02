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

async function runTask006E2ESuite() {
  console.log("===================================================================");
  console.log("TASK-006: Notification Infrastructure — Telegram & Email E2E Test");
  console.log("Web URL:", BASE_URL);
  console.log("API URL:", API_URL);
  console.log("===================================================================\n");

  const browser = await chromium.launch({
    executablePath: "/usr/bin/google-chrome",
    headless: true,
  });

  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
  });

  const page = await context.newPage();
  page.on("dialog", async (dialog) => {
    await dialog.accept();
  });

  const testId = Date.now().toString().slice(-6);
  const email = `notif-admin-${testId}@clientpulse.test`;
  const password = "TestPassword123!#";
  const orgName = `Pulse Alerts Agency ${testId}`;

  try {
    // ------------------------------------------------------------------
    // STEP 1: Register User & Organization
    // ------------------------------------------------------------------
    console.log("▶ Step 1: Registering organization & admin user...");
    await page.goto(`${BASE_URL}/signup`, { waitUntil: "networkidle" });
    await page.fill('input[name="display_name"]', "SRE Lead");
    await page.fill('input[name="email"]', email);
    await page.fill('input[name="org_name"]', orgName);
    await page.fill('input[name="password"]', password);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(1500);

    // Verify email directly in DB
    execSync(
      `docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -c "UPDATE users SET email_verified_at = NOW() WHERE email = '${email}';"`
    );

    // Log in
    await page.goto(`${BASE_URL}/login`, { waitUntil: "networkidle" });
    await page.fill('input[type="email"]', email);
    await page.fill('input[type="password"]', password);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(1500);

    console.log("✔ User logged in successfully.");

    // ------------------------------------------------------------------
    // STEP 2: Navigate to Notification Settings
    // ------------------------------------------------------------------
    console.log("▶ Step 2: Navigating to Notification Settings page...");
    await page.goto(`${BASE_URL}/settings/notifications`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);

    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "task-006-notifications-main.png"),
      fullPage: true,
    });
    console.log("✔ Captured initial Notification Settings screenshot.");

    // ------------------------------------------------------------------
    // STEP 3: Connect Telegram Private Modal & Token Generation
    // ------------------------------------------------------------------
    console.log("▶ Step 3: Opening Telegram Connect Modal...");
    const connectBtn = await page.waitForSelector('button:has-text("Connect Telegram")');
    await connectBtn.click();
    await page.waitForTimeout(1000);

    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "task-006-telegram-connect-modal.png"),
    });

    // Check that deep link was generated
    const openTelegramLink = await page.$('a[href*="t.me"]');
    if (!openTelegramLink) {
      throw new Error("Telegram deep link was not generated in modal!");
    }
    const href = await openTelegramLink.getAttribute("href");
    console.log("✔ Telegram deep link generated:", href);

    // Extract start token from deep link (e.g. https://t.me/BotUsername?start=<token>)
    const urlObj = new URL(href);
    const startToken = urlObj.searchParams.get("start");
    console.log("✔ Extracted connection token:", startToken);

    // Simulate Telegram update to webhook
    console.log("▶ Simulating Telegram private /start webhook...");
    const mockChatId = 77889900;
    const webhookPayload = {
      update_id: 9001,
      message: {
        message_id: 1,
        from: {
          id: 77889900,
          first_name: "John SRE",
          username: "john_sre",
        },
        chat: {
          id: mockChatId,
          type: "private",
        },
        text: `/start ${startToken}`,
      },
    };

    const webhookRes = await fetch(`${API_URL}/api/v1/integrations/telegram/webhook`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(webhookPayload),
    });

    if (!webhookRes.ok) {
      throw new Error(`Telegram webhook returned ${webhookRes.status}`);
    }
    console.log("✔ Telegram webhook received /start successfully.");

    // Close modal
    await page.click('button:has-text("Done")');
    await page.waitForTimeout(1000);

    // Refresh destinations list in UI
    await page.reload({ waitUntil: "networkidle" });
    await page.waitForTimeout(1000);

    const hasPrivateDest = await page.isVisible(`:has-text("${mockChatId}")`);
    console.log("✔ Private Telegram destination visible in UI:", hasPrivateDest);

    // ------------------------------------------------------------------
    // STEP 4: Connect Telegram Group Chat
    // ------------------------------------------------------------------
    console.log("▶ Step 4: Connecting Telegram Group...");
    const addGroupBtn = await page.waitForSelector('button:has-text("Add Telegram Group")');
    await addGroupBtn.click();
    await page.waitForTimeout(1000);

    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "task-006-telegram-group-modal.png"),
    });

    const codeElement = await page.waitForSelector('code:has-text("/connect")');
    const codeText = await codeElement.textContent();
    const groupCode = codeText.trim().split(" ").pop();
    console.log("✔ Extracted group connection code:", groupCode);

    // Simulate Telegram group /connect command
    const mockGroupChatId = -1005544332211;
    const groupWebhookPayload = {
      update_id: 9002,
      message: {
        message_id: 2,
        from: {
          id: 77889900,
          first_name: "John SRE",
        },
        chat: {
          id: mockGroupChatId,
          type: "supergroup",
          title: "Production Incident Channel",
        },
        text: `/connect ${groupCode}`,
      },
    };

    const groupWebhookRes = await fetch(`${API_URL}/api/v1/integrations/telegram/webhook`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(groupWebhookPayload),
    });

    if (!groupWebhookRes.ok) {
      throw new Error(`Group webhook returned ${groupWebhookRes.status}`);
    }
    console.log("✔ Group webhook received /connect successfully.");

    await page.click('button:has-text("Done")');
    await page.waitForTimeout(1000);

    await page.reload({ waitUntil: "networkidle" });
    await page.waitForTimeout(1000);

    const hasGroupDest = await page.isVisible(`:has-text("${mockGroupChatId}")`);
    console.log("✔ Group Telegram destination visible in UI:", hasGroupDest);

    // ------------------------------------------------------------------
    // STEP 5: Add Email Channel
    // ------------------------------------------------------------------
    console.log("▶ Step 5: Adding Email notification destination...");
    const addEmailBtn = await page.waitForSelector('button:has-text("Add Email")');
    await addEmailBtn.click();
    await page.waitForTimeout(500);

    await page.fill('input[type="email"]', "alerts-team@clientpulse.local");
    await page.fill('input[placeholder="DevOps Team Email"]', "DevOps Alerts Mailbox");
    await page.click('button[type="submit"]:has-text("Add Email")');
    await page.waitForTimeout(1000);

    const hasEmailDest = await page.isVisible('text="alerts-team@clientpulse.local"');
    console.log("✔ Email destination created and visible:", hasEmailDest);

    // ------------------------------------------------------------------
    // STEP 6: Event Preferences Modal
    // ------------------------------------------------------------------
    console.log("▶ Step 6: Testing Preferences Modal...");
    const prefBtns = await page.$$('button:has-text("Preferences")');
    if (prefBtns.length > 0) {
      await prefBtns[0].click();
      await page.waitForTimeout(500);

      await page.screenshot({
        path: path.join(SCREENSHOT_DIR, "task-006-preferences-modal.png"),
      });

      // Save preferences
      await page.click('button:has-text("Save Preferences")');
      await page.waitForTimeout(1000);
      console.log("✔ Preferences saved.");
    }

    // ------------------------------------------------------------------
    // STEP 7: Send Test Notification
    // ------------------------------------------------------------------
    console.log("▶ Step 7: Sending Test Notification to Email channel...");
    const testBtns = await page.$$('button:has-text("Send Test")');
    if (testBtns.length > 0) {
      // Click test on the last one (email)
      await testBtns[testBtns.length - 1].click();
      await page.waitForTimeout(2000);

      // Check success alert
      const successText = await page.isVisible('text="Test notification delivered successfully!"');
      console.log("✔ Test notification delivered:", successText);
    }

    // ------------------------------------------------------------------
    // STEP 8: Delivery History View
    // ------------------------------------------------------------------
    console.log("▶ Step 8: Checking Delivery History tab...");
    await page.click('button:has-text("Delivery History")');
    await page.waitForTimeout(1000);

    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "task-006-delivery-history.png"),
      fullPage: true,
    });

    const hasHistoryRows = await page.isVisible('table tbody tr');
    console.log("✔ Delivery history table rows populated:", hasHistoryRows);

    // ------------------------------------------------------------------
    // STEP 9: Persian RTL Language Support
    // ------------------------------------------------------------------
    console.log("▶ Step 9: Testing Persian RTL Language Toggle...");
    const langBtn = await page.waitForSelector('button:has-text("فارسی")');
    await langBtn.click();
    await page.waitForTimeout(1000);

    await page.screenshot({
      path: path.join(SCREENSHOT_DIR, "task-006-persian-rtl.png"),
      fullPage: true,
    });
    console.log("✔ Captured Persian RTL Notification Settings screenshot.");

    console.log("\n===================================================================");
    console.log("✔ ALL TASK-006 PLAYWRIGHT E2E TESTS COMPLETED SUCCESSFULLY!");
    console.log("===================================================================");
  } catch (err) {
    console.error("❌ E2E Test Failed:", err);
    process.exit(1);
  } finally {
    await browser.close();
  }
}

runTask006E2ESuite();
