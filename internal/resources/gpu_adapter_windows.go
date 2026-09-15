package resources

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var createDXGIFactory = windows.NewLazySystemDLL("dxgi.dll").NewProc("CreateDXGIFactory1")

type adapterDescription struct {
	Name                                                            [128]uint16
	Vendor, Device, Subsystem, Revision                             uint32
	DedicatedVideoMemory, DedicatedSystemMemory, SharedSystemMemory uintptr
	LUIDLow                                                         uint32
	LUIDHigh                                                        int32
	Flags                                                           uint32
}

type gpuAdapterMetadata struct {
	name     string
	capacity uint64
	software bool
}

// Enumerate adapter metadata only. No graphics device, context or workload is
// created; hardware names and capacities complement PDH's aggregate usage.
func adapterMetadata() map[string]gpuAdapterMetadata {
	result := make(map[string]gpuAdapterMetadata)
	iid := windows.GUID{Data1: 0x770aae78, Data2: 0xf26f, Data3: 0x4dba, Data4: [8]byte{0xa8, 0x29, 0x25, 0x3c, 0x83, 0xd1, 0xb3, 0x87}}
	var factory uintptr
	if status, _, _ := createDXGIFactory.Call(uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&factory))); status != 0 || factory == 0 {
		return result
	}
	defer comMethod(factory, 2)
	for index := range 8 {
		var adapter uintptr
		if comMethod(factory, 12, uintptr(index), uintptr(unsafe.Pointer(&adapter))) != 0 || adapter == 0 {
			break
		}
		var desc adapterDescription
		status := comMethod(adapter, 10, uintptr(unsafe.Pointer(&desc)))
		comMethod(adapter, 2)
		if status != 0 {
			continue
		}
		key := fmt.Sprintf("luid_0x%08x_0x%08x_phys_0", uint32(desc.LUIDHigh), desc.LUIDLow)
		result[key] = gpuAdapterMetadata{name: windows.UTF16ToString(desc.Name[:]), capacity: uint64(desc.DedicatedVideoMemory), software: desc.Flags&2 != 0}
	}
	return result
}

// Keep Go buffers passed through COM uintptr arguments alive and immovable
// through this wrapper, just as syscall.SyscallN does for direct callers.
//
//go:uintptrescapes
func comMethod(object uintptr, index int, args ...uintptr) uintptr {
	vtable := *(*uintptr)(unsafe.Pointer(object))
	method := *(*uintptr)(unsafe.Pointer(vtable + uintptr(index)*unsafe.Sizeof(uintptr(0))))
	result, _, _ := syscall.SyscallN(method, append([]uintptr{object}, args...)...)
	return result
}
