package managedruntime

import (
	"context"
	"testing"
)

func TestStartupProgressFencesOldLaunch(t *testing.T) {
	w := &worker{busy: true, status: Status{Phase: "start", Operation: Operation{Outcome: "running"}}}
	ctx := w.observeStartup(context.Background())
	reportStartupProgress(ctx, "verifying_runtime")
	first := w.GetStatus()
	if first.StartupProgress == nil || first.StartupProgress.Phase != "verifying_runtime" || first.StartupProgress.StartedAt <= 0 {
		t.Fatalf("missing progress: %+v", first)
	}
	reportStartupProgress(ctx, "sensitive arbitrary text")
	if w.GetStatus().StartupProgress.Phase != "verifying_runtime" {
		t.Fatal("non-allowlisted progress")
	}
	next := w.observeStartup(context.Background())
	reportStartupProgress(next, "waiting_ready")
	reportStartupProgress(ctx, "warming_up")
	w.appendProcessOutput(1, "stdout", []byte("old"))
	w.appendProcessOutput(2, "stdout", []byte("new"))
	if w.GetStatus().StartupProgress.Phase != "waiting_ready" || len(w.output.chunks) != 1 || w.output.chunks[0].Text != "new" {
		t.Fatal("old launch was not fenced")
	}
	w.closeNow()
	reportStartupProgress(next, "warming_up")
	if w.GetStatus().StartupProgress != nil || len(w.output.chunks) != 0 {
		t.Fatal("shutdown retained progress/output")
	}
}
