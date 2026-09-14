package inference

import (
	"encoding/json"
	"io"
)

func readTranscriptionJSON(reader io.Reader, limit int64, key string) (TranscriptionResult, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return TranscriptionResult{}, &Error{Kind: "response", Message: "transcription response could not be read"}
	}
	if int64(len(body)) > limit {
		return TranscriptionResult{}, &Error{Kind: "response_too_large", Message: "transcription response is too large"}
	}
	return decodeTranscriptionJSON(body, key)
}

// decodeTranscriptionJSON shares the completed response contract between Voice
// and files. Each caller retains its own read limit and final text admission.
func decodeTranscriptionJSON(body []byte, key string) (TranscriptionResult, error) {
	var out struct {
		Text              *string         `json:"text"`
		ID                string          `json:"id"`
		RequestID         string          `json:"request_id"`
		Model             string          `json:"model"`
		Provider          string          `json:"provider"`
		Created           json.RawMessage `json:"created"`
		Usage             json.RawMessage `json:"usage"`
		Timings           json.RawMessage `json:"timings"`
		Languages         json.RawMessage `json:"languages"`
		Language          string          `json:"language"`
		Duration          json.RawMessage `json:"duration"`
		ServiceTier       string          `json:"service_tier"`
		SystemFingerprint string          `json:"system_fingerprint"`
	}
	if err := json.Unmarshal(body, &out); err != nil || out.Text == nil {
		return TranscriptionResult{}, &Error{Kind: "malformed_response", Message: "expected JSON object with text"}
	}
	metadata := ResponseMetadata{
		RequestID:          safePeerString(out.RequestID, key),
		ResponseID:         safePeerString(out.ID, key),
		EffectiveModel:     safePeerString(out.Model, key),
		Provider:           safePeerString(out.Provider, key),
		CreatedAtUnix:      optionalInt(out.Created),
		DetectedLanguages:  parseLanguages(out.Languages, out.Language, key),
		ServerAudioSeconds: optionalFloat(out.Duration),
		ServiceTier:        safePeerString(out.ServiceTier, key),
		SystemFingerprint:  safePeerString(out.SystemFingerprint, key),
		RequestCount:       1,
	}
	applyUsageMetadata(&metadata, out.Usage, key)
	applyPerformanceMetadata(&metadata, out.Timings)
	return TranscriptionResult{Text: *out.Text, Metadata: metadata}, nil
}
