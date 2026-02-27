#!/usr/bin/env node
'use strict';
/**
 * scrape.cjs — Navigate to a URL and evaluate JavaScript in the same process.
 *
 * This is the PRIMARY tool for data extraction. By running nav + eval in a
 * single Playwright process, it avoids the "Execution context was destroyed"
 * error that happens when nav.cjs and eval.cjs run as separate processes.
 *
 * Usage:
 *   node scrape.cjs <url> '<javascript-expression>'
 *
 * The JS expression is evaluated with page.evaluate() and must return a
 * JSON-serialisable value. Use document.* APIs freely — runs in page context.
 *
 * Examples:
 *   # Get page title
 *   node scrape.cjs https://example.com 'document.title'
 *
 *   # Extract a list of items
 *   node scrape.cjs https://movie.douban.com/top250 \
 *     'Array.from(document.querySelectorAll(".item")).slice(0,10).map(el => ({
 *        rank:   el.querySelector(".pic em")?.textContent,
 *        title:  el.querySelector(".title")?.textContent,
 *        rating: el.querySelector(".rating_num")?.textContent,
 *        info:   el.querySelector(".bd p")?.textContent?.trim()
 *     }))'
 *
 *   # Get all links on a page
 *   node scrape.cjs https://news.ycombinator.com \
 *     'Array.from(document.querySelectorAll("a.titlelink")).map(a=>({title:a.textContent,url:a.href}))'
 *
 * Output: JSON to stdout, error to stderr.
 * Exit 0 on success, 1 on failure.
 *
 * Chrome lifecycle:
 *   - Starts Chrome automatically if not already running.
 *   - Does NOT stop Chrome after scraping (reuse across multiple calls).
 *   - Call stop.cjs explicitly when completely done.
 */

const { spawn } = require('child_process');
const path = require('path');
const os   = require('fs');
const fs   = require('fs');

const SKILL_DIR  = path.join(require('os').homedir(), '.aevitas/workspace/.claude/skills/browser');
const { chromium } = require(path.join(SKILL_DIR, 'node_modules/playwright'));

const CDP_PORT    = 9222;
const CDP_URL     = `http://127.0.0.1:${CDP_PORT}`;
const PROFILE_DIR = path.join(require('os').homedir(), '.aevitas', 'browser-profile');

// ── Chrome lifecycle ──────────────────────────────────────────────────────────

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

async function ensureChrome() {
  if (await isChromeRunning()) return;

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
    if (await isChromeRunning()) return;
  }
  throw new Error('Chrome started but CDP not ready after 10s');
}

// ── Main ─────────────────────────────────────────────────────────────────────

async function main() {
  const url  = process.argv[2];
  const code = process.argv[3];

  if (!url || !code) {
    console.error('Usage: node scrape.cjs <url> \'<javascript-expression>\'');
    process.exit(1);
  }

  await ensureChrome();

  const browser = await chromium.connectOverCDP(CDP_URL, { timeout: 5000 });
  const contexts = browser.contexts();
  if (!contexts.length) throw new Error('No browser context');

  const page = contexts[0].pages()[0] ?? await contexts[0].newPage();

  // Navigate and wait for network to settle.
  // 3s extra wait is intentional — many sites (e.g. Douban ranking) populate
  // list content via JS after networkidle, and need extra time to render.
  await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 30000 });
  await page.waitForLoadState('networkidle', { timeout: 8000 }).catch(() => {});
  await page.waitForTimeout(3000);

  // Evaluate JS in page context — same process as nav, no context destruction
  const result = await page.evaluate(code);

  // Include final URL and title so the caller can verify the source
  const finalUrl   = page.url();
  const finalTitle = await page.title();

  console.log(JSON.stringify({ url: finalUrl, title: finalTitle, data: result }, null, 2));

  // Disconnect but leave Chrome running for subsequent calls
  await browser.close().catch(() => {});
}

main().catch(e => { console.error(e.message); process.exit(1); });
