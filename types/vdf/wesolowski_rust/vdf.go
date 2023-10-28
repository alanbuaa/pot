package wesolowski_rust

import (
	"encoding/hex"
	"fmt"
	"github.com/zzz136454872/upgradeable-consensus/types/vdf/utils"
	"strconv"
)

func Execute(challenge []byte, iterations int, ctrl *utils.Controller, cpuCounter *utils.CPUCounter) []byte {
	// 分配CPU，CPU不够则阻塞
	cpuCounter.Occupy(ctrl)
	// 判断终止
	if ctrl.Abort {
		return nil
	}
	// 判断可执行文件
	if utils.CheckVDFExists() != nil {
		fmt.Println("missing vdf executable file")
		return nil
	}
	// 执行命令
	command := utils.GetCurrentAbPathByCaller() + "/vdf-cli " + hex.EncodeToString(challenge) + " " + strconv.Itoa(iterations)
	output := utils.ExecCmdAffinity(command, ctrl)
	// 释放CPU
	cpuCounter.Release(ctrl)
	// 中止的输出
	if len(output) == 0 {
		return nil
	}
	// 转换输出为字节数组
	res := make([]byte, hex.DecodedLen(len(output)-1))
	n, err := hex.Decode(res, output[:len(output)-1])
	if err != nil {
		fmt.Println("vdf result: ", err)
	}
	return res[:n]
}

func Verify(challenge []byte, iterations int, res []byte) bool {
	// 判断可执行文件
	if utils.CheckVDFExists() != nil {
		return false
	}
	// 转换为字符串
	hexRes := hex.EncodeToString(res)
	// 执行命令
	command := utils.GetCurrentAbPathByCaller() + "/vdf-cli " + hex.EncodeToString(challenge) + " " + strconv.Itoa(iterations) + " " + hexRes
	out := utils.ExecCmd(command)

	if string(out) == "1\n" {
		return true
	}
	return false
}

func CPUverify(challenge []byte, iterations int, ctrl *utils.Controller, cpuCounter *utils.CPUCounter, res []byte) bool {
	// 分配CPU，CPU不够则阻塞
	cpuCounter.Occupy(ctrl)
	// 判断终止
	if ctrl.Abort {
		return false
	}
	// 判断可执行文件
	if utils.CheckVDFExists() != nil {
		fmt.Println("missing vdf executable file")
		return false
	}
	// 执行命令
	// 转换为字符串
	hexRes := hex.EncodeToString(res)
	// 执行命令
	command := utils.GetCurrentAbPathByCaller() + "/vdf-cli " + hex.EncodeToString(challenge) + " " + strconv.Itoa(iterations) + " " + hexRes
	output := utils.ExecCmdAffinity(command, ctrl)
	// 释放CPU
	cpuCounter.Release(ctrl)
	if string(output) == "1\n" {
		return true
	}
	return false
}
