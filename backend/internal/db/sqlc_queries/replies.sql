-- name: CreateReply :exec
INSERT INTO replies (uri, author_did, parent_uri, root_uri, text, format, created_at, indexed_at, cid)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT(uri) DO UPDATE SET
    text = EXCLUDED.text,
    format = EXCLUDED.format,
    indexed_at = EXCLUDED.indexed_at,
    cid = EXCLUDED.cid;

-- name: GetRepliesByRoot :many
SELECT uri, author_did, parent_uri, root_uri, text, format, created_at, indexed_at, cid
FROM replies
WHERE root_uri = $1
ORDER BY created_at ASC;

-- name: GetReplyByURI :one
SELECT uri, author_did, parent_uri, root_uri, text, format, created_at, indexed_at, cid
FROM replies
WHERE uri = $1;

-- name: DeleteReply :exec
DELETE FROM replies WHERE uri = $1;

-- name: GetRepliesByAuthor :many
SELECT uri, author_did, parent_uri, root_uri, text, format, created_at, indexed_at, cid
FROM replies
WHERE author_did = $1
ORDER BY created_at DESC;

-- name: GetOrphanedRepliesByAuthor :many
SELECT r.uri, r.author_did, r.parent_uri, r.root_uri, r.text, r.format, r.created_at, r.indexed_at, r.cid
FROM replies r
LEFT JOIN annotations a ON r.root_uri = a.uri
WHERE r.author_did = $1 AND a.uri IS NULL;

-- name: GetReplyCount :one
SELECT COUNT(*) FROM replies WHERE root_uri = $1;

-- name: GetReplyCounts :many
SELECT root_uri, COUNT(*)
FROM replies
WHERE root_uri = ANY($1::text[])
GROUP BY root_uri;

-- name: GetRepliesByURIs :many
SELECT uri, author_did, parent_uri, root_uri, text, format, created_at, indexed_at, cid
FROM replies
WHERE uri = ANY($1::text[]);
