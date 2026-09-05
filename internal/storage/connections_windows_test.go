//go:build windows

package storage

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"github.com/tnware/freehand-stt/internal/settings"
)

func connectionService(t *testing.T) (*Store, *settings.Service) {
	t.Helper()
	s := testStore(t)
	v := loadStore(t, s)
	svc := settings.NewService(s, v, s.STTCredentials(), s.CleanupCredentials(), &fixtureStartup{}, func() (bool, string) { return true, "" }, nil, nil, nil, nil, nil, nil, settings.WithTextToSpeechCredential(s.SpeechCredentials()), settings.WithConfigurationLoad(s, nil, config.LoadReport{}))
	t.Cleanup(func() { svc.ServiceShutdown() })
	for _, p := range []savedconnection.Purpose{savedconnection.Transcription, savedconnection.Cleanup, savedconnection.Speech} {
		d := savedconnection.Details{CompatibilityProfile: compatibility.Generic, BaseURL: "https://" + string(p) + ".example.test/v1", AuthenticationMode: config.AuthenticationModeAPIKey, Headers: map[string]string{}}
		if p == savedconnection.Cleanup {
			d.AuthenticationMode = config.AuthenticationModeNone
		}
		out, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Create, Purpose: p, Name: "Original " + string(p), Details: &d}, ConnectionCredentialDraft: string(p) + "-canary"})
		if err != nil {
			t.Fatal(err)
		}
		id := connectionID(t, out, "Original "+string(p))
		changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Select, Purpose: p, ID: id})
	}
	v = svc.GetSettings().Settings
	v.Model = "original-stt"
	v.PostProcessing.Model = "original-cleanup"
	v.PostProcessing.Enabled = true
	v.TextToSpeech.Model = "original-speech"
	v.TextToSpeech.Voice = "voice"
	v.TextToSpeech.Enabled = true
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: v}); err != nil {
		t.Fatal(err)
	}
	return s, svc
}
func connectionID(t *testing.T, v settings.SettingsDTO, name string) string {
	t.Helper()
	for _, c := range v.SavedConnections.Entries {
		if c.Name == name {
			return c.ID
		}
	}
	t.Fatal("missing connection", name)
	return ""
}
func changeConnection(t *testing.T, svc *settings.Service, change savedconnection.Change) settings.SettingsDTO {
	t.Helper()
	v, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &change})
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func TestNewConnectionIsInactiveAndRuntimeSettingsDoNotEditIt(t *testing.T) {
	s, svc := connectionService(t)
	original := svc.GetSettings()
	captured, err := settings.RequestProfiles(svc).Capture()
	if err != nil {
		t.Fatal(err)
	}
	d := savedconnection.Extract(original.Settings, savedconnection.Transcription)
	d.BaseURL = "https://second.example.test/v1"
	added, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Create, Purpose: savedconnection.Transcription, Name: "Second", Details: &d}, ConnectionCredentialDraft: "second-canary"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(added.Settings, original.Settings) || !reflect.DeepEqual(added.SavedConnections.Selected, original.SavedConnections.Selected) {
		t.Fatal("creating a connection activated it")
	}
	id := connectionID(t, added, "Second")
	selected := changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Select, Purpose: savedconnection.Transcription, ID: id})
	if selected.Model != "" || selected.SetupCompleted {
		t.Fatal("switch did not require a new model choice")
	}
	v := selected.Settings
	v.Model = "second-model"
	v.Language = "ja"
	v.BaseURL = "https://runtime-cannot-edit.example.test/v1"
	saved, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: v})
	if err != nil {
		t.Fatal(err)
	}
	if saved.BaseURL != d.BaseURL || saved.Model != "second-model" || saved.Language != "ja" {
		t.Fatal("connection and runtime fields were not separated")
	}
	key, _ := s.STTCredentials().Get()
	if key != "second-canary" {
		t.Fatal("wrong selected credential")
	}
	if captured.STTCredential != "stt-canary" || captured.Settings.Model != "original-stt" {
		t.Fatal("in-flight snapshot changed")
	}
	if !reflect.DeepEqual(saved.PostProcessing, original.PostProcessing) {
		t.Fatal("STT selection changed cleanup settings")
	}
	body, _ := json.Marshal(saved)
	if strings.Contains(string(body), "canary") {
		t.Fatal("credential leaked to renderer")
	}
	s.Close()
	again := newStore(s.path, s.legacy, s.vault)
	defer again.Close()
	if got := loadStore(t, again); got.Model != "second-model" {
		t.Fatal("runtime model not durable")
	}
}
func TestInactiveConnectionEditAndDuplicateKeysAreIndependent(t *testing.T) {
	s, svc := connectionService(t)
	original := svc.GetSettings()
	id := original.SavedConnections.Selected[savedconnection.Transcription]
	duplicate := changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Duplicate, Purpose: savedconnection.Transcription, ID: id, Name: "Copy"})
	copyID := connectionID(t, duplicate, "Copy")
	if duplicate.SavedConnections.Selected[savedconnection.Transcription] != id {
		t.Fatal("duplicate activated itself")
	}
	d := savedconnection.Extract(original.Settings, savedconnection.Transcription)
	d.BaseURL = "https://copy.example.test/v1"
	updated, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Update, Purpose: savedconnection.Transcription, ID: copyID, Name: "Copy edited", Details: &d}, ConnectionCredentialDraft: "copy-canary"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(updated.Settings, original.Settings) {
		t.Fatal("inactive edit changed runtime")
	}
	key, _ := s.STTCredentials().Get()
	if key != "stt-canary" {
		t.Fatal("editing duplicate changed original key")
	}
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Delete, Purpose: savedconnection.Transcription, ID: id}}); err == nil {
		t.Fatal("deleted active connection")
	}
	changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Select, Purpose: savedconnection.Transcription, ID: copyID})
	changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Delete, Purpose: savedconnection.Transcription, ID: id})
	if len(s.vault.(*memoryVault).values) != 3 {
		t.Fatal("unused original key retained")
	}
	key, _ = s.STTCredentials().Get()
	if key != "copy-canary" {
		t.Fatal("copy key not active")
	}
}
func TestConnectionChangesRejectStaleSelectionsAndRollBackSQLFailure(t *testing.T) {
	s, svc := connectionService(t)
	original := svc.GetSettings()
	id := original.SavedConnections.Selected[savedconnection.Transcription]
	duplicate := changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Duplicate, Purpose: savedconnection.Transcription, ID: id, Name: "Copy"})
	copyID := connectionID(t, duplicate, "Copy")
	active := changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Select, Purpose: savedconnection.Transcription, ID: copyID})
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: original.Settings, ExpectedConnections: original.SavedConnections.Selected}); err == nil {
		t.Fatal("stale editor accepted")
	}
	if _, err := s.db.Exec(`CREATE TRIGGER reject_connection BEFORE UPDATE ON saved_connections BEGIN SELECT RAISE(ABORT,'fixture'); END;`); err != nil {
		t.Fatal(err)
	}
	d := savedconnection.Extract(active.Settings, savedconnection.Transcription)
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Update, Purpose: savedconnection.Transcription, ID: copyID, Name: "Failed edit", Details: &d}, ConnectionCredentialDraft: "failed-canary"}); err == nil {
		t.Fatal("failed save accepted")
	}
	if got := svc.GetSettings(); !reflect.DeepEqual(got.SavedConnections, active.SavedConnections) {
		t.Fatal("failed save published catalog")
	}
	if len(s.vault.(*memoryVault).values) != 3 {
		t.Fatal("failed edit leaked staged key")
	}
	if got := loadStore(t, s); !reflect.DeepEqual(got, active.Settings) {
		t.Fatal("failed save changed runtime")
	}
}
func TestConnectionValidationAndEmptyCatalog(t *testing.T) {
	fresh := testStore(t)
	loadStore(t, fresh)
	if len(fresh.ConnectionCatalog().Entries) != 0 {
		t.Fatal("fresh install contains invented connections")
	}
	_, svc := connectionService(t)
	v := svc.GetSettings()
	id := v.SavedConnections.Selected[savedconnection.Cleanup]
	for _, c := range []savedconnection.Change{{Action: savedconnection.Select, Purpose: savedconnection.Speech, ID: id}, {Action: savedconnection.Rename, Purpose: savedconnection.Cleanup, ID: id, Name: "bad\nname"}, {Action: savedconnection.Select, Purpose: savedconnection.Cleanup, ID: "missing"}} {
		if _, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &c}); err == nil {
			t.Fatal("invalid change accepted")
		}
	}
	changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Select, Purpose: savedconnection.Cleanup, ID: ""})
	out := changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Delete, Purpose: savedconnection.Cleanup, ID: id})
	if out.PostProcessing.Enabled || out.PostProcessing.BaseURL != "" || out.SavedConnections.Selected[savedconnection.Cleanup] != "" {
		t.Fatal("last connection not cleared")
	}
}
func TestConnectionSwitchPreservesIncompatibleSharedOptions(t *testing.T) {
	_, svc := connectionService(t)
	original := svc.GetSettings()
	id := original.SavedConnections.Selected[savedconnection.Transcription]
	d := savedconnection.Extract(original.Settings, savedconnection.Transcription)
	d.CompatibilityProfile = compatibility.Speaches
	added, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Create, Purpose: savedconnection.Transcription, Name: "Speaches", Details: &d}})
	if err != nil {
		t.Fatal(err)
	}
	selected := changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Select, Purpose: savedconnection.Transcription, ID: connectionID(t, added, "Speaches")})
	next := selected.Settings
	next.Model = "whisper"
	next.TranscriptionOptions.Hotwords = "Freehand"
	saved, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: next})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Select, Purpose: savedconnection.Transcription, ID: id}}); err == nil {
		t.Fatal("unsupported hotwords accepted")
	}
	got := svc.GetSettings()
	if got.TranscriptionOptions.Hotwords != "Freehand" || !reflect.DeepEqual(got.SavedConnections, saved.SavedConnections) {
		t.Fatal("failed switch lost values")
	}
}
