-- name: ListRememberedModels :many
SELECT connection_id,purpose,model,selected,profile,prompt,temperature_override,temperature,limit_output_tokens,max_output_tokens,disable_reasoning,voice,speech_language,speech_instructions FROM remembered_models ORDER BY connection_id,purpose,model LIMIT 16385;

-- name: ClearRememberedModels :exec
DELETE FROM remembered_models;

-- name: PutRememberedModel :exec
INSERT INTO remembered_models (connection_id,purpose,model,selected,profile,prompt,temperature_override,temperature,limit_output_tokens,max_output_tokens,disable_reasoning,voice,speech_language,speech_instructions) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?);
