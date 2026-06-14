-- name: CreateBookmark :exec
INSERT INTO bookmarks (uri, author_did, source, source_hash, title, description, tags_json, created_at, indexed_at, cid)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT(uri) DO UPDATE SET
    source = EXCLUDED.source,
    source_hash = EXCLUDED.source_hash,
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    tags_json = EXCLUDED.tags_json,
    indexed_at = EXCLUDED.indexed_at,
    cid = EXCLUDED.cid;

-- name: GetBookmarkByURI :one
SELECT uri, author_did, source, source_hash, title, description, tags_json, created_at, indexed_at, cid
FROM all_bookmarks
WHERE uri = $1;

-- name: GetRecentBookmarks :many
SELECT uri, author_did, source, source_hash, title, description, tags_json, created_at, indexed_at, cid
FROM all_bookmarks
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetPopularBookmarks :many
SELECT
    b.uri, b.author_did, b.source, b.source_hash, b.title,
    b.description, b.tags_json, b.created_at, b.indexed_at, b.cid
FROM all_bookmarks b
LEFT JOIN LATERAL (
    SELECT COUNT(*) as cnt FROM likes WHERE subject_uri = b.uri
) l ON true
LEFT JOIN LATERAL (
    SELECT COUNT(*) as cnt FROM replies WHERE root_uri = b.uri
) r ON true
WHERE b.created_at > $1 AND (l.cnt + r.cnt) > 0
ORDER BY (l.cnt + r.cnt) DESC, b.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetShelvedBookmarks :many
SELECT
    b.uri, b.author_did, b.source, b.source_hash, b.title,
    b.description, b.tags_json, b.created_at, b.indexed_at, b.cid
FROM all_bookmarks b
WHERE b.created_at < $1 AND b.created_at > $2
    AND NOT EXISTS (SELECT 1 FROM likes WHERE subject_uri = b.uri)
    AND NOT EXISTS (SELECT 1 FROM replies WHERE root_uri = b.uri)
ORDER BY RANDOM()
LIMIT $3 OFFSET $4;

-- name: GetMarginBookmarks :many
SELECT uri, author_did, source, source_hash, title, description, tags_json, created_at, indexed_at, cid
FROM all_bookmarks
WHERE uri NOT LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetSembleBookmarks :many
SELECT uri, author_did, source, source_hash, title, description, tags_json, created_at, indexed_at, cid
FROM all_bookmarks
WHERE uri LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetBookmarksByTag :many
SELECT uri, author_did, source, source_hash, title, description, tags_json, created_at, indexed_at, cid
FROM all_bookmarks
WHERE lower(tags_json)::jsonb ? $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetMarginBookmarksByTag :many
SELECT uri, author_did, source, source_hash, title, description, tags_json, created_at, indexed_at, cid
FROM all_bookmarks
WHERE lower(tags_json)::jsonb ? $1 AND uri NOT LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSembleBookmarksByTag :many
SELECT uri, author_did, source, source_hash, title, description, tags_json, created_at, indexed_at, cid
FROM all_bookmarks
WHERE lower(tags_json)::jsonb ? $1 AND uri LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetBookmarksByTagAndAuthor :many
SELECT uri, author_did, source, source_hash, title, description, tags_json, created_at, indexed_at, cid
FROM all_bookmarks
WHERE author_did = $1 AND lower(tags_json)::jsonb ? $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetMarginBookmarksByTagAndAuthor :many
SELECT uri, author_did, source, source_hash, title, description, tags_json, created_at, indexed_at, cid
FROM all_bookmarks
WHERE author_did = $1 AND lower(tags_json)::jsonb ? $2 AND uri NOT LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetSembleBookmarksByTagAndAuthor :many
SELECT uri, author_did, source, source_hash, title, description, tags_json, created_at, indexed_at, cid
FROM all_bookmarks
WHERE author_did = $1 AND lower(tags_json)::jsonb ? $2 AND uri LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetBookmarksByAuthor :many
SELECT uri, author_did, source, source_hash, title, description, tags_json, created_at, indexed_at, cid
FROM all_bookmarks
WHERE author_did = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetMarginBookmarksByAuthor :many
SELECT uri, author_did, source, source_hash, title, description, tags_json, created_at, indexed_at, cid
FROM all_bookmarks
WHERE author_did = $1 AND uri NOT LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSembleBookmarksByAuthor :many
SELECT uri, author_did, source, source_hash, title, description, tags_json, created_at, indexed_at, cid
FROM all_bookmarks
WHERE author_did = $1 AND uri LIKE '%network.cosmik%'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteBookmark :exec
DELETE FROM bookmarks WHERE uri = $1;

-- name: UpdateBookmark :exec
UPDATE bookmarks
SET title = $1, description = $2, tags_json = $3, cid = $4, indexed_at = $5
WHERE uri = $6;

-- name: GetBookmarksByURIs :many
SELECT uri, author_did, source, source_hash, title, description, tags_json, created_at, indexed_at, cid
FROM all_bookmarks
WHERE uri = ANY($1::text[]);

-- name: GetBookmarkURIs :many
SELECT uri FROM all_bookmarks WHERE author_did = $1;

-- name: GetBookmarksByTargetHash :many
SELECT uri, author_did, source, source_hash, title, description, tags_json, created_at, indexed_at, cid
FROM all_bookmarks
WHERE source_hash = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
