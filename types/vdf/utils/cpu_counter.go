package utils

import (
	"sync"
)

type CPUCounter struct {
	mu          sync.Mutex
	IdleCPUList []uint8
	MaxCPUNum   uint8
}

type Controller struct {
	Pid        int
	CPU        uint8
	Abort      bool
	Allocated  bool
	CpuCounter *CPUCounter
}

func NewCPUCounter(cpuList []uint8) (c *CPUCounter) {
	return &CPUCounter{
		mu:          sync.Mutex{},
		IdleCPUList: cpuList,
		MaxCPUNum:   uint8(len(cpuList)),
	}
}

func (c *CPUCounter) Occupy(ctrl *Controller) {
	for {
		if ctrl.Abort {
			break
		}
		c.mu.Lock()
		// 从空闲队列中获取
		if len(c.IdleCPUList) > 0 {
			ctrl.CPU = c.IdleCPUList[0]
			c.IdleCPUList = c.IdleCPUList[1:]
			ctrl.Allocated = true
		}
		c.mu.Unlock()
		if ctrl.Allocated {
			break
		}
	}
}

func (c *CPUCounter) Release(ctrl *Controller) {
	c.mu.Lock()
	// 添加到空闲队列
	if ctrl.CPU < c.MaxCPUNum {
		c.IdleCPUList = append(c.IdleCPUList, ctrl.CPU)
	}
	//ctrl.Allocated = false
	c.mu.Unlock()
}
