package main
/****************************
* 参考: https://github.com/cppla/ServerStatus/blob/master/clients/client-psutil.py
* Author: chn-student & cppla
* 依赖于 gopstil 跨平台库
* 编译版本:go 1.15.5
* 时间: 20201211
* 说明: 默认情况下修改server和user就可以了。丢包率监测方向可以自定义，例如：CU = "www.facebook.com"。
****************************/

import (
	"fmt"
	"github.com/json-iterator/go"
	"github.com/phachon/go-logger"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	SERVER string = "127.0.0.1"
	USER  string = "test"
	PORT int = 35601
	PASSWORD string = "123456"
	INTERVAL int = 1
	PORBEPORT int = 80
	NETWORKCHECK bool = true
	CU string = "cu.tz.cloudcpp.com" //120.52.99.224 河北联通
	CT string = "ct.tz.cloudcpp.com" //183.78.182.66 北京电信
	CM string = "cm.tz.cloudcpp.com" //211.139.145.129 广州移动
)

type Config struct {
	Server string `json:"server"`
	User string `json:"user"`
	Port int `json:"port"`
	Password string `json:"password"`
	Interval int `json:"interval"`
	Porbeport int `json:"porbeport"`
	Cu string `json:"cu"`
	Ct string `json:"cu"`
	Cm string `json:"cm"`
	NetworkCheck bool `json:"networkCheck"`
}

func NewConfig() Config {
	return Config{
		Server:    SERVER,
		User:      USER,
		Port:      PORT,
		Password:  PASSWORD,
		Interval:  INTERVAL,
		Porbeport: PORBEPORT,
		Cu:        CU,
		Ct:        CT,
		Cm:        CM,
		NetworkCheck: NETWORKCHECK,
	}
}

type ClientInfo struct {
	Load1 float64 `json:"load_1"`
	Load5 float64 `json:"load_5"`
	Load15 float64 `json:"load_15"`
	IpStatus bool `json:"ip_status"`
	Thread uint64 `json:"thread"`
	Process uint64 `json:"process"`
	NetworkTx uint64 `json:"network_tx"`
	NetworkRx uint64 `json:"network_rx"`
	NetworkIn uint64 `json:"network_in"`
	NetworkOut uint64 `json:"network_out"`
	Ping10010 float64 `json:"ping_10010"`
	Ping10086 float64 `json:"ping_10086"`
	Ping189 float64 `json:"ping_189"`
	Time10010 uint64 `json:"time_10010"`
	Time10086 uint64 `json:"time_10086"`
	Time189 uint64 `json:"time_189"`
	TCP uint64 `json:"tcp"`
	UDP uint64 `json:"udp"`
	CPU float64 `json:"cpu"`
	MemoryTotal uint64 `json:"memory_total"`
	MemoryUsed uint64 `json:"memory_used"`
	SwapTotal uint64 `json:"swap_total"`
	SwapUsed uint64 `json:"swap_used"`
	Uptime uint64 `json:"uptime"`
	HddTotal uint64 `json:"hdd_total"`
	HddUsed uint64`json:"hdd_used"`
}


func NewDefaultClientInfo() ClientInfo {
	return ClientInfo {
		Load1: 0.0,
		Load5: 0.0,
		Load15: 0.0,
		IpStatus: false,
		Thread: 0,
		Process: 0,
		NetworkTx: 0,
		NetworkRx: 0,
		NetworkIn: 0,
		NetworkOut: 0,
		Ping10010: 0.0,
		Ping10086: 0.0,
		Ping189: 0.0,
		Time10010: 0,
		Time10086: 0,
		Time189: 0,
		TCP: 0,
		UDP: 0,
		CPU: 0.0,
		MemoryTotal: 0,
		MemoryUsed: 0,
		SwapTotal: 0,
		SwapUsed: 0,
		Uptime: 0,
		HddTotal: 0,
		HddUsed: 0,
	}
}



var CU_ADDR = CU + ":" + strconv.Itoa(PORBEPORT)
var CT_ADDR = CT + ":" + strconv.Itoa(PORBEPORT)
var CM_ADDR = CM + ":" + strconv.Itoa(PORBEPORT)

//func getNetworkStatus()  {
//	defaulttimeout  :=  1 * time.Second
//	count := 0
//	conn1 , err1 := net.DialTimeout("tcp",CU_ADDR,defaulttimeout)
//	if err1 != nil {
//		fmt.Println("[getNetworkStatus]Error try to connect China unicom :", err1)
//		count += 1
//	}
//	tcpconn1, ok := conn1.(*net.TCPConn)
//	if ok {
//		tcpconn1.SetLinger(0)
//	}
//	if conn1 != nil {
//		conn1.Close()
//	}
//	conn2 , err2 :=  net.DialTimeout("tcp", CT_ADDR,defaulttimeout)
//	if err2 != nil {
//		fmt.Println("[getNetworkStatus]Error try to connect China telecom :", err2)
//		count += 1
//	}
//	tcpconn2, ok := conn2.(*net.TCPConn)
//	if ok {
//		tcpconn2.SetLinger(0)
//	}
//	if conn2 != nil {
//		conn2.Close()
//	}
//	conn3 , err3 :=  net.DialTimeout("tcp", CM_ADDR,defaulttimeout)
//	if err3 != nil {
//		fmt.Println("[getNetworkStatus]Error try to connect China mobile :", err3)
//		count += 1
//	}
//	tcpconn3, ok := conn2.(*net.TCPConn)
//	if ok {
//		tcpconn3.SetLinger(0)
//	}
//	if conn3 != nil {
//		conn3.Close()
//	}
//	if count >= 2 {
//		clientInfo.IpStatus = false
//	} else {
//		clientInfo.IpStatus = true
//	}
//	count = 0
//}

func str2bytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	b := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&b))
}

func bytes2str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

var clientInfo ClientInfo

func SetupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Info("[main] Ctrl+C pressed in Terminal,Stop client program")
		//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [main] Ctrl+C pressed in Terminal,Stop client program")
		if mainConnect != nil {
			if NETWORKCHECK == true {
				pingValueCU.Stop()
				pingValueCT.Stop()
				pingValueCM.Stop()
			}
			netSpeed.Stop()
			run.StopRunInfo()
			mainConnect.Close()
		}
		os.Exit(0)
	}()
}

var mainConnect net.Conn
var netSpeed *NetSpeed
var run *Run
var pingValueCU *PingValue
var pingValueCT *PingValue
var pingValueCM *PingValue


func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [main] Get current directory error")
		os.Exit(-1)
	}
	return dir
}

var logger *go_logger.Logger

func main() {

	logger = go_logger.NewLogger()
	logger.Detach("console")

	// 命令行输出配置
	consoleConfig := &go_logger.ConsoleConfig{
		Color: true, // 命令行输出字符串是否显示颜色
		JsonFormat: true, // 命令行输出字符串是否格式化
		Format: "", // 如果输出的不是 json 字符串，JsonFormat: false, 自定义输出的格式
	}
	// 添加 console 为 logger 的一个输出
	logger.Attach("console", go_logger.LOGGER_LEVEL_DEBUG, consoleConfig)

	// 文件输出配置
	fileConfig := &go_logger.FileConfig {
		Filename : "./test.log", // 日志输出文件名，不自动存在
		// 如果要将单独的日志分离为文件，请配置LealFrimeNem参数。
		LevelFileName : map[int]string {
			logger.LoggerLevel("error"): "./error.log",    // Error 级别日志被写入 error .log 文件
			logger.LoggerLevel("info"): "./info.log",      // Info 级别日志被写入到 info.log 文件中
			logger.LoggerLevel("debug"): "./debug.log",    // Debug 级别日志被写入到 debug.log 文件中
		},
		MaxSize : 1024 * 1024,  // 文件最大值（KB），默认值0不限
		MaxLine : 100000, // 文件最大行数，默认 0 不限制
		DateSlice : "d",  // 文件根据日期切分， 支持 "Y" (年), "m" (月), "d" (日), "H" (时), 默认 "no"， 不切分
		JsonFormat: true, // 写入文件的数据是否 json 格式化
		Format: "", // 如果写入文件的数据不 json 格式化，自定义日志格式
	}
	// 添加 file 为 logger 的一个输出
	logger.Attach("file", go_logger.LOGGER_LEVEL_DEBUG, fileConfig)
	logger.Info(" [main]Logger init complete")

	// Setup our Ctrl+C handler
	SetupCloseHandler()
	config := NewConfig()
	path :=  getCurrentDirectory() + "\\config.json"
	logger.Infof(" [main]Try to load config file from %s\n",path)
	//fmt.Printf(time.Now().Format("2006-01-02 15:04:05")+" [main]Try to load config file from %s\n",path)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		logger.Alertf(" [main]Read config file error:%s\n",err)
		//fmt.Printf(time.Now().Format("2006-01-02 15:04:05")+" [main]Read config file error:%s\n",err)
		goto Run
	}
	err = jsoniter.Unmarshal(data, &config)
	if err != nil {
		logger.Errorf(" [main]Parse config file error:%s\n",err)
		//fmt.Printf(time.Now().Format("2006-01-02 15:04:05")+" [main]Parse config file error:%s\n",err)
	}
	if config.User != "" {
		USER = config.User
	}
	if config.Password != "" {
		PASSWORD = config.Password
	}
	if config.Cm != "" {
		CM = config.Cm
	}
	if config.Cu != "" {
		CU = config.Cu
	}
	if config.Ct != "" {
		CT = config.Ct
	}
	if config.Port >=0 && config.Port <= 65535 {
		PORT = config.Port
	}
	if config.Porbeport >=0 && config.Porbeport <= 65535 {
		PORBEPORT = config.Porbeport
	}
	if config.Interval > 0 {
		INTERVAL = config.Interval
	}
	if config.Server != "" {
		SERVER = config.Server
	}
	NETWORKCHECK = config.NetworkCheck
	Run:
	for _,  args := range os.Args {
		if strings.Index(args,"SERVER") > -1 {
			strArr :=  strings.Split(args,"SERVER=")
			SERVER = strArr[len(strArr)-1]
		} else if strings.Index( args,"PORT") > -1 {
			strArr :=  strings.Split(args,"PORT=")
			PORT, _ = strconv.Atoi(strArr[len(strArr)-1])
		} else if strings.Index( args,"USER") > -1 {
			strArr :=  strings.Split(args,"USER=")
			USER = strArr[len(strArr)-1]
		} else if strings.Index(args,"PASSWORD") > -1 {
			strArr :=  strings.Split(args,"PASSWORD=")
			PASSWORD = strArr[len(strArr)-1]
		} else if strings.Index( args,"INTERVAL")  > -1{
			strArr :=  strings.Split(args,"INTERVAL=")
			INTERVAL, _ = strconv.Atoi(strArr[len(strArr)-1])
		}  else if strings.Index( args,"NETWORKCHECK")  > -1{
			strArr :=  strings.Split(args,"NETWORKCHECK=")
			settings := strings.ToUpper(strArr[len(strArr)-1])
			if strings.Index(settings,"FALSE") > -1 {
				NETWORKCHECK = false
			}
		}
	}
	defaulttimeout  :=  30 * time.Second
	clientInfo = NewDefaultClientInfo()
	netSpeed = NewNetSpeed()
	netSpeed.Run()
	if NETWORKCHECK == true {
		pingValueCU = NewPingValue()
		pingValueCT = NewPingValue()
		pingValueCM = NewPingValue()
		pingValueCU.RunCU()
		pingValueCT.RunCT()
		pingValueCM.RunCM()
	}
	run = NewRunInfo()
	run.StartGetRunInfo()
	logger.Info("[main] Try to connect server")
	for {
		var err error
		mainConnect , err = net.DialTimeout("tcp", SERVER + ":" + strconv.Itoa(PORT),defaulttimeout)
		if err != nil {
			logger.Errorf("[main]Error listening:%s\n", err)
			//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [main]Error listening:", err)
		}
		defer mainConnect.Close()
		logger.Info("[main]Listening done,get server info now")
		//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [main]Listening done,get server info now")
		buff := make([]byte, 1024)
		mainConnect.Read(buff)
		str := bytes2str(buff)
		if strings.Index(str,"Authentication required") > -1 {
			logger.Info("[main]Authentication required,send it to server")
			//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [main]Authentication required,send it to server")
			auth := str2bytes(USER + ":" + PASSWORD + "\n")
			_ , err = mainConnect.Write(auth)
			if err != nil {
				logger.Errorf("[main]Error sending auth info:", err)
				//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [main]Error sending auth info:", err)
				return
			}
			buff = make([]byte, 1024)
			_ , err = mainConnect.Read(buff)
			if err != nil {
				logger.Errorf("[main]Error getting server data:", err)
				//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [main]Error getting server data:", err)
				return
			}
			str = bytes2str(buff)
			if strings.Index(str,"Authentication required") < 0 {
				if strings.Index(str,"Wrong username and/or password.") > -1 {
					logger.Errorf(" [main]Wrong username and/or password.")
					//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [main]Wrong username and/or password.")
					return
				}else if strings.Index(str,"Authentication successful. Access granted.") > -1 {
					logger.Info("[main]Authentication successful. Access granted.")
					//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [main]Authentication successful. Access granted.")
				} else {
					logger.Error(str)
					//fmt.Println(str)
					return
				}
			}
		} else if strings.Index(str,"You have been banned for 1 minute (Wrong username and/or password.)") > -1{
			logger.Alert("[main]You have been banned for 1 minute (Wrong username and/or password.)")
			//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [main]You have been banned for 1 minute (Wrong username and/or password.)")
			return
		} else {
			logger.Error(str)
			fmt.Println(str)
		}
		if strings.Index(str,"You are connecting via") < 0 {
			buff = make([]byte, 1024)
			_ , err = mainConnect.Read(buff)
			if err != nil {
				logger.Errorf("[main]Error getting server data:", err)
				//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [main]Error getting server data:", err)
				return
			}
			str = bytes2str(buff)
			//logger.Info("[main] " + str)
			//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [main]"+str)
		}
		//var checkIP int = 0
		//if strings.Index(str,"IPv4") > -1 {
		//	checkIP = 6
		//} else if strings.Index(str,"IPv6") > -1 {
		//	checkIP = 4
		//} else  {
		//	fmt.Println(str)
		//}
		//fmt.Println(checkIP)
		var (
			status10086 uint = 0
			status189 uint = 0
			status10010 uint = 0
		)
		for {
			run.GetRunInfo()
			//getNetworkStatus()
			netSpeed.Get()
			if NETWORKCHECK {
				clientInfo.Ping10086, clientInfo.Time10086, status10086 = pingValueCM.Get()
				clientInfo.Ping189, clientInfo.Time189, status189 = pingValueCT.Get()
				clientInfo.Ping10010, clientInfo.Time10010,status10010 = pingValueCU.Get()
				if (status189+status10010+status10086) >= 2 {
					clientInfo.IpStatus = false
				} else {
					clientInfo.IpStatus = true
				}
			} else {
				clientInfo.Ping10086, clientInfo.Time10086 = 0.0,0
				clientInfo.Ping189, clientInfo.Time189 =  0.0,0
				clientInfo.Ping10010, clientInfo.Time10010 = 0.0,0
				clientInfo.IpStatus = false
			}
			status10086 = 0
			status189 = 0
			status10010  = 0
			//结构体转json字符串
			data, err := clientInfo.MarshalToString()
			//fmt.Println(data)
			if err != nil {
				logger.Errorf("[main]Error transforming client info: ", err)
				//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [main]Error transforming client info: ", err)
				break
			}
			logger.Info("\nInfo:\n"+data+"\n")
			info := "update " + data + "\n"
			_ , err = mainConnect.Write(str2bytes(info))
			if err != nil {
				logger.Errorf("[main]Error sending client info:", err)
				fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [main]Error sending client info:", err)
				break
			}
		}
	}

}

func (info *ClientInfo) MarshalToString() (string, error) {
	type Alias ClientInfo
	return jsoniter.MarshalToString(&struct {
		CPU float64 `json:"cpu"`
		Ping10010 float64 `json:"ping_10010"`
		Ping10086 float64 `json:"ping_10086"`
		Ping189 float64 `json:"ping_189"`
		Load1 float64 `json:"load_1"`
		Load5 float64 `json:"load_5"`
		Load15 float64 `json:"load_15"`
		*Alias
	}{
		CPU: info.CPU,
		Ping10010: info.Ping10010,
		Ping10086: info.Ping10086,
		Ping189: info.Ping189,
		Load1: info.Load1,
		Load5: info.Load5,
		Load15: info.Load15,
		Alias:    (*Alias)(info),
	})
}
