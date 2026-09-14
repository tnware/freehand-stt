package storage

import (
	"github.com/tnware/freehand-stt/internal/config"
	"reflect"
	"testing"
)

func TestSharedVocabularyRoundTrip(t *testing.T) {
	s := testStore(t)
	got := loadStore(t, s)
	got.Vocabulary = config.VocabularySettings{Terms: "Freehand\n東京\nFile term", Voice: true, Files: true, Boost: 2.5}
	if err := s.Save(got); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	if loaded := loadStore(t, s); !reflect.DeepEqual(loaded, got) {
		t.Fatal("shared vocabulary did not survive restart")
	}
	got.Vocabulary.Files = false
	got.Vocabulary.Terms = "Updated\n日本語"
	if err := s.Save(got); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	if loaded := loadStore(t, s); !reflect.DeepEqual(loaded, got) {
		t.Fatal("vocabulary edit did not survive restart")
	}
}
