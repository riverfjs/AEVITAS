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

async function main() {
  const args   = process.argv.slice(2);
  const symbol = (args[0] || '').toUpperCase();
  const mode   = (args[1] || 'all').toLowerCase();

  if (!symbol) {
    console.error('Usage: node fetch.cjs <SYMBOL> [quote|financials|balance|cashflow|keystats|analysis|all]');
    process.exit(1);
  }

  const all = mode === 'all';

  await notify(`📊 开始抓取 ${symbol} 基本面数据…`);

  const weStartedChrome = await startChrome();
  let browser;
  try {
    browser = await connectBrowser();
  } catch (e) {
    console.error(`Cannot connect to browser: ${e.message}`);
    process.exit(1);
  }

  const page   = await getPage(browser);
  const result = { symbol, fetchedAt: new Date().toISOString() };

  try {
    if (all || mode === 'quote') {
      await notify(`${symbol} ⏳ 报价 & 估值…`);
      result.quote = await fetchQuote(page, symbol);
    }
    if (all || mode === 'financials') {
      await notify(`${symbol} ⏳ 利润表…`);
      result.financials = await fetchFinancials(page, symbol);
    }
    if (all || mode === 'balance') {
      await notify(`${symbol} ⏳ 资产负债表…`);
      result.balanceSheet = await fetchBalanceSheet(page, symbol);
    }
    if (all || mode === 'cashflow') {
      await notify(`${symbol} ⏳ 现金流量表…`);
      result.cashFlow = await fetchCashFlow(page, symbol);
    }
    if (all || mode === 'keystats') {
      await notify(`${symbol} ⏳ 关键统计…`);
      result.keyStats = await fetchKeyStats(page, symbol);
    }
    if (all || mode === 'analysis') {
      await notify(`${symbol} ⏳ 分析师预期…`);
      result.analysis = await fetchAnalysis(page, symbol);
    }
  } catch (e) {
    result.error = e.message;
  }

  await notify(`${symbol} ✅ 数据抓取完成`);

  console.log(JSON.stringify(result, null, 2));

  await browser.close().catch(() => {});
  if (weStartedChrome) await stopChrome();
}

main().catch(e => { console.error(e.message); process.exit(1); });
