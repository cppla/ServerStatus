package main

import (
	"fmt"
	nnet "github.com/shirou/gopsutil/net"
	"strings"
	"sync"
	"time"
)

type NetSpeed struct {
	netrx float64
	nettx float64
	clock float64
	diff float64
	avgrx uint64
	avgtx uint64
	stop chan struct{}
	mtx sync.Mutex
}

func NewNetSpeed() *NetSpeed{
	return &NetSpeed{
		netrx: 0.0,
		nettx: 0.0,
		clock: 0.0,
		diff:  0.0,
		avgrx: 0,
		avgtx: 0,
		stop: make (chan struct{}),
	}
}

func (netSpeed *NetSpeed) Run()  {
	go func() {
		t1 :=  time.Duration(INTERVAL) * time.Second
		t := time.NewTicker(t1)
		for {
			select {

				case <- netSpeed.stop:
					t.Stop()
					return
				case <-t.C:
					netSpeed.mtx.Lock()
					var bytesSent uint64 = 0
					var bytesRecv uint64 = 0
					netInfo, err := nnet.IOCounters(true)
					if err != nil {
						fmt.Println("Get network speed error:",err)
					}
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
					timeUnix:= float64(time.Now().Unix())
					netSpeed.diff = timeUnix - netSpeed.clock
					netSpeed.clock = timeUnix
					netSpeed.netrx = float64(bytesRecv - netSpeed.avgrx)/netSpeed.diff
					netSpeed.nettx = float64(bytesSent - netSpeed.avgtx)/netSpeed.diff
					netSpeed.avgtx = bytesSent
					netSpeed.avgrx = bytesRecv
					netSpeed.mtx.Unlock()
				}
		}
	}()
}

func (netSpeed *NetSpeed) Stop() {
	close(netSpeed.stop)
}


func (netSpeed *NetSpeed) Get() {
	netSpeed.mtx.Lock()
	defer netSpeed.mtx.Unlock()
	clientInfo.NetworkTx = uint64(netSpeed.nettx)
	clientInfo.NetworkRx = uint64(netSpeed.netrx)
}
