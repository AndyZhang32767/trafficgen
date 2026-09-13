#!/usr/bin/env bash
# 纯 curl 版客户端：无需 Go，直接持续下载并丢弃，断线自动重连。
# 用法: ./client.sh http://SERVER_IP:8080/stream [并发数]
set -u

URL="${1:-}"
CONN="${2:-4}"

if [ -z "$URL" ]; then
  echo "用法: $0 http://SERVER_IP:8080/stream [并发数]"
  exit 1
fi

echo "开始刷流量，目标: $URL，并发: $CONN（Ctrl+C 停止）"

worker() {
  while true; do
    # -s 静默, 数据全部丢到 /dev/null（即客户端拿到即 drop）
    curl -s --no-buffer "$URL" -o /dev/null
    # 断开后稍等重连
    sleep 1
  done
}

pids=()
for i in $(seq 1 "$CONN"); do
  worker &
  pids+=($!)
done

# 收到中断时清理子进程
trap 'echo "停止中..."; kill "${pids[@]}" 2>/dev/null; exit 0' INT TERM

wait
