package managedruntime

import (
	"errors"
	"strings"
	"time"
)

// OutputRequest is exclusively for the explicitly opted-in process viewer.
type OutputRequest struct {
	InstanceID string `json:"instanceID"`
	After      uint64 `json:"after"`
}
type OutputChunk struct {
	Sequence  uint64 `json:"sequence"`
	Timestamp int64  `json:"timestamp"`
	Stream    string `json:"stream"`
	Text      string `json:"text"`
}
type OutputSnapshot struct {
	Enabled   bool          `json:"enabled"`
	Chunks    []OutputChunk `json:"chunks"`
	Next      uint64        `json:"next"`
	Truncated bool          `json:"truncated"`
}
type processOutput struct {
	enabled bool
	chunks  []OutputChunk
	bytes   int
	next    uint64
	floor   uint64
	// Decoder state is per stream and survives writes until clear/next launch.
	decoder [2]processDisplayDecoder
}

func (b *processOutput) clear() {
	b.chunks = nil
	b.bytes = 0
	// Reserve a reset cursor even if the viewer consumed every prior chunk.
	// The next read must invalidate its old text before new output arrives.
	b.next++
	b.floor = b.next
	b.decoder = [2]processDisplayDecoder{}
}

// outputAccess never uses operation admission: viewers work during startup.
// Lock order remains manager -> worker; no output enters status or callbacks.
func (m *Manager) outputAccess(id string, f func(*worker)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.initErr != nil {
		return errNotReady
	}
	w := m.workers[id]
	if w == nil {
		return errors.New("Choose an existing managed runtime instance.")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return errNotReady
	}
	f(w)
	return nil
}

// EnableProcessOutput authorizes reads of the bounded private runtime tail.
// Output may contain sensitive text and paths; this is not redaction.
func (m *Manager) EnableProcessOutput(r InstanceRequest) error {
	return m.outputAccess(r.InstanceID, func(w *worker) { w.output.enabled = true })
}

// DisableProcessOutput revokes reads but retains the bounded private tail.
func (m *Manager) DisableProcessOutput(r InstanceRequest) error {
	return m.outputAccess(r.InstanceID, func(w *worker) { w.output.enabled = false })
}
func (m *Manager) ClearProcessOutput(r InstanceRequest) error {
	return m.outputAccess(r.InstanceID, func(w *worker) { w.output.clear() })
}
func (m *Manager) ReadProcessOutput(r OutputRequest) (OutputSnapshot, error) {
	out := OutputSnapshot{Chunks: []OutputChunk{}}
	err := m.outputAccess(r.InstanceID, func(w *worker) {
		b := &w.output
		out.Enabled = b.enabled
		out.Next = b.next
		out.Truncated = r.After < b.floor || r.After > b.next
		if !b.enabled {
			return
		}
		for _, c := range b.chunks {
			if c.Sequence > r.After || r.After > b.next {
				out.Chunks = append(out.Chunks, c)
			}
		}
	})
	return out, err
}
func (w *worker) appendProcessOutput(launch uint64, stream string, p []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || launch != w.outputLaunch {
		return
	}
	b := &w.output
	index := 0
	if stream == "stderr" {
		index = 1
	} else if stream != "stdout" {
		return
	}
	for len(p) > 0 {
		n := len(p)
		if n > 4096 {
			n = 4096
		}
		text := b.decoder[index].write(p[:n])
		p = p[n:]
		if text == "" {
			continue
		}
		// Invalid UTF-8 replacement and completed pending CSI may expand input.
		// Evict only whole chunks, each containing complete display tokens.
		for len(text) > 0 {
			size := processDisplayChunkSize(text, 4096)
			part := strings.Clone(text[:size])
			text = text[size:]
			b.next++
			for len(b.chunks) > 0 && (b.bytes+len(part) > outputLimit || len(b.chunks) >= 1024) {
				b.bytes -= len(b.chunks[0].Text)
				b.floor = b.chunks[0].Sequence
				copy(b.chunks, b.chunks[1:])
				b.chunks[len(b.chunks)-1] = OutputChunk{}
				b.chunks = b.chunks[:len(b.chunks)-1]
			}
			b.chunks = append(b.chunks, OutputChunk{Sequence: b.next, Timestamp: time.Now().UnixMilli(), Stream: stream, Text: part})
			b.bytes += len(part)
		}
	}
}
