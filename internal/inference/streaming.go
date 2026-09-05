package inference

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tnware/freehand-stt/internal/compatibility"
)

func readTranscriptionSSE(reader io.Reader, key string, onDelta func(string)) (TranscriptionResult, error) {
	contract, _ := compatibility.Resolve(compatibility.Generic, compatibility.Transcription)
	return readTranscriptionSSEContract(reader, key, onDelta, contract.Capabilities)
}

func readTranscriptionSSEContract(reader io.Reader, key string, onDelta func(string), caps compatibility.Capabilities) (TranscriptionResult, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	var accumulated strings.Builder
	var completed *string
	var metadata ResponseMetadata
	typedStream := false
	// Only already accepted, credential-checked text may survive an incomplete
	// response. Keep this a response failure, not evidence to disable streaming.
	incomplete := func(message string) (TranscriptionResult, error) {
		return TranscriptionResult{Text: accumulated.String(), Metadata: ResponseMetadata{RequestCount: 1}}, &Error{Kind: "response", Message: message}
	}
	readBytes := 0
	dataEvents := 0
	recognizedEvents := 0
	for scanner.Scan() {
		line := scanner.Text()
		readBytes += len(line) + 1
		if readBytes > maxFileResponse {
			return TranscriptionResult{}, &Error{Kind: "response_too_large", Message: "transcription response is too large"}
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		dataEvents++
		var event struct {
			Type              string          `json:"type"`
			Delta             *string         `json:"delta"`
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
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return TranscriptionResult{}, &FileStreamUnsupportedError{Reason: "invalid_sse_event", PartialText: accumulated.String()}
		}
		switch event.Type {
		case "transcript.text.delta":
			if !caps.TypedTranscriptionEvents {
				continue
			}
			typedStream = true
			if event.Delta == nil {
				return TranscriptionResult{}, &FileStreamUnsupportedError{Reason: "invalid_sse_event", PartialText: accumulated.String()}
			}
			recognizedEvents++
			if reflectsCredential(accumulated.String(), *event.Delta, key) {
				return TranscriptionResult{}, &Error{Kind: "credential_reflection", Message: "transcription response rejected"}
			}
			accumulated.WriteString(*event.Delta)
			if onDelta != nil {
				onDelta(*event.Delta)
			}
		case "transcript.text.done":
			if !caps.TypedTranscriptionEvents {
				continue
			}
			typedStream = true
			if event.Text == nil {
				return TranscriptionResult{}, &FileStreamUnsupportedError{Reason: "invalid_sse_event", PartialText: accumulated.String()}
			}
			recognizedEvents++
			if key != "" && strings.Contains(*event.Text, key) {
				return TranscriptionResult{}, &Error{Kind: "credential_reflection", Message: "transcription response rejected"}
			}
			completed = event.Text
			metadata.ResponseID = safePeerString(event.ID, key)
			metadata.RequestID = safePeerString(event.RequestID, key)
			metadata.EffectiveModel = safePeerString(event.Model, key)
			metadata.Provider = safePeerString(event.Provider, key)
			metadata.CreatedAtUnix = optionalInt(event.Created)
			metadata.DetectedLanguages = parseLanguages(event.Languages, event.Language, key)
			metadata.ServerAudioSeconds = optionalFloat(event.Duration)
			metadata.ServiceTier = safePeerString(event.ServiceTier, key)
			metadata.SystemFingerprint = safePeerString(event.SystemFingerprint, key)
			applyUsageMetadata(&metadata, event.Usage, key)
			applyPerformanceMetadata(&metadata, event.Timings)
		case "error":
			return incomplete("transcription stream failed")
		case "":
			// Speaches <=0.8 emits one untyped {"text": ...} event per
			// segment and signals completion by closing the response.
			if event.Text != nil && caps.LegacyTranscriptionSegments {
				recognizedEvents++
				delta := legacySegmentDelta(accumulated.String(), *event.Text)
				if reflectsCredential(accumulated.String(), delta, key) {
					return TranscriptionResult{}, &Error{Kind: "credential_reflection", Message: "transcription response rejected"}
				}
				accumulated.WriteString(delta)
				if onDelta != nil {
					onDelta(delta)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return incomplete("transcription stream could not be read")
	}
	if dataEvents > 0 && recognizedEvents == 0 {
		return TranscriptionResult{}, &FileStreamUnsupportedError{Reason: "incompatible_sse_contract"}
	}
	if recognizedEvents == 0 || (typedStream && completed == nil) {
		return incomplete("transcription stream ended before its final transcript")
	}
	metadata.RequestCount = 1
	if completed != nil {
		return TranscriptionResult{Text: *completed, Metadata: metadata}, nil
	}
	return TranscriptionResult{Text: accumulated.String(), Metadata: metadata}, nil
}

func legacySegmentDelta(existing, segment string) string {
	if existing == "" || segment == "" {
		return segment
	}
	last, _ := utf8.DecodeLastRuneInString(existing)
	first, _ := utf8.DecodeRuneInString(segment)
	if unicode.IsSpace(last) || unicode.IsSpace(first) || strings.ContainsRune(",.!?;:%)]}", first) || strings.ContainsRune("([{", last) {
		return segment
	}
	return " " + segment
}

func reflectsCredential(existing, next, key string) bool {
	if key == "" {
		return false
	}
	if strings.Contains(next, key) {
		return true
	}
	tailBytes := len(key) - 1
	if tailBytes <= 0 {
		return false
	}
	if len(existing) > tailBytes {
		existing = existing[len(existing)-tailBytes:]
	}
	return strings.Contains(existing+next, key)
}
