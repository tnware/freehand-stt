-- +goose Up
CREATE TABLE reusable_connections (
 id TEXT PRIMARY KEY CHECK(length(id) BETWEEN 1 AND 64),
 name TEXT NOT NULL COLLATE NOCASE CHECK(length(CAST(name AS BLOB)) BETWEEN 1 AND 80),
 compatibility_profile TEXT NOT NULL CHECK(length(compatibility_profile)<=64),
 base_url TEXT NOT NULL CHECK(length(CAST(base_url AS BLOB))<=2048),
 allow_insecure_http INTEGER NOT NULL CHECK(allow_insecure_http IN (0,1)),
 authentication_mode TEXT NOT NULL CHECK(authentication_mode IN ('none','api-key')),
 health_path TEXT NOT NULL CHECK(length(CAST(health_path AS BLOB))<=1024),
 credential_account TEXT NOT NULL CHECK(length(credential_account)<=128)
) STRICT;
CREATE TABLE reusable_connection_uses (
 connection_id TEXT NOT NULL REFERENCES reusable_connections(id) ON DELETE CASCADE,
 purpose TEXT NOT NULL CHECK(purpose IN ('stt','cleanup','speech')),
 PRIMARY KEY(connection_id,purpose)
) STRICT;
CREATE TABLE reusable_selections (
 purpose TEXT PRIMARY KEY CHECK(purpose IN ('stt','cleanup','speech')),
 connection_id TEXT NOT NULL,
 FOREIGN KEY(connection_id,purpose) REFERENCES reusable_connection_uses(connection_id,purpose) ON DELETE RESTRICT
) STRICT;
CREATE TABLE reusable_headers (
 connection_id TEXT NOT NULL REFERENCES reusable_connections(id) ON DELETE CASCADE,
 name TEXT NOT NULL COLLATE NOCASE CHECK(length(CAST(name AS BLOB)) BETWEEN 1 AND 256),
 value TEXT NOT NULL CHECK(length(CAST(value AS BLOB))<=4096),
 PRIMARY KEY(connection_id,name)
) STRICT;
INSERT INTO reusable_connections SELECT id,name,compatibility_profile,base_url,allow_insecure_http,
 CASE WHEN purpose='cleanup' AND credential_account<>'' THEN 'api-key' ELSE authentication_mode END,
 health_path,credential_account FROM saved_connections;
INSERT INTO reusable_connection_uses SELECT id,purpose FROM saved_connections;
INSERT INTO reusable_selections SELECT purpose,connection_id FROM selected_connections;
INSERT INTO reusable_headers SELECT connection_id,name,value FROM saved_connection_headers;
DROP TABLE selected_connections;
DROP TABLE saved_connection_headers;
DROP TABLE saved_connections;
ALTER TABLE reusable_connections RENAME TO saved_connections;
ALTER TABLE reusable_connection_uses RENAME TO saved_connection_uses;
ALTER TABLE reusable_selections RENAME TO selected_connections;
ALTER TABLE reusable_headers RENAME TO saved_connection_headers;
