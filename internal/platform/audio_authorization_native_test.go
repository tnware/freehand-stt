//go:build windows || darwin

package platform

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCaptureStartRequiresExplicitAuthorization(t *testing.T) {
	denied := errors.New("microphone denied")
	device := &captureDeviceFake{}
	capture := &Capture{dev: device, authorize: func(ctx context.Context, request bool) error {
		if !request {
			t.Fatal("Start did not request authorization")
		}
		return denied
	}}
	if _, err := capture.Start(context.Background(), "", 1); !errors.Is(err, denied) {
		t.Fatalf("Start = %v, want denied", err)
	}
	if device.starts != 0 || capture.active {
		t.Fatal("unauthorized capture started")
	}
}

func TestCaptureRechecksCancellationAfterAuthorization(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	device := &captureDeviceFake{}
	capture := &Capture{dev: device, authorize: func(context.Context, bool) error { cancel(); return nil }}
	if _, err := capture.Start(ctx, "", 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("Start = %v", err)
	}
	if device.starts != 0 || capture.active {
		t.Fatal("canceled authorization started capture")
	}
}

func TestCaptureCloseDoesNotWaitForPermissionPrompt(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	device := &captureDeviceFake{}
	capture := &Capture{dev: device, authorize: func(context.Context, bool) error { close(entered); <-release; return nil }}
	started := make(chan error, 1)
	go func() { _, err := capture.Start(context.Background(), "", 1); started <- err }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("permission not entered")
	}
	closed := make(chan error, 1)
	go func() { closed <- capture.Close() }()
	select {
	case err := <-closed:
		if err != nil {
			t.Error(err)
		}
	case <-time.After(time.Second):
		close(release)
		t.Fatal("Close blocked behind permission prompt")
	}
	close(release)
	select {
	case err := <-started:
		if err == nil {
			t.Fatal("Start succeeded after Close")
		}
	case <-time.After(time.Second):
		t.Fatal("Start did not finish")
	}
	if device.starts != 0 || device.uninit != 1 {
		t.Fatal("closed capture started or retained device")
	}
}

func TestCaptureInvalidDurationDoesNotRequestPermission(t *testing.T) {
	capture := &Capture{authorize: func(context.Context, bool) error { t.Fatal("invalid duration prompted"); return nil }}
	if _, err := capture.Start(context.Background(), "", 0); err == nil {
		t.Fatal("invalid duration accepted")
	}
}

func TestCapturePrepareRechecksCancellationAfterAuthorization(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	device := &captureDeviceFake{}
	capture := &Capture{dev: device, authorize: func(context.Context, bool) error { cancel(); return nil }}
	if err := capture.Prepare(ctx, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("Prepare = %v, want cancellation", err)
	}
}

func TestCapturePrepareChecksPermissionWithoutRequesting(t *testing.T) {
	denied := errors.New("microphone denied")
	device := &captureDeviceFake{}
	calls := 0
	capture := &Capture{dev: device, authorize: func(ctx context.Context, request bool) error {
		calls++
		if request {
			t.Fatal("Prepare requested microphone permission")
		}
		return denied
	}}
	if err := capture.Prepare(context.Background(), ""); !errors.Is(err, denied) {
		t.Fatalf("Prepare = %v, want permission rejection", err)
	}
	if calls != 1 || device.starts != 0 || capture.active {
		t.Fatal("Prepare bypassed permission or started capture")
	}
}
