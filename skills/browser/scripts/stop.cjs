#!/usr/bin/env node
// Stop Chrome
const { stopChrome } = require('../browser-manager.js');
const { exec } = require('child_process');

async function stop() {
  try {
    await stopChrome();
    
    // Force kill if needed
    const platform = process.platform;
    let killCmd;

    if (platform === 'darwin') {
      killCmd = 'pkill -9 -f "remote-debugging-port=9222"';
    } else if (platform === 'linux') {
      killCmd = 'pkill -9 -f "remote-debugging-port=9222"';
    } else if (platform === 'win32') {
      killCmd = 'taskkill /F /IM chrome.exe /T';
    }

    if (killCmd) {
      exec(killCmd, () => {});
    }

    console.log('Browser closed');
    process.exit(0);
  } catch (error) {
    console.error('Error:', error.message);
    process.exit(1);
  }
}

stop();
