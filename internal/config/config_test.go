package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestValidation(t *testing.T) {
	s := Default()
	if e := Validate(s); e != nil {
		t.Fatal(e)
	}
	s.Headers["Authorization"] = "x"
	if Validate(s) == nil {
		t.Fatal("secret header accepted")
	}
	s = Default()
	s.BaseURL = "http://x"
	s.Model = "speech/stt"
	if Validate(s) == nil {
		t.Fatal("http accepted without explicit opt-in")
	}
	s.AllowInsecureHTTP = true
	if err := Validate(s); err != nil {
		t.Fatalf("explicitly opted-in HTTP endpoint was rejected: %v", err)
	}
}

func TestHistoryIsOptionalAndValid(t *testing.T) {
	settings := Default()
	if settings.HistoryEnabled {
		t.Fatal("history must remain disabled by default")
	}
	settings.HistoryEnabled = true
	if err := Validate(settings); err != nil {
		t.Fatalf("opt-in history was rejected: %v", err)
	}
}

func TestUpdateChecksAreEnabledByDefault(t *testing.T) {
	if !Default().CheckForUpdates {
		t.Fatal("automatic update checks must be enabled by default")
	}
}

func TestSetupReviewIsOneTimePersistedState(t *testing.T) {
	settings := Default()
	if settings.SetupCompleted {
		t.Fatal("first launch must require the setup review")
	}
	settings.BaseURL = "https://example.test/v1"
	settings.Model = "speech/stt"
	settings.SetupCompleted = true
	store := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	if err := store.Save(settings); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.SetupCompleted {
		t.Fatal("completed setup review was not preserved")
	}
}

func TestUnconfiguredDefaultsDoNotSelectAnEndpointOrCredentialMode(t *testing.T) {
	settings := Default()
	if settings.BaseURL != "" || settings.Model != "" {
		t.Fatalf("default connection = %q model %q", settings.BaseURL, settings.Model)
	}
	if settings.AuthenticationMode != AuthenticationModeNone {
		t.Fatalf("default authentication mode = %q", settings.AuthenticationMode)
	}
	if err := Validate(settings); err != nil {
		t.Fatalf("safe first-run settings were rejected: %v", err)
	}
	settings.AuthenticationMode = AuthenticationModeAPIKey
	if err := Validate(settings); err != nil {
		t.Fatalf("explicit API-key mode was rejected during setup: %v", err)
	}
	settings.AuthenticationMode = "automatic"
	if err := Validate(settings); err == nil {
		t.Fatal("unknown authentication mode was accepted")
	}
}

func TestCompletedSetupRequiresAnExplicitConnection(t *testing.T) {
	settings := Default()
	settings.SetupCompleted = true
	if err := Validate(settings); err == nil {
		t.Fatal("completed setup accepted an empty STT connection")
	}
	settings.BaseURL = "https://example.test/v1"
	settings.Model = "speech/stt"
	if err := Validate(settings); err != nil {
		t.Fatalf("explicit STT connection was rejected: %v", err)
	}
}

func TestUnknownSettingsAreLoadedAndPreservedForNewerVersions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"unknownFutureSetting":{"enabled":true},"postProcessing":{"futureControl":"kept"},"showWindowOnLaunch":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &Store{Path: path}
	settings, err := store.Load()
	if err != nil {
		t.Fatalf("forward-compatible settings were rejected: %v", err)
	}
	if settings.ShowWindowOnLaunch {
		t.Fatal("recognized setting was not loaded")
	}
	report := store.LoadReport()
	if report.PreservedFieldCount != 2 {
		t.Fatalf("preserved field count = %d", report.PreservedFieldCount)
	}
	if strings.Join(report.PreservedFields, ",") != "postProcessing.futureControl,unknownFutureSetting" {
		t.Fatalf("preserved fields = %v", report.PreservedFields)
	}
	settings.ShowWindowOnLaunch = true
	if err := store.Save(settings); err != nil {
		t.Fatal(err)
	}
	var saved map[string]json.RawMessage
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	var futureSetting struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal(saved["unknownFutureSetting"], &futureSetting); err != nil || !futureSetting.Enabled {
		t.Fatalf("unknown top-level setting was changed: %s (%v)", saved["unknownFutureSetting"], err)
	}
	var processing map[string]json.RawMessage
	if err := json.Unmarshal(saved["postProcessing"], &processing); err != nil {
		t.Fatal(err)
	}
	if string(processing["futureControl"]) != `"kept"` {
		t.Fatalf("unknown nested setting was changed: %s", processing["futureControl"])
	}
}

func TestInvalidSettingsReturnSafeRecoveryDefaultsAndReason(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	original := []byte(`{"appearanceMode":"ultraviolet"}`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	settings, err := (&Store{Path: path}).Load()
	if err == nil {
		t.Fatal("invalid known setting was accepted")
	}
	failure := LoadFailureFor(err)
	if failure.Kind != "invalid_values" || !strings.Contains(failure.Message, "appearance mode") {
		t.Fatalf("load failure = %#v", failure)
	}
	if !reflect.DeepEqual(settings, Default()) {
		t.Fatalf("unsafe recovery settings = %#v", settings)
	}
	after, readErr := os.ReadFile(path)
	if readErr != nil || !reflect.DeepEqual(after, original) {
		t.Fatalf("failed load changed the original file: %q (%v)", after, readErr)
	}
}

func TestFocusedConnectionValidationAllowsDiscoveryWithoutUnrelatedSettings(t *testing.T) {
	settings := Default()
	settings.BaseURL = "https://example.test/v1"
	settings.Model = ""
	settings.ToggleShortcut = "not a shortcut"
	settings.VADMode = "not-a-mode"
	settings.PostProcessing.Enabled = true
	settings.PostProcessing.Preset = "not-a-preset"

	if err := ValidateSTTConnection(
		settings.BaseURL,
		settings.AllowInsecureHTTP,
		settings.AuthenticationMode,
		settings.Model,
		settings.HealthPath,
		settings.Headers,
	); err != nil {
		t.Fatalf("focused STT connection values were rejected: %v", err)
	}
	if err := Validate(settings); err == nil {
		t.Fatal("invalid full settings were accepted")
	}
}

func TestFocusedPostProcessingConnectionValidationAllowsModelDiscovery(t *testing.T) {
	settings := Default().PostProcessing
	settings.Model = ""
	settings.Preset = "not-a-preset"
	settings.SystemPrompt = ""

	if err := ValidatePostProcessingConnection(settings.BaseURL, settings.AllowInsecureHTTP, settings.Model); err == nil {
		t.Fatal("insecure post-processing endpoint was accepted without opt-in")
	}
	settings.AllowInsecureHTTP = true
	if err := ValidatePostProcessingConnection(settings.BaseURL, settings.AllowInsecureHTTP, settings.Model); err != nil {
		t.Fatalf("focused post-processing connection values were rejected: %v", err)
	}
	if err := ValidatePostProcessing(settings); err == nil {
		t.Fatal("invalid full post-processing settings were accepted")
	}
}

func TestTextToSpeechValidationIsDormantUntilEnabled(t *testing.T) {
	settings := Default()
	if err := Validate(settings); err != nil {
		t.Fatalf("default settings: %v", err)
	}
	settings.TextToSpeech.Enabled = true
	if err := Validate(settings); err == nil {
		t.Fatal("enabled speech playback accepted an empty connection")
	}
	settings.TextToSpeech = TextToSpeechSettings{
		Enabled: true, BaseURL: "http://127.0.0.1:8000/v1", AllowInsecureHTTP: true,
		AuthenticationMode: AuthenticationModeNone, Model: "tts-1", Voice: "alloy", Speed: 1,
		TimeoutSeconds: DefaultTextToSpeechTimeoutSeconds,
	}
	if err := Validate(settings); err != nil {
		t.Fatalf("configured speech playback: %v", err)
	}
}

func TestTextToSpeechConnectionValidationAllowsModelDiscovery(t *testing.T) {
	if err := ValidateTextToSpeechConnection("http://127.0.0.1:8000/v1", true, AuthenticationModeNone, ""); err != nil {
		t.Fatalf("model discovery: %v", err)
	}
	if err := ValidateTextToSpeechConnection("http://127.0.0.1:8000/v1", false, AuthenticationModeNone, ""); err == nil {
		t.Fatal("insecure HTTP was accepted without opt-in")
	}
}

func TestShortcutAndDurationValidation(t *testing.T) {
	for _, mutate := range []func(*Settings){
		func(s *Settings) { s.HoldShortcut = s.ToggleShortcut },
		func(s *Settings) { s.HoldShortcut = "Ctrl+F12" },
		func(s *Settings) { s.MaxDurationSeconds = 263 },
		func(s *Settings) { s.MicrophoneID = string(make([]byte, 1025)) },
	} {
		s := Default()
		mutate(&s)
		if Validate(s) == nil {
			t.Fatalf("invalid settings accepted: %+v", s)
		}
	}
}

func TestInferenceTimeoutValidation(t *testing.T) {
	for name, mutate := range map[string]func(*Settings){
		"recording below minimum": func(s *Settings) { s.TranscriptionTimeoutSeconds = MinRequestTimeoutSeconds - 1 },
		"recording above maximum": func(s *Settings) { s.TranscriptionTimeoutSeconds = MaxRequestTimeoutSeconds + 1 },
		"file below minimum":      func(s *Settings) { s.FileTranscriptionTimeoutSeconds = MinFileTranscriptionTimeoutSeconds - 1 },
		"file above maximum":      func(s *Settings) { s.FileTranscriptionTimeoutSeconds = MaxFileTranscriptionTimeoutSeconds + 1 },
		"processing below minimum": func(s *Settings) {
			s.PostProcessing.TimeoutSeconds = MinRequestTimeoutSeconds - 1
		},
		"speech above maximum": func(s *Settings) {
			s.TextToSpeech.TimeoutSeconds = MaxRequestTimeoutSeconds + 1
		},
	} {
		t.Run(name, func(t *testing.T) {
			settings := Default()
			mutate(&settings)
			if err := Validate(settings); err == nil {
				t.Fatal("invalid inference timeout was accepted")
			}
		})
	}
}

func TestSilenceSplittingExtendsTotalDurationWithinBounds(t *testing.T) {
	settings := Default()
	settings.MaxDurationSeconds = 600
	if err := Validate(settings); err == nil {
		t.Fatal("long single-request recording was accepted")
	}
	settings.SilenceSplitting = true
	if err := Validate(settings); err != nil {
		t.Fatalf("segmented long recording was rejected: %v", err)
	}
	settings.MaxDurationSeconds = 3601
	if err := Validate(settings); err == nil {
		t.Fatal("recording beyond the segmented safety limit was accepted")
	}
}

func TestSilenceSplittingControlsAreBounded(t *testing.T) {
	for _, mutate := range []func(*Settings){
		func(s *Settings) { s.SegmentSeconds = 14 },
		func(s *Settings) { s.SegmentSeconds = 181 },
		func(s *Settings) { s.SegmentSilenceMS = 199 },
		func(s *Settings) { s.SegmentSilenceMS = 3001 },
	} {
		settings := Default()
		mutate(&settings)
		if err := Validate(settings); err == nil {
			t.Fatalf("invalid segmentation settings accepted: %+v", settings)
		}
	}
}

func TestVoiceActivityDetectionSettingsAreConnectedAndBounded(t *testing.T) {
	for _, mode := range []VADMode{VADModeQuality, VADModeLowBitrate, VADModeAggressive, VADModeVeryAggressive} {
		settings := Default()
		settings.VADMode = mode
		if err := Validate(settings); err != nil {
			t.Fatalf("valid VAD mode %q was rejected: %v", mode, err)
		}
	}
	settings := Default()
	settings.VADMode = "invented"
	if err := Validate(settings); err == nil {
		t.Fatal("unknown VAD mode was accepted")
	}
	settings = Default()
	settings.VADEnabled = false
	settings.SilenceSplitting = true
	if err := Validate(settings); err == nil {
		t.Fatal("silence splitting was accepted while VAD was disabled")
	}
	for name, mutate := range map[string]func(*Settings){
		"activity debounce below minimum": func(s *Settings) { s.VADActivitySilenceMS = MinVADActivitySilenceMS - 1 },
		"activity debounce above maximum": func(s *Settings) { s.VADActivitySilenceMS = MaxVADActivitySilenceMS + 1 },
		"speech padding below minimum":    func(s *Settings) { s.SpeechPaddingMS = MinSpeechPaddingMS - 1 },
		"speech padding above maximum":    func(s *Settings) { s.SpeechPaddingMS = MaxSpeechPaddingMS + 1 },
		"auto-stop silence below minimum": func(s *Settings) { s.AutoStopSilenceMS = MinAutoStopSilenceMS - 1 },
		"auto-stop silence above maximum": func(s *Settings) { s.AutoStopSilenceMS = MaxAutoStopSilenceMS + 1 },
		"auto-stop speech below minimum":  func(s *Settings) { s.AutoStopMinimumSpeechMS = MinAutoStopSpeechMS - 1 },
		"auto-stop speech above maximum":  func(s *Settings) { s.AutoStopMinimumSpeechMS = MaxAutoStopSpeechMS + 1 },
	} {
		t.Run(name, func(t *testing.T) {
			settings := Default()
			mutate(&settings)
			if err := Validate(settings); err == nil {
				t.Fatalf("invalid VAD settings accepted: %+v", settings)
			}
		})
	}
	for name, enable := range map[string]func(*Settings){
		"silence trimming": func(s *Settings) { s.SilenceTrimming = true },
		"automatic stop":   func(s *Settings) { s.AutoStopEnabled = true },
	} {
		t.Run(name+" requires VAD", func(t *testing.T) {
			settings := Default()
			settings.VADEnabled = false
			enable(&settings)
			if err := Validate(settings); err == nil {
				t.Fatalf("%s was accepted while VAD was disabled", name)
			}
		})
	}
	settings = Default()
	settings.AutoStopEnabled = true
	settings.AutoStopSilenceMS = 500
	settings.VADActivitySilenceMS = 600
	if err := Validate(settings); err == nil {
		t.Fatal("automatic stop was accepted before the stabilized silence indicator could activate")
	}
}

func TestSettingsSizeBounds(t *testing.T) {
	tests := map[string]func(*Settings){
		"base URL": func(s *Settings) {
			s.BaseURL = "https://example.com/" + strings.Repeat("a", MaxBaseURLBytes-len("https://example.com/")+1)
		},
		"health path": func(s *Settings) {
			s.HealthPath = "/" + strings.Repeat("a", MaxHealthPathBytes)
		},
		"header count": func(s *Settings) {
			for i := 0; i <= MaxHeaderCount; i++ {
				s.Headers["X-Setting-"+strconv.Itoa(i)] = "value"
			}
		},
		"header name": func(s *Settings) {
			s.Headers[strings.Repeat("X", MaxHeaderNameBytes+1)] = "value"
		},
		"header value": func(s *Settings) {
			s.Headers["X-Setting"] = strings.Repeat("v", MaxHeaderValueBytes+1)
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			s := Default()
			mutate(&s)
			if err := Validate(s); err == nil {
				t.Fatalf("oversized %s accepted", name)
			}
		})
	}
}

func TestSettingsSizeBoundariesAreInclusive(t *testing.T) {
	s := Default()
	prefix := "https://example.com/"
	s.BaseURL = prefix + strings.Repeat("a", MaxBaseURLBytes-len(prefix))
	s.Model = "speech/stt"
	s.HealthPath = "/" + strings.Repeat("a", MaxHealthPathBytes-1)
	s.Headers[strings.Repeat("X", MaxHeaderNameBytes)] = strings.Repeat("v", MaxHeaderValueBytes)
	if err := Validate(s); err != nil {
		t.Fatalf("settings at size boundaries rejected: %v", err)
	}
}

func TestAggregateHeaderSizeBound(t *testing.T) {
	s := Default()
	for i := 0; i < 4; i++ {
		s.Headers["X"+strconv.Itoa(i)] = strings.Repeat("v", (MaxHeaderBytes-8)/4)
	}
	if err := Validate(s); err != nil {
		t.Fatalf("headers at aggregate boundary rejected: %v", err)
	}
	s.Headers["X0"] += "v"
	if err := Validate(s); err == nil {
		t.Fatal("headers above aggregate boundary accepted")
	}
}

func TestHeaderValueControlBytes(t *testing.T) {
	for _, value := range []string{"value\x00", "value\x01", "value\x7f", "value\r", "value\n"} {
		s := Default()
		s.Headers["X-Setting"] = value
		if err := Validate(s); err == nil {
			t.Fatalf("header control value %q accepted", value)
		}
	}
	s := Default()
	s.Headers["X-Setting"] = "one\ttwo"
	if err := Validate(s); err != nil {
		t.Fatalf("valid horizontal tab rejected: %v", err)
	}
}

func TestHeaderNameWhitespaceIsRejected(t *testing.T) {
	for _, name := range []string{" X-Setting", "X-Setting ", "X-Setting\t"} {
		s := Default()
		s.Headers[name] = "value"
		if err := Validate(s); err == nil {
			t.Fatalf("header name %q with surrounding whitespace accepted", name)
		}
	}
}

// Hold-to-talk runs on the low-level keyboard hook and may be modifiers only.
// The toggle and show shortcuts go through RegisterHotKey, which needs a
// virtual-key code, so the same value has to be refused for them.
func TestModifierOnlyShortcutsAreHoldOnly(t *testing.T) {
	hold := Default()
	hold.HoldShortcut = "Ctrl+Meta"
	if err := Validate(hold); err != nil {
		t.Fatalf("hold-to-talk rejected a modifier-only chord: %v", err)
	}

	single := Default()
	single.HoldShortcut = "Ctrl"
	if err := Validate(single); err == nil {
		t.Fatal("a single modifier was accepted; it would arm on ordinary typing")
	}

	for name, apply := range map[string]func(*Settings){
		"toggle": func(s *Settings) { s.ToggleShortcut = "Ctrl+Meta" },
		"show":   func(s *Settings) { s.ShowShortcut = "Ctrl+Meta" },
	} {
		settings := Default()
		apply(&settings)
		if err := Validate(settings); err == nil {
			t.Fatalf("%s accepted a modifier-only chord it cannot register", name)
		}
	}

	// Ordinary chords must keep working for hold-to-talk.
	ordinary := Default()
	ordinary.HoldShortcut = "Ctrl+Alt+D"
	if err := Validate(ordinary); err != nil {
		t.Fatalf("hold-to-talk rejected an ordinary chord: %v", err)
	}
}

func TestDedicatedFunctionKeysAndWindowsAliasesAreValidShortcuts(t *testing.T) {
	settings := Default()
	settings.ToggleShortcut = "F13"
	settings.ShowShortcut = "Win+F24"
	settings.HoldShortcut = "Control+Command"
	if err := Validate(settings); err != nil {
		t.Fatalf("supported shortcut policy was rejected: %v", err)
	}
}

func TestLoadPreservesDefaultWindowVisibilityForSparseFiles(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	settings, err := (&Store{Path: path}).Load()
	if err != nil {
		t.Fatal(err)
	}
	if !settings.ShowWindowOnLaunch {
		t.Fatal("a sparse settings file silently changed normal launches to tray-only")
	}
	if settings.AuthenticationMode != AuthenticationModeNone {
		t.Fatalf("sparse settings authentication mode = %q", settings.AuthenticationMode)
	}
	if settings.TranscriptionTimeoutSeconds != DefaultTranscriptionTimeoutSeconds ||
		settings.FileTranscriptionTimeoutSeconds != DefaultFileTranscriptionTimeoutSeconds ||
		settings.PostProcessing.TimeoutSeconds != DefaultPostProcessingTimeoutSeconds ||
		settings.TextToSpeech.TimeoutSeconds != DefaultTextToSpeechTimeoutSeconds {
		t.Fatalf("sparse settings request budgets = recording %d file %d processing %d speech %d", settings.TranscriptionTimeoutSeconds, settings.FileTranscriptionTimeoutSeconds, settings.PostProcessing.TimeoutSeconds, settings.TextToSpeech.TimeoutSeconds)
	}
	if settings.UseMica {
		t.Fatal("a sparse settings file silently enabled Mica")
	}
	if !settings.OverlayEnabled {
		t.Fatal("a sparse settings file silently disabled the status overlay")
	}
	if settings.OverlaySizePercent != 100 || settings.OverlayOpacityPercent != 100 || settings.OverlayTopOffset != 18 || settings.OverlayGlowPercent != 100 {
		t.Fatalf("sparse settings overlay appearance = size %d opacity %d offset %d glow %d", settings.OverlaySizePercent, settings.OverlayOpacityPercent, settings.OverlayTopOffset, settings.OverlayGlowPercent)
	}
	if settings.OverlayLayout != OverlayLayoutCapsule || settings.OverlayAnchor != OverlayAnchorTopCenter || settings.OverlayVisibility != OverlayVisibilityAll ||
		settings.OverlayMotion != OverlayMotionSystem || settings.OverlaySurface != OverlaySurfaceGlass || settings.OverlayVisualizer != OverlayVisualizerBars {
		t.Fatalf("sparse settings overlay presentation = %+v", settings.OverlayPreferences())
	}
}

func TestOverlayAppearanceValidation(t *testing.T) {
	tests := []struct {
		name  string
		apply func(*Settings)
	}{
		{name: "size below minimum", apply: func(s *Settings) { s.OverlaySizePercent = MinOverlaySizePercent - 1 }},
		{name: "size above maximum", apply: func(s *Settings) { s.OverlaySizePercent = MaxOverlaySizePercent + 1 }},
		{name: "opacity below minimum", apply: func(s *Settings) { s.OverlayOpacityPercent = MinOverlayOpacityPercent - 1 }},
		{name: "opacity above maximum", apply: func(s *Settings) { s.OverlayOpacityPercent = MaxOverlayOpacityPercent + 1 }},
		{name: "offset below minimum", apply: func(s *Settings) { s.OverlayTopOffset = MinOverlayTopOffset - 1 }},
		{name: "offset above maximum", apply: func(s *Settings) { s.OverlayTopOffset = MaxOverlayTopOffset + 1 }},
		{name: "glow below minimum", apply: func(s *Settings) { s.OverlayGlowPercent = MinOverlayGlowPercent - 1 }},
		{name: "glow above maximum", apply: func(s *Settings) { s.OverlayGlowPercent = MaxOverlayGlowPercent + 1 }},
		{name: "unknown layout", apply: func(s *Settings) { s.OverlayLayout = "banner" }},
		{name: "unknown anchor", apply: func(s *Settings) { s.OverlayAnchor = "middle" }},
		{name: "unknown visibility", apply: func(s *Settings) { s.OverlayVisibility = "sometimes" }},
		{name: "unknown motion", apply: func(s *Settings) { s.OverlayMotion = "bounce" }},
		{name: "unknown surface", apply: func(s *Settings) { s.OverlaySurface = "neon" }},
		{name: "unknown visualizer", apply: func(s *Settings) { s.OverlayVisualizer = "spectrum" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			settings := Default()
			test.apply(&settings)
			if err := Validate(settings); err == nil {
				t.Fatal("invalid overlay appearance was accepted")
			}
		})
	}
}

func TestAppearanceModeValidationAndMicaPolicy(t *testing.T) {
	for _, mode := range []AppearanceMode{AppearanceModeSystem, AppearanceModeLight, AppearanceModeDark} {
		settings := Default()
		settings.AppearanceMode = mode
		if err := Validate(settings); err != nil {
			t.Fatalf("valid appearance mode %q rejected: %v", mode, err)
		}
		if got := settings.EffectiveAppearanceMode(); got != mode {
			t.Fatalf("solid appearance mode = %q, want %q", got, mode)
		}
		settings.UseMica = true
		if got := settings.EffectiveAppearanceMode(); got != AppearanceModeSystem {
			t.Fatalf("Mica appearance mode = %q, want system", got)
		}
	}

	settings := Default()
	settings.AppearanceMode = "sepia"
	if err := Validate(settings); err == nil {
		t.Fatal("invalid appearance mode was accepted")
	}
}

func TestSaveUsesWindowLaunchKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := (&Store{Path: path}).Save(Default()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"showWindowOnLaunch"`) {
		t.Fatal("saved settings omitted showWindowOnLaunch")
	}
	if !strings.Contains(string(data), `"overlayEnabled"`) {
		t.Fatal("saved settings omitted overlayEnabled")
	}
	if !strings.Contains(string(data), `"appearanceMode"`) {
		t.Fatal("saved settings omitted appearanceMode")
	}
	for _, key := range []string{
		`"overlaySizePercent"`, `"overlayOpacityPercent"`, `"overlayTopOffset"`, `"overlayGlowPercent"`,
		`"overlayLayout"`, `"overlayAnchor"`, `"overlayVisibility"`, `"overlayMotion"`, `"overlaySurface"`, `"overlayVisualizer"`,
	} {
		if !strings.Contains(string(data), key) {
			t.Fatalf("saved settings omitted %s", key)
		}
	}
	if strings.Contains(string(data), `"showSettingsOnLaunch"`) {
		t.Fatal("saved settings retained the obsolete launch key")
	}
}

func TestPostProcessingRequiresExplicitHTTPOptInWhenEnabled(t *testing.T) {
	settings := Default()
	settings.PostProcessing.Enabled = true
	settings.PostProcessing.Model = "local-model"
	if err := Validate(settings); err == nil || !strings.Contains(err.Error(), "post-processing base URL uses insecure HTTP") {
		t.Fatalf("HTTP post-processing without opt-in error = %v", err)
	}
	settings.PostProcessing.AllowInsecureHTTP = true
	if err := Validate(settings); err != nil {
		t.Fatalf("explicit local HTTP opt-in rejected: %v", err)
	}
}

func TestS1MiniProfileRejectsUnknownControlValues(t *testing.T) {
	settings := Default()
	settings.PostProcessing.Enabled = true
	settings.PostProcessing.BaseURL = "https://processor.example/v1"
	settings.PostProcessing.Model = "s1-mini"
	settings.PostProcessing.Preset = PostProcessingPresetS1Mini
	settings.PostProcessing.Styling = "invented"
	if err := Validate(settings); err == nil || !strings.Contains(err.Error(), "S1-mini styling") {
		t.Fatalf("invalid S1-mini control error = %v", err)
	}
}
