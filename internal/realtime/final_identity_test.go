package realtime

import (
	"context"
	"github.com/coder/websocket"
	"strings"
	"testing"
)

func TestNeMoFinalEventIdentity(t *testing.T) {
	for _, tc := range []struct {
		name    string
		events  []string
		want    string
		updates int
		fail    bool
	}{
		{"duplicate", []string{`{"event_id":"one","transcript":"hello"}`, `{"event_id":"one","transcript":"hello"}`}, "hello", 1, false},
		{"repeated-phrase", []string{`{"event_id":"one","transcript":"hello"}`, `{"event_id":"two","transcript":"hello"}`}, "hello hello", 2, false},
		{"conflicting", []string{`{"event_id":"one","transcript":"hello"}`, `{"event_id":"one","transcript":"changed"}`}, "", 1, true},
		{"legacy", []string{`{"transcript":"hello"}`}, "hello", 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := fixtureServer(t, func(ctx context.Context, c *websocket.Conn) {
				for _, event := range tc.events {
					event = strings.Replace(event, `{`, `{"type":"conversation.item.input_audio_transcription.completed",`, 1)
					if err := c.Write(ctx, websocket.MessageText, []byte(event)); err != nil {
						return
					}
				}
				for {
					kind, _, err := c.Read(ctx)
					if err != nil {
						return
					}
					if kind == websocket.MessageText {
						_ = c.Write(ctx, websocket.MessageText, []byte(`{"type":"input_audio_buffer.committed"}`))
						return
					}
				}
			}, nil)
			defer server.Close()
			var updates []Update
			session, err := Open(context.Background(), fixtureConfig(server), "fixture-key", func(u Update) {
				updates = append(updates, u)
			})
			if err != nil {
				t.Fatal(err)
			}
			session.Pipe.Close()
			result := session.Wait()
			if (result.Err != nil) != tc.fail || result.Text != tc.want || len(updates) != tc.updates {
				t.Fatalf("result=%+v updates=%+v", result, updates)
			}
			for _, u := range updates {
				if !u.Finalized || (tc.name != "legacy" && u.EventID == "") {
					t.Fatalf("missing final authority: %+v", u)
				}
			}
		})
	}
}
