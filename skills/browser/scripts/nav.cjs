#!/usr/bin/env node
// Navigate to URL
const { getActivePage, connectToBrowser } = require('../browser-manager.js');

const url = process.argv[2];
const openInNewTab = process.argv.includes('--new');

if (!url) {
  console.error('Usage: nav.cjs <url> [--new]');
  process.exit(1);
}

async function navigate() {
  try {
    const browser = await connectToBrowser();
    const contexts = browser.contexts();
    if (contexts.length === 0) {
      throw new Error('No browser context');
    }
    const context = contexts[0];
    
    let page;
    if (openInNewTab) {
      page = await context.newPage();
    } else {
      page = await getActivePage();
    }

    // Inject stealth scripts before navigation
    await page.addInitScript(() => {
      // Remove webdriver property
      Object.defineProperty(navigator, 'webdriver', { get: () => undefined });
      
      // Fix plugins
      Object.defineProperty(navigator, 'plugins', {
        get: () => [1, 2, 3, 4, 5]
      });
      
      // Fix languages
      Object.defineProperty(navigator, 'languages', {
        get: () => ['en-US', 'en']
      });
    });

    await page.goto(url, {
      timeout: 30000,
      waitUntil: 'domcontentloaded',
    });

    await page.waitForLoadState('networkidle', { timeout: 5000 }).catch(() => {});

    console.log(JSON.stringify({
      action: 'navigated',
      url: page.url(),
      title: await page.title(),
      newTab: openInNewTab,
    }));

    process.exit(0);
  } catch (error) {
    console.error(JSON.stringify({
      error: 'Navigation failed',
      message: error.message,
      url: url,
    }));
    process.exit(1);
  }
}

navigate();
