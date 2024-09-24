package vdf

import (
	"blockchain-crypto/vdf/pietrzak"
	"blockchain-crypto/vdf/utils"
	"blockchain-crypto/vdf/wesolowski_rust"
)

var cpuList = []uint8{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
var cpuCounter = utils.NewCPUCounter(cpuList)
var cpuChecker = utils.NewCPUCounter([]uint8{4, 5})

type Vdf struct {
	// Id         uint64
	Type       string
	Challenge  []byte
	Iterations int
	Controller utils.Controller
}

func New(vdfType string, challenge []byte, iterations int, id int64) *Vdf {
	return &Vdf{
		Type:       vdfType,
		Challenge:  challenge,
		Iterations: iterations,
		Controller: utils.Controller{Pid: -1, CpuNo: 0, IsAbort: false, IsAllocated: false},
	}
}

func (vdf *Vdf) Execute() ([]byte, error) {
	switch vdf.Type {
	case "pietrzak":
		return pietrzak.Execute(vdf.Challenge, vdf.Iterations, &vdf.Controller, cpuCounter)
	default:
		return wesolowski_rust.Execute(vdf.Challenge, vdf.Iterations, &vdf.Controller, cpuCounter)
	}
}

func (vdf *Vdf) Verify(res []byte) bool {
	switch vdf.Type {
	case "pietrzak":
		return pietrzak.Verify(vdf.Challenge, vdf.Iterations, res)
	default:
		return wesolowski_rust.Verify(vdf.Challenge, vdf.Iterations, res)
	}
}

func (vdf *Vdf) Abort() error {
	ctrl := vdf.Controller
	// 如果实例正在等待分配CPU，则停止等待
	ctrl.IsAbort = true
	// 如果实例正在执行,则终止该进程
	if ctrl.IsAllocated {
		utils.KillProc(ctrl.Pid)
	}
	// 如果实例执行完成正在释放CPU，不做处理
	return nil
}

func (vdf *Vdf) CheckVDF(input []byte, res []byte) bool {
	// return pietrzak.CpuVerify(input, vdf.Iterations, &vdf.Controller, cpuChecker, res)
	return wesolowski_rust.Verify(input, vdf.Iterations, res)
	// return true
}
