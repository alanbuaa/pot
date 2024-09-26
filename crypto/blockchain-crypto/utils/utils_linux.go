package utils

import (
	"os/exec"
)

var (
	caulkCmd        *exec.Cmd
	caulkExecutable = "./caulk-plus-server"
)

// RunCaulkPlusGRPC 启动Caulk+gRPC进程
func RunCaulkPlusGRPC() error {
	if caulkCmd != nil {
		return nil
	}
	// 创建一个新的命令
	caulkCmd = exec.Command(caulkExecutable)

	// 启动命令，但不要等待它结束
	err := caulkCmd.Start()
	if err != nil {
		return err
	}

	// 返回命令结构体，以便可以稍后进行其他操作（如等待结束）
	return nil
}

func KillCaulkPlusGRPC() error {
	if caulkCmd != nil {
		return caulkCmd.Process.Kill()
	}
	return nil
}
