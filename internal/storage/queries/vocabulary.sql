-- name: GetVocabulary :one
SELECT terms, voice, files, boost FROM vocabulary_settings WHERE id=1;
-- name: PutVocabulary :exec
INSERT INTO vocabulary_settings(id,terms,voice,files,boost) VALUES(1,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET terms=excluded.terms,voice=excluded.voice,files=excluded.files,boost=excluded.boost;
