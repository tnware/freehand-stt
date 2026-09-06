package storage

import (
	"context"
	"sort"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/storage/dbgen"
)

func boolean(v bool) int64 {
	if v {
		return 1
	}
	return 0
}
func writeSettings(ctx context.Context, q *dbgen.Queries, v config.Settings) error {
	if err := q.PutPreferences(ctx, dbgen.PutPreferencesParams{
		ToggleShortcut:          v.ToggleShortcut,
		ShowShortcut:            v.ShowShortcut,
		HoldShortcut:            v.HoldShortcut,
		MicrophoneID:            v.MicrophoneID,
		MaxDurationSeconds:      int64(v.MaxDurationSeconds),
		AutoInsert:              boolean(v.AutoInsert),
		StartWithWindows:        boolean(v.StartWithWindows),
		ShowWindowOnLaunch:      boolean(v.ShowWindowOnLaunch),
		CheckForUpdates:         boolean(v.CheckForUpdates),
		SetupCompleted:          boolean(v.SetupCompleted),
		UseMica:                 boolean(v.UseMica),
		AppearanceMode:          string(v.AppearanceMode),
		OverlayEnabled:          boolean(v.OverlayEnabled),
		OverlaySizePercent:      int64(v.OverlaySizePercent),
		OverlayOpacityPercent:   int64(v.OverlayOpacityPercent),
		OverlayTopOffset:        int64(v.OverlayTopOffset),
		OverlayGlowPercent:      int64(v.OverlayGlowPercent),
		OverlayLayout:           string(v.OverlayLayout),
		OverlayAnchor:           string(v.OverlayAnchor),
		OverlayVisibility:       string(v.OverlayVisibility),
		OverlayMotion:           string(v.OverlayMotion),
		OverlaySurface:          string(v.OverlaySurface),
		OverlayVisualizer:       string(v.OverlayVisualizer),
		HistoryEnabled:          boolean(v.HistoryEnabled),
		VadEnabled:              boolean(v.VADEnabled),
		VadMode:                 string(v.VADMode),
		VadActivitySilenceMs:    int64(v.VADActivitySilenceMS),
		SilenceTrimming:         boolean(v.SilenceTrimming),
		SpeechPaddingMs:         int64(v.SpeechPaddingMS),
		AutoStopEnabled:         boolean(v.AutoStopEnabled),
		AutoStopSilenceMs:       int64(v.AutoStopSilenceMS),
		AutoStopMinimumSpeechMs: int64(v.AutoStopMinimumSpeechMS),
		SilenceSplitting:        boolean(v.SilenceSplitting),
		SegmentSeconds:          int64(v.SegmentSeconds),
		SegmentSilenceMs:        int64(v.SegmentSilenceMS),
	}); err != nil {
		return err
	}
	if err := q.PutTranscription(ctx, dbgen.PutTranscriptionParams{
		ModelProfile:                            string(modelprofile.Effective(v.ModelProfile)),
		CompatibilityProfile:                    string(v.CompatibilityProfile),
		BaseUrl:                                 v.BaseURL,
		AllowInsecureHttp:                       boolean(v.AllowInsecureHTTP),
		AuthenticationMode:                      string(v.AuthenticationMode),
		Model:                                   v.Model,
		Language:                                v.Language,
		HealthPath:                              v.HealthPath,
		TranscriptionTimeoutSeconds:             int64(v.TranscriptionTimeoutSeconds),
		FileTranscriptionTimeoutSeconds:         int64(v.FileTranscriptionTimeoutSeconds),
		TranscriptionOptionsPrompt:              v.TranscriptionOptions.Prompt,
		TranscriptionOptionsHotwords:            v.TranscriptionOptions.Hotwords,
		TranscriptionOptionsTemperatureOverride: boolean(v.TranscriptionOptions.TemperatureOverride),
		TranscriptionOptionsTemperature:         v.TranscriptionOptions.Temperature,
	}); err != nil {
		return err
	}
	if err := q.PutCleanup(ctx, dbgen.PutCleanupParams{
		GenerationOptionsLimitOutputTokens: boolean(v.PostProcessing.GenerationOptions.LimitOutputTokens),
		GenerationOptionsMaxOutputTokens:   int64(v.PostProcessing.GenerationOptions.MaxOutputTokens),
		GenerationOptionsDisableReasoning:  boolean(v.PostProcessing.GenerationOptions.DisableReasoning),
		CompatibilityProfile:               string(v.PostProcessing.CompatibilityProfile),
		Enabled:                            boolean(v.PostProcessing.Enabled),
		BaseUrl:                            v.PostProcessing.BaseURL,
		AllowInsecureHttp:                  boolean(v.PostProcessing.AllowInsecureHTTP),
		Model:                              v.PostProcessing.Model,
		Preset:                             string(v.PostProcessing.Preset),
		SystemPrompt:                       v.PostProcessing.SystemPrompt,
		Styling:                            v.PostProcessing.Styling,
		Structure:                          v.PostProcessing.Structure,
		Context:                            v.PostProcessing.Context,
		TimeoutSeconds:                     int64(v.PostProcessing.TimeoutSeconds),
	}); err != nil {
		return err
	}
	if err := q.PutSpeech(ctx, dbgen.PutSpeechParams{
		ModelProfile:         string(modelprofile.Effective(v.TextToSpeech.ModelProfile)),
		CompatibilityProfile: string(v.TextToSpeech.CompatibilityProfile),
		Enabled:              boolean(v.TextToSpeech.Enabled),
		BaseUrl:              v.TextToSpeech.BaseURL,
		AllowInsecureHttp:    boolean(v.TextToSpeech.AllowInsecureHTTP),
		AuthenticationMode:   string(v.TextToSpeech.AuthenticationMode),
		Model:                v.TextToSpeech.Model,
		Voice:                v.TextToSpeech.Voice,
		Speed:                v.TextToSpeech.Speed,
		TimeoutSeconds:       int64(v.TextToSpeech.TimeoutSeconds),
	}); err != nil {
		return err
	}
	if err := q.ClearHeaders(ctx); err != nil {
		return err
	}
	names := make([]string, 0, len(v.Headers))
	for name := range v.Headers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := q.PutHeader(ctx, dbgen.PutHeaderParams{Name: name, Value: v.Headers[name]}); err != nil {
			return err
		}
	}
	return nil
}
func readSettings(ctx context.Context, q *dbgen.Queries) (config.Settings, error) {
	v := config.Default()
	if _, err := q.GetInitialization(ctx); err != nil {
		return v, err
	}
	rPreferences, err := q.GetPreferences(ctx)
	if err != nil {
		return v, err
	}
	v.ToggleShortcut = rPreferences.ToggleShortcut
	v.ShowShortcut = rPreferences.ShowShortcut
	v.HoldShortcut = rPreferences.HoldShortcut
	v.MicrophoneID = rPreferences.MicrophoneID
	v.MaxDurationSeconds = int(rPreferences.MaxDurationSeconds)
	v.AutoInsert = rPreferences.AutoInsert != 0
	v.StartWithWindows = rPreferences.StartWithWindows != 0
	v.ShowWindowOnLaunch = rPreferences.ShowWindowOnLaunch != 0
	v.CheckForUpdates = rPreferences.CheckForUpdates != 0
	v.SetupCompleted = rPreferences.SetupCompleted != 0
	v.UseMica = rPreferences.UseMica != 0
	v.AppearanceMode = config.AppearanceMode(rPreferences.AppearanceMode)
	v.OverlayEnabled = rPreferences.OverlayEnabled != 0
	v.OverlaySizePercent = int(rPreferences.OverlaySizePercent)
	v.OverlayOpacityPercent = int(rPreferences.OverlayOpacityPercent)
	v.OverlayTopOffset = int(rPreferences.OverlayTopOffset)
	v.OverlayGlowPercent = int(rPreferences.OverlayGlowPercent)
	v.OverlayLayout = config.OverlayLayout(rPreferences.OverlayLayout)
	v.OverlayAnchor = config.OverlayAnchor(rPreferences.OverlayAnchor)
	v.OverlayVisibility = config.OverlayVisibility(rPreferences.OverlayVisibility)
	v.OverlayMotion = config.OverlayMotion(rPreferences.OverlayMotion)
	v.OverlaySurface = config.OverlaySurface(rPreferences.OverlaySurface)
	v.OverlayVisualizer = config.OverlayVisualizer(rPreferences.OverlayVisualizer)
	v.HistoryEnabled = rPreferences.HistoryEnabled != 0
	v.VADEnabled = rPreferences.VadEnabled != 0
	v.VADMode = config.VADMode(rPreferences.VadMode)
	v.VADActivitySilenceMS = int(rPreferences.VadActivitySilenceMs)
	v.SilenceTrimming = rPreferences.SilenceTrimming != 0
	v.SpeechPaddingMS = int(rPreferences.SpeechPaddingMs)
	v.AutoStopEnabled = rPreferences.AutoStopEnabled != 0
	v.AutoStopSilenceMS = int(rPreferences.AutoStopSilenceMs)
	v.AutoStopMinimumSpeechMS = int(rPreferences.AutoStopMinimumSpeechMs)
	v.SilenceSplitting = rPreferences.SilenceSplitting != 0
	v.SegmentSeconds = int(rPreferences.SegmentSeconds)
	v.SegmentSilenceMS = int(rPreferences.SegmentSilenceMs)
	rTranscription, err := q.GetTranscription(ctx)
	if err != nil {
		return v, err
	}
	v.CompatibilityProfile = compatibility.ID(rTranscription.CompatibilityProfile)
	v.BaseURL = rTranscription.BaseUrl
	v.AllowInsecureHTTP = rTranscription.AllowInsecureHttp != 0
	v.AuthenticationMode = config.AuthenticationMode(rTranscription.AuthenticationMode)
	v.ModelProfile = modelprofile.ID(rTranscription.ModelProfile)
	v.Model = rTranscription.Model
	v.Language = rTranscription.Language
	v.HealthPath = rTranscription.HealthPath
	v.TranscriptionTimeoutSeconds = int(rTranscription.TranscriptionTimeoutSeconds)
	v.FileTranscriptionTimeoutSeconds = int(rTranscription.FileTranscriptionTimeoutSeconds)
	v.TranscriptionOptions.Prompt = rTranscription.TranscriptionOptionsPrompt
	v.TranscriptionOptions.Hotwords = rTranscription.TranscriptionOptionsHotwords
	v.TranscriptionOptions.TemperatureOverride = rTranscription.TranscriptionOptionsTemperatureOverride != 0
	v.TranscriptionOptions.Temperature = rTranscription.TranscriptionOptionsTemperature
	rCleanup, err := q.GetCleanup(ctx)
	if err != nil {
		return v, err
	}
	v.PostProcessing.GenerationOptions.LimitOutputTokens = rCleanup.GenerationOptionsLimitOutputTokens != 0
	v.PostProcessing.GenerationOptions.MaxOutputTokens = int(rCleanup.GenerationOptionsMaxOutputTokens)
	v.PostProcessing.GenerationOptions.DisableReasoning = rCleanup.GenerationOptionsDisableReasoning != 0
	v.PostProcessing.CompatibilityProfile = compatibility.ID(rCleanup.CompatibilityProfile)
	v.PostProcessing.Enabled = rCleanup.Enabled != 0
	v.PostProcessing.BaseURL = rCleanup.BaseUrl
	v.PostProcessing.AllowInsecureHTTP = rCleanup.AllowInsecureHttp != 0
	v.PostProcessing.Model = rCleanup.Model
	v.PostProcessing.Preset = config.PostProcessingPreset(rCleanup.Preset)
	v.PostProcessing.SystemPrompt = rCleanup.SystemPrompt
	v.PostProcessing.Styling = rCleanup.Styling
	v.PostProcessing.Structure = rCleanup.Structure
	v.PostProcessing.Context = rCleanup.Context
	v.PostProcessing.TimeoutSeconds = int(rCleanup.TimeoutSeconds)
	rSpeech, err := q.GetSpeech(ctx)
	if err != nil {
		return v, err
	}
	v.TextToSpeech.CompatibilityProfile = compatibility.ID(rSpeech.CompatibilityProfile)
	v.TextToSpeech.Enabled = rSpeech.Enabled != 0
	v.TextToSpeech.BaseURL = rSpeech.BaseUrl
	v.TextToSpeech.AllowInsecureHTTP = rSpeech.AllowInsecureHttp != 0
	v.TextToSpeech.AuthenticationMode = config.AuthenticationMode(rSpeech.AuthenticationMode)
	v.TextToSpeech.ModelProfile = modelprofile.ID(rSpeech.ModelProfile)
	v.TextToSpeech.Model = rSpeech.Model
	v.TextToSpeech.Voice = rSpeech.Voice
	v.TextToSpeech.Speed = rSpeech.Speed
	v.TextToSpeech.TimeoutSeconds = int(rSpeech.TimeoutSeconds)
	headers, err := q.GetHeaders(ctx)
	if err != nil {
		return v, err
	}
	v.Headers = map[string]string{}
	for _, h := range headers {
		v.Headers[h.Name] = h.Value
	}
	return v, nil
}
