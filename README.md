## ServerStatus 中文版

一个轻量、可扩展的多服务器状态/探针面板，支持：系统资源统计、站点/端口监控、Watchdog 表达式告警、SSL 证书有效期与域名匹配监控（新增）。

在线演示：https://tz.cloudcpp.com

[![Python Support](https://img.shields.io/badge/python-3.6%2B%20-blue.svg)](https://github.com/cppla/ServerStatus)
[![C++ Compiler](http://img.shields.io/badge/C++-GNU-blue.svg?style=flat&logo=cplusplus)](https://github.com/cppla/ServerStatus)
[![License](https://img.shields.io/badge/license-MIT-4EB1BA.svg?style=flat-square)](https://github.com/cppla/ServerStatus)
[![Version](https://img.shields.io/badge/Version-Build%201.1.5-red)](https://github.com/cppla/ServerStatus)

![Latest Host Version](https://dl.cpp.la/Archive/serverstatus_1.1.5.png)

> Watchdog 采用“触发式”告警，interval 用于抑制频率，并非采样周期。Exprtk 库当前使用窄字符，含中文的表达式暂不支持（待改进）。

### 目录结构

- `clients/` 客户端脚本
- `server/`  服务端核心 (C/C++)
- `web/`     前端展示与静态资源
- `server/config.json` 主配置文件
- `web/json/` 持久化月流量快照

### 新特性：SSL 证书监控 (sslcerts)
支持：
1. 自动解析证书到期时间 (notAfter)
2. 阈值到期提醒：7 / 3 / 1 天（分别独立冷却 20h）
3. 证书域名匹配校验（解析 SAN DNS 列表与 Subject CN，支持通配符 *.example.com）；域名与证书不匹配发送独立告警（默认 24h 冷却）
4. 前端单独 “SSL” 面板，区分：正常 / 将到期 / 紧急 / 已过期 / 域名不匹配

配置段示例：
```jsonc
"sslcerts": [
	{
		"name": "my.cloudcpp.com",     // 自定义显示名称
		"domain": "https://my.cloudcpp.com", // 可含 http(s) 前缀/路径，会自动规范化
		"port": 443,                    // TLS 端口
		"interval": 600,                // 拉取间隔(秒)，每条证书独立
		"callback": "https://yourSMSurl" // 告警回调(URL + 文本拼接)，POST body: signature=ServerStatusSSL
	}
]
```
JSON 输出(sslcerts) 字段：
```json
{
	"name": "my.cloudcpp.com",
	"domain": "https://my.cloudcpp.com",
	"port": 443,
	"expire_ts": 1723507200,
	"expire_days": 14,
	"mismatch": false
}
```
告警触发逻辑：
| 类型 | 条件 | 冷却 | 文本示例 |
|------|------|------|----------|
| 7 天提醒 | 0 < days ≤ 7 且 >3 | 20h | 【SSL证书提醒】name(domain) 将在 7 天后(...) 到期 |
| 3 天提醒 | 0 < days ≤ 3 且 >1 | 20h | 同上 target=3 |
| 1 天提醒 | 0 < days ≤ 1 | 20h | target=1 |
| 已过期 | days ≤ 0 | 不发送额外（前端显示已过期，可自行扩展） |
| 域名不匹配 | mismatch=true | 24h | 【SSL证书域名不匹配】name(domain) 证书域名与配置不一致 |

注意事项：
- `domain` 可带协议/路径/端口，程序会提取主机名并使用 `-servername` 及 `-connect host:port` 方式获取证书。
- 若站点使用 SNI，请确保 `port` 与 SNI 主机匹配；未解析到 SAN/CN 时不立即判定不匹配（避免误报）。
- 回调是 `POST`，内容直接拼接在 callback URL 后，必要时自行 URLDecode。
- mismatch 与到期提醒互不影响，可同时存在（前端优先显示“已过期” 其次“不匹配”）。

---

## 部署

【服务端】：
```bash

`Docker`:     

wget --no-check-certificate -qO ~/serverstatus-config.json https://raw.githubusercontent.com/cppla/ServerStatus/master/server/config.json && mkdir ~/serverstatus-monthtraffic    
docker run -d --restart=always --name=serverstatus -v ~/serverstatus-config.json:/ServerStatus/server/config.json -v ~/serverstatus-monthtraffic:/usr/share/nginx/html/json -p 80:80 -p 35601:35601 cppla/serverstatus:latest     

`Docker-compose(推荐)`: docker-compose up -d
```

【客户端】：
```bash
wget --no-check-certificate -qO client-linux.py 'https://raw.githubusercontent.com/cppla/ServerStatus/master/clients/client-linux.py' && nohup python3 client-linux.py SERVER={$SERVER} USER={$USER} PASSWORD={$PASSWORD} >/dev/null 2>&1 &

eg:
wget --no-check-certificate -qO client-linux.py 'https://raw.githubusercontent.com/cppla/ServerStatus/master/clients/client-linux.py' && nohup python3 client-linux.py SERVER=45.79.67.132 USER=s04  >/dev/null 2>&1 &
```

## 主题

* layui：https://github.com/zeyudada/StatusServerLayui ，预览：https://sslt.8zyw.cn            
<img src=https://dl.cpp.la/Archive/serverstatus_layui.png width=200 height=100 />

* light：https://github.com/orilights/ServerStatus-Theme-Light ，预览：https://tz.cloudcpp.com/index3.html    
<img src=https://dl.cpp.la/Archive/serverstatus_light.png width=200 height=100 />  


## 手动安装
   
**【服务端配置】**           
          
#### 一、生成服务端程序              
```
`Debian/Ubuntu`: apt-get -y install gcc g++ make libcurl4-openssl-dev
`Centos/Redhat`: yum -y install gcc gcc-c++ make libcurl-devel

cd ServerStatus/server && make
./sergate
```
如果没错误提示，OK，ctrl+c关闭；如果有错误提示，检查35601端口是否被占用    

#### 二、修改配置文件 (含 sslcerts 示例)
```diff
! watchdog rule 可以为任何已知字段的表达式。注意Exprtk库默认使用窄字符类型，中文等Unicode字符无法解析计算，等待修复       
! watchdog interval 最小通知间隔
! watchdog callback 可自定义为Post方法的URL，告警内容将拼接其后并发起回调    

! Telegram: https://api.telegram.org/bot你自己的密钥/sendMessage?parse_mode=HTML&disable_web_page_preview=true&chat_id=你自己的标识&text=
! Server酱: https://sctapi.ftqq.com/你自己的密钥.send?title=ServerStatus&desp=
! PushDeer: https://api2.pushdeer.com/message/push?pushkey=你自己的密钥&text=
! HttpBasicAuth: https://用户名:密码@你自己的域名/api/push?message=
```

```
{
	"servers":
	[
		{
			"username": "s01",
			"name": "vps-1",
			"type": "kvm",
			"host": "chengdu",
			"location": "🇨🇳",
			"password": "USER_DEFAULT_PASSWORD",
			"monthstart": 1
		}
	],
	"monitors": [
		{
			"name": "监测网站，默认为一天在线率",
			"host": "https://www.baidu.com",
			"interval": 1200,
			"type": "https"
		},
		{
			"name": "监测tcp服务端口",
			"host": "1.1.1.1:80",
			"interval": 1200,
			"type": "tcp"
		}
	],
	"sslcerts": [
		{
			"name": "demo域名",
			"domain": "https://demo.example.com/path",
			"port": 443,
			"interval": 600,
			"callback": "https://yourSMSurl"
		}
	],
	"watchdog":
	[
	        {
			"name": "服务器负载高监控，排除内存大于32G物理机，同时排除node1机器",
			"rule": "cpu>90&load_1>4&memory_total<33554432&name!='node1'",
			"interval": 600,
			"callback": "https://yourSMSurl"
		},
		{
                        "name": "服务器内存使用率过高监控，排除小于1G的机器",
                        "rule": "(memory_used/memory_total)*100>90&memory_total>1048576",
                        "interval": 600,
                        "callback": "https://yourSMSurl"
                },
                {
                        "name": "服务器宕机告警",
                        "rule": "online4=0&online6=0",
                        "interval": 600,
                        "callback": "https://yourSMSurl"
                },
		{
                        "name": "DDOS和CC攻击监控，限制甲骨文机器",
                        "rule": "tcp_count>600&type='Oracle'",
                        "interval": 300,
                        "callback": "https://yourSMSurl"
                },
		{
			"name": "服务器月出口流量999GB告警",
			"rule": "(network_out-last_network_out)/1024/1024/1024>999",
			"interval": 3600,
			"callback": "https://yourSMSurl"
		},
		{
			"name": "阿里云服务器流量18GB告警,限制username为乌兰察布",
			"rule": "(network_out-last_network_out)/1024/1024/1024>18&(username='wlcb1'|username='wlcb2'|username='wlcb3'|username='wlcb4')",
			"interval": 3600,
			"callback": "https://yourSMSurl"
		},
		{
			"name": "重要线路丢包率过高检查",
			"rule": "(ping_10010>10|ping_189>10|ping_10086>10)&(host='sgp'|host='qqhk'|host='hk-21-x'|host='hk-31-x')",
			"interval": 600,
			"callback": "https://yourSMSurl"
		},
		{
			"name": "你可以组合任何已知字段的表达式",
			"rule": "(hdd_used/hdd_total)*100>95",
			"interval": 1800,
			"callback": "https://yourSMSurl"
		}
	]
}       
```

#### 三、拷贝前端到 Web 根目录
例如：
```
sudo cp -r ServerStatus/web/* /home/wwwroot/default
```

#### 四、运行服务端
web-dir参数为上一步设置的网站根目录，务必修改成自己网站的路径   
```
./sergate --config=config.json --web-dir=/home/wwwroot/default   
```

**客户端配置**

客户端有两个版本，client-linux为普通linux，client-psutil为跨平台版，普通版不成功，换成跨平台版即可。        

#### 一、client-linux版配置：       
1、vim client-linux.py, 修改SERVER地址，username帐号， password密码        
2、python3 client-linux.py 运行即可。      

#### 二、client-psutil版配置:                
1、安装psutil跨平台依赖库       
```
`Debian/Ubuntu`: apt -y install python3-pip && pip3 install psutil    
`Centos/Redhat`: yum -y install python3-pip gcc python3-devel && pip3 install psutil      
`Windows`: https://pypi.org/project/psutil/    
```
2、vim client-psutil.py, 修改SERVER地址，username帐号， password密码       
3、python3 client-psutil.py 运行即可。    

服务器和客户端自行加入开机启动，或进程守护，或后台方式运行。 例如： nohup python3 client-linux.py &    

`extra scene (run web/ssview.py)`
![Shell View](https://dl.cpp.la/Archive/serverstatus-shell.png?version=2023)


## Make Better

* BotoX：https://github.com/BotoX/ServerStatus
* mojeda: https://github.com/mojeda 
* mojeda's ServerStatus: https://github.com/mojeda/ServerStatus
* BlueVM's project: http://www.lowendtalk.com/discussion/comment/169690#Comment_169690

## Jetbrains

<a href="https://www.jetbrains.com/?from=ServerStatus"><img src="https://resources.jetbrains.com/storage/products/company/brand/logos/jb_square.png" width="100px"></a>
