package main

import (
	"net"
	"sync"
	"time"
)

type PingValue struct {
	ping uint64
	status uint
	lostRate float64
	stop chan struct{}
	mtx sync.Mutex
}

func NewPingValue() *PingValue {
	return &PingValue{
		status: 0,
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
		var lostConnect = false
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
					logger.Alertf("[ping]Error try to connect China Unicom :", err)
					//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [ping]Error try to connect China unicom :", err)
					lostConnect = true
					lostPacket += 1
				}
				tcpconn, ok := conn.(*net.TCPConn)
				if ok {
					tcpconn.SetLinger(0)
				}
				if conn != nil {
					conn.Close()
				}
				diffTime := time.Since(t)
				//TODO:三网延迟和丢包率算法存在问题
				//fmt.Println(diffTime)
				allPacket += 1
				if allPacket > 100 {
					pingValue.lostRate = float64(lostPacket/allPacket) * 100
				}
				//fmt.Println("ALL     LOST    RATE")
				//fmt.Printf("%10d  %10d %10f\n",allPacket,lostPacket,pingValue.lostRate)
				if lostConnect {
					pingValue.ping = 0
					pingValue.status = 1
				} else {
					pingValue.ping = uint64(diffTime/time.Millisecond)
					pingValue.status = 0
				}
				lostConnect = false
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
		var lostConnect = false
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
					logger.Alertf("[ping]Error try to connect China Telecom :", err)
					//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [ping]Error try to connect China Telecom :", err)
					lostConnect = true
					lostPacket += 1
				}
				tcpconn, ok := conn.(*net.TCPConn)
				if ok {
					tcpconn.SetLinger(0)
				}
				if conn != nil {
					conn.Close()
				}
				diffTime := time.Since(t)
				allPacket += 1
				if allPacket > 100 {
					pingValue.lostRate = float64(lostPacket/allPacket) * 100
				}
				if lostConnect {
					pingValue.ping = 0
					pingValue.status = 1
				} else {
					pingValue.ping = uint64(diffTime/time.Millisecond)
					pingValue.status = 0
				}
				lostConnect = false
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
		var lostConnect = false
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
					logger.Alertf("[ping]Error try to connect China Mobile :", err)
					//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [ping]Error try to connect China mobile :", err)
					lostConnect = true
					lostPacket += 1
				}
				tcpconn, ok := conn.(*net.TCPConn)
				if ok {
					tcpconn.SetLinger(0)
				}
				if conn != nil {
					conn.Close()
				}
				diffTime := time.Since(t)
				allPacket += 1
				if allPacket > 100 {
					pingValue.lostRate = float64(lostPacket/allPacket) * 100
				}
				if lostConnect {
					pingValue.ping = 0
					pingValue.status = 1
				} else {
					pingValue.ping = uint64(diffTime/time.Millisecond)
					pingValue.status = 0
				}
				lostConnect = false
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

func (pingValue *PingValue) Get() (float64,uint64,uint) {
	pingValue.mtx.Lock()
	defer pingValue.mtx.Unlock()
	status := pingValue.status
	pingValue.status = 0
	return pingValue.lostRate,pingValue.ping,status
}