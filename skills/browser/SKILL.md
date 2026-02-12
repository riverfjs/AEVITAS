---
name: browser
description: Browser automation using Playwright. Use for web scraping, screenshots, and extracting dynamic content from websites.
---

# Browser Automation

Control Chrome browser for web scraping and automation tasks.

## Capabilities

- Navigate to websites and extract content
- Execute JavaScript on pages
- Capture screenshots
- Handle dynamic content that loads with JavaScript

## Implementation

**Start browser:**
```bash
node ~/.myclaw/workspace/.claude/skills/browser/scripts/start.cjs
```

**Navigate:**
```bash
node ~/.myclaw/workspace/.claude/skills/browser/scripts/nav.cjs <url>
node ~/.myclaw/workspace/.claude/skills/browser/scripts/nav.cjs <url> --new  # new tab
```

**Execute JavaScript:**
```bash
node ~/.myclaw/workspace/.claude/skills/browser/scripts/eval.cjs '<javascript>'
```

**Screenshot:**
```bash
node ~/.myclaw/workspace/.claude/skills/browser/scripts/screenshot.cjs
node ~/.myclaw/workspace/.claude/skills/browser/scripts/screenshot.cjs --full
```

**Stop browser:**
```bash
node ~/.myclaw/workspace/.claude/skills/browser/scripts/stop.cjs
```

## Notes

- Chrome runs in background; start once, use multiple times
- Always stop when done to free resources
- Screenshots saved to system temp directory (`os.tmpdir()`)
- Requires Node.js and Chrome installed

**Content Extraction Tips:**

Don't assume fixed selectors. Use flexible JS patterns:

```javascript
// ❌ Bad: Assumes specific tags exist
document.querySelectorAll('article h2')

// ✅ Good: Try multiple selectors, find what exists
const headings = document.querySelectorAll('h1, h2, h3, [role="heading"]');
const articles = document.querySelectorAll('article, [role="article"], .post, .entry');

// ✅ Find largest text blocks (likely main content)
Array.from(document.querySelectorAll('*'))
  .filter(el => el.children.length < 3 && el.textContent.length > 200)
  .map(el => el.textContent.trim())

// ✅ Find links by pattern, not by location
Array.from(document.querySelectorAll('a'))
  .filter(a => a.textContent.trim().length > 10)
  .map(a => ({title: a.textContent.trim(), url: a.href}))

// ✅ Get visible text only (ignore hidden elements)
Array.from(document.querySelectorAll('p, div'))
  .filter(el => el.offsetHeight > 0)
  .map(el => el.textContent.trim())
  .filter(text => text.length > 50)
```

Always check what exists first before assuming structure.
