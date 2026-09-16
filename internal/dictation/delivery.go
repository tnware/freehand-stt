package dictation

import (
	"context"
	"errors"
)

func (c *recorder) copyPending() error {
	c.mu.Lock()
	if c.closed.Load() {
		c.mu.Unlock()
		return errors.New("application is shutting down")
	}
	text := c.pending
	gen := c.status.Generation
	if text == "" || !c.status.CanCopy {
		c.mu.Unlock()
		return errors.New("no transcript is waiting to be copied")
	}
	if e := c.targetPlatform.Copy(context.Background(), text); e != nil {
		c.mu.Unlock()
		return e
	}
	if c.status.Generation == gen && c.pending == text {
		c.pending = ""
		c.status = Status{State: Idle, Generation: gen, Message: "Transcript copied", Transcript: text}
		s := c.status
		c.mu.Unlock()
		c.publish(s)
		return nil
	}
	c.mu.Unlock()
	return errors.New("pending transcript changed before copy completed")
}

// Explicit current-result commands never depend on optional history retention.
func (c *recorder) copyCurrent(generation uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed.Load() || generation != c.status.Generation || c.status.Transcript == "" {
		return errors.New("current transcript is no longer available")
	}
	return c.targetPlatform.Copy(context.Background(), c.status.Transcript)
}

func (c *recorder) clearCurrent(generation uint64) error {
	c.mu.Lock()
	if c.closed.Load() || generation != c.status.Generation || (c.status.State != Idle && c.status.State != Failed) {
		c.mu.Unlock()
		return errors.New("current transcript cannot be cleared")
	}
	c.pending = ""
	c.status = Status{State: Idle, Generation: generation}
	status := c.status
	c.mu.Unlock()
	c.publish(status)
	return nil
}
