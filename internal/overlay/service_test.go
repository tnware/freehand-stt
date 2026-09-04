package overlay

import (
	"context"
	"testing"
	"time"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/dictation"
	"github.com/tnware/freehand-stt/internal/platform"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type levelFake struct{}

func (*levelFake) TakeLevel() float64 { return 0 }

type statusOverlayFake struct {
	levels  platform.LevelSource
	options []platform.OverlayOptions
	updates []platform.OverlayStatus
	closes  int
}

func (o *statusOverlayFake) SetLevelSource(levels platform.LevelSource) { o.levels = levels }
func (o *statusOverlayFake) Configure(options platform.OverlayOptions) error {
	o.options = append(o.options, options)
	return nil
}
func (o *statusOverlayFake) Update(status platform.OverlayStatus) error {
	o.updates = append(o.updates, status)
	return nil
}
func (o *statusOverlayFake) Close() error { o.closes++; return nil }

func TestOverlayForStatus(t *testing.T) {
	cases := []struct {
		name   string
		status dictation.Status
		kind   platform.OverlayKind
	}{
		{"idle hides", dictation.Status{State: dictation.Idle}, platform.OverlayHidden},
		{"recording", dictation.Status{State: dictation.Recording}, platform.OverlayRecording},
		{"speech", dictation.Status{State: dictation.Recording, VADState: dictation.VADSpeech}, platform.OverlayRecordingSpeech},
		{"silence", dictation.Status{State: dictation.Recording, VADState: dictation.VADSilence}, platform.OverlayRecordingSilence},
		{"countdown", dictation.Status{State: dictation.Recording, AutoStopState: dictation.AutoStopCountdown}, platform.OverlayRecordingCountdown},
		{"transcribing", dictation.Status{State: dictation.Transcribing}, platform.OverlayTranscribing},
		{"post-processing", dictation.Status{State: dictation.PostProcessing}, platform.OverlayPostProcessing},
		{"ready", dictation.Status{State: dictation.Ready}, platform.OverlayReady},
		{"cancelling", dictation.Status{State: dictation.Cancelling}, platform.OverlayCancelling},
		{"ordinary failure", dictation.Status{State: dictation.Failed}, platform.OverlayFailed},
		{"copy required", dictation.Status{State: dictation.Failed, CanCopy: true}, platform.OverlayCopyRequired},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := overlayForStatus(test.status)
			if got.Kind != test.kind {
				t.Fatalf("overlay kind = %v, want %v", got.Kind, test.kind)
			}
		})
	}
}

func TestOptionsMapAllCuratedPreferences(t *testing.T) {
	preferences := config.OverlayPreferences{
		Layout: config.OverlayLayoutDetailed, Anchor: config.OverlayAnchorBottomRight,
		Visibility: config.OverlayVisibilityActive, Motion: config.OverlayMotionReduced,
		Surface: config.OverlaySurfaceSolid, Visualizer: config.OverlayVisualizerEnvelope,
		SizePercent: 125, OpacityPercent: 80, EdgeOffset: 42, GlowPercent: 50,
	}
	want := platform.OverlayOptions{
		Layout: platform.OverlayLayoutDetailed, Anchor: platform.OverlayAnchorBottomRight,
		Motion: platform.OverlayMotionReduced, Surface: platform.OverlaySurfaceSolid,
		Visualizer: platform.OverlayVisualizerEnvelope,
		Scale:      1.25, Opacity: 0.8, EdgeOffset: 42, Glow: 0.5,
	}
	if got := optionsFromPreferences(preferences); got != want {
		t.Fatalf("overlay options = %#v, want %#v", got, want)
	}
}

func TestServiceOwnsNativeLifecycleAndPreservesRunMetadata(t *testing.T) {
	settings := config.Default()
	var created []*statusOverlayFake
	levels := &levelFake{}
	service := NewService(settings, func() platform.LevelSource { return levels }, nil)
	service.newOverlay = func() (statusOverlay, error) {
		overlay := &statusOverlayFake{}
		created = append(created, overlay)
		return overlay, nil
	}
	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	Start(service)
	Start(service)
	if len(created) != 1 || created[0].levels != levels {
		t.Fatalf("native start created=%d levels=%#v", len(created), created[0].levels)
	}

	started := time.Now().Add(-time.Second)
	ApplyStatus(service, dictation.Status{
		State: dictation.Recording, Generation: 4, StartedAt: started,
		RecordingMode: dictation.RecordingToggle, SegmentNumber: 2, VADState: dictation.VADSpeech,
	})
	ApplyStatus(service, dictation.Status{State: dictation.Transcribing, Generation: 4})
	last := created[0].updates[len(created[0].updates)-1]
	if last.Kind != platform.OverlayTranscribing || last.Generation != 4 || !last.StartedAt.Equal(started) || last.Checkpoints != 2 || last.Shortcut == "" {
		t.Fatalf("transcription presentation lost run metadata: %#v", last)
	}

	settings.OverlayEnabled = false
	ApplySettings(service, settings)
	deadline := time.Now().Add(time.Second)
	for created[0].closes != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if created[0].closes != 1 {
		t.Fatalf("disabled overlay close count = %d", created[0].closes)
	}
	if err := service.ServiceShutdown(); err != nil {
		t.Fatal(err)
	}
}

func TestPreviewUsesDraftPreferencesAndRealDictationPreemptsIt(t *testing.T) {
	settings := config.Default()
	settings.OverlayEnabled = false
	fake := &statusOverlayFake{}
	service := NewService(settings, nil, nil)
	service.newOverlay = func() (statusOverlay, error) { return fake, nil }
	_ = service.ServiceStartup(context.Background(), application.ServiceOptions{})
	Start(service)
	request := PreviewRequest{
		Preferences:    settings.OverlayPreferences(),
		ToggleShortcut: settings.ToggleShortcut,
		HoldShortcut:   settings.HoldShortcut,
	}
	request.Preferences.Layout = config.OverlayLayoutMeter
	request.Preferences.Anchor = config.OverlayAnchorBottomCenter
	if err := service.StartPreview(request); err != nil {
		t.Fatal(err)
	}
	if len(fake.options) == 0 || fake.options[len(fake.options)-1].Layout != platform.OverlayLayoutMeter {
		t.Fatalf("preview options = %#v", fake.options)
	}
	preview := fake.updates[len(fake.updates)-1]
	if !preview.Preview || preview.Kind != platform.OverlayRecordingSpeech || preview.Shortcut == "" {
		t.Fatalf("initial preview = %#v", preview)
	}

	ApplyStatus(service, dictation.Status{State: dictation.Recording, Generation: 8, StartedAt: time.Now()})
	service.mu.Lock()
	previewing := service.preview
	service.mu.Unlock()
	if previewing {
		t.Fatal("real dictation did not preempt the native preview")
	}
	if fake.closes != 1 {
		t.Fatalf("disabled applied configuration left the preview surface open: closes=%d", fake.closes)
	}
	_ = service.ServiceShutdown()
}

func TestVisibilityPolicyIsBounded(t *testing.T) {
	if !visibilityIncludes(config.OverlayVisibilityRecording, platform.OverlayRecordingSpeech) {
		t.Fatal("recording-only visibility hid recording")
	}
	if visibilityIncludes(config.OverlayVisibilityRecording, platform.OverlayTranscribing) {
		t.Fatal("recording-only visibility showed transcription")
	}
	if !visibilityIncludes(config.OverlayVisibilityActive, platform.OverlayPostProcessing) || visibilityIncludes(config.OverlayVisibilityActive, platform.OverlayReady) {
		t.Fatal("active visibility does not separate processing from outcomes")
	}
	if !visibilityIncludes(config.OverlayVisibilityAll, platform.OverlayCopyRequired) {
		t.Fatal("all-phase visibility hid a terminal outcome")
	}
}
