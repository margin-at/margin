-- name: CreateAPIKey :exec
INSERT INTO api_keys (id, owner_did, name, key_hash, created_at, uri, cid)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    key_hash = EXCLUDED.key_hash,
    uri = EXCLUDED.uri,
    cid = EXCLUDED.cid;

-- name: GetAPIKeysByOwner :many
SELECT id, owner_did, name, key_hash, created_at, last_used_at
FROM api_keys
WHERE owner_did = $1
ORDER BY created_at DESC;

-- name: GetAPIKeyByHash :one
SELECT id, owner_did, name, key_hash, created_at, last_used_at
FROM api_keys
WHERE key_hash = $1;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE api_keys SET last_used_at = $1 WHERE id = $2;

-- name: DeleteAPIKeyReturningURI :one
DELETE FROM api_keys WHERE id = $1 AND owner_did = $2 RETURNING uri;
