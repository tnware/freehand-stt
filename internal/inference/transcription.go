package inference

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

// Transcribe sends one bounded in-memory microphone recording and expects a
// completed OpenAI-compatible transcription response.
func (c *Client) Transcribe(ctx context.Context, base, model, language, key string, headers map[string]string, wav []byte) (TranscriptionResult, error) {
	contract, err := c.contract(compatibility.Transcription)
	if err != nil {
		return TranscriptionResult{}, err
	}
	if !contract.Capabilities.ServerLoadedModel && strings.TrimSpace(model) == "" {
		return TranscriptionResult{}, &Error{Kind: "invalid_settings", Message: "choose a transcription model"}
	}
	if err := c.validateTranscriptionOptions(); err != nil {
		return TranscriptionResult{}, err
	}
	if err := modelprofile.ValidateLanguage(c.modelProfile, compatibility.Transcription, language, nil); err != nil {
		return TranscriptionResult{}, &Error{Kind: "invalid_settings", Message: err.Error()}
	}
	language, err = contract.TranscriptionLanguage(language)
	if err != nil {
		return TranscriptionResult{}, &Error{Kind: "invalid_settings", Message: "invalid transcription language selection"}
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
	if !contract.Capabilities.ServerLoadedModel {
		_ = mw.WriteField("model", model)
	}
	_ = mw.WriteField("response_format", "json")
	if contract.ID == compatibility.NeMoSpeechV1 {
		_ = mw.WriteField("automatic_punctuation", "true")
		_ = mw.WriteField("verbatim", "true")
	}
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
	result, err := decodeTranscriptionJSON(bodyBytes, key)
	if err != nil {
		return TranscriptionResult{}, err
	}
	text := strings.TrimSpace(result.Text)
	if c.modelProfile == modelprofile.Nemotron35 {
		text, _ = modelprofile.StripNemotronLanguageTag(text)
	}
	if key != "" && strings.Contains(text, key) {
		return TranscriptionResult{}, &Error{Kind: "credential_reflection", Message: "transcription response rejected"}
	}
	if result.Metadata.RequestID == "" {
		result.Metadata.RequestID = metadataFromHeaders(resp.Header, key).RequestID
	}
	result.Text = text
	result.Metadata = sanitizeResponseMetadata(result.Metadata, key)
	return result, nil
}
