package storage

import (
	"errors"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/savedconnection"
)

func TestCombinedNeMoSelectionRequiresExplicitNormalSpeed(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	v.ManagedRuntimes = []managedruntime.Instance{{ID: "combined", Name: "NeMo", Provider: managedruntime.NeMoSpeechCPP, Model: "nemotron-3.5", SpeechModel: "magpie-tts"}}
	v.TextToSpeech.Speed = 1.5
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	id := savedconnection.BuiltInID("combined")
	next, err := s.BeginConnectionChange(savedconnection.Change{Action: savedconnection.Select, Purpose: savedconnection.Speech, ID: id}, v)
	if err != nil {
		t.Fatal(err)
	}
	if next.TextToSpeech.Speed != 1.5 {
		t.Fatal("selection silently replaced the task's speaking speed")
	}
	err = s.Save(next)
	field, ok := errors.AsType[*config.FieldError](err)
	if !ok || field.Field != "textToSpeech.speed" {
		t.Fatalf("expected actionable speed guidance, got %v", err)
	}
	s = reopen(t, s)
	v = loadStore(t, s)
	if s.ConnectionCatalog().Selected[savedconnection.Speech] != "" || v.TextToSpeech.Speed != 1.5 {
		t.Fatal("rejected selection changed saved speech settings")
	}
	v.TextToSpeech.Speed = 1
	v = selectConnection(t, s, v, savedconnection.Speech, id)
	if v.TextToSpeech.Model != "magpie-tts" || v.TextToSpeech.Speed != 1 {
		t.Fatal("explicit normal speed did not admit the managed speech model")
	}
}

func TestCombinedNeMoSelectionsAndSpeechPreferencesRoundTrip(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	v.ManagedRuntimes = []managedruntime.Instance{{ID: "combined", Name: "NeMo", Provider: managedruntime.NeMoSpeechCPP, Model: "nemotron-3.5", SpeechModel: "magpie-tts"}}
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	id := savedconnection.BuiltInID("combined")
	v = selectConnection(t, s, v, savedconnection.Voice, id)
	v = selectConnection(t, s, v, savedconnection.Speech, id)
	if v.TextToSpeech.Voice != "default" {
		t.Fatal("managed Magpie must begin with its server default voice")
	}
	v.TextToSpeech.Enabled = true
	v.TextToSpeech.Options.Language = "fr-FR"
	v.TextToSpeech.Voice = "John"
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	v = loadStore(t, s)
	if v.VoiceTranscription.Model != "nemotron-3.5" || v.TextToSpeech.Model != "magpie-tts" || v.TextToSpeech.ModelProfile != modelprofile.MagpieTTS || v.TextToSpeech.BaseURL != "" || v.TextToSpeech.ManagedInstanceID != "combined" {
		t.Fatal("shared process lost its independent task model projection")
	}
	if !v.TextToSpeech.Enabled || v.TextToSpeech.Options.Language != "fr-FR" || v.TextToSpeech.Voice != "John" {
		t.Fatal("speech options did not survive reload")
	}
	// Deselecting and reselecting preserves remembered speech options.
	v = selectConnection(t, s, v, savedconnection.Speech, "")
	v = selectConnection(t, s, v, savedconnection.Speech, id)
	if v.TextToSpeech.Options.Language != "fr-FR" || v.TextToSpeech.Voice != "John" {
		t.Fatal("reselecting the companion lost remembered Magpie options")
	}
	// Explicitly disabling removes the speech capability without changing Voice.
	v = selectConnection(t, s, v, savedconnection.Speech, "")
	v.ManagedRuntimes[0].SpeechModel = ""
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	s = reopen(t, s)
	v = loadStore(t, s)
	if s.ConnectionCatalog().Selected[savedconnection.Voice] != id || v.TextToSpeech.Enabled {
		t.Fatal("disabling speech changed Voice or kept speech enabled")
	}
	v.ManagedRuntimes[0].SpeechModel = "magpie-tts"
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	v = selectConnection(t, s, v, savedconnection.Speech, id)
	if v.TextToSpeech.Model != "magpie-tts" || v.TextToSpeech.Voice != "default" {
		t.Fatal("re-enabled speech capability did not restore valid defaults")
	}
}
