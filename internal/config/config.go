package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/hotkey"
)

const (
	MaxBaseURLBytes       = 2048
	MaxHealthPathBytes    = 1024
	MaxHeaderCount        = 32
	MaxHeaderNameBytes    = 256
	MaxHeaderValueBytes   = 4096
	MaxHeaderBytes        = 16 * 1024
	MaxPromptBytes        = 8 * 1024
	MaxTTSInputBytes      = 256 * 1024
	MaxTTSInputCharacters = 4096
	MaxSettingsFileBytes  = 1 << 20
	MaxReportedFields     = 16

	MinRequestTimeoutSeconds               = 10
	MaxRequestTimeoutSeconds               = 3600
	MinFileTranscriptionTimeoutSeconds     = 60
	MaxFileTranscriptionTimeoutSeconds     = 24 * 60 * 60
	DefaultTranscriptionTimeoutSeconds     = 120
	DefaultFileTranscriptionTimeoutSeconds = 6 * 60 * 60
	DefaultPostProcessingTimeoutSeconds    = 120
	DefaultTextToSpeechTimeoutSeconds      = 180

	MinOverlaySizePercent    = 75
	MaxOverlaySizePercent    = 150
	MinOverlayOpacityPercent = 40
	MaxOverlayOpacityPercent = 100
	MinOverlayTopOffset      = 0
	MaxOverlayTopOffset      = 240
	MinOverlayGlowPercent    = 0
	MaxOverlayGlowPercent    = 100

	MinVADActivitySilenceMS = 100
	MaxVADActivitySilenceMS = 1500
	MinSpeechPaddingMS      = 0
	MaxSpeechPaddingMS      = 1000
	MinAutoStopSilenceMS    = 500
	MaxAutoStopSilenceMS    = 10000
	MinAutoStopSpeechMS     = 100
	MaxAutoStopSpeechMS     = 5000

	// DefaultPostProcessingInstruction is the recoverable starting point for
	// the editable custom processing profile. It remains ordinary durable user
	// configuration after the first save.
	DefaultPostProcessingInstruction = "Clean this speech-to-text transcript without changing its meaning. Return only the cleaned transcript."
)

type PostProcessingPreset string

type AuthenticationMode string

type AppearanceMode string

type OverlayLayout string

type OverlayAnchor string

type OverlayVisibility string

type OverlayMotion string

type OverlaySurface string

type OverlayVisualizer string

const (
	AuthenticationModeAPIKey AuthenticationMode = "api-key"
	AuthenticationModeNone   AuthenticationMode = "none"
)

const (
	AppearanceModeSystem AppearanceMode = "system"
	AppearanceModeLight  AppearanceMode = "light"
	AppearanceModeDark   AppearanceMode = "dark"
)

const (
	OverlayLayoutMinimal  OverlayLayout = "minimal"
	OverlayLayoutCapsule  OverlayLayout = "capsule"
	OverlayLayoutMeter    OverlayLayout = "meter"
	OverlayLayoutDetailed OverlayLayout = "detailed"
)

const (
	OverlayAnchorTopLeft      OverlayAnchor = "top-left"
	OverlayAnchorTopCenter    OverlayAnchor = "top-center"
	OverlayAnchorTopRight     OverlayAnchor = "top-right"
	OverlayAnchorBottomLeft   OverlayAnchor = "bottom-left"
	OverlayAnchorBottomCenter OverlayAnchor = "bottom-center"
	OverlayAnchorBottomRight  OverlayAnchor = "bottom-right"
)

const (
	OverlayVisibilityRecording OverlayVisibility = "recording"
	OverlayVisibilityActive    OverlayVisibility = "active"
	OverlayVisibilityAll       OverlayVisibility = "all"
)

const (
	OverlayMotionSystem  OverlayMotion = "system"
	OverlayMotionReduced OverlayMotion = "reduced"
)

const (
	OverlaySurfaceGlass   OverlaySurface = "glass"
	OverlaySurfaceSolid   OverlaySurface = "solid"
	OverlaySurfaceMinimal OverlaySurface = "minimal"
)

const (
	OverlayVisualizerBars     OverlayVisualizer = "bars"
	OverlayVisualizerPulse    OverlayVisualizer = "pulse"
	OverlayVisualizerEnvelope OverlayVisualizer = "envelope"
	OverlayVisualizerMeter    OverlayVisualizer = "meter"
)

const (
	PostProcessingPresetGeneric PostProcessingPreset = "generic"
	PostProcessingPresetS1Mini  PostProcessingPreset = "s1-mini"
)

var (
	s1MiniStylingValues   = []string{"casual", "semi-casual", "semi-formal", "formal"}
	s1MiniStructureValues = []string{"prose", "lists"}
	s1MiniContextValues   = []string{"general", "email"}
)

// S1MiniStylingValues returns the exact styling vocabulary trained by the
// supported S1-mini v1 profile.
func S1MiniStylingValues() []string {
	return append([]string(nil), s1MiniStylingValues...)
}

// S1MiniStructureValues returns the exact supported output structures.
func S1MiniStructureValues() []string {
	return append([]string(nil), s1MiniStructureValues...)
}

// S1MiniContextValues returns the exact supported transcript contexts.
func S1MiniContextValues() []string {
	return append([]string(nil), s1MiniContextValues...)
}

type PostProcessingSettings struct {
	CompatibilityProfile compatibility.ID     `json:"compatibilityProfile"`
	Enabled              bool                 `json:"enabled"`
	BaseURL              string               `json:"baseURL"`
	AllowInsecureHTTP    bool                 `json:"allowInsecureHTTP"`
	Model                string               `json:"model"`
	Preset               PostProcessingPreset `json:"preset"`
	SystemPrompt         string               `json:"systemPrompt"`
	Styling              string               `json:"styling"`
	Structure            string               `json:"structure"`
	Context              string               `json:"context"`
	TimeoutSeconds       int                  `json:"timeoutSeconds"`
}

type TextToSpeechSettings struct {
	CompatibilityProfile compatibility.ID   `json:"compatibilityProfile"`
	Enabled              bool               `json:"enabled"`
	BaseURL              string             `json:"baseURL"`
	AllowInsecureHTTP    bool               `json:"allowInsecureHTTP"`
	AuthenticationMode   AuthenticationMode `json:"authenticationMode"`
	Model                string             `json:"model"`
	Voice                string             `json:"voice"`
	Speed                float64            `json:"speed"`
	TimeoutSeconds       int                `json:"timeoutSeconds"`
}

type VADMode string

const (
	VADModeQuality        VADMode = "quality"
	VADModeLowBitrate     VADMode = "low-bitrate"
	VADModeAggressive     VADMode = "aggressive"
	VADModeVeryAggressive VADMode = "very-aggressive"
)

type Settings struct {
	CompatibilityProfile            compatibility.ID       `json:"compatibilityProfile"`
	BaseURL                         string                 `json:"baseURL"`
	AllowInsecureHTTP               bool                   `json:"allowInsecureHTTP"`
	AuthenticationMode              AuthenticationMode     `json:"authenticationMode"`
	Model                           string                 `json:"model"`
	Language                        string                 `json:"language,omitempty"`
	Headers                         map[string]string      `json:"headers,omitempty"`
	HealthPath                      string                 `json:"healthPath,omitempty"`
	ToggleShortcut                  string                 `json:"toggleShortcut"`
	ShowShortcut                    string                 `json:"showShortcut"`
	HoldShortcut                    string                 `json:"holdShortcut,omitempty"`
	MicrophoneID                    string                 `json:"microphoneID,omitempty"`
	MaxDurationSeconds              int                    `json:"maxDurationSeconds"`
	TranscriptionTimeoutSeconds     int                    `json:"transcriptionTimeoutSeconds"`
	FileTranscriptionTimeoutSeconds int                    `json:"fileTranscriptionTimeoutSeconds"`
	AutoInsert                      bool                   `json:"autoInsert"`
	StartWithWindows                bool                   `json:"startWithWindows"`
	ShowWindowOnLaunch              bool                   `json:"showWindowOnLaunch"`
	CheckForUpdates                 bool                   `json:"checkForUpdates"`
	SetupCompleted                  bool                   `json:"setupCompleted"`
	UseMica                         bool                   `json:"useMica"`
	AppearanceMode                  AppearanceMode         `json:"appearanceMode"`
	OverlayEnabled                  bool                   `json:"overlayEnabled"`
	OverlaySizePercent              int                    `json:"overlaySizePercent"`
	OverlayOpacityPercent           int                    `json:"overlayOpacityPercent"`
	OverlayTopOffset                int                    `json:"overlayTopOffset"`
	OverlayGlowPercent              int                    `json:"overlayGlowPercent"`
	OverlayLayout                   OverlayLayout          `json:"overlayLayout"`
	OverlayAnchor                   OverlayAnchor          `json:"overlayAnchor"`
	OverlayVisibility               OverlayVisibility      `json:"overlayVisibility"`
	OverlayMotion                   OverlayMotion          `json:"overlayMotion"`
	OverlaySurface                  OverlaySurface         `json:"overlaySurface"`
	OverlayVisualizer               OverlayVisualizer      `json:"overlayVisualizer"`
	HistoryEnabled                  bool                   `json:"historyEnabled"`
	VADEnabled                      bool                   `json:"vadEnabled"`
	VADMode                         VADMode                `json:"vadMode"`
	VADActivitySilenceMS            int                    `json:"vadActivitySilenceMilliseconds"`
	SilenceTrimming                 bool                   `json:"silenceTrimming"`
	SpeechPaddingMS                 int                    `json:"speechPaddingMilliseconds"`
	AutoStopEnabled                 bool                   `json:"autoStopEnabled"`
	AutoStopSilenceMS               int                    `json:"autoStopSilenceMilliseconds"`
	AutoStopMinimumSpeechMS         int                    `json:"autoStopMinimumSpeechMilliseconds"`
	SilenceSplitting                bool                   `json:"silenceSplitting"`
	SegmentSeconds                  int                    `json:"segmentSeconds"`
	SegmentSilenceMS                int                    `json:"segmentSilenceMilliseconds"`
	PostProcessing                  PostProcessingSettings `json:"postProcessing"`
	TextToSpeech                    TextToSpeechSettings   `json:"textToSpeech"`
}

func Default() Settings {
	return Settings{
		CompatibilityProfile: compatibility.Generic,
		// First launch and settings recovery must not select a network peer or a
		// credential-bearing authentication mode on the user's behalf. The setup
		// flow owns that explicit trust decision.
		BaseURL: "", Model: "",
		AuthenticationMode: AuthenticationModeNone,
		ToggleShortcut:     "CmdOrCtrl+Shift+Space", ShowShortcut: "CmdOrCtrl+Shift+D",
		MaxDurationSeconds: 120, TranscriptionTimeoutSeconds: DefaultTranscriptionTimeoutSeconds,
		FileTranscriptionTimeoutSeconds: DefaultFileTranscriptionTimeoutSeconds,
		AutoInsert:                      true, ShowWindowOnLaunch: true, CheckForUpdates: true,
		AppearanceMode: AppearanceModeSystem,
		OverlayEnabled: true, OverlaySizePercent: 100, OverlayOpacityPercent: 100, OverlayTopOffset: 18, OverlayGlowPercent: 100,
		OverlayLayout: OverlayLayoutCapsule, OverlayAnchor: OverlayAnchorTopCenter, OverlayVisibility: OverlayVisibilityAll,
		OverlayMotion: OverlayMotionSystem, OverlaySurface: OverlaySurfaceGlass, OverlayVisualizer: OverlayVisualizerBars,
		Headers: map[string]string{}, VADEnabled: true, VADMode: VADModeAggressive,
		VADActivitySilenceMS: 400, SpeechPaddingMS: 300,
		AutoStopSilenceMS: 2000, AutoStopMinimumSpeechMS: 300,
		SegmentSeconds: 90, SegmentSilenceMS: 700,
		PostProcessing: PostProcessingSettings{
			CompatibilityProfile: compatibility.Generic,
			BaseURL:              "http://127.0.0.1:8080/v1",
			Preset:               PostProcessingPresetGeneric,
			SystemPrompt:         DefaultPostProcessingInstruction,
			Styling:              "semi-casual", Structure: "prose", Context: "general",
			TimeoutSeconds: DefaultPostProcessingTimeoutSeconds,
		},
		TextToSpeech: TextToSpeechSettings{
			CompatibilityProfile: compatibility.Generic,
			AuthenticationMode:   AuthenticationModeNone,
			Speed:                1,
			TimeoutSeconds:       DefaultTextToSpeechTimeoutSeconds,
		},
	}
}

// EffectiveAppearanceMode returns the colour mode that may be applied to the
// native windows and renderers. Windows owns Mica's light/dark appearance, so
// a saved solid-window preference is deliberately ignored while Mica is on.
func EffectiveAppearanceMode(useMica bool, mode AppearanceMode) AppearanceMode {
	if useMica {
		return AppearanceModeSystem
	}
	return mode
}

func (s Settings) EffectiveAppearanceMode() AppearanceMode {
	return EffectiveAppearanceMode(s.UseMica, s.AppearanceMode)
}

var headerNameRE = regexp.MustCompile(`^[!#$%&'*+\-.^_` + "`" + `|~0-9A-Za-z]+$`)

func Validate(s Settings) error {
	if _, err := compatibility.Resolve(s.CompatibilityProfile, compatibility.Transcription); err != nil {
		return err
	}
	if _, err := compatibility.Resolve(s.PostProcessing.CompatibilityProfile, compatibility.PostProcessing); err != nil {
		return err
	}
	if _, err := compatibility.Resolve(s.TextToSpeech.CompatibilityProfile, compatibility.Speech); err != nil {
		return err
	}
	if err := validateTimeout("transcription request", s.TranscriptionTimeoutSeconds, MinRequestTimeoutSeconds, MaxRequestTimeoutSeconds); err != nil {
		return err
	}
	if err := validateTimeout("audio file transcription", s.FileTranscriptionTimeoutSeconds, MinFileTranscriptionTimeoutSeconds, MaxFileTranscriptionTimeoutSeconds); err != nil {
		return err
	}
	if err := validateTimeout("post-processing request", s.PostProcessing.TimeoutSeconds, MinRequestTimeoutSeconds, MaxRequestTimeoutSeconds); err != nil {
		return err
	}
	if err := validatePersistedSTTSettings(s); err != nil {
		return err
	}
	switch s.AppearanceMode {
	case AppearanceModeSystem, AppearanceModeLight, AppearanceModeDark:
	default:
		return errors.New("appearance mode is invalid")
	}
	if len(s.Language) > 32 || strings.ContainsAny(s.Language, "\r\n") {
		return errors.New("language must be at most 32 characters")
	}
	maximumDuration := 262
	if s.SilenceSplitting {
		maximumDuration = 3600
	}
	if s.MaxDurationSeconds < 1 || s.MaxDurationSeconds > maximumDuration {
		return fmt.Errorf("maximum duration must be between 1 and %d seconds", maximumDuration)
	}
	if err := ValidateOverlayPreferences(s.OverlayPreferences()); err != nil {
		return err
	}
	switch s.VADMode {
	case VADModeQuality, VADModeLowBitrate, VADModeAggressive, VADModeVeryAggressive:
	default:
		return errors.New("voice activity detection mode is invalid")
	}
	if s.VADActivitySilenceMS < MinVADActivitySilenceMS || s.VADActivitySilenceMS > MaxVADActivitySilenceMS {
		return fmt.Errorf("voice activity silence must be between %d and %d milliseconds", MinVADActivitySilenceMS, MaxVADActivitySilenceMS)
	}
	if s.SpeechPaddingMS < MinSpeechPaddingMS || s.SpeechPaddingMS > MaxSpeechPaddingMS {
		return fmt.Errorf("speech padding must be between %d and %d milliseconds", MinSpeechPaddingMS, MaxSpeechPaddingMS)
	}
	if s.AutoStopSilenceMS < MinAutoStopSilenceMS || s.AutoStopSilenceMS > MaxAutoStopSilenceMS {
		return fmt.Errorf("automatic stop silence must be between %d and %d milliseconds", MinAutoStopSilenceMS, MaxAutoStopSilenceMS)
	}
	if s.AutoStopMinimumSpeechMS < MinAutoStopSpeechMS || s.AutoStopMinimumSpeechMS > MaxAutoStopSpeechMS {
		return fmt.Errorf("automatic stop minimum speech must be between %d and %d milliseconds", MinAutoStopSpeechMS, MaxAutoStopSpeechMS)
	}
	if s.AutoStopEnabled && s.AutoStopSilenceMS < s.VADActivitySilenceMS {
		return errors.New("automatic stop silence must be at least the voice activity silence delay")
	}
	if (s.SilenceTrimming || s.AutoStopEnabled || s.SilenceSplitting) && !s.VADEnabled {
		return errors.New("silence trimming, automatic stop, and silence splitting require voice activity detection")
	}
	if s.SegmentSeconds < 15 || s.SegmentSeconds > 180 {
		return errors.New("segment target must be between 15 and 180 seconds")
	}
	if s.SegmentSilenceMS < 200 || s.SegmentSilenceMS > 3000 {
		return errors.New("segment silence must be between 200 and 3000 milliseconds")
	}
	if len(s.MicrophoneID) > 1024 {
		return errors.New("microphone identifier is too long")
	}
	if err := hotkey.ValidateAssignments(hotkey.ShortcutAssignments{
		ToggleRecording: s.ToggleShortcut,
		ShowFreehand:    s.ShowShortcut,
		HoldToTalk:      s.HoldShortcut,
	}); err != nil {
		return fmt.Errorf("invalid shortcut settings: %w", err)
	}
	if s.PostProcessing.Enabled {
		if err := ValidatePostProcessing(s.PostProcessing); err != nil {
			return err
		}
	}
	if s.TextToSpeech.Enabled {
		if err := ValidateTextToSpeech(s.TextToSpeech, true); err != nil {
			return err
		}
	} else if err := ValidateTextToSpeech(s.TextToSpeech, false); err != nil {
		return err
	}
	return nil
}

func ValidateTextToSpeech(s TextToSpeechSettings, requireConnection bool) error {
	if _, err := compatibility.Resolve(s.CompatibilityProfile, compatibility.Speech); err != nil {
		return err
	}
	if err := validateTimeout("speech generation", s.TimeoutSeconds, MinRequestTimeoutSeconds, MaxRequestTimeoutSeconds); err != nil {
		return err
	}
	if !requireConnection && strings.TrimSpace(s.BaseURL) == "" && strings.TrimSpace(s.Model) == "" && strings.TrimSpace(s.Voice) == "" {
		if s.Speed == 0 {
			return nil
		}
		if s.Speed < 0.25 || s.Speed > 4 {
			return errors.New("speech playback speed must be between 0.25 and 4")
		}
		return nil
	}
	if err := validateTextToSpeechConnection(s.BaseURL, s.AllowInsecureHTTP, s.AuthenticationMode, s.Model, requireConnection); err != nil {
		return err
	}
	if (requireConnection && strings.TrimSpace(s.Voice) == "") || len(s.Voice) > 200 {
		return errors.New("speech playback voice is required and must be at most 200 characters")
	}
	if s.Speed < 0.25 || s.Speed > 4 {
		return errors.New("speech playback speed must be between 0.25 and 4")
	}
	return nil
}

func validateTimeout(label string, seconds, minimum, maximum int) error {
	if seconds < minimum || seconds > maximum {
		return fmt.Errorf("%s timeout must be between %d and %d seconds", label, minimum, maximum)
	}
	return nil
}

// ValidateTextToSpeechConnection validates only values needed for standard
// OpenAI-compatible model discovery. A model is optional until one is chosen.
func ValidateTextToSpeechConnection(baseURL string, allowInsecureHTTP bool, authenticationMode AuthenticationMode, model string) error {
	return validateTextToSpeechConnection(baseURL, allowInsecureHTTP, authenticationMode, model, false)
}

func validateTextToSpeechConnection(baseURL string, allowInsecureHTTP bool, authenticationMode AuthenticationMode, model string, requireModel bool) error {
	if err := validateBaseURL(baseURL, allowInsecureHTTP, "speech playback "); err != nil {
		return err
	}
	switch authenticationMode {
	case AuthenticationModeAPIKey, AuthenticationModeNone:
	default:
		return errors.New("speech playback authentication mode is invalid")
	}
	if (requireModel && strings.TrimSpace(model) == "") || len(model) > 200 {
		return errors.New("speech playback model is required and must be at most 200 characters")
	}
	return nil
}

// OverlayPreferences is the renderer-safe, credential-free portion of Settings
// accepted by the native preview binding. It is deliberately narrower than a
// settings save request so previewing cannot mutate application configuration.
type OverlayPreferences struct {
	Layout         OverlayLayout     `json:"layout"`
	Anchor         OverlayAnchor     `json:"anchor"`
	Visibility     OverlayVisibility `json:"visibility"`
	Motion         OverlayMotion     `json:"motion"`
	Surface        OverlaySurface    `json:"surface"`
	Visualizer     OverlayVisualizer `json:"visualizer"`
	SizePercent    int               `json:"sizePercent"`
	OpacityPercent int               `json:"opacityPercent"`
	EdgeOffset     int               `json:"edgeOffset"`
	GlowPercent    int               `json:"glowPercent"`
}

func (s Settings) OverlayPreferences() OverlayPreferences {
	return OverlayPreferences{
		Layout: s.OverlayLayout, Anchor: s.OverlayAnchor, Visibility: s.OverlayVisibility,
		Motion: s.OverlayMotion, Surface: s.OverlaySurface, Visualizer: s.OverlayVisualizer,
		SizePercent: s.OverlaySizePercent, OpacityPercent: s.OverlayOpacityPercent,
		EdgeOffset: s.OverlayTopOffset, GlowPercent: s.OverlayGlowPercent,
	}
}

func ValidateOverlayPreferences(preferences OverlayPreferences) error {
	switch preferences.Layout {
	case OverlayLayoutMinimal, OverlayLayoutCapsule, OverlayLayoutMeter, OverlayLayoutDetailed:
	default:
		return errors.New("overlay layout is invalid")
	}
	switch preferences.Anchor {
	case OverlayAnchorTopLeft, OverlayAnchorTopCenter, OverlayAnchorTopRight,
		OverlayAnchorBottomLeft, OverlayAnchorBottomCenter, OverlayAnchorBottomRight:
	default:
		return errors.New("overlay anchor is invalid")
	}
	switch preferences.Visibility {
	case OverlayVisibilityRecording, OverlayVisibilityActive, OverlayVisibilityAll:
	default:
		return errors.New("overlay visibility is invalid")
	}
	switch preferences.Motion {
	case OverlayMotionSystem, OverlayMotionReduced:
	default:
		return errors.New("overlay motion is invalid")
	}
	switch preferences.Surface {
	case OverlaySurfaceGlass, OverlaySurfaceSolid, OverlaySurfaceMinimal:
	default:
		return errors.New("overlay surface is invalid")
	}
	switch preferences.Visualizer {
	case OverlayVisualizerBars, OverlayVisualizerPulse, OverlayVisualizerEnvelope, OverlayVisualizerMeter:
	default:
		return errors.New("overlay visualizer is invalid")
	}
	if preferences.SizePercent < MinOverlaySizePercent || preferences.SizePercent > MaxOverlaySizePercent {
		return fmt.Errorf("overlay size must be between %d and %d percent", MinOverlaySizePercent, MaxOverlaySizePercent)
	}
	if preferences.OpacityPercent < MinOverlayOpacityPercent || preferences.OpacityPercent > MaxOverlayOpacityPercent {
		return fmt.Errorf("overlay opacity must be between %d and %d percent", MinOverlayOpacityPercent, MaxOverlayOpacityPercent)
	}
	if preferences.EdgeOffset < MinOverlayTopOffset || preferences.EdgeOffset > MaxOverlayTopOffset {
		return fmt.Errorf("overlay edge distance must be between %d and %d pixels", MinOverlayTopOffset, MaxOverlayTopOffset)
	}
	if preferences.GlowPercent < MinOverlayGlowPercent || preferences.GlowPercent > MaxOverlayGlowPercent {
		return fmt.Errorf("overlay glow must be between %d and %d percent", MinOverlayGlowPercent, MaxOverlayGlowPercent)
	}
	return nil
}

// validatePersistedSTTSettings permits exactly one unconfigured connection
// state: both endpoint and model are empty while first-run setup is incomplete.
// This lets settings recovery persist safe defaults without making a partially
// configured or previously completed profile valid.
func validatePersistedSTTSettings(s Settings) error {
	if !s.SetupCompleted && s.BaseURL == "" && s.Model == "" {
		switch s.AuthenticationMode {
		case AuthenticationModeAPIKey, AuthenticationModeNone:
		default:
			return errors.New("authentication mode is invalid")
		}
		if len(s.HealthPath) > MaxHealthPathBytes {
			return fmt.Errorf("health path must be at most %d bytes", MaxHealthPathBytes)
		}
		if s.HealthPath != "" && (!strings.HasPrefix(s.HealthPath, "/") || strings.ContainsAny(s.HealthPath, "?#\r\n")) {
			return errors.New("health path must start with / and contain no query or fragment; it is appended to the base URL path")
		}
		return validateHeaders(s.Headers)
	}
	return validateSTTConnection(s.BaseURL, s.AllowInsecureHTTP, s.AuthenticationMode, s.Model, s.HealthPath, s.Headers, true)
}

// ValidateSTTConnection validates only renderer-controlled values needed for
// an STT metadata probe. A model is optional so discovery can run before the
// user has selected one.
func ValidateSTTConnection(baseURL string, allowInsecureHTTP bool, authenticationMode AuthenticationMode, model, healthPath string, headers map[string]string) error {
	return validateSTTConnection(baseURL, allowInsecureHTTP, authenticationMode, model, healthPath, headers, false)
}

func validateSTTConnection(baseURL string, allowInsecureHTTP bool, authenticationMode AuthenticationMode, model, healthPath string, headers map[string]string, requireModel bool) error {
	if err := validateBaseURL(baseURL, allowInsecureHTTP, ""); err != nil {
		return err
	}
	switch authenticationMode {
	case AuthenticationModeAPIKey, AuthenticationModeNone:
	default:
		return errors.New("authentication mode is invalid")
	}
	if (requireModel && strings.TrimSpace(model) == "") || len(model) > 200 {
		return errors.New("model is required and must be at most 200 characters")
	}
	if len(healthPath) > MaxHealthPathBytes {
		return fmt.Errorf("health path must be at most %d bytes", MaxHealthPathBytes)
	}
	if healthPath != "" && (!strings.HasPrefix(healthPath, "/") || strings.ContainsAny(healthPath, "?#\r\n")) {
		return errors.New("health path must start with / and contain no query or fragment; it is appended to the base URL path")
	}
	return validateHeaders(headers)
}

func validateHeaders(headers map[string]string) error {
	blocked := map[string]bool{"authorization": true, "host": true, "content-length": true, "cookie": true, "connection": true, "keep-alive": true, "proxy-authenticate": true, "proxy-authorization": true, "te": true, "trailer": true, "transfer-encoding": true, "upgrade": true}
	if len(headers) > MaxHeaderCount {
		return fmt.Errorf("at most %d custom headers are allowed", MaxHeaderCount)
	}
	headerBytes := 0
	for k, v := range headers {
		if len(k) > MaxHeaderNameBytes {
			return fmt.Errorf("header name must be at most %d bytes", MaxHeaderNameBytes)
		}
		if len(v) > MaxHeaderValueBytes {
			return fmt.Errorf("header %q value must be at most %d bytes", k, MaxHeaderValueBytes)
		}
		headerBytes += len(k) + len(v)
		if headerBytes > MaxHeaderBytes {
			return fmt.Errorf("custom headers must total at most %d bytes", MaxHeaderBytes)
		}
		trimmedName := strings.TrimSpace(k)
		ck := http.CanonicalHeaderKey(trimmedName)
		lower := strings.ToLower(ck)
		secretLooking := strings.Contains(lower, "api-key") || strings.Contains(lower, "token") || strings.Contains(lower, "secret")
		if k != trimmedName || ck == "" || !headerNameRE.MatchString(trimmedName) || blocked[lower] || secretLooking || !validHeaderValue(v) {
			return fmt.Errorf("header %q is not allowed", k)
		}
	}
	return nil
}

func ValidatePostProcessing(s PostProcessingSettings) error {
	if _, err := compatibility.Resolve(s.CompatibilityProfile, compatibility.PostProcessing); err != nil {
		return err
	}
	if err := validateTimeout("post-processing request", s.TimeoutSeconds, MinRequestTimeoutSeconds, MaxRequestTimeoutSeconds); err != nil {
		return err
	}
	if err := validatePostProcessingConnection(s.BaseURL, s.AllowInsecureHTTP, s.Model, true); err != nil {
		return err
	}
	if len(s.SystemPrompt) > MaxPromptBytes {
		return fmt.Errorf("post-processing system prompt must be at most %d bytes", MaxPromptBytes)
	}
	if len(s.Styling) > 32 || len(s.Structure) > 32 || len(s.Context) > 32 {
		return errors.New("post-processing profile controls must be at most 32 bytes")
	}
	switch s.Preset {
	case PostProcessingPresetGeneric:
		if strings.TrimSpace(s.SystemPrompt) == "" {
			return errors.New("a system instruction is required for the custom post-processing profile")
		}
	case PostProcessingPresetS1Mini:
		if !oneOf(s.Styling, s1MiniStylingValues...) {
			return errors.New("S1-mini styling is invalid")
		}
		if !oneOf(s.Structure, s1MiniStructureValues...) {
			return errors.New("S1-mini structure is invalid")
		}
		if !oneOf(s.Context, s1MiniContextValues...) {
			return errors.New("S1-mini context is invalid")
		}
	default:
		return errors.New("post-processing preset is invalid")
	}
	return nil
}

// ValidatePostProcessingConnection validates only endpoint and optional model
// values needed for a metadata probe.
func ValidatePostProcessingConnection(baseURL string, allowInsecureHTTP bool, model string) error {
	return validatePostProcessingConnection(baseURL, allowInsecureHTTP, model, false)
}

func validatePostProcessingConnection(baseURL string, allowInsecureHTTP bool, model string, requireModel bool) error {
	if err := validateBaseURL(baseURL, allowInsecureHTTP, "post-processing "); err != nil {
		return err
	}
	if (requireModel && strings.TrimSpace(model) == "") || len(model) > 200 {
		return errors.New("post-processing model is required and must be at most 200 characters")
	}
	return nil
}

func validateBaseURL(baseURL string, allowInsecureHTTP bool, prefix string) error {
	if len(baseURL) > MaxBaseURLBytes {
		return fmt.Errorf("%sbase URL must be at most %d bytes", prefix, MaxBaseURLBytes)
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("%sbase URL must be an HTTP or HTTPS URL without credentials, query, or fragment", prefix)
	}
	if u.Scheme == "http" && !allowInsecureHTTP {
		return fmt.Errorf("%sbase URL uses insecure HTTP; enable Allow insecure HTTP to continue", prefix)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("%sbase URL must use HTTPS unless insecure HTTP is explicitly enabled", prefix)
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func validHeaderValue(value string) bool {
	for i := 0; i < len(value); i++ {
		b := value[i]
		if b == '\t' || (b >= 0x20 && b != 0x7f) {
			continue
		}
		return false
	}
	return true
}

type LoadFailure struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type LoadReport struct {
	PreservedFieldCount int      `json:"preservedFieldCount"`
	PreservedFields     []string `json:"preservedFields"`
}

type loadError struct {
	failure LoadFailure
	cause   error
}

func (e *loadError) Error() string { return e.failure.Message }
func (e *loadError) Unwrap() error { return e.cause }

func LoadFailureFor(err error) LoadFailure {
	var loadErr *loadError
	if errors.As(err, &loadErr) {
		return loadErr.failure
	}
	return LoadFailure{Kind: "unavailable", Message: "The saved configuration could not be loaded."}
}

func newLoadError(kind, message string, cause error) error {
	return &loadError{failure: LoadFailure{Kind: kind, Message: message}, cause: cause}
}

type Store struct {
	Path string

	mu       sync.Mutex
	document map[string]json.RawMessage
	report   LoadReport
}

func NewStore() (*Store, error) {
	d, e := os.UserConfigDir()
	if e != nil {
		return nil, e
	}
	return &Store{Path: filepath.Join(d, "Freehand", "settings.json")}, nil
}
func (s *Store) Load() (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.document = nil
	s.report = LoadReport{}
	v := Default()
	f, e := os.Open(s.Path)
	if os.IsNotExist(e) {
		return v, nil
	}
	if e != nil {
		return Default(), newLoadError("unreadable", "The saved configuration could not be read.", e)
	}
	b, readErr := io.ReadAll(io.LimitReader(f, MaxSettingsFileBytes+1))
	closeErr := f.Close()
	if readErr != nil {
		return Default(), newLoadError("unreadable", "The saved configuration could not be read.", readErr)
	}
	if closeErr != nil {
		return Default(), newLoadError("unreadable", "The saved configuration could not be closed after reading.", closeErr)
	}
	if len(b) > MaxSettingsFileBytes {
		return Default(), newLoadError("too_large", fmt.Sprintf("The settings file exceeds the %d KiB safety limit.", MaxSettingsFileBytes/1024), nil)
	}
	d := json.NewDecoder(bytes.NewReader(b))
	if e = d.Decode(&v); e != nil {
		return Default(), newLoadError("invalid_json", jsonLoadFailureMessage(e), e)
	}
	var trailing any
	if e = d.Decode(&trailing); !errors.Is(e, io.EOF) {
		return Default(), newLoadError("invalid_json", "The settings file contains content after the configuration object.", e)
	}
	if e = Validate(v); e != nil {
		return Default(), newLoadError("invalid_values", "A saved setting is invalid: "+e.Error()+".", e)
	}
	var document map[string]json.RawMessage
	if e = json.Unmarshal(b, &document); e != nil || document == nil {
		return Default(), newLoadError("invalid_json", "The settings file must contain one JSON object.", e)
	}
	s.document = document
	s.report = loadReport(document, reflect.TypeOf(v))
	return v, nil
}

func (s *Store) LoadReport() LoadReport {
	s.mu.Lock()
	defer s.mu.Unlock()
	return LoadReport{
		PreservedFieldCount: s.report.PreservedFieldCount,
		PreservedFields:     append([]string(nil), s.report.PreservedFields...),
	}
}

func (s *Store) Save(v Settings) error {
	if e := Validate(v); e != nil {
		return e
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if e := os.MkdirAll(filepath.Dir(s.Path), 0700); e != nil {
		return e
	}
	currentBytes, e := json.Marshal(v)
	if e != nil {
		return e
	}
	var current map[string]json.RawMessage
	if e = json.Unmarshal(currentBytes, &current); e != nil {
		return e
	}
	merged := mergeSettingsDocument(cloneDocument(s.document), current, reflect.TypeOf(v))
	b, e := json.MarshalIndent(merged, "", "  ")
	if e != nil {
		return e
	}
	tmp, e := os.CreateTemp(filepath.Dir(s.Path), "settings-*.tmp")
	if e != nil {
		return e
	}
	name := tmp.Name()
	defer os.Remove(name)
	if e = tmp.Chmod(0600); e == nil {
		_, e = tmp.Write(b)
	}
	if e == nil {
		e = tmp.Sync()
	}
	if ce := tmp.Close(); e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	if e = os.Rename(name, s.Path); e != nil {
		return e
	}
	s.document = merged
	s.report = loadReport(merged, reflect.TypeOf(v))
	return nil
}

func jsonLoadFailureMessage(err error) string {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) && typeErr.Field != "" {
		return fmt.Sprintf("The saved value for %q has the wrong type.", typeErr.Field)
	}
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return fmt.Sprintf("The settings file contains invalid JSON near byte %d.", syntaxErr.Offset)
	}
	return "The settings file is not valid JSON."
}

func cloneDocument(document map[string]json.RawMessage) map[string]json.RawMessage {
	clone := make(map[string]json.RawMessage, len(document))
	for key, value := range document {
		clone[key] = append(json.RawMessage(nil), value...)
	}
	return clone
}

func mergeSettingsDocument(original, current map[string]json.RawMessage, valueType reflect.Type) map[string]json.RawMessage {
	if original == nil {
		original = make(map[string]json.RawMessage)
	}
	for index := 0; index < valueType.NumField(); index++ {
		field := valueType.Field(index)
		name := jsonFieldName(field)
		if name == "" {
			continue
		}
		currentValue, present := current[name]
		if !present {
			delete(original, name)
			continue
		}
		fieldType := field.Type
		if fieldType.Kind() == reflect.Struct {
			var originalChild, currentChild map[string]json.RawMessage
			if json.Unmarshal(original[name], &originalChild) == nil && json.Unmarshal(currentValue, &currentChild) == nil {
				child := mergeSettingsDocument(originalChild, currentChild, fieldType)
				if encoded, err := json.Marshal(child); err == nil {
					original[name] = encoded
					continue
				}
			}
		}
		original[name] = append(json.RawMessage(nil), currentValue...)
	}
	return original
}

func unknownFieldPaths(document map[string]json.RawMessage, valueType reflect.Type, prefix string) []string {
	known := make(map[string]reflect.Type)
	for index := 0; index < valueType.NumField(); index++ {
		field := valueType.Field(index)
		if name := jsonFieldName(field); name != "" {
			known[name] = field.Type
		}
	}
	paths := make([]string, 0)
	for name, raw := range document {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		fieldType, found := known[name]
		if !found {
			paths = append(paths, safeSettingsPath(path))
			continue
		}
		if fieldType.Kind() == reflect.Struct {
			var child map[string]json.RawMessage
			if json.Unmarshal(raw, &child) == nil {
				paths = append(paths, unknownFieldPaths(child, fieldType, path)...)
			}
		}
	}
	sort.Strings(paths)
	return paths
}

func loadReport(document map[string]json.RawMessage, valueType reflect.Type) LoadReport {
	paths := unknownFieldPaths(document, valueType, "")
	report := LoadReport{PreservedFieldCount: len(paths), PreservedFields: paths}
	if len(report.PreservedFields) > MaxReportedFields {
		report.PreservedFields = append([]string(nil), report.PreservedFields[:MaxReportedFields]...)
	}
	return report
}

func jsonFieldName(field reflect.StructField) string {
	tag := strings.Split(field.Tag.Get("json"), ",")[0]
	if tag == "-" {
		return ""
	}
	if tag != "" {
		return tag
	}
	return field.Name
}

func safeSettingsPath(path string) string {
	if len(path) > 128 {
		return "unrecognized setting"
	}
	for _, character := range path {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || character == '.' || character == '_' || character == '-' {
			continue
		}
		return "unrecognized setting"
	}
	return path
}
