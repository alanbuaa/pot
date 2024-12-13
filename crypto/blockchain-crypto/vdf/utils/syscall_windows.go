package utils

import (
	"golang.org/x/sys/windows"
	"syscall"
	"unsafe"
)

var (
	modkernel32                          = windows.NewLazySystemDLL("kernel32.dll")
	procSetProcessAffinityMask           = modkernel32.NewProc("SetProcessAffinityMask")
	procSetThreadAffinityMask            = modkernel32.NewProc("SetThreadAffinityMask")
	procGetProcessAffinityMask           = modkernel32.NewProc("GetProcessAffinityMask")
	procGetLogicalProcessorInformationEx = modkernel32.NewProc("GetLogicalProcessorInformationEx")
)

const (
	ErrorInsufficientBuffer = 122
)

const (
	RelationProcessorCore    = 0
	RelationNumaNode         = 1
	RelationCache            = 2
	RelationProcessorPackage = 3
	RelationAll              = 0xffff
)

type SystemLogicalProcessorInformationEx struct {
	Relationship uint32
	Size         uint32
	Processor    ProcessorRelationship
}
type GroupAffinity struct {
	Mask     uint64
	Group    uint16
	Reserved [3]uint16
}

// ProcessorRelationship describes the logical processors associated with either a processor core or a processor package.
// If the Relationship member of the SystemLogicalProcessorInformationEx structure is RelationProcessorCore.
type ProcessorRelationship struct {
	// Flags is 1 if the core has more than one logical processor, or 0 if the core has one logical processor.
	Flags byte
	// EfficiencyClass specifies the intrinsic tradeoff between performance and power for the applicable core.
	// EfficiencyClass is only nonzero on systems with a heterogeneous set of cores.
	EfficiencyClass byte
	Reserved        [20]byte
	// This member specifies the number of entries in the GroupMask array.
	GroupCount uint16
	GroupMask  GroupAffinity
}

func SetProcessAffinityMask(s windows.Handle, processAffinityMask uint64) (err error) {
	ret, _, err := procSetProcessAffinityMask.Call(uintptr(s), uintptr(processAffinityMask))
	if ret == 0 {
		if err != nil {
			return err
		}
		return syscall.GetLastError()
	}
	return nil
}

func SetThreadAffinityMask(s windows.Handle, threadAffinityMask uint64) (err error) {
	ret, _, err := procSetThreadAffinityMask.Call(uintptr(s), uintptr(threadAffinityMask))
	if ret == 0 {
		if err != nil {
			return err
		}
		return syscall.GetLastError()
	}
	return nil
}

func GetProcessAffinityMask(s windows.Handle) (uint64, error) {
	var processAffinityMask, systemAffinityMask uint64

	ret, _, err := procGetProcessAffinityMask.Call(
		uintptr(s),
		uintptr(unsafe.Pointer(&processAffinityMask)),
		uintptr(unsafe.Pointer(&systemAffinityMask)),
	)
	if ret == 0 {
		return 0, err
	}
	return processAffinityMask, nil
}

func GetLogicalProcessorInformationEx() ([]SystemLogicalProcessorInformationEx, error) {
	var buffer []SystemLogicalProcessorInformationEx
	var bufferSize uint32
	_, _, err := procGetLogicalProcessorInformationEx.Call(
		uintptr(RelationProcessorCore),
		uintptr(unsafe.Pointer(&buffer)),
		uintptr(unsafe.Pointer(&bufferSize)),
	)
	if err.(syscall.Errno) != ErrorInsufficientBuffer {
		return nil, err
	}
	buffer = make([]SystemLogicalProcessorInformationEx, bufferSize/uint32(unsafe.Sizeof(SystemLogicalProcessorInformationEx{})))
	ret, _, err := procGetLogicalProcessorInformationEx.Call(
		uintptr(RelationProcessorCore),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&bufferSize)),
	)
	if ret == 0 {
		return nil, err
	}

	return buffer, nil
}

func SplitBitmapIntoIndex(n uint64) []uint64 {
	var powers []uint64
	power := uint64(0)
	for n > 0 {
		if n&1 == 1 {
			powers = append(powers, power)
		}
		n >>= 1
		power++
	}
	return powers
}
