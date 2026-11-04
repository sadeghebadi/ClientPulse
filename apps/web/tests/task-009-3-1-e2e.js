const { chromium } = require("playwright");
const path = require("path");
const fs = require("fs");
const { execSync } = require("child_process");

const BASE_URL = process.env.BASE_URL || "http://localhost:3001";
const SCREENSHOT_DIR = path.resolve(__dirname, "../../../task-report/screenshots/task-009-3-1");

if (!fs.existsSync(SCREENSHOT_DIR)) {
  fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
}

function runSql(query) {
  const sanitized = query.replace(/"/g, '\\"');
  return execSync(`docker exec clientpulse_postgres psql -U clientpulse -d clientpulse -c "${sanitized}"`).toString();
}

async function runTask009_3_1_E2E() {
  console.log("===================================================================");
  console.log("TASK-009.3.1: Persian & English Language Activation Management E2E");
  console.log("Web URL:", BASE_URL);
  console.log("Screenshots:", SCREENSHOT_DIR);
  console.log("===================================================================\n");

  const adminEmail = `pa-e2e-task009-3@pulse.test`;
  const adminPassword = "PlatformAdminSecurePass123!#";

  // Ensure DB baseline state has both languages enabled with English default
  runSql("UPDATE platform_localization_settings SET enabled_locales = ARRAY['en', 'fa'], default_locale = 'en' WHERE id = 1;");

  const browser = await chromium.launch({
    executablePath: "/usr/bin/google-chrome",
    headless: true,
    args: ["--no-sandbox", "--disable-setuid-sandbox"],
  });

  try {
    const desktopContext = await browser.newContext({
      viewport: { width: 1440, height: 900 },
    });

    const mobileContext = await browser.newContext({
      viewport: { width: 390, height: 844 },
    });

    const page = await desktopContext.newPage();
    page.on("dialog", async (dialog) => {
      await dialog.accept();
    });

    // -------------------------------------------------------------
    // SCENARIO 1: Both Languages Enabled (Baseline)
    // -------------------------------------------------------------
    console.log("▶ Step 1: Testing Landing Page with Both Languages Enabled...");
    await page.goto(`${BASE_URL}/`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1000);

    // Verify language selector is present
    const langBtn = page.locator('[data-testid="language-selector-button"]');
    const isLangBtnVisible = await langBtn.isVisible();
    console.log(`  Language selector button visible on landing page: ${isLangBtnVisible}`);
    if (!isLangBtnVisible) throw new Error("Expected language selector to be visible when both languages are enabled");

    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "01_both_languages_landing_desktop.png"), fullPage: true });
    console.log("✔ Saved 01_both_languages_landing_desktop.png");

    // Capture mobile landing view
    const mobilePage = await mobileContext.newPage();
    await mobilePage.goto(`${BASE_URL}/`, { waitUntil: "networkidle" });
    await mobilePage.waitForTimeout(1000);
    await mobilePage.screenshot({ path: path.join(SCREENSHOT_DIR, "02_both_languages_landing_mobile.png") });
    console.log("✔ Saved 02_both_languages_landing_mobile.png");
    await mobilePage.close();

    // Toggle to Persian from landing page
    console.log("  Switching to Persian via language selector...");
    await langBtn.click();
    await page.waitForTimeout(800);
    const htmlDirFa = await page.getAttribute("html", "dir");
    console.log(`  HTML dir after toggle to Persian: ${htmlDirFa}`);
    if (htmlDirFa !== "rtl") throw new Error("Expected html dir to be rtl after switching to Persian");

    // Toggle back to English
    await page.locator('[data-testid="language-selector-button"]').click();
    await page.waitForTimeout(800);
    const htmlDirEn = await page.getAttribute("html", "dir");
    console.log(`  HTML dir after toggle back to English: ${htmlDirEn}`);
    if (htmlDirEn !== "ltr") throw new Error("Expected html dir to be ltr after switching to English");

    // -------------------------------------------------------------
    // SCENARIO 2: Platform Admin Panel Localization Settings (Both Enabled)
    // -------------------------------------------------------------
    console.log("\n▶ Step 2: Logging into Platform Admin Settings Localization...");
    await page.goto(`${BASE_URL}/login`, { waitUntil: "networkidle" });
    await page.fill('input[type="email"]', adminEmail);
    await page.fill('input[type="password"]', adminPassword);
    await Promise.all([
      page.waitForURL("**/dashboard", { timeout: 10000 }),
      page.click('button[type="submit"]'),
    ]);

    await page.goto(`${BASE_URL}/platform-admin/settings/localization`, { waitUntil: "networkidle" });
    await page.waitForTimeout(1500);

    // Verify both language cards are displayed
    const enCard = page.locator('[data-testid="language-card-en"]');
    const faCard = page.locator('[data-testid="language-card-fa"]');
    if (!(await enCard.isVisible()) || !(await faCard.isVisible())) {
      throw new Error("Expected both English and Persian cards to be visible in admin settings");
    }

    // Verify English has default indicator
    const enDef = page.locator('[data-testid="default-indicator-en"]');
    if (!(await enDef.isVisible())) {
      throw new Error("Expected English to have default indicator");
    }

    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "03_both_languages_enabled_admin.png"), fullPage: true });
    console.log("✔ Saved 03_both_languages_enabled_admin.png");

    // -------------------------------------------------------------
    // SCENARIO 3: Default Language Transition (English -> Persian)
    // -------------------------------------------------------------
    console.log("\n▶ Step 3: Transitioning Default Language to Persian...");
    const setDefaultFaBtn = page.locator('[data-testid="set-default-btn-fa"]');
    await setDefaultFaBtn.click();
    await page.waitForTimeout(1500);

    // Verify success banner and new default indicator on Persian
    const successAlert = page.locator('[data-testid="localization-success-alert"]');
    console.log("  Success banner text:", await successAlert.innerText());
    const faDef = page.locator('[data-testid="default-indicator-fa"]');
    if (!(await faDef.isVisible())) {
      throw new Error("Expected Persian to become default platform fallback");
    }

    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "04_default_language_transition_to_persian.png"), fullPage: true });
    console.log("✔ Saved 04_default_language_transition_to_persian.png");

    // -------------------------------------------------------------
    // SCENARIO 4: Persian Only (English Disabled)
    // -------------------------------------------------------------
    console.log("\n▶ Step 4: Disabling English (Persian Only)...");
    const switchEn = page.locator('[data-testid="switch-en"]');
    await switchEn.click();
    await page.waitForTimeout(1500);

    // Verify English card is now disabled
    const enDisabledBadge = page.locator('[data-testid="status-badge-en-disabled"]');
    if (!(await enDisabledBadge.isVisible())) {
      throw new Error("Expected English to show disabled status");
    }

    // Verify Persian card has disabled switch because it's the final active language
    const switchFa = page.locator('[data-testid="switch-fa"]');
    const isFaSwitchDisabled = await switchFa.isDisabled();
    console.log(`  Persian switch disabled (final language protection): ${isFaSwitchDisabled}`);
    if (!isFaSwitchDisabled) throw new Error("Expected Persian switch to be disabled as final active language");

    // Verify header language switcher is completely hidden
    const adminHeaderLangBtn = page.locator('[data-testid="language-selector-button"]');
    const isHeaderLangVisible = await adminHeaderLangBtn.isVisible();
    console.log(`  Language selector hidden in header: ${!isHeaderLangVisible}`);
    if (isHeaderLangVisible) throw new Error("Expected language selector to be completely hidden when only one language is enabled");

    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "05_persian_only_admin.png"), fullPage: true });
    console.log("✔ Saved 05_persian_only_admin.png");

    // Verify Public Landing Page in Persian-only mode
    const publicContextFa = await browser.newContext({ viewport: { width: 1440, height: 900 } });
    const publicPageFa = await publicContextFa.newPage();
    await publicPageFa.goto(`${BASE_URL}/`, { waitUntil: "networkidle" });
    await publicPageFa.waitForTimeout(1000);

    const publicFaLangBtn = publicPageFa.locator('[data-testid="language-selector-button"]');
    if (await publicFaLangBtn.isVisible()) {
      throw new Error("Expected language selector to be hidden on landing page when only Persian is enabled");
    }
    const publicFaDir = await publicPageFa.getAttribute("html", "dir");
    console.log(`  Landing page direction under Persian-only: ${publicFaDir}`);

    await publicPageFa.screenshot({ path: path.join(SCREENSHOT_DIR, "06_persian_only_landing_selector_hidden.png"), fullPage: true });
    console.log("✔ Saved 06_persian_only_landing_selector_hidden.png");

    // Mobile landing page in Persian-only mode
    const mobileFaContext = await browser.newContext({ viewport: { width: 390, height: 844 } });
    const mobileFaPage = await mobileFaContext.newPage();
    await mobileFaPage.goto(`${BASE_URL}/`, { waitUntil: "networkidle" });
    await mobileFaPage.waitForTimeout(1000);
    await mobileFaPage.screenshot({ path: path.join(SCREENSHOT_DIR, "07_persian_only_landing_mobile.png") });
    console.log("✔ Saved 07_persian_only_landing_mobile.png");
    await mobileFaContext.close();

    // -------------------------------------------------------------
    // SCENARIO 5: Automatic Fallback from Disabled English
    // -------------------------------------------------------------
    console.log("\n▶ Step 5: Testing Automatic Fallback for user with saved English locale...");
    const fallbackContextEn = await browser.newContext({ viewport: { width: 1440, height: 900 } });
    const fallbackPageEn = await fallbackContextEn.newPage();
    // Simulate visitor who previously saved 'en' in localStorage
    await fallbackPageEn.goto(`${BASE_URL}/login`, { waitUntil: "commit" });
    await fallbackPageEn.evaluate(() => localStorage.setItem("clientpulse_locale", "en"));
    await fallbackPageEn.goto(`${BASE_URL}/login`, { waitUntil: "networkidle" });
    await fallbackPageEn.waitForTimeout(1500);

    // Verify localStorage automatically switched to 'fa'
    const storedLocaleAfterFallback = await fallbackPageEn.evaluate(() => localStorage.getItem("clientpulse_locale"));
    const htmlLangAfterFallback = await fallbackPageEn.getAttribute("html", "lang");
    const htmlDirAfterFallback = await fallbackPageEn.getAttribute("html", "dir");
    console.log(`  Stored locale after fallback: ${storedLocaleAfterFallback}`);
    console.log(`  HTML lang: ${htmlLangAfterFallback}, dir: ${htmlDirAfterFallback}`);
    if (storedLocaleAfterFallback !== "fa" || htmlDirAfterFallback !== "rtl") {
      throw new Error("Expected automatic fallback from disabled English to Persian (fa / rtl)");
    }

    await fallbackPageEn.screenshot({ path: path.join(SCREENSHOT_DIR, "08_automatic_fallback_from_disabled_english.png"), fullPage: true });
    console.log("✔ Saved 08_automatic_fallback_from_disabled_english.png");
    await fallbackContextEn.close();
    await publicContextFa.close();

    // -------------------------------------------------------------
    // SCENARIO 6: Platform Admin Re-Enabling English
    // -------------------------------------------------------------
    console.log("\n▶ Step 6: Platform Admin Re-Enabling English...");
    // English card is still visible even when disabled!
    await page.bringToFront();
    await switchEn.click();
    await page.waitForTimeout(1500);

    // Verify English is active again
    const enActiveBadge = page.locator('[data-testid="status-badge-en-active"]');
    if (!(await enActiveBadge.isVisible())) {
      throw new Error("Expected English to be active again");
    }

    // Verify language switcher in header immediately reappears
    const reenabledHeaderLangBtn = page.locator('[data-testid="language-selector-button"]');
    if (!(await reenabledHeaderLangBtn.isVisible())) {
      throw new Error("Expected language selector to reappear in header once both are enabled");
    }

    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "09_admin_re_enabling_english.png"), fullPage: true });
    console.log("✔ Saved 09_admin_re_enabling_english.png");

    // -------------------------------------------------------------
    // SCENARIO 7: English Only (Persian Disabled)
    // -------------------------------------------------------------
    console.log("\n▶ Step 7: Setting English Default & Disabling Persian (English Only)...");
    // Switch default back to English
    const setDefaultEnBtn = page.locator('[data-testid="set-default-btn-en"]');
    await setDefaultEnBtn.click();
    await page.waitForTimeout(1500);

    // Now disable Persian
    await switchFa.click();
    await page.waitForTimeout(1500);

    // Verify Persian card is disabled
    const faDisabledBadge = page.locator('[data-testid="status-badge-fa-disabled"]');
    if (!(await faDisabledBadge.isVisible())) {
      throw new Error("Expected Persian to show disabled status");
    }

    // Verify English switch is disabled (final language protection)
    if (!(await switchEn.isDisabled())) {
      throw new Error("Expected English switch to be disabled as final active language");
    }

    // Verify language selector is hidden
    if (await page.locator('[data-testid="language-selector-button"]').isVisible()) {
      throw new Error("Expected language selector to be hidden when only English is enabled");
    }

    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "10_english_only_admin.png"), fullPage: true });
    console.log("✔ Saved 10_english_only_admin.png");

    // Landing page in English-only mode
    const publicContextEn = await browser.newContext({ viewport: { width: 1440, height: 900 } });
    const publicPageEn = await publicContextEn.newPage();
    await publicPageEn.goto(`${BASE_URL}/`, { waitUntil: "networkidle" });
    await publicPageEn.waitForTimeout(1000);

    if (await publicPageEn.locator('[data-testid="language-selector-button"]').isVisible()) {
      throw new Error("Expected language selector to be hidden on landing page in English-only mode");
    }

    await publicPageEn.screenshot({ path: path.join(SCREENSHOT_DIR, "11_english_only_landing_selector_hidden.png"), fullPage: true });
    console.log("✔ Saved 11_english_only_landing_selector_hidden.png");

    // Mobile landing page in English-only mode
    const mobileEnContext = await browser.newContext({ viewport: { width: 390, height: 844 } });
    const mobileEnPage = await mobileEnContext.newPage();
    await mobileEnPage.goto(`${BASE_URL}/`, { waitUntil: "networkidle" });
    await mobileEnPage.waitForTimeout(1000);
    await mobileEnPage.screenshot({ path: path.join(SCREENSHOT_DIR, "12_english_only_landing_mobile.png") });
    console.log("✔ Saved 12_english_only_landing_mobile.png");
    await mobileEnContext.close();

    // -------------------------------------------------------------
    // SCENARIO 8: Automatic Fallback from Disabled Persian
    // -------------------------------------------------------------
    console.log("\n▶ Step 8: Testing Automatic Fallback for user with saved Persian locale...");
    const fallbackContextFa = await browser.newContext({ viewport: { width: 1440, height: 900 } });
    const fallbackPageFa = await fallbackContextFa.newPage();
    // Simulate visitor who previously saved 'fa' in localStorage
    await fallbackPageFa.goto(`${BASE_URL}/login`, { waitUntil: "commit" });
    await fallbackPageFa.evaluate(() => localStorage.setItem("clientpulse_locale", "fa"));
    await fallbackPageFa.goto(`${BASE_URL}/login`, { waitUntil: "networkidle" });
    await fallbackPageFa.waitForTimeout(1500);

    // Verify localStorage automatically switched to 'en'
    const storedLocaleAfterFaFallback = await fallbackPageFa.evaluate(() => localStorage.getItem("clientpulse_locale"));
    const htmlLangAfterFaFallback = await fallbackPageFa.getAttribute("html", "lang");
    const htmlDirAfterFaFallback = await fallbackPageFa.getAttribute("html", "dir");
    console.log(`  Stored locale after fallback: ${storedLocaleAfterFaFallback}`);
    console.log(`  HTML lang: ${htmlLangAfterFaFallback}, dir: ${htmlDirAfterFaFallback}`);
    if (storedLocaleAfterFaFallback !== "en" || htmlDirAfterFaFallback !== "ltr") {
      throw new Error("Expected automatic fallback from disabled Persian to English (en / ltr)");
    }

    await fallbackPageFa.screenshot({ path: path.join(SCREENSHOT_DIR, "13_automatic_fallback_from_disabled_persian.png"), fullPage: true });
    console.log("✔ Saved 13_automatic_fallback_from_disabled_persian.png");
    await fallbackContextFa.close();
    await publicContextEn.close();

    // -------------------------------------------------------------
    // SCENARIO 9: Platform Admin Re-Enabling Persian
    // -------------------------------------------------------------
    console.log("\n▶ Step 9: Platform Admin Re-Enabling Persian...");
    await page.bringToFront();
    await switchFa.click();
    await page.waitForTimeout(1500);

    const faActiveBadge = page.locator('[data-testid="status-badge-fa-active"]');
    if (!(await faActiveBadge.isVisible())) {
      throw new Error("Expected Persian to be active again");
    }

    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "14_admin_re_enabling_persian.png"), fullPage: true });
    console.log("✔ Saved 14_admin_re_enabling_persian.png");

    // -------------------------------------------------------------
    // SCENARIO 10: Graceful Language Transition for Admin in Disabled Locale
    // -------------------------------------------------------------
    console.log("\n▶ Step 10: Testing Admin viewing in Persian disabling Persian...");
    // Switch admin session to Persian
    await page.locator('[data-testid="language-selector-button"]').click();
    await page.waitForTimeout(1000);
    const adminDirInFa = await page.getAttribute("html", "dir");
    console.log(`  Admin panel direction in Persian: ${adminDirInFa}`);

    // Now disable Persian while admin is viewing in Persian!
    // Since Persian is not default (English is default), clicking switch-fa disables Persian
    await page.locator('[data-testid="switch-fa"]').click();
    await page.waitForTimeout(2000);

    // Verify admin was gracefully switched to English without being logged out or redirected
    const currentUrl = page.url();
    const adminDirAfterDisabling = await page.getAttribute("html", "dir");
    console.log(`  Admin URL after disabling Persian: ${currentUrl}`);
    console.log(`  Admin direction after disabling Persian: ${adminDirAfterDisabling}`);

    if (!currentUrl.includes("/platform-admin/settings/localization")) {
      throw new Error("Admin was redirected away from settings page!");
    }
    if (adminDirAfterDisabling !== "ltr") {
      throw new Error("Admin UI did not gracefully switch to English (ltr)!");
    }

    // Verify admin can re-enable Persian right now
    await page.locator('[data-testid="switch-fa"]').click();
    await page.waitForTimeout(1500);
    console.log("  Successfully re-enabled Persian from English view!");

    await page.screenshot({ path: path.join(SCREENSHOT_DIR, "15_admin_graceful_language_transition.png"), fullPage: true });
    console.log("✔ Saved 15_admin_graceful_language_transition.png");

    console.log("\n===================================================================");
    console.log("🎉 ALL E2E TEST SCENARIOS PASSED SUCCESSFULLY!");
    console.log("===================================================================");
  } finally {
    // Restore baseline
    runSql("UPDATE platform_localization_settings SET enabled_locales = ARRAY['en', 'fa'], default_locale = 'en' WHERE id = 1;");
    await browser.close();
  }
}

runTask009_3_1_E2E().catch((err) => {
  console.error("❌ E2E Suite Failed:", err);
  process.exit(1);
});

