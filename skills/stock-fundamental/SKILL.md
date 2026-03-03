---
name: stock-fundamental
description: Fetch and analyze stock fundamental data from Yahoo Finance. Use when user asks about a stock's price, valuation, earnings, revenue, analyst estimates, or wants a fundamental analysis report. Supports US (AAPL, CLF), HK (0700.HK), A-shares (600519.SS).
---

# Stock Fundamental Analysis

## ⚠️ Rules — Read First

- **Use ONLY the one command below. Nothing else.**
- `fetch.cjs` starts and stops Chrome automatically — do NOT call start.cjs or stop.cjs separately
- **NEVER** use `screenshot.cjs`, `nav.cjs`, `eval.cjs` directly
- **NEVER** use `WebSearch` or `WebFetch` to get stock data
- Do NOT set short per-call timeout (e.g. `timeout: 60`) for this command.
- Default to one-shot full fetch: `fetch.cjs <SYMBOL> all`.
- If fetch fails, retry the SAME `all` command from scratch (full retry), do not split into quote/analysis/financials sub-calls.
- If full retry still fails, report the final error directly.

## Command

```bash
node ~/.aevitas/workspace/.claude/skills/stock-fundamental/scripts/fetch.cjs <SYMBOL> [quote|financials|balance|cashflow|keystats|analysis|all]
```

Symbol format:
- US stocks: `AAPL`, `CLF`, `TSLA`
- HK stocks: `0700.HK`, `9988.HK`
- A-shares Shanghai: `600519.SS`
- A-shares Shenzhen: `000858.SZ`

Mode (default `all`, recommended):
- `quote` — price + key stats only
- `financials` — income statement (4 years)
- `balance` — balance sheet (debt, cash, equity)
- `cashflow` — cash flow (FCF, operating, capex)
- `keystats` — valuation multiples (PB, PS, EV/EBITDA, ROE, debt ratios, short interest)
- `analysis` — analyst estimates + EPS trend + revision direction
- `all` — everything

Examples:
```bash
node ~/.aevitas/workspace/.claude/skills/stock-fundamental/scripts/fetch.cjs CLF
node ~/.aevitas/workspace/.claude/skills/stock-fundamental/scripts/fetch.cjs AAPL quote
node ~/.aevitas/workspace/.claude/skills/stock-fundamental/scripts/fetch.cjs 0700.HK all
# on failure, retry the same full command once:
node ~/.aevitas/workspace/.claude/skills/stock-fundamental/scripts/fetch.cjs 0700.HK all
```

## Output Format (always use this exact structure)

After running the command, analyze the JSON and reply using this format:

---

## {SYMBOL}（{Company Name}）基本面分析

**数据来源：** Yahoo Finance | **更新时间：** {date}

---

### 📊 一、价格与估值

```text
| 指标 | 数值 |
|------|------|
| 当前价格 | ${price}（盘前 ${preMarket}）|
| 今日涨跌 | {change}（{changePct}）|
| 52周区间 | ${52wLow} – ${52wHigh}，现价处于 **{pct}%** 位置 |
| 50日均线 | ${ma50}（现价在均线{上方/下方} {diff}%）|
| 200日均线 | ${ma200} |
| 市值 | {marketCap} |
| 企业价值（EV）| {ev} |
| P/B | {pb} |
| P/S | {ps} |
| EV/EBITDA | {evEbitda} |
| 分析师目标价 | ${target}（**{upside}% 上行空间**）|
```
---

### 💰 二、收入与盈利趋势

```text
| 年份 | 营收 | 毛利润 | 营业利润 | 净利润 | EPS |
|------|------|--------|--------|--------|-----|
| FY{y-3} | ... | ... | ... | ... | ... |
| FY{y-2} | ... | ... | ... | ... | ... |
| FY{y-1} | ... | ... | ... | ... | ... |
| FY{y}   | ... | ... | ... | ... | ... |
```

**趋势：** {1-2句概括，营收/利润方向}

---

### 🏦 三、资产负债（最新季度）

```text
| 指标 | 最新 | 上年 |
|------|------|------|
| 总资产 | ... | ... |
| 总负债 | ... | ... |
| 股东权益 | ... | ... |
| 总债务 | ... | ... |
| 净债务 | ... | ... |
| 现金余额 | ... | ... |
| 债务/股权比 | ... | — |
```

---

### 💸 四、现金流

```text
| 指标 | FY{y} | FY{y-1} | FY{y-2} | FY{y-3} |
|------|--------|--------|--------|--------|
| 经营现金流 | ... | ... | ... | ... |
| 资本支出 | ... | ... | ... | ... |
| 自由现金流 | ... | ... | ... | ... |
```

---

### 📈 五、分析师预期

**EPS预期：**

```text
| 期间 | 当前预期 | 30天前 | 方向 |
|------|--------|--------|------|
| 当前季 | ... | ... | ↑/↓ |
| 下季   | ... | ... | ↑/↓ |
| 当年   | ... | ... | ↑/↓ |
| 明年   | ... | ... | ↑/↓ |
```

**成长预期 vs S&P500：**

```text
| | 当前季 | 下季 | 当年 | 明年 |
|--|--------|------|------|------|
| {SYMBOL} | ...% | ...% | ...% | ...% |
| S&P 500  | ...% | ...% | ...% | ...% |
```

---

### 📋 六、近期财报超/未达预期

```text
| 季度 | EPS预期 | EPS实际 | 超预期 |
|------|--------|--------|--------|
| Q{n} | ... | ... | ✅/❌ {pct}% |
| Q{n-1} | ... | ... | ✅/❌ {pct}% |
| Q{n-2} | ... | ... | ✅/❌ {pct}% |
| Q{n-3} | ... | ... | ✅/❌ {pct}% |
```

---

### ⚠️ 七、风险指标

```text
| 指标 | 数值 | 评级 |
|------|------|------|
| Beta | ... | 🟢/🟡/🔴 |
| 做空比率 | ...% | 🟢/🟡/🔴 |
| 净债务/市值 | ...% | 🟢/🟡/🔴 |
| 年利息支出 | ... | 🟢/🟡/🔴 |
| ROE(TTM) | ...% | 🟢/🟡/🔴 |
| 现金余额 | ... | 🟢/🟡/🔴 |
```

风险评级：🟢 低 | 🟡 中 | 🔴 高

---

### 📝 八、综合结论

**多头逻辑：**
- ...
- ...

**空头逻辑：**
- ...
- ...

**关键日期：** 📅 下次财报 **{earnings date}**
