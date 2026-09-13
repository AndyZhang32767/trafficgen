# 上行流量生成器（刷流量）

一套自有两端的流量生成工具：**服务端**在 Linux 上持续对外发送数据（产生上行/egress 流量），**客户端**连上 `服务端IP:端口` 后不停下载并**直接丢弃**数据包；网页面板实时显示 **当前速率 / 当月 / 累计** 流量。断线、重启、崩溃后都会**自动续上**。

最省事的用法：服务端启动后，**用浏览器打开 `http://服务端IP:54430/` 即可**——页面本身就是客户端，打开就自动连接开始刷流量（可在页面上设并发数、随时停止）。若要更高吞吐，再用下面的 Go / curl 客户端。

> 适用场景：消耗/测试你自己 VPS 的带宽与流量额度、带宽压测。请只对你自己拥有或获授权的机器使用。

## 组成

| 路径 | 说明 |
|------|------|
| `server/` | HTTP 服务端：`/` 面板（内置网页客户端）、`/stats` 统计接口、`/stream` 流量出口 |
| `client/` | Go 客户端：多连接持续下载并丢弃，断线自动重连 |
| `scripts/client.sh` | 免 Go 的 curl 版客户端（自动重连） |
| `scripts/*.service` | systemd 单元（`Restart=always` 自动续流） |
| `scripts/deploy.sh` | 一键编译+安装为开机自启服务 |
| `stats.json` | 累计/当月流量持久化文件（重启后续上，自动生成） |

## 工作原理

- 访问 `http://服务端IP:54430/stream` 即开始刷流量：服务端持续 `Write` 数据，客户端持续读取并 `丢弃(io.Discard / /dev/null)`。
- 服务端对每次发送的字节累加到 **累计** 和 **当月**（按 `年-月` 分桶），每 5 秒和退出时写入 `stats.json`，重启后自动加载。
- 面板每秒轮询 `/stats`，展示当前速率（按 bit 计，Mbps/Gbps）与流量（按 byte 计）。
- “自动续上”由两层保证：客户端程序内断线重连 + systemd `Restart=always` 进程级拉起。

## 快速开始

### 1. 服务端（Linux VPS，一行部署）

```bash
curl -fsSL https://raw.githubusercontent.com/AndyZhang32767/trafficgen/main/scripts/install.sh | sudo bash
```

会自动安装依赖、编译并做成开机自启服务。完成后浏览器打开 `http://你的服务端IP:54430/`。
换端口：`PORT=9090 curl -fsSL https://raw.githubusercontent.com/AndyZhang32767/trafficgen/main/scripts/install.sh | sudo bash`
（记得在云厂商安全组/防火墙放行对应端口。）

本地已有代码时也可以：

```bash
sudo scripts/deploy.sh server
```

完成后浏览器打开 `http://你的服务端IP:54430/` 查看面板。
（记得在云厂商安全组/防火墙放行 54430 端口。）

手动运行（不装服务）也可以：

```bash
go build -o server ./server
./server -addr :54430 -stats stats.json
```

### 2. 客户端

**方式 A：网页客户端（最简单，打开即用）**

浏览器直接访问 `http://服务端IP:54430/`，页面加载后会自动作为客户端连接并开始刷流量。可在页面上调整“并发连接”并点“停止/开始”。
注意：HTTP/1.1 下浏览器对同一站点最多约 6 个并发连接，所以网页客户端单页吞吐有上限；要更快请用下面的 Go 客户端或多开几个浏览器标签/机器。

**方式 B：Go 客户端（吞吐高）**

```bash
sudo scripts/deploy.sh client http://服务端IP:54430/stream 4
```

`4` 是并发连接数，越大刷得越快（注意别打满客户端自己的带宽）。

**方式 C：curl 脚本（免 Go）**

```bash
chmod +x scripts/client.sh
scripts/client.sh http://服务端IP:54430/stream 4
```

## 常用运维命令

```bash
# 查看/重启/停止
systemctl status trafficgen-server
systemctl restart trafficgen-server
systemctl stop trafficgen-client

# 看日志
journalctl -u trafficgen-server -f
journalctl -u trafficgen-client -f
```

## 参数说明

服务端 `server`：
- `-addr`：监听地址，默认 `:54430`
- `-stats`：统计持久化文件路径，默认 `stats.json`

客户端 `client`：
- `-url`：服务端流地址，如 `http://IP:54430/stream`
- `-c`：并发连接数，默认 `4`

## 调节刷流量速度

- 增加客户端 `-c` 并发数，或部署多台客户端。
- 单机速率主要受两端带宽限制；想更快就并行更多连接/机器。

## 注意

- 该工具会真实消耗带宽和流量费用，请确认额度与计费方式后再长期运行。
- 仅用于你自有或已获授权的机器之间，切勿指向他人服务。
