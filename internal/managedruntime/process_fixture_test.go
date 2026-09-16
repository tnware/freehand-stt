package managedruntime

import (
	"context"
	"errors"
	"sync"
	"time"
)

type boundedOutput struct {
	mu       sync.Mutex
	b        []byte
	overflow bool
}

func (b *boundedOutput) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	room := outputLimit - len(b.b)
	if len(p) > room {
		b.overflow = true
		p = p[:room]
	}
	b.b = append(b.b, p...)
	return n, nil
}
func (b *boundedOutput) bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte{}, b.b...)
}

type fakeProcess struct {
	stdout, stderr boundedOutput
	done           chan struct{}
	err            error
	closeJob       func()
	once           sync.Once
	pid            int
}

func (p *fakeProcess) Kill() { p.once.Do(p.closeJob) }

func (p *fakeProcess) PID() int              { return p.pid }
func (p *fakeProcess) Done() <-chan struct{} { return p.done }
func (p *fakeProcess) Stdout() []byte        { return p.stdout.bytes() }
func (p *fakeProcess) Stderr() []byte        { return p.stderr.bytes() }
func (p *fakeProcess) StdoutOverflow() bool {
	p.stdout.mu.Lock()
	defer p.stdout.mu.Unlock()
	return p.stdout.overflow
}
func (p *fakeProcess) Wait(ctx context.Context) error {
	select {
	case <-p.done:
		return p.err
	case <-ctx.Done():
		p.Kill()
		select {
		case <-p.done:
			return ctx.Err()
		case <-time.After(4 * time.Second):
			return errors.New("fixture process did not stop within its deadline")
		}
	}
}
