# ServerStatus 客户端 go 语言版

懒得改项目名了，需要的话自己改。

### 使用方法
可选：更改源码中的连接信息
```
# go env -w  GOPROXY=https://goproxy.cn,direct (速度慢可以换源）
go build .
./awesomeProject
# 更改链接参数
./awesomeProject --SERVER=127.0.0.1 --PORT=35601 --USER=test --PASSWORD=test1234 --INTERVAL=1
```

### TODO

- 外网连接状态
- 三网丢包计算
- 三网延迟计算
- 其他算法优化

### 鸣谢

 - https://github.com/cppla/ServerStatus
 - https://github.com/bitcav/nitr-core
 - https://github.com/shirou/gopsutil
 - https://github.com/json-iterator/go
