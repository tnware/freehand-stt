package inference

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestTranscriptionStreamCompletionContracts(t *testing.T) {
	const delta = "data: {\"type\":\"transcript.text.delta\",\"delta\":\"partial\"}\n\n"
	const done = "data: {\"type\":\"transcript.text.done\",\"text\":\"Final.\"}\n\n"
	const legacy = "data: {\"text\":\"legacy\"}\n\n"
	for _, tc := range []struct {
		name, body, want  string
		fail, unsupported bool
	}{
		{"typed final", delta + done, "Final.", false, false},
		{"final only", done, "Final.", false, false},
		{"typed EOF", delta, "partial", true, false},
		{"sentinel is not typed final", delta + "data: [DONE]\n\n", "partial", true, false},
		{"typed empty final replaces deltas", delta + "data: {\"type\":\"transcript.text.done\",\"text\":\"\"}\n\n", "", false, false},
		{"missing final text", delta + "data: {\"type\":\"transcript.text.done\"}\n\n", "partial", true, true},
		{"null final text", delta + "data: {\"type\":\"transcript.text.done\",\"text\":null}\n\n", "partial", true, true},
		{"missing delta", "data: {\"type\":\"transcript.text.delta\"}\n\n", "", true, true},
		{"legacy EOF", legacy, "legacy", false, false},
		{"legacy sentinel", legacy + "data: [DONE]\n\n", "legacy", false, false},
		{"mixed stream still needs typed final", legacy + delta, "legacypartial", true, false},
		{"empty response", "", "", true, false},
		{"keepalive only", ": keepalive\n\n", "", true, false},
		{"sentinel only", "data: [DONE]\n\n", "", true, false},
		{"unknown dialect", "data: {\"choices\":[{\"delta\":{\"content\":\"unread\"}}]}\n\n", "", true, true},
		{"server error preserves accepted text", delta + "data: {\"type\":\"error\",\"message\":\"private server detail\"}\n\n", "partial", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := readTranscriptionSSE(strings.NewReader(tc.body), "", nil)
			var unsupported *FileStreamUnsupportedError
			isUnsupported := errors.As(err, &unsupported)
			text := result.Text
			if isUnsupported {
				text = unsupported.PartialText
			}
			if (err != nil) != tc.fail || isUnsupported != tc.unsupported || text != tc.want {
				t.Fatalf("text=%q err=%v unsupported=%v", text, err, isUnsupported)
			}
			if err != nil && strings.Contains(err.Error(), "private server detail") {
				t.Fatal("provider error leaked")
			}
		})
	}
}

type interruptedStream struct{}

func (interruptedStream) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestTranscriptionStreamReadFailurePreservesAcceptedText(t *testing.T) {
	reader := io.MultiReader(strings.NewReader("data: {\"type\":\"transcript.text.delta\",\"delta\":\"accepted\"}\n\n"), interruptedStream{})
	result, err := readTranscriptionSSE(reader, "", nil)
	var unsupported *FileStreamUnsupportedError
	if err == nil || result.Text != "accepted" || errors.As(err, &unsupported) {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}
