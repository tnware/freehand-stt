package insertion

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

type rejectingPlatform struct {
	fake
	foregroundErr, sendErr error
}

func (f *rejectingPlatform) Foreground() (Target, error) { return f.target, f.foregroundErr }
func (f *rejectingPlatform) InsertUnicode(context.Context, Target, string) error {
	f.inserts++
	return f.sendErr
}

func TestBoundedRejectionSurvivesPolicy(t *testing.T) {
	want := Target{DarwinToken: 1, ProcessID: 42, ProcessCreationTime: 99}
	for _, stage := range []Stage{Capture, Validate, Send} {
		err := NewRejection(stage, "value_not_settable")
		if !errors.Is(err, ErrCopyRequired) {
			t.Fatal("lost sentinel")
		}
		expected := "Automatic insertion unavailable (" + string(stage) + ": value_not_settable). Copy the transcript and paste it where you want it."
		if stage != Capture {
			expected += " Check the target for any text already inserted before pasting."
		}
		if got := CopyRequiredMessage(fmt.Errorf("private wrapper: %w", err)); got != expected {
			t.Fatalf("message=%q", got)
		}
		if stage == Capture {
			continue
		}
		p := &rejectingPlatform{fake: fake{target: want}}
		if stage == Validate {
			p.foregroundErr = err
		} else {
			p.sendErr = err
		}
		got := (Policy{p}).Deliver(context.Background(), want, "private transcript", DirectInput)
		if !errors.Is(got, ErrCopyRequired) || CopyRequiredMessage(got) != expected || p.copies != 0 {
			t.Fatalf("lost bounded rejection: %v", got)
		}
	}
}

func TestAttributeRejectionCategories(t *testing.T) {
	for _, field := range []string{"focused_app", "focused_element", "focused_window", "element_window", "role", "enabled"} {
		for _, kind := range []string{"cannot_complete", "unsupported", "no_value", "invalid_element", "api_disabled", "not_implemented", "failure", "other"} {
			reason := field + "__" + kind
			if err := NewRejection(Capture, reason); err == ErrCopyRequired {
				t.Fatalf("lost diagnostic %s", reason)
			}
		}
	}
	for _, reason := range []string{"private__failure", "role__private", "role__failure__private", "role"} {
		if NewRejection(Capture, reason) != ErrCopyRequired {
			t.Fatal("unbounded category accepted")
		}
	}
}

func TestUnknownRejectionsNeverExposeRawData(t *testing.T) {
	generic := "Automatic insertion unavailable. Copy the transcript and paste it where you want it. Check the target for any text already inserted before pasting."
	for _, err := range []error{nil, errors.New("private target text"), NewRejection(Stage("private"), "value_not_settable"), NewRejection(Capture, "private text"), ErrCopyRequired} {
		if got := CopyRequiredMessage(err); got != generic {
			t.Fatalf("unsafe message=%q", got)
		}
	}
	want := Target{DarwinToken: 1, ProcessID: 42, ProcessCreationTime: 99}
	for _, sending := range []bool{false, true} {
		p := &rejectingPlatform{fake: fake{target: want}}
		if sending {
			p.sendErr = errors.New("private send error")
		} else {
			p.foregroundErr = errors.New("private AX error")
		}
		got := (Policy{p}).Deliver(context.Background(), want, "secret", DirectInput)
		if got != ErrCopyRequired {
			t.Fatalf("raw error retained: %v", got)
		}
	}
}
