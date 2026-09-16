//go:build darwin

package audio

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMicrophoneRequestCanceledBeforeNativeAccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := RequestMicrophoneAuthorization(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("request = %v, want cancellation", err)
	}
}

func TestMicrophoneRequestTransitionsAndTerminates(t *testing.T) {
	for _, initial := range []string{"authorized", "denied", "restricted", "not-determined"} {
		t.Run(initial, func(t *testing.T) {
			state, requests := initial, 0
			err := requestMicrophoneAuthorization(context.Background(), func() string { return state }, func() bool {
				requests++
				state = "authorized"
				return true
			}, time.Second)
			if (initial == "authorized" || initial == "not-determined") != (err == nil) {
				t.Fatalf("request = %v", err)
			}
			want := 0
			if initial == "not-determined" {
				want = 1
			}
			if requests != want {
				t.Fatalf("native requests = %d, want %d", requests, want)
			}
		})
	}
}

func TestMicrophoneRequestIsBoundedWhenUserDoesNotAnswer(t *testing.T) {
	requests := 0
	err := requestMicrophoneAuthorization(context.Background(), func() string { return "not-determined" }, func() bool {
		requests++
		return true
	}, 5*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) || requests != 1 {
		t.Fatalf("request = %v, calls = %d", err, requests)
	}
}

func TestMicrophoneRequestCancellationDuringPrompt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := requestMicrophoneAuthorization(ctx, func() string { return "not-determined" }, func() bool {
		cancel()
		return true
	}, time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("request = %v, want cancellation", err)
	}
}

func TestMicrophoneRequestFailsClosedWithoutUsageDescription(t *testing.T) {
	err := requestMicrophoneAuthorization(context.Background(), func() string { return "not-determined" }, func() bool { return false }, time.Second)
	if err == nil {
		t.Fatal("request accepted missing usage description")
	}
}

func TestMicrophoneAuthorizationReadHasKnownState(t *testing.T) {
	// A read is metadata-only; it must never request permission or open capture.
	switch state := MicrophoneAuthorization(); state {
	case "not-determined", "authorized", "denied", "restricted":
	default:
		t.Fatalf("unknown authorization state %q", state)
	}
}
