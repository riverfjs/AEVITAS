---
name: flight-search
description: Fetch round-trip flight options and prices from ly.com. This skill is search/fetch only; monitoring, scheduling, and notifications are handled by flight-monitor.
---

# Flight Search

This skill is a data fetch utility. It does not persist monitor configs and does not send scheduled alerts.
Use `flight-monitor` for monitoring workflows.
Monitoring mode definitions and schedule semantics are defined by `flight-monitor`.

## Price Semantics (Important)

- `flights[i].price` is an object:
  - `price.amount`: numeric price for sorting/comparison
  - `price.text`: display text from source (e.g. `¥2025起`)
- In `mode: "outbound_day"`, `price` is outbound fare.
- In `mode: "return_after_outbound"` and `mode: "roundtrip_locked"`, `price` is round-trip total from source.
- In `mode: "return_after_outbound"`, `flights[i].extra = price.amount - outboundPrice`.
- The script output is source of truth; agent must not recompute totals by itself.

## Route Support

- Mainland domestic and international routes both use browser-driven extraction.
- `search.cjs` reuses browser skill lifecycle (`start.cjs` + CDP connection).
- All three modes support international routes.

## Tool

```
node skills/flight-search/scripts/search.cjs outbound_day <DEPART> <ARRIVE> <DEPART_DATE> [oneway|roundtrip_context] [RETURN_DATE]
node skills/flight-search/scripts/search.cjs return_after_outbound <DEPART> <ARRIVE> <DEPART_DATE> <RETURN_DATE> <OUTBOUND_FLIGHT> <OUTBOUND_PRICE>
node skills/flight-search/scripts/search.cjs roundtrip_locked <DEPART> <ARRIVE> <DEPART_DATE> <RETURN_DATE> <OUTBOUND_FLIGHT>
```

- `DEPART` / `ARRIVE`: IATA airport codes (e.g. `SZX`, `CKG`, `PEK`, `SHA`, `CAN`, `CTU`)
- `DEPART_DATE` / `RETURN_DATE`: `YYYY-MM-DD`
- `tripType`: `oneway` or `roundtrip_context`
- `OUTBOUND_FLIGHT`: selected outbound flight number
- `OUTBOUND_PRICE`: outbound fare used by script to compute `extra = total - outboundPrice`

Output JSON:
```json
{
  "mode": "outbound_day" | "return_after_outbound" | "roundtrip_locked",
  "view": {
    "table": "formatted table block for direct display",
    "hint": "next-step hint"
  },
  "flights": [{ "flight": "CZ3455", "dep": "13:25", "arr": "15:45", "price": { "amount": 2071, "text": "¥2071起" } }, ...]
}
```

## Conversation Flow

### Step 0 — Collect info
Ask the user (one message):
- Origin city / airport
- Destination city / airport
- Departure date
- Return date

Map city names to IATA codes (e.g. SZX=Shenzhen, CKG=Chongqing, PEK/PKX=Beijing, SHA/PVG=Shanghai, CAN=Guangzhou, CTU=Chengdu).

### Step 1 — Show outbound flights
Run:
```
node skills/flight-search/scripts/search.cjs outbound_day SZX CKG 2026-04-03 oneway
```

Display `view.table` directly, then ask with `view.hint`.

### Step 2 — Show return flights
After the user picks an outbound flight, pass its flight number and `price.amount` to the script:
```
node skills/flight-search/scripts/search.cjs return_after_outbound SZX CKG 2026-04-03 2026-04-07 CZ3455 2071
```

Display `view.table` directly, then ask with `view.hint`.

### Step 3 — Summary
After the user picks a return flight, show the final summary using script fields directly:
- Outbound fare = selected outbound `price.amount` from step 1
- Return incremental = selected return `extra` from step 2
- Final round-trip total = selected return `price.amount` from step 2 (already total)

```
Trip Confirmed

Outbound  CZ3455  Shenzhen->Chongqing  04-03  13:25->15:45  ¥2071
Return    CZ3466  Chongqing->Shenzhen  04-07  11:45->13:55  +¥436
Round-trip total: ¥2507

Booking URL:
https://www.ly.com/flights/itinerary/roundtrip/SZX-CKG?date=2026-04-03,2026-04-07&from=Shenzhen&to=Chongqing&fromairport=&toairport=&p=&childticket=0,0&flightno=CZ3455
```

## Rules

- Always run the script; never guess prices or times.
- Must present the full `flights` array returned by the script; never truncate to top N.
- Prefer showing `view.table` directly instead of re-formatting tables in the skill.
- Do NOT call WebSearch or use the browser skill for this task.
- If `flights` array is empty, tell the user no flights were found and suggest changing dates.
- When user says a time like "the 13:25 one" or "the 5th one", map it to the correct flight number before proceeding.
- The booking URL uses the outbound flight number as `flightno=` parameter — always include it.
