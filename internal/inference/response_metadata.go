package inference

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
