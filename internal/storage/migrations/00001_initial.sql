-- +goose Up

CREATE TABLE preferences_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  toggle_shortcut TEXT NOT NULL CHECK (length(CAST(toggle_shortcut AS BLOB)) <= 8192),
  show_shortcut TEXT NOT NULL CHECK (length(CAST(show_shortcut AS BLOB)) <= 8192),
  hold_shortcut TEXT NOT NULL CHECK (length(CAST(hold_shortcut AS BLOB)) <= 8192),
  microphone_id TEXT NOT NULL CHECK (length(CAST(microphone_id AS BLOB)) <= 8192),
  max_duration_seconds INTEGER NOT NULL CHECK (max_duration_seconds BETWEEN 1 AND 86400),
  auto_insert INTEGER NOT NULL CHECK (auto_insert IN (0,1)),
  start_with_windows INTEGER NOT NULL CHECK (start_with_windows IN (0,1)),
  show_window_on_launch INTEGER NOT NULL CHECK (show_window_on_launch IN (0,1)),
  check_for_updates INTEGER NOT NULL CHECK (check_for_updates IN (0,1)),
  setup_completed INTEGER NOT NULL CHECK (setup_completed IN (0,1)),
  use_mica INTEGER NOT NULL CHECK (use_mica IN (0,1)),
  appearance_mode TEXT NOT NULL CHECK (length(CAST(appearance_mode AS BLOB)) <= 8192),
  overlay_enabled INTEGER NOT NULL CHECK (overlay_enabled IN (0,1)),
  overlay_size_percent INTEGER NOT NULL CHECK (overlay_size_percent BETWEEN 75 AND 150),
  overlay_opacity_percent INTEGER NOT NULL CHECK (overlay_opacity_percent BETWEEN 40 AND 100),
  overlay_top_offset INTEGER NOT NULL CHECK (overlay_top_offset BETWEEN 0 AND 240),
  overlay_glow_percent INTEGER NOT NULL CHECK (overlay_glow_percent BETWEEN 0 AND 100),
  overlay_layout TEXT NOT NULL CHECK (length(CAST(overlay_layout AS BLOB)) <= 8192),
  overlay_anchor TEXT NOT NULL CHECK (length(CAST(overlay_anchor AS BLOB)) <= 8192),
  overlay_visibility TEXT NOT NULL CHECK (length(CAST(overlay_visibility AS BLOB)) <= 8192),
  overlay_motion TEXT NOT NULL CHECK (length(CAST(overlay_motion AS BLOB)) <= 8192),
  overlay_surface TEXT NOT NULL CHECK (length(CAST(overlay_surface AS BLOB)) <= 8192),
  overlay_visualizer TEXT NOT NULL CHECK (length(CAST(overlay_visualizer AS BLOB)) <= 8192),
  history_enabled INTEGER NOT NULL CHECK (history_enabled IN (0,1)),
  vadenabled INTEGER NOT NULL CHECK (vadenabled IN (0,1)),
  vadmode TEXT NOT NULL CHECK (length(CAST(vadmode AS BLOB)) <= 8192),
  vadactivity_silence_ms INTEGER NOT NULL,
  silence_trimming INTEGER NOT NULL CHECK (silence_trimming IN (0,1)),
  speech_padding_ms INTEGER NOT NULL,
  auto_stop_enabled INTEGER NOT NULL CHECK (auto_stop_enabled IN (0,1)),
  auto_stop_silence_ms INTEGER NOT NULL,
  auto_stop_minimum_speech_ms INTEGER NOT NULL,
  silence_splitting INTEGER NOT NULL CHECK (silence_splitting IN (0,1)),
  segment_seconds INTEGER NOT NULL CHECK (segment_seconds BETWEEN 1 AND 86400),
  segment_silence_ms INTEGER NOT NULL CHECK (segment_silence_ms BETWEEN 0 AND 10000)
) STRICT;

CREATE TABLE transcription_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1) REFERENCES preferences_settings(id) ON DELETE RESTRICT,
  compatibility_profile TEXT NOT NULL CHECK (length(CAST(compatibility_profile AS BLOB)) <= 8192),
  base_url TEXT NOT NULL CHECK (length(CAST(base_url AS BLOB)) <= 8192),
  allow_insecure_http INTEGER NOT NULL CHECK (allow_insecure_http IN (0,1)),
  authentication_mode TEXT NOT NULL CHECK (length(CAST(authentication_mode AS BLOB)) <= 8192),
  model TEXT NOT NULL CHECK (length(CAST(model AS BLOB)) <= 8192),
  language TEXT NOT NULL CHECK (length(CAST(language AS BLOB)) <= 8192),
  health_path TEXT NOT NULL CHECK (length(CAST(health_path AS BLOB)) <= 8192),
  transcription_timeout_seconds INTEGER NOT NULL CHECK (transcription_timeout_seconds BETWEEN 10 AND 3600),
  file_transcription_timeout_seconds INTEGER NOT NULL CHECK (file_transcription_timeout_seconds BETWEEN 60 AND 86400),
  transcription_options_prompt TEXT NOT NULL CHECK (length(CAST(transcription_options_prompt AS BLOB)) <= 8192),
  transcription_options_hotwords TEXT NOT NULL CHECK (length(CAST(transcription_options_hotwords AS BLOB)) <= 8192),
  transcription_options_temperature_override INTEGER NOT NULL CHECK (transcription_options_temperature_override IN (0,1)),
  transcription_options_temperature REAL NOT NULL CHECK (transcription_options_temperature BETWEEN 0 AND 1)
) STRICT;

CREATE TABLE cleanup_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1) REFERENCES preferences_settings(id) ON DELETE RESTRICT,
  generation_options_limit_output_tokens INTEGER NOT NULL CHECK (generation_options_limit_output_tokens IN (0,1)),
  generation_options_max_output_tokens INTEGER NOT NULL CHECK (generation_options_max_output_tokens BETWEEN 0 AND 65536),
  generation_options_disable_reasoning INTEGER NOT NULL CHECK (generation_options_disable_reasoning IN (0,1)),
  compatibility_profile TEXT NOT NULL CHECK (length(CAST(compatibility_profile AS BLOB)) <= 8192),
  enabled INTEGER NOT NULL CHECK (enabled IN (0,1)),
  base_url TEXT NOT NULL CHECK (length(CAST(base_url AS BLOB)) <= 8192),
  allow_insecure_http INTEGER NOT NULL CHECK (allow_insecure_http IN (0,1)),
  model TEXT NOT NULL CHECK (length(CAST(model AS BLOB)) <= 8192),
  preset TEXT NOT NULL CHECK (length(CAST(preset AS BLOB)) <= 8192),
  system_prompt TEXT NOT NULL CHECK (length(CAST(system_prompt AS BLOB)) <= 8192),
  styling TEXT NOT NULL CHECK (length(CAST(styling AS BLOB)) <= 8192),
  structure TEXT NOT NULL CHECK (length(CAST(structure AS BLOB)) <= 8192),
  context TEXT NOT NULL CHECK (length(CAST(context AS BLOB)) <= 8192),
  timeout_seconds INTEGER NOT NULL CHECK (timeout_seconds BETWEEN 10 AND 3600)
) STRICT;

CREATE TABLE speech_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1) REFERENCES preferences_settings(id) ON DELETE RESTRICT,
  compatibility_profile TEXT NOT NULL CHECK (length(CAST(compatibility_profile AS BLOB)) <= 8192),
  enabled INTEGER NOT NULL CHECK (enabled IN (0,1)),
  base_url TEXT NOT NULL CHECK (length(CAST(base_url AS BLOB)) <= 8192),
  allow_insecure_http INTEGER NOT NULL CHECK (allow_insecure_http IN (0,1)),
  authentication_mode TEXT NOT NULL CHECK (length(CAST(authentication_mode AS BLOB)) <= 8192),
  model TEXT NOT NULL CHECK (length(CAST(model AS BLOB)) <= 8192),
  voice TEXT NOT NULL CHECK (length(CAST(voice AS BLOB)) <= 8192),
  speed REAL NOT NULL CHECK (speed BETWEEN 0.25 AND 4),
  timeout_seconds INTEGER NOT NULL CHECK (timeout_seconds BETWEEN 10 AND 3600)
) STRICT;

CREATE TABLE request_headers (
  settings_id INTEGER NOT NULL REFERENCES transcription_settings(id) ON DELETE RESTRICT CHECK(settings_id=1),
  name TEXT PRIMARY KEY COLLATE NOCASE CHECK(length(CAST(name AS BLOB)) BETWEEN 1 AND 256),
  value TEXT NOT NULL CHECK(length(CAST(value AS BLOB))<=4096)
) STRICT;
CREATE TABLE credential_refs (
  purpose TEXT PRIMARY KEY CHECK(purpose IN ('stt','cleanup','speech')),
  account TEXT NOT NULL CHECK(length(account)<=128)
) STRICT;
CREATE TABLE credential_gc (
  account TEXT PRIMARY KEY CHECK(length(account)<=128)
) STRICT;
CREATE TABLE initialization (
  id INTEGER PRIMARY KEY CHECK(id=1),
  source TEXT NOT NULL CHECK(source IN ('defaults','legacy','reset'))
) STRICT;
