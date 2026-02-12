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

1. **Concise** - Only add what agent doesn't know
2. **Self-contained** - Handle own dependencies
3. **Clear triggers** - Description explains when to use
4. **User-friendly** - Explain capabilities in natural language, not command syntax

## Skill Structure

```
workspace/.claude/skills/skill-name/
├── SKILL.md (required - metadata + brief docs)
└── scripts/ (optional - executable code)
```

## Creating a Skill

### 1. Ask User Requirements

- What should this skill do?
- When should it trigger?

### 2. Write SKILL.md

**Frontmatter** (YAML):
```yaml
---
name: skill-name
description: WHAT it does and WHEN to use it. Be specific.
---
```

**Body** (Keep brief):
```markdown
# Skill Name

What this skill does (1-2 sentences).

## Capabilities (User-facing - natural language)

- Capability 1 in plain language
- Capability 2 in plain language
- Capability 3 in plain language

## Implementation (Agent-only - technical details)

Internal commands, paths, or implementation notes that agent needs but users don't.

## Notes

Any important limitations or requirements.
```

**User-Facing Language:**
- ✅ Good: "I can manage your tasks and reminders"
- ❌ Bad: "`todo list`, `todo add`, `todo complete`"
- ❌ Bad: "`~/.myclaw/workspace/.claude/skills/todo/bin/todo`"

**Document Structure:**
- **Capabilities section** - For users, use natural language
- **Implementation section** - For agent, technical details and commands
- Users ask naturally, agent reads Implementation to know how to execute

### 3. Create Scripts (if needed)

Use absolute paths for reliability:
- `~/.myclaw/workspace/.claude/skills/skill-name/scripts/script.sh`

Make executable: `chmod +x scripts/*`

### 4. Test & Install

```bash
# Test in workspace/.claude/skills/skill-name
bash scripts/script.sh

# Install from myclaw root
./myclaw skills install skill-name
```

## Tools

- **Write** - Create new files
- **Edit** - Modify existing files
- **Read** - Read file contents
- **Grep** - Search in files
- **Glob** - Find files by pattern
- **Bash** - Execute commands, test scripts
- **Task** - Manage long-running tasks
- **NEVER** use `echo >` or `cat >` - bypass permissions

## Error Handling

If skill tools fail 2-3 times:
1. STOP - don't auto-switch methods
2. Ask user via `AskUserQuestion`
3. Explain what failed and why

## Commands

```bash
./myclaw skills list           # List installed
./myclaw skills install <name> # Install skill
./myclaw skills update <name>  # Update skill
```
