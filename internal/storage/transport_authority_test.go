package storage

import (
	"reflect"
	"testing"

	"github.com/tnware/freehand-stt/internal/savedconnection"
)

func TestSaveRejectsUnownedTransport(t *testing.T) {
	for _, p := range []savedconnection.Purpose{savedconnection.Transcription, savedconnection.Voice, savedconnection.Cleanup, savedconnection.Speech} {
		t.Run(string(p), func(t *testing.T) {
			s := testStore(t)
			old := loadStore(t, s)
			d := savedconnection.Extract(old, p)
			d.BaseURL = "https://unowned.example.test/v1"
			next := savedconnection.Apply(old, p, d)
			next.Language = "fr"
			if err := s.Save(next); err == nil {
				t.Fatal("accepted transport without a selected connection owner")
			}
			if got := loadStore(t, reopen(t, s)); !reflect.DeepEqual(got, old) {
				t.Fatal("rejected transport changed committed settings")
			}
		})
	}
}

func TestSelectedConnectionTransportSaveIsNormalized(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	d := savedconnection.Extract(v, savedconnection.Transcription)
	d.BaseURL = "https://owned.example.test/v1"
	v = createSelectedConnection(t, s, v, "Shared", savedconnection.Transcription, d, savedconnection.Transcription, savedconnection.Voice, savedconnection.Cleanup)
	id := s.ConnectionCatalog().Selected[savedconnection.Transcription]
	v = selectConnection(t, s, v, savedconnection.Voice, id)
	v = selectConnection(t, s, v, savedconnection.Cleanup, id)
	v.Headers = map[string]string{"X-Test": "日本語"}
	v.VoiceTranscription.Headers = map[string]string{"X-Test": "日本語"}
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	got := loadStore(t, s)
	if !reflect.DeepEqual(got, v) {
		t.Fatal("selected transport did not round trip")
	}
	c := s.ConnectionCatalog().Entries[0]
	if !reflect.DeepEqual(c.Details.Headers, v.Headers) {
		t.Fatal("saved connection is not transport owner")
	}
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM saved_connection_headers WHERE connection_id=? AND name='X-Test' AND value='日本語'`, id).Scan(&count); err != nil || count != 1 {
		t.Fatalf("normalized header count=%d err=%v", count, err)
	}
}

func TestSaveRejectsConflictingSharedTransport(t *testing.T) {
	s := testStore(t)
	old := loadStore(t, s)
	d := savedconnection.Extract(old, savedconnection.Transcription)
	d.BaseURL = "https://shared.example.test/v1"
	old = createSelectedConnection(t, s, old, "Shared", savedconnection.Transcription, d, savedconnection.Transcription, savedconnection.Voice)
	old = selectConnection(t, s, old, savedconnection.Voice, s.ConnectionCatalog().Selected[savedconnection.Transcription])
	next := old
	next.Headers = map[string]string{"X-Test": "conflict"}
	if err := s.Save(next); err == nil {
		t.Fatal("accepted conflicting projections of shared transport")
	}
	if got := loadStore(t, reopen(t, s)); !reflect.DeepEqual(got, old) {
		t.Fatal("conflicting save changed committed settings")
	}
}
