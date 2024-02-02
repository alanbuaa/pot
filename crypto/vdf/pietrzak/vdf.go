package pietrzak

import (
	"encoding/hex"
	"fmt"
	"github.com/zzz136454872/upgradeable-consensus/crypto/vdf/utils"
	"strconv"
)

func Execute(challenge []byte, iterations int, ctrl *utils.Controller, cpuCounter *utils.CPUCounter) []byte {
	// 分配CPU，CPU不够则阻塞
	cpuCounter.Occupy(ctrl)
	// 判断终止
	if ctrl.IsAbort {
		cpuCounter.Release(ctrl)
		return nil
	}
	// 判断可执行文件
	if utils.CheckVDFExists() != nil {
		fmt.Println("missing vdf executable file")
		return nil
	}
	// 执行命令
	output := utils.ExecPietrzakVDFAffinity(challenge, iterations, ctrl)
	// 释放CPU
	cpuCounter.Release(ctrl)
	// 中止的输出
	outputLen := len(output)
	if outputLen == 0 {
		return nil
	}
	// 转换输出为字节数组
	res := make([]byte, hex.DecodedLen(outputLen-1)) // 去除末尾换行符
	_, err := hex.Decode(res, output[:outputLen-1])
	if err != nil {
		fmt.Println("vdf result: ", err)
		return nil
	}
	return res
}

func Verify(challenge []byte, iterations int, res []byte) bool {
	// check vdf exec
	if utils.CheckVDFExists() != nil {
		fmt.Println("missing vdf executable file")
		return false
	}
	// 转换为字符串
	hexRes := hex.EncodeToString(res)
	// 执行命令
	command := utils.FilePath + " -tpietrzak " + hex.EncodeToString(challenge) + " " + strconv.Itoa(iterations) + " " + hexRes
	out := utils.ExecCmd(command)
	if string(out) == "1\n" {
		return true
	}
	return false
}
