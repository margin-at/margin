-- name: InsertKVIgnore :exec
INSERT INTO kv_store (key, value, updated_at) VALUES ($1, $2, NOW()) ON CONFLICT (key) DO NOTHING;

-- name: GetKVValue :one
SELECT value FROM kv_store WHERE key = $1;
