#!/usr/bin/env bash
# 一键部署脚本：在 Linux 服务器上编译并把 server/client 装成开机自启服务。
# 用法:
#   服务端:  sudo ./deploy.sh server
#   客户端:  sudo ./deploy.sh client http://SERVER_IP:54430/stream 4
set -e

ROLE="${1:-}"
DEST=/opt/trafficgen
HERE="$(cd "$(dirname "$0")/.." && pwd)"

if ! command -v go >/dev/null 2>&1; then
  echo "未检测到 Go，请先安装: https://go.dev/dl/  （或用 apt install golang）"
  exit 1
fi

already_installed() {
  [ -x "$DEST/server" ] || [ -x "$DEST/client" ] || [ -f /etc/systemd/system/trafficgen-server.service ] || [ -f /etc/systemd/system/trafficgen-client.service ]
}

if already_installed && [ "${FORCE:-}" != "1" ]; then
  echo "检测到已安装 trafficgen（$DEST）。"
  echo -n "是否覆盖更新？[y/N] "
  read -r ans || true
  case "$ans" in
    y|Y|yes|YES) echo "开始覆盖更新..." ;;
    *) echo "已中止，未做更改。"; exit 1 ;;
  esac
fi

mkdir -p "$DEST"

build() {
  echo "编译 $1 ..."
  ( cd "$HERE" && go build -o "$DEST/$1" "./$1" )
}

case "$ROLE" in
  server)
    build server
    install -m644 "$HERE/scripts/trafficgen-server.service" /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable --now trafficgen-server
    echo "服务端已启动。面板: http://<本机IP>:54430/  ，刷流量入口: /stream"
    systemctl status trafficgen-server --no-pager -l | head -n 5
    ;;
  client)
    URL="${2:-}"
    CONN="${3:-4}"
    if [ -z "$URL" ]; then echo "用法: sudo ./deploy.sh client http://SERVER_IP:54430/stream [并发]"; exit 1; fi
    build client
    sed "s#http://SERVER_IP:54430/stream#$URL#; s#-c 4#-c $CONN#" \
      "$HERE/scripts/trafficgen-client.service" > /etc/systemd/system/trafficgen-client.service
    systemctl daemon-reload
    systemctl enable --now trafficgen-client
    echo "客户端已启动，正在持续刷流量。"
    systemctl status trafficgen-client --no-pager -l | head -n 5
    ;;
  *)
    echo "用法: sudo ./deploy.sh server | client <url> [并发]"
    exit 1
    ;;
esac
