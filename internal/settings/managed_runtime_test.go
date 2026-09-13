package settings

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"reflect"
	"testing"
)

func TestManagedCaptureOverridesWholeTransportWithoutBYOCredentials(t *testing.T) {
	s, log, _, keys := transactionalService(false)
	s.cfg.ManagedRuntime.Enabled = true
	s.cfg.BaseURL = "https://files.example.test/v1"
	s.cfg.Model = "remote-file"
	s.cfg.AuthenticationMode = config.AuthenticationModeAPIKey
	s.cfg.Headers = map[string]string{"X-Private": "file-secret"}
	s.cfg.HealthPath = "/private-health"
	s.cfg.Language = "fr-FR"
	s.cfg.VoiceTranscription.BaseURL = "https://voice.example.test/v1"
	s.cfg.VoiceTranscription.Model = "remote-voice"
	s.cfg.VoiceTranscription.AuthenticationMode = config.AuthenticationModeAPIKey
	s.cfg.VoiceTranscription.Headers = map[string]string{"X-Private": "voice-secret"}
	s.cfg.VoiceTranscription.HealthPath = "/private-health"
	s.cfg.VoiceTranscription.Language = "en-US"
	s.voiceKeys = keys
	before := s.current()
	endpoint := managedruntime.Endpoint{Enabled: true, BaseURL: "http://127.0.0.1:43210/v1", Model: "server-nemotron-id", Realtime: true, Profile: string(modelprofile.Nemotron35)}
	WithManagedRuntime(func(managedruntime.Preferences) (managedruntime.Endpoint, error) { return endpoint, nil }, nil)(s)
	voice, err := DictationProfiles(s).Capture()
	if err != nil {
		t.Fatal(err)
	}
	files, err := RequestProfiles(s).Capture()
	if err != nil {
		t.Fatal(err)
	}
	for name, p := range map[string]RequestProfile{"voice": voice, "files": files} {
		if p.Settings.BaseURL != endpoint.BaseURL || p.Settings.Model != endpoint.Model || p.Settings.ModelProfile != modelprofile.Nemotron35 || p.Settings.CompatibilityProfile != compatibility.NeMoSpeechV1 {
			t.Fatalf("%s wrong managed transport: %#v", name, p.Settings)
		}
		if p.STTCredential != "" || p.VoiceCredential != "" || p.Settings.AuthenticationMode != config.AuthenticationModeNone || len(p.Settings.Headers) != 0 || len(p.Settings.VoiceTranscription.Headers) != 0 || p.Settings.HealthPath != "" {
			t.Fatalf("%s leaked BYO transport", name)
		}
	}
	if !voice.Settings.VoiceTranscription.Realtime || files.Settings.VoiceTranscription.Realtime {
		t.Fatal("managed realtime must apply only to voice snapshots")
	}
	if voice.Settings.Language != "en-US" || files.Settings.Language != "fr-FR" {
		t.Fatal("source spoken language lost")
	}
	if len(*log) != 0 {
		t.Fatalf("managed capture read BYO credentials: %v", *log)
	}
	if !reflect.DeepEqual(s.GetSettings().Settings, before) {
		t.Fatal("capture changed saved BYO edit settings")
	}
}
