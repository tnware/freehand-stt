package settings

import (
	"errors"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/savedconnection"
)

func TestCombinedNeMoCapturesRoleSpecificEndpoint(t *testing.T) {
	s, _, _, keys := transactionalService(false)
	i := testInstance()
	i.SpeechModel = "magpie-tts"
	s.cfg.ManagedRuntimes = []managedruntime.Instance{i}
	for _, purpose := range []savedconnection.Purpose{savedconnection.Voice, savedconnection.Speech} {
		s.cfg = savedconnection.Apply(s.cfg, purpose, savedconnection.Details{ManagedInstanceID: i.ID})
	}
	s.cfg.TextToSpeech.Enabled = true
	s.cfg.TextToSpeech.Voice = "default"
	s.cfg.TextToSpeech.Options.Language = "fr-FR"
	s.ttsKeys = keys
	keys.getErr = errors.New("manual credential unavailable")
	WithManagedRuntimes(func(instance managedruntime.Instance, role compatibility.Role) (managedruntime.ResolvedEndpoint, error) {
		model := instance.ModelForRole(role)
		contract, err := managedruntime.Qualify(instance.Provider, model, role)
		return managedruntime.ResolvedEndpoint{InstanceID: instance.ID, Provider: instance.Provider, CatalogModel: model, Generation: 3, BaseURL: "http://127.0.0.1:43210/v1", Model: "loaded-" + model, Contract: contract}, err
	}, nil)(s)
	speech, err := TextToSpeechProfiles(s).Capture()
	if err != nil {
		t.Fatal(err)
	}
	voice, err := DictationProfiles(s).Capture()
	if err != nil {
		t.Fatal(err)
	}
	if speech.Settings.Model != "loaded-magpie-tts" || voice.Settings.Model != "loaded-nemotron-3.5" || speech.Settings.BaseURL != voice.Settings.BaseURL || speech.Credential != "" || voice.STTCredential != "" {
		t.Fatal("combined runtime mixed task identities or manual credentials")
	}
	if s.current().TextToSpeech.BaseURL != "" || s.current().TextToSpeech.Model != "magpie-tts" {
		t.Fatal("request capture persisted the live endpoint")
	}
	WithManagedRuntimes(readyManagedEndpoint, nil)(s)
	if _, err := TextToSpeechProfiles(s).Capture(); !errors.Is(err, ErrManagedUnavailable) {
		t.Fatal("speech accepted the ASR catalog identity")
	}
}
