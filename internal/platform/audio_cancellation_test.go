//go:build windows || darwin

package platform

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
