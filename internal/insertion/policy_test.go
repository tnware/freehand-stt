package insertion

import (
	"context"
	"testing"
)

type fake struct {
	target          Target
	insertTarget    Target
	inserts, copies int
}

func (f *fake) CaptureTarget() (Target, error) { return f.target, nil }
func (f *fake) Foreground() (Target, error)    { return f.target, nil }
func (f *fake) InsertUnicode(_ context.Context, target Target, _ string) error {
	f.inserts++
	f.insertTarget = target
	return nil
}
func (f *fake) Copy(context.Context, string) error { f.copies++; return nil }
func TestFocusPolicy(t *testing.T) {
	want := Target{HWND: 1, FocusHWND: 2, ThreadID: 3, ProcessID: 4, ProcessCreationTime: 5}
	f := &fake{target: Target{HWND: 9, FocusHWND: 2, ThreadID: 3, ProcessID: 4, ProcessCreationTime: 5}}
	_ = (Policy{f}).Deliver(context.Background(), want, "text", DirectInput)
	if f.inserts != 0 || f.copies != 0 {
		t.Fatal("changed focus inserted or copied implicitly")
	}
	f.target = want
	_ = (Policy{f}).Deliver(context.Background(), want, "text", DirectInput)
	if f.inserts != 1 || f.insertTarget != want {
		t.Fatal("same focus did not insert with the expected target identity")
	}
}

func TestInvalidTargetAndInsertFailureRequireExplicitCopy(t *testing.T) {
	f := &fake{}
	if e := (Policy{f}).Deliver(context.Background(), Target{}, "text", DirectInput); e != ErrCopyRequired || f.copies != 0 {
		t.Fatalf("invalid target result = %v, copies = %d", e, f.copies)
	}
}

func TestManualAndDeferredClipboardModesNeverDeliverAutomatically(t *testing.T) {
	want := Target{HWND: 1, FocusHWND: 2, ThreadID: 3, ProcessID: 4, ProcessCreationTime: 5}
	for _, mode := range []Mode{ManualCopy, ClipboardPaste, Mode("unknown")} {
		f := &fake{target: want}
		if err := (Policy{f}).Deliver(context.Background(), want, "text", mode); err != ErrCopyRequired {
			t.Fatalf("mode %q result = %v, want copy required", mode, err)
		}
		if f.inserts != 0 || f.copies != 0 {
			t.Fatalf("mode %q inserted=%d copied=%d", mode, f.inserts, f.copies)
		}
	}
}
