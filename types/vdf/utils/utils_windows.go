package utils

import (
	"encoding/hex"
	"fmt"
	"golang.org/x/sys/windows"
	"io"
	"log"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"syscall"
)

var (
	FileName                  = "/vdf-win.exe"
	FilePath                  string
	PROCESS_SET_INFORMATION   = uint32(0x200)
	PROCESS_QUERY_INFORMATION = uint32(0x400)
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
	cmd := exec.Command("cmd", "/C", command)

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

func ExecWesolowskiVDFAffinity(challenge []byte, iterations int, ctrl *Controller) []byte {
	cmd := exec.Command(FilePath, hex.EncodeToString(challenge), strconv.Itoa(iterations))
	stdout, _ := cmd.StdoutPipe() // 创建输出管道
	defer closeStdoutPipe(stdout)

	if err := cmd.Start(); err != nil {
		fmt.Printf("cmd.Start: %v\n", err)
	}

	// set pid
	ctrl.Pid = cmd.Process.Pid

	// open handle
	handle, err := syscall.OpenProcess(PROCESS_SET_INFORMATION|PROCESS_QUERY_INFORMATION, false, uint32(ctrl.Pid))
	if err != nil {
		log.Fatal(err)
	}
	handleForWin := windows.Handle(handle)

	// set affinity
	err = SetProcessAffinityMask(handleForWin, 1<<ctrl.CpuNo)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	// set priority
	err = windows.SetPriorityClass(handleForWin, windows.HIGH_PRIORITY_CLASS)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	// 关闭句柄
	err = syscall.CloseHandle(handle)
	if err != nil {
		log.Fatal(err)
	}

	output, err := io.ReadAll(stdout) // 读取输出结果
	if err != nil {
		return nil
	}
	return output
}

func ExecPietrzakVDF2ffinity(challenge []byte, iterations int, ctrl *Controller) []byte {
	cmd := exec.Command(FilePath, "-tpietrzak", hex.EncodeToString(challenge), strconv.Itoa(iterations))
	stdout, _ := cmd.StdoutPipe() // 创建输出管道
	defer closeStdoutPipe(stdout)

	if err := cmd.Start(); err != nil {
		fmt.Printf("cmd.Start: %v\n", err)
	}

	// set pid
	ctrl.Pid = cmd.Process.Pid

	// open handle
	handle, err := syscall.OpenProcess(PROCESS_SET_INFORMATION|PROCESS_QUERY_INFORMATION, false, uint32(ctrl.Pid))
	if err != nil {
		log.Fatal(err)
	}
	handleForWin := windows.Handle(handle)

	// set affinity
	err = SetProcessAffinityMask(handleForWin, 1<<ctrl.CpuNo)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	// set priority
	err = windows.SetPriorityClass(handleForWin, windows.HIGH_PRIORITY_CLASS)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	// 关闭句柄
	err = syscall.CloseHandle(handle)
	if err != nil {
		log.Fatal(err)
	}

	output, err := io.ReadAll(stdout) // 读取输出结果
	if err != nil {
		return nil
	}
	return output
}

func KillProc(pid int) {
	command := fmt.Sprintf("taskkill /f /pid %s", strconv.Itoa(pid))
	taskkillCmd := exec.Command("cmd", "/C", command)
	err := taskkillCmd.Run()
	if err != nil {
		fmt.Println("杀死进程失败:", err)
		// os.Exit(1)
	}
}
