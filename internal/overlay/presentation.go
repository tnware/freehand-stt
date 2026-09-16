package overlay

import (
	"time"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/dictation"
	"github.com/tnware/freehand-stt/internal/hotkey"
	nativeoverlay "github.com/tnware/freehand-stt/internal/platform/overlay"
)

type runPresentation struct {
	generation  uint64
	startedAt   time.Time
	finishedAt  time.Time
	mode        nativeoverlay.OverlayRecordingMode
	shortcut    string
	checkpoints int
}

func (s *Service) trackRunLocked(status dictation.Status) {
	if status.State == dictation.Recording && s.run.generation != status.Generation {
		s.run = runPresentation{
			generation: status.Generation,
			startedAt:  status.StartedAt,
			mode:       overlayRecordingMode(status.RecordingMode),
			shortcut:   shortcutForMode(s.settings, status.RecordingMode),
		}
	}
	if status.State == dictation.Recording && status.SegmentNumber > s.run.checkpoints {
		s.run.checkpoints = status.SegmentNumber
	}
	if (status.State == dictation.Ready || status.State == dictation.Failed) && s.run.finishedAt.IsZero() {
		s.run.finishedAt = time.Now()
	}
}

func (s *Service) presentationLocked(status dictation.Status) nativeoverlay.OverlayStatus {
	presentation := overlayForStatus(status)
	if s.outcomeDismissed || !visibilityIncludes(s.settings.OverlayVisibility, presentation.Kind) {
		presentation.Kind = nativeoverlay.OverlayHidden
	}
	if status.StartRejected {
		// No new run started. Do not present the retained result's timing,
		// shortcut or checkpoints as metadata for the rejected attempt.
		return presentation
	}
	if presentation.Generation == 0 {
		presentation.Generation = s.run.generation
	}
	presentation.StartedAt = s.run.startedAt
	presentation.FinishedAt = s.run.finishedAt
	presentation.RecordingMode = s.run.mode
	presentation.Shortcut = s.run.shortcut
	presentation.Checkpoints = s.run.checkpoints
	return presentation
}

func overlayForStatus(status dictation.Status) nativeoverlay.OverlayStatus {
	kind := nativeoverlay.OverlayHidden
	switch status.State {
	case dictation.Recording:
		switch {
		case status.AutoStopState == dictation.AutoStopCountdown:
			kind = nativeoverlay.OverlayRecordingCountdown
		case status.VADState == dictation.VADSpeech:
			kind = nativeoverlay.OverlayRecordingSpeech
		case status.VADState == dictation.VADSilence:
			kind = nativeoverlay.OverlayRecordingSilence
		default:
			kind = nativeoverlay.OverlayRecording
		}
	case dictation.Transcribing:
		kind = nativeoverlay.OverlayTranscribing
	case dictation.PostProcessing:
		kind = nativeoverlay.OverlayPostProcessing
	case dictation.Ready:
		kind = nativeoverlay.OverlayReady
	case dictation.Cancelling:
		kind = nativeoverlay.OverlayCancelling
	case dictation.Failed:
		if status.CanCopy && !status.StartRejected {
			kind = nativeoverlay.OverlayCopyRequired
		} else {
			kind = nativeoverlay.OverlayFailed
		}
	}
	captionEnabled := status.Live && status.LiveCaptions && (status.State == dictation.Recording || status.State == dictation.Transcribing)
	caption := ""
	if captionEnabled {
		caption = nativeoverlay.BoundedOverlayCaption(status.LiveFinal + " " + status.LivePartial)
	}
	return nativeoverlay.OverlayStatus{
		CaptionEnabled:    captionEnabled,
		Caption:           caption,
		Kind:              kind,
		Generation:        status.Generation,
		CountdownDeadline: status.AutoStopDeadline,
		CountdownDuration: time.Duration(status.AutoStopDurationMilliseconds) * time.Millisecond,
	}
}

func optionsFromPreferences(preferences config.OverlayPreferences) nativeoverlay.OverlayOptions {
	return nativeoverlay.OverlayOptions{
		Layout:     overlayLayout(preferences.Layout),
		Anchor:     overlayAnchor(preferences.Anchor),
		Motion:     overlayMotion(preferences.Motion),
		Surface:    overlaySurface(preferences.Surface),
		Visualizer: overlayVisualizer(preferences.Visualizer),
		Scale:      float64(preferences.SizePercent) / 100,
		Opacity:    float64(preferences.OpacityPercent) / 100,
		EdgeOffset: int32(preferences.EdgeOffset),
		Glow:       float64(preferences.GlowPercent) / 100,
	}
}

func overlayLayout(value config.OverlayLayout) nativeoverlay.OverlayLayout {
	switch value {
	case config.OverlayLayoutMinimal:
		return nativeoverlay.OverlayLayoutMinimal
	case config.OverlayLayoutMeter:
		return nativeoverlay.OverlayLayoutMeter
	case config.OverlayLayoutDetailed:
		return nativeoverlay.OverlayLayoutDetailed
	default:
		return nativeoverlay.OverlayLayoutCapsule
	}
}

func overlayAnchor(value config.OverlayAnchor) nativeoverlay.OverlayAnchor {
	switch value {
	case config.OverlayAnchorTopLeft:
		return nativeoverlay.OverlayAnchorTopLeft
	case config.OverlayAnchorTopRight:
		return nativeoverlay.OverlayAnchorTopRight
	case config.OverlayAnchorBottomLeft:
		return nativeoverlay.OverlayAnchorBottomLeft
	case config.OverlayAnchorBottomCenter:
		return nativeoverlay.OverlayAnchorBottomCenter
	case config.OverlayAnchorBottomRight:
		return nativeoverlay.OverlayAnchorBottomRight
	default:
		return nativeoverlay.OverlayAnchorTopCenter
	}
}

func overlayMotion(value config.OverlayMotion) nativeoverlay.OverlayMotion {
	if value == config.OverlayMotionReduced {
		return nativeoverlay.OverlayMotionReduced
	}
	return nativeoverlay.OverlayMotionSystem
}

func overlaySurface(value config.OverlaySurface) nativeoverlay.OverlaySurface {
	switch value {
	case config.OverlaySurfaceSolid:
		return nativeoverlay.OverlaySurfaceSolid
	case config.OverlaySurfaceMinimal:
		return nativeoverlay.OverlaySurfaceMinimal
	default:
		return nativeoverlay.OverlaySurfaceGlass
	}
}

func overlayVisualizer(value config.OverlayVisualizer) nativeoverlay.OverlayVisualizer {
	switch value {
	case config.OverlayVisualizerPulse:
		return nativeoverlay.OverlayVisualizerPulse
	case config.OverlayVisualizerEnvelope:
		return nativeoverlay.OverlayVisualizerEnvelope
	case config.OverlayVisualizerMeter:
		return nativeoverlay.OverlayVisualizerMeter
	default:
		return nativeoverlay.OverlayVisualizerBars
	}
}

func overlayRecordingMode(mode dictation.RecordingMode) nativeoverlay.OverlayRecordingMode {
	if mode == dictation.RecordingHold {
		return nativeoverlay.OverlayRecordingHold
	}
	return nativeoverlay.OverlayRecordingToggle
}

func shortcutForMode(settings config.Settings, mode dictation.RecordingMode) string {
	value := settings.ToggleShortcut
	action := hotkey.ToggleRecording
	if mode == dictation.RecordingHold && settings.HoldShortcut != "" {
		value = settings.HoldShortcut
		action = hotkey.HoldToTalk
	}
	normalized, err := normalizeShortcut(action, value)
	if err != nil {
		return ""
	}
	return normalized
}

func normalizeShortcut(action hotkey.ShortcutAction, value string) (string, error) {
	return hotkey.NormalizeFor(action, value)
}

func dictationActive(status dictation.Status) bool {
	return status.State != dictation.Idle && status.State != dictation.Failed
}

func visibilityIncludes(visibility config.OverlayVisibility, kind nativeoverlay.OverlayKind) bool {
	switch kind {
	case nativeoverlay.OverlayRecording, nativeoverlay.OverlayRecordingSpeech, nativeoverlay.OverlayRecordingSilence, nativeoverlay.OverlayRecordingCountdown:
		return true
	case nativeoverlay.OverlayTranscribing, nativeoverlay.OverlayPostProcessing, nativeoverlay.OverlayCancelling:
		return visibility != config.OverlayVisibilityRecording
	case nativeoverlay.OverlayReady, nativeoverlay.OverlayCopyRequired, nativeoverlay.OverlayFailed:
		return visibility == config.OverlayVisibilityAll
	default:
		return false
	}
}
