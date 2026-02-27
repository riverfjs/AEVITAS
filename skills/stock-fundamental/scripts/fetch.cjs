#!/usr/bin/env node
'use strict';
/**
 * fetch.cjs — Entry point for stock-fundamental skill.
 * Responsibilities: parse args, orchestrate browser + scrapers + progress notifications, output JSON.
 *
 * Usage:
 *   node fetch.cjs <SYMBOL> [quote|financials|balance|cashflow|keystats|analysis|all]
 *
 * Examples:
 *   node fetch.cjs AAPL
 *   node fetch.cjs CLF all
 *   node fetch.cjs 0700.HK quote
 */

const path = require('path');
const dir  = __dirname;

const { startChrome, stopChrome, connectBrowser, getPage } = require(path.join(dir, 'browser.cjs'));
const { fetchQuote, fetchFinancials, fetchBalanceSheet, fetchCashFlow, fetchKeyStats, fetchAnalysis } = require(path.join(dir, 'yahoo.cjs'));
const { notify } = require(path.join(dir, 'notify.cjs'));

function isEmptyData(data) {
  if (data == null) return true;
  if (Array.isArray(data)) return data.length === 0;
  if (typeof data === 'object') return Object.keys(data).length === 0;
  return false;
}

async function main() {
  const args   = process.argv.slice(2);
  const symbol = (args[0] || '').toUpperCase();
  const mode   = (args[1] || 'all').toLowerCase();
  const validModes = new Set(['quote', 'financials', 'balance', 'cashflow', 'keystats', 'analysis', 'all']);

  if (!symbol) {
    console.error('Usage: node fetch.cjs <SYMBOL> [quote|financials|balance|cashflow|keystats|analysis|all]');
    process.exit(1);
  }
  if (!validModes.has(mode)) {
    console.error(`Invalid mode: ${mode}`);
    process.exit(1);
  }

  const all = mode === 'all';

  await notify(`📊 开始抓取 ${symbol} 基本面数据（${mode}）…`);

  let weStartedChrome = false;
  let browser;
  try {
    weStartedChrome = await startChrome();
    browser = await connectBrowser();
    const page = await getPage(browser);
    const result = { symbol, fetchedAt: new Date().toISOString() };

    async function runStep(field, title, fn) {
      const data = await fn();
      if (isEmptyData(data)) throw new Error(`${title} empty result`);
      result[field] = data;
    }

    if (all || mode === 'quote') {
      await runStep('quote', '报价 & 估值', () => fetchQuote(page, symbol));
    }
    if (all || mode === 'financials') {
      await runStep('financials', '利润表', () => fetchFinancials(page, symbol));
    }
    if (all || mode === 'balance') {
      await runStep('balanceSheet', '资产负债表', () => fetchBalanceSheet(page, symbol));
    }
    if (all || mode === 'cashflow') {
      await runStep('cashFlow', '现金流量表', () => fetchCashFlow(page, symbol));
    }
    if (all || mode === 'keystats') {
      await runStep('keyStats', '关键统计', () => fetchKeyStats(page, symbol));
    }
    if (all || mode === 'analysis') {
      await runStep('analysis', '分析师预期', () => fetchAnalysis(page, symbol));
    }

    await notify(`${symbol} ✅ 数据抓取完成，开始分析`);
    console.log(JSON.stringify(result, null, 2));
  } catch (e) {
    await notify(`${symbol} ❌ 数据抓取失败，不进行分析`);
    console.error(JSON.stringify({
      symbol,
      failedAt: new Date().toISOString(),
      error: e.message || 'unknown error',
    }, null, 2));
    process.exit(1);
  } finally {
    if (browser) await browser.close().catch(() => {});
    if (weStartedChrome) await stopChrome();
  }
}

main().catch(e => { console.error(e.message); process.exit(1); });
