package inference

import (
	"context"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"net/http"
	"strings"
	"testing"
)

type modelProfileTransport struct{ calls int }

func (r *modelProfileTransport) RoundTrip(*http.Request) (*http.Response, error) {
	r.calls++
	panic("invalid profile reached network")
}

func TestWrongRoleModelProfilesFailBeforeAnyRequest(t *testing.T) {
	transport := &modelProfileTransport{}
	base := New()
	base.HTTP = &http.Client{Transport: transport}
	client := base.WithModelProfile(modelprofile.S1Mini)
	if _, err := client.Transcribe(context.Background(), "https://example.invalid", "model", "", "", nil, []byte("audio")); err == nil {
		t.Fatal("accepted cleanup profile for microphone")
	}
	if _, err := client.TranscribeFile(context.Background(), "https://example.invalid", "model", "", "", nil, "sample.wav", 5, strings.NewReader("audio"), true, FileTranscriptionCallbacks{}); err == nil {
		t.Fatal("accepted cleanup profile for files")
	}
	if _, err := base.SynthesizeSpeech(context.Background(), "https://example.invalid", "", SpeechRequest{ModelProfile: modelprofile.S1Mini, Speed: 1}); err == nil {
		t.Fatal("accepted cleanup profile for speech")
	}
	if _, err := base.WithModelProfile("future").ChatCompletion(context.Background(), "https://example.invalid", "model", "", "system", "text"); err == nil {
		t.Fatal("accepted unknown cleanup profile")
	}
	if transport.calls != 0 {
		t.Fatal("unqualified request reached network")
	}
	if base.modelProfile != "" || client.modelProfile != modelprofile.S1Mini {
		t.Fatal("model selection mutated shared client")
	}
}
