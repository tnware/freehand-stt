package process

import (
	"context"
	"errors"
	"os/exec"
	"sync"
	"time"
)

// OutputLimit bounds each private diagnostic prefix.
const OutputLimit = 256 << 10

const outputLimit = OutputLimit

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

// Process is an opaque OS-owned child tree with bounded diagnostic snapshots.
// Obtain it through Launch; the zero value is not usable.
type Process struct {
	stdout, stderr boundedOutput
	done           chan struct{}
	err            error
	closeJob       func()
	pid            int
}

// Kill terminates the entire owned tree and is safe to call repeatedly.
func (p *Process) Kill() { p.closeJob() }

// Launch returns only after native tree ownership has been established.
func Launch(ctx context.Context, exe string, args []string, dir string, env []string, observer Observer) (*Process, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	p := &Process{done: make(chan struct{})}
	cmd := exec.Command(exe, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = &p.stdout
	cmd.Stderr = &p.stderr
	if observer.Output != nil {
		cmd.Stdout = observedOutput{&p.stdout, observer.Output, "stdout"}
		cmd.Stderr = observedOutput{&p.stderr, observer.Output, "stderr"}
	}
	cmd.WaitDelay = 2 * time.Second
	closeJob, pid, err := startOwnedProcess(cmd)
	if err != nil {
		return nil, err
	}
	p.closeJob = sync.OnceFunc(closeJob)
	p.pid = pid
	if observer.Started != nil {
		observer.Started(p)
	}
	stopCancellation := context.AfterFunc(ctx, p.Kill)
	go func() {
		p.err = cmd.Wait()
		p.Kill()
		stopCancellation()
		close(p.done)
	}()
	return p, nil
}
func (p *Process) Wait(ctx context.Context) error {
	select {
	case <-p.done:
		return p.err
	case <-ctx.Done():
		p.Kill()
		select {
		case <-p.done:
			return ctx.Err()
		case <-time.After(4 * time.Second):
			return errors.New("Managed speech process did not stop within its deadline.")
		}
	}
}

// Observer connects private process output and admission to the feature owner.
// Callbacks must be bounded and may run concurrently for stdout and stderr.
// Started runs after OS ownership is established, before Launch returns.
type Observer struct {
	Output  func(stream string, data []byte)
	Started func(*Process)
}

func (p *Process) PID() int              { return p.pid }
func (p *Process) Done() <-chan struct{} { return p.done }
func (p *Process) Stdout() []byte        { return p.stdout.bytes() }
func (p *Process) Stderr() []byte        { return p.stderr.bytes() }
func (p *Process) StdoutOverflow() bool {
	p.stdout.mu.Lock()
	defer p.stdout.mu.Unlock()
	return p.stdout.overflow
}

type observedOutput struct {
	prefix *boundedOutput
	output func(string, []byte)
	stream string
}

func (o observedOutput) Write(p []byte) (int, error) {
	n, err := o.prefix.Write(p)
	o.output(o.stream, p)
	return n, err
}
