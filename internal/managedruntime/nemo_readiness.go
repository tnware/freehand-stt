package managedruntime

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

type nemoLoadedModel struct{ ID, Capability string }

func nemoModelIdentities(models []nemoLoadedModel, speech bool) (string, string, error) {
	want := 1
	if speech {
		want++
	}
	invalid := errors.New("Managed speech returned an unexpected model identity.")
	if len(models) != want {
		return "", "", invalid
	}
	var asr, tts string
	for _, model := range models {
		if strings.TrimSpace(model.ID) == "" || len(model.ID) > 512 || !utf8.ValidString(model.ID) || strings.IndexFunc(model.ID, unicode.IsControl) >= 0 {
			return "", "", invalid
		}
		switch model.Capability {
		case "transcription":
			if asr != "" {
				return "", "", invalid
			}
			asr = model.ID
		case "speech":
			if !speech || tts != "" {
				return "", "", invalid
			}
			tts = model.ID
		default:
			return "", "", invalid
		}
	}
	if asr == "" || (speech && tts == "") || (tts != "" && asr == tts) {
		return "", "", invalid
	}
	return asr, tts, nil
}
