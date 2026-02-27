'use strict';
/**
 * yahoo.cjs — Yahoo Finance page scrapers.
 * Responsibilities: navigate to each Yahoo Finance section and extract structured data.
 * All functions share a single Playwright page passed in from the caller.
 */

/** Navigate to url and wait for the page to settle. */
async function navAndWait(page, url) {
  await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 30000 });
  await page.waitForLoadState('networkidle', { timeout: 8000 }).catch(() => {});
  await page.waitForTimeout(randomInt(1200, 2600));
}

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

function isEmptyData(data) {
  if (data == null) return true;
  if (Array.isArray(data)) return data.length === 0;
  if (typeof data === 'object') return Object.keys(data).length === 0;
  return false;
}

async function withRetry(label, fn, attempts = 3) {
  let lastErr;
  for (let i = 1; i <= attempts; i++) {
    try {
      return await fn();
    } catch (err) {
      lastErr = err;
      if (i < attempts) {
        const baseMs = 600 * Math.pow(2, i - 1);
        const jitterMs = randomInt(120, 420);
        const backoffMs = baseMs + jitterMs;
        await sleep(backoffMs);
      }
    }
  }
  throw new Error(`${label} failed after ${attempts} attempts: ${lastErr?.message || 'unknown error'}`);
}

async function fetchSection(page, url, extractor, label) {
  return withRetry(label, async () => {
    await sleep(randomInt(400, 900));
    await navAndWait(page, url);
    const data = await page.evaluate(extractor);
    if (isEmptyData(data)) {
      throw new Error('empty section data');
    }
    return data;
  });
}

/** Extract all rows from Yahoo's financial table (financials / balance / cashflow). */
function extractTableRows() {
  const rows = {};
  document.querySelectorAll('.tableBody .row').forEach(row => {
    const cols  = row.querySelectorAll('.column');
    const label = cols[0]?.textContent?.trim();
    const vals  = Array.from(cols).slice(1).map(c => c.textContent.trim()).filter(Boolean);
    if (label && vals.length) rows[label] = vals;
  });
  return rows;
}

async function fetchQuote(page, symbol) {
  return fetchSection(
    page,
    `https://finance.yahoo.com/quote/${symbol}`,
    () => {
      const d = {};
      const put = (k, v) => {
        if (k && v) d[k] = v;
      };
      put('price', document.querySelector('[data-testid="qsp-price"]')?.textContent?.trim());
      put('change', document.querySelector('[data-testid="qsp-price-change"]')?.textContent?.trim());
      put('changePct', document.querySelector('[data-testid="qsp-price-change-percent"]')?.textContent?.trim());
      put('preMarket', document.querySelector('[data-testid="qsp-pre-price"]')?.textContent?.trim());
      document.querySelectorAll('[data-testid="quote-statistics"] li').forEach(li => {
        const label = li.querySelector('span:first-child')?.textContent?.trim();
        const value = li.querySelector('span:last-child')?.textContent?.trim();
        if (label && value && label !== value) d[label] = value;
      });
      return d;
    },
    `quote:${symbol}`
  );
}

async function fetchFinancials(page, symbol) {
  return fetchSection(page, `https://finance.yahoo.com/quote/${symbol}/financials`, extractTableRows, `financials:${symbol}`);
}

async function fetchBalanceSheet(page, symbol) {
  return fetchSection(page, `https://finance.yahoo.com/quote/${symbol}/balance-sheet`, extractTableRows, `balance:${symbol}`);
}

async function fetchCashFlow(page, symbol) {
  return fetchSection(page, `https://finance.yahoo.com/quote/${symbol}/cash-flow`, extractTableRows, `cashflow:${symbol}`);
}

async function fetchKeyStats(page, symbol) {
  return fetchSection(
    page,
    `https://finance.yahoo.com/quote/${symbol}/key-statistics`,
    () => {
    const stats = {};
    document.querySelectorAll('table tr').forEach(tr => {
      const cells = tr.querySelectorAll('td');
      if (cells.length >= 2) {
        const label = cells[0]?.textContent?.trim();
        const value = cells[1]?.textContent?.trim();
        if (label && value) stats[label] = value;
      }
    });
      return stats;
    },
    `keystats:${symbol}`
  );
}

async function fetchAnalysis(page, symbol) {
  return fetchSection(
    page,
    `https://finance.yahoo.com/quote/${symbol}/analysis`,
    () => {
    const s = {};
    ['revenueEstimate', 'earningsEstimate', 'earningsHistory', 'epsTrend', 'epsRevisions', 'growthEstimate']
      .forEach(id => {
        const el = document.querySelector(`[data-testid="${id}"]`);
        if (el) s[id] = el.innerText.replace(/\t/g, ' | ').trim();
      });
    return s;
    },
    `analysis:${symbol}`
  );
}

module.exports = { fetchQuote, fetchFinancials, fetchBalanceSheet, fetchCashFlow, fetchKeyStats, fetchAnalysis };
