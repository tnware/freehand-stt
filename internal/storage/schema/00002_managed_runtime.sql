-- +goose Up
CREATE TABLE managed_runtime_preferences (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    enabled INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
    model TEXT NOT NULL DEFAULT 'nemotron-3.5' CHECK (length(model) BETWEEN 1 AND 128),
    realtime INTEGER NOT NULL DEFAULT 1 CHECK (realtime IN (0, 1))
);
INSERT INTO managed_runtime_preferences (id) VALUES (1);

-- +goose Down
DROP TABLE managed_runtime_preferences;
