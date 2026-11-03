-- +goose Up
ALTER TABLE preferences_settings ADD COLUMN live_insert INTEGER NOT NULL DEFAULT 0 CHECK (live_insert IN (0,1));

-- +goose Down
ALTER TABLE preferences_settings DROP COLUMN live_insert;
