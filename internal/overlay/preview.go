package overlay

import (
	"context"
	"errors"
	"time"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/hotkey"
	nativeoverlay "github.com/tnware/freehand-stt/internal/platform/overlay"
)

type PreviewRequest struct {
	Preferences    config.OverlayPreferences `json:"preferences"`
	ToggleShortcut string                    `json:"toggleShortcut"`
	HoldShortcut   string                    `json:"holdShortcut,omitempty"`
}

// StartPreview starts or updates the native preview. It accepts only the
// validated presentation subset and normalized shortcut labels; it neither
// saves settings nor affects dictation state.
func (s *Service) StartPreview(request PreviewRequest) error {
	if err := config.ValidateOverlayPreferences(request.Preferences); err != nil {
		return err
	}
	toggle, err := normalizeShortcut(hotkey.ToggleRecording, request.ToggleShortcut)
	if err != nil {
		return errors.New("overlay preview toggle shortcut is invalid")
	}
	hold := ""
	if request.HoldShortcut != "" {
		hold, err = normalizeShortcut(hotkey.HoldToTalk, request.HoldShortcut)
		if err != nil {
			return errors.New("overlay preview hold shortcut is invalid")
		}
	}
	request.ToggleShortcut = toggle
	request.HoldShortcut = hold

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || !s.started {
		return errors.New("native status overlay preview is unavailable")
	}
	if dictationActive(s.status) {
		return errors.New("finish the current dictation before previewing the overlay")
	}
	if _, err := s.ensureOverlayLocked(); err != nil {
		return errors.New("native status overlay preview could not be started")
	}
	s.previewRequest = request
	s.configureLocked(optionsFromPreferences(request.Preferences))
	if s.preview {
		s.updateLocked(s.previewPresentationLocked(s.previewKind))
		return nil
	}

	s.preview = true
	s.previewVersion++
	version := s.previewVersion
	ctx, cancel := context.WithCancel(s.rootContext)
	s.previewCancel = cancel
	s.previewKind = nativeoverlay.OverlayRecordingSpeech
	s.updateLocked(s.previewPresentationLocked(s.previewKind))
	s.previewWG.Add(1)
	go s.runPreview(ctx, version)
	s.logger.Info("native status overlay preview started")
	return nil
}

func (s *Service) StopPreview() error {
	s.mu.Lock()
	wasPreviewing := s.preview
	s.stopPreviewLocked()
	if !wasPreviewing {
		s.mu.Unlock()
		return nil
	}
	if s.settings.OverlayEnabled {
		s.configureLocked(optionsFromPreferences(s.settings.OverlayPreferences()))
		s.updateLocked(s.presentationLocked(s.status))
		s.mu.Unlock()
	} else {
		overlay := s.overlay
		s.overlay = nil
		s.mu.Unlock()
		if err := s.closeOverlay(overlay); err != nil {
			return err
		}
	}
	s.logger.Info("native status overlay preview stopped")
	return nil
}

func (s *Service) runPreview(ctx context.Context, version uint64) {
	defer s.previewWG.Done()
	kinds := []nativeoverlay.OverlayKind{
		nativeoverlay.OverlayRecordingSpeech,
		nativeoverlay.OverlayRecordingSilence,
		nativeoverlay.OverlayRecordingCountdown,
		nativeoverlay.OverlayTranscribing,
		nativeoverlay.OverlayPostProcessing,
		nativeoverlay.OverlayReady,
		nativeoverlay.OverlayCopyRequired,
		nativeoverlay.OverlayFailed,
	}
	ticker := time.NewTicker(previewStepDuration)
	defer ticker.Stop()
	index := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			if !s.preview || s.previewVersion != version || s.closed {
				s.mu.Unlock()
				return
			}
			visible := previewKinds(s.previewRequest.Preferences.Visibility, kinds)
			if len(visible) == 0 {
				s.mu.Unlock()
				continue
			}
			index = (index + 1) % len(visible)
			s.previewKind = visible[index]
			s.updateLocked(s.previewPresentationLocked(s.previewKind))
			s.mu.Unlock()
		}
	}
}

func (s *Service) stopPreviewLocked() {
	if !s.preview {
		return
	}
	s.preview = false
	s.previewVersion++
	if s.previewCancel != nil {
		s.previewCancel()
	}
	s.previewCancel = nil
}

func (s *Service) previewPresentationLocked(kind nativeoverlay.OverlayKind) nativeoverlay.OverlayStatus {
	now := time.Now()
	status := nativeoverlay.OverlayStatus{
		Kind:          kind,
		Generation:    ^uint64(0),
		Preview:       true,
		StartedAt:     now.Add(-12 * time.Second),
		RecordingMode: nativeoverlay.OverlayRecordingToggle,
		Shortcut:      s.previewRequest.ToggleShortcut,
		Checkpoints:   2,
	}
	if kind == nativeoverlay.OverlayRecordingCountdown {
		status.CountdownDuration = 3 * time.Second
		status.CountdownDeadline = now.Add(2 * time.Second)
	}
	return status
}

func previewKinds(visibility config.OverlayVisibility, kinds []nativeoverlay.OverlayKind) []nativeoverlay.OverlayKind {
	result := make([]nativeoverlay.OverlayKind, 0, len(kinds))
	for _, kind := range kinds {
		if visibilityIncludes(visibility, kind) {
			result = append(result, kind)
		}
	}
	return result
}
