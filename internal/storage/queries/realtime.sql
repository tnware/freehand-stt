-- name: GetRealtime :one
SELECT * FROM realtime_settings WHERE id=1;

-- name: PutRealtime :exec
INSERT INTO realtime_settings(id,enabled,compatibility_profile,model_profile,base_url,allow_insecure_http,authentication_mode,model,language,captions,vocabulary,boost) VALUES(1,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET enabled=excluded.enabled,compatibility_profile=excluded.compatibility_profile,model_profile=excluded.model_profile,base_url=excluded.base_url,allow_insecure_http=excluded.allow_insecure_http,authentication_mode=excluded.authentication_mode,model=excluded.model,language=excluded.language,captions=excluded.captions,vocabulary=excluded.vocabulary,boost=excluded.boost;
