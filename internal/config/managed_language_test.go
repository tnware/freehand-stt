package config

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

func TestManagedLanguageUsesExplicitQualifiedModel(t *testing.T) {
	for _, model := range []string{"nemotron-3.5", "parakeet-tdt"} {
		for _, task := range []string{"voice", "file"} {
			t.Run(model+"/"+task, func(t *testing.T) {
				v := Default() // Generic manual profiles accept both language hints.
				language := "en-US"
				if model == "nemotron-3.5" {
					language = "en" // Nemotron requires its qualified locale, not generic English.
				}
				if task == "voice" {
					v.VoiceTranscription.Language = language
				} else {
					v.Language = language
				}
				if err := Validate(v); err != nil {
					t.Fatalf("generic manual language rejected: %v", err)
				}
				v.ManagedRuntime.Enabled = true
				v.ManagedRuntime.Model = model
				v.ManagedRuntime.Realtime = false
				if err := Validate(v); err == nil {
					t.Fatal("unsupported managed task language accepted without any runtime readiness")
				}
			})
		}
	}
}

func TestManagedLanguageValidationPreservesManualChecks(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, task := range []string{"voice", "file"} {
			for _, invalid := range []string{"connection", "model", "profile", "options"} {
				t.Run(fmt.Sprintf("managed=%t/%s/%s", enabled, task, invalid), func(t *testing.T) {
					v := Default()
					v.ManagedRuntime.Enabled = enabled
					manual := DefaultVoiceTranscription()
					manual.BaseURL, manual.Model = "https://manual.example/v1", "manual"
					manual.TranscriptionOptions.Prompt = "Remembered manual context"
					if task == "voice" {
						v.VoiceTranscription = manual
					} else {
						v.VoiceTranscription = manual
						v = WithVoiceTranscription(v)
						v.VoiceTranscription = DefaultVoiceTranscription()
					}
					before := v
					if err := Validate(v); err != nil {
						t.Fatalf("saved manual options should not be checked against managed NeMo: %v", err)
					}
					if !reflect.DeepEqual(v, before) {
						t.Fatal("validation mutated saved manual options")
					}
					switch invalid {
					case "connection":
						manual.BaseURL = "not a URL"
					case "model":
						manual.Model = strings.Repeat("x", 201)
					case "profile":
						manual.ModelProfile = "unqualified"
					case "options":
						manual.TranscriptionOptions.Prompt = strings.Repeat("x", MaxPromptBytes+1)
					}
					v.VoiceTranscription = manual
					if task == "file" {
						v = WithVoiceTranscription(v)
						v.VoiceTranscription = DefaultVoiceTranscription()
					}
					before = v
					if err := Validate(v); err == nil {
						t.Fatal("invalid saved manual settings accepted")
					}
					if !reflect.DeepEqual(v, before) {
						t.Fatal("failed validation mutated saved manual settings")
					}
				})
			}
		}
	}
}

func TestManagedDisabledRetainsManualLanguageContract(t *testing.T) {
	for _, task := range []string{"voice", "file"} {
		t.Run(task, func(t *testing.T) {
			v := Default()
			if task == "voice" {
				v.VoiceTranscription.ModelProfile = modelprofile.VoxtralRealtime
				v.VoiceTranscription.CompatibilityProfile = compatibility.VLLM
				v.VoiceTranscription.Language = "en-US"
			} else {
				v.ModelProfile, v.CompatibilityProfile = modelprofile.VoxtralRealtime, compatibility.VLLM
				v.Language = "en-US"
			}
			if err := Validate(v); err == nil {
				t.Fatal("disabled managed runtime bypassed manual automatic-only profile")
			}
		})
	}
}

func TestManagedLanguageIgnoresInactiveManualLanguageContract(t *testing.T) {
	v := Default()
	v.ManagedRuntime.Enabled = true
	v.ModelProfile, v.CompatibilityProfile = modelprofile.VoxtralRealtime, compatibility.VLLM
	v.BaseURL, v.Model = "https://manual.example/v1", "manual-file"
	v.Headers = map[string]string{"X-Manual": "file"}
	v.Language = "en-US"
	v.VoiceTranscription.ModelProfile = modelprofile.VoxtralRealtime
	v.VoiceTranscription.CompatibilityProfile = compatibility.VLLM
	v.VoiceTranscription.BaseURL, v.VoiceTranscription.Model = "https://voice.example/v1", "manual-voice"
	v.VoiceTranscription.Headers = map[string]string{"X-Manual": "voice"}
	v.VoiceTranscription.Realtime = true
	v.VoiceTranscription.Language = "en-US"
	before := v
	before.Headers = map[string]string{"X-Manual": "file"}
	before.VoiceTranscription.Headers = map[string]string{"X-Manual": "voice"}
	if err := Validate(v); err != nil {
		t.Fatalf("managed Nemotron should accept en-US despite inactive automatic-only Voxtral: %v", err)
	}
	if !reflect.DeepEqual(v, before) {
		t.Fatal("validation changed task intent or saved manual settings")
	}
}
