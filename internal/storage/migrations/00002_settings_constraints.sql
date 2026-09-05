-- +goose Up

-- Version 1 was used by local alpha builds. Preserve it and upgrade in place.
ALTER TABLE preferences_settings RENAME COLUMN vadenabled TO vad_enabled;
ALTER TABLE preferences_settings RENAME COLUMN vadmode TO vad_mode;
ALTER TABLE preferences_settings RENAME COLUMN vadactivity_silence_ms TO vad_activity_silence_ms;

-- Zero speed is a valid inactive value for an unconfigured speech endpoint.
ALTER TABLE speech_settings RENAME TO speech_settings_v1;
CREATE TABLE speech_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1) REFERENCES preferences_settings(id) ON DELETE RESTRICT,
  compatibility_profile TEXT NOT NULL CHECK (length(CAST(compatibility_profile AS BLOB)) <= 8192),
  enabled INTEGER NOT NULL CHECK (enabled IN (0,1)),
  base_url TEXT NOT NULL CHECK (length(CAST(base_url AS BLOB)) <= 8192),
  allow_insecure_http INTEGER NOT NULL CHECK (allow_insecure_http IN (0,1)),
  authentication_mode TEXT NOT NULL CHECK (length(CAST(authentication_mode AS BLOB)) <= 8192),
  model TEXT NOT NULL CHECK (length(CAST(model AS BLOB)) <= 8192),
  voice TEXT NOT NULL CHECK (length(CAST(voice AS BLOB)) <= 8192),
  speed REAL NOT NULL CHECK ((enabled=0 AND speed=0) OR speed BETWEEN 0.25 AND 4),
  timeout_seconds INTEGER NOT NULL CHECK (timeout_seconds BETWEEN 10 AND 3600)
) STRICT;
INSERT INTO speech_settings (
  id, compatibility_profile, enabled, base_url, allow_insecure_http,
  authentication_mode, model, voice, speed, timeout_seconds
)
SELECT id, compatibility_profile, enabled, base_url, allow_insecure_http,
  authentication_mode, model, voice, speed, timeout_seconds
FROM speech_settings_v1;
DROP TABLE speech_settings_v1;
