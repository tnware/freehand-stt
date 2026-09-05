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
	v.BaseURL = "https://first.example.test/v1"
	v.Model = "original-stt"
	v.AuthenticationMode = config.AuthenticationModeAPIKey
	v.PostProcessing.BaseURL = "https://cleanup.example.test/v1"
	v.PostProcessing.Model = "original-cleanup"
	v.PostProcessing.Enabled = true
	v.TextToSpeech.BaseURL = "https://speech.example.test/v1"
	v.TextToSpeech.Model = "original-speech"
	v.TextToSpeech.Voice = "voice"
	v.TextToSpeech.Enabled = true
	v.TextToSpeech.AuthenticationMode = config.AuthenticationModeAPIKey
	if err := s.BeginCredentialChanges(); err != nil {
		t.Fatal(err)
	}
	if err := s.STTCredentials().Set("first-stt-canary"); err != nil {
		t.Fatal(err)
	}
	if err := s.CleanupCredentials().Set("cleanup-canary"); err != nil {
		t.Fatal(err)
	}
	if err := s.SpeechCredentials().Set("speech-canary"); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	svc := settings.NewService(s, v, s.STTCredentials(), s.CleanupCredentials(), &fixtureStartup{}, func() (bool, string) { return true, "" }, nil, nil, nil, nil, nil, nil, settings.WithTextToSpeechCredential(s.SpeechCredentials()), settings.WithConfigurationLoad(s, nil, config.LoadReport{}))
	t.Cleanup(func() { svc.ServiceShutdown() })
	return s, svc
}
func changeConnection(t *testing.T, svc *settings.Service, change savedconnection.Change) settings.SettingsDTO {
	t.Helper()
	v, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &change})
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func TestConnectionsSelectPreserveInactiveKeysAndSnapshots(t *testing.T) {
	s, svc := connectionService(t)
	first := svc.GetSettings()
	captured, err := settings.RequestProfiles(svc).Capture()
	if err != nil {
		t.Fatal(err)
	}
	firstID := first.SavedConnections.Selected[savedconnection.Transcription]
	next := first.Settings
	next.BaseURL = "https://second.example.test/v1"
	next.Model = "second-stt"
	next.Language = "ja"
	added, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: next, ConnectionChange: &savedconnection.Change{Action: savedconnection.Create, Purpose: savedconnection.Transcription, Name: "Second"}})
	if err != nil {
		t.Fatal(err)
	}
	secondID := added.SavedConnections.Selected[savedconnection.Transcription]
	if added.CredentialConfigured {
		t.Fatal("new connection inherited an existing key")
	}
	if _, err := settings.RequestProfiles(svc).Capture(); err == nil {
		t.Fatal("unauthenticated new connection borrowed old key")
	}
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: next, STTCredentialDraft: "second-stt-canary"}); err != nil {
		t.Fatal(err)
	}
	selected := changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Select, Purpose: savedconnection.Transcription, ID: firstID})
	profile, err := settings.RequestProfiles(svc).Capture()
	if err != nil {
		t.Fatal(err)
	}
	if profile.STTCredential != "first-stt-canary" || profile.Settings.Model != "original-stt" || profile.Settings.Language != "ja" {
		t.Fatal("selection lost credential/model or changed operation language")
	}
	if !reflect.DeepEqual(captured.Settings, first.Settings) || captured.STTCredential != "first-stt-canary" {
		t.Fatal("in-flight snapshot changed")
	}
	if !reflect.DeepEqual(selected.PostProcessing, first.PostProcessing) || selected.SavedConnections.Selected[savedconnection.Speech] != first.SavedConnections.Selected[savedconnection.Speech] {
		t.Fatal("selection changed another capability")
	}
	if len(s.vault.(*memoryVault).values) != 4 {
		t.Fatal("inactive credential reclaimed prematurely")
	}
	body, _ := json.Marshal(selected)
	for _, secret := range []string{"first-stt-canary", "second-stt-canary", "cleanup-canary", "speech-canary"} {
		if strings.Contains(string(body), secret) {
			t.Fatal("credential leaked to renderer")
		}
	}
	changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Delete, Purpose: savedconnection.Transcription, ID: secondID})
	if len(s.vault.(*memoryVault).values) != 3 {
		t.Fatal("deleted connection credential not reclaimed")
	}
	s.Close()
	again := newStore(s.path, s.legacy, s.vault)
	defer again.Close()
	if got := loadStore(t, again); got.BaseURL != first.BaseURL {
		t.Fatal("selection not durable")
	}
}
func TestDuplicateCredentialsRemainIndependentAndDeleteRequiresReplacement(t *testing.T) {
	s, svc := connectionService(t)
	original := svc.GetSettings()
	id := original.SavedConnections.Selected[savedconnection.Transcription]
	duplicate := changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Duplicate, Purpose: savedconnection.Transcription, ID: id, Name: "Copy"})
	copyID := duplicate.SavedConnections.Selected[savedconnection.Transcription]
	if len(s.vault.(*memoryVault).values) != 3 {
		t.Fatal("duplicate rewrote credential")
	}
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: duplicate.Settings, STTCredentialDraft: "independent-copy-canary"}); err != nil {
		t.Fatal(err)
	}
	changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Select, Purpose: savedconnection.Transcription, ID: id})
	key, _ := s.STTCredentials().Get()
	if key != "first-stt-canary" {
		t.Fatal("editing duplicate changed original credential")
	}
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Delete, Purpose: savedconnection.Transcription, ID: id}}); err == nil {
		t.Fatal("deleted selected connection without replacement")
	}
	changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Delete, Purpose: savedconnection.Transcription, ID: id, ReplacementID: copyID})
	key, _ = s.STTCredentials().Get()
	if key != "independent-copy-canary" {
		t.Fatal("replacement not selected")
	}
	if len(s.vault.(*memoryVault).values) != 3 {
		t.Fatal("obsolete original credential retained")
	}
}
func TestConnectionChangesRejectStaleSelectionsAndRollBackSQLFailure(t *testing.T) {
	s, svc := connectionService(t)
	original := svc.GetSettings()
	id := original.SavedConnections.Selected[savedconnection.Transcription]
	duplicate := changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Duplicate, Purpose: savedconnection.Transcription, ID: id, Name: "Copy"})
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: original.Settings, ExpectedConnections: original.SavedConnections.Selected}); err == nil {
		t.Fatal("stale editor overwrote selected connection")
	}
	if _, err := s.db.Exec(`CREATE TRIGGER reject_connection BEFORE UPDATE ON saved_connections BEGIN SELECT RAISE(ABORT,'fixture'); END;`); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Select, Purpose: savedconnection.Transcription, ID: id}}); err == nil {
		t.Fatal("failed catalog transaction accepted")
	}
	if got := svc.GetSettings(); !reflect.DeepEqual(got.SavedConnections, duplicate.SavedConnections) {
		t.Fatal("failed save published connection change")
	}
	if got := loadStore(t, s); !reflect.DeepEqual(got, duplicate.Settings) {
		t.Fatal("failed connection save changed database")
	}
}
func TestConnectionValidationAndIndependentSelections(t *testing.T) {
	_, svc := connectionService(t)
	v := svc.GetSettings()
	id := v.SavedConnections.Selected[savedconnection.Cleanup]
	for _, change := range []savedconnection.Change{
		{Action: savedconnection.Select, Purpose: savedconnection.Speech, ID: id},
		{Action: savedconnection.Rename, Purpose: savedconnection.Cleanup, ID: id, Name: "invalid\nname"},
		{Action: savedconnection.Select, Purpose: savedconnection.Cleanup, ID: "missing"},
	} {
		if _, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &change}); err == nil {
			t.Fatal("invalid change accepted")
		}
	}
	updated := changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Duplicate, Purpose: savedconnection.Cleanup, ID: id, Name: "Cleanup copy"})
	if updated.SavedConnections.Selected[savedconnection.Transcription] != v.SavedConnections.Selected[savedconnection.Transcription] {
		t.Fatal("cleanup selection changed STT")
	}
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Rename, Purpose: savedconnection.Cleanup, ID: id, Name: "cleanup COPY"}}); err == nil {
		t.Fatal("duplicate name accepted")
	}
}

func TestConnectionSwitchPreservesIncompatibleSharedOptions(t *testing.T) {
	_, svc := connectionService(t)
	original := svc.GetSettings()
	id := original.SavedConnections.Selected[savedconnection.Transcription]
	next := original.Settings
	next.CompatibilityProfile = compatibility.Speaches
	next.TranscriptionOptions.Hotwords = "Freehand"
	added, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: next, ConnectionChange: &savedconnection.Change{Action: savedconnection.Create, Purpose: savedconnection.Transcription, Name: "Speaches"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Select, Purpose: savedconnection.Transcription, ID: id}}); err == nil {
		t.Fatal("switch accepted shared options unsupported by the target provider")
	}
	got := svc.GetSettings()
	if got.TranscriptionOptions.Hotwords != "Freehand" || !reflect.DeepEqual(got.SavedConnections, added.SavedConnections) {
		t.Fatal("failed switch discarded options or changed selection")
	}
}
