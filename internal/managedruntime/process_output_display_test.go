package managedruntime

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

// Exercise the same capture, consent and polling boundary as the viewer. Every
// byte split includes UTF-8 and control-string terminators split across writes.
func TestProcessOutputDisplayPolicyAcrossWrites(t *testing.T) {
	for _, tc := range []struct{ name, input, want string }{
		{"colors", "\x1b[1;31mred\x1b[0m \x1b[38;5;202;48;2;1;2;3m世🙂\x1b[m", "\x1b[1;31mred\x1b[0m \x1b[38;5;202;48;2;1;2;3m世🙂\x1b[m"},
		{"styles", "\x1b[2;3;4;5;7;8;9;21;22;23;24;25;27;28;29;53mstyled\x1b[55;39;49;97;107m", "\x1b[2;3;4;5;7;8;9;21;22;23;24;25;27;28;29;53mstyled\x1b[55;39;49;97;107m"},
		{"progress", "10%\r\x1b[K20%\b0\tready\r\x1b[0Kdone\x1b[1K\x1b[2K\n", "10%\r\x1b[K20%\b0\tready\r\x1b[0Kdone\x1b[1K\x1b[2K\n"},
		{"osc", "a\x1b]52;c;Y2xpcA==\ab\x1b]8;;https://host\x1b\\link\x1b]8;;\x1b\\c\x1b]0;title\ad", "ablinkcd"},
		{"strings", "a\x1bPsecret\aSTILL SECRET\x1b\\b\x1b_hidden\x1b\\c\x1b^hidden\x1b\\d\x1bXhidden\x1b\\e", "abcde"},
		{"embedded-esc", "a\x1b]0;hidden\x1b[31mstill hidden\x1b\x1b\\b", "ab"},
		{"c1", "a\u009d52;c;secret\u009cb\u0090secret\u009cc\u009b31md\u009b2Je", "abcde"},
		{"other-csi", "a\x1b[2J\x1b[H\x1b[1A\x1b[2C\x1b[?1049h\x1b[?25l\x1b[20t\x1b[6n\x1b[!p\x1b[3Kb", "ab"},
		{"malformed", "a\x1b[?31m\x1b[31 m\x1b[38;5;256m\x1b[38;2;1;2m\x1b[38:2:1:2:3m\x1b[31;;1m\x1b[999m\x1b[0000m\x1b[31\x00mb", "ab"},
		{"oversized", "a\x1b[" + strings.Repeat("1;", 40) + "31mb\x1b[" + strings.Repeat("1;", 16) + "1mc", "abc"},
		{"truncated-csi", "safe\x1b[38;2;1", "safe"},
		{"truncated-osc", "safe\x1b]52;c;secret", "safe"},
		{"truncated-esc", "safe\x1b", "safe"},
		{"controls", "a\a\x00\x7f\u0085\u202eb\x1b(Bc\x1b7d", "abcd"},
		{"cancelled", "a\x1b[31\x18b\x1b[31\x1b[32mc", "ab\x1b[32mc"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for split := 0; split <= len(tc.input); split++ {
				w := &worker{}
				m := &Manager{workers: map[string]*worker{"test": w}}
				w.appendProcessOutput(0, "stdout", []byte(tc.input[:split]))
				w.appendProcessOutput(0, "stdout", []byte(tc.input[split:]))
				hidden, err := m.ReadProcessOutput(OutputRequest{InstanceID: "test"})
				if err != nil || hidden.Enabled || len(hidden.Chunks) != 0 {
					t.Fatal("private capture escaped consent")
				}
				if err := m.EnableProcessOutput(InstanceRequest{InstanceID: "test"}); err != nil {
					t.Fatal(err)
				}
				snap, err := m.ReadProcessOutput(OutputRequest{InstanceID: "test"})
				if err != nil {
					t.Fatal(err)
				}
				if got := displaySnapshotText(t, snap); got != tc.want {
					t.Fatalf("split %d: got %q, want %q", split, got, tc.want)
				}
			}
		})
	}
}

// An independent output grammar checks every chunk from a fresh parser state:
// eviction or polling must never expose an unfinished escape or UTF-8 rune.
func displaySnapshotText(t *testing.T, snap OutputSnapshot) string {
	t.Helper()
	var out strings.Builder
	for _, c := range snap.Chunks {
		if len(c.Text) > 4096 || !utf8.ValidString(c.Text) {
			t.Fatalf("invalid chunk: %q", c.Text)
		}
		text := c.Text
		for len(text) > 0 {
			if text[0] == '\x1b' {
				if !strings.HasPrefix(text, "\x1b[") {
					t.Fatalf("unsafe escape: %q", text)
				}
				end := 2
				for end < len(text) && (text[end] >= '0' && text[end] <= '9' || text[end] == ';') {
					end++
				}
				if end >= len(text) || (text[end] != 'm' && text[end] != 'K') || end > 66 {
					t.Fatalf("unsafe/partial CSI: %q", text)
				}
				text = text[end+1:]
				continue
			}
			r, n := utf8.DecodeRuneInString(text)
			if unicode.IsControl(r) && r != '\r' && r != '\b' && r != '\t' && r != '\n' || unicode.Is(unicode.Cf, r) {
				t.Fatalf("unsafe control %U", r)
			}
			text = text[n:]
		}
		out.WriteString(c.Text)
	}
	return out.String()
}

func TestProcessOutputDisplayChunkBoundary(t *testing.T) {
	for _, prefix := range []string{strings.Repeat("x", 4093), strings.Repeat("\xff", 1364) + "x"} {
		w := &worker{}
		input := prefix + "\x1b[38;2;1;2;3m世🙂\x1b[0m"
		w.appendProcessOutput(0, "stdout", []byte(input))
		got := displaySnapshotText(t, OutputSnapshot{Chunks: w.output.chunks})
		want := strings.ReplaceAll(input, "\xff", "�")
		if got != want {
			t.Fatalf("chunk boundary changed display tokens: got %q, want %q", got, want)
		}
	}
}

func TestProcessOutputDisplayChunkEviction(t *testing.T) {
	for _, tinyWrites := range []bool{false, true} {
		t.Run(fmt.Sprint(tinyWrites), func(t *testing.T) {
			w := &worker{}
			m := &Manager{workers: map[string]*worker{"test": w}}
			input := strings.Repeat("x", 4093) + "\x1b[31m世\xff\x1b[0m\r\x1b[K"
			count := 100
			if tinyWrites {
				input = "\x1b[31m世\x1b[0m"
				count = 1100
			}
			for i := 0; i < count; i++ {
				w.appendProcessOutput(0, "stdout", []byte(input))
			}
			if err := m.EnableProcessOutput(InstanceRequest{InstanceID: "test"}); err != nil {
				t.Fatal(err)
			}
			snap, err := m.ReadProcessOutput(OutputRequest{InstanceID: "test"})
			if err != nil {
				t.Fatal(err)
			}
			text := displaySnapshotText(t, snap)
			if !snap.Truncated || len(text) > outputLimit || len(snap.Chunks) > 1024 || len(snap.Chunks) == 0 {
				t.Fatalf("unbounded/unmarked snapshot: bytes %d chunks %d", len(text), len(snap.Chunks))
			}
			if !strings.Contains(text, "\x1b[31m") {
				t.Fatal("colors lost from retained tail")
			}
			// Resume from every retained boundary, not only from the oldest chunk.
			for _, c := range snap.Chunks {
				page, err := m.ReadProcessOutput(OutputRequest{InstanceID: "test", After: c.Sequence})
				if err != nil {
					t.Fatal(err)
				}
				displaySnapshotText(t, page)
			}
		})
	}
}

func TestProcessOutputDisplayDiscardedPayloadRecovery(t *testing.T) {
	for _, tc := range []struct{ start, end string }{{"\x1b]52;c;", "\x1b\\"}, {"\x1bP", "\x1b\\"}, {"\x1b[", "m"}} {
		w := &worker{}
		w.appendProcessOutput(0, "stdout", []byte(tc.start+strings.Repeat("9", outputLimit*2)))
		if len(w.output.chunks) != 0 || w.output.bytes != 0 {
			t.Fatal("unfinished/oversized control retained as display output")
		}
		w.appendProcessOutput(0, "stdout", []byte(tc.end+"\x1b[32mrecovered\x1b[0m"))
		if got := displaySnapshotText(t, OutputSnapshot{Chunks: w.output.chunks}); got != "\x1b[32mrecovered\x1b[0m" {
			t.Fatalf("discard did not recover: %q", got)
		}
	}
}

func TestProcessOutputDisplayStreamAndResetIsolation(t *testing.T) {
	for _, reset := range []bool{false, true} {
		for _, partial := range []string{"\x1b[31", "\x1b]52;c;private", "\x1bPprivate", "\xe4\xb8"} {
			t.Run(fmt.Sprintf("reset=%v/%q", reset, partial), func(t *testing.T) {
				w := &worker{}
				m := &Manager{workers: map[string]*worker{"test": w}}
				req := InstanceRequest{InstanceID: "test"}
				_ = m.EnableProcessOutput(req)
				w.appendProcessOutput(0, "stdout", []byte("old"+partial))
				w.appendProcessOutput(0, "stderr", []byte("\x1b[32mstderr\x1b[0m"))
				before, _ := m.ReadProcessOutput(OutputRequest{InstanceID: "test"})
				if displaySnapshotText(t, before) != "old\x1b[32mstderr\x1b[0m" {
					t.Fatal("stdout parser swallowed stderr")
				}
				if reset {
					w.observeStartup(t.Context())
					w.appendProcessOutput(0, "stdout", []byte("stale"))
				} else if err := m.ClearProcessOutput(req); err != nil {
					t.Fatal(err)
				}
				cleared, _ := m.ReadProcessOutput(OutputRequest{InstanceID: "test", After: before.Next})
				if !cleared.Enabled || !cleared.Truncated || len(cleared.Chunks) != 0 || cleared.Next <= before.Next {
					t.Fatal("clear/reset did not invalidate consumed cursor")
				}
				w.appendProcessOutput(w.outputLaunch, "stdout", []byte("\x1b[31mfresh\x1b[0m"))
				w.appendProcessOutput(w.outputLaunch, "stderr", []byte("\r\x1b[Kdone"))
				after, _ := m.ReadProcessOutput(OutputRequest{InstanceID: "test", After: cleared.Next})
				if got := displaySnapshotText(t, after); got != "\x1b[31mfresh\x1b[0m\r\x1b[Kdone" {
					t.Fatalf("decoder state survived reset: %q", got)
				}
			})
		}
	}
}
