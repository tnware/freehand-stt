package managedruntime

import (
	"context"

	"github.com/tnware/freehand-stt/internal/managedruntime/internal/process"
)

const outputLimit = process.OutputLimit

// processHandle exposes process lifetime and bounded diagnostic snapshots;
// native ownership remains opaque and test adapters provide controlled handles.
type processHandle interface {
	PID() int
	Done() <-chan struct{}
	Kill()
	Wait(context.Context) error
	Stdout() []byte
	Stderr() []byte
	StdoutOverflow() bool
}

func launchOwned(ctx context.Context, exe string, args []string, dir string, env []string) (processHandle, error) {
	observer := process.Observer{}
	if marked, _ := ctx.Value(runtimeProcessKey{}).(bool); marked {
		if owner, _ := ctx.Value(startupObserverKey{}).(*startupObserver); owner != nil {
			observer.Output = owner.output
			observer.Started = func(p *process.Process) { owner.child(p) }
		}
	}
	p, err := process.Launch(ctx, exe, args, dir, env, observer)
	if err != nil {
		return nil, err
	}
	return p, nil
}
