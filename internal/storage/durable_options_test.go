package storage

import (
	"encoding/json"
	"github.com/tnware/freehand-stt/internal/config"
	"reflect"
	"testing"
)

func TestDurableTranscriptionOptionsSaveReopen(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	// Retired renderer keys have no writable representation in current settings.
	if err := json.Unmarshal([]byte(`{"transcriptionOptions":{"prompt":"File context","hotwords":"retired file"},"voiceTranscription":{"transcriptionOptions":{"prompt":"Voice context","hotwords":"retired voice"},"options":{"vocabulary":"retired realtime","boost":9}}}`), &v); err != nil {
		t.Fatal(err)
	}
	v.Vocabulary = config.VocabularySettings{Terms: "Shared terms", Voice: true, Files: false, Boost: 4}
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	got := loadStore(t, s)
	if !reflect.DeepEqual(v, got) {
		t.Fatalf("accepted durable options changed on reopen: before=%+v after=%+v", v, got)
	}
	projected, err := config.WithVocabulary(got, false)
	if err != nil {
		t.Fatal(err)
	}
	if projected.TranscriptionOptions.Inference().Hotwords != "" {
		t.Fatal("retired file hotwords survived disabled shared vocabulary")
	}
}
