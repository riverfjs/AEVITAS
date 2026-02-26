'use strict';
/**
 * check.cjs — Scheduled price-check runner for flight-monitor (3 modes).
 *
 * Supported modes:
 * 1) roundtrip_locked:
 *    - outbound + return flights are fixed
 *    - monitor the exact round-trip total for that pair
 * 2) outbound_day:
 *    - outbound flight not fixed
 *    - monitor the minimum outbound price of that day
 * 3) return_after_outbound:
 *    - outbound is fixed, return not fixed
 *    - monitor the best round-trip total among return options
 *
 * Legacy records without mode are auto-migrated to return_after_outbound.
 */

const { execFileSync } = require('child_process');
const fs               = require('fs');
const os               = require('os');
const path             = require('path');

const SKILL_DIR  = path.join(os.homedir(), '.myclaw/workspace/.claude/skills');
const SEARCH_CJS = path.join(SKILL_DIR, 'flight-search/scripts/search.cjs');
const DATA_FILE  = path.join(__dirname, '../data/monitors.json');

const { notify } = require(path.join(__dirname, 'notify.cjs'));

function toNumber(v, fallback = 0) {
  const n = Number(v);
  return Number.isFinite(n) ? n : fallback;
}

function readJson(file, fallback) {
  try {
    return JSON.parse(fs.readFileSync(file, 'utf8'));
  } catch {
    return fallback;
  }
}

function addDays(dateStr, days) {
  const d = new Date(`${dateStr}T00:00:00`);
  if (Number.isNaN(d.getTime())) return dateStr;
  d.setDate(d.getDate() + days);
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

function runSearch({ depart, arrive, departDate, returnDate, outboundFlight = '', outboundPrice = 0 }) {
  const args = [SEARCH_CJS, depart, arrive, departDate, returnDate];
  if (outboundFlight) {
    args.push(outboundFlight, String(toNumber(outboundPrice, 0)));
  }

  const raw = execFileSync('node', args, { encoding: 'utf8', timeout: 30_000 });
  const data = JSON.parse(raw);
  if (!Array.isArray(data.flights)) data.flights = [];
  return data;
}

function normalizeLegacyMonitor(m) {
  if (m.mode) return false;

  if (m.flight && m.refPrice != null) {
    m.mode = 'return_after_outbound';
    m.outboundFlight = m.flight;
    m.outboundPrice = toNumber(m.refPrice, 0);
    m.lastObservedBestTotal = m.lastPrice != null ? toNumber(m.lastPrice, null) : null;
    m.lastObservedBestReturnPrice = null;
    m.lastObservedBestReturnFlight = null;
    m.lastObservedBestReturnDep = null;
    m.lastObservedBestReturnArr = null;
    m.status = m.status || 'enabled';
    delete m.flight;
    delete m.refPrice;
    delete m.lastPrice;
    return true;
  }

  m.mode = 'outbound_day';
  m.status = m.status || 'enabled';
  return true;
}

function formatDelta(delta) {
  if (delta > 0) return `上涨 ¥${delta}`;
  if (delta < 0) return `下降 ¥${Math.abs(delta)}`;
  return '无变化';
}

function formatOutboundDayReport(m, tripType, flights, best, previousMin, currentMin) {
  const lines = [];
  lines.push(`✈️ 去程：${m.depart} -> ${m.arrive}  ${m.departDate}（共 ${flights.length} 班）`);
  lines.push(`模式：outbound_day/${tripType}`);
  lines.push('```');
  lines.push('序号 | 航班号 | 起飞->到达 | 价格');
  lines.push('-----+--------+-----------+------');
  flights.forEach((f, idx) => {
    const no = String(idx + 1).padStart(2, ' ');
    const flight = String(f.flight || '-').padEnd(6, ' ');
    const time = `${f.dep || '--:--'}->${f.arr || '--:--'}`.padEnd(11, ' ');
    lines.push(`${no}   | ${flight} | ${time} | ¥${f.price}`);
  });
  lines.push('```');
  if (previousMin == null) {
    lines.push(`最低价：¥${currentMin}（首次记录）`);
  } else {
    lines.push(`最低价：上次 ¥${previousMin} -> 当前 ¥${currentMin}（${formatDelta(currentMin - previousMin)}）`);
  }
  lines.push(`当前最低航班：${best.flight}  ${best.dep || '--:--'}->${best.arr || '--:--'}`);
  return lines.join('\n');
}

function formatRoundtripLockedReport(m, match, previousTotal, currentTotal) {
  const lines = [];
  lines.push(`✈️ 往返锁定：${m.depart} -> ${m.arrive}  ${m.departDate}/${m.returnDate}`);
  lines.push('```');
  lines.push(`去程航班 | ${m.outboundFlight}`);
  lines.push(`返程航班 | ${m.returnFlight}`);
  lines.push(`基准总价 | ¥${toNumber(m.baselineTotalPrice, 0)}`);
  if (previousTotal == null) {
    lines.push(`当前总价 | ¥${currentTotal}（首次记录）`);
  } else {
    lines.push(`当前总价 | ¥${currentTotal}`);
    lines.push(`价格变化 | ${formatDelta(currentTotal - previousTotal)}（上次 ¥${previousTotal}）`);
  }
  lines.push(`返程时刻 | ${match.dep || '--:--'}->${match.arr || '--:--'}`);
  lines.push('```');
  return lines.join('\n');
}

function formatReturnWatchReport(m, flights, best, outboundPrice, previousTotal, bestTotal, bestReturnPrice) {
  const lines = [];
  lines.push(`✈️ 返程优选：${m.depart} -> ${m.arrive}  ${m.departDate}/${m.returnDate}`);
  lines.push(`已定去程：${m.outboundFlight}  ¥${outboundPrice}`);
  if (previousTotal == null) {
    lines.push(`当前最优总价：¥${bestTotal}（首次记录）`);
  } else {
    lines.push(`最优总价变化：上次 ¥${previousTotal} -> 当前 ¥${bestTotal}（${formatDelta(bestTotal - previousTotal)}）`);
  }
  lines.push(`当前最优返程：${best.flight}  ${best.dep || '--:--'}->${best.arr || '--:--'}  +¥${bestReturnPrice}`);
  lines.push('');
  lines.push('```');
  lines.push('序号 | 返程航班 | 起飞->到达 | 往返总价 | 返程增量');
  lines.push('-----+----------+-----------+----------+---------');
  flights.forEach((f, idx) => {
    const no = String(idx + 1).padStart(2, ' ');
    const flight = String(f.flight || '-').padEnd(8, ' ');
    const time = `${f.dep || '--:--'}->${f.arr || '--:--'}`.padEnd(11, ' ');
    const total = `¥${toNumber(f.price, 0)}`.padEnd(8, ' ');
    const inc = `+¥${toNumber(f.extra, toNumber(f.price, 0) - outboundPrice)}`;
    lines.push(`${no}   | ${flight} | ${time} | ${total} | ${inc}`);
  });
  lines.push('```');
  return lines.join('\n');
}

async function maybeNotifyChange(title, previous, current, detailLines) {
  if (previous == null || previous === current) return false;
  const delta = current - previous;
  const msg = [
    title,
    ...detailLines,
    `上次 ¥${previous} -> 现在 ¥${current}（${formatDelta(delta)}）`,
  ].join('\n');
  await notify(msg);
  return true;
}

async function checkRoundtripLocked(m) {
  const data = runSearch({
    depart: m.depart,
    arrive: m.arrive,
    departDate: m.departDate,
    returnDate: m.returnDate,
    outboundFlight: m.outboundFlight,
    outboundPrice: m.baselineTotalPrice,
  });

  const match = data.flights.find(f => f.flight === m.returnFlight);
  if (!match) {
    m.lastChecked = Date.now();
    return { dirty: true, report: `✈️ 往返锁定：${m.depart} -> ${m.arrive}  ${m.departDate}/${m.returnDate}\n未找到返程航班：${m.returnFlight}` };
  }

  const currentTotal = toNumber(match.price, 0);
  const previousTotal = m.lastObservedTotalPrice != null ? toNumber(m.lastObservedTotalPrice, null) : null;

  m.lastObservedTotalPrice = currentTotal;
  m.lastObservedReturnDep = match.dep || null;
  m.lastObservedReturnArr = match.arr || null;
  m.lastChecked = Date.now();
  const report = formatRoundtripLockedReport(m, match, previousTotal, currentTotal);

  await maybeNotifyChange(
    '✈️ 往返锁定航班价格变化',
    previousTotal,
    currentTotal,
    [
      `${m.depart}->${m.arrive}  ${m.departDate}/${m.returnDate}`,
      `去程 ${m.outboundFlight} | 返程 ${m.returnFlight}`,
      `基准总价 ¥${toNumber(m.baselineTotalPrice, 0)}`,
    ],
  );

  return { dirty: true, report };
}

async function checkOutboundDay(m) {
  const tripType = m.tripType || (m.returnDate ? 'roundtrip_context' : 'oneway');
  const queryReturnDate = tripType === 'oneway'
    ? (m.returnDate || addDays(m.departDate, 1))
    : m.returnDate;

  const data = runSearch({
    depart: m.depart,
    arrive: m.arrive,
    departDate: m.departDate,
    returnDate: queryReturnDate,
  });

  if (!data.flights.length) {
    m.lastChecked = Date.now();
    return { dirty: true, report: `✈️ 去程：${m.depart} -> ${m.arrive}  ${m.departDate}\n未查询到可用航班。` };
  }

  const best = data.flights.reduce((min, f) => (f.price < min.price ? f : min), data.flights[0]);
  const currentMin = toNumber(best.price, 0);
  const previousMin = m.lastObservedMinPrice != null ? toNumber(m.lastObservedMinPrice, null) : null;
  const report = formatOutboundDayReport(m, tripType, data.flights, best, previousMin, currentMin);

  m.lastObservedMinPrice = currentMin;
  m.lastObservedFlight = best.flight || null;
  m.lastObservedDep = best.dep || null;
  m.lastObservedArr = best.arr || null;
  m.tripType = tripType;
  m.lastChecked = Date.now();

  await maybeNotifyChange(
    '✈️ 去程整天最低价变化',
    previousMin,
    currentMin,
    [
      `${m.depart}->${m.arrive}  ${m.departDate}`,
      `模式: ${tripType}`,
      `当前最低: ${best.flight} ${best.dep || '--:--'}->${best.arr || '--:--'}`,
    ],
  );

  return { dirty: true, report };
}

async function checkReturnAfterOutbound(m) {
  const outboundPrice = toNumber(m.outboundPrice, 0);
  const data = runSearch({
    depart: m.depart,
    arrive: m.arrive,
    departDate: m.departDate,
    returnDate: m.returnDate,
    outboundFlight: m.outboundFlight,
    outboundPrice,
  });

  if (!data.flights.length) {
    m.lastChecked = Date.now();
    return { dirty: true, report: `✈️ 返程优选：${m.depart} -> ${m.arrive}  ${m.departDate}/${m.returnDate}\n未查询到返程选项。` };
  }

  const best = data.flights.reduce((min, f) => (f.price < min.price ? f : min), data.flights[0]);
  const bestTotal = toNumber(best.price, 0);
  const bestReturnPrice = toNumber(best.extra, bestTotal - outboundPrice);
  const previousTotal = m.lastObservedBestTotal != null ? toNumber(m.lastObservedBestTotal, null) : null;
  const report = formatReturnWatchReport(m, data.flights, best, outboundPrice, previousTotal, bestTotal, bestReturnPrice);

  m.lastObservedBestTotal = bestTotal;
  m.lastObservedBestReturnPrice = bestReturnPrice;
  m.lastObservedBestReturnFlight = best.flight || null;
  m.lastObservedBestReturnDep = best.dep || null;
  m.lastObservedBestReturnArr = best.arr || null;
  m.lastChecked = Date.now();

  await maybeNotifyChange(
    '✈️ 固定去程下返程最优总价变化',
    previousTotal,
    bestTotal,
    [
      `${m.depart}->${m.arrive}  ${m.departDate}/${m.returnDate}`,
      `已定去程: ${m.outboundFlight} (¥${outboundPrice})`,
      `当前最优返程: ${best.flight} ${best.dep || '--:--'}->${best.arr || '--:--'} (+¥${bestReturnPrice})`,
    ],
  );

  return { dirty: true, report };
}

async function checkOne(m) {
  const mode = m.mode;
  if (mode === 'roundtrip_locked') return checkRoundtripLocked(m);
  if (mode === 'outbound_day') return checkOutboundDay(m);
  if (mode === 'return_after_outbound') return checkReturnAfterOutbound(m);
  console.log(`  [${m.id}] 未知模式 ${mode}，已跳过`);
  return { dirty: false };
}

async function main() {
  if (!fs.existsSync(DATA_FILE)) {
    console.log('暂无监控任务');
    return;
  }

  const monitors = readJson(DATA_FILE, []);
  if (!Array.isArray(monitors) || !monitors.length) {
    console.log('暂无监控任务');
    return;
  }

  let dirty = false;
  const reports = [];

  for (const m of monitors) {
    if (normalizeLegacyMonitor(m)) dirty = true;
    if ((m.status || 'enabled') !== 'enabled') {
      console.log(`  [${m.id}] 已禁用，跳过`);
      continue;
    }

    try {
      const result = await checkOne(m);
      if (result.dirty) dirty = true;
      if (result.report) reports.push(result.report);
    } catch (err) {
      m.lastChecked = Date.now();
      dirty = true;
      reports.push(`✈️ ${m.id}\n查询失败：${err.message}`);
    }
  }

  if (dirty) {
    fs.writeFileSync(DATA_FILE, JSON.stringify(monitors, null, 2));
  }

  if (reports.length) {
    console.log(reports.join('\n\n'));
  }
}

main().catch(err => {
  console.error('check.cjs error:', err.message);
  process.exit(1);
});
