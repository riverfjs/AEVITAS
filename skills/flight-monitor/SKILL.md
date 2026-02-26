---
name: flight-monitor
description: Manage recurring flight price monitoring in three modes (locked round-trip, outbound-day, return-watch). flight-search is only used to fetch flight options; monitor.sh + check.cjs own monitoring state and notifications.
---

# Flight Monitor

This skill is the monitoring orchestrator.
It stores monitor configs, performs scheduled checks, compares price changes, and pushes notifications.
`flight-search` is a fetch tool only; it does not own monitor state.

## Monitoring Modes

1. `roundtrip_locked`
   - Outbound and return are both fixed.
   - Target: total price of that exact pair only.

2. `outbound_day`
   - Outbound is not fixed.
   - Target: minimum outbound price for that route/date.
   - Note: `returnDate` is query context only, not a locked return monitor target.

3. `return_after_outbound`
   - Outbound is fixed, return is not fixed.
   - Target: best round-trip total across return options.

## Decision Flow

Ask two questions first:
- Is outbound flight fixed?
- Is return flight fixed?

Map answers to mode:
- both fixed -> `roundtrip_locked`
- outbound not fixed -> `outbound_day`
- outbound fixed, return not fixed -> `return_after_outbound`

Use `flight-search` first whenever needed to fetch current options/prices, then create the monitor in this skill.

## Commands

### 1) Locked round-trip

```bash
bash ~/.myclaw/workspace/.claude/skills/flight-monitor/scripts/monitor.sh \
  add-roundtrip SZX CKG 2026-04-03 2026-04-07 CZ3455 CZ2335 4142
#                from to  depart     return     out     ret     baseline_total
```

### 2) Outbound day-level (no fixed outbound flight)

```bash
bash ~/.myclaw/workspace/.claude/skills/flight-monitor/scripts/monitor.sh \
  add-outbound-day SZX CKG 2026-04-03
```

Optional round-trip query context:
```bash
bash ~/.myclaw/workspace/.claude/skills/flight-monitor/scripts/monitor.sh \
  add-outbound-day SZX CKG 2026-04-03 2026-04-07
```

### 3) Outbound fixed, monitor best return

```bash
bash ~/.myclaw/workspace/.claude/skills/flight-monitor/scripts/monitor.sh \
  add-return-watch SZX CKG 2026-04-03 2026-04-07 CZ3455 2071
#                   from to  depart     return     out     outbound_price
```

## Management

```bash
bash ~/.myclaw/workspace/.claude/skills/flight-monitor/scripts/monitor.sh list
```

```bash
node ~/.myclaw/workspace/.claude/skills/flight-monitor/scripts/check.cjs
```

```bash
bash ~/.myclaw/workspace/.claude/skills/flight-monitor/scripts/monitor.sh delete <id>
```

Create schedule (ask user interval first):

```bash
# Example: every 6h
~/.myclaw/workspace/.claude/skills/todoist/bin/todoist cron-add \
  "flight-monitor-<id>" \
  "node ~/.myclaw/workspace/.claude/skills/flight-monitor/scripts/check.cjs" \
  21600000
```

Delete schedule when monitor is removed:

```bash
# 1) Find the cron job id first
~/.myclaw/workspace/.claude/skills/todoist/bin/todoist cron-list

# 2) Delete by job id (NOT by name)
~/.myclaw/workspace/.claude/skills/todoist/bin/todoist cron-delete <job-id>
```

## Rules

- `flight-search` fetches data; `flight-monitor` owns monitoring lifecycle.
- Do not force a specific mode before asking user confirmation status.
- Do not hardcode check interval; ask user (6h / 12h / 24h etc.).
- Each scheduled run sends execution result to Telegram via gateway delivery.
- If current check cannot find a target flight, keep monitor and retry next run.
