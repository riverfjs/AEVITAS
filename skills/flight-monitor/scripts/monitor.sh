#!/bin/bash

# Flight Price Monitor Script (Trip.com)
# monitors.txt 格式（| 分隔，每行一条）:
#   id|from_iata|to_iata|depart_date|return_date|created|status|ob_dep|ob_arr|ret_dep|ret_arr
#
# from/to 使用 IATA 机场三字码（如 SZX、CKG、PEK）

SKILL_DIR="$HOME/.myclaw/workspace/.claude/skills/flight-monitor"
DATA_DIR="$SKILL_DIR/data"
MONITORS_FILE="$DATA_DIR/monitors.txt"
HISTORY_FILE="$DATA_DIR/price_history.txt"
BROWSER_SCRIPT="$HOME/.myclaw/workspace/.claude/skills/browser/scripts"
CHECK_PRICE="$SKILL_DIR/scripts/check-price.cjs"

mkdir -p "$DATA_DIR"

generate_id() { echo "$(date +%s)$RANDOM"; }

# 添加监控任务
# 用法: add <出发机场IATA> <到达机场IATA> <出发日期> <返程日期> <去程出发> <去程到达> [返程出发] [返程到达]
# 示例: add SZX CKG 2026-04-03 2026-04-07 13:05 15:30 15:50 18:05
add_monitor() {
    local from="$1" to="$2" depart="$3" ret="$4"
    local ob_dep="$5" ob_arr="$6" ret_dep="${7:-}" ret_arr="${8:-}"

    if [ -z "$from" ] || [ -z "$to" ] || [ -z "$depart" ] || [ -z "$ob_dep" ] || [ -z "$ob_arr" ]; then
        echo "用法: monitor.sh add <出发IATA> <到达IATA> <出发日期> <返程日期> <去程出发> <去程到达> [返程出发] [返程到达]"
        echo "示例: monitor.sh add SZX CKG 2026-04-03 2026-04-07 13:05 15:30 15:50 18:05"
        return 1
    fi

    local id=$(generate_id)
    echo "$id|$from|$to|$depart|$ret|$(date +%s)|enabled|$ob_dep|$ob_arr|$ret_dep|$ret_arr" >> "$MONITORS_FILE"

    echo "已添加监控任务 #$id"
    echo "  路线: $from → $to  ($depart → $ret)"
    echo "  去程: $ob_dep – $ob_arr"
    [ -n "$ret_dep" ] && echo "  返程: $ret_dep – $ret_arr"
}

list_monitors() {
    if [ ! -f "$MONITORS_FILE" ]; then echo "暂无监控任务"; return 0; fi
    echo "当前监控的航班："
    local count=0
    while IFS='|' read -r id from to depart ret created status ob_dep ob_arr ret_dep ret_arr; do
        count=$((count + 1))
        echo ""
        echo "[$count] #$id  $from → $to  ($depart / $ret)"
        echo "    去程: $ob_dep – $ob_arr"
        [ -n "$ret_dep" ] && echo "    返程: $ret_dep – $ret_arr"
        if [ -f "$HISTORY_FILE" ]; then
            local last
            last=$(grep "^[^|]*|$id|" "$HISTORY_FILE" | tail -1)
            if [ -n "$last" ]; then
                local ts price
                ts=$(echo "$last" | cut -d'|' -f1)
                price=$(echo "$last" | cut -d'|' -f3)
                echo "    最近价格: $price  ($(date -r "$ts" '+%m-%d %H:%M'))"
            fi
            local min_price
            min_price=$(grep "^[^|]*|$id|" "$HISTORY_FILE" | cut -d'|' -f3 | awk '
                {
                    price = $0; num = $0
                    gsub(/[^0-9,]/, "", num); gsub(/,/, "", num)
                    if (num+0 > 0 && (min == "" || num+0 < min)) { min = num+0; minraw = price }
                }
                END { print minraw }
            ')
            [ -n "$min_price" ] && echo "    历史最低: $min_price"
        fi
    done < "$MONITORS_FILE"
    [ $count -eq 0 ] && echo "暂无监控任务"
    return 0
}

check_flight_price() {
    local id="$1" from="$2" to="$3" depart="$4" ret="$5"
    local ob_dep="$6" ob_arr="$7" ret_dep="$8" ret_arr="$9"

    echo "查询 $from -> $to  去程 $depart $ob_dep-$ob_arr  返程 $ret $ret_dep-$ret_arr"

    local result
    result=$(node "$CHECK_PRICE" \
        "--from=$from" "--to=$to" \
        "--depart=$depart" "--return=$ret" \
        "--ob-dep=$ob_dep" "--ob-arr=$ob_arr" \
        "--ret-dep=$ret_dep" "--ret-arr=$ret_arr" \
        2>/dev/null)

    if [ -z "$result" ]; then
        echo "[RESULT] 查询失败: check-price.cjs 无输出（浏览器未启动或超时）"
        return 1
    fi

    local err
    err=$(python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('error',''))" <<< "$result" 2>/dev/null)
    if [ -n "$err" ]; then
        echo "[RESULT] 查询失败: $err"
        return 1
    fi

    local raw current
    raw=$(python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('raw',''))" <<< "$result" 2>/dev/null)
    current=$(python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('price',''))" <<< "$result" 2>/dev/null)

    # 航班详情
    local ob_airline ob_flight ob_aircraft ret_airline ret_flight ret_aircraft
    ob_airline=$(python3 -c "import sys,json; d=json.load(sys.stdin); ob=d.get('outbound',{}); print(ob.get('airline','') if ob else '')" <<< "$result" 2>/dev/null)
    ob_flight=$(python3 -c "import sys,json; d=json.load(sys.stdin); ob=d.get('outbound',{}); print(ob.get('flight','') if ob else '')" <<< "$result" 2>/dev/null)
    ob_aircraft=$(python3 -c "import sys,json; d=json.load(sys.stdin); ob=d.get('outbound',{}); print(ob.get('aircraft','') if ob else '')" <<< "$result" 2>/dev/null)
    ret_airline=$(python3 -c "import sys,json; d=json.load(sys.stdin); r=d.get('returnFlight',{}); print(r.get('airline','') if r else '')" <<< "$result" 2>/dev/null)
    ret_flight=$(python3 -c "import sys,json; d=json.load(sys.stdin); r=d.get('returnFlight',{}); print(r.get('flight','') if r else '')" <<< "$result" 2>/dev/null)
    ret_aircraft=$(python3 -c "import sys,json; d=json.load(sys.stdin); r=d.get('returnFlight',{}); print(r.get('aircraft','') if r else '')" <<< "$result" 2>/dev/null)

    # 行李详情（取 baggage 数组第一、二条）
    local bag_ob_cabin bag_ob_checked bag_ret_cabin bag_ret_checked
    bag_ob_cabin=$(python3 -c "import sys,json; d=json.load(sys.stdin); b=d.get('baggage',[]); print(b[0].get('cabin','') if b else '')" <<< "$result" 2>/dev/null)
    bag_ob_checked=$(python3 -c "import sys,json; d=json.load(sys.stdin); b=d.get('baggage',[]); print(b[0].get('checked','') if b else '')" <<< "$result" 2>/dev/null)
    bag_ret_cabin=$(python3 -c "import sys,json; d=json.load(sys.stdin); b=d.get('baggage',[]); print(b[1].get('cabin','') if len(b)>1 else '')" <<< "$result" 2>/dev/null)
    bag_ret_checked=$(python3 -c "import sys,json; d=json.load(sys.stdin); b=d.get('baggage',[]); print(b[1].get('checked','') if len(b)>1 else '')" <<< "$result" 2>/dev/null)

    # 价格对比
    local last=""
    if [ -f "$HISTORY_FILE" ]; then
        last=$(grep "^[^|]*|$id|" "$HISTORY_FILE" | tail -1 | cut -d'|' -f3)
    fi

    echo "$(date +%s)|$id|$raw" >> "$HISTORY_FILE"

    local change_msg="无变化"
    if [ -n "$last" ] && [ "$last" != "$raw" ]; then
        local last_num cur_num diff pct
        last_num=$(echo "$last" | sed 's/[^0-9]//g')
        cur_num=$(echo "$raw" | sed 's/[^0-9]//g')
        diff=$((cur_num - last_num))
        pct=$(awk "BEGIN{printf \"%.1f\", ($diff/$last_num)*100}")
        if [ $diff -lt 0 ]; then
            local abs=${diff#-}
            change_msg="价格下降 -$abs ($pct%)  上次: $last"
        else
            change_msg="价格上涨 +$diff ($pct%)  上次: $last"
        fi
    elif [ -z "$last" ]; then
        change_msg="首次记录"
    fi

    echo ""
    echo "[RESULT] $from->$to | 价格: $raw | $change_msg"
    echo "  去程: $depart $ob_dep-$ob_arr  $ob_airline $ob_flight  $ob_aircraft"
    echo "  返程: $ret $ret_dep-$ret_arr  $ret_airline $ret_flight  $ret_aircraft"
    echo "  行李 去程: 手提 $bag_ob_cabin  托运 $bag_ob_checked"
    echo "  行李 返程: 手提 $bag_ret_cabin  托运 $bag_ret_checked"
}

check_all() {
    echo "开始检查 [$(date '+%Y-%m-%d %H:%M:%S')]"
    if [ ! -f "$MONITORS_FILE" ]; then echo "暂无监控任务"; return 0; fi
    node "$BROWSER_SCRIPT/start.cjs" 2>/dev/null
    sleep 3
    while IFS='|' read -r id from to depart ret created status ob_dep ob_arr ret_dep ret_arr; do
        [ "$status" != "enabled" ] && continue
        echo "---"
        check_flight_price "$id" "$from" "$to" "$depart" "$ret" "$ob_dep" "$ob_arr" "$ret_dep" "$ret_arr"
        echo ""
    done < "$MONITORS_FILE"
    node "$BROWSER_SCRIPT/stop.cjs" 2>/dev/null
    echo "检查完成"
}

check_by_id() {
    local target_id="$1"
    [ -z "$target_id" ] && echo "缺少 ID" && return 1
    [ ! -f "$MONITORS_FILE" ] && echo "暂无监控任务" && return 1
    node "$BROWSER_SCRIPT/start.cjs" 2>/dev/null
    sleep 3
    local found=false
    while IFS='|' read -r id from to depart ret created status ob_dep ob_arr ret_dep ret_arr; do
        if [ "$id" = "$target_id" ]; then
            found=true
            check_flight_price "$id" "$from" "$to" "$depart" "$ret" "$ob_dep" "$ob_arr" "$ret_dep" "$ret_arr"
            break
        fi
    done < "$MONITORS_FILE"
    node "$BROWSER_SCRIPT/stop.cjs" 2>/dev/null
    [ "$found" = false ] && echo "未找到 #$target_id"
}

show_history() {
    [ ! -f "$HISTORY_FILE" ] && echo "暂无历史记录" && return 0
    local monitor_id="$1"
    echo "价格历史："
    if [ -n "$monitor_id" ]; then
        grep "^[^|]*|$monitor_id|" "$HISTORY_FILE" | while IFS='|' read -r ts id price; do
            echo "  $(date -r "$ts" '+%Y-%m-%d %H:%M')  $price"
        done
    else
        while IFS='|' read -r ts id price; do
            echo "  [#$id] $(date -r "$ts" '+%Y-%m-%d %H:%M')  $price"
        done < "$HISTORY_FILE"
    fi
}

delete_monitor() {
    local target_id="$1"
    [ -z "$target_id" ] && echo "缺少 ID" && return 1
    [ ! -f "$MONITORS_FILE" ] && echo "暂无监控任务" && return 1
    grep -v "^$target_id|" "$MONITORS_FILE" > "$MONITORS_FILE.tmp"
    mv "$MONITORS_FILE.tmp" "$MONITORS_FILE"
    echo "已删除 #$target_id"
}

main() {
    local cmd="$1"; shift
    case "$cmd" in
        add)       add_monitor "$@" ;;
        list)      list_monitors ;;
        check)     check_by_id "$@" ;;
        check-all) check_all ;;
        history)   show_history "$@" ;;
        delete)    delete_monitor "$@" ;;
        *)
            echo "航班价格监控 (Trip.com)"
            echo "用法:"
            echo "  monitor.sh add <出发IATA> <到达IATA> <出发日期> <返程日期> <去程出发> <去程到达> [返程出发] [返程到达]"
            echo "  monitor.sh list | check <id> | check-all | history [id] | delete <id>"
            echo ""
            echo "示例:"
            echo "  monitor.sh add SZX CKG 2026-04-03 2026-04-07 13:05 15:30 15:50 18:05"
            ;;
    esac
}

main "$@"
