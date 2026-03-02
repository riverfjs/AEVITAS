#!/usr/bin/env node
'use strict';

const { execFileSync } = require('child_process');
const fs = require('fs');
const os = require('os');
const path = require('path');

const SKILL_DIR = path.join(os.homedir(), '.aevitas/workspace/.claude/skills');
const SEARCH_CJS = path.join(SKILL_DIR, 'flight-search/scripts/search.cjs');
const DATA_FILE = path.join(__dirname, '../data/monitors.json');

const { notify } = require(path.join(__dirname, 'notify.cjs'));

function toNumber(v, fallback = 0) {
  const n = Number(v);
  return Number.isFinite(n) ? n : fallback;
}

function readJson(file, fallback = []) {
  try {
    return JSON.parse(fs.readFileSync(file, 'utf8'));
  } catch {
    return fallback;
  }
}

function parseArgs(argv) {
  const out = { id: '' };
  for (let i = 0; i < argv.length; i += 1) {
    if (argv[i] === '--id') out.id = argv[i + 1] || '';
  }
  return out;
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

function runSearch(mode, ...modeArgs) {
  const raw = execFileSync('node', [SEARCH_CJS, mode, ...modeArgs], { encoding: 'utf8', timeout: 30_000 });
  const data = JSON.parse(raw);
  if (!Array.isArray(data.flights)) data.flights = [];
  return data;
}

function markChecked(m) {
  m.lastChecked = Date.now();
}

function priceAmount(f) {
  if (f && typeof f.price === 'object') return toNumber(f.price.amount, 0);
  return 0;
}

function legInfo(f) {
  return (f && (f.transferInfo || f.stopoverInfo)) || '';
}

function renderLegTime(f) {
  if (f.depDateTime || f.arrDateTime) return `${f.depDateTime || '--'} -> ${f.arrDateTime || '--'}`;
  return `${f.dep || '--:--'}->${f.arr || '--:--'}`;
}

function renderTransfer(f) {
  if (f.routeType === 'transfer') return `${toNumber(f.transferCount, 1)}次中转`;
  if (f.routeType === 'stopover') return `${toNumber(f.stopoverCount, 1)}次经停`;
  if (f.routeType === 'direct') return '直飞';
  return toNumber(f.stops, 0) > 0 ? `${toNumber(f.stops, 0)}次停留` : '直飞';
}

function pickCheapest(flights) {
  return flights.reduce((min, f) => (priceAmount(f) < priceAmount(min) ? f : min), flights[0]);
}

function formatDelta(delta) {
  if (delta > 0) return `上涨 ¥${delta}`;
  if (delta < 0) return `下降 ¥${Math.abs(delta)}`;
  return '无变化';
}

function buildReport(title, viewTable, tailLines = []) {
  const lines = [title];
  if (viewTable) lines.push(viewTable);
  lines.push(...tailLines.filter(Boolean));
  return lines.join('\n');
}

function noFlightReport(title, message) {
  return `${title}\n${message}`;
}

function syncLegSnapshot(m, f, prefix) {
  m[`${prefix}Flight`] = f.flight || null;
  m[`${prefix}Dep`] = f.depDateTime || f.dep || null;
  m[`${prefix}Arr`] = f.arrDateTime || f.arr || null;
}

async function maybeNotifyChange(title, previous, current, details) {
  if (previous == null || previous === current) return false;
  await notify([
    title,
    ...details,
    `上次 ¥${previous} -> 现在 ¥${current}（${formatDelta(current - previous)}）`,
  ].join('\n'));
  return true;
}

function modeOfOutbound(m) {
  const cfg = m.config || {};
  const route = m.route || {};
  return cfg.tripType || (route.returnDate ? 'roundtrip_context' : 'oneway');
}

const MODE_HANDLERS = {
  roundtrip_locked: {
    searchMode: 'roundtrip_locked',
    searchArgs: m => [m.route.depart, m.route.arrive, m.route.departDate, m.route.returnDate, m.config.outboundFlight],
    reportTitle: m => `✈️ 往返锁定：${m.route.depart} -> ${m.route.arrive}  ${m.route.departDate}/${m.route.returnDate}`,
    emptyMessage: '未查询到返程选项。',
    selectTarget: (flights, m) => flights.find(f => f.flight === m.config.returnFlight),
    missingTargetMessage: m => `未找到返程航班：${m.config.returnFlight}`,
    currentValue: target => priceAmount(target),
    previousValue: m => (m.observed?.lastTotalPrice != null ? toNumber(m.observed.lastTotalPrice, null) : null),
    reportTail: ({ m, target, current, previous }) => [
      `去程航班：${m.config.outboundFlight}  返程航班：${m.config.returnFlight}`,
      `基准总价：¥${toNumber(m.config.baselineTotalPrice, 0)}`,
      previous == null
        ? `当前总价：¥${current}（首次记录）`
        : `当前总价：¥${current}（上次 ¥${previous}，${formatDelta(current - previous)}）`,
      legInfo(target) ? `停留信息：${legInfo(target)}` : '',
    ],
    persist: ({ m, target, current }) => {
      m.observed = m.observed || {};
      m.observed.lastTotalPrice = current;
      m.observed.lastReturnDep = target.depDateTime || target.dep || null;
      m.observed.lastReturnArr = target.arrDateTime || target.arr || null;
    },
    notifyTitle: '✈️ 往返锁定航班价格变化',
    notifyDetail: ({ m }) => [
      `${m.route.depart}->${m.route.arrive}  ${m.route.departDate}/${m.route.returnDate}`,
      `去程 ${m.config.outboundFlight} | 返程 ${m.config.returnFlight}`,
      `基准总价 ¥${toNumber(m.config.baselineTotalPrice, 0)}`,
    ],
  },
  outbound_day: {
    searchMode: 'outbound_day',
    searchArgs: m => [m.route.depart, m.route.arrive, m.route.departDate, modeOfOutbound(m), m.route.returnDate || addDays(m.route.departDate, 1)],
    reportTitle: m => `✈️ 去程：${m.route.depart} -> ${m.route.arrive}  ${m.route.departDate}`,
    emptyMessage: '未查询到可用航班。',
    selectTarget: flights => pickCheapest(flights),
    currentValue: target => priceAmount(target),
    previousValue: m => (m.observed?.lastMinPrice != null ? toNumber(m.observed.lastMinPrice, null) : null),
    reportTail: ({ m, target, current, previous }) => [
      previous == null
        ? `最低价：¥${current}（首次记录）`
        : `最低价：上次 ¥${previous} -> 当前 ¥${current}（${formatDelta(current - previous)}）`,
      `当前最低航班：${target.flight}  ${renderLegTime(target)} [${renderTransfer(target)}]`,
      legInfo(target) ? `停留信息：${legInfo(target)}` : '',
      `模式: ${modeOfOutbound(m)}`,
    ],
    persist: ({ m, target, current }) => {
      m.observed = m.observed || {};
      m.observed.lastMinPrice = current;
      m.observed.lastFlight = target.flight || null;
      m.observed.lastDep = target.depDateTime || target.dep || null;
      m.observed.lastArr = target.arrDateTime || target.arr || null;
      m.config = m.config || {};
      m.config.tripType = modeOfOutbound(m);
    },
    notifyTitle: '✈️ 去程整天最低价变化',
    notifyDetail: ({ m, target }) => [
      `${m.route.depart}->${m.route.arrive}  ${m.route.departDate}`,
      `模式: ${modeOfOutbound(m)}`,
      `当前最低: ${target.flight} ${renderLegTime(target)} [${renderTransfer(target)}]`,
    ],
  },
  return_after_outbound: {
    searchMode: 'return_after_outbound',
    searchArgs: m => [m.route.depart, m.route.arrive, m.route.departDate, m.route.returnDate, m.config.outboundFlight, String(toNumber(m.config.outboundPrice, 0))],
    reportTitle: m => `✈️ 返程优选：${m.route.depart} -> ${m.route.arrive}  ${m.route.departDate}/${m.route.returnDate}`,
    emptyMessage: '未查询到返程选项。',
    selectTarget: flights => pickCheapest(flights),
    currentValue: target => priceAmount(target),
    previousValue: m => (m.observed?.lastBestTotal != null ? toNumber(m.observed.lastBestTotal, null) : null),
    reportTail: ({ m, target, current, previous }) => {
      const outboundPrice = toNumber(m.config.outboundPrice, 0);
      const bestReturnPrice = toNumber(target.extra, current - outboundPrice);
      return [
        `已定去程：${m.config.outboundFlight}  ¥${outboundPrice}`,
        previous == null
          ? `当前最优总价：¥${current}（首次记录）`
          : `最优总价：上次 ¥${previous} -> 当前 ¥${current}（${formatDelta(current - previous)}）`,
        `当前最优返程：${target.flight}  ${renderLegTime(target)} [${renderTransfer(target)}] +¥${bestReturnPrice}`,
        legInfo(target) ? `停留信息：${legInfo(target)}` : '',
      ];
    },
    persist: ({ m, target, current }) => {
      const outboundPrice = toNumber(m.config.outboundPrice, 0);
      m.observed = m.observed || {};
      m.observed.lastBestTotal = current;
      m.observed.lastBestReturnPrice = toNumber(target.extra, current - outboundPrice);
      m.observed.lastBestReturnFlight = target.flight || null;
      m.observed.lastBestReturnDep = target.depDateTime || target.dep || null;
      m.observed.lastBestReturnArr = target.arrDateTime || target.arr || null;
    },
    notifyTitle: '✈️ 固定去程下返程最优总价变化',
    notifyDetail: ({ m, target, current }) => {
      const outboundPrice = toNumber(m.config.outboundPrice, 0);
      const bestReturnPrice = toNumber(target.extra, current - outboundPrice);
      return [
        `${m.route.depart}->${m.route.arrive}  ${m.route.departDate}/${m.route.returnDate}`,
        `已定去程: ${m.config.outboundFlight} (¥${outboundPrice})`,
        `当前最优返程: ${target.flight} ${renderLegTime(target)} [${renderTransfer(target)}] (+¥${bestReturnPrice})`,
      ];
    },
  },
};

async function checkOne(m) {
  const cfg = MODE_HANDLERS[m.mode];
  if (!cfg) {
    markChecked(m);
    return { dirty: true, report: `✈️ ${m.id || 'unknown'}\n配置错误：不支持的模式 ${m.mode || '(empty)'}` };
  }

  const data = runSearch(cfg.searchMode, ...cfg.searchArgs(m));
  if (!data.flights.length) {
    markChecked(m);
    return { dirty: true, report: noFlightReport(cfg.reportTitle(m), cfg.emptyMessage) };
  }

  const target = cfg.selectTarget(data.flights, m);
  if (!target) {
    markChecked(m);
    return { dirty: true, report: noFlightReport(cfg.reportTitle(m), cfg.missingTargetMessage ? cfg.missingTargetMessage(m) : '未找到目标航班。') };
  }

  const current = cfg.currentValue(target, m);
  const previous = cfg.previousValue(m);
  const report = buildReport(cfg.reportTitle(m), data?.view?.table || '', cfg.reportTail({ m, target, current, previous }));

  cfg.persist({ m, target, current });
  markChecked(m);
  await maybeNotifyChange(cfg.notifyTitle, previous, current, cfg.notifyDetail({ m, target, current }));
  return { dirty: true, report };
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  if (!fs.existsSync(DATA_FILE)) {
    console.log('暂无监控任务');
    return;
  }

  const monitors = readJson(DATA_FILE, []);
  if (!Array.isArray(monitors) || !monitors.length) {
    console.log('暂无监控任务');
    return;
  }

  let target = null;
  if (args.id) {
    target = monitors.find(m => m.id === args.id);
    if (!target) {
      console.log(`monitor not found: ${args.id}`);
      process.exit(1);
    }
  }

  let dirty = false;
  const reports = [];

  const candidates = target ? [target] : monitors.filter(m => (m.status || 'enabled') === 'enabled' && !!m.cronJobId);
  for (const m of candidates) {
    try {
      const result = await checkOne(m);
      if (result.dirty) dirty = true;
      if (result.report) reports.push(result.report);
    } catch (err) {
      markChecked(m);
      dirty = true;
      reports.push(`✈️ ${m.id || 'unknown'}\n查询失败：${err.message}`);
    }
  }

  if (dirty) fs.writeFileSync(DATA_FILE, JSON.stringify(monitors, null, 2));
  if (reports.length) console.log(reports.join('\n\n'));
}

main().catch(err => {
  console.error('check.cjs error:', err.message);
  process.exit(1);
});
