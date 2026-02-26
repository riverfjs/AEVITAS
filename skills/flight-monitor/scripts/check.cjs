'use strict';
/**
 * check.cjs — Scheduled price-check runner for flight-monitor.
 *
 * For each entry in monitors.json:
 *   1. Calls flight-search/scripts/search.cjs (ly.com SSR, no browser)
 *   2. Finds the monitored flight number in the results
 *   3. Compares current price against the reference price
 *   4. Sends a Telegram notification via notify.cjs if price dropped
 *   5. Updates lastPrice / lastChecked in monitors.json
 *
 * Run directly:  node check.cjs
 * Or via cron:   todoist cron-add "flight-monitor-<id>" "node .../check.cjs" <ms>
 */

const { execSync } = require('child_process');
const fs           = require('fs');
const os           = require('os');
const path         = require('path');

const SKILL_DIR  = path.join(os.homedir(), '.myclaw/workspace/.claude/skills');
const SEARCH_CJS = path.join(SKILL_DIR, 'flight-search/scripts/search.cjs');
const DATA_FILE  = path.join(__dirname, '../data/monitors.json');

const { notify } = require(path.join(__dirname, 'notify.cjs'));

async function main() {
  if (!fs.existsSync(DATA_FILE)) {
    console.log('暂无监控任务');
    return;
  }

  const monitors = JSON.parse(fs.readFileSync(DATA_FILE, 'utf8'));
  if (!monitors.length) {
    console.log('暂无监控任务');
    return;
  }

  console.log(`开始检查 [${new Date().toLocaleString('zh-CN')}]  共 ${monitors.length} 条`);

  let dirty = false;

  for (const m of monitors) {
    try {
      const raw = execSync(
        `node "${SEARCH_CJS}" ${m.depart} ${m.arrive} ${m.departDate} ${m.returnDate || ''}`,
        { timeout: 30_000, encoding: 'utf8' }
      );

      const data  = JSON.parse(raw);
      const match = (data.flights || []).find(f => f.flight === m.flight);

      if (!match) {
        console.log(`  [${m.flight}] 未在今日航班列表中找到（航班可能已停飞或日期已过）`);
        continue;
      }

      const current = match.price;
      const ref     = m.refPrice;
      console.log(`  [${m.flight}] ${m.depart}→${m.arrive} ${m.departDate}  当前¥${current}  参考¥${ref}`);

      m.lastPrice   = current;
      m.lastChecked = Date.now();
      dirty = true;

      if (current < ref) {
        const diff = ref - current;
        const msg  =
          `✈️ 机票降价！\n` +
          `${m.depart}→${m.arrive}  ${m.departDate}\n` +
          `${m.flight}\n` +
          `参考价 ¥${ref} → 现价 ¥${current}（降 ¥${diff}）`;
        await notify(msg);
        console.log(`  已发送降价通知 -¥${diff}`);
      }
    } catch (err) {
      console.log(`  [${m.flight}] 查询失败: ${err.message}`);
    }
  }

  if (dirty) {
    fs.writeFileSync(DATA_FILE, JSON.stringify(monitors, null, 2));
  }

  console.log('检查完成');
}

main().catch(err => {
  console.error('check.cjs error:', err.message);
  process.exit(1);
});
