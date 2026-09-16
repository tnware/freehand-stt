package settings

import (
	"runtime"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/modelsettings"
	"github.com/tnware/freehand-stt/internal/postprocess"
	"github.com/tnware/freehand-stt/internal/savedconnection"
	"github.com/tnware/freehand-stt/internal/speechlanguage"
)

// SettingsDTO is the renderer-safe settings snapshot. It reports credential
// presence and native capability state but never returns credential values.
type SettingsDTO struct {
	Platform               string                  `json:"platform"`
	RememberedModels       modelsettings.Catalog   `json:"rememberedModels"`
	ModelProfiles          modelprofile.Catalog    `json:"modelProfiles"`
	SavedConnections       savedconnection.Catalog `json:"savedConnections"`
	RealtimeLanguages      []speechlanguage.Option `json:"realtimeLanguages"`
	TranscriptionLanguages []speechlanguage.Option `json:"transcriptionLanguages"`
	CompatibilityProfiles  compatibility.Catalog   `json:"compatibilityProfiles"`
	config.Settings
	Configuration                      ConfigurationStatus   `json:"configuration"`
	CredentialConfigured               bool                  `json:"credentialConfigured"`
	PostProcessingCredentialConfigured bool                  `json:"postProcessingCredentialConfigured"`
	TextToSpeechCredentialConfigured   bool                  `json:"textToSpeechCredentialConfigured"`
	HoldAvailable                      bool                  `json:"holdAvailable"`
	HoldAvailabilityReason             string                `json:"holdAvailabilityReason"`
	MicaActive                         bool                  `json:"micaActive"`
	AppearanceModeActive               config.AppearanceMode `json:"appearanceModeActive"`
}

// ConfigurationStatus describes whether the durable settings database is
// usable. Ordinary mutations are blocked while RecoveryRequired is true.
type ConfigurationStatus struct {
	RecoveryRequired bool   `json:"recoveryRequired"`
	ErrorKind        string `json:"errorKind,omitempty"`
	Message          string `json:"message,omitempty"`
}

// GetSettings returns a renderer-safe snapshot of the active runtime profile.
func (s *Service) GetSettings() SettingsDTO {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()
	return s.settingsSnapshotLocked()
}

func (s *Service) settingsSnapshotLocked() SettingsDTO {
	v := s.current()
	ok, reason := s.hold()
	processingCredentialConfigured := s.processKeys != nil && s.processKeys.Configured()
	ttsCredentialConfigured := s.ttsKeys != nil && s.ttsKeys.Configured()
	catalog := savedconnection.Catalog{Entries: []savedconnection.Connection{}, Selected: map[savedconnection.Purpose]string{}}
	if store, ok := s.store.(interface {
		ConnectionCatalog() savedconnection.Catalog
	}); ok {
		catalog = store.ConnectionCatalog()
	}
	models := modelsettings.Catalog{Entries: []modelsettings.Entry{}, Defaults: modelsettings.Defaults()}
	if store, ok := s.store.(interface{ RememberedModels() modelsettings.Catalog }); ok {
		models = store.RememberedModels()
	}
	return SettingsDTO{
		Platform:                           runtime.GOOS,
		RememberedModels:                   models,
		SavedConnections:                   catalog,
		CompatibilityProfiles:              compatibility.Profiles(),
		RealtimeLanguages:                  modelprofile.NemotronLanguages(),
		ModelProfiles:                      modelCatalog(v),
		TranscriptionLanguages:             speechlanguage.Options(),
		Settings:                           v,
		Configuration:                      s.configuration,
		CredentialConfigured:               s.keys.Configured(),
		PostProcessingCredentialConfigured: processingCredentialConfigured,
		TextToSpeechCredentialConfigured:   ttsCredentialConfigured,
		HoldAvailable:                      ok,
		HoldAvailabilityReason:             reason,
		MicaActive:                         s.micaActive,
		AppearanceModeActive:               s.appearanceModeActive,
	}
}

// GetPostProcessingProfiles returns backend-owned request behavior metadata so
// the renderer can present the exact built-in instruction and editable custom
// path without duplicating model-specific protocol text.
func (s *Service) GetPostProcessingProfiles() []postprocess.ProfileDescriptor {
	return postprocess.Profiles()
}

// PreviewVocabulary evaluates a bounded renderer draft without saving or inference.
func (s *Service) PreviewVocabulary(request config.VocabularyPreviewRequest) config.VocabularyPreview {
	return config.InspectVocabulary(request)
}

func modelCatalog(v config.Settings) modelprofile.Catalog {
	c := modelprofile.Profiles(v.CompatibilityProfile, v.PostProcessing.CompatibilityProfile, v.TextToSpeech.CompatibilityProfile)
	c.VoiceTranscription = modelprofile.VoiceProfiles(v.VoiceTranscription.CompatibilityProfile)
	return c
}
