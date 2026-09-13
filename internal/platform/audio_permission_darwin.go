//go:build darwin

package platform

/*
#cgo CFLAGS: -mmacosx-version-min=13.0 -fblocks
#cgo LDFLAGS: -framework AVFoundation -framework Foundation
#include "audio_permission_darwin.h"
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const microphoneAuthorizationTimeout = 60 * time.Second

// MicrophoneAuthorization reads TCC state without prompting or opening a device.
// Values are not-determined, authorized, denied, or restricted.
func MicrophoneAuthorization() string {
	switch C.freehand_microphone_authorization() {
	case 0:
		return "not-determined"
	case 1:
		return "authorized"
	case 2:
		return "denied"
	default:
		return "restricted"
	}
}

// RequestMicrophoneAuthorization is for explicit user actions only. Waiting is
// bounded to 60 seconds (or the caller's earlier cancellation/deadline). Cancel
// releases the Go waiter; macOS owns and may keep its already-visible TCC dialog.
// No capture device is opened and no Go callback/goroutine outlives this call.
func RequestMicrophoneAuthorization(ctx context.Context) error {
	return requestMicrophoneAuthorization(ctx, MicrophoneAuthorization, func() bool {
		return C.freehand_microphone_request() != 0
	}, microphoneAuthorizationTimeout)
}

func requestMicrophoneAuthorization(ctx context.Context, status func() string, begin func() bool, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	requested := false
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		switch state := status(); state {
		case "authorized":
			return ctx.Err()
		case "not-determined":
			if !requested {
				if err := ctx.Err(); err != nil {
					return err
				}
				if !begin() {
					return errors.New("microphone permission requires NSMicrophoneUsageDescription in the application bundle")
				}
				requested = true
				continue
			}
		default:
			return fmt.Errorf("microphone permission is %s; enable it in System Settings > Privacy & Security > Microphone", state)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func authorizeMicrophone(ctx context.Context, request bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if request {
		return RequestMicrophoneAuthorization(ctx)
	}
	if state := MicrophoneAuthorization(); state != "authorized" {
		return fmt.Errorf("microphone permission is %s", state)
	}
	return nil
}
