package hotkey

import "testing"

const (
	vkCtrl  = 0xA2
	vkShift = 0xA0
	vkAlt   = 0xA4
	vkWin   = 0x5B
	vkK     = 0x4B
)

type edge struct {
	vk   uint32
	down bool
}

// replay drives a reducer through physical edges and reports the last outcome
// that was not a plain wait.
func replay(reducer *CaptureReducer, edges []edge) CaptureResult {
	last := CaptureResult{State: CaptureWaiting}
	for _, e := range edges {
		if result := reducer.Event(e.vk, e.down); result.State != CaptureWaiting {
			last = result
		}
	}
	return last
}

func press(vks ...uint32) []edge {
	edges := make([]edge, 0, len(vks)*2)
	for _, vk := range vks {
		edges = append(edges, edge{vk, true})
	}
	for i := len(vks) - 1; i >= 0; i-- {
		edges = append(edges, edge{vks[i], false})
	}
	return edges
}

// Hold-to-talk runs on the low-level hook, so it can arm on modifiers alone.
func TestCaptureAcceptsModifierOnlyChordsForHold(t *testing.T) {
	cases := []struct {
		name string
		keys []uint32
		want string
	}{
		{"ctrl and win", []uint32{vkCtrl, vkWin}, "Ctrl+Super"},
		{"ctrl and shift", []uint32{vkCtrl, vkShift}, "Ctrl+Shift"},
		{"three modifiers", []uint32{vkCtrl, vkShift, vkAlt}, "Ctrl+Alt+Shift"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy, _ := PolicyFor(HoldToTalk)
			reducer := &CaptureReducer{Policy: policy}
			result := replay(reducer, press(tc.keys...))
			if result.State != CaptureComplete {
				t.Fatalf("state = %v, want CaptureComplete (err %v)", result.State, result.Err)
			}
			if got := result.Chord.String(); got != tc.want {
				t.Fatalf("chord = %q, want %q", got, tc.want)
			}
			if !result.Chord.ModifierOnly() {
				t.Fatal("chord should report itself as modifier-only")
			}
		})
	}
}

// A single modifier would arm recording on ordinary typing.
func TestCaptureRejectsASingleModifier(t *testing.T) {
	policy, _ := PolicyFor(HoldToTalk)
	reducer := &CaptureReducer{Policy: policy}
	result := replay(reducer, press(vkCtrl))
	if result.State != CaptureRejected {
		t.Fatalf("state = %v, want CaptureRejected", result.State)
	}
}

// Toggle shortcuts go through RegisterHotKey, which needs a virtual-key code.
// The old behaviour here was silence until the capture timed out.
func TestCaptureRejectsModifierOnlyWhenNotAllowed(t *testing.T) {
	for _, keys := range [][]uint32{{vkCtrl, vkWin}, {vkCtrl}} {
		reducer := &CaptureReducer{}
		result := replay(reducer, press(keys...))
		if result.State != CaptureRejected {
			t.Fatalf("keys %v: state = %v, want CaptureRejected", keys, result.State)
		}
		if result.Err == nil || result.Err.Error() == "" {
			t.Fatalf("keys %v: rejection carried no explanation", keys)
		}
	}
}

// Allowing modifier-only must not change how an ordinary chord records.
func TestCaptureStillRecordsPrimaryKeyChords(t *testing.T) {
	for _, allow := range []bool{false, true} {
		action := ToggleRecording
		if allow {
			action = HoldToTalk
		}
		policy, _ := PolicyFor(action)
		reducer := &CaptureReducer{Policy: policy}
		result := replay(reducer, press(vkCtrl, vkK))
		if result.State != CaptureComplete {
			t.Fatalf("allowModifierOnly=%v: state = %v, want CaptureComplete", allow, result.State)
		}
		if got := result.Chord.String(); got != "Ctrl+K" {
			t.Fatalf("allowModifierOnly=%v: chord = %q, want \"Ctrl+K\"", allow, got)
		}
		if result.Chord.ModifierOnly() {
			t.Fatal("a chord with a primary key must not report as modifier-only")
		}
	}
}

func TestParseHoldRoundTrip(t *testing.T) {
	for _, value := range []string{"Ctrl+Super", "Ctrl+Shift", "Ctrl+Alt+Shift"} {
		chord, err := ParseHold(value)
		if err != nil {
			t.Fatalf("ParseHold(%q) failed: %v", value, err)
		}
		if got := chord.String(); got != value {
			t.Fatalf("ParseHold(%q) round-tripped to %q", value, got)
		}
	}
	// The strict parser must keep refusing what RegisterHotKey cannot take.
	if _, err := Parse("Ctrl+Meta"); err == nil {
		t.Fatal("Parse accepted a modifier-only chord")
	}
	if _, err := ParseHold("Ctrl"); err == nil {
		t.Fatal("ParseHold accepted a single modifier")
	}
	if _, err := ParseHold("Ctrl+K"); err != nil {
		t.Fatalf("ParseHold rejected an ordinary chord: %v", err)
	}
}

// The runtime matcher has to arm and release on the modifier set itself.
func TestRuntimeReducerHandlesModifierOnlyChords(t *testing.T) {
	chord, err := ParseHold("Ctrl+Super")
	if err != nil {
		t.Fatalf("ParseHold failed: %v", err)
	}
	reducer := &Reducer{Chord: chord}

	if got := reducer.Event(vkCtrl, true); got != NoEdge {
		t.Fatalf("first modifier produced %v, want NoEdge", got)
	}
	if got := reducer.Event(vkWin, true); got != Pressed {
		t.Fatalf("completing the set produced %v, want Pressed", got)
	}
	// Typing while held must not end the recording.
	if got := reducer.Event(vkK, true); got != NoEdge {
		t.Fatalf("a letter produced %v, want NoEdge", got)
	}
	if got := reducer.Event(vkK, false); got != NoEdge {
		t.Fatalf("releasing a letter produced %v, want NoEdge", got)
	}
	if got := reducer.Event(vkWin, false); got != Released {
		t.Fatalf("breaking the set produced %v, want Released", got)
	}
}
