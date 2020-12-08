package main
/****************************
* 参考: https://github.com/cppla/ServerStatus/blob/master/clients/client-psutil.py
* Author: chn-student & cppla
* 依赖于 gopstil 跨平台库
* 编译版本:go 1.15.5
* 时间: 20201202 （未完工）
* 说明: 默认情况下修改server和user就可以了。丢包率监测方向可以自定义，例如：CU = "www.facebook.com"。
****************************/

import (
	"fmt"
	"github.com/bitcav/nitr-core/cpu"
	"github.com/bitcav/nitr-core/host"
	"github.com/bitcav/nitr-core/ram"
	"github.com/json-iterator/go"
	"github.com/shirou/gopsutil/mem"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

var (
	SERVER string = "172.16.172.2"
	USER  string = "test"
	PORT int = 35601
	PASSWORD string = "test1234"
	INTERVAL int = 1
	PORBEPORT int = 80
	CU string = "cu.tz.cloudcpp.com"
	CT string = "ct.tz.cloudcpp.com"
	CM string = "cm.tz.cloudcpp.com"
)

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
	Time10010 float64 `json:"time_10010"`
	Time10086 float64 `json:"time_10086"`
	Time189 float64 `json:"time_189"`
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
		Time10010: 0.0,
		Time10086: 0.0,
		Time189: 0.0,
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
		fmt.Println("Get traffic count error:",err)
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
		if fsType != "ext4" &&
			fsType != "ext3" &&
			fsType != "ext2" &&
			fsType  != "reiserfs" &&
			fsType  != "jfs" &&
			fsType  != "btrfs" &&
			fsType  != "fuseblk" &&
			fsType  != "zfs" &&
			fsType  != "simfs" &&
			fsType  != "ntfs" &&
			fsType  != "fat32" &&
			fsType  != "exfat" &&
			fsType  != "xfs" {
			//if(d.Device == "A") { //特殊盘符自己写处理
				continue
			//}
		}
		diskUsageOf, _ := disk.Usage(d.Mountpoint)
		used += diskUsageOf.Used
		total += diskUsageOf.Total
	}
	clientInfo.HddUsed = used / 1024.0 / 1024.0
	clientInfo.HddTotal = total / 1024.0 / 1024.0
}

func getLoad() {
	// linux or freebsd only
	if host.Info().OS == "linux" || host.Info().OS == "freebsd" {
		l, err :=	load.Avg()
		if err != nil {
			fmt.Println("Get CPU Loads failed:",err)
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

func tupd()  {
	//if sys.platform.startswith("linux") is True:
	//t = int(os.popen('ss -t|wc -l').read()[:-1])-1
	//u = int(os.popen('ss -u|wc -l').read()[:-1])-1
	//p = int(os.popen('ps -ef|wc -l').read()[:-1])-2
	//d = int(os.popen('ps -eLf|wc -l').read()[:-1])-2
	//elif sys.platform.startswith("win") is True:
	//t = int(os.popen('netstat -an|find "TCP" /c').read()[:-1])-1
	//u = int(os.popen('netstat -an|find "UDP" /c').read()[:-1])-1
	//p = len(psutil.pids())
	//d = 0
	if host.Info().OS == "linux" {
		byte1 ,err := exec.Command("ss -t|wc -l").Output()
		if err != nil {
			clientInfo.TCP = 0
			fmt.Println("Get TCP count error:",err)
		} else {
			result := bytes2str(byte1)
			intNum, _ := strconv.Atoi(result)
			clientInfo.TCP = uint64(intNum)
		}
		byte2 ,err := exec.Command("ss -u|wc -l").Output()
		if err != nil {
			clientInfo.UDP = 0
			fmt.Println("Get UDP count error:",err)
		} else {
			result := bytes2str(byte2)
			intNum, _ := strconv.Atoi(result)
			clientInfo.UDP = uint64(intNum)
		}
		byte3 ,err := exec.Command("ps -ef|wc -l").Output()
		if err != nil {
			clientInfo.Process = 0
			fmt.Println("Get process count error:",err)
		} else {
			result := bytes2str(byte3)
			intNum, _ := strconv.Atoi(result)
			clientInfo.Process = uint64(intNum)
		}
		byte4 ,err := exec.Command("ps -eLf|wc -l").Output()
		if err != nil {
			clientInfo.Process = 0
			fmt.Println("Get threads count error:",err)
		} else {
			result := bytes2str(byte4)
			intNum, _ := strconv.Atoi(result)
			clientInfo.Process = uint64(intNum)
		}
	} else if host.Info().OS == "windows" {
		// 不知道为何，tcp和udp数量没法获取
		byte1 ,err := exec.Command("cmd", "/C","netstat -an|find \"TCP\" /c").Output()
		if err != nil {
			clientInfo.TCP = 0
			fmt.Println("Get TCP count error:",err)
		} else {
			result := bytes2str(byte1)
			intNum, _ := strconv.Atoi(result)
			clientInfo.TCP = uint64(intNum)
		}
		byte2 ,err := exec.Command("cmd", "/C","netstat -an|find \"UDP\" /c").Output()
		if err != nil {
			clientInfo.UDP = 0
			fmt.Println("Get UDP count error:",err)
		} else {
			result := bytes2str(byte2)
			intNum, _ := strconv.Atoi(result)
			clientInfo.UDP = uint64(intNum)
		}
		pids, err := process.Processes()
		if err != nil {
			fmt.Println("Get process count error:",err)
		} else {
			clientInfo.Process = uint64(len(pids))
		}
		clientInfo.Thread = 0
	}
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

func main() {
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
			USER = strArr[len(strArr)-1]
		} else if strings.Index( args,"INTERVAL")  > -1{
			strArr :=  strings.Split(args,"INTERVAL=")
			INTERVAL, _ = strconv.Atoi(strArr[len(strArr)-1])
		}
	}
	defaulttimeout  :=  30 * time.Second
	for {
		conn , err := net.DialTimeout("tcp", SERVER + ":" + strconv.Itoa(PORT),defaulttimeout)
		if err != nil {
			fmt.Println("Error listening:", err)
		}
		defer conn.Close()
		buff := make([]byte, 1024)
		conn.Read(buff)
		str := bytes2str(buff)
		if strings.Index(str,"Authentication required") > -1 {
			auth := str2bytes(USER + ":" + PASSWORD + "\n")
			_ , err = conn.Write(auth)
			if err != nil {
				fmt.Println("Error Sending auth info:", err)
			}
			buff = make([]byte, 1024)
			_ , err = conn.Read(buff)
			if err != nil {
				fmt.Println("Error Getting Server Data:", err)
			}
			str = bytes2str(buff)
			if strings.Index(str,"Authentication required") < 0 {
				fmt.Println(str)
			}
		} else {
			fmt.Println(str)
		}
		fmt.Println(str)
		if strings.Index(str,"You are connecting via") < 0 {
			buff = make([]byte, 1024)
			_ , err = conn.Read(buff)
			if err != nil {
				fmt.Println("Error Getting Server Data:", err)
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
		clientInfo = NewDefaultClientInfo()

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
			//TODO:三网延迟，三网丢包，tcp/udp连接数，实时网络速度
			//结构体转json字符串
			data, err := jsoniter.MarshalToString(&clientInfo)
			//fmt.Println(data)
			if err != nil {
				fmt.Println("transformation error: ", err)
			}
			info := "update " + data + "\n"
			//fmt.Println(info)
			_ , err = conn.Write(str2bytes(info))
			if err != nil {
				fmt.Println("Error Sending auth info:", err)
			}
		}
	}


}