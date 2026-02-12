---
name: todo
description: Manage your todo tasks and reminders. Use when user wants to track tasks, create reminders, check deadlines, or manage their to-do list.
allowed-tools:
  - Bash
---

# Todo Management

Help you track tasks, set reminders, and manage your to-do list.

## Capabilities

- View your task list
- Add new tasks with optional due dates
- Mark tasks as complete
- Delete tasks
- Get reminders for upcoming or overdue tasks

## Implementation

**Binary path:**
```
~/.myclaw/workspace/.claude/skills/todo/bin/todo
```

**Commands:**
```bash
# List all tasks
~/.myclaw/workspace/.claude/skills/todo/bin/todo list

# Add task
~/.myclaw/workspace/.claude/skills/todo/bin/todo add "description"
~/.myclaw/workspace/.claude/skills/todo/bin/todo add "description" --due 2026-02-15

# Complete task
~/.myclaw/workspace/.claude/skills/todo/bin/todo complete <id>

# Delete task
~/.myclaw/workspace/.claude/skills/todo/bin/todo delete <id>

# Check reminders
~/.myclaw/workspace/.claude/skills/todo/bin/todo reminders
```

## Notes

- Tasks stored in `~/.myclaw/workspace/.claude/skills/todo/data/tasks.json`
- Persistent across sessions
- First-time setup: `cd ~/.myclaw/workspace/.claude/skills/todo && go build -o bin/todo ./scripts/`
