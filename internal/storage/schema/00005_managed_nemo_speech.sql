-- +goose Up
ALTER TABLE managed_runtime_instances ADD COLUMN speech_model TEXT NOT NULL DEFAULT '' CHECK(length(speech_model) <= 128 AND (speech_model = '' OR provider = 'nemo-speech-cpp'));

-- +goose Down
ALTER TABLE managed_runtime_instances DROP COLUMN speech_model;
