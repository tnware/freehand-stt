//go:build windows

package storage

import (
	"fmt"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/modelsettings"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"github.com/tnware/freehand-stt/internal/settings"
	"reflect"
	"testing"
)

func TestRememberedModelsPersistAndForgetAtomically(t *testing.T) {
	s, svc := connectionService(t)
	original := svc.GetSettings()
	id := original.SavedConnections.Selected[savedconnection.Cleanup]
	v := original.Settings
	v.PostProcessing.Model = "s1-alias"
	v.PostProcessing.Preset = config.PostProcessingPresetS1Mini
	v.PostProcessing.Styling = "formal"
	first, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: v})
	if err != nil {
		t.Fatal(err)
	}
	v = first.Settings
	v.PostProcessing.Model = "generic-alias"
	v.PostProcessing.Preset = config.PostProcessingPresetGeneric
	v.PostProcessing.SystemPrompt = "Custom cleanup instruction."
	second, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: v})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, e := range second.RememberedModels.Entries {
		if e.ConnectionID == id && e.Purpose == savedconnection.Cleanup {
			count++
			if e.Model == "s1-alias" && (e.Options.Styling != "formal" || e.Options.Profile != "s1-mini") {
				t.Fatal("specialized model overwritten")
			}
		}
	}
	if count < 2 {
		t.Fatal("model options were not retained")
	}
	before := s.RememberedModels()
	if _, err = svc.SaveSettings(settings.SaveSettingsRequest{ForgetModel: &modelsettings.Key{ConnectionID: "stale", Purpose: savedconnection.Cleanup, Model: "s1-alias"}}); err == nil {
		t.Fatal("stale connection accepted")
	}
	if !reflect.DeepEqual(before, s.RememberedModels()) {
		t.Fatal("failed forget mutated memory")
	}
	forgotten, err := svc.SaveSettings(settings.SaveSettingsRequest{ForgetModel: &modelsettings.Key{ConnectionID: id, Purpose: savedconnection.Cleanup, Model: "s1-alias"}})
	if err != nil {
		t.Fatal(err)
	}
	if forgotten.PostProcessing.Model != "generic-alias" {
		t.Fatal("forget changed another model")
	}
	path, legacy, vault := s.path, s.legacy, s.vault
	s.Close()
	reopened := newStore(path, legacy, vault)
	defer reopened.Close()
	got := loadStore(t, reopened)
	if got.PostProcessing.Model != "generic-alias" {
		t.Fatal("restart lost model")
	}
	for _, e := range reopened.RememberedModels().Entries {
		if e.Model == "s1-alias" {
			t.Fatal("forgotten model returned after restart")
		}
	}
}
func TestFailedSaveRollsBackRememberedOptions(t *testing.T) {
	s, svc := connectionService(t)
	before := svc.GetSettings()
	if _, err := s.db.Exec(`CREATE TRIGGER fail_remembered BEFORE INSERT ON remembered_models BEGIN SELECT RAISE(ABORT,'fixture'); END;`); err != nil {
		t.Fatal(err)
	}
	v := before.Settings
	v.Model = "new-model"
	v.Language = "ja"
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: v}); err == nil {
		t.Fatal("failed SQL save accepted")
	}
	got := svc.GetSettings()
	if !reflect.DeepEqual(before.RememberedModels, got.RememberedModels) || got.Model != before.Model {
		t.Fatal("rollback lost active or saved options")
	}
}
func TestVersionSixUpgradePreservesCurrentModelOptions(t *testing.T) {
	s, svc := connectionService(t)
	v := svc.GetSettings().Settings
	v.Model = "asr-model"
	v.Language = "ja"
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: v}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`ALTER TABLE speech_settings DROP COLUMN speech_language; ALTER TABLE speech_settings DROP COLUMN speech_instructions; ALTER TABLE remembered_models DROP COLUMN speech_language; ALTER TABLE remembered_models DROP COLUMN speech_instructions; DROP TABLE vocabulary_settings; DROP TABLE voice_transcription_settings; DROP TABLE voice_request_headers; DELETE FROM selected_connections WHERE purpose='voice'; DELETE FROM saved_connection_uses WHERE purpose='voice'; DELETE FROM credential_refs WHERE purpose='voice'; DROP TABLE remembered_models;DELETE FROM goose_db_version WHERE version_id>=7;`); err != nil {
		t.Fatal(err)
	}
	path, legacy, vault := s.path, s.legacy, s.vault
	s.Close()
	next := newStore(path, legacy, vault)
	defer next.Close()
	got := loadStore(t, next)
	if got.Model != "asr-model" || got.Language != "ja" {
		t.Fatal("migration changed active options")
	}
	found := false
	for _, e := range next.RememberedModels().Entries {
		if e.Purpose == savedconnection.Transcription && e.Model == "asr-model" {
			found = true
			if e.Options.Language != "ja" {
				t.Fatal("migration lost model language")
			}
		}
	}
	if !found {
		t.Fatal("migration did not seed preferences")
	}
}

func TestRememberedModelLimitAndDeletion(t *testing.T) {
	s, svc := connectionService(t)
	original := svc.GetSettings()
	id := original.SavedConnections.Selected[savedconnection.Cleanup]
	count := 0
	for _, e := range original.RememberedModels.Entries {
		if e.ConnectionID == id && e.Purpose == savedconnection.Cleanup {
			count++
		}
	}
	for i := count; i < modelsettings.MaxPerUse; i++ {
		v := svc.GetSettings().Settings
		v.PostProcessing.Model = fmt.Sprintf("model-%d", i)
		if _, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: v}); err != nil {
			t.Fatal(err)
		}
	}
	before := svc.GetSettings()
	v := before.Settings
	v.PostProcessing.Model = "overflow"
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: v}); err == nil {
		t.Fatal("model limit not enforced")
	}
	if !reflect.DeepEqual(before.RememberedModels, svc.GetSettings().RememberedModels) {
		t.Fatal("limit failure evicted preferences")
	}
	changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Select, Purpose: savedconnection.Cleanup, ID: ""})
	changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Delete, ID: id})
	var rows int
	if err := s.db.QueryRow("SELECT count(*) FROM remembered_models WHERE connection_id=?", id).Scan(&rows); err != nil || rows != 0 {
		t.Fatal("deleted connection retained preferences", err)
	}
}
func TestForgetActiveModelClearsSelection(t *testing.T) {
	_, svc := connectionService(t)
	v := svc.GetSettings().Settings
	v.PostProcessing.Model = "temporary"
	v.PostProcessing.Enabled = true
	saved, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: v})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.SaveSettings(settings.SaveSettingsRequest{ForgetModel: &modelsettings.Key{ConnectionID: saved.SavedConnections.Selected[savedconnection.Cleanup], Purpose: savedconnection.Cleanup, Model: "temporary"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.PostProcessing.Model != "" || got.PostProcessing.Enabled {
		t.Fatal("forget left model active")
	}
	for _, e := range got.RememberedModels.Entries {
		if e.Purpose == savedconnection.Cleanup && e.Model == "temporary" {
			t.Fatal("forgotten model was written again")
		}
	}
}

func TestModelDraftBatchIsAtomicAndRestrictedToSelectedConnections(t *testing.T) {
	_, svc := connectionService(t)
	before := svc.GetSettings()
	id := before.SavedConnections.Selected[savedconnection.Cleanup]
	a := modelsettings.Defaults()[savedconnection.Cleanup]
	a.Styling = "formal"
	a.Profile = "s1-mini"
	b := modelsettings.Defaults()[savedconnection.Cleanup]
	b.SystemPrompt = "Preserve punctuation."
	edits := []modelsettings.Edit{{ConnectionID: id, Purpose: savedconnection.Cleanup, Model: "edited-s1", Options: a}, {ConnectionID: id, Purpose: savedconnection.Cleanup, Model: "edited-generic", Options: b}}
	bad := append([]modelsettings.Edit{}, edits...)
	bad[1].ConnectionID = "stale"
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: before.Settings, ModelEdits: bad}); err == nil {
		t.Fatal("accepted stale model owner")
	}
	if !reflect.DeepEqual(before.RememberedModels, svc.GetSettings().RememberedModels) {
		t.Fatal("failed batch partially changed models")
	}
	active := modelsettings.Apply(before.Settings, savedconnection.Cleanup, "edited-generic", b)
	got, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: active, ModelEdits: edits})
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, e := range got.RememberedModels.Entries {
		if e.Model == "edited-s1" {
			found++
			if e.Options.Styling != "formal" || e.Selected {
				t.Fatal("inactive model edit lost")
			}
		}
		if e.Model == "edited-generic" {
			found++
			if !e.Selected || e.Options.SystemPrompt != b.SystemPrompt {
				t.Fatal("active model edit lost")
			}
		}
	}
	if found != 2 {
		t.Fatal("batch did not save both models")
	}
}
