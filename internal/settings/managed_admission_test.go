package settings

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"reflect"
	"strings"
	"testing"
)

func readyManagedEndpoint(p managedruntime.Preferences) (managedruntime.Endpoint, error) {
	return managedruntime.Endpoint{Enabled: true, BaseURL: "http://127.0.0.1:43210/v1", Model: "served-model", Realtime: p.Realtime, Profile: string(modelprofile.Nemotron35)}, nil
}

func TestManagedCurrentSourceUsesEffectiveAdmissionNotSavedBYO(t *testing.T) {
	s, _, _, _ := transactionalService(false)
	s.cfg.ManagedRuntime.Enabled = true
	// An unrelated manual draft must not prevent local Voice admission.
	s.cfg.VoiceTranscription.BaseURL = "invalid manual url"
	s.cfg.VoiceTranscription.TranscriptionOptions.Prompt = "manual model prompt"
	WithManagedRuntime(readyManagedEndpoint, nil)(s)
	effective := CurrentSource(s).Current()
	if err := config.ValidateVoiceRecording(effective.VoiceTranscription); err != nil {
		t.Fatal(err)
	}
	if !effective.VoiceTranscription.Realtime {
		t.Fatal("recommended mode not used for admission")
	}
	if s.GetSettings().VoiceTranscription.BaseURL != "invalid manual url" {
		t.Fatal("saved BYO draft overwritten")
	}
}

func TestManagedCaptureFailsClosedBeforeCredentialRead(t *testing.T) {
	for _, mode := range []string{"missing-resolver", "not-ready", "disabled-endpoint", "invalid-language", "oversized-vocabulary"} {
		t.Run(mode, func(t *testing.T) {
			s, log, _, _ := transactionalService(false)
			s.cfg.ManagedRuntime.Enabled = true
			switch mode {
			case "missing-resolver":
			case "not-ready":
				WithManagedRuntime(func(managedruntime.Preferences) (managedruntime.Endpoint, error) {
					return managedruntime.Endpoint{}, errors.New("private runtime error")
				}, nil)(s)
			case "disabled-endpoint":
				WithManagedRuntime(func(managedruntime.Preferences) (managedruntime.Endpoint, error) {
					return managedruntime.Endpoint{}, nil
				}, nil)(s)
			default:
				WithManagedRuntime(readyManagedEndpoint, nil)(s)
			}
			if mode == "invalid-language" {
				s.cfg.Language = "af"
				s.cfg.VoiceTranscription.Language = "af"
			}
			if mode == "oversized-vocabulary" {
				s.cfg.Vocabulary = config.VocabularySettings{Terms: strings.Repeat("x", 129), Voice: true, Files: true, Boost: 3}
			}
			for _, source := range []ProfileSource{DictationProfiles(s), RequestProfiles(s)} {
				p, err := source.Capture()
				if err == nil || !reflect.DeepEqual(p, RequestProfile{}) {
					t.Fatalf("managed failure returned a usable remote snapshot: %#v %v", p, err)
				}
			}
			if len(*log) != 0 {
				t.Fatalf("managed failure read credentials: %v", *log)
			}
		})
	}
}

func TestManagedDisabledRestoresUntouchedManualSnapshot(t *testing.T) {
	s, _, _, keys := transactionalService(false)
	s.cfg.BaseURL = "https://file.example.test/v1"
	s.cfg.Model = "manual"
	s.cfg.AuthenticationMode = config.AuthenticationModeAPIKey
	s.cfg.VoiceTranscription.BaseURL = "https://voice.example.test/v1"
	s.cfg.VoiceTranscription.Model = "manual-voice"
	s.cfg.VoiceTranscription.AuthenticationMode = config.AuthenticationModeAPIKey
	s.voiceKeys = keys
	before, err := DictationProfiles(s).Capture()
	if err != nil {
		t.Fatal(err)
	}
	WithManagedRuntime(func(managedruntime.Preferences) (managedruntime.Endpoint, error) {
		t.Fatal("disabled preference resolved local runtime")
		return managedruntime.Endpoint{}, nil
	}, nil)(s)
	after, err := DictationProfiles(s).Capture()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("disabled mode changed manual request")
	}
}
