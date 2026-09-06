-- +goose Up

-- Model choices and cleanup presets remain in their runtime settings tables.
ALTER TABLE saved_connections DROP COLUMN model;
ALTER TABLE saved_connections DROP COLUMN cleanup_preset;

-- Empty bootstrap entries are not user-created connections. Preserve configured entries.
DELETE FROM selected_connections WHERE connection_id IN (
 SELECT id FROM saved_connections WHERE base_url = '' AND credential_account = ''
);
DELETE FROM saved_connections WHERE base_url = '' AND credential_account = '';
