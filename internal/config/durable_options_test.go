package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDurableSettingsExcludeRetiredVocabularyOptions(t *testing.T) {
	raw, err := json.Marshal(Default())
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	for _, data := range []json.RawMessage{wire["transcriptionOptions"], wire["voiceTranscription"]} {
		for _, key := range []string{`"hotwords"`, `"vocabularyBoost"`, `"vocabulary"`, `"boost"`} {
			if strings.Contains(string(data), key) {
				t.Errorf("durable settings expose retired %s: %s", key, data)
			}
		}
	}
}
