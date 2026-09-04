package platform

import (
	"math"
	"sync/atomic"
)

// LevelSource supplies capture amplitude to a meter. It is deliberately the
// narrowest possible surface: one scalar, taken and cleared, carrying no audio
// and no transcript content.
type LevelSource interface {
	// TakeLevel returns the peak amplitude since the previous call, 0..1.
	TakeLevel() float64
}

// LevelTap is one consumer's peak-hold.
//
// Reading clears it, so every consumer needs its own: the overlay and the
// settings window sample at different rates, and sharing a single hold would
// have each of them stealing the other's peaks and both reading low.
//
// It is written from the audio thread, so it allocates nothing, takes no lock
// and touches one word.
type LevelTap struct{ bits atomic.Uint32 }

// observe records a level if it is louder than the one already held.
func (t *LevelTap) observe(level float64) {
	// IEEE-754 bit patterns of non-negative floats compare in value order, so
	// the maximum can be kept without decoding.
	next := math.Float32bits(float32(level))
	for {
		current := t.bits.Load()
		if next <= current {
			return
		}
		if t.bits.CompareAndSwap(current, next) {
			return
		}
	}
}

// TakeLevel returns the peak since the previous call and clears it, so a
// consumer polling more slowly than buffers arrive still sees every peak
// rather than whichever buffer landed on its tick.
func (t *LevelTap) TakeLevel() float64 {
	return float64(math.Float32frombits(t.bits.Swap(0)))
}
