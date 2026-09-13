package insertion

import (
	"context"
	"reflect"
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
func TestDarwinOpaqueTargetIdentity(t *testing.T) {
	want := Target{ProcessID: 42, ProcessCreationTime: 99}
	field := reflect.ValueOf(&want).Elem().FieldByName("DarwinToken")
	if !field.IsValid() || field.Kind() != reflect.Uint64 {
		t.Fatal("Target needs a comparable DarwinToken, not fabricated Windows handles")
	}
	field.SetUint(7)
	if !want.Valid() {
		t.Fatal("native Darwin identity should be valid")
	}
	f := &fake{target: want}
	if err := (Policy{f}).Deliver(context.Background(), want, "text", DirectInput); err != nil || f.inserts != 1 {
		t.Fatal("matching native identity was rejected", err)
	}
	field.SetUint(8)
	if err := (Policy{f}).Deliver(context.Background(), want, "text", DirectInput); err != ErrCopyRequired || f.inserts != 1 {
		t.Fatal("different native identity must fail closed")
	}
	want.HWND = 1
	if want.Valid() {
		t.Fatal("mixed-platform identities must fail closed")
	}
	for _, v := range []Target{{ProcessID: 42}, {ProcessCreationTime: 99}, {}} {
		if v.Valid() {
			t.Fatal("incomplete identity accepted")
		}
	}
}

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
