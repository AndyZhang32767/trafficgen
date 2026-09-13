#!/usr/bin/env bash
# 一行部署服务端（可 curl | sudo bash）。
# 可选环境变量: PORT=54430
set -euo pipefail

PORT="${PORT:-54430}"
DEST=/opt/trafficgen
REPO="https://github.com/AndyZhang32767/trafficgen.git"
GO_VER="1.22.12"

if [ "$(id -u)" -ne 0 ]; then
  echo "请用 root 运行，例如: curl -fsSL ... | sudo bash"
  exit 1
fi

export DEBIAN_FRONTEND=noninteractive
export PATH="/usr/local/go/bin:$PATH"

pkg_install() {
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update -y
    apt-get install -y --no-install-recommends ca-certificates curl git tar
  elif command -v dnf >/dev/null 2>&1; then
    dnf install -y ca-certificates curl git tar
  elif command -v yum >/dev/null 2>&1; then
    yum install -y ca-certificates curl git tar
  elif command -v apk >/dev/null 2>&1; then
    apk add --no-cache ca-certificates curl git tar
  else
    echo "未识别的包管理器，请先手动安装: curl git tar"
    exit 1
  fi
}

need_go() {
  if ! command -v go >/dev/null 2>&1; then
    return 0
  fi
  local major minor
  major="$(go env GOVERSION 2>/dev/null | sed -E 's/^go([0-9]+)\.([0-9]+).*/\1/')"
  minor="$(go env GOVERSION 2>/dev/null | sed -E 's/^go([0-9]+)\.([0-9]+).*/\2/')"
  [ -z "$major" ] && return 0
  [ "$major" -lt 1 ] && return 0
  [ "$major" -eq 1 ] && [ "$minor" -lt 21 ] && return 0
  return 1
}

install_go() {
  local arch tarball
  case "$(uname -m)" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) echo "不支持的架构: $(uname -m)"; exit 1 ;;
  esac
  tarball="go${GO_VER}.linux-${arch}.tar.gz"
  echo "安装 Go ${GO_VER} ..."
  curl -fsSL "https://go.dev/dl/${tarball}" -o "/tmp/${tarball}"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "/tmp/${tarball}"
  rm -f "/tmp/${tarball}"
  ln -sfn /usr/local/go/bin/go /usr/local/bin/go
}

pkg_install
if need_go; then
  install_go
fi

mkdir -p "$DEST"
if [ -d "$DEST/src/.git" ]; then
  echo "更新代码 ..."
  git -C "$DEST/src" fetch --depth 1 origin main
  git -C "$DEST/src" reset --hard origin/main
else
  rm -rf "$DEST/src"
  git clone --depth 1 -b main "$REPO" "$DEST/src"
fi

echo "编译服务端 ..."
( cd "$DEST/src" && go build -o "$DEST/server" ./server )

sed "s/:54430/:${PORT}/g" "$DEST/src/scripts/trafficgen-server.service" \
  > /etc/systemd/system/trafficgen-server.service

systemctl daemon-reload
systemctl enable --now trafficgen-server
systemctl restart trafficgen-server

IP="$(curl -fsS --max-time 5 https://ifconfig.me 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}' || echo '<服务器IP>')"
echo
echo "部署完成。"
echo "面板: http://${IP}:${PORT}/"
echo "刷流量入口: http://${IP}:${PORT}/stream"
echo "记得在云安全组/防火墙放行 ${PORT} 端口。"
systemctl --no-pager -l status trafficgen-server | head -n 8
