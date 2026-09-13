# trafficgen

Linux 上行流量生成器。服务端持续对外发送数据；打开网页即可当客户端，数据到达后直接丢弃。面板显示当前速率、当月流量、累计流量。断线或重启后会自动续上。

默认端口 `54430`。

## 部署

```bash
curl -fsSL https://raw.githubusercontent.com/AndyZhang32767/trafficgen/main/scripts/install.sh | sudo bash
```

完成后打开 `http://服务器IP:54430/`。已安装过会询问是否覆盖，不同意则退出。

换端口：

```bash
PORT=9090 curl -fsSL https://raw.githubusercontent.com/AndyZhang32767/trafficgen/main/scripts/install.sh | sudo bash
```

本地已有代码时：

```bash
sudo scripts/deploy.sh server
```

或不装 systemd，直接跑：

```bash
go build -o server ./server
./server -addr :54430 -stats stats.json
```

云安全组需要放行对应端口。

## 客户端

浏览器打开 `http://服务器IP:54430/` 会自动连接并开始刷流量，页面上可改并发、停止或开始。单个标签页受浏览器同站连接数限制（大约 6 条），要更快可用下面两种方式。

Go 客户端：

```bash
sudo scripts/deploy.sh client http://服务器IP:54430/stream 4
```

curl：

```bash
chmod +x scripts/client.sh
scripts/client.sh http://服务器IP:54430/stream 4
```

## 运维

```bash
systemctl status trafficgen-server
systemctl restart trafficgen-server
systemctl stop trafficgen-client
journalctl -u trafficgen-server -f
```

## 参数

服务端 `server`：

- `-addr` 监听地址，默认 `:54430`
- `-stats` 统计文件，默认 `stats.json`

客户端 `client`：

- `-url` 流地址，例如 `http://IP:54430/stream`
- `-c` 并发连接数，默认 `4`

统计按累计和当月写入 `stats.json`，重启后继续累加。覆盖更新不会清掉这份统计。
