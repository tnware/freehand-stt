package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/tnware/freehand-stt/internal/audio"
	"github.com/tnware/freehand-stt/internal/compatibility"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/modelprofile"
)

func fixtureServer(t *testing.T, after func(context.Context, *websocket.Conn), options chan<- map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/transcriptions/realtime" || r.URL.RawQuery != "" || r.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Error("unexpected upgrade target or authentication")
			http.Error(w, "bad request", 400)
			return
		}
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		_ = c.Write(ctx, websocket.MessageText, []byte(`{"type":"session.created","session":{"model":"fixture-model"}}`))
		_, b, err := c.Read(ctx)
		if err != nil {
			return
		}
		var update map[string]any
		if json.Unmarshal(b, &update) != nil {
			t.Error("invalid session update")
			return
		}
		if options != nil {
			options <- update
		}
		_ = c.Write(ctx, websocket.MessageText, []byte(`{"type":"session.updated"}`))
		after(ctx, c)
	}))
}

func fixtureConfig(server *httptest.Server) config.VoiceTranscriptionSettings {
	v := config.DefaultVoiceTranscription()
	v.Realtime = true
	v.CompatibilityProfile = "nemo-speech-v1"
	v.ModelProfile = "nemotron-3.5-streaming"
	v.BaseURL = server.URL + "/v1"
	v.Model = "fixture-model"
	v.AuthenticationMode = config.AuthenticationModeAPIKey
	v.AllowInsecureHTTP = true
	return v
}

func TestNeMoControlsAndMixedFinalLanguages(t *testing.T) {
	opts := make(chan map[string]any, 1)
	server := fixtureServer(t, func(ctx context.Context, c *websocket.Conn) {
		for {
			kind, body, err := c.Read(ctx)
			if err != nil {
				return
			}
			if kind == websocket.MessageText && strings.Contains(string(body), "input_audio_buffer.commit") {
				for _, event := range []string{
					`{"type":"conversation.item.input_audio_transcription.completed","transcript":"Bonjour. <fr-FR>"}`,
					`{"type":"conversation.item.input_audio_transcription.completed","transcript":"Hello.","language":"en-US","languages":["de-DE"]}`,
					`{"type":"input_audio_buffer.committed"}`,
				} {
					_ = c.Write(ctx, websocket.MessageText, []byte(event))
				}
				return
			}
		}
	}, opts)
	defer server.Close()
	cfg := fixtureConfig(server)
	cfg.TranscriptionOptions.NeMo = compatibility.NeMoOptions{DisablePunctuation: true, Normalize: true, ProfanityFilter: true, EndpointingMilliseconds: 1200}
	session, err := Open(t.Context(), cfg, "fixture-key", nil)
	if err != nil {
		t.Fatal(err)
	}
	cfg.TranscriptionOptions.NeMo = compatibility.NeMoOptions{}
	session.Pipe.Close()
	result := session.Wait()
	if result.Err != nil || result.Text != "Bonjour. Hello." || strings.Join(result.Languages, ",") != "fr-FR,de-DE,en-US" {
		t.Fatalf("lost language evidence: %+v", result)
	}
	if err := modelprofile.ValidateLanguage(modelprofile.S1Mini, compatibility.PostProcessing, "auto", result.Languages); err == nil {
		t.Fatal("English-only cleanup admitted a multilingual final")
	}
	fields := (<-opts)["session"].(map[string]any)
	if fields["automatic_punctuation"] != false || fields["verbatim"] != false || fields["profanity_filter"] != true || fields["endpointing_ms"] != float64(1200) {
		t.Fatalf("controls were not snapshotted: %v", fields)
	}
}

func TestFinalReplacesPartialAndConfigurationUsesVocabulary(t *testing.T) {
	opts := make(chan map[string]any, 1)
	server := fixtureServer(t, func(ctx context.Context, c *websocket.Conn) {
		for {
			kind, b, err := c.Read(ctx)
			if err != nil {
				return
			}
			if kind == websocket.MessageBinary {
				_ = c.Write(ctx, websocket.MessageText, []byte(`{"type":"conversation.item.input_audio_transcription.delta","delta":"provisional words"}`))
			} else if strings.Contains(string(b), "input_audio_buffer.commit") {
				_ = c.Write(ctx, websocket.MessageText, []byte(`{"type":"conversation.item.input_audio_transcription.completed","transcript":"Correct final words. <en-US>"}`))
				_ = c.Write(ctx, websocket.MessageText, []byte(`{"type":"input_audio_buffer.committed"}`))
				return
			}
		}
	}, opts)
	defer server.Close()
	cfg := fixtureConfig(server)
	snapshot, err := config.WithVocabulary(config.Settings{VoiceTranscription: cfg, Vocabulary: config.VocabularySettings{Terms: "Freehand\nNemotron", Voice: true, Boost: 3}}, true)
	if err != nil {
		t.Fatal(err)
	}
	cfg = snapshot.VoiceTranscription
	var updates []Update
	session, err := Open(t.Context(), cfg, "fixture-key", func(u Update) { updates = append(updates, u) })
	if err != nil {
		t.Fatal(err)
	}
	if !session.Pipe.WritePCM(make([]byte, audio.VADFrameBytes)) {
		t.Fatal("frame refused")
	}
	session.Pipe.Close()
	result := session.Wait()
	if result.Err != nil || result.Text != "Correct final words." || strings.Join(result.Languages, ",") != "en-US" || result.AudioMilliseconds != 20 {
		t.Fatalf("wrong final result: %#v", result)
	}
	if len(updates) != 2 || updates[0].Partial != "provisional words" || updates[1].Partial != "" || updates[1].Final != result.Text {
		t.Fatal("final did not replace provisional text")
	}
	update := <-opts
	fields := update["session"].(map[string]any)
	if fields["sample_rate"] != float64(16000) || fields["language"] != "auto" || fields["automatic_punctuation"] != true || fields["verbatim"] != true {
		t.Fatal("incorrect qualified session options")
	}
	if _, exists := fields["prompt"]; exists {
		t.Fatal("vocabulary sent as free-form prompt")
	}
	contexts := fields["speech_contexts"].([]any)
	if contexts[0].(map[string]any)["boost"] != float64(3) {
		t.Fatal("vocabulary boost lost")
	}
}

func TestCancelDrainsCapturedAudioAndNeverReturnsPartial(t *testing.T) {
	server := fixtureServer(t, func(ctx context.Context, c *websocket.Conn) {
		for {
			if _, _, err := c.Read(ctx); err != nil {
				return
			}
		}
	}, nil)
	defer server.Close()
	ctx, cancel := context.WithCancel(t.Context())
	session, err := Open(ctx, fixtureConfig(server), "fixture-key", nil)
	if err != nil {
		t.Fatal(err)
	}
	for range 64 {
		if !session.Pipe.WritePCM(make([]byte, audio.VADFrameBytes)) {
			break
		}
	}
	cancel()
	session.Pipe.Close()
	select {
	case <-session.Done:
	case <-time.After(2 * time.Second):
		t.Fatal("cancel left streaming workers running")
	}
	result := session.Wait()
	if result.Err == nil || result.Text != "" {
		t.Fatal("cancellation returned deliverable text")
	}
}

func TestCommitWithoutFinalFailsClosed(t *testing.T) {
	server := fixtureServer(t, func(ctx context.Context, c *websocket.Conn) {
		for {
			_, b, e := c.Read(ctx)
			if e != nil {
				return
			}
			if strings.Contains(string(b), "input_audio_buffer.commit") {
				_ = c.Write(ctx, websocket.MessageText, []byte(`{"type":"input_audio_buffer.committed"}`))
				return
			}
		}
	}, nil)
	defer server.Close()
	s, err := Open(t.Context(), fixtureConfig(server), "fixture-key", nil)
	if err != nil {
		t.Fatal(err)
	}
	s.Pipe.Close()
	result := s.Wait()
	if result.Err == nil || result.Text != "" {
		t.Fatal("missing final was admitted")
	}
}
