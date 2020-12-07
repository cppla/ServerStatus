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


func str2bytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	b := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&b))
}

func bytes2str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

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
		var clientInfo = NewDefaultClientInfo()

		for {
			clientInfo.MemoryTotal = ram.Info().Total / 1024 // 需要转单位
			clientInfo.MemoryUsed = ram.Info().Usage / 1024 // 需要转单位
			clientInfo.CPU = cpu.Info().Usage
			clientInfo.Uptime = host.Info().Uptime

			//swap 没有造好的轮子，自己加的
			swapMemory, _ := mem.SwapMemory()
			clientInfo.SwapTotal = swapMemory.Total / 1024 // 需要转单位
			clientInfo.SwapUsed = swapMemory.Used / 1024 // 需要转单位

			//结构体转json字符串
			data, err := jsoniter.MarshalToString(&clientInfo)
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