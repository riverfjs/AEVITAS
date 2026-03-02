#!/usr/bin/env bash
set -euo pipefail

SKILL_DIR="$HOME/.aevitas/workspace/.claude/skills/flight-monitor"
DATA_DIR="$SKILL_DIR/data"
MONITORS_FILE="$DATA_DIR/monitors.json"
TODOIST_BIN="$HOME/.aevitas/workspace/.claude/skills/todoist/bin/todoist"
CHECK_CJS="$HOME/.aevitas/workspace/.claude/skills/flight-monitor/scripts/check.cjs"

mkdir -p "$DATA_DIR"
[ -f "$MONITORS_FILE" ] || echo "[]" >"$MONITORS_FILE"

generate_id() { echo "fm$(date +%s)"; }

usage() {
  cat <<'EOF'
Flight Monitor (new API only)

Usage:
  monitor.sh create roundtrip_locked <DEPART> <ARRIVE> <DEPART_DATE> <RETURN_DATE> <OUTBOUND_FLIGHT> <RETURN_FLIGHT> <BASELINE_TOTAL>
  monitor.sh create outbound_day <DEPART> <ARRIVE> <DEPART_DATE> [oneway|roundtrip_context] [RETURN_DATE]
  monitor.sh create return_after_outbound <DEPART> <ARRIVE> <DEPART_DATE> <RETURN_DATE> <OUTBOUND_FLIGHT> <OUTBOUND_PRICE>

  monitor.sh bind-cron <MONITOR_ID> <INTERVAL_MS>
  monitor.sh unbind-cron <MONITOR_ID>
  monitor.sh delete <MONITOR_ID> --with-cron|--keep-cron
  monitor.sh list

Examples:
  monitor.sh create roundtrip_locked SZX CKG 2026-04-03 2026-04-08 ZH9465 CZ3466 2303
  monitor.sh create outbound_day SZX CKG 2026-04-03 roundtrip_context 2026-04-08
  monitor.sh create return_after_outbound SZX CKG 2026-04-03 2026-04-08 ZH9465 2303
  monitor.sh bind-cron fm1770000000 21600000
  monitor.sh unbind-cron fm1770000000
  monitor.sh delete fm1770000000 --with-cron
EOF
}

require_todoist() {
  if [ ! -x "$TODOIST_BIN" ]; then
    echo "ERROR: todoist binary not found: $TODOIST_BIN"
    exit 1
  fi
}

monitor_exists() {
  local monitor_id="$1"
  node -e "
const fs=require('fs');
const data=JSON.parse(fs.readFileSync('$MONITORS_FILE','utf8'));
process.exit(data.some(m=>m.id==='$monitor_id') ? 0 : 1);
"
}

get_monitor_field() {
  local monitor_id="$1"
  local field="$2"
  node -e "
const fs=require('fs');
const data=JSON.parse(fs.readFileSync('$MONITORS_FILE','utf8'));
const m=data.find(x=>x.id==='$monitor_id');
if(!m){process.exit(2);}
const v=m['$field'];
if(v===undefined||v===null){process.exit(0);}
if(typeof v==='object'){console.log(JSON.stringify(v));} else {console.log(String(v));}
"
}

save_new_monitor() {
  local payload="$1"
  node -e "
const fs=require('fs');
const data=JSON.parse(fs.readFileSync('$MONITORS_FILE','utf8'));
data.push(JSON.parse(process.argv[1]));
fs.writeFileSync('$MONITORS_FILE', JSON.stringify(data,null,2));
" "$payload"
}

create_monitor() {
  local mode="${1:-}"
  shift || true

  if [ -z "$mode" ]; then
    echo "ERROR: create requires mode"
    usage
    exit 1
  fi

  local id
  id="$(generate_id)"
  local created
  created="$(date +%s000)"

  case "$mode" in
    roundtrip_locked)
      local depart="${1:-}" arrive="${2:-}" depart_date="${3:-}" return_date="${4:-}" out_flight="${5:-}" ret_flight="${6:-}" baseline_total="${7:-}"
      if [ -z "$depart" ] || [ -z "$arrive" ] || [ -z "$depart_date" ] || [ -z "$return_date" ] || [ -z "$out_flight" ] || [ -z "$ret_flight" ] || [ -z "$baseline_total" ]; then
        echo "ERROR: invalid args for roundtrip_locked"
        usage
        exit 1
      fi
      save_new_monitor "$(cat <<EOF
{"id":"$id","mode":"roundtrip_locked","status":"enabled","route":{"depart":"$depart","arrive":"$arrive","departDate":"$depart_date","returnDate":"$return_date"},"config":{"outboundFlight":"$out_flight","returnFlight":"$ret_flight","baselineTotalPrice":$baseline_total},"observed":{"lastTotalPrice":null,"lastReturnDep":null,"lastReturnArr":null},"cronJobId":null,"cronJobName":null,"created":$created,"lastChecked":null}
EOF
)"
      ;;
    outbound_day)
      local depart="${1:-}" arrive="${2:-}" depart_date="${3:-}" trip_type="${4:-oneway}" return_date="${5:-}"
      if [ -z "$depart" ] || [ -z "$arrive" ] || [ -z "$depart_date" ]; then
        echo "ERROR: invalid args for outbound_day"
        usage
        exit 1
      fi
      if [ "$trip_type" != "oneway" ] && [ "$trip_type" != "roundtrip_context" ]; then
        trip_type="oneway"
      fi
      if [ -n "$return_date" ]; then
        save_new_monitor "$(cat <<EOF
{"id":"$id","mode":"outbound_day","status":"enabled","route":{"depart":"$depart","arrive":"$arrive","departDate":"$depart_date","returnDate":"$return_date"},"config":{"tripType":"$trip_type"},"observed":{"lastMinPrice":null,"lastFlight":null,"lastDep":null,"lastArr":null},"cronJobId":null,"cronJobName":null,"created":$created,"lastChecked":null}
EOF
)"
      else
        save_new_monitor "$(cat <<EOF
{"id":"$id","mode":"outbound_day","status":"enabled","route":{"depart":"$depart","arrive":"$arrive","departDate":"$depart_date"},"config":{"tripType":"$trip_type"},"observed":{"lastMinPrice":null,"lastFlight":null,"lastDep":null,"lastArr":null},"cronJobId":null,"cronJobName":null,"created":$created,"lastChecked":null}
EOF
)"
      fi
      ;;
    return_after_outbound)
      local depart="${1:-}" arrive="${2:-}" depart_date="${3:-}" return_date="${4:-}" out_flight="${5:-}" out_price="${6:-}"
      if [ -z "$depart" ] || [ -z "$arrive" ] || [ -z "$depart_date" ] || [ -z "$return_date" ] || [ -z "$out_flight" ] || [ -z "$out_price" ]; then
        echo "ERROR: invalid args for return_after_outbound"
        usage
        exit 1
      fi
      save_new_monitor "$(cat <<EOF
{"id":"$id","mode":"return_after_outbound","status":"enabled","route":{"depart":"$depart","arrive":"$arrive","departDate":"$depart_date","returnDate":"$return_date"},"config":{"outboundFlight":"$out_flight","outboundPrice":$out_price},"observed":{"lastBestTotal":null,"lastBestReturnPrice":null,"lastBestReturnFlight":null,"lastBestReturnDep":null,"lastBestReturnArr":null},"cronJobId":null,"cronJobName":null,"created":$created,"lastChecked":null}
EOF
)"
      ;;
    *)
      echo "ERROR: unsupported mode: $mode"
      usage
      exit 1
      ;;
  esac

  echo "Created monitor: $id ($mode)"
}

extract_job_id_from_list() {
  local job_name="$1"
  "$TODOIST_BIN" cron-list | python3 - "$job_name" <<'PY'
import re,sys
name=sys.argv[1]
for line in sys.stdin:
    line=line.rstrip("\n")
    if f"] {name}  (id: " in line:
        m=re.search(r"\(id: ([^)]+)\)", line)
        if m:
            print(m.group(1))
            sys.exit(0)
sys.exit(1)
PY
}

bind_cron() {
  local monitor_id="${1:-}" interval_ms="${2:-}"
  if [ -z "$monitor_id" ] || [ -z "$interval_ms" ]; then
    echo "ERROR: Usage: monitor.sh bind-cron <MONITOR_ID> <INTERVAL_MS>"
    exit 1
  fi
  monitor_exists "$monitor_id" || { echo "ERROR: monitor not found: $monitor_id"; exit 1; }
  require_todoist

  local existing_job
  existing_job="$(get_monitor_field "$monitor_id" "cronJobId" || true)"
  if [ -n "$existing_job" ]; then
    echo "ERROR: monitor already bound to cron job: $existing_job"
    exit 1
  fi

  local job_name="flight-monitor-$monitor_id"
  local command="node $CHECK_CJS --id $monitor_id"
  "$TODOIST_BIN" cron-add "$job_name" "$command" "$interval_ms" >/dev/null

  local job_id
  if ! job_id="$(extract_job_id_from_list "$job_name")"; then
    echo "ERROR: cron created but job id lookup failed for: $job_name"
    exit 1
  fi

  node -e "
const fs=require('fs');
const data=JSON.parse(fs.readFileSync('$MONITORS_FILE','utf8'));
const idx=data.findIndex(m=>m.id==='$monitor_id');
if(idx<0){process.exit(2);}
data[idx].cronJobId='$job_id';
data[idx].cronJobName='$job_name';
fs.writeFileSync('$MONITORS_FILE', JSON.stringify(data,null,2));
"
  echo "Bound monitor #$monitor_id -> cron $job_id ($job_name)"
}

unbind_cron() {
  local monitor_id="${1:-}"
  if [ -z "$monitor_id" ]; then
    echo "ERROR: Usage: monitor.sh unbind-cron <MONITOR_ID>"
    exit 1
  fi
  monitor_exists "$monitor_id" || { echo "ERROR: monitor not found: $monitor_id"; exit 1; }
  require_todoist

  local job_id
  job_id="$(get_monitor_field "$monitor_id" "cronJobId" || true)"
  if [ -n "$job_id" ]; then
    "$TODOIST_BIN" cron-delete "$job_id" >/dev/null
  fi

  node -e "
const fs=require('fs');
const data=JSON.parse(fs.readFileSync('$MONITORS_FILE','utf8'));
const idx=data.findIndex(m=>m.id==='$monitor_id');
if(idx<0){process.exit(2);}
data[idx].cronJobId=null;
data[idx].cronJobName=null;
fs.writeFileSync('$MONITORS_FILE', JSON.stringify(data,null,2));
"
  echo "Unbound cron from monitor #$monitor_id"
}

delete_monitor() {
  local monitor_id="${1:-}" mode_flag="${2:-}"
  if [ -z "$monitor_id" ] || [ -z "$mode_flag" ]; then
    echo "ERROR: Usage: monitor.sh delete <MONITOR_ID> --with-cron|--keep-cron"
    exit 1
  fi
  monitor_exists "$monitor_id" || { echo "ERROR: monitor not found: $monitor_id"; exit 1; }
  require_todoist

  local job_id
  job_id="$(get_monitor_field "$monitor_id" "cronJobId" || true)"
  if [ "$mode_flag" = "--with-cron" ] && [ -n "$job_id" ]; then
    "$TODOIST_BIN" cron-delete "$job_id" >/dev/null
  elif [ "$mode_flag" != "--keep-cron" ] && [ "$mode_flag" != "--with-cron" ]; then
    echo "ERROR: mode must be --with-cron or --keep-cron"
    exit 1
  fi

  node -e "
const fs=require('fs');
const data=JSON.parse(fs.readFileSync('$MONITORS_FILE','utf8')).filter(m=>m.id!=='$monitor_id');
fs.writeFileSync('$MONITORS_FILE', JSON.stringify(data,null,2));
"
  echo "Deleted monitor #$monitor_id"
}

list_monitors() {
  node -e "
const fs=require('fs');
const data=JSON.parse(fs.readFileSync('$MONITORS_FILE','utf8'));
if(!Array.isArray(data)||!data.length){console.log('No monitors.');process.exit(0);}
data.forEach((m,i)=>{
  const r=m.route||{};
  const c=m.config||{};
  const o=m.observed||{};
  const range=r.returnDate?`${r.departDate}/${r.returnDate}`:r.departDate;
  const checked=m.lastChecked?new Date(m.lastChecked).toLocaleString('zh-CN'):'-';
  const cron=m.cronJobId?`${m.cronJobName} (${m.cronJobId})`:'(unbound)';
  let info='';
  if(m.mode==='roundtrip_locked'){
    info=`out:${c.outboundFlight} ret:${c.returnFlight} base:¥${c.baselineTotalPrice} last:${o.lastTotalPrice==null?'-':'¥'+o.lastTotalPrice}`;
  } else if(m.mode==='outbound_day'){
    info=`trip:${c.tripType||'oneway'} min:${o.lastMinPrice==null?'-':'¥'+o.lastMinPrice} flight:${o.lastFlight||'-'}`;
  } else if(m.mode==='return_after_outbound'){
    info=`out:${c.outboundFlight}(¥${c.outboundPrice}) bestTotal:${o.lastBestTotal==null?'-':'¥'+o.lastBestTotal} bestReturn:${o.lastBestReturnFlight||'-'}`;
  }
  console.log(`[${i+1}] #${m.id} [${m.mode}] ${r.depart}->${r.arrive} ${range} status:${m.status} cron:${cron} ${info} (checked:${checked})`);
});
"
}

main() {
  local cmd="${1:-}"
  shift || true
  case "$cmd" in
    create) create_monitor "$@" ;;
    bind-cron) bind_cron "$@" ;;
    unbind-cron) unbind_cron "$@" ;;
    delete) delete_monitor "$@" ;;
    list) list_monitors ;;
    *) usage ;;
  esac
}

main "$@"
