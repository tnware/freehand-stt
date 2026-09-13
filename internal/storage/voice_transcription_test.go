package storage

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"reflect"
	"testing"
)

func TestCurrentVoiceAndFileSelectionsAreIndependent(t *testing.T) {
	for _, realtime := range []bool{false, true} {
		t.Run(map[bool]string{false: "completed", true: "realtime"}[realtime], func(t *testing.T) {
			s := testStore(t)
			v := loadStore(t, s)
			d := savedconnection.Extract(v, savedconnection.Transcription)
			d.BaseURL = "https://file.example.test/v1"
			d.Headers["X-Route"] = "file"
			v = createSelectedConnection(t, s, v, "Files", savedconnection.Transcription, d)
			v.Model = "file-model"
			v.Language = "de"
			if err := s.Save(v); err != nil {
				t.Fatal(err)
			}
			before := v
			d = savedconnection.Extract(v, savedconnection.Voice)
			d.BaseURL = "https://voice.example.test/v1"
			if realtime {
				d.CompatibilityProfile = compatibility.NeMoSpeechV1
			}
			v = createSelectedConnection(t, s, v, "Voice", savedconnection.Voice, d)
			v.VoiceTranscription.Model = "voice-model"
			if realtime {
				v.VoiceTranscription.ModelProfile = modelprofile.Nemotron35
			}
			v.VoiceTranscription.Realtime = realtime
			v.VoiceTranscription.Language = "auto"
			if err := s.Save(v); err != nil {
				t.Fatal(err)
			}
			s = reopen(t, s)
			got := loadStore(t, s)
			if !reflect.DeepEqual(got, v) {
				t.Fatalf("voice selection did not survive restart: got %#v want %#v", got.VoiceTranscription, v.VoiceTranscription)
			}
			if got.BaseURL != before.BaseURL || got.Model != before.Model || got.Language != before.Language || !reflect.DeepEqual(got.Headers, before.Headers) {
				t.Fatal("voice selection changed file task")
			}
			if s.ConnectionCatalog().Selected[savedconnection.Voice] == s.ConnectionCatalog().Selected[savedconnection.Transcription] {
				t.Fatal("distinct tasks share wrong selection")
			}
		})
	}
}
