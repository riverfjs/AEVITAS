---
name: flight-search
description: Search round-trip domestic flights (China) via ly.com. Guides the user through selecting outbound then return flights with pricing.
---

# Flight Search

## Tool

```
node skills/flight-search/scripts/search.cjs <DEPART> <ARRIVE> <DEPART_DATE> <RETURN_DATE> [OUTBOUND_FLIGHT]
```

- `DEPART` / `ARRIVE`: IATA airport codes (e.g. `SZX`, `CKG`, `PEK`, `SHA`, `CAN`, `CTU`)
- `DEPART_DATE` / `RETURN_DATE`: `YYYY-MM-DD`
- `OUTBOUND_FLIGHT` (optional): selected outbound flight number → triggers return-flight mode

Output JSON:
```json
{
  "mode": "outbound" | "return",
  "flights": [{ "flight": "CZ3455", "dep": "13:25", "arr": "15:45", "price": 2071 }, ...]
}
```

## Conversation Flow

### Step 0 — Collect info
Ask the user (one message):
- Origin city / airport
- Destination city / airport
- Departure date
- Return date

Map city names to IATA codes (SZX=深圳, CKG=重庆, PEK/PKX=北京, SHA/PVG=上海, CAN=广州, CTU=成都, etc.)

### Step 1 — Show outbound flights
Run (no `OUTBOUND_FLIGHT`):
```
node skills/flight-search/scripts/search.cjs SZX CKG 2026-04-03 2026-04-07
```

Present results as a numbered table, sorted by price:

```
去程：深圳 → 重庆  2026-04-03（共 N 班）
 1. CZ2346  20:40→23:05  ¥1734
 2. CZ3641  21:10→23:35  ¥1816
 ...
请选择去程航班（输入序号或航班号）：
```

### Step 2 — Show return flights
After the user picks an outbound flight, pass its flight number **and price** to the script:
```
node skills/flight-search/scripts/search.cjs SZX CKG 2026-04-03 2026-04-07 CZ3455 2071
```

The `price` in the return result is the **round-trip total** (ly.com accumulates both legs).
The `extra` field = total − outbound price = incremental cost of the return leg.

Present return flights showing both:
```
返程：重庆 → 深圳  2026-04-07（共 N 班，价格为往返合计）
 1. CZ2335  08:00→10:20  往返¥4142（返程+¥2071）
 2. CZ5920  20:50→23:05  往返¥4324（返程+¥2253）
 ...
请选择返程航班：
```

### Step 3 — Summary
After the user picks a return flight, show the final summary using the **total price** (`price` from step 2 output):

```
✅ 行程确认

去程  CZ3455  深圳→重庆  04-03  13:25→15:45  ¥2071
返程  CZ2335  重庆→深圳  04-07  08:00→10:20  +¥2071
往返合计：¥4142

🔗 同程订票：https://www.ly.com/flights/itinerary/roundtrip/SZX-CKG?date=2026-04-03,2026-04-07&from=深圳&to=重庆&fromairport=&toairport=&p=&childticket=0,0&flightno=CZ3455
```

## Rules

- Always run the script; never guess prices or flight times.
- Do NOT call WebSearch or use the browser skill for this task.
- If `flights` array is empty, tell the user no flights were found and suggest changing dates.
- When user says a time like "13:25那班" or "第5个", match it to the correct flight number before proceeding.
- The booking URL uses the outbound flight number as `flightno=` parameter — always include it.
