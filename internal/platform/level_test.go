package platform

import (
	"math"
	"sync"
	"testing"
)

// A tap is written from the audio thread and read from a meter thread, so it
// has to hold the loudest value between reads rather than whichever buffer
// happened to land on the reader's tick.
func TestLevelTapHoldsTheMaximumBetweenReads(t *testing.T) {
	var meter LevelTap
	if got := meter.TakeLevel(); got != 0 {
		t.Fatalf("a fresh tap reported %v, want 0", got)
	}

	meter.observe(0.2)
	meter.observe(0.9)
	meter.observe(0.4)
	if got := meter.TakeLevel(); math.Abs(got-0.9) > 1e-6 {
		t.Fatalf("peak = %v, want the maximum 0.9", got)
	}
	// Reading clears it, so a quiet moment reads as quiet.
	if got := meter.TakeLevel(); got != 0 {
		t.Fatalf("the meter did not reset on read: %v", got)
	}
}

func TestLevelTapIsSafeUnderConcurrentObservation(t *testing.T) {
	var meter LevelTap
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for step := 0; step < 500; step++ {
				meter.observe(float64(n*500+step) / 4000)
			}
		}(worker)
	}
	wg.Wait()
	if got := meter.TakeLevel(); got <= 0 || got > 1 {
		t.Fatalf("concurrent peak = %v, outside 0..1", got)
	}
}
