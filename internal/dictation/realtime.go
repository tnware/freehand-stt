package dictation

import "github.com/tnware/freehand-stt/internal/realtime"

func realtimeFailed(session *realtime.Session) <-chan struct{} {
	if session == nil {
		return nil
	}
	return session.Failed
}

func (c *recorder) publishLive(generation uint64, update realtime.Update) {
	c.mu.Lock()
	if c.closed.Load() || c.status.Generation != generation || (c.status.State != Recording && c.status.State != Transcribing) {
		c.mu.Unlock()
		return
	}
	c.status.LiveFinal = update.Final
	c.status.LivePartial = update.Partial
	status := c.status
	c.mu.Unlock()
	c.publish(status)
}
