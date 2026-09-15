-- +goose Up
ALTER TABLE transcription_settings ADD COLUMN nemo_disable_punctuation INTEGER NOT NULL DEFAULT 0 CHECK (nemo_disable_punctuation IN (0,1));
ALTER TABLE transcription_settings ADD COLUMN nemo_normalize INTEGER NOT NULL DEFAULT 0 CHECK (nemo_normalize IN (0,1));
ALTER TABLE transcription_settings ADD COLUMN nemo_profanity_filter INTEGER NOT NULL DEFAULT 0 CHECK (nemo_profanity_filter IN (0,1));
ALTER TABLE transcription_settings ADD COLUMN nemo_endpointing_ms INTEGER NOT NULL DEFAULT 0 CHECK (nemo_endpointing_ms = 0 OR nemo_endpointing_ms BETWEEN 100 AND 10000);
ALTER TABLE voice_transcription_settings ADD COLUMN nemo_disable_punctuation INTEGER NOT NULL DEFAULT 0 CHECK (nemo_disable_punctuation IN (0,1));
ALTER TABLE voice_transcription_settings ADD COLUMN nemo_normalize INTEGER NOT NULL DEFAULT 0 CHECK (nemo_normalize IN (0,1));
ALTER TABLE voice_transcription_settings ADD COLUMN nemo_profanity_filter INTEGER NOT NULL DEFAULT 0 CHECK (nemo_profanity_filter IN (0,1));
ALTER TABLE voice_transcription_settings ADD COLUMN nemo_endpointing_ms INTEGER NOT NULL DEFAULT 0 CHECK (nemo_endpointing_ms = 0 OR nemo_endpointing_ms BETWEEN 100 AND 10000);
ALTER TABLE remembered_models ADD COLUMN nemo_disable_punctuation INTEGER NOT NULL DEFAULT 0 CHECK (nemo_disable_punctuation IN (0,1));
ALTER TABLE remembered_models ADD COLUMN nemo_normalize INTEGER NOT NULL DEFAULT 0 CHECK (nemo_normalize IN (0,1));
ALTER TABLE remembered_models ADD COLUMN nemo_profanity_filter INTEGER NOT NULL DEFAULT 0 CHECK (nemo_profanity_filter IN (0,1));
ALTER TABLE remembered_models ADD COLUMN nemo_endpointing_ms INTEGER NOT NULL DEFAULT 0 CHECK (nemo_endpointing_ms = 0 OR nemo_endpointing_ms BETWEEN 100 AND 10000);
