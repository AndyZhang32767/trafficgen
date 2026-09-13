# trafficgen

Linux 上行流量生成器：服务端在 VPS 上持续对外发送数据，客户端不停接收并直接丢弃，从而在服务端产生真实的上行（egress）流量。自带网页面板，实时显示当前速率、当月流量和累计流量；进程崩溃、断线或重启后都会自动续上。

适合消耗或压测你自己拥有、或已获授权机器的带宽与流量额度。

- **默认端口**：`54430`
- **自动续流**：客户端断线自动重连，systemd `Restart=always` 进程级拉起
- **统计持久化**：累计与当月流量写入 `stats.json`，重启后继续累加

## 工作原理

服务端提供三个入口：

| 路径 | 说明 |
|------|------|
| `/` | 网页面板，内置浏览器客户端，打开即开始刷流量 |
| `/stream` | 流量出口，持续向连接方发送数据 |
| `/stats` | JSON 统计接口，面板每秒轮询 |

每次发送的字节会累加到累计和当月（按 `年-月` 分桶），每 5 秒及退出时写入 `stats.json`。客户端收到数据后立即丢弃（`io.Discard` 或 `/dev/null`），不占内存。

## 部署服务端

一行安装（自动装依赖与 Go、拉代码、编译并注册为开机自启服务）：

```bash
curl -fsSL https://raw.githubusercontent.com/AndyZhang32767/trafficgen/main/scripts/install.sh | sudo bash
```

完成后打开 `http://服务器IP:54430/`。若检测到已安装，会询问是否覆盖更新；输入 `y` 覆盖（`stats.json` 保留），其他输入则中止。非交互场景可用 `FORCE=1` 跳过询问。

指定端口：

```bash
PORT=9090 curl -fsSL https://raw.githubusercontent.com/AndyZhang32767/trafficgen/main/scripts/install.sh | sudo bash
```

已把仓库拉到本地时，也可用脚本部署：

```bash
sudo scripts/deploy.sh server
```

或不装 systemd，直接前台运行：

```bash
go build -o server ./server
./server -addr :54430 -stats stats.json
```

> 记得在云厂商安全组／防火墙放行对应端口。

## 连接客户端

**网页版（最简单）**：浏览器打开 `http://服务器IP:54430/`，页面加载后自动连接并开始刷流量，可在页面上调整并发、随时停止或开始。单个标签页受浏览器同站连接数限制（约 6 条），想更快就多开标签或用下面的方式。

**Go 客户端（吞吐更高）**：

```bash
sudo scripts/deploy.sh client http://服务器IP:54430/stream 4
```

**curl 脚本（免 Go）**：

```bash
chmod +x scripts/client.sh
scripts/client.sh http://服务器IP:54430/stream 4
```

末尾的 `4` 是并发连接数，越大刷得越快，注意别打满客户端自己的带宽。

## 运维

```bash
systemctl status trafficgen-server      # 查看状态
systemctl restart trafficgen-server     # 重启
systemctl stop trafficgen-client        # 停止客户端
journalctl -u trafficgen-server -f      # 实时日志
```

## 参数

服务端 `server`：

- `-addr`：监听地址，默认 `:54430`
- `-stats`：统计文件路径，默认 `stats.json`

客户端 `client`：

- `-url`：流地址，例如 `http://IP:54430/stream`
- `-c`：并发连接数，默认 `4`

## 目录结构

```
server/    HTTP 服务端与网页面板
client/    Go 客户端
scripts/   install.sh 一行安装、deploy.sh 本地部署、client.sh curl 客户端、systemd 单元
```
