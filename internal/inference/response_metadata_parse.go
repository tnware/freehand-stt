package inference

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/tnware/freehand-stt/internal/speechlanguage"
)

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
		languages = speechlanguage.MergeDetected(languages, []string{value})
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
