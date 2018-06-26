#!/bin/bash

#=================================================
#	System Required: CentOS 6+/Debian 6+/Ubuntu 14.04+/others(test)
#	Description: Auto-install the ServerStatus Client
#	Version: 1.0.2
#	Author: dovela
#=================================================

check_sys(){
	 if [[ -f /etc/redhat-release ]]; then
		release="centos"
	 elif cat /etc/issue | grep -q -E -i "debian"; then
		release="debian"
	 elif cat /etc/issue | grep -q -E -i "ubuntu"; then
		release="ubuntu"
	 elif cat /etc/issue | grep -q -E -i "centos|red hat|redhat"; then
		release="centos"
	 elif cat /proc/version | grep -q -E -i "debian"; then
		release="debian"
	 elif cat /proc/version | grep -q -E -i "ubuntu"; then
		release="ubuntu"
	 elif cat /proc/version | grep -q -E -i "centos|red hat|redhat"; then
		release="centos"
         fi
	bit=`uname -m`
}

centos_yum(){
	yum install -y epel-release && yum clean all && yum update
        yum install -y git lsof python-pip gcc python-devel
}

debian_apt(){
	apt-get install -y git lsof python-setuptools python-dev build-essential python-pip
}

install_sss(){
   	 clear
	stty erase '^H' && read -p " 服务端地址:" sserver
	stty erase '^H' && read -p " 远程端口（默认35601）:" sport
	[[ -z ${sport} ]] && sport="35601"
  	stty erase '^H' && read -p " 客户端username:" suser
  	stty erase '^H' && read -p " 客户端password:" spasswd
   	 clear
  	git clone https://github.com/dovela/ServerStatus.git
  	rm -rf oneclick.sh
  	echo 'ServerStatus客户端安装完成'
         cd ServerStatus/clients
  	sed -i -e "s/sserver/$sserver/g" client-linux.py
	sed -i -e "s/sport/$sport/g" client-linux.py
  	sed -i -e "s/suser/$suser/g" client-linux.py
  	sed -i -e "s/spasswd/$spasswd/g" client-linux.py
 	sed -i -e "s/sserver/$sserver/g" client-psutil.py
	sed -i -e "s/sport/$sport/g" client-psutil.py
  	sed -i -e "s/suser/$suser/g" client-psutil.py
  	sed -i -e "s/spasswd/$spasswd/g" client-psutil.py
	 clear
	echo ' ServerStatus客户端配置完成，请进行下一步'
	echo ' 1. 运行 client-linux'
	echo ' 2. 运行 client-psutil'
	stty erase '^H' && read -p " 请输入数字 [1-2]:" num
	 case "$num" in
 		1)
		run_linux
		;;
		2)
		run_psutil
		;;
esac
}

run_linux(){
	nohup python client-linux.py >> serverstatus.log 2>&1 &	
   	 cd ../..
  	echo 'ServerStatus-linux客户端已开始运行'
}

run_psutil(){
	nohup python client-psutil.py >> serverstatus.log 2>&1 &
	 cd ../..
  	echo 'ServerStatus-psutil客户端已开始运行'
}

stop_client(){
	kill -9 $(lsof -i:35601 |awk '{print $2}' | tail -n 1)
}

install_env(){
	 clear
	  if [[ ${release} == "centos" ]]; then
		centos_yum
	  else
		debian_apt
	  fi
	pip install --upgrade
	pip install psutil
	echo '依赖环境安装完成，请再次运行脚本'
}

 clear
check_sys
[ $(id -u) != "0" ] && echo -e "Error: You must be root to run this script" && exit 1
echo -e " 默认端口35601，出现问题请在 https://github.com/dovela/ServerStatus1Click 处提issue
————————————
  1.首次安装并启动 ServerStatus客户端
  2.运行 client_linux
  3.运行 client_psutil
  4.停止运行
  5.首次安装linux依赖，直接安装失败请执行
————————————
  输入数字开始，或ctrl + c退出
"
echo && stty erase '^H' && read -p " 请输入数字[1-5]:" num
 case "$num" in
 	1)
	install_sss
	;;
	2)
	cd ServerStatus/clients
	run_linux
	;;
	3)
	cd ServerStatus/clients
	run_psutil
	;;
	4)
	stop_client
	;;
	5)
	install_env
	;;
	*)
	echo -e "${Error} 请输入正确的数字 [1-5]!"
	;;
esac
