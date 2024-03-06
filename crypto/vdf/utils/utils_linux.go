package utils

import (
	"encoding/hex"
	"fmt"
	"io"
	"os/exec"
	"path"
	"runtime"
	"strconv"
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
func ExecWesolowskiVDFAffinity(challenge []byte, iterations int, ctrl *Controller) ([]byte, error) {
	vdfCmd := exec.Command(FilePath, hex.EncodeToString(challenge), strconv.Itoa(iterations))

	stdout, _ := vdfCmd.StdoutPipe() // 创建输出管道
	//defer closeStdoutPipe(stdout)

	if err := vdfCmd.Start(); err != nil {
		fmt.Printf("vdfCmd.Start: %v\n", err)
		return nil, err
	}

	ctrl.Pid = vdfCmd.Process.Pid // 查看命令pid
	// affinity
	//tasksetCmd := exec.Command("taskset", " -pc", strconv.Itoa(int(ctrl.CpuNo)), strconv.Itoa(ctrl.Pid))
	tasksetCmd := exec.Command("bash", "-c", fmt.Sprintf("taskset -pc %d %d", ctrl.CpuNo, ctrl.Pid))
	if err := tasksetCmd.Run(); err != nil {
		fmt.Printf("tasksetCmd.Start: %v\n", err)
		return nil, err
	}
	output, err := io.ReadAll(stdout) // 读取输出结果
	if err != nil {
		fmt.Println("read output", err)
		return nil, err
	}
	err = vdfCmd.Wait()
	if err != nil {

		// 检查错误类型是否是 *exec.ExitError
		if _, ok := err.(*exec.ExitError); !ok {
			fmt.Printf("vdf cpu-%d wait:%v\n", ctrl.CpuNo, err)
			//// 获取 WaitStatus
			//if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
			//	// 检查是否为 Signaled
			//	if status.Signaled() {
			//		// 获取终止进程的信号
			//		signal := status.Signal()
			//		if signal == syscall.SIGTERM {
			//			fmt.Println("Process was terminated by SIGTERM")
			//		}
			//	}
			//}
			return nil, err
		}
		return nil, err
	}
	return output, nil
}

func ExecPietrzakVDFAffinity(challenge []byte, iterations int, ctrl *Controller) ([]byte, error) {
	command := FilePath + " -tpietrzak " + hex.EncodeToString(challenge) + " " + strconv.Itoa(iterations)
	cmd := exec.Command("bash", "-c", command)

	stdout, _ := cmd.StdoutPipe() // 创建输出管道
	//defer closeStdoutPipe(stdout)

	if err := cmd.Start(); err != nil {
		fmt.Printf("cmd.Start: %v\n", err)
		return nil, err
	}

	ctrl.Pid = cmd.Process.Pid // 查看命令pid
	// affinity
	commandAffinity := fmt.Sprintf("taskset -pc %d %d", ctrl.CpuNo, ctrl.Pid)
	cmdAffinity := exec.Command("bash", "-c", commandAffinity)
	if err := cmdAffinity.Start(); err != nil {
		fmt.Printf("cmdAffinity.Start: %v\n", err)
		return nil, err
	}

	output, err := io.ReadAll(stdout) // 读取输出结果
	if err != nil {
		return nil, err
	}
	return output, nil
}

func KillProc(pid int) error {
	command := fmt.Sprintf("kill %d", pid)
	cmd := exec.Command("bash", "-c", command)
	return cmd.Run()
}
