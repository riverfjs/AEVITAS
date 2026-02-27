// Browser automation configuration
module.exports = {
  // CDP connection settings
  cdp: {
    url: 'http://127.0.0.1:9222',
    port: 9222,
    timeout: 5000,
  },

  // Playwright settings
  playwright: {
    // Navigation timeout (ms)
    navigationTimeout: 30000,
    
    // Page load wait strategy: 'load' | 'domcontentloaded' | 'networkidle'
    waitUntil: 'domcontentloaded',
    
    // Timeout for JavaScript evaluation (ms)
    evaluateTimeout: 10000,
    
    // Screenshot settings
    screenshot: {
      quality: 90,
      type: 'png',
    },
  },

  // Chrome launch settings
  chrome: {
    // Profile directory for persistent sessions
    profileDir: '~/.aevitas/browser-profile',
    
    // Additional Chrome flags
    args: [
      '--disable-blink-features=AutomationControlled',
      '--disable-dev-shm-usage',
    ],
  },

  // Advanced features
  features: {
    // Enable console message capture
    captureConsole: false,
    
    // Enable network request tracking
    trackNetwork: false,
    
    // Auto-wait for dynamic content (ms)
    networkIdleTimeout: 5000,
  },
};

