// Package config defines non-secret settings, defaults, and validation.
// Durable changes and credential snapshots belong to the settings package.
package config

import (
	"errors"
	"fmt"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/hotkey"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/speechlanguage"
)

const (
	MinRequestTimeoutSeconds               = 10
	MaxRequestTimeoutSeconds               = 3600
	MinFileTranscriptionTimeoutSeconds     = 60
	MaxFileTranscriptionTimeoutSeconds     = 24 * 60 * 60
	DefaultTranscriptionTimeoutSeconds     = 120
	DefaultFileTranscriptionTimeoutSeconds = 6 * 60 * 60
	DefaultPostProcessingTimeoutSeconds    = 120
	DefaultTextToSpeechTimeoutSeconds      = 180

	MinVADActivitySilenceMS = 100
	MaxVADActivitySilenceMS = 1500
	MinSpeechPaddingMS      = 0
	MaxSpeechPaddingMS      = 1000
	MinAutoStopSilenceMS    = 500
	MaxAutoStopSilenceMS    = 10000
	MinAutoStopSpeechMS     = 100
	MaxAutoStopSpeechMS     = 5000
)

type VADMode string

const (
	VADModeQuality        VADMode = "quality"
	VADModeLowBitrate     VADMode = "low-bitrate"
	VADModeAggressive     VADMode = "aggressive"
	VADModeVeryAggressive VADMode = "very-aggressive"
)

type Settings struct {
	ManagedRuntimes                 []managedruntime.Instance  `json:"managedRuntimes"`
	ManagedInstanceID               string                     `json:"managedInstanceID,omitempty"`
	Vocabulary                      VocabularySettings         `json:"vocabulary"`
	VoiceTranscription              VoiceTranscriptionSettings `json:"voiceTranscription"`
	ModelProfile                    modelprofile.ID            `json:"modelProfile"`
	CompatibilityProfile            compatibility.ID           `json:"compatibilityProfile"`
	BaseURL                         string                     `json:"baseURL"`
	AllowInsecureHTTP               bool                       `json:"allowInsecureHTTP"`
	AuthenticationMode              AuthenticationMode         `json:"authenticationMode"`
	Model                           string                     `json:"model"`
	Language                        string                     `json:"language,omitempty"`
	Headers                         map[string]string          `json:"headers,omitempty"`
	HealthPath                      string                     `json:"healthPath,omitempty"`
	ToggleShortcut                  string                     `json:"toggleShortcut"`
	ShowShortcut                    string                     `json:"showShortcut"`
	HoldShortcut                    string                     `json:"holdShortcut,omitempty"`
	MicrophoneID                    string                     `json:"microphoneID,omitempty"`
	MaxDurationSeconds              int                        `json:"maxDurationSeconds"`
	TranscriptionTimeoutSeconds     int                        `json:"transcriptionTimeoutSeconds"`
	FileTranscriptionTimeoutSeconds int                        `json:"fileTranscriptionTimeoutSeconds"`
	AutoInsert                      bool                       `json:"autoInsert"`
	StartWithWindows                bool                       `json:"startWithWindows"`
	ShowWindowOnLaunch              bool                       `json:"showWindowOnLaunch"`
	CheckForUpdates                 bool                       `json:"checkForUpdates"`
	SetupCompleted                  bool                       `json:"setupCompleted"`
	UseMica                         bool                       `json:"useMica"`
	AppearanceMode                  AppearanceMode             `json:"appearanceMode"`
	OverlayEnabled                  bool                       `json:"overlayEnabled"`
	OverlaySizePercent              int                        `json:"overlaySizePercent"`
	OverlayOpacityPercent           int                        `json:"overlayOpacityPercent"`
	OverlayTopOffset                int                        `json:"overlayTopOffset"`
	OverlayGlowPercent              int                        `json:"overlayGlowPercent"`
	OverlayLayout                   OverlayLayout              `json:"overlayLayout"`
	OverlayAnchor                   OverlayAnchor              `json:"overlayAnchor"`
	OverlayVisibility               OverlayVisibility          `json:"overlayVisibility"`
	OverlayMotion                   OverlayMotion              `json:"overlayMotion"`
	OverlaySurface                  OverlaySurface             `json:"overlaySurface"`
	OverlayVisualizer               OverlayVisualizer          `json:"overlayVisualizer"`
	HistoryEnabled                  bool                       `json:"historyEnabled"`
	VADEnabled                      bool                       `json:"vadEnabled"`
	VADMode                         VADMode                    `json:"vadMode"`
	VADActivitySilenceMS            int                        `json:"vadActivitySilenceMilliseconds"`
	SilenceTrimming                 bool                       `json:"silenceTrimming"`
	SpeechPaddingMS                 int                        `json:"speechPaddingMilliseconds"`
	AutoStopEnabled                 bool                       `json:"autoStopEnabled"`
	AutoStopSilenceMS               int                        `json:"autoStopSilenceMilliseconds"`
	AutoStopMinimumSpeechMS         int                        `json:"autoStopMinimumSpeechMilliseconds"`
	SilenceSplitting                bool                       `json:"silenceSplitting"`
	SegmentSeconds                  int                        `json:"segmentSeconds"`
	SegmentSilenceMS                int                        `json:"segmentSilenceMilliseconds"`
	PostProcessing                  PostProcessingSettings     `json:"postProcessing"`
	TextToSpeech                    TextToSpeechSettings       `json:"textToSpeech"`

	TranscriptionOptions TranscriptionOptions `json:"transcriptionOptions"`
}

func Default() Settings {
	return Settings{
		ManagedRuntimes:      []managedruntime.Instance{},
		Vocabulary:           VocabularySettings{Boost: 3},
		VoiceTranscription:   DefaultVoiceTranscription(),
		CompatibilityProfile: compatibility.Generic,
		// First launch and settings recovery must not select a network peer or a
		// credential-bearing authentication mode on the user's behalf. The setup
		// flow owns that explicit trust decision.
		BaseURL: "", Model: "", ModelProfile: modelprofile.Generic,
		AuthenticationMode: AuthenticationModeNone,
		ToggleShortcut:     "CmdOrCtrl+Shift+Space", ShowShortcut: "",
		MaxDurationSeconds: 120, TranscriptionTimeoutSeconds: DefaultTranscriptionTimeoutSeconds,
		FileTranscriptionTimeoutSeconds: DefaultFileTranscriptionTimeoutSeconds,
		AutoInsert:                      true, ShowWindowOnLaunch: true, CheckForUpdates: true,
		AppearanceMode: AppearanceModeSystem,
		OverlayEnabled: true, OverlaySizePercent: 100, OverlayOpacityPercent: 85, OverlayTopOffset: 18, OverlayGlowPercent: 70,
		OverlayLayout: OverlayLayoutCapsule, OverlayAnchor: OverlayAnchorBottomCenter, OverlayVisibility: OverlayVisibilityAll,
		OverlayMotion: OverlayMotionSystem, OverlaySurface: OverlaySurfaceMinimal, OverlayVisualizer: OverlayVisualizerEnvelope,
		Headers: map[string]string{}, VADEnabled: true, VADMode: VADModeAggressive,
		VADActivitySilenceMS: 400, SpeechPaddingMS: 300,
		AutoStopSilenceMS: 2000, AutoStopMinimumSpeechMS: 300,
		SegmentSeconds: 90, SegmentSilenceMS: 700,
		PostProcessing: PostProcessingSettings{
			CompatibilityProfile: compatibility.Generic,
			BaseURL:              "",
			Preset:               PostProcessingPresetGeneric,
			SystemPrompt:         DefaultPostProcessingInstruction,
			Styling:              "semi-casual", Structure: "prose", Context: "general",
			TimeoutSeconds: DefaultPostProcessingTimeoutSeconds,
		},
		TextToSpeech: TextToSpeechSettings{
			ModelProfile:         modelprofile.Generic,
			CompatibilityProfile: compatibility.Generic,
			AuthenticationMode:   AuthenticationModeNone,
			Speed:                1,
			TimeoutSeconds:       DefaultTextToSpeechTimeoutSeconds,
		},
	}
}

func Validate(s Settings) error {
	return validate(s, false)
}

// ValidateStored validates durable settings when reopening a database. Managed
// task preferences survive a runtime migration even when the selected model
// cannot execute them. This is not renderer or request admission: those callers
// must use Validate and the resolved task validators.
func ValidateStored(s Settings) error {
	return validate(s, true)
}

func validate(s Settings, stored bool) error {
	if err := managedruntime.ValidateInstances(s.ManagedRuntimes); err != nil {
		return err
	}
	if err := validateManagedReferences(s); err != nil {
		return err
	}
	if err := ValidateVocabulary(s.Vocabulary); err != nil {
		return fieldError("vocabulary", "Check vocabulary text and strength.", err)
	}
	if err := validateVoiceTranscription(s.VoiceTranscription, stored && s.VoiceTranscription.ManagedInstanceID != ""); err != nil {
		return fieldError("voice-transcription", "Check the voice transcription connection, model profile, language, and options.", err)
	}
	if _, err := compatibility.Resolve(s.CompatibilityProfile, compatibility.Transcription); err != nil {
		return fieldError("compatibilityProfile", "Choose a supported transcription server profile.", err)
	}
	if _, err := compatibility.Resolve(s.PostProcessing.CompatibilityProfile, compatibility.PostProcessing); err != nil {
		return fieldError("postProcessing.compatibilityProfile", "Choose a supported cleanup server profile.", err)
	}
	if _, err := compatibility.Resolve(s.TextToSpeech.CompatibilityProfile, compatibility.Speech); err != nil {
		return fieldError("textToSpeech.compatibilityProfile", "Choose a supported speech server profile.", err)
	}
	if err := validateTimeout("transcription request", s.TranscriptionTimeoutSeconds, MinRequestTimeoutSeconds, MaxRequestTimeoutSeconds); err != nil {
		return fieldError("transcriptionTimeoutSeconds", fmt.Sprintf("Enter a transcription timeout from %d to %d seconds.", MinRequestTimeoutSeconds, MaxRequestTimeoutSeconds), err)
	}
	if err := validateTimeout("audio file transcription", s.FileTranscriptionTimeoutSeconds, MinFileTranscriptionTimeoutSeconds, MaxFileTranscriptionTimeoutSeconds); err != nil {
		return fieldError("fileTranscriptionTimeoutSeconds", fmt.Sprintf("Enter an audio file timeout from %d to %d seconds.", MinFileTranscriptionTimeoutSeconds, MaxFileTranscriptionTimeoutSeconds), err)
	}
	if err := validateTimeout("post-processing request", s.PostProcessing.TimeoutSeconds, MinRequestTimeoutSeconds, MaxRequestTimeoutSeconds); err != nil {
		return fieldError("postProcessing.timeoutSeconds", fmt.Sprintf("Enter a cleanup timeout from %d to %d seconds.", MinRequestTimeoutSeconds, MaxRequestTimeoutSeconds), err)
	}
	if err := validatePersistedSTTSettings(s); err != nil {
		return fieldError("baseURL", "Check the transcription connection, authentication mode, model, health path, and custom headers.", err)
	}
	switch s.AppearanceMode {
	case AppearanceModeSystem, AppearanceModeLight, AppearanceModeDark:
	default:
		return fieldError("appearanceMode", "Choose system, light, or dark appearance mode.", errors.New("appearance mode is invalid"))
	}
	validateTranscription := modelprofile.ValidateTranscription
	if stored && s.ManagedInstanceID != "" {
		validateTranscription = modelprofile.ValidateStoredTranscription
	}
	if err := validateTranscription(s.ModelProfile, s.CompatibilityProfile, s.Language, s.TranscriptionOptions.Inference()); err != nil {
		return fieldError("modelProfile", "Choose a compatible transcription model profile, language, and options.", err)
	}
	if err := speechlanguage.Validate(s.Language); err != nil {
		return fieldError("language", "Choose a supported transcription language.", err)
	}
	maximumDuration := 262
	if s.SilenceSplitting {
		maximumDuration = 3600
	}
	if s.MaxDurationSeconds < 1 || s.MaxDurationSeconds > maximumDuration {
		return fieldError("maxDurationSeconds", fmt.Sprintf("Enter a recording limit from 1 to %d seconds.", maximumDuration), fmt.Errorf("maximum duration must be between 1 and %d seconds", maximumDuration))
	}
	if err := ValidateOverlayPreferences(s.OverlayPreferences()); err != nil {
		return fieldError("overlayEnabled", "Check the overlay layout, placement, appearance, and size.", err)
	}
	switch s.VADMode {
	case VADModeQuality, VADModeLowBitrate, VADModeAggressive, VADModeVeryAggressive:
	default:
		return fieldError("vadMode", "Choose a supported voice activity detection mode.", errors.New("voice activity detection mode is invalid"))
	}
	if s.VADActivitySilenceMS < MinVADActivitySilenceMS || s.VADActivitySilenceMS > MaxVADActivitySilenceMS {
		return fieldError("vadActivitySilenceMilliseconds", fmt.Sprintf("Enter a voice activity silence delay from %d to %d milliseconds.", MinVADActivitySilenceMS, MaxVADActivitySilenceMS), fmt.Errorf("voice activity silence must be between %d and %d milliseconds", MinVADActivitySilenceMS, MaxVADActivitySilenceMS))
	}
	if s.SpeechPaddingMS < MinSpeechPaddingMS || s.SpeechPaddingMS > MaxSpeechPaddingMS {
		return fieldError("speechPaddingMilliseconds", fmt.Sprintf("Enter speech padding from %d to %d milliseconds.", MinSpeechPaddingMS, MaxSpeechPaddingMS), fmt.Errorf("speech padding must be between %d and %d milliseconds", MinSpeechPaddingMS, MaxSpeechPaddingMS))
	}
	if s.AutoStopSilenceMS < MinAutoStopSilenceMS || s.AutoStopSilenceMS > MaxAutoStopSilenceMS {
		return fieldError("autoStopSilenceMilliseconds", fmt.Sprintf("Enter an automatic stop silence delay from %d to %d milliseconds.", MinAutoStopSilenceMS, MaxAutoStopSilenceMS), fmt.Errorf("automatic stop silence must be between %d and %d milliseconds", MinAutoStopSilenceMS, MaxAutoStopSilenceMS))
	}
	if s.AutoStopMinimumSpeechMS < MinAutoStopSpeechMS || s.AutoStopMinimumSpeechMS > MaxAutoStopSpeechMS {
		return fieldError("autoStopMinimumSpeechMilliseconds", fmt.Sprintf("Enter minimum speech before automatic stop from %d to %d milliseconds.", MinAutoStopSpeechMS, MaxAutoStopSpeechMS), fmt.Errorf("automatic stop minimum speech must be between %d and %d milliseconds", MinAutoStopSpeechMS, MaxAutoStopSpeechMS))
	}
	if s.AutoStopEnabled && s.AutoStopSilenceMS < s.VADActivitySilenceMS {
		return fieldError("autoStopSilenceMilliseconds", "Set automatic stop silence at least as long as the voice activity silence delay.", errors.New("automatic stop silence must be at least the voice activity silence delay"))
	}
	if (s.SilenceTrimming || s.AutoStopEnabled || s.SilenceSplitting) && !s.VADEnabled {
		return fieldError("vadEnabled", "Enable voice activity detection to use silence trimming, automatic stop, or silence splitting.", errors.New("silence trimming, automatic stop, and silence splitting require voice activity detection"))
	}
	if s.SegmentSeconds < 15 || s.SegmentSeconds > 180 {
		return fieldError("segmentSeconds", "Enter a segment target from 15 to 180 seconds.", errors.New("segment target must be between 15 and 180 seconds"))
	}
	if s.SegmentSilenceMS < 200 || s.SegmentSilenceMS > 3000 {
		return fieldError("segmentSilenceMilliseconds", "Enter a segment silence delay from 200 to 3000 milliseconds.", errors.New("segment silence must be between 200 and 3000 milliseconds"))
	}
	if len(s.MicrophoneID) > 1024 {
		return fieldError("microphoneID", "Choose a microphone with an identifier of at most 1024 bytes.", errors.New("microphone identifier is too long"))
	}
	if err := hotkey.ValidateAssignments(hotkey.ShortcutAssignments{
		ToggleRecording: s.ToggleShortcut,
		ShowFreehand:    s.ShowShortcut,
		HoldToTalk:      s.HoldShortcut,
	}); err != nil {
		return fieldError("toggleShortcut", "Choose valid, distinct shortcuts for recording, showing Freehand, and hold-to-talk.", fmt.Errorf("invalid shortcut settings: %w", err))
	}
	if err := modelprofile.ValidateCleanup(modelprofile.ID(s.PostProcessing.Preset), s.PostProcessing.CompatibilityProfile, s.PostProcessing.GenerationOptions); err != nil {
		return fieldError("postProcessing.preset", "Choose a compatible cleanup model profile and generation options.", err)
	}
	if s.PostProcessing.ManagedInstanceID != "" {
		if err := validateManagedTransport(s.PostProcessing.BaseURL, s.PostProcessing.AllowInsecureHTTP, AuthenticationModeNone, "", nil); err != nil {
			return err
		}
		if err := validatePostProcessingOptions(s.PostProcessing); err != nil {
			return err
		}
	} else if s.PostProcessing.Enabled {
		if err := ValidatePostProcessing(s.PostProcessing); err != nil {
			return err
		}
	}
	if s.TextToSpeech.ManagedInstanceID != "" {
		if err := validateTextToSpeech(s.TextToSpeech, s.TextToSpeech.Enabled, true); err != nil {
			return err
		}
	} else if s.TextToSpeech.Enabled {
		if err := ValidateTextToSpeech(s.TextToSpeech, true); err != nil {
			return err
		}
	} else if err := ValidateTextToSpeech(s.TextToSpeech, false); err != nil {
		return err
	}
	return nil
}

func validateTimeout(label string, seconds, minimum, maximum int) error {
	if seconds < minimum || seconds > maximum {
		return fmt.Errorf("%s timeout must be between %d and %d seconds", label, minimum, maximum)
	}
	return nil
}
