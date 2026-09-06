-- +goose Up
-- Cleanup's existing preset is already its model behavior ID; preserve it.
ALTER TABLE transcription_settings ADD COLUMN model_profile TEXT NOT NULL DEFAULT 'generic';
ALTER TABLE speech_settings ADD COLUMN model_profile TEXT NOT NULL DEFAULT 'generic';

-- +goose Down
ALTER TABLE speech_settings DROP COLUMN model_profile;
ALTER TABLE transcription_settings DROP COLUMN model_profile;
