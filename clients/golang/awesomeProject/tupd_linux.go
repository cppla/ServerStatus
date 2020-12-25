// +build linux

package main
import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)
func tupd()  {
	byte1 ,err := exec.Command("bash", "-c","ss -t|wc -l").Output()
	if err != nil {
		clientInfo.TCP = 0
		fmt.Println("[tupd]Get TCP count error:",err)
	} else {
		result := bytes2str(byte1)
		result = strings.Replace(result, "\n", "", -1)
		intNum, err := strconv.Atoi(result)
		if err != nil {
			fmt.Println("[tupd]Get TCP count error::",err)
		}
		clientInfo.TCP = uint64(intNum)
	}
	byte2 ,err := exec.Command("bash", "-c","ss -u|wc -l").Output()
	if err != nil {
		clientInfo.UDP = 0
		fmt.Println("[tupd]Get UDP count error:",err)
	} else {
		result := bytes2str(byte2)
		result = strings.Replace(result, "\n", "", -1)
		intNum, err := strconv.Atoi(result)
		if err != nil {
			fmt.Println("[tupd]Get UDP count error:",err)
		}
		clientInfo.UDP = uint64(intNum)
	}
	byte3 ,err := exec.Command("bash", "-c","ps -ef|wc -l").Output()
	if err != nil {
		clientInfo.Process = 0
		fmt.Println("[tupd]Get process count error:",err)
	} else {
		result := bytes2str(byte3)
		result = strings.Replace(result, "\n", "", -1)
		intNum, err := strconv.Atoi(result)
		if err != nil {
			fmt.Println("[tupd]Get process count error:",err)
		}
		clientInfo.Process = uint64(intNum)
	}
	byte4 ,err := exec.Command("bash", "-c","ps -eLf|wc -l").Output()
	if err != nil {
		clientInfo.Process = 0
		fmt.Println("[tupd]Get threads count error:",err)
	} else {
		result := bytes2str(byte4)
		result = strings.Replace(result, "\n", "", -1)
		intNum, err := strconv.Atoi(result)
		if err != nil {
			fmt.Println("[tupd]Get threads count error:",err)
		}
		clientInfo.Thread = uint64(intNum)
	}
}

