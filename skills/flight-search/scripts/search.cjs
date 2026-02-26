#!/usr/bin/env node
'use strict';
/**
 * search.cjs — Fetch round-trip flight list from ly.com (server-side rendered, no browser needed).
 *
 * Usage:
 *   # Step 1 — outbound flights only
 *   node search.cjs <DEPART> <ARRIVE> <DEPART_DATE> <RETURN_DATE>
 *
 *   # Step 2 — after user picks outbound flight, get return flights
 *   node search.cjs <DEPART> <ARRIVE> <DEPART_DATE> <RETURN_DATE> <OUTBOUND_FLIGHT_NO>
 *
 * Arguments:
 *   DEPART / ARRIVE       IATA codes, e.g. SZX CKG
 *   DEPART_DATE / RETURN_DATE   YYYY-MM-DD
 *   OUTBOUND_FLIGHT_NO    Optional, e.g. CZ3455 — triggers return-flight mode
 *
 * Output: JSON  { mode, depart, arrive, departDate, returnDate, outboundFlight, flights[] }
 *   mode: "outbound" | "return"
 *   flights[]: { flight, dep, arr, price }  sorted by price asc
 */

const https = require('https');

// IATA → Chinese city name (for ly.com query string)
const CITY = {
  SZX: '深圳', CKG: '重庆', PEK: '北京', PKX: '北京',
  SHA: '上海', PVG: '上海', CAN: '广州', CTU: '成都',
  XIY: '西安', WUH: '武汉', CSX: '长沙', KMG: '昆明',
  NKG: '南京', HGH: '杭州', XMN: '厦门', TSN: '天津',
  DLC: '大连', TAO: '青岛', SHE: '沈阳', HRB: '哈尔滨',
  URC: '乌鲁木齐', KWE: '贵阳', NNG: '南宁', HAK: '海口',
  SYX: '三亚', LHW: '兰州', TNA: '济南',
};

function get(urlStr) {
  return new Promise((resolve, reject) => {
    const requestUrl = new URL(urlStr);
    const opts = { headers: {
      'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
      'Accept-Language': 'zh-CN,zh;q=0.9',
      'Accept': 'text/html,application/xhtml+xml',
    }};
    https.get(requestUrl, opts, res => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        const redirected = new URL(res.headers.location, requestUrl).toString();
        return get(redirected).then(resolve).catch(reject);
      }
      const chunks = [];
      res.on('data', c => chunks.push(c));
      res.on('end', () => resolve(Buffer.concat(chunks).toString('utf-8')));
    }).on('error', reject);
  });
}

function parseFlights(html) {
  // Strip style/script blocks, then all tags → plain text
  let text = html.replace(/<style[^>]*>[\s\S]*?<\/style>/gi, ' ');
  text = text.replace(/<script[^>]*>[\s\S]*?<\/script>/gi, ' ');
  text = text.replace(/<[^>]+>/g, ' ');
  text = text.replace(/[ \t]+/g, ' ');

  // Pattern: FLIGHT_NO  HH:MM  AIRPORT  HH:MM  AIRPORT  ¥PRICE 起
  const re = /([A-Z][A-Z0-9]\d{3,4})\s+(\d{2}:\d{2})\s+\S+\s+(\d{2}:\d{2})\s+\S+\s+¥(\d+)\s+起/g;
  const seen = new Set();
  const results = [];
  let m;
  while ((m = re.exec(text)) !== null) {
    const key = m[1] + m[2];
    if (!seen.has(key)) {
      seen.add(key);
      results.push({ flight: m[1], dep: m[2], arr: m[3], price: parseInt(m[4], 10) });
    }
  }
  return results.sort((a, b) => a.price - b.price);
}

async function main() {
  const [,, depart, arrive, departDate, returnDate, outboundFlight = '', outboundPriceArg = ''] = process.argv;
  const outboundPrice = parseInt(outboundPriceArg, 10) || 0;

  if (!depart || !arrive || !departDate || !returnDate) {
    process.stderr.write('Usage: node search.cjs DEPART ARRIVE DEPART_DATE RETURN_DATE [OUTBOUND_FLIGHT]\n');
    process.exit(1);
  }

  const fromCity = encodeURIComponent(CITY[depart] || depart);
  const toCity   = encodeURIComponent(CITY[arrive]  || arrive);
  const pageUrl  = `https://www.ly.com/flights/itinerary/roundtrip/${depart}-${arrive}`
    + `?date=${departDate},${returnDate}`
    + `&from=${fromCity}&to=${toCity}`
    + `&fromairport=&toairport=&p=&childticket=0,0`
    + `&flightno=${outboundFlight}`;

  const html = await get(pageUrl);
  const all  = parseFlights(html);

  let flights;
  let mode;

  if (!outboundFlight) {
    // Step 1: outbound only — all flights on page are outbound
    mode    = 'outbound';
    flights = all;
  } else {
    // Step 2: page shows outbound (pre-selected) + return flights
    // Fetch step-1 page to know which flights are outbound, then exclude them
    const baseUrl = `https://www.ly.com/flights/itinerary/roundtrip/${depart}-${arrive}`
      + `?date=${departDate},${returnDate}`
      + `&from=${fromCity}&to=${toCity}`
      + `&fromairport=&toairport=&p=&childticket=0,0&flightno=`;
    const baseHtml = await get(baseUrl);
    const outboundKeys = new Set(parseFlights(baseHtml).map(f => f.flight + f.dep));

    mode    = 'return';
    flights = all.filter(f => !outboundKeys.has(f.flight + f.dep));
    if (!flights.length) flights = all;

    // price shown by ly.com in return mode = round-trip total.
    // Add extra = total - outbound so the AI shows incremental cost.
    if (outboundPrice > 0) {
      flights = flights.map(f => ({ ...f, extra: f.price - outboundPrice }));
    }
  }

  console.log(JSON.stringify({
    mode,
    depart, arrive, departDate, returnDate,
    outboundFlight: outboundFlight || null,
    outboundPrice: outboundPrice || null,
    sourceUrl: pageUrl,
    // price in return mode = round-trip total; extra = incremental return cost
    flights,
  }, null, 2));
}

main().catch(e => { process.stderr.write(e.message + '\n'); process.exit(1); });
