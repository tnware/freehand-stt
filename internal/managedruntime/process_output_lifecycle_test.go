package managedruntime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

type outputStartupAdapter struct {
	runtimeAdapter
	launched chan struct{}
	finish   chan struct{}
}

func (a *outputStartupAdapter) Start(ctx context.Context, model string) (processHandle, Endpoint, error) {
	reportStartupProgress(ctx, "launching")
	p, err := launchOwned(runtimeProcessContext(ctx), os.Args[0], []string{"-test.run=^TestProcessOutputChild$"}, "", append(os.Environ(), "FREEHAND_OUTPUT_HELPER=1"))
	if err != nil {
		return nil, Endpoint{}, err
	}
	reportStartupProgress(ctx, "waiting_ready")
	close(a.launched)
	select {
	case <-a.finish:
	case <-ctx.Done():
	}
	p.Kill()
	_ = p.Wait(ctx)
	return p, Endpoint{}, errors.New("fixture readiness failure")
}
func TestProcessOutputChild(t *testing.T) {
	if os.Getenv("FREEHAND_OUTPUT_HELPER") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, "\x1b[32mstartup private text\x1b[0m\r\x1b[Kready\n\x1b]52;c;hidden clipboard\a")
	fmt.Fprint(os.Stderr, "startup stderr\n")
	time.Sleep(20 * time.Second)
	os.Exit(0)
}
func TestProcessOutputDuringStartupAndFailure(t *testing.T) {
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		t.Skip("owned runtime processes are supported only on Windows and macOS")
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	a := &outputStartupAdapter{launched: make(chan struct{}), finish: make(chan struct{})}
	w := &worker{ctx: ctx, cancel: cancel, adapter: a, providerProcess: &providerProcess{}, configuration: workerConfig{Enabled: true, Model: "fixture"}, status: Status{Supported: true, Models: []Model{{ID: "fixture", Installed: true}}}}
	m := &Manager{workers: map[string]*worker{"test": w}}
	defer w.ServiceShutdown()
	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-a.launched:
	case <-time.After(5 * time.Second):
		t.Fatal("launch deadline")
	}
	req := InstanceRequest{InstanceID: "test"}
	hidden, _ := m.ReadProcessOutput(OutputRequest{InstanceID: "test"})
	if len(hidden.Chunks) != 0 {
		t.Fatal("not opted in")
	}
	if err := m.EnableProcessOutput(req); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		snap, _ := m.ReadProcessOutput(OutputRequest{InstanceID: "test"})
		var text strings.Builder
		for _, c := range snap.Chunks {
			text.WriteString(c.Text)
		}
		if strings.Contains(text.String(), "\x1b[32mstartup private text\x1b[0m\r\x1b[Kready\n") && strings.Contains(text.String(), "startup stderr") {
			displaySnapshotText(t, snap)
			if strings.Contains(text.String(), "hidden clipboard") {
				t.Fatal("hostile child OSC escaped the capture policy")
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("output unavailable before readiness")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if w.GetStatus().StartupProgress.Phase != "waiting_ready" {
		t.Fatal("startup progress missing")
	}
	close(a.finish)
	for w.GetStatus().Operation.Outcome == "running" {
		if time.Now().After(deadline) {
			t.Fatal("operation deadline")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if w.GetStatus().StartupProgress != nil {
		t.Fatal("terminal progress retained")
	}
	_ = m.DisableProcessOutput(req)
	_ = m.EnableProcessOutput(req)
	snap, _ := m.ReadProcessOutput(OutputRequest{InstanceID: "test"})
	if len(snap.Chunks) == 0 {
		t.Fatal("failed launch tail lost on reopen")
	}
	_ = m.ClearProcessOutput(req)
	snap, _ = m.ReadProcessOutput(OutputRequest{InstanceID: "test"})
	if len(snap.Chunks) != 0 || !snap.Enabled {
		t.Fatal("clear changed authorization or retained text")
	}
}

func TestProcessOutputKeepsTailBeyondDiagnosticPrefix(t *testing.T) {
	w := &worker{}
	input := "\x1b[31m" + strings.Repeat("a", outputLimit) + "tail\x1b[0m"
	w.appendProcessOutput(0, "stdout", []byte(input))
	chunks := w.output.chunks
	if !strings.HasSuffix(chunks[len(chunks)-1].Text, "tail\x1b[0m") {
		t.Fatal("viewer not rolling")
	}
}
