package resources

import (
	"strings"
	"testing"
	"time"
)

func TestGPUEnginesAggregateByAdapter(t *testing.T) {
	const adapter = "luid_0x00000000_0x000012ab_phys_0"
	rows := aggregateGPUCounters([]counterValue{
		{"pid_1_" + adapter + "_eng_0_engtype_Compute", 40},
		{"pid_2_" + strings.ToUpper(adapter) + "_eng_0_engtype_Compute", 30},
		{"pid_1_" + adapter + "_eng_1_engtype_3D", 60},
		{"pid_1_luid_0x00000000_0x00005678_phys_0_eng_0_engtype_Compute", 90},
	}, []counterValue{{adapter, 4 << 30}}, []counterValue{{adapter, 1 << 30}})
	got := rows[adapter]
	if !got.UtilizationAvailable || got.UtilizationPercent != 70 {
		t.Fatalf("expected busiest engine summed across processes, got %+v", got)
	}
	if got.MemoryUsedBytes != 4<<30 || !got.MemoryAvailable || got.SharedMemoryUsedBytes != 1<<30 || !got.SharedMemoryAvailable {
		t.Fatalf("incorrect adapter memory: %+v", got)
	}
	if len(rows) != 2 {
		t.Fatalf("expected two independent adapters, got %d", len(rows))
	}
	rows = aggregateGPUCounters([]counterValue{{"pid_1_" + adapter + "_eng_0_engtype_Compute", 120}}, nil, nil)
	if rows[adapter].UtilizationPercent != 100 {
		t.Fatal("GPU percent must be bounded")
	}
	if rows[adapter].MemoryAvailable {
		t.Fatal("absent memory is unavailable")
	}
}

func TestWindowsGPUCounters(t *testing.T) {
	r := &windowsGPUReader{}
	t.Cleanup(r.close)
	first := r.read(time.Now())
	if len(first) == 0 {
		t.Skip("WDDM GPU counters unavailable on this host")
	}
	// PDH utilization needs two real samples. This performs no GPU workload.
	time.Sleep(1100 * time.Millisecond)
	rows := r.read(time.Now())
	if len(rows) == 0 {
		t.Fatal("GPU counters disappeared")
	}
	for _, row := range rows {
		if row.Name == "" || row.UtilizationPercent < 0 || row.UtilizationPercent > 100 {
			t.Fatalf("invalid GPU snapshot: %+v", row)
		}
		t.Logf("%s: utilization available=%t, dedicated memory available=%t, capacity available=%t", row.Name, row.UtilizationAvailable, row.MemoryAvailable, row.MemoryTotalBytes > 0)
	}
	r.close()
	if r.query != 0 {
		t.Fatal("query handle leaked")
	}
}
