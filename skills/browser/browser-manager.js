#!/usr/bin/env node
// Browser manager - connects to running Chrome via CDP
const { chromium } = require('playwright');

const CDP_URL = 'http://127.0.0.1:9222';

// Cache for single script execution
let browserConnection = null;

async function connectToBrowser() {
  if (browserConnection && browserConnection.isConnected()) {
    return browserConnection;
  }

  try {
    browserConnection = await chromium.connectOverCDP(CDP_URL, { timeout: 5000 });
    return browserConnection;
  } catch (error) {
    throw new Error(`Failed to connect to Chrome. Is it running? Try: node scripts/start.cjs\n${error.message}`);
  }
}

async function getActivePage() {
  const browser = await connectToBrowser();
  const contexts = browser.contexts();
  
  if (contexts.length === 0) {
    throw new Error('No browser context available');
  }

  const context = contexts[0];
  const pages = context.pages();

  if (pages.length === 0) {
    // Create initial page
    return await context.newPage();
  }

  // Return last active page
  return pages[pages.length - 1];
}

async function stopChrome() {
  try {
    await fetch('http://127.0.0.1:9222/json/close');
  } catch {}
}

module.exports = {
  connectToBrowser,
  getActivePage,
  stopChrome,
};
