package utils

import (
	"fmt"
	"sync"
	"time"
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

func (c *CPUCounter) Occupy(ctrl *Controller, id string) {
	c.cond.L.Lock()
	for len(c.idleCpuList) == 0 {
		c.cond.Wait()
	}
	if !ctrl.IsAbort {
		ctrl.CpuNo = c.idleCpuList[0]
		fmt.Println(id, " occupy:", ctrl.CpuNo)
		fmt.Println("before occupy, cpuList:", c.idleCpuList)
		c.idleCpuList = c.idleCpuList[1:]
		ctrl.IsAllocated = true
		fmt.Println("after occupy, cpuList: ", c.idleCpuList)
	}
	c.cond.L.Unlock()
}

func (c *CPUCounter) Release(ctrl *Controller, id string) {
	start := time.Now()
	c.cond.L.Lock()
	fmt.Println(id, "release:", ctrl.CpuNo)
	fmt.Println("before release, cpuList:", c.idleCpuList)
	if c.cpuSet.Contains(ctrl.CpuNo) {
		c.idleCpuList = append(c.idleCpuList, ctrl.CpuNo)
		c.cond.Signal()
	}
	fmt.Println("after release, cpuList: ", c.idleCpuList)

	c.cond.L.Unlock()
	fmt.Println("release time :", time.Since(start)/time.Millisecond)
}
