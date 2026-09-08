-- +goose Up
ALTER TABLE speech_settings ADD COLUMN speech_language TEXT NOT NULL DEFAULT '';
ALTER TABLE speech_settings ADD COLUMN speech_instructions TEXT NOT NULL DEFAULT '';
ALTER TABLE remembered_models ADD COLUMN speech_language TEXT NOT NULL DEFAULT '';
ALTER TABLE remembered_models ADD COLUMN speech_instructions TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE remembered_models DROP COLUMN speech_instructions;
ALTER TABLE remembered_models DROP COLUMN speech_language;
ALTER TABLE speech_settings DROP COLUMN speech_instructions;
ALTER TABLE speech_settings DROP COLUMN speech_language;
