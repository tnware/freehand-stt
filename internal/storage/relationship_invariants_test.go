package storage

import (
	"reflect"
	"testing"

	"github.com/tnware/freehand-stt/internal/savedconnection"
)

func TestNormalizedConnectionRelationships(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	d := savedconnection.Extract(v, savedconnection.Transcription)
	d.BaseURL = "https://relationships.example.test/v1"
	v = createSelectedConnection(t, s, v, "Relationships", savedconnection.Transcription, d)
	id := s.ConnectionCatalog().Selected[savedconnection.Transcription]
	for _, tc := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO saved_connection_uses(connection_id,purpose) VALUES('missing','stt')`, nil},
		{`INSERT INTO selected_connections(purpose,connection_id) VALUES('speech',?)`, []any{id}},
		{`INSERT INTO saved_connection_headers(connection_id,name,value) VALUES('missing','X-Test','value')`, nil},
		{`INSERT INTO remembered_models(connection_id,purpose,model,selected) VALUES('missing','stt','orphan',0)`, nil},
		{`DELETE FROM saved_connection_uses WHERE connection_id=?`, []any{id}},
		{`DELETE FROM saved_connections WHERE id=?`, []any{id}},
	} {
		if _, err := s.db.Exec(tc.query, tc.args...); err == nil {
			t.Fatalf("relationship constraint accepted %s", tc.query)
		}
	}
	v.Model = "first"
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	v.Model = "second"
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	before := s.RememberedModels()
	if len(before.Entries) != 2 {
		t.Fatal("missing remembered model history")
	}
	if _, err := s.db.Exec(`UPDATE remembered_models SET selected=1 WHERE connection_id=?`, id); err == nil {
		t.Fatal("multiple selected models accepted for one connection/use")
	}
	if _, err := s.db.Exec(`UPDATE remembered_models SET purpose='speech' WHERE connection_id=?`, id); err == nil {
		t.Fatal("remembered model update accepted unsupported use")
	}
	v.Language = "ja"
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	v = loadStore(t, s)
	if !reflect.DeepEqual(s.RememberedModels(), before) {
		t.Fatal("connection use rewrite lost remembered models")
	}
	v = selectConnection(t, s, v, savedconnection.Transcription, "")
	if _, err := s.db.Exec(`DELETE FROM saved_connection_uses WHERE connection_id=?`, id); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM remembered_models WHERE connection_id=?`, id).Scan(&count); err != nil || count != 0 {
		t.Fatalf("removed use left orphan models: count=%d err=%v", count, err)
	}
}
