package settings

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/savedconnection"
)

type speechPreviewStore struct {
	storeFake
	catalog savedconnection.Catalog
}

func (s *speechPreviewStore) ConnectionCatalog() savedconnection.Catalog { return s.catalog }

func previewHarness() (*transactionHarness, TextToSpeechPreview) {
	store := &speechPreviewStore{storeFake: storeFake{log: &[]string{}}, catalog: savedconnection.Catalog{Selected: map[savedconnection.Purpose]string{savedconnection.Speech: "speech-server"}}}
	h := newTransactionHarness(store)
	h.service.cfg.TextToSpeech = config.Default().TextToSpeech
	h.service.cfg.TextToSpeech.Enabled = false
	h.service.cfg.TextToSpeech.BaseURL = "https://saved.example.test/v1"
	h.service.cfg.TextToSpeech.Model = "saved-model"
	h.service.cfg.TextToSpeech.Voice = "saved-voice"
	h.service.cfg.TextToSpeech.AuthenticationMode = config.AuthenticationModeAPIKey
	return h, TextToSpeechPreview{ConnectionID: "speech-server", Enabled: true, ModelProfile: modelprofile.Generic, Model: "draft-model", Voice: "draft-voice", Speed: 1.25, TimeoutSeconds: 45}
}

func TestSpeechPreviewCapturesDraftWithoutSaving(t *testing.T) {
	h, draft := previewHarness()
	before := h.service.current()
	profile, err := TextToSpeechProfiles(h.service).CapturePreview(&draft)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Settings.Model != draft.Model || profile.Settings.Voice != draft.Voice || profile.Settings.Speed != draft.Speed || profile.Settings.ModelProfile != draft.ModelProfile || profile.Settings.TimeoutSeconds != draft.TimeoutSeconds || !profile.Settings.Enabled {
		t.Fatal("preview did not use draft options")
	}
	if profile.Settings.BaseURL != before.TextToSpeech.BaseURL || profile.Credential != "old-tts-secret" {
		t.Fatal("preview did not capture the saved connection and credential")
	}
	if !reflect.DeepEqual(before, h.service.current()) {
		t.Fatal("preview changed active settings")
	}
	if !reflect.DeepEqual(*h.log, []string{"credential:get"}) {
		t.Fatalf("unexpected side effects: %v", *h.log)
	}
	draft.Voice = "later edit"
	h.service.cfg.TextToSpeech.BaseURL = "https://changed.example.test/v1"
	h.ttsKeys.value = "replacement-key"
	if profile.Settings.Voice != "draft-voice" || profile.Settings.BaseURL != "https://saved.example.test/v1" || profile.Credential != "old-tts-secret" {
		t.Fatal("captured preview changed after edits")
	}
	if _, err := TextToSpeechProfiles(h.service).Capture(); err == nil {
		t.Fatal("preview enabled ordinary playback without saving")
	}
}

func TestSpeechPreviewRejectsInvalidOrStaleDraftBeforeCredentialRead(t *testing.T) {
	cases := map[string]func(*TextToSpeechPreview){
		"changed connection": func(d *TextToSpeechPreview) { d.ConnectionID = "other" },
		"missing connection": func(d *TextToSpeechPreview) { d.ConnectionID = "" },
		"disabled":           func(d *TextToSpeechPreview) { d.Enabled = false },
		"empty model":        func(d *TextToSpeechPreview) { d.Model = " " },
		"oversized model":    func(d *TextToSpeechPreview) { d.Model = strings.Repeat("m", 10000) },
		"empty voice":        func(d *TextToSpeechPreview) { d.Voice = " " },
		"oversized voice":    func(d *TextToSpeechPreview) { d.Voice = strings.Repeat("v", 201) },
		"invalid profile":    func(d *TextToSpeechPreview) { d.ModelProfile = modelprofile.S1Mini },
		"speed":              func(d *TextToSpeechPreview) { d.Speed = 9 },
		"nonfinite speed":    func(d *TextToSpeechPreview) { d.Speed = math.NaN() },
		"timeout":            func(d *TextToSpeechPreview) { d.TimeoutSeconds = 0 },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			h, draft := previewHarness()
			change(&draft)
			if _, err := TextToSpeechProfiles(h.service).CapturePreview(&draft); err == nil {
				t.Fatal("invalid preview accepted")
			}
			if len(*h.log) != 0 {
				t.Fatalf("invalid preview read credentials or saved: %v", *h.log)
			}
		})
	}
}
