package managedruntime

import (
	"context"
	"testing"
	"time"
)

type cancelledStartAdapter struct {
	workerAdapter
	child  processHandle
	cancel context.CancelFunc
}

func (a *cancelledStartAdapter) Start(context.Context, string) (processHandle, Endpoint, error) {
	a.cancel()
	return a.child, Endpoint{Enabled: true, BaseURL: "http://127.0.0.1:12345/v1", Model: "advertised"}, nil
}

func TestCancelledStartRemainsOwnedUntilChildExits(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	child := &fakeProcess{done: make(chan struct{}), closeJob: func() {}}
	defer close(child.done)
	w := newWorker(t.TempDir(), nemoProvider{}, workerConfig{Enabled: true, Model: "nemotron-3.5"}, nil, nil)
	w.status.Models = []Model{{ID: "nemotron-3.5", Installed: true}}
	w.adapter = &cancelledStartAdapter{child: child, cancel: cancel}
	if err := w.startProcess(ctx); err == nil {
		t.Fatal("published cancelled launch")
	}
	w.closeNow()
	if err := waitWorkers([]*worker{w}, 10*time.Millisecond); err == nil {
		t.Fatal("shutdown/reaping forgot a cancelled launch before its child exited")
	}
}

func TestResolveRejectsExitedProcessBeforeMonitorPublishes(t *testing.T) {
	w := newWorker(t.TempDir(), nemoProvider{}, workerConfig{Enabled: true, Model: "nemotron-3.5"}, nil, nil)
	w.status.Supported = true
	w.status.State = "running"
	w.endpoint = Endpoint{Enabled: true, BaseURL: "http://127.0.0.1:12345/v1", Model: "advertised"}
	proc := &fakeProcess{done: make(chan struct{})}
	w.process = proc
	close(proc.done)
	w.mu.Lock()
	_, err := w.resolveLocked()
	w.mu.Unlock()
	if err == nil {
		t.Fatal("resolved an exited child while its monitor waited for the worker lock")
	}
}
