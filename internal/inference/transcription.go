package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

// Transcribe sends one bounded in-memory microphone recording and expects a
// completed OpenAI-compatible transcription response.
func (c *Client) Transcribe(ctx context.Context, base, model, language, key string, headers map[string]string, wav []byte) (TranscriptionResult, error) {
	contract, err := c.contract(compatibility.Transcription)
	if err != nil {
		return TranscriptionResult{}, err
	}
	if err := c.validateTranscriptionOptions(); err != nil {
		return TranscriptionResult{}, err
	}
	if language != "" && !contract.Capabilities.LanguageHint {
		return TranscriptionResult{}, &Error{Kind: "invalid_settings", Message: "language hints are unavailable for this profile"}
	}
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
	_ = mw.WriteField("response_format", "json")
	if language != "" {
		_ = mw.WriteField("language", language)
	}
	if err = writeTranscriptionOptions(mw, c.transcriptionOptions); err != nil {
		return TranscriptionResult{}, err
	}
	if err = mw.Close(); err != nil {
		return TranscriptionResult{}, err
	}
	defer zeroBytes(body.Bytes())

	u, err := endpoint(base, contract.Path)
	if err != nil {
		return TranscriptionResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, &body)
	if err != nil {
		return TranscriptionResult{}, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Accept", "application/json")
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
	metadata := metadataFromHeaders(resp.Header, key)
	if requestID := safePeerString(out.RequestID, key); requestID != "" {
		metadata.RequestID = requestID
	}
	metadata.RequestCount = 1
	metadata.ResponseID = safePeerString(out.ID, key)
	metadata.EffectiveModel = safePeerString(out.Model, key)
	metadata.Provider = safePeerString(out.Provider, key)
	metadata.CreatedAtUnix = optionalInt(out.Created)
	metadata.DetectedLanguages = parseLanguages(out.Languages, out.Language, key)
	metadata.ServerAudioSeconds = optionalFloat(out.Duration)
	metadata.ServiceTier = safePeerString(out.ServiceTier, key)
	metadata.SystemFingerprint = safePeerString(out.SystemFingerprint, key)
	applyUsageMetadata(&metadata, out.Usage, key)
	applyPerformanceMetadata(&metadata, out.Timings)
	return TranscriptionResult{Text: text, Metadata: sanitizeResponseMetadata(metadata, key)}, nil
}
