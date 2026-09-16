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
	process processHandle
}

func (p *providerProcess) start(ctx context.Context, a runtimeAdapter, model string) (processHandle, Endpoint, error) {
	return p.startModels(ctx, a, model, "")
}

func (p *providerProcess) startModels(ctx context.Context, a runtimeAdapter, model, speechModel string) (processHandle, Endpoint, error) {
	if !p.mu.TryLock() {
		return nil, Endpoint{}, errProviderRunning
	}
	defer p.mu.Unlock()
	if p.process != nil {
		select {
		case <-p.process.Done():
			p.process = nil
		default:
			return nil, Endpoint{}, errProviderRunning
		}
	}
	var proc processHandle
	var endpoint Endpoint
	var err error
	if speechModel == "" {
		proc, endpoint, err = a.Start(ctx, model)
	} else if combined, ok := a.(interface {
		StartModels(context.Context, string, string) (processHandle, Endpoint, error)
	}); ok {
		proc, endpoint, err = combined.StartModels(ctx, model, speechModel)
	} else {
		return nil, Endpoint{}, errors.New("This runtime cannot load a separate speech model.")
	}
	p.process = proc
	return proc, endpoint, err
}
