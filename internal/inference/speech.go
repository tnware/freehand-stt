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

const maxSpeechResponse = 32 << 20

type SpeechRequest struct {
	CompatibilityProfile compatibility.ID
	Model                string
	Voice                string
	Input                string
	Speed                float64
}

// SynthesizeSpeech invokes the standard OpenAI-compatible speech endpoint.
// Freehand requests WAV so native playback stays deterministic and requires no
// compressed-audio decoder or helper process.
func (c *Client) SynthesizeSpeech(ctx context.Context, base, key string, input SpeechRequest) ([]byte, error) {
	contract, err := c.WithCompatibility(input.CompatibilityProfile).contract(compatibility.Speech)
	if err != nil {
		return nil, err
	}
	if input.Speed != 1 && !contract.Capabilities.SpeechSpeed {
		return nil, &Error{Kind: "invalid_settings", Message: "speech speed is unavailable for this profile"}
	}
	request := struct {
		Model          string  `json:"model"`
		Input          string  `json:"input"`
		Voice          string  `json:"voice"`
		ResponseFormat string  `json:"response_format"`
		Speed          float64 `json:"speed"`
	}{Model: input.Model, Input: input.Input, Voice: input.Voice, ResponseFormat: "wav", Speed: input.Speed}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	defer zeroBytes(body)
	u, err := endpoint(base, contract.Path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/wav, audio/*")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, requestFailure(err, ctx, "speech generation")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		return nil, &Error{Kind: "http", Status: resp.StatusCode, Message: "speech generation request failed"}
	}
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "json") || strings.Contains(contentType, "text/") {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
		return nil, &Error{Kind: "unexpected_response", Message: "speech endpoint did not return audio"}
	}
	audio, err := io.ReadAll(io.LimitReader(resp.Body, maxSpeechResponse+1))
	if err != nil {
		return nil, &Error{Kind: "response", Message: "speech audio could not be read"}
	}
	if len(audio) > maxSpeechResponse {
		zeroBytes(audio)
		return nil, &Error{Kind: "response_too_large", Message: "speech audio exceeds 32 MiB"}
	}
	if len(audio) == 0 {
		return nil, &Error{Kind: "empty_response", Message: "speech endpoint returned no audio"}
	}
	return audio, nil
}
