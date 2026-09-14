package storage

import (
	"reflect"
	"testing"

	"github.com/tnware/freehand-stt/internal/modelsettings"
	"github.com/tnware/freehand-stt/internal/savedconnection"
)

func TestRememberedModelCatalogIgnoresTaskIntentAcrossRestart(t *testing.T) {
	s := testStore(t)
	v := loadStore(t, s)
	d := savedconnection.Extract(v, savedconnection.Transcription)
	d.BaseURL = "https://model.example.test/v1"
	v = createSelectedConnection(t, s, v, "Models", savedconnection.Transcription, d)
	v.Model = "first-model"
	v.Language = "ja"
	v.TranscriptionOptions.Prompt = "Names and punctuation"
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	before := s.RememberedModels()
	s = reopen(t, s)
	got := loadStore(t, s)
	if !reflect.DeepEqual(got, v) || !reflect.DeepEqual(s.RememberedModels(), before) {
		t.Fatal("model/task state differs after restart")
	}
	got.Language = "de"
	if len(before.Entries) != 1 {
		t.Fatal("missing remembered model")
	}
	next := modelsettings.Select(got, savedconnection.Transcription, before.Entries[0].Model, before.Entries[0].Options)
	if next.Language != "de" || next.TranscriptionOptions.Prompt != v.TranscriptionOptions.Prompt {
		t.Fatal("model restore replaced task intent or lost model options")
	}
}

func TestRememberedCleanupAndSpeechKeepTaskIntent(t *testing.T) {
	for _, purpose := range []savedconnection.Purpose{savedconnection.Cleanup, savedconnection.Speech} {
		t.Run(string(purpose), func(t *testing.T) {
			s := testStore(t)
			v := loadStore(t, s)
			d := savedconnection.Extract(v, purpose)
			d.BaseURL = "https://model.example.test/v1"
			v = createSelectedConnection(t, s, v, "Models", purpose, d)
			if purpose == savedconnection.Cleanup {
				v.PostProcessing.Model = "cleanup-model"
				v.PostProcessing.SystemPrompt = "Task instruction"
				v.PostProcessing.Styling = "formal"
			} else {
				v.TextToSpeech.Model = "speech-model"
				v.TextToSpeech.Voice = "voice"
				v.TextToSpeech.Speed = 1.5
			}
			if err := s.Save(v); err != nil {
				t.Fatal(err)
			}
			before := s.RememberedModels()
			s = reopen(t, s)
			got := loadStore(t, s)
			if !reflect.DeepEqual(got, v) || !reflect.DeepEqual(s.RememberedModels(), before) {
				t.Fatal("model/task state differs after restart")
			}
			if len(before.Entries) != 1 {
				t.Fatal("missing model options")
			}
			got.PostProcessing.SystemPrompt = "New task instruction"
			got.PostProcessing.Styling = "casual"
			got.TextToSpeech.Speed = 2
			next := modelsettings.Select(got, purpose, before.Entries[0].Model, before.Entries[0].Options)
			if next.PostProcessing.SystemPrompt != got.PostProcessing.SystemPrompt || next.PostProcessing.Styling != got.PostProcessing.Styling || next.TextToSpeech.Speed != got.TextToSpeech.Speed {
				t.Fatal("restored model overwrote task intent")
			}
		})
	}
}
