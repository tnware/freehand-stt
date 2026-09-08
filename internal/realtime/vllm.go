package realtime

import (
	"context"
	"encoding/base64"
	"errors"

	"github.com/coder/websocket"
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

func (s *Session) writeAudio(ctx context.Context, frame []byte) error {
	if s.backend != compatibility.VLLM {
		return s.conn.Write(ctx, websocket.MessageBinary, frame)
	}
	// Base64 contains no JSON delimiters. Avoid a persistent string copy of audio.
	message := append([]byte(`{"type":"input_audio_buffer.append","audio":"`), make([]byte, base64.StdEncoding.EncodedLen(len(frame)))...)
	base64.StdEncoding.Encode(message[len(`{"type":"input_audio_buffer.append","audio":"`):], frame)
	message = append(message, '"', '}')
	defer clear(message)
	return s.conn.Write(ctx, websocket.MessageText, message)
}

// vLLM 0.28.0 emits transcription.delta/done, not the NeMo conversation events.
// Only done after our stop request can yield deliverable text. A disconnected
// stream, server error, or premature done never promotes the preview.
func (s *Session) readVLLM(publish func(Update)) Result {
	raw := ""
	for {
		event, err := readEvent(s.ctx, s.conn)
		if err != nil {
			return Result{Err: err}
		}
		switch event.Type {
		case "transcription.delta":
			if len(raw)+len(event.Delta) > MaxTranscriptBytes {
				return Result{Err: errors.New("live transcript exceeded its size limit")}
			}
			raw += event.Delta
			text, _ := modelprofile.QwenRealtimeText(raw, false)
			if publish != nil {
				publish(Update{Partial: text, Turn: 1})
			}
		case "transcription.done":
			if !s.committed.Load() || event.Text == nil {
				return Result{Err: errors.New("realtime server finished before recording stopped")}
			}
			if len(*event.Text) > MaxTranscriptBytes {
				return Result{Err: errors.New("live transcript exceeded its size limit")}
			}
			text, language := modelprofile.QwenRealtimeText(*event.Text, true)
			if publish != nil {
				publish(Update{Final: text, Turn: 2})
			}
			return Result{Text: text, Language: language}
		case "error":
			return Result{Err: errors.New("realtime server could not finish this recording")}
		}
	}
}
