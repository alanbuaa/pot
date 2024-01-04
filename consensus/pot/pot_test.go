package pot

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"testing"
)

func TestPoTMsgReceive(t *testing.T) {
	cpuInfo, err := cpu.Info()
	if err != nil {
		fmt.Printf("Failed to get CPU info: %v", err)
		return
	}
	fmt.Printf("CPU info: %+v\n", cpuInfo)

}
