package storage

import (
	"strings"
	"testing"

	"github.com/tnware/freehand-stt/internal/config"
)

func TestBaselineRejectsRemainingBroadBounds(t *testing.T) {
	s := testStore(t)
	loadStore(t, s)
	for _, tc := range []struct {
		query string
		value any
	}{
		{`UPDATE preferences_settings SET segment_seconds=?`, 181},
		{`UPDATE preferences_settings SET segment_seconds=?`, 14},
		{`UPDATE preferences_settings SET segment_silence_ms=?`, 199},
		{`UPDATE preferences_settings SET segment_silence_ms=?`, 3001},
		{`UPDATE preferences_settings SET max_duration_seconds=?`, 263},
		{`UPDATE preferences_settings SET silence_splitting=1,max_duration_seconds=?`, 3601},
		{`UPDATE preferences_settings SET microphone_id=?`, strings.Repeat("x", 1025)},
		{`UPDATE transcription_settings SET language=?`, strings.Repeat("x", 33)},
		{`UPDATE voice_transcription_settings SET language=?`, strings.Repeat("日", 11)},
		{`UPDATE cleanup_settings SET styling=?`, strings.Repeat("x", 33)},
		{`UPDATE cleanup_settings SET structure=?`, strings.Repeat("x", 33)},
		{`UPDATE cleanup_settings SET context=?`, strings.Repeat("x", 33)},
		{`UPDATE cleanup_settings SET preset=?`, "invalid"},
	} {
		if _, err := s.db.Exec(tc.query, tc.value); err == nil {
			t.Errorf("invalid scalar accepted: %s", tc.query)
		}
	}
}

func TestBaselineRejectsInvalidTaskScalars(t *testing.T) {
	s := testStore(t)
	loadStore(t, s)
	for _, column := range []string{"appearance_mode", "overlay_layout", "overlay_anchor", "overlay_visibility", "overlay_motion", "overlay_surface", "overlay_visualizer", "vad_mode"} {
		t.Run(column, func(t *testing.T) {
			if _, err := s.db.Exec(`UPDATE preferences_settings SET ` + column + `='invalid'`); err == nil {
				t.Fatal("invalid enum accepted")
			}
		})
	}
	for _, tc := range []struct {
		column   string
		min, max int
	}{
		{"vad_activity_silence_ms", config.MinVADActivitySilenceMS, config.MaxVADActivitySilenceMS},
		{"speech_padding_ms", config.MinSpeechPaddingMS, config.MaxSpeechPaddingMS},
		{"auto_stop_silence_ms", config.MinAutoStopSilenceMS, config.MaxAutoStopSilenceMS},
		{"auto_stop_minimum_speech_ms", config.MinAutoStopSpeechMS, config.MaxAutoStopSpeechMS},
	} {
		t.Run(tc.column, func(t *testing.T) {
			for _, v := range []int{tc.min - 1, tc.max + 1} {
				if _, err := s.db.Exec(`UPDATE preferences_settings SET `+tc.column+`=?`, v); err == nil {
					t.Errorf("out-of-range value %d accepted", v)
				}
			}
			for _, v := range []int{tc.min, tc.max} {
				if _, err := s.db.Exec(`UPDATE preferences_settings SET `+tc.column+`=?`, v); err != nil {
					t.Errorf("valid boundary %d rejected: %v", v, err)
				}
			}
		})
	}
	for _, tc := range []struct{ table, column string }{
		{"transcription_settings", "model"}, {"cleanup_settings", "model"}, {"speech_settings", "model"}, {"speech_settings", "voice"}, {"voice_transcription_settings", "model"},
	} {
		t.Run(tc.table+"_"+tc.column, func(t *testing.T) {
			query := `UPDATE ` + tc.table + ` SET ` + tc.column + `=?`
			if _, err := s.db.Exec(query, strings.Repeat("x", 201)); err == nil {
				t.Error("oversize model/voice accepted")
			}
			if _, err := s.db.Exec(query, strings.Repeat("日", 67)); err == nil {
				t.Error("multibyte model/voice byte limit not enforced")
			}
			if _, err := s.db.Exec(query, strings.Repeat("x", 200)); err != nil {
				t.Fatalf("valid model/voice boundary rejected: %v", err)
			}
		})
	}
}
