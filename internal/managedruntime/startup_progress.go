package managedruntime

import (
	"context"
	"time"
)

// StartupProgress is a truthful adapter-reported detail, not the operation phase.
type StartupProgress struct {
	Phase     string `json:"phase"`
	StartedAt int64  `json:"startedAt"`
}
type startupObserverKey struct{}
type runtimeProcessKey struct{}
type startupObserver struct {
	output   func(string, []byte)
	child    func(processHandle)
	progress func(string)
}

// runtimeProcessContext marks ONLY the long-lived inference child. Never pass
// it to model-manager commands or probes; those keep the unmarked Start ctx.
func runtimeProcessContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, runtimeProcessKey{}, true)
}
func reportStartupProgress(ctx context.Context, phase string) {
	switch phase {
	case "verifying_runtime", "verifying_model", "launching", "waiting_ready", "warming_up", "loading_warming":
	default:
		return
	}
	if ctx.Err() != nil {
		return
	}
	if o, _ := ctx.Value(startupObserverKey{}).(*startupObserver); o != nil {
		o.progress(phase)
	}
}
func (w *worker) observeStartup(ctx context.Context) context.Context {
	w.mu.Lock()
	w.outputLaunch++
	launch := w.outputLaunch
	w.output.clear()
	w.status.StartupProgress = nil
	w.mu.Unlock()
	o := &startupObserver{}
	o.output = func(stream string, p []byte) { w.appendProcessOutput(launch, stream, p) }
	o.child = func(p processHandle) {
		w.mu.Lock()
		stale := w.closed || ctx.Err() != nil || launch != w.outputLaunch
		if !stale {
			w.startingProcess = p
		}
		w.mu.Unlock()
		if stale {
			p.Kill()
		}
	}
	o.progress = func(phase string) {
		w.mu.Lock()
		if w.closed || ctx.Err() != nil || launch != w.outputLaunch || !w.busy || w.status.Operation.Outcome != "running" {
			w.mu.Unlock()
			return
		}
		if w.status.StartupProgress != nil && w.status.StartupProgress.Phase == phase {
			w.mu.Unlock()
			return
		}
		w.status.StartupProgress = &StartupProgress{Phase: phase, StartedAt: time.Now().UnixMilli()}
		w.mu.Unlock()
		w.notify()
	}
	return context.WithValue(ctx, startupObserverKey{}, o)
}
