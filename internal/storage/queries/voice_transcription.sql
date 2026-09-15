-- name: GetVoiceTranscription :one
SELECT realtime, model_profile, model, language, timeout_seconds, prompt, temperature_override, temperature, captions, nemo_disable_punctuation, nemo_normalize, nemo_profanity_filter, nemo_endpointing_ms FROM voice_transcription_settings WHERE id=1;

-- name: PutVoiceTranscription :exec
INSERT INTO voice_transcription_settings (id, realtime, model_profile, model, language, timeout_seconds, prompt, temperature_override, temperature, captions, nemo_disable_punctuation, nemo_normalize, nemo_profanity_filter, nemo_endpointing_ms)
VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET realtime=excluded.realtime, model_profile=excluded.model_profile, model=excluded.model, language=excluded.language, timeout_seconds=excluded.timeout_seconds, prompt=excluded.prompt, temperature_override=excluded.temperature_override, temperature=excluded.temperature, captions=excluded.captions, nemo_disable_punctuation=excluded.nemo_disable_punctuation, nemo_normalize=excluded.nemo_normalize, nemo_profanity_filter=excluded.nemo_profanity_filter, nemo_endpointing_ms=excluded.nemo_endpointing_ms;
