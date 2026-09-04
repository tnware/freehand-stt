//go:build windows

package platform

import (
	"context"
	"errors"
	"testing"

	"github.com/tnware/freehand-stt/internal/audio"
)

type captureDeviceFake struct {
	starts   int
	stops    int
	uninit   int
	startErr error
}

func (d *captureDeviceFake) Start() error {
	d.starts++
	return d.startErr
}
func (d *captureDeviceFake) Stop() error { d.stops++; return nil }
func (d *captureDeviceFake) Uninit()     { d.uninit++ }

func TestCaptureReusesStoppedDevice(t *testing.T) {
	device := &captureDeviceFake{}
	capture := &Capture{dev: device}
	for run := 1; run <= 2; run++ {
		if _, err := capture.Start(context.Background(), "", 1); err != nil {
			t.Fatalf("start %d: %v", run, err)
		}
		if _, err := capture.Stop(context.Background()); err != nil {
			t.Fatalf("stop %d: %v", run, err)
		}
	}
	if device.starts != 2 || device.stops != 2 || device.uninit != 0 {
		t.Fatalf("device lifecycle = %d starts, %d stops, %d uninitializations", device.starts, device.stops, device.uninit)
	}
	if capture.dev != device {
		t.Fatal("clean stop discarded the prepared device")
	}
	if err := capture.Close(); err != nil {
		t.Fatal(err)
	}
	if device.uninit != 1 {
		t.Fatalf("shutdown uninitialized device %d times, want 1", device.uninit)
	}
}

func TestCaptureDeviceLossInvalidatesPreparedDevice(t *testing.T) {
	device := &captureDeviceFake{}
	capture := &Capture{dev: device, active: true}
	capture.deviceLost.Store(true)
	_, err := capture.Stop(context.Background())
	if !errors.Is(err, audio.ErrDeviceInterrupted) {
		t.Fatalf("stop error = %v, want device interruption", err)
	}
	if capture.dev != nil || device.uninit != 1 {
		t.Fatalf("lost device was retained: dev=%v uninitializations=%d", capture.dev, device.uninit)
	}
}

func TestCaptureCannotPrepareOrStartAfterClose(t *testing.T) {
	capture := &Capture{}
	if err := capture.Close(); err != nil {
		t.Fatal(err)
	}
	if err := capture.Prepare(context.Background(), ""); err == nil {
		t.Fatal("prepare succeeded after close")
	}
	if _, err := capture.Start(context.Background(), "", 1); err == nil {
		t.Fatal("start succeeded after close")
	}
}

func TestCaptureUnexpectedStopSignalsOnceWithoutBlocking(t *testing.T) {
	capture := &Capture{}
	capture.interruptionArmed.Store(true)
	interrupted := make(chan error, 1)
	capture.session.Store(&captureSession{interrupted: interrupted})

	capture.onDeviceStopped()
	capture.onDeviceStopped()

	select {
	case err := <-interrupted:
		if err != audio.ErrDeviceInterrupted {
			t.Fatalf("interruption = %v, want %v", err, audio.ErrDeviceInterrupted)
		}
	default:
		t.Fatal("unexpected stop did not signal an interruption")
	}
	select {
	case err := <-interrupted:
		t.Fatalf("duplicate interruption = %v", err)
	default:
	}
	if !capture.deviceLost.Load() {
		t.Fatal("unexpected stop was not retained for cleanup")
	}
}

func TestCaptureIntentionalOrInactiveStopDoesNotSignal(t *testing.T) {
	for _, test := range []struct {
		name string
	}{
		{name: "intentional"},
		{name: "inactive"},
	} {
		t.Run(test.name, func(t *testing.T) {
			capture := &Capture{}
			if test.name == "intentional" {
				capture.interruptionArmed.Store(true)
				capture.interruptionArmed.Store(false)
			}
			interrupted := make(chan error, 1)
			capture.session.Store(&captureSession{interrupted: interrupted})
			capture.onDeviceStopped()
			select {
			case err := <-interrupted:
				t.Fatalf("stop signalled device loss: %v", err)
			default:
			}
			if capture.deviceLost.Load() {
				t.Fatal("stop was marked as device loss")
			}
		})
	}
}
