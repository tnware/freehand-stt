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
	svc := settings.NewService(s, v, s.STTCredentials(), s.CleanupCredentials(), &fixtureStartup{}, func() (bool, string) { return true, "" }, nil, nil, nil, nil, nil, nil, settings.WithTextToSpeechCredential(s.SpeechCredentials()), settings.WithVoiceCredential(s.VoiceCredentials()), settings.WithConfigurationLoad(s, nil, config.LoadReport{}))
	t.Cleanup(func() { svc.ServiceShutdown() })
	for _, p := range []savedconnection.Purpose{savedconnection.Transcription, savedconnection.Cleanup, savedconnection.Speech} {
		d := savedconnection.Details{CompatibilityProfile: compatibility.Generic, BaseURL: "https://" + string(p) + ".example.test/v1", AuthenticationMode: config.AuthenticationModeAPIKey, Headers: map[string]string{}}
		out, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Create, Purpose: p, Uses: []savedconnection.Purpose{p}, Name: "Original " + string(p), Details: &d}, ConnectionCredentialDraft: string(p) + "-canary"})
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
	added, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Create, Purpose: savedconnection.Transcription, Uses: []savedconnection.Purpose{savedconnection.Transcription}, Name: "Second", Details: &d}, ConnectionCredentialDraft: "second-canary"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(added.Settings, original.Settings) || !reflect.DeepEqual(added.SavedConnections.Selected, original.SavedConnections.Selected) {
		t.Fatal("creating a connection activated it")
	}
	id := connectionID(t, added, "Second")
	selected := changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Select, Purpose: savedconnection.Transcription, ID: id})
	if selected.Model != "" || selected.SetupCompleted != original.SetupCompleted {
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
	updated, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Update, Purpose: savedconnection.Transcription, Uses: []savedconnection.Purpose{savedconnection.Transcription}, ID: copyID, Name: "Copy edited", Details: &d}, ConnectionCredentialDraft: "copy-canary"})
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
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Update, Purpose: savedconnection.Transcription, Uses: []savedconnection.Purpose{savedconnection.Transcription}, ID: copyID, Name: "Failed edit", Details: &d}, ConnectionCredentialDraft: "failed-canary"}); err == nil {
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
func TestConnectionSwitchRestoresOptionsForEachBackend(t *testing.T) {
	_, svc := connectionService(t)
	original := svc.GetSettings()
	id := original.SavedConnections.Selected[savedconnection.Transcription]
	d := savedconnection.Extract(original.Settings, savedconnection.Transcription)
	d.CompatibilityProfile = compatibility.Speaches
	added, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Create, Purpose: savedconnection.Transcription, Uses: []savedconnection.Purpose{savedconnection.Transcription}, Name: "Speaches", Details: &d}})
	if err != nil {
		t.Fatal(err)
	}
	selected := changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Select, Purpose: savedconnection.Transcription, ID: connectionID(t, added, "Speaches")})
	next := selected.Settings
	next.Model = "whisper"
	next.Language = "ja"
	next.Vocabulary.Terms = "Freehand"
	next.Vocabulary.Files = true
	saved, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: next})
	if err != nil {
		t.Fatal(err)
	}
	got := changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Select, Purpose: savedconnection.Transcription, ID: id})
	if got.Language != "ja" {
		t.Fatal("connection switch replaced task language")
	}
	if got.TranscriptionOptions.Hotwords != "" {
		t.Fatal("hotwords leaked to another backend")
	}
	got = changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Select, Purpose: savedconnection.Transcription, ID: saved.SavedConnections.Selected[savedconnection.Transcription]})
	if got.Vocabulary.Terms != "Freehand" || !got.Vocabulary.Files || got.TranscriptionOptions.Hotwords != "" || got.Model != "whisper" {
		t.Fatal("returning to the connection lost model options")
	}

}

func TestReusableConnectionSharesTransportAndKeyAcrossFeatures(t *testing.T) {
	s, svc := connectionService(t)
	d := savedconnection.Details{CompatibilityProfile: compatibility.Generic, BaseURL: "https://gateway.example.test/v1", AuthenticationMode: config.AuthenticationModeAPIKey, Headers: map[string]string{"X-Gateway": "speech"}}
	uses := []savedconnection.Purpose{savedconnection.Transcription, savedconnection.Cleanup, savedconnection.Speech}
	added, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Create, Name: "Shared gateway", Uses: uses, Details: &d}, ConnectionCredentialDraft: "shared-canary"})
	if err != nil {
		t.Fatal(err)
	}
	id := connectionID(t, added, "Shared gateway")
	for _, p := range uses {
		changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Select, Purpose: p, ID: id})
	}
	v := svc.GetSettings().Settings
	v.Model = "stt-model"
	v.PostProcessing.Model = "chat-model"
	v.PostProcessing.Enabled = true
	v.TextToSpeech.Model = "tts-model"
	v.TextToSpeech.Voice = "voice"
	v.TextToSpeech.Enabled = true
	if _, err = svc.SaveSettings(settings.SaveSettingsRequest{Settings: v}); err != nil {
		t.Fatal(err)
	}
	captured, err := settings.RequestProfiles(svc).Capture()
	if err != nil {
		t.Fatal(err)
	}
	for _, keys := range []interface{ Get() (string, error) }{s.STTCredentials(), s.CleanupCredentials(), s.SpeechCredentials()} {
		if key, err := keys.Get(); err != nil || key != "shared-canary" {
			t.Fatal("shared key unavailable")
		}
	}
	d.BaseURL = "https://new-gateway.example.test/v1"
	updated, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Update, ID: id, Name: "Shared gateway", Uses: uses, Details: &d}, ConnectionCredentialDraft: "replacement-canary"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.BaseURL != d.BaseURL || updated.PostProcessing.BaseURL != d.BaseURL || updated.TextToSpeech.BaseURL != d.BaseURL {
		t.Fatal("shared endpoint did not update every active use")
	}
	if updated.Model != "" || updated.PostProcessing.Model != "" || updated.TextToSpeech.Model != "" {
		t.Fatal("endpoint change retained old feature models")
	}
	if captured.STTCredential != "shared-canary" || captured.Settings.BaseURL == d.BaseURL {
		t.Fatal("captured request mutated")
	}
	for _, keys := range []interface{ Get() (string, error) }{s.STTCredentials(), s.CleanupCredentials(), s.SpeechCredentials()} {
		if key, err := keys.Get(); err != nil || key != "replacement-canary" {
			t.Fatal("shared key replacement not atomic")
		}
	}
	// A shared duplicate retains the previous reference independently.
	copied := changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Duplicate, ID: id, Name: "Shared copy"})
	copyID := connectionID(t, copied, "Shared copy")
	if _, err = svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Update, ID: id, Name: "Shared gateway", Uses: []savedconnection.Purpose{savedconnection.Transcription}, Details: &d}}); err == nil {
		t.Fatal("removed an active use")
	}
	for _, p := range uses {
		changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Select, Purpose: p, ID: ""})
	}
	changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Delete, ID: id})
	if _, key, err := s.ResolveSavedConnection(copyID); err != nil || key != "replacement-canary" {
		t.Fatal("delete reclaimed duplicate key")
	}
	s = reopen(t, s)
	loadStore(t, s)
	c, key, err := s.ResolveSavedConnection(copyID)
	if err != nil || key != "replacement-canary" || !c.Supports(savedconnection.Speech) || !c.Supports(savedconnection.Cleanup) {
		t.Fatal("shared connection failed durable reopen")
	}
}
func TestSharedConnectionRejectsUnqualifiedUsesAndRollsBackAllSelections(t *testing.T) {
	s, svc := connectionService(t)
	d := savedconnection.Details{CompatibilityProfile: compatibility.Speaches, BaseURL: "https://speaches.example.test/v1", AuthenticationMode: config.AuthenticationModeAPIKey, Headers: map[string]string{}}
	invalid := savedconnection.Change{Action: savedconnection.Create, Name: "Unsupported", Uses: []savedconnection.Purpose{savedconnection.Cleanup}, Details: &d}
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &invalid}); err == nil {
		t.Fatal("Speaches cleanup accepted")
	}
	uses := []savedconnection.Purpose{savedconnection.Transcription, savedconnection.Speech}
	added, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Create, Name: "Shared Speaches", Uses: uses, Details: &d}, ConnectionCredentialDraft: "before-canary"})
	if err != nil {
		t.Fatal(err)
	}
	id := connectionID(t, added, "Shared Speaches")
	for _, p := range uses {
		changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Select, Purpose: p, ID: id})
	}
	before := svc.GetSettings()
	if _, err = s.db.Exec(`CREATE TRIGGER reject_shared BEFORE UPDATE ON saved_connections BEGIN SELECT RAISE(ABORT,'fixture'); END;`); err != nil {
		t.Fatal(err)
	}
	d.BaseURL = "https://rejected.example.test/v1"
	_, err = svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Update, ID: id, Name: "Shared Speaches", Uses: uses, Details: &d}, ConnectionCredentialDraft: "rejected-canary"})
	if err == nil {
		t.Fatal("expected injected SQL failure")
	}
	if !reflect.DeepEqual(svc.GetSettings().Settings, before.Settings) || !reflect.DeepEqual(s.ConnectionCatalog(), before.SavedConnections) {
		t.Fatal("partial shared update published")
	}
	for _, keys := range []interface{ Get() (string, error) }{s.STTCredentials(), s.SpeechCredentials()} {
		if key, _ := keys.Get(); key != "before-canary" {
			t.Fatal("shared rollback lost key")
		}
	}
	for _, key := range s.vault.(*memoryVault).values {
		if key == "rejected-canary" {
			t.Fatal("failed replacement leaked vault reference")
		}
	}
}

func TestCreateAndActivateConnectionIsOneCoherentSave(t *testing.T) {
	for _, purpose := range []savedconnection.Purpose{savedconnection.Transcription, savedconnection.Cleanup, savedconnection.Speech} {
		t.Run(string(purpose), func(t *testing.T) {
			s, svc := connectionService(t)
			before := svc.GetSettings()
			captured, err := settings.RequestProfiles(svc).Capture()
			if err != nil {
				t.Fatal(err)
			}
			d := savedconnection.Details{CompatibilityProfile: compatibility.Generic, BaseURL: "https://new.example.test/v1", AuthenticationMode: config.AuthenticationModeAPIKey, Headers: map[string]string{}}
			saved, err := svc.SaveSettings(settings.SaveSettingsRequest{
				ExpectedConnections:       before.SavedConnections.Selected,
				ConnectionChange:          &savedconnection.Change{Action: savedconnection.Create, Name: "New task server", Uses: []savedconnection.Purpose{purpose}, Details: &d, ActivateFor: purpose},
				ConnectionCredentialDraft: "new-task-canary",
			})
			if err != nil {
				t.Fatal(err)
			}
			id := connectionID(t, saved, "New task server")
			if saved.SavedConnections.Selected[purpose] != id {
				t.Fatal("new connection not selected")
			}
			for _, other := range []savedconnection.Purpose{savedconnection.Transcription, savedconnection.Cleanup, savedconnection.Speech} {
				if other != purpose && saved.SavedConnections.Selected[other] != before.SavedConnections.Selected[other] {
					t.Fatal("changed an unrelated selection")
				}
			}
			if saved.Language != before.Language || saved.PostProcessing.SystemPrompt != before.PostProcessing.SystemPrompt || saved.TextToSpeech.Speed != before.TextToSpeech.Speed {
				t.Fatal("changed task intent")
			}
			profile, err := settings.RequestProfiles(svc).Capture()
			if err != nil {
				t.Fatal(err)
			}
			switch purpose {
			case savedconnection.Transcription:
				if profile.STTCredential != "new-task-canary" || saved.Model != "" || saved.SetupCompleted != before.SetupCompleted {
					t.Fatal("incoherent STT activation")
				}
			case savedconnection.Cleanup:
				key, keyErr := s.CleanupCredentials().Get()
				if keyErr != nil || key != "new-task-canary" || saved.PostProcessing.Model != "" || saved.PostProcessing.Enabled {
					t.Fatal("incoherent cleanup activation")
				}
			case savedconnection.Speech:
				key, keyErr := s.SpeechCredentials().Get()
				if keyErr != nil || key != "new-task-canary" || saved.TextToSpeech.Model != "" || saved.TextToSpeech.Enabled {
					t.Fatal("incoherent speech activation")
				}
			}
			if captured.STTCredential != "stt-canary" || captured.Settings.Model != "original-stt" {
				t.Fatal("mutated an in-flight request")
			}
			body, _ := json.Marshal(saved)
			if strings.Contains(string(body), "canary") {
				t.Fatal("credential in renderer snapshot")
			}
			s.Close()
			reopened := newStore(s.path, s.legacy, s.vault)
			defer reopened.Close()
			loadStore(t, reopened)
			if reopened.ConnectionCatalog().Selected[purpose] != id {
				t.Fatal("selection not durable")
			}
		})
	}
}

func TestCreateAndActivateRejectsInvalidIntentAndRollsBackFailure(t *testing.T) {
	s, svc := connectionService(t)
	before := svc.GetSettings()
	d := savedconnection.Extract(before.Settings, savedconnection.Transcription)
	change := savedconnection.Change{Action: savedconnection.Create, Name: "New", Uses: []savedconnection.Purpose{savedconnection.Transcription}, Details: &d, ActivateFor: savedconnection.Transcription}
	for _, invalid := range []savedconnection.Change{
		{Action: savedconnection.Create, Name: "Wrong role", Uses: change.Uses, Details: &d, ActivateFor: savedconnection.Speech},
		{Action: savedconnection.Create, Name: "Unknown role", Uses: change.Uses, Details: &d, ActivateFor: "invented"},
		{Action: savedconnection.Update, ID: before.SavedConnections.Selected[savedconnection.Transcription], Name: "Original stt", Uses: change.Uses, Details: &d, ActivateFor: savedconnection.Transcription},
	} {
		if _, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &invalid}); err == nil {
			t.Fatal("invalid activation accepted")
		}
	}
	if _, err := s.db.Exec(`CREATE TRIGGER reject_setup BEFORE INSERT ON saved_connections BEGIN SELECT RAISE(ABORT,'fixture'); END;`); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &change, ConnectionCredentialDraft: "failed-task-canary"}); err == nil {
		t.Fatal("failed save accepted")
	}
	if got := svc.GetSettings(); !reflect.DeepEqual(got.SavedConnections, before.SavedConnections) || !reflect.DeepEqual(got.Settings, before.Settings) {
		t.Fatal("failed activation published state")
	}
	if len(s.vault.(*memoryVault).values) != 3 {
		t.Fatal("failed activation leaked credential")
	}
	if got := loadStore(t, s); !reflect.DeepEqual(got, before.Settings) {
		t.Fatal("failed activation changed disk settings")
	}
}

func TestFreshTaskSetupNeedsNoOtherConfiguredFeature(t *testing.T) {
	for _, p := range []savedconnection.Purpose{savedconnection.Transcription, savedconnection.Cleanup, savedconnection.Speech} {
		t.Run(string(p), func(t *testing.T) {
			s := testStore(t)
			initial := loadStore(t, s)
			svc := settings.NewService(s, initial, s.STTCredentials(), s.CleanupCredentials(), &fixtureStartup{}, func() (bool, string) { return true, "" }, nil, nil, nil, nil, nil, nil, settings.WithTextToSpeechCredential(s.SpeechCredentials()), settings.WithConfigurationLoad(s, nil, config.LoadReport{}))
			defer svc.ServiceShutdown()
			d := savedconnection.Details{CompatibilityProfile: compatibility.Generic, BaseURL: "https://fresh.example.test/v1", AuthenticationMode: config.AuthenticationModeNone, Headers: map[string]string{}}
			saved, err := svc.SaveSettings(settings.SaveSettingsRequest{ExpectedConnections: map[savedconnection.Purpose]string{}, ConnectionChange: &savedconnection.Change{Action: savedconnection.Create, Name: "First server", Uses: []savedconnection.Purpose{p}, Details: &d, ActivateFor: p}})
			if err != nil {
				t.Fatal(err)
			}
			if len(saved.SavedConnections.Selected) != 1 || saved.SavedConnections.Selected[p] == "" {
				t.Fatal("fresh task requires another selection")
			}
			next := saved.Settings
			switch p {
			case savedconnection.Transcription:
				next.Model = "fixture-model"
			case savedconnection.Cleanup:
				next.PostProcessing.Model = "fixture-model"
				next.PostProcessing.Enabled = true
			case savedconnection.Speech:
				next.TextToSpeech.Model = "fixture-model"
				next.TextToSpeech.Voice = "fixture-voice"
				next.TextToSpeech.Enabled = true
			}
			configured, err := svc.SaveSettings(settings.SaveSettingsRequest{Settings: next, ExpectedConnections: saved.SavedConnections.Selected})
			if err != nil {
				t.Fatal(err)
			}
			if configured.SetupCompleted {
				t.Fatal("companion setup completed dictation implicitly")
			}
			if p != savedconnection.Transcription && configured.BaseURL != "" {
				t.Fatal("companion setup invented STT endpoint")
			}
		})
	}
}

func TestRealtimeConnectionAndModelOptionsSurviveRestart(t *testing.T) {
	s, svc := connectionService(t)
	original := svc.GetSettings()
	d := savedconnection.Details{CompatibilityProfile: compatibility.NeMoSpeechV1, BaseURL: "https://live.example.test/v1", AuthenticationMode: config.AuthenticationModeAPIKey}
	out, err := svc.SaveSettings(settings.SaveSettingsRequest{ConnectionChange: &savedconnection.Change{Action: savedconnection.Create, Purpose: savedconnection.Voice, Uses: []savedconnection.Purpose{savedconnection.Voice}, Name: "Live", Details: &d}, ConnectionCredentialDraft: "live-canary"})
	if err != nil {
		t.Fatal(err)
	}
	id := connectionID(t, out, "Live")
	out = changeConnection(t, svc, savedconnection.Change{Action: savedconnection.Select, Purpose: savedconnection.Voice, ID: id})
	v := out.Settings
	v.VoiceTranscription.Realtime = true
	v.VoiceTranscription.CompatibilityProfile = "nemo-speech-v1"
	v.VoiceTranscription.ModelProfile = "nemotron-3.5-streaming"
	v.VoiceTranscription.Model = "nemotron-fixture"
	v.VoiceTranscription.Language = "fr-FR"
	v.VoiceTranscription.Options.Vocabulary = "Freehand\nNemotron"
	v.VoiceTranscription.Options.Boost = 2.5
	if _, err = svc.SaveSettings(settings.SaveSettingsRequest{Settings: v}); err != nil {
		t.Fatal(err)
	}
	captured, err := settings.DictationProfiles(svc).Capture()
	if err != nil {
		t.Fatal(err)
	}
	if captured.VoiceCredential != "live-canary" || captured.STTCredential != "live-canary" {
		t.Fatal("wrong live credential snapshot")
	}
	if svc.GetSettings().BaseURL != original.BaseURL || svc.GetSettings().Model != original.Model {
		t.Fatal("live selection changed completed STT")
	}
	path, legacy, vault := s.path, s.legacy, s.vault
	s.Close()
	reopened := newStore(path, legacy, vault)
	defer reopened.Close()
	got := loadStore(t, reopened)
	if !reflect.DeepEqual(got.VoiceTranscription, v.VoiceTranscription) {
		t.Fatal("realtime settings did not survive restart")
	}
	key, err := reopened.VoiceCredentials().Get()
	if err != nil || key != "live-canary" {
		t.Fatal("realtime credential reference did not survive restart")
	}
	found := false
	for _, e := range reopened.RememberedModels().Entries {
		if e.Purpose == savedconnection.Voice && e.Model == v.VoiceTranscription.Model {
			found = true
			if e.Options.Realtime != v.VoiceTranscription.Options {
				t.Fatal("vocabulary options lost")
			}
		}
	}
	if !found {
		t.Fatal("realtime model preferences were not persisted")
	}
}
