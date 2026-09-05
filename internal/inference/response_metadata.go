package inference

import (
	"encoding/json"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

const (
	maxMetadataString = 512
	maxUsageValue     = int64(1_000_000_000_000)
	maxMetricValue    = 1_000_000_000_000.0
	maxLanguages      = 8
)

// Usage contains optional usage values reported by an inference endpoint.
// Nil values mean the endpoint did not report that metric; they are never
// inferred from audio duration, transcript length, or another metric.
type Usage struct {
	Type                  string
	InputTokens           *int64
	OutputTokens          *int64
	TotalTokens           *int64
	AudioInputTokens      *int64
	TextInputTokens       *int64
	CachedInputTokens     *int64
	CacheWriteTokens      *int64
	ReasoningOutputTokens *int64
	AudioSeconds          *float64
	ReportedCost          *float64
	UpstreamCost          *float64
}

func (u Usage) Reported() bool {
	return u.Type != "" || u.InputTokens != nil || u.OutputTokens != nil ||
		u.TotalTokens != nil || u.AudioInputTokens != nil || u.TextInputTokens != nil ||
		u.CachedInputTokens != nil || u.CacheWriteTokens != nil ||
		u.ReasoningOutputTokens != nil || u.AudioSeconds != nil ||
		u.ReportedCost != nil || u.UpstreamCost != nil
}

// Performance contains optional, endpoint-specific processing metrics. These
// are returned by servers such as llama.cpp but are not part of the portable
// OpenAI response contract.
type Performance struct {
	PromptTokens                   *int64
	PromptMilliseconds             *float64
	PromptMillisecondsPerToken     *float64
	PromptTokensPerSecond          *float64
	GeneratedTokens                *int64
	GenerationMilliseconds         *float64
	GenerationMillisecondsPerToken *float64
	GenerationTokensPerSecond      *float64
	CachedPromptTokens             *int64
}

func (p Performance) Reported() bool {
	return p.PromptTokens != nil || p.PromptMilliseconds != nil ||
		p.PromptMillisecondsPerToken != nil || p.PromptTokensPerSecond != nil ||
		p.GeneratedTokens != nil || p.GenerationMilliseconds != nil ||
		p.GenerationMillisecondsPerToken != nil || p.GenerationTokensPerSecond != nil ||
		p.CachedPromptTokens != nil
}

// ResponseMetadata is bounded metadata returned alongside a successful
// inference result. RequestCount and the report counts make partial metadata
// explicit when several checkpoint requests are aggregated.
type ResponseMetadata struct {
	RequestID              string
	ResponseID             string
	EffectiveModel         string
	Provider               string
	FinishReason           string
	ServiceTier            string
	SystemFingerprint      string
	CreatedAtUnix          *int64
	DetectedLanguages      []string
	ServerAudioSeconds     *float64
	Usage                  Usage
	Performance            Performance
	RequestCount           int
	UsageReportCount       int
	CostReportCount        int
	PerformanceReportCount int
}

func (m ResponseMetadata) Reported() bool {
	return m.RequestID != "" || m.ResponseID != "" || m.EffectiveModel != "" ||
		m.Provider != "" || m.FinishReason != "" || m.ServiceTier != "" ||
		m.SystemFingerprint != "" || m.CreatedAtUnix != nil ||
		len(m.DetectedLanguages) > 0 || m.ServerAudioSeconds != nil ||
		m.Usage.Reported() || m.Performance.Reported()
}

type TranscriptionResult struct {
	Text     string
	Metadata ResponseMetadata
}

type ChatCompletionResult struct {
	Text     string
	Metadata ResponseMetadata
}

// safePeerString drops an entire optional value rather than substituting a
// marker that could itself equal a short credential. Input/collection bounds
// are enforced by the decoders; retained strings are bounded here as well.
func safePeerString(value, key string) string {
	if key != "" && strings.Contains(value, key) {
		return ""
	}
	return boundedMetadataString(value)
}

// sanitizeResponseMetadata is the publication boundary for successful STT and
// chat metadata, after JSON/SSE and header metadata have been combined. Keep
// valid text and benign metrics while removing literal credential reflections.
// Every retained peer string, including nested values, belongs here.
func sanitizeResponseMetadata(metadata ResponseMetadata, key string) ResponseMetadata {
	metadata.RequestID = safePeerString(metadata.RequestID, key)
	metadata.ResponseID = safePeerString(metadata.ResponseID, key)
	metadata.EffectiveModel = safePeerString(metadata.EffectiveModel, key)
	metadata.Provider = safePeerString(metadata.Provider, key)
	metadata.FinishReason = safePeerString(metadata.FinishReason, key)
	metadata.ServiceTier = safePeerString(metadata.ServiceTier, key)
	metadata.SystemFingerprint = safePeerString(metadata.SystemFingerprint, key)
	metadata.Usage.Type = safePeerString(metadata.Usage.Type, key)
	languages := make([]string, 0, min(len(metadata.DetectedLanguages), maxLanguages))
	for _, language := range metadata.DetectedLanguages {
		if language = safePeerString(language, key); language != "" && len(languages) < maxLanguages {
			languages = append(languages, language)
		}
	}
	metadata.DetectedLanguages = languages
	if !metadata.Usage.Reported() {
		metadata.UsageReportCount = 0
	}
	return metadata
}

func metadataFromHeaders(headers http.Header, key string) ResponseMetadata {
	for _, name := range []string{"X-Request-Id", "OpenAI-Request-Id", "Request-Id"} {
		if value := safePeerString(headers.Get(name), key); value != "" {
			return ResponseMetadata{RequestID: value}
		}
	}
	return ResponseMetadata{}
}

func parseUsage(raw json.RawMessage, key string) Usage {
	fields := rawObject(raw)
	if fields == nil {
		return Usage{}
	}
	usage := Usage{
		Type:         rawString(fields["type"], key),
		InputTokens:  firstInt(fields, "input_tokens", "prompt_tokens"),
		OutputTokens: firstInt(fields, "output_tokens", "completion_tokens"),
		TotalTokens:  optionalInt(fields["total_tokens"]),
		AudioSeconds: optionalFloat(fields["seconds"]),
		ReportedCost: optionalFloat(fields["cost"]),
	}
	inputDetails := firstObject(fields, "input_token_details", "input_tokens_details", "prompt_tokens_details")
	usage.AudioInputTokens = optionalInt(inputDetails["audio_tokens"])
	usage.TextInputTokens = optionalInt(inputDetails["text_tokens"])
	usage.CachedInputTokens = optionalInt(inputDetails["cached_tokens"])
	usage.CacheWriteTokens = optionalInt(inputDetails["cache_write_tokens"])
	outputDetails := firstObject(fields, "output_tokens_details", "completion_tokens_details")
	usage.ReasoningOutputTokens = optionalInt(outputDetails["reasoning_tokens"])
	if costDetails := rawObject(fields["cost_details"]); costDetails != nil {
		usage.UpstreamCost = optionalFloat(costDetails["upstream_inference_cost"])
	}
	return usage
}

func parsePerformance(raw json.RawMessage) Performance {
	fields := rawObject(raw)
	if fields == nil {
		return Performance{}
	}
	return Performance{
		PromptTokens:                   optionalInt(fields["prompt_n"]),
		PromptMilliseconds:             optionalFloat(fields["prompt_ms"]),
		PromptMillisecondsPerToken:     optionalFloat(fields["prompt_per_token_ms"]),
		PromptTokensPerSecond:          optionalFloat(fields["prompt_per_second"]),
		GeneratedTokens:                optionalInt(fields["predicted_n"]),
		GenerationMilliseconds:         optionalFloat(fields["predicted_ms"]),
		GenerationMillisecondsPerToken: optionalFloat(fields["predicted_per_token_ms"]),
		GenerationTokensPerSecond:      optionalFloat(fields["predicted_per_second"]),
		CachedPromptTokens:             optionalInt(fields["cache_n"]),
	}
}

func applyUsageMetadata(metadata *ResponseMetadata, raw json.RawMessage, key string) {
	metadata.Usage = parseUsage(raw, key)
	if metadata.Usage.Reported() {
		metadata.UsageReportCount = 1
	}
	if metadata.Usage.ReportedCost != nil || metadata.Usage.UpstreamCost != nil {
		metadata.CostReportCount = 1
	}
}

func applyPerformanceMetadata(metadata *ResponseMetadata, raw json.RawMessage) {
	metadata.Performance = parsePerformance(raw)
	if metadata.Performance.Reported() {
		metadata.PerformanceReportCount = 1
	}
}

func parseLanguages(raw json.RawMessage, single, key string) []string {
	languages := make([]string, 0, maxLanguages)
	appendLanguage := func(value string) {
		value = safePeerString(value, key)
		if value != "" && !slices.Contains(languages, value) && len(languages) < maxLanguages {
			languages = append(languages, value)
		}
	}
	appendLanguage(single)
	if len(raw) == 0 || string(raw) == "null" {
		return languages
	}
	var values []json.RawMessage
	if json.Unmarshal(raw, &values) != nil {
		return languages
	}
	for _, value := range values {
		if code := rawString(value, key); code != "" {
			appendLanguage(code)
			continue
		}
		appendLanguage(rawString(rawObject(value)["code"], key))
	}
	return languages
}

func (m *ResponseMetadata) Add(other ResponseMetadata) {
	if other.RequestCount == 0 {
		other.RequestCount = 1
	}
	if m.RequestCount == 0 {
		*m = cloneMetadata(other)
		return
	}
	previousRequests := m.RequestCount
	m.RequestCount += other.RequestCount
	m.UsageReportCount += other.UsageReportCount
	m.CostReportCount += other.CostReportCount
	m.PerformanceReportCount += other.PerformanceReportCount
	if previousRequests > 0 || other.RequestCount > 1 {
		m.RequestID = ""
		m.ResponseID = ""
	}
	m.EffectiveModel = stableString(m.EffectiveModel, other.EffectiveModel)
	m.Provider = stableString(m.Provider, other.Provider)
	m.FinishReason = stableString(m.FinishReason, other.FinishReason)
	m.ServiceTier = stableString(m.ServiceTier, other.ServiceTier)
	m.SystemFingerprint = stableString(m.SystemFingerprint, other.SystemFingerprint)
	m.CreatedAtUnix = nil
	m.DetectedLanguages = mergeLanguages(m.DetectedLanguages, other.DetectedLanguages)
	m.ServerAudioSeconds = sumFloat(m.ServerAudioSeconds, other.ServerAudioSeconds)
	m.Usage = addUsage(m.Usage, other.Usage)
	m.Performance = addPerformance(m.Performance, other.Performance)
}

func cloneMetadata(metadata ResponseMetadata) ResponseMetadata {
	metadata.DetectedLanguages = append([]string(nil), metadata.DetectedLanguages...)
	metadata.CreatedAtUnix = cloneInt(metadata.CreatedAtUnix)
	metadata.ServerAudioSeconds = cloneFloat(metadata.ServerAudioSeconds)
	metadata.Usage = cloneUsage(metadata.Usage)
	metadata.Performance = clonePerformance(metadata.Performance)
	return metadata
}

func cloneUsage(usage Usage) Usage {
	usage.InputTokens = cloneInt(usage.InputTokens)
	usage.OutputTokens = cloneInt(usage.OutputTokens)
	usage.TotalTokens = cloneInt(usage.TotalTokens)
	usage.AudioInputTokens = cloneInt(usage.AudioInputTokens)
	usage.TextInputTokens = cloneInt(usage.TextInputTokens)
	usage.CachedInputTokens = cloneInt(usage.CachedInputTokens)
	usage.CacheWriteTokens = cloneInt(usage.CacheWriteTokens)
	usage.ReasoningOutputTokens = cloneInt(usage.ReasoningOutputTokens)
	usage.AudioSeconds = cloneFloat(usage.AudioSeconds)
	usage.ReportedCost = cloneFloat(usage.ReportedCost)
	usage.UpstreamCost = cloneFloat(usage.UpstreamCost)
	return usage
}

func clonePerformance(performance Performance) Performance {
	performance.PromptTokens = cloneInt(performance.PromptTokens)
	performance.PromptMilliseconds = cloneFloat(performance.PromptMilliseconds)
	performance.PromptMillisecondsPerToken = cloneFloat(performance.PromptMillisecondsPerToken)
	performance.PromptTokensPerSecond = cloneFloat(performance.PromptTokensPerSecond)
	performance.GeneratedTokens = cloneInt(performance.GeneratedTokens)
	performance.GenerationMilliseconds = cloneFloat(performance.GenerationMilliseconds)
	performance.GenerationMillisecondsPerToken = cloneFloat(performance.GenerationMillisecondsPerToken)
	performance.GenerationTokensPerSecond = cloneFloat(performance.GenerationTokensPerSecond)
	performance.CachedPromptTokens = cloneInt(performance.CachedPromptTokens)
	return performance
}

func addUsage(left, right Usage) Usage {
	left.Type = stableString(left.Type, right.Type)
	left.InputTokens = sumInt(left.InputTokens, right.InputTokens)
	left.OutputTokens = sumInt(left.OutputTokens, right.OutputTokens)
	left.TotalTokens = sumInt(left.TotalTokens, right.TotalTokens)
	left.AudioInputTokens = sumInt(left.AudioInputTokens, right.AudioInputTokens)
	left.TextInputTokens = sumInt(left.TextInputTokens, right.TextInputTokens)
	left.CachedInputTokens = sumInt(left.CachedInputTokens, right.CachedInputTokens)
	left.CacheWriteTokens = sumInt(left.CacheWriteTokens, right.CacheWriteTokens)
	left.ReasoningOutputTokens = sumInt(left.ReasoningOutputTokens, right.ReasoningOutputTokens)
	left.AudioSeconds = sumFloat(left.AudioSeconds, right.AudioSeconds)
	left.ReportedCost = sumFloat(left.ReportedCost, right.ReportedCost)
	left.UpstreamCost = sumFloat(left.UpstreamCost, right.UpstreamCost)
	return left
}

func addPerformance(left, right Performance) Performance {
	promptMillisecondsPerToken := stableFloat(left.PromptMillisecondsPerToken, right.PromptMillisecondsPerToken)
	promptTokensPerSecond := stableFloat(left.PromptTokensPerSecond, right.PromptTokensPerSecond)
	generationMillisecondsPerToken := stableFloat(left.GenerationMillisecondsPerToken, right.GenerationMillisecondsPerToken)
	generationTokensPerSecond := stableFloat(left.GenerationTokensPerSecond, right.GenerationTokensPerSecond)
	left.PromptTokens = sumInt(left.PromptTokens, right.PromptTokens)
	left.PromptMilliseconds = sumFloat(left.PromptMilliseconds, right.PromptMilliseconds)
	left.GeneratedTokens = sumInt(left.GeneratedTokens, right.GeneratedTokens)
	left.GenerationMilliseconds = sumFloat(left.GenerationMilliseconds, right.GenerationMilliseconds)
	left.CachedPromptTokens = sumInt(left.CachedPromptTokens, right.CachedPromptTokens)
	left.PromptMillisecondsPerToken = firstFloat(ratio(left.PromptMilliseconds, left.PromptTokens), promptMillisecondsPerToken)
	left.PromptTokensPerSecond = firstFloat(rate(left.PromptTokens, left.PromptMilliseconds), promptTokensPerSecond)
	left.GenerationMillisecondsPerToken = firstFloat(ratio(left.GenerationMilliseconds, left.GeneratedTokens), generationMillisecondsPerToken)
	left.GenerationTokensPerSecond = firstFloat(rate(left.GeneratedTokens, left.GenerationMilliseconds), generationTokensPerSecond)
	return left
}

func stableFloat(left, right *float64) *float64 {
	if left == nil || right == nil || *left != *right {
		return nil
	}
	return cloneFloat(left)
}

func firstFloat(preferred, fallback *float64) *float64 {
	if preferred != nil {
		return preferred
	}
	return fallback
}

func stableString(left, right string) string {
	if left == "" {
		return right
	}
	if right == "" || left == right {
		return left
	}
	return ""
}

func mergeLanguages(left, right []string) []string {
	result := append([]string(nil), left...)
	for _, language := range right {
		language = boundedMetadataString(language)
		if language == "" || slices.Contains(result, language) || len(result) >= maxLanguages {
			continue
		}
		result = append(result, language)
	}
	return result
}

func sumInt(left, right *int64) *int64 {
	if left == nil {
		return cloneInt(right)
	}
	if right == nil {
		return left
	}
	if *left > maxUsageValue-*right {
		return nil
	}
	value := *left + *right
	return &value
}

func sumFloat(left, right *float64) *float64 {
	if left == nil {
		return cloneFloat(right)
	}
	if right == nil {
		return left
	}
	value := *left + *right
	if !validMetric(value) {
		return nil
	}
	return &value
}

func ratio(milliseconds *float64, tokens *int64) *float64 {
	if milliseconds == nil || tokens == nil || *tokens <= 0 {
		return nil
	}
	value := *milliseconds / float64(*tokens)
	if !validMetric(value) {
		return nil
	}
	return &value
}

func rate(tokens *int64, milliseconds *float64) *float64 {
	if tokens == nil || milliseconds == nil || *milliseconds <= 0 {
		return nil
	}
	value := float64(*tokens) * 1000 / *milliseconds
	if !validMetric(value) {
		return nil
	}
	return &value
}

func rawObject(raw json.RawMessage) map[string]json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil {
		return nil
	}
	return object
}

func firstObject(fields map[string]json.RawMessage, names ...string) map[string]json.RawMessage {
	for _, name := range names {
		if object := rawObject(fields[name]); object != nil {
			return object
		}
	}
	return nil
}

func firstInt(fields map[string]json.RawMessage, names ...string) *int64 {
	for _, name := range names {
		if value := optionalInt(fields[name]); value != nil {
			return value
		}
	}
	return nil
}

func rawString(raw json.RawMessage, key string) string {
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return ""
	}
	return safePeerString(value, key)
}

func optionalInt(raw json.RawMessage) *int64 {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var number json.Number
	if json.Unmarshal(raw, &number) != nil {
		return nil
	}
	value, err := strconv.ParseInt(number.String(), 10, 64)
	if err != nil || value < 0 || value > maxUsageValue {
		return nil
	}
	return &value
}

func optionalFloat(raw json.RawMessage) *float64 {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var number json.Number
	if json.Unmarshal(raw, &number) != nil {
		return nil
	}
	value, err := strconv.ParseFloat(number.String(), 64)
	if err != nil || !validMetric(value) {
		return nil
	}
	return &value
}

func validMetric(value float64) bool {
	return value >= 0 && value <= maxMetricValue && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func boundedMetadataString(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > maxMetadataString {
		value = value[:maxMetadataString]
	}
	return value
}

func cloneInt(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
