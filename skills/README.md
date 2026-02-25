# Skills

Pluggable skill scripts for AEVITAS. Each skill is a self-contained directory with a `SKILL.md` (agent instructions) and a `scripts/` folder.

## Available Skills

### todoist
**Task management + cron scheduling via WebSocket RPC**

- Add/list/complete/delete tasks with optional due dates
- Reminder cron 2 hours before task due date (delivered to Telegram)
- Create recurring cron jobs (shell commands, interval-based)
- Trigger any cron job immediately on demand
- All cron operations talk to myclaw gateway via `ws://127.0.0.1:18790`

```bash
todoist add "buy groceries" --due 2026-04-01
todoist list
todoist complete 1
todoist cron-add "daily-report" "bash ~/report.sh" 86400000
todoist cron-list
todoist cron-run flight-monitor-auto
todoist cron-delete <job-id>
```

---

### flight-monitor
**Flight price tracking on Trip.com via Playwright**

- Monitor specific routes and flight numbers
- Compares price against history, records lowest price
- Auto-checks every 6 hours via cron, reports to Telegram
- Trigger immediate check via `todoist cron-run flight-monitor-auto`

```bash
bash monitor.sh list                          # List monitored routes
bash monitor.sh add SZX CKG 2026-04-03 2026-04-07 13:05 15:30 15:50 18:05
bash monitor.sh check <id>                    # Manual single check
bash monitor.sh history                       # Price history
bash monitor.sh remove <id>
```

---

### browser
**Browser automation via Playwright (headless Chrome)**

- Navigate pages, extract content, run JavaScript
- Capture screenshots
- Handles dynamic / JS-rendered content

```bash
node scripts/start.cjs          # Start browser
node scripts/nav.cjs <url>      # Navigate
node scripts/eval.cjs <js>      # Evaluate JS
node scripts/screenshot.cjs     # Screenshot
node scripts/stop.cjs           # Stop browser
```

---

### skill-creator
**Guide for creating new skills**

Not a runnable skill — provides instructions and templates for adding new skills to AEVITAS. Read `SKILL.md` before creating any new skill.

---

## Skill Structure

```
skills/
  <name>/
    SKILL.md        # Agent instructions (name, description, rules, commands)
    scripts/        # Executable scripts (Go, Node.js, Bash)
    data/           # Runtime data (gitignored where sensitive)
    go.mod          # (Go skills only)
    config.json     # (if needed)
```

## Adding a New Skill

Use the `skill-creator` skill — it provides step-by-step guidance for structuring, documenting, and registering a new skill.
