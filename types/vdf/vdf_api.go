package vdf

import (
	"errors"
	"fmt"
	"github.com/zzz136454872/upgradeable-consensus/types/vdf/pietrzak"
	"github.com/zzz136454872/upgradeable-consensus/types/vdf/utils"
	"github.com/zzz136454872/upgradeable-consensus/types/vdf/wesolowski_rust"
)

var cpuList = []uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
var cpuCounter = utils.NewCPUCounter(cpuList)
var cpuChecker = utils.NewCPUCounter([]uint8{4, 5})

type Vdf struct {
	//Id         uint64
	Type       string
	Challenge  []byte
	Iterations int
	Controller utils.Controller
}

func New(vdfType string, challenge []byte, iterations int, id int64) *Vdf {
	cpucount := utils.NewCPUCounter(cpuList[3*id : 3*id+3])
	return &Vdf{
		Type:       vdfType,
		Challenge:  challenge,
		Iterations: iterations,
		Controller: utils.Controller{Pid: -1, CPU: 0, Abort: false, Allocated: false, CpuCounter: cpucount},
	}
}

func (vdf *Vdf) Execute() ([]byte, error) {
	switch vdf.Type {
	case "pietrzak":
		return pietrzak.Execute(vdf.Challenge, vdf.Iterations, &vdf.Controller, cpuCounter), nil
	case "wesolowski_rust":
		return wesolowski_rust.Execute(vdf.Challenge, vdf.Iterations, &vdf.Controller, cpuCounter), nil
	default:
		return nil, errors.New("no such vdf scheme")
	}
}

func (vdf *Vdf) Verify(input []byte, res []byte) (bool, error) {
	switch vdf.Type {
	case "pietrzak":
		return pietrzak.CpuVerify(input, vdf.Iterations, &vdf.Controller, cpuCounter, res), nil
	case "wesolowski_rust":
		return wesolowski_rust.Verify(vdf.Challenge, vdf.Iterations, res), nil
	default:
		return false, errors.New("no such vdf scheme")
	}
}

func (vdf *Vdf) Abort() error {

	ctrl := vdf.Controller
	// 如果实例正在等待分配CPU，则停止等待
	ctrl.Abort = true
	// 如果实例正在执行,则终止该进程
	if ctrl.Allocated {
		command := fmt.Sprintf("kill %d", ctrl.Pid)
		utils.ExecCmd(command)
	}
	// 如果实例执行完成正在释放CPU，不做处理
	return nil
}

func (vdf *Vdf) CheckVDF(input []byte, res []byte) bool {
	//return pietrzak.CpuVerify(input, vdf.Iterations, &vdf.Controller, cpuChecker, res)
	//return wesolowski_rust.CPUverify(input, vdf.Iterations, &vdf.Controller, cpuChecker, res)
	return true
}
