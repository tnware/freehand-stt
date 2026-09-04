package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// Transcribe sends one bounded in-memory microphone recording and expects a
// completed OpenAI-compatible transcription response.
func (c *Client) Transcribe(ctx context.Context, base, model, language, key string, headers map[string]string, wav []byte) (TranscriptionResult, error) {
	if len(wav) > 8<<20 {
		return TranscriptionResult{}, &Error{Kind: "request_too_large", Message: "recording exceeds 8 MiB"}
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	p, err := mw.CreateFormFile("file", "recording.wav")
	if err != nil {
		return TranscriptionResult{}, err
	}
	if _, err = p.Write(wav); err != nil {
		return TranscriptionResult{}, err
	}
	_ = mw.WriteField("model", model)
	if language != "" {
		_ = mw.WriteField("language", language)
	}
	if err = mw.Close(); err != nil {
		return TranscriptionResult{}, err
	}
	defer zeroBytes(body.Bytes())

	u, err := endpoint(base, "audio/transcriptions")
	if err != nil {
		return TranscriptionResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, &body)
	if err != nil {
		return TranscriptionResult{}, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return TranscriptionResult{}, requestFailure(err, ctx, "transcription request")
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil {
		return TranscriptionResult{}, &Error{Kind: "response", Message: "transcription response could not be read"}
	}
	if len(bodyBytes) > maxResponse {
		return TranscriptionResult{}, &Error{Kind: "response_too_large", Message: "response exceeds 1 MiB"}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// The peer controls the response body and can reflect Authorization or
		// transcript data. Preserve only the status code across this boundary.
		return TranscriptionResult{}, &Error{Kind: "http", Status: resp.StatusCode, Message: "transcription request failed"}
	}
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
	if err = json.Unmarshal(bodyBytes, &out); err != nil || out.Text == nil {
		return TranscriptionResult{}, &Error{Kind: "malformed_response", Message: "expected JSON object with text"}
	}
	text := strings.TrimSpace(*out.Text)
	if key != "" && strings.Contains(text, key) {
		return TranscriptionResult{}, &Error{Kind: "credential_reflection", Message: "transcription response rejected"}
	}
	metadata := metadataFromHeaders(resp.Header)
	if requestID := boundedMetadataString(out.RequestID); requestID != "" {
		metadata.RequestID = requestID
	}
	metadata.RequestCount = 1
	metadata.ResponseID = boundedMetadataString(out.ID)
	metadata.EffectiveModel = boundedMetadataString(out.Model)
	metadata.Provider = boundedMetadataString(out.Provider)
	metadata.CreatedAtUnix = optionalInt(out.Created)
	metadata.DetectedLanguages = parseLanguages(out.Languages, out.Language)
	metadata.ServerAudioSeconds = optionalFloat(out.Duration)
	metadata.ServiceTier = boundedMetadataString(out.ServiceTier)
	metadata.SystemFingerprint = boundedMetadataString(out.SystemFingerprint)
	applyUsageMetadata(&metadata, out.Usage)
	applyPerformanceMetadata(&metadata, out.Timings)
	return TranscriptionResult{Text: text, Metadata: metadata}, nil
}
