-- name: SeedTranscriptionConnection :exec
INSERT INTO saved_connections(id,name,compatibility_profile,base_url,allow_insecure_http,authentication_mode,health_path,credential_account) SELECT 'initial-stt','Transcription',s.compatibility_profile,s.base_url,s.allow_insecure_http,s.authentication_mode,s.health_path,c.account FROM transcription_settings s JOIN credential_refs c ON c.purpose='stt' WHERE s.base_url <> '' AND NOT EXISTS(SELECT 1 FROM saved_connections WHERE id='initial-stt');

-- name: SeedCleanupConnection :exec
INSERT INTO saved_connections(id,name,compatibility_profile,base_url,allow_insecure_http,authentication_mode,health_path,credential_account) SELECT 'initial-cleanup','Post-processing',s.compatibility_profile,s.base_url,s.allow_insecure_http,CASE WHEN c.account<>'' THEN 'api-key' ELSE 'none' END,'',c.account FROM cleanup_settings s JOIN credential_refs c ON c.purpose='cleanup' WHERE s.base_url <> '' AND NOT EXISTS(SELECT 1 FROM saved_connections WHERE id='initial-cleanup');

-- name: SeedSpeechConnection :exec
INSERT INTO saved_connections(id,name,compatibility_profile,base_url,allow_insecure_http,authentication_mode,health_path,credential_account) SELECT 'initial-speech','Speech playback',s.compatibility_profile,s.base_url,s.allow_insecure_http,s.authentication_mode,'',c.account FROM speech_settings s JOIN credential_refs c ON c.purpose='speech' WHERE s.base_url <> '' AND NOT EXISTS(SELECT 1 FROM saved_connections WHERE id='initial-speech');

-- name: SeedSelectedConnections :exec
INSERT INTO selected_connections(purpose,connection_id) SELECT purpose,connection_id FROM saved_connection_uses WHERE connection_id IN ('initial-stt','initial-cleanup','initial-speech') ON CONFLICT(purpose) DO NOTHING;

-- name: SeedConnectionHeaders :exec
INSERT INTO saved_connection_headers(connection_id,name,value) SELECT 'initial-stt',name,value FROM request_headers WHERE EXISTS(SELECT 1 FROM saved_connections WHERE id='initial-stt') ON CONFLICT(connection_id,name) DO NOTHING;

-- name: ListSavedConnections :many
SELECT id,name,compatibility_profile,base_url,allow_insecure_http,authentication_mode,health_path,credential_account FROM saved_connections ORDER BY name,id LIMIT 129;
-- name: ListSelectedConnections :many
SELECT purpose,connection_id FROM selected_connections ORDER BY purpose;
-- name: ListConnectionHeaders :many
SELECT connection_id,name,value FROM saved_connection_headers ORDER BY connection_id,name LIMIT 4097;
-- name: PutSavedConnection :exec
INSERT INTO saved_connections(id,name,compatibility_profile,base_url,allow_insecure_http,authentication_mode,health_path,credential_account) VALUES(?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET name=excluded.name,compatibility_profile=excluded.compatibility_profile,base_url=excluded.base_url,allow_insecure_http=excluded.allow_insecure_http,authentication_mode=excluded.authentication_mode,health_path=excluded.health_path,credential_account=excluded.credential_account;
-- name: SelectSavedConnection :exec
INSERT INTO selected_connections(purpose,connection_id) VALUES(?,?) ON CONFLICT(purpose) DO UPDATE SET connection_id=excluded.connection_id;
-- name: DeleteSavedConnection :exec
DELETE FROM saved_connections WHERE id=?;
-- name: ClearConnectionHeaders :exec
DELETE FROM saved_connection_headers WHERE connection_id=?;
-- name: PutConnectionHeader :exec
INSERT INTO saved_connection_headers(connection_id,name,value) VALUES(?,?,?);

-- name: ClearSelectedConnections :exec
DELETE FROM selected_connections;

-- name: SeedConnectionUses :exec
INSERT INTO saved_connection_uses(connection_id,purpose) SELECT id,CASE id WHEN 'initial-stt' THEN 'stt' WHEN 'initial-cleanup' THEN 'cleanup' ELSE 'speech' END FROM saved_connections WHERE id IN ('initial-stt','initial-cleanup','initial-speech') ON CONFLICT DO NOTHING;
-- name: ListConnectionUses :many
SELECT connection_id,purpose FROM saved_connection_uses ORDER BY connection_id,purpose LIMIT 513;
-- name: ClearConnectionUses :exec
DELETE FROM saved_connection_uses;
-- name: PutConnectionUse :exec
INSERT INTO saved_connection_uses(connection_id,purpose) VALUES(?,?);
