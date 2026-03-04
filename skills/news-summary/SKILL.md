---
name: news-summary
description: This skill should be used when the user asks for news updates, daily briefings, or what's happening in the world. Fetches news from trusted international RSS feeds and can create voice summaries.
---

# News Summary

## Overview

Use the bundled script to fetch and parse RSS reliably.  
Do **not** use ad-hoc `curl | python -c` pipelines for this skill.

## Script

Path:
```bash
~/.aevitas/workspace/.claude/skills/news-summary/scripts/fetch_news.py
```

Run:
```bash
python3 ~/.aevitas/workspace/.claude/skills/news-summary/scripts/fetch_news.py --group brief --limit 5
```

JSON output (for downstream summarization logic):
```bash
python3 ~/.aevitas/workspace/.claude/skills/news-summary/scripts/fetch_news.py --group all --limit 4 --json
```

## Feed Sets

- `brief`: `world`, `business`, `tech`
- `all`: `world`, `top`, `business`, `tech`, `reuters`, `npr`, `aljazeera`

You can also select explicit feeds:
```bash
python3 ~/.aevitas/workspace/.claude/skills/news-summary/scripts/fetch_news.py --feeds world,reuters,npr --limit 4
```

## Workflow

1. Run the script (`brief` by default).
2. Build a concise summary from fetched items.
3. Group by topic and highlight major events.
4. If user asks for detail, include source links from script output.

## Best Practices

- Keep to 5-8 top stories unless user asks for more.
- Prefer script output over manual RSS parsing commands.
- Balance perspectives (BBC + Reuters/NPR/Al Jazeera).
- Mention source when content may be controversial.