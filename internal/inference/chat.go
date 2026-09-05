package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

const maxChatRequest = 2 << 20

// ChatCompletion sends one bounded, non-streaming OpenAI-compatible chat
// completion. Response bodies controlled by the peer are never reflected in
// errors, matching the transcription boundary above.
func (c *Client) ChatCompletion(ctx context.Context, base, model, key, systemPrompt, userPrompt string) (ChatCompletionResult, error) {
	contract, err := c.contract(compatibility.PostProcessing)
	if err != nil {
		return ChatCompletionResult{}, err
	}
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	request := struct {
		Model       string    `json:"model"`
		Messages    []message `json:"messages"`
		Temperature float64   `json:"temperature"`
		Stream      bool      `json:"stream"`
	}{
		Model: model,
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0,
		Stream:      false,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return ChatCompletionResult{}, err
	}
	if len(body) > maxChatRequest {
		return ChatCompletionResult{}, &Error{Kind: "request_too_large", Message: "post-processing request exceeds 2 MiB"}
	}
	defer zeroBytes(body)
	u, err := endpoint(base, contract.Path)
	if err != nil {
		return ChatCompletionResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return ChatCompletionResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return ChatCompletionResult{}, requestFailure(err, ctx, "post-processing request")
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil {
		return ChatCompletionResult{}, &Error{Kind: "response", Message: "post-processing response could not be read"}
	}
	if len(responseBody) > maxResponse {
		return ChatCompletionResult{}, &Error{Kind: "response_too_large", Message: "post-processing response exceeds 1 MiB"}
	}
	defer zeroBytes(responseBody)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ChatCompletionResult{}, &Error{Kind: "http", Status: resp.StatusCode, Message: "post-processing request failed"}
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content *string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		ID                string          `json:"id"`
		RequestID         string          `json:"request_id"`
		Model             string          `json:"model"`
		Provider          string          `json:"provider"`
		Created           json.RawMessage `json:"created"`
		ServiceTier       string          `json:"service_tier"`
		SystemFingerprint string          `json:"system_fingerprint"`
		Usage             json.RawMessage `json:"usage"`
		Timings           json.RawMessage `json:"timings"`
	}
	if err := json.Unmarshal(responseBody, &out); err != nil || len(out.Choices) == 0 || out.Choices[0].Message.Content == nil {
		return ChatCompletionResult{}, &Error{Kind: "malformed_response", Message: "expected a chat completion message"}
	}
	text := strings.TrimSpace(*out.Choices[0].Message.Content)
	if key != "" && strings.Contains(text, key) {
		return ChatCompletionResult{}, &Error{Kind: "credential_reflection", Message: "post-processing response rejected"}
	}
	metadata := metadataFromHeaders(resp.Header, key)
	if requestID := safePeerString(out.RequestID, key); requestID != "" {
		metadata.RequestID = requestID
	}
	metadata.ResponseID = safePeerString(out.ID, key)
	metadata.EffectiveModel = safePeerString(out.Model, key)
	metadata.Provider = safePeerString(out.Provider, key)
	metadata.FinishReason = safePeerString(out.Choices[0].FinishReason, key)
	metadata.ServiceTier = safePeerString(out.ServiceTier, key)
	metadata.SystemFingerprint = safePeerString(out.SystemFingerprint, key)
	metadata.CreatedAtUnix = optionalInt(out.Created)
	metadata.RequestCount = 1
	applyUsageMetadata(&metadata, out.Usage, key)
	applyPerformanceMetadata(&metadata, out.Timings)
	result := ChatCompletionResult{Metadata: sanitizeResponseMetadata(metadata, key)}
	// Interpret protocol control fields before metadata redaction: a credential
	// can coincide with "length", but must not turn a truncated result into a
	// successful cleanup. Keep safe diagnostics without exposing partial text.
	if out.Choices[0].FinishReason == "length" {
		return result, &Error{Kind: "incomplete_response", Message: "post-processing reached the server output limit"}
	}
	result.Text = text
	return result, nil
}
