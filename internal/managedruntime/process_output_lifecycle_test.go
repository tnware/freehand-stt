package managedruntime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

type outputStartupAdapter struct {
	runtimeAdapter
	launched chan struct{}
	finish   chan struct{}
}

func (a *outputStartupAdapter) Start(ctx context.Context, model string) (*ownedProcess, Endpoint, error) {
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
	p.kill()
	_ = p.wait(ctx)
	return p, Endpoint{}, errors.New("fixture readiness failure")
}
func TestProcessOutputChild(t *testing.T) {
	if os.Getenv("FREEHAND_OUTPUT_HELPER") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, "startup private text\n")
	fmt.Fprint(os.Stderr, "startup stderr\n")
	time.Sleep(20 * time.Second)
	os.Exit(0)
}
func TestProcessOutputDuringStartupAndFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
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
		if strings.Contains(text.String(), "startup private text") && strings.Contains(text.String(), "startup stderr") {
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

func TestProcessOutputPreservesParserPrefix(t *testing.T) {
	w := &worker{}
	prefix := &boundedOutput{}
	writer := observedOutput{prefix: prefix, observer: &startupObserver{output: func(stream string, p []byte) { w.appendProcessOutput(0, stream, p) }}, stream: "stdout"}
	input := strings.Repeat("a", outputLimit) + "tail"
	n, err := writer.Write([]byte(input))
	if n != len(input) || err != nil || !prefix.overflow || string(prefix.bytes()) != input[:outputLimit] {
		t.Fatal("parser prefix/overflow changed")
	}
	chunks := w.output.chunks
	if !strings.HasSuffix(chunks[len(chunks)-1].Text, "tail") {
		t.Fatal("viewer not rolling")
	}
}
