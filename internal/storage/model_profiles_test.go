package storage

import (
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"reflect"
	"testing"
)

func TestCurrentModelProfilesPreserveTaskIntentAndConnections(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	d := savedconnection.Extract(v, savedconnection.Transcription)
	d.BaseURL = "https://speech.example.test/v1"
	v = createSelectedConnection(t, s, v, "Shared", savedconnection.Transcription, d, savedconnection.Transcription, savedconnection.Cleanup)
	id := s.ConnectionCatalog().Selected[savedconnection.Transcription]
	v = selectConnection(t, s, v, savedconnection.Cleanup, id)
	v.Model = "server-model"
	v.PostProcessing.Model = "cleanup-alias"
	v.PostProcessing.Preset = config.PostProcessingPresetS1Mini
	v.PostProcessing.Styling = "formal"
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	catalog := s.ConnectionCatalog()
	s = reopen(t, s)
	if got := loadStore(t, s); !reflect.DeepEqual(got, v) || !reflect.DeepEqual(catalog, s.ConnectionCatalog()) {
		t.Fatal("current profile/task/connection choices did not survive restart")
	}
	if v.ModelProfile != modelprofile.Generic {
		t.Fatal("unexpected default model profile")
	}
	v.PostProcessing.Preset = config.PostProcessingPresetGeneric
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	if got := loadStore(t, s); !reflect.DeepEqual(got, v) {
		t.Fatal("profile selection did not survive restart")
	}
	if _, err := s.db.Exec(`UPDATE transcription_settings SET model_profile='future' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	if _, err := s.Load(); err == nil {
		t.Fatal("unknown stored model profile accepted")
	}
}
