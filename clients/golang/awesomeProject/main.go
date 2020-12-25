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
	"io/ioutil"
	"os/signal"
	"path/filepath"
	"syscall"

	//下面这是已经封装好的轮子
	"github.com/bitcav/nitr-core/cpu"
	"github.com/bitcav/nitr-core/host"
	"github.com/bitcav/nitr-core/ram"
	"github.com/json-iterator/go"
	//没轮子的自己封装
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	nnet "github.com/shirou/gopsutil/net"
	"net"
	"os"
	"strconv"
	"strings"
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

func trafficCount()  {
	netInfo, err := nnet.IOCounters(true)
	if err != nil {
		fmt.Println("[trafficCount]Get traffic count error:",err)
	}
	var bytesSent uint64 = 0
	var bytesRecv uint64 = 0
	for _, v := range netInfo {
		if strings.Index(v.Name,"lo") > -1 ||
			strings.Index(v.Name,"tun") > -1 ||
			strings.Index(v.Name,"docker") > -1 ||
			strings.Index(v.Name,"veth") > -1 ||
			strings.Index(v.Name,"br-") > -1 ||
			strings.Index(v.Name,"vmbr") > -1 ||
			strings.Index(v.Name,"vnet") > -1 ||
			strings.Index(v.Name,"kube") > -1 {
			continue
		}
		bytesSent += v.BytesSent
		bytesRecv += v.BytesRecv
	}
	clientInfo.NetworkIn = bytesRecv
	clientInfo.NetworkOut = bytesSent
}

func spaceCount() {
	// golang 没有类似于在 python 的 dict 或 tuple 的 in 查找关键字，自己写多重判断实现
	diskList, _ := disk.Partitions(false)
	var total uint64 = 0
	var used uint64 = 0
	for _,d := range diskList {
		fsType := strings.ToLower(d.Fstype)
		//fmt.Println(d.Fstype)
		if strings.Index(fsType, "ext4") < 0 &&
			strings.Index(fsType, "ext3") < 0  &&
			strings.Index(fsType, "ext2") < 0  &&
			strings.Index(fsType, "reiserfs") < 0  &&
			strings.Index(fsType, "jfs") < 0  &&
			strings.Index(fsType, "btrfs") < 0  &&
			strings.Index(fsType, "fuseblk") < 0  &&
			strings.Index(fsType, "zfs") < 0  &&
			strings.Index(fsType, "simfs") < 0  &&
			strings.Index(fsType, "ntfs")< 0 &&
			strings.Index(fsType, "fat32") < 0  &&
			strings.Index(fsType, "exfat") < 0  &&
			strings.Index(fsType, "xfs") < 0 {
		}  else  {
			if strings.Index(d.Device, "Z:") > -1 { //特殊盘符自己写处理
				continue
			} else {
				diskUsageOf, _ := disk.Usage(d.Mountpoint)
				used += diskUsageOf.Used
				total += diskUsageOf.Total
			}
		}
	}
	clientInfo.HddUsed = used / 1024.0 / 1024.0
	clientInfo.HddTotal = total / 1024.0 / 1024.0
}

func getLoad() {
	// linux or freebsd only
	if host.Info().OS == "linux" || host.Info().OS == "freebsd" {
		l, err :=	load.Avg()
		if err != nil {
			fmt.Println("[getLoad]Get CPU Loads failed:",err)
		} else  {
			clientInfo.Load1 = l.Load1
			clientInfo.Load5 = l.Load5
			clientInfo.Load15 = l.Load15
		}
	} else {
		clientInfo.Load1 = 0.0
		clientInfo.Load5 = 0.0
		clientInfo.Load15 = 0.0
	}
}

var CU_ADDR = CU + ":" + strconv.Itoa(PORBEPORT)
var CT_ADDR = CT + ":" + strconv.Itoa(PORBEPORT)
var CM_ADDR = CM + ":" + strconv.Itoa(PORBEPORT)

func getNetworkStatus()  {
	defaulttimeout  :=  1 * time.Second
	count := 0
	conn1 , err1 := net.DialTimeout("tcp",CU_ADDR,defaulttimeout)
	if err1 != nil {
		fmt.Println("[getNetworkStatus]Error try to connect China unicom :", err1)
		count += 1
	}
	tcpconn1, ok := conn1.(*net.TCPConn)
	if ok {
		tcpconn1.SetLinger(0)
	}
	if conn1 != nil {
		conn1.Close()
	}
	conn2 , err2 :=  net.DialTimeout("tcp", CT_ADDR,defaulttimeout)
	if err2 != nil {
		fmt.Println("[getNetworkStatus]Error try to connect China telecom :", err2)
		count += 1
	}
	tcpconn2, ok := conn2.(*net.TCPConn)
	if ok {
		tcpconn2.SetLinger(0)
	}
	if conn2 != nil {
		conn2.Close()
	}
	conn3 , err3 :=  net.DialTimeout("tcp", CM_ADDR,defaulttimeout)
	if err3 != nil {
		fmt.Println("[getNetworkStatus]Error try to connect China mobile :", err3)
		count += 1
	}
	tcpconn3, ok := conn2.(*net.TCPConn)
	if ok {
		tcpconn3.SetLinger(0)
	}
	if conn3 != nil {
		conn3.Close()
	}
	if count >= 2 {
		clientInfo.IpStatus = false
	} else {
		clientInfo.IpStatus = true
	}
	count = 0
}

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
		fmt.Println("\r[main] Ctrl+C pressed in Terminal,Stop client program")
		if mainConnect != nil {
			pingValueCU.Stop()
			pingValueCT.Stop()
			pingValueCM.Stop()
			netSpeed.Stop()
			mainConnect.Close()
		}
		os.Exit(0)
	}()
}

var mainConnect net.Conn
var netSpeed *NetSpeed
var pingValueCU *PingValue
var pingValueCT *PingValue
var pingValueCM *PingValue

func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println("[main] Get Current Directory Error")
		os.Exit(-1)
	}
	return dir
}

func main() {
	// Setup our Ctrl+C handler
	SetupCloseHandler()
	config := NewConfig()
	path :=  getCurrentDirectory() + "\\config.json"
	fmt.Printf("[main]Try to Load Config File From %s\n",path)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("[main]Read Config File Error:%s\n",err)
	}
	err = jsoniter.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("[main]Parse Config File Error:%s\n",err)
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
		}
	}
	defaulttimeout  :=  30 * time.Second
	clientInfo = NewDefaultClientInfo()
	netSpeed = NewNetSpeed()
	pingValueCU = NewPingValue()
	pingValueCT = NewPingValue()
	pingValueCM = NewPingValue()
	pingValueCU.RunCU()
	pingValueCT.RunCT()
	pingValueCM.RunCM()
	netSpeed.Run()
	for {
		var err error
		mainConnect , err = net.DialTimeout("tcp", SERVER + ":" + strconv.Itoa(PORT),defaulttimeout)
		if err != nil {
			fmt.Println("[main]Error listening:", err)
		}
		defer mainConnect.Close()
		buff := make([]byte, 1024)
		mainConnect.Read(buff)
		str := bytes2str(buff)
		if strings.Index(str,"[main]Authentication required") > -1 {
			auth := str2bytes(USER + ":" + PASSWORD + "\n")
			_ , err = mainConnect.Write(auth)
			if err != nil {
				fmt.Println("[main]Error Sending auth info:", err)
				return
			}
			buff = make([]byte, 1024)
			_ , err = mainConnect.Read(buff)
			if err != nil {
				fmt.Println("[main]Error Getting Server Data:", err)
				return
			}
			str = bytes2str(buff)
			if strings.Index(str,"[main]Authentication required") < 0 {
				fmt.Println(str)
			}
		} else {
			fmt.Println(str)
		}
		fmt.Println(str)
		if strings.Index(str,"[main]You are connecting via") < 0 {
			buff = make([]byte, 1024)
			_ , err = mainConnect.Read(buff)
			if err != nil {
				fmt.Println("[main]Error Getting Server Data:", err)
				return
			}
			str = bytes2str(buff)
			fmt.Println(str)
		}
		var checkIP int = 0
		if strings.Index(str,"IPv4") > -1 {
			checkIP = 6
		} else if strings.Index(str,"IPv6") > -1 {
			checkIP = 4
		} else  {
			fmt.Println(str)
		}
		fmt.Println(checkIP)
		for {
			clientInfo.MemoryTotal = ram.Info().Total / 1024 // 需要转单位
			clientInfo.MemoryUsed = ram.Info().Usage / 1024 // 需要转单位
			clientInfo.CPU = cpu.Info().Usage
			clientInfo.Uptime = host.Info().Uptime
			//swap 没有造好的轮子，自己加的
			swapMemory, _ := mem.SwapMemory()
			clientInfo.SwapTotal = swapMemory.Total / 1024 // 需要转单位
			clientInfo.SwapUsed = swapMemory.Used / 1024 // 需要转单位
			getLoad()
			tupd()
			trafficCount()
			spaceCount()
			getNetworkStatus()
			netSpeed.Get()
			clientInfo.Ping10086, clientInfo.Time10086 = pingValueCM.Get()
			clientInfo.Ping189, clientInfo.Time189 = pingValueCT.Get()
			clientInfo.Ping10010, clientInfo.Time10010 = pingValueCU.Get()
			//结构体转json字符串
			data, err := jsoniter.MarshalToString(&clientInfo)
			//fmt.Println(data)
			if err != nil {
				fmt.Println("[main]Transformation Error: ", err)
				break
			}
			info := "update " + data + "\n"
			_ , err = mainConnect.Write(str2bytes(info))
			if err != nil {
				fmt.Println("[main]Error Sending Data Info:", err)
				break
			}
		}
	}

}

