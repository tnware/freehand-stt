package inference

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

// vLLM v0.28.0 emits choices[].delta.content and per-audio-chunk finish
// reasons, then [DONE] for the entire file. A chunk's stop is not file completion.
func readVLLMTranscriptionSSE(reader io.Reader, key string, onDelta func(string)) (TranscriptionResult, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	var text strings.Builder
	metadata := ResponseMetadata{RequestCount: 1}
	seen, finished, done := false, false, false
	readBytes := 0
	fail := func(message string) (TranscriptionResult, error) {
		return TranscriptionResult{Text: text.String(), Metadata: sanitizeResponseMetadata(metadata, key)}, &Error{Kind: "response", Message: message}
	}
	for scanner.Scan() {
		line := scanner.Text()
		readBytes += len(line) + 1
		if readBytes > maxFileResponse {
			return fail("transcription response is too large")
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		if done {
			return fail("transcription stream continued after completion")
		}
		if data == "[DONE]" {
			if !seen || !finished {
				return fail("transcription stream ended before its final chunk")
			}
			done = true
			continue
		}
		var event struct {
			Object  string          `json:"object"`
			ID      string          `json:"id"`
			Model   string          `json:"model"`
			Created json.RawMessage `json:"created"`
			Usage   json.RawMessage `json:"usage"`
			Error   json.RawMessage `json:"error"`
			Choices []struct {
				Index *int `json:"index"`
				Delta *struct {
					Content *string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}
		if json.Unmarshal([]byte(data), &event) != nil {
			return fail("invalid vLLM transcription event")
		}
		if len(event.Error) != 0 && string(event.Error) != "null" {
			return fail("transcription stream failed")
		}
		if event.Object != "transcription.chunk" || event.Choices == nil {
			return fail("incompatible vLLM transcription event")
		}
		if len(event.Choices) == 0 {
			if !seen || !finished || len(event.Usage) == 0 || string(event.Usage) == "null" {
				return fail("invalid transcription usage event")
			}
		} else {
			if len(event.Choices) != 1 {
				return fail("unexpected transcription choices")
			}
			choice := event.Choices[0]
			if (choice.Index != nil && *choice.Index != 0) || choice.Delta == nil || choice.Delta.Content == nil {
				return fail("invalid transcription delta")
			}
			delta := *choice.Delta.Content
			if reflectsCredential(text.String(), delta, key) {
				return TranscriptionResult{}, &Error{Kind: "credential_reflection", Message: "transcription response rejected"}
			}
			text.WriteString(delta)
			if onDelta != nil && delta != "" {
				onDelta(delta)
			}
			seen = true
			finished = choice.FinishReason != nil
			if finished && *choice.FinishReason != "stop" {
				return fail("transcription chunk did not finish successfully")
			}
		}
		metadata.ResponseID = safePeerString(event.ID, key)
		metadata.EffectiveModel = safePeerString(event.Model, key)
		metadata.CreatedAtUnix = optionalInt(event.Created)
		applyUsageMetadata(&metadata, event.Usage, key)
	}
	if scanner.Err() != nil {
		return fail("transcription stream could not be read")
	}
	if !done {
		return fail("transcription stream ended before completion")
	}
	return TranscriptionResult{Text: text.String(), Metadata: sanitizeResponseMetadata(metadata, key)}, nil
}
