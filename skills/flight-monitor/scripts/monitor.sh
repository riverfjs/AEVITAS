#!/bin/bash
# Flight Monitor — data management only (no browser).
# Price checking is handled by check.cjs which calls flight-search/scripts/search.cjs.

SKILL_DIR="$HOME/.myclaw/workspace/.claude/skills/flight-monitor"
DATA_DIR="$SKILL_DIR/data"
MONITORS_FILE="$DATA_DIR/monitors.json"

mkdir -p "$DATA_DIR"
[ -f "$MONITORS_FILE" ] || echo "[]" > "$MONITORS_FILE"

generate_id() { echo "fm$(date +%s)"; }

add_monitor() {
    local from="$1" to="$2" depart="$3" ret="$4" flight="$5" ref_price="$6"

    if [ -z "$from" ] || [ -z "$to" ] || [ -z "$depart" ] || [ -z "$flight" ] || [ -z "$ref_price" ]; then
        echo "用法: monitor.sh add <出发IATA> <到达IATA> <出发日期> <返程日期> <航班号> <参考价格>"
        echo "示例: monitor.sh add SZX CKG 2026-04-03 2026-04-07 CZ3455 2071"
        return 1
    fi

    local id
    id=$(generate_id)

    node -e "
const fs = require('fs');
const data = JSON.parse(fs.readFileSync('$MONITORS_FILE', 'utf8'));
data.push({
  id: '$id',
  depart: '$from', arrive: '$to',
  departDate: '$depart', returnDate: '$ret',
  flight: '$flight', refPrice: $ref_price,
  lastPrice: null, lastChecked: null,
  created: Date.now()
});
fs.writeFileSync('$MONITORS_FILE', JSON.stringify(data, null, 2));
console.log('已添加监控 #$id  $from→$to  $flight  参考价¥$ref_price');
"
}

list_monitors() {
    node -e "
const fs = require('fs');
const data = JSON.parse(fs.readFileSync('$MONITORS_FILE', 'utf8'));
if (!data.length) { console.log('暂无监控任务'); process.exit(0); }
data.forEach((m, i) => {
  const last    = m.lastPrice != null ? '¥' + m.lastPrice : '未查询';
  const checked = m.lastChecked ? new Date(m.lastChecked).toLocaleString('zh-CN') : '-';
  console.log(\`[\${i+1}] #\${m.id}  \${m.depart}→\${m.arrive}  \${m.departDate}/\${m.returnDate}  \${m.flight}  参考¥\${m.refPrice}  最近:\${last}  (\${checked})\`);
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
        add)    add_monitor "$@" ;;
        list)   list_monitors ;;
        delete) delete_monitor "$@" ;;
        *)
            echo "航班价格监控"
            echo "用法:"
            echo "  monitor.sh add <出发IATA> <到达IATA> <出发日期> <返程日期> <航班号> <参考价格>"
            echo "  monitor.sh list"
            echo "  monitor.sh delete <id>"
            echo ""
            echo "价格检查: node \$SKILL_DIR/scripts/check.cjs"
            ;;
    esac
}

main "$@"
