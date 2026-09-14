-- +goose Up

-- Current Freehand baseline. Runtime defaults are written transactionally by Go.

-- Saved connections own transport, headers, and opaque credential accounts.

CREATE TABLE preferences_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  toggle_shortcut TEXT NOT NULL CHECK (length(CAST(toggle_shortcut AS BLOB)) <= 8192),
  show_shortcut TEXT NOT NULL CHECK (length(CAST(show_shortcut AS BLOB)) <= 8192),
  hold_shortcut TEXT NOT NULL CHECK (length(CAST(hold_shortcut AS BLOB)) <= 8192),
  microphone_id TEXT NOT NULL CHECK (length(CAST(microphone_id AS BLOB)) <= 1024),
  max_duration_seconds INTEGER NOT NULL CHECK (max_duration_seconds BETWEEN 1 AND CASE WHEN silence_splitting=1 THEN 3600 ELSE 262 END),
  auto_insert INTEGER NOT NULL CHECK (auto_insert IN (0,1)),
  start_with_windows INTEGER NOT NULL CHECK (start_with_windows IN (0,1)),
  show_window_on_launch INTEGER NOT NULL CHECK (show_window_on_launch IN (0,1)),
  check_for_updates INTEGER NOT NULL CHECK (check_for_updates IN (0,1)),
  setup_completed INTEGER NOT NULL CHECK (setup_completed IN (0,1)),
  use_mica INTEGER NOT NULL CHECK (use_mica IN (0,1)),
  appearance_mode TEXT NOT NULL CHECK (appearance_mode IN ('system','light','dark')),
  overlay_enabled INTEGER NOT NULL CHECK (overlay_enabled IN (0,1)),
  overlay_size_percent INTEGER NOT NULL CHECK (overlay_size_percent BETWEEN 75 AND 150),
  overlay_opacity_percent INTEGER NOT NULL CHECK (overlay_opacity_percent BETWEEN 40 AND 100),
  overlay_top_offset INTEGER NOT NULL CHECK (overlay_top_offset BETWEEN 0 AND 240),
  overlay_glow_percent INTEGER NOT NULL CHECK (overlay_glow_percent BETWEEN 0 AND 100),
  overlay_layout TEXT NOT NULL CHECK (overlay_layout IN ('minimal','capsule','meter','detailed')),
  overlay_anchor TEXT NOT NULL CHECK (overlay_anchor IN ('top-left','top-center','top-right','bottom-left','bottom-center','bottom-right')),
  overlay_visibility TEXT NOT NULL CHECK (overlay_visibility IN ('recording','active','all')),
  overlay_motion TEXT NOT NULL CHECK (overlay_motion IN ('system','reduced')),
  overlay_surface TEXT NOT NULL CHECK (overlay_surface IN ('glass','solid','minimal')),
  overlay_visualizer TEXT NOT NULL CHECK (overlay_visualizer IN ('bars','pulse','envelope','meter')),
  history_enabled INTEGER NOT NULL CHECK (history_enabled IN (0,1)),
  vad_enabled INTEGER NOT NULL CHECK (vad_enabled IN (0,1)),
  vad_mode TEXT NOT NULL CHECK (vad_mode IN ('quality','low-bitrate','aggressive','very-aggressive')),
  vad_activity_silence_ms INTEGER NOT NULL CHECK (vad_activity_silence_ms BETWEEN 100 AND 1500),
  silence_trimming INTEGER NOT NULL CHECK (silence_trimming IN (0,1)),
  speech_padding_ms INTEGER NOT NULL CHECK (speech_padding_ms BETWEEN 0 AND 1000),
  auto_stop_enabled INTEGER NOT NULL CHECK (auto_stop_enabled IN (0,1)),
  auto_stop_silence_ms INTEGER NOT NULL CHECK (auto_stop_silence_ms BETWEEN 500 AND 10000),
  auto_stop_minimum_speech_ms INTEGER NOT NULL CHECK (auto_stop_minimum_speech_ms BETWEEN 100 AND 5000),
  silence_splitting INTEGER NOT NULL CHECK (silence_splitting IN (0,1)),
  segment_seconds INTEGER NOT NULL CHECK (segment_seconds BETWEEN 15 AND 180),
  segment_silence_ms INTEGER NOT NULL CHECK (segment_silence_ms BETWEEN 200 AND 3000)
) STRICT;

CREATE TABLE transcription_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1) REFERENCES preferences_settings(id) ON DELETE RESTRICT,
  model TEXT NOT NULL CHECK (length(CAST(model AS BLOB)) <= 200),
  language TEXT NOT NULL CHECK (length(CAST(language AS BLOB)) <= 32),
  transcription_timeout_seconds INTEGER NOT NULL CHECK (transcription_timeout_seconds BETWEEN 10 AND 3600),
  file_transcription_timeout_seconds INTEGER NOT NULL CHECK (file_transcription_timeout_seconds BETWEEN 60 AND 86400),
  transcription_options_prompt TEXT NOT NULL CHECK (length(CAST(transcription_options_prompt AS BLOB)) <= 8192),
  transcription_options_temperature_override INTEGER NOT NULL CHECK (transcription_options_temperature_override IN (0,1)),
  transcription_options_temperature REAL NOT NULL CHECK (transcription_options_temperature BETWEEN 0 AND 1),
  model_profile TEXT NOT NULL CHECK(length(model_profile)<=64)
) STRICT;

CREATE TABLE cleanup_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1) REFERENCES preferences_settings(id) ON DELETE RESTRICT,
  generation_options_limit_output_tokens INTEGER NOT NULL CHECK (generation_options_limit_output_tokens IN (0,1)),
  generation_options_max_output_tokens INTEGER NOT NULL CHECK (generation_options_max_output_tokens BETWEEN 0 AND 65536),
  generation_options_disable_reasoning INTEGER NOT NULL CHECK (generation_options_disable_reasoning IN (0,1)),
  enabled INTEGER NOT NULL CHECK (enabled IN (0,1)),
  model TEXT NOT NULL CHECK (length(CAST(model AS BLOB)) <= 200),
  preset TEXT NOT NULL CHECK (preset IN ('generic','s1-mini')),
  system_prompt TEXT NOT NULL CHECK (length(CAST(system_prompt AS BLOB)) <= 8192),
  styling TEXT NOT NULL CHECK (length(CAST(styling AS BLOB)) <= 32),
  structure TEXT NOT NULL CHECK (length(CAST(structure AS BLOB)) <= 32),
  context TEXT NOT NULL CHECK (length(CAST(context AS BLOB)) <= 32),
  timeout_seconds INTEGER NOT NULL CHECK (timeout_seconds BETWEEN 10 AND 3600)
) STRICT;

CREATE TABLE speech_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1) REFERENCES preferences_settings(id) ON DELETE RESTRICT,
  enabled INTEGER NOT NULL CHECK (enabled IN (0,1)),
  model TEXT NOT NULL CHECK (length(CAST(model AS BLOB)) <= 200),
  voice TEXT NOT NULL CHECK (length(CAST(voice AS BLOB)) <= 200),
  speed REAL NOT NULL CHECK ((enabled=0 AND speed=0) OR speed BETWEEN 0.25 AND 4),
  timeout_seconds INTEGER NOT NULL CHECK (timeout_seconds BETWEEN 10 AND 3600),
  model_profile TEXT NOT NULL CHECK(length(model_profile)<=64),
  speech_language TEXT NOT NULL CHECK(length(speech_language)<=32),
  speech_instructions TEXT NOT NULL CHECK(length(speech_instructions)<=500)
) STRICT;

CREATE TABLE voice_transcription_settings (
 id INTEGER PRIMARY KEY CHECK(id=1),
 realtime INTEGER NOT NULL CHECK(realtime IN (0,1)),
 model_profile TEXT NOT NULL CHECK(length(model_profile)<=64),
 model TEXT NOT NULL CHECK(length(CAST(model AS BLOB))<=200),
 language TEXT NOT NULL CHECK(length(CAST(language AS BLOB))<=32),
 timeout_seconds INTEGER NOT NULL CHECK(timeout_seconds BETWEEN 10 AND 3600),
 prompt TEXT NOT NULL CHECK(length(CAST(prompt AS BLOB))<=8192),
 temperature_override INTEGER NOT NULL CHECK(temperature_override IN (0,1)),
 temperature REAL NOT NULL CHECK(temperature BETWEEN 0 AND 1),
 captions INTEGER NOT NULL CHECK(captions IN (0,1))
) STRICT;

CREATE TABLE vocabulary_settings (
 id INTEGER PRIMARY KEY CHECK(id=1),
 terms TEXT NOT NULL CHECK(length(CAST(terms AS BLOB))<=16384),
 voice INTEGER NOT NULL CHECK(voice IN (0,1)),
 files INTEGER NOT NULL CHECK(files IN (0,1)),
 boost REAL NOT NULL CHECK(boost BETWEEN 0 AND 5)
) STRICT;

CREATE TABLE saved_connections (
 id TEXT PRIMARY KEY CHECK(length(id) BETWEEN 1 AND 64),
 name TEXT NOT NULL COLLATE NOCASE CHECK(length(CAST(name AS BLOB)) BETWEEN 1 AND 80),
 compatibility_profile TEXT NOT NULL CHECK(length(compatibility_profile)<=64),
 base_url TEXT NOT NULL CHECK(length(CAST(base_url AS BLOB))<=2048),
 allow_insecure_http INTEGER NOT NULL CHECK(allow_insecure_http IN (0,1)),
 authentication_mode TEXT NOT NULL CHECK(authentication_mode IN ('none','api-key')),
 health_path TEXT NOT NULL CHECK(length(CAST(health_path AS BLOB))<=1024),
 credential_account TEXT NOT NULL CHECK(length(credential_account)<=128)
) STRICT;

CREATE TABLE saved_connection_uses (
 connection_id TEXT NOT NULL REFERENCES saved_connections(id) ON DELETE CASCADE,
 purpose TEXT NOT NULL CHECK(purpose IN ('stt','cleanup','speech','voice')),
 PRIMARY KEY(connection_id,purpose)
) STRICT;

CREATE TABLE selected_connections (
 purpose TEXT PRIMARY KEY CHECK(purpose IN ('stt','cleanup','speech','voice')),
 connection_id TEXT NOT NULL,
 FOREIGN KEY(connection_id,purpose) REFERENCES saved_connection_uses(connection_id,purpose) ON DELETE RESTRICT
) STRICT;

CREATE TABLE saved_connection_headers (
 connection_id TEXT NOT NULL REFERENCES saved_connections(id) ON DELETE CASCADE,
 name TEXT NOT NULL COLLATE NOCASE CHECK(length(CAST(name AS BLOB)) BETWEEN 1 AND 256),
 value TEXT NOT NULL CHECK(length(CAST(value AS BLOB))<=4096),
 PRIMARY KEY(connection_id,name)
) STRICT;

CREATE TABLE remembered_models (
 connection_id TEXT NOT NULL REFERENCES saved_connections(id) ON DELETE CASCADE,
 purpose TEXT NOT NULL CHECK(purpose IN ('stt','cleanup','speech','voice')),
 model TEXT NOT NULL CHECK(length(CAST(model AS BLOB))<=200),
 selected INTEGER NOT NULL CHECK(selected IN (0,1)),
 profile TEXT NOT NULL DEFAULT 'generic' CHECK(length(profile)<=64),
 prompt TEXT NOT NULL DEFAULT '' CHECK(length(CAST(prompt AS BLOB))<=8192),
 temperature_override INTEGER NOT NULL DEFAULT 0 CHECK(temperature_override IN (0,1)),
 temperature REAL NOT NULL DEFAULT 0 CHECK(temperature BETWEEN 0 AND 1),
 limit_output_tokens INTEGER NOT NULL DEFAULT 0 CHECK(limit_output_tokens IN (0,1)),
 max_output_tokens INTEGER NOT NULL DEFAULT 0 CHECK(max_output_tokens BETWEEN 0 AND 65536),
 disable_reasoning INTEGER NOT NULL DEFAULT 0 CHECK(disable_reasoning IN (0,1)),
 voice TEXT NOT NULL DEFAULT '' CHECK(length(voice)<=200),
 speech_language TEXT NOT NULL DEFAULT '' CHECK(length(speech_language)<=32),
 speech_instructions TEXT NOT NULL DEFAULT '' CHECK(length(speech_instructions)<=500),
 PRIMARY KEY(connection_id,purpose,model),
 FOREIGN KEY(connection_id,purpose) REFERENCES saved_connection_uses(connection_id,purpose) ON DELETE CASCADE
) STRICT;

CREATE UNIQUE INDEX remembered_models_selected ON remembered_models(connection_id,purpose) WHERE selected=1;

CREATE TABLE credential_gc (
  account TEXT PRIMARY KEY CHECK(length(account)<=128)
) STRICT;
