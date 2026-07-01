# ServerStatus 中文版

ServerStatus 是一个轻量的服务器探针和云监控面板，支持多节点在线状态、资源占用、三网延迟、服务监测、SSL 证书检查、Watchdog 告警和 Web 配置管理。

在线演示：https://tz.cloudcpp.com

[![Python Support](https://img.shields.io/badge/python-3.6%2B%20-blue.svg)](https://github.com/cppla/ServerStatus)
[![C++ Compiler](http://img.shields.io/badge/C++-GNU-blue.svg?style=flat&logo=cplusplus)](https://github.com/cppla/ServerStatus)
[![License](https://img.shields.io/badge/license-MIT-4EB1BA.svg?style=flat-square)](https://github.com/cppla/ServerStatus)
[![Version](https://img.shields.io/badge/Version-Build%201.1.7-red)](https://github.com/cppla/ServerStatus)

![Latest Host Version](https://dl.cpp.la/Archive/serverstatus_1_1_7.png)

## 功能

- 主机监控：在线状态、CPU、内存、虚存、硬盘、IO、负载、进程、连接、流量。
- 网络质量：联通、电信、移动三网延迟和丢包。
- 服务监测：通过客户端上报自定义监测结果。
- SSL 证书：检查证书剩余天数、域名不匹配和过期风险。
- Watchdog：基于表达式规则触发告警回调。
- WebUI：主机、服务、证书、配置四个页面；配置页可增删改节点、监测、证书和 Watchdog 规则。
- HTTP 管理 API：方便 WebUI 或 AI Agent 读写配置、重载和重启服务。

注意：Watchdog 的 `interval` 是最小通知间隔，用于避免频繁报警，并不是探测间隔。`rule` 使用 Exprtk 表达式，当前窄字符解析对中文等 Unicode 字符不友好，规则中建议使用英文、数字和字段名。

## 快速启动

### 服务端 Docker Compose

推荐用 Compose 启动服务端。默认 Web 端口映射为 `8080:80`，客户端连接端口为 `35601`。

```bash
ADMIN_TOKEN='your-strong-token' docker compose -f docker-compose-server.yml up -d --build
```

启动后访问：

- WebUI：http://127.0.0.1:8080/
- HTTP API 自检：http://127.0.0.1:8080/api/health
- HTTP API 文档：http://127.0.0.1:8080/api/schema

`ADMIN_TOKEN` 可选；不设置时 WebUI 仍可查看监控数据，但「配置」页和写入类 API 不允许修改配置。

`docker-compose-server.yml` 默认挂载：

```text
./server/config.json -> /ServerStatus/server/config.json
./web/json          -> /usr/share/nginx/html/json
```

### 服务端 Docker Run

```bash
wget --no-check-certificate -qO ~/serverstatus-config.json \
  https://raw.githubusercontent.com/cppla/ServerStatus/master/server/config.json
mkdir -p ~/serverstatus-monthtraffic

docker run -d --restart=always --name=serverstatus-server \
  -e ADMIN_TOKEN='your-strong-token' \
  -v ~/serverstatus-config.json:/ServerStatus/server/config.json \
  -v ~/serverstatus-monthtraffic:/usr/share/nginx/html/json \
  -p 8080:80 -p 35601:35601 \
  cppla/serverstatus:server
```

### 客户端 Docker Compose

客户端必须使用服务端 `config.json` 中存在的 `username/password`。默认示例用户名是 `s01`。

```bash
SERVER=127.0.0.1 USER=s01 PASSWORD=USER_DEFAULT_PASSWORD \
  docker compose -f docker-compose-client.yml up -d --force-recreate
```

注意：`USER` 是 docker compose 常见的宿主机环境变量名。如果没有在运行命令里显式传递，或传递方式不正确，Compose 可能会把系统里的 `$USER` 解析成本机用户名，而不是默认 `s01`。推荐优先级：

1. 运行命令显式传递 `USER=...`
2. 用户修改 `docker-compose-client.yml` 里的 `USER` 默认值
3. 依赖 Docker/系统环境

### 客户端 Docker Run

```bash
docker run -d --restart=always --name=serverstatus-client \
  --network=host --pid=host \
  -e SERVER=127.0.0.1 \
  -e USER=s01 \
  -e PASSWORD=USER_DEFAULT_PASSWORD \
  cppla/serverstatus:client
```

### 客户端环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `SERVER` | `127.0.0.1` | 服务端地址 |
| `USER` | `s01` | 客户端用户名，必须匹配服务端配置 |
| `PORT` | `35601` | 服务端 sergate 端口 |
| `PASSWORD` | `USER_DEFAULT_PASSWORD` | 客户端密码，必须匹配服务端配置 |
| `INTERVAL` | `1` | 上报间隔 |
| `PROBEPORT` | `80` | 探测端口 |
| `PROBE_PROTOCOL_PREFER` | `ipv4` | 探测协议偏好 |
| `PING_PACKET_HISTORY_LEN` | `100` | Ping 历史长度 |
| `CU` | `cu.tz.cloudcpp.com` | 联通探测地址 |
| `CT` | `ct.tz.cloudcpp.com` | 电信探测地址 |
| `CM` | `cm.tz.cloudcpp.com` | 移动探测地址 |
| `CLIENT` | `psutil` | 客户端实现，可选 `psutil` 或 `linux` |

## WebUI 配置管理

WebUI 顶部保留四个入口：

- `主机`：节点列表、筛选、排序和右侧详情。
- `服务`：服务监测结果。
- `证书`：SSL 证书检查结果。
- `配置`：编辑 `servers`、`monitors`、`sslcerts`、`watchdog`。

使用配置页前需要：

1. 服务端设置 `ADMIN_TOKEN`。
2. 打开 http://127.0.0.1:8080/。
3. 进入「配置」页，输入 `ADMIN_TOKEN`。
4. 保存后服务端会写入 `config.json` 并向 `sergate` 发送 `SIGHUP` 重载。

如果你使用 Docker 单文件 bind mount 挂载 `config.json`，管理 API 会自动兼容 Docker 的 `EBUSY` 写入限制：先备份，再原地写回。

## HTTP 管理 API

Docker 服务端镜像已内置 HTTP API，并通过 nginx 暴露在 Web 端口下。源码手动运行时需要单独启动 `server/manage_api.py`；如果要让 WebUI 的「配置」页可用，还需要把 `/api/` 反代到 `manage_api.py`。认证方式：

```bash
Authorization: Bearer your-strong-token
```

无需认证的接口：

```bash
curl http://127.0.0.1:8080/api/health
curl http://127.0.0.1:8080/api/schema
```

读取完整配置：

```bash
curl -H 'Authorization: Bearer your-strong-token' \
  http://127.0.0.1:8080/api/config
```

节点 CRUD：

```bash
curl -X POST http://127.0.0.1:8080/api/servers \
  -H 'Authorization: Bearer your-strong-token' \
  -H 'Content-Type: application/json' \
  -d '{"username":"s05","name":"node5","type":"kvm","host":"host5","location":"SG","password":"USER_DEFAULT_PASSWORD","monthstart":1}'

curl -X PUT http://127.0.0.1:8080/api/servers/s05 \
  -H 'Authorization: Bearer your-strong-token' \
  -H 'Content-Type: application/json' \
  -d '{"username":"s05","name":"node5-new","type":"kvm","host":"host5","location":"SG","password":"USER_DEFAULT_PASSWORD","monthstart":1}'

curl -X DELETE -H 'Authorization: Bearer your-strong-token' \
  http://127.0.0.1:8080/api/servers/s05
```

`monitors`、`sslcerts`、`watchdog` 也支持细粒度 CRUD。更新和删除可以用数字 `index`；如果 `name` 唯一，也可以用 URL 编码后的 `name`。

```bash
curl -X POST http://127.0.0.1:8080/api/monitors \
  -H 'Authorization: Bearer your-strong-token' \
  -H 'Content-Type: application/json' \
  -d '{"name":"demo","host":"https://example.com","type":"https","interval":600}'

curl -X PUT http://127.0.0.1:8080/api/sslcerts/0 \
  -H 'Authorization: Bearer your-strong-token' \
  -H 'Content-Type: application/json' \
  -d '{"name":"example","domain":"https://example.com","port":443,"interval":7200,"callback":"https://yourSMSurl"}'

curl -X DELETE -H 'Authorization: Bearer your-strong-token' \
  http://127.0.0.1:8080/api/watchdog/0
```

重载配置或重启 `sergate`：

```bash
curl -X POST -H 'Authorization: Bearer your-strong-token' \
  http://127.0.0.1:8080/api/reload

curl -X POST -H 'Authorization: Bearer your-strong-token' \
  http://127.0.0.1:8080/api/restart
```

完整端点以 `/api/schema` 输出为准。

## 配置文件

配置文件默认路径：

- Docker 服务端：`/ServerStatus/server/config.json`
- 源码运行：`server/config.json`，或通过 `--config` 指定

基础示例：

```json
{
  "servers": [
    {
      "username": "s01",
      "name": "node1",
      "type": "kvm",
      "host": "host1",
      "location": "CN",
      "password": "USER_DEFAULT_PASSWORD",
      "monthstart": 1
    }
  ],
  "monitors": [
    {
      "name": "example",
      "host": "https://example.com",
      "interval": 600,
      "type": "https"
    }
  ],
  "sslcerts": [
    {
      "name": "example",
      "domain": "https://example.com",
      "port": 443,
      "interval": 7200,
      "callback": "https://yourSMSurl"
    }
  ],
  "watchdog": [
    {
      "name": "offline warning",
      "rule": "online4=0&online6=0",
      "interval": 600,
      "callback": "https://yourSMSurl"
    },
    {
      "name": "cpu high warning",
      "rule": "cpu>90&load_1>5&username!='s01'",
      "interval": 600,
      "callback": "https://yourSMSurl"
    }
  ]
}
```

常见 Watchdog 回调：

```text
Telegram: https://api.telegram.org/bot你的密钥/sendMessage?parse_mode=HTML&disable_web_page_preview=true&chat_id=你的标识&text=
Server酱: https://sctapi.ftqq.com/你的密钥.send?title=ServerStatus&desp=
PushDeer: https://api2.pushdeer.com/message/push?pushkey=你的密钥&text=
HttpBasicAuth: https://用户名:密码@你的域名/api/push?message=
```

## 源码编译和运行

服务端依赖：

```bash
# Debian/Ubuntu
apt-get -y install gcc g++ make libcurl4-openssl-dev python3 nginx openssl

# CentOS/RedHat
yum -y install gcc gcc-c++ make libcurl-devel python3 nginx openssl
```

编译并运行 `sergate`：

```bash
cd ServerStatus/server
make
mkdir -p ../web/json
./sergate --config=config.json --web-dir=../web &
echo $! > /tmp/serverstatus-sergate.pid
```

如果只需要 HTTP API，可以再启动 `manage_api.py`：

```bash
cd ServerStatus/server
ADMIN_TOKEN='your-strong-token' \
CONFIG_PATH="$PWD/config.json" \
SERGATE_PID_FILE=/tmp/serverstatus-sergate.pid \
ADMIN_API_BIND=127.0.0.1 \
ADMIN_API_PORT=35602 \
python3 manage_api.py
```

源码方式直连 API：

```bash
curl http://127.0.0.1:35602/api/health
curl -H 'Authorization: Bearer your-strong-token' \
  http://127.0.0.1:35602/api/config
```

如果要通过 WebUI 使用「配置」页，需要 nginx 同时提供静态文件并反代 `/api/`。示例配置：

```nginx
server {
    listen 8080;
    server_name _;

    root /path/to/ServerStatus/web;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }

    location /json/ {
        add_header Cache-Control "no-store";
        try_files $uri =404;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:35602;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 30s;
    }
}
```

启动失败时优先检查端口占用：`35601` 是 `sergate` 客户端上报端口，`35602` 是源码方式的管理 API 端口，`8080` 是上面示例的 Web 端口。源码手动运行没有 Docker 入口脚本守护；调用 `/api/restart` 会向 `sergate` 发送 `SIGTERM`，需要你用 systemd、supervisor 或 shell 循环自行拉起。

## 客户端源码运行

`client-linux.py`：

```bash
wget --no-check-certificate -qO client-linux.py \
  https://raw.githubusercontent.com/cppla/ServerStatus/master/clients/client-linux.py

nohup python3 client-linux.py SERVER=127.0.0.1 USER=s01 PASSWORD=USER_DEFAULT_PASSWORD \
  >/dev/null 2>&1 &
```

`client-psutil.py`：

```bash
# Debian/Ubuntu
apt -y install python3-psutil

# CentOS/RedHat
yum -y install python3-pip gcc python3-devel
pip3 install psutil

python3 clients/client-psutil.py SERVER=127.0.0.1 USER=s01 PASSWORD=USER_DEFAULT_PASSWORD
```

后台运行与开机启动：

```bash
nohup python3 client-linux.py SERVER=127.0.0.1 USER=s01 PASSWORD=USER_DEFAULT_PASSWORD &

# crontab -e
@reboot /usr/bin/python3 /path/to/client-linux.py SERVER=127.0.0.1 USER=s01 PASSWORD=USER_DEFAULT_PASSWORD
```

## 本地构建镜像

```bash
docker build -f Dockerfile.server -t cppla/serverstatus:server .
docker build -f Dockerfile.client -t cppla/serverstatus:client .
```

## Make Better

* BotoX：https://github.com/BotoX/ServerStatus
* mojeda：https://github.com/mojeda
* mojeda's ServerStatus：https://github.com/mojeda/ServerStatus
* BlueVM's project：http://www.lowendtalk.com/discussion/comment/169690#Comment_169690
