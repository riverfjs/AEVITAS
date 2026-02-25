---
name: flight-monitor
description: Monitor specific flight prices on Trip.com using Playwright. Tracks price changes and reports to Telegram automatically every 6 hours. Use when user asks about flight prices, wants to add/remove a monitored route, check current prices, or trigger an immediate price check.
---

# Flight Monitor

Track specific routes and flight numbers on Trip.com; compare current prices against history and report changes via Telegram.

## ⚠️ Rules — Read Before Anything Else

- **NEVER** use `cat`, `ls`, `grep`, `head`, `mkdir` or any shell command to inspect files under `data/`
- **NEVER** read or parse `monitors.txt`, `price_history.txt`, `prices.json` directly
- **NEVER** create directories manually — the scripts handle their own setup
- Use **only** the commands listed in Implementation below
- After `cron-run`, do NOT also run `monitor.sh` manually or analyze results — gateway delivers to Telegram automatically

## Capabilities

- Add a flight route to monitor (origin, destination, dates, flight times)
- Check current prices immediately — result auto-delivered to Telegram
- List active monitors
- View price history
- Delete a monitor
- Runs automatically every 6 hours via cron

## Implementation

**Scripts location:** `~/.myclaw/workspace/.claude/skills/flight-monitor/scripts/`

**List all monitors:**
```bash
bash ~/.myclaw/workspace/.claude/skills/flight-monitor/scripts/monitor.sh list
```

**Check prices NOW (async → result sent to Telegram):**
```bash
todoist cron-run flight-monitor-auto
```

**View price history:**
```bash
bash ~/.myclaw/workspace/.claude/skills/flight-monitor/scripts/monitor.sh history
```

**Add a monitor** (IATA codes + flight times):
```bash
bash ~/.myclaw/workspace/.claude/skills/flight-monitor/scripts/monitor.sh \
  add SZX CKG 2026-04-03 2026-04-07 13:05 15:30 15:50 18:05
#       from to  depart-date return-date  ob-dep ob-arr ret-dep ret-arr
```

**Delete a monitor:**
```bash
bash ~/.myclaw/workspace/.claude/skills/flight-monitor/scripts/monitor.sh delete <id>
```

## Notes

- Uses Trip.com (not Google Flights); prices in CNY
- Browser lifecycle (Chrome CDP port 9222) managed automatically by monitor.sh — do not start/stop manually
- Cron job `flight-monitor-auto` already registered — do not re-register
