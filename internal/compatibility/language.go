package compatibility

import (
	"errors"

	"github.com/tnware/freehand-stt/internal/speechlanguage"
)

var errLanguageUnavailable = errors.New("language selection is unavailable for this profile")

// TranscriptionLanguage maps Freehand's selected language to a provider field.
// Empty preserves the server default. Automatic detection uses omission for
// OpenAI-shaped contracts and an explicit auto value for native whisper.cpp.
// Custom values remain byte-for-byte unchanged for existing compatible servers.
func (c Contract) TranscriptionLanguage(selected string) (string, error) {
	if err := speechlanguage.Validate(selected); err != nil {
		return "", err
	}
	if selected != "" && !c.Capabilities.LanguageHint {
		return "", errLanguageUnavailable
	}
	if selected == speechlanguage.Automatic && c.ID != WhisperCPP {
		return "", nil
	}
	return selected, nil
}
