package managedruntime

import (
	"context"
	"sync"
	"testing"
	"time"
)

// A metadata-only fake holds launch and process exit at explicit barriers.
// No executable, model file, network request, or inference is involved.
type publicationAdapter struct {
	workerAdapter
	launchState func() string
	entered     chan string
	launch      chan struct{}
	stopEntered chan struct{}
	processDone chan struct{}
}

func (a *publicationAdapter) Start(ctx context.Context, model string) (processHandle, Endpoint, error) {
	a.startContext = ctx
	a.entered <- a.launchState()
	reportStartupProgress(ctx, "verifying_runtime")
	reportStartupProgress(ctx, "waiting_ready")
	select {
	case <-a.launch:
	case <-ctx.Done():
		return nil, Endpoint{}, ctx.Err()
	}
	p := &fakeProcess{done: a.processDone, closeJob: func() { close(a.stopEntered) }}
	return p, Endpoint{Enabled: true, BaseURL: "http://127.0.0.1:12345", Model: "authoritative", Profile: qualified[model].Profile}, nil
}

func TestManagerRestartOwnsStopAndLaunch(t *testing.T) {
	for _, outcome := range []string{"ready", "cancelled", "cancel_at_start", "stop_failed", "shutdown"} {
		t.Run(outcome, func(t *testing.T) {
			instance := Instance{ID: "speech", Name: "Speech", Provider: NeMoSpeechCPP, Model: "nemotron-3.5", AutoStart: true}
			events := make(chan InstanceStatus, 64)
			var w *worker
			m := NewManager(ManagerOptions{
				Directory: t.TempDir(), Instances: []Instance{instance},
				Changed: func(status InstanceStatus) {
					events <- status
					if outcome == "cancel_at_start" && status.Status.State == "starting" && status.Status.Operation.Kind == "restart" {
						_ = w.Cancel()
					}
				},
			})
			w = m.workers[instance.ID]
			adapter := func() *publicationAdapter {
				return &publicationAdapter{
					installed: true, downloaded: true,
					launchState: func() string { return w.GetStatus().State },
					entered:     make(chan string, 1), launch: make(chan struct{}),
					stopEntered: make(chan struct{}), processDone: make(chan struct{}),
				}
			}
			old, next := adapter(), adapter()
			close(old.launch)
			releaseOldExit := sync.OnceFunc(func() { close(old.processDone) })
			releaseNextLaunch := sync.OnceFunc(func() { close(next.launch) })
			releaseNextExit := sync.OnceFunc(func() { close(next.processDone) })
			t.Cleanup(func() {
				releaseOldExit()
				releaseNextLaunch()
				releaseNextExit()
				if err := m.ServiceShutdown(); err != nil {
					t.Error(err)
				}
			})
			w.adapter = old
			w.status.Supported = true
			if err := m.startup(t.Context()); err != nil {
				t.Fatal(err)
			}
			awaitLifecyclePublication(t, events, func(row InstanceStatus) bool {
				return row.Status.State == "running" && row.Status.Operation.Outcome == "succeeded"
			})
			w.adapter = next
			request := InstanceRequest{InstanceID: instance.ID}
			if err := m.Restart(request); err != nil {
				t.Fatal(err)
			}
			stopping := awaitLifecyclePublication(t, events, func(row InstanceStatus) bool {
				return row.Status.State == "stopping" && row.Status.Operation.Kind == "restart" && row.Status.Operation.Outcome == "running"
			})
			select {
			case <-old.stopEntered:
			case <-time.After(3 * time.Second):
				t.Fatal("restart did not request the old process stop")
			}
			if err := m.Start(request); err == nil {
				t.Fatal("accepted a separate start while restart owns the stop")
			}
			select {
			case <-next.entered:
				t.Fatal("restart launched before the old process exited")
			default:
			}
			if outcome == "cancelled" {
				if err := m.Cancel(request); err != nil {
					t.Fatal(err)
				}
			} else if outcome == "shutdown" {
				w.closeNow()
			} else if outcome == "cancel_at_start" {
				releaseOldExit()
			}
			if outcome != "ready" {
				terminal := awaitLifecyclePublication(t, events, func(row InstanceStatus) bool {
					return row.Status.Operation.Kind == "restart" && row.Status.Operation.Outcome != "running"
				})
				want := "cancelled"
				if outcome == "stop_failed" {
					want = "failed"
				}
				if terminal.Status.Operation.Outcome != want || terminal.ActiveModel != "" {
					t.Fatalf("restart did not stop after %s: %+v", outcome, terminal)
				}
				select {
				case <-next.entered:
					t.Fatal("restart launched after stop failure, cancellation, or shutdown")
				default:
				}
				return
			}
			releaseOldExit()
			starting := awaitLifecyclePublication(t, events, func(row InstanceStatus) bool {
				return row.Status.State == "starting" && row.Status.StartupProgress != nil && row.Status.StartupProgress.Phase == "waiting_ready"
			})
			if starting.Status.Operation.ID != stopping.Status.Operation.ID || starting.Status.Operation.Kind != "restart" || starting.ActiveModel != "" {
				t.Fatalf("restart lost its single operation identity: %+v", starting)
			}
			releaseNextLaunch()
			running := awaitLifecyclePublication(t, events, func(row InstanceStatus) bool {
				return row.Status.State == "running" && row.Status.Operation.Kind == "restart" && row.Status.Operation.Outcome == "succeeded"
			})
			if running.Status.Operation.ID != stopping.Status.Operation.ID || running.ActiveModel != "authoritative" || running.Status.StartupProgress != nil {
				t.Fatalf("incoherent restart completion: %+v", running)
			}
			if err := next.startContext.Err(); err != nil {
				t.Fatalf("restart cancelled the replacement process context: %v", err)
			}
		})
	}
}

func awaitLifecyclePublication(t *testing.T, events <-chan InstanceStatus, match func(InstanceStatus) bool) InstanceStatus {
	t.Helper()
	deadline := time.NewTimer(6 * time.Second)
	defer deadline.Stop()
	for {
		select {
		case status := <-events:
			if status.Status.StartupProgress != nil && status.Status.State != "starting" {
				t.Fatalf("startup progress has no starting state: %+v", status.Status)
			}
			if match(status) {
				return status
			}
		case <-deadline.C:
			t.Fatal("runtime lifecycle publication missing")
		}
	}
}

func TestManagerPublishesDelayedStartAndStopLifecycle(t *testing.T) {
	for _, autoStart := range []bool{false, true} {
		name := "explicit"
		if autoStart {
			name = "auto-start"
		}
		t.Run(name, func(t *testing.T) {
			instance := Instance{ID: "speech", Name: "Speech", Provider: NeMoSpeechCPP, Model: "nemotron-3.5", AutoStart: autoStart}
			events := make(chan InstanceStatus, 32)
			m := NewManager(ManagerOptions{
				Directory: t.TempDir(), Instances: []Instance{instance},
				Changed: func(status InstanceStatus) { events <- status },
			})
			w := m.workers[instance.ID]
			a := &publicationAdapter{
				installed: true, downloaded: true,
				launchState: func() string { return w.GetStatus().State },
				entered:     make(chan string, 1), launch: make(chan struct{}),
				stopEntered: make(chan struct{}), processDone: make(chan struct{}),
			}
			w.adapter = a
			w.status.Supported = true
			releaseLaunch := sync.OnceFunc(func() { close(a.launch) })
			releaseExit := sync.OnceFunc(func() { close(a.processDone) })
			t.Cleanup(func() {
				releaseLaunch()
				releaseExit()
				if err := m.ServiceShutdown(); err != nil {
					t.Error(err)
				}
			})
			if err := m.startup(t.Context()); err != nil {
				t.Fatal(err)
			}
			kind := "startup"
			if !autoStart {
				awaitLifecyclePublication(t, events, func(row InstanceStatus) bool {
					return row.Status.State == "installed" && row.Status.Operation.Outcome == "succeeded"
				})
				kind = "start"
				if err := m.Start(InstanceRequest{InstanceID: instance.ID}); err != nil {
					t.Fatal(err)
				}
			}
			select {
			case state := <-a.entered:
				if state != "starting" {
					t.Fatalf("adapter entered before starting transition: %s", state)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("adapter was not entered")
			}
			awaitLifecyclePublication(t, events, func(row InstanceStatus) bool {
				return row.Status.State == "starting" && row.Status.Operation.Kind == kind && row.Status.StartupProgress == nil
			})
			awaitLifecyclePublication(t, events, func(row InstanceStatus) bool {
				return row.Status.State == "starting" && row.Status.StartupProgress != nil && row.Status.StartupProgress.Phase == "waiting_ready"
			})
			pending := m.GetInstances()[0]
			if pending.Status.State != "starting" || pending.Status.Operation.Outcome != "running" || pending.ActiveModel != "" {
				t.Fatalf("launch admission reported completion: %+v", pending)
			}
			releaseLaunch()
			running := awaitLifecyclePublication(t, events, func(row InstanceStatus) bool {
				return row.Status.State == "running" && row.Status.Operation.Kind == kind && row.Status.Operation.Outcome == "succeeded"
			})
			if running.ActiveModel != "authoritative" || running.Status.Phase != "" || running.Status.StartupProgress != nil {
				t.Fatalf("incoherent ready publication: %+v", running)
			}
			if err := m.Stop(InstanceRequest{InstanceID: instance.ID}); err != nil {
				t.Fatal(err)
			}
			awaitLifecyclePublication(t, events, func(row InstanceStatus) bool {
				return row.Status.State == "stopping" && row.Status.Operation.Kind == "stop" && row.Status.Operation.Outcome == "running"
			})
			select {
			case <-a.stopEntered:
			case <-time.After(3 * time.Second):
				t.Fatal("process stop was not requested")
			}
			pending = m.GetInstances()[0]
			if pending.Status.State != "stopping" || pending.Status.Operation.Outcome != "running" || pending.ActiveModel != "" {
				t.Fatalf("stop admission reported completion before exit: %+v", pending)
			}
			releaseExit()
			stopped := awaitLifecyclePublication(t, events, func(row InstanceStatus) bool {
				return row.Status.State == "stopped" && row.Status.Operation.Kind == "stop" && row.Status.Operation.Outcome == "succeeded"
			})
			if stopped.ActiveModel != "" || stopped.Status.Phase != "" || stopped.Status.Error != "" {
				t.Fatalf("incoherent stopped publication: %+v", stopped)
			}
		})
	}
}
