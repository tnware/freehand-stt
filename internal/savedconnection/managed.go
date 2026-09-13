package savedconnection

import (
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
)

func Role(p Purpose) compatibility.Role {
	switch p {
	case Cleanup:
		return compatibility.PostProcessing
	case Speech:
		return compatibility.Speech
	default:
		return compatibility.Transcription
	}
}
func ValidateUsesForSettings(v config.Settings, uses []Purpose, d Details) error {
	if e := ValidateUses(uses, d); e != nil {
		return e
	}
	if d.ManagedInstanceID == "" {
		return nil
	}
	for _, p := range uses {
		if _, _, e := config.ManagedContract(v, d.ManagedInstanceID, Role(p)); e != nil {
			return e
		}
	}
	return nil
}

// QualifyProjection sets catalog identity only; live endpoints belong to settings admission.
func QualifyProjection(v config.Settings, p Purpose, id string) config.Settings {
	i, c, e := config.ManagedContract(v, id, Role(p))
	if e != nil {
		return v
	}
	switch p {
	case Voice:
		v.VoiceTranscription.Model = i.Model
		v.VoiceTranscription.ModelProfile = c.ModelProfile
		v.VoiceTranscription.CompatibilityProfile = c.CompatibilityProfile
		v.VoiceTranscription.AuthenticationMode = config.AuthenticationModeNone
	case Transcription:
		v.Model = i.Model
		v.ModelProfile = c.ModelProfile
		v.CompatibilityProfile = c.CompatibilityProfile
		v.AuthenticationMode = config.AuthenticationModeNone
	case Cleanup:
		v.PostProcessing.Model = i.Model
		v.PostProcessing.Preset = config.PostProcessingPreset(c.ModelProfile)
		v.PostProcessing.CompatibilityProfile = c.CompatibilityProfile
	case Speech:
		v.TextToSpeech.Model = i.Model
		v.TextToSpeech.ModelProfile = c.ModelProfile
		v.TextToSpeech.CompatibilityProfile = c.CompatibilityProfile
		v.TextToSpeech.AuthenticationMode = config.AuthenticationModeNone
	}
	return v
}
