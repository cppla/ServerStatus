# ServerStatus1Click

## 说明
客户端源码fork自@cppla，一键脚本自制
<br>ServerStatus客户端的一键安装脚本，只需输入必要数据即可全自动安装并启动，脚本只需运行一次。
<br>脚本在centos7上测试通过，其他发行版未测试。
<br>理论上其他版本linux也可使用，请测试通过后提issue，谢谢。
<br>欢迎star、fork，本脚本不定时更新，不定时从@cppla处合并新版本ServerStatus源码。
<br>不支持window各版本

## 使用教程
````
wget -N --no-check-certificate https://raw.githubusercontent.com/dovela/ServerStatus/master/ServerStatus1Click.sh
chmod +x ServerStatus1Click.sh
./ServerStatus1Click.sh
````

## 其他
默认远程端口35601，安装依赖选项为安装git、lsof、psutil，建议执行一次

## 特别感谢
* cppla's ServerStatus: https://github.com/cppla/ServerStatus
* ServerStatus: https://github.com/BotoX/ServerStatus
* mojeda: https://github.com/mojeda 
* mojeda's ServerStatus: https://github.com/mojeda/ServerStatus
* BlueVM's project: http://www.lowendtalk.com/discussion/comment/169690#Comment_169690
* 参考了ToyoDAdoubi的一键脚本: https://github.com/ToyoDAdoubi/doubi
