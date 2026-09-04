package history

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/inference"
	"github.com/tnware/freehand-stt/internal/insertion"
)

const (
	MaxHistoryEntries  = 20
	MaxHistoryBytes    = 2 << 20
	MaxHistorySegments = 128

	historyBudgetMessage   = "Processed transcript was not retained because the history memory limit was reached. The raw transcript was kept."
	historyBudgetErrorKind = "history_budget"
)

type HistoryOutcome string
type HistoryProcessingStatus string
type HistoryTextVersion string
type HistorySource string
type HistoryResponseMode string

const (
	HistoryInserted     HistoryOutcome = "inserted"
	HistoryCopyRequired HistoryOutcome = "copy-required"
	HistoryFailed       HistoryOutcome = "failed"
	HistoryTranscribed  HistoryOutcome = "transcribed"
	HistoryCancelled    HistoryOutcome = "cancelled"
)

const (
	HistorySourceVoice     HistorySource = "voice"
	HistorySourceAudioFile HistorySource = "audio-file"
)

const (
	HistoryResponseCompleted HistoryResponseMode = "completed"
	HistoryResponseStreamed  HistoryResponseMode = "streamed"
)

type HistorySegmentDetails struct {
	Number              int    `json:"number"`
	AudioMilliseconds   int64  `json:"audioMilliseconds"`
	Boundary            string `json:"boundary"`
	RequestMilliseconds int64  `json:"requestMilliseconds"`
	CharacterCount      int    `json:"characterCount"`
}

type HistoryUsageDetails struct {
	Type                  string   `json:"type,omitempty"`
	InputTokens           *int64   `json:"inputTokens,omitempty"`
	OutputTokens          *int64   `json:"outputTokens,omitempty"`
	TotalTokens           *int64   `json:"totalTokens,omitempty"`
	AudioInputTokens      *int64   `json:"audioInputTokens,omitempty"`
	TextInputTokens       *int64   `json:"textInputTokens,omitempty"`
	CachedInputTokens     *int64   `json:"cachedInputTokens,omitempty"`
	CacheWriteTokens      *int64   `json:"cacheWriteTokens,omitempty"`
	ReasoningOutputTokens *int64   `json:"reasoningOutputTokens,omitempty"`
	AudioSeconds          *float64 `json:"audioSeconds,omitempty"`
	ReportedCost          *float64 `json:"reportedCost,omitempty"`
	UpstreamCost          *float64 `json:"upstreamCost,omitempty"`
}

type HistoryPerformanceDetails struct {
	PromptTokens                   *int64   `json:"promptTokens,omitempty"`
	PromptMilliseconds             *float64 `json:"promptMilliseconds,omitempty"`
	PromptMillisecondsPerToken     *float64 `json:"promptMillisecondsPerToken,omitempty"`
	PromptTokensPerSecond          *float64 `json:"promptTokensPerSecond,omitempty"`
	GeneratedTokens                *int64   `json:"generatedTokens,omitempty"`
	GenerationMilliseconds         *float64 `json:"generationMilliseconds,omitempty"`
	GenerationMillisecondsPerToken *float64 `json:"generationMillisecondsPerToken,omitempty"`
	GenerationTokensPerSecond      *float64 `json:"generationTokensPerSecond,omitempty"`
	CachedPromptTokens             *int64   `json:"cachedPromptTokens,omitempty"`
}

type HistoryResponseDetails struct {
	RequestID              string                    `json:"requestId,omitempty"`
	ResponseID             string                    `json:"responseId,omitempty"`
	EffectiveModel         string                    `json:"effectiveModel,omitempty"`
	Provider               string                    `json:"provider,omitempty"`
	FinishReason           string                    `json:"finishReason,omitempty"`
	ServiceTier            string                    `json:"serviceTier,omitempty"`
	SystemFingerprint      string                    `json:"systemFingerprint,omitempty"`
	CreatedAtUnix          *int64                    `json:"createdAtUnix,omitempty"`
	DetectedLanguages      []string                  `json:"detectedLanguages,omitempty"`
	ServerAudioSeconds     *float64                  `json:"serverAudioSeconds,omitempty"`
	Usage                  HistoryUsageDetails       `json:"usage"`
	Performance            HistoryPerformanceDetails `json:"performance"`
	RequestCount           int                       `json:"requestCount,omitempty"`
	UsageReportCount       int                       `json:"usageReportCount,omitempty"`
	CostReportCount        int                       `json:"costReportCount,omitempty"`
	PerformanceReportCount int                       `json:"performanceReportCount,omitempty"`
}

type HistoryProcessingDetails struct {
	Requested           bool                    `json:"requested"`
	Server              string                  `json:"server,omitempty"`
	Model               string                  `json:"model,omitempty"`
	Preset              string                  `json:"preset,omitempty"`
	StartedAt           time.Time               `json:"startedAt,omitempty"`
	CompletedAt         time.Time               `json:"completedAt,omitempty"`
	ElapsedMilliseconds int64                   `json:"elapsedMilliseconds,omitempty"`
	TimeoutSeconds      int                     `json:"timeoutSeconds,omitempty"`
	Status              HistoryProcessingStatus `json:"status"`
	RawCharacterCount   int                     `json:"rawCharacterCount,omitempty"`
	ProcessedCharacters int                     `json:"processedCharacters,omitempty"`
	DeliveredCharacters int                     `json:"deliveredCharacters,omitempty"`
	Styling             string                  `json:"styling,omitempty"`
	Structure           string                  `json:"structure,omitempty"`
	Context             string                  `json:"context,omitempty"`
	ErrorKind           string                  `json:"errorKind,omitempty"`
	Response            *HistoryResponseDetails `json:"response,omitempty"`
}

type HistoryRunDetails struct {
	Source                            HistorySource            `json:"source"`
	RecordingMode                     string                   `json:"recordingMode,omitempty"`
	StartedAt                         time.Time                `json:"startedAt"`
	CompletedAt                       time.Time                `json:"completedAt,omitempty"`
	ElapsedMilliseconds               int64                    `json:"elapsedMilliseconds,omitempty"`
	Server                            string                   `json:"server,omitempty"`
	Route                             string                   `json:"route"`
	AuthenticationMode                string                   `json:"authenticationMode"`
	Model                             string                   `json:"model"`
	Language                          string                   `json:"language,omitempty"`
	ResponseMode                      HistoryResponseMode      `json:"responseMode"`
	InsertionMode                     insertion.Mode           `json:"insertionMode,omitempty"`
	Buffered                          bool                     `json:"buffered"`
	StreamFallbackReason              string                   `json:"streamFallbackReason,omitempty"`
	ErrorKind                         string                   `json:"errorKind,omitempty"`
	AudioDurationMilliseconds         int64                    `json:"audioDurationMilliseconds,omitempty"`
	CaptureDurationMilliseconds       int64                    `json:"captureDurationMilliseconds,omitempty"`
	Microphone                        string                   `json:"microphone,omitempty"`
	VADEnabled                        bool                     `json:"vadEnabled"`
	VADMode                           string                   `json:"vadMode,omitempty"`
	VADActivitySilenceMilliseconds    int                      `json:"vadActivitySilenceMilliseconds,omitempty"`
	SilenceTrimming                   bool                     `json:"silenceTrimming"`
	SpeechPaddingMilliseconds         int                      `json:"speechPaddingMilliseconds"`
	AutoStopEnabled                   bool                     `json:"autoStopEnabled"`
	AutoStopActive                    bool                     `json:"autoStopActive"`
	AutoStopSilenceMilliseconds       int                      `json:"autoStopSilenceMilliseconds,omitempty"`
	AutoStopMinimumSpeechMilliseconds int                      `json:"autoStopMinimumSpeechMilliseconds,omitempty"`
	AutoStopped                       bool                     `json:"autoStopped"`
	SilenceSplitting                  bool                     `json:"silenceSplitting"`
	SegmentCount                      int                      `json:"segmentCount,omitempty"`
	Segments                          []HistorySegmentDetails  `json:"segments,omitempty"`
	SegmentsTruncated                 bool                     `json:"segmentsTruncated"`
	DurationLimitReached              bool                     `json:"durationLimitReached"`
	FileName                          string                   `json:"fileName,omitempty"`
	FileSize                          int64                    `json:"fileSize,omitempty"`
	UploadMilliseconds                int64                    `json:"uploadMilliseconds,omitempty"`
	TranscriptionMilliseconds         int64                    `json:"transcriptionMilliseconds,omitempty"`
	RequestTimeoutSeconds             int                      `json:"requestTimeoutSeconds,omitempty"`
	Transcription                     *HistoryResponseDetails  `json:"transcription,omitempty"`
	Processing                        HistoryProcessingDetails `json:"processing"`
}

const (
	HistoryProcessingNotRequested HistoryProcessingStatus = "not-requested"
	HistoryProcessingPending      HistoryProcessingStatus = "pending"
	HistoryProcessingCompleted    HistoryProcessingStatus = "completed"
	HistoryProcessingFailed       HistoryProcessingStatus = "failed"
	HistoryProcessingCancelled    HistoryProcessingStatus = "cancelled"
)

const (
	HistoryTextFinal     HistoryTextVersion = "final"
	HistoryTextRaw       HistoryTextVersion = "raw"
	HistoryTextProcessed HistoryTextVersion = "processed"
)

type HistoryEntry struct {
	ID                uint64                  `json:"id"`
	Text              string                  `json:"text"`
	RawText           string                  `json:"rawText"`
	ProcessedText     string                  `json:"processedText,omitempty"`
	CompletedAt       time.Time               `json:"completedAt"`
	CharacterCount    int                     `json:"characterCount"`
	Outcome           HistoryOutcome          `json:"outcome"`
	ProcessingStatus  HistoryProcessingStatus `json:"processingStatus"`
	ProcessingMessage string                  `json:"processingMessage,omitempty"`
	Details           HistoryRunDetails       `json:"details"`
}

type historyBuffer struct {
	entries []HistoryEntry
	bytes   int
}

func (b *historyBuffer) add(entry HistoryEntry) bool {
	if entry.RawText == "" {
		entry.RawText = entry.Text
	}
	if entry.ProcessingStatus == "" {
		entry.ProcessingStatus = HistoryProcessingNotRequested
	}
	entry, ok := boundedHistoryEntry(entry)
	if !ok {
		return false
	}
	b.entries = append(b.entries, entry)
	b.bytes += historyEntryBytes(entry)
	b.enforceLimits()
	return true
}

func (b *historyBuffer) update(id uint64, apply func(*HistoryEntry)) bool {
	for i := range b.entries {
		if b.entries[i].ID != id {
			continue
		}
		b.bytes -= historyEntryBytes(b.entries[i])
		apply(&b.entries[i])
		entry, ok := boundedHistoryEntry(b.entries[i])
		if !ok {
			b.discardAt(i)
			return true
		}
		b.entries[i] = entry
		b.bytes += historyEntryBytes(entry)
		b.enforceLimits()
		return true
	}
	return false
}

func boundedHistoryEntry(entry HistoryEntry) (HistoryEntry, bool) {
	entry = cloneHistoryEntry(entry)
	if entry.ProcessingMessage == historyBudgetMessage {
		applyHistoryBudgetFallback(&entry)
	}
	normalizeHistoryEntry(&entry)
	entry.CharacterCount = utf8.RuneCountInString(entry.Text)
	entry = cloneHistoryEntry(entry)
	if bytes := historyEntryBytes(entry); bytes > 0 && bytes <= MaxHistoryBytes {
		return entry, true
	}
	if entry.ProcessedText == "" || entry.RawText == "" {
		return HistoryEntry{}, false
	}
	applyHistoryBudgetFallback(&entry)
	normalizeHistoryEntry(&entry)
	entry.CharacterCount = utf8.RuneCountInString(entry.Text)
	entry = cloneHistoryEntry(entry)
	if bytes := historyEntryBytes(entry); bytes > 0 && bytes <= MaxHistoryBytes {
		return entry, true
	}
	return HistoryEntry{}, false
}

func applyHistoryBudgetFallback(entry *HistoryEntry) {
	entry.Text = entry.RawText
	entry.ProcessedText = ""
	entry.ProcessingStatus = HistoryProcessingFailed
	entry.ProcessingMessage = historyBudgetMessage
	entry.Details.Processing.Status = HistoryProcessingFailed
	entry.Details.Processing.ErrorKind = historyBudgetErrorKind
}

func (b *historyBuffer) enforceLimits() {
	for len(b.entries) > MaxHistoryEntries || b.bytes > MaxHistoryBytes {
		b.removeAt(0)
	}
}

func (b *historyBuffer) removeAt(index int) {
	b.bytes -= historyEntryBytes(b.entries[index])
	b.discardAt(index)
}

func (b *historyBuffer) discardAt(index int) {
	copy(b.entries[index:], b.entries[index+1:])
	b.entries[len(b.entries)-1] = HistoryEntry{}
	b.entries = b.entries[:len(b.entries)-1]
	if len(b.entries) == 0 {
		b.bytes = 0
	}
}

func historyEntryBytes(entry HistoryEntry) int {
	details, _ := json.Marshal(entry.Details)
	return len(entry.RawText) + len(entry.ProcessedText) + len(entry.ProcessingMessage) + len(details)
}

func (b *historyBuffer) newestFirst() []HistoryEntry {
	entries := make([]HistoryEntry, len(b.entries))
	for i := range b.entries {
		entries[len(b.entries)-1-i] = cloneHistoryEntry(b.entries[i])
	}
	return entries
}

func cloneHistoryEntry(entry HistoryEntry) HistoryEntry {
	entry.Details.Segments = append([]HistorySegmentDetails(nil), entry.Details.Segments...)
	entry.Details.Transcription = cloneResponseDetails(entry.Details.Transcription)
	entry.Details.Processing.Response = cloneResponseDetails(entry.Details.Processing.Response)
	return entry
}

func normalizeHistoryEntry(entry *HistoryEntry) {
	if len(entry.ProcessingMessage) > 256 {
		entry.ProcessingMessage = entry.ProcessingMessage[:256]
	}
	entry.Details.Server = boundedHistoryString(entry.Details.Server, 2048)
	entry.Details.Route = boundedHistoryString(entry.Details.Route, 256)
	entry.Details.AuthenticationMode = boundedHistoryString(entry.Details.AuthenticationMode, 32)
	entry.Details.Model = boundedHistoryString(entry.Details.Model, 200)
	entry.Details.Language = boundedHistoryString(entry.Details.Language, 32)
	entry.Details.Microphone = boundedHistoryString(entry.Details.Microphone, 256)
	entry.Details.FileName = boundedHistoryString(entry.Details.FileName, 256)
	entry.Details.ErrorKind = boundedHistoryString(entry.Details.ErrorKind, 64)
	entry.Details.StreamFallbackReason = boundedHistoryString(entry.Details.StreamFallbackReason, 64)
	entry.Details.Processing.Server = boundedHistoryString(entry.Details.Processing.Server, 2048)
	entry.Details.Processing.Model = boundedHistoryString(entry.Details.Processing.Model, 200)
	entry.Details.Processing.Preset = boundedHistoryString(entry.Details.Processing.Preset, 32)
	entry.Details.Processing.ErrorKind = boundedHistoryString(entry.Details.Processing.ErrorKind, 64)
	entry.Details.Processing.Styling = boundedHistoryString(entry.Details.Processing.Styling, 32)
	entry.Details.Processing.Structure = boundedHistoryString(entry.Details.Processing.Structure, 32)
	entry.Details.Processing.Context = boundedHistoryString(entry.Details.Processing.Context, 32)
	normalizeResponseDetails(entry.Details.Transcription)
	normalizeResponseDetails(entry.Details.Processing.Response)
	if len(entry.Details.Segments) > MaxHistorySegments {
		entry.Details.Segments = append([]HistorySegmentDetails(nil), entry.Details.Segments[:MaxHistorySegments]...)
		entry.Details.SegmentsTruncated = true
	}
	for i := range entry.Details.Segments {
		entry.Details.Segments[i].Boundary = boundedHistoryString(entry.Details.Segments[i].Boundary, 64)
	}
}

func SanitizedServer(baseURL string) string {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func ErrorKind(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	var requestError *inference.Error
	if errors.As(err, &requestError) {
		return requestError.Kind
	}
	var streamUnsupported *inference.FileStreamUnsupportedError
	if errors.As(err, &streamUnsupported) {
		return "stream_unsupported"
	}
	return "processing"
}

func NewResponseDetails(metadata inference.ResponseMetadata) *HistoryResponseDetails {
	if !metadata.Reported() {
		return nil
	}
	return &HistoryResponseDetails{
		RequestID:              metadata.RequestID,
		ResponseID:             metadata.ResponseID,
		EffectiveModel:         metadata.EffectiveModel,
		Provider:               metadata.Provider,
		FinishReason:           metadata.FinishReason,
		ServiceTier:            metadata.ServiceTier,
		SystemFingerprint:      metadata.SystemFingerprint,
		CreatedAtUnix:          cloneValue(metadata.CreatedAtUnix),
		DetectedLanguages:      append([]string(nil), metadata.DetectedLanguages...),
		ServerAudioSeconds:     cloneValue(metadata.ServerAudioSeconds),
		RequestCount:           metadata.RequestCount,
		UsageReportCount:       metadata.UsageReportCount,
		CostReportCount:        metadata.CostReportCount,
		PerformanceReportCount: metadata.PerformanceReportCount,
		Usage: HistoryUsageDetails{
			Type:                  metadata.Usage.Type,
			InputTokens:           cloneValue(metadata.Usage.InputTokens),
			OutputTokens:          cloneValue(metadata.Usage.OutputTokens),
			TotalTokens:           cloneValue(metadata.Usage.TotalTokens),
			AudioInputTokens:      cloneValue(metadata.Usage.AudioInputTokens),
			TextInputTokens:       cloneValue(metadata.Usage.TextInputTokens),
			CachedInputTokens:     cloneValue(metadata.Usage.CachedInputTokens),
			CacheWriteTokens:      cloneValue(metadata.Usage.CacheWriteTokens),
			ReasoningOutputTokens: cloneValue(metadata.Usage.ReasoningOutputTokens),
			AudioSeconds:          cloneValue(metadata.Usage.AudioSeconds),
			ReportedCost:          cloneValue(metadata.Usage.ReportedCost),
			UpstreamCost:          cloneValue(metadata.Usage.UpstreamCost),
		},
		Performance: HistoryPerformanceDetails{
			PromptTokens:                   cloneValue(metadata.Performance.PromptTokens),
			PromptMilliseconds:             cloneValue(metadata.Performance.PromptMilliseconds),
			PromptMillisecondsPerToken:     cloneValue(metadata.Performance.PromptMillisecondsPerToken),
			PromptTokensPerSecond:          cloneValue(metadata.Performance.PromptTokensPerSecond),
			GeneratedTokens:                cloneValue(metadata.Performance.GeneratedTokens),
			GenerationMilliseconds:         cloneValue(metadata.Performance.GenerationMilliseconds),
			GenerationMillisecondsPerToken: cloneValue(metadata.Performance.GenerationMillisecondsPerToken),
			GenerationTokensPerSecond:      cloneValue(metadata.Performance.GenerationTokensPerSecond),
			CachedPromptTokens:             cloneValue(metadata.Performance.CachedPromptTokens),
		},
	}
}

func cloneResponseDetails(details *HistoryResponseDetails) *HistoryResponseDetails {
	if details == nil {
		return nil
	}
	copy := *details
	details = &copy
	details.CreatedAtUnix = cloneValue(details.CreatedAtUnix)
	details.DetectedLanguages = append([]string(nil), details.DetectedLanguages...)
	details.ServerAudioSeconds = cloneValue(details.ServerAudioSeconds)
	details.Usage.InputTokens = cloneValue(details.Usage.InputTokens)
	details.Usage.OutputTokens = cloneValue(details.Usage.OutputTokens)
	details.Usage.TotalTokens = cloneValue(details.Usage.TotalTokens)
	details.Usage.AudioInputTokens = cloneValue(details.Usage.AudioInputTokens)
	details.Usage.TextInputTokens = cloneValue(details.Usage.TextInputTokens)
	details.Usage.CachedInputTokens = cloneValue(details.Usage.CachedInputTokens)
	details.Usage.CacheWriteTokens = cloneValue(details.Usage.CacheWriteTokens)
	details.Usage.ReasoningOutputTokens = cloneValue(details.Usage.ReasoningOutputTokens)
	details.Usage.AudioSeconds = cloneValue(details.Usage.AudioSeconds)
	details.Usage.ReportedCost = cloneValue(details.Usage.ReportedCost)
	details.Usage.UpstreamCost = cloneValue(details.Usage.UpstreamCost)
	details.Performance.PromptTokens = cloneValue(details.Performance.PromptTokens)
	details.Performance.PromptMilliseconds = cloneValue(details.Performance.PromptMilliseconds)
	details.Performance.PromptMillisecondsPerToken = cloneValue(details.Performance.PromptMillisecondsPerToken)
	details.Performance.PromptTokensPerSecond = cloneValue(details.Performance.PromptTokensPerSecond)
	details.Performance.GeneratedTokens = cloneValue(details.Performance.GeneratedTokens)
	details.Performance.GenerationMilliseconds = cloneValue(details.Performance.GenerationMilliseconds)
	details.Performance.GenerationMillisecondsPerToken = cloneValue(details.Performance.GenerationMillisecondsPerToken)
	details.Performance.GenerationTokensPerSecond = cloneValue(details.Performance.GenerationTokensPerSecond)
	details.Performance.CachedPromptTokens = cloneValue(details.Performance.CachedPromptTokens)
	return details
}

func normalizeResponseDetails(details *HistoryResponseDetails) {
	if details == nil {
		return
	}
	details.RequestID = boundedHistoryString(details.RequestID, 256)
	details.ResponseID = boundedHistoryString(details.ResponseID, 256)
	details.EffectiveModel = boundedHistoryString(details.EffectiveModel, 200)
	details.Provider = boundedHistoryString(details.Provider, 128)
	details.FinishReason = boundedHistoryString(details.FinishReason, 64)
	details.ServiceTier = boundedHistoryString(details.ServiceTier, 64)
	details.SystemFingerprint = boundedHistoryString(details.SystemFingerprint, 256)
	details.Usage.Type = boundedHistoryString(details.Usage.Type, 32)
	if details.RequestCount < 0 || details.RequestCount > 10_000 {
		details.RequestCount = 0
	}
	for _, count := range []*int{&details.UsageReportCount, &details.CostReportCount, &details.PerformanceReportCount} {
		if *count < 0 || *count > details.RequestCount {
			*count = 0
		}
	}
	if len(details.DetectedLanguages) > 8 {
		details.DetectedLanguages = append([]string(nil), details.DetectedLanguages[:8]...)
	}
	for i := range details.DetectedLanguages {
		details.DetectedLanguages[i] = boundedHistoryString(details.DetectedLanguages[i], 32)
	}
}

func cloneValue[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func NewProcessingDetails(settings config.PostProcessingSettings, raw string) HistoryProcessingDetails {
	details := HistoryProcessingDetails{
		Requested:         settings.Enabled,
		Status:            HistoryProcessingNotRequested,
		RawCharacterCount: utf8.RuneCountInString(raw),
		TimeoutSeconds:    settings.TimeoutSeconds,
	}
	if !settings.Enabled {
		return details
	}
	details.Server = SanitizedServer(settings.BaseURL)
	details.Model = boundedHistoryString(settings.Model, 200)
	details.Preset = boundedHistoryString(string(settings.Preset), 32)
	details.Status = HistoryProcessingPending
	if settings.Preset == config.PostProcessingPresetS1Mini {
		details.Styling = boundedHistoryString(settings.Styling, 32)
		details.Structure = boundedHistoryString(settings.Structure, 32)
		details.Context = boundedHistoryString(settings.Context, 32)
	}
	return details
}

func FinalizeDetails(details *HistoryRunDetails, completedAt time.Time, durationLimitReached bool) {
	details.CompletedAt = completedAt
	if !details.StartedAt.IsZero() {
		details.ElapsedMilliseconds = max(0, completedAt.Sub(details.StartedAt).Milliseconds())
	}
	details.DurationLimitReached = durationLimitReached
	if len(details.Segments) > MaxHistorySegments {
		details.Segments = append([]HistorySegmentDetails(nil), details.Segments[:MaxHistorySegments]...)
		details.SegmentsTruncated = true
	}
}

func boundedHistoryString(value string, maximum int) string {
	value = strings.TrimSpace(value)
	if len(value) <= maximum {
		return value
	}
	return value[:maximum]
}

func (b *historyBuffer) remove(id uint64) bool {
	for i := range b.entries {
		if b.entries[i].ID != id {
			continue
		}
		b.removeAt(i)
		return true
	}
	return false
}

func (b *historyBuffer) clear() {
	for i := range b.entries {
		b.entries[i] = HistoryEntry{}
	}
	b.entries = nil
	b.bytes = 0
}
