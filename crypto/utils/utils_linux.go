package utils

import (
	"fmt"
	"os/exec"
)

// TODO RunCaulkPlusGRPC 启动Caulk+gRPC进程
func RunCaulkPlusGRPC() (*exec.Cmd, error){
	// 创建一个新的命令
	cmd := exec.Command("caulk-plus-server")

	// 启动命令，但不要等待它结束
	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	// 返回命令结构体，以便可以稍后进行其他操作（如等待结束）
	return cmd, nil
}

func KillCaulkPlusGRPC(cmd *exec.Cmd) error {
	if cmd == nil {
		return fmt.Errorf("error: try to kill a nil process")
	}
	return cmd.Process.Kill()
}