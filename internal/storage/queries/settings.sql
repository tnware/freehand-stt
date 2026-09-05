-- name: GetPreferences :one
SELECT toggle_shortcut, show_shortcut, hold_shortcut, microphone_id, max_duration_seconds, auto_insert, start_with_windows, show_window_on_launch, check_for_updates, setup_completed, use_mica, appearance_mode, overlay_enabled, overlay_size_percent, overlay_opacity_percent, overlay_top_offset, overlay_glow_percent, overlay_layout, overlay_anchor, overlay_visibility, overlay_motion, overlay_surface, overlay_visualizer, history_enabled, vad_enabled, vad_mode, vad_activity_silence_ms, silence_trimming, speech_padding_ms, auto_stop_enabled, auto_stop_silence_ms, auto_stop_minimum_speech_ms, silence_splitting, segment_seconds, segment_silence_ms FROM preferences_settings WHERE id=1;

-- name: PutPreferences :exec
INSERT INTO preferences_settings (id, toggle_shortcut, show_shortcut, hold_shortcut, microphone_id, max_duration_seconds, auto_insert, start_with_windows, show_window_on_launch, check_for_updates, setup_completed, use_mica, appearance_mode, overlay_enabled, overlay_size_percent, overlay_opacity_percent, overlay_top_offset, overlay_glow_percent, overlay_layout, overlay_anchor, overlay_visibility, overlay_motion, overlay_surface, overlay_visualizer, history_enabled, vad_enabled, vad_mode, vad_activity_silence_ms, silence_trimming, speech_padding_ms, auto_stop_enabled, auto_stop_silence_ms, auto_stop_minimum_speech_ms, silence_splitting, segment_seconds, segment_silence_ms)
VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET toggle_shortcut=excluded.toggle_shortcut, show_shortcut=excluded.show_shortcut, hold_shortcut=excluded.hold_shortcut, microphone_id=excluded.microphone_id, max_duration_seconds=excluded.max_duration_seconds, auto_insert=excluded.auto_insert, start_with_windows=excluded.start_with_windows, show_window_on_launch=excluded.show_window_on_launch, check_for_updates=excluded.check_for_updates, setup_completed=excluded.setup_completed, use_mica=excluded.use_mica, appearance_mode=excluded.appearance_mode, overlay_enabled=excluded.overlay_enabled, overlay_size_percent=excluded.overlay_size_percent, overlay_opacity_percent=excluded.overlay_opacity_percent, overlay_top_offset=excluded.overlay_top_offset, overlay_glow_percent=excluded.overlay_glow_percent, overlay_layout=excluded.overlay_layout, overlay_anchor=excluded.overlay_anchor, overlay_visibility=excluded.overlay_visibility, overlay_motion=excluded.overlay_motion, overlay_surface=excluded.overlay_surface, overlay_visualizer=excluded.overlay_visualizer, history_enabled=excluded.history_enabled, vad_enabled=excluded.vad_enabled, vad_mode=excluded.vad_mode, vad_activity_silence_ms=excluded.vad_activity_silence_ms, silence_trimming=excluded.silence_trimming, speech_padding_ms=excluded.speech_padding_ms, auto_stop_enabled=excluded.auto_stop_enabled, auto_stop_silence_ms=excluded.auto_stop_silence_ms, auto_stop_minimum_speech_ms=excluded.auto_stop_minimum_speech_ms, silence_splitting=excluded.silence_splitting, segment_seconds=excluded.segment_seconds, segment_silence_ms=excluded.segment_silence_ms;

-- name: GetTranscription :one
SELECT compatibility_profile, base_url, allow_insecure_http, authentication_mode, model, language, health_path, transcription_timeout_seconds, file_transcription_timeout_seconds, transcription_options_prompt, transcription_options_hotwords, transcription_options_temperature_override, transcription_options_temperature FROM transcription_settings WHERE id=1;

-- name: PutTranscription :exec
INSERT INTO transcription_settings (id, compatibility_profile, base_url, allow_insecure_http, authentication_mode, model, language, health_path, transcription_timeout_seconds, file_transcription_timeout_seconds, transcription_options_prompt, transcription_options_hotwords, transcription_options_temperature_override, transcription_options_temperature)
VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET compatibility_profile=excluded.compatibility_profile, base_url=excluded.base_url, allow_insecure_http=excluded.allow_insecure_http, authentication_mode=excluded.authentication_mode, model=excluded.model, language=excluded.language, health_path=excluded.health_path, transcription_timeout_seconds=excluded.transcription_timeout_seconds, file_transcription_timeout_seconds=excluded.file_transcription_timeout_seconds, transcription_options_prompt=excluded.transcription_options_prompt, transcription_options_hotwords=excluded.transcription_options_hotwords, transcription_options_temperature_override=excluded.transcription_options_temperature_override, transcription_options_temperature=excluded.transcription_options_temperature;

-- name: GetCleanup :one
SELECT generation_options_limit_output_tokens, generation_options_max_output_tokens, generation_options_disable_reasoning, compatibility_profile, enabled, base_url, allow_insecure_http, model, preset, system_prompt, styling, structure, context, timeout_seconds FROM cleanup_settings WHERE id=1;

-- name: PutCleanup :exec
INSERT INTO cleanup_settings (id, generation_options_limit_output_tokens, generation_options_max_output_tokens, generation_options_disable_reasoning, compatibility_profile, enabled, base_url, allow_insecure_http, model, preset, system_prompt, styling, structure, context, timeout_seconds)
VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET generation_options_limit_output_tokens=excluded.generation_options_limit_output_tokens, generation_options_max_output_tokens=excluded.generation_options_max_output_tokens, generation_options_disable_reasoning=excluded.generation_options_disable_reasoning, compatibility_profile=excluded.compatibility_profile, enabled=excluded.enabled, base_url=excluded.base_url, allow_insecure_http=excluded.allow_insecure_http, model=excluded.model, preset=excluded.preset, system_prompt=excluded.system_prompt, styling=excluded.styling, structure=excluded.structure, context=excluded.context, timeout_seconds=excluded.timeout_seconds;

-- name: GetSpeech :one
SELECT compatibility_profile, enabled, base_url, allow_insecure_http, authentication_mode, model, voice, speed, timeout_seconds FROM speech_settings WHERE id=1;

-- name: PutSpeech :exec
INSERT INTO speech_settings (id, compatibility_profile, enabled, base_url, allow_insecure_http, authentication_mode, model, voice, speed, timeout_seconds)
VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET compatibility_profile=excluded.compatibility_profile, enabled=excluded.enabled, base_url=excluded.base_url, allow_insecure_http=excluded.allow_insecure_http, authentication_mode=excluded.authentication_mode, model=excluded.model, voice=excluded.voice, speed=excluded.speed, timeout_seconds=excluded.timeout_seconds;

-- name: GetHeaders :many
SELECT name, value FROM request_headers ORDER BY name LIMIT 33;
-- name: ClearHeaders :exec
DELETE FROM request_headers WHERE settings_id=1;
-- name: PutHeader :exec
INSERT INTO request_headers(settings_id,name,value) VALUES(1,?,?);
-- name: GetCredentialRefs :many
SELECT purpose, account FROM credential_refs ORDER BY purpose;
-- name: PutCredentialRef :exec
INSERT INTO credential_refs(purpose,account) VALUES(?,?) ON CONFLICT(purpose) DO UPDATE SET account=excluded.account;
-- name: QueueCredentialGC :exec
INSERT INTO credential_gc(account) VALUES(?) ON CONFLICT(account) DO NOTHING;
-- name: CompleteCredentialGC :exec
DELETE FROM credential_gc WHERE account=?;
-- name: PendingCredentialGC :many
SELECT account FROM credential_gc WHERE account NOT IN (SELECT account FROM credential_refs) ORDER BY account LIMIT 128;
-- name: GetInitialization :one
SELECT source FROM initialization WHERE id=1;
-- name: Initialize :exec
INSERT INTO initialization(id,source) VALUES(1,?);


-- name: CountCredentialGC :one
SELECT count(*) FROM credential_gc;
