package utils

import (
	"fmt"
	"sync"
)

type CPUCounter struct {
	cond        *sync.Cond
	idleCpuList []uint8
	cpuSet      Set
}

type Controller struct {
	Pid         int
	CpuNo       uint8
	IsAbort     bool
	IsAllocated bool
	CpuCounter  *CPUCounter
}

func NewCPUCounter(cpuList []uint8) (c *CPUCounter) {
	// 创建一个新的切片，拷贝 cpuList 中的元素到新的切片中
	idleCpuList := make([]uint8, len(cpuList))
	copy(idleCpuList, cpuList)
	return &CPUCounter{
		cond:        sync.NewCond(&sync.Mutex{}),
		idleCpuList: cpuList,
		cpuSet:      NewSet(cpuList),
	}
}

func (c *CPUCounter) Occupy(ctrl *Controller) {
	c.cond.L.Lock()
	for len(c.idleCpuList) == 0 {
		c.cond.Wait()
	}
	if !ctrl.IsAbort {
		ctrl.CpuNo = c.idleCpuList[0]
		c.idleCpuList = c.idleCpuList[1:]
		ctrl.IsAllocated = true

	}
	c.cond.L.Unlock()
}

func (c *CPUCounter) Release(ctrl *Controller) {
	c.cond.L.Lock()
	if c.cpuSet.Contains(ctrl.CpuNo) {
		c.idleCpuList = append(c.idleCpuList, ctrl.CpuNo)
		c.cond.Signal()
	}
	fmt.Println(c.idleCpuList)
	c.cond.L.Unlock()
}
