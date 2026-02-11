#!/usr/bin/env node
// Capture screenshot
const { getActivePage } = require('../browser-manager.js');
const path = require('path');
const os = require('os');

const fullPage = process.argv.includes('--full');
const selectorArg = process.argv.find(arg => arg.startsWith('--selector='));
const selector = selectorArg ? selectorArg.split('=')[1] : null;

async function screenshot() {
  try {
    const page = await getActivePage();
    
    await page.waitForLoadState('domcontentloaded', { timeout: 3000 }).catch(() => {});

    const filename = `screenshot-${Date.now()}.png`;
    const filepath = path.join(os.tmpdir(), filename);

    const options = { path: filepath, timeout: 10000 };
    if (fullPage) options.fullPage = true;

    if (selector) {
      await page.locator(selector).first().screenshot(options);
    } else {
      await page.screenshot(options);
    }

    console.log(JSON.stringify({
      path: filepath,
      filename: filename,
      fullPage: fullPage || false,
      selector: selector,
    }));

    process.exit(0);
  } catch (error) {
    console.error(JSON.stringify({
      error: 'Screenshot failed',
      message: error.message,
    }));
    process.exit(1);
  }
}

screenshot();
