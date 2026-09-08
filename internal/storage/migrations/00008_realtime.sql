-- +goose Up
CREATE TABLE realtime_connection_uses (
 connection_id TEXT NOT NULL REFERENCES saved_connections(id) ON DELETE CASCADE,
 purpose TEXT NOT NULL CHECK(purpose IN ('stt','cleanup','speech','realtime')),
 PRIMARY KEY(connection_id,purpose)
) STRICT;
INSERT INTO realtime_connection_uses SELECT * FROM saved_connection_uses;
CREATE TABLE realtime_selections (
 purpose TEXT PRIMARY KEY CHECK(purpose IN ('stt','cleanup','speech','realtime')),
 connection_id TEXT NOT NULL,
 FOREIGN KEY(connection_id,purpose) REFERENCES realtime_connection_uses(connection_id,purpose) ON DELETE RESTRICT
) STRICT;
INSERT INTO realtime_selections SELECT * FROM selected_connections;
DROP TABLE selected_connections;
DROP TABLE saved_connection_uses;
ALTER TABLE realtime_connection_uses RENAME TO saved_connection_uses;
ALTER TABLE realtime_selections RENAME TO selected_connections;
CREATE TABLE realtime_credential_refs (
 purpose TEXT PRIMARY KEY CHECK(purpose IN ('stt','cleanup','speech','realtime')),
 account TEXT NOT NULL CHECK(length(account)<=128)
) STRICT;
INSERT INTO realtime_credential_refs SELECT * FROM credential_refs;
INSERT INTO realtime_credential_refs VALUES('realtime','');
DROP TABLE credential_refs;
ALTER TABLE realtime_credential_refs RENAME TO credential_refs;
CREATE TABLE realtime_settings (
 id INTEGER PRIMARY KEY CHECK(id=1),
 enabled INTEGER NOT NULL CHECK(enabled IN (0,1)),
 compatibility_profile TEXT NOT NULL CHECK(length(compatibility_profile)<=64),
 model_profile TEXT NOT NULL CHECK(length(model_profile)<=64),
 base_url TEXT NOT NULL CHECK(length(CAST(base_url AS BLOB))<=2048),
 allow_insecure_http INTEGER NOT NULL CHECK(allow_insecure_http IN (0,1)),
 authentication_mode TEXT NOT NULL CHECK(authentication_mode IN ('none','api-key')),
 model TEXT NOT NULL CHECK(length(CAST(model AS BLOB))<=200),
 language TEXT NOT NULL CHECK(length(language)<=32),
 captions INTEGER NOT NULL CHECK(captions IN (0,1)),
 vocabulary TEXT NOT NULL CHECK(length(CAST(vocabulary AS BLOB))<=2048),
 boost REAL NOT NULL CHECK(boost BETWEEN 0 AND 5)
) STRICT;
INSERT INTO realtime_settings VALUES(1,0,'nemo-speech-v1','nemotron-3.5-streaming','',0,'none','','auto',1,'',3);
CREATE TABLE realtime_remembered_models (
 connection_id TEXT NOT NULL REFERENCES saved_connections(id) ON DELETE CASCADE,
 purpose TEXT NOT NULL CHECK(purpose IN ('stt','cleanup','speech','realtime')),
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
INSERT INTO realtime_remembered_models(connection_id,purpose,model,selected,profile,language,prompt,hotwords,temperature_override,temperature,limit_output_tokens,max_output_tokens,disable_reasoning,system_prompt,styling,structure,context,voice,speed) SELECT connection_id,purpose,model,selected,profile,language,prompt,hotwords,temperature_override,temperature,limit_output_tokens,max_output_tokens,disable_reasoning,system_prompt,styling,structure,context,voice,speed FROM remembered_models;
DROP TABLE remembered_models;
ALTER TABLE realtime_remembered_models RENAME TO remembered_models;
CREATE UNIQUE INDEX remembered_models_selected ON remembered_models(connection_id,purpose) WHERE selected=1;
