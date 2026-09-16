package resources

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

func TestCOMMetadataCallPreservesNativePointersAndOutput(t *testing.T) {
	var vtable [13]uintptr
	object := &comObject{vtable: &vtable[0]}
	var called bool
	vtable[10] = syscall.NewCallback(func(this *comObject, description *adapterDescription) uintptr {
		runtime.GC()
		called = this == object
		description.DedicatedVideoMemory = 12345
		return 7
	})
	var description adapterDescription
	status := comMethod(object, 10, uintptr(unsafe.Pointer(&description)))
	if !called || status != 7 || description.DedicatedVideoMemory != 12345 {
		t.Fatal("COM dispatch lost the object, output buffer, or return value")
	}
}
