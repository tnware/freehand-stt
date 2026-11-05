//go:build linux && cgo

package app

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	nativeinput "github.com/tnware/freehand-stt/internal/platform/input"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const clipboardProbeEnvironment = "FREEHAND_LINUX_CLIPBOARD_HELPER"

// This explicit desktop test changes the clipboard to a synthetic Unicode
// fixture. It never reads previous clipboard contents or opens a WebView.
func init() {
	if os.Getenv(clipboardProbeEnvironment) != "1" {
		return
	}
	time.AfterFunc(15*time.Second, func() { os.Exit(124) })
	desktop := application.New(application.Options{Name: "Freehand clipboard qualification"})
	owner := &App{wails: desktop}
	input := nativeinput.NewInput(nil)
	owner.configureNativeInput(input)
	result := make(chan error, 1)
	desktop.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		go func() {
			const fixture = "Freehand Linux copy: ✓ 漢字 😀"
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := input.Copy(ctx, fixture)
			if err == nil {
				value, ok := desktop.Clipboard.Text()
				if !ok || value != fixture {
					err = errors.New("native Unicode clipboard round trip failed")
				}
			}
			result <- err
			desktop.Quit()
		}()
	})
	if desktop.Run() != nil {
		os.Exit(1)
	}
	select {
	case err := <-result:
		if err != nil {
			os.Exit(2)
		}
	default:
		os.Exit(3)
	}
	os.Exit(0)
}

func TestLinuxNativeClipboardAcceptance(t *testing.T) {
	if os.Getenv("FREEHAND_LINUX_CLIPBOARD_ACCEPTANCE") != "1" {
		t.Skip("explicit opt-in writes a synthetic Unicode fixture to the actual desktop clipboard")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, self, "-test.run=^$")
	command.Env = append(os.Environ(), clipboardProbeEnvironment+"=1")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("native desktop clipboard probe: %v\n%s", err, output)
	}
	t.Log("actual GTK desktop clipboard preserved Unicode through explicit Copy; no insertion or WebView acceptance implied")
}
