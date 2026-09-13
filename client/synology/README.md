# Synology 客户端

在群晖 NAS（DSM，x86_64 机型）上当作刷流量客户端：持续下载服务端 `/stream` 并直接丢弃，断线自动重连，开机自启。用纯 `curl` 实现，DSM 自带，无需安装 Go 或任何依赖。

## 一、放置脚本

把 `trafficgen-client.sh` 上传到一个持久目录（例如共享文件夹 `docker` 或 `homes`），假设放在：

```
/volume1/docker/trafficgen/trafficgen-client.sh
```

用 SSH（控制面板 → 终端机与 SNMP → 启用 SSH）登录后赋予执行权限：

```sh
chmod +x /volume1/docker/trafficgen/trafficgen-client.sh
```

先手动测试（把地址换成你的服务端）：

```sh
/volume1/docker/trafficgen/trafficgen-client.sh http://SERVER_IP:54430/stream 4
```

服务端面板 `http://SERVER_IP:54430/` 能看到速率和流量上涨即正常，`Ctrl+C` 停止。

## 二、开机自启（任务计划）

控制面板 → 任务计划 → 新增 → 触发的任务 → 用户定义的脚本：

- 任务名称：`trafficgen-client`
- 用户：`root`
- 事件：`开机`
- 任务设置 → 运行命令：

```sh
URL=http://SERVER_IP:54430/stream CONN=4 /volume1/docker/trafficgen/trafficgen-client.sh
```

保存后可右键「运行」立即启动，之后每次开机自动拉起。脚本内已做单实例保护，重复触发不会叠加连接。

## 三、可选：进程守护（防止意外退出）

开机任务只在启动时运行一次。若担心进程被杀，可再建一个「计划的任务」，设为每 5 分钟运行同一条命令——脚本检测到已有实例会直接退出，没有实例则重新拉起，等于轻量守护。

## 参数

- 第一个参数或 `URL`：服务端流地址，如 `http://IP:54430/stream`
- 第二个参数或 `CONN`：并发连接数，默认 `4`（越大越快，注意别打满 NAS 上行）
- `LOG`：日志文件，默认 `/var/log/trafficgen-client.log`，不可写时退回 `/tmp`
- `PIDFILE`：单实例锁，默认 `/tmp/trafficgen-client.pid`

## 停止

停用或删除任务计划；正在运行的进程可用：

```sh
kill "$(cat /tmp/trafficgen-client.pid)"
```

## 想要更高吞吐？

curl 版足够日常使用。若追求更低 CPU、更高吞吐，可在开发机交叉编译 Go 客户端（x86_64）后上传运行：

```sh
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o trafficgen-client-amd64 ./client
```

上传后运行：`./trafficgen-client-amd64 -url http://SERVER_IP:54430/stream -c 4`，同样可注册进任务计划自启。
