#!/usr/bin/env node
// Start Chrome with CDP for Playwright connection
const { spawn } = require('child_process');
const path = require('path');
const os = require('os');
const fs = require('fs');

const PROFILE_DIR = path.join(os.homedir(), '.myclaw', 'browser-profile');
const CDP_PORT = 9222;

function findChrome() {
  const paths = {
    darwin: [
      '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
      '/Applications/Chromium.app/Contents/MacOS/Chromium',
    ],
    linux: [
      '/usr/bin/google-chrome',
      '/usr/bin/google-chrome-stable',
      '/usr/bin/chromium',
      '/usr/bin/chromium-browser',
    ],
    win32: [
      'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe',
      'C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe',
    ],
  };

  const candidates = paths[process.platform] || [];
  for (const p of candidates) {
    if (fs.existsSync(p)) return p;
  }

  throw new Error('Chrome not found. Please install Google Chrome.');
}

async function isChromeRunning() {
  try {
    const res = await fetch(`http://127.0.0.1:${CDP_PORT}/json/version`, {
      signal: AbortSignal.timeout(500),
    });
    return res.ok;
  } catch {
    return false;
  }
}

async function startChrome() {
  if (await isChromeRunning()) {
    console.log('Chrome already running on port', CDP_PORT);
    return;
  }

  const chromePath = findChrome();
  fs.mkdirSync(PROFILE_DIR, { recursive: true });

  const args = [
    `--remote-debugging-port=${CDP_PORT}`,
    `--user-data-dir=${PROFILE_DIR}`,
    '--no-first-run',
    '--no-default-browser-check',
    '--headless=new',
    '--window-size=1440,900',
    '--force-device-scale-factor=2',
    // Anti-bot detection
    '--disable-blink-features=AutomationControlled',
    '--disable-dev-shm-usage',
    '--disable-infobars',
    '--disable-sync',
    '--disable-background-networking',
    '--disable-component-update',
    '--disable-features=Translate,MediaRouter',
    '--disable-session-crashed-bubble',
    '--hide-crash-restore-bubble',
    '--password-store=basic',
    '--user-agent=Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
    'about:blank',
  ];

  console.log('Starting Chrome (headless) on port', CDP_PORT);
  
  const chrome = spawn(chromePath, args, {
    detached: true,
    stdio: 'ignore',
  });

  chrome.unref();

  // Wait for Chrome to be ready
  for (let i = 0; i < 20; i++) {
    await new Promise(r => setTimeout(r, 500));
    if (await isChromeRunning()) {
      console.log('Chrome ready');
      return;
    }
  }

  throw new Error('Chrome started but CDP not ready');
}

startChrome().catch(err => {
  console.error('Error:', err.message);
  process.exit(1);
});

