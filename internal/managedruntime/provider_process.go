package managedruntime

import (
	"context"
	"errors"
	"sync"
)

var errProviderRunning = errors.New("This runtime provider is already starting or running. Stop it before starting another installation.")

// Shared by all current and retired workers of a provider. Keep the child
// identity until actual exit, even after cancellation clears its endpoint.
// Holding this lock through Start also fences concurrent auto-starts.
type providerProcess struct {
	mu      sync.Mutex
	process *ownedProcess
}

func (p *providerProcess) start(ctx context.Context, a runtimeAdapter, model string) (*ownedProcess, Endpoint, error) {
	if !p.mu.TryLock() {
		return nil, Endpoint{}, errProviderRunning
	}
	defer p.mu.Unlock()
	if p.process != nil {
		select {
		case <-p.process.done:
			p.process = nil
		default:
			return nil, Endpoint{}, errProviderRunning
		}
	}
	proc, endpoint, err := a.Start(ctx, model)
	p.process = proc
	return proc, endpoint, err
}
