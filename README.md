# ServerStatus中文版：   

* ServerStatus中文版是一个酷炫高逼格的云探针、云监控、服务器云监控、多服务器探针~。
* 在线演示：https://tz.cloudcpp.com    

# 目录介绍：

* autodeploy    自动部署脚本
* clients       客户端文件
* server        服务端文件
* web           网站文件  

# 更新说明：

* 20180626, 增加客户端自动部署脚本，代码:https://github.com/dovela/ServerStatus1Click
* 20180314, 调整前端，置默认密码为，设置ip和user即可上线　　　　　　
* 20180312, 加入失联(被照顾)检测【正常：MH361, 屏蔽：MH370】，校准虚拟化(container)流量统计异常　　　　　　
* 20170807, 更新平均1，5，15负载
* 20170607, 去掉无用的IPV6信息，增加服务器总流量监控                           

# 自动部署：

【服务端】：暂无

【客户端】：
```bash
wget -N --no-check-certificate https://github.com/cppla/ServerStatus/blob/master/autodeploy/client-auto.sh
chmod +x client-auto.sh
./client-auto.sh
```

# 手动安装教程：     
   
【克隆代码】:
```
git clone https://github.com/cppla/ServerStatus.git
```

【服务端配置】（服务端程序在ServerStatus/web下）:  
          
一、生成服务端程序              
```
cd ServerStatus/server
make
./sergate
```
如果没错误提示，OK，ctrl+c关闭；如果有错误提示，检查35601端口是否被占用    

二、修改配置文件         
修改config.json文件，注意username, password的值需要和客户端对应一致                 
```
{"servers":
	[
		{
			"username": "s01",
			"name": "Mainserver 1",
			"type": "Dedicated Server",
			"host": "GenericServerHost123",
			"location": "Austria",
			"password": "some-hard-to-guess-copy-paste-password"
		},
	]
}       
```

三、拷贝ServerStatus/status到你的网站目录        
例如：
```
sudo cp -r ServerStatus/web/* /home/wwwroot/default
```

四、运行服务端：             
web-dir参数为上一步设置的网站根目录，务必修改成自己网站的路径   
```
./sergate --config=config.json --web-dir=/home/wwwroot/default   
```

【客户端配置】（客户端程序在ServerStatus/clients下）：          
客户端有两个版本，client-linux为普通linux，client-psutil为跨平台版，普通版不成功，换成跨平台版即可。        

一、client-linux版配置：       
1、vim client-linux.py, 修改SERVER地址，username帐号， password密码        
2、python client-linux.py 运行即可。      

二、client-psutil版配置:                
1、安装psutil跨平台依赖库      
2、vim client-psutil.py, 修改SERVER地址，username帐号， password密码       
3、python client-psutil.py 运行即可。           
```
### for Centos：
sudo yum -y install epel-release
sudo yum -y install python-pip
sudo yum clean all
sudo yum -y install gcc
sudo yum -y install python-devel
sudo pip install psutil
### for Ubuntu/Debian:
sudo root
apt-get -y install python-setuptools python-dev build-essential
apt-get -y install python-pip
pip install psutil
### for Windows:
打开网址：https://pypi.python.org/pypi?:action=display&name=psutil#downloads
下载psutil for windows程序包
安装即可
```

打开云探针页面，就可以正常的监控。接下来把服务器和客户端脚本自行加入开机启动，或者进程守护，或以后台方式运行即可！例如： nohup python client-linux.py &      

# 为什么会有ServerStatus中文版：

* 有些功能确实没用
* 原版本部署，英文说明复杂
* 不符合中文版的习惯
* 没有一次又一次的轮子，哪来如此优秀的云探针

# 相关开源项目，感谢： 

* ServerStatus：https://github.com/BotoX/ServerStatus
* mojeda: https://github.com/mojeda 
* mojeda's ServerStatus: https://github.com/mojeda/ServerStatus
* BlueVM's project: http://www.lowendtalk.com/discussion/comment/169690#Comment_169690
