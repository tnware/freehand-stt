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

const (
	maxFileResponse    = 8 << 20
	maxStreamRejection = 64 << 10
)

// FileTranscriptionCallbacks reports bounded, non-audio progress while a
// stored file is uploaded and, for SSE responses, transcript text arrives.
type FileTranscriptionCallbacks struct {
	UploadProgress    func(sent, total int64)
	UploadComplete    func()
	Delta             func(string)
	StreamBuffered    func()
	StreamUnsupported func(reason string)
}

// TranscribeFile sends a stored audio file without buffering it in memory.
// The caller owns and closes r. A streaming request accepts both current
// OpenAI typed events and the segment-shaped SSE emitted by older Speaches
// releases. Some compatible peers ignore stream=true and return JSON; that is
// treated as a successful completed response rather than retried.
func (c *Client) TranscribeFile(ctx context.Context, base, model, language, key string, headers map[string]string, filename string, size int64, r io.Reader, stream bool, callbacks FileTranscriptionCallbacks) (TranscriptionResult, error) {
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
	if stream && !contract.Capabilities.FileStreaming {
		return TranscriptionResult{}, &FileStreamUnsupportedError{Reason: "profile_streaming_unavailable"}
	}
	if size <= 0 {
		return TranscriptionResult{}, &Error{Kind: "invalid_file", Message: "audio file is empty"}
	}
	u, err := endpoint(base, contract.Path)
	if err != nil {
		return TranscriptionResult{}, err
	}

	pipeReader, pipeWriter := io.Pipe()
	mw := multipart.NewWriter(pipeWriter)
	boundary := mw.Boundary()
	contentLength, err := multipartLength(boundary, filename, size, model, language, stream, c.transcriptionOptions)
	if err != nil {
		_ = pipeReader.Close()
		_ = pipeWriter.Close()
		return TranscriptionResult{}, err
	}
	writeDone := make(chan error, 1)
	go func() {
		writeErr := writeFileMultipart(mw, filename, model, language, stream, c.transcriptionOptions, &progressReader{
			reader: r,
			total:  size,
			onRead: callbacks.UploadProgress,
		})
		if writeErr == nil {
			writeErr = mw.Close()
		}
		if writeErr != nil {
			_ = pipeWriter.CloseWithError(writeErr)
		} else {
			_ = pipeWriter.Close()
			if callbacks.UploadComplete != nil {
				callbacks.UploadComplete()
			}
		}
		writeDone <- writeErr
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, pipeReader)
	if err != nil {
		_ = pipeReader.Close()
		return TranscriptionResult{}, err
	}
	req.ContentLength = contentLength
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json")
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		_ = pipeReader.Close()
		<-writeDone
		return TranscriptionResult{}, requestFailure(err, ctx, "audio file transcription")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = pipeReader.Close()
		<-writeDone
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxStreamRejection+1))
		if stream && len(body) <= maxStreamRejection && rejectsStreamingParameter(resp.StatusCode, body) {
			return TranscriptionResult{}, &FileStreamUnsupportedError{Reason: "stream_parameter_rejected"}
		}
		message := "transcription request failed"
		if resp.StatusCode == http.StatusRequestEntityTooLarge {
			message = "audio file exceeds the server upload limit"
		}
		return TranscriptionResult{}, &Error{Kind: "http", Status: resp.StatusCode, Message: message}
	}
	// A peer may send successful response headers while the transport is still
	// consuming the multipart pipe. Keep the reader alive until the writer
	// has finished; closing it as soon as Do returns truncates larger uploads.
	writeErr := <-writeDone
	_ = pipeReader.Close()
	if writeErr != nil && ctx.Err() == nil {
		return TranscriptionResult{}, &Error{Kind: "request", Message: "audio file could not be uploaded"}
	}

	var result TranscriptionResult
	if stream && strings.HasPrefix(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream") {
		result, err = readTranscriptionSSEContract(resp.Body, key, callbacks.Delta, contract.Capabilities)
	} else {
		result, err = readTranscriptionJSON(resp.Body, maxFileResponse, key)
		if err == nil && stream {
			if callbacks.StreamBuffered != nil {
				callbacks.StreamBuffered()
			}
			if looksLikeBufferedTranscriptionSSE(result.Text) {
				result, err = readTranscriptionSSEContract(strings.NewReader(result.Text), key, callbacks.Delta, contract.Capabilities)
			} else if callbacks.StreamUnsupported != nil {
				callbacks.StreamUnsupported("completed_json")
			}
		}
	}
	result.Text = strings.TrimSpace(result.Text)
	if key != "" && strings.Contains(result.Text, key) {
		return TranscriptionResult{}, &Error{Kind: "credential_reflection", Message: "transcription response rejected"}
	}
	if result.Metadata.RequestID == "" {
		result.Metadata.RequestID = metadataFromHeaders(resp.Header, key).RequestID
	}
	result.Metadata.RequestCount = 1
	result.Metadata = sanitizeResponseMetadata(result.Metadata, key)
	// A failed stream can carry accepted partial text for manual recovery.
	// The caller must retain the error and must not process it as a success.
	return result, err
}

func rejectsStreamingParameter(status int, body []byte) bool {
	switch status {
	case http.StatusBadRequest, http.StatusMethodNotAllowed, http.StatusUnsupportedMediaType,
		http.StatusUnprocessableEntity, http.StatusNotImplemented:
	default:
		return false
	}
	var payload any
	if json.Unmarshal(body, &payload) != nil {
		return false
	}
	normalized := strings.ToLower(string(body))
	if !strings.Contains(normalized, "stream") {
		return false
	}
	for _, evidence := range []string{
		"not supported", "unsupported", "not implemented", "unknown parameter",
		"unrecognized", "unexpected", "extra_forbidden", "not permitted",
	} {
		if strings.Contains(normalized, evidence) {
			return true
		}
	}
	return false
}

func looksLikeBufferedTranscriptionSSE(text string) bool {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "data:") {
		return false
	}
	line, _, _ := strings.Cut(trimmed, "\n")
	data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	var event struct {
		Type  string  `json:"type"`
		Delta *string `json:"delta"`
		Text  *string `json:"text"`
	}
	return json.Unmarshal([]byte(data), &event) == nil && (event.Type != "" || event.Delta != nil || event.Text != nil)
}

type progressReader struct {
	reader io.Reader
	total  int64
	sent   int64
	onRead func(int64, int64)
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.sent += int64(n)
		if r.onRead != nil {
			r.onRead(r.sent, r.total)
		}
	}
	return n, err
}

func writeFileMultipart(mw *multipart.Writer, filename, model, language string, stream bool, options compatibility.TranscriptionOptions, file io.Reader) error {
	if err := mw.WriteField("model", model); err != nil {
		return err
	}
	if language != "" {
		if err := mw.WriteField("language", language); err != nil {
			return err
		}
	}
	if err := mw.WriteField("response_format", "json"); err != nil {
		return err
	}
	if stream {
		if err := mw.WriteField("stream", "true"); err != nil {
			return err
		}
	}
	if err := writeTranscriptionOptions(mw, options); err != nil {
		return err
	}
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, file)
	return err
}

func multipartLength(boundary, filename string, size int64, model, language string, stream bool, options compatibility.TranscriptionOptions) (int64, error) {
	var overhead bytes.Buffer
	mw := multipart.NewWriter(&overhead)
	if err := mw.SetBoundary(boundary); err != nil {
		return 0, err
	}
	if err := writeFileMultipart(mw, filename, model, language, stream, options, strings.NewReader("")); err != nil {
		return 0, err
	}
	if err := mw.Close(); err != nil {
		return 0, err
	}
	return int64(overhead.Len()) + size, nil
}

func readTranscriptionJSON(reader io.Reader, limit int64, key string) (TranscriptionResult, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return TranscriptionResult{}, &Error{Kind: "response", Message: "transcription response could not be read"}
	}
	if int64(len(body)) > limit {
		return TranscriptionResult{}, &Error{Kind: "response_too_large", Message: "transcription response is too large"}
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
