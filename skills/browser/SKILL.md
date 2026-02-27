---
name: browser
description: Browser automation using Playwright. Use for web scraping, extracting dynamic content, and navigating websites. Always start from the site homepage and navigate naturally to the target page.
---

# Browser Automation

Extract data from websites using a headless Chrome browser.

## ⚠️ Rules

- **Always start from the homepage** — never guess deep URLs directly
- **Use `scrape.cjs` for all data extraction** — it runs nav + eval in one process, no context errors
- **Never use `screenshot.cjs` to read data** — only for visual debugging when JS returns nothing
- **Never chain `nav.cjs` + `eval.cjs` for simple scraping** — use `scrape.cjs` instead
- **If result is empty `[]`** — the selector is wrong, adjust JS; do NOT take a screenshot
- **Always show the source URL to the user** — every scrape.cjs result includes `url` and `title`; always include them in your reply so the user can verify the data source

## Standard Workflow (3 steps)

### Step 1 — Find the entry point from homepage

```bash
node ~/.aevitas/workspace/.claude/skills/browser/scripts/scrape.cjs \
  'https://www.example.com' \
  'Array.from(document.querySelectorAll("a")).filter(a=>a.textContent.trim()).map(a=>({text:a.textContent.trim(),url:a.href})).slice(0,30)'
```

Read the links, find which one leads to your target section.

### Step 2 — Navigate deeper, find the right sub-page

```bash
node ~/.aevitas/workspace/.claude/skills/browser/scripts/scrape.cjs \
  '<url-from-step-1>' \
  'Array.from(document.querySelectorAll("a")).filter(a=>a.textContent.trim()).map(a=>({text:a.textContent.trim(),url:a.href}))'
```

### Step 3 — Extract data from the target page

**First, discover class names** (when selectors are unknown):
```bash
node ~/.aevitas/workspace/.claude/skills/browser/scripts/scrape.cjs \
  '<target-url>' \
  '[...new Set(Array.from(document.querySelectorAll("[class]")).map(el=>el.className.split(" ")[0]))].filter(c=>c.length>2).slice(0,30)'
```

**Then extract with the right selectors:**
```bash
node ~/.aevitas/workspace/.claude/skills/browser/scripts/scrape.cjs \
  '<target-url>' \
  'Array.from(document.querySelectorAll(".item-class")).slice(0,10).map((el,i)=>({ rank: i+1, title: el.querySelector(".title-class")?.textContent?.trim(), value: el.querySelector(".value-class")?.textContent?.trim() }))'
```

**If `data` is still `[]` after trying selectors, fall back to body text — do NOT retry selectors:**
```bash
node ~/.aevitas/workspace/.claude/skills/browser/scripts/scrape.cjs \
  '<target-url>' \
  'document.body.innerText.slice(0, 3000)'
```
Parse the plain text result directly to extract what you need.

**Always include the source URL in your reply:**
Every result contains `url` and `title`. Always show the user where the data came from, e.g.:
> 数据来源：[Page Title](url)

## scrape.cjs Reference

```bash
node ~/.aevitas/workspace/.claude/skills/browser/scripts/scrape.cjs <url> '<javascript>'
```

- JS runs inside the page (use `document.*` freely)
- Must return a JSON-serialisable value
- Output always includes `url` and `title` of the actual page scraped — show these to the user for verification
- Chrome starts automatically if not running; stays running after each call
- Call `stop.cjs` only when completely done

Output format:
```json
{
  "url":   "https://...",
  "title": "Page Title",
  "data":  [ ... ]
}
```

## Other Scripts

```bash
# Only needed for multi-step interactions (click, type, wait between steps)
node ~/.aevitas/workspace/.claude/skills/browser/scripts/start.cjs
node ~/.aevitas/workspace/.claude/skills/browser/scripts/nav.cjs <url>
node ~/.aevitas/workspace/.claude/skills/browser/scripts/eval.cjs '<js>'
node ~/.aevitas/workspace/.claude/skills/browser/scripts/stop.cjs

# Visual debug only — not for reading data
node ~/.aevitas/workspace/.claude/skills/browser/scripts/screenshot.cjs
```

## Useful JS Patterns

```javascript
// Get all links (step 1 & 2)
Array.from(document.querySelectorAll("a"))
  .filter(a => a.textContent.trim())
  .map(a => ({ text: a.textContent.trim(), url: a.href }))

// Discover class names (when page structure is unknown)
[...new Set(Array.from(document.querySelectorAll("[class]"))
  .map(el => el.className.split(" ")[0]))]
  .filter(c => c.length > 2).slice(0, 30)

// Extract a ranked list
Array.from(document.querySelectorAll(".item"))
  .slice(0, 10)
  .map((el, i) => ({
    rank:  i + 1,
    title: el.querySelector(".title")?.textContent?.trim(),
    value: el.querySelector(".score")?.textContent?.trim()
  }))

// Get visible text blocks (when structure is unclear)
Array.from(document.querySelectorAll("p, h2, h3, li"))
  .filter(el => el.offsetHeight > 0 && el.textContent.trim().length > 5)
  .map(el => el.textContent.trim())
  .slice(0, 20)
```
