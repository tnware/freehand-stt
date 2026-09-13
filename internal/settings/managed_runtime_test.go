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
