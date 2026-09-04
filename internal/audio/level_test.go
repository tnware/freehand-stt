package audio

import (
	"encoding/binary"
	"math"
	"testing"
)

func pcm16(samples ...int16) []byte {
	out := make([]byte, len(samples)*2)
	for i, sample := range samples {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(sample))
	}
	return out
}

func TestPeak(t *testing.T) {
	if got := Peak(nil); got != 0 {
		t.Fatalf("Peak(nil) = %v, want 0", got)
	}
	if got := Peak(pcm16(0, 0, 0)); got != 0 {
		t.Fatalf("silence = %v, want 0", got)
	}
	// A trailing odd byte is not half a sample.
	if got := Peak([]byte{0x00}); got != 0 {
		t.Fatalf("odd-length buffer = %v, want 0", got)
	}
	if got := Peak(pcm16(-32768, 0)); math.Abs(got-1) > 1e-9 {
		t.Fatalf("full scale = %v, want 1", got)
	}
	// The loudest sample wins, however brief. This is the whole reason the
	// meter uses peak: a tap is mostly silence, and its RMS is tiny.
	if got := Peak(pcm16(0, 0, 16384, 0, 0, 0, 0, 0)); math.Abs(got-0.5) > 1e-4 {
		t.Fatalf("a single loud sample read %v, want 0.5", got)
	}
	if quiet, loud := Peak(pcm16(500)), Peak(pcm16(20000)); !(quiet < loud) {
		t.Fatalf("expected quiet(%v) < loud(%v)", quiet, loud)
	}
}

// A linear meter leaves speech in the bottom tenth of its range. The decibel
// mapping is what makes a half-scale signal look like a half-scale signal.
func TestNormalizeLevel(t *testing.T) {
	if got := NormalizeLevel(0); got != 0 {
		t.Fatalf("silence = %v, want 0", got)
	}
	if got := NormalizeLevel(1); math.Abs(got-1) > 1e-9 {
		t.Fatalf("full scale = %v, want 1", got)
	}
	// Anything at or below the floor is off the bottom of the meter.
	if got := NormalizeLevel(math.Pow(10, LevelFloorDB/20)); got != 0 {
		t.Fatalf("the floor itself = %v, want 0", got)
	}
	if got := NormalizeLevel(0.0001); got != 0 {
		t.Fatalf("below the floor = %v, want 0", got)
	}
	// Out of range input must saturate rather than exceed the meter.
	if got := NormalizeLevel(4); got != 1 {
		t.Fatalf("above full scale = %v, want 1", got)
	}

	// Half amplitude is -6 dBFS, which should sit high on the meter, not near
	// the bottom the way a linear mapping would put it.
	half := NormalizeLevel(0.5)
	if half < 0.8 || half > 0.95 {
		t.Fatalf("half amplitude read %v, want roughly 0.87", half)
	}
	// Every halving of amplitude costs the same amount of meter.
	first := NormalizeLevel(0.5) - NormalizeLevel(0.25)
	second := NormalizeLevel(0.25) - NormalizeLevel(0.125)
	if math.Abs(first-second) > 1e-6 {
		t.Fatalf("decibel spacing is uneven: %v then %v", first, second)
	}
	if !(NormalizeLevel(0.02) < NormalizeLevel(0.1) && NormalizeLevel(0.1) < half) {
		t.Fatal("the mapping is not monotonic")
	}
}

// The meter has to jump onto speech and ease off it. Equal attack and release
// would either look like jitter or lag behind the voice.
func TestEnvelopeRisesFastAndFallsSlowly(t *testing.T) {
	envelope := NewEnvelope()

	first := envelope.Push(1)
	if first < 0.5 {
		t.Fatalf("a full-scale sample only reached %v on the first push", first)
	}
	second := envelope.Push(1)
	if second <= first {
		t.Fatalf("level did not keep rising: %v then %v", first, second)
	}

	// Silence must not drop the meter to zero in a single step.
	afterOne := envelope.Push(0)
	if afterOne >= second || afterOne < second*0.5 {
		t.Fatalf("release was not gradual: %v then %v", second, afterOne)
	}
	for range 40 {
		envelope.Push(0)
	}
	if envelope.Value() > 0.01 {
		t.Fatalf("level never settled to silence: %v", envelope.Value())
	}

	envelope.Push(1)
	envelope.Reset()
	if envelope.Value() != 0 {
		t.Fatalf("Reset left %v", envelope.Value())
	}
}

func TestEnvelopeClampsItsInputs(t *testing.T) {
	envelope := Envelope{Attack: -1, Release: 5}
	if got := envelope.Push(-3); got != 0 {
		t.Fatalf("a negative sample produced %v, want 0", got)
	}
	// Out-of-range factors must not run away or invert.
	if got := envelope.Push(0.5); got < 0 || got > 1 {
		t.Fatalf("out-of-range factors produced %v", got)
	}
}

func TestLevelRingScrollsOldestFirst(t *testing.T) {
	ring := NewLevelRing(4)
	if ring.Len() != 4 {
		t.Fatalf("Len = %d, want 4", ring.Len())
	}

	snapshot := ring.Snapshot(nil)
	for _, value := range snapshot {
		if value != 0 {
			t.Fatalf("a fresh ring held %v", value)
		}
	}

	for _, level := range []float64{0.1, 0.2, 0.3, 0.4} {
		ring.Push(level)
	}
	snapshot = ring.Snapshot(snapshot)
	want := []float64{0.1, 0.2, 0.3, 0.4}
	for i := range want {
		if math.Abs(snapshot[i]-want[i]) > 1e-9 {
			t.Fatalf("snapshot = %v, want %v", snapshot, want)
		}
	}

	// A fifth push evicts the oldest and everything shifts down.
	ring.Push(0.5)
	snapshot = ring.Snapshot(snapshot)
	want = []float64{0.2, 0.3, 0.4, 0.5}
	for i := range want {
		if math.Abs(snapshot[i]-want[i]) > 1e-9 {
			t.Fatalf("after eviction = %v, want %v", snapshot, want)
		}
	}

	ring.Push(-2)
	ring.Push(9)
	snapshot = ring.Snapshot(snapshot)
	if snapshot[len(snapshot)-2] != 0 || snapshot[len(snapshot)-1] != 1 {
		t.Fatalf("out-of-range levels were not clamped: %v", snapshot)
	}

	ring.Reset()
	snapshot = ring.Snapshot(snapshot)
	for _, value := range snapshot {
		if value != 0 {
			t.Fatalf("Reset left %v", value)
		}
	}
}

// Snapshot is called every frame, so it must reuse the caller's buffer.
func TestSnapshotReusesTheDestination(t *testing.T) {
	ring := NewLevelRing(8)
	dst := make([]float64, 8)
	if got := ring.Snapshot(dst); &got[0] != &dst[0] {
		t.Fatal("Snapshot allocated instead of filling the buffer it was given")
	}
	if got := ring.Snapshot(make([]float64, 2)); len(got) != 8 {
		t.Fatalf("an undersized buffer produced %d entries, want 8", len(got))
	}
}
