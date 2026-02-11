---
name: browser
description: Advanced browser automation using Playwright framework. Use for web scraping, dynamic content extraction, automated testing, screenshot capture, and complex browser interactions. Handles modern web apps with JavaScript rendering, Shadow DOM, and async content loading.
---

# Browser Automation (Playwright)

Professional-grade browser automation powered by Playwright framework. Provides robust waiting mechanisms, automatic retries, and comprehensive error handling for reliable web automation tasks.

## Features

✅ **Smart Waiting** - Automatically waits for page loads, network idle, and DOM updates
✅ **Dynamic Content** - Handles JavaScript-rendered content, lazy loading, and SPAs  
✅ **Robust Error Handling** - Built-in timeouts, retries, and detailed error messages
✅ **Modern APIs** - Playwright-core for stable, cross-browser automation
✅ **Persistent Sessions** - Save cookies, localStorage, and auth state with `--profile` flag

## Installation

First-time setup:

```bash
npm install --prefix ~/.myclaw/workspace/.claude/skills/browser
```

**架构说明：** 使用Chrome + CDP (Chrome DevTools Protocol) + Playwright连接。需要系统已安装Google Chrome。浏览器在后台持续运行，所有脚本通过CDP连接到同一个浏览器实例，因此页面状态在多次调用间保持。

## Usage Guide

### 1. Start Chrome (一次性启动)

```bash
# Start headless Chrome with CDP on port 9222
node ~/.myclaw/workspace/.claude/skills/browser/scripts/start.cjs
```

**重要：** Chrome会在后台持续运行，只需启动一次。所有后续操作都连接到这个运行中的浏览器实例，页面状态会保持。

### 2. Navigate to URL

```bash
# Navigate current tab
node ~/.myclaw/workspace/.claude/skills/browser/scripts/nav.cjs https://example.com

# Open in new tab
node ~/.myclaw/workspace/.claude/skills/browser/scripts/nav.cjs https://example.com --new
```

**Output:**
```json
{
  "action": "navigated",
  "url": "https://example.com",
  "title": "Example Domain",
  "newTab": false
}
```

### 3. Execute JavaScript

```bash
# Simple expression
node ~/.myclaw/workspace/.claude/skills/browser/scripts/eval.cjs 'document.title'

# Complex queries with waiting
node ~/.myclaw/workspace/.claude/skills/browser/scripts/eval.cjs '
  Array.from(document.querySelectorAll("h1, h2, h3"))
    .map(el => ({ tag: el.tagName, text: el.textContent.trim() }))
    .filter(item => item.text.length > 0)
'

# IIFE for multiple statements
node ~/.myclaw/workspace/.claude/skills/browser/scripts/eval.cjs '
  (() => {
    const headlines = [];
    document.querySelectorAll("article").forEach(article => {
      const title = article.querySelector("h2")?.textContent;
      const link = article.querySelector("a")?.href;
      if (title && link) headlines.push({ title, link });
    });
    return headlines;
  })()
'
```

**Key Improvements over CDP:**
- Automatically waits for page load before execution
- Better error messages with context
- Handles async JavaScript properly

### 4. Capture Screenshots

```bash
# Viewport screenshot
node ~/.myclaw/workspace/.claude/skills/browser/scripts/screenshot.cjs

# Full page screenshot (entire scrollable area)
node ~/.myclaw/workspace/.claude/skills/browser/scripts/screenshot.cjs --full

# Screenshot specific element
node ~/.myclaw/workspace/.claude/skills/browser/scripts/screenshot.cjs --selector='.main-content'
```

**Output:**
```json
{
  "path": "/tmp/screenshot-1234567890.png",
  "filename": "screenshot-1234567890.png",
  "fullPage": false,
  "selector": null
}
```

### 5. Stop Chrome

```bash
node ~/.myclaw/workspace/.claude/skills/browser/scripts/stop.cjs
```

**Always stop Chrome after completing tasks** to free system resources.

## Best Practices

### For News/Article Scraping

```bash
# 1. Launch with profile (preserves region/language settings)
node scripts/start.cjs --profile

# 2. Navigate and wait for content
node scripts/nav.cjs https://news.google.com/

# 3. Wait a moment for dynamic content to load
sleep 2

# 4. Extract with smart selectors
node scripts/eval.cjs '
  (() => {
    const articles = [];
    // Try multiple selectors for robustness
    const elements = document.querySelectorAll("article, [role=article], .article");
    
    elements.forEach(el => {
      const heading = el.querySelector("h1, h2, h3, h4");
      const link = el.querySelector("a");
      
      if (heading && link) {
        articles.push({
          title: heading.textContent.trim(),
          url: link.href,
          snippet: el.textContent.substring(0, 200).trim()
        });
      }
    });
    
    return articles.slice(0, 10); // Top 10
  })()
'

# 5. Clean up
node scripts/stop.cjs
```

### For Complex Websites

```bash
# Use screenshots to debug selectors
node scripts/screenshot.cjs

# Check what's actually rendered
node scripts/eval.cjs 'document.body.innerText.substring(0, 500)'

# Check page structure
node scripts/eval.cjs '
  ({
    title: document.title,
    url: window.location.href,
    articles: document.querySelectorAll("article").length,
    h1Count: document.querySelectorAll("h1").length,
    h2Count: document.querySelectorAll("h2").length
  })
'
```

## Troubleshooting

### Page Content Not Found

**Problem:** `eval.cjs` returns empty arrays

**Solutions:**
1. Add delay after navigation: `sleep 3`
2. Check if content is in iframe/shadow DOM
3. Take screenshot to see actual page state
4. Try different selectors (inspect with browser DevTools)

### Navigation Timeout

**Problem:** Page takes too long to load

**Solution:** Script already uses `domcontentloaded` instead of full `load` - this is optimal for most sites.

### Chrome Not Starting

**Problem:** `start.cjs` fails

**Checks:**
1. Chrome installed? (Google Chrome required)
2. Port 9222 already in use? Run `stop.cjs` first
3. Check: `lsof -i:9222` (macOS/Linux)

## Configuration

Advanced settings in `browser.config.js`:

```javascript
{
  playwright: {
    navigationTimeout: 30000,      // Page load timeout
    evaluateTimeout: 10000,         // JS execution timeout
    waitUntil: 'domcontentloaded', // When to consider nav complete
  }
}
```

## Architecture

```
Browser Skill (Playwright)
├── browser-manager.js      # Connection pool & state management
├── scripts/
│   ├── start.cjs          # Launch Chrome with CDP
│   ├── nav.cjs            # Navigate with smart waiting
│   ├── eval.cjs           # Execute JS with error handling
│   ├── screenshot.cjs     # Capture page/element screenshots
│   └── stop.cjs           # Graceful shutdown
└── browser.config.js      # Configuration settings
```

## Workflow Example

```bash
# 1. Start Chrome once
node scripts/start.cjs

# 2. Navigate to news site
node scripts/nav.cjs https://news.ycombinator.com/

# 3. Extract headlines (page is still loaded!)
node scripts/eval.cjs '
  Array.from(document.querySelectorAll(".titleline > a"))
    .slice(0, 10)
    .map(a => ({ title: a.textContent, url: a.href }))
'

# 4. Take screenshot
node scripts/screenshot.cjs

# 5. Navigate to another site
node scripts/nav.cjs https://www.bbc.com/news

# 6. Extract content
node scripts/eval.cjs '
  Array.from(document.querySelectorAll("h2, h3"))
    .slice(0, 5)
    .map(h => h.textContent.trim())
'

# 7. Stop Chrome when done
node scripts/stop.cjs
```

**关键优势：** 浏览器持续运行在后台，页面状态保持，多次脚本调用共享同一个浏览器会话。

## Comparison with Previous Version

| Feature | Old (CDP only) | New (Playwright) |
|---------|---------------|------------------|
| Waiting | Manual sleep | Automatic smart wait |
| Errors | Basic | Detailed with context |
| Dynamic Content | ❌ Fails | ✅ Handles properly |
| Timeouts | None | Configurable |
| Dependencies | 60KB (ws) | ~50MB (playwright-core) |

## Key Improvements

1. **Automatic Waiting** - No more manual sleep; waits for DOM + network idle
2. **Better Errors** - Clear messages about what failed and why
3. **Robust Selection** - Handles modern web frameworks (React, Vue, etc.)
4. **State Management** - Proper browser/context/page lifecycle
5. **Production-Ready** - Used by thousands of projects via Playwright

---

**Migration Note:** This version replaces the simple CDP WebSocket implementation with professional Playwright framework. All scripts maintain backward compatibility with the same command-line interface.
