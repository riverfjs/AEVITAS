#!/usr/bin/env node
'use strict';

const os = require('os');
const path = require('path');
const { execFileSync } = require('child_process');
const { chromium } = require(path.join(os.homedir(), '.aevitas/workspace/.claude/skills/browser/node_modules/playwright'));

const CDP_URL = 'http://127.0.0.1:9222';
const BROWSER_START = path.join(os.homedir(), '.aevitas/workspace/.claude/skills/browser/scripts/start.cjs');

const MAINLAND_CITY = {
  SZX: '深圳', CKG: '重庆', PEK: '北京', PKX: '北京',
  SHA: '上海', PVG: '上海', CAN: '广州', CTU: '成都',
  XIY: '西安', WUH: '武汉', CSX: '长沙', KMG: '昆明',
  NKG: '南京', HGH: '杭州', XMN: '厦门', TSN: '天津',
  DLC: '大连', TAO: '青岛', SHE: '沈阳', HRB: '哈尔滨',
  URC: '乌鲁木齐', KWE: '贵阳', NNG: '南宁', HAK: '海口',
  SYX: '三亚', LHW: '兰州', TNA: '济南',
};
const INTL_CITY = { HKG: '中国香港', MFM: '中国澳门', TPE: '中国台北' };
const MAINLAND_IATA = new Set(Object.keys(MAINLAND_CITY));

function addDays(dateStr, days) {
  const d = new Date(`${dateStr}T00:00:00`);
  if (Number.isNaN(d.getTime())) return dateStr;
  d.setDate(d.getDate() + days);
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

function isMainland(code) {
  return MAINLAND_IATA.has((code || '').toUpperCase());
}

function isIntlRoute(depart, arrive) {
  return !isMainland(depart) || !isMainland(arrive);
}

function cityLabel(code) {
  return MAINLAND_CITY[code] || INTL_CITY[code] || code;
}

async function ensureBrowserFromSkill() {
  try {
    const probe = await fetch(`${CDP_URL}/json/version`, { signal: AbortSignal.timeout(600) });
    if (probe.ok) return;
  } catch {}
  execFileSync('node', [BROWSER_START], { encoding: 'utf8', timeout: 20000 });
}

async function withBrowser(task) {
  await ensureBrowserFromSkill();
  const browser = await chromium.connectOverCDP(CDP_URL, { timeout: 7000 });
  try {
    const ctx = browser.contexts()[0] || await browser.newContext();
    const page = ctx.pages()[0] || await ctx.newPage();
    return await task(page);
  } finally {
    await browser.close().catch(() => {});
  }
}

function buildDomesticUrl({ depart, arrive, departDate, returnDate, outboundFlight = '' }) {
  const fromCity = encodeURIComponent(MAINLAND_CITY[depart] || depart);
  const toCity = encodeURIComponent(MAINLAND_CITY[arrive] || arrive);
  return `https://www.ly.com/flights/itinerary/roundtrip/${depart}-${arrive}`
    + `?date=${departDate},${returnDate}`
    + `&from=${fromCity}&to=${toCity}`
    + `&fromairport=&toairport=&p=&childticket=0,0`
    + `&flightno=${outboundFlight}`;
}

function buildIntlUrl({ depart, arrive, departDate, returnDate, tripType }) {
  const isRT = tripType === 'roundtrip_context';
  const para = isRT
    ? `${depart}*${arrive}*${departDate}*${returnDate}*RT*1_0_0*Y|S|C|F`
    : `${depart}*${arrive}*${departDate}**OW*1_0_0*Y|S|C|F`;
  const departInter = isMainland(depart) ? '0' : 'true';
  const arriveInter = isMainland(arrive) ? '0' : 'true';
  return `https://www.ly.com/iflight/book1.html?para=${para}`
    + `&departureCity=${encodeURIComponent(cityLabel(depart))}&departureCityIsInter=${departInter}`
    + `&departAirport=&departAirportCode=`
    + `&arrivalCity=${encodeURIComponent(cityLabel(arrive))}&arrivalCityIsInter=${arriveInter}`
    + `&arriveAirport=&arriveAirportCode=&advanced=false`;
}

async function preparePage(page, url) {
  await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 45000 });
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(2500);
}

function flightKey(f) {
  return [f.flight, f.dep, f.arr, f.price?.amount ?? 0].join('|');
}

function formatCnDate(dateStr) {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(dateStr || '');
  if (!m) return dateStr || '';
  return `${Number(m[2])}月${Number(m[3])}日`;
}

function toMinutes(hhmm) {
  const m = /^(\d{2}):(\d{2})$/.exec(hhmm || '');
  if (!m) return null;
  return Number(m[1]) * 60 + Number(m[2]);
}

function enrichFlightDateTime(f, baseDate) {
  const depMinutes = toMinutes(f.dep);
  const arrMinutes = toMinutes(f.arr);
  const crossDay = depMinutes != null && arrMinutes != null && arrMinutes < depMinutes;
  const arrDate = crossDay ? addDays(baseDate, 1) : baseDate;
  return {
    ...f,
    depDateTime: `${formatCnDate(baseDate)} ${f.dep}`,
    arrDateTime: `${formatCnDate(arrDate)} ${f.arr}`,
  };
}

function legTime(f) {
  if (f.depDateTime || f.arrDateTime) return `${f.depDateTime || '--'} -> ${f.arrDateTime || '--'}`;
  return `${f.dep || '--:--'}->${f.arr || '--:--'}`;
}

function legRoute(f) {
  if (f.routeType === 'transfer') return `${f.transferCount || 1}次中转`;
  if (f.routeType === 'stopover') return `${f.stopoverCount || 1}次经停`;
  return '直飞';
}

function legInfo(f) {
  return f.transferInfo || f.stopoverInfo || '';
}

function priceText(f) {
  return f?.price?.text || `¥${f?.price?.amount ?? 0}`;
}

function renderTable(title, flights, showExtra = false) {
  const lines = [];
  lines.push(`${title}（共 ${flights.length} 班）`);
  lines.push('```');
  lines.push(showExtra ? '序号 | 航班号 | 日期时间 | 中转 | 总价 | 返程增量' : '序号 | 航班号 | 日期时间 | 中转 | 价格');
  lines.push(showExtra ? '-----+--------+----------+------+-----+--------' : '-----+--------+----------+------+-----');
  flights.forEach((f, i) => {
    const idx = String(i + 1).padStart(2, ' ');
    const flight = String(f.flight || '-').padEnd(8, ' ');
    const time = legTime(f).slice(0, 24).padEnd(24, ' ');
    const route = legRoute(f).padEnd(4, ' ');
    const price = priceText(f).padEnd(6, ' ');
    if (showExtra) {
      const extra = `+¥${Number.isFinite(Number(f.extra)) ? Number(f.extra) : 0}`;
      lines.push(`${idx}   | ${flight} | ${time} | ${route} | ${price} | ${extra}`);
    } else {
      lines.push(`${idx}   | ${flight} | ${time} | ${route} | ${price}`);
    }
  });
  lines.push('```');
  return lines.join('\n');
}

function buildView(payload) {
  const { mode, depart, arrive, departDate, returnDate, tripType, outboundFlight, outboundPrice, flights } = payload;
  if (!Array.isArray(flights) || flights.length === 0) return { table: '无可用航班。' };
  if (mode === 'outbound_day') {
    return {
      table: renderTable(`去程：${depart} -> ${arrive}  ${departDate}  (${tripType})`, flights, false),
      hint: '请选择去程航班（序号或航班号）',
    };
  }
  if (mode === 'return_after_outbound') {
    return {
      table: renderTable(`返程：${arrive} -> ${depart}  ${returnDate}（已定去程 ${outboundFlight} ¥${outboundPrice}）`, flights, true),
      hint: '请选择返程航班（序号或航班号）',
    };
  }
  return {
    table: renderTable(`返程列表：${arrive} -> ${depart}  ${returnDate}（锁定去程 ${outboundFlight}）`, flights, false),
    hint: 'roundtrip_locked 用于监控固定往返组合',
  };
}

async function collectFlights(page, baseDate) {
  const merged = new Map();
  let stableRounds = 0;
  let lastCount = 0;

  for (let round = 0; round < 18; round++) {
    const rows = await page.evaluate(async () => {
      const visible = el => !!el && el.offsetParent !== null;
      const textOf = el => (el?.innerText || '').replace(/\s+/g, ' ').trim();
      const wait = ms => new Promise(resolve => setTimeout(resolve, ms));
      const keyOf = item => [item.flight, item.dep, item.arr, item.price.amount].join('|');

      const clean = text => (text || '')
        .replace(/\s+/g, ' ')
        .replace(/[，,;；\s]+$/, '')
        .trim();

      const trimNoise = text => {
        const s = clean(text);
        if (!s) return '';
        const cut = s.search(/(?:¥|￥|余\d+张|选择|選擇|预订|預訂|订|訂)/);
        return clean(cut > 0 ? s.slice(0, cut) : s);
      };

      const readStopoverTooltip = () => {
        const tips = Array.from(document.querySelectorAll('.tooltip.popover.vue-popover-theme.open, .tooltip-inner.popover-inner'))
          .filter(visible)
          .map(textOf)
          .filter(t => t && t.includes('经停'));
        return trimNoise((tips[0] || '').replace(/^经停信息\s*/, '经停 '));
      };

      const readPrice = card => {
        const priceTextNode = card.querySelector('.price-info')
          || card.querySelector('.head-prices')
          || card.querySelector('.flight-price')
          || card.querySelector('.lowestPrice');
        const priceText = clean(textOf(priceTextNode || ''));
        const symbolMatch = priceText.match(/[¥￥]/);
        const amountMatch = priceText.match(/[¥￥]\s?([\d,]{2,7})/);
        if (amountMatch) {
          return {
            amount: Number(amountMatch[1].replace(/,/g, '')),
            text: priceText,
          };
        }
        const cardText = textOf(card);
        const fallback = cardText.match(/[¥￥]\s?([\d,]{2,7})/);
        if (!fallback) return null;
        return {
          amount: Number(fallback[1].replace(/,/g, '')),
          text: fallback[0],
        };
      };

      const cards = Array.from(document.querySelectorAll('.flight-item')).filter(visible);
      const out = [];
      const seen = new Set();

      for (let idx = 0; idx < cards.length; idx++) {
        const card = cards[idx];
        const t = textOf(card);
        if (!t) continue;
        const btn = card.querySelector('.buy-btn,.btn-select,.flight-btn');
        if (!btn) continue;

        const fm = t.match(/[A-Z0-9]{2}\d{3,4}/g);
        if (!fm || !fm.length) continue;
        const tm = [...t.matchAll(/\b\d{2}:\d{2}\b/g)].map(x => x[0]);
        if (tm.length < 2) continue;

        const priceMeta = readPrice(card);
        if (!priceMeta || !Number.isFinite(priceMeta.amount)) continue;

        const chain = [...new Set(fm)];
        const segmentCount = chain.length;
        const transferBySegments = Math.max(segmentCount - 1, 0);
        const transferNodeText = clean(textOf(card.querySelector('.arrow-item')));
        const transferByText = (() => {
          const m = t.match(/(\d+)次中转/);
          if (m) return Number(m[1]);
          return /中转|转机|转\s*[\u4e00-\u9fa5A-Za-z]/.test(t) ? 1 : 0;
        })();
        const transferCount = Math.max(transferBySegments, transferByText);

        const hasStopTag = !!Array.from(card.querySelectorAll('*')).find(el => visible(el) && textOf(el) === '经停');
        const stopoverByText = /经停|停\s*[\u4e00-\u9fa5A-Za-z]{2,}/.test(t) ? 1 : 0;
        const stopoverCount = hasStopTag ? 1 : stopoverByText;
        const transferInfo = trimNoise(transferNodeText || ((t.match(/(转[^。；\n]{0,30}\d+时\d+分|中转[^。；\n]{0,20}|转机[^。；\n]{0,20})/) || [])[1] || ''));
        const stopoverInfo = trimNoise((t.match(/(经停[^。；\n]{0,60}|停\s*[\u4e00-\u9fa5A-Za-z]{2,8})/) || [])[1] || '');

        const item = {
          __idx: idx,
          primaryFlight: chain[0],
          flight: chain.join('+'),
          dep: tm[0],
          arr: tm[1],
          price: { amount: priceMeta.amount, text: priceMeta.text || `¥${priceMeta.amount}` },
          segmentCount,
          transferCount,
          stopoverCount,
          routeType: transferCount > 0 ? 'transfer' : (stopoverCount > 0 ? 'stopover' : 'direct'),
          stops: transferCount + stopoverCount,
          isTransfer: transferCount + stopoverCount > 0,
          transferInfo,
          stopoverInfo,
        };
        const key = keyOf(item);
        if (seen.has(key)) continue;
        seen.add(key);
        out.push(item);
      }

      for (const item of out) {
        if (!item.stopoverCount) continue;
        const card = cards[item.__idx];
        if (!card) continue;
        const stopTag = Array.from(card.querySelectorAll('*')).find(el => visible(el) && textOf(el) === '经停');
        if (!stopTag) continue;
        try {
          stopTag.dispatchEvent(new MouseEvent('mouseenter', { bubbles: true }));
          stopTag.dispatchEvent(new MouseEvent('mouseover', { bubbles: true }));
          await wait(220);
          const tip = readStopoverTooltip();
          if (tip) {
            item.stopoverInfo = tip;
          }
        } finally {
          stopTag.dispatchEvent(new MouseEvent('mouseout', { bubbles: true }));
          stopTag.dispatchEvent(new MouseEvent('mouseleave', { bubbles: true }));
        }
      }
      return out.map(({ __idx, ...rest }) => rest);
    });

    for (const r of rows) {
      const key = flightKey(r);
      if (!merged.has(key)) merged.set(key, r);
    }

    const count = merged.size;
    if (count === lastCount) stableRounds += 1;
    else stableRounds = 0;
    lastCount = count;

    const atBottom = await page.evaluate(() => window.innerHeight + window.scrollY >= document.body.scrollHeight - 8);
    if (stableRounds >= 2 && atBottom) break;

    await page.evaluate(() => window.scrollBy(0, Math.max(window.innerHeight * 0.9, 700)));
    await page.waitForTimeout(900);
  }

  return Array.from(merged.values())
    .map(f => enrichFlightDateTime(f, baseDate))
    .sort((a, b) => (a.price?.amount ?? 0) - (b.price?.amount ?? 0));
}

async function clickOutbound(page, flightNo) {
  if (!flightNo) return false;
  const target = String(flightNo);
  const primary = target.split('+')[0];
  const clicked = await page.evaluate(targetFlight => {
    const visible = el => !!el && el.offsetParent !== null;
    const textOf = el => (el.innerText || '').replace(/\s+/g, ' ').trim();
    const clickCardButton = card => {
      if (!card) return false;
      const btn = Array.from(card.querySelectorAll('.flight-btn,.buy-btn,.btn-select,.tripui-online-btn'))
        .find(visible);
      if (!btn) return false;
      btn.click();
      return true;
    };

    const [full, first] = targetFlight.split('||');
    const cards = Array.from(document.querySelectorAll('.flight-item')).filter(visible);
    const exact = cards.find(card => textOf(card).includes(full));
    if (clickCardButton(exact)) return true;
    const primaryOnly = cards.find(card => textOf(card).includes(first));
    return clickCardButton(primaryOnly);
  }, `${target}||${primary}`);
  if (!clicked) return false;

  await page.waitForTimeout(1200);
  await page.waitForFunction(() => {
    const txt = document.body?.innerText || '';
    return txt.includes('去程已选') || txt.includes('选择返程') || txt.includes('返回日期');
  }, { timeout: 20000 }).catch(() => {});
  await page.waitForLoadState('networkidle', { timeout: 10000 }).catch(() => {});
  await page.waitForTimeout(1800);
  return true;
}

async function fetchOutbound({ depart, arrive, departDate, returnDate, tripType }) {
  const sourceUrl = isIntlRoute(depart, arrive)
    ? buildIntlUrl({ depart, arrive, departDate, returnDate, tripType })
    : buildDomesticUrl({ depart, arrive, departDate, returnDate, outboundFlight: '' });
  return withBrowser(async page => {
    await preparePage(page, sourceUrl);
    const flights = await collectFlights(page, departDate);
    return { sourceUrl: page.url(), flights };
  });
}

async function fetchReturn({ depart, arrive, departDate, returnDate, outboundFlight }) {
  const sourceUrl = isIntlRoute(depart, arrive)
    ? buildIntlUrl({ depart, arrive, departDate, returnDate, tripType: 'roundtrip_context' })
    : buildDomesticUrl({ depart, arrive, departDate, returnDate, outboundFlight: '' });
  return withBrowser(async page => {
    await preparePage(page, sourceUrl);
    const ok = await clickOutbound(page, outboundFlight);
    if (!ok) throw new Error(`Cannot select outbound flight: ${outboundFlight}`);
    const flights = await collectFlights(page, returnDate);
    return { sourceUrl: page.url(), flights };
  });
}

async function runOutboundDay(args) {
  const [depart, arrive, departDate, tripTypeArg = 'oneway', returnDateArg = ''] = args;
  if (!depart || !arrive || !departDate) {
    throw new Error('Usage: node search.cjs outbound_day <DEPART> <ARRIVE> <DEPART_DATE> [oneway|roundtrip_context] [RETURN_DATE]');
  }
  const tripType = tripTypeArg === 'roundtrip_context' ? 'roundtrip_context' : 'oneway';
  const returnDate = returnDateArg || addDays(departDate, 1);
  const data = await fetchOutbound({ depart, arrive, departDate, returnDate, tripType });
  const payload = { mode: 'outbound_day', tripType, depart, arrive, departDate, returnDate, sourceUrl: data.sourceUrl, flights: data.flights };
  return { ...payload, view: buildView(payload) };
}

async function runReturnAfterOutbound(args) {
  const [depart, arrive, departDate, returnDate, outboundFlight, outboundPriceArg] = args;
  const outboundPrice = Number(outboundPriceArg || 0);
  if (!depart || !arrive || !departDate || !returnDate || !outboundFlight) {
    throw new Error('Usage: node search.cjs return_after_outbound <DEPART> <ARRIVE> <DEPART_DATE> <RETURN_DATE> <OUTBOUND_FLIGHT> <OUTBOUND_PRICE>');
  }
  const data = await fetchReturn({ depart, arrive, departDate, returnDate, outboundFlight });
  const flights = Number.isFinite(outboundPrice) && outboundPrice > 0
    ? data.flights.map(f => ({ ...f, extra: (f.price?.amount ?? 0) - outboundPrice }))
    : data.flights;
  const payload = {
    mode: 'return_after_outbound',
    depart,
    arrive,
    departDate,
    returnDate,
    outboundFlight,
    outboundPrice: Number.isFinite(outboundPrice) ? outboundPrice : 0,
    sourceUrl: data.sourceUrl,
    flights,
  };
  return { ...payload, view: buildView(payload) };
}

async function runRoundtripLocked(args) {
  const [depart, arrive, departDate, returnDate, outboundFlight] = args;
  if (!depart || !arrive || !departDate || !returnDate || !outboundFlight) {
    throw new Error('Usage: node search.cjs roundtrip_locked <DEPART> <ARRIVE> <DEPART_DATE> <RETURN_DATE> <OUTBOUND_FLIGHT>');
  }
  const data = await fetchReturn({ depart, arrive, departDate, returnDate, outboundFlight });
  const payload = { mode: 'roundtrip_locked', depart, arrive, departDate, returnDate, outboundFlight, sourceUrl: data.sourceUrl, flights: data.flights };
  return { ...payload, view: buildView(payload) };
}

async function main() {
  const [,, mode, ...args] = process.argv;
  if (!mode) throw new Error('Mode required: outbound_day | return_after_outbound | roundtrip_locked');
  if (mode === 'outbound_day') return console.log(JSON.stringify(await runOutboundDay(args), null, 2));
  if (mode === 'return_after_outbound') return console.log(JSON.stringify(await runReturnAfterOutbound(args), null, 2));
  if (mode === 'roundtrip_locked') return console.log(JSON.stringify(await runRoundtripLocked(args), null, 2));
  throw new Error(`Unknown mode: ${mode}`);
}

main().catch(e => {
  process.stderr.write(`${e.message}\n`);
  process.exit(1);
});
