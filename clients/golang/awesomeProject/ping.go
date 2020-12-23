package main

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type PingValue struct {
	ping uint64
	lostRate float64
	stop chan struct{}
	mtx sync.Mutex
}

func NewPingValue() *PingValue {
	return &PingValue{
		ping: 0.0,
		lostRate: 0.0,
		stop: make (chan struct{}),
	}
}

func (pingValue *PingValue) RunCU() {
	go func() {
		t1 :=  time.Duration(INTERVAL) * time.Second
		t := time.NewTicker(t1)
		var lostPacket = 0
		var allPacket = 0
		startTime := time.Now()
		defaulttimeout  :=  1 * time.Second
		for {
			select {
			case <- pingValue.stop:
				t.Stop()
				return
			case <-t.C:
				pingValue.mtx.Lock()
				t := time.Now()
				conn , err := net.DialTimeout("tcp",CU_ADDR,defaulttimeout)
				if err != nil {
					fmt.Println("Error try to connect China unicom :", err)
					lostPacket += 1
					return
				}
				defer conn.Close()
				diffTime := time.Since(t)
				//TODO:三网延迟和丢包率算法存在问题
				//fmt.Println(diffTime)
				allPacket += 1
				if allPacket > 100 {
					pingValue.lostRate = float64(lostPacket/allPacket)
				}
				pingValue.ping = uint64(diffTime/time.Millisecond)
				resetTime := uint64(time.Since(startTime) / time.Second)
				if resetTime > 3600 {
					lostPacket = 0
					allPacket = 0
					startTime = time.Now()
				}
				pingValue.mtx.Unlock()
			}
		}
	}()
}

func (pingValue *PingValue) RunCT() {
	go func() {
		t1 :=  time.Duration(INTERVAL) * time.Second
		t := time.NewTicker(t1)
		var lostPacket = 0
		var allPacket = 0
		startTime := time.Now()
		defaulttimeout  :=  1 * time.Second
		for {
			select {
			case <- pingValue.stop:
				t.Stop()
				return
			case <-t.C:
				pingValue.mtx.Lock()
				t := time.Now()
				conn , err := net.DialTimeout("tcp",CT_ADDR,defaulttimeout)
				if err != nil {
					fmt.Println("Error try to connect China Telecom :", err)
					lostPacket += 1
					return
				}
				defer conn.Close()
				diffTime := time.Since(t)
				allPacket += 1
				if allPacket > 100 {
					pingValue.lostRate = float64(lostPacket/allPacket)
				}
				pingValue.ping = uint64(diffTime/time.Millisecond)
				resetTime := uint64(time.Since(startTime) / time.Second)
				if resetTime > 3600 {
					lostPacket = 0
					allPacket = 0
					startTime = time.Now()
				}
				pingValue.mtx.Unlock()
			}
		}
	}()
}

func (pingValue *PingValue) RunCM() {
	go func() {
		t1 :=  time.Duration(INTERVAL) * time.Second
		t := time.NewTicker(t1)
		var lostPacket = 0
		var allPacket = 0
		startTime := time.Now()
		defaulttimeout  :=  1 * time.Second
		for {
			select {
			case <- pingValue.stop:
				t.Stop()
				return
			case <-t.C:
				pingValue.mtx.Lock()
				t := time.Now()
				conn , err := net.DialTimeout("tcp",CM_ADDR,defaulttimeout)
				if err != nil {
					fmt.Println("Error try to connect China mobile :", err)
					lostPacket += 1
					return
				}
				defer conn.Close()
				diffTime := time.Since(t)
				allPacket += 1
				if allPacket > 100 {
					pingValue.lostRate = float64(lostPacket/allPacket)
				}
				pingValue.ping = uint64(diffTime/time.Millisecond)
				resetTime := uint64(time.Since(startTime) / time.Second)
				if resetTime > 3600 {
					lostPacket = 0
					allPacket = 0
					startTime = time.Now()
				}
				pingValue.mtx.Unlock()
			}
		}
	}()
}

func (pingValue *PingValue) Stop() {
	close(pingValue.stop)
}

func (pingValue *PingValue) Get() (float64,uint64) {
	pingValue.mtx.Lock()
	defer pingValue.mtx.Unlock()
	return pingValue.lostRate,pingValue.ping
}