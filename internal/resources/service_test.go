package resources

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestSamplingAndCache(t *testing.T) {
	now := time.Unix(100, 0)
	reads := 0
	value := counters{cpuOK: true, total: 1000, idle: 500, memoryTotal: 1600, memoryAvailable: 400}
	s := &Service{now: func() time.Time { return now }, read: func() counters { reads++; return value }}
	if err := s.ServiceStartup(t.Context(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	first := s.Current()
	if first.CPUState != CPUMeasuring || !first.MemoryAvailable || first.MemoryAvailableBytes != 400 {
		t.Fatalf("first sample = %+v", first)
	}
	now = now.Add(500 * time.Millisecond)
	if cached := s.Current(); !reflect.DeepEqual(cached, first) || reads != 1 {
		t.Fatalf("rate limit did not reuse sample: %+v, reads %d", cached, reads)
	}
	now = now.Add(1500 * time.Millisecond)
	value.total += 400
	value.idle += 100
	if next := s.Current(); next.CPUState != CPUReady || next.CPUPercent != 75 {
		t.Fatalf("delta should be 75%% busy, got %+v", next)
	}
	now = now.Add(6 * time.Second)
	value.total += 400
	if next := s.Current(); next.CPUState != CPUMeasuring {
		t.Fatalf("resumed sample averaged across pause: %+v", next)
	}
}

func TestInvalidCountersAndRecovery(t *testing.T) {
	for _, tc := range []struct {
		name   string
		next   counters
		cpu    CPUState
		memory bool
	}{
		{"failed", counters{}, CPUUnavailable, false},
		{"cpu reset", counters{cpuOK: true, total: 1, idle: 1, memoryTotal: 100, memoryAvailable: 20}, CPUMeasuring, true},
		{"idle reset", counters{cpuOK: true, total: 200, idle: 1}, CPUMeasuring, false},
		{"idle exceeds total delta", counters{cpuOK: true, total: 200, idle: 180}, CPUMeasuring, false},
		{"unchanged clock", counters{cpuOK: true, total: 100, idle: 50}, CPUMeasuring, false},
		{"invalid memory", counters{cpuOK: true, total: 200, idle: 100, memoryTotal: 100, memoryAvailable: 101}, CPUReady, false},
		{"no available memory", counters{memoryTotal: 100}, CPUUnavailable, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Unix(100, 0)
			value := counters{cpuOK: true, total: 100, idle: 50}
			s := &Service{ctx: t.Context(), now: func() time.Time { return now }, read: func() counters { return value }}
			s.Current()
			now = now.Add(2 * time.Second)
			value = tc.next
			if got := s.Current(); got.CPUState != tc.cpu || got.MemoryAvailable != tc.memory {
				t.Fatalf("invalid sample = %+v", got)
			}
			// Every failure is recoverable with a fresh pair of valid samples.
			value = counters{cpuOK: true, total: 1000, idle: 500, memoryTotal: 100, memoryAvailable: 30}
			now = now.Add(2 * time.Second)
			s.Current()
			value.total += 100
			value.idle += 100
			now = now.Add(2 * time.Second)
			if got := s.Current(); got.CPUState != CPUReady || got.CPUPercent != 0 || !got.MemoryAvailable {
				t.Fatalf("recovered idle sample = %+v", got)
			}
		})
	}
}

func TestLifecyclePreventsReads(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	s := NewService()
	s.gpu = nil
	reads := 0
	s.read = func() counters { reads++; return counters{memoryTotal: 100} }
	s.Current()
	if reads != 0 {
		t.Fatal("sample before startup")
	}
	if err := s.ServiceStartup(ctx, application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	s.Current()
	cancel()
	if got := s.Current(); reads != 1 || got.MemoryAvailable {
		t.Fatal("sample after cancellation")
	}
	if err := s.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
	if got := s.Current(); reads != 1 || got.SampledAt != 0 {
		t.Fatal("sample after shutdown")
	}
}

type fixtureGPUReader struct{ reads, closes int }

func (g *fixtureGPUReader) read(time.Time) []GPU {
	g.reads++
	return []GPU{{Name: "GPU", UtilizationAvailable: true, UtilizationPercent: 50}}
}
func (g *fixtureGPUReader) close() { g.closes++ }

func TestGPUCacheIsolationAndShutdown(t *testing.T) {
	gpu := &fixtureGPUReader{}
	s := &Service{ctx: t.Context(), now: func() time.Time { return time.Unix(100, 0) }, read: func() counters { return counters{} }, gpu: gpu}
	first := s.Current()
	first.GPUs[0].Name = "modified by caller"
	if got := s.Current(); got.GPUs[0].Name != "GPU" || gpu.reads != 1 {
		t.Fatal("GPU cache was mutated or sampled more than once")
	}
	if err := s.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
	if gpu.closes != 1 {
		t.Fatal("GPU reader did not close")
	}
	if got := s.Current(); len(got.GPUs) != 0 || gpu.reads != 1 {
		t.Fatal("GPU reader used after shutdown")
	}
}
