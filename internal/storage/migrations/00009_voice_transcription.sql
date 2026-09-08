-- +goose Up
CREATE TABLE voice_connection_uses (
 connection_id TEXT NOT NULL REFERENCES saved_connections(id) ON DELETE CASCADE,
 purpose TEXT NOT NULL CHECK(purpose IN ('stt','cleanup','speech','voice')),
 PRIMARY KEY(connection_id,purpose)
) STRICT;
INSERT INTO voice_connection_uses SELECT connection_id,CASE purpose WHEN 'realtime' THEN 'voice' ELSE purpose END FROM saved_connection_uses;
INSERT OR IGNORE INTO voice_connection_uses SELECT connection_id,'voice' FROM saved_connection_uses WHERE purpose='stt';
CREATE TABLE voice_selections (
 purpose TEXT PRIMARY KEY CHECK(purpose IN ('stt','cleanup','speech','voice')),
 connection_id TEXT NOT NULL,
 FOREIGN KEY(connection_id,purpose) REFERENCES voice_connection_uses(connection_id,purpose) ON DELETE RESTRICT
) STRICT;
INSERT INTO voice_selections SELECT purpose,connection_id FROM selected_connections WHERE purpose<>'realtime';
INSERT INTO voice_selections SELECT 'voice',connection_id FROM selected_connections WHERE purpose=CASE WHEN (SELECT enabled FROM realtime_settings WHERE id=1)=1 THEN 'realtime' ELSE 'stt' END;
DROP TABLE selected_connections;
DROP TABLE saved_connection_uses;
ALTER TABLE voice_connection_uses RENAME TO saved_connection_uses;
ALTER TABLE voice_selections RENAME TO selected_connections;
CREATE TABLE voice_credential_refs (
 purpose TEXT PRIMARY KEY CHECK(purpose IN ('stt','cleanup','speech','voice')),
 account TEXT NOT NULL CHECK(length(account)<=128)
) STRICT;
INSERT INTO voice_credential_refs SELECT purpose,account FROM credential_refs WHERE purpose<>'realtime';
INSERT INTO voice_credential_refs SELECT 'voice',COALESCE((SELECT c.credential_account FROM selected_connections s JOIN saved_connections c ON c.id=s.connection_id WHERE s.purpose='voice'),'');
DROP TABLE credential_refs;
ALTER TABLE voice_credential_refs RENAME TO credential_refs;
CREATE TABLE voice_transcription_settings (
 id INTEGER PRIMARY KEY CHECK(id=1),
 realtime INTEGER NOT NULL CHECK(realtime IN (0,1)),
 compatibility_profile TEXT NOT NULL CHECK(length(compatibility_profile)<=64),
 model_profile TEXT NOT NULL CHECK(length(model_profile)<=64),
 base_url TEXT NOT NULL CHECK(length(CAST(base_url AS BLOB))<=2048),
 allow_insecure_http INTEGER NOT NULL CHECK(allow_insecure_http IN (0,1)),
 authentication_mode TEXT NOT NULL CHECK(authentication_mode IN ('none','api-key')),
 model TEXT NOT NULL CHECK(length(CAST(model AS BLOB))<=200),
 language TEXT NOT NULL CHECK(length(language)<=32),
 health_path TEXT NOT NULL CHECK(length(CAST(health_path AS BLOB))<=1024),
 timeout_seconds INTEGER NOT NULL CHECK(timeout_seconds BETWEEN 10 AND 3600),
 prompt TEXT NOT NULL CHECK(length(CAST(prompt AS BLOB))<=8192),
 hotwords TEXT NOT NULL CHECK(length(CAST(hotwords AS BLOB))<=2048),
 temperature_override INTEGER NOT NULL CHECK(temperature_override IN (0,1)),
 temperature REAL NOT NULL CHECK(temperature BETWEEN 0 AND 1),
 captions INTEGER NOT NULL CHECK(captions IN (0,1)),
 vocabulary TEXT NOT NULL CHECK(length(CAST(vocabulary AS BLOB))<=2048),
 boost REAL NOT NULL CHECK(boost BETWEEN 0 AND 5)
) STRICT;
INSERT INTO voice_transcription_settings SELECT 1,r.enabled,
 CASE r.enabled WHEN 1 THEN r.compatibility_profile ELSE t.compatibility_profile END,
 CASE r.enabled WHEN 1 THEN r.model_profile ELSE t.model_profile END,
 CASE r.enabled WHEN 1 THEN r.base_url ELSE t.base_url END,
 CASE r.enabled WHEN 1 THEN r.allow_insecure_http ELSE t.allow_insecure_http END,
 CASE r.enabled WHEN 1 THEN r.authentication_mode ELSE t.authentication_mode END,
 CASE r.enabled WHEN 1 THEN r.model ELSE t.model END,
 CASE r.enabled WHEN 1 THEN r.language ELSE t.language END,
 CASE r.enabled WHEN 1 THEN '' ELSE t.health_path END,
 t.transcription_timeout_seconds,
 CASE r.enabled WHEN 1 THEN '' ELSE t.transcription_options_prompt END,
 CASE r.enabled WHEN 1 THEN '' ELSE t.transcription_options_hotwords END,
 CASE r.enabled WHEN 1 THEN 0 ELSE t.transcription_options_temperature_override END,
 CASE r.enabled WHEN 1 THEN 0 ELSE t.transcription_options_temperature END,
 r.captions,CASE r.enabled WHEN 1 THEN r.vocabulary ELSE '' END,CASE r.enabled WHEN 1 THEN r.boost ELSE 0 END
 FROM realtime_settings r CROSS JOIN transcription_settings t WHERE r.id=1 AND t.id=1;
CREATE TABLE voice_request_headers(name TEXT PRIMARY KEY COLLATE NOCASE CHECK(length(CAST(name AS BLOB)) BETWEEN 1 AND 256), value TEXT NOT NULL CHECK(length(CAST(value AS BLOB))<=4096)) STRICT;
INSERT INTO voice_request_headers SELECT name,value FROM request_headers WHERE (SELECT realtime FROM voice_transcription_settings WHERE id=1)=0;
UPDATE preferences_settings SET setup_completed=1 WHERE (SELECT enabled FROM realtime_settings WHERE id=1)=1;
DROP TABLE realtime_settings;
CREATE TABLE voice_remembered_models (
 connection_id TEXT NOT NULL REFERENCES saved_connections(id) ON DELETE CASCADE,
 purpose TEXT NOT NULL CHECK(purpose IN ('stt','cleanup','speech','voice')),
 model TEXT NOT NULL CHECK(length(CAST(model AS BLOB))<=200),
 selected INTEGER NOT NULL CHECK(selected IN (0,1)),
 vocabulary TEXT NOT NULL DEFAULT '' CHECK(length(CAST(vocabulary AS BLOB))<=2048),
 boost REAL NOT NULL DEFAULT 0 CHECK(boost BETWEEN 0 AND 5),
 profile TEXT NOT NULL DEFAULT 'generic' CHECK(length(profile)<=64),
 language TEXT NOT NULL DEFAULT '' CHECK(length(language)<=32),
 prompt TEXT NOT NULL DEFAULT '' CHECK(length(CAST(prompt AS BLOB))<=8192),
 hotwords TEXT NOT NULL DEFAULT '' CHECK(length(CAST(hotwords AS BLOB))<=2048),
 temperature_override INTEGER NOT NULL DEFAULT 0 CHECK(temperature_override IN (0,1)),
 temperature REAL NOT NULL DEFAULT 0 CHECK(temperature BETWEEN 0 AND 1),
 limit_output_tokens INTEGER NOT NULL DEFAULT 0 CHECK(limit_output_tokens IN (0,1)),
 max_output_tokens INTEGER NOT NULL DEFAULT 0 CHECK(max_output_tokens BETWEEN 0 AND 65536),
 disable_reasoning INTEGER NOT NULL DEFAULT 0 CHECK(disable_reasoning IN (0,1)),
 system_prompt TEXT NOT NULL DEFAULT '' CHECK(length(CAST(system_prompt AS BLOB))<=8192),
 styling TEXT NOT NULL DEFAULT '' CHECK(length(styling)<=32),
 structure TEXT NOT NULL DEFAULT '' CHECK(length(structure)<=32),
 context TEXT NOT NULL DEFAULT '' CHECK(length(context)<=32),
 voice TEXT NOT NULL DEFAULT '' CHECK(length(voice)<=200),
 speed REAL NOT NULL DEFAULT 0 CHECK(speed=0 OR speed BETWEEN 0.25 AND 4),
 PRIMARY KEY(connection_id,purpose,model)
) STRICT;
INSERT INTO voice_remembered_models(connection_id,purpose,model,selected,profile,language,prompt,hotwords,temperature_override,temperature,limit_output_tokens,max_output_tokens,disable_reasoning,system_prompt,styling,structure,context,voice,speed,vocabulary,boost) SELECT connection_id,CASE purpose WHEN 'realtime' THEN 'voice' ELSE purpose END,model,selected,profile,language,prompt,hotwords,temperature_override,temperature,limit_output_tokens,max_output_tokens,disable_reasoning,system_prompt,styling,structure,context,voice,speed,vocabulary,boost FROM remembered_models;
INSERT OR IGNORE INTO voice_remembered_models(connection_id,purpose,model,selected,profile,language,prompt,hotwords,temperature_override,temperature,limit_output_tokens,max_output_tokens,disable_reasoning,system_prompt,styling,structure,context,voice,speed,vocabulary,boost) SELECT connection_id,'voice',model,selected,profile,language,prompt,hotwords,temperature_override,temperature,limit_output_tokens,max_output_tokens,disable_reasoning,system_prompt,styling,structure,context,voice,speed,vocabulary,boost FROM remembered_models WHERE purpose='stt';
DROP TABLE remembered_models;
ALTER TABLE voice_remembered_models RENAME TO remembered_models;
CREATE UNIQUE INDEX remembered_models_selected ON remembered_models(connection_id,purpose) WHERE selected=1;
