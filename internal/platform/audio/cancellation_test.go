//go:build windows || darwin || (linux && cgo)

package audio

import (
	"context"
	"errors"
	"testing"
)

func TestCaptureCanceledStartDoesNotOpenMicrophone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	capture := &Capture{}
	_, err := capture.Start(ctx, "", 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Start with canceled context = %v, want context.Canceled before device access", err)
	}
}

func TestCaptureCanceledListDoesNotInitializeAudio(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	capture := &Capture{}
	devices, err := capture.List(ctx)
	if !errors.Is(err, context.Canceled) || devices != nil {
		t.Fatalf("List = %v, %v, want no devices and cancellation", devices, err)
	}
	if capture.ctx != nil || capture.dev != nil {
		t.Fatal("canceled enumeration retained native resources")
	}
}
