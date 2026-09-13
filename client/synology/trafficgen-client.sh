#!/bin/sh
# Synology DSM 上行流量客户端（纯 curl，零依赖）。
# 持续下载 /stream 并丢弃，单条连接断开自动重连；进程退出时清理子进程。
# 适合注册到「控制面板 → 任务计划 → 新增 → 触发的任务 → 开机」实现自启。
#
# 用法（二选一）：
#   ./trafficgen-client.sh http://SERVER_IP:54430/stream 4
#   URL=http://SERVER_IP:54430/stream CONN=4 ./trafficgen-client.sh
#
# 可选环境变量：
#   URL   服务端流地址
#   CONN  并发连接数（默认 4）
#   LOG   日志文件（默认 /var/log/trafficgen-client.log，不可写则退回 /tmp）
#   PIDFILE 单实例锁文件（默认 /tmp/trafficgen-client.pid）
set -u

URL="${1:-${URL:-}}"
CONN="${2:-${CONN:-4}}"
PIDFILE="${PIDFILE:-/tmp/trafficgen-client.pid}"
LOG="${LOG:-/var/log/trafficgen-client.log}"

if [ -z "$URL" ]; then
  echo "用法: $0 http://SERVER_IP:54430/stream [并发数]"
  echo "或设置环境变量 URL、CONN 后运行。"
  exit 1
fi

# 日志目录不可写时退回 /tmp
if ! ( : >> "$LOG" ) 2>/dev/null; then
  LOG="/tmp/trafficgen-client.log"
fi

log() { echo "$(date '+%Y-%m-%d %H:%M:%S') $*" >> "$LOG"; }

# 单实例保护：已有存活实例则退出，避免任务计划重复触发叠加连接
if [ -f "$PIDFILE" ]; then
  old="$(cat "$PIDFILE" 2>/dev/null || true)"
  if [ -n "$old" ] && kill -0 "$old" 2>/dev/null; then
    log "已有实例在运行 (pid=$old)，本次退出。"
    exit 0
  fi
fi
echo "$$" > "$PIDFILE"

log "启动: URL=$URL CONN=$CONN"

worker() {
  while true; do
    # -s 静默；--no-buffer 不缓存直接丢到 /dev/null；断开后稍等重连
    curl -s --no-buffer "$URL" -o /dev/null 2>/dev/null
    sleep 1
  done
}

pids=""
i=1
while [ "$i" -le "$CONN" ]; do
  worker &
  pids="$pids $!"
  i=$((i + 1))
done

cleanup() {
  log "收到停止信号，清理子进程。"
  # shellcheck disable=SC2086
  kill $pids 2>/dev/null || true
  rm -f "$PIDFILE"
  exit 0
}
trap cleanup INT TERM

wait
