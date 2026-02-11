---
name: skill-creator
description: Create or update skills for myclaw. Use when user asks to create a new skill, add functionality, or when you need to package reusable scripts/workflows as a skill. Triggers on requests like "create a skill for X", "make a new skill", or when repeatedly writing similar code that should be packaged.
allowed-tools:
  - Bash
---

# Skill Creator

Guide for creating effective skills in myclaw.

## Core Principles

1. **Concise** - Only add what agent doesn't know
2. **Self-contained** - Handle own dependencies
3. **Clear triggers** - Description explains when to use

## Skill Structure

```
workspace/.claude/skills/skill-name/
├── SKILL.md (required - metadata + docs)
├── scripts/     (optional - executable code in any language)
├── package.json (optional - npm dependencies)
└── go.mod       (optional - Go dependencies)
```

## Workflow

### 1. Understand Requirements

Ask user:
- What should this skill do?
- When should it trigger?
- What language/tools needed?
- Examples of usage?

### 2. Create in Template Directory

Find myclaw project root, then:

```bash
cd workspace/.claude/skills
mkdir -p skill-name/scripts
cd skill-name
```

### 3. Write SKILL.md

**Frontmatter** (YAML - triggers skill):

```yaml
---
name: skill-name
description: WHAT it does and WHEN to use it. Be specific - this is the ONLY thing agent reads to decide activation. Include triggers, use cases, contexts.
allowed-tools:
  - Bash  # or Python, Node, Go, etc.
---
```

**Body** (Markdown - loaded after activation):

```markdown
# Skill Name

Brief overview.

## Setup

Dependencies with ABSOLUTE paths (use ~/.myclaw/workspace/.claude/skills/skill-name):

\`\`\`bash
# Node.js
npm install --prefix ~/.myclaw/workspace/.claude/skills/skill-name package-name

# Python
pip install -r ~/.myclaw/workspace/.claude/skills/skill-name/requirements.txt

# Go
cd ~/.myclaw/workspace/.claude/skills/skill-name && go build -o bin/tool ./scripts/main.go
\`\`\`

## Usage

All scripts with full paths:

\`\`\`bash
# Node.js
node ~/.myclaw/workspace/.claude/skills/skill-name/scripts/run.cjs <args>

# Python
python3 ~/.myclaw/workspace/.claude/skills/skill-name/scripts/run.py <args>

# Go binary
~/.myclaw/workspace/.claude/skills/skill-name/bin/tool <args>

# Bash
bash ~/.myclaw/workspace/.claude/skills/skill-name/scripts/run.sh <args>
\`\`\`

## Scripts

- `scripts/run.*` - Main entry point
- `scripts/helper.*` - Helper functions

## Key Points

- Important notes
- Error handling
- Limitations
```

### 4. Create Scripts

**Node.js** (`scripts/run.cjs`):

```javascript
#!/usr/bin/env node
console.log(JSON.stringify({ result: 'success' }));
```

**Python** (`scripts/run.py`):

```python
#!/usr/bin/env python3
import json
print(json.dumps({'result': 'success'}))
```

**Go** (`scripts/main.go`):

```go
package main
import ("encoding/json"; "fmt")
func main() {
    fmt.Println(`{"result":"success"}`)
}
```

**Bash** (`scripts/run.sh`):

```bash
#!/bin/bash
echo '{"result":"success"}'
```

Make executable: `chmod +x scripts/*`

### 5. Test Locally

Test in template directory:

```bash
# From myclaw project root
cd workspace/.claude/skills/skill-name

# Test based on language
node scripts/run.cjs
# or
python3 scripts/run.py
# or
go run scripts/main.go
# or
bash scripts/run.sh
```

### 6. Install

Skills in `workspace/.claude/skills/` are templates. Install to make available:

```bash
# From myclaw project root
./myclaw skills install skill-name
```

Agent detects it on next run. **No restart needed!**

## Path Convention

❌ **Wrong**: `~/.claude/skills/my-skill/`  
✅ **Correct**: `~/.myclaw/workspace/.claude/skills/my-skill/`

Always use `~/.myclaw/workspace/.claude/skills/` prefix in docs.

## Behavior Rules

When skill tools fail repeatedly (2-3 times):

1. **STOP** - Don't auto-switch to other methods
2. **Ask User** - Use `AskUserQuestion` to present options
3. **Explain** - What failed and why
4. **Never Assume** - Always get confirmation to switch approaches

**Example**: Browser nav fails 3x:
```
浏览器工具无法访问该页面（失败3次）。
可能原因：需要登录 / 反爬虫 / 网络问题

是否改用 WebSearch？
```

## Commands

```bash
# List installed
./myclaw skills list

# Install (skip existing)
./myclaw skills install skill-name

# Update (overwrite)
./myclaw skills update skill-name

# Remove
./myclaw skills uninstall skill-name

# Verify
./myclaw skills verify
```

## Example

See `browser` skill for complete CDP automation example with Node.js scripts and npm dependencies.



