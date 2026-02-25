#!/usr/bin/env node
/**
 * check-price.cjs  — Trip.com 往返机票价格 & 详情提取
 *
 * 用法:
 *   node check-price.cjs \
 *     --from=SZX --to=CKG \
 *     --depart=2026-04-03 --return=2026-04-07 \
 *     --ob-dep=13:05 --ob-arr=15:30 \
 *     --ret-dep=15:50 --ret-arr=18:05
 *
 * 输出 JSON:
 * {
 *   price: "2800", currency: "CNY", raw: "CNY2,800",
 *   ticketPrice: "CNY2,660 × 1", tax: "CNY140 × 1",
 *   outbound: { dep: "13:05", arr: "15:30", airline: "南方航空", flight: "CZ3455", aircraft: "空中巴士 A320" },
 *   returnFlight: { dep: "15:50", arr: "18:05", airline: "深圳航空", flight: "ZH9464", aircraft: "波音 737-800" },
 *   baggage: { cabin: "免費", checked: "免費" }
 * }
 */

const PLAYWRIGHT = '/Users/fanjinsong/.myclaw/workspace/.claude/skills/browser/node_modules/playwright-core';
const { chromium } = require(PLAYWRIGHT);

const args = Object.fromEntries(
  process.argv.slice(2)
    .filter(a => a.startsWith('--'))
    .map(a => { const [k, ...v] = a.slice(2).split('='); return [k, v.join('=')]; })
);

const fromCode = (args['from'] || '').toLowerCase();
const toCode   = (args['to']   || '').toLowerCase();
const depart   = args['depart']  || '';
const ret      = args['return']  || '';
const obDep    = args['ob-dep']  || '';
const obArr    = args['ob-arr']  || '';
const retDep   = args['ret-dep'] || '';
const retArr   = args['ret-arr'] || '';

if (!fromCode || !toCode || !depart || !obDep || !obArr) {
  console.log(JSON.stringify({ error: 'Missing required args: --from --to --depart --ob-dep --ob-arr' }));
  process.exit(1);
}

function log(msg) { process.stderr.write(`[info] ${msg}\n`); }

/**
 * 滚动页面触发懒加载，找到目标班次后点击「選擇」按钮
 */
async function scrollAndSelectFlight(page, depTime, arrTime, label) {
  const deadline = Date.now() + 30000;
  let scrollY = 0;

  while (Date.now() < deadline) {
    const clicked = await page.evaluate(({ dep, arr }) => {
      const rows = Array.from(document.querySelectorAll('div.result-item'));
      const target = rows.find(r => {
        const times = r.innerText.match(/\d{2}:\d{2}/g) || [];
        return times[0] === dep && times[1] === arr;
      });
      if (!target) return null;
      const btn = target.querySelector('button.tripui-online-btn');
      if (!btn) return 'no-btn';
      btn.scrollIntoView({ block: 'center' });
      btn.click();
      return 'clicked';
    }, { dep: depTime, arr: arrTime });

    if (clicked === 'clicked') {
      log(`selected ${label}: ${depTime}–${arrTime}`);
      return { ok: true };
    }
    if (clicked === 'no-btn') {
      return { ok: false, error: `Found flight ${label} but no button` };
    }

    scrollY += 600;
    await page.evaluate(y => window.scrollTo(0, y), scrollY);
    await page.waitForTimeout(1000);
  }

  return { ok: false, error: `Flight not found: ${depTime}–${arrTime} (${label})` };
}

/**
 * 弹窗出现后点击「繼續」按钮
 * 若有 #dialogWrapper 遮挡，先强制关闭它
 */
async function clickContinue(page) {
  try {
    // 等候价格选择弹窗出现（含「繼續」文字）
    await page.waitForFunction(
      () => Array.from(document.querySelectorAll('button.tripui-online-btn'))
              .some(b => b.innerText.trim() === '繼續'),
      { timeout: 12000 }
    );

    // 若有 #dialogWrapper 遮挡，先隐藏它
    await page.evaluate(() => {
      const dlg = document.querySelector('#dialogWrapper');
      if (dlg) dlg.style.display = 'none';
    });

    // 直接用 evaluate 找到「繼續」按钮并点击（避免 Playwright click 被遮挡）
    const clicked = await page.evaluate(() => {
      const btns = Array.from(document.querySelectorAll('button.tripui-online-btn'));
      const btn = btns.filter(b => b.innerText.trim() === '繼續').pop();
      if (!btn) return false;
      btn.click();
      return true;
    });

    if (!clicked) return { ok: false, error: '繼續 button not found in DOM' };
    log('clicked 繼續');
    return { ok: true };
  } catch (e) {
    return { ok: false, error: `繼續 button not found: ${e.message}` };
  }
}

/**
 * 在预订确认页提取：航班详情 + 价格 + 行李
 */
async function extractBookingDetails(page) {
  await page.waitForLoadState('domcontentloaded', { timeout: 15000 }).catch(() => {});
  await page.waitForTimeout(3000);

  // 先隐藏所有遮挡浮层（#dialogWrapper、modal-mask 等）
  await page.evaluate(() => {
    const dlg = document.querySelector('#dialogWrapper');
    if (dlg) dlg.style.display = 'none';
    document.querySelectorAll('.ift-modal-wrap, [class*=modal-mask], [class*=overlay]')
      .forEach(el => { el.style.display = 'none'; });
  });

  // 用 JS click 展开左侧航班详情（避免 Playwright 被遮挡层拦截）
  const expanded = await page.evaluate(() => {
    const btn = document.querySelector('.flight-info-expand-detail');
    if (btn) { btn.click(); return true; }
    return false;
  });
  if (expanded) { await page.waitForTimeout(1500); log('expanded flight details'); }

  // 用 JS click 展开右侧价格明细
  const toggled = await page.evaluate(() => {
    const el = Array.from(document.querySelectorAll('*'))
      .find(e => e.children.length === 0 && e.innerText.trim() === '機票（1位成人）');
    if (el) { el.click(); return true; }
    return false;
  });
  if (toggled) { await page.waitForTimeout(1000); log('expanded price breakdown'); }

  return page.evaluate(() => {
    const result = {};

    // ── 左侧航班详情 (.m-flightInfo-booking) ─────────────────────
    // 展开后文本示例（按行）:
    //   深圳(SZX) → 重慶(CKG)  4月3日 週五  時間2小時25分
    //   13:05  SZX 深圳寶安國際機場T3  南方航空  CZ3455  空中巴士 A320  經濟艙
    //   15:30  CKG 重慶江北國際機場T3
    //   重慶(CKG) → 深圳(SZX)  4月7日 週二  時間2小時15分
    //   15:50  CKG ...  深圳航空  ZH9464  波音 737-800  經濟艙
    //   18:05  SZX ...
    const leftPanel = document.querySelector('.m-flightInfo-booking');
    if (leftPanel) {
      const lines = leftPanel.innerText.split('\n').map(l => l.trim()).filter(Boolean);

      const parseSegmentLines = (ls) => {
        const times    = ls.filter(l => /^\d{2}:\d{2}$/.test(l));
        const flight   = ls.find(l => /^[A-Z]{2}\d{3,4}$/.test(l)) || '';
        const airline  = ls.find(l => l.endsWith('航空') || l.endsWith('Airlines') || l.endsWith('Air')) || '';
        const aircraft = ls.find(l => /(空中巴士|波音|A\d{3}|737|321|320|319)/.test(l)) || '';
        return { dep: times[0] || '', arr: times[1] || '', flight, airline, aircraft };
      };

      // 用「→」分隔两段行程
      const sep = lines.reduce((acc, l, i) => {
        if (l.includes('→')) acc.push(i);
        return acc;
      }, []);

      if (sep.length >= 2) {
        result.outbound     = parseSegmentLines(lines.slice(sep[0], sep[1]));
        result.returnFlight = parseSegmentLines(lines.slice(sep[1]));
      } else if (sep.length === 1) {
        result.outbound     = parseSegmentLines(lines.slice(0, sep[0] + 10));
        result.returnFlight = parseSegmentLines(lines.slice(sep[0]));
      }
    }

    // ── 右侧价格卡片 ─────────────────────────────────────────────
    // 找右侧「價格詳情」卡片：先用 class*=price 缩小范围，取最小的包含行李+总额的元素
    const priceEls = Array.from(document.querySelectorAll('[class*=price]'));
    // 过滤：必须包含「總額」「行李」「CNY」，且 innerText 行数 <= 30（避免匹配整页）
    const card = priceEls
      .filter(el => {
        const t = el.innerText || '';
        return t.includes('總額') && t.includes('行李') && t.includes('CNY') && t.split('\n').length <= 30;
      })
      .sort((a, b) => (a.innerText || '').length - (b.innerText || '').length)[0];

    if (card) {
      const lines = card.innerText.split('\n').map(l => l.trim()).filter(Boolean);

      // 总额
      const totalIdx = lines.findIndex(l => l === '總額');
      if (totalIdx >= 0) {
        const v = lines.slice(totalIdx + 1).find(l => /^CNY[\d,]+$/.test(l));
        if (v) { result.raw = v; result.currency = 'CNY'; result.price = v.replace(/[^0-9]/g, ''); }
      }

      // 票价 & 税费（格式："CNY2,660 × 1"）
      const ticketIdx = lines.findIndex(l => l === '票價');
      if (ticketIdx >= 0) result.ticketPrice = lines[ticketIdx + 1] || '';
      const taxIdx = lines.findIndex(l => l.includes('稅項'));
      if (taxIdx >= 0) result.tax = lines[taxIdx + 1] || '';

      // 行李
      const cabinIdx   = lines.findIndex(l => l === '手提行李');
      const checkedIdx = lines.findIndex(l => l === '託運行李' || l === '托運行李');
      // 右侧价格卡片中仅保留简单行李标注（免費/付费）
      result.baggageSummary = {
        cabin:   cabinIdx   >= 0 ? (lines[cabinIdx + 1]   || '') : '',
        checked: checkedIdx >= 0 ? (lines[checkedIdx + 1] || '') : '',
      };
    } else {
      // Fallback：只拿总额
      const totalEl = Array.from(document.querySelectorAll('*')).find(el =>
        el.children.length === 0 && /^CNY[\d,]+$/.test((el.innerText || '').trim())
      );
      if (totalEl) {
        const v = totalEl.innerText.trim();
        result.raw = v; result.currency = 'CNY'; result.price = v.replace(/[^0-9]/g, '');
      }
    }

    // ── 左侧行李限额区域（不需要点击，直接读取） ─────────────────
    // 结构（通用）：「行李限額」块内，含「－」的行是路线分隔，
    // 其后几行包含 "1 × X公斤"（手提）和 "總共X公斤"（托运）
    const baggageSection = Array.from(document.querySelectorAll('*')).find(el =>
      el.offsetParent !== null &&
      (el.innerText || '').includes('行李限額') &&
      (el.innerText || '').includes('公斤') &&
      (el.innerText || '').split('\n').length < 40
    );
    if (baggageSection) {
      const bLines = baggageSection.innerText.split('\n').map(l => l.trim()).filter(Boolean);
      const parseBaggageSegment = (startIdx) => {
        const seg = bLines.slice(startIdx, startIdx + 10);
        const cabin   = seg.find(l => /\d\s*[×x]\s*\d+公斤/.test(l)) || '';
        const checked = seg.find(l => /總共\d+公斤/.test(l)) || '';
        return { route: bLines[startIdx] || '', cabin, checked };
      };
      // 找所有含「－」的路线行（通用，不依赖城市名）
      const routeIdxs = bLines.reduce((acc, l, i) => {
        if (i > 2 && l.includes('－')) acc.push(i);
        return acc;
      }, []);
      result.baggage = routeIdxs.map(idx => parseBaggageSegment(idx));
    }

    return result;
  });
}

(async () => {
  let browser;
  try {
    browser = await chromium.connectOverCDP('http://127.0.0.1:9222');
    const ctx  = browser.contexts()[0] || await browser.newContext();
    const page = ctx.pages()[ctx.pages().length - 1] || await ctx.newPage();

    // 1. 导航到 Trip.com 搜索页
    const tripType = ret ? 'rt' : 'ow';
    const urlParams = new URLSearchParams({
      dcity: fromCode, acity: toCode,
      ddate: depart,
      ...(ret ? { rdate: ret } : {}),
      triptype: tripType,
      class: 'y', lowpricesource: 'searchform',
      quantity: '1', searchboxarg: 't',
      nonstoponly: 'off', locale: 'zh-HK', curr: 'CNY',
    });
    const url = `https://hk.trip.com/chinaflights/showfarefirst?${urlParams}`;
    log(`navigate: ${url}`);
    await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 30000 });
    await page.waitForTimeout(5000);

    // 2. 选去程（含滚动懒加载）
    log(`select outbound: ${obDep}–${obArr}`);
    const ob = await scrollAndSelectFlight(page, obDep, obArr, 'outbound');
    if (!ob.ok) { console.log(JSON.stringify({ error: ob.error })); process.exit(1); }
    await page.waitForTimeout(5000);

    // 3. 选返程 + 点弹窗继续
    if (ret && retDep && retArr) {
      log(`select return: ${retDep}–${retArr}`);
      const rt = await scrollAndSelectFlight(page, retDep, retArr, 'return');
      if (!rt.ok) { console.log(JSON.stringify({ error: rt.error })); process.exit(1); }
      await page.waitForTimeout(3000);

      const cont = await clickContinue(page);
      if (!cont.ok) { console.log(JSON.stringify({ error: cont.error })); process.exit(1); }
      await page.waitForTimeout(6000);
    }

    // 4. 提取预订页详情
    log('extracting booking details...');
    const details = await extractBookingDetails(page);

    if (!details || !details.price) {
      console.log(JSON.stringify({ error: 'Failed to extract price', debug: details }));
      process.exit(1);
    }

    console.log(JSON.stringify(details, null, 2));
  } catch (err) {
    console.log(JSON.stringify({ error: err.message }));
    process.exit(1);
  } finally {
    if (browser) await browser.close().catch(() => {});
  }
})();
