package resources

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32           = windows.NewLazySystemDLL("kernel32.dll")
	getSystemTimes     = kernel32.NewProc("GetSystemTimes")
	globalMemoryStatus = kernel32.NewProc("GlobalMemoryStatusEx")
)

// MEMORYSTATUSEX from sysinfoapi.h. The unused fields preserve native layout.
type memoryStatus struct {
	length, load                     uint32
	total, available                 uint64
	totalPageFile, availablePageFile uint64
	totalVirtual, availableVirtual   uint64
	availableExtendedVirtual         uint64
}

func readCounters() counters {
	var result counters
	var idle, kernel, user windows.Filetime
	// GetSystemTimes covers only one processor group on machines with >64
	// logical processors. Do not label a partial group as whole-computer use.
	if count := windows.GetActiveProcessorCount(0xffff); count > 0 && count <= 64 {
		ok, _, _ := getSystemTimes.Call(uintptr(unsafe.Pointer(&idle)), uintptr(unsafe.Pointer(&kernel)), uintptr(unsafe.Pointer(&user)))
		if ok != 0 {
			result.cpuOK = true
			result.idle = filetimeTicks(idle)
			// Kernel time includes idle time; count idle exactly once.
			result.total = filetimeTicks(kernel) + filetimeTicks(user)
		}
	}
	var memory memoryStatus
	memory.length = uint32(unsafe.Sizeof(memory))
	if ok, _, _ := globalMemoryStatus.Call(uintptr(unsafe.Pointer(&memory))); ok != 0 {
		result.memoryTotal, result.memoryAvailable = memory.total, memory.available
	}
	return result
}

func filetimeTicks(value windows.Filetime) uint64 {
	return uint64(value.HighDateTime)<<32 | uint64(value.LowDateTime)
}
