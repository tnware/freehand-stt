-- name: ListManagedInstances :many
SELECT * FROM managed_runtime_instances ORDER BY id LIMIT 9;
-- name: PutManagedInstance :exec
INSERT INTO managed_runtime_instances(id,name,provider,model,auto_start,speech_model) VALUES(?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name,provider=excluded.provider,model=excluded.model,auto_start=excluded.auto_start,speech_model=excluded.speech_model;
-- name: DeleteManagedInstance :exec
DELETE FROM managed_runtime_instances WHERE id=?;
