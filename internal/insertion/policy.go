package insertion

import (
	"context"
	"errors"
)

type Target struct {
	// DarwinToken is an opaque, process-local capture generation. Native AX
	// references remain owned by the capturing Input, never by this value.
	// Darwin identities use PID/start time and leave all Windows fields zero.
	DarwinToken         uint64
	HWND                uintptr
	FocusHWND           uintptr
	ThreadID            uint32
	ProcessID           uint32
	ProcessCreationTime uint64
}

func (t Target) Valid() bool {
	if t.DarwinToken != 0 {
		return t.HWND == 0 && t.FocusHWND == 0 && t.ThreadID == 0 && t.ProcessID != 0 && t.ProcessCreationTime != 0
	}
	return t.HWND != 0 && t.FocusHWND != 0 && t.ThreadID != 0 && t.ProcessID != 0 && t.ProcessCreationTime != 0
}

type Platform interface {
	CaptureTarget() (Target, error)
	Foreground() (Target, error)
	InsertUnicode(context.Context, Target, string) error
	Copy(context.Context, string) error
}

// Mode is the backend delivery policy selected for a completed microphone
// transcript. ClipboardPaste is reserved so the future compatibility mode has
// an explicit boundary, but Policy deliberately rejects it until complete
// clipboard snapshot and conditional restoration are implemented.
type Mode string

const (
	DirectInput    Mode = "direct-input"
	ManualCopy     Mode = "manual-copy"
	ClipboardPaste Mode = "clipboard-paste"
)

var ErrCopyRequired = errors.New("automatic insertion is unsafe; explicit copy required")

// Explicit clipboard outcomes are separate from automatic insertion policy.
var ErrCopyNotPerformed = errors.New("clipboard copy was not performed")
var ErrCopyAmbiguous = errors.New("clipboard copy may still complete; do not automatically retry")
var ErrCopyFailed = errors.New("clipboard copy failed; clipboard may have changed")

type Policy struct{ Platform Platform }

func (p Policy) Deliver(ctx context.Context, want Target, text string, mode Mode) error {
	if text == "" {
		return nil
	}
	if mode != DirectInput || !want.Valid() {
		return ErrCopyRequired
	}
	got, e := p.Platform.Foreground()
	if e != nil || !got.Valid() || got != want {
		return CopyRequired(e)
	}
	if e = p.Platform.InsertUnicode(ctx, want, text); e != nil {
		return CopyRequired(e)
	}
	return nil
}
