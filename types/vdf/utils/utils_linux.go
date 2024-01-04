package utils

import (
	"fmt"
	"io"
	"os/exec"
	"path"
	"runtime"
)

var (
	FileName = "/vdf-linux"
	FilePath string
)

func init() {
	FilePath = GetCurrentAbPathByCaller() + FileName
}

// GetCurrentAbPathByCaller 获取当前执行文件绝对路径;
// 需要保证vdf-cli与该函数位于同一路径
func GetCurrentAbPathByCaller() string {
	var abPath string
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		abPath = path.Dir(filename)
	}
	return abPath
}

func CheckVDFExists() error {
	// verify hash
	_, err := exec.LookPath(FilePath)
	return err
}

// closeStdoutPipe 关闭输出管道
func closeStdoutPipe(stdout io.ReadCloser) {
	err := stdout.Close()
	if err != nil {
		fmt.Printf("cmd.Close: %v\n", err)
	}
}

func ExecCmd(command string) []byte {
	cmd := exec.Command("bash", "-c", command)

	stdout, _ := cmd.StdoutPipe() // 创建输出管道
	defer closeStdoutPipe(stdout)

	if err := cmd.Start(); err != nil {
		fmt.Printf("cmd.Start: %v\n", err)
	}

	output, err := io.ReadAll(stdout) // 读取输出结果
	if err != nil {
		fmt.Println("ExecCmd", err)
		return nil
	}
	return output
}

func ExecCmdAffinity(command string, ctrl *Controller) []byte {
	cmd := exec.Command("bash", "-c", command)

	stdout, _ := cmd.StdoutPipe() // 创建输出管道
	defer closeStdoutPipe(stdout)

	if err := cmd.Start(); err != nil {
		fmt.Printf("cmd.Start: %v\n", err)
	}

	ctrl.Pid = cmd.Process.Pid // 查看命令pid
	// affinity
	commandAffinity := fmt.Sprintf("taskset -pc %d %d", ctrl.CpuNo, ctrl.Pid)
	cmdAffinity := exec.Command("bash", "-c", commandAffinity)
	if err := cmdAffinity.Start(); err != nil {
		fmt.Printf("cmdAffinity.Start: %v\n", err)
	}

	output, err := io.ReadAll(stdout) // 读取输出结果
	if err != nil {
		return nil
	}
	return output
}

func KillProc(pid int) {
	command := fmt.Sprintf("kill %d", ctrl.Pid)
	ExecCmd(command)
}
