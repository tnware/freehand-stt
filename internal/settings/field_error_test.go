package settings

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
)

func TestSaveSettingsFieldErrorPreservesAppliedSettings(t *testing.T) {
	for _, tc := range []struct {
		name, field string
		mutate      func(*config.Settings)
	}{
		{"single recording limit", "maxDurationSeconds", func(s *config.Settings) { s.MaxDurationSeconds = 263 }},
		{"split recording limit", "maxDurationSeconds", func(s *config.Settings) { s.SilenceSplitting = true; s.MaxDurationSeconds = 3601 }},
		{"transcription timeout", "transcriptionTimeoutSeconds", func(s *config.Settings) { s.TranscriptionTimeoutSeconds = 0 }},
		{"cleanup timeout", "postProcessing.timeoutSeconds", func(s *config.Settings) { s.PostProcessing.TimeoutSeconds = 3601 }},
		{"speech speed", "textToSpeech.speed", func(s *config.Settings) { s.TextToSpeech.Speed = 5 }},
		{"unsafe header", "baseURL", func(s *config.Settings) { s.Headers = map[string]string{"rejected-secret-token": "rejected-value"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service, log, startup, keys := transactionalService(false)
			applied := config.Default()
			applied.MaxDurationSeconds = 125
			applied.ToggleShortcut = "Ctrl+Alt+A"
			if _, err := service.SaveSettings(request(applied, "")); err != nil {
				t.Fatal(err)
			}
			before := service.GetSettings()
			*log = nil
			published := false
			service.settingsChanged = func(SettingsDTO) { published = true }
			next := applied
			next.StartWithWindows = true
			next.ToggleShortcut = "Ctrl+Alt+B"
			tc.mutate(&next)
			rejected := request(next, "rejected-secret")
			rejected.PostProcessingCredentialDraft = "rejected-processing-secret"
			_, err := service.SaveSettings(rejected)
			var fieldErr *config.FieldError
			if !errors.As(err, &fieldErr) || fieldErr.Field != tc.field {
				t.Fatalf("SaveSettings error = %T %v, want field %s", err, err, tc.field)
			}
			// Exercise the pinned Wails error-marshalling convention, not a DTO.
			data, marshalErr := json.Marshal(&err)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			var wire map[string]any
			if err := json.Unmarshal(data, &wire); err != nil {
				t.Fatal(err)
			}
			want := map[string]any{"kind": "settings_validation", "field": tc.field, "message": fieldErr.Message}
			if !reflect.DeepEqual(wire, want) || strings.Contains(string(data), "rejected-") {
				t.Fatalf("unsafe validation cause JSON = %s", data)
			}
			if !reflect.DeepEqual(service.GetSettings(), before) || !reflect.DeepEqual(service.current(), applied) {
				t.Fatal("rejection changed applied settings")
			}
			if len(*log) != 0 || startup.on || keys.value != "old-secret" || published {
				t.Fatalf("rejection reached dependencies or publication: %v", *log)
			}
		})
	}
}
