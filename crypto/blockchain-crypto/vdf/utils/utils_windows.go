package utils

import (
	"encoding/hex"
	"fmt"
	"io"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"syscall"

	"golang.org/x/sys/windows"
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

func GetCurrentAbPathByCaller() string {
	var abPath string
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		abPath = path.Dir(filename)
	}
	return abPath
}

func CheckVDFExists() error {
	_, err := exec.LookPath(FilePath)
	return err
}

func closeStdoutPipe(stdout io.ReadCloser) {
	err := stdout.Close()
	if err != nil {
		fmt.Printf("cmd.Close: %v\n", err)
	}
}

func ExecCmd(command string) []byte {
	cmd := exec.Command("cmd", "/C", command)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("ExecCmd", err)
		return nil
	}
	defer closeStdoutPipe(stdout)

	if err := cmd.Start(); err != nil {
		fmt.Printf("cmd.Start: %v\n", err)
		return nil
	}

	output, err := io.ReadAll(stdout)
	if err != nil {
		fmt.Println("ExecCmd", err)
		return nil
	}
	if err := cmd.Wait(); err != nil {
		fmt.Println("ExecCmd", err)
		return nil
	}
	return output
}

func ExecWesolowskiVDFAffinity(challenge []byte, iterations int, ctrl *Controller) ([]byte, error) {
	return execVDFAffinity([]string{hex.EncodeToString(challenge), strconv.Itoa(iterations)}, ctrl)
}

func ExecPietrzakVDFAffinity(challenge []byte, iterations int, ctrl *Controller) ([]byte, error) {
	return execVDFAffinity([]string{"-tpietrzak", hex.EncodeToString(challenge), strconv.Itoa(iterations)}, ctrl)
}

func execVDFAffinity(args []string, ctrl *Controller) ([]byte, error) {
	cmd := exec.Command(FilePath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer closeStdoutPipe(stdout)

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	ctrl.Pid = cmd.Process.Pid

	handle, err := syscall.OpenProcess(PROCESS_SET_INFORMATION|PROCESS_QUERY_INFORMATION, false, uint32(ctrl.Pid))
	if err != nil {
		fmt.Printf("open process failed: %v\n", err)
	} else {
		handleForWin := windows.Handle(handle)
		if err := SetProcessAffinityMask(handleForWin, 1<<ctrl.CpuNo); err != nil {
			fmt.Printf("set process affinity failed: %v\n", err)
		}
		if err := windows.SetPriorityClass(handleForWin, windows.HIGH_PRIORITY_CLASS); err != nil {
			fmt.Printf("set priority class failed: %v\n", err)
		}
		if err := syscall.CloseHandle(handle); err != nil {
			fmt.Printf("close process handle failed: %v\n", err)
		}
	}

	output, err := io.ReadAll(stdout)
	if err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return output, nil
}

func KillProc(pid int) {
	command := fmt.Sprintf("taskkill /f /pid %s", strconv.Itoa(pid))
	taskkillCmd := exec.Command("cmd", "/C", command)
	err := taskkillCmd.Run()
	if err != nil {
		fmt.Println("kill process failed", err)
	}
}
