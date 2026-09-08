package settings

import (
	"reflect"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

func TestQwenSpeechPreviewCapturesUnsavedLanguageAndStyle(t *testing.T) {
	h, draft := previewHarness()
	h.service.cfg.TextToSpeech.CompatibilityProfile = compatibility.VLLMOmni
	before := h.service.current()
	draft.ModelProfile = modelprofile.Qwen3TTS
	draft.Voice = "ryan"
	draft.Options = modelprofile.SpeechOptions{Language: "ja", Instructions: "Warm delivery"}
	want := draft.Options
	profile, err := TextToSpeechProfiles(h.service).CapturePreview(&draft)
	if err != nil {
		t.Fatal(err)
	}
	draft.Options.Instructions = "Later edit"
	if profile.Settings.Options != want || !reflect.DeepEqual(before, h.service.current()) {
		t.Fatal("preview did not capture independent unsaved options")
	}
	if !reflect.DeepEqual(*h.log, []string{"credential:get"}) {
		t.Fatal("preview wrote settings")
	}
	*h.log = nil
	draft.Options.Language = "unsupported"
	if _, err := TextToSpeechProfiles(h.service).CapturePreview(&draft); err == nil || len(*h.log) != 0 {
		t.Fatal("invalid speech options reached credentials")
	}
}
