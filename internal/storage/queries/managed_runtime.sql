-- name: GetManagedRuntime :one
SELECT enabled, model, realtime FROM managed_runtime_preferences WHERE id = 1;

-- name: PutManagedRuntime :exec
INSERT INTO managed_runtime_preferences (id, enabled, model, realtime)
VALUES (1, ?, ?, ?)
ON CONFLICT (id) DO UPDATE SET enabled = excluded.enabled, model = excluded.model, realtime = excluded.realtime;
