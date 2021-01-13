package main

import (
	"github.com/shirou/gopsutil/process"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func tupd()  {
	cmd ,err := Command("cmd","/c netstat -an|find \"TCP\" /c")
	if err != nil {
		clientInfo.TCP = 0
		logger.Errorf("[tupd]Get TCP count error:",err)
		//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [tupd]Get TCP count error:",err)
	} else {
		byte1, err := cmd.Output()
		result := bytes2str(byte1)
		result = strings.Replace(result, "\r", "", -1)
		result = strings.Replace(result, "\n", "", -1)
		intNum, err := strconv.Atoi(result)
		if err != nil {
			logger.Errorf("[tupd]Get TCP count error:",err)
			//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [tupd]Get TCP count error:",err)
		}
		clientInfo.TCP = uint64(intNum)
	}
	cmd2 ,err := Command("cmd", "/c netstat -an|find \"UDP\" /c")
	if err != nil {
		clientInfo.UDP = 0
		logger.Errorf("[tupd]Get UDP count error:",err)
		//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [tupd]Get UDP count error:",err)
	} else {
		byte2, err := cmd2.Output()
		result := bytes2str(byte2)
		result = strings.Replace(result, "\r", "", -1)
		result = strings.Replace(result, "\n", "", -1)
		intNum, err := strconv.Atoi(result)
		if err != nil {
			logger.Errorf("[tupd]Get UDP count error:",err)
			//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [tupd]Get UDP count error:",err)
		}
		clientInfo.UDP = uint64(intNum)
	}
	pids, err := process.Processes()
	if err != nil {
		logger.Errorf("[tupd]Get process count error:",err)
		//fmt.Println(time.Now().Format("2006-01-02 15:04:05")," [tupd]Get process count error:",err)
	} else {
		clientInfo.Process = uint64(len(pids))
	}
	clientInfo.Thread = 0

}

func Command(name, args string) (*exec.Cmd, error) {
	// golang 使用 exec.Comand 运行含管道的 cmd 命令会产生问题（如 netstat -an | find "TCP" /c），因此使用此办法调用
	// 参考：https://studygolang.com/topics/10284
	if filepath.Base(name) == name {
		lp, err := exec.LookPath(name)
		if err != nil {
			return nil, err
		}
		name = lp
	}
	return &exec.Cmd{
		Path:        name,
		SysProcAttr: &syscall.SysProcAttr{CmdLine: name + " " + args},
	}, nil
}
