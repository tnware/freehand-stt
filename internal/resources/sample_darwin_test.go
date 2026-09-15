//go:build darwin && cgo

package resources

import (
	"testing"
	"time"
)

// Requires native macOS execution; a cross-build is not sampling acceptance.
func TestDarwinHostCounters(t *testing.T) {
	got := readCounters()
	if got.memoryTotal == 0 || got.memoryAvailable > got.memoryTotal {
		t.Fatalf("invalid physical memory counters: %+v", got)
	}
	if !got.cpuOK || got.total == 0 || got.idle > got.total {
		t.Fatalf("invalid CPU counters: %+v", got)
	}
}

func TestDarwinGPUCounterAvailability(t *testing.T) {
	r := newGPUReader()
	defer r.close()
	rows := r.read(time.Now())
	if len(rows) > 8 {
		t.Fatal("unbounded GPU enumeration")
	}
	for _, row := range rows {
		if row.Name == "" || row.UtilizationPercent < 0 || row.UtilizationPercent > 100 {
			t.Fatalf("invalid GPU metric: %+v", row)
		}
		if row.UnifiedMemory && row.MemoryTotalBytes != 0 {
			t.Fatal("unified RAM has an invented VRAM capacity")
		}
	}
}
