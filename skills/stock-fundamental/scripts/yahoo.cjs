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
  await page.waitForTimeout(1500);
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
  await navAndWait(page, `https://finance.yahoo.com/quote/${symbol}`);
  return page.evaluate(() => {
    const d = {};
    d.price     = document.querySelector('[data-testid="qsp-price"]')?.textContent?.trim();
    d.change    = document.querySelector('[data-testid="qsp-price-change"]')?.textContent?.trim();
    d.changePct = document.querySelector('[data-testid="qsp-price-change-percent"]')?.textContent?.trim();
    d.preMarket = document.querySelector('[data-testid="qsp-pre-price"]')?.textContent?.trim();
    document.querySelectorAll('[data-testid="quote-statistics"] li').forEach(li => {
      const label = li.querySelector('span:first-child')?.textContent?.trim();
      const value = li.querySelector('span:last-child')?.textContent?.trim();
      if (label && value && label !== value) d[label] = value;
    });
    return d;
  });
}

async function fetchFinancials(page, symbol) {
  await navAndWait(page, `https://finance.yahoo.com/quote/${symbol}/financials`);
  return page.evaluate(extractTableRows);
}

async function fetchBalanceSheet(page, symbol) {
  await navAndWait(page, `https://finance.yahoo.com/quote/${symbol}/balance-sheet`);
  return page.evaluate(extractTableRows);
}

async function fetchCashFlow(page, symbol) {
  await navAndWait(page, `https://finance.yahoo.com/quote/${symbol}/cash-flow`);
  return page.evaluate(extractTableRows);
}

async function fetchKeyStats(page, symbol) {
  await navAndWait(page, `https://finance.yahoo.com/quote/${symbol}/key-statistics`);
  return page.evaluate(() => {
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
  });
}

async function fetchAnalysis(page, symbol) {
  await navAndWait(page, `https://finance.yahoo.com/quote/${symbol}/analysis`);
  return page.evaluate(() => {
    const s = {};
    ['revenueEstimate', 'earningsEstimate', 'earningsHistory', 'epsTrend', 'epsRevisions', 'growthEstimate']
      .forEach(id => {
        const el = document.querySelector(`[data-testid="${id}"]`);
        if (el) s[id] = el.innerText.replace(/\t/g, ' | ').trim();
      });
    return s;
  });
}

module.exports = { fetchQuote, fetchFinancials, fetchBalanceSheet, fetchCashFlow, fetchKeyStats, fetchAnalysis };
