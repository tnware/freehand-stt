-- +goose Up
CREATE TABLE remembered_models (
 connection_id TEXT NOT NULL REFERENCES saved_connections(id) ON DELETE CASCADE,
 purpose TEXT NOT NULL CHECK(purpose IN ('stt','cleanup','speech')),
 model TEXT NOT NULL CHECK(length(CAST(model AS BLOB))<=200),
 selected INTEGER NOT NULL CHECK(selected IN (0,1)),
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
CREATE UNIQUE INDEX remembered_models_selected ON remembered_models(connection_id,purpose) WHERE selected=1;

-- Preserve currently selected model choices when upgrading.
INSERT INTO remembered_models (connection_id,purpose,model,selected,profile,language,prompt,hotwords,temperature_override,temperature) SELECT c.connection_id,'stt',s.model,1,s.model_profile,s.language,s.transcription_options_prompt,s.transcription_options_hotwords,s.transcription_options_temperature_override,s.transcription_options_temperature FROM transcription_settings s JOIN selected_connections c ON c.purpose='stt' WHERE s.model<>'' OR s.compatibility_profile='whisper-cpp';
INSERT INTO remembered_models (connection_id,purpose,model,selected,profile,limit_output_tokens,max_output_tokens,disable_reasoning,system_prompt,styling,structure,context) SELECT c.connection_id,'cleanup',s.model,1,s.preset,s.generation_options_limit_output_tokens,s.generation_options_max_output_tokens,s.generation_options_disable_reasoning,s.system_prompt,s.styling,s.structure,s.context FROM cleanup_settings s JOIN selected_connections c ON c.purpose='cleanup' WHERE s.model<>'';
INSERT INTO remembered_models (connection_id,purpose,model,selected,profile,voice,speed) SELECT c.connection_id,'speech',s.model,1,s.model_profile,s.voice,s.speed FROM speech_settings s JOIN selected_connections c ON c.purpose='speech' WHERE s.model<>'';

-- +goose Down
DROP TABLE remembered_models;
