package tts

import (
	"errors"

	"github.com/tnware/freehand-stt/internal/history"
	"github.com/tnware/freehand-stt/internal/settings"
)

func (s *Service) PlayHistoryEntry(id uint64, version history.HistoryTextVersion) error {
	if s.history == nil {
		return errors.New("transcript history is unavailable")
	}
	text, err := s.history.Text(id, version)
	if err != nil {
		return err
	}
	return s.start(text, SourceHistory, id, version)
}

func (s *Service) PlayFileTranscript() error {
	if s.texts.File == nil {
		return errors.New("audio file transcript playback is unavailable")
	}
	text, err := s.texts.File()
	if err != nil {
		return err
	}
	return s.start(text, SourceFile, 0, history.HistoryTextFinal)
}

// PlayVoiceTranscript plays the completed result the user selected, never a newer recording.
func (s *Service) PlayVoiceTranscript(generation uint64) error {
	if s.texts.Voice == nil {
		return errors.New("voice transcript playback is unavailable")
	}
	text, err := s.texts.Voice(generation)
	if err != nil {
		return err
	}
	return s.start(text, SourceVoice, 0, history.HistoryTextFinal)
}

func (s *Service) PreviewVoice(draft *settings.TextToSpeechPreview) error {
	return s.startWithPreview("This is Freehand's speech playback preview.", SourcePreview, 0, history.HistoryTextFinal, draft)
}

// SpeakText generates speech for bounded user-authored text from the
// first-class Text to speech workspace. It never writes to transcript history.
func (s *Service) SpeakText(text string) error {
	return s.start(text, SourceCompose, 0, history.HistoryTextFinal)
}
