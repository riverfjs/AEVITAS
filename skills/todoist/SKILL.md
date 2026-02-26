---
name: todoist
description: Task management with due-date reminders and cron job scheduling. Use when user wants to add/list/complete tasks, set reminders, or manage recurring scheduled jobs (flight monitor, price checks, etc). Cron operations talk directly to myclaw gateway via WebSocket RPC.
---

# Todoist

Manage tasks with reminders and schedule recurring/one-shot jobs through myclaw's cron engine.

## Capabilities

- Add tasks with optional due dates
- Reminder via cron 2 hours before due (delivered to Telegram)
- Create recurring cron jobs that run shell commands and deliver output to Telegram
- Trigger any cron job immediately on demand
- List, delete, and manage all cron jobs

## Implementation

**Binary:** `~/.myclaw/workspace/.claude/skills/todoist/bin/todoist`

### First run
```bash
bash ~/.myclaw/workspace/.claude/skills/todoist/scripts/bootstrap.sh
```

### Standard command entry
```bash
TODOIST=~/.myclaw/workspace/.claude/skills/todoist/bin/todoist
$TODOIST <subcommand> ...
```

**Config:** `~/.myclaw/workspace/.claude/skills/todoist/config.json`
```json
{ "channel": "telegram", "chat_id": "<user-chat-id>" }
```

### Task commands
```bash
$TODOIST add "description"
$TODOIST add "description" --due 2026-04-01
$TODOIST list
$TODOIST complete <id>
$TODOIST delete <id>
$TODOIST reminders          # show overdue tasks
```

### Cron commands (→ gateway ws://127.0.0.1:18790 via WS RPC)
```bash
$TODOIST cron-list                              # list all cron jobs and get job id
$TODOIST cron-run <job-id>                      # trigger immediately
$TODOIST cron-add "<name>" "<shell cmd>" <ms>   # add recurring job
$TODOIST cron-delete <job-id>                   # delete a job
```

**Common intervals:**
```
1h  = 3600000    6h = 21600000    12h = 43200000    24h = 86400000
```

> `cron-run` is **async** — returns immediately after triggering. The result is delivered to Telegram by the gateway automatically. Do not run extra diagnostic commands unless the user explicitly asks.

## Source layout

```
scripts/
├── main.go      # CLI entry point, command dispatch
├── todo.go      # Task / TodoList types and persistence
├── cron.go      # CronJob types and CronManager (RPC calls)
└── gateway.go   # WebSocket RPC client (callGateway)
```

## Rules

- Always use `todoist` commands to manage cron jobs — never read or write `jobs.json` directly, never use `ls`/`cat` to inspect files
- `todoist cron-list` is the only correct way to list jobs
- `todoist cron-run` accepts **job id only** (from `cron-list`), not job name
- After `cron-run`, do not chain extra checks unless the user explicitly asks

## Notes

- Gateway must be running for cron commands to work
- `chat_id` in config enables Telegram delivery of cron results
- Payload `kind: "command"` runs shell directly; `kind: "agentTurn"` runs agent with a prompt
- Schedule kinds: `"every"` (interval), `"at"` (one-shot unix ms), `"cron"` (cron expr)
