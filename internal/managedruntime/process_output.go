package managedruntime

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
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
	escape  [2]byte
	pending [2][]byte
}

func (b *processOutput) clear() {
	b.chunks = nil
	b.bytes = 0
	// Reserve a reset cursor even if the viewer consumed every prior chunk.
	// The next read must invalidate its old text before new output arrives.
	b.next++
	b.floor = b.next
	b.escape = [2]byte{}
	b.pending = [2][]byte{}
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
		text := plainProcessText(p[:n], &b.escape[index], &b.pending[index])
		p = p[n:]
		if text == "" {
			continue
		}
		// Invalid UTF-8 replacement may expand input; retain bounded chunks.
		for len(text) > 0 {
			size := len(text)
			if size > 4096 {
				size = 4096
				for size > 0 && text[size]&0xc0 == 0x80 {
					size--
				}
			}
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

// Strip terminal escape sequences across writes and non-text controls. The
// renderer must still use text nodes, never HTML or a terminal interpreter.
func plainProcessText(p []byte, state *byte, pending *[]byte) string {
	out := append(make([]byte, 0, len(p)+len(*pending)), (*pending)...)
	*pending = nil
	for _, c := range p {
		switch *state {
		case 1:
			switch c {
			case '[':
				*state = 2
			case ']', 'P', '^', '_':
				*state = 3
			default:
				*state = 0
			}
			continue
		case 2:
			if c >= 0x40 && c <= 0x7e {
				*state = 0
			}
			continue
		case 3:
			if c == 7 {
				*state = 0
			}
			if c == 27 {
				*state = 4
			}
			continue
		case 4:
			if c == '\\' {
				*state = 0
			} else {
				*state = 3
			}
			continue
		}
		if c == 27 {
			*state = 1
			continue
		}
		if c < 32 && c != '\n' && c != '\t' || c == 127 {
			continue
		}
		out = append(out, c)
	}
	end := 0
	for end < len(out) {
		if !utf8.FullRune(out[end:]) {
			*pending = append([]byte(nil), out[end:]...)
			break
		}
		_, size := utf8.DecodeRune(out[end:])
		end += size
	}
	out = out[:end]
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, strings.ToValidUTF8(string(out), "�"))
}
