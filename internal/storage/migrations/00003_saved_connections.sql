-- +goose Up

CREATE TABLE saved_connections (
 id TEXT PRIMARY KEY CHECK(length(id) BETWEEN 1 AND 64),
 purpose TEXT NOT NULL CHECK(purpose IN ('stt','cleanup','speech')),
 name TEXT NOT NULL COLLATE NOCASE CHECK(length(CAST(name AS BLOB)) BETWEEN 1 AND 80),
 compatibility_profile TEXT NOT NULL CHECK(length(compatibility_profile)<=64),
 base_url TEXT NOT NULL CHECK(length(CAST(base_url AS BLOB))<=2048),
 allow_insecure_http INTEGER NOT NULL CHECK(allow_insecure_http IN (0,1)),
 authentication_mode TEXT NOT NULL CHECK(authentication_mode IN ('none','api-key')),
 model TEXT NOT NULL CHECK(length(CAST(model AS BLOB))<=8192),
 health_path TEXT NOT NULL CHECK(length(CAST(health_path AS BLOB))<=1024),
 cleanup_preset TEXT NOT NULL CHECK(length(cleanup_preset)<=64),
 credential_account TEXT NOT NULL CHECK(length(credential_account)<=128),
 UNIQUE(id,purpose), UNIQUE(purpose,name)
) STRICT;
CREATE TABLE selected_connections (
 purpose TEXT PRIMARY KEY CHECK(purpose IN ('stt','cleanup','speech')),
 connection_id TEXT NOT NULL,
 FOREIGN KEY(connection_id,purpose) REFERENCES saved_connections(id,purpose) ON DELETE RESTRICT
) STRICT;
CREATE TABLE saved_connection_headers (
 connection_id TEXT NOT NULL REFERENCES saved_connections(id) ON DELETE CASCADE,
 name TEXT NOT NULL COLLATE NOCASE CHECK(length(CAST(name AS BLOB)) BETWEEN 1 AND 256),
 value TEXT NOT NULL CHECK(length(CAST(value AS BLOB))<=4096),
 PRIMARY KEY(connection_id,name)
) STRICT;

INSERT INTO saved_connections(id,purpose,name,compatibility_profile,base_url,allow_insecure_http,authentication_mode,model,health_path,cleanup_preset,credential_account) SELECT 'initial-stt','stt','Transcription',s.compatibility_profile,s.base_url,s.allow_insecure_http,s.authentication_mode,s.model,s.health_path,'',c.account FROM transcription_settings s JOIN credential_refs c ON c.purpose='stt' WHERE NOT EXISTS(SELECT 1 FROM saved_connections WHERE purpose='stt');
INSERT INTO saved_connections(id,purpose,name,compatibility_profile,base_url,allow_insecure_http,authentication_mode,model,health_path,cleanup_preset,credential_account) SELECT 'initial-cleanup','cleanup','Post-processing',s.compatibility_profile,s.base_url,s.allow_insecure_http,'none',s.model,'',s.preset,c.account FROM cleanup_settings s JOIN credential_refs c ON c.purpose='cleanup' WHERE NOT EXISTS(SELECT 1 FROM saved_connections WHERE purpose='cleanup');
INSERT INTO saved_connections(id,purpose,name,compatibility_profile,base_url,allow_insecure_http,authentication_mode,model,health_path,cleanup_preset,credential_account) SELECT 'initial-speech','speech','Speech playback',s.compatibility_profile,s.base_url,s.allow_insecure_http,s.authentication_mode,s.model,'','',c.account FROM speech_settings s JOIN credential_refs c ON c.purpose='speech' WHERE NOT EXISTS(SELECT 1 FROM saved_connections WHERE purpose='speech');
INSERT INTO selected_connections(purpose,connection_id) SELECT purpose,id FROM saved_connections WHERE id IN ('initial-stt','initial-cleanup','initial-speech') ON CONFLICT(purpose) DO NOTHING;
INSERT INTO saved_connection_headers(connection_id,name,value) SELECT 'initial-stt',name,value FROM request_headers WHERE EXISTS(SELECT 1 FROM saved_connections WHERE id='initial-stt') ON CONFLICT(connection_id,name) DO NOTHING;
