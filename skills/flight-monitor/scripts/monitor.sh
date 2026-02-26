#!/bin/bash
# Flight Monitor — monitor config management only (no browser).
# Actual checking is handled by check.cjs.

SKILL_DIR="$HOME/.myclaw/workspace/.claude/skills/flight-monitor"
DATA_DIR="$SKILL_DIR/data"
MONITORS_FILE="$DATA_DIR/monitors.json"

mkdir -p "$DATA_DIR"
[ -f "$MONITORS_FILE" ] || echo "[]" > "$MONITORS_FILE"

generate_id() { echo "fm$(date +%s)"; }

add_roundtrip_locked() {
    local from="$1" to="$2" depart="$3" ret="$4" out_flight="$5" ret_flight="$6" base_total="$7"

    if [ -z "$from" ] || [ -z "$to" ] || [ -z "$depart" ] || [ -z "$ret" ] || [ -z "$out_flight" ] || [ -z "$ret_flight" ] || [ -z "$base_total" ]; then
        echo "用法: monitor.sh add-roundtrip <出发IATA> <到达IATA> <出发日期> <返程日期> <去程航班号> <返程航班号> <往返基准总价>"
        echo "示例: monitor.sh add-roundtrip SZX CKG 2026-04-03 2026-04-07 CZ3455 CZ2335 4142"
        return 1
    fi

    local id
    id=$(generate_id)

    node -e "
const fs = require('fs');
const data = JSON.parse(fs.readFileSync('$MONITORS_FILE', 'utf8'));
data.push({
  id: '$id',
  mode: 'roundtrip_locked',
  depart: '$from', arrive: '$to',
  departDate: '$depart', returnDate: '$ret',
  outboundFlight: '$out_flight',
  returnFlight: '$ret_flight',
  baselineTotalPrice: Number($base_total),
  lastObservedTotalPrice: null,
  lastChecked: null,
  status: 'enabled',
  created: Date.now()
});
fs.writeFileSync('$MONITORS_FILE', JSON.stringify(data, null, 2));
console.log('已添加监控 #$id [roundtrip_locked] $from→$to  去程:$out_flight  返程:$ret_flight  基准总价¥$base_total');
"
}

add_outbound_day() {
    local from="$1" to="$2" depart="$3" ret="${4:-}"
    local trip_type="oneway"

    if [ -z "$from" ] || [ -z "$to" ] || [ -z "$depart" ]; then
        echo "用法(单程): monitor.sh add-outbound-day <出发IATA> <到达IATA> <出发日期>"
        echo "用法(往返上下文): monitor.sh add-outbound-day <出发IATA> <到达IATA> <出发日期> <返程日期>"
        echo "示例(单程): monitor.sh add-outbound-day SZX CKG 2026-04-03"
        echo "示例(往返): monitor.sh add-outbound-day SZX CKG 2026-04-03 2026-04-07"
        return 1
    fi

    if [ -n "$ret" ]; then
        trip_type="roundtrip_context"
    fi

    local id
    id=$(generate_id)

    node -e "
const fs = require('fs');
const data = JSON.parse(fs.readFileSync('$MONITORS_FILE', 'utf8'));
data.push({
  id: '$id',
  mode: 'outbound_day',
  tripType: '$trip_type',
  depart: '$from', arrive: '$to',
  departDate: '$depart', returnDate: '${ret}',
  lastObservedMinPrice: null,
  lastObservedFlight: null,
  lastObservedDep: null,
  lastObservedArr: null,
  lastChecked: null,
  status: 'enabled',
  created: Date.now()
});
fs.writeFileSync('$MONITORS_FILE', JSON.stringify(data, null, 2));
console.log('已添加监控 #$id [outbound_day/$trip_type] $from→$to  $depart（不限定班次）');
"
}

add_return_watch() {
    local from="$1" to="$2" depart="$3" ret="$4" out_flight="$5" out_price="$6"

    if [ -z "$from" ] || [ -z "$to" ] || [ -z "$depart" ] || [ -z "$ret" ] || [ -z "$out_flight" ] || [ -z "$out_price" ]; then
        echo "用法: monitor.sh add-return-watch <出发IATA> <到达IATA> <出发日期> <返程日期> <已定去程航班号> <去程价格>"
        echo "示例: monitor.sh add-return-watch SZX CKG 2026-04-03 2026-04-07 CZ3455 2071"
        return 1
    fi

    local id
    id=$(generate_id)

    node -e "
const fs = require('fs');
const data = JSON.parse(fs.readFileSync('$MONITORS_FILE', 'utf8'));
data.push({
  id: '$id',
  mode: 'return_after_outbound',
  depart: '$from', arrive: '$to',
  departDate: '$depart', returnDate: '$ret',
  outboundFlight: '$out_flight',
  outboundPrice: Number($out_price),
  lastObservedBestTotal: null,
  lastObservedBestReturnPrice: null,
  lastObservedBestReturnFlight: null,
  lastObservedBestReturnDep: null,
  lastObservedBestReturnArr: null,
  lastChecked: null,
  status: 'enabled',
  created: Date.now()
});
fs.writeFileSync('$MONITORS_FILE', JSON.stringify(data, null, 2));
console.log('已添加监控 #$id [return_after_outbound] $from→$to  去程:$out_flight(¥$out_price)  监控返程最优总价');
"
}

list_monitors() {
    node -e "
const fs = require('fs');
const data = JSON.parse(fs.readFileSync('$MONITORS_FILE', 'utf8'));
if (!data.length) { console.log('暂无监控任务'); process.exit(0); }
data.forEach((m, i) => {
  const mode = m.mode || 'legacy';
  const checked = m.lastChecked ? new Date(m.lastChecked).toLocaleString('zh-CN') : '-';
  let extra = '';
  if (mode === 'roundtrip_locked') {
    const last = m.lastObservedTotalPrice != null ? ('¥' + m.lastObservedTotalPrice) : '未查询';
    extra = \`去:\${m.outboundFlight} 返:\${m.returnFlight} 基准¥\${m.baselineTotalPrice} 最近总价:\${last}\`;
  } else if (mode === 'outbound_day') {
    const last = m.lastObservedMinPrice != null ? ('¥' + m.lastObservedMinPrice) : '未查询';
    const f = m.lastObservedFlight ? m.lastObservedFlight : '-';
    const t = m.tripType || (m.returnDate ? 'roundtrip_context' : 'oneway');
    extra = \`类型:\${t} 去程最低:\${last} 航班:\${f}\`;
  } else if (mode === 'return_after_outbound') {
    const total = m.lastObservedBestTotal != null ? ('¥' + m.lastObservedBestTotal) : '未查询';
    const rf = m.lastObservedBestReturnFlight ? m.lastObservedBestReturnFlight : '-';
    extra = \`去程:\${m.outboundFlight}(¥\${m.outboundPrice}) 当前最优总价:\${total} 最优返程:\${rf}\`;
  } else {
    const last = m.lastPrice != null ? ('¥' + m.lastPrice) : '未查询';
    extra = \`旧格式 航班:\${m.flight || '-'} 最近:\${last}\`;
  }
  const dateRange = m.returnDate ? \`\${m.departDate}/\${m.returnDate}\` : m.departDate;
  console.log(\`[\${i+1}] #\${m.id} [\${mode}] \${m.depart}→\${m.arrive} \${dateRange}  \${extra} (\${checked})\`);
});
"
}

delete_monitor() {
    local target_id="$1"
    [ -z "$target_id" ] && echo "缺少 ID" && return 1
    node -e "
const fs = require('fs');
const data = JSON.parse(fs.readFileSync('$MONITORS_FILE', 'utf8')).filter(m => m.id !== '$target_id');
fs.writeFileSync('$MONITORS_FILE', JSON.stringify(data, null, 2));
console.log('已删除 #$target_id');
"
}

main() {
    local cmd="${1:-help}"; shift || true
    case "$cmd" in
        add-roundtrip)     add_roundtrip_locked "$@" ;;
        add-outbound-day)  add_outbound_day "$@" ;;
        add-return-watch)  add_return_watch "$@" ;;
        list)              list_monitors ;;
        delete)            delete_monitor "$@" ;;
        *)
            echo "航班价格监控"
            echo "用法:"
            echo "  monitor.sh add-roundtrip <出发IATA> <到达IATA> <出发日期> <返程日期> <去程航班号> <返程航班号> <往返基准总价>"
            echo "  monitor.sh add-outbound-day <出发IATA> <到达IATA> <出发日期> [返程日期]"
            echo "  monitor.sh add-return-watch <出发IATA> <到达IATA> <出发日期> <返程日期> <已定去程航班号> <去程价格>"
            echo "  monitor.sh list"
            echo "  monitor.sh delete <id>"
            echo ""
            echo "价格检查: node \$SKILL_DIR/scripts/check.cjs"
            ;;
    esac
}

main "$@"
