package resources

import (
	"testing"

	"golang.org/x/sys/windows"
)

// Reads host counters only; never starts a runtime or performs inference.
func TestWindowsHostCounters(t *testing.T) {
	got := readCounters()
	if got.memoryTotal == 0 || got.memoryAvailable > got.memoryTotal {
		t.Fatalf("invalid physical memory counters: %+v", got)
	}
	if count := windows.GetActiveProcessorCount(0xffff); count > 0 && count <= 64 {
		if !got.cpuOK || got.total == 0 || got.idle > got.total {
			t.Fatalf("invalid CPU counters: %+v", got)
		}
	}
}
