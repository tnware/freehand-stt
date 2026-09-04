package insertion

import (
	"context"
	"errors"
)

type Target struct {
	HWND                uintptr
	FocusHWND           uintptr
	ThreadID            uint32
	ProcessID           uint32
	ProcessCreationTime uint64
}

func (t Target) Valid() bool {
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
		return ErrCopyRequired
	}
	if e = p.Platform.InsertUnicode(ctx, want, text); e != nil {
		return ErrCopyRequired
	}
	return nil
}
