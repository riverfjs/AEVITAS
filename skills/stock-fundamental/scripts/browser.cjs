'use strict';
/**
 * browser.cjs — Chrome lifecycle management via CDP.
 * Responsibilities: find Chrome binary, start/stop headless Chrome, connect Playwright over CDP.
 */

const { spawn, exec } = require('child_process');
const fs   = require('fs');
const os   = require('os');
const path = require('path');
const { chromium } = require(
  path.join(os.homedir(), '.myclaw/workspace/.claude/skills/browser/node_modules/playwright')
);

const CDP_PORT    = 9222;
const CDP_URL     = `http://127.0.0.1:${CDP_PORT}`;
const PROFILE_DIR = path.join(os.homedir(), '.myclaw', 'browser-profile');

function findChrome() {
  const candidates = {
    darwin: [
      '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
      '/Applications/Chromium.app/Contents/MacOS/Chromium',
    ],
    linux:  ['/usr/bin/google-chrome', '/usr/bin/google-chrome-stable', '/usr/bin/chromium-browser'],
    win32:  ['C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe'],
  }[process.platform] || [];

  for (const p of candidates) if (fs.existsSync(p)) return p;
  throw new Error('Chrome not found. Install Google Chrome.');
}

async function isChromeRunning() {
  try {
    const res = await fetch(`${CDP_URL}/json/version`, { signal: AbortSignal.timeout(500) });
    return res.ok;
  } catch { return false; }
}

/**
 * Start Chrome if not already running.
 * @returns {boolean} true if we started Chrome (caller should stop it when done)
 */
async function startChrome() {
  if (await isChromeRunning()) return false;

  const chromePath = findChrome();
  fs.mkdirSync(PROFILE_DIR, { recursive: true });

  const chrome = spawn(chromePath, [
    `--remote-debugging-port=${CDP_PORT}`,
    `--user-data-dir=${PROFILE_DIR}`,
    '--headless=new',
    '--no-first-run',
    '--no-default-browser-check',
    '--window-size=1440,900',
    '--disable-blink-features=AutomationControlled',
    '--disable-dev-shm-usage',
    '--disable-features=Translate,MediaRouter',
    '--user-agent=Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
    'about:blank',
  ], { detached: true, stdio: 'ignore' });
  chrome.unref();

  for (let i = 0; i < 20; i++) {
    await new Promise(r => setTimeout(r, 500));
    if (await isChromeRunning()) return true;
  }
  throw new Error('Chrome started but CDP not ready after 10s');
}

/** Kill the Chrome instance started by startChrome(). */
function stopChrome() {
  return new Promise(resolve => {
    const cmd = process.platform === 'win32'
      ? 'taskkill /F /IM chrome.exe /T'
      : `pkill -9 -f "remote-debugging-port=${CDP_PORT}"`;
    exec(cmd, () => resolve());
  });
}

/** Connect Playwright to the running Chrome instance. */
async function connectBrowser() {
  return chromium.connectOverCDP(CDP_URL, { timeout: 5000 });
}

/** Get or create a page in the first browser context. */
async function getPage(browser) {
  const contexts = browser.contexts();
  if (!contexts.length) throw new Error('No browser context available');
  const pages = contexts[0].pages();
  return pages.length ? pages[0] : await contexts[0].newPage();
}

module.exports = { startChrome, stopChrome, connectBrowser, getPage };
