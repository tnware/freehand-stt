package audio

import (
	"encoding/binary"
	"errors"
)

const (
	defaultPreRollMilliseconds = 300
	defaultStartFrames         = 2
	MaxSegmentSeconds          = 240
)

type VoiceDetector interface {
	Speech([]int16) (bool, error)
	Close() error
}

// SpeechSegmenterOptions configures the one local speech-analysis pipeline.
// A zero TargetSeconds keeps one buffered utterance until Flush; a non-zero
// target emits ordered chunks after the target and a sustained pause.
type SpeechSegmenterOptions struct {
	TargetSeconds                     int
	SegmentSilenceMilliseconds        int
	ActivitySilenceMilliseconds       int
	TrimSilence                       bool
	SpeechPaddingMilliseconds         int
	AutoStopSilenceMilliseconds       int
	AutoStopMinimumSpeechMilliseconds int
}

type Segmenter struct {
	detector               VoiceDetector
	segmentSilenceFrames   int
	activitySilenceFrames  int
	paddingFrames          int
	startFrames            int
	targetBytes            int
	maxBytes               int
	trimSilence            bool
	preserveWholeRecording bool
	autoStopSilenceFrames  int
	autoStopSpeechFrames   int

	samples          []int16
	preRoll          []byte
	utterance        []byte
	consecutiveVoice int
	trailingSilence  int
	utteranceSpeech  bool
	lastVoiceEnd     int

	activityVoice   int
	activitySilence int
	activityKnown   bool
	activityActive  bool

	confirmedSpeech   bool
	totalVoiceFrames  int
	autoSilenceFrames int
}

// NewSegmenter preserves the original silence-splitting defaults for callers
// that do not need the broader speech policy controls.
func NewSegmenter(detector VoiceDetector, targetSeconds, silenceMilliseconds int) (*Segmenter, error) {
	return NewSpeechSegmenter(detector, SpeechSegmenterOptions{
		TargetSeconds:               targetSeconds,
		SegmentSilenceMilliseconds:  silenceMilliseconds,
		ActivitySilenceMilliseconds: silenceMilliseconds,
		SpeechPaddingMilliseconds:   defaultPreRollMilliseconds,
	})
}

func NewSpeechSegmenter(detector VoiceDetector, options SpeechSegmenterOptions) (*Segmenter, error) {
	if detector == nil {
		return nil, errors.New("voice detector is required")
	}
	if options.TargetSeconds < 0 || options.TargetSeconds >= MaxSegmentSeconds {
		return nil, errors.New("segment target must be below the hard segment limit")
	}
	if options.TargetSeconds > 0 && options.SegmentSilenceMilliseconds < VADFrameMilliseconds {
		return nil, errors.New("segment silence duration must be at least one VAD frame")
	}
	if options.ActivitySilenceMilliseconds < VADFrameMilliseconds {
		return nil, errors.New("activity silence duration must be at least one VAD frame")
	}
	if options.SpeechPaddingMilliseconds < 0 {
		return nil, errors.New("speech padding cannot be negative")
	}
	if options.AutoStopSilenceMilliseconds < 0 || options.AutoStopMinimumSpeechMilliseconds < 0 {
		return nil, errors.New("automatic stop durations cannot be negative")
	}
	if (options.AutoStopSilenceMilliseconds == 0) != (options.AutoStopMinimumSpeechMilliseconds == 0) {
		return nil, errors.New("automatic stop requires both silence and minimum speech durations")
	}

	paddingFrames := millisecondsToFrames(options.SpeechPaddingMilliseconds)
	preRollFrames := paddingFrames + defaultStartFrames
	segmenter := &Segmenter{
		detector:               detector,
		segmentSilenceFrames:   millisecondsToFrames(options.SegmentSilenceMilliseconds),
		activitySilenceFrames:  millisecondsToFrames(options.ActivitySilenceMilliseconds),
		paddingFrames:          paddingFrames,
		startFrames:            defaultStartFrames,
		trimSilence:            options.TrimSilence,
		preserveWholeRecording: options.TargetSeconds == 0 && !options.TrimSilence,
		autoStopSilenceFrames:  millisecondsToFrames(options.AutoStopSilenceMilliseconds),
		autoStopSpeechFrames:   millisecondsToFrames(options.AutoStopMinimumSpeechMilliseconds),
		samples:                make([]int16, VADFrameSamples),
		preRoll:                make([]byte, 0, preRollFrames*VADFrameBytes),
		utterance:              make([]byte, 0, 10*SampleRate*2),
	}
	if options.TargetSeconds > 0 {
		segmenter.targetBytes = options.TargetSeconds * SampleRate * 2
		segmenter.maxBytes = MaxSegmentSeconds * SampleRate * 2
	}
	return segmenter, nil
}

// Push consumes one 20 ms PCM16 frame and returns a complete utterance when a
// configured segment boundary is reached. Returned audio belongs to the caller.
func (s *Segmenter) Push(frame []byte) ([]byte, error) {
	if len(frame) != VADFrameBytes {
		return nil, errors.New("VAD frame has unexpected length")
	}
	for i := range s.samples {
		s.samples[i] = int16(binary.LittleEndian.Uint16(frame[i*2:]))
	}
	voice, err := s.detector.Speech(s.samples)
	if err != nil {
		return nil, err
	}
	s.observeDecision(voice)

	if s.preserveWholeRecording {
		s.utterance = append(s.utterance, frame...)
		if voice {
			s.utteranceSpeech = true
			s.lastVoiceEnd = len(s.utterance)
		}
		return nil, nil
	}

	if len(s.utterance) == 0 {
		s.pushPreRoll(frame)
		if voice {
			s.utteranceSpeech = true
		}
		if s.consecutiveVoice >= s.startFrames {
			start := 0
			if s.trimSilence {
				firstVoice := len(s.preRoll) - s.startFrames*VADFrameBytes
				start = firstVoice - s.paddingFrames*VADFrameBytes
				if start < 0 {
					start = 0
				}
			}
			s.utterance = append(s.utterance, s.preRoll[start:]...)
			s.lastVoiceEnd = len(s.utterance)
			s.zeroPreRoll()
			s.trailingSilence = 0
		}
		return nil, nil
	}

	s.utterance = append(s.utterance, frame...)
	if voice {
		s.utteranceSpeech = true
		s.trailingSilence = 0
		s.lastVoiceEnd = len(s.utterance)
	} else {
		s.trailingSilence++
	}
	if s.targetBytes > 0 && ((len(s.utterance) >= s.targetBytes && s.trailingSilence >= s.segmentSilenceFrames) || len(s.utterance) >= s.maxBytes) {
		return s.finish(), nil
	}
	return nil, nil
}

func (s *Segmenter) observeDecision(voice bool) {
	if voice {
		s.consecutiveVoice++
		s.autoSilenceFrames = 0
		if s.confirmedSpeech {
			s.totalVoiceFrames++
		} else if s.consecutiveVoice >= s.startFrames {
			s.confirmedSpeech = true
			s.totalVoiceFrames += s.startFrames
		}
	} else {
		s.consecutiveVoice = 0
		if s.confirmedSpeech {
			s.autoSilenceFrames++
		}
	}
	s.updateActivity(voice)
}

// VoiceActivity reports the stabilized detector decision. Speech activates
// after two positive frames. Silence activates only after the configured UI
// debounce, so ordinary gaps between words do not flicker the renderer state.
func (s *Segmenter) VoiceActivity() (active, known bool) {
	return s.activityActive, s.activityKnown
}

// AutoStopStatus exposes policy state without clocks. The coordinator turns
// remainingMilliseconds into a UI deadline, while this frame-based tracker
// remains the sole authority that actually stops capture.
func (s *Segmenter) AutoStopStatus() (enabled, armed, countdown bool, remainingMilliseconds int) {
	if s.autoStopSilenceFrames == 0 {
		return false, false, false, 0
	}
	armed = s.confirmedSpeech && s.totalVoiceFrames >= s.autoStopSpeechFrames
	remainingFrames := s.autoStopSilenceFrames - s.autoSilenceFrames
	if remainingFrames < 0 {
		remainingFrames = 0
	}
	countdown = armed && s.activityKnown && !s.activityActive && s.autoSilenceFrames > 0
	return true, armed, countdown, remainingFrames * VADFrameMilliseconds
}

func (s *Segmenter) AutoStopReady() bool {
	return s.autoStopSilenceFrames > 0 &&
		s.confirmedSpeech &&
		s.totalVoiceFrames >= s.autoStopSpeechFrames &&
		s.autoSilenceFrames >= s.autoStopSilenceFrames
}

func (s *Segmenter) Close() error {
	s.resetAll()
	return s.detector.Close()
}

func (s *Segmenter) Flush() []byte {
	if len(s.utterance) == 0 {
		if !s.utteranceSpeech {
			s.resetUtterance()
			return nil
		}
		s.utterance = append(s.utterance, s.preRoll...)
		s.lastVoiceEnd = len(s.utterance)
	}
	if !s.utteranceSpeech {
		s.resetUtterance()
		return nil
	}
	return s.finish()
}

func (s *Segmenter) pushPreRoll(frame []byte) {
	limit := (s.paddingFrames + s.startFrames) * VADFrameBytes
	if len(s.preRoll)+len(frame) > limit {
		drop := len(s.preRoll) + len(frame) - limit
		copy(s.preRoll, s.preRoll[drop:])
		s.preRoll = s.preRoll[:len(s.preRoll)-drop]
	}
	s.preRoll = append(s.preRoll, frame...)
}

func (s *Segmenter) finish() []byte {
	end := len(s.utterance)
	if s.trimSilence && s.lastVoiceEnd > 0 {
		end = s.lastVoiceEnd + s.paddingFrames*VADFrameBytes
		if end > len(s.utterance) {
			end = len(s.utterance)
		}
	}
	out := append([]byte(nil), s.utterance[:end]...)
	s.resetUtterance()
	return out
}

func (s *Segmenter) resetUtterance() {
	s.zeroPreRoll()
	for i := range s.utterance {
		s.utterance[i] = 0
	}
	s.utterance = s.utterance[:0]
	s.trailingSilence = 0
	s.utteranceSpeech = false
	s.lastVoiceEnd = 0
}

func (s *Segmenter) resetAll() {
	s.resetUtterance()
	s.consecutiveVoice = 0
	s.activityVoice = 0
	s.activitySilence = 0
	s.activityKnown = false
	s.activityActive = false
	s.confirmedSpeech = false
	s.totalVoiceFrames = 0
	s.autoSilenceFrames = 0
}

func (s *Segmenter) zeroPreRoll() {
	for i := range s.preRoll {
		s.preRoll[i] = 0
	}
	s.preRoll = s.preRoll[:0]
}

func (s *Segmenter) updateActivity(voice bool) {
	if voice {
		s.activityVoice++
		s.activitySilence = 0
		if s.activityVoice >= s.startFrames {
			s.activityKnown = true
			s.activityActive = true
		}
		return
	}
	s.activityVoice = 0
	s.activitySilence++
	if s.activitySilence >= s.activitySilenceFrames {
		s.activityKnown = true
		s.activityActive = false
	}
}

func millisecondsToFrames(milliseconds int) int {
	if milliseconds <= 0 {
		return 0
	}
	return (milliseconds + VADFrameMilliseconds - 1) / VADFrameMilliseconds
}
