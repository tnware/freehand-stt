-- name: GetVoiceTranscription :one
SELECT * FROM voice_transcription_settings WHERE id=1;

-- name: PutVoiceTranscription :exec
INSERT INTO voice_transcription_settings(id,realtime,compatibility_profile,model_profile,base_url,allow_insecure_http,authentication_mode,model,language,health_path,timeout_seconds,prompt,hotwords,temperature_override,temperature,captions,vocabulary,boost) VALUES(1,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET realtime=excluded.realtime,compatibility_profile=excluded.compatibility_profile,model_profile=excluded.model_profile,base_url=excluded.base_url,allow_insecure_http=excluded.allow_insecure_http,authentication_mode=excluded.authentication_mode,model=excluded.model,language=excluded.language,health_path=excluded.health_path,timeout_seconds=excluded.timeout_seconds,prompt=excluded.prompt,hotwords=excluded.hotwords,temperature_override=excluded.temperature_override,temperature=excluded.temperature,captions=excluded.captions,vocabulary=excluded.vocabulary,boost=excluded.boost;

-- name: GetVoiceHeaders :many
SELECT name,value FROM voice_request_headers ORDER BY name LIMIT 33;
-- name: ClearVoiceHeaders :exec
DELETE FROM voice_request_headers;
-- name: PutVoiceHeader :exec
INSERT INTO voice_request_headers(name,value) VALUES(?,?);
