package modelsettings

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/savedconnection"
)

func TestRememberedOptionsWireContainsOnlyModelPreferences(t *testing.T) {
	data, err := json.Marshal(Options{})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"profile": "", "voice": "",
		"speech":        map[string]any{"language": "", "instructions": ""},
		"transcription": map[string]any{"nemo": map[string]any{"disablePunctuation": false, "normalize": false, "profanityFilter": false, "endpointingMilliseconds": float64(0)}, "prompt": "", "temperatureOverride": false, "temperature": float64(0)},
		"cleanup":       map[string]any{"limitOutputTokens": false, "maxOutputTokens": float64(0), "disableReasoning": false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("remembered wire contains task-owned fields: %s", data)
	}
}

func TestOptionsNeverReplaceConnectionOrWorkflow(t *testing.T) {
	v := config.Default()
	v.BaseURL = "https://example.test/v1"
	v.PostProcessing.Enabled = true
	v.TextToSpeech.Enabled = true
	v.MicrophoneID = "chosen-device"
	v.TranscriptionTimeoutSeconds = 240
	for p, o := range Defaults() {
		got := Apply(v, p, "model", o)
		if got.BaseURL != v.BaseURL || !got.PostProcessing.Enabled || !got.TextToSpeech.Enabled || got.MicrophoneID != v.MicrophoneID || got.TranscriptionTimeoutSeconds != 240 {
			t.Fatal("model options changed connection or workflow")
		}
		if Extract(got, p) != o {
			t.Fatal("options failed round trip")
		}
	}
}
func TestModelOptionsValidateWhileFeatureDisabled(t *testing.T) {
	d := savedconnection.Details{BaseURL: "https://example.test/v1", AuthenticationMode: config.AuthenticationModeNone}
	e := Entry{Model: "cleanup", Purpose: savedconnection.Cleanup, Options: Defaults()[savedconnection.Cleanup]}
	e.Options.Profile = modelprofile.S1Mini
	e.Options.Cleanup.LimitOutputTokens = true
	e.Options.Cleanup.MaxOutputTokens = -1
	if Validate(e, d) == nil {
		t.Fatal("invalid S1 controls accepted")
	}
	e.Options = Defaults()[savedconnection.Cleanup]
	e.Options.Voice = "cross-role"
	if Validate(e, d) == nil {
		t.Fatal("cross-role options accepted")
	}
	e.Options = Defaults()[savedconnection.Cleanup]
	e.Model = "bad\nmodel"
	if Validate(e, d) == nil {
		t.Fatal("control characters accepted")
	}
}

func TestSelectionGatesRealtimeByBackendAndModelProfile(t *testing.T) {
	for _, tt := range []struct {
		name    string
		backend compatibility.ID
		profile modelprofile.ID
		want    bool
	}{
		{"qualified", compatibility.NeMoSpeechV1, modelprofile.Nemotron35, true},
		{"wrong backend", compatibility.Generic, modelprofile.Nemotron35, false},
		{"completed model", compatibility.NeMoSpeechV1, modelprofile.Generic, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			v := config.Default()
			v.VoiceTranscription.Realtime = true
			v.VoiceTranscription.CompatibilityProfile = tt.backend
			o := Defaults()[savedconnection.Voice]
			o.Profile = tt.profile
			next := Select(v, savedconnection.Voice, "model", o)
			if next.VoiceTranscription.Realtime != tt.want {
				t.Fatalf("realtime = %v, want %v", next.VoiceTranscription.Realtime, tt.want)
			}
			v.VoiceTranscription.Realtime = false
			if Select(v, savedconnection.Voice, "model", o).VoiceTranscription.Realtime {
				t.Fatal("selection enabled realtime")
			}
		})
	}
}

func TestSelectionPreservesTaskIntent(t *testing.T) {
	v := config.Default()
	v.Language = "ja"
	v.TranscriptionOptions.Prompt = "Proper names"
	v.VoiceTranscription.Language = "en"
	v.Vocabulary = config.VocabularySettings{Terms: "Task terms", Voice: true, Files: true, Boost: 4}
	v.PostProcessing.Structure = "list"
	v.PostProcessing.Context = "Task context"
	v.PostProcessing.SystemPrompt = "Keep punctuation."
	v.PostProcessing.Styling = "formal"
	v.TextToSpeech.Speed = 1.5
	for p, o := range Defaults() {
		o.Voice = "model-voice"
		next := Select(v, p, "new-model", o)
		if next.Language != v.Language || next.PostProcessing.SystemPrompt != v.PostProcessing.SystemPrompt || next.PostProcessing.Styling != v.PostProcessing.Styling || next.PostProcessing.Structure != v.PostProcessing.Structure || next.PostProcessing.Context != v.PostProcessing.Context || next.TextToSpeech.Speed != v.TextToSpeech.Speed || next.VoiceTranscription.Language != v.VoiceTranscription.Language || next.Vocabulary != v.Vocabulary {
			t.Fatal("model selection replaced task intent")
		}
		if Model(next, p) != "new-model" {
			t.Fatal("model was not selected")
		}
		if p == savedconnection.Speech && next.TextToSpeech.Voice != "model-voice" {
			t.Fatal("model voice was not restored")
		}
	}
}
