package resources

import "time"

// GPU contains one adapter's aggregate driver metrics. Zero capacity means the
// driver did not report a dedicated-memory capacity, not an empty VRAM pool.
type GPU struct {
	Name                  string  `json:"name"`
	UtilizationAvailable  bool    `json:"utilizationAvailable"`
	UtilizationPercent    float64 `json:"utilizationPercent"`
	MemoryAvailable       bool    `json:"memoryAvailable"`
	MemoryUsedBytes       uint64  `json:"memoryUsedBytes"`
	MemoryTotalBytes      uint64  `json:"memoryTotalBytes"`
	SharedMemoryAvailable bool    `json:"sharedMemoryAvailable"`
	SharedMemoryUsedBytes uint64  `json:"sharedMemoryUsedBytes"`
	UnifiedMemory         bool    `json:"unifiedMemory"`
}

type gpuReader interface {
	read(time.Time) []GPU
	close()
}
