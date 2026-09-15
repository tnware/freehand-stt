// Package resources provides bounded, memory-only utilization samples for the
// computer running Freehand. It never contacts an inference server or inspects
// individual process metadata, and sampling does not own any runtime lifecycle.
package resources

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type CPUState string

const (
	CPUUnavailable CPUState = "unavailable"
	CPUMeasuring   CPUState = "measuring"
	CPUReady       CPUState = "ready"
)

// Snapshot contains aggregate host metrics only. Unavailable metrics are never
// reported as zero utilization. Memory availability is an OS-specific estimate.
type Snapshot struct {
	SampledAt            int64    `json:"sampledAt"`
	CPUState             CPUState `json:"cpuState"`
	CPUPercent           float64  `json:"cpuPercent"`
	MemoryAvailable      bool     `json:"memoryAvailable"`
	MemoryTotalBytes     uint64   `json:"memoryTotalBytes"`
	MemoryAvailableBytes uint64   `json:"memoryAvailableBytes"`
	GPUs                 []GPU    `json:"gpus"`
}

type counters struct {
	idle, total                  uint64
	cpuOK                        bool
	memoryTotal, memoryAvailable uint64
}

// Service samples on demand. There are no background workers or subscriptions
// to stop when the workspace is hidden. Calls are serialized and
// rate limited even when multiple WebViews request a sample.
type Service struct {
	mu       sync.Mutex
	read     func() counters
	now      func() time.Time
	ctx      context.Context
	closed   bool
	previous counters
	lastRead time.Time
	latest   Snapshot
	gpu      gpuReader
}

func NewService() *Service { return &Service{read: readCounters, now: time.Now, gpu: newGPUReader()} }

func (s *Service) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx = ctx
	return nil
}

func (s *Service) ServiceShutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.gpu != nil {
		s.gpu.close()
	}
	s.previous = counters{}
	s.latest = Snapshot{CPUState: CPUUnavailable}
	return nil
}

// Current reads cheap native counters, at most once per second. CPU is the
// fraction of aggregate processor time spent busy between consecutive samples.
// After an interruption, establish a fresh baseline instead of averaging over
// the time the workspace was hidden or the machine was asleep.
func (s *Service) Current() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ctx == nil || s.ctx.Err() != nil {
		return Snapshot{CPUState: CPUUnavailable}
	}
	now := s.now()
	elapsed := now.Sub(s.lastRead)
	if !s.lastRead.IsZero() && elapsed >= 0 && elapsed < time.Second {
		cached := s.latest
		cached.GPUs = slices.Clone(cached.GPUs)
		return cached
	}
	next := s.read()
	snapshot := Snapshot{SampledAt: now.UnixMilli(), CPUState: CPUUnavailable}
	if next.cpuOK {
		snapshot.CPUState = CPUMeasuring
		if !s.lastRead.IsZero() && elapsed > 0 && elapsed <= 5*time.Second && s.previous.cpuOK &&
			next.total > s.previous.total && next.idle >= s.previous.idle {
			total := next.total - s.previous.total
			idle := next.idle - s.previous.idle
			if idle <= total {
				snapshot.CPUState = CPUReady
				snapshot.CPUPercent = 100 * float64(total-idle) / float64(total)
			}
		}
	}
	if next.memoryTotal > 0 && next.memoryAvailable <= next.memoryTotal {
		snapshot.MemoryAvailable = true
		snapshot.MemoryTotalBytes = next.memoryTotal
		snapshot.MemoryAvailableBytes = next.memoryAvailable
	}
	if s.gpu != nil {
		snapshot.GPUs = s.gpu.read(now)
	}
	s.previous, s.lastRead, s.latest = next, now, snapshot
	snapshot.GPUs = slices.Clone(snapshot.GPUs)
	return snapshot
}
