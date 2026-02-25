---
name: skill-creator
description: Create or update skills for myclaw. Use when user asks to create a new skill, add functionality, or when you need to package reusable scripts/workflows as a skill. Triggers on requests like "create a skill for X", "make a new skill", or when repeatedly writing similar code that should be packaged.
allowed-tools:
  - Write
  - Edit
  - Read
  - Grep
  - Glob
  - Bash
  - Task/TaskCreate/TaskGet/TaskList/TaskUpdate
---

# Skill Creator

Guide for creating effective skills in myclaw.

## Core Principles

1. **One file, one responsibility** — split by function, not by language
2. **Self-contained** — handle own dependencies
3. **Clear triggers** — description explains when to use
4. **English only** — all SKILL.md files must be written in English

## Skill Structure

```
workspace/.claude/skills/skill-name/
├── SKILL.md          required — metadata + docs (English)
├── bin/              compiled binaries (Go skills)
├── scripts/          executable source files
│   ├── action-a.cjs  one action per file (Node.js)
│   ├── action-b.cjs
│   ├── main.go       CLI entry point only (Go)
│   ├── feature-x.go  one domain per file (Go)
│   └── feature-y.go
└── data/             runtime data / state
```

## File Splitting Rule

**One file = one concern.** Split by what the script does, not by language.

### Node.js example (browser skill)
```
scripts/
├── start.cjs       # start browser
├── stop.cjs        # stop browser
├── nav.cjs         # navigate to URL
├── eval.cjs        # run JavaScript in page
└── screenshot.cjs  # capture screenshot
```

### Go example (todoist skill)
```
scripts/
├── main.go         # CLI entry point, command dispatch only
├── todo.go         # Task / TodoList types + persistence
├── cron.go         # CronJob types + CronManager
└── gateway.go      # WebSocket RPC client
```

### Shell example
```
scripts/
├── check.sh        # check one thing
├── monitor.sh      # orchestration (calls check.sh)
└── notify.sh       # send notification
```

Never dump everything into a single `main.go` / `index.js` / `script.sh`.

## SKILL.md Template

```markdown
---
name: skill-name
description: WHAT it does and WHEN to use it. Be specific.
---

# Skill Name

One-sentence description.

## Capabilities

- What users can ask for, in plain language
- Another capability

## Implementation

**Entry point:**
` ` `bash
~/.myclaw/workspace/.claude/skills/skill-name/bin/tool
` ` `

**Commands (grouped by concern):**
` ` `bash
# Group A
tool action-a <arg>

# Group B
tool action-b <arg>
` ` `

**Source layout:**
` ` `
scripts/
├── main.go    # what it does
└── feature.go # what it does
` ` `

## Rules

- Explicit constraints the agent must follow when using this skill
- What NOT to do

## Notes

- Requirements, limitations, edge cases
```

**Language rules for SKILL.md:**
- Capabilities: plain English, user-facing ("I can manage your tasks")
- Implementation: exact commands the agent needs to run
- Rules: hard constraints — what agent must and must not do

## Creating a Skill

### 1. Understand requirements
- What does it do? When should the agent use it?

### 2. Plan file layout
Map each distinct action or domain to its own file before writing any code.

### 3. Write scripts
- Each file does exactly one thing
- Use absolute paths (`$HOME/...`)
- Make scripts executable: `chmod +x scripts/*.sh`

### 4. Write SKILL.md (in English)
- Frontmatter: `name` + `description`
- Capabilities: plain English
- Implementation: exact commands grouped by concern
- Rules: explicit constraints on agent behavior
- Source layout: brief file table

### 5. Test & install
```bash
node scripts/start.cjs   # Node.js
bash scripts/check.sh    # Shell
go build -o bin/tool ./scripts/  # Go

./myclaw skills install skill-name
```

## Build instructions by language

**Node.js:** no build step; run directly with `node`

**Go:**
```bash
cd skill-dir
go mod init skill-name   # if no go.mod
go get <dep>@latest
go build -o bin/tool ./scripts/
```

**Shell:** `chmod +x` then run directly

## Commands

```bash
./myclaw skills list           # list installed
./myclaw skills install <name> # install skill
./myclaw skills update <name>  # update skill
```
