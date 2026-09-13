package managedruntime

import (
	"context"
	"errors"
	"os/exec"
	"sync"
	"time"
)

const outputLimit = 256 << 10

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

type ownedProcess struct {
	stdout, stderr boundedOutput
	done           chan struct{}
	err            error
	closeJob       func()
	once           sync.Once
	pid            int
}

func (p *ownedProcess) kill() { p.once.Do(p.closeJob) }
func launchOwned(ctx context.Context, exe string, args []string, dir string, env []string) (*ownedProcess, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p := &ownedProcess{done: make(chan struct{})}
	cmd := exec.Command(exe, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = &p.stdout
	cmd.Stderr = &p.stderr
	cmd.WaitDelay = 2 * time.Second
	closeJob, err := startInJob(cmd)
	if err != nil {
		return nil, err
	}
	p.closeJob = closeJob
	p.pid = cmd.Process.Pid
	go func() { p.err = cmd.Wait(); p.kill(); close(p.done) }()
	go func() {
		select {
		case <-ctx.Done():
			p.kill()
		case <-p.done:
		}
	}()
	return p, nil
}
func (p *ownedProcess) wait(ctx context.Context) error {
	select {
	case <-p.done:
		return p.err
	case <-ctx.Done():
		p.kill()
		select {
		case <-p.done:
			return ctx.Err()
		case <-time.After(4 * time.Second):
			return errors.New("Managed speech process did not stop within its deadline.")
		}
	}
}
