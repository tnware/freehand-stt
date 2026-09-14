package managedruntime

import (
	"strings"
	"testing"
)

func TestProcessOutputSplitUnicode(t *testing.T) {
	w := &worker{}
	text := []byte("世🙂")
	for _, b := range text {
		w.appendProcessOutput(0, "stdout", []byte{b})
	}
	var got strings.Builder
	for _, c := range w.output.chunks {
		got.WriteString(c.Text)
	}
	if got.String() != string(text) {
		t.Fatalf("split unicode: %q", got.String())
	}
}

func TestProcessOutputResetInvalidatesFullyConsumedCursor(t *testing.T) {
	w := &worker{}
	m := &Manager{workers: map[string]*worker{"test": w}}
	_ = m.EnableProcessOutput(InstanceRequest{InstanceID: "test"})
	w.appendProcessOutput(0, "stdout", []byte("old launch"))
	before, _ := m.ReadProcessOutput(OutputRequest{InstanceID: "test"})
	w.observeStartup(t.Context())
	after, _ := m.ReadProcessOutput(OutputRequest{InstanceID: "test", After: before.Next})
	if !after.Truncated || after.Next <= before.Next || len(after.Chunks) != 0 {
		t.Fatal("new startup did not invalidate the viewer's consumed cursor")
	}
}

func TestProcessOutputOptInBoundedOrdered(t *testing.T) {
	w := &worker{}
	m := &Manager{workers: map[string]*worker{"test": w}}
	req := InstanceRequest{InstanceID: "test"}
	w.appendProcessOutput(0, "stdout", []byte("private before open"))
	hidden, _ := m.ReadProcessOutput(OutputRequest{InstanceID: "test"})
	if len(hidden.Chunks) != 0 {
		t.Fatal("unauthorized read")
	}
	if err := m.EnableProcessOutput(req); err != nil {
		t.Fatal(err)
	}
	w.appendProcessOutput(0, "stdout", []byte("first\x1b[31m red\x1b[0m\r\n"))
	w.appendProcessOutput(0, "stderr", []byte("second"))
	s, err := m.ReadProcessOutput(OutputRequest{InstanceID: "test"})
	if err != nil || len(s.Chunks) != 3 || s.Chunks[0].Text != "private before open" || s.Chunks[1].Text != "first red\n" || s.Chunks[2].Stream != "stderr" || s.Chunks[2].Sequence <= s.Chunks[1].Sequence {
		t.Fatalf("snapshot: %+v %v", s, err)
	}
	cursor := s.Next
	for i := 0; i < 100; i++ {
		w.appendProcessOutput(0, "stdout", []byte(strings.Repeat("x", 8192)))
	}
	s, _ = m.ReadProcessOutput(OutputRequest{InstanceID: "test", After: cursor})
	total := 0
	for _, c := range s.Chunks {
		total += len(c.Text)
		if len(c.Text) > 4096 {
			t.Fatal("unbounded chunk")
		}
	}
	if total > outputLimit || !s.Truncated {
		t.Fatalf("unbounded/unmarked: %d %+v", total, s.Truncated)
	}
	if err := m.DisableProcessOutput(req); err != nil {
		t.Fatal(err)
	}
	w.appendProcessOutput(0, "stdout", []byte("after close"))
	s, _ = m.ReadProcessOutput(OutputRequest{InstanceID: "test"})
	if s.Enabled || len(s.Chunks) != 0 {
		t.Fatal("disable must clear and stop capture")
	}
}
