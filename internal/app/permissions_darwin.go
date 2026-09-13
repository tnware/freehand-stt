//go:build darwin && cgo

package app

/*
#cgo LDFLAGS: -framework ApplicationServices
#include <ApplicationServices/ApplicationServices.h>
static void freehand_request_accessibility_ui(void) {
 const void *keys[] = { kAXTrustedCheckOptionPrompt };
 const void *values[] = { kCFBooleanTrue };
 CFDictionaryRef options = CFDictionaryCreate(NULL, keys, values, 1, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
 AXIsProcessTrustedWithOptions(options);
 CFRelease(options);
}
*/
import "C"

import (
	"context"
	"errors"

	inputservice "github.com/tnware/freehand-stt/internal/input"
	"github.com/tnware/freehand-stt/internal/platform"
)

type nativePermissionAccess struct{ app *App }

func (p nativePermissionAccess) Current() inputservice.PermissionStatus {
	return inputservice.PermissionStatus{Required: true, Microphone: platform.MicrophoneAuthorization(), Accessibility: C.AXIsProcessTrusted() != 0, Keyboard: platform.KeyboardAuthorization()}
}
func (p nativePermissionAccess) Request(ctx context.Context, kind string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch kind {
	case "microphone":
		return platform.RequestMicrophoneAuthorization(ctx)
	case "accessibility":
		C.freehand_request_accessibility_ui()
	case "keyboard":
		C.CGRequestListenEventAccess()
	default:
		return errors.New("unknown permission")
	}
	return ctx.Err()
}
func (p nativePermissionAccess) OpenSettings(kind string) error {
	var suffix string
	switch kind {
	case "microphone":
		suffix = "Privacy_Microphone"
	case "accessibility":
		suffix = "Privacy_Accessibility"
	case "keyboard":
		suffix = "Privacy_ListenEvent"
	default:
		return errors.New("unknown permission")
	}
	return p.app.wails.Browser.OpenURL("x-apple.systempreferences:com.apple.preference.security?" + suffix)
}
func (a *App) configureNativePermissions() {
	inputservice.ConfigurePermissions(a.inputService, nativePermissionAccess{app: a})
}
