package inference

import (
	"github.com/tnware/freehand-stt/internal/speechlanguage"
)

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

func mergeLanguages(left, right []string) []string { return speechlanguage.MergeDetected(left, right) }

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
