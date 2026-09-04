//go:build windows

package platform

import (
	"testing"
	"time"
)

type stubLevels struct{ level float64 }

func (s *stubLevels) TakeLevel() float64 { return s.level }

// Without a capture attached the overlay must keep its clock-driven animation
// rather than going flat.
func TestOverlayLevelsDegradeWithoutASource(t *testing.T) {
	overlay := &StatusOverlay{}
	if got := overlay.liveLevels(true); got != nil {
		t.Fatalf("levels appeared without a source: %v", got)
	}
	overlay.SetLevelSource(nil)
	if overlay.levelSource() != nil {
		t.Fatal("a nil source was stored")
	}
	// Sampling without a source must be inert rather than panic.
	overlay.sampleLevels(time.Now(), true)
}

func TestOverlayLevelsRecordAndResetBetweenRecordings(t *testing.T) {
	overlay := &StatusOverlay{}
	overlay.SetLevelSource(&stubLevels{level: 0.8})

	now := time.Now()
	overlay.sampleLevels(now, true)
	levels := overlay.liveLevels(true)
	if len(levels) != overlayLevelBars {
		t.Fatalf("history had %d bars, want %d", len(levels), overlayLevelBars)
	}
	if levels[len(levels)-1] <= 0 {
		t.Fatal("the newest reading was silent despite a loud source")
	}

	// Readings are taken on their own cadence, not once per frame.
	overlay.sampleLevels(now.Add(5*time.Millisecond), true)
	if levels[len(levels)-2] != 0 {
		t.Fatal("a second reading was taken inside the sampling interval")
	}
	overlay.sampleLevels(now.Add(overlayLevelMS*time.Millisecond), true)
	levels = overlay.liveLevels(true)
	if levels[len(levels)-2] == 0 {
		t.Fatal("no reading was taken once the interval had passed")
	}

	// Leaving the recording state clears the meter, so the next recording
	// starts from silence rather than inheriting the tail of the last.
	overlay.sampleLevels(now, false)
	if got := overlay.liveLevels(false); got != nil {
		t.Fatalf("levels survived the end of recording: %v", got)
	}
	for index, level := range overlay.liveLevels(true) {
		if level != 0 {
			t.Fatalf("stale history remained at %d: %v", index, level)
		}
	}
}

func TestCuratedOverlayLayoutsKeepCapsuleAsTheExactDefault(t *testing.T) {
	defaultGeometry := overlayLayout(DefaultOverlayOptions().Layout, 1, 0)
	if defaultGeometry.pillWidth != overlayPillWidth || defaultGeometry.pillHeight != overlayPillHeight ||
		defaultGeometry.stageX != defaultGeometry.pillX+overlayStageLeft || defaultGeometry.stageWidth != overlayPillWidth-overlayStageLeft-overlayStageRight {
		t.Fatalf("default capsule geometry drifted: %#v", defaultGeometry)
	}

	sizes := map[OverlayLayout][2]int32{
		OverlayLayoutMinimal:  {104, 114},
		OverlayLayoutCapsule:  {256, 110},
		OverlayLayoutMeter:    {352, 120},
		OverlayLayoutDetailed: {430, 212},
	}
	for layout, want := range sizes {
		geometry := overlayLayout(layout, 1, 0)
		if geometry.windowWidth != want[0] || geometry.windowHigh != want[1] {
			t.Errorf("layout %d window = %dx%d, want %dx%d", layout, geometry.windowWidth, geometry.windowHigh, want[0], want[1])
		}
	}
}

func TestEveryAnchorKeepsTheVisibleBodyInsideTheMonitorWorkArea(t *testing.T) {
	work := nativeRect{Left: 100, Top: 50, Right: 1100, Bottom: 850}
	geometry := overlayLayout(OverlayLayoutCapsule, 1, 0)
	for anchor := OverlayAnchorTopLeft; anchor <= OverlayAnchorBottomRight; anchor++ {
		options := DefaultOverlayOptions()
		options.Anchor = anchor
		destination := overlayDestination(work, geometry, options)
		left := destination.X + int32(geometry.pillX)
		top := destination.Y + int32(geometry.pillY)
		right := left + int32(geometry.pillWidth)
		bottom := top + int32(geometry.pillHeight)
		if left < work.Left || top < work.Top || right > work.Right || bottom > work.Bottom {
			t.Errorf("anchor %d placed body %d,%d-%d,%d outside work area %#v", anchor, left, top, right, bottom, work)
		}
	}
}

func TestDetailedCopyUsesOnlyFixedOperationalLabels(t *testing.T) {
	for kind, want := range map[OverlayKind]string{
		OverlayRecordingSpeech:    "Listening",
		OverlayRecordingSilence:   "Silence detected",
		OverlayRecordingCountdown: "Silence countdown",
		OverlayTranscribing:       "Transcribing",
		OverlayPostProcessing:     "Cleaning up",
		OverlayReady:              "Ready",
		OverlayCopyRequired:       "Copy required",
		OverlayFailed:             "Something went wrong",
		OverlayCancelling:         "Cancelling",
	} {
		view := resolveOverlayView(OverlayStatus{Kind: kind})
		phase, instruction := overlayPhaseText(view)
		if phase != want || instruction == "" {
			t.Errorf("kind %d copy = %q / %q, want phase %q", kind, phase, instruction, want)
		}
	}
	if got := recordingInstruction(OverlayRecordingHold); got != "Release the shortcut to finish" {
		t.Fatalf("hold instruction = %q", got)
	}
	if got := formatOverlayElapsed(overlayView{StartedAt: time.Unix(100, 0)}, time.Unix(165, 0)); got != "1:05 elapsed" {
		t.Fatalf("elapsed label = %q", got)
	}
}
