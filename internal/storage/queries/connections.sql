-- name: ListSavedConnections :many
SELECT id,name,compatibility_profile,base_url,allow_insecure_http,authentication_mode,health_path,credential_account FROM saved_connections ORDER BY name,id LIMIT 129;


-- name: ListSelectedConnections :many
SELECT purpose,connection_id FROM selected_connections ORDER BY purpose;


-- name: ListConnectionHeaders :many
SELECT connection_id,name,value FROM saved_connection_headers ORDER BY connection_id,name LIMIT 4097;


-- name: PutSavedConnection :exec
INSERT INTO saved_connections(id,name,compatibility_profile,base_url,allow_insecure_http,authentication_mode,health_path,credential_account) VALUES(?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET name=excluded.name,compatibility_profile=excluded.compatibility_profile,base_url=excluded.base_url,allow_insecure_http=excluded.allow_insecure_http,authentication_mode=excluded.authentication_mode,health_path=excluded.health_path,credential_account=excluded.credential_account;


-- name: SelectSavedConnection :exec
INSERT INTO selected_connections(purpose,connection_id) VALUES(?,?) ON CONFLICT(purpose) DO UPDATE SET connection_id=excluded.connection_id;


-- name: DeleteSavedConnection :exec
DELETE FROM saved_connections WHERE id=?;


-- name: ClearConnectionHeaders :exec
DELETE FROM saved_connection_headers WHERE connection_id=?;


-- name: PutConnectionHeader :exec
INSERT INTO saved_connection_headers(connection_id,name,value) VALUES(?,?,?);



-- name: ClearSelectedConnections :exec
DELETE FROM selected_connections;



-- name: ListConnectionUses :many
SELECT connection_id,purpose FROM saved_connection_uses ORDER BY connection_id,purpose LIMIT 513;


-- name: ClearConnectionUses :exec
DELETE FROM saved_connection_uses;


-- name: PutConnectionUse :exec
INSERT INTO saved_connection_uses(connection_id,purpose) VALUES(?,?);


-- name: GetSelectedConnectionDetails :many
SELECT s.purpose, c.* FROM selected_connections s JOIN saved_connections c ON c.id=s.connection_id ORDER BY s.purpose;
