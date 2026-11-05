package main

import (
	"bytes"
	"sync"
	"testing"
)

func TestShellReadyMarkerRequiresExactOptIn(t *testing.T) {
	for _, value := range []string{"", "0", "true", "yes", " 1", "1\n"} {
		var output bytes.Buffer
		if callback := shellReadyMarker("linux", value, &output); callback != nil {
			t.Fatalf("unexpected marker callback for %q", value)
		}
		if output.Len() != 0 {
			t.Fatal("disabled marker wrote output")
		}
	}
}

func TestShellReadyMarkerEmitsOnlyOnceAfterCallback(t *testing.T) {
	var output bytes.Buffer
	callback := shellReadyMarker("linux", "1", &output)
	if callback == nil || output.Len() != 0 {
		t.Fatal("marker must wait for the renderer handshake")
	}
	var calls sync.WaitGroup
	for range 32 {
		calls.Go(callback)
	}
	calls.Wait()
	if got := output.String(); got != "FREEHAND_FRONTEND_BRIDGE_READY_V1\n" {
		t.Fatalf("unexpected marker output %q", got)
	}
}

func TestShellReadyMarkerIsLinuxOnly(t *testing.T) {
	for _, platform := range []string{"windows", "darwin", ""} {
		var output bytes.Buffer
		if shellReadyMarker(platform, "1", &output) != nil || output.Len() != 0 {
			t.Fatalf("Linux smoke marker enabled on %q", platform)
		}
	}
}
