---
name: flight-search
description: Fetch round-trip flight options and prices from ly.com. This skill is search/fetch only; monitoring, scheduling, and notifications are handled by flight-monitor.
---

# Flight Search

This skill is a data fetch utility. It does not persist monitor configs and does not send scheduled alerts.
Use `flight-monitor` for monitoring workflows.
Monitoring mode definitions and schedule semantics are defined by `flight-monitor`.

## Price Semantics (Important)

- In `mode: "outbound"`, each `flights[i].price` is the outbound fare shown by the source page.
- In `mode: "return"`, each `flights[i].price` is already the **round-trip total** returned by the script/source.
- In `mode: "return"`, `flights[i].extra` is computed by the script as:
  - `extra = price - outboundPrice`
  - This is the incremental return-leg amount for display only.
- The agent must not recompute total prices independently. Use script output as source of truth.

## Tool

```
node skills/flight-search/scripts/search.cjs <DEPART> <ARRIVE> <DEPART_DATE> <RETURN_DATE> [OUTBOUND_FLIGHT] [OUTBOUND_PRICE]
```

- `DEPART` / `ARRIVE`: IATA airport codes (e.g. `SZX`, `CKG`, `PEK`, `SHA`, `CAN`, `CTU`)
- `DEPART_DATE` / `RETURN_DATE`: `YYYY-MM-DD`
- `OUTBOUND_FLIGHT` (optional): selected outbound flight number → triggers return-flight mode
- `OUTBOUND_PRICE` (optional, used with `OUTBOUND_FLIGHT`): outbound fare used by script to compute `extra = total - outboundPrice`

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

Map city names to IATA codes (e.g. SZX=Shenzhen, CKG=Chongqing, PEK/PKX=Beijing, SHA/PVG=Shanghai, CAN=Guangzhou, CTU=Chengdu).

### Step 1 — Show outbound flights
Run (no `OUTBOUND_FLIGHT`):
```
node skills/flight-search/scripts/search.cjs SZX CKG 2026-04-03 2026-04-07
```

Present results as a numbered table, sorted by price:

```
Outbound: Shenzhen -> Chongqing  2026-04-03 (N flights)
 1. CZ2346  20:40→23:05  ¥1734
 2. CZ3641  21:10→23:35  ¥1816
 ...
Please choose an outbound flight (index or flight number):
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
Return: Chongqing -> Shenzhen  2026-04-07 (N flights, price is round-trip total)
 1. CZ3466  11:45→13:55  Total ¥2507  (Return +¥436)
 2. CZ5920  20:50→23:05  Total ¥2715  (Return +¥644)
 ...
Please choose a return flight:
```

### Step 3 — Summary
After the user picks a return flight, show the final summary using script fields directly:
- Outbound fare = selected outbound `price` from step 1
- Return incremental = selected return `extra` from step 2
- Final round-trip total = selected return `price` from step 2 (already total)

```
Trip Confirmed

Outbound  CZ3455  Shenzhen->Chongqing  04-03  13:25->15:45  ¥2071
Return    CZ3466  Chongqing->Shenzhen  04-07  11:45->13:55  +¥436
Round-trip total: ¥2507

Booking URL:
https://www.ly.com/flights/itinerary/roundtrip/SZX-CKG?date=2026-04-03,2026-04-07&from=Shenzhen&to=Chongqing&fromairport=&toairport=&p=&childticket=0,0&flightno=CZ3455
```

## Rules

- Always run the script; never guess prices or flight times.
- Do NOT call WebSearch or use the browser skill for this task.
- If `flights` array is empty, tell the user no flights were found and suggest changing dates.
- When user says a time like "the 13:25 one" or "the 5th one", map it to the correct flight number before proceeding.
- The booking URL uses the outbound flight number as `flightno=` parameter — always include it.
