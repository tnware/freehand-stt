package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/coder/websocket"
	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/modelprofile"
	"github.com/tnware/freehand-stt/internal/speechlanguage"
)

func configureNeMo(ctx context.Context, conn *websocket.Conn, cfg config.VoiceTranscriptionSettings, created wireEvent) error {
	if cfg.Model != "" && created.Session.Model != cfg.Model {
		return errors.New("the realtime server loaded a different model; refresh its model selection")
	}
	phrases := []string{}
	for line := range strings.SplitSeq(cfg.RealtimeOptions().Vocabulary, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			phrases = append(phrases, line)
		}
	}
	nemo := cfg.TranscriptionOptions.NeMo
	options := map[string]any{"sample_rate": audio.SampleRate, "language": cfg.Language, "automatic_punctuation": !nemo.DisablePunctuation, "verbatim": !nemo.Normalize, "profanity_filter": nemo.ProfanityFilter, "speaker_diarization": false}
	if nemo.EndpointingMilliseconds != 0 {
		options["endpointing_ms"] = nemo.EndpointingMilliseconds
	}
	if len(phrases) > 0 {
		options["speech_contexts"] = []any{map[string]any{"phrases": phrases, "boost": cfg.RealtimeOptions().Boost}}
	}
	message, _ := json.Marshal(map[string]any{"type": "session.update", "session": options})
	if conn.Write(ctx, websocket.MessageText, message) != nil {
		return errors.New("realtime server configuration was not accepted")
	}
	updated, err := readEvent(ctx, conn)
	if err != nil || updated.Type != "session.updated" {
		return errors.New("realtime server configuration was not accepted")
	}
	return nil
}

func (s *Session) readNeMo(guard credentialGuard, publish func(Update)) Result {
	update := Update{Turn: 1}
	var languages []string
	for {
		event, err := readEvent(s.ctx, s.conn)
		if err != nil {
			return Result{Err: err}
		}
		switch event.Type {
		case "conversation.item.input_audio_transcription.delta":
			if len(update.Final)+len(update.Partial)+len(event.Delta) > MaxTranscriptBytes {
				return Result{Err: errors.New("live transcript exceeded its size limit")}
			}
			update.Partial += event.Delta
			if err := guard.check(update.Partial, update.Final+" "+update.Partial); err != nil {
				return Result{Err: err}
			}
			if event.Delta != "" && publish != nil {
				publish(update)
			}
		case "conversation.item.input_audio_transcription.completed":
			if err := guard.check(append(event.Languages, event.Language, event.Transcript)...); err != nil {
				return Result{Err: err}
			}
			text, detected := modelprofile.StripNemotronLanguageTag(event.Transcript)
			for _, candidate := range append(event.Languages, event.Language, detected) {
				if speechlanguage.Unspecified(candidate) {
					continue
				}
				// Structured fields are optional in newer servers. Retain only qualified
				// locale codes; unknown evidence blocks language-restricted cleanup without
				// publishing arbitrary peer strings or reflected credentials.
				if modelprofile.ValidateNemotron(candidate, modelprofile.NemotronOptions{}) != nil {
					candidate = "mul"
				}
				languages = speechlanguage.MergeDetected(languages, []string{candidate})
			}
			if len(update.Final)+len(text)+1 > MaxTranscriptBytes || update.Turn > 1024 {
				return Result{Err: errors.New("live transcript exceeded its size limit")}
			}
			update.Final = strings.TrimSpace(update.Final + " " + text)
			if err := guard.check(update.Final); err != nil {
				return Result{Err: err}
			}
			update.Partial = ""
			update.Turn++
			if publish != nil {
				publish(update)
			}
		case "input_audio_buffer.committed":
			if !s.committed.Load() || update.Turn == 1 || update.Partial != "" {
				return Result{Err: errors.New("realtime completion was missing its final transcript")}
			}
			return Result{Text: update.Final, Languages: languages}
		case "error", "input_audio_buffer.cleared":
			return Result{Err: errors.New("realtime server could not finish this recording")}
		}
	}
}
