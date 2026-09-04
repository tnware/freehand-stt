package hotkey

import "testing"

func TestParseBoundedGrammar(t *testing.T) {
	for _, value := range []string{"Ctrl+Shift+Space", "CmdOrCtrl+D", "Alt+F11", "F13", "Ctrl+F24"} {
		if _, err := Parse(value); err != nil {
			t.Fatalf("Parse(%q): %v", value, err)
		}
	}
	for _, value := range []string{"F12", "Ctrl+F12", "Ctrl", "Ctrl+Shift", "Ctrl+Ctrl+A", "Ctrl+A+B", "Ctrl+Escape"} {
		if _, err := Parse(value); err == nil {
			t.Fatalf("Parse(%q) succeeded", value)
		}
	}
}

func TestPolicyMatrixAndNormalization(t *testing.T) {
	cases := []struct {
		action ShortcutAction
		value  string
		want   string
	}{
		{ToggleRecording, "Shift+CmdOrCtrl+D", "Ctrl+Shift+D"},
		{ShowFreehand, "win+f13", "Super+F13"},
		{HoldToTalk, "control+command", "Ctrl+Super"},
		{HoldToTalk, "F24", "F24"},
	}
	for _, tc := range cases {
		got, err := NormalizeFor(tc.action, tc.value)
		if err != nil || got != tc.want {
			t.Fatalf("NormalizeFor(%q, %q) = %q, %v; want %q", tc.action, tc.value, got, err, tc.want)
		}
	}
	for _, tc := range []struct {
		action ShortcutAction
		value  string
		kind   ShortcutRejectionKind
	}{
		{ToggleRecording, "Ctrl+Shift", RejectionIncomplete},
		{HoldToTalk, "Ctrl", RejectionIncomplete},
		{ShowFreehand, "F12", RejectionReserved},
		{ToggleRecording, "A", RejectionIncomplete},
		{HoldToTalk, "Ctrl+Escape", RejectionUnsupported},
		{ToggleRecording, "CmdOrCtrl+Ctrl+A", RejectionUnsupported},
	} {
		_, err := ParseFor(tc.action, tc.value)
		rejection, ok := RejectionDetails(err)
		if !ok || rejection.Kind != tc.kind {
			t.Fatalf("ParseFor(%q, %q) error = %v; want %q", tc.action, tc.value, err, tc.kind)
		}
	}
}

func TestValidateAssignmentsFindsAliasAndOrderDuplicates(t *testing.T) {
	err := ValidateAssignments(ShortcutAssignments{
		ToggleRecording: "Ctrl+Shift+D",
		ShowFreehand:    "Shift+CmdOrCtrl+D",
	})
	rejection, ok := RejectionDetails(err)
	if !ok || rejection.Kind != RejectionDuplicate || rejection.ConflictingAction != ToggleRecording {
		t.Fatalf("rejection = %#v, err = %v", rejection, err)
	}
}

func TestCaptureReducerAcceptsDedicatedFunctionKey(t *testing.T) {
	policy, _ := PolicyFor(ToggleRecording)
	reducer := &CaptureReducer{Policy: policy}
	if got := reducer.Event(0x7C, true); got.State != CaptureWaiting || reducer.Preview().String() != "F13" {
		t.Fatalf("F13 down = %#v, preview = %q", got, reducer.Preview().String())
	}
	if got := reducer.Event(0x7C, false); got.State != CaptureComplete || got.Chord.String() != "F13" {
		t.Fatalf("F13 up = %#v", got)
	}
}

func TestReducerEdgesRepeatAndModifierRelease(t *testing.T) {
	chord, _ := Parse("Ctrl+Shift+Space")
	r := Reducer{Chord: chord}
	for _, event := range []struct {
		vk   uint32
		down bool
		want Edge
	}{{0x11, true, NoEdge}, {0x10, true, NoEdge}, {0x20, true, Pressed}, {0x20, true, NoEdge}, {0x10, false, Released}, {0x20, false, NoEdge}} {
		if got := r.Event(event.vk, event.down); got != event.want {
			t.Fatalf("event %#x/%v = %v, want %v", event.vk, event.down, got, event.want)
		}
	}
}

func TestForceReleaseOnlyOnce(t *testing.T) {
	chord, _ := Parse("Ctrl+A")
	r := Reducer{Chord: chord}
	r.Event(0x11, true)
	r.Event('A', true)
	if r.ForceRelease() != Released || r.ForceRelease() != NoEdge {
		t.Fatal("forced release was not a single edge")
	}
}

func TestCaptureReducerCompletesOnPrimaryRelease(t *testing.T) {
	r := CaptureReducer{}
	for _, event := range []struct {
		vk    uint32
		down  bool
		state CaptureState
	}{{0x11, true, CaptureWaiting}, {0x10, true, CaptureWaiting}, {'D', true, CaptureWaiting}, {'D', true, CaptureWaiting}, {0x10, false, CaptureWaiting}, {'D', false, CaptureComplete}} {
		got := r.Event(event.vk, event.down)
		if got.State != event.state {
			t.Fatalf("event %#x/%v state = %v, want %v", event.vk, event.down, got.State, event.state)
		}
		if got.State == CaptureComplete && got.Chord.String() != "Ctrl+Shift+D" {
			t.Fatalf("captured chord = %q", got.Chord.String())
		}
	}
}

func TestCaptureReducerCancelAndReject(t *testing.T) {
	if got := (&CaptureReducer{}).Event(0x1B, true); got.State != CaptureCanceled {
		t.Fatalf("Escape state = %v", got.State)
	}
	for _, vk := range []uint32{'A', 0x7B, 0xBA} {
		got := (&CaptureReducer{}).Event(vk, true)
		if got.State != CaptureRejected || got.Err == nil {
			t.Fatalf("key %#x result = %#v", vk, got)
		}
	}
}
