package settings

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"reflect"
	"testing"
)

func testInstance() managedruntime.Instance {
	return managedruntime.Instance{ID: "speech", Name: "Local speech", Provider: managedruntime.NeMoSpeechCPP, Model: "nemotron-3.5"}
}
func readyManagedEndpoint(i managedruntime.Instance, role compatibility.Role) (managedruntime.ResolvedEndpoint, error) {
	c, err := managedruntime.Qualify(i.Provider, i.Model, role)
	return managedruntime.ResolvedEndpoint{InstanceID: i.ID, Provider: i.Provider, CatalogModel: i.Model, Generation: 1, BaseURL: "http://127.0.0.1:43210/v1", Model: "served-model", Contract: c}, err
}
func TestManagedFilesDoNotOverrideManualVoice(t *testing.T) {
	s, _, _, keys := transactionalService(false)
	i := testInstance()
	s.cfg.ManagedRuntimes = []managedruntime.Instance{i}
	s.cfg.ManagedInstanceID = i.ID
	s.cfg.VoiceTranscription.BaseURL = "https://voice.example.test/v1"
	s.cfg.VoiceTranscription.Model = "manual-voice"
	s.cfg.VoiceTranscription.AuthenticationMode = config.AuthenticationModeAPIKey
	s.voiceKeys = keys
	WithManagedRuntimes(func(managedruntime.Instance, compatibility.Role) (managedruntime.ResolvedEndpoint, error) {
		return managedruntime.ResolvedEndpoint{}, errors.New("stopped")
	}, nil)(s)
	p, err := DictationProfiles(s).Capture()
	if err != nil {
		t.Fatal(err)
	}
	if p.Settings.BaseURL != s.cfg.VoiceTranscription.BaseURL || p.STTCredential != "old-secret" {
		t.Fatal("manual Voice was replaced")
	}
	if _, err := RequestProfiles(s).Capture(); !errors.Is(err, ErrManagedUnavailable) {
		t.Fatalf("stopped files admitted: %v", err)
	}
	effective := CurrentSource(s).Current()
	if effective.BaseURL != "" || !reflect.DeepEqual(effective.VoiceTranscription, s.cfg.VoiceTranscription) {
		t.Fatal("failure damaged unrelated Voice")
	}
}

func TestManagedVoiceAdmitsResolvedTransportWithoutPersistingIt(t *testing.T) {
	for _, realtime := range []bool{false, true} {
		s, log, _, keys := transactionalService(false)
		i := testInstance()
		endpoint, err := readyManagedEndpoint(i, compatibility.Transcription)
		if err != nil {
			t.Fatal(err)
		}
		s.cfg.ManagedRuntimes = []managedruntime.Instance{i}
		voice := config.DefaultVoiceTranscription()
		voice.ManagedInstanceID, voice.Model = i.ID, i.Model
		voice.ModelProfile = endpoint.Contract.ModelProfile
		voice.CompatibilityProfile = endpoint.Contract.CompatibilityProfile
		voice.Realtime = realtime
		s.cfg.VoiceTranscription = voice
		s.voiceKeys = keys
		keys.getErr = errors.New("locked")
		WithManagedRuntimes(readyManagedEndpoint, nil)(s)
		if err := config.ValidateVoiceRecording(CurrentSource(s).Current().VoiceTranscription); err != nil {
			t.Fatalf("managed recording admission (realtime=%v): %v", realtime, err)
		}
		profile, err := DictationProfiles(s).Capture()
		if err != nil {
			t.Fatalf("managed request capture (realtime=%v): %v", realtime, err)
		}
		if profile.Settings.BaseURL != endpoint.BaseURL || profile.Settings.Model != endpoint.Model || profile.STTCredential != "" || len(*log) != 0 {
			t.Fatal("managed capture did not retain its credential-free resolved endpoint")
		}
		if !reflect.DeepEqual(s.current().VoiceTranscription, voice) {
			t.Fatal("request resolution changed saved Voice settings")
		}
		if err := config.ValidateVoiceTranscription(profile.Settings.VoiceTranscription); err == nil {
			t.Fatal("resolved transport was accepted as persistable settings")
		}
	}
}
