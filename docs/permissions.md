# Permission System

`myclaw` uses agentsdk-go's permission system to control tool access via `.claude/settings.json`.

## Configuration Location

```
~/.myclaw/workspace/.claude/settings.json
```

## Format

```json
{
  "permissions": {
    "allow": [],   // Auto-allow these tools
    "ask": [],     // Require confirmation
    "deny": []     // Block these tools
  }
}
```

## Rule Syntax

**Basic format:** `ToolName(pattern)` or `ToolName`

### Examples

#### Bash Tool
```json
{
  "allow": [
    "Bash(ls:*)",           // Allow all ls commands
    "Bash(git add:*)",      // Allow all git add commands
    "Bash(git commit:*)",   // Allow all git commit commands
    "Bash(pwd)",            // Allow pwd only
    "Bash(*)"               // Allow all bash commands (NOT recommended)
  ],
  "deny": [
    "Bash(rm:*)",           // Block all rm commands (already blocked by Validator)
    "Bash(sudo:*)"          // Block all sudo commands
  ]
}
```

#### File Tools
```json
{
  "allow": [
    "Read(*.go)",           // Allow reading .go files
    "Write(logs/*)",        // Allow writing to logs directory
    "Edit(src/**/*.ts)"     // Allow editing TypeScript files in src
  ],
  "deny": [
    "Read(.env)",           // Block reading .env
    "Write(config.json)"    // Block writing to config.json
  ]
}
```

#### Other Tools
```json
{
  "allow": [
    "Skill(codex)",         // Allow codex skill
    "WebFetch(domain:docs.anthropic.com)", // Allow fetching from specific domain
    "Glob(**/*.md)"         // Allow globbing markdown files
  ]
}
```

#### No Pattern
```json
{
  "allow": [
    "AskUserQuestion",      // Allow without parameter restriction
    "TaskList"
  ]
}
```

## Permission Modes

### `allow`
- Tools execute **immediately** without confirmation
- Use for safe, frequently-used operations
- Examples: `ls`, `pwd`, `git status`

### `ask` (Future Support)
- Tool execution requires **user confirmation**
- Use for potentially dangerous operations
- Currently not fully implemented in myclaw

### `deny`
- Tools are **blocked** completely
- Use for dangerous or unauthorized operations
- Examples: `rm -rf`, `.env` access

## Priority

When rules conflict:
1. **deny** has highest priority (always blocks)
2. **allow** next (auto-allows if not denied)
3. **ask** next (requires confirmation if not denied/allowed)
4. Default: block (unless sandbox is disabled)

## Pattern Matching

### Wildcards
- `*` - matches any characters except `/`
- `**` - matches any characters including `/`

### Examples
```
"Read(*.go)"        → matches "main.go", NOT "src/main.go"
"Read(**/*.go)"     → matches "main.go", "src/main.go", "a/b/c/test.go"
"Bash(git *)"       → matches "git status", "git add .", NOT "git-lfs"
"Bash(git:*)"       → matches "git" with any arguments
```

## Security Notes

1. **Sandbox + Validator**: Permissions work **in addition to** the built-in Validator
   - Even if you `allow` `Bash(rm -rf:*)`, the Validator will **still block** it
   - This is by design for defense-in-depth

2. **Least Privilege**: Only allow what you need
   - ❌ `"Bash(*)"`
   - ✅ `"Bash(git add:*)"`, `"Bash(git commit:*)"`

3. **Review Regularly**: Audit your `.claude/settings.json` periodically

4. **Sandbox Disabled**: With `"sandbox": {"enabled": false}`, permissions are **ignored**
   - Only Validator rules apply
   - This is the current myclaw default

## Recommended Starter Config

```json
{
  "permissions": {
    "allow": [
      "Read(**/*.md)",
      "Write(logs/*)",
      "Bash(ls:*)",
      "Bash(pwd)",
      "Bash(git status:*)",
      "Bash(git log:*)",
      "Bash(git diff:*)"
    ],
    "deny": [
      "Read(.env)",
      "Read(secrets/**)",
      "Write(config.json)"
    ],
    "ask": []
  },
  "sandbox": {
    "enabled": false
  }
}
```

## See Also

- [SDK Security Guide](../pkg/agentsdk-go/docs/security.md)
- [Validator Rules](../pkg/agentsdk-go/pkg/security/validator.go)

