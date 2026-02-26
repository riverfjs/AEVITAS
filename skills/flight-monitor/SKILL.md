---
name: flight-monitor
description: Set up recurring price monitoring for a specific flight. Guides the user through flight-search first to pick a flight, then schedules automatic price-drop alerts.
---

# Flight Monitor

Track a specific flight's price via ly.com and send a Telegram notification when it drops below the reference price.

## Setup Flow

### Step 1 — Search first

If the user hasn't selected a flight yet, invoke the `flight-search` skill first so they can browse options and pick one.

Once the user has chosen an outbound flight, you will have:
- Route: depart / arrive IATA codes
- Dates: departure date, return date
- Flight number (e.g. `CZ3455`)
- Current price (reference price, e.g. `2071`)

### Step 2 — Add the monitor

```bash
bash ~/.myclaw/workspace/.claude/skills/flight-monitor/scripts/monitor.sh \
  add SZX CKG 2026-04-03 2026-04-07 CZ3455 2071
#       from to  depart     return     flight  refPrice
```

### Step 3 — Schedule

Ask the user how often to check (e.g. every 6 h, 12 h, 24 h). Then create a cron job:

```bash
# 6 h = 21600000 ms | 12 h = 43200000 ms | 24 h = 86400000 ms
~/.myclaw/workspace/.claude/skills/todoist/bin/todoist cron-add \
  "flight-monitor-<id>" \
  "node ~/.myclaw/workspace/.claude/skills/flight-monitor/scripts/check.cjs" \
  21600000
```

Confirm: "已设置，每 X 小时检查一次，价格低于参考价 ¥Y 时会发 Telegram 通知。"

## Management

**List monitors:**
```bash
bash ~/.myclaw/workspace/.claude/skills/flight-monitor/scripts/monitor.sh list
```

**Check prices now:**
```bash
node ~/.myclaw/workspace/.claude/skills/flight-monitor/scripts/check.cjs
```

**Delete a monitor:**
```bash
bash ~/.myclaw/workspace/.claude/skills/flight-monitor/scripts/monitor.sh delete <id>
```

**Delete the cron job** (when monitor is deleted):
```bash
~/.myclaw/workspace/.claude/skills/todoist/bin/todoist cron-delete flight-monitor-<id>
```

## Rules

- Always run `flight-search` first — never ask the user to manually type flight details
- Reference price = the price seen in `flight-search` at the time of setup
- Notification is sent only when current price **drops below** the reference price
- Do not hardcode a check interval — ask the user
