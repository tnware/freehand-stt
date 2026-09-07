package config

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func assertFieldError(t *testing.T, err error, field, message string) {
	t.Helper()
	if err == nil {
		t.Fatal("invalid settings accepted")
	}
	var classified *FieldError
	if !errors.As(err, &classified) {
		t.Fatalf("error %T is not a FieldError: %v", err, err)
	}
	if classified.Field != field || classified.Message != message || err.Error() != message {
		t.Fatalf("error = %#v, want field %q message %q", classified, field, message)
	}
	if errors.Unwrap(err) == nil {
		t.Fatal("original validation cause was lost")
	}
	data, marshalErr := json.Marshal(&err)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"kind": "settings_validation", "field": field, "message": message}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unsafe error JSON = %s", data)
	}
}

func TestValidateDirectFieldErrors(t *testing.T) {
	for _, tc := range []struct {
		field, message string
		mutate         func(*Settings)
	}{
		{"transcriptionTimeoutSeconds", "Enter a transcription timeout from 10 to 3600 seconds.", func(s *Settings) { s.TranscriptionTimeoutSeconds = 9 }},
		{"fileTranscriptionTimeoutSeconds", "Enter an audio file timeout from 60 to 86400 seconds.", func(s *Settings) { s.FileTranscriptionTimeoutSeconds = 86401 }},
		{"postProcessing.timeoutSeconds", "Enter a cleanup timeout from 10 to 3600 seconds.", func(s *Settings) { s.PostProcessing.TimeoutSeconds = 0 }},
		{"appearanceMode", "Choose system, light, or dark appearance mode.", func(s *Settings) { s.AppearanceMode = "rejected-value" }},
		{"vadMode", "Choose a supported voice activity detection mode.", func(s *Settings) { s.VADMode = "rejected-value" }},
		{"vadActivitySilenceMilliseconds", "Enter a voice activity silence delay from 100 to 1500 milliseconds.", func(s *Settings) { s.VADActivitySilenceMS = 99 }},
		{"speechPaddingMilliseconds", "Enter speech padding from 0 to 1000 milliseconds.", func(s *Settings) { s.SpeechPaddingMS = -1 }},
		{"autoStopSilenceMilliseconds", "Enter an automatic stop silence delay from 500 to 10000 milliseconds.", func(s *Settings) { s.AutoStopSilenceMS = 10001 }},
		{"autoStopMinimumSpeechMilliseconds", "Enter minimum speech before automatic stop from 100 to 5000 milliseconds.", func(s *Settings) { s.AutoStopMinimumSpeechMS = 99 }},
		{"autoStopSilenceMilliseconds", "Set automatic stop silence at least as long as the voice activity silence delay.", func(s *Settings) { s.AutoStopEnabled = true; s.AutoStopSilenceMS = 500; s.VADActivitySilenceMS = 600 }},
		{"vadEnabled", "Enable voice activity detection to use silence trimming, automatic stop, or silence splitting.", func(s *Settings) { s.VADEnabled = false; s.SilenceTrimming = true }},
		{"segmentSeconds", "Enter a segment target from 15 to 180 seconds.", func(s *Settings) { s.SegmentSeconds = 14 }},
		{"segmentSilenceMilliseconds", "Enter a segment silence delay from 200 to 3000 milliseconds.", func(s *Settings) { s.SegmentSilenceMS = 3001 }},
		{"microphoneID", "Choose a microphone with an identifier of at most 1024 bytes.", func(s *Settings) { s.MicrophoneID = strings.Repeat("x", 1025) }},
	} {
		t.Run(tc.field+"/"+tc.message, func(t *testing.T) {
			s := Default()
			tc.mutate(&s)
			assertFieldError(t, Validate(s), tc.field, tc.message)
		})
	}
}

func TestValidateNestedFieldErrors(t *testing.T) {
	for _, tc := range []struct {
		name, field, message string
		mutate               func(*Settings)
	}{
		{"transcription compatibility", "compatibilityProfile", "Choose a supported transcription server profile.", func(s *Settings) { s.CompatibilityProfile = "rejected-value" }},
		{"cleanup compatibility", "postProcessing.compatibilityProfile", "Choose a supported cleanup server profile.", func(s *Settings) { s.PostProcessing.CompatibilityProfile = "rejected-value" }},
		{"speech compatibility", "textToSpeech.compatibilityProfile", "Choose a supported speech server profile.", func(s *Settings) { s.TextToSpeech.CompatibilityProfile = "rejected-value" }},
		{"connection", "baseURL", "Check the transcription connection, authentication mode, model, health path, and custom headers.", func(s *Settings) { s.BaseURL = "https://user:rejected-secret@example.test/?token=rejected-value" }},
		{"headers", "baseURL", "Check the transcription connection, authentication mode, model, health path, and custom headers.", func(s *Settings) { s.Headers = map[string]string{"rejected-secret-token": "rejected-value"} }},
		{"transcription profile", "modelProfile", "Choose a compatible transcription model profile, language, and options.", func(s *Settings) { s.ModelProfile = "rejected-value" }},
		{"language", "language", "Choose a supported transcription language.", func(s *Settings) { s.Language = "rejected-value\n" }},
		{"overlay", "overlayEnabled", "Check the overlay layout, placement, appearance, and size.", func(s *Settings) { s.OverlaySizePercent = 0 }},
		{"shortcuts", "toggleShortcut", "Choose valid, distinct shortcuts for recording, showing Freehand, and hold-to-talk.", func(s *Settings) { s.ShowShortcut = s.ToggleShortcut }},
		{"cleanup profile", "postProcessing.preset", "Choose a compatible cleanup model profile and generation options.", func(s *Settings) { s.PostProcessing.Preset = "rejected-value" }},
		{"cleanup connection", "postProcessing.baseURL", "post-processing base URL must be an HTTP or HTTPS URL without credentials, query, or fragment", func(s *Settings) { s.PostProcessing.Enabled = true }},
		{"cleanup HTTP", "postProcessing.baseURL", "post-processing base URL uses insecure HTTP; enable Allow insecure HTTP to continue", func(s *Settings) {
			s.PostProcessing.Enabled = true
			s.PostProcessing.BaseURL = "http://example.test"
			s.PostProcessing.Model = "cleanup"
		}},
		{"cleanup styling", "postProcessing.styling", "Choose a valid S1-mini styling.", func(s *Settings) {
			s.PostProcessing.Enabled = true
			s.PostProcessing.BaseURL = "https://example.test"
			s.PostProcessing.Model = "cleanup"
			s.PostProcessing.Preset = PostProcessingPresetS1Mini
			s.PostProcessing.Styling = "rejected-value"
		}},
		{"speech profile", "textToSpeech.modelProfile", "Choose a compatible speech model profile and speaking speed.", func(s *Settings) { s.TextToSpeech.ModelProfile = "rejected-value" }},
		{"speech timeout", "textToSpeech.timeoutSeconds", "Enter a speech timeout from 10 to 3600 seconds.", func(s *Settings) { s.TextToSpeech.TimeoutSeconds = 9 }},
		{"speech voice", "textToSpeech.voice", "Choose a speech voice of at most 200 characters.", func(s *Settings) { s.TextToSpeech.Enabled = true }},
		{"speech speed dormant", "textToSpeech.speed", "Enter a speaking speed from 0.25 to 4.", func(s *Settings) { s.TextToSpeech.Speed = 4.1 }},
		{"speech connection", "textToSpeech.baseURL", "Check the speech connection, authentication mode, and model.", func(s *Settings) { s.TextToSpeech.Enabled = true; s.TextToSpeech.Voice = "alloy" }},
		{"speech speed configured", "textToSpeech.speed", "Enter a speaking speed from 0.25 to 4.", func(s *Settings) { s.TextToSpeech.BaseURL = "https://example.test"; s.TextToSpeech.Speed = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := Default()
			tc.mutate(&s)
			err := Validate(s)
			assertFieldError(t, err, tc.field, tc.message)
			data, _ := json.Marshal(&err)
			if strings.Contains(string(data), "rejected-") {
				t.Fatalf("rejected value leaked: %s", data)
			}
		})
	}
}

func TestValidateCleanupFieldErrors(t *testing.T) {
	for _, tc := range []struct {
		name, field, message string
		mutate               func(*PostProcessingSettings)
	}{
		{"model", "postProcessing.model", "Choose a cleanup model of at most 200 characters.", func(s *PostProcessingSettings) { s.Model = "" }},
		{"instruction required", "postProcessing.systemPrompt", "Enter a system instruction for the custom cleanup profile.", func(s *PostProcessingSettings) { s.SystemPrompt = "" }},
		{"instruction size", "postProcessing.systemPrompt", "Enter a cleanup system instruction of at most 8192 bytes.", func(s *PostProcessingSettings) { s.SystemPrompt = strings.Repeat("x", 8193) }},
		{"controls size", "postProcessing", "Keep each cleanup profile control to at most 32 bytes.", func(s *PostProcessingSettings) { s.Styling = strings.Repeat("x", 33) }},
		{"structure", "postProcessing.structure", "Choose a valid S1-mini structure.", func(s *PostProcessingSettings) { s.Preset = PostProcessingPresetS1Mini; s.Structure = "rejected-value" }},
		{"context", "postProcessing.context", "Choose a valid S1-mini context.", func(s *PostProcessingSettings) { s.Preset = PostProcessingPresetS1Mini; s.Context = "rejected-value" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := Default()
			s.PostProcessing.Enabled = true
			s.PostProcessing.BaseURL = "https://example.test"
			s.PostProcessing.Model = "cleanup"
			tc.mutate(&s.PostProcessing)
			assertFieldError(t, Validate(s), tc.field, tc.message)
		})
	}
}

func TestValidateRecordingLimitErrorJSON(t *testing.T) {
	for _, tc := range []struct {
		name      string
		splitting bool
		limit     int
		message   string
	}{
		{"single request", false, 263, "Enter a recording limit from 1 to 262 seconds."},
		{"split requests", true, 3601, "Enter a recording limit from 1 to 3600 seconds."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := Default()
			s.SilenceSplitting = tc.splitting
			s.MaxDurationSeconds = tc.limit
			err := Validate(s)
			if err == nil {
				t.Fatal("invalid recording limit accepted")
			}
			// Wails' defaultMarshalError marshals a pointer to the error interface.
			data, marshalErr := json.Marshal(&err)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			var got map[string]any
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			want := map[string]any{"kind": "settings_validation", "field": "maxDurationSeconds", "message": tc.message}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("error JSON = %s, want %v", data, want)
			}
			if err.Error() != tc.message {
				t.Fatalf("error message = %q", err.Error())
			}
		})
	}
}
