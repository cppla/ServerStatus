# ServerStatus中文版：   

* ServerStatus中文版是一个酷炫高逼格的云探针、云监控、服务器云监控、多服务器探针~。
* 在线演示：https://tz.cloudcpp.com    

[![Python Support](https://img.shields.io/badge/python-2.7%2B%20-blue.svg)](https://github.com/cppla/ServerStatus)
[![C++ Compiler](http://img.shields.io/badge/C++-GNU-blue.svg?style=flat&logo=cplusplus)](https://github.com/cppla/ServerStatus)
[![License](https://img.shields.io/badge/license-MIT-4EB1BA.svg?style=flat-square)](https://github.com/cppla/ServerStatus)
[![Version](https://img.shields.io/badge/Version-Beta%201.0.9-red)](https://github.com/cppla/ServerStatus)

![Latest Version](http://dl.cpp.la/Archive/serverstatus-latest.png)

# 目录介绍：

* clients       	客户端文件
* server       	 	服务端文件  
* web           	网站文件

* server/config.json	探针配置文件                                
* web/json      	探针月流量        

# 自动部署：

【服务端】：
```bash

`Docker`:     

wget --no-check-certificate -qO ~/serverstatus-config.json https://raw.githubusercontent.com/cppla/ServerStatus/master/server/config.json && mkdir ~/serverstatus-monthtraffic    
docker run -d --restart=always --name=serverstatus -v ~/serverstatus-config.json:/ServerStatus/server/config.json -v ~/serverstatus-monthtraffic:/usr/share/nginx/html/json -p 80:80 -p 35601:35601 cppla/serverstatus:latest     

`Docker-compose`: docker-compose up -d
```

【客户端】：
```bash
wget --no-check-certificate -qO client-linux.py 'https://raw.githubusercontent.com/cppla/ServerStatus/master/clients/client-linux.py' && nohup python3 client-linux.py SERVER={$SERVER} USER={$USER} PASSWORD={$PASSWORD} >/dev/null 2>&1 &

eg:
wget --no-check-certificate -qO client-linux.py 'https://raw.githubusercontent.com/cppla/ServerStatus/master/clients/client-linux.py' && nohup python3 client-linux.py SERVER=45.79.67.132 USER=s04  >/dev/null 2>&1 &
```

# 手动安装教程：     
   
【克隆代码】:
```
git clone https://github.com/cppla/ServerStatus.git
```

##【服务端配置】:   
          
#### 一、生成服务端程序              
```
`Debian/Ubuntu`: apt-get -y install gcc g++ make libcurl4-openssl-dev
`Centos/Redhat`: yum -y install gcc gcc-c++ make libcurl-devel

cd ServerStatus/server && make
./sergate
```
如果没错误提示，OK，ctrl+c关闭；如果有错误提示，检查35601端口是否被占用    

#### 二、修改配置文件         
修改config.json文件，注意username, password的值需要和客户端对应一致。watchdog规则可以为任何已知字段的表达式                 
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
		},
	],
	"watchdog":
	[
	    {
			"name": "服务器负载高监控",
			"rule": "load_5>10",
			"interval": 1200,
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

##【客户端配置】：           
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

打开云探针页面，就可以正常的监控。接下来把服务器和客户端脚本自行加入开机启动，或者进程守护，或以后台方式运行即可！例如： nohup python3 client-linux.py &  

`extra scene (run web/ssview.py)`
![Shell View](http://dl.cpp.la/Archive/serverstatus-shell.png)


# Make Better        

* BotoX：https://github.com/BotoX/ServerStatus
* mojeda: https://github.com/mojeda 
* mojeda's ServerStatus: https://github.com/mojeda/ServerStatus
* BlueVM's project: http://www.lowendtalk.com/discussion/comment/169690#Comment_169690

# Jetbrains    

<a href="https://www.jetbrains.com/?from=ServerStatus"><img src="https://resources.jetbrains.com/storage/products/company/brand/logos/jb_square.png" width="100px"></a>
