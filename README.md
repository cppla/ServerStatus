# ServerStatus中文版：   

* ServerStatus中文版是一个酷炫高逼格的云探针、云监控、服务器云监控、多服务器探针~。
* 在线演示：https://tz.cloudcpp.com    

[![Python Support](https://img.shields.io/badge/python-3.6%2B%20-blue.svg)](https://github.com/cppla/ServerStatus)
[![C++ Compiler](http://img.shields.io/badge/C++-GNU-blue.svg?style=flat&logo=cplusplus)](https://github.com/cppla/ServerStatus)
[![License](https://img.shields.io/badge/license-MIT-4EB1BA.svg?style=flat-square)](https://github.com/cppla/ServerStatus)
[![Version](https://img.shields.io/badge/Version-Build%201.1.7-red)](https://github.com/cppla/ServerStatus)

![Latest Host Version](https://dl.cpp.la/Archive/serverstatus_1_1_7.png)

`Watchdog触发式告警，interval只是为了防止频繁收到报警，并不是探测间隔。值得注意的是Exprtk使用窄字符类型，中文等Unicode字符无法解析计算。 AI已经能够取代大部分程序员`    
    

# 部署：

【服务端】：
```bash

`Docker`:

wget --no-check-certificate -qO ~/serverstatus-config.json https://raw.githubusercontent.com/cppla/ServerStatus/master/server/config.json && mkdir ~/serverstatus-monthtraffic    
docker run -d --restart=always --name=serverstatus-server -v ~/serverstatus-config.json:/ServerStatus/server/config.json -v ~/serverstatus-monthtraffic:/usr/share/nginx/html/json -p 80:80 -p 35601:35601 cppla/serverstatus:server     

`Docker-compose`: 
docker compose -f docker-compose-server.yml up -d

`Dockerfile.server`(本地构建服务端):
docker build -f Dockerfile.server -t serverstatus-server .
```

【客户端】：
```bash

`Shell`:

wget --no-check-certificate -qO client-linux.py 'https://raw.githubusercontent.com/cppla/ServerStatus/master/clients/client-linux.py' && nohup python3 client-linux.py SERVER={$SERVER} USER={$USER} >/dev/null 2>&1 &  

`Docker`: 
docker run --restart=always --name=serverstatus-client --network=host -e SERVER=127.0.0.1 -e USER=s01 cppla/serverstatus:client

`Docker-compose`: 
docker compose -f docker-compose-client.yml up -d

`Dockerfile.client`(本地构建客户端):
docker build -f Dockerfile.client -t serverstatus-client .

`docker环境变量`: 
SERVER --- 可选 - 默认 127.0.0.1
USER --- 可选 - 默认 s01
PORT --- 可选 - 默认 35601
PASSWORD --- 可选 - 默认 USER_DEFAULT_PASSWORD
INTERVAL --- 可选 - 默认 1
PROBEPORT --- 可选 - 默认 80
PROBE_PROTOCOL_PREFER --- 可选 - 默认 ipv4
PING_PACKET_HISTORY_LEN --- 可选 - 默认 100
CU --- 可选 - 默认 cu.tz.cloudcpp.com
CT --- 可选 - 默认 ct.tz.cloudcpp.com
CM --- 可选 - 默认 cm.tz.cloudcpp.com
CLIENT --- 可选 - 默认client, psutil可选
```

# 教程：     
   
**【服务端配置】**           
          
#### 一、生成服务端程序              
```
`Debian/Ubuntu`: apt-get -y install gcc g++ make libcurl4-openssl-dev
`Centos/Redhat`: yum -y install gcc gcc-c++ make libcurl-devel

cd ServerStatus/server && make
./sergate
```
如果没错误提示，OK，ctrl+c关闭；如果有错误提示，检查35601端口是否被占用    

#### 二、修改配置文件         
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
			"name": "抖音",
			"host": "https://www.douyin.com",
			"interval": 600,
			"type": "https"
		},
		{
			"name": "百度",
			"host": "https://www.baidu.com",
			"interval": 600,
			"type": "https"
		}
	],
	"sslcerts": [
		{
			"name": "demo域名",
			"domain": "https://demo.example.com",
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

#### 三、拷贝ServerStatus/status到你的网站目录        
例如：
```
sudo cp -r ServerStatus/web/* /home/wwwroot/default
```

#### 四、运行服务端：             
web-dir参数为上一步设置的网站根目录，务必修改成自己网站的路径   
```
./sergate --config=config.json --web-dir=/home/wwwroot/default   
```

**【客户端配置】**    
    
#### client-linux.py Linux版
```bash
# 1、修改 client-linux.py 中的 SERVER、username、password
python3 client-linux.py
# 2、以传参的方式启动
python3 client-linux.py SERVER=127.0.0.1 USER=s01

```

#### client-psutil.py 跨平台版
```bash
# 安装依赖
# Debian/Ubuntu
apt -y install python3-psutil
# Centos/Redhat
yum -y install python3-pip gcc python3-devel && pip3 install psutil
# Windows: 从 https://pypi.org/project/psutil/ 安装
```

#### 后台运行与开机启动
```bash
# 后台运行
nohup python3 client-linux.py &

# 开机启动 (crontab -e)
@reboot /usr/bin/python3 /path/to/client-linux.py
```

# Make Better        

* BotoX：https://github.com/BotoX/ServerStatus
* mojeda: https://github.com/mojeda 
* mojeda's ServerStatus: https://github.com/mojeda/ServerStatus
* BlueVM's project: http://www.lowendtalk.com/discussion/comment/169690#Comment_169690
