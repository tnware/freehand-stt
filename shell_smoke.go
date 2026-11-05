package main

import (
	"io"
	"sync"
)

// shellReadyMarker observes the real renderer-to-Go handshake without enabling
// bridge logging or recording application data. It does not assert that session
// initialization succeeded: the shell also becomes navigable after a load error.
func shellReadyMarker(platform, optIn string, output io.Writer) func() {
	if platform != "linux" || optIn != "1" {
		return nil
	}
	var once sync.Once
	return func() {
		once.Do(func() { _, _ = io.WriteString(output, "FREEHAND_FRONTEND_BRIDGE_READY_V1\n") })
	}
}
