#!/usr/bin/env node
// Execute JavaScript in active page
const { getActivePage } = require('../browser-manager.js');

const code = process.argv[2];

if (!code) {
  console.error('Usage: eval.cjs <javascript-expression>');
  process.exit(1);
}

async function evaluate() {
  try {
    const page = await getActivePage();
    
    // Ensure page is ready
    await page.waitForLoadState('domcontentloaded', { timeout: 3000 }).catch(() => {});

    // Execute JavaScript
    const result = await page.evaluate(code);
    
    console.log(JSON.stringify(result));
    process.exit(0);
  } catch (error) {
    console.error(JSON.stringify({
      error: 'Evaluation failed',
      message: error.message,
      code: code.substring(0, 100),
    }));
    process.exit(1);
  }
}

evaluate();
