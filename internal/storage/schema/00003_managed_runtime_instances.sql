-- +goose Up
CREATE TABLE managed_runtime_instances (
 id TEXT PRIMARY KEY CHECK(length(id) BETWEEN 1 AND 64),
 name TEXT NOT NULL CHECK(length(CAST(name AS BLOB)) BETWEEN 1 AND 80),
 provider TEXT NOT NULL CHECK(length(provider) BETWEEN 1 AND 64),
 model TEXT NOT NULL CHECK(length(model) BETWEEN 1 AND 128),
 auto_start INTEGER NOT NULL CHECK(auto_start IN (0,1))
) STRICT;
ALTER TABLE saved_connections ADD COLUMN managed_instance_id TEXT REFERENCES managed_runtime_instances(id) ON DELETE RESTRICT;
INSERT INTO managed_runtime_instances SELECT 'nemo-default','Local NeMo','nemo-speech-cpp',model,enabled FROM managed_runtime_preferences WHERE EXISTS(SELECT 1 FROM preferences_settings);
INSERT INTO saved_connections(id,name,compatibility_profile,base_url,allow_insecure_http,authentication_mode,health_path,credential_account,managed_instance_id)
SELECT 'managed-nemo-default','Local NeMo','','',0,'none','','','nemo-default' FROM managed_runtime_preferences WHERE enabled=1 AND EXISTS(SELECT 1 FROM preferences_settings);
INSERT INTO saved_connection_uses SELECT 'managed-nemo-default','voice' WHERE EXISTS(SELECT 1 FROM saved_connections WHERE id='managed-nemo-default');
INSERT INTO saved_connection_uses SELECT 'managed-nemo-default','stt' WHERE EXISTS(SELECT 1 FROM saved_connections WHERE id='managed-nemo-default');
INSERT OR REPLACE INTO selected_connections SELECT purpose,connection_id FROM saved_connection_uses WHERE connection_id='managed-nemo-default';
UPDATE voice_transcription_settings SET realtime=(SELECT realtime FROM managed_runtime_preferences) WHERE EXISTS(SELECT 1 FROM saved_connections WHERE id='managed-nemo-default');
DROP TABLE managed_runtime_preferences;
-- +goose StatementBegin
CREATE TRIGGER managed_transport_insert BEFORE INSERT ON saved_connections
WHEN NEW.managed_instance_id IS NOT NULL AND (NEW.base_url<>'' OR NEW.compatibility_profile<>'' OR NEW.allow_insecure_http<>0 OR NEW.authentication_mode<>'none' OR NEW.health_path<>'' OR NEW.credential_account<>'')
BEGIN SELECT RAISE(ABORT,'managed connection cannot contain manual transport'); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER managed_transport_update BEFORE UPDATE ON saved_connections
WHEN NEW.managed_instance_id IS NOT NULL AND (NEW.base_url<>'' OR NEW.compatibility_profile<>'' OR NEW.allow_insecure_http<>0 OR NEW.authentication_mode<>'none' OR NEW.health_path<>'' OR NEW.credential_account<>'' OR EXISTS(SELECT 1 FROM saved_connection_headers WHERE connection_id=OLD.id))
BEGIN SELECT RAISE(ABORT,'managed connection cannot contain manual transport'); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER managed_headers_insert BEFORE INSERT ON saved_connection_headers
WHEN EXISTS(SELECT 1 FROM saved_connections WHERE id=NEW.connection_id AND managed_instance_id IS NOT NULL)
BEGIN SELECT RAISE(ABORT,'managed connection cannot contain headers'); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER managed_headers_update BEFORE UPDATE ON saved_connection_headers
WHEN EXISTS(SELECT 1 FROM saved_connections WHERE id=NEW.connection_id AND managed_instance_id IS NOT NULL)
BEGIN SELECT RAISE(ABORT,'managed connection cannot contain headers'); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER managed_provider_immutable BEFORE UPDATE OF provider ON managed_runtime_instances WHEN NEW.provider<>OLD.provider
BEGIN SELECT RAISE(ABORT,'runtime provider is immutable'); END;
-- +goose StatementEnd
