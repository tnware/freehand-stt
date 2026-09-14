package managedruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/tnware/freehand-stt/internal/audio"
)

var errWarmup = errors.New("GPU warm-up failed. Try starting the runtime again or select CPU.")

// warmWhisper is startup work, never a health probe. The pinned v1.8.3
// /inference handler executes audio but has no startup warm-up operation.
// One second avoids its short-input early return. See the pinned source
// evidence in testdata/ggml-startup-source.md. Output is discarded.
func warmWhisper(ctx context.Context, base string) error {
	u, err := url.Parse(base)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return errWarmup
	}
	bounded, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	wav, err := audio.PCM16WAV(make([]byte, 16000*2), 16000, 1)
	if err != nil {
		return errWarmup
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", "warmup.wav")
	if err != nil {
		return errWarmup
	}
	if _, err = file.Write(wav); err != nil {
		return errWarmup
	}
	for _, field := range [][2]string{
		{"response_format", "json"}, {"language", "en"}, {"detect_language", "false"},
		{"translate", "false"}, {"temperature", "0"}, {"temperature_inc", "0"},
		{"best_of", "1"}, {"beam_size", "1"}, {"audio_ctx", "0"}, {"offset_t", "0"},
		{"duration", "1000"}, {"no_timestamps", "true"}, {"debug_mode", "false"},
		{"vad", "false"}, {"prompt", ""},
	} {
		if err = writer.WriteField(field[0], field[1]); err != nil {
			return errWarmup
		}
	}
	if err = writer.Close(); err != nil {
		return errWarmup
	}
	request, err := http.NewRequestWithContext(bounded, http.MethodPost, base+"/inference", &body)
	if err != nil {
		return errWarmup
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	client := loopbackClient()
	// The startup context owns the inference deadline; metadata's two-second
	// response timeout is inappropriate for first-use GPU initialization.
	client.Timeout = 0
	defer client.CloseIdleConnections()
	response, err := client.Do(request)
	if err != nil {
		if bounded.Err() != nil {
			return bounded.Err()
		}
		return errWarmup
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errWarmup
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
	if bounded.Err() != nil {
		return bounded.Err()
	}
	if err != nil || len(data) > 64<<10 {
		return errWarmup
	}
	var result struct {
		Text  *string         `json:"text"`
		Error json.RawMessage `json:"error"`
	}
	if json.Unmarshal(data, &result) != nil || result.Text == nil || len(result.Error) > 0 {
		return errWarmup
	}
	return nil
}
