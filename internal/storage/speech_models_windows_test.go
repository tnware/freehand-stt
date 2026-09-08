//go:build windows

package storage

import (
	"reflect"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/modelsettings"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"github.com/tnware/freehand-stt/internal/settings"
)

func TestSpeechModelOptionsRememberedAcrossSelectionRestartAndRollback(t *testing.T) {
	s, svc := connectionService(t)
	d := savedconnection.Details{CompatibilityProfile: compatibility.VLLMOmni, BaseURL: "https://speech.example.test/v1", AuthenticationMode: config.AuthenticationModeNone}
	created := changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Create, Purpose: savedconnection.Speech, Uses: []savedconnection.Purpose{savedconnection.Speech}, Name: "Omni", Details: &d})
	id := connectionID(t, created, "Omni")
	changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Select, Purpose: savedconnection.Speech, ID: id})
	want := modelprofile.SpeechOptions{Language: "ja", Instructions: "穏やかに話す"}
	v := svc.GetSettings().Settings
	v.TextToSpeech.Model = "customvoice-alias"
	v.TextToSpeech.ModelProfile = modelprofile.Qwen3TTS
	v.TextToSpeech.Voice = "ryan"
	v.TextToSpeech.Options = want
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: v}); err != nil {
		t.Fatal(err)
	}
	v.TextToSpeech.Model = "another-model"
	v.TextToSpeech.ModelProfile = modelprofile.Generic
	v.TextToSpeech.Options = modelprofile.SpeechOptions{}
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: v}); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	v = loadStore(t, s)
	catalog := s.RememberedModels()
	found := false
	for _, entry := range catalog.Entries {
		if entry.ConnectionID != id || entry.Model != "customvoice-alias" {
			continue
		}
		found = true
		if entry.Selected || entry.Options.Speech != want {
			t.Fatal("inactive model options did not survive restart")
		}
		v.TextToSpeech.Speed = 1.5
		restored := modelsettings.Select(v, savedconnection.Speech, entry.Model, entry.Options)
		if restored.TextToSpeech.Options != want || restored.TextToSpeech.Voice != "ryan" || restored.TextToSpeech.Speed != 1.5 {
			t.Fatal("selection lost model options or replaced workflow speed")
		}
	}
	if !found {
		t.Fatal("remembered Qwen model missing")
	}
	// A failed transaction must retain both active settings and the inactive profile.
	if _, err := s.db.Exec(`CREATE TRIGGER fail_speech_models BEFORE INSERT ON remembered_models BEGIN SELECT RAISE(ABORT,'fixture'); END;`); err != nil {
		t.Fatal(err)
	}
	before := loadStore(t, s)
	changed := before
	changed.TextToSpeech.Model = "failed-model"
	if err := s.Save(changed); err == nil {
		t.Fatal("failed remembered-model transaction accepted")
	}
	s = reopen(t, s)
	if !reflect.DeepEqual(before, loadStore(t, s)) || !reflect.DeepEqual(catalog, s.RememberedModels()) {
		t.Fatal("failed transaction changed active or remembered speech settings")
	}
}
