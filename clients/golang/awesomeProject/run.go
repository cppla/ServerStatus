package main

import (
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	nnet "github.com/shirou/gopsutil/net"
	"strings"
	"sync"
	"time"
)

type Run struct {
	memoryTotal uint64
	memoryUsed uint64
	CPU float64
	uptime uint64
	swapTotal uint64
	swapUsed uint64
	load1 float64
	load5 float64
	load15 float64
	networkIn uint64
	networkOut uint64
	hddUsed uint64
	hddTotal uint64
	stop chan struct{}
	mtx sync.Mutex
}

func NewRunInfo() *Run{
	return &Run{
		memoryTotal: 0,
		memoryUsed:  0,
		CPU:         0.0,
		uptime:      0,
		swapTotal:   0,
		swapUsed:    0,
		load1:       0.0,
		load5:       0.0,
		load15:      0.0,
		networkIn:   0,
		networkOut:  0,
		hddUsed:     0,
		hddTotal:    0,
		stop: make (chan struct{}),
	}
}

func (run *Run) StopRunInfo() {
	close(run.stop)
}

func (run *Run) GetRunInfo() {
	run.mtx.Lock()
	defer run.mtx.Unlock()
	clientInfo.HddUsed = run.hddUsed
	clientInfo.HddTotal  = run.hddTotal
	clientInfo.MemoryTotal = run.memoryTotal
	clientInfo.MemoryUsed = run.memoryUsed
	clientInfo.CPU = run.CPU
	clientInfo.Uptime = run.uptime
	clientInfo.SwapTotal = run.swapTotal
	clientInfo.SwapUsed = run.swapUsed
	clientInfo.NetworkIn = run.networkIn
	clientInfo.NetworkOut = run.networkOut
	clientInfo.Load1 = run.load1
	clientInfo.Load5 = run.load5
	clientInfo.Load15 = run.load15
}

func (run *Run) StartGetRunInfo()  {
	go func() {
		t1 :=  time.Duration(1) * time.Second
		t := time.NewTicker(t1)
		for {
			select {
			case <- run.stop:
				t.Stop()
				return
			case <-t.C:
				run.mtx.Lock()
				memInfo, err := mem.VirtualMemory()
				if err != nil {
					logger.Errorf("[getInfo]Get memory usage error:",err)
					//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [getInfo]Get memory usage error:",err)
					run.memoryTotal = 0
					run.memoryUsed = 0
				} else {
					run.memoryTotal = memInfo.Total / 1024 // 需要转单位
					run.memoryUsed = memInfo.Used / 1024 // 需要转单位
				}

				totalPercent, err := cpu.Percent(time.Second, false)
				if err != nil {
					logger.Errorf("[getInfo]Get cpu usage error:",err)
					//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [GetInfo]Get cpu usage error:",err)
					run.CPU = 0.0
				} else {
					if totalPercent != nil {
						run.CPU = totalPercent[0]
					} else {
						logger.Errorf("[getInfo]Get cpu usage error:",err)
						//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [getInfo]Get cpu usage error:",err)
					}
				}
				hInfo, err := host.Info()
				if err != nil {
					logger.Errorf("[getInfo]get uptime error",err)
					//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [getInfo]get uptime error",err)
					run.uptime = 0
				} else {
					run.uptime = hInfo.Uptime
				}
				//swap 没有造好的轮子，自己加的
				swapMemory, err := mem.SwapMemory()
				if err != nil {
					logger.Errorf("[getInfo]Get swap memory error:",err)
					//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [getInfo]Get swap memory error:",err)
					run.swapTotal = 0
					run.swapUsed = 0
				} else {
					run.swapTotal = swapMemory.Total / 1024 // 需要转单位
					run.swapUsed = swapMemory.Used / 1024 // 需要转单位
				}
				getLoad()
				trafficCount()
				spaceCount()
				tupd()
				run.mtx.Unlock()
			}
		}
	}()
}

func trafficCount()  {
	netInfo, err := nnet.IOCounters(true)
	if err != nil {
		logger.Errorf("[trafficCount]Getting traffic count error:",err)
		//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [trafficCount]Getting traffic count error:",err)
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
	run.networkIn = bytesRecv
	run.networkOut = bytesSent
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
				path := diskUsageOf.Path
				//不统计K8s的虚拟挂载点，see here：https://github.com/shirou/gopsutil/issues/1007
				if !strings.Contains(path,"/var/lib/kubelet") {
					used += diskUsageOf.Used
					total += diskUsageOf.Total
				}
			}
		}
	}
	run.hddUsed = used / 1024.0 / 1024.0
	run.hddTotal = total / 1024.0 / 1024.0
}

func getLoad() {
	// linux or freebsd only
	hInfo, err := host.Info()
	if err != nil {
		logger.Errorf("[getLoad]Get load info error",err)
		//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [getLoad]get load info error",err)
		run.load1 = 0.0
		run.load5 = 0.0
		run.load15 = 0.0
	} else {
		if hInfo.OS == "linux" || hInfo.OS == "freebsd" {
			l, err :=	load.Avg()
			if err != nil {
				logger.Errorf("[getLoad]Get CPU loads failed:",err)
				//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [getLoad]Get CPU loads failed:",err)
				run.load1 = 0.0
				run.load5 = 0.0
				run.load15 = 0.0
			} else  {
				run.load1 = l.Load1
				run.load5 = l.Load5
				run.load15 = l.Load15
			}
		} else {
			run.load1 = 0.0
			run.load5 = 0.0
			run.load15 = 0.0
		}
	}
}