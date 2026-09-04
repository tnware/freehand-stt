package app

import (
	"math"
	"testing"
)

type stubTap struct{ level float64 }

func (s *stubTap) TakeLevel() float64 { return s.level }

func newTestPump(tap *stubTap, wanted *bool) (*levelPump, *[]float64) {
	var sent []float64
	pump := newLevelPump(tap, func(level float64) { sent = append(sent, level) }, func() bool { return *wanted })
	return pump, &sent
}

// Nothing should cross the bridge while the settings window is hidden, which
// during dictation is most of the time.
func TestLevelPumpIsSilentWhenNothingCanSeeIt(t *testing.T) {
	wanted := false
	pump, sent := newTestPump(&stubTap{level: 0.9}, &wanted)

	for range 10 {
		pump.tick()
	}
	if len(*sent) != 0 {
		t.Fatalf("sent %d events with nothing watching: %v", len(*sent), *sent)
	}
}

func TestLevelPumpEmitsWhileWatched(t *testing.T) {
	wanted := true
	pump, sent := newTestPump(&stubTap{level: 0.5}, &wanted)

	for range 5 {
		pump.tick()
	}
	if len(*sent) != 5 {
		t.Fatalf("sent %d events, want 5", len(*sent))
	}
	// Half amplitude is -6 dBFS, so the meter should sit high rather than near
	// the bottom the way a linear mapping would put it.
	last := (*sent)[len(*sent)-1]
	if last < 0.7 || last > 1 {
		t.Fatalf("half amplitude settled at %v, want a high reading", last)
	}
	// The envelope rises across ticks rather than snapping to its target.
	if (*sent)[0] >= last {
		t.Fatalf("level did not rise: first %v, last %v", (*sent)[0], last)
	}
}

// Leaving the watched state must settle the meter once instead of freezing the
// last loud reading on screen, and then stop sending.
func TestLevelPumpSettlesOnceWhenItStops(t *testing.T) {
	wanted := true
	tap := &stubTap{level: 0.9}
	pump, sent := newTestPump(tap, &wanted)

	for range 3 {
		pump.tick()
	}
	before := len(*sent)

	wanted = false
	pump.tick()
	if len(*sent) != before+1 {
		t.Fatalf("expected exactly one settling event, got %d", len(*sent)-before)
	}
	if final := (*sent)[len(*sent)-1]; final != 0 {
		t.Fatalf("the settling event carried %v, want 0", final)
	}

	for range 10 {
		pump.tick()
	}
	if len(*sent) != before+1 {
		t.Fatalf("kept sending after going quiet: %d extra", len(*sent)-before-1)
	}

	// Resuming carries nothing over from the level it left at. A fast attack
	// means a loud source would jump straight back up, so the check is against
	// a quiet one: any residue would show as a reading well above it.
	tap.level = 0.01
	wanted = true
	pump.tick()
	resumed := (*sent)[len(*sent)-1]
	if resumed > 0.35 {
		t.Fatalf("resumed at %v against a quiet source: the envelope kept its old value", resumed)
	}
}

func TestLevelPumpStopIsIdempotent(t *testing.T) {
	wanted := false
	pump, _ := newTestPump(&stubTap{}, &wanted)
	pump.stop()
	pump.stop()
}

// Silence must read as silence rather than as a floor the meter never leaves.
func TestLevelPumpReportsSilence(t *testing.T) {
	wanted := true
	pump, sent := newTestPump(&stubTap{level: 0}, &wanted)
	for range 20 {
		pump.tick()
	}
	if last := (*sent)[len(*sent)-1]; math.Abs(last) > 1e-9 {
		t.Fatalf("silence read as %v", last)
	}
}
