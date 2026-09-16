package dictation

import (
	"time"
)

type State string

type SegmentPhase string

type VADState string

type AutoStopState string

type RecordingMode string

const (
	Idle           State = "idle"
	Recording      State = "recording"
	Transcribing   State = "transcribing"
	PostProcessing State = "post-processing"
	Ready          State = "ready-to-insert"
	Cancelling     State = "cancelling"
	Failed         State = "failed"
)

const (
	SegmentTranscribing SegmentPhase = "transcribing"
	SegmentCompleted    SegmentPhase = "completed"
)

const (
	VADSpeech  VADState = "speech"
	VADSilence VADState = "silence"
)

const (
	AutoStopWaiting   AutoStopState = "waiting-for-speech"
	AutoStopListening AutoStopState = "listening"
	AutoStopCountdown AutoStopState = "countdown"
)

const (
	RecordingToggle RecordingMode = "toggle"
	RecordingHold   RecordingMode = "hold"
)

type Status struct {
	Live                         bool          `json:"live"`
	LiveCaptions                 bool          `json:"liveCaptions"`
	LiveFinal                    string        `json:"liveFinal,omitempty"`
	LivePartial                  string        `json:"livePartial,omitempty"`
	Transcript                   string        `json:"transcript,omitempty"` // Current result only; cleared by the next recording, Clear, or shutdown.
	State                        State         `json:"state"`
	RecordingMode                RecordingMode `json:"recordingMode,omitempty"`
	Generation                   uint64        `json:"generation"`
	StartedAt                    time.Time     `json:"startedAt,omitempty"`
	Message                      string        `json:"message,omitempty"`
	StartRejected                bool          `json:"startRejected,omitempty"` // Admission feedback; the retained result generation is unchanged.
	CanCancel                    bool          `json:"canCancel"`
	CanCopy                      bool          `json:"canCopy"`
	SegmentNumber                int           `json:"segmentNumber,omitempty"`
	SegmentPhase                 SegmentPhase  `json:"segmentPhase,omitempty"`
	VADState                     VADState      `json:"vadState,omitempty"`
	AutoStopState                AutoStopState `json:"autoStopState,omitempty"`
	AutoStopDeadline             time.Time     `json:"autoStopDeadline,omitempty"`
	AutoStopDurationMilliseconds int           `json:"autoStopDurationMilliseconds,omitempty"`
}

func (c *recorder) snapshotLocked() Status { return c.status }
func (c *recorder) publish(s Status) {
	if c.closed.Load() {
		return
	}
	if c.changed != nil {
		c.changed(s)
	}
}

func (c *recorder) currentStatus() Status { c.mu.Lock(); defer c.mu.Unlock(); return c.status }

func (c *recorder) publishVADState(gen uint64, active bool) {
	state := VADSilence
	if active {
		state = VADSpeech
	}
	c.mu.Lock()
	if c.closed.Load() || c.status.Generation != gen || c.status.State != Recording || c.status.VADState == state {
		c.mu.Unlock()
		return
	}
	c.status.VADState = state
	s := c.status
	c.mu.Unlock()
	c.publish(s)
}

func (c *recorder) publishAutoStopState(gen uint64, armed, countdown bool, remainingMilliseconds int) {
	state := AutoStopWaiting
	deadline := time.Time{}
	if armed {
		state = AutoStopListening
	}
	if countdown {
		state = AutoStopCountdown
		deadline = time.Now().UTC().Add(time.Duration(remainingMilliseconds) * time.Millisecond)
	}
	c.mu.Lock()
	if c.closed.Load() || c.status.Generation != gen || c.status.State != Recording ||
		(c.status.AutoStopState == state && c.status.AutoStopDeadline.Equal(deadline)) {
		c.mu.Unlock()
		return
	}
	c.status.AutoStopState = state
	c.status.AutoStopDeadline = deadline
	s := c.status
	c.mu.Unlock()
	c.publish(s)
}

func (c *recorder) publishSegmentProgress(gen uint64, segment int, phase SegmentPhase) {
	c.mu.Lock()
	if c.closed.Load() || c.status.Generation != gen || c.status.State != Recording {
		c.mu.Unlock()
		return
	}
	c.status.SegmentNumber = segment
	c.status.SegmentPhase = phase
	s := c.status
	c.mu.Unlock()
	c.publish(s)
}
